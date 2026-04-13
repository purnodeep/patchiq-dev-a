-- +goose Up

-- Full-text search index on audit_events payload JSONB.
-- Applied to parent table; PostgreSQL propagates to all partitions.
CREATE INDEX idx_audit_payload_fts
  ON audit_events
  USING GIN (to_tsvector('english', COALESCE(payload::text, '')));

-- Tenant-level settings (initially for audit retention, extensible for M2+).
CREATE TABLE tenant_settings (
    tenant_id UUID NOT NULL REFERENCES tenants(id) PRIMARY KEY,
    audit_retention_days INTEGER NOT NULL DEFAULT 365,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE tenant_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_settings FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON tenant_settings
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);

GRANT SELECT, INSERT, UPDATE ON tenant_settings TO patchiq_app;

-- +goose Down
DROP TABLE IF EXISTS tenant_settings;
DROP INDEX IF EXISTS idx_audit_payload_fts;
