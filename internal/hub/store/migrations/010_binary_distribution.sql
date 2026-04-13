-- +goose Up

-- Binary distribution: store MinIO object key and SHA256 checksum for patch binaries.
ALTER TABLE patch_catalog ADD COLUMN binary_ref TEXT NOT NULL DEFAULT '';
ALTER TABLE patch_catalog ADD COLUMN checksum_sha256 TEXT NOT NULL DEFAULT '';

-- Index for quick lookup of entries with binaries available.
CREATE INDEX idx_patch_catalog_binary_ref ON patch_catalog (binary_ref) WHERE binary_ref != '';

-- +goose Down

DROP INDEX IF EXISTS idx_patch_catalog_binary_ref;
ALTER TABLE patch_catalog DROP COLUMN IF EXISTS checksum_sha256;
ALTER TABLE patch_catalog DROP COLUMN IF EXISTS binary_ref;
