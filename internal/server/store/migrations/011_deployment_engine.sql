-- +goose Up

-- Commands table: queued instructions for agents
CREATE TABLE commands (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id),
    agent_id      UUID NOT NULL REFERENCES endpoints(id),
    deployment_id UUID REFERENCES deployments(id),
    target_id     UUID REFERENCES deployment_targets(id),
    type          TEXT NOT NULL,
    payload       BYTEA,
    priority      INTEGER NOT NULL DEFAULT 0,
    status        TEXT NOT NULL DEFAULT 'pending',
    deadline      TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    delivered_at  TIMESTAMPTZ,
    completed_at  TIMESTAMPTZ,
    error_message TEXT
);

CREATE INDEX idx_commands_agent_status ON commands (agent_id, status);
CREATE INDEX idx_commands_deployment ON commands (deployment_id) WHERE deployment_id IS NOT NULL;
CREATE INDEX idx_commands_deadline ON commands (deadline) WHERE status IN ('pending', 'delivered');

-- RLS for commands
ALTER TABLE commands ENABLE ROW LEVEL SECURITY;
ALTER TABLE commands FORCE ROW LEVEL SECURITY;
CREATE POLICY commands_tenant_isolation ON commands
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- Policy filter columns
ALTER TABLE policies ADD COLUMN severity_filter TEXT[];
ALTER TABLE policies ADD COLUMN classification_filter TEXT[];
ALTER TABLE policies ADD COLUMN product_filter TEXT[];

-- Deployment counter columns for tracking progress
ALTER TABLE deployments ADD COLUMN failure_threshold NUMERIC(3,2) NOT NULL DEFAULT 0.20;
ALTER TABLE deployments ADD COLUMN total_targets INTEGER NOT NULL DEFAULT 0;
ALTER TABLE deployments ADD COLUMN completed_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE deployments ADD COLUMN success_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE deployments ADD COLUMN failed_count INTEGER NOT NULL DEFAULT 0;

-- Check constraints for command status and type
ALTER TABLE commands ADD CONSTRAINT chk_command_status
    CHECK (status IN ('pending', 'delivered', 'succeeded', 'failed', 'cancelled'));

ALTER TABLE commands ADD CONSTRAINT chk_command_type
    CHECK (type IN ('install_patch', 'run_scan', 'update_config', 'reboot', 'run_script'));

-- +goose Down
DROP TABLE IF EXISTS commands;
ALTER TABLE policies DROP COLUMN IF EXISTS severity_filter;
ALTER TABLE policies DROP COLUMN IF EXISTS classification_filter;
ALTER TABLE policies DROP COLUMN IF EXISTS product_filter;
ALTER TABLE deployments DROP COLUMN IF EXISTS failure_threshold;
ALTER TABLE deployments DROP COLUMN IF EXISTS total_targets;
ALTER TABLE deployments DROP COLUMN IF EXISTS completed_count;
ALTER TABLE deployments DROP COLUMN IF EXISTS success_count;
ALTER TABLE deployments DROP COLUMN IF EXISTS failed_count;
