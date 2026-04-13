-- +goose Up
-- Migration 037 was applied with key/value/color columns but was later changed
-- to use name/description only. This migration reconciles the schema.
ALTER TABLE tags RENAME COLUMN key TO name;
ALTER TABLE tags DROP COLUMN IF EXISTS value;
ALTER TABLE tags DROP COLUMN IF EXISTS color;

DROP INDEX IF EXISTS idx_tags_tenant_key;
ALTER TABLE tags DROP CONSTRAINT IF EXISTS tags_tenant_id_key_value_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_tags_tenant_name ON tags (tenant_id, lower(name));

-- +goose Down
ALTER TABLE tags RENAME COLUMN name TO key;
ALTER TABLE tags ADD COLUMN IF NOT EXISTS value TEXT NOT NULL DEFAULT '';
ALTER TABLE tags ADD COLUMN IF NOT EXISTS color TEXT;
DROP INDEX IF EXISTS idx_tags_tenant_name;
CREATE INDEX IF NOT EXISTS idx_tags_tenant_key ON tags (tenant_id, key);
