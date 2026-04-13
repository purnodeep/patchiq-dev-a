package auth

// RoleScope identifies whether a preset role grants permissions within a
// single tenant or across all tenants in an organization.
type RoleScope string

const (
	// RoleScopeTenant means the role is assigned via user_roles in a specific
	// tenant. Its permissions apply only to that tenant.
	RoleScopeTenant RoleScope = "tenant"
	// RoleScopeOrg means the role is assigned via org_user_roles at the org
	// level. Its permissions apply across every tenant in the org (MSP model).
	// These roles are seeded into the org's hidden platform tenant.
	RoleScopeOrg RoleScope = "org"
)

// RoleTemplate defines a preset role with its permissions.
type RoleTemplate struct {
	Name        string
	Description string
	Scope       RoleScope // RoleScopeTenant if empty (backward compatibility)
	Permissions []string
}

// PresetRoles returns the built-in role templates.
func PresetRoles() []RoleTemplate {
	return []RoleTemplate{
		// ------------------------------------------------------------
		// Tenant-scoped presets (seeded into each tenant at provisioning).
		// ------------------------------------------------------------
		{
			Name:        "Super Admin",
			Description: "Full access to everything including RBAC management",
			Scope:       RoleScopeTenant,
			Permissions: []string{"*:*:*"},
		},
		{
			Name:        "IT Manager",
			Description: "Full read, create/approve deployments, manage policies",
			Scope:       RoleScopeTenant,
			Permissions: []string{
				"endpoints:read:*",
				"endpoints:update:*",
				"tags:*:*",
				"patches:read:*",
				"patches:sync:*",
				"policies:*:*",
				"deployments:*:*",
				"reports:*:*",
				"audit:read:*",
				"users:read:*",
				"settings:read:*",
			},
		},
		{
			Name:        "Operator",
			Description: "Read endpoints, create deployments (no approve), run scans",
			Scope:       RoleScopeTenant,
			Permissions: []string{
				"endpoints:read:*",
				"endpoints:scan:*",
				"endpoints:tag:*",
				"tags:read:*",
				"patches:read:*",
				"policies:read:*",
				"deployments:read:*",
				"deployments:create:*",
				"deployments:retry:*",
			},
		},
		{
			Name:        "Security Analyst",
			Description: "Read-only plus compliance reports and CVE search",
			Scope:       RoleScopeTenant,
			Permissions: []string{
				"endpoints:read:*",
				"tags:read:*",
				"patches:read:*",
				"policies:read:*",
				"deployments:read:*",
				"reports:*:*",
				"audit:read:*",
			},
		},
		{
			Name:        "Auditor",
			Description: "Read-only on all resources including audit logs",
			Scope:       RoleScopeTenant,
			Permissions: []string{
				"endpoints:read:*",
				"tags:read:*",
				"patches:read:*",
				"policies:read:*",
				"deployments:read:*",
				"reports:read:*",
				"audit:read:*",
				"users:read:*",
				"roles:read:*",
				"settings:read:*",
			},
		},
		{
			Name:        "Help Desk",
			Description: "Read endpoints, trigger scans, view patch status",
			Scope:       RoleScopeTenant,
			Permissions: []string{
				"endpoints:read:*",
				"endpoints:scan:*",
				"tags:read:*",
				"patches:read:*",
				"deployments:read:*",
			},
		},

		// ------------------------------------------------------------
		// Org-scoped presets (seeded into the org's platform tenant on
		// first conversion to msp-type). Granted via org_user_roles.
		// See docs/adr/025 decision #4.
		// ------------------------------------------------------------
		{
			Name:        "MSP Admin",
			Description: "Full access across every tenant in the organization (MSP operator)",
			Scope:       RoleScopeOrg,
			Permissions: []string{"*:*:*"},
		},
		{
			Name:        "MSP Technician",
			Description: "Cross-tenant endpoint/patch/deployment operations without RBAC or billing",
			Scope:       RoleScopeOrg,
			Permissions: []string{
				"endpoints:read:*",
				"endpoints:update:*",
				"endpoints:scan:*",
				"endpoints:tag:*",
				"tags:read:*",
				"patches:read:*",
				"patches:sync:*",
				"policies:read:*",
				"deployments:read:*",
				"deployments:create:*",
				"deployments:execute:*",
				"deployments:retry:*",
				"reports:read:*",
			},
		},
		{
			Name:        "MSP Auditor",
			Description: "Read-only across every tenant in the organization, including audit logs",
			Scope:       RoleScopeOrg,
			Permissions: []string{
				"endpoints:read:*",
				"tags:read:*",
				"patches:read:*",
				"policies:read:*",
				"deployments:read:*",
				"reports:read:*",
				"audit:read:*",
				"users:read:*",
				"roles:read:*",
			},
		},
	}
}

// TenantScopedPresets returns only the tenant-scoped preset roles.
// Used by tenant provisioning to seed the initial role set.
func TenantScopedPresets() []RoleTemplate {
	var out []RoleTemplate
	for _, r := range PresetRoles() {
		if r.Scope == "" || r.Scope == RoleScopeTenant {
			out = append(out, r)
		}
	}
	return out
}

// OrgScopedPresets returns only the org-scoped preset roles.
// Used by org platform-tenant bootstrap to seed MSP roles.
func OrgScopedPresets() []RoleTemplate {
	var out []RoleTemplate
	for _, r := range PresetRoles() {
		if r.Scope == RoleScopeOrg {
			out = append(out, r)
		}
	}
	return out
}
