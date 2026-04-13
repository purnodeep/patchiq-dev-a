-- name: CreateWorkflowExecution :one
INSERT INTO workflow_executions (tenant_id, workflow_id, version_id, status, triggered_by, triggered_by_user_id, context)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetWorkflowExecution :one
SELECT * FROM workflow_executions
WHERE id = $1 AND tenant_id = $2;

-- name: ListWorkflowExecutions :many
SELECT * FROM workflow_executions
WHERE workflow_id = @workflow_id AND tenant_id = @tenant_id
  AND (@status_filter::text = '' OR status = @status_filter)
ORDER BY created_at DESC
LIMIT @page_limit;

-- name: CountWorkflowExecutions :one
SELECT count(*) FROM workflow_executions
WHERE workflow_id = @workflow_id AND tenant_id = @tenant_id
  AND (@status_filter::text = '' OR status = @status_filter);

-- name: UpdateWorkflowExecutionStatus :one
UPDATE workflow_executions
SET status = $3, current_node_id = $4, context = $5, error_message = $6,
    started_at = COALESCE(started_at, $7),
    completed_at = $8
WHERE id = $1 AND tenant_id = $2
RETURNING *;

-- name: GetRunningExecutionsForWorkflow :many
SELECT * FROM workflow_executions
WHERE workflow_id = $1 AND tenant_id = $2 AND status IN ('running', 'paused')
ORDER BY created_at DESC;

-- name: CreateNodeExecution :one
INSERT INTO workflow_node_executions (tenant_id, execution_id, node_id, node_type, status, started_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateNodeExecution :one
UPDATE workflow_node_executions
SET status = $3, output = $4, error_message = $5, completed_at = $6
WHERE id = $1 AND tenant_id = $2
RETURNING *;

-- name: ListNodeExecutions :many
SELECT * FROM workflow_node_executions
WHERE execution_id = $1 AND tenant_id = $2
ORDER BY started_at ASC NULLS LAST
LIMIT 1000;

-- name: GetNodeExecutionByNodeID :one
SELECT * FROM workflow_node_executions
WHERE execution_id = $1 AND node_id = $2 AND tenant_id = $3;

-- name: CreateApprovalRequest :one
INSERT INTO approval_requests (tenant_id, execution_id, node_id, approver_roles, escalation_role, timeout_action, timeout_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetApprovalRequest :one
SELECT * FROM approval_requests
WHERE execution_id = $1 AND node_id = $2 AND tenant_id = $3;

-- name: UpdateApprovalRequest :one
UPDATE approval_requests
SET status = $3, acted_by = $4, acted_at = $5, comment = $6
WHERE id = $1 AND tenant_id = $2
RETURNING *;

-- name: GetPendingApprovalByExecution :one
SELECT * FROM approval_requests
WHERE execution_id = $1 AND tenant_id = $2 AND status = 'pending';
