-- name: CreateTenant :one
INSERT INTO tenants (name, slug, license_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetTenantByID :one
SELECT * FROM tenants WHERE id = $1;

-- name: GetTenantBySlug :one
SELECT * FROM tenants WHERE slug = $1;

-- name: ListTenants :many
SELECT * FROM tenants ORDER BY created_at
LIMIT 1000;

-- name: UpdateTenant :one
UPDATE tenants
SET name = $2, slug = $3, license_id = $4, updated_at = now()
WHERE id = $1
RETURNING *;
