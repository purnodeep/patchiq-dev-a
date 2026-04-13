package store_test

import (
	"context"
	"sort"
	"strings"
	"testing"

	"github.com/skenzeriq/patchiq/internal/server/auth"
	"github.com/skenzeriq/patchiq/internal/server/store"
)

// TestBootstrapPlatformTenant_CreatesTenantAndRoles verifies that bootstrapping
// a platform tenant for an MSP org creates a tenant row, links it via
// organizations.platform_tenant_id, and seeds the full preset role catalog
// (6 tenant-scoped + 3 org-scoped = 9 roles) with the expected permission
// counts in the new tenant.
func TestBootstrapPlatformTenant_CreatesTenantAndRoles(t *testing.T) {
	superPool, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	s := store.NewStoreWithBypass(superPool, superPool)

	// Create the org row directly (mirroring what CreateOrganization does).
	var orgID string
	if err := superPool.QueryRow(ctx,
		"INSERT INTO organizations (name, slug, type) VALUES ('Acme MSP', 'acme-msp', 'msp') RETURNING id::text",
	).Scan(&orgID); err != nil {
		t.Fatalf("seed org: %v", err)
	}

	tenantID, err := s.BootstrapPlatformTenant(ctx, orgID, "Acme MSP")
	if err != nil {
		t.Fatalf("BootstrapPlatformTenant: %v", err)
	}
	if tenantID == "" {
		t.Fatal("expected non-empty tenant ID")
	}

	// Verify organizations.platform_tenant_id is set to the new tenant.
	var linked string
	if err := superPool.QueryRow(ctx,
		"SELECT platform_tenant_id::text FROM organizations WHERE id = $1",
		orgID,
	).Scan(&linked); err != nil {
		t.Fatalf("read platform_tenant_id: %v", err)
	}
	if linked != tenantID {
		t.Errorf("platform_tenant_id = %q, want %q", linked, tenantID)
	}

	// Verify the tenant exists, belongs to the org, and has the expected slug.
	var (
		tenantOrg  string
		tenantSlug string
		tenantName string
	)
	if err := superPool.QueryRow(ctx,
		"SELECT organization_id::text, slug, name FROM tenants WHERE id = $1",
		tenantID,
	).Scan(&tenantOrg, &tenantSlug, &tenantName); err != nil {
		t.Fatalf("read platform tenant: %v", err)
	}
	if tenantOrg != orgID {
		t.Errorf("tenant.organization_id = %q, want %q", tenantOrg, orgID)
	}
	if tenantSlug != "platform-acme-msp" {
		t.Errorf("tenant.slug = %q, want %q", tenantSlug, "platform-acme-msp")
	}
	if tenantName != "Acme MSP Platform" {
		t.Errorf("tenant.name = %q, want %q", tenantName, "Acme MSP Platform")
	}

	// Verify exactly 9 system roles exist in the platform tenant.
	var roleCount int
	if err := superPool.QueryRow(ctx,
		"SELECT count(*) FROM roles WHERE tenant_id = $1 AND is_system = true",
		tenantID,
	).Scan(&roleCount); err != nil {
		t.Fatalf("count roles: %v", err)
	}
	expectedRoles := len(auth.PresetRoles())
	if roleCount != expectedRoles {
		t.Errorf("role count = %d, want %d", roleCount, expectedRoles)
	}
	if expectedRoles != 9 {
		t.Errorf("preset role catalog drifted: PresetRoles() returned %d, expected 9 (6 tenant + 3 org)", expectedRoles)
	}

	// Verify each preset has the EXACT permission tuples from its template,
	// not just the same count. A regression in seedPresetRoleInTx that
	// inserts wrong-but-same-count rows (e.g. swapped scope, truncated parts)
	// would silently grant the wrong privileges — security-relevant.
	for _, tmpl := range auth.PresetRoles() {
		rows, err := superPool.Query(ctx, `
			SELECT rp.resource, rp.action, rp.scope
			  FROM role_permissions rp
			  JOIN roles r ON r.id = rp.role_id
			 WHERE r.tenant_id = $1 AND r.name = $2
		`, tenantID, tmpl.Name)
		if err != nil {
			t.Fatalf("query permissions for %q: %v", tmpl.Name, err)
		}
		var got []string
		for rows.Next() {
			var resource, action, scope string
			if err := rows.Scan(&resource, &action, &scope); err != nil {
				rows.Close()
				t.Fatalf("scan permission for %q: %v", tmpl.Name, err)
			}
			got = append(got, resource+":"+action+":"+scope)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			t.Fatalf("iterate permissions for %q: %v", tmpl.Name, err)
		}

		want := append([]string(nil), tmpl.Permissions...)
		sort.Strings(want)
		sort.Strings(got)

		if len(got) != len(want) {
			t.Errorf("role %q: permission count = %d, want %d (got=%v want=%v)",
				tmpl.Name, len(got), len(want), got, want)
			continue
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("role %q: permission tuple mismatch\n  got:  %s\n  want: %s\n  full got=%s\n  full want=%s",
					tmpl.Name, got[i], want[i],
					strings.Join(got, ","), strings.Join(want, ","))
				break
			}
		}
	}
}

// TestBootstrapPlatformTenant_AtomicOnConflict verifies that when the
// platform tenant slug collides with an existing tenant, the entire bootstrap
// transaction rolls back: no roles are leaked into the partially-created
// tenant, and the org's platform_tenant_id stays NULL. This is the failure
// mode the handler-level rollback exists to recover from — if the store
// transaction is not atomic, the handler can't put the world back together.
func TestBootstrapPlatformTenant_AtomicOnConflict(t *testing.T) {
	superPool, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	s := store.NewStoreWithBypass(superPool, superPool)

	var orgID string
	if err := superPool.QueryRow(ctx,
		"INSERT INTO organizations (name, slug, type) VALUES ('Initech', 'initech', 'msp') RETURNING id::text",
	).Scan(&orgID); err != nil {
		t.Fatalf("seed org: %v", err)
	}

	// Squat on the slug that BootstrapPlatformTenant will derive
	// ("platform-" + org.Slug) so the INSERT inside the bootstrap tx fails.
	if _, err := superPool.Exec(ctx,
		"INSERT INTO tenants (name, slug, organization_id) VALUES ('Squatter', 'platform-initech', $1)",
		orgID,
	); err != nil {
		t.Fatalf("seed conflicting tenant: %v", err)
	}

	if _, err := s.BootstrapPlatformTenant(ctx, orgID, "Initech"); err == nil {
		t.Fatal("expected bootstrap to fail on slug conflict, got nil")
	}

	// Atomicity: org.platform_tenant_id must still be NULL.
	var linked *string
	if err := superPool.QueryRow(ctx,
		"SELECT platform_tenant_id::text FROM organizations WHERE id = $1",
		orgID,
	).Scan(&linked); err != nil {
		t.Fatalf("read platform_tenant_id: %v", err)
	}
	if linked != nil {
		t.Errorf("organization.platform_tenant_id = %q after failed bootstrap, want NULL", *linked)
	}

	// Atomicity: no roles for any tenant other than the squatter (which has none).
	var roleCount int
	if err := superPool.QueryRow(ctx, `
		SELECT count(*) FROM roles r
		  JOIN tenants t ON t.id = r.tenant_id
		 WHERE t.organization_id = $1 AND r.is_system = true
	`, orgID).Scan(&roleCount); err != nil {
		t.Fatalf("count leaked roles: %v", err)
	}
	if roleCount != 0 {
		t.Errorf("found %d leaked system roles after failed bootstrap, want 0", roleCount)
	}
}

// TestListClientTenants_ExcludesPlatform verifies that ListClientTenants
// returns only the org's working tenants and excludes the hidden platform
// tenant created by BootstrapPlatformTenant.
func TestListClientTenants_ExcludesPlatform(t *testing.T) {
	superPool, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	s := store.NewStoreWithBypass(superPool, superPool)

	// MSP org + bootstrap platform tenant.
	var orgID string
	if err := superPool.QueryRow(ctx,
		"INSERT INTO organizations (name, slug, type) VALUES ('Globex', 'globex', 'msp') RETURNING id::text",
	).Scan(&orgID); err != nil {
		t.Fatalf("seed org: %v", err)
	}
	platformID, err := s.BootstrapPlatformTenant(ctx, orgID, "Globex")
	if err != nil {
		t.Fatalf("BootstrapPlatformTenant: %v", err)
	}

	// Add two client tenants.
	var clientA, clientB string
	if err := superPool.QueryRow(ctx,
		"INSERT INTO tenants (name, slug, organization_id) VALUES ('Client A', 'globex-a', $1) RETURNING id::text",
		orgID,
	).Scan(&clientA); err != nil {
		t.Fatalf("seed clientA: %v", err)
	}
	if err := superPool.QueryRow(ctx,
		"INSERT INTO tenants (name, slug, organization_id) VALUES ('Client B', 'globex-b', $1) RETURNING id::text",
		orgID,
	).Scan(&clientB); err != nil {
		t.Fatalf("seed clientB: %v", err)
	}

	clients, err := s.ListClientTenants(ctx, orgID)
	if err != nil {
		t.Fatalf("ListClientTenants: %v", err)
	}
	if len(clients) != 2 {
		t.Fatalf("got %d client tenants, want 2", len(clients))
	}
	for _, c := range clients {
		id := uuidString(c.ID)
		if id == platformID {
			t.Errorf("platform tenant %q must not appear in client list", id)
		}
		if id != clientA && id != clientB {
			t.Errorf("unexpected tenant %q in client list", id)
		}
	}
}
