-- name: CreateClient :one
INSERT INTO clients (tenant_id, hostname, version, os, endpoint_count, contact_email, status, bootstrap_token)
VALUES ($1, $2, $3, $4, $5, $6, 'pending', $7)
RETURNING *;

-- name: GetClientByID :one
SELECT * FROM clients WHERE id = $1;

-- name: GetClientByBootstrapToken :one
SELECT * FROM clients WHERE bootstrap_token = $1;

-- name: GetClientByAPIKeyHash :one
SELECT * FROM clients WHERE api_key_hash = $1 AND status = 'approved';

-- name: ListClients :many
SELECT * FROM clients
WHERE (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'))
ORDER BY created_at DESC
LIMIT sqlc.arg('query_limit') OFFSET sqlc.arg('query_offset');

-- name: CountClients :one
SELECT count(*) FROM clients
WHERE (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'));

-- name: CountPendingClients :one
SELECT count(*) FROM clients WHERE status = 'pending';

-- name: UpdateClient :one
UPDATE clients
SET hostname = COALESCE(sqlc.narg('hostname'), hostname),
    sync_interval = COALESCE(sqlc.narg('sync_interval'), sync_interval),
    notes = COALESCE(sqlc.narg('notes'), notes),
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: ApproveClient :one
UPDATE clients
SET status = 'approved', api_key_hash = $2, updated_at = now()
WHERE id = $1 AND status = 'pending'
RETURNING *;

-- name: DeclineClient :one
UPDATE clients
SET status = 'declined', updated_at = now()
WHERE id = $1 AND status = 'pending'
RETURNING *;

-- name: SuspendClient :one
UPDATE clients
SET status = 'suspended', updated_at = now()
WHERE id = $1 AND status = 'approved'
RETURNING *;

-- name: DeleteClient :exec
DELETE FROM clients WHERE id = $1;

-- name: UpdateClientSyncTime :exec
UPDATE clients
SET last_sync_at = now(), endpoint_count = $2, version = $3, updated_at = now()
WHERE id = $1;

-- name: UpdateClientSummaries :one
UPDATE clients
SET endpoint_count = @endpoint_count,
    last_sync_at = now(),
    os_summary = CASE WHEN @os_summary::jsonb = '{}'::jsonb THEN os_summary ELSE @os_summary END,
    endpoint_status_summary = CASE WHEN @endpoint_status_summary::jsonb = '{}'::jsonb THEN endpoint_status_summary ELSE @endpoint_status_summary END,
    compliance_summary = CASE WHEN @compliance_summary::jsonb = '{}'::jsonb THEN compliance_summary ELSE @compliance_summary END,
    updated_at = now()
WHERE id = @id
RETURNING *;
