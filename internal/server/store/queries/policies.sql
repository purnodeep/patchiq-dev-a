-- name: CreatePolicy :one
INSERT INTO policies (
    tenant_id, name, description, enabled, mode,
    selection_mode, min_severity, cve_ids, package_regex, exclude_packages,
    schedule_type, schedule_cron, mw_start, mw_end, deployment_strategy, policy_type, timezone, mw_enabled
) VALUES (
    @tenant_id, @name, @description, @enabled, @mode,
    @selection_mode, @min_severity, @cve_ids, @package_regex, @exclude_packages,
    @schedule_type, @schedule_cron, @mw_start, @mw_end, @deployment_strategy, @policy_type, @timezone, @mw_enabled
) RETURNING *;

-- name: GetPolicyByID :one
SELECT * FROM policies
WHERE id = @id AND tenant_id = @tenant_id AND deleted_at IS NULL;

-- name: ListPolicies :many
SELECT p.*
FROM policies p
WHERE p.tenant_id = @tenant_id
  AND p.deleted_at IS NULL
  AND (@search::text = '' OR p.name ILIKE '%' || @search || '%')
  AND (@enabled_filter::text = '' OR p.enabled = (@enabled_filter = 'true'))
  AND (@mode_filter::text = '' OR p.mode = @mode_filter)
  AND (
    @cursor_created_at::timestamptz IS NULL
    OR (p.created_at, p.id) > (@cursor_created_at, @cursor_id::uuid)
  )
ORDER BY p.created_at, p.id
LIMIT @page_limit;

-- name: CountPolicies :one
SELECT count(*) FROM policies p
WHERE p.tenant_id = @tenant_id
  AND p.deleted_at IS NULL
  AND (@search::text = '' OR p.name ILIKE '%' || @search || '%')
  AND (@enabled_filter::text = '' OR p.enabled = (@enabled_filter = 'true'))
  AND (@mode_filter::text = '' OR p.mode = @mode_filter);

-- name: UpdatePolicy :one
UPDATE policies SET
    name = @name,
    description = @description,
    enabled = @enabled,
    mode = @mode,
    selection_mode = @selection_mode,
    min_severity = @min_severity,
    cve_ids = @cve_ids,
    package_regex = @package_regex,
    exclude_packages = @exclude_packages,
    schedule_type = @schedule_type,
    schedule_cron = @schedule_cron,
    mw_start = @mw_start,
    mw_end = @mw_end,
    deployment_strategy = @deployment_strategy,
    policy_type = @policy_type,
    timezone = @timezone,
    mw_enabled = @mw_enabled,
    updated_at = now()
WHERE id = @id AND tenant_id = @tenant_id AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeletePolicy :one
UPDATE policies SET deleted_at = now(), updated_at = now()
WHERE id = @id AND tenant_id = @tenant_id AND deleted_at IS NULL
RETURNING *;

-- name: ListAvailablePatchesForEndpoint :many
SELECT p.* FROM patches p
WHERE p.tenant_id = @tenant_id
  AND p.os_family = @os_family
  AND p.status = 'available'
ORDER BY p.name, p.version
LIMIT 1000;

-- name: ListCVEsForPatches :many
SELECT pc.patch_id, c.* FROM cves c
JOIN patch_cves pc ON c.id = pc.cve_id AND c.tenant_id = pc.tenant_id
WHERE pc.tenant_id = @tenant_id
  AND pc.patch_id = ANY(@patch_ids::uuid[])
ORDER BY pc.patch_id, c.severity
LIMIT 5000;

-- name: ListPoliciesWithStats :many
-- target_endpoints_count was previously derived from policy_groups; in the
-- key=value world the authoritative answer requires evaluating the policy's
-- target_selector against endpoint_tags, which is not expressible in a
-- single sqlc query. The handler populates this from targeting.Resolver.
-- We keep the column in the shape (set to 0 here) so the downstream Go
-- type and response DTOs keep their field layout.
SELECT p.*,
  0::int AS target_endpoints_count
FROM policies p
WHERE p.tenant_id = @tenant_id
  AND p.deleted_at IS NULL
  AND (@search::text = '' OR p.name ILIKE '%' || @search || '%')
  AND (@enabled_filter::text = '' OR p.enabled = (@enabled_filter = 'true'))
  AND (@mode_filter::text = '' OR p.mode = @mode_filter)
  AND (@type_filter::text = '' OR p.policy_type = @type_filter)
  AND (
    @cursor_created_at::timestamptz IS NULL
    OR (p.created_at, p.id) > (@cursor_created_at, @cursor_id::uuid)
  )
ORDER BY p.created_at, p.id
LIMIT @page_limit;

-- name: CountPoliciesFiltered :one
SELECT count(*) FROM policies p
WHERE p.tenant_id = @tenant_id
  AND p.deleted_at IS NULL
  AND (@search::text = '' OR p.name ILIKE '%' || @search || '%')
  AND (@enabled_filter::text = '' OR p.enabled = (@enabled_filter = 'true'))
  AND (@mode_filter::text = '' OR p.mode = @mode_filter)
  AND (@type_filter::text = '' OR p.policy_type = @type_filter);

-- name: BulkUpdatePolicyEnabled :exec
UPDATE policies SET enabled = @enabled, updated_at = now()
WHERE tenant_id = @tenant_id AND id = ANY(@ids::uuid[]) AND deleted_at IS NULL;

-- name: BulkSoftDeletePolicies :exec
UPDATE policies SET deleted_at = now(), updated_at = now()
WHERE tenant_id = @tenant_id AND id = ANY(@ids::uuid[]) AND deleted_at IS NULL;

-- name: ListPolicyEvaluations :many
SELECT * FROM policy_evaluations
WHERE tenant_id = @tenant_id AND policy_id = @policy_id
ORDER BY evaluated_at DESC
LIMIT @page_limit;

-- name: CreatePolicyEvaluation :one
INSERT INTO policy_evaluations (
    tenant_id, policy_id, matched_patches, in_scope_endpoints,
    compliant_count, non_compliant_count, duration_ms, pass
) VALUES (
    @tenant_id, @policy_id, @matched_patches, @in_scope_endpoints,
    @compliant_count, @non_compliant_count, @duration_ms, @pass
) RETURNING *;

-- name: UpdatePolicyEvalStats :exec
UPDATE policies SET
    last_evaluated_at = @last_evaluated_at,
    last_eval_pass = @last_eval_pass,
    last_eval_endpoint_count = @last_eval_endpoint_count,
    last_eval_compliant_count = @last_eval_compliant_count,
    updated_at = now()
WHERE id = @id AND tenant_id = @tenant_id;

-- name: CountDeploymentsForPolicy :one
SELECT count(*) FROM deployments
WHERE tenant_id = @tenant_id AND policy_id = @policy_id;

-- name: ListDeploymentsForPolicy :many
SELECT * FROM deployments
WHERE tenant_id = @tenant_id AND policy_id = @policy_id
ORDER BY created_at DESC
LIMIT @page_limit;


