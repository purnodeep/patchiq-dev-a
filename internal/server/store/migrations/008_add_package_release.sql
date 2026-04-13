-- +goose Up
ALTER TABLE endpoint_packages ADD COLUMN release TEXT;

-- +goose Down
ALTER TABLE endpoint_packages DROP COLUMN release;
