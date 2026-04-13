-- +goose Up
ALTER TABLE endpoints ADD COLUMN IF NOT EXISTS cpu_usage_percent SMALLINT;

-- +goose Down
ALTER TABLE endpoints DROP COLUMN IF EXISTS cpu_usage_percent;
