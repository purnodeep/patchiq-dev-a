-- +goose Up
ALTER TABLE iam_settings ADD COLUMN sso_url TEXT NOT NULL DEFAULT '';
ALTER TABLE iam_settings ADD COLUMN client_id_encrypted BYTEA;
ALTER TABLE iam_settings ADD COLUMN last_test_status TEXT;
ALTER TABLE iam_settings ADD COLUMN last_tested_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE iam_settings DROP COLUMN IF EXISTS last_tested_at;
ALTER TABLE iam_settings DROP COLUMN IF EXISTS last_test_status;
ALTER TABLE iam_settings DROP COLUMN IF EXISTS client_id_encrypted;
ALTER TABLE iam_settings DROP COLUMN IF EXISTS sso_url;
