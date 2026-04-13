-- name: CreateDeployment :one
INSERT INTO deployments (tenant_id, policy_id, status, created_by)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetDeploymentByID :one
SELECT * FROM deployments WHERE id = $1 AND tenant_id = $2;

-- name: ListDeploymentsByTenant :many
SELECT * FROM deployments WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT 1000;

-- name: UpdateDeploymentStatus :one
UPDATE deployments
SET status = $2, updated_at = now()
WHERE id = $1 AND tenant_id = $3 AND status = $4
RETURNING *;

-- name: CreateDeploymentTarget :one
INSERT INTO deployment_targets (tenant_id, deployment_id, endpoint_id, patch_id, status)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListDeploymentTargets :many
SELECT * FROM deployment_targets
WHERE deployment_id = $1 AND tenant_id = $2
ORDER BY status
LIMIT 5000;

-- name: UpdateDeploymentTargetStatus :one
UPDATE deployment_targets
SET status = $2, started_at = COALESCE($3, started_at), completed_at = $4, error_message = $5,
    stdout = $6, stderr = $7, exit_code = $8
WHERE id = $1 AND tenant_id = $9
RETURNING *;

-- name: CreateDeploymentWave :one
INSERT INTO deployment_waves (tenant_id, deployment_id, wave_number, status)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListDeploymentWaves :many
SELECT * FROM deployment_waves
WHERE deployment_id = $1 AND tenant_id = $2
ORDER BY wave_number
LIMIT 100;

-- name: UpdateDeploymentWaveStatus :one
UPDATE deployment_waves
SET status = $2, started_at = $3, completed_at = $4
WHERE id = $1 AND tenant_id = $5
RETURNING *;

-- name: SetDeploymentStarted :one
UPDATE deployments
SET status = 'running', started_at = now(), updated_at = now()
WHERE id = $1 AND tenant_id = $2 AND status = 'created'
RETURNING *;

-- name: SetDeploymentCompleted :one
UPDATE deployments
SET status = 'completed', completed_at = now(), updated_at = now()
WHERE id = $1 AND tenant_id = $2 AND status = 'running'
RETURNING *;

-- name: SetDeploymentFailed :one
UPDATE deployments
SET status = 'failed', completed_at = now(), updated_at = now()
WHERE id = $1 AND tenant_id = $2 AND status = 'running'
RETURNING *;

-- name: SetDeploymentCancelled :one
UPDATE deployments
SET status = 'cancelled', completed_at = now(), updated_at = now()
WHERE id = $1 AND tenant_id = $2 AND status IN ('created', 'running')
RETURNING *;

-- name: SetDeploymentTotalTargets :one
UPDATE deployments
SET total_targets = $2, updated_at = now()
WHERE id = $1 AND tenant_id = $3
RETURNING *;

-- name: IncrementDeploymentCounters :one
UPDATE deployments
SET completed_count = completed_count + 1,
    success_count = success_count + CASE WHEN @is_success::bool THEN 1 ELSE 0 END,
    failed_count = failed_count + CASE WHEN @is_success::bool THEN 0 ELSE 1 END,
    updated_at = now()
WHERE id = @id AND tenant_id = @tenant_id
RETURNING *;

-- name: CancelDeploymentTargets :exec
UPDATE deployment_targets
SET status = 'cancelled', completed_at = now()
WHERE deployment_id = $1 AND tenant_id = $2
  AND status IN ('pending', 'sent');

-- name: ListPendingDeploymentTargets :many
SELECT * FROM deployment_targets
WHERE deployment_id = $1 AND tenant_id = $2 AND status = 'pending'
ORDER BY created_at
LIMIT 5000;

-- name: ListDeploymentsByTenantFiltered :many
SELECT * FROM deployments
WHERE tenant_id = @tenant_id
  AND (@status::text = '' OR status = @status)
  AND (@policy_id::uuid IS NULL OR policy_id = @policy_id)
  AND (@created_after::timestamptz IS NULL OR created_at >= @created_after)
  AND (@created_before::timestamptz IS NULL OR created_at <= @created_before)
  AND (
    @cursor_created_at::timestamptz IS NULL
    OR (created_at, id) > (@cursor_created_at, @cursor_id::uuid)
  )
ORDER BY created_at, id
LIMIT @page_limit;

-- name: CountDeploymentsByTenantFiltered :one
SELECT count(*) FROM deployments
WHERE tenant_id = @tenant_id
  AND (@status::text = '' OR status = @status)
  AND (@policy_id::uuid IS NULL OR policy_id = @policy_id)
  AND (@created_after::timestamptz IS NULL OR created_at >= @created_after)
  AND (@created_before::timestamptz IS NULL OR created_at <= @created_before);

-- name: CreateDeploymentWithWaveConfig :one
INSERT INTO deployments (tenant_id, policy_id, status, created_by, wave_config, max_concurrent, scheduled_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: CreateDeploymentWithOrchestration :one
INSERT INTO deployments (tenant_id, policy_id, status, created_by, wave_config, max_concurrent, scheduled_at, source_type, target_expression, rollback_config, reboot_config, workflow_template_id, name)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
RETURNING *;

-- name: CreateQuickDeployment :one
INSERT INTO deployments (tenant_id, patch_id, name, status, wave_config)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: CreateDeploymentWaveWithConfig :one
INSERT INTO deployment_waves (tenant_id, deployment_id, wave_number, status, percentage, success_threshold, error_rate_max, delay_after_minutes)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: CreateDeploymentTargetWithWave :one
INSERT INTO deployment_targets (tenant_id, deployment_id, endpoint_id, patch_id, status, wave_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: SetDeploymentWaveTargetCount :one
UPDATE deployment_waves
SET target_count = $2
WHERE id = $1 AND tenant_id = $3
RETURNING *;

-- name: ListRunningDeployments :many
SELECT * FROM deployments WHERE status = 'running' AND tenant_id = @tenant_id
LIMIT 1000;

-- name: ListScheduledDeploymentsDue :many
SELECT * FROM deployments WHERE status = 'scheduled' AND scheduled_at <= now()
LIMIT 1000;

-- name: ListTenantIDsWithRunningDeployments :many
SELECT DISTINCT tenant_id FROM deployments WHERE status = 'running'
LIMIT 1000;

-- name: GetCurrentWave :one
SELECT * FROM deployment_waves
WHERE deployment_id = $1 AND tenant_id = $2 AND status IN ('pending', 'running')
ORDER BY wave_number
LIMIT 1;

-- name: ListPendingWaveTargets :many
SELECT * FROM deployment_targets
WHERE wave_id = $1 AND tenant_id = $2 AND status = 'pending'
ORDER BY created_at
LIMIT 5000;

-- name: CountActiveTargets :one
SELECT count(*) FROM deployment_targets
WHERE deployment_id = $1 AND tenant_id = $2 AND status IN ('sent', 'executing');

-- name: SetWaveRunning :one
UPDATE deployment_waves
SET status = 'running', started_at = now()
WHERE id = $1 AND tenant_id = $2 AND status = 'pending'
RETURNING *;

-- name: SetWaveCompleted :one
UPDATE deployment_waves
SET status = 'completed', completed_at = now()
WHERE id = $1 AND tenant_id = $2 AND status = 'running'
RETURNING *;

-- name: SetWaveFailed :one
UPDATE deployment_waves
SET status = 'failed', completed_at = now()
WHERE id = $1 AND tenant_id = $2 AND status = 'running'
RETURNING *;

-- name: CancelRemainingWaves :exec
UPDATE deployment_waves
SET status = 'cancelled', completed_at = now()
WHERE deployment_id = $1 AND tenant_id = $2 AND status IN ('pending', 'running');

-- name: CancelWaveTargets :exec
UPDATE deployment_targets
SET status = 'cancelled', completed_at = now()
WHERE deployment_id = $1 AND tenant_id = $2 AND status IN ('pending', 'sent');

-- name: IncrementWaveCounters :one
UPDATE deployment_waves
SET success_count = success_count + CASE WHEN @is_success::bool THEN 1 ELSE 0 END,
    failed_count = failed_count + CASE WHEN @is_success::bool THEN 0 ELSE 1 END
WHERE id = @wave_id AND tenant_id = @tenant_id
RETURNING *;

-- name: SetWaveEligibleAt :exec
UPDATE deployment_waves
SET eligible_at = $2
WHERE id = $1 AND tenant_id = $3;

-- name: GetEndpointMaintenanceWindow :one
SELECT maintenance_window FROM endpoints WHERE id = $1 AND tenant_id = $2;

-- name: SetDeploymentRollingBack :one
UPDATE deployments
SET status = 'rolling_back', completed_at = NULL, updated_at = now()
WHERE id = $1 AND tenant_id = $2 AND status IN ('running', 'failed')
RETURNING *;

-- name: SetDeploymentRolledBack :one
UPDATE deployments
SET status = 'rolled_back', completed_at = now(), updated_at = now()
WHERE id = $1 AND tenant_id = $2 AND status = 'rolling_back'
RETURNING *;

-- name: SetDeploymentRollbackFailed :one
UPDATE deployments
SET status = 'rollback_failed', completed_at = now(), updated_at = now()
WHERE id = $1 AND tenant_id = $2 AND status = 'rolling_back'
RETURNING *;

-- name: SetDeploymentScheduledToCreated :one
UPDATE deployments
SET status = 'created', updated_at = now()
WHERE id = $1 AND tenant_id = $2 AND status = 'scheduled'
RETURNING *;

-- name: GetDeploymentTargetWaveID :one
SELECT wave_id FROM deployment_targets WHERE id = $1 AND tenant_id = $2;

-- name: ListDeploymentTargetsByEndpoint :many
SELECT * FROM deployment_targets
WHERE endpoint_id = $1 AND tenant_id = $2
ORDER BY created_at DESC
LIMIT 1000;

-- name: ListDeploymentTargetsWithHostname :many
SELECT dt.id, dt.tenant_id, dt.deployment_id, dt.endpoint_id, dt.patch_id,
       dt.status, dt.started_at, dt.completed_at, dt.error_message,
       dt.stdout, dt.stderr, dt.exit_code, dt.wave_id, dt.created_at,
       e.hostname
FROM deployment_targets dt
JOIN endpoints e ON e.id = dt.endpoint_id
WHERE dt.deployment_id = $1 AND dt.tenant_id = $2
ORDER BY dt.status, dt.created_at
LIMIT 5000;

-- name: ListDeploymentTargetsByWave :many
SELECT dt.id, dt.tenant_id, dt.deployment_id, dt.endpoint_id, dt.patch_id,
       dt.status, dt.started_at, dt.completed_at, dt.error_message,
       dt.stdout, dt.stderr, dt.exit_code, dt.wave_id, dt.created_at,
       e.hostname
FROM deployment_targets dt
JOIN endpoints e ON e.id = dt.endpoint_id
WHERE dt.wave_id = $1 AND dt.tenant_id = $2
ORDER BY dt.status, dt.created_at
LIMIT 5000;

-- name: RetryFailedTargets :execrows
UPDATE deployment_targets
SET status = 'pending', started_at = NULL, completed_at = NULL,
    error_message = NULL, stdout = NULL, stderr = NULL, exit_code = NULL
WHERE deployment_id = $1 AND tenant_id = $2 AND status = 'failed';

-- name: SetDeploymentRetrying :one
UPDATE deployments
SET status = 'running', completed_at = NULL, updated_at = now()
WHERE id = $1 AND tenant_id = $2 AND status = 'failed'
RETURNING *;

-- name: CountDeploymentsByStatus :many
SELECT status, count(*)::int AS count
FROM deployments
WHERE tenant_id = $1
GROUP BY status
LIMIT 20;

-- name: ListDeploymentPatchSummary :many
SELECT
  p.id AS patch_id,
  p.name AS patch_title,
  p.version AS patch_version,
  p.severity AS patch_severity,
  count(*)::int AS total_targets,
  count(*) FILTER (WHERE dt.status = 'succeeded')::int AS success_count,
  count(*) FILTER (WHERE dt.status = 'failed')::int AS failed_count
FROM deployment_targets dt
JOIN patches p ON p.id = dt.patch_id
WHERE dt.deployment_id = $1 AND dt.tenant_id = $2
GROUP BY p.id, p.name, p.version, p.severity
ORDER BY p.severity, p.name
LIMIT 1000;

-- name: BulkCreateDeploymentTargets :copyfrom
INSERT INTO deployment_targets (tenant_id, deployment_id, endpoint_id, patch_id, status, wave_id)
VALUES ($1, $2, $3, $4, $5, $6);
