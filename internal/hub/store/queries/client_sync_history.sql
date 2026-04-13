-- name: InsertClientSyncHistory :one
INSERT INTO client_sync_history (
    tenant_id, client_id, started_at, finished_at, duration_ms,
    entries_delivered, deletes_delivered, endpoint_count, status, error_message
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: ListClientSyncHistory :many
SELECT *
FROM client_sync_history
WHERE tenant_id = @tenant_id
  AND client_id = @client_id
ORDER BY started_at DESC
LIMIT sqlc.arg('query_limit') OFFSET sqlc.arg('query_offset');

-- name: CountClientSyncHistory :one
SELECT COUNT(*)::bigint
FROM client_sync_history
WHERE tenant_id = @tenant_id
  AND client_id = @client_id;

-- name: GetClientEndpointTrend :many
SELECT DATE(started_at) AS date, MAX(endpoint_count)::int AS endpoint_count
FROM client_sync_history
WHERE tenant_id = @tenant_id
  AND client_id = @client_id
  AND started_at > now() - make_interval(days => @days::int)
GROUP BY DATE(started_at)
ORDER BY date
LIMIT 365;

-- name: GetLicenseUsageHistory :many
SELECT DATE(csh.started_at) AS date, MAX(csh.endpoint_count)::int AS endpoint_count
FROM client_sync_history csh
JOIN licenses l ON l.client_id = csh.client_id AND l.tenant_id = csh.tenant_id
WHERE l.id = @license_id
  AND l.tenant_id = @tenant_id
  AND csh.started_at > now() - make_interval(days => @days::int)
GROUP BY DATE(csh.started_at)
ORDER BY date
LIMIT 365;
