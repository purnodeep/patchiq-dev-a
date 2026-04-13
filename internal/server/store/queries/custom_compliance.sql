-- ============================================================
-- Custom Compliance Framework queries
-- ============================================================

-- ------------------------------------------------------------
-- Custom Frameworks
-- ------------------------------------------------------------

-- name: CreateCustomFramework :one
INSERT INTO custom_compliance_frameworks (tenant_id, name, version, description, scoring_method)
VALUES (@tenant_id, @name, @version, @description, @scoring_method)
RETURNING *;

-- name: GetCustomFramework :one
SELECT * FROM custom_compliance_frameworks
WHERE id = @id AND tenant_id = @tenant_id;

-- name: ListCustomFrameworks :many
SELECT cf.*,
  (SELECT COUNT(*) FROM custom_compliance_controls cc
   WHERE cc.framework_id = cf.id AND cc.tenant_id = cf.tenant_id)::bigint AS control_count
FROM custom_compliance_frameworks cf
WHERE cf.tenant_id = @tenant_id
ORDER BY cf.created_at DESC
LIMIT 100;

-- name: UpdateCustomFramework :one
UPDATE custom_compliance_frameworks
SET name           = @name,
    version        = @version,
    description    = @description,
    scoring_method = @scoring_method,
    updated_at     = now()
WHERE id = @id AND tenant_id = @tenant_id
RETURNING *;

-- name: DeleteCustomFramework :exec
DELETE FROM custom_compliance_frameworks
WHERE id = @id AND tenant_id = @tenant_id;

-- ------------------------------------------------------------
-- Custom Controls
-- ------------------------------------------------------------

-- name: CreateCustomControl :one
INSERT INTO custom_compliance_controls (tenant_id, framework_id, control_id, name, description, category, remediation_hint, sla_tiers, check_type, check_config)
VALUES (@tenant_id, @framework_id, @control_id, @name, @description, @category, @remediation_hint, @sla_tiers, @check_type, @check_config)
RETURNING *;

-- name: ListCustomControls :many
SELECT * FROM custom_compliance_controls
WHERE tenant_id = @tenant_id AND framework_id = @framework_id
ORDER BY category ASC, control_id ASC
LIMIT 1000;

-- name: DeleteCustomControls :exec
DELETE FROM custom_compliance_controls
WHERE tenant_id = @tenant_id AND framework_id = @framework_id;
