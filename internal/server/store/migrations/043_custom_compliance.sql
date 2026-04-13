-- +goose Up

CREATE TABLE custom_compliance_frameworks (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL REFERENCES tenants(id),
    name             TEXT NOT NULL,
    version          TEXT NOT NULL DEFAULT '1.0',
    description      TEXT,
    scoring_method   TEXT NOT NULL DEFAULT 'average',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, name),
    CONSTRAINT chk_ccf_scoring CHECK (scoring_method IN ('strictest', 'average', 'worst_case'))
);

CREATE INDEX idx_ccf_tenant ON custom_compliance_frameworks(tenant_id);

ALTER TABLE custom_compliance_frameworks ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON custom_compliance_frameworks
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
ALTER TABLE custom_compliance_frameworks FORCE ROW LEVEL SECURITY;
GRANT SELECT, INSERT, UPDATE, DELETE ON custom_compliance_frameworks TO patchiq_app;

CREATE TABLE custom_compliance_controls (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL REFERENCES tenants(id),
    framework_id     UUID NOT NULL REFERENCES custom_compliance_frameworks(id) ON DELETE CASCADE,
    control_id       TEXT NOT NULL,
    name             TEXT NOT NULL,
    description      TEXT,
    category         TEXT NOT NULL DEFAULT 'General',
    remediation_hint TEXT,
    sla_tiers        JSONB NOT NULL DEFAULT '[]',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, framework_id, control_id)
);

CREATE INDEX idx_ccc_tenant ON custom_compliance_controls(tenant_id);
CREATE INDEX idx_ccc_framework ON custom_compliance_controls(framework_id);

ALTER TABLE custom_compliance_controls ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON custom_compliance_controls
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
ALTER TABLE custom_compliance_controls FORCE ROW LEVEL SECURITY;
GRANT SELECT, INSERT, UPDATE, DELETE ON custom_compliance_controls TO patchiq_app;

-- +goose Down
DROP TABLE IF EXISTS custom_compliance_controls CASCADE;
DROP TABLE IF EXISTS custom_compliance_frameworks CASCADE;
