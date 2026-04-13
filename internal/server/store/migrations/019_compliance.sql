-- +goose Up

-- ============================================================
-- Compliance engine tables: frameworks, evaluations, scores
-- Issue #176
-- ============================================================

-- ------------------------------------------------------------
-- 1. compliance_tenant_frameworks
-- ------------------------------------------------------------

CREATE TABLE compliance_tenant_frameworks (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL REFERENCES tenants(id),
    framework_id     TEXT NOT NULL,
    enabled          BOOLEAN NOT NULL DEFAULT true,
    sla_overrides    JSONB,
    scoring_method   TEXT NOT NULL DEFAULT 'average',
    at_risk_threshold NUMERIC(3,2) NOT NULL DEFAULT 0.75,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, framework_id),
    CONSTRAINT chk_ctf_scoring_method CHECK (scoring_method IN ('strictest', 'average', 'worst_case')),
    CONSTRAINT chk_ctf_at_risk_threshold CHECK (at_risk_threshold >= 0.0 AND at_risk_threshold <= 1.0)
);

CREATE INDEX idx_compliance_tenant_frameworks_tenant ON compliance_tenant_frameworks(tenant_id);
CREATE INDEX idx_compliance_tenant_frameworks_tenant_framework ON compliance_tenant_frameworks(tenant_id, framework_id);

-- ------------------------------------------------------------
-- 2. compliance_evaluations
-- ------------------------------------------------------------

CREATE TABLE compliance_evaluations (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES tenants(id),
    evaluation_run_id UUID NOT NULL,
    endpoint_id       UUID NOT NULL,
    cve_id            TEXT NOT NULL,
    framework_id      TEXT NOT NULL,
    control_id        TEXT NOT NULL,
    state             TEXT NOT NULL,
    sla_deadline_at   TIMESTAMPTZ,
    remediated_at     TIMESTAMPTZ,
    days_remaining    INTEGER,
    evaluated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_ce_state CHECK (state IN ('COMPLIANT', 'AT_RISK', 'NON_COMPLIANT', 'LATE_REMEDIATION'))
);

CREATE INDEX idx_compliance_evaluations_tenant ON compliance_evaluations(tenant_id);
CREATE INDEX idx_compliance_evaluations_tenant_framework ON compliance_evaluations(tenant_id, framework_id);
CREATE INDEX idx_compliance_evaluations_tenant_state ON compliance_evaluations(tenant_id, state);
CREATE INDEX idx_compliance_evaluations_tenant_evaluated ON compliance_evaluations(tenant_id, evaluated_at DESC);
CREATE INDEX idx_compliance_evaluations_run ON compliance_evaluations(evaluation_run_id);

-- ------------------------------------------------------------
-- 3. compliance_scores
-- ------------------------------------------------------------

CREATE TABLE compliance_scores (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             UUID NOT NULL REFERENCES tenants(id),
    evaluation_run_id     UUID NOT NULL,
    framework_id          TEXT NOT NULL,
    scope_type            TEXT NOT NULL,
    scope_id              UUID NOT NULL,
    score                 NUMERIC(5,2) NOT NULL,
    total_cves            INTEGER NOT NULL DEFAULT 0,
    compliant_cves        INTEGER NOT NULL DEFAULT 0,
    at_risk_cves          INTEGER NOT NULL DEFAULT 0,
    non_compliant_cves    INTEGER NOT NULL DEFAULT 0,
    late_remediation_cves INTEGER NOT NULL DEFAULT 0,
    evaluated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_cs_scope_type CHECK (scope_type IN ('endpoint', 'group', 'tenant')),
    CONSTRAINT chk_cs_score CHECK (score >= 0.00 AND score <= 100.00)
);

CREATE INDEX idx_compliance_scores_tenant ON compliance_scores(tenant_id);
CREATE INDEX idx_compliance_scores_tenant_framework ON compliance_scores(tenant_id, framework_id);
CREATE INDEX idx_compliance_scores_tenant_evaluated ON compliance_scores(tenant_id, evaluated_at DESC);
CREATE INDEX idx_compliance_scores_run ON compliance_scores(evaluation_run_id);
CREATE INDEX idx_compliance_scores_scope ON compliance_scores(tenant_id, scope_type, scope_id);

-- ============================================================
-- RLS policies
-- ============================================================

ALTER TABLE compliance_tenant_frameworks ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON compliance_tenant_frameworks
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
ALTER TABLE compliance_tenant_frameworks FORCE ROW LEVEL SECURITY;

ALTER TABLE compliance_evaluations ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON compliance_evaluations
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
ALTER TABLE compliance_evaluations FORCE ROW LEVEL SECURITY;

ALTER TABLE compliance_scores ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON compliance_scores
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
ALTER TABLE compliance_scores FORCE ROW LEVEL SECURITY;

-- ============================================================
-- Grants for patchiq_app role
-- ============================================================

GRANT SELECT, INSERT, UPDATE, DELETE ON compliance_tenant_frameworks TO patchiq_app;
GRANT SELECT, INSERT, DELETE ON compliance_evaluations TO patchiq_app;
GRANT SELECT, INSERT, DELETE ON compliance_scores TO patchiq_app;

-- +goose Down

DROP TABLE IF EXISTS compliance_scores CASCADE;
DROP TABLE IF EXISTS compliance_evaluations CASCADE;
DROP TABLE IF EXISTS compliance_tenant_frameworks CASCADE;
