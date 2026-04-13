-- +goose Up

CREATE TABLE package_aliases (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    feed_product     TEXT NOT NULL,
    os_family        TEXT NOT NULL,
    os_distribution  TEXT NOT NULL,
    os_package_name  TEXT NOT NULL,
    confidence       TEXT NOT NULL DEFAULT 'manual',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (feed_product, os_family, os_distribution)
);

CREATE INDEX idx_package_aliases_product ON package_aliases(feed_product);
CREATE INDEX idx_package_aliases_os ON package_aliases(os_family, os_distribution);

-- +goose Down

DROP TABLE IF EXISTS package_aliases;
