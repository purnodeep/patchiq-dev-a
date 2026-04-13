-- name: CreateNotificationChannel :one
INSERT INTO notification_channels (tenant_id, name, channel_type, config_encrypted, enabled)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetNotificationChannel :one
SELECT * FROM notification_channels
WHERE id = $1 AND tenant_id = $2;

-- name: ListNotificationChannels :many
SELECT * FROM notification_channels
WHERE tenant_id = $1
ORDER BY created_at DESC
LIMIT 100;

-- name: UpdateNotificationChannel :one
UPDATE notification_channels
SET name = $3,
    channel_type = $4,
    config_encrypted = $5,
    enabled = $6,
    updated_at = now()
WHERE id = $1 AND tenant_id = $2
RETURNING *;

-- name: DeleteNotificationChannel :execrows
DELETE FROM notification_channels
WHERE id = $1 AND tenant_id = $2;

-- name: UpsertNotificationPreference :one
INSERT INTO notification_preferences (tenant_id, user_id, trigger_type, email_enabled, slack_enabled, webhook_enabled, urgency)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (tenant_id, user_id, trigger_type)
DO UPDATE SET
    email_enabled   = EXCLUDED.email_enabled,
    slack_enabled   = EXCLUDED.slack_enabled,
    webhook_enabled = EXCLUDED.webhook_enabled,
    urgency         = EXCLUDED.urgency
RETURNING *;

-- name: ListNotificationPreferences :many
SELECT * FROM notification_preferences
WHERE tenant_id = $1
ORDER BY trigger_type
LIMIT 100;

-- name: ListEnabledPreferencesForTrigger :many
SELECT np.*, nc.config_encrypted, nc.channel_type, nc.name AS channel_name, nc.id AS channel_id
FROM notification_preferences np
JOIN notification_channels nc ON nc.tenant_id = np.tenant_id AND nc.enabled = true
WHERE np.tenant_id = $1
  AND np.trigger_type = $2
  AND (
      (np.email_enabled   AND nc.channel_type = 'email')   OR
      (np.slack_enabled   AND nc.channel_type = 'slack')   OR
      (np.webhook_enabled AND nc.channel_type = 'webhook')
  );

-- name: InsertNotificationHistory :exec
INSERT INTO notification_history (id, tenant_id, trigger_type, channel_id, channel_type, recipient, subject, user_id, status, payload, error_message)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: GetNotificationHistoryByID :one
SELECT * FROM notification_history
WHERE id = $1 AND tenant_id = $2;

-- name: UpdateNotificationHistoryStatus :exec
UPDATE notification_history
SET status = $3
WHERE id = $1 AND tenant_id = $2;

-- name: RetryNotificationHistory :exec
UPDATE notification_history
SET status = $3, retry_count = retry_count + 1
WHERE id = $1 AND tenant_id = $2;

-- name: ListNotificationHistory :many
SELECT * FROM notification_history
WHERE tenant_id = $1
  AND (sqlc.narg('trigger_type')::TEXT IS NULL OR trigger_type = sqlc.narg('trigger_type'))
  AND (sqlc.narg('status')::TEXT IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('channel_type')::TEXT IS NULL OR channel_type = sqlc.narg('channel_type'))
  AND (sqlc.narg('from_date')::TIMESTAMPTZ IS NULL OR created_at >= sqlc.narg('from_date'))
  AND (sqlc.narg('to_date')::TIMESTAMPTZ IS NULL OR created_at <= sqlc.narg('to_date'))
  AND (sqlc.narg('cursor_id')::TEXT IS NULL OR id < sqlc.narg('cursor_id'))
ORDER BY id DESC
LIMIT $2;

-- name: GetNotificationChannelByType :one
SELECT * FROM notification_channels
WHERE tenant_id = $1 AND channel_type = $2
LIMIT 1;

-- name: UpdateNotificationChannelTestResult :exec
UPDATE notification_channels
SET last_tested_at = $3, last_test_status = $4, updated_at = now()
WHERE id = $1 AND tenant_id = $2;

-- name: CountNotificationHistory :one
SELECT count(*) FROM notification_history
WHERE tenant_id = $1
  AND (sqlc.narg('trigger_type')::TEXT IS NULL OR trigger_type = sqlc.narg('trigger_type'))
  AND (sqlc.narg('status')::TEXT IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('channel_type')::TEXT IS NULL OR channel_type = sqlc.narg('channel_type'))
  AND (sqlc.narg('from_date')::TIMESTAMPTZ IS NULL OR created_at >= sqlc.narg('from_date'))
  AND (sqlc.narg('to_date')::TIMESTAMPTZ IS NULL OR created_at <= sqlc.narg('to_date'));

-- name: GetDigestConfig :one
SELECT * FROM notification_digest_config
WHERE tenant_id = $1;

-- name: UpsertDigestConfig :one
INSERT INTO notification_digest_config (tenant_id, frequency, delivery_time, format)
VALUES ($1, $2, $3, $4)
ON CONFLICT (tenant_id)
DO UPDATE SET
    frequency     = EXCLUDED.frequency,
    delivery_time = EXCLUDED.delivery_time,
    format        = EXCLUDED.format
RETURNING *;
