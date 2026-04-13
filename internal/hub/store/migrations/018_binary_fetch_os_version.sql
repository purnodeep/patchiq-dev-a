-- +goose Up
ALTER TABLE binary_fetch_state ADD COLUMN os_version TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE binary_fetch_state DROP COLUMN os_version;
