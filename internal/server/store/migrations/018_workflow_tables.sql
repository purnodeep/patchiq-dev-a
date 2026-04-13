-- +goose Up

-- ============================================================
-- Workflow tables: workflows, workflow_versions, workflow_nodes, workflow_edges
-- ============================================================

CREATE TABLE workflows (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ,
    UNIQUE (tenant_id, name),
    CONSTRAINT chk_workflow_name_not_empty CHECK (name <> '')
);

CREATE INDEX idx_workflows_tenant ON workflows(tenant_id);

CREATE TABLE workflow_versions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    version     INTEGER NOT NULL DEFAULT 1,
    status      TEXT NOT NULL DEFAULT 'draft',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workflow_id, version),
    CONSTRAINT chk_version_status CHECK (status IN ('draft', 'published', 'archived'))
);

CREATE INDEX idx_workflow_versions_workflow ON workflow_versions(workflow_id);
CREATE UNIQUE INDEX idx_one_published_version
    ON workflow_versions (workflow_id)
    WHERE status = 'published';

CREATE TABLE workflow_nodes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    version_id  UUID NOT NULL REFERENCES workflow_versions(id) ON DELETE CASCADE,
    node_type   TEXT NOT NULL,
    label       TEXT NOT NULL DEFAULT '',
    position_x  DOUBLE PRECISION NOT NULL DEFAULT 0,
    position_y  DOUBLE PRECISION NOT NULL DEFAULT 0,
    config      JSONB NOT NULL DEFAULT '{}',
    CONSTRAINT chk_node_type CHECK (node_type IN (
        'trigger', 'filter', 'approval', 'deployment_wave',
        'gate', 'script', 'notification', 'rollback', 'decision', 'complete'
    ))
);

CREATE INDEX idx_workflow_nodes_version ON workflow_nodes(version_id);

CREATE TABLE workflow_edges (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    version_id      UUID NOT NULL REFERENCES workflow_versions(id) ON DELETE CASCADE,
    source_node_id  UUID NOT NULL REFERENCES workflow_nodes(id) ON DELETE CASCADE,
    target_node_id  UUID NOT NULL REFERENCES workflow_nodes(id) ON DELETE CASCADE,
    label           TEXT NOT NULL DEFAULT '',
    UNIQUE (version_id, source_node_id, target_node_id),
    CONSTRAINT chk_no_self_loop CHECK (source_node_id <> target_node_id)
);

CREATE INDEX idx_workflow_edges_version ON workflow_edges(version_id);

-- ============================================================
-- RLS policies
-- ============================================================

ALTER TABLE workflows ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON workflows
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
ALTER TABLE workflows FORCE ROW LEVEL SECURITY;

ALTER TABLE workflow_versions ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON workflow_versions
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
ALTER TABLE workflow_versions FORCE ROW LEVEL SECURITY;

ALTER TABLE workflow_nodes ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON workflow_nodes
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
ALTER TABLE workflow_nodes FORCE ROW LEVEL SECURITY;

ALTER TABLE workflow_edges ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON workflow_edges
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
ALTER TABLE workflow_edges FORCE ROW LEVEL SECURITY;

-- ============================================================
-- Grants for patchiq_app role
-- ============================================================

GRANT SELECT, INSERT, UPDATE, DELETE ON workflows TO patchiq_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON workflow_versions TO patchiq_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON workflow_nodes TO patchiq_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON workflow_edges TO patchiq_app;

-- +goose Down

DROP TABLE IF EXISTS workflow_edges CASCADE;
DROP TABLE IF EXISTS workflow_nodes CASCADE;
DROP TABLE IF EXISTS workflow_versions CASCADE;
DROP TABLE IF EXISTS workflows CASCADE;
