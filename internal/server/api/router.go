package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"

	v1 "github.com/skenzeriq/patchiq/internal/server/api/v1"
	"github.com/skenzeriq/patchiq/internal/server/auth"

	serverlicense "github.com/skenzeriq/patchiq/internal/server/license"
	"github.com/skenzeriq/patchiq/internal/server/policy"
	"github.com/skenzeriq/patchiq/internal/server/repo"
	"github.com/skenzeriq/patchiq/internal/server/store"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/server/targeting"
	"github.com/skenzeriq/patchiq/internal/shared/bodylimit"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/idempotency"
	piqotel "github.com/skenzeriq/patchiq/internal/shared/otel"
	"github.com/skenzeriq/patchiq/internal/shared/ratelimit"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"github.com/skenzeriq/patchiq/internal/shared/user"
)

// orgScopeLookupAdapter adapts *store.Store to auth.OrgScopeLookup so the
// SSOHandler.Me endpoint can populate organization + accessible tenants in
// its response without the auth package importing sqlcgen.
type orgScopeLookupAdapter struct {
	s *store.Store
}

func (a *orgScopeLookupAdapter) GetOrganizationByIDForMe(ctx context.Context, orgID string) (string, string, string, error) {
	return a.s.GetOrganizationByIDForMe(ctx, orgID)
}

func (a *orgScopeLookupAdapter) UserAccessibleTenantsForMe(ctx context.Context, orgID, userID string) ([]auth.AccessibleTenantInfo, error) {
	tenants, err := a.s.UserAccessibleTenants(ctx, orgID, userID)
	if err != nil {
		return nil, err
	}
	out := make([]auth.AccessibleTenantInfo, len(tenants))
	for i, t := range tenants {
		out[i] = auth.AccessibleTenantInfo{
			ID:   uuid.UUID(t.ID.Bytes).String(),
			Name: t.Name,
			Slug: t.Slug,
		}
	}
	return out, nil
}

// NewRouter creates the chi router with the full middleware chain.
// Health endpoints are outside tenant middleware (must work without X-Tenant-ID).
func NewRouter(st *store.Store, eventBus domain.EventBus, hubURL, hubAPIKey string, startTime time.Time, idempotencyStore idempotency.Store, version string, discoveryHandler *v1.DiscoveryHandler, deploymentHandler *v1.DeploymentHandler, scheduleHandler *v1.ScheduleHandler, scanScheduler v1.EndpointScanner, licenseSvc *serverlicense.Service, corsOrigins []string, notificationHandler *v1.NotificationHandler, complianceHandler *v1.ComplianceHandler, jwtMiddleware func(http.Handler) http.Handler, ssoHandler *auth.SSOHandler, iamHandler *v1.IAMSettingsHandler, roleMappingHandler *v1.RoleMappingHandler, hubSyncAPIHandler *v1.HubSyncAPIHandler, cveMatchInserter v1.CVEMatchInserter, notifByTypeHandler *v1.NotificationByTypeHandler, generalSettingsHandler *v1.GeneralSettingsHandler, loginHandler *auth.LoginHandler, inviteHandler *auth.InviteHandler, alertBackfiller v1.AlertBackfiller, agentBinariesHandler *v1.AgentBinariesHandler, reportHandler *v1.ReportHandler, healthChecks map[string]v1.CheckFunc) chi.Router {
	r := chi.NewRouter()

	r.Use(piqotel.HTTPMiddleware("patchiq-server"))
	r.Use(middleware.RequestID)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if rid := middleware.GetReqID(r.Context()); rid != "" {
				ctx := piqotel.WithRequestID(r.Context(), rid)
				next.ServeHTTP(w, r.WithContext(ctx))
			} else {
				next.ServeHTTP(w, r)
			}
		})
	})
	r.Use(middleware.RealIP)
	r.Use(slogRequestLogger)
	r.Use(middleware.Recoverer)
	r.Use(bodylimit.Middleware(bodylimit.DefaultMaxBodySize))
	origins := corsOrigins
	if len(origins) == 0 {
		origins = []string{"http://localhost:5173", "http://localhost:5174", "http://localhost:5175"}
		slog.Warn("no CORS origins configured, using dev defaults", "origins", origins)
	}
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Tenant-ID", "X-Request-ID", "Idempotency-Key", "X-User-ID"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	health := v1.NewHealthHandler(st, startTime, version, healthChecks)
	r.Get("/health", health.Health)
	r.Get("/ready", health.Ready)

	repoCacheDir := os.Getenv("PATCHIQ_REPO_CACHE_DIR")
	if repoCacheDir == "" {
		repoCacheDir = "/tmp/patchiq-repo-cache"
	}
	repo.MountFileServer(r, repoCacheDir)

	if ssoHandler != nil {
		v1.RegisterAuthRoutes(r, ssoHandler)
	}

	// Shared rate limit store for all in-memory rate limiters.
	rateLimitStore := ratelimit.NewMemoryStore()

	// Register login, forgot-password, register, and invite validation routes (public, rate-limited).
	if loginHandler != nil || inviteHandler != nil {
		rl := auth.RateLimitMiddleware(rateLimitStore, 10, 1*time.Minute)
		v1.RegisterLoginRoutes(r, loginHandler, inviteHandler, rl)
	}

	// Create sqlc querier from store pool
	q := sqlcgen.New(st.Pool())

	permStore := auth.NewSQLPermissionStore(st).WithOrgLoader(st)
	if ssoHandler != nil {
		ssoHandler.PermStore = permStore
		ssoHandler.OrgPerms = permStore
		ssoHandler.RoleStore = permStore
		ssoHandler.OrgScope = &orgScopeLookupAdapter{s: st}
	}
	rbacEval := auth.NewEvaluator(permStore).WithOrgStore(permStore)
	orgResolver := auth.NewOrgResolver(st, st)
	rp := func(resource, action string) func(http.Handler) http.Handler {
		return auth.RequirePermission(rbacEval, resource, action)
	}

	eh := v1.NewEndpointHandler(q, eventBus, scanScheduler)
	ch := v1.NewCommandsHandler(q)
	if cveMatchInserter != nil {
		eh.SetCVEMatchInserter(cveMatchInserter)
	}
	th := v1.NewTagHandler(q, st.Pool(), eventBus)
	trh := v1.NewTagRuleHandler(q, eventBus)
	tkh := v1.NewTagKeyHandler(q, eventBus)

	patchH := v1.NewPatchHandler(q).WithPool(st.Pool()).WithEventBus(eventBus)
	cveH := v1.NewCVEHandler(q)
	dashH := v1.NewDashboardHandler(q)
	auditH := v1.NewAuditHandler(q)

	// Single targeting resolver shared by the policy evaluator and the
	// policy handler's match-count previews. It reads the raw pgxpool
	// so it can run its own transactions (RLS via SET LOCAL).
	resolver := targeting.NewResolver(st.Pool())

	polDS := policy.NewSQLDataSource(q, resolver)
	evaluator := policy.NewEvaluator(polDS)
	ph := v1.NewPolicyHandler(q, st.Pool(), eventBus, evaluator, resolver)
	tsh := v1.NewTagSelectorHandler(resolver)

	rh := v1.NewRoleHandler(q, st.Pool(), eventBus, nil)
	urh := v1.NewUserRoleHandler(q, eventBus)
	orgH := v1.NewOrganizationHandler(st, eventBus)

	r.Route("/api/v1", func(r chi.Router) {
		// General API rate limit: 100 requests/minute per IP.
		r.Use(ratelimit.Middleware(rateLimitStore, 100, 1*time.Minute, "api"))

		if jwtMiddleware != nil {
			r.Use(jwtMiddleware)
		} else {
			r.Use(tenant.Middleware)
			r.Use(auth.UserMiddleware)
		}
		// Org scope middleware: resolves Zitadel org → PatchIQ organization
		// and refines the active tenant based on user's accessible tenants.
		// Safe to always install — no-op when the resolver finds no mapping,
		// so single-tenant deployments behave exactly as before.
		r.Use(auth.NewOrgScopeMiddleware(orgResolver))
		r.Use(idempotency.Middleware(idempotencyStore))

		if ssoHandler != nil {
			v1.RegisterAuthenticatedAuthRoutes(r, ssoHandler)
		}
		if inviteHandler != nil {
			r.With(rp("settings", "manage")).Post("/auth/invite", inviteHandler.CreateInvite)
		}
		if ssoHandler == nil {
			slog.Warn("SSO handler not configured: registering dev stub handlers for /auth/me and /auth/logout")
			r.Get("/auth/me", devStubMeHandlerFunc(permStore))
			r.Post("/auth/logout", devStubLogoutHandler)
		}
		if iamHandler != nil {
			r.With(rp("settings", "read")).Get("/settings/iam", iamHandler.Get)
			r.With(rp("settings", "write")).Put("/settings/iam", iamHandler.Update)
			r.With(rp("settings", "write")).Post("/settings/iam/test", iamHandler.TestConnection)
		}
		r.With(rp("settings", "read")).Get("/settings", generalSettingsHandler.Get)
		r.With(rp("settings", "write")).Put("/settings", generalSettingsHandler.Update)
		if roleMappingHandler != nil {
			r.With(rp("rbac", "manage")).Get("/settings/role-mapping", roleMappingHandler.Get)
			r.With(rp("rbac", "manage")).Put("/settings/role-mapping", roleMappingHandler.Update)
		}

		if agentBinariesHandler != nil {
			r.With(rp("endpoints", "read")).Get("/agent-binaries", agentBinariesHandler.List)
			r.With(rp("endpoints", "read")).Get("/agent-binaries/{filename}/download", agentBinariesHandler.Download)
		}

		regH := v1.NewRegistrationHandler(q, eventBus)
		r.Route("/registrations", func(r chi.Router) {
			r.With(rp("endpoints", "create")).Post("/", regH.Create)
			r.With(rp("endpoints", "read")).Get("/", regH.List)
			r.With(rp("endpoints", "create")).Delete("/{id}", regH.Revoke)
		})

		r.Route("/organizations", func(r chi.Router) {
			r.With(rp("organizations", "read")).Get("/", orgH.List)
			r.With(rp("organizations", "manage")).Post("/", orgH.Create)
			r.With(rp("organizations", "read")).Get("/{id}", orgH.Get)
			r.With(rp("organizations", "manage")).Delete("/{id}", orgH.Delete)
			r.With(rp("organizations", "read")).Get("/{id}/tenants", orgH.ListTenants)
			r.With(rp("organizations", "provision")).Post("/{id}/tenants", orgH.ProvisionTenant)
			r.With(rp("organizations", "read")).Get("/{id}/dashboard", orgH.Dashboard)
			r.With(rp("organizations", "manage")).Post("/{id}/users/{user_id}/roles", orgH.AssignUserRole)
			r.With(rp("organizations", "manage")).Delete("/{id}/users/{user_id}/roles/{role_id}", orgH.RevokeUserRole)
		})

		r.Route("/endpoints", func(r chi.Router) {
			r.With(rp("endpoints", "read")).Get("/", eh.List)
			r.With(rp("endpoints", "read")).Get("/export", eh.Export)
			r.With(rp("endpoints", "read")).Get("/{id}", eh.Get)
			r.With(rp("endpoints", "update")).Put("/{id}", eh.Update)
			r.With(rp("endpoints", "delete")).Delete("/{id}", eh.Delete)
			r.With(rp("endpoints", "scan")).Post("/{id}/scan", eh.Scan)
			r.With(rp("endpoints", "read")).Get("/{id}/active-scan", eh.ActiveScan)
			r.With(rp("endpoints", "scan")).Post("/{id}/scan-cves", eh.ScanCVEs)
			r.With(rp("endpoints", "read")).Get("/{id}/cves", eh.ListCVEs)
			r.With(rp("endpoints", "read")).Get("/{id}/packages", eh.ListPackages)
			r.With(rp("endpoints", "read")).Get("/{id}/patches", eh.ListPatches)
			r.With(rp("deployments", "create")).Post("/{id}/deploy-critical", patchH.DeployCritical)
			r.With(rp("endpoints", "read")).Get("/{id}/deployments", eh.ListDeploymentHistory)
		})
		r.Route("/commands", func(r chi.Router) {
			r.With(rp("endpoints", "read")).Get("/{id}", ch.Get)
		})
		r.Route("/tags", func(r chi.Router) {
			r.With(rp("endpoints", "read")).Get("/", th.List)
			r.With(rp("endpoints", "read")).Get("/keys", th.ListKeys)
			r.With(rp("endpoints", "create")).Post("/", th.Create)
			r.With(rp("endpoints", "read")).Post("/selectors/validate", tsh.Validate)
			r.With(rp("endpoints", "read")).Get("/{id}", th.Get)
			r.With(rp("endpoints", "update")).Put("/{id}", th.Update)
			r.With(rp("endpoints", "delete")).Delete("/{id}", th.Delete)
			r.With(rp("endpoints", "update")).Post("/{id}/assign", th.Assign)
			r.With(rp("endpoints", "update")).Post("/{id}/unassign", th.Unassign)
		})
		r.Route("/tag-keys", func(r chi.Router) {
			r.With(rp("endpoints", "read")).Get("/", tkh.List)
			r.With(rp("endpoints", "create")).Post("/", tkh.Upsert)
			r.With(rp("endpoints", "delete")).Delete("/{key}", tkh.Delete)
		})
		r.Route("/tag-rules", func(r chi.Router) {
			r.With(rp("endpoints", "read")).Get("/", trh.List)
			r.With(rp("endpoints", "create")).Post("/", trh.Create)
			r.With(rp("endpoints", "read")).Get("/{id}", trh.Get)
			r.With(rp("endpoints", "update")).Put("/{id}", trh.Update)
			r.With(rp("endpoints", "delete")).Delete("/{id}", trh.Delete)
		})
		r.Route("/policies", func(r chi.Router) {
			r.With(rp("policies", "create")).Post("/", ph.Create)
			r.With(rp("policies", "read")).Get("/", ph.List)
			r.With(rp("policies", "update")).Post("/bulk", ph.BulkAction)
			r.With(rp("policies", "read")).Get("/{id}", ph.Get)
			r.With(rp("policies", "update")).Put("/{id}", ph.Update)
			r.With(rp("policies", "update")).Patch("/{id}", ph.Toggle)
			r.With(rp("policies", "delete")).Delete("/{id}", ph.Delete)
			r.With(rp("policies", "read")).Post("/{id}/evaluate", ph.Evaluate)
		})
		r.Route("/deployments", func(r chi.Router) {
			r.With(rp("deployments", "create")).Post("/", deploymentHandler.Create)
			r.With(rp("deployments", "read")).Get("/", deploymentHandler.List)
			r.With(rp("deployments", "read")).Get("/{id}", deploymentHandler.Get)
			r.With(rp("deployments", "read")).Get("/{id}/waves", deploymentHandler.GetWaves)
			r.With(rp("deployments", "cancel")).Post("/{id}/cancel", deploymentHandler.Cancel)
			r.With(rp("deployments", "update")).Post("/{id}/retry", deploymentHandler.Retry)
			r.With(rp("deployments", "update")).Post("/{id}/rollback", deploymentHandler.Rollback)
			r.With(rp("deployments", "read")).Get("/{id}/patches", deploymentHandler.GetPatchSummary)
		})
		r.Route("/deployment-schedules", func(r chi.Router) {
			r.With(rp("deployments", "create")).Post("/", scheduleHandler.Create)
			r.With(rp("deployments", "read")).Get("/", scheduleHandler.List)
			r.With(rp("deployments", "read")).Get("/{id}", scheduleHandler.Get)
			r.With(rp("deployments", "update")).Patch("/{id}", scheduleHandler.Update)
			r.With(rp("deployments", "delete")).Delete("/{id}", scheduleHandler.Delete)
		})

		r.With(rp("settings", "update")).Post("/admin/discovery/trigger", discoveryHandler.Trigger)

		hubSync := v1.NewHubSyncHandler(hubURL, hubAPIKey, eventBus, st)
		r.With(rp("settings", "update")).Post("/admin/sync/hub", hubSync.TriggerSync)

		if hubSyncAPIHandler != nil {
			r.Route("/sync", func(r chi.Router) {
				r.With(rp("settings", "read")).Get("/status", hubSyncAPIHandler.Status)
				r.With(rp("settings", "update")).Post("/trigger", hubSyncAPIHandler.Trigger)
				r.With(rp("settings", "update")).Put("/config", hubSyncAPIHandler.UpdateConfig)
			})
		}

		r.Route("/patches", func(r chi.Router) {
			r.With(rp("patches", "read")).Get("/", patchH.List)
			r.With(rp("patches", "read")).Get("/severity-counts", patchH.SeverityCounts)
			r.With(rp("patches", "read")).Get("/{id}", patchH.Get)
			r.With(rp("deployments", "create")).Post("/{id}/deploy", patchH.QuickDeploy)
		})
		r.Route("/cves", func(r chi.Router) {
			r.With(rp("patches", "read")).Get("/", cveH.List)
			r.With(rp("patches", "read")).Get("/summary", cveH.Summary)
			r.With(rp("patches", "read")).Get("/{id}", cveH.Get)
		})
		r.With(rp("endpoints", "read")).Get("/dashboard/summary", dashH.Summary)
		r.With(rp("endpoints", "read")).Get("/dashboard/stats", dashH.Summary) // alias
		r.With(rp("endpoints", "read")).Get("/dashboard/activity", dashH.Activity)
		r.With(rp("endpoints", "read")).Get("/dashboard/blast-radius", dashH.BlastRadius)
		r.With(rp("endpoints", "read")).Get("/dashboard/endpoints-risk", dashH.EndpointsRisk)
		r.With(rp("endpoints", "read")).Get("/dashboard/exposure-windows", dashH.ExposureWindows)
		r.With(rp("endpoints", "read")).Get("/dashboard/mttr", dashH.MTTR)
		r.With(rp("endpoints", "read")).Get("/dashboard/attack-paths", dashH.AttackPaths)
		r.With(rp("endpoints", "read")).Get("/dashboard/drift", dashH.Drift)
		r.With(rp("endpoints", "read")).Get("/dashboard/sla-forecast", dashH.SLAForecast)
		r.With(rp("endpoints", "read")).Get("/dashboard/sla-deadlines", dashH.SLADeadlines)
		r.With(rp("endpoints", "read")).Get("/dashboard/sla-tiers", dashH.SLATiers)
		r.With(rp("endpoints", "read")).Get("/dashboard/risk-projection", dashH.RiskProjection)
		r.With(rp("audit", "read")).Get("/audit", auditH.List)
		r.With(rp("audit", "read")).Get("/audit/export", auditH.Export)

		alertH := v1.NewAlertHandler(q, st.Pool(), eventBus, alertBackfiller)

		r.Route("/alerts", func(r chi.Router) {
			r.With(rp("alerts", "read")).Get("/", alertH.List)
			r.With(rp("alerts", "read")).Get("/count", alertH.Count)
			r.With(rp("alerts", "update")).Patch("/{id}/status", alertH.UpdateStatus)
			r.With(rp("alerts", "update")).Patch("/bulk-status", alertH.BulkUpdateStatus)
		})

		r.Route("/alert-rules", func(r chi.Router) {
			r.With(rp("alerts", "manage")).Get("/", alertH.ListRules)
			r.With(rp("alerts", "manage")).Post("/", alertH.CreateRule)
			r.With(rp("alerts", "manage")).Put("/{id}", alertH.UpdateRule)
			r.With(rp("alerts", "manage")).Delete("/{id}", alertH.DeleteRule)
		})

		customComplianceH := v1.NewCustomComplianceHandler(q, eventBus, st)

		r.Route("/compliance", func(r chi.Router) {
			r.With(rp("compliance", "read")).Get("/summary", complianceHandler.Summary)
			r.With(rp("compliance", "read")).Get("/score", complianceHandler.GetOverallScore)
			r.With(rp("compliance", "read")).Get("/overdue", complianceHandler.ListOverdueControls)
			r.With(rp("compliance", "read")).Get("/frameworks", complianceHandler.ListFrameworks)
			r.With(rp("compliance", "create")).Post("/frameworks", complianceHandler.EnableFramework)
			r.With(rp("compliance", "read")).Get("/frameworks/{frameworkId}", complianceHandler.GetFrameworkDetail)
			r.With(rp("compliance", "read")).Get("/frameworks/{frameworkId}/controls", complianceHandler.ListFrameworkControls)
			r.With(rp("compliance", "read")).Get("/frameworks/{frameworkId}/trend", complianceHandler.GetFrameworkTrend)
			r.With(rp("compliance", "create")).Post("/frameworks/{frameworkId}/evaluate", complianceHandler.TriggerFrameworkEvaluation)
			r.With(rp("compliance", "update")).Put("/frameworks/{id}", complianceHandler.UpdateFramework)
			r.With(rp("compliance", "delete")).Delete("/frameworks/{id}", complianceHandler.DisableFramework)
			r.With(rp("compliance", "read")).Get("/endpoints/{id}", complianceHandler.GetEndpointCompliance)
			r.With(rp("compliance", "create")).Post("/evaluate", complianceHandler.TriggerEvaluation)

			// Check types (available evaluators for custom controls)
			r.With(rp("compliance", "read")).Get("/check-types", customComplianceH.ListCheckTypes)

			// Custom framework routes
			r.With(rp("compliance", "read")).Get("/custom-frameworks", customComplianceH.List)
			r.With(rp("compliance", "create")).Post("/custom-frameworks", customComplianceH.Create)
			r.With(rp("compliance", "read")).Get("/custom-frameworks/{id}", customComplianceH.Get)
			r.With(rp("compliance", "update")).Put("/custom-frameworks/{id}", customComplianceH.Update)
			r.With(rp("compliance", "delete")).Delete("/custom-frameworks/{id}", customComplianceH.Delete)
			r.With(rp("compliance", "update")).Put("/custom-frameworks/{id}/controls", customComplianceH.UpdateControls)
		})

		if reportHandler != nil {
			r.Route("/reports", func(r chi.Router) {
				r.With(rp("reports", "create")).Post("/generate", reportHandler.Generate)
				r.With(rp("reports", "read")).Get("/counts", reportHandler.Counts)
				r.With(rp("reports", "read")).Get("/", reportHandler.List)
				r.With(rp("reports", "read")).Get("/{id}", reportHandler.Get)
				r.With(rp("reports", "export")).Get("/{id}/download", reportHandler.Download)
				r.With(rp("reports", "delete")).Delete("/{id}", reportHandler.Delete)
			})
		}

		r.Route("/notifications", func(r chi.Router) {
			r.Route("/channels", func(r chi.Router) {
				r.With(rp("settings", "create")).Post("/", notificationHandler.CreateChannel)
				r.With(rp("settings", "read")).Get("/", notificationHandler.ListChannels)
				r.With(rp("settings", "read")).Get("/{id}", notificationHandler.GetChannel)
				r.With(rp("settings", "update")).Put("/{id}", notificationHandler.UpdateChannel)
				r.With(rp("settings", "delete")).Delete("/{id}", notificationHandler.DeleteChannel)
				r.With(rp("settings", "update")).Post("/{id}/test", notificationHandler.TestChannel)
				if notifByTypeHandler != nil {
					r.Route("/by-type/{type}", func(r chi.Router) {
						r.With(rp("settings", "read")).Get("/", notifByTypeHandler.GetByType)
						r.With(rp("settings", "update")).Put("/", notifByTypeHandler.UpdateByType)
						r.With(rp("settings", "update")).Post("/test", notifByTypeHandler.TestByType)
					})
				}
			})
			r.With(rp("settings", "read")).Get("/preferences", notificationHandler.GetPreferences)
			r.With(rp("settings", "update")).Put("/preferences", notificationHandler.UpdatePreferences)
			r.With(rp("settings", "read")).Get("/history", notificationHandler.ListHistory)
			r.With(rp("settings", "update")).Post("/history/{id}/retry", notificationHandler.RetryNotification)
			r.With(rp("settings", "read")).Get("/digest-config", notificationHandler.GetDigestConfig)
			r.With(rp("settings", "update")).Put("/digest-config", notificationHandler.UpdateDigestConfig)
			r.With(rp("settings", "update")).Post("/digest/test", notificationHandler.TestDigest)
		})

		wfH := v1.NewWorkflowHandler(q, st.Pool(), eventBus)
		wfExecH := v1.NewWorkflowExecutionHandler(q, st.Pool(), eventBus, cveMatchInserter)
		r.Route("/workflows", func(r chi.Router) {
			r.With(rp("workflows", "read")).Get("/", wfH.List)
			r.With(rp("workflows", "create")).Post("/", wfH.Create)
			r.With(rp("workflows", "read")).Get("/{id}", wfH.Get)
			r.With(rp("workflows", "update")).Put("/{id}", wfH.Update)
			r.With(rp("workflows", "delete")).Delete("/{id}", wfH.Delete)
			r.With(rp("workflows", "update")).Put("/{id}/publish", wfH.Publish)
			r.With(rp("workflows", "read")).Get("/{id}/versions", wfH.ListVersions)

			r.With(rp("workflows", "execute")).Post("/{id}/execute", wfExecH.Execute)
			r.With(rp("workflows", "read")).Get("/{id}/executions", wfExecH.List)
			r.With(rp("workflows", "read")).Get("/{id}/executions/{execId}", wfExecH.Get)
			r.With(rp("workflows", "execute")).Post("/{id}/executions/{execId}/cancel", wfExecH.Cancel)
			r.With(rp("workflows", "execute")).Post("/{id}/executions/{execId}/approve", wfExecH.Approve)
			r.With(rp("workflows", "execute")).Post("/{id}/executions/{execId}/reject", wfExecH.Reject)
		})
		r.With(rp("workflows", "read")).Get("/workflow-templates", wfH.ListTemplates)

		licSvc := licenseSvc
		if licSvc == nil {
			licSvc = serverlicense.NewService(serverlicense.NewValidator(nil), nil)
		}
		lh := v1.NewLicenseHandler(licSvc, nil)
		r.With(rp("settings", "read")).Get("/license/status", lh.Status)
		r.With(rp("settings", "read")).Get("/license", lh.Status)

		r.Route("/roles", func(r chi.Router) {
			r.With(rp("roles", "read")).Get("/", rh.List)
			r.With(rp("roles", "create")).Post("/", rh.Create)
			r.With(rp("roles", "read")).Get("/{id}", rh.Get)
			r.With(rp("roles", "update")).Put("/{id}", rh.Update)
			r.With(rp("roles", "delete")).Delete("/{id}", rh.Delete)
			r.With(rp("roles", "read")).Get("/{id}/permissions", rh.GetPermissions)
		})
		r.Route("/users/{id}/roles", func(r chi.Router) {
			r.With(rp("roles", "read")).Get("/", urh.List)
			r.With(rp("roles", "update")).Post("/", urh.Assign)
			r.With(rp("roles", "update")).Delete("/{roleId}", urh.Revoke)
		})
	})

	return r
}

// devStubMeHandlerFunc returns a handler that builds a user identity from
// the request context and the database (permissions + roles). Used when
// Zitadel SSO is not configured.
func devStubMeHandlerFunc(store *auth.SQLPermissionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := user.UserIDFromContext(r.Context())
		if userID == "" {
			userID = "dev-user-id"
		}
		tenantID, _ := tenant.TenantIDFromContext(r.Context())

		resp := map[string]any{
			"user_id":   userID,
			"tenant_id": tenantID,
			"name":      userID,
		}

		if store != nil && tenantID != "" {
			if perms, err := store.GetUserPermissions(r.Context(), tenantID, userID); err == nil {
				entries := make([]map[string]string, len(perms))
				for i, p := range perms {
					entries[i] = map[string]string{"resource": p.Resource, "action": p.Action, "scope": p.Scope}
				}
				resp["permissions"] = entries
			} else {
				slog.WarnContext(r.Context(), "dev stub me: failed to load permissions", "error", err)
			}

			if roles, err := store.GetUserRoles(r.Context(), tenantID, userID); err == nil {
				resp["roles"] = roles
			} else {
				slog.WarnContext(r.Context(), "dev stub me: failed to load roles", "error", err)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			slog.ErrorContext(r.Context(), "dev stub me: failed to encode response", "error", err)
		}
	}
}

// devStubLogoutHandler returns 200 OK when Zitadel SSO is not configured.
func devStubLogoutHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		slog.ErrorContext(r.Context(), "dev stub logout: failed to encode response", "error", err)
	}
}

func slogRequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		start := time.Now()

		defer func() {
			status := ww.Status()
			attrs := []any{
				"method", r.Method,
				"path", r.URL.Path,
				"status", status,
				"bytes", ww.BytesWritten(),
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", middleware.GetReqID(r.Context()),
			}
			switch {
			case status >= 500:
				slog.ErrorContext(r.Context(), "http request", attrs...)
			case status >= 400:
				slog.WarnContext(r.Context(), "http request", attrs...)
			default:
				slog.InfoContext(r.Context(), "http request", attrs...)
			}
		}()

		next.ServeHTTP(ww, r)
	})
}
