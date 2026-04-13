-- name: UpsertHubConfig :one
INSERT INTO hub_config (tenant_id, key, value, updated_by)
VALUES ($1, $2, $3, $4)
ON CONFLICT (tenant_id, key)
DO UPDATE SET value = EXCLUDED.value, updated_by = EXCLUDED.updated_by, updated_at = now()
RETURNING *;

-- name: GetHubConfig :one
SELECT * FROM hub_config
WHERE tenant_id = $1 AND key = $2;

-- name: ListHubConfigByTenant :many
SELECT * FROM hub_config
WHERE tenant_id = $1
ORDER BY key
LIMIT 100;

-- name: DeleteHubConfig :exec
DELETE FROM hub_config
WHERE tenant_id = $1 AND key = $2;
