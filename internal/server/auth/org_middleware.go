package auth

import (
	"log/slog"
	"net/http"

	"github.com/skenzeriq/patchiq/internal/shared/organization"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"github.com/skenzeriq/patchiq/internal/shared/user"
)

// NewOrgScopeMiddleware returns HTTP middleware that refines the tenant
// context set by NewJWTMiddleware into an (organization, active tenant) pair,
// using the supplied resolver.
//
// It MUST run after NewJWTMiddleware so that user ID and a preliminary
// tenant/org claim are already in context. Behavior:
//
//  1. If the resolver is nil (single-tenant deployment), pass through
//     unchanged. The existing tenant context is preserved.
//  2. If no user ID is in context, pass through — downstream middleware
//     will reject the request.
//  3. Read the preliminary value that NewJWTMiddleware put into the tenant
//     context. This value is actually the Zitadel org ID claim (the JWT
//     middleware treats it as a tenant ID for backward compatibility).
//     Try to resolve it to a PatchIQ organization; on success, rewrite the
//     tenant context to the user's active tenant in that org.
//  4. On any resolver error, log and pass through — falling back to the
//     existing M0 behavior so single-tenant deployments and tests keep
//     working when the resolver has no matching org binding.
//
// The X-Tenant-ID header (when present and a valid UUID in the user's
// accessible set) overrides the default-tenant pick. This is how the
// frontend tenant switcher targets a specific client from an MSP operator's
// session.
func NewOrgScopeMiddleware(resolver OrgResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if resolver == nil {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			userID, ok := user.UserIDFromContext(ctx)
			if !ok || userID == "" {
				next.ServeHTTP(w, r)
				return
			}

			preliminary, ok := tenant.TenantIDFromContext(ctx)
			if !ok || preliminary == "" {
				next.ServeHTTP(w, r)
				return
			}

			orgID, err := resolver.ResolveZitadelOrg(ctx, preliminary)
			if err != nil {
				// No org mapping — fall back to legacy tenant-first behavior.
				// This is NOT a failure: single-tenant deployments do not
				// register org mappings, and the preliminary value may already
				// be a valid tenant UUID that the rest of the chain accepts.
				next.ServeHTTP(w, r)
				return
			}

			preferred := r.Header.Get(tenant.HeaderTenantID)
			activeTenant, err := resolver.ResolveActiveTenant(ctx, orgID, userID, preferred)
			if err != nil {
				slog.WarnContext(ctx, "org scope middleware: active tenant resolution failed",
					"user_id", userID,
					"org_id", orgID,
					"preferred_tenant", preferred,
					"error", err,
					"path", r.URL.Path,
				)
				writeAuthError(ctx, w, http.StatusForbidden, "no accessible tenant for this organization")
				return
			}

			ctx = organization.WithOrgID(ctx, orgID)
			ctx = tenant.WithTenantID(ctx, activeTenant)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
