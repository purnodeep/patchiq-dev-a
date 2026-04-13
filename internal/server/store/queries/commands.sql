-- name: CreateCommand :one
INSERT INTO commands (tenant_id, agent_id, deployment_id, target_id, type, payload, priority, status, deadline)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetCommandByID :one
SELECT * FROM commands WHERE id = $1 AND tenant_id = $2;

-- name: CountPendingCommandsByAgent :one
SELECT count(*) FROM commands
WHERE agent_id = $1 AND tenant_id = $2 AND status = 'pending';

-- name: ListPendingCommandsByAgent :many
SELECT * FROM commands
WHERE agent_id = $1 AND tenant_id = $2 AND status = 'pending'
ORDER BY priority DESC, created_at ASC
LIMIT 1000;

-- name: MarkCommandDelivered :one
UPDATE commands
SET status = 'delivered', delivered_at = now()
WHERE id = $1 AND tenant_id = $2 AND status = 'pending'
RETURNING *;

-- name: UpdateCommandStatus :one
UPDATE commands
SET status = $2, completed_at = $3, error_message = $4
WHERE id = $1 AND tenant_id = $5
RETURNING *;

-- name: ListTimedOutCommands :many
-- Cross-tenant system sweep: must run via Pool() (bypasses RLS).
SELECT * FROM commands
WHERE status IN ('pending', 'delivered')
  AND deadline IS NOT NULL
  AND deadline < now()
LIMIT 100;

-- name: CancelCommandsByDeployment :exec
UPDATE commands
SET status = 'cancelled'
WHERE deployment_id = $1 AND tenant_id = $2
  AND status IN ('pending', 'delivered');

-- name: GetActiveRunScanByAgent :one
SELECT * FROM commands
WHERE agent_id = $1 AND tenant_id = $2
  AND type = 'run_scan'
  AND status IN ('pending', 'delivered')
ORDER BY created_at DESC
LIMIT 1;

-- name: ListActiveEndpointsByTenant :many
SELECT * FROM endpoints
WHERE tenant_id = $1 AND status = 'online'
ORDER BY hostname
LIMIT 1000;

-- name: ListPatchesForPolicyFilters :many
-- Used by deployment evaluator: find patches matching severity + OS families.
SELECT * FROM patches
WHERE tenant_id = @tenant_id
  AND (coalesce(cardinality(@severity_filter::text[]), 0) = 0
       OR severity = ANY(@severity_filter::text[]))
  AND os_family = ANY(@os_families::text[])
  AND status = 'available'
ORDER BY name, version
LIMIT 1000;
