-- ============================================================
-- Alert Rules
-- ============================================================

-- name: ListAlertRules :many
SELECT * FROM alert_rules
WHERE tenant_id = $1
ORDER BY created_at DESC
LIMIT 100;

-- name: GetAlertRule :one
SELECT * FROM alert_rules
WHERE id = $1 AND tenant_id = $2;

-- name: CreateAlertRule :one
INSERT INTO alert_rules (tenant_id, event_type, severity, category, title_template, description_template, enabled)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateAlertRule :one
UPDATE alert_rules
SET event_type           = $3,
    severity             = $4,
    category             = $5,
    title_template       = $6,
    description_template = $7,
    enabled              = $8,
    updated_at           = now()
WHERE id = $1 AND tenant_id = $2
RETURNING *;

-- name: DeleteAlertRule :execrows
DELETE FROM alert_rules
WHERE id = $1 AND tenant_id = $2;

-- name: ListEnabledAlertRules :many
SELECT * FROM alert_rules
WHERE tenant_id = @tenant_id AND enabled = true
ORDER BY created_at DESC
LIMIT 100;

-- ============================================================
-- Alerts
-- ============================================================

-- name: InsertAlert :exec
INSERT INTO alerts (id, tenant_id, rule_id, event_id, severity, category, title, description, resource, resource_id, status, payload, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
ON CONFLICT (event_id, created_at) DO NOTHING;

-- name: ListAlertsFiltered :many
SELECT * FROM alerts
WHERE tenant_id = @tenant_id
  AND (@severity::text = '' OR severity = @severity)
  AND (@category::text = '' OR category = @category)
  AND (@status::text = '' OR status = @status)
  AND (@from_date::timestamptz IS NULL OR created_at >= @from_date)
  AND (@to_date::timestamptz IS NULL OR created_at <= @to_date)
  AND (@search::text = '' OR title ILIKE '%' || @search || '%'
       OR description ILIKE '%' || @search || '%')
  AND (
    @cursor_timestamp::timestamptz IS NULL
    OR (created_at, id) < (@cursor_timestamp, @cursor_id::text)
  )
ORDER BY created_at DESC, id DESC
LIMIT @page_limit;

-- name: CountAlertsFiltered :one
SELECT count(*) FROM alerts
WHERE tenant_id = @tenant_id
  AND (@severity::text = '' OR severity = @severity)
  AND (@category::text = '' OR category = @category)
  AND (@status::text = '' OR status = @status)
  AND (@from_date::timestamptz IS NULL OR created_at >= @from_date)
  AND (@to_date::timestamptz IS NULL OR created_at <= @to_date)
  AND (@search::text = '' OR title ILIKE '%' || @search || '%'
       OR description ILIKE '%' || @search || '%');

-- name: CountUnreadAlerts :one
SELECT
    count(*) FILTER (WHERE severity = 'critical') AS critical_unread,
    count(*) FILTER (WHERE severity = 'warning')  AS warning_unread,
    count(*) FILTER (WHERE severity = 'info')     AS info_unread,
    count(*)                                       AS total_unread
FROM alerts
WHERE tenant_id = @tenant_id
  AND status = 'unread'
  AND (@from_date::timestamptz IS NULL OR created_at >= @from_date)
  AND (@to_date::timestamptz IS NULL OR created_at <= @to_date);

-- name: GetAlertCreatedAt :one
SELECT created_at FROM alerts
WHERE id = @id AND tenant_id = @tenant_id
LIMIT 1;

-- name: UpdateAlertStatus :one
UPDATE alerts
SET status          = @status,
    read_at         = CASE WHEN @status::text = 'read'         THEN now() ELSE read_at         END,
    acknowledged_at = CASE WHEN @status::text = 'acknowledged' THEN now() ELSE acknowledged_at END,
    dismissed_at    = CASE WHEN @status::text = 'dismissed'    THEN now() ELSE dismissed_at    END
WHERE id = @id AND created_at = @created_at AND tenant_id = @tenant_id
RETURNING *;

-- name: BulkUpdateAlertStatus :execrows
UPDATE alerts
SET status          = @status,
    read_at         = CASE WHEN @status::text = 'read'         THEN now() ELSE read_at         END,
    acknowledged_at = CASE WHEN @status::text = 'acknowledged' THEN now() ELSE acknowledged_at END,
    dismissed_at    = CASE WHEN @status::text = 'dismissed'    THEN now() ELSE dismissed_at    END
WHERE tenant_id = @tenant_id
  AND id = ANY(@ids::text[])
  AND created_at > now() - interval '90 days';
