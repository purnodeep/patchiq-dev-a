-- +goose Up

-- ============================================================
-- Deployment waves: wave config, schedules, maintenance windows
-- Issue #174
-- ============================================================

-- ------------------------------------------------------------
-- 1. Extend deployments table
-- ------------------------------------------------------------

ALTER TABLE deployments ADD COLUMN wave_config JSONB;
ALTER TABLE deployments ADD COLUMN max_concurrent INTEGER;
ALTER TABLE deployments ADD COLUMN scheduled_at TIMESTAMPTZ;

-- Expand status CHECK to include rollback states and scheduled.
ALTER TABLE deployments DROP CONSTRAINT chk_deployments_status;
ALTER TABLE deployments ADD CONSTRAINT chk_deployments_status
    CHECK (status IN ('created', 'scheduled', 'running', 'completed', 'failed', 'cancelled', 'rolling_back', 'rolled_back', 'rollback_failed'));

-- Fix temporal constraint: 'scheduled' does not need started_at (like 'created' and 'cancelled').
ALTER TABLE deployments DROP CONSTRAINT chk_deployments_started_if_running;
ALTER TABLE deployments ADD CONSTRAINT chk_deployments_started_if_running
    CHECK (status IN ('created', 'scheduled', 'cancelled') OR started_at IS NOT NULL);

-- Fix completed_if_done: rolled_back and rollback_failed are terminal states needing completed_at.
ALTER TABLE deployments DROP CONSTRAINT chk_deployments_completed_if_done;
ALTER TABLE deployments ADD CONSTRAINT chk_deployments_completed_if_done
    CHECK (status NOT IN ('completed', 'failed', 'cancelled', 'rolled_back', 'rollback_failed') OR completed_at IS NOT NULL);

-- ------------------------------------------------------------
-- 2. Extend deployment_waves table
-- ------------------------------------------------------------

ALTER TABLE deployment_waves ADD COLUMN percentage INTEGER NOT NULL DEFAULT 100;
ALTER TABLE deployment_waves ADD COLUMN target_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE deployment_waves ADD COLUMN success_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE deployment_waves ADD COLUMN failed_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE deployment_waves ADD COLUMN success_threshold NUMERIC(3,2) NOT NULL DEFAULT 0.80;
ALTER TABLE deployment_waves ADD COLUMN error_rate_max NUMERIC(3,2) NOT NULL DEFAULT 0.20;
ALTER TABLE deployment_waves ADD COLUMN delay_after_minutes INTEGER NOT NULL DEFAULT 0;
ALTER TABLE deployment_waves ADD COLUMN eligible_at TIMESTAMPTZ;

-- Expand wave status CHECK to include 'cancelled'.
ALTER TABLE deployment_waves DROP CONSTRAINT chk_deployment_waves_status;
ALTER TABLE deployment_waves ADD CONSTRAINT chk_deployment_waves_status
    CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled'));

-- ------------------------------------------------------------
-- 3. Extend deployment_targets table
-- ------------------------------------------------------------

ALTER TABLE deployment_targets ADD COLUMN wave_id UUID REFERENCES deployment_waves(id);

CREATE INDEX idx_deployment_targets_wave ON deployment_targets(wave_id) WHERE wave_id IS NOT NULL;

-- Expand target status CHECK to include 'sent' and 'executing'.
ALTER TABLE deployment_targets DROP CONSTRAINT chk_deployment_targets_status;
ALTER TABLE deployment_targets ADD CONSTRAINT chk_deployment_targets_status
    CHECK (status IN ('pending', 'sent', 'executing', 'running', 'succeeded', 'failed', 'cancelled'));

-- ------------------------------------------------------------
-- 4. New deployment_schedules table
-- ------------------------------------------------------------

CREATE TABLE deployment_schedules (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    policy_id       UUID NOT NULL REFERENCES policies(id),
    cron_expression TEXT NOT NULL,
    wave_config     JSONB,
    max_concurrent  INTEGER,
    enabled         BOOLEAN NOT NULL DEFAULT true,
    last_run_at     TIMESTAMPTZ,
    next_run_at     TIMESTAMPTZ NOT NULL,
    created_by      UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_deployment_schedules_tenant ON deployment_schedules(tenant_id);
CREATE INDEX idx_deployment_schedules_next_run ON deployment_schedules(next_run_at) WHERE enabled = true;

ALTER TABLE deployment_schedules ENABLE ROW LEVEL SECURITY;
ALTER TABLE deployment_schedules FORCE ROW LEVEL SECURITY;
CREATE POLICY deployment_schedules_tenant_isolation ON deployment_schedules
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- ------------------------------------------------------------
-- 5. Extend endpoints table
-- ------------------------------------------------------------

ALTER TABLE endpoints ADD COLUMN maintenance_window JSONB;

-- +goose Down

-- 5. Remove maintenance_window from endpoints
ALTER TABLE endpoints DROP COLUMN IF EXISTS maintenance_window;

-- 4. Drop deployment_schedules
DROP TABLE IF EXISTS deployment_schedules CASCADE;

-- 3. Revert deployment_targets changes
DROP INDEX IF EXISTS idx_deployment_targets_wave;
ALTER TABLE deployment_targets DROP COLUMN IF EXISTS wave_id;

ALTER TABLE deployment_targets DROP CONSTRAINT IF EXISTS chk_deployment_targets_status;
ALTER TABLE deployment_targets ADD CONSTRAINT chk_deployment_targets_status
    CHECK (status IN ('pending', 'running', 'succeeded', 'failed', 'cancelled'));

-- 2. Revert deployment_waves changes
ALTER TABLE deployment_waves DROP COLUMN IF EXISTS percentage;
ALTER TABLE deployment_waves DROP COLUMN IF EXISTS target_count;
ALTER TABLE deployment_waves DROP COLUMN IF EXISTS success_count;
ALTER TABLE deployment_waves DROP COLUMN IF EXISTS failed_count;
ALTER TABLE deployment_waves DROP COLUMN IF EXISTS success_threshold;
ALTER TABLE deployment_waves DROP COLUMN IF EXISTS error_rate_max;
ALTER TABLE deployment_waves DROP COLUMN IF EXISTS delay_after_minutes;
ALTER TABLE deployment_waves DROP COLUMN IF EXISTS eligible_at;

ALTER TABLE deployment_waves DROP CONSTRAINT IF EXISTS chk_deployment_waves_status;
ALTER TABLE deployment_waves ADD CONSTRAINT chk_deployment_waves_status
    CHECK (status IN ('pending', 'running', 'completed', 'failed'));

-- 1. Revert deployments changes
ALTER TABLE deployments DROP COLUMN IF EXISTS wave_config;
ALTER TABLE deployments DROP COLUMN IF EXISTS max_concurrent;
ALTER TABLE deployments DROP COLUMN IF EXISTS scheduled_at;

ALTER TABLE deployments DROP CONSTRAINT IF EXISTS chk_deployments_status;
ALTER TABLE deployments ADD CONSTRAINT chk_deployments_status
    CHECK (status IN ('created', 'running', 'completed', 'failed', 'cancelled'));

ALTER TABLE deployments DROP CONSTRAINT IF EXISTS chk_deployments_started_if_running;
ALTER TABLE deployments ADD CONSTRAINT chk_deployments_started_if_running
    CHECK (status IN ('created', 'cancelled') OR started_at IS NOT NULL);

ALTER TABLE deployments DROP CONSTRAINT IF EXISTS chk_deployments_completed_if_done;
ALTER TABLE deployments ADD CONSTRAINT chk_deployments_completed_if_done
    CHECK (status NOT IN ('completed', 'failed', 'cancelled') OR completed_at IS NOT NULL);
