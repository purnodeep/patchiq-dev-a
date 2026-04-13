-- +goose Up

-- ============================================================
-- Notification tables: channels, preferences, history
-- ============================================================

CREATE TABLE notification_channels (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL REFERENCES tenants(id),
    name             TEXT NOT NULL,
    channel_type     TEXT NOT NULL,
    config_encrypted BYTEA NOT NULL,
    enabled          BOOLEAN NOT NULL DEFAULT true,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_nc_name_not_empty CHECK (name <> ''),
    CONSTRAINT chk_nc_channel_type CHECK (channel_type IN ('email', 'slack', 'teams', 'webhook'))
);

CREATE INDEX idx_notification_channels_tenant ON notification_channels(tenant_id);

CREATE TABLE notification_preferences (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL REFERENCES tenants(id),
    user_id          TEXT NOT NULL,
    trigger_type     TEXT NOT NULL,
    enabled          BOOLEAN NOT NULL DEFAULT true,
    channel_ids      UUID[] NOT NULL DEFAULT '{}',
    digest_frequency TEXT NOT NULL DEFAULT 'realtime',
    UNIQUE (tenant_id, user_id, trigger_type),
    CONSTRAINT chk_np_user_id_not_empty CHECK (user_id <> ''),
    CONSTRAINT chk_np_trigger_type CHECK (trigger_type IN (
        'deployment.started',
        'deployment.completed',
        'deployment.failed',
        'compliance.threshold_breach',
        'agent.disconnected',
        'cve.critical_discovered'
    )),
    CONSTRAINT chk_np_digest_frequency CHECK (digest_frequency IN ('realtime', 'daily', 'weekly'))
);

CREATE INDEX idx_notification_preferences_tenant ON notification_preferences(tenant_id);
CREATE INDEX idx_notification_preferences_user ON notification_preferences(tenant_id, user_id);
CREATE INDEX idx_notification_preferences_trigger ON notification_preferences(tenant_id, trigger_type);

CREATE TABLE notification_history (
    id            TEXT PRIMARY KEY,
    tenant_id     UUID NOT NULL REFERENCES tenants(id),
    trigger_type  TEXT NOT NULL,
    channel_id    UUID REFERENCES notification_channels(id),
    user_id       TEXT NOT NULL,
    status        TEXT NOT NULL,
    payload       JSONB,
    error_message TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_nh_status CHECK (status IN ('sent', 'failed'))
);

CREATE INDEX idx_notification_history_tenant ON notification_history(tenant_id);
CREATE INDEX idx_notification_history_created ON notification_history(tenant_id, created_at DESC);
CREATE INDEX idx_notification_history_trigger ON notification_history(tenant_id, trigger_type);
CREATE INDEX idx_notification_history_status ON notification_history(tenant_id, status);

-- ============================================================
-- RLS policies
-- ============================================================

ALTER TABLE notification_channels ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON notification_channels
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
ALTER TABLE notification_channels FORCE ROW LEVEL SECURITY;

ALTER TABLE notification_preferences ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON notification_preferences
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
ALTER TABLE notification_preferences FORCE ROW LEVEL SECURITY;

ALTER TABLE notification_history ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON notification_history
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
ALTER TABLE notification_history FORCE ROW LEVEL SECURITY;

-- ============================================================
-- Grants for patchiq_app role
-- ============================================================

GRANT SELECT, INSERT, UPDATE, DELETE ON notification_channels TO patchiq_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON notification_preferences TO patchiq_app;
GRANT SELECT, INSERT ON notification_history TO patchiq_app;

-- +goose Down

DROP TABLE IF EXISTS notification_history CASCADE;
DROP TABLE IF EXISTS notification_preferences CASCADE;
DROP TABLE IF EXISTS notification_channels CASCADE;
