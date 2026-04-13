-- +goose Up

-- H-I1: Enrollment tokens must expire. Add expires_at with 7-day default TTL.
ALTER TABLE agent_registrations
    ADD COLUMN expires_at TIMESTAMPTZ;

-- Backfill existing tokens: set expiry to 7 days after creation.
UPDATE agent_registrations
SET expires_at = created_at + INTERVAL '7 days'
WHERE expires_at IS NULL;

-- Make NOT NULL going forward.
ALTER TABLE agent_registrations
    ALTER COLUMN expires_at SET NOT NULL,
    ALTER COLUMN expires_at SET DEFAULT now() + INTERVAL '7 days';

-- Index for efficient expiry lookups.
CREATE INDEX idx_agent_registrations_expires
    ON agent_registrations (expires_at)
    WHERE status = 'pending';

-- H-I5: Track when endpoint last received config to enable config push via heartbeat.
ALTER TABLE endpoints
    ADD COLUMN config_pushed_at TIMESTAMPTZ;

-- +goose Down

DROP INDEX IF EXISTS idx_agent_registrations_expires;
ALTER TABLE agent_registrations DROP COLUMN IF EXISTS expires_at;
ALTER TABLE endpoints DROP COLUMN IF EXISTS config_pushed_at;
