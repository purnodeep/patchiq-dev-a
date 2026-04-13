-- name: ListFeedSyncHistory :many
SELECT id, feed_source_id, started_at, finished_at, duration_ms,
       new_entries, updated_entries, total_scanned, error_count,
       status, error_message, log_output, created_at
FROM feed_sync_history
WHERE feed_source_id = $1
ORDER BY started_at DESC
LIMIT $2 OFFSET $3;

-- name: CountFeedSyncHistory :one
SELECT count(*) FROM feed_sync_history WHERE feed_source_id = $1;

-- name: CreateFeedSyncHistory :one
INSERT INTO feed_sync_history (feed_source_id, started_at, finished_at, duration_ms,
    new_entries, updated_entries, total_scanned, error_count, status, error_message, log_output)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: ListRecentFeedSyncStatus :many
SELECT status, started_at
FROM feed_sync_history
WHERE feed_source_id = $1
ORDER BY started_at DESC
LIMIT 30;

-- name: GetFeedNewThisWeek :one
SELECT count(*)::bigint AS new_this_week
FROM patch_catalog
WHERE feed_source_id = $1
  AND created_at > now() - interval '7 days'
  AND deleted_at IS NULL;

-- name: GetFeedErrorRate :one
SELECT
    count(*) FILTER (WHERE status = 'failed') AS failed_count,
    count(*) AS total_count
FROM feed_sync_history
WHERE feed_source_id = $1 AND started_at > now() - interval '30 days';
