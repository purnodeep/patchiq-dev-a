package store_test

import (
	"context"
	"testing"
)

// TestMigration059_OrganizationsBackfill verifies that migration 059:
//   - creates the organizations table
//   - backfills every existing tenant into its own direct-type organization
//   - sets tenants.organization_id for each existing row
//   - creates the org_user_roles table
//
// The default tenant seeded by migration 001 is the invariant we pin to:
// after migrations run, it must have an organization_id pointing at an
// organization named "Default" with slug "default" and type "direct".
func TestMigration059_OrganizationsBackfill(t *testing.T) {
	superPool, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Default tenant must be linked to a default organization.
	var (
		orgID   string
		orgName string
		orgSlug string
		orgType string
	)
	err := superPool.QueryRow(ctx, `
		SELECT o.id::text, o.name, o.slug, o.type
		FROM tenants t
		JOIN organizations o ON o.id = t.organization_id
		WHERE t.id = $1
	`, defaultTenant).Scan(&orgID, &orgName, &orgSlug, &orgType)
	if err != nil {
		t.Fatalf("query default tenant's organization: %v", err)
	}
	if orgName != "Default" {
		t.Errorf("org name = %q, want %q", orgName, "Default")
	}
	if orgSlug != "default" {
		t.Errorf("org slug = %q, want %q", orgSlug, "default")
	}
	if orgType != "direct" {
		t.Errorf("org type = %q, want %q", orgType, "direct")
	}

	// Every tenant should have an organization_id after backfill.
	var unlinked int
	if err := superPool.QueryRow(ctx,
		"SELECT COUNT(*) FROM tenants WHERE organization_id IS NULL",
	).Scan(&unlinked); err != nil {
		t.Fatalf("count unlinked tenants: %v", err)
	}
	if unlinked != 0 {
		t.Errorf("unlinked tenants = %d, want 0", unlinked)
	}

	// org_user_roles table exists and is empty.
	var orgRolesCount int
	if err := superPool.QueryRow(ctx, "SELECT COUNT(*) FROM org_user_roles").Scan(&orgRolesCount); err != nil {
		t.Fatalf("count org_user_roles: %v", err)
	}
	if orgRolesCount != 0 {
		t.Errorf("org_user_roles count = %d, want 0 (table should be empty after migration)", orgRolesCount)
	}
}

// TestMigration059_OrganizationsConstraints verifies CHECK constraints on
// organizations.type and the partial unique index on zitadel_org_id.
func TestMigration059_OrganizationsConstraints(t *testing.T) {
	superPool, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Invalid type should fail the CHECK constraint.
	_, err := superPool.Exec(ctx,
		"INSERT INTO organizations (name, slug, type) VALUES ('Bad', 'bad-org', 'invalid')",
	)
	if err == nil {
		t.Error("expected CHECK constraint violation for type='invalid', got nil")
	}

	// Empty name should fail the CHECK constraint.
	_, err = superPool.Exec(ctx,
		"INSERT INTO organizations (name, slug, type) VALUES ('', 'empty-name-org', 'direct')",
	)
	if err == nil {
		t.Error("expected CHECK constraint violation for empty name, got nil")
	}

	// Two orgs sharing a non-null zitadel_org_id should fail the partial unique index.
	if _, err := superPool.Exec(ctx,
		"INSERT INTO organizations (name, slug, type, zitadel_org_id) VALUES ('Org1', 'org1-zcollide', 'msp', 'zitadel-123')",
	); err != nil {
		t.Fatalf("first zitadel-bound org insert: %v", err)
	}
	if _, err := superPool.Exec(ctx,
		"INSERT INTO organizations (name, slug, type, zitadel_org_id) VALUES ('Org2', 'org2-zcollide', 'msp', 'zitadel-123')",
	); err == nil {
		t.Error("expected unique violation for duplicate zitadel_org_id, got nil")
	}

	// Two orgs with NULL zitadel_org_id should coexist (partial unique index).
	if _, err := superPool.Exec(ctx,
		"INSERT INTO organizations (name, slug, type) VALUES ('OrgNull1', 'null-org-1', 'direct')",
	); err != nil {
		t.Fatalf("first null-zitadel org insert: %v", err)
	}
	if _, err := superPool.Exec(ctx,
		"INSERT INTO organizations (name, slug, type) VALUES ('OrgNull2', 'null-org-2', 'direct')",
	); err != nil {
		t.Errorf("second null-zitadel org should succeed, got: %v", err)
	}
}

// TestMigration059_OrgUserRolesFK verifies that org_user_roles has
// CASCADE delete on both organization_id and role_id.
func TestMigration059_OrgUserRolesFK(t *testing.T) {
	superPool, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Create an org.
	var orgID string
	if err := superPool.QueryRow(ctx,
		"INSERT INTO organizations (name, slug, type) VALUES ('TestOrg', 'test-org-fk', 'msp') RETURNING id::text",
	).Scan(&orgID); err != nil {
		t.Fatalf("create org: %v", err)
	}

	// Create a role under the default tenant (roles is tenant-scoped).
	var roleID string
	if err := superPool.QueryRow(ctx,
		"INSERT INTO roles (tenant_id, name, description, is_system) VALUES ($1, 'TestOrgRole', 'test', false) RETURNING id::text",
		defaultTenant,
	).Scan(&roleID); err != nil {
		t.Fatalf("create role: %v", err)
	}

	// Grant the role at org scope.
	if _, err := superPool.Exec(ctx,
		"INSERT INTO org_user_roles (organization_id, user_id, role_id) VALUES ($1, $2, $3)",
		orgID, "user-1", roleID,
	); err != nil {
		t.Fatalf("create org_user_role grant: %v", err)
	}

	// Deleting the role should cascade-delete the grant.
	if _, err := superPool.Exec(ctx, "DELETE FROM roles WHERE id = $1", roleID); err != nil {
		t.Fatalf("delete role: %v", err)
	}
	var remaining int
	if err := superPool.QueryRow(ctx,
		"SELECT COUNT(*) FROM org_user_roles WHERE organization_id = $1",
		orgID,
	).Scan(&remaining); err != nil {
		t.Fatalf("count org_user_roles: %v", err)
	}
	// We inserted the grant with orgID + the now-deleted role; cascade on
	// role_id should remove it. But our SELECT filters by organization_id, so
	// remaining == 0 proves the cascade worked.
	if remaining != 0 {
		t.Errorf("remaining grants after role delete = %d, want 0 (CASCADE on role_id)", remaining)
	}

	// Now test cascade on organization_id.
	// Re-create role, re-grant, then delete the org.
	if err := superPool.QueryRow(ctx,
		"INSERT INTO roles (tenant_id, name, description, is_system) VALUES ($1, 'TestOrgRole2', 'test', false) RETURNING id::text",
		defaultTenant,
	).Scan(&roleID); err != nil {
		t.Fatalf("recreate role: %v", err)
	}
	if _, err := superPool.Exec(ctx,
		"INSERT INTO org_user_roles (organization_id, user_id, role_id) VALUES ($1, $2, $3)",
		orgID, "user-1", roleID,
	); err != nil {
		t.Fatalf("regrant: %v", err)
	}

	if _, err := superPool.Exec(ctx, "DELETE FROM organizations WHERE id = $1", orgID); err != nil {
		t.Fatalf("delete org: %v", err)
	}
	if err := superPool.QueryRow(ctx,
		"SELECT COUNT(*) FROM org_user_roles WHERE role_id = $1",
		roleID,
	).Scan(&remaining); err != nil {
		t.Fatalf("count org_user_roles after org delete: %v", err)
	}
	if remaining != 0 {
		t.Errorf("remaining grants after org delete = %d, want 0 (CASCADE on organization_id)", remaining)
	}
}

// TestMigration059_RLSRegression verifies that migration 059 did NOT weaken
// the tenant RLS boundary. With app.current_tenant_id=A set, the app role
// must not see any row from tenant B in the roles table, regardless of
// organization membership.
func TestMigration059_RLSRegression(t *testing.T) {
	superPool, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Create two orgs and two tenants linked to them.
	var orgA, orgB string
	if err := superPool.QueryRow(ctx,
		"INSERT INTO organizations (name, slug, type) VALUES ('OrgA', 'org-a-rls', 'direct') RETURNING id::text",
	).Scan(&orgA); err != nil {
		t.Fatalf("create orgA: %v", err)
	}
	if err := superPool.QueryRow(ctx,
		"INSERT INTO organizations (name, slug, type) VALUES ('OrgB', 'org-b-rls', 'direct') RETURNING id::text",
	).Scan(&orgB); err != nil {
		t.Fatalf("create orgB: %v", err)
	}

	var tenantA, tenantB string
	if err := superPool.QueryRow(ctx,
		"INSERT INTO tenants (name, slug, organization_id) VALUES ('TenA', 'ten-a-rls', $1) RETURNING id::text",
		orgA,
	).Scan(&tenantA); err != nil {
		t.Fatalf("create tenantA: %v", err)
	}
	if err := superPool.QueryRow(ctx,
		"INSERT INTO tenants (name, slug, organization_id) VALUES ('TenB', 'ten-b-rls', $1) RETURNING id::text",
		orgB,
	).Scan(&tenantB); err != nil {
		t.Fatalf("create tenantB: %v", err)
	}

	// Create a role in each tenant via superuser (bypasses RLS).
	if _, err := superPool.Exec(ctx,
		"INSERT INTO roles (tenant_id, name, is_system) VALUES ($1, 'RoleA', false), ($2, 'RoleB', false)",
		tenantA, tenantB,
	); err != nil {
		t.Fatalf("seed roles: %v", err)
	}

	// Query as the app role with tenantA context; must see only RoleA.
	app := appPool(t, superPool)
	defer app.Close()

	tx, err := app.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantA); err != nil {
		t.Fatalf("set tenant context: %v", err)
	}

	rows, err := tx.Query(ctx, "SELECT name FROM roles")
	if err != nil {
		t.Fatalf("query roles: %v", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatalf("scan role name: %v", err)
		}
		names = append(names, n)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows err: %v", err)
	}

	for _, n := range names {
		if n == "RoleB" {
			t.Errorf("RLS leak: saw RoleB under tenantA context")
		}
	}
}
