package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	v1 "github.com/skenzeriq/patchiq/internal/hub/api/v1"
	hubauth "github.com/skenzeriq/patchiq/internal/hub/auth"
	"github.com/skenzeriq/patchiq/internal/hub/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/bodylimit"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/idempotency"
	piqotel "github.com/skenzeriq/patchiq/internal/shared/otel"
	"github.com/skenzeriq/patchiq/internal/shared/ratelimit"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// NewRouter creates the chi router with the full middleware chain.
// Health endpoints are outside tenant middleware (must work without X-Tenant-ID).
func NewRouter(pool *pgxpool.Pool, eventBus domain.EventBus, syncAPIKey string, startTime time.Time, version string, idempotencyStore idempotency.Store, corsOrigins []string, riverClient v1.RiverEnqueuer, binaryStore v1.BinaryStore, binaryBucket string, jwtMW func(http.Handler) http.Handler, loginHandler *hubauth.LoginHandler, defaultTenantID string, healthChecks map[string]v1.CheckFunc) chi.Router {
	r := chi.NewRouter()

	r.Use(piqotel.HTTPMiddleware("patchiq-hub"))
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
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Tenant-ID", "X-Request-ID", "Idempotency-Key"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	health := v1.NewHealthHandler(pool, startTime, version, healthChecks)
	r.Get("/health", health.Health)
	r.Get("/ready", health.Ready)

	queries := sqlcgen.New(pool)
	catalog := v1.NewCatalogHandler(queries, eventBus)
	feeds := v1.NewFeedHandler(queries, riverClient, eventBus)
	clients := v1.NewClientHandler(queries, eventBus, defaultTenantID)
	licenses := v1.NewLicenseHandler(queries, eventBus)
	dashboard := v1.NewDashboardHandler(queries)

	// Shared rate limit store for hub API and auth routes.
	rateLimitStore := ratelimit.NewMemoryStore()

	sync := v1.NewSyncHandler(queries, syncAPIKey, eventBus)
	r.Get("/api/v1/sync", sync.Sync)
	r.Get("/api/v1/sync/cves", sync.SyncCVEs)

	// Binary download endpoint — protected by sync API key.
	// Used by Patch Manager to download cached binaries during catalog sync.
	if binaryStore != nil {
		catalogBinary := v1.NewCatalogBinaryHandler(queries, binaryStore, binaryBucket)
		r.With(v1.SyncAuthMiddleware(syncAPIKey)).Get("/api/v1/catalog/{id}/binary", catalogBinary.GetBinary)
	}

	// Auth routes (outside tenant middleware, auth-scoped rate limit: 10 req/min).
	authRL := ratelimit.Middleware(rateLimitStore, 10, 1*time.Minute, "auth")
	if loginHandler != nil {
		r.With(authRL).Post("/api/v1/auth/login", loginHandler.Login)
		if jwtMW != nil {
			r.Group(func(r chi.Router) {
				r.Use(jwtMW)
				r.Get("/api/v1/auth/me", loginHandler.Me)
				r.Post("/api/v1/auth/logout", loginHandler.Logout)
			})
		}
	}

	// Public routes — client registration (no tenant middleware).
	// Uses explicit paths instead of r.Route() to avoid creating a catch-all
	// subrouter that would shadow /api/v1/clients in the tenant-scoped group.
	r.Post("/api/v1/clients/register", clients.Register)
	r.Get("/api/v1/clients/register/status", clients.RegistrationStatus)

	// TODO(#319): add per-route RBAC middleware — hub currently has zero permission checks.
	// All authenticated users can access every route. Requires hub role/permission tables first.
	r.Route("/api/v1", func(r chi.Router) {
		// General API rate limit: 100 requests/minute per IP.
		r.Use(ratelimit.Middleware(rateLimitStore, 100, 1*time.Minute, "api"))

		if jwtMW != nil {
			r.Use(jwtMW)
		}
		r.Use(tenant.Middleware)
		r.Use(idempotency.Middleware(idempotencyStore))

		// Settings
		settings := v1.NewSettingsHandler(queries, eventBus)
		r.Route("/settings", func(r chi.Router) {
			r.Get("/", settings.List)
			r.Put("/", settings.Upsert)
			r.Get("/{key}", settings.Get)
		})

		// Dashboard
		r.Get("/dashboard/stats", dashboard.Stats)
		r.Get("/dashboard/license-breakdown", dashboard.LicenseBreakdown)
		r.Get("/dashboard/catalog-growth", dashboard.CatalogGrowth)
		r.Get("/dashboard/clients", dashboard.ClientSummary)
		r.Get("/dashboard/activity", dashboard.Activity)

		// Clients (admin)
		r.Route("/clients", func(r chi.Router) {
			r.Get("/", clients.List)
			r.Get("/pending-count", clients.PendingCount) // MUST be before /{id}
			r.Get("/{id}", clients.Get)
			r.Put("/{id}", clients.Update)
			r.Post("/{id}/approve", clients.Approve)
			r.Post("/{id}/decline", clients.Decline)
			r.Post("/{id}/suspend", clients.Suspend)
			r.Get("/{id}/sync-history", clients.SyncHistory)
			r.Get("/{id}/endpoint-trend", clients.EndpointTrend)
			r.Delete("/{id}", clients.Delete)
		})

		// Licenses
		r.Route("/licenses", func(r chi.Router) {
			r.Get("/", licenses.List)
			r.Post("/", licenses.Create)
			r.Get("/{id}", licenses.Get)
			r.Post("/{id}/revoke", licenses.Revoke)
			r.Post("/{id}/assign", licenses.Assign)
			r.Put("/{id}/renew", licenses.Renew)
			r.Get("/{id}/usage-history", licenses.UsageHistory)
			r.Get("/{id}/audit-trail", licenses.AuditTrail)
		})

		// Feeds
		r.Route("/feeds", func(r chi.Router) {
			r.Get("/", feeds.List)
			r.Get("/{id}", feeds.Get)
			r.Put("/{id}", feeds.Update)
			r.Post("/{id}/sync", feeds.TriggerSync)
			r.Get("/{id}/history", feeds.History)
		})

		// Catalog
		r.Get("/catalog/stats", catalog.Stats)
		r.Get("/catalog", catalog.List)
		r.Post("/catalog", catalog.Create)
		r.Get("/catalog/{id}", catalog.Get)
		r.Put("/catalog/{id}", catalog.Update)
		r.Delete("/catalog/{id}", catalog.Delete)

		// Binary download for UI users (JWT-protected)
		if binaryStore != nil {
			catalogBinary := v1.NewCatalogBinaryHandler(queries, binaryStore, binaryBucket)
			r.Get("/catalog/{id}/download", catalogBinary.GetBinary)
		}

		// Package aliases
		aliases := v1.NewPackageAliasHandler(queries, eventBus)
		r.Route("/package-aliases", func(r chi.Router) {
			r.Get("/", aliases.List)
			r.Post("/", aliases.Create)
			r.Post("/discover", aliases.Discover)
			r.Delete("/{id}", aliases.Delete)
			r.Put("/{id}", aliases.Update)
		})
	})

	return r
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
