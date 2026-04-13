package store

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/skenzeriq/patchiq/internal/server/auth"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
)

// BootstrapPlatformTenant creates the hidden "platform tenant" for an MSP or
// reseller organization and seeds it with the full preset role catalog (both
// tenant-scoped and org-scoped templates from auth.PresetRoles). The org's
// platform_tenant_id column is updated to point at the new tenant in the same
// transaction.
//
// All work runs against the bypass pool because the roles and role_permissions
// tables are RLS-protected and the calling request typically does not have
// the platform tenant set as its current tenant context. The bypass pool is
// expected to connect as a superuser (or otherwise non-RLS-restricted role).
//
// Returns the new platform tenant ID on success. Idempotency is NOT guaranteed:
// callers should only invoke this once per org, immediately after creating it.
// Re-running will fail on the unique constraint of tenants.slug.
func (s *Store) BootstrapPlatformTenant(ctx context.Context, orgID, orgName string) (string, error) {
	if orgID == "" {
		return "", fmt.Errorf("bootstrap platform tenant: empty org ID")
	}
	orgUUID, err := parsePgUUID(orgID)
	if err != nil {
		return "", fmt.Errorf("bootstrap platform tenant: parse org ID: %w", err)
	}

	pool := s.BypassPool()
	tx, err := pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("bootstrap platform tenant: begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Look up the org slug so we can derive a deterministic platform tenant slug.
	q := sqlcgen.New(tx)
	org, err := q.GetOrganizationByID(ctx, orgUUID)
	if err != nil {
		return "", fmt.Errorf("bootstrap platform tenant: load org: %w", err)
	}

	platformSlug := "platform-" + org.Slug
	platformName := orgName + " Platform"
	if orgName == "" {
		platformName = org.Name + " Platform"
	}

	// Create the platform tenant row.
	var (
		newTenantID pgtype.UUID
	)
	if err := tx.QueryRow(ctx,
		`INSERT INTO tenants (name, slug, organization_id)
		 VALUES ($1, $2, $3)
		 RETURNING id`,
		platformName, platformSlug, orgUUID,
	).Scan(&newTenantID); err != nil {
		return "", fmt.Errorf("bootstrap platform tenant: insert tenant: %w", err)
	}

	// Seed every preset role (both tenant-scoped and org-scoped) into the
	// platform tenant. Org-scoped role definitions MUST live somewhere, and
	// the platform tenant exists precisely for that purpose. Tenant-scoped
	// presets are also seeded so the org's MSP operators have a consistent
	// catalog should they ever need to reference them via this tenant.
	for _, tmpl := range auth.PresetRoles() {
		if err := seedPresetRoleInTx(ctx, tx, newTenantID, tmpl); err != nil {
			return "", fmt.Errorf("bootstrap platform tenant: seed role %q: %w", tmpl.Name, err)
		}
	}

	// Link the org to its new platform tenant.
	if err := q.SetOrganizationPlatformTenant(ctx, sqlcgen.SetOrganizationPlatformTenantParams{
		ID:               orgUUID,
		PlatformTenantID: newTenantID,
	}); err != nil {
		return "", fmt.Errorf("bootstrap platform tenant: link org: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("bootstrap platform tenant: commit: %w", err)
	}

	tenantIDStr := uuid.UUID(newTenantID.Bytes).String()
	slog.InfoContext(ctx, "bootstrapped platform tenant for organization",
		"org_id", orgID,
		"platform_tenant_id", tenantIDStr,
		"platform_tenant_slug", platformSlug,
	)
	return tenantIDStr, nil
}

// seedPresetRoleInTx inserts a single preset RoleTemplate (and its parsed
// permissions) into the given tenant within the supplied transaction. Each
// permission string is expected to be of the form "resource:action:scope" —
// any other shape returns an error.
func seedPresetRoleInTx(ctx context.Context, tx pgx.Tx, tenantID pgtype.UUID, tmpl auth.RoleTemplate) error {
	var roleID pgtype.UUID
	if err := tx.QueryRow(ctx,
		`INSERT INTO roles (tenant_id, name, description, is_system)
		 VALUES ($1, $2, $3, true)
		 RETURNING id`,
		tenantID, tmpl.Name, tmpl.Description,
	).Scan(&roleID); err != nil {
		return fmt.Errorf("insert role: %w", err)
	}
	for _, perm := range tmpl.Permissions {
		parts := strings.SplitN(perm, ":", 3)
		if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
			return fmt.Errorf("invalid permission %q: expected resource:action:scope", perm)
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO role_permissions (tenant_id, role_id, resource, action, scope)
			 VALUES ($1, $2, $3, $4, $5)
			 ON CONFLICT (tenant_id, role_id, resource, action, scope) DO NOTHING`,
			tenantID, roleID, parts[0], parts[1], parts[2],
		); err != nil {
			return fmt.Errorf("insert permission %q: %w", perm, err)
		}
	}
	return nil
}

// ListClientTenants returns the tenants under an organization that are
// visible to MSP operators (i.e. excluding the hidden platform tenant).
// Uses the regular pool because tenants is not RLS-protected.
func (s *Store) ListClientTenants(ctx context.Context, orgID string) ([]sqlcgen.Tenant, error) {
	orgUUID, err := parsePgUUID(orgID)
	if err != nil {
		return nil, fmt.Errorf("list client tenants: parse org ID: %w", err)
	}
	q := sqlcgen.New(s.Pool())
	tenants, err := q.ListClientTenantsByOrganization(ctx, orgUUID)
	if err != nil {
		return nil, fmt.Errorf("list client tenants: %w", err)
	}
	return tenants, nil
}
