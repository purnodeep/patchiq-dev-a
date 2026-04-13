-- +goose Up

-- ============================================================
-- Workflow execution tables
-- ============================================================

CREATE TABLE workflow_executions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenants(id),
    workflow_id         UUID NOT NULL REFERENCES workflows(id),
    version_id          UUID NOT NULL REFERENCES workflow_versions(id),
    status              TEXT NOT NULL DEFAULT 'pending',
    triggered_by        TEXT NOT NULL DEFAULT 'manual',
    triggered_by_user_id UUID,
    current_node_id     UUID,
    context             JSONB NOT NULL DEFAULT '{}',
    error_message       TEXT NOT NULL DEFAULT '',
    started_at          TIMESTAMPTZ,
    completed_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_execution_status CHECK (status IN ('pending', 'running', 'completed', 'failed', 'paused', 'cancelled')),
    CONSTRAINT chk_triggered_by CHECK (triggered_by IN ('manual', 'cron', 'cve_severity', 'policy_evaluation'))
);

CREATE INDEX idx_workflow_executions_tenant ON workflow_executions(tenant_id);
CREATE INDEX idx_workflow_executions_workflow ON workflow_executions(workflow_id);
CREATE INDEX idx_workflow_executions_status ON workflow_executions(tenant_id, status);

CREATE TABLE workflow_node_executions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    execution_id    UUID NOT NULL REFERENCES workflow_executions(id) ON DELETE CASCADE,
    node_id         UUID NOT NULL REFERENCES workflow_nodes(id),
    node_type       TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending',
    output          JSONB NOT NULL DEFAULT '{}',
    error_message   TEXT NOT NULL DEFAULT '',
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    CONSTRAINT chk_node_execution_status CHECK (status IN ('pending', 'running', 'completed', 'failed', 'skipped'))
);

CREATE INDEX idx_workflow_node_executions_execution ON workflow_node_executions(execution_id);
CREATE INDEX idx_workflow_node_executions_tenant ON workflow_node_executions(tenant_id);

CREATE TABLE approval_requests (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    execution_id    UUID NOT NULL REFERENCES workflow_executions(id) ON DELETE CASCADE,
    node_id         UUID NOT NULL REFERENCES workflow_nodes(id),
    approver_roles  TEXT[] NOT NULL,
    escalation_role TEXT NOT NULL DEFAULT '',
    timeout_action  TEXT NOT NULL DEFAULT 'reject',
    timeout_at      TIMESTAMPTZ NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending',
    acted_by        UUID,
    acted_at        TIMESTAMPTZ,
    comment         TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_approval_status CHECK (status IN ('pending', 'approved', 'rejected', 'escalated', 'timed_out')),
    CONSTRAINT chk_timeout_action CHECK (timeout_action IN ('reject', 'escalate'))
);

CREATE INDEX idx_approval_requests_execution ON approval_requests(execution_id);
CREATE INDEX idx_approval_requests_tenant ON approval_requests(tenant_id);
CREATE INDEX idx_approval_requests_pending ON approval_requests(tenant_id, status) WHERE status = 'pending';

-- ============================================================
-- RLS policies
-- ============================================================

ALTER TABLE workflow_executions ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON workflow_executions
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
ALTER TABLE workflow_executions FORCE ROW LEVEL SECURITY;

ALTER TABLE workflow_node_executions ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON workflow_node_executions
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
ALTER TABLE workflow_node_executions FORCE ROW LEVEL SECURITY;

ALTER TABLE approval_requests ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON approval_requests
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
ALTER TABLE approval_requests FORCE ROW LEVEL SECURITY;

-- ============================================================
-- Grants
-- ============================================================

GRANT SELECT, INSERT, UPDATE, DELETE ON workflow_executions TO patchiq_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON workflow_node_executions TO patchiq_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON approval_requests TO patchiq_app;

-- +goose Down

DROP TABLE IF EXISTS approval_requests CASCADE;
DROP TABLE IF EXISTS workflow_node_executions CASCADE;
DROP TABLE IF EXISTS workflow_executions CASCADE;
