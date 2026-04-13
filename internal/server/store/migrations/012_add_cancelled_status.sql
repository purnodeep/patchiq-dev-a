-- +goose Up

-- Add 'cancelled' to deployments status CHECK constraint (#154).
-- The Go state machine (statemachine.go) supports created|running → cancelled,
-- and CancelDeploymentTargets sets targets to 'cancelled', but the DB
-- CHECK constraints from 003 only allowed (created, running, completed, failed).

ALTER TABLE deployments DROP CONSTRAINT chk_deployments_status;
ALTER TABLE deployments ADD CONSTRAINT chk_deployments_status
    CHECK (status IN ('created', 'running', 'completed', 'failed', 'cancelled'));

ALTER TABLE deployment_targets DROP CONSTRAINT chk_deployment_targets_status;
ALTER TABLE deployment_targets ADD CONSTRAINT chk_deployment_targets_status
    CHECK (status IN ('pending', 'running', 'succeeded', 'failed', 'cancelled'));

-- Fix temporal constraint: a deployment cancelled from 'created' state has no
-- started_at, so 'cancelled' must be exempt alongside 'created'.
ALTER TABLE deployments DROP CONSTRAINT chk_deployments_started_if_running;
ALTER TABLE deployments ADD CONSTRAINT chk_deployments_started_if_running
    CHECK (status IN ('created', 'cancelled') OR started_at IS NOT NULL);

-- Fix completed_if_done: cancelled deployments also set completed_at.
ALTER TABLE deployments DROP CONSTRAINT chk_deployments_completed_if_done;
ALTER TABLE deployments ADD CONSTRAINT chk_deployments_completed_if_done
    CHECK (status NOT IN ('completed', 'failed', 'cancelled') OR completed_at IS NOT NULL);

-- +goose Down

-- Restore original constraints from 003.
ALTER TABLE deployments DROP CONSTRAINT IF EXISTS chk_deployments_status;
ALTER TABLE deployments ADD CONSTRAINT chk_deployments_status
    CHECK (status IN ('created', 'running', 'completed', 'failed'));

ALTER TABLE deployment_targets DROP CONSTRAINT IF EXISTS chk_deployment_targets_status;
ALTER TABLE deployment_targets ADD CONSTRAINT chk_deployment_targets_status
    CHECK (status IN ('pending', 'running', 'succeeded', 'failed'));

ALTER TABLE deployments DROP CONSTRAINT IF EXISTS chk_deployments_started_if_running;
ALTER TABLE deployments ADD CONSTRAINT chk_deployments_started_if_running
    CHECK (status = 'created' OR started_at IS NOT NULL);

ALTER TABLE deployments DROP CONSTRAINT IF EXISTS chk_deployments_completed_if_done;
ALTER TABLE deployments ADD CONSTRAINT chk_deployments_completed_if_done
    CHECK (status NOT IN ('completed', 'failed') OR completed_at IS NOT NULL);
