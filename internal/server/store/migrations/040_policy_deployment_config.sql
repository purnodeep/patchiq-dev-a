-- +goose Up

-- Deployment configuration for automatic-mode policies.
-- Stores wave config, reboot preferences, cron schedule, etc.
ALTER TABLE policies ADD COLUMN deployment_config JSONB;

-- +goose Down
ALTER TABLE policies DROP COLUMN IF EXISTS deployment_config;
