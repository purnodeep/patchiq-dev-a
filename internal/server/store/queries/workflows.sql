-- name: CreateWorkflow :one
INSERT INTO workflows (tenant_id, name, description)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetWorkflowByID :one
SELECT * FROM workflows
WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL;

-- name: ListWorkflows :many
SELECT w.*,
       COALESCE(v.version, 0)::int AS current_version,
       COALESCE(v.status, 'draft') AS current_status,
       (SELECT count(*) FROM workflow_nodes wn WHERE wn.version_id = v.id AND wn.tenant_id = @tenant_id)::int AS node_count,
       (SELECT count(*) FROM workflow_executions we WHERE we.workflow_id = w.id AND we.tenant_id = @tenant_id)::int AS total_runs,
       COALESCE((SELECT we2.status FROM workflow_executions we2 WHERE we2.workflow_id = w.id AND we2.tenant_id = @tenant_id ORDER BY we2.created_at DESC LIMIT 1), '')::text AS last_run_status,
       (SELECT we3.created_at FROM workflow_executions we3 WHERE we3.workflow_id = w.id AND we3.tenant_id = @tenant_id ORDER BY we3.created_at DESC LIMIT 1) AS last_run_at
FROM workflows w
LEFT JOIN workflow_versions v ON v.workflow_id = w.id
    AND v.version = (
        SELECT max(v2.version) FROM workflow_versions v2
        WHERE v2.workflow_id = w.id AND v2.tenant_id = @tenant_id
    )
WHERE w.tenant_id = @tenant_id
  AND w.deleted_at IS NULL
  AND (@search::text = '' OR w.name ILIKE '%' || @search || '%' ESCAPE '\')
  AND (@status_filter::text = '' OR COALESCE(v.status, 'draft') = @status_filter)
  AND (
    @cursor_created_at::timestamptz IS NULL
    OR (w.created_at, w.id) > (@cursor_created_at, @cursor_id::uuid)
  )
ORDER BY w.created_at, w.id
LIMIT @page_limit;

-- name: CountWorkflows :one
SELECT count(*) FROM workflows w
LEFT JOIN workflow_versions v ON v.workflow_id = w.id
    AND v.version = (
        SELECT max(v2.version) FROM workflow_versions v2
        WHERE v2.workflow_id = w.id AND v2.tenant_id = @tenant_id
    )
WHERE w.tenant_id = @tenant_id
  AND w.deleted_at IS NULL
  AND (@search::text = '' OR w.name ILIKE '%' || @search || '%' ESCAPE '\')
  AND (@status_filter::text = '' OR COALESCE(v.status, 'draft') = @status_filter);

-- name: CountWorkflowsByStatus :many
SELECT COALESCE(v.status, 'draft')::text AS status, count(*)::int AS count
FROM workflows w
LEFT JOIN workflow_versions v ON v.workflow_id = w.id
    AND v.version = (
        SELECT max(v2.version) FROM workflow_versions v2
        WHERE v2.workflow_id = w.id AND v2.tenant_id = @tenant_id
    )
WHERE w.tenant_id = @tenant_id
  AND w.deleted_at IS NULL
GROUP BY COALESCE(v.status, 'draft');

-- name: UpdateWorkflow :one
UPDATE workflows
SET name = $2, description = $3, updated_at = now()
WHERE id = $1 AND tenant_id = $4 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteWorkflow :one
UPDATE workflows
SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: CreateWorkflowVersion :one
INSERT INTO workflow_versions (tenant_id, workflow_id, version, status)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetLatestVersion :one
SELECT * FROM workflow_versions
WHERE workflow_id = $1 AND tenant_id = $2
ORDER BY version DESC
LIMIT 1;

-- name: GetVersionByID :one
SELECT * FROM workflow_versions
WHERE id = $1 AND tenant_id = $2;

-- name: ListWorkflowVersions :many
SELECT * FROM workflow_versions
WHERE workflow_id = $1 AND tenant_id = $2
ORDER BY version DESC
LIMIT 100;

-- name: GetMaxVersionNumber :one
SELECT COALESCE(max(version), 0)::int FROM workflow_versions
WHERE workflow_id = $1 AND tenant_id = $2;

-- name: ArchiveWorkflowVersion :exec
UPDATE workflow_versions SET status = 'archived'
WHERE id = $1 AND tenant_id = $2;

-- name: PublishWorkflowVersion :one
UPDATE workflow_versions SET status = 'published'
WHERE id = $1 AND tenant_id = $2 AND status = 'draft'
RETURNING *;

-- name: GetPublishedVersion :one
SELECT * FROM workflow_versions
WHERE workflow_id = $1 AND tenant_id = $2 AND status = 'published';

-- name: GetDraftVersion :one
SELECT * FROM workflow_versions
WHERE workflow_id = $1 AND tenant_id = $2 AND status = 'draft'
ORDER BY version DESC
LIMIT 1;

-- name: CreateWorkflowNode :one
INSERT INTO workflow_nodes (tenant_id, version_id, node_type, label, position_x, position_y, config)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListWorkflowNodes :many
SELECT * FROM workflow_nodes
WHERE version_id = $1 AND tenant_id = $2
ORDER BY node_type, label
LIMIT 1000;

-- name: CreateWorkflowEdge :one
INSERT INTO workflow_edges (tenant_id, version_id, source_node_id, target_node_id, label)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListWorkflowEdges :many
SELECT * FROM workflow_edges
WHERE version_id = $1 AND tenant_id = $2
LIMIT 5000;
