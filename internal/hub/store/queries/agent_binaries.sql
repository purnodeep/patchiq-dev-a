-- name: CreateAgentBinary :one
INSERT INTO agent_binaries (os_family, arch, version, download_url, checksum, released_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetAgentBinaryByID :one
SELECT * FROM agent_binaries WHERE id = $1;

-- name: ListAgentBinaries :many
SELECT * FROM agent_binaries ORDER BY released_at DESC
LIMIT $1 OFFSET $2;

-- name: GetLatestBinary :one
SELECT * FROM agent_binaries
WHERE os_family = $1 AND arch = $2
ORDER BY released_at DESC
LIMIT 1;
