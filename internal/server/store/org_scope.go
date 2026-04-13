package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// GetOrganizationByIDForMe is a narrow accessor used by the SSO /auth/me
// handler to populate the organization block of the auth response. It returns
// the name, slug, and type, matching the signature of auth.OrgScopeLookup.
func (s *Store) GetOrganizationByIDForMe(ctx context.Context, orgID string) (string, string, string, error) {
	parsed, err := parsePgUUID(orgID)
	if err != nil {
		return "", "", "", fmt.Errorf("get org by id for me: %w", err)
	}
	q := sqlcgen.New(s.Pool())
	org, err := q.GetOrganizationByID(ctx, parsed)
	if err != nil {
		return "", "", "", fmt.Errorf("get org by id for me: %w", err)
	}
	return org.Name, org.Slug, org.Type, nil
}

// GetOrganizationByZitadelOrgID resolves a PatchIQ organization from its
// bound Zitadel org ID. Returns pgx.ErrNoRows when no mapping exists.
// Uses the regular pool (organizations is not RLS-protected).
func (s *Store) GetOrganizationByZitadelOrgID(ctx context.Context, zitadelOrgID string) (sqlcgen.Organization, error) {
	q := sqlcgen.New(s.Pool())
	return q.GetOrganizationByZitadelOrgID(ctx, pgtype.Text{String: zitadelOrgID, Valid: zitadelOrgID != ""})
}

// UserAccessibleTenants returns the set of tenants within the given organization
// that the user can access, either via an org-scoped role grant (which grants
// access to all tenants in the org) or via a tenant-scoped role grant (which
// grants access only to that specific tenant).
//
// This query runs on the bypass pool because user_roles is RLS-protected and
// we need to evaluate membership across every tenant in the org without
// switching tenant context N times.
func (s *Store) UserAccessibleTenants(ctx context.Context, orgID, userID string) ([]sqlcgen.Tenant, error) {
	if userID == "" {
		return nil, fmt.Errorf("user accessible tenants: empty user ID")
	}
	orgUUID, err := parsePgUUID(orgID)
	if err != nil {
		return nil, fmt.Errorf("user accessible tenants: parse org ID: %w", err)
	}
	q := sqlcgen.New(s.BypassPool())
	tenants, err := q.ListUserAccessibleTenants(ctx, sqlcgen.ListUserAccessibleTenantsParams{
		OrganizationID: orgUUID,
		UserID:         userID,
	})
	if err != nil {
		return nil, fmt.Errorf("user accessible tenants: list: %w", err)
	}
	return tenants, nil
}

// GetUserOrgPermissions returns the permissions the user holds via org-scoped
// role assignments in the given organization. Runs on the bypass pool because
// the query joins role_permissions (RLS-protected) with roles defined in the
// org's platform tenant — a tenant that is typically not the active tenant
// context of the calling request.
func (s *Store) GetUserOrgPermissions(ctx context.Context, orgID, userID string) ([]sqlcgen.GetUserOrgPermissionsRow, error) {
	if userID == "" {
		return nil, fmt.Errorf("user org permissions: empty user ID")
	}
	orgUUID, err := parsePgUUID(orgID)
	if err != nil {
		return nil, fmt.Errorf("user org permissions: parse org ID: %w", err)
	}
	q := sqlcgen.New(s.BypassPool())
	perms, err := q.GetUserOrgPermissions(ctx, sqlcgen.GetUserOrgPermissionsParams{
		OrganizationID: orgUUID,
		UserID:         userID,
	})
	if err != nil {
		return nil, fmt.Errorf("user org permissions: load: %w", err)
	}
	return perms, nil
}

// ForEachTenantInOrg iterates the tenants the user has access to within the
// given organization and invokes fn for each one in its own tenant-scoped
// transaction. fn receives a derived context whose tenant ID is set to the
// current iteration's tenant, so any sqlcgen query inside fn will see rows
// from that tenant only (RLS-enforced).
//
// The iteration is sequential and short-circuits on first error. The caller's
// transaction context is NOT propagated — each iteration opens a fresh tx via
// BeginTx, which is committed (or rolled back) before moving to the next tenant.
// This guarantees that partial failures in one tenant do not leak state into
// the next.
//
// Typical use: MSP dashboard aggregations that need to sum counts across child
// tenants. Do NOT use for writes that must span tenants atomically — there is
// no such guarantee, and that pattern is disallowed by design (ADR-025).
func (s *Store) ForEachTenantInOrg(
	ctx context.Context,
	orgID, userID string,
	fn func(ctx context.Context, t sqlcgen.Tenant) error,
) error {
	tenants, err := s.UserAccessibleTenants(ctx, orgID, userID)
	if err != nil {
		return err
	}
	for _, t := range tenants {
		tenantID := uuidToString(t.ID)
		if tenantID == "" {
			return fmt.Errorf("for each tenant: null tenant ID in org %s", orgID)
		}
		tenantCtx := tenant.WithTenantID(ctx, tenantID)

		tx, err := s.BeginTx(tenantCtx)
		if err != nil {
			return fmt.Errorf("for each tenant: begin tx for %s: %w", tenantID, err)
		}

		if err := fn(tenantCtx, t); err != nil {
			if rbErr := tx.Rollback(tenantCtx); rbErr != nil {
				return fmt.Errorf("for each tenant: fn failed for %s: %w (rollback also failed: %v)", tenantID, err, rbErr)
			}
			return fmt.Errorf("for each tenant: fn failed for %s: %w", tenantID, err)
		}
		if err := tx.Commit(tenantCtx); err != nil {
			return fmt.Errorf("for each tenant: commit for %s: %w", tenantID, err)
		}
	}
	return nil
}

// parsePgUUID converts a UUID string into the pgtype.UUID shape used by sqlcgen.
func parsePgUUID(s string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(s)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("invalid UUID %q: %w", s, err)
	}
	var out pgtype.UUID
	copy(out.Bytes[:], parsed[:])
	out.Valid = true
	return out, nil
}

// uuidToString returns the canonical string form of a pgtype.UUID, or "" if
// the value is not set.
func uuidToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return uuid.UUID(u.Bytes).String()
}
