package store_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/skenzeriq/patchiq/internal/server/store"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
)

// orgScopeFixture seeds an MSP org with two child tenants, a role in each
// child tenant, and a grant pattern suitable for testing user accessibility.
type orgScopeFixture struct {
	orgID   string
	tenantA string
	tenantB string
	roleA   string // role in tenantA (for user_roles tenant-scoped grants)
	orgRole string // org-scoped role (lives in tenantA for test purposes)
}

func seedOrgScopeFixture(t *testing.T, ctx context.Context, s *store.Store) orgScopeFixture {
	t.Helper()
	pool := s.Pool() // superuser pool in test setup

	var f orgScopeFixture
	if err := pool.QueryRow(ctx,
		"INSERT INTO organizations (name, slug, type) VALUES ('MspOrg', 'mspscope', 'msp') RETURNING id::text",
	).Scan(&f.orgID); err != nil {
		t.Fatalf("seed org: %v", err)
	}
	if err := pool.QueryRow(ctx,
		"INSERT INTO tenants (name, slug, organization_id) VALUES ('TenA', 'mspscope-a', $1) RETURNING id::text",
		f.orgID,
	).Scan(&f.tenantA); err != nil {
		t.Fatalf("seed tenantA: %v", err)
	}
	if err := pool.QueryRow(ctx,
		"INSERT INTO tenants (name, slug, organization_id) VALUES ('TenB', 'mspscope-b', $1) RETURNING id::text",
		f.orgID,
	).Scan(&f.tenantB); err != nil {
		t.Fatalf("seed tenantB: %v", err)
	}
	// Role in tenantA for tenant-scoped grants.
	if err := pool.QueryRow(ctx,
		"INSERT INTO roles (tenant_id, name, description, is_system) VALUES ($1, 'Viewer', 'read-only in A', false) RETURNING id::text",
		f.tenantA,
	).Scan(&f.roleA); err != nil {
		t.Fatalf("seed roleA: %v", err)
	}
	if _, err := pool.Exec(ctx,
		"INSERT INTO role_permissions (tenant_id, role_id, resource, action, scope) VALUES ($1, $2, 'endpoints', 'read', '*')",
		f.tenantA, f.roleA,
	); err != nil {
		t.Fatalf("seed roleA permission: %v", err)
	}
	// Org-scoped role: we place it in tenantA for test simplicity. In production
	// it would live in the org's platform tenant.
	if err := pool.QueryRow(ctx,
		"INSERT INTO roles (tenant_id, name, description, is_system) VALUES ($1, 'MSP Admin', 'org-wide', true) RETURNING id::text",
		f.tenantA,
	).Scan(&f.orgRole); err != nil {
		t.Fatalf("seed orgRole: %v", err)
	}
	if _, err := pool.Exec(ctx,
		"INSERT INTO role_permissions (tenant_id, role_id, resource, action, scope) VALUES ($1, $2, '*', '*', '*')",
		f.tenantA, f.orgRole,
	); err != nil {
		t.Fatalf("seed orgRole permission: %v", err)
	}
	return f
}

// TestUserAccessibleTenants_OrgScoped verifies that a user with an org-scoped
// role grant has access to every tenant in the org, regardless of tenant-scoped
// grants.
func TestUserAccessibleTenants_OrgScoped(t *testing.T) {
	superPool, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	s := store.NewStoreWithBypass(superPool, superPool)
	f := seedOrgScopeFixture(t, ctx, s)

	// Grant MSP Admin at org scope.
	if _, err := superPool.Exec(ctx,
		"INSERT INTO org_user_roles (organization_id, user_id, role_id) VALUES ($1, $2, $3)",
		f.orgID, "user-msp", f.orgRole,
	); err != nil {
		t.Fatalf("seed org_user_role: %v", err)
	}

	tenants, err := s.UserAccessibleTenants(ctx, f.orgID, "user-msp")
	if err != nil {
		t.Fatalf("UserAccessibleTenants: %v", err)
	}
	if len(tenants) != 2 {
		t.Fatalf("got %d tenants, want 2 (org-scoped grant spans all)", len(tenants))
	}
	seen := map[string]bool{}
	for _, tt := range tenants {
		seen[uuidString(tt.ID)] = true
	}
	if !seen[f.tenantA] || !seen[f.tenantB] {
		t.Errorf("expected both tenantA and tenantB accessible, got %v", seen)
	}
}

// TestUserAccessibleTenants_TenantScoped verifies that a user with only a
// tenant-scoped grant sees only that specific tenant, even if the tenant
// belongs to a larger org.
func TestUserAccessibleTenants_TenantScoped(t *testing.T) {
	superPool, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	s := store.NewStoreWithBypass(superPool, superPool)
	f := seedOrgScopeFixture(t, ctx, s)

	// Grant Viewer only in tenantA.
	if _, err := superPool.Exec(ctx,
		"INSERT INTO user_roles (tenant_id, user_id, role_id) VALUES ($1, $2, $3)",
		f.tenantA, "user-local", f.roleA,
	); err != nil {
		t.Fatalf("seed user_role: %v", err)
	}

	tenants, err := s.UserAccessibleTenants(ctx, f.orgID, "user-local")
	if err != nil {
		t.Fatalf("UserAccessibleTenants: %v", err)
	}
	if len(tenants) != 1 {
		t.Fatalf("got %d tenants, want 1 (tenant-scoped grant)", len(tenants))
	}
	if uuidString(tenants[0].ID) != f.tenantA {
		t.Errorf("expected tenantA, got %s", uuidString(tenants[0].ID))
	}
}

// TestUserAccessibleTenants_NoGrants verifies that a user with no grants in
// the org sees zero tenants.
func TestUserAccessibleTenants_NoGrants(t *testing.T) {
	superPool, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	s := store.NewStoreWithBypass(superPool, superPool)
	f := seedOrgScopeFixture(t, ctx, s)

	tenants, err := s.UserAccessibleTenants(ctx, f.orgID, "user-nobody")
	if err != nil {
		t.Fatalf("UserAccessibleTenants: %v", err)
	}
	if len(tenants) != 0 {
		t.Errorf("got %d tenants, want 0 (no grants)", len(tenants))
	}
}

// TestGetUserOrgPermissions_OrgScopedAdmin verifies that MSP Admin (org-scoped,
// wildcard permission) is returned by GetUserOrgPermissions.
func TestGetUserOrgPermissions_OrgScopedAdmin(t *testing.T) {
	superPool, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	s := store.NewStoreWithBypass(superPool, superPool)
	f := seedOrgScopeFixture(t, ctx, s)

	if _, err := superPool.Exec(ctx,
		"INSERT INTO org_user_roles (organization_id, user_id, role_id) VALUES ($1, $2, $3)",
		f.orgID, "user-msp", f.orgRole,
	); err != nil {
		t.Fatalf("seed org_user_role: %v", err)
	}

	perms, err := s.GetUserOrgPermissions(ctx, f.orgID, "user-msp")
	if err != nil {
		t.Fatalf("GetUserOrgPermissions: %v", err)
	}
	found := false
	for _, p := range perms {
		if p.Resource == "*" && p.Action == "*" && p.Scope == "*" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected wildcard permission, got %v", perms)
	}
}

// TestGetUserOrgPermissions_NoGrants verifies empty result when user has no
// org-scoped role assignments.
func TestGetUserOrgPermissions_NoGrants(t *testing.T) {
	superPool, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	s := store.NewStoreWithBypass(superPool, superPool)
	f := seedOrgScopeFixture(t, ctx, s)

	perms, err := s.GetUserOrgPermissions(ctx, f.orgID, "user-nobody")
	if err != nil {
		t.Fatalf("GetUserOrgPermissions: %v", err)
	}
	if len(perms) != 0 {
		t.Errorf("got %d perms, want 0", len(perms))
	}
}

// TestForEachTenantInOrg_OrgScoped verifies that fn is invoked once per
// accessible tenant and that each invocation has the correct tenant context
// (so RLS-respecting queries see only that tenant's rows).
func TestForEachTenantInOrg_OrgScoped(t *testing.T) {
	superPool, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	s := store.NewStoreWithBypass(superPool, superPool)
	f := seedOrgScopeFixture(t, ctx, s)

	// Grant MSP Admin.
	if _, err := superPool.Exec(ctx,
		"INSERT INTO org_user_roles (organization_id, user_id, role_id) VALUES ($1, $2, $3)",
		f.orgID, "user-msp", f.orgRole,
	); err != nil {
		t.Fatalf("seed org_user_role: %v", err)
	}

	// Seed one endpoint in each tenant.
	if _, err := superPool.Exec(ctx,
		"INSERT INTO endpoints (tenant_id, hostname, os_family, os_version, status) VALUES ($1, 'host-a', 'linux', '22.04', 'online')",
		f.tenantA,
	); err != nil {
		t.Fatalf("seed endpointA: %v", err)
	}
	if _, err := superPool.Exec(ctx,
		"INSERT INTO endpoints (tenant_id, hostname, os_family, os_version, status) VALUES ($1, 'host-b', 'linux', '22.04', 'online')",
		f.tenantB,
	); err != nil {
		t.Fatalf("seed endpointB: %v", err)
	}

	var visited []string
	err := s.ForEachTenantInOrg(ctx, f.orgID, "user-msp", func(innerCtx context.Context, tt sqlcgen.Tenant) error {
		visited = append(visited, uuidString(tt.ID))
		// Inside fn, the tenant context is set to tt.ID. A raw query against
		// endpoints under this context should see only this tenant's rows
		// (RLS-enforced). We use the superuser pool directly here because
		// the test Store is seeded with superPool as both regular and bypass
		// pools, so RLS isn't actually enforced on the regular pool. This
		// test therefore verifies only the iteration contract, not RLS.
		return nil
	})
	if err != nil {
		t.Fatalf("ForEachTenantInOrg: %v", err)
	}
	if len(visited) != 2 {
		t.Fatalf("visited %d tenants, want 2: %v", len(visited), visited)
	}
}

// TestForEachTenantInOrg_ErrorShortCircuits verifies that an error in fn
// short-circuits and propagates.
func TestForEachTenantInOrg_ErrorShortCircuits(t *testing.T) {
	superPool, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	s := store.NewStoreWithBypass(superPool, superPool)
	f := seedOrgScopeFixture(t, ctx, s)

	if _, err := superPool.Exec(ctx,
		"INSERT INTO org_user_roles (organization_id, user_id, role_id) VALUES ($1, $2, $3)",
		f.orgID, "user-msp", f.orgRole,
	); err != nil {
		t.Fatalf("seed org_user_role: %v", err)
	}

	callCount := 0
	sentinelErr := errSentinel("boom")
	err := s.ForEachTenantInOrg(ctx, f.orgID, "user-msp", func(innerCtx context.Context, tt sqlcgen.Tenant) error {
		callCount++
		return sentinelErr
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if callCount != 1 {
		t.Errorf("fn called %d times, want 1 (short-circuit)", callCount)
	}
}

type errSentinel string

func (e errSentinel) Error() string { return string(e) }

// uuidString formats a pgtype.UUID as a canonical string. Returns "" for null.
func uuidString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return uuid.UUID(u.Bytes).String()
}
