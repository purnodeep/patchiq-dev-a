-- +goose Up
ALTER TABLE patches ADD COLUMN IF NOT EXISTS released_at TIMESTAMPTZ;
-- Backfill: use created_at as a reasonable default for existing patches
UPDATE patches SET released_at = created_at WHERE released_at IS NULL;

-- +goose Down
ALTER TABLE patches DROP COLUMN IF EXISTS released_at;
