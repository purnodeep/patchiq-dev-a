package auth_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/skenzeriq/patchiq/internal/server/auth"
	"github.com/skenzeriq/patchiq/internal/shared/organization"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"github.com/skenzeriq/patchiq/internal/shared/user"
)

type mockPermissionStore struct {
	permissions []auth.Permission
	err         error
}

func (m *mockPermissionStore) GetUserPermissions(_ context.Context, _, _ string) ([]auth.Permission, error) {
	return m.permissions, m.err
}

type mockOrgPermissionStore struct {
	orgPermissions []auth.Permission
	err            error
}

func (m *mockOrgPermissionStore) GetUserOrgPermissions(_ context.Context, _, _ string) ([]auth.Permission, error) {
	return m.orgPermissions, m.err
}

func TestEvaluatorHasPermission(t *testing.T) {
	tests := []struct {
		name     string
		held     []string
		required string
		want     bool
	}{
		{"super admin has everything", []string{"*:*:*"}, "endpoints:read:*", true},
		{"exact match", []string{"endpoints:read:*"}, "endpoints:read:*", true},
		{"no matching permission", []string{"policies:read:*"}, "endpoints:read:*", false},
		{"multiple roles one matches", []string{"policies:read:*", "endpoints:read:*"}, "endpoints:read:*", true},
		{"wildcard action covers specific", []string{"endpoints:*:*"}, "endpoints:read:*", true},
		{"group scope covers same group", []string{"deployments:approve:group:prod"}, "deployments:approve:group:prod", true},
		{"group scope does not cover wildcard", []string{"deployments:approve:group:prod"}, "deployments:approve:*", false},
		{"empty permissions denies", []string{}, "endpoints:read:*", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			perms := make([]auth.Permission, 0, len(tt.held))
			for _, s := range tt.held {
				p, err := auth.ParsePermission(s)
				if err != nil {
					t.Fatalf("parse %q: %v", s, err)
				}
				perms = append(perms, p)
			}

			store := &mockPermissionStore{permissions: perms}
			eval := auth.NewEvaluator(store)

			ctx := tenant.WithTenantID(context.Background(), "00000000-0000-0000-0000-000000000001")
			ctx = user.WithUserID(ctx, "test-user")

			required, err := auth.ParsePermission(tt.required)
			if err != nil {
				t.Fatalf("parse required: %v", err)
			}

			got, err := eval.HasPermission(ctx, required)
			if err != nil {
				t.Fatalf("HasPermission() error: %v", err)
			}
			if got != tt.want {
				t.Errorf("HasPermission(%q) = %v, want %v", tt.required, got, tt.want)
			}
		})
	}
}

func TestNewEvaluatorPanicsOnNil(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("NewEvaluator(nil) did not panic")
		}
	}()
	auth.NewEvaluator(nil)
}

func TestEvaluatorStoreError(t *testing.T) {
	store := &mockPermissionStore{err: fmt.Errorf("db connection lost")}
	eval := auth.NewEvaluator(store)

	ctx := tenant.WithTenantID(context.Background(), "00000000-0000-0000-0000-000000000001")
	ctx = user.WithUserID(ctx, "test-user")

	_, err := eval.HasPermission(ctx, auth.Permission{Resource: "endpoints", Action: "read", Scope: "*"})
	if err == nil {
		t.Fatal("expected error from store")
	}
	if !strings.Contains(err.Error(), "db connection lost") {
		t.Errorf("error should wrap store error, got: %v", err)
	}
}

// TestEvaluatorOrgScopedGrant verifies that an org-scoped permission grant
// satisfies a permission check even when the tenant-scoped store returns
// nothing. This is the MSP Admin path: grant at the org level, check in
// any tenant in the org.
func TestEvaluatorOrgScopedGrant(t *testing.T) {
	tenantStore := &mockPermissionStore{} // empty — tenant-scoped grants absent
	orgStore := &mockOrgPermissionStore{
		orgPermissions: []auth.Permission{{Resource: "*", Action: "*", Scope: "*"}},
	}
	eval := auth.NewEvaluator(tenantStore).WithOrgStore(orgStore)

	ctx := tenant.WithTenantID(context.Background(), "00000000-0000-0000-0000-000000000001")
	ctx = user.WithUserID(ctx, "msp-admin")
	ctx = organization.WithOrgID(ctx, "11111111-1111-1111-1111-111111111111")

	got, err := eval.HasPermission(ctx, auth.Permission{Resource: "endpoints", Action: "delete", Scope: "*"})
	if err != nil {
		t.Fatalf("HasPermission: %v", err)
	}
	if !got {
		t.Error("expected org-scoped wildcard grant to satisfy endpoints:delete:*")
	}
}

// TestEvaluatorOrgGrantMissWithTenantGrantHit verifies the fall-through:
// org-scoped grant doesn't cover the required permission, but a tenant-scoped
// grant does.
func TestEvaluatorOrgGrantMissWithTenantGrantHit(t *testing.T) {
	tenantStore := &mockPermissionStore{
		permissions: []auth.Permission{{Resource: "endpoints", Action: "read", Scope: "*"}},
	}
	orgStore := &mockOrgPermissionStore{
		orgPermissions: []auth.Permission{{Resource: "reports", Action: "read", Scope: "*"}},
	}
	eval := auth.NewEvaluator(tenantStore).WithOrgStore(orgStore)

	ctx := tenant.WithTenantID(context.Background(), "00000000-0000-0000-0000-000000000001")
	ctx = user.WithUserID(ctx, "msp-technician")
	ctx = organization.WithOrgID(ctx, "11111111-1111-1111-1111-111111111111")

	got, err := eval.HasPermission(ctx, auth.Permission{Resource: "endpoints", Action: "read", Scope: "*"})
	if err != nil {
		t.Fatalf("HasPermission: %v", err)
	}
	if !got {
		t.Error("expected tenant-scoped grant to satisfy when org grant misses")
	}
}

// TestEvaluatorOrgStoreError verifies that an error from the org store
// propagates rather than being silently swallowed.
func TestEvaluatorOrgStoreError(t *testing.T) {
	tenantStore := &mockPermissionStore{}
	orgStore := &mockOrgPermissionStore{err: fmt.Errorf("bypass pool down")}
	eval := auth.NewEvaluator(tenantStore).WithOrgStore(orgStore)

	ctx := tenant.WithTenantID(context.Background(), "00000000-0000-0000-0000-000000000001")
	ctx = user.WithUserID(ctx, "user-1")
	ctx = organization.WithOrgID(ctx, "11111111-1111-1111-1111-111111111111")

	_, err := eval.HasPermission(ctx, auth.Permission{Resource: "endpoints", Action: "read", Scope: "*"})
	if err == nil {
		t.Fatal("expected error from org store")
	}
	if !strings.Contains(err.Error(), "bypass pool down") {
		t.Errorf("error should wrap store error, got: %v", err)
	}
}

// TestEvaluatorNoOrgStoreFallsBack verifies that when no org store is
// configured, the evaluator behaves exactly as before (tenant-scoped only).
func TestEvaluatorNoOrgStoreFallsBack(t *testing.T) {
	tenantStore := &mockPermissionStore{
		permissions: []auth.Permission{{Resource: "endpoints", Action: "read", Scope: "*"}},
	}
	eval := auth.NewEvaluator(tenantStore) // no WithOrgStore

	ctx := tenant.WithTenantID(context.Background(), "00000000-0000-0000-0000-000000000001")
	ctx = user.WithUserID(ctx, "user-1")
	// Even with an orgID in context, the evaluator should ignore it.
	ctx = organization.WithOrgID(ctx, "11111111-1111-1111-1111-111111111111")

	got, err := eval.HasPermission(ctx, auth.Permission{Resource: "endpoints", Action: "read", Scope: "*"})
	if err != nil {
		t.Fatalf("HasPermission: %v", err)
	}
	if !got {
		t.Error("expected tenant-scoped grant to satisfy when org store is not configured")
	}
}

func TestEvaluatorMissingContext(t *testing.T) {
	eval := auth.NewEvaluator(&mockPermissionStore{})

	t.Run("missing tenant ID", func(t *testing.T) {
		ctx := user.WithUserID(context.Background(), "user-1")
		_, err := eval.HasPermission(ctx, auth.Permission{Resource: "endpoints", Action: "read", Scope: "*"})
		if err == nil {
			t.Fatal("expected error for missing tenant ID")
		}
		if !errors.Is(err, auth.ErrMissingTenantID) {
			t.Errorf("expected ErrMissingTenantID, got: %v", err)
		}
	})

	t.Run("missing user ID", func(t *testing.T) {
		ctx := tenant.WithTenantID(context.Background(), "00000000-0000-0000-0000-000000000001")
		_, err := eval.HasPermission(ctx, auth.Permission{Resource: "endpoints", Action: "read", Scope: "*"})
		if err == nil {
			t.Fatal("expected error for missing user ID")
		}
		if !errors.Is(err, auth.ErrMissingUserID) {
			t.Errorf("expected ErrMissingUserID, got: %v", err)
		}
	})
}
