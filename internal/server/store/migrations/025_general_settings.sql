-- +goose Up
ALTER TABLE tenant_settings ADD COLUMN org_name TEXT NOT NULL DEFAULT '';
ALTER TABLE tenant_settings ADD COLUMN timezone TEXT NOT NULL DEFAULT 'UTC';
ALTER TABLE tenant_settings ADD COLUMN date_format TEXT NOT NULL DEFAULT 'YYYY-MM-DD';
ALTER TABLE tenant_settings ADD COLUMN scan_interval_hours INTEGER NOT NULL DEFAULT 6;

-- +goose Down
ALTER TABLE tenant_settings DROP COLUMN IF EXISTS scan_interval_hours;
ALTER TABLE tenant_settings DROP COLUMN IF EXISTS date_format;
ALTER TABLE tenant_settings DROP COLUMN IF EXISTS timezone;
ALTER TABLE tenant_settings DROP COLUMN IF EXISTS org_name;
