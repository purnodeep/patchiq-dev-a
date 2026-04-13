-- +goose Up
-- +goose StatementBegin
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='hub_sync_state' AND column_name='last_cve_sync_at') THEN
    ALTER TABLE hub_sync_state ADD COLUMN last_cve_sync_at TIMESTAMPTZ;
  END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
ALTER TABLE hub_sync_state DROP COLUMN IF EXISTS last_cve_sync_at;
