-- +goose Up
CREATE TABLE IF NOT EXISTS agent_binaries (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    os_family   TEXT NOT NULL,
    arch        TEXT NOT NULL,
    version     TEXT NOT NULL,
    download_url TEXT NOT NULL,
    checksum    TEXT NOT NULL,
    released_at TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (os_family, arch, version)
);

-- +goose Down
DROP TABLE IF EXISTS agent_binaries CASCADE;
