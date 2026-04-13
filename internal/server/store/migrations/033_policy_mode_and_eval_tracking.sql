-- +goose Up

-- Policy mode: automatic (scheduled deployment), manual (on-demand), advisory (report only)
ALTER TABLE policies ADD COLUMN mode TEXT NOT NULL DEFAULT 'manual';
ALTER TABLE policies ADD CONSTRAINT chk_policy_mode
    CHECK (mode IN ('automatic', 'manual', 'advisory'));

-- Denormalized evaluation tracking for list page performance
ALTER TABLE policies ADD COLUMN last_evaluated_at TIMESTAMPTZ;
ALTER TABLE policies ADD COLUMN last_eval_pass BOOLEAN;
ALTER TABLE policies ADD COLUMN last_eval_endpoint_count INTEGER;
ALTER TABLE policies ADD COLUMN last_eval_compliant_count INTEGER;

-- Policy evaluation history
CREATE TABLE policy_evaluations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    policy_id UUID NOT NULL REFERENCES policies(id),
    evaluated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    matched_patches INTEGER NOT NULL DEFAULT 0,
    in_scope_endpoints INTEGER NOT NULL DEFAULT 0,
    compliant_count INTEGER NOT NULL DEFAULT 0,
    non_compliant_count INTEGER NOT NULL DEFAULT 0,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    pass BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_policy_evaluations_policy
    ON policy_evaluations(tenant_id, policy_id, evaluated_at DESC);

-- RLS for tenant isolation
ALTER TABLE policy_evaluations ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON policy_evaluations
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);

-- +goose Down
DROP POLICY IF EXISTS tenant_isolation ON policy_evaluations;
ALTER TABLE policy_evaluations DISABLE ROW LEVEL SECURITY;
DROP TABLE IF EXISTS policy_evaluations;
ALTER TABLE policies DROP COLUMN IF EXISTS last_eval_compliant_count;
ALTER TABLE policies DROP COLUMN IF EXISTS last_eval_endpoint_count;
ALTER TABLE policies DROP COLUMN IF EXISTS last_eval_pass;
ALTER TABLE policies DROP COLUMN IF EXISTS last_evaluated_at;
ALTER TABLE policies DROP CONSTRAINT IF EXISTS chk_policy_mode;
ALTER TABLE policies DROP COLUMN IF EXISTS mode;
