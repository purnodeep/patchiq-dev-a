-- +goose Up
ALTER TABLE cves ADD COLUMN IF NOT EXISTS attack_vector TEXT;
ALTER TABLE cves ADD COLUMN IF NOT EXISTS external_references JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE cves ADD COLUMN IF NOT EXISTS cwe_id TEXT;
ALTER TABLE cves ADD COLUMN IF NOT EXISTS source TEXT NOT NULL DEFAULT 'NVD';

-- +goose StatementBegin
DO $$ BEGIN
    ALTER TABLE cves ADD CONSTRAINT chk_cves_attack_vector
        CHECK (attack_vector IS NULL OR attack_vector IN ('Network', 'Adjacent', 'Local', 'Physical'));
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose Down
ALTER TABLE cves DROP CONSTRAINT IF EXISTS chk_cves_attack_vector;
ALTER TABLE cves DROP COLUMN IF EXISTS source;
ALTER TABLE cves DROP COLUMN IF EXISTS cwe_id;
ALTER TABLE cves DROP COLUMN IF EXISTS external_references;
ALTER TABLE cves DROP COLUMN IF EXISTS attack_vector;
