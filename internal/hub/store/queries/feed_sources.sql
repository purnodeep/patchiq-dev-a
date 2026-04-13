-- name: GetFeedSourceByName :one
SELECT * FROM feed_sources WHERE name = $1;

-- name: ListFeedSources :many
SELECT * FROM feed_sources ORDER BY name
LIMIT 100;

-- name: ListEnabledFeedSources :many
SELECT * FROM feed_sources WHERE enabled = true ORDER BY name
LIMIT 100;

-- name: UpdateFeedSourceEnabled :exec
UPDATE feed_sources SET enabled = $2, updated_at = now() WHERE id = $1;

-- name: GetFeedSyncState :one
SELECT * FROM feed_sync_state WHERE feed_source_id = $1;

-- name: UpdateFeedSyncStateStart :exec
UPDATE feed_sync_state SET status = 'syncing', updated_at = now() WHERE feed_source_id = $1;

-- name: UpdateFeedSyncStateSuccess :exec
UPDATE feed_sync_state
SET status = 'idle',
    last_sync_at = now(),
    next_sync_at = $2,
    cursor = $3,
    entries_ingested = entries_ingested + $4,
    error_count = 0,
    last_error = NULL,
    updated_at = now()
WHERE feed_source_id = $1;

-- name: UpdateFeedSyncStateError :exec
UPDATE feed_sync_state
SET status = 'error',
    error_count = error_count + 1,
    last_error = $2,
    updated_at = now()
WHERE feed_source_id = $1;

-- name: ListFeedSourcesWithSyncState :many
SELECT
    fs.id,
    fs.name,
    fs.display_name,
    fs.enabled,
    fs.sync_interval_seconds,
    fs.url,
    fs.auth_type,
    fss.last_sync_at,
    fss.next_sync_at,
    fss.status,
    fss.error_count,
    fss.last_error,
    fss.entries_ingested,
    fss.cursor
FROM feed_sources fs
LEFT JOIN feed_sync_state fss ON fss.feed_source_id = fs.id
ORDER BY fs.name
LIMIT 100;

-- name: GetFeedSourceByID :one
SELECT * FROM feed_sources WHERE id = $1;

-- name: UpdateFeedSource :one
UPDATE feed_sources
SET enabled = COALESCE(sqlc.narg('enabled'), enabled),
    sync_interval_seconds = COALESCE(sqlc.narg('sync_interval_seconds'), sync_interval_seconds),
    url = COALESCE(sqlc.narg('url'), url),
    auth_type = COALESCE(sqlc.narg('auth_type'), auth_type),
    severity_filter = COALESCE(sqlc.narg('severity_filter'), severity_filter),
    os_filter = COALESCE(sqlc.narg('os_filter'), os_filter),
    severity_mapping = COALESCE(sqlc.narg('severity_mapping'), severity_mapping),
    updated_at = now()
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: GetFeedSourceWithSyncStateByID :one
SELECT fs.id, fs.name, fs.display_name, fs.enabled, fs.sync_interval_seconds,
       fs.url, fs.auth_type, fs.severity_filter, fs.os_filter, fs.severity_mapping,
       fs.created_at, fs.updated_at,
       fss.last_sync_at, fss.next_sync_at, fss.status, fss.error_count,
       fss.last_error, fss.entries_ingested, fss.cursor
FROM feed_sources fs
LEFT JOIN feed_sync_state fss ON fss.feed_source_id = fs.id
WHERE fs.id = $1;
