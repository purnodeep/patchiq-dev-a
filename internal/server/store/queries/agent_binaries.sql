-- name: GetLatestAgentBinary :one
SELECT id, os_family, arch, version, download_url, checksum, released_at
FROM agent_binaries
WHERE os_family = $1 AND arch = $2
ORDER BY released_at DESC
LIMIT 1;
