-- +goose Up

-- Store the actual package name (e.g. "curl", "openssl") separately from the
-- advisory name (e.g. "RHSA-2024:0893"). Feeds extract this as Product but it
-- was previously discarded.
ALTER TABLE patch_catalog ADD COLUMN product TEXT NOT NULL DEFAULT '';

-- +goose Down

ALTER TABLE patch_catalog DROP COLUMN IF EXISTS product;
