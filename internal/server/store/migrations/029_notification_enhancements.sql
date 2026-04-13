-- +goose Up

-- ============================================================
-- 1. Expand notification_preferences
-- ============================================================

-- 1a. Add new per-channel toggle columns
ALTER TABLE notification_preferences
    ADD COLUMN email_enabled   BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN slack_enabled   BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN webhook_enabled BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN urgency         TEXT    NOT NULL DEFAULT 'digest';

-- 1b. Data-migrate: set per-channel booleans from channel_ids
--     Join against notification_channels to determine channel type.
--     WHERE np.enabled = true: skip rows that were already disabled — their
--     per-channel flags remain false (the safe default for a disabled preference).
UPDATE notification_preferences np
SET
    email_enabled   = EXISTS (
        SELECT 1 FROM notification_channels nc
        WHERE nc.id = ANY(np.channel_ids)
          AND nc.channel_type = 'email'
          AND nc.tenant_id = np.tenant_id
    ),
    slack_enabled   = EXISTS (
        SELECT 1 FROM notification_channels nc
        WHERE nc.id = ANY(np.channel_ids)
          AND nc.channel_type = 'slack'
          AND nc.tenant_id = np.tenant_id
    ),
    webhook_enabled = EXISTS (
        SELECT 1 FROM notification_channels nc
        WHERE nc.id = ANY(np.channel_ids)
          AND nc.channel_type = 'webhook'
          AND nc.tenant_id = np.tenant_id
    )
WHERE np.enabled = true;

-- 1c. Add urgency CHECK
ALTER TABLE notification_preferences
    ADD CONSTRAINT chk_np_urgency CHECK (urgency IN ('immediate', 'digest'));

-- 1d. Drop old columns
ALTER TABLE notification_preferences
    DROP COLUMN IF EXISTS channel_ids,
    DROP COLUMN IF EXISTS enabled,
    DROP COLUMN IF EXISTS digest_frequency;

-- 1e. Drop old trigger_type CHECK and recreate with all 18 values (16 UI + 2 legacy)
ALTER TABLE notification_preferences
    DROP CONSTRAINT IF EXISTS chk_np_trigger_type;

ALTER TABLE notification_preferences
    ADD CONSTRAINT chk_np_trigger_type CHECK (trigger_type IN (
        'deployment.started',
        'deployment.completed',
        'deployment.failed',
        'deployment.rollback_initiated',
        'compliance.threshold_breach',
        'compliance.evaluation_complete',
        'compliance.control_failed',
        'compliance.sla_approaching',
        'compliance.sla_overdue',
        'cve.critical_discovered',
        'cve.exploit_detected',
        'cve.kev_added',
        'cve.patch_available',
        'agent.disconnected',
        'agent.offline',
        'system.hub_sync_failed',
        'system.license_expiring',
        'system.scan_completed'
    ));

-- ============================================================
-- 2. Expand notification_history
-- ============================================================

-- 2a. Add new columns
ALTER TABLE notification_history
    ADD COLUMN channel_type TEXT, -- intentionally unconstrained: tolerates future channel types added without migration
    ADD COLUMN recipient    TEXT NOT NULL DEFAULT '',
    ADD COLUMN subject      TEXT NOT NULL DEFAULT '',
    ADD COLUMN retry_count  INTEGER NOT NULL DEFAULT 0;

-- 2b. Expand status CHECK (drop old, add new)
ALTER TABLE notification_history
    DROP CONSTRAINT IF EXISTS chk_nh_status;

ALTER TABLE notification_history
    ADD CONSTRAINT chk_nh_status CHECK (status IN ('sent', 'failed', 'pending', 'delivered'));

-- 2c. Grant UPDATE to patchiq_app (retry endpoint needs it)
GRANT UPDATE ON notification_history TO patchiq_app;

-- ============================================================
-- 3. Create notification_digest_config table
-- ============================================================

CREATE TABLE notification_digest_config (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id),
    frequency     TEXT NOT NULL DEFAULT 'daily',
    delivery_time TIME NOT NULL DEFAULT '09:00',
    format        TEXT NOT NULL DEFAULT 'html',
    UNIQUE(tenant_id),
    CONSTRAINT chk_ndc_frequency CHECK (frequency IN ('daily', 'weekly')),
    CONSTRAINT chk_ndc_format    CHECK (format IN ('html', 'plaintext'))
);

-- Index omitted: UNIQUE(tenant_id) already creates an implicit index on tenant_id.

ALTER TABLE notification_digest_config ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON notification_digest_config
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
ALTER TABLE notification_digest_config FORCE ROW LEVEL SECURITY;

GRANT SELECT, INSERT, UPDATE, DELETE ON notification_digest_config TO patchiq_app;

-- +goose Down

-- Revoke UPDATE grant added in Up
REVOKE UPDATE ON notification_history FROM patchiq_app;

-- 3. Drop digest config
DROP TABLE IF EXISTS notification_digest_config CASCADE;

-- 2. Revert notification_history
ALTER TABLE notification_history
    DROP CONSTRAINT IF EXISTS chk_nh_status;
ALTER TABLE notification_history
    ADD CONSTRAINT chk_nh_status CHECK (status IN ('sent', 'failed'));

ALTER TABLE notification_history
    DROP COLUMN IF EXISTS channel_type,
    DROP COLUMN IF EXISTS recipient,
    DROP COLUMN IF EXISTS subject,
    DROP COLUMN IF EXISTS retry_count;

-- 1. Revert notification_preferences
ALTER TABLE notification_preferences
    DROP CONSTRAINT IF EXISTS chk_np_trigger_type;
ALTER TABLE notification_preferences
    ADD CONSTRAINT chk_np_trigger_type CHECK (trigger_type IN (
        'deployment.started',
        'deployment.completed',
        'deployment.failed',
        'compliance.threshold_breach',
        'agent.disconnected',
        'cve.critical_discovered'
    ));

ALTER TABLE notification_preferences
    DROP CONSTRAINT IF EXISTS chk_np_urgency;

-- Re-add old columns
ALTER TABLE notification_preferences
    ADD COLUMN channel_ids      UUID[]  NOT NULL DEFAULT '{}',
    ADD COLUMN enabled          BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN digest_frequency TEXT    NOT NULL DEFAULT 'realtime';

ALTER TABLE notification_preferences
    ADD CONSTRAINT chk_np_digest_frequency CHECK (digest_frequency IN ('realtime', 'daily', 'weekly'));

ALTER TABLE notification_preferences
    DROP COLUMN IF EXISTS email_enabled,
    DROP COLUMN IF EXISTS slack_enabled,
    DROP COLUMN IF EXISTS webhook_enabled,
    DROP COLUMN IF EXISTS urgency;
