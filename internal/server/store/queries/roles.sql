-- name: CreateRole :one
INSERT INTO roles (tenant_id, name, description, parent_role_id, is_system)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetRoleByID :one
SELECT * FROM roles WHERE id = $1 AND tenant_id = $2;

-- name: GetRoleByName :one
SELECT * FROM roles WHERE name = $1 AND tenant_id = $2;

-- name: ListRoles :many
SELECT * FROM roles WHERE tenant_id = $1 ORDER BY name
LIMIT 100;

-- name: UpdateRole :one
UPDATE roles
SET name = $2, description = $3, parent_role_id = $4, updated_at = now()
WHERE id = $1 AND tenant_id = $5 AND is_system = false
RETURNING *;

-- name: DeleteRole :execrows
DELETE FROM roles WHERE id = $1 AND tenant_id = $2 AND is_system = false;

-- name: CreateRolePermission :exec
INSERT INTO role_permissions (tenant_id, role_id, resource, action, scope)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (tenant_id, role_id, resource, action, scope) DO NOTHING;

-- name: ListRolePermissions :many
SELECT * FROM role_permissions WHERE role_id = $1 AND tenant_id = $2
LIMIT 1000;

-- name: DeleteRolePermissions :exec
DELETE FROM role_permissions WHERE role_id = $1 AND tenant_id = $2;

-- name: AssignUserRole :exec
INSERT INTO user_roles (tenant_id, user_id, role_id)
VALUES ($1, $2, $3)
ON CONFLICT (tenant_id, user_id, role_id) DO NOTHING;

-- name: RevokeUserRole :execrows
DELETE FROM user_roles WHERE tenant_id = $1 AND user_id = $2 AND role_id = $3;

-- name: ListUserRoles :many
SELECT r.* FROM roles r
JOIN user_roles ur ON ur.role_id = r.id AND ur.tenant_id = r.tenant_id
WHERE ur.user_id = $1 AND ur.tenant_id = $2
ORDER BY r.name
LIMIT 100;

-- name: GetUserPermissions :many
WITH RECURSIVE role_chain AS (
    -- Base: roles directly assigned to the user
    SELECT r.id, r.parent_role_id
    FROM roles r
    JOIN user_roles ur ON ur.role_id = r.id AND ur.tenant_id = r.tenant_id
    WHERE ur.user_id = $1 AND ur.tenant_id = $2

    UNION

    -- Recursive: walk parent chain
    SELECT r.id, r.parent_role_id
    FROM roles r
    JOIN role_chain rc ON rc.parent_role_id = r.id
    WHERE r.tenant_id = $2
) CYCLE id SET is_cycle USING path
SELECT DISTINCT rp.resource, rp.action, rp.scope
FROM role_permissions rp
JOIN role_chain rc ON rc.id = rp.role_id
WHERE rp.tenant_id = $2
LIMIT 1000;

-- name: ListRolesWithCount :many
SELECT r.*,
       (SELECT count(*) FROM role_permissions rp WHERE rp.role_id = r.id AND rp.tenant_id = r.tenant_id) AS permission_count,
       (SELECT count(*) FROM user_roles ur WHERE ur.role_id = r.id AND ur.tenant_id = r.tenant_id) AS user_count
FROM roles r
WHERE r.tenant_id = $1
  AND ($2::text = '' OR r.name ILIKE '%' || $2 || '%')
  AND (
    ($3::timestamptz IS NULL AND $4::uuid IS NULL)
    OR (r.created_at, r.id) < ($3, $4)
  )
ORDER BY r.created_at DESC, r.id DESC
LIMIT $5;

-- name: CountRoles :one
SELECT count(*) FROM roles
WHERE tenant_id = $1
  AND ($2::text = '' OR name ILIKE '%' || $2 || '%');

-- name: ListRoleUsers :many
SELECT ur.user_id, ur.assigned_at
FROM user_roles ur
WHERE ur.role_id = $1 AND ur.tenant_id = $2
ORDER BY ur.assigned_at
LIMIT 1000;

-- name: CountRoleUsers :one
SELECT count(*) FROM user_roles
WHERE role_id = $1 AND tenant_id = $2;
