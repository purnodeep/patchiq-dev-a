-- +goose Up
ALTER TABLE notification_channels ADD COLUMN last_tested_at TIMESTAMPTZ;
ALTER TABLE notification_channels ADD COLUMN last_test_status TEXT;

-- +goose Down
ALTER TABLE notification_channels DROP COLUMN IF EXISTS last_test_status;
ALTER TABLE notification_channels DROP COLUMN IF EXISTS last_tested_at;
