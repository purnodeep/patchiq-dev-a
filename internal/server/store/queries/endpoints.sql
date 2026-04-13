-- name: CreateEndpoint :one
INSERT INTO endpoints (tenant_id, hostname, os_family, os_version, agent_version, status, ip_address, arch, kernel_version)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: ListEndpointsByIDs :many
-- Used by the policy/deployment evaluators to hydrate a selector-resolved
-- set of endpoint UUIDs into the minimal shape their downstream logic
-- needs (id, hostname, os_family, status).
SELECT id, hostname, os_family, status
FROM endpoints
WHERE tenant_id = @tenant_id
  AND id = ANY(@ids::UUID[])
  AND status != 'decommissioned'
ORDER BY hostname
LIMIT 5000;

-- name: GetEndpointByID :one
WITH cve_counts AS (
    SELECT ec.endpoint_id,
        COUNT(*) AS cve_count,
        COUNT(*) FILTER (WHERE c.severity = 'critical') AS critical_cve_count,
        COUNT(*) FILTER (WHERE c.severity = 'high') AS high_cve_count,
        COUNT(*) FILTER (WHERE c.severity = 'medium') AS medium_cve_count
    FROM endpoint_cves ec
    JOIN cves c ON c.id = ec.cve_id
    WHERE ec.tenant_id = $2 AND ec.status = 'affected'
    AND ec.endpoint_id = $1
    GROUP BY ec.endpoint_id
),
patch_counts AS (
    SELECT dt.endpoint_id,
        COUNT(*) AS pending_patches_count,
        COUNT(*) FILTER (WHERE p.severity = 'critical') AS critical_patch_count,
        COUNT(*) FILTER (WHERE p.severity = 'high') AS high_patch_count,
        COUNT(*) FILTER (WHERE p.severity = 'medium') AS medium_patch_count
    FROM deployment_targets dt
    JOIN deployments d ON d.id = dt.deployment_id
    LEFT JOIN patches p ON p.id = dt.patch_id
    WHERE d.tenant_id = $2
      AND dt.endpoint_id = $1
      AND dt.status IN ('pending', 'sent', 'executing', 'running')
    GROUP BY dt.endpoint_id
)
SELECT
    e.id, e.tenant_id, e.hostname, e.os_family, e.os_version, e.agent_version,
    e.status, e.last_seen, e.ip_address, e.arch, e.kernel_version,
    e.cpu_model, e.cpu_cores, e.cpu_usage_percent, e.memory_total_mb, e.memory_used_mb,
    e.disk_total_gb, e.disk_used_gb, e.gpu_model, e.uptime_seconds,
    e.enrolled_at, e.last_heartbeat, e.cert_expiry,
    e.maintenance_window,
    e.hardware_details, e.software_summary,
    e.created_at, e.updated_at,
    COALESCE(cc.cve_count, 0)::bigint AS cve_count,
    COALESCE(cc.critical_cve_count, 0)::bigint AS critical_cve_count,
    COALESCE(cc.high_cve_count, 0)::bigint AS high_cve_count,
    COALESCE(cc.medium_cve_count, 0)::bigint AS medium_cve_count,
    COALESCE(pc.pending_patches_count, 0)::bigint AS pending_patches_count,
    COALESCE(pc.critical_patch_count, 0)::bigint AS critical_patch_count,
    COALESCE(pc.high_patch_count, 0)::bigint AS high_patch_count,
    COALESCE(pc.medium_patch_count, 0)::bigint AS medium_patch_count
FROM endpoints e
LEFT JOIN cve_counts cc ON cc.endpoint_id = e.id
LEFT JOIN patch_counts pc ON pc.endpoint_id = e.id
WHERE e.id = $1 AND e.tenant_id = $2 AND e.status != 'decommissioned';

-- name: ListEndpointsByTenant :many
SELECT * FROM endpoints WHERE tenant_id = $1 ORDER BY created_at LIMIT 1000;

-- name: GetEndpointByHostnameAndOS :one
SELECT * FROM endpoints
WHERE tenant_id = $1 AND hostname = $2 AND os_family = $3;

-- name: LookupEndpointByID :one
SELECT * FROM endpoints WHERE id = $1;

-- name: UpdateEndpoint :one
UPDATE endpoints
SET hostname = $2, os_family = $3, os_version = $4, agent_version = $5,
    ip_address = $6, arch = $7, kernel_version = $8, updated_at = now()
WHERE id = $1 AND tenant_id = $9 AND status != 'decommissioned'
RETURNING *;

-- name: UpdateEndpointStatus :one
UPDATE endpoints
SET status = $2, last_seen = now(), updated_at = now()
WHERE id = $1 AND tenant_id = $3
RETURNING *;

-- name: ListEndpoints :many
WITH cve_counts AS (
    SELECT ec.endpoint_id,
        COUNT(*) AS cve_count,
        COUNT(*) FILTER (WHERE c.severity = 'critical') AS critical_cve_count,
        COUNT(*) FILTER (WHERE c.severity = 'high') AS high_cve_count,
        COUNT(*) FILTER (WHERE c.severity = 'medium') AS medium_cve_count
    FROM endpoint_cves ec
    JOIN cves c ON c.id = ec.cve_id
    WHERE ec.tenant_id = @tenant_id AND ec.status = 'affected'
    GROUP BY ec.endpoint_id
),
patch_counts AS (
    SELECT dt.endpoint_id,
        COUNT(*) AS pending_patches_count,
        COUNT(*) FILTER (WHERE p.severity = 'critical') AS critical_patch_count,
        COUNT(*) FILTER (WHERE p.severity = 'high') AS high_patch_count,
        COUNT(*) FILTER (WHERE p.severity = 'medium') AS medium_patch_count
    FROM deployment_targets dt
    JOIN deployments d ON d.id = dt.deployment_id
    LEFT JOIN patches p ON p.id = dt.patch_id
    WHERE d.tenant_id = @tenant_id
      AND dt.status IN ('pending', 'sent', 'executing', 'running')
    GROUP BY dt.endpoint_id
),
compliance_avgs AS (
    SELECT scope_id AS endpoint_id, AVG(score) AS compliance_pct
    FROM (
        SELECT DISTINCT ON (scope_id, framework_id)
            scope_id, framework_id, score
        FROM compliance_scores
        WHERE tenant_id = @tenant_id
          AND scope_type = 'endpoint'
        ORDER BY scope_id, framework_id, evaluated_at DESC
    ) latest
    GROUP BY scope_id
),
tag_info AS (
    SELECT et.endpoint_id,
        COALESCE(json_agg(json_build_object('id', t.id, 'key', t.key, 'value', t.value) ORDER BY t.key, t.value), '[]'::json) AS tags
    FROM endpoint_tags et
    JOIN tags t ON t.id = et.tag_id
    WHERE et.tenant_id = @tenant_id
    GROUP BY et.endpoint_id
)
SELECT
    e.id,
    e.tenant_id,
    e.hostname,
    e.os_family,
    e.os_version,
    e.agent_version,
    e.status,
    e.last_seen,
    e.ip_address,
    e.arch,
    e.kernel_version,
    e.created_at,
    e.updated_at,
    COALESCE(cc.cve_count, 0)::bigint AS cve_count,
    COALESCE(cc.critical_cve_count, 0)::bigint AS critical_cve_count,
    COALESCE(cc.high_cve_count, 0)::bigint AS high_cve_count,
    COALESCE(cc.medium_cve_count, 0)::bigint AS medium_cve_count,
    COALESCE(pc.pending_patches_count, 0)::bigint AS pending_patches_count,
    COALESCE(pc.critical_patch_count, 0)::bigint AS critical_patch_count,
    COALESCE(pc.high_patch_count, 0)::bigint AS high_patch_count,
    COALESCE(pc.medium_patch_count, 0)::bigint AS medium_patch_count,
    ca.compliance_pct,
    COALESCE(ti.tags, '[]'::json) AS tags,
    e.cpu_cores,
    e.cpu_usage_percent,
    e.memory_total_mb,
    e.memory_used_mb,
    e.disk_total_gb,
    e.disk_used_gb
FROM endpoints e
LEFT JOIN cve_counts cc ON cc.endpoint_id = e.id
LEFT JOIN patch_counts pc ON pc.endpoint_id = e.id
LEFT JOIN compliance_avgs ca ON ca.endpoint_id = e.id
LEFT JOIN tag_info ti ON ti.endpoint_id = e.id
WHERE e.tenant_id = @tenant_id
  AND (e.status != 'decommissioned' OR @status::text = 'decommissioned')
  AND (@status::text = '' OR e.status = @status)
  AND (@os_family::text = '' OR e.os_family = @os_family)
  AND (@search::text = '' OR e.hostname ILIKE '%' || @search || '%')
  AND (@tag_id::uuid IS NULL OR e.id IN (
    SELECT et2.endpoint_id FROM endpoint_tags et2
    WHERE et2.tag_id = @tag_id AND et2.tenant_id = @tenant_id
  ))
  AND (
    @cursor_created_at::timestamptz IS NULL
    OR e.created_at < @cursor_created_at
    OR (e.created_at = @cursor_created_at AND e.id < @cursor_id::uuid)
  )
ORDER BY e.created_at DESC, e.id DESC
LIMIT @page_limit;

-- name: CountEndpoints :one
SELECT count(*) FROM endpoints e
WHERE e.tenant_id = @tenant_id
  AND (@status::text = '' OR e.status = @status)
  AND (@os_family::text = '' OR e.os_family = @os_family)
  AND (@search::text = '' OR e.hostname ILIKE '%' || @search || '%')
  AND (@tag_id::uuid IS NULL OR e.id IN (
    SELECT et2.endpoint_id FROM endpoint_tags et2
    WHERE et2.tag_id = @tag_id AND et2.tenant_id = @tenant_id
  ))
  AND (e.status != 'decommissioned' OR @status::text = 'decommissioned');

-- name: ListPatchesForEndpoint :many
SELECT
    p.id,
    p.name,
    p.version,
    p.severity,
    p.os_family,
    p.status,
    p.source_repo,
    p.created_at,
    dt.status AS deploy_status,
    COALESCE(
        (SELECT MAX(c.cvss_v3_score) FROM patch_cves pc JOIN cves c ON c.id = pc.cve_id WHERE pc.patch_id = p.id AND pc.tenant_id = @tenant_id),
        0
    )::real AS highest_cvss,
    COALESCE(
        (SELECT COUNT(*) FROM patch_cves pc WHERE pc.patch_id = p.id AND pc.tenant_id = @tenant_id),
        0
    )::bigint AS cve_count
FROM deployment_targets dt
JOIN patches p ON p.id = dt.patch_id
WHERE dt.endpoint_id = @endpoint_id
  AND dt.tenant_id = @tenant_id
ORDER BY
    CASE p.severity
        WHEN 'critical' THEN 0
        WHEN 'high' THEN 1
        WHEN 'medium' THEN 2
        WHEN 'low' THEN 3
        ELSE 4
    END,
    p.name
LIMIT 1000;

-- name: ListAvailablePatchesForEndpointByPackage :many
-- Returns patches that match the endpoint's installed packages (by name)
-- but have NOT been deployed to this endpoint yet.
SELECT
    p.id,
    p.name,
    p.version,
    p.severity,
    p.os_family,
    p.created_at,
    COALESCE(
        (SELECT MAX(c.cvss_v3_score) FROM patch_cves pc JOIN cves c ON c.id = pc.cve_id WHERE pc.patch_id = p.id AND pc.tenant_id = @tenant_id),
        0
    )::real AS highest_cvss,
    COALESCE(
        (SELECT COUNT(*) FROM patch_cves pc WHERE pc.patch_id = p.id AND pc.tenant_id = @tenant_id),
        0
    )::bigint AS cve_count
FROM patches p
WHERE p.tenant_id = @tenant_id
  AND p.status = 'available'
  AND lower(p.name) IN (
    SELECT lower(ep.package_name) FROM endpoint_packages ep
    JOIN endpoint_inventories ei ON ep.inventory_id = ei.id
    WHERE ep.endpoint_id = @endpoint_id AND ep.tenant_id = @tenant_id
      AND ei.id = (SELECT id FROM endpoint_inventories
                   WHERE endpoint_id = @endpoint_id AND tenant_id = @tenant_id
                   ORDER BY scanned_at DESC LIMIT 1)
  )
  AND NOT EXISTS (
    SELECT 1 FROM deployment_targets dt
    WHERE dt.patch_id = p.id AND dt.endpoint_id = @endpoint_id AND dt.tenant_id = @tenant_id
  )
ORDER BY
    CASE p.severity
        WHEN 'critical' THEN 0
        WHEN 'high' THEN 1
        WHEN 'medium' THEN 2
        WHEN 'low' THEN 3
        ELSE 4
    END,
    p.name
LIMIT 1000;

-- name: ListAvailablePatchesForEndpointByOS :many
SELECT
    p.id,
    p.name,
    p.version,
    p.severity,
    p.os_family,
    p.status,
    p.source_repo,
    p.created_at,
    'available'::text AS deploy_status,
    COALESCE(
        (SELECT MAX(c.cvss_v3_score) FROM patch_cves pc JOIN cves c ON c.id = pc.cve_id WHERE pc.patch_id = p.id AND pc.tenant_id = @tenant_id),
        0
    )::real AS highest_cvss,
    COALESCE(
        (SELECT COUNT(*) FROM patch_cves pc WHERE pc.patch_id = p.id AND pc.tenant_id = @tenant_id),
        0
    )::bigint AS cve_count
FROM patches p
WHERE p.tenant_id = @tenant_id
  AND p.os_family = (SELECT e.os_family FROM endpoints e WHERE e.id = @endpoint_id AND e.tenant_id = @tenant_id)
  AND p.id NOT IN (
      SELECT dt.patch_id FROM deployment_targets dt WHERE dt.endpoint_id = @endpoint_id AND dt.tenant_id = @tenant_id
  )
ORDER BY
    CASE p.severity
        WHEN 'critical' THEN 0
        WHEN 'high' THEN 1
        WHEN 'medium' THEN 2
        WHEN 'low' THEN 3
        ELSE 4
    END,
    p.created_at DESC
LIMIT 1000;

-- name: UpdateEndpointHeartbeat :one
UPDATE endpoints
SET last_heartbeat = now(),
    last_seen = now(),
    status = $2,
    uptime_seconds = $3,
    memory_used_mb = $4,
    disk_used_gb = $5,
    cpu_usage_percent = $6,
    updated_at = now()
WHERE id = $1 AND tenant_id = $7
RETURNING *;

-- name: UpdateEndpointHardware :one
UPDATE endpoints
SET cpu_model = COALESCE(@cpu_model, cpu_model),
    cpu_cores = COALESCE(@cpu_cores, cpu_cores),
    memory_total_mb = COALESCE(@memory_total_mb, memory_total_mb),
    memory_used_mb = COALESCE(@memory_used_mb, memory_used_mb),
    disk_total_gb = COALESCE(@disk_total_gb, disk_total_gb),
    arch = COALESCE(@arch, arch),
    kernel_version = COALESCE(@kernel_version, kernel_version),
    ip_address = COALESCE(@ip_address, ip_address),
    gpu_model = COALESCE(@gpu_model, gpu_model),
    os_version = CASE WHEN @os_version::text = '' THEN os_version ELSE @os_version END,
    agent_version = COALESCE(@agent_version, agent_version),
    enrolled_at = COALESCE(enrolled_at, now()),
    updated_at = now()
WHERE id = @id AND tenant_id = @tenant_id
RETURNING *;

-- name: UpdateEndpointHardwareDetails :exec
UPDATE endpoints
SET hardware_details = @hardware_details,
    software_summary = @software_summary,
    updated_at = NOW()
WHERE id = @id AND tenant_id = @tenant_id;

-- name: SoftDeleteEndpoint :one
UPDATE endpoints
SET status = 'decommissioned', updated_at = now()
WHERE id = @id AND tenant_id = @tenant_id
RETURNING *;

-- name: ListEndpointsForExport :many
-- Returns all matching endpoints (no cursor/pagination) for CSV export.
WITH cve_counts AS (
    SELECT endpoint_id, COUNT(*) AS cve_count
    FROM endpoint_cves
    WHERE tenant_id = @tenant_id AND status = 'affected'
    GROUP BY endpoint_id
),
patch_counts AS (
    SELECT dt.endpoint_id,
        COUNT(*) AS pending_patches_count,
        COUNT(*) FILTER (WHERE p.severity = 'critical') AS critical_patch_count
    FROM deployment_targets dt
    JOIN deployments d ON d.id = dt.deployment_id
    LEFT JOIN patches p ON p.id = dt.patch_id
    WHERE d.tenant_id = @tenant_id
      AND dt.status IN ('pending', 'sent', 'executing', 'running')
    GROUP BY dt.endpoint_id
),
tag_info AS (
    SELECT et.endpoint_id,
        COALESCE(json_agg(json_build_object('id', t.id, 'key', t.key, 'value', t.value) ORDER BY t.key, t.value), '[]'::json) AS tags
    FROM endpoint_tags et
    JOIN tags t ON t.id = et.tag_id
    WHERE et.tenant_id = @tenant_id
    GROUP BY et.endpoint_id
)
SELECT
    e.hostname,
    e.os_family,
    e.os_version,
    e.status,
    e.agent_version,
    e.ip_address,
    e.arch,
    e.kernel_version,
    e.last_seen,
    COALESCE(pc.pending_patches_count, 0)::bigint AS pending_patches_count,
    COALESCE(pc.critical_patch_count, 0)::bigint AS critical_patch_count,
    COALESCE(cc.cve_count, 0)::bigint AS cve_count,
    COALESCE(ti.tags, '[]'::json) AS tags
FROM endpoints e
LEFT JOIN cve_counts cc ON cc.endpoint_id = e.id
LEFT JOIN patch_counts pc ON pc.endpoint_id = e.id
LEFT JOIN tag_info ti ON ti.endpoint_id = e.id
WHERE e.tenant_id = @tenant_id
  AND (e.status != 'decommissioned' OR @status::text = 'decommissioned')
  AND (@status::text = '' OR e.status = @status)
  AND (@os_family::text = '' OR e.os_family = @os_family)
  AND (@search::text = '' OR e.hostname ILIKE '%' || @search || '%')
  AND (@tag_id::uuid IS NULL OR e.id IN (
    SELECT et2.endpoint_id FROM endpoint_tags et2
    WHERE et2.tag_id = @tag_id AND et2.tenant_id = @tenant_id
  ))
ORDER BY e.hostname
LIMIT 10000;

-- name: GetEndpointOsSummary :many
SELECT COALESCE(os_family, 'unknown') AS os_family, COUNT(*)::int AS count
FROM endpoints
WHERE tenant_id = @tenant_id
GROUP BY os_family
LIMIT 20;

-- name: GetEndpointStatusSummary :many
SELECT status, COUNT(*)::int AS count
FROM endpoints
WHERE tenant_id = @tenant_id
GROUP BY status
LIMIT 20;

-- name: UpdateEndpointConfigPushedAt :exec
UPDATE endpoints
SET config_pushed_at = now(), updated_at = now()
WHERE id = $1 AND tenant_id = $2;
