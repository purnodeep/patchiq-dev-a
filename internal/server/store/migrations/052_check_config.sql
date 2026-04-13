-- +goose Up
ALTER TABLE custom_compliance_controls ADD COLUMN IF NOT EXISTS check_config JSONB NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE custom_compliance_controls DROP COLUMN IF EXISTS check_config;
