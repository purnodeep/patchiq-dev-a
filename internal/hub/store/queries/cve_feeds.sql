-- name: CreateCVEFeed :one
INSERT INTO cve_feeds (
    cve_id, severity, description, published_at, source,
    cvss_v3_score, cvss_v3_vector, attack_vector, cwe_id,
    cisa_kev_due_date, external_references, nvd_last_modified
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;

-- name: UpsertCVEFeed :one
INSERT INTO cve_feeds (
    cve_id, severity, description, published_at, source,
    cvss_v3_score, cvss_v3_vector, attack_vector, cwe_id,
    cisa_kev_due_date, external_references, nvd_last_modified,
    exploit_known, in_kev
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
ON CONFLICT (cve_id) DO UPDATE SET
    severity           = COALESCE(NULLIF(EXCLUDED.severity, ''), cve_feeds.severity),
    description        = COALESCE(EXCLUDED.description, cve_feeds.description),
    published_at       = COALESCE(EXCLUDED.published_at, cve_feeds.published_at),
    source             = COALESCE(NULLIF(EXCLUDED.source, ''), cve_feeds.source),
    cvss_v3_score      = COALESCE(EXCLUDED.cvss_v3_score, cve_feeds.cvss_v3_score),
    cvss_v3_vector     = COALESCE(NULLIF(EXCLUDED.cvss_v3_vector, ''), cve_feeds.cvss_v3_vector),
    attack_vector      = COALESCE(NULLIF(EXCLUDED.attack_vector, ''), cve_feeds.attack_vector),
    cwe_id             = COALESCE(NULLIF(EXCLUDED.cwe_id, ''), cve_feeds.cwe_id),
    cisa_kev_due_date  = COALESCE(EXCLUDED.cisa_kev_due_date, cve_feeds.cisa_kev_due_date),
    external_references = CASE
        WHEN EXCLUDED.external_references IS NOT NULL AND EXCLUDED.external_references != '[]'::jsonb
        THEN EXCLUDED.external_references
        ELSE cve_feeds.external_references
    END,
    nvd_last_modified  = COALESCE(EXCLUDED.nvd_last_modified, cve_feeds.nvd_last_modified),
    exploit_known      = EXCLUDED.exploit_known OR cve_feeds.exploit_known,
    in_kev             = EXCLUDED.in_kev OR cve_feeds.in_kev,
    updated_at         = now()
RETURNING *;

-- name: GetCVEFeedByID :one
SELECT * FROM cve_feeds WHERE id = $1;

-- name: GetCVEFeedByCVEID :one
SELECT * FROM cve_feeds WHERE cve_id = $1;

-- name: ListCVEFeeds :many
SELECT * FROM cve_feeds ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListCVEFeedsUpdatedSince :many
SELECT * FROM cve_feeds WHERE updated_at > $1
ORDER BY updated_at ASC
LIMIT $2;

-- name: UpdateCVEFeed :one
UPDATE cve_feeds
SET severity = $2, description = $3, published_at = $4, source = $5, updated_at = now()
WHERE id = $1
RETURNING *;
