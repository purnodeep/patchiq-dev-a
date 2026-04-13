-- +goose Up
ALTER TABLE policies ADD COLUMN policy_type TEXT NOT NULL DEFAULT 'patch';
ALTER TABLE policies ADD CONSTRAINT chk_policy_type
    CHECK (policy_type IN ('patch', 'deploy', 'compliance'));

ALTER TABLE policies ADD COLUMN timezone TEXT NOT NULL DEFAULT 'UTC';

ALTER TABLE policies ADD COLUMN mw_enabled BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE policies DROP CONSTRAINT IF EXISTS chk_policy_type;
ALTER TABLE policies DROP COLUMN IF EXISTS mw_enabled;
ALTER TABLE policies DROP COLUMN IF EXISTS timezone;
ALTER TABLE policies DROP COLUMN IF EXISTS policy_type;
