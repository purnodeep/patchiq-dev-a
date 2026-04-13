-- +goose Up

-- Soft delete support for catalog entries.
ALTER TABLE patch_catalog ADD COLUMN deleted_at TIMESTAMPTZ;

-- Index for delta sync queries (updated_at filter).
CREATE INDEX idx_patch_catalog_updated_at ON patch_catalog(updated_at);

-- Index for filtering active-only entries.
CREATE INDEX idx_patch_catalog_deleted_at ON patch_catalog(deleted_at);

-- Many-to-many link between catalog entries and CVE feeds.
CREATE TABLE patch_catalog_cves (
    catalog_id UUID NOT NULL REFERENCES patch_catalog(id) ON DELETE CASCADE,
    cve_id     UUID NOT NULL REFERENCES cve_feeds(id) ON DELETE CASCADE,
    PRIMARY KEY (catalog_id, cve_id)
);

CREATE INDEX idx_patch_catalog_cves_cve ON patch_catalog_cves(cve_id);

-- +goose Down

DROP TABLE IF EXISTS patch_catalog_cves;
DROP INDEX IF EXISTS idx_patch_catalog_deleted_at;
DROP INDEX IF EXISTS idx_patch_catalog_updated_at;
ALTER TABLE patch_catalog DROP COLUMN IF EXISTS deleted_at;
