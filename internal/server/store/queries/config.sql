-- name: UpsertConfigOverride :one
INSERT INTO config_overrides (tenant_id, scope_type, scope_id, module, config, updated_by)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (tenant_id, scope_type, scope_id, module)
DO UPDATE SET config = EXCLUDED.config, updated_by = EXCLUDED.updated_by, updated_at = now()
RETURNING *;

-- name: GetConfigOverride :one
SELECT * FROM config_overrides
WHERE tenant_id = $1 AND scope_type = $2 AND scope_id = $3 AND module = $4;

-- name: ListConfigOverridesByTenant :many
SELECT * FROM config_overrides
WHERE tenant_id = $1
ORDER BY scope_type, module
LIMIT 100;

-- name: ListConfigOverridesByScope :many
SELECT * FROM config_overrides
WHERE tenant_id = $1 AND scope_type = $2 AND scope_id = $3
ORDER BY module
LIMIT 100;

-- name: DeleteConfigOverride :exec
DELETE FROM config_overrides
WHERE tenant_id = $1 AND scope_type = $2 AND scope_id = $3 AND module = $4;

-- name: GetCommsConfigForEndpoint :one
-- Returns the comms config override for a specific endpoint, if it was updated
-- after the given timestamp (used for config push via heartbeat).
SELECT * FROM config_overrides
WHERE tenant_id = @tenant_id AND scope_type = 'endpoint' AND scope_id = @scope_id AND module = 'comms'
  AND (sqlc.narg('updated_after')::timestamptz IS NULL OR updated_at > sqlc.narg('updated_after'));
