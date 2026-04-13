-- +goose Up

-- ============================================================
-- Alert rules — defines which events become alerts
-- ============================================================

CREATE TABLE alert_rules (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             UUID NOT NULL REFERENCES tenants(id),
    event_type            TEXT NOT NULL,
    severity              TEXT NOT NULL CHECK (severity IN ('critical', 'warning', 'info')),
    category              TEXT NOT NULL CHECK (category IN ('deployments', 'agents', 'cves', 'compliance', 'system')),
    title_template        TEXT NOT NULL,
    description_template  TEXT NOT NULL,
    enabled               BOOLEAN NOT NULL DEFAULT true,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, event_type)
);

CREATE INDEX idx_alert_rules_tenant_enabled ON alert_rules(tenant_id, enabled);

ALTER TABLE alert_rules ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON alert_rules
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
ALTER TABLE alert_rules FORCE ROW LEVEL SECURITY;
GRANT SELECT, INSERT, UPDATE, DELETE ON alert_rules TO patchiq_app;

-- ============================================================
-- Alerts — partitioned monthly, same strategy as audit_events
-- ============================================================

CREATE TABLE alerts (
    id              TEXT NOT NULL,
    tenant_id       UUID NOT NULL,
    rule_id         UUID,  -- no FK: PG 16 cannot reference non-partitioned table from partitioned; enforced at app level
    event_id        TEXT NOT NULL,
    severity        TEXT NOT NULL CHECK (severity IN ('critical', 'warning', 'info')),
    category        TEXT NOT NULL CHECK (category IN ('deployments', 'agents', 'cves', 'compliance', 'system')),
    title           TEXT NOT NULL,
    description     TEXT NOT NULL,
    resource        TEXT NOT NULL,
    resource_id     TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'unread' CHECK (status IN ('unread', 'read', 'acknowledged', 'dismissed')),
    payload         JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL,
    read_at         TIMESTAMPTZ,
    acknowledged_at TIMESTAMPTZ,
    dismissed_at    TIMESTAMPTZ,
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Monthly partitions for 2026 (hardcoded for M2).
-- Inserts with timestamps outside defined ranges route to the default partition.
-- A future migration should create partitions for subsequent years or adopt pg_partman.
CREATE TABLE alerts_2026_01 PARTITION OF alerts FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');
CREATE TABLE alerts_2026_02 PARTITION OF alerts FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');
CREATE TABLE alerts_2026_03 PARTITION OF alerts FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');
CREATE TABLE alerts_2026_04 PARTITION OF alerts FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');
CREATE TABLE alerts_2026_05 PARTITION OF alerts FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');
CREATE TABLE alerts_2026_06 PARTITION OF alerts FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');
CREATE TABLE alerts_2026_07 PARTITION OF alerts FOR VALUES FROM ('2026-07-01') TO ('2026-08-01');
CREATE TABLE alerts_2026_08 PARTITION OF alerts FOR VALUES FROM ('2026-08-01') TO ('2026-09-01');
CREATE TABLE alerts_2026_09 PARTITION OF alerts FOR VALUES FROM ('2026-09-01') TO ('2026-10-01');
CREATE TABLE alerts_2026_10 PARTITION OF alerts FOR VALUES FROM ('2026-10-01') TO ('2026-11-01');
CREATE TABLE alerts_2026_11 PARTITION OF alerts FOR VALUES FROM ('2026-11-01') TO ('2026-12-01');
CREATE TABLE alerts_2026_12 PARTITION OF alerts FOR VALUES FROM ('2026-12-01') TO ('2027-01-01');

-- Default partition catches out-of-range timestamps (pre-2026 or post-2026).
-- WARNING: Rows landing here indicate missing partitions for new time ranges.
CREATE TABLE alerts_default PARTITION OF alerts DEFAULT;

CREATE INDEX idx_alerts_tenant_status_created ON alerts(tenant_id, status, created_at DESC);
CREATE INDEX idx_alerts_tenant_unread_severity ON alerts(tenant_id, severity) WHERE status = 'unread';
CREATE UNIQUE INDEX idx_alerts_event_id ON alerts(event_id, created_at);

ALTER TABLE alerts ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON alerts
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
ALTER TABLE alerts FORCE ROW LEVEL SECURITY;
GRANT SELECT, INSERT, UPDATE, DELETE ON alerts TO patchiq_app;

-- ============================================================
-- Seed: 15 default alert rules for the default tenant
-- ============================================================

-- Templates use ONLY fields that exist in the actual event payloads.
-- Events with nil payloads use static text (no template variables).
-- See internal/server/events/topics.go for event type definitions.
INSERT INTO alert_rules (tenant_id, event_type, severity, category, title_template, description_template, enabled) VALUES
-- Critical (6): deployment.* emit nil payload; agent/compliance/license not yet emitted
('00000000-0000-0000-0000-000000000001', 'deployment.failed',            'critical', 'deployments', 'Deployment failed',             'A deployment has failed. Check the deployment detail page for error information.', true),
('00000000-0000-0000-0000-000000000001', 'deployment.rollback_triggered', 'critical', 'deployments', 'Rollback triggered',            'A deployment rollback has been triggered due to failure threshold.', true),
('00000000-0000-0000-0000-000000000001', 'agent.disconnected',            'critical', 'agents',      'Agent disconnected',            'An endpoint agent has lost connection to the server.', true),
('00000000-0000-0000-0000-000000000001', 'compliance.threshold_breach',   'critical', 'compliance',  'Compliance threshold breach',   'A compliance framework score has dropped below the configured threshold.', true),
('00000000-0000-0000-0000-000000000001', 'license.expired',               'critical', 'system',      'License expired',               'Your PatchIQ license has expired. Renew immediately to restore full functionality.', true),
('00000000-0000-0000-0000-000000000001', 'catalog.sync_failed',           'critical', 'system',      'Catalog sync failed',           'Hub catalog synchronization failed. Error: {{.error}}', true),
-- Warning (5): cve.discovered has {cve_id,severity,cvss}; notification.failed has {trigger_type,channel_id,status}
('00000000-0000-0000-0000-000000000001', 'command.timed_out',             'warning',  'deployments', 'Command timed out',             'A deployment command timed out waiting for agent response.', true),
('00000000-0000-0000-0000-000000000001', 'license.expiring',              'warning',  'system',      'License expiring soon',         'Your PatchIQ license is expiring soon. Plan for renewal.', true),
('00000000-0000-0000-0000-000000000001', 'cve.discovered',                'warning',  'cves',        'New CVE: {{.cve_id}}',          'CVE {{.cve_id}} discovered (CVSS {{.cvss}}, severity: {{.severity}}).', true),
('00000000-0000-0000-0000-000000000001', 'notification.failed',           'warning',  'system',      'Notification delivery failed',  'Failed to deliver {{.trigger_type}} notification via channel {{.channel_id}}. Status: {{.status}}', true),
('00000000-0000-0000-0000-000000000001', 'deployment.wave_failed',        'warning',  'deployments', 'Deployment wave failed',        'A deployment wave has failed. Some targets reported errors.', true),
-- Info (4): endpoint.enrolled has {hostname}; cve.remediation has {cve_id,patch_id,package_name}
('00000000-0000-0000-0000-000000000001', 'deployment.completed',          'info',     'deployments', 'Deployment completed',          'A deployment has completed successfully.', true),
('00000000-0000-0000-0000-000000000001', 'deployment.started',            'info',     'deployments', 'Deployment started',            'A new deployment has started execution.', true),
('00000000-0000-0000-0000-000000000001', 'endpoint.enrolled',             'info',     'agents',      'New endpoint: {{.hostname}}',   'Endpoint "{{.hostname}}" has enrolled successfully.', true),
('00000000-0000-0000-0000-000000000001', 'cve.remediation_available',     'info',     'cves',        'Remediation available: {{.cve_id}}', 'A patch ({{.package_name}}) is now available for CVE {{.cve_id}}.', true);

-- +goose Down

DROP TABLE IF EXISTS alerts CASCADE;
DROP TABLE IF EXISTS alert_rules CASCADE;
