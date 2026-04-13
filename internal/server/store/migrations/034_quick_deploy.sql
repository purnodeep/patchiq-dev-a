-- +goose Up
-- Allow deployments created without a policy (quick/ad-hoc deploys).
-- Also add human-readable name and direct patch reference.
ALTER TABLE deployments
    ALTER COLUMN policy_id DROP NOT NULL,
    ADD COLUMN IF NOT EXISTS name    TEXT,
    ADD COLUMN IF NOT EXISTS patch_id UUID REFERENCES patches(id);

-- +goose Down
ALTER TABLE deployments
    ALTER COLUMN policy_id SET NOT NULL,
    DROP COLUMN IF EXISTS name,
    DROP COLUMN IF EXISTS patch_id;
