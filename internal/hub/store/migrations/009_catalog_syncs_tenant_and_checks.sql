-- +goose Up

-- Add tenant_id to catalog_entry_syncs for multi-tenancy isolation.
ALTER TABLE catalog_entry_syncs
    ADD COLUMN tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001';

-- Remove default after backfill.
ALTER TABLE catalog_entry_syncs ALTER COLUMN tenant_id DROP DEFAULT;

CREATE INDEX idx_catalog_entry_syncs_tenant ON catalog_entry_syncs(tenant_id);

ALTER TABLE catalog_entry_syncs ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON catalog_entry_syncs
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- Add CHECK constraints on status columns.
ALTER TABLE catalog_entry_syncs
    ADD CONSTRAINT chk_catalog_entry_syncs_status
    CHECK (status IN ('pending', 'synced', 'failed'));

ALTER TABLE feed_sync_history
    ADD CONSTRAINT chk_feed_sync_history_status
    CHECK (status IN ('running', 'success', 'failed'));

-- +goose Down
DROP POLICY IF EXISTS tenant_isolation ON catalog_entry_syncs;
ALTER TABLE catalog_entry_syncs DISABLE ROW LEVEL SECURITY;
ALTER TABLE catalog_entry_syncs DROP CONSTRAINT IF EXISTS chk_catalog_entry_syncs_status;
ALTER TABLE catalog_entry_syncs DROP COLUMN IF EXISTS tenant_id;
DROP INDEX IF EXISTS idx_catalog_entry_syncs_tenant;
ALTER TABLE feed_sync_history DROP CONSTRAINT IF EXISTS chk_feed_sync_history_status;
