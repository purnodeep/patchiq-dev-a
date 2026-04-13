-- name: LinkCatalogCVE :exec
INSERT INTO patch_catalog_cves (catalog_id, cve_id)
VALUES ($1, $2)
ON CONFLICT (catalog_id, cve_id) DO NOTHING;

-- name: UnlinkAllCatalogCVEs :exec
DELETE FROM patch_catalog_cves WHERE catalog_id = $1;

-- name: ListCVEsForCatalogEntry :many
SELECT cf.id, cf.cve_id, cf.severity, cf.description, cf.published_at, cf.source,
       cf.cvss_v3_score, cf.exploit_known, cf.in_kev,
       cf.cvss_v3_vector, cf.attack_vector, cf.cwe_id, cf.cisa_kev_due_date,
       cf.nvd_last_modified, cf.external_references
FROM cve_feeds cf
JOIN patch_catalog_cves pcc ON pcc.cve_id = cf.id
WHERE pcc.catalog_id = $1
ORDER BY cf.severity, cf.cve_id
LIMIT 1000;

-- name: ListCatalogCVELinks :many
SELECT pcc.catalog_id, cf.cve_id
FROM patch_catalog_cves pcc
JOIN cve_feeds cf ON pcc.cve_id = cf.id
WHERE pcc.catalog_id = ANY(@catalog_ids::uuid[])
ORDER BY pcc.catalog_id, cf.cve_id;

-- name: CountCVEsForCatalogEntry :one
SELECT count(*) FROM patch_catalog_cves WHERE catalog_id = $1;
