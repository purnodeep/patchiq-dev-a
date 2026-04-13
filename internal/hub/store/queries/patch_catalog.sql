-- name: CreateCatalogEntry :one
INSERT INTO patch_catalog (name, vendor, os_family, version, severity, release_date, description)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetCatalogEntryByID :one
SELECT * FROM patch_catalog WHERE id = $1 AND deleted_at IS NULL;

-- name: ListCatalogEntries :many
SELECT * FROM patch_catalog
WHERE deleted_at IS NULL
  AND (sqlc.narg('os_family')::text IS NULL OR os_family = sqlc.narg('os_family'))
  AND (sqlc.narg('severity')::text IS NULL OR severity = sqlc.narg('severity'))
  AND (sqlc.narg('search')::text IS NULL OR name ILIKE '%' || sqlc.narg('search') || '%')
ORDER BY created_at DESC
LIMIT sqlc.arg('query_limit') OFFSET sqlc.arg('query_offset');

-- name: CountCatalogEntries :one
SELECT count(*) FROM patch_catalog
WHERE deleted_at IS NULL
  AND (sqlc.narg('os_family')::text IS NULL OR os_family = sqlc.narg('os_family'))
  AND (sqlc.narg('severity')::text IS NULL OR severity = sqlc.narg('severity'))
  AND (sqlc.narg('search')::text IS NULL OR name ILIKE '%' || sqlc.narg('search') || '%');

-- name: UpdateCatalogEntry :one
UPDATE patch_catalog
SET name = $2, vendor = $3, os_family = $4, version = $5, severity = $6,
    release_date = $7, description = $8, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteCatalogEntry :exec
UPDATE patch_catalog SET deleted_at = now(), updated_at = now() WHERE id = $1 AND deleted_at IS NULL;

-- name: ListCatalogEntriesUpdatedSince :many
SELECT * FROM patch_catalog
WHERE updated_at > $1 AND deleted_at IS NULL
LIMIT 10000;

-- name: ListCatalogEntriesDeletedSince :many
SELECT id FROM patch_catalog
WHERE deleted_at > $1
LIMIT 10000;

-- name: UpsertCatalogEntryFromFeed :one
INSERT INTO patch_catalog (name, vendor, os_family, version, severity, release_date, description, feed_source_id, source_url, installer_type, binary_ref, checksum_sha256, product, os_package_name, silent_args)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
ON CONFLICT (feed_source_id, vendor, name, version)
    WHERE feed_source_id IS NOT NULL AND deleted_at IS NULL
DO UPDATE SET
    severity = EXCLUDED.severity,
    release_date = EXCLUDED.release_date,
    description = EXCLUDED.description,
    os_family = EXCLUDED.os_family,
    source_url = EXCLUDED.source_url,
    installer_type = EXCLUDED.installer_type,
    binary_ref = EXCLUDED.binary_ref,
    checksum_sha256 = EXCLUDED.checksum_sha256,
    product = EXCLUDED.product,
    os_package_name = EXCLUDED.os_package_name,
    silent_args = EXCLUDED.silent_args,
    updated_at = now()
RETURNING *;

-- name: UpdateCatalogEntryBinaryRef :exec
UPDATE patch_catalog
SET binary_ref = $2, checksum_sha256 = $3, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListCatalogEntriesEnriched :many
SELECT pc.*,
       fs.name AS feed_source_name,
       (SELECT count(*) FROM patch_catalog_cves WHERE catalog_id = pc.id) AS cve_count,
       (SELECT count(*) FROM catalog_entry_syncs WHERE catalog_id = pc.id AND status = 'synced') AS synced_count
FROM patch_catalog pc
LEFT JOIN feed_sources fs ON fs.id = pc.feed_source_id
WHERE pc.deleted_at IS NULL
  AND (sqlc.narg('os_family')::text IS NULL OR pc.os_family = sqlc.narg('os_family'))
  AND (sqlc.narg('severity')::text IS NULL OR pc.severity = sqlc.narg('severity'))
  AND (sqlc.narg('search')::text IS NULL OR pc.name ILIKE '%' || sqlc.narg('search') || '%')
  AND (sqlc.narg('feed_source_id')::uuid IS NULL OR pc.feed_source_id = sqlc.narg('feed_source_id'))
  AND (sqlc.narg('date_range')::text IS NULL
       OR (sqlc.narg('date_range') = '7d' AND pc.release_date > now() - interval '7 days')
       OR (sqlc.narg('date_range') = '30d' AND pc.release_date > now() - interval '30 days')
       OR (sqlc.narg('date_range') = '90d' AND pc.release_date > now() - interval '90 days'))
  AND (sqlc.narg('entry_type')::text IS NULL
       OR (sqlc.narg('entry_type') = 'cve' AND pc.name ILIKE 'CVE-%')
       OR (sqlc.narg('entry_type') = 'patch' AND pc.name NOT ILIKE 'CVE-%'))
ORDER BY pc.created_at DESC
LIMIT sqlc.arg('query_limit') OFFSET sqlc.arg('query_offset');

-- name: CountCatalogEntriesEnriched :one
SELECT count(*)
FROM patch_catalog pc
WHERE pc.deleted_at IS NULL
  AND (sqlc.narg('os_family')::text IS NULL OR pc.os_family = sqlc.narg('os_family'))
  AND (sqlc.narg('severity')::text IS NULL OR pc.severity = sqlc.narg('severity'))
  AND (sqlc.narg('search')::text IS NULL OR pc.name ILIKE '%' || sqlc.narg('search') || '%')
  AND (sqlc.narg('feed_source_id')::uuid IS NULL OR pc.feed_source_id = sqlc.narg('feed_source_id'))
  AND (sqlc.narg('date_range')::text IS NULL
       OR (sqlc.narg('date_range') = '7d' AND pc.release_date > now() - interval '7 days')
       OR (sqlc.narg('date_range') = '30d' AND pc.release_date > now() - interval '30 days')
       OR (sqlc.narg('date_range') = '90d' AND pc.release_date > now() - interval '90 days'))
  AND (sqlc.narg('entry_type')::text IS NULL
       OR (sqlc.narg('entry_type') = 'cve' AND pc.name ILIKE 'CVE-%')
       OR (sqlc.narg('entry_type') = 'patch' AND pc.name NOT ILIKE 'CVE-%'));
