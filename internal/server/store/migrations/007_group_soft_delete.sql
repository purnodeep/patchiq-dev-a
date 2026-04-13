-- +goose Up
ALTER TABLE endpoint_groups ADD COLUMN deleted_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE endpoint_groups DROP COLUMN deleted_at;
