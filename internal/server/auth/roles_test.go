package auth_test

import (
	"testing"

	"github.com/skenzeriq/patchiq/internal/server/auth"
)

func TestPresetRoles(t *testing.T) {
	roles := auth.PresetRoles()

	expected := map[string]auth.RoleScope{
		"Super Admin":      auth.RoleScopeTenant,
		"IT Manager":       auth.RoleScopeTenant,
		"Operator":         auth.RoleScopeTenant,
		"Security Analyst": auth.RoleScopeTenant,
		"Auditor":          auth.RoleScopeTenant,
		"Help Desk":        auth.RoleScopeTenant,
		"MSP Admin":        auth.RoleScopeOrg,
		"MSP Technician":   auth.RoleScopeOrg,
		"MSP Auditor":      auth.RoleScopeOrg,
	}

	if len(roles) != len(expected) {
		t.Fatalf("PresetRoles() returned %d roles, want %d", len(roles), len(expected))
	}

	seen := map[string]bool{}
	for _, r := range roles {
		wantScope, ok := expected[r.Name]
		if !ok {
			t.Errorf("unexpected role: %q", r.Name)
			continue
		}
		if r.Scope != wantScope {
			t.Errorf("role %q has scope %q, want %q", r.Name, r.Scope, wantScope)
		}
		seen[r.Name] = true

		if len(r.Permissions) == 0 {
			t.Errorf("role %q has no permissions", r.Name)
		}

		for _, p := range r.Permissions {
			if _, err := auth.ParsePermission(p); err != nil {
				t.Errorf("role %q has invalid permission %q: %v", r.Name, p, err)
			}
		}
	}

	for name := range expected {
		if !seen[name] {
			t.Errorf("missing expected role: %q", name)
		}
	}
}

func TestSuperAdminCoversEverything(t *testing.T) {
	roles := auth.PresetRoles()
	var superAdmin auth.RoleTemplate
	for _, r := range roles {
		if r.Name == "Super Admin" {
			superAdmin = r
			break
		}
	}

	resources := []string{"endpoints", "deployments", "policies", "audit", "roles", "settings"}
	actions := []string{"read", "create", "update", "delete", "approve"}

	for _, res := range resources {
		for _, act := range actions {
			required := auth.Permission{Resource: res, Action: act, Scope: "*"}
			covered := false
			for _, ps := range superAdmin.Permissions {
				p, _ := auth.ParsePermission(ps)
				if p.Covers(required) {
					covered = true
					break
				}
			}
			if !covered {
				t.Errorf("Super Admin does not cover %s", required.String())
			}
		}
	}
}

func TestMSPAdminCoversEverythingAcrossOrg(t *testing.T) {
	var mspAdmin auth.RoleTemplate
	for _, r := range auth.PresetRoles() {
		if r.Name == "MSP Admin" {
			mspAdmin = r
			break
		}
	}
	if mspAdmin.Name == "" {
		t.Fatal("MSP Admin preset not found")
	}
	if mspAdmin.Scope != auth.RoleScopeOrg {
		t.Fatalf("MSP Admin scope = %q, want %q", mspAdmin.Scope, auth.RoleScopeOrg)
	}
	if len(mspAdmin.Permissions) != 1 || mspAdmin.Permissions[0] != "*:*:*" {
		t.Errorf("MSP Admin permissions = %v, want [*:*:*]", mspAdmin.Permissions)
	}
}

func TestTenantScopedPresets_ExcludesOrgRoles(t *testing.T) {
	presets := auth.TenantScopedPresets()
	for _, r := range presets {
		if r.Scope == auth.RoleScopeOrg {
			t.Errorf("TenantScopedPresets returned org-scoped role %q", r.Name)
		}
	}
	if len(presets) == 0 {
		t.Error("TenantScopedPresets returned empty slice")
	}
}

func TestOrgScopedPresets_ExcludesTenantRoles(t *testing.T) {
	presets := auth.OrgScopedPresets()
	for _, r := range presets {
		if r.Scope != auth.RoleScopeOrg {
			t.Errorf("OrgScopedPresets returned non-org role %q with scope %q", r.Name, r.Scope)
		}
	}
	wantNames := map[string]bool{"MSP Admin": false, "MSP Technician": false, "MSP Auditor": false}
	for _, r := range presets {
		if _, ok := wantNames[r.Name]; ok {
			wantNames[r.Name] = true
		}
	}
	for n, found := range wantNames {
		if !found {
			t.Errorf("OrgScopedPresets missing %q", n)
		}
	}
}
