-- name: CreateOrganization :one
INSERT INTO organizations (name, slug, type, parent_org_id, zitadel_org_id, license_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetOrganizationByID :one
SELECT * FROM organizations WHERE id = $1;

-- name: GetOrganizationBySlug :one
SELECT * FROM organizations WHERE slug = $1;

-- name: GetOrganizationByZitadelOrgID :one
SELECT * FROM organizations WHERE zitadel_org_id = $1 AND zitadel_org_id IS NOT NULL;

-- name: ListOrganizations :many
SELECT * FROM organizations
ORDER BY created_at DESC, id DESC
LIMIT $1 OFFSET $2;

-- name: CountOrganizations :one
SELECT count(*) FROM organizations;

-- name: UpdateOrganization :one
UPDATE organizations
SET name           = $2,
    type           = $3,
    zitadel_org_id = $4,
    license_id     = $5,
    updated_at     = now()
WHERE id = $1
RETURNING *;

-- name: DeleteOrganization :execrows
DELETE FROM organizations WHERE id = $1;

-- name: ListTenantsByOrganization :many
SELECT * FROM tenants
WHERE organization_id = $1
ORDER BY created_at
LIMIT 1000;

-- name: ListClientTenantsByOrganization :many
-- Returns tenants under an organization EXCLUDING the org's hidden platform
-- tenant (which exists only to host org-scoped role definitions for MSP/
-- reseller orgs). Used by MSP operator UIs that show the client tenant list.
SELECT * FROM tenants
WHERE organization_id = $1
  AND id <> COALESCE(
        (SELECT platform_tenant_id FROM organizations WHERE id = $1),
        '00000000-0000-0000-0000-000000000000'::uuid
      )
ORDER BY created_at
LIMIT 1000;

-- name: SetOrganizationPlatformTenant :exec
UPDATE organizations
SET platform_tenant_id = $2,
    updated_at = now()
WHERE id = $1;

-- name: CountTenantsByOrganization :one
SELECT count(*) FROM tenants WHERE organization_id = $1;

-- name: CreateTenantInOrganization :one
INSERT INTO tenants (name, slug, license_id, organization_id)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: AssignOrgUserRole :exec
INSERT INTO org_user_roles (organization_id, user_id, role_id)
VALUES ($1, $2, $3)
ON CONFLICT (organization_id, user_id, role_id) DO NOTHING;

-- name: RevokeOrgUserRole :execrows
DELETE FROM org_user_roles
WHERE organization_id = $1 AND user_id = $2 AND role_id = $3;

-- name: ListOrgUserRoles :many
SELECT our.organization_id, our.user_id, our.role_id, our.assigned_at,
       r.name AS role_name, r.description AS role_description
FROM org_user_roles our
JOIN roles r ON r.id = our.role_id
WHERE our.organization_id = $1 AND our.user_id = $2
ORDER BY our.assigned_at;

-- name: ListOrgUsersWithRoles :many
SELECT our.user_id, our.role_id, our.assigned_at,
       r.name AS role_name
FROM org_user_roles our
JOIN roles r ON r.id = our.role_id
WHERE our.organization_id = $1
ORDER BY our.user_id, our.assigned_at;

-- name: GetUserOrgPermissions :many
-- Returns all permissions granted to a user via org-scoped role assignments.
-- Walks the role_chain recursively (like GetUserPermissions) to honor role
-- inheritance. Uses a single tenant context (the role's tenant — the platform
-- tenant of the org). This query MUST be run via BypassPool because RLS on
-- roles/role_permissions would otherwise filter out the platform tenant's rows
-- when the active user tenant is different.
WITH RECURSIVE role_chain AS (
    -- Base: roles directly assigned to the user via org_user_roles
    SELECT r.id, r.parent_role_id, r.tenant_id
    FROM roles r
    JOIN org_user_roles our ON our.role_id = r.id
    WHERE our.organization_id = $1 AND our.user_id = $2

    UNION

    -- Recursive: walk parent chain within the same tenant
    SELECT r.id, r.parent_role_id, r.tenant_id
    FROM roles r
    JOIN role_chain rc ON rc.parent_role_id = r.id AND r.tenant_id = rc.tenant_id
) CYCLE id SET is_cycle USING path
SELECT DISTINCT rp.resource, rp.action, rp.scope
FROM role_permissions rp
JOIN role_chain rc ON rc.id = rp.role_id AND rp.tenant_id = rc.tenant_id
LIMIT 1000;

-- name: ListUserAccessibleTenants :many
-- Returns tenants the user can access within the given organization.
-- A user can access a tenant if they hold any org-scoped role in that org
-- (→ access to all tenants in the org) OR a tenant-scoped role in that
-- specific tenant.
-- MUST be run via BypassPool because user_roles has RLS.
SELECT DISTINCT t.*
FROM tenants t
WHERE t.organization_id = $1
  AND (
    EXISTS (
      SELECT 1 FROM org_user_roles our
      WHERE our.organization_id = $1 AND our.user_id = $2
    )
    OR EXISTS (
      SELECT 1 FROM user_roles ur
      WHERE ur.tenant_id = t.id AND ur.user_id = $2
    )
  )
ORDER BY t.created_at
LIMIT 1000;
