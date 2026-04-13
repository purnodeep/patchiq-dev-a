-- +goose Up
ALTER TABLE patch_catalog ADD COLUMN silent_args TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE patch_catalog DROP COLUMN IF EXISTS silent_args;
