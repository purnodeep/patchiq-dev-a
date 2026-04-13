package store_test

import (
	"context"
	"testing"

	"github.com/skenzeriq/patchiq/internal/server/store"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
)

// TestMSPFlow_EndToEnd walks the full MSP scenario:
//
//  1. Create an MSP organization.
//  2. Create two child tenants under it.
//  3. Seed a Viewer role in tenantA and an MSP Admin role in tenantA.
//  4. Grant MSP Admin at org scope to user-msp.
//  5. Grant Viewer at tenant scope in tenantA to user-local.
//  6. Verify:
//     - user-msp sees both child tenants (org-scoped grant spans all).
//     - user-msp holds the wildcard permission via GetUserOrgPermissions.
//     - ForEachTenantInOrg invokes fn exactly twice for user-msp.
//     - user-local sees only tenantA.
//     - user-nobody sees zero tenants.
//
// This is the contract test for the MSP data model. Regressions here mean
// the MSP story is broken.
func TestMSPFlow_EndToEnd(t *testing.T) {
	superPool, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	s := store.NewStoreWithBypass(superPool, superPool)

	var orgID string
	if err := superPool.QueryRow(ctx,
		"INSERT INTO organizations (name, slug, type) VALUES ('Acme MSP', 'acme-msp', 'msp') RETURNING id::text",
	).Scan(&orgID); err != nil {
		t.Fatalf("create MSP org: %v", err)
	}

	var tenantA, tenantB string
	if err := superPool.QueryRow(ctx,
		"INSERT INTO tenants (name, slug, organization_id) VALUES ('Client Alpha', 'msp-alpha', $1) RETURNING id::text",
		orgID,
	).Scan(&tenantA); err != nil {
		t.Fatalf("create tenantA: %v", err)
	}
	if err := superPool.QueryRow(ctx,
		"INSERT INTO tenants (name, slug, organization_id) VALUES ('Client Bravo', 'msp-bravo', $1) RETURNING id::text",
		orgID,
	).Scan(&tenantB); err != nil {
		t.Fatalf("create tenantB: %v", err)
	}

	var viewerRoleID string
	if err := superPool.QueryRow(ctx,
		"INSERT INTO roles (tenant_id, name, is_system) VALUES ($1, 'Viewer', false) RETURNING id::text",
		tenantA,
	).Scan(&viewerRoleID); err != nil {
		t.Fatalf("create viewer role: %v", err)
	}
	if _, err := superPool.Exec(ctx,
		"INSERT INTO role_permissions (tenant_id, role_id, resource, action, scope) VALUES ($1, $2, 'endpoints', 'read', '*')",
		tenantA, viewerRoleID,
	); err != nil {
		t.Fatalf("seed viewer permission: %v", err)
	}

	var mspAdminRoleID string
	if err := superPool.QueryRow(ctx,
		"INSERT INTO roles (tenant_id, name, is_system) VALUES ($1, 'MSP Admin', true) RETURNING id::text",
		tenantA,
	).Scan(&mspAdminRoleID); err != nil {
		t.Fatalf("create msp admin role: %v", err)
	}
	if _, err := superPool.Exec(ctx,
		"INSERT INTO role_permissions (tenant_id, role_id, resource, action, scope) VALUES ($1, $2, '*', '*', '*')",
		tenantA, mspAdminRoleID,
	); err != nil {
		t.Fatalf("seed msp admin permission: %v", err)
	}
	if _, err := superPool.Exec(ctx,
		"INSERT INTO org_user_roles (organization_id, user_id, role_id) VALUES ($1, 'user-msp', $2)",
		orgID, mspAdminRoleID,
	); err != nil {
		t.Fatalf("grant msp admin: %v", err)
	}

	mspTenants, err := s.UserAccessibleTenants(ctx, orgID, "user-msp")
	if err != nil {
		t.Fatalf("UserAccessibleTenants(user-msp): %v", err)
	}
	if len(mspTenants) != 2 {
		t.Errorf("user-msp should see 2 tenants, saw %d", len(mspTenants))
	}

	perms, err := s.GetUserOrgPermissions(ctx, orgID, "user-msp")
	if err != nil {
		t.Fatalf("GetUserOrgPermissions(user-msp): %v", err)
	}
	wildcardSeen := false
	for _, p := range perms {
		if p.Resource == "*" && p.Action == "*" && p.Scope == "*" {
			wildcardSeen = true
			break
		}
	}
	if !wildcardSeen {
		t.Errorf("user-msp should hold wildcard permission, perms=%v", perms)
	}

	var iterations int
	err = s.ForEachTenantInOrg(ctx, orgID, "user-msp", func(_ context.Context, _ sqlcgen.Tenant) error {
		iterations++
		return nil
	})
	if err != nil {
		t.Fatalf("ForEachTenantInOrg(user-msp): %v", err)
	}
	if iterations != 2 {
		t.Errorf("ForEachTenantInOrg called fn %d times, want 2", iterations)
	}

	if _, err := superPool.Exec(ctx,
		"INSERT INTO user_roles (tenant_id, user_id, role_id) VALUES ($1, 'user-local', $2)",
		tenantA, viewerRoleID,
	); err != nil {
		t.Fatalf("grant viewer to user-local: %v", err)
	}
	localTenants, err := s.UserAccessibleTenants(ctx, orgID, "user-local")
	if err != nil {
		t.Fatalf("UserAccessibleTenants(user-local): %v", err)
	}
	if len(localTenants) != 1 {
		t.Errorf("user-local should see 1 tenant, saw %d", len(localTenants))
	}
	if len(localTenants) == 1 && uuidString(localTenants[0].ID) != tenantA {
		t.Errorf("user-local should see tenantA, saw %s", uuidString(localTenants[0].ID))
	}

	noTenants, err := s.UserAccessibleTenants(ctx, orgID, "user-nobody")
	if err != nil {
		t.Fatalf("UserAccessibleTenants(user-nobody): %v", err)
	}
	if len(noTenants) != 0 {
		t.Errorf("user-nobody should see 0 tenants, saw %d", len(noTenants))
	}

	// Silence unused variable warning from tenantB — the variable is used to
	// populate the 2-tenant expectation above via org-scoped grants, but the
	// linter sees it as potentially unused after the accessibility check.
	_ = tenantB
}
