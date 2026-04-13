-- +goose Up

-- ============================================================
-- Feed sources registry (global, no tenant_id)
-- ============================================================

CREATE TABLE feed_sources (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                  TEXT NOT NULL UNIQUE,
    display_name          TEXT NOT NULL,
    enabled               BOOLEAN NOT NULL DEFAULT true,
    sync_interval_seconds INT NOT NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- Feed sync state (global, one row per feed source)
-- ============================================================

CREATE TABLE feed_sync_state (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    feed_source_id   UUID NOT NULL REFERENCES feed_sources(id) UNIQUE,
    last_sync_at     TIMESTAMPTZ,
    next_sync_at     TIMESTAMPTZ,
    cursor           TEXT NOT NULL DEFAULT '',
    entries_ingested BIGINT NOT NULL DEFAULT 0,
    error_count      INT NOT NULL DEFAULT 0,
    last_error       TEXT,
    status           TEXT NOT NULL DEFAULT 'idle',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_feed_sync_state_feed_source ON feed_sync_state(feed_source_id);

-- ============================================================
-- Extend patch_catalog for feed aggregation
-- ============================================================

ALTER TABLE patch_catalog ADD COLUMN feed_source_id UUID REFERENCES feed_sources(id);
ALTER TABLE patch_catalog ADD COLUMN source_url TEXT NOT NULL DEFAULT '';
ALTER TABLE patch_catalog ADD COLUMN installer_type TEXT NOT NULL DEFAULT '';

-- Dedup index: prevent duplicate entries from the same feed source.
CREATE UNIQUE INDEX idx_patch_catalog_feed_dedup
    ON patch_catalog (feed_source_id, vendor, name, version)
    WHERE feed_source_id IS NOT NULL AND deleted_at IS NULL;

-- ============================================================
-- Seed feed sources
-- ============================================================

INSERT INTO feed_sources (name, display_name, sync_interval_seconds) VALUES
    ('nvd',         'National Vulnerability Database',          21600),
    ('cisa_kev',    'CISA Known Exploited Vulnerabilities',     43200),
    ('msrc',        'Microsoft Security Response Center',       86400),
    ('redhat_oval', 'Red Hat OVAL Definitions',                 86400),
    ('ubuntu_usn',  'Ubuntu Security Notices',                  86400),
    ('apple',       'Apple Security Updates',                   86400);

-- Create sync state rows for each feed source.
INSERT INTO feed_sync_state (feed_source_id)
    SELECT id FROM feed_sources;

-- +goose Down

DROP INDEX IF EXISTS idx_patch_catalog_feed_dedup;
ALTER TABLE patch_catalog DROP COLUMN IF EXISTS installer_type;
ALTER TABLE patch_catalog DROP COLUMN IF EXISTS source_url;
ALTER TABLE patch_catalog DROP COLUMN IF EXISTS feed_source_id;
DROP TABLE IF EXISTS feed_sync_state;
DROP TABLE IF EXISTS feed_sources;
