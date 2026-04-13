-- name: ListPendingBinaryFetches :many
SELECT * FROM binary_fetch_state
WHERE status IN ('pending', 'failed')
AND (retry_count < 3 OR status = 'pending')
ORDER BY created_at ASC
LIMIT $1;

-- name: CreateBinaryFetchState :one
INSERT INTO binary_fetch_state (catalog_id, os_distribution, os_version, status, fetch_url)
VALUES ($1, $2, $3, 'pending', $4)
ON CONFLICT (catalog_id, os_distribution) DO NOTHING
RETURNING *;

-- name: UpdateBinaryFetchSuccess :exec
UPDATE binary_fetch_state
SET status = 'complete',
    binary_ref = $2,
    checksum_sha256 = $3,
    file_size_bytes = $4,
    last_attempt_at = now(),
    updated_at = now()
WHERE id = $1;

-- name: UpdateBinaryFetchFailed :exec
UPDATE binary_fetch_state
SET status = 'failed',
    error_message = $2,
    retry_count = retry_count + 1,
    last_attempt_at = now(),
    updated_at = now()
WHERE id = $1;

-- name: GetBinaryFetchState :one
SELECT * FROM binary_fetch_state
WHERE catalog_id = $1 AND os_distribution = $2;
