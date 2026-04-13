-- +goose Up

CREATE TABLE compliance_control_results (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES tenants(id),
    evaluation_run_id UUID NOT NULL,
    framework_id      TEXT NOT NULL,
    control_id        TEXT NOT NULL,
    category          TEXT NOT NULL,
    status            TEXT NOT NULL,
    passing_endpoints INTEGER NOT NULL DEFAULT 0,
    total_endpoints   INTEGER NOT NULL DEFAULT 0,
    remediation_hint  TEXT,
    sla_deadline_at   TIMESTAMPTZ,
    days_overdue      INTEGER,
    evaluated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_ccr_status CHECK (status IN ('pass', 'fail', 'partial', 'na'))
);

CREATE INDEX idx_ccr_tenant ON compliance_control_results(tenant_id);
CREATE INDEX idx_ccr_tenant_framework ON compliance_control_results(tenant_id, framework_id);
CREATE INDEX idx_ccr_run ON compliance_control_results(evaluation_run_id);
CREATE INDEX idx_ccr_tenant_framework_category ON compliance_control_results(tenant_id, framework_id, category);

ALTER TABLE compliance_control_results ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON compliance_control_results
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
ALTER TABLE compliance_control_results FORCE ROW LEVEL SECURITY;

GRANT SELECT, INSERT, DELETE ON compliance_control_results TO patchiq_app;

-- +goose Down
DROP TABLE IF EXISTS compliance_control_results CASCADE;
