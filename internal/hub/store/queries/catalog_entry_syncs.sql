-- name: CountSyncedClientsForCatalogEntry :one
SELECT count(*) FROM catalog_entry_syncs
WHERE catalog_id = $1 AND status = 'synced';

-- name: CountApprovedClients :one
SELECT count(*) FROM clients WHERE status = 'approved';

-- name: ListSyncsForCatalogEntry :many
-- No JOIN on clients (RLS blocks it). Client info resolved in handler.
SELECT id, catalog_id, client_id, status, synced_at, created_at
FROM catalog_entry_syncs
WHERE catalog_id = $1
ORDER BY synced_at NULLS LAST
LIMIT 1000;

-- name: ListApprovedClientsBasic :many
-- Separate query for client info (runs with tenant context from middleware).
SELECT id, hostname, endpoint_count
FROM clients
WHERE status = 'approved'
ORDER BY hostname
LIMIT 1000;

-- name: GetCatalogStats :one
SELECT
    (SELECT count(*) FROM patch_catalog WHERE deleted_at IS NULL) AS total_entries,
    (SELECT count(*) FROM patch_catalog WHERE deleted_at IS NULL AND created_at > now() - interval '7 days') AS new_this_week,
    (SELECT count(*) FROM patch_catalog_cves pcc JOIN patch_catalog pc ON pc.id = pcc.catalog_id WHERE pc.deleted_at IS NULL) AS cves_tracked,
    (SELECT count(DISTINCT ces.catalog_id) FROM catalog_entry_syncs ces JOIN patch_catalog pc ON pc.id = ces.catalog_id WHERE pc.deleted_at IS NULL AND ces.status = 'synced') AS synced_entries,
    (SELECT count(*) FROM patch_catalog WHERE deleted_at IS NULL) AS total_for_sync_pct,
    (SELECT count(*) FROM patch_catalog WHERE deleted_at IS NULL AND severity = 'critical') AS critical_count,
    (SELECT count(*) FROM patch_catalog WHERE deleted_at IS NULL AND severity = 'high') AS high_count,
    (SELECT count(*) FROM patch_catalog WHERE deleted_at IS NULL AND severity = 'medium') AS medium_count,
    (SELECT count(*) FROM patch_catalog WHERE deleted_at IS NULL AND severity = 'low') AS low_count;
