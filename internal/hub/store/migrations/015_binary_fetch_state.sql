-- +goose Up

CREATE TABLE binary_fetch_state (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    catalog_id       UUID NOT NULL REFERENCES patch_catalog(id) ON DELETE CASCADE,
    os_distribution  TEXT NOT NULL DEFAULT '',
    status           TEXT NOT NULL DEFAULT 'pending',
    binary_ref       TEXT,
    checksum_sha256  TEXT,
    file_size_bytes  BIGINT,
    fetch_url        TEXT,
    error_message    TEXT,
    retry_count      INT NOT NULL DEFAULT 0,
    last_attempt_at  TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (catalog_id, os_distribution)
);

CREATE INDEX idx_binary_fetch_state_status ON binary_fetch_state(status);
CREATE INDEX idx_binary_fetch_state_catalog ON binary_fetch_state(catalog_id);

-- +goose Down

DROP TABLE IF EXISTS binary_fetch_state;
