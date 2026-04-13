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

-- name: ListAuditEventsFiltered :many
SELECT * FROM audit_events
WHERE tenant_id = @tenant_id
  AND (@actor_id::text = '' OR actor_id ILIKE '%' || @actor_id || '%')
  AND (@actor_type::text = '' OR actor_type = @actor_type)
  AND (@resource::text = '' OR resource = @resource)
  AND (@resource_id::text = '' OR resource_id = @resource_id)
  AND (@action::text = '' OR action = @action)
  AND (@event_type::text = '' OR type = @event_type)
  AND (@exclude_type::text = '' OR type != @exclude_type)
  AND (@from_date::timestamptz IS NULL OR timestamp >= @from_date)
  AND (@to_date::timestamptz IS NULL OR timestamp <= @to_date)
  AND (@search::text = '' OR type ILIKE '%' || @search || '%'
       OR resource ILIKE '%' || @search || '%'
       OR action ILIKE '%' || @search || '%'
       OR to_tsvector('english', COALESCE(payload::text, '')) @@ plainto_tsquery('english', @search))
  AND (
    @cursor_timestamp::timestamptz IS NULL
    OR (timestamp, id) < (@cursor_timestamp, @cursor_id::text)
  )
ORDER BY timestamp DESC, id DESC
LIMIT @page_limit;

-- name: CountAuditEventsFiltered :one
SELECT count(*) FROM audit_events
WHERE tenant_id = @tenant_id
  AND (@actor_id::text = '' OR actor_id ILIKE '%' || @actor_id || '%')
  AND (@actor_type::text = '' OR actor_type = @actor_type)
  AND (@resource::text = '' OR resource = @resource)
  AND (@resource_id::text = '' OR resource_id = @resource_id)
  AND (@action::text = '' OR action = @action)
  AND (@event_type::text = '' OR type = @event_type)
  AND (@exclude_type::text = '' OR type != @exclude_type)
  AND (@from_date::timestamptz IS NULL OR timestamp >= @from_date)
  AND (@to_date::timestamptz IS NULL OR timestamp <= @to_date)
  AND (@search::text = '' OR type ILIKE '%' || @search || '%'
       OR resource ILIKE '%' || @search || '%'
       OR action ILIKE '%' || @search || '%'
       OR to_tsvector('english', COALESCE(payload::text, '')) @@ plainto_tsquery('english', @search));

-- name: ListAuditEventsByEndpoint :many
SELECT * FROM audit_events
WHERE tenant_id = @tenant_id
  AND (
    resource_id = @endpoint_id
    OR (payload->>'endpoint_id') = @endpoint_id
    OR (payload->>'endpoint') = @endpoint_id
  )
  AND (@actor_id::text = '' OR actor_id ILIKE '%' || @actor_id || '%')
  AND (@event_type::text = '' OR type = @event_type)
  AND (@from_date::timestamptz IS NULL OR timestamp >= @from_date)
  AND (@to_date::timestamptz IS NULL OR timestamp <= @to_date)
  AND (
    @cursor_timestamp::timestamptz IS NULL
    OR (timestamp, id) < (@cursor_timestamp, @cursor_id::text)
  )
ORDER BY timestamp DESC, id DESC
LIMIT @page_limit;

-- name: CountAuditEventsByEndpoint :one
SELECT count(*) FROM audit_events
WHERE tenant_id = @tenant_id
  AND (
    resource_id = @endpoint_id
    OR (payload->>'endpoint_id') = @endpoint_id
    OR (payload->>'endpoint') = @endpoint_id
  )
  AND (@actor_id::text = '' OR actor_id ILIKE '%' || @actor_id || '%')
  AND (@event_type::text = '' OR type = @event_type)
  AND (@from_date::timestamptz IS NULL OR timestamp >= @from_date)
  AND (@to_date::timestamptz IS NULL OR timestamp <= @to_date);
