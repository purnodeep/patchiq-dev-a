-- +goose Up

-- Extended deployment fields for orchestration and UX improvements.
ALTER TABLE deployments
    ADD COLUMN source_type           TEXT NOT NULL DEFAULT 'adhoc',
    ADD COLUMN target_expression     JSONB,
    ADD COLUMN rollback_config       JSONB,
    ADD COLUMN reboot_config         JSONB,
    ADD COLUMN workflow_template_id  UUID REFERENCES workflows(id);

ALTER TABLE deployments ADD CONSTRAINT chk_deployment_source_type
    CHECK (source_type IN ('catalog', 'policy', 'adhoc'));

CREATE INDEX idx_deployments_source_type ON deployments(tenant_id, source_type);
CREATE INDEX idx_deployments_workflow_template ON deployments(tenant_id, workflow_template_id) WHERE workflow_template_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_deployments_workflow_template;
DROP INDEX IF EXISTS idx_deployments_source_type;
ALTER TABLE deployments DROP CONSTRAINT IF EXISTS chk_deployment_source_type;
ALTER TABLE deployments
    DROP COLUMN IF EXISTS workflow_template_id,
    DROP COLUMN IF EXISTS reboot_config,
    DROP COLUMN IF EXISTS rollback_config,
    DROP COLUMN IF EXISTS target_expression,
    DROP COLUMN IF EXISTS source_type;
