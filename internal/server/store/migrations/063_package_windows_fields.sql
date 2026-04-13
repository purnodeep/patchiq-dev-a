-- +goose Up
-- ============================================================
-- Windows-specific package fields and collection error tracking
-- ============================================================

-- Add structured Windows fields to endpoint_packages.
ALTER TABLE endpoint_packages ADD COLUMN IF NOT EXISTS kb_article TEXT;
ALTER TABLE endpoint_packages ADD COLUMN IF NOT EXISTS severity TEXT;
ALTER TABLE endpoint_packages ADD COLUMN IF NOT EXISTS install_date TEXT;
ALTER TABLE endpoint_packages ADD COLUMN IF NOT EXISTS category TEXT;
ALTER TABLE endpoint_packages ADD COLUMN IF NOT EXISTS publisher TEXT;

-- Track collection errors on inventory snapshots so partial reports
-- are distinguishable from clean endpoints with no software.
ALTER TABLE endpoint_inventories ADD COLUMN IF NOT EXISTS collection_errors JSONB DEFAULT '[]';

-- Index for severity-based queries on endpoint packages.
CREATE INDEX IF NOT EXISTS idx_endpoint_packages_severity
    ON endpoint_packages(tenant_id, severity) WHERE severity IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_endpoint_packages_severity;
ALTER TABLE endpoint_inventories DROP COLUMN IF EXISTS collection_errors;
ALTER TABLE endpoint_packages DROP COLUMN IF EXISTS publisher;
ALTER TABLE endpoint_packages DROP COLUMN IF EXISTS category;
ALTER TABLE endpoint_packages DROP COLUMN IF EXISTS install_date;
ALTER TABLE endpoint_packages DROP COLUMN IF EXISTS severity;
ALTER TABLE endpoint_packages DROP COLUMN IF EXISTS kb_article;
