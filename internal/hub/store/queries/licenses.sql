-- name: CreateLicense :one
INSERT INTO licenses (tenant_id, license_key, tier, max_endpoints, issued_at, expires_at, customer_name, customer_email, client_id, notes)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetLicenseByID :one
SELECT l.*, c.hostname AS client_hostname, c.endpoint_count AS client_endpoint_count
FROM licenses l
LEFT JOIN clients c ON c.id = l.client_id
WHERE l.id = $1;

-- name: ListLicenses :many
SELECT l.*, c.hostname AS client_hostname, c.endpoint_count AS client_endpoint_count
FROM licenses l
LEFT JOIN clients c ON c.id = l.client_id
WHERE (sqlc.narg('tier')::text IS NULL OR l.tier = sqlc.narg('tier'))
  AND (sqlc.narg('status_filter')::text IS NULL
       OR (sqlc.narg('status_filter') = 'active' AND l.revoked_at IS NULL AND l.expires_at > now())
       OR (sqlc.narg('status_filter') = 'expired' AND l.revoked_at IS NULL AND l.expires_at <= now())
       OR (sqlc.narg('status_filter') = 'revoked' AND l.revoked_at IS NOT NULL))
ORDER BY l.created_at DESC
LIMIT sqlc.arg('query_limit') OFFSET sqlc.arg('query_offset');

-- name: CountLicenses :one
SELECT count(*) FROM licenses
WHERE (sqlc.narg('tier')::text IS NULL OR tier = sqlc.narg('tier'))
  AND (sqlc.narg('status_filter')::text IS NULL
       OR (sqlc.narg('status_filter') = 'active' AND revoked_at IS NULL AND expires_at > now())
       OR (sqlc.narg('status_filter') = 'expired' AND revoked_at IS NULL AND expires_at <= now())
       OR (sqlc.narg('status_filter') = 'revoked' AND revoked_at IS NOT NULL));

-- name: RevokeLicense :one
UPDATE licenses
SET revoked_at = now(), updated_at = now()
WHERE id = $1 AND revoked_at IS NULL
RETURNING *;

-- name: AssignLicenseToClient :one
UPDATE licenses
SET client_id = $2, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: RenewLicense :one
UPDATE licenses
SET tier = COALESCE(sqlc.narg('new_tier')::text, tier),
    max_endpoints = COALESCE(sqlc.narg('new_max_endpoints')::int, max_endpoints),
    expires_at = @expires_at,
    revoked_at = NULL,
    updated_at = now()
WHERE id = @id
RETURNING *;
