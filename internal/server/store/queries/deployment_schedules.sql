-- name: CreateDeploymentSchedule :one
INSERT INTO deployment_schedules (tenant_id, policy_id, cron_expression, wave_config, max_concurrent, enabled, next_run_at, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetDeploymentScheduleByID :one
SELECT * FROM deployment_schedules WHERE id = $1 AND tenant_id = $2;

-- name: ListDeploymentSchedulesByTenant :many
SELECT * FROM deployment_schedules WHERE tenant_id = $1 ORDER BY created_at DESC
LIMIT 1000;

-- name: UpdateDeploymentSchedule :one
UPDATE deployment_schedules
SET cron_expression = COALESCE(NULLIF($2, ''), cron_expression),
    wave_config = COALESCE($3, wave_config),
    max_concurrent = $4,
    enabled = $5,
    updated_at = now()
WHERE id = $1 AND tenant_id = $6
RETURNING *;

-- name: DeleteDeploymentSchedule :exec
DELETE FROM deployment_schedules WHERE id = $1 AND tenant_id = $2;

-- name: ListDueSchedules :many
SELECT * FROM deployment_schedules
WHERE enabled = true AND next_run_at <= now()
LIMIT 1000;

-- name: UpdateScheduleAfterRun :exec
UPDATE deployment_schedules
SET last_run_at = now(), next_run_at = $2, updated_at = now()
WHERE id = $1 AND tenant_id = $3;

-- name: HasActiveDeploymentForSchedule :one
SELECT EXISTS(
    SELECT 1 FROM deployments
    WHERE policy_id = $1 AND tenant_id = $2 AND status IN ('created', 'running', 'scheduled')
) AS has_active;
