-- +goose Up

CREATE TABLE feed_sync_history (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    feed_source_id  UUID NOT NULL REFERENCES feed_sources(id) ON DELETE CASCADE,
    started_at      TIMESTAMPTZ NOT NULL,
    finished_at     TIMESTAMPTZ,
    duration_ms     INT,
    new_entries     INT NOT NULL DEFAULT 0,
    updated_entries INT NOT NULL DEFAULT 0,
    total_scanned   INT NOT NULL DEFAULT 0,
    error_count     INT NOT NULL DEFAULT 0,
    status          TEXT NOT NULL DEFAULT 'running',
    error_message   TEXT,
    log_output      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_feed_sync_history_feed ON feed_sync_history(feed_source_id, started_at DESC);

ALTER TABLE feed_sources
    ADD COLUMN auth_type TEXT NOT NULL DEFAULT 'none',
    ADD COLUMN severity_filter TEXT[] NOT NULL DEFAULT '{critical,high,medium,low}',
    ADD COLUMN os_filter TEXT[] NOT NULL DEFAULT '{windows,ubuntu,rhel,debian}',
    ADD COLUMN severity_mapping JSONB NOT NULL DEFAULT '{"critical":"critical","high":"high","medium":"medium","low":"low"}',
    ADD COLUMN url TEXT NOT NULL DEFAULT '';

ALTER TABLE cve_feeds
    ADD COLUMN cvss_score NUMERIC(3,1),
    ADD COLUMN exploit_known BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN in_kev BOOLEAN NOT NULL DEFAULT false;

-- +goose Down

ALTER TABLE cve_feeds
    DROP COLUMN IF EXISTS cvss_score,
    DROP COLUMN IF EXISTS exploit_known,
    DROP COLUMN IF EXISTS in_kev;

ALTER TABLE feed_sources
    DROP COLUMN IF EXISTS auth_type,
    DROP COLUMN IF EXISTS severity_filter,
    DROP COLUMN IF EXISTS os_filter,
    DROP COLUMN IF EXISTS severity_mapping,
    DROP COLUMN IF EXISTS url;

DROP TABLE IF EXISTS feed_sync_history;
