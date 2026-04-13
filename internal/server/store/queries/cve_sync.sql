-- name: GetCVESyncCursor :one
SELECT * FROM cve_sync_cursors
WHERE tenant_id = $1 AND source = $2;

-- name: UpsertCVESyncCursor :one
INSERT INTO cve_sync_cursors (tenant_id, source, last_synced)
VALUES ($1, $2, $3)
ON CONFLICT (tenant_id, source)
DO UPDATE SET last_synced = EXCLUDED.last_synced, updated_at = now()
RETURNING *;

-- name: ListCVEsByPackageName :many
SELECT c.*,
       pc.version_end_excluding,
       pc.version_end_including
FROM cves c
JOIN patch_cves pc ON c.id = pc.cve_id AND c.tenant_id = pc.tenant_id
JOIN patches p ON pc.patch_id = p.id AND pc.tenant_id = p.tenant_id
WHERE p.package_name = $1 AND p.tenant_id = $2
LIMIT 1000;

-- name: ListPatchesByName :many
SELECT * FROM patches
WHERE package_name = $1 AND tenant_id = $2
LIMIT 1000;

-- name: GetCVEDBIDsByCVEIDs :many
SELECT id, cve_id FROM cves
WHERE tenant_id = $1 AND cve_id = ANY(@cve_ids::text[]);

-- name: ListCVEsByOsFamily :many
SELECT c.id, c.cve_id, c.severity, c.cvss_v3_score, c.cisa_kev_due_date,
       c.exploit_available, c.published_at, '' AS version_end_excluding, '' AS version_end_including
FROM cves c
WHERE c.tenant_id = $1
  AND c.description ILIKE '%' || $2 || '%'
LIMIT 1000;
