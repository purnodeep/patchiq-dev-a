-- +goose Up

-- ============================================================
-- CHECK constraints for type safety and invariant enforcement
-- ============================================================

-- Status constraints
ALTER TABLE endpoints ADD CONSTRAINT chk_endpoints_status
    CHECK (status IN ('pending', 'online', 'offline', 'stale'));

ALTER TABLE patches ADD CONSTRAINT chk_patches_status
    CHECK (status IN ('available', 'superseded', 'withdrawn'));

ALTER TABLE deployments ADD CONSTRAINT chk_deployments_status
    CHECK (status IN ('created', 'running', 'completed', 'failed'));

ALTER TABLE deployment_targets ADD CONSTRAINT chk_deployment_targets_status
    CHECK (status IN ('pending', 'running', 'succeeded', 'failed'));

ALTER TABLE deployment_waves ADD CONSTRAINT chk_deployment_waves_status
    CHECK (status IN ('pending', 'running', 'completed', 'failed'));

ALTER TABLE agent_registrations ADD CONSTRAINT chk_agent_registrations_status
    CHECK (status IN ('pending', 'registered', 'revoked'));

-- Severity constraints
ALTER TABLE patches ADD CONSTRAINT chk_patches_severity
    CHECK (severity IN ('critical', 'high', 'medium', 'low', 'none'));

ALTER TABLE cves ADD CONSTRAINT chk_cves_severity
    CHECK (severity IN ('critical', 'high', 'medium', 'low', 'none'));

-- config_overrides constraints
ALTER TABLE config_overrides ADD CONSTRAINT chk_config_scope_type
    CHECK (scope_type IN ('tenant', 'group', 'endpoint'));

ALTER TABLE config_overrides ADD CONSTRAINT chk_config_module
    CHECK (module IN ('patcher', 'inventory', 'comms', 'updater'));

-- Make updated_by NOT NULL (require attribution for config changes)
ALTER TABLE config_overrides ALTER COLUMN updated_by SET NOT NULL;

-- audit_events constraints
ALTER TABLE audit_events ADD CONSTRAINT chk_audit_actor_type
    CHECK (actor_type IN ('user', 'system', 'ai_assistant'));

ALTER TABLE audit_events ADD CONSTRAINT chk_audit_id_length
    CHECK (length(id) = 26);

-- agent_registrations state consistency
ALTER TABLE agent_registrations ADD CONSTRAINT chk_registration_has_endpoint
    CHECK (status != 'registered' OR endpoint_id IS NOT NULL);

ALTER TABLE agent_registrations ADD CONSTRAINT chk_registration_has_timestamp
    CHECK (status != 'registered' OR registered_at IS NOT NULL);

-- Deployment temporal ordering
ALTER TABLE deployments ADD CONSTRAINT chk_deployments_started_if_running
    CHECK (status = 'created' OR started_at IS NOT NULL);

ALTER TABLE deployments ADD CONSTRAINT chk_deployments_completed_if_done
    CHECK (status NOT IN ('completed', 'failed') OR completed_at IS NOT NULL);

ALTER TABLE deployments ADD CONSTRAINT chk_deployments_temporal_order
    CHECK (started_at IS NULL OR completed_at IS NULL OR completed_at >= started_at);

-- deployment_waves
ALTER TABLE deployment_waves ADD CONSTRAINT chk_wave_number_positive
    CHECK (wave_number > 0);

-- deployment_targets missing created_at
ALTER TABLE deployment_targets ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT now();

-- +goose Down

-- Drop all CHECK constraints added in this migration
ALTER TABLE endpoints DROP CONSTRAINT IF EXISTS chk_endpoints_status;
ALTER TABLE patches DROP CONSTRAINT IF EXISTS chk_patches_status;
ALTER TABLE deployments DROP CONSTRAINT IF EXISTS chk_deployments_status;
ALTER TABLE deployment_targets DROP CONSTRAINT IF EXISTS chk_deployment_targets_status;
ALTER TABLE deployment_waves DROP CONSTRAINT IF EXISTS chk_deployment_waves_status;
ALTER TABLE agent_registrations DROP CONSTRAINT IF EXISTS chk_agent_registrations_status;

ALTER TABLE patches DROP CONSTRAINT IF EXISTS chk_patches_severity;
ALTER TABLE cves DROP CONSTRAINT IF EXISTS chk_cves_severity;

ALTER TABLE config_overrides DROP CONSTRAINT IF EXISTS chk_config_scope_type;
ALTER TABLE config_overrides DROP CONSTRAINT IF EXISTS chk_config_module;

ALTER TABLE audit_events DROP CONSTRAINT IF EXISTS chk_audit_actor_type;
ALTER TABLE audit_events DROP CONSTRAINT IF EXISTS chk_audit_id_length;

ALTER TABLE agent_registrations DROP CONSTRAINT IF EXISTS chk_registration_has_endpoint;
ALTER TABLE agent_registrations DROP CONSTRAINT IF EXISTS chk_registration_has_timestamp;

ALTER TABLE deployments DROP CONSTRAINT IF EXISTS chk_deployments_started_if_running;
ALTER TABLE deployments DROP CONSTRAINT IF EXISTS chk_deployments_completed_if_done;
ALTER TABLE deployments DROP CONSTRAINT IF EXISTS chk_deployments_temporal_order;

ALTER TABLE deployment_waves DROP CONSTRAINT IF EXISTS chk_wave_number_positive;

ALTER TABLE deployment_targets DROP COLUMN IF EXISTS created_at;
ALTER TABLE config_overrides ALTER COLUMN updated_by DROP NOT NULL;
