-- +goose Up
-- Allow discord as a notification channel type.
ALTER TABLE notification_channels
    DROP CONSTRAINT IF EXISTS chk_nc_channel_type;
ALTER TABLE notification_channels
    ADD CONSTRAINT chk_nc_channel_type CHECK (channel_type IN ('email', 'slack', 'teams', 'webhook', 'discord'));

-- +goose Down
ALTER TABLE notification_channels
    DROP CONSTRAINT IF EXISTS chk_nc_channel_type;
ALTER TABLE notification_channels
    ADD CONSTRAINT chk_nc_channel_type CHECK (channel_type IN ('email', 'slack', 'teams', 'webhook'));
