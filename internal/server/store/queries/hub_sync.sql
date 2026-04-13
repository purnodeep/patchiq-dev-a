-- name: GetHubSyncState :one
SELECT * FROM hub_sync_state WHERE tenant_id = @tenant_id LIMIT 1;

-- name: ListAllHubSyncStates :many
SELECT * FROM hub_sync_state WHERE status != 'disabled' ORDER BY created_at
LIMIT 100;

-- name: UpsertHubSyncState :one
INSERT INTO hub_sync_state (tenant_id, hub_url, api_key, sync_interval)
VALUES (@tenant_id, @hub_url, @api_key, @sync_interval)
ON CONFLICT (tenant_id) DO UPDATE
SET hub_url = EXCLUDED.hub_url,
    api_key = EXCLUDED.api_key,
    sync_interval = EXCLUDED.sync_interval,
    updated_at = now()
RETURNING *;

-- name: UpdateHubSyncStarted :exec
UPDATE hub_sync_state
SET status = 'syncing', updated_at = now()
WHERE tenant_id = @tenant_id;

-- name: UpdateHubSyncCompleted :exec
UPDATE hub_sync_state
SET status = 'idle',
    last_sync_at = now(),
    next_sync_at = @next_sync_at,
    entries_received = entries_received + @entry_count,
    last_entry_count = @entry_count,
    last_error = NULL,
    updated_at = now()
WHERE tenant_id = @tenant_id;

-- name: UpdateHubSyncFailed :exec
UPDATE hub_sync_state
SET status = 'error',
    last_error = @error_message,
    updated_at = now()
WHERE tenant_id = @tenant_id;

-- name: UpdateHubCVESyncCompleted :exec
UPDATE hub_sync_state
SET last_cve_sync_at = now(), updated_at = now()
WHERE tenant_id = $1;

-- name: UpdateHubCVESyncFailed :exec
UPDATE hub_sync_state
SET last_error = $1, updated_at = now()
WHERE tenant_id = $2;
