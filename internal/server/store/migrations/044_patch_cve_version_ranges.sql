-- +goose Up
-- Add version range columns to patch_cves so the CVE matcher can perform
-- accurate version-range comparison instead of treating all linked CVEs as
-- affecting every version.
ALTER TABLE patch_cves
    ADD COLUMN version_end_excluding TEXT NOT NULL DEFAULT '',
    ADD COLUMN version_end_including TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE patch_cves
    DROP COLUMN IF EXISTS version_end_excluding,
    DROP COLUMN IF EXISTS version_end_including;
