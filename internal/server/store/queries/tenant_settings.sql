-- name: GetTenantSettings :one
SELECT * FROM tenant_settings
WHERE tenant_id = @tenant_id;

-- name: UpsertTenantSettings :one
INSERT INTO tenant_settings (tenant_id, audit_retention_days)
VALUES (@tenant_id, @audit_retention_days)
ON CONFLICT (tenant_id) DO UPDATE SET
  audit_retention_days = @audit_retention_days,
  updated_at = now()
RETURNING *;

-- name: ListTenantRetentionPolicies :many
SELECT tenant_id, audit_retention_days
FROM tenant_settings
LIMIT 100;

-- name: GetGeneralSettings :one
SELECT org_name, timezone, date_format, scan_interval_hours
FROM tenant_settings
WHERE tenant_id = @tenant_id;

-- name: UpdateGeneralSettings :one
INSERT INTO tenant_settings (tenant_id, org_name, timezone, date_format, scan_interval_hours)
VALUES (@tenant_id, @org_name, @timezone, @date_format, @scan_interval_hours)
ON CONFLICT (tenant_id) DO UPDATE SET
  org_name = @org_name,
  timezone = @timezone,
  date_format = @date_format,
  scan_interval_hours = @scan_interval_hours,
  updated_at = now()
RETURNING org_name, timezone, date_format, scan_interval_hours;
