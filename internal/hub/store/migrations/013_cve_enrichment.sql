-- +goose Up

-- Rename cvss_score → cvss_v3_score for PM consistency.
-- sqlc cannot parse RENAME COLUMN, so we drop and re-add.
ALTER TABLE cve_feeds DROP COLUMN IF EXISTS cvss_score;
ALTER TABLE cve_feeds ADD COLUMN cvss_v3_score NUMERIC(3,1);

-- Add enrichment columns to cve_feeds.
ALTER TABLE cve_feeds
    ADD COLUMN cvss_v3_vector      TEXT,
    ADD COLUMN attack_vector        TEXT,
    ADD COLUMN cwe_id               TEXT,
    ADD COLUMN cisa_kev_due_date    DATE,
    ADD COLUMN external_references  JSONB DEFAULT '[]'::jsonb,
    ADD COLUMN nvd_last_modified    TIMESTAMPTZ;

-- +goose Down

ALTER TABLE cve_feeds
    DROP COLUMN IF EXISTS nvd_last_modified,
    DROP COLUMN IF EXISTS external_references,
    DROP COLUMN IF EXISTS cisa_kev_due_date,
    DROP COLUMN IF EXISTS cwe_id,
    DROP COLUMN IF EXISTS attack_vector,
    DROP COLUMN IF EXISTS cvss_v3_vector,
    DROP COLUMN IF EXISTS cvss_v3_score;

ALTER TABLE cve_feeds ADD COLUMN cvss_score NUMERIC(3,1);
