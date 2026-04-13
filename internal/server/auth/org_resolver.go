package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
)

// ErrOrgNotFound is returned by OrgResolver when no PatchIQ organization is
// bound to the given Zitadel org ID.
var ErrOrgNotFound = errors.New("auth: organization not found for Zitadel org ID")

// ErrNoAccessibleTenants is returned when the resolver cannot find any
// tenant accessible to the user in the given organization.
var ErrNoAccessibleTenants = errors.New("auth: user has no accessible tenants in organization")

// ErrTenantNotAccessible is returned when a client-supplied X-Tenant-ID
// does not belong to the user's accessible tenant set for the active org.
var ErrTenantNotAccessible = errors.New("auth: requested tenant is not accessible")

// OrgResolver maps a Zitadel org claim to a PatchIQ organization and picks
// an active tenant for a user in that organization. Used by the JWT middleware
// to bridge IdP-level identity to the organization/tenant hierarchy.
//
// Implementations must be safe for concurrent use across requests and should
// use the RLS-bypass data path (user_roles and roles are RLS-protected).
type OrgResolver interface {
	// ResolveZitadelOrg looks up a PatchIQ organization by its bound Zitadel
	// org ID. Returns ErrOrgNotFound when no mapping exists.
	ResolveZitadelOrg(ctx context.Context, zitadelOrgID string) (orgID string, err error)

	// ResolveActiveTenant returns the tenant ID the user should operate under.
	// If preferred is non-empty and refers to an accessible tenant, it is used
	// (supports the X-Tenant-ID override and MSP tenant switcher). Otherwise
	// the first accessible tenant is chosen. Returns ErrNoAccessibleTenants
	// when the user has no access to any tenant in the org, or
	// ErrTenantNotAccessible when preferred is set but not in the user's
	// accessible set.
	ResolveActiveTenant(ctx context.Context, orgID, userID, preferred string) (tenantID string, err error)
}

// storeOrgResolver implements OrgResolver on top of *store.Store.
type storeOrgResolver struct {
	orgLookup      OrgLookupBySource
	tenantsForUser TenantsForUserSource
}

// OrgLookupBySource resolves PatchIQ organization IDs from Zitadel org IDs.
// Implemented by *store.Store via sqlcgen.
type OrgLookupBySource interface {
	GetOrganizationByZitadelOrgID(ctx context.Context, zitadelOrgID string) (sqlcgen.Organization, error)
}

// TenantsForUserSource lists tenants in an org that the user can access.
// Implemented by *store.Store (UserAccessibleTenants, which uses BypassPool).
type TenantsForUserSource interface {
	UserAccessibleTenants(ctx context.Context, orgID, userID string) ([]sqlcgen.Tenant, error)
}

// NewOrgResolver creates a resolver backed by the store layer.
func NewOrgResolver(orgLookup OrgLookupBySource, tenantsForUser TenantsForUserSource) OrgResolver {
	if orgLookup == nil {
		panic("auth: NewOrgResolver called with nil org lookup")
	}
	if tenantsForUser == nil {
		panic("auth: NewOrgResolver called with nil tenants source")
	}
	return &storeOrgResolver{orgLookup: orgLookup, tenantsForUser: tenantsForUser}
}

func (r *storeOrgResolver) ResolveZitadelOrg(ctx context.Context, zitadelOrgID string) (string, error) {
	if zitadelOrgID == "" {
		return "", ErrOrgNotFound
	}
	org, err := r.orgLookup.GetOrganizationByZitadelOrgID(ctx, zitadelOrgID)
	if err != nil {
		// sqlc returns pgx.ErrNoRows for missing rows, but the caller shouldn't
		// need to import pgx. Treat any error as not-found for the purposes of
		// the auth flow; the middleware will log and fall back to defaults.
		return "", fmt.Errorf("resolve zitadel org %q: %w", zitadelOrgID, ErrOrgNotFound)
	}
	if !org.ID.Valid {
		return "", ErrOrgNotFound
	}
	return uuid.UUID(org.ID.Bytes).String(), nil
}

func (r *storeOrgResolver) ResolveActiveTenant(ctx context.Context, orgID, userID, preferred string) (string, error) {
	if userID == "" {
		return "", fmt.Errorf("resolve active tenant: empty user ID")
	}
	tenants, err := r.tenantsForUser.UserAccessibleTenants(ctx, orgID, userID)
	if err != nil {
		return "", fmt.Errorf("resolve active tenant: %w", err)
	}
	if len(tenants) == 0 {
		return "", ErrNoAccessibleTenants
	}
	// If a preferred tenant was specified (X-Tenant-ID header or session
	// claim), verify it is in the accessible set. Reject otherwise.
	if preferred != "" {
		for _, t := range tenants {
			if uuid.UUID(t.ID.Bytes).String() == preferred {
				return preferred, nil
			}
		}
		return "", ErrTenantNotAccessible
	}
	// No preference: use the first accessible tenant (stable ordering via
	// created_at from the sqlc query).
	first := tenants[0]
	if !first.ID.Valid {
		return "", ErrNoAccessibleTenants
	}
	return uuid.UUID(first.ID.Bytes).String(), nil
}

// Compile-time check: storeOrgResolver implements OrgResolver.
var _ OrgResolver = (*storeOrgResolver)(nil)

// Compile-time helper: use pgtype.UUID to silence unused-import if the
// generated sqlcgen helper signatures shift. Not exported.
var _ = pgtype.UUID{}
