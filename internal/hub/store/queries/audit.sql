-- name: InsertAuditEvent :exec
INSERT INTO audit_events (id, type, tenant_id, actor_id, actor_type, resource, resource_id, action, payload, metadata, timestamp)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (id, timestamp) DO NOTHING;

-- name: ListAuditEventsByTenant :many
SELECT * FROM audit_events
WHERE tenant_id = $1
ORDER BY timestamp DESC
LIMIT $2 OFFSET $3;

-- name: ListAuditEventsByResource :many
SELECT * FROM audit_events
WHERE tenant_id = $1 AND resource = $2 AND resource_id = $3
ORDER BY timestamp DESC
LIMIT $4 OFFSET $5;

-- name: ListAuditEventsByActor :many
SELECT * FROM audit_events
WHERE tenant_id = $1 AND actor_id = $2
ORDER BY timestamp DESC
LIMIT $3 OFFSET $4;

-- name: ListAuditEventsByType :many
SELECT * FROM audit_events
WHERE tenant_id = $1 AND type = $2
ORDER BY timestamp DESC
LIMIT $3 OFFSET $4;

-- name: ListAuditEventsByResourceID :many
SELECT *
FROM audit_events
WHERE tenant_id = @tenant_id
  AND resource = @resource
  AND resource_id = @resource_id
ORDER BY timestamp DESC
LIMIT sqlc.arg('query_limit') OFFSET sqlc.arg('query_offset');

-- name: CountAuditEventsByResourceID :one
SELECT COUNT(*)
FROM audit_events
WHERE tenant_id = @tenant_id
  AND resource = @resource
  AND resource_id = @resource_id;
