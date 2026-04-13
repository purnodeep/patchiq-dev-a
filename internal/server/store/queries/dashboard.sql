-- name: GetDashboardSummary :one
SELECT
  (SELECT count(*) FROM endpoints e WHERE e.tenant_id = @tenant_id AND e.status != 'decommissioned')::int AS endpoints_total,
  (SELECT count(*) FROM endpoints e WHERE e.tenant_id = @tenant_id AND e.status = 'online')::int AS endpoints_online,
  (SELECT count(*) FROM patches p WHERE p.tenant_id = @tenant_id AND p.status = 'available')::int AS patches_available,
  (SELECT count(*) FROM patches p WHERE p.tenant_id = @tenant_id AND p.status = 'available' AND p.severity = 'critical')::int AS patches_critical,
  (SELECT count(*) FROM patches p WHERE p.tenant_id = @tenant_id AND p.status = 'available' AND p.severity = 'high')::int AS patches_high,
  (SELECT count(*) FROM endpoint_cves ec WHERE ec.tenant_id = @tenant_id)::int AS cves_total,
  (SELECT count(*) FROM endpoint_cves ec WHERE ec.tenant_id = @tenant_id AND ec.status != 'patched')::int AS cves_unpatched,
  (SELECT count(*) FROM endpoint_cves ecv JOIN cves cv ON ecv.cve_id = cv.id AND ecv.tenant_id = cv.tenant_id WHERE ecv.tenant_id = @tenant_id AND ecv.status != 'patched' AND cv.severity = 'critical')::int AS cves_critical,
  (SELECT count(*) FROM deployments d WHERE d.tenant_id = @tenant_id AND d.status IN ('created', 'running'))::int AS deployments_running,
  (SELECT count(*) FROM deployments d WHERE d.tenant_id = @tenant_id AND d.status = 'completed' AND d.completed_at >= CURRENT_DATE)::int AS deployments_completed_today,
  (SELECT count(*) FROM endpoints e WHERE e.tenant_id = @tenant_id AND e.status = 'degraded')::int AS endpoints_degraded,
  (SELECT count(*) FROM patches p2 WHERE p2.tenant_id = @tenant_id AND p2.status = 'available' AND p2.severity = 'medium')::int AS patches_medium,
  (SELECT count(*) FROM patches p3 WHERE p3.tenant_id = @tenant_id AND p3.status = 'available' AND p3.severity = 'low')::int AS patches_low,
  (SELECT count(*) FROM deployments d WHERE d.tenant_id = @tenant_id AND d.status = 'failed')::int AS failed_deployments_count,
  (SELECT count(*) FROM deployments d WHERE d.tenant_id = @tenant_id AND d.status IN ('created', 'running') AND d.created_at < now() - interval '24 hours')::int AS overdue_sla_count,
  COALESCE((SELECT count(DISTINCT ec.endpoint_id) FILTER (WHERE ec.status = 'patched') * 100.0 / NULLIF(count(DISTINCT ec.endpoint_id), 0) FROM endpoint_cves ec WHERE ec.tenant_id = @tenant_id), 100.0)::float AS compliance_pct,
  (SELECT count(*) FROM compliance_tenant_frameworks ctf WHERE ctf.tenant_id = @tenant_id AND ctf.enabled = true)::int AS frameworks_enabled;

-- name: GetActiveDeployments :many
SELECT
  d.id,
  p.name AS policy_name,
  d.status,
  CASE WHEN d.total_targets > 0
    THEN (d.completed_count * 100 / d.total_targets)::int
    ELSE 0
  END AS progress_pct
FROM deployments d
LEFT JOIN policies p ON p.id = d.policy_id AND p.tenant_id = d.tenant_id
WHERE d.tenant_id = @tenant_id
  AND d.status IN ('created', 'running')
ORDER BY d.created_at DESC
LIMIT 5;

-- name: GetFailedDeploymentTrend7d :many
SELECT
  d.date::date AS day,
  COALESCE(cnt, 0)::int AS count
FROM generate_series(
  CURRENT_DATE - INTERVAL '6 days',
  CURRENT_DATE,
  '1 day'
) AS d(date)
LEFT JOIN (
  SELECT completed_at::date AS day, count(*)::int AS cnt
  FROM deployments
  WHERE tenant_id = @tenant_id
    AND status = 'failed'
    AND completed_at >= CURRENT_DATE - INTERVAL '6 days'
  GROUP BY completed_at::date
) sub ON sub.day = d.date::date
ORDER BY d.date;

-- name: GetRunningWorkflows :many
SELECT
  we.id,
  w.name,
  COALESCE(we.current_node_id::text, '') AS current_stage
FROM workflow_executions we
JOIN workflows w ON w.id = we.workflow_id AND w.tenant_id = we.tenant_id
WHERE we.tenant_id = @tenant_id
  AND we.status IN ('pending', 'running')
ORDER BY we.created_at DESC
LIMIT 5;

-- name: GetTopEndpointsByRisk :many
SELECT
  e.hostname,
  count(ec.id)::int AS cve_count,
  LEAST(10, (count(ec.id) FILTER (WHERE c.severity = 'critical') * 3
   + count(ec.id) FILTER (WHERE c.severity = 'high') * 2
   + count(ec.id) FILTER (WHERE c.severity = 'medium')) / 10)::int AS risk_score
FROM endpoints e
LEFT JOIN endpoint_cves ec ON ec.endpoint_id = e.id AND ec.tenant_id = e.tenant_id AND ec.status = 'affected'
LEFT JOIN cves c ON c.id = ec.cve_id
WHERE e.tenant_id = @tenant_id AND e.status != 'decommissioned'
GROUP BY e.id, e.hostname
ORDER BY risk_score DESC
LIMIT 30;

-- name: GetCVEByUUID :one
SELECT c.id, c.cve_id, c.severity,
  COALESCE(c.cvss_v3_score, 0)::float AS cvss_score,
  count(ec.id)::int AS affected_count
FROM cves c
LEFT JOIN endpoint_cves ec ON ec.cve_id = c.id AND ec.tenant_id = c.tenant_id AND ec.status = 'affected'
WHERE c.id = @id AND c.tenant_id = @tenant_id
GROUP BY c.id;

-- name: GetDashboardActivity :many
SELECT
  d.id,
  'deployment' AS type,
  p.name AS title,
  d.status,
  d.total_targets,
  d.completed_count,
  d.failed_count,
  d.created_at AS timestamp
FROM deployments d
LEFT JOIN policies p ON p.id = d.policy_id AND p.tenant_id = d.tenant_id
WHERE d.tenant_id = @tenant_id
ORDER BY d.updated_at DESC
LIMIT 20;

-- name: GetHighestUnpatchedCVE :one
SELECT c.id, c.cve_id, c.severity,
  COALESCE(c.cvss_v3_score, 0)::float AS cvss_score,
  count(ec.id)::int AS affected_count
FROM cves c
JOIN endpoint_cves ec ON ec.cve_id = c.id AND ec.tenant_id = c.tenant_id
WHERE c.tenant_id = @tenant_id AND ec.status = 'affected'
GROUP BY c.id
ORDER BY COALESCE(c.cvss_v3_score, 0) DESC
LIMIT 1;

-- name: GetBlastRadiusGroups :many
-- Blast radius broken down by tag value, replacing the legacy groups
-- version. `name` now reports "key=value" so the response shape stays
-- stable for the dashboard consumer.
SELECT
  COALESCE(t.key || '=' || t.value, '')::text AS name,
  e.os_family AS os,
  count(DISTINCT e.id)::int AS host_count
FROM endpoint_cves ec
JOIN endpoints e ON e.id = ec.endpoint_id AND e.tenant_id = ec.tenant_id
LEFT JOIN endpoint_tags et ON et.endpoint_id = e.id AND et.tenant_id = e.tenant_id
LEFT JOIN tags t ON t.id = et.tag_id AND t.tenant_id = e.tenant_id
WHERE ec.cve_id = @cve_id AND ec.tenant_id = @tenant_id AND ec.status = 'affected'
GROUP BY t.key, t.value, e.os_family
ORDER BY host_count DESC
LIMIT 100;

-- name: GetExposureWindows :many
SELECT c.id, c.cve_id, c.severity,
  COALESCE(c.cvss_v3_score, 0)::float AS cvss_score,
  count(DISTINCT ec.endpoint_id)::int AS affected_count,
  min(ec.created_at) AS first_seen,
  max(CASE WHEN ec.status = 'patched' THEN ec.updated_at END) AS patched_at
FROM cves c
JOIN endpoint_cves ec ON ec.cve_id = c.id AND ec.tenant_id = c.tenant_id
WHERE c.tenant_id = @tenant_id
  AND c.severity IN ('critical', 'high')
  AND ec.created_at >= now() - interval '90 days'
GROUP BY c.id
ORDER BY min(ec.created_at) ASC
LIMIT 15;

-- name: GetMTTR :many
SELECT
  date_trunc('week', ec.updated_at)::date AS week,
  c.severity,
  avg(EXTRACT(EPOCH FROM (ec.updated_at - ec.created_at)) / 3600)::float AS avg_hours
FROM endpoint_cves ec
JOIN cves c ON c.id = ec.cve_id AND c.tenant_id = ec.tenant_id
WHERE ec.tenant_id = @tenant_id
  AND ec.status = 'patched'
  AND ec.updated_at >= now() - interval '26 weeks'
GROUP BY date_trunc('week', ec.updated_at)::date, c.severity
ORDER BY week;

-- name: GetAttackPaths :many
WITH risky_endpoints AS (
  SELECT e.id, e.hostname, e.os_family,
    count(ec.id) FILTER (WHERE c.severity = 'critical')::int AS critical_count,
    count(ec.id) FILTER (WHERE c.severity = 'high')::int AS high_count,
    bool_or(e.status = 'online') AS is_online
  FROM endpoints e
  JOIN endpoint_cves ec ON ec.endpoint_id = e.id AND ec.tenant_id = e.tenant_id AND ec.status = 'affected'
  JOIN cves c ON c.id = ec.cve_id
  WHERE e.tenant_id = @tenant_id AND e.status != 'decommissioned'
  GROUP BY e.id
  HAVING count(ec.id) FILTER (WHERE c.severity IN ('critical', 'high')) > 0
  ORDER BY critical_count DESC, high_count DESC
  LIMIT 50
),
shared_cves AS (
  SELECT ec1.endpoint_id AS source_id, ec2.endpoint_id AS target_id, count(*)::int AS shared_count
  FROM endpoint_cves ec1
  JOIN endpoint_cves ec2 ON ec1.cve_id = ec2.cve_id AND ec1.tenant_id = ec2.tenant_id
    AND ec1.endpoint_id < ec2.endpoint_id
  WHERE ec1.tenant_id = @tenant_id
    AND ec1.endpoint_id IN (SELECT id FROM risky_endpoints)
    AND ec2.endpoint_id IN (SELECT id FROM risky_endpoints)
    AND ec1.status = 'affected' AND ec2.status = 'affected'
  GROUP BY ec1.endpoint_id, ec2.endpoint_id
  HAVING count(*) >= 2
)
SELECT 'node' AS row_type,
  re.id::text AS id, re.hostname AS label, re.os_family AS os,
  re.critical_count, re.high_count, re.is_online,
  ''::text AS source_id, ''::text AS target_id, 0 AS shared_count
FROM risky_endpoints re
UNION ALL
SELECT 'edge' AS row_type,
  ''::text AS id, ''::text AS label, ''::text AS os,
  0 AS critical_count, 0 AS high_count, false AS is_online,
  sc.source_id::text, sc.target_id::text, sc.shared_count
FROM shared_cves sc;

-- name: GetPolicyDrift :many
SELECT e.id, e.hostname, e.os_family,
  count(ec.id) FILTER (WHERE ec.status = 'affected')::int AS unpatched_count,
  count(ec.id)::int AS total_cve_count,
  CASE WHEN count(ec.id) > 0
    THEN (count(ec.id) FILTER (WHERE ec.status = 'affected') * 100 / count(ec.id))::int
    ELSE 0
  END AS drift_score,
  max(CASE WHEN ec.status = 'patched' THEN ec.updated_at END) AS last_compliant_at
FROM endpoints e
LEFT JOIN endpoint_cves ec ON ec.endpoint_id = e.id AND ec.tenant_id = e.tenant_id
WHERE e.tenant_id = @tenant_id AND e.status != 'decommissioned'
GROUP BY e.id, e.hostname, e.os_family
HAVING count(ec.id) FILTER (WHERE ec.status = 'affected') > 0
ORDER BY drift_score DESC
LIMIT 50;

-- name: GetSLAForecast :many
SELECT e.id, e.hostname,
  c.severity,
  min(ec.created_at) AS oldest_open_since,
  CASE c.severity
    WHEN 'critical' THEN 24
    WHEN 'high' THEN 72
    WHEN 'medium' THEN 168
    ELSE 720
  END AS sla_window_hours,
  EXTRACT(EPOCH FROM (
    min(ec.created_at) + CASE c.severity
      WHEN 'critical' THEN interval '24 hours'
      WHEN 'high' THEN interval '72 hours'
      WHEN 'medium' THEN interval '7 days'
      ELSE interval '30 days'
    END - now()
  ))::int AS remaining_seconds
FROM endpoints e
JOIN endpoint_cves ec ON ec.endpoint_id = e.id AND ec.tenant_id = e.tenant_id AND ec.status = 'affected'
JOIN cves c ON c.id = ec.cve_id AND c.tenant_id = ec.tenant_id
WHERE e.tenant_id = @tenant_id AND e.status != 'decommissioned'
GROUP BY e.id, e.hostname, c.severity
HAVING min(ec.created_at) + CASE c.severity
    WHEN 'critical' THEN interval '24 hours'
    WHEN 'high' THEN interval '72 hours'
    WHEN 'medium' THEN interval '7 days'
    ELSE interval '30 days'
  END > now() - interval '48 hours'
ORDER BY remaining_seconds ASC
LIMIT 20;

-- name: GetSLADeadlines :many
SELECT e.id AS endpoint_id, e.hostname,
  c.severity,
  c.cve_id AS patch_name,
  EXTRACT(EPOCH FROM (
    ec.created_at + CASE c.severity
      WHEN 'critical' THEN interval '24 hours'
      WHEN 'high' THEN interval '72 hours'
      WHEN 'medium' THEN interval '7 days'
      ELSE interval '30 days'
    END - now()
  ))::int AS remaining_seconds
FROM endpoint_cves ec
JOIN endpoints e ON e.id = ec.endpoint_id AND e.tenant_id = ec.tenant_id
JOIN cves c ON c.id = ec.cve_id AND c.tenant_id = ec.tenant_id
WHERE ec.tenant_id = @tenant_id
  AND ec.status = 'affected'
  AND e.status != 'decommissioned'
ORDER BY remaining_seconds ASC
LIMIT 10;

-- name: GetSLATiers :many
SELECT
  c.severity,
  count(*)::int AS total,
  count(*) FILTER (WHERE ec.created_at + CASE c.severity
    WHEN 'critical' THEN interval '24 hours'
    WHEN 'high' THEN interval '72 hours'
    WHEN 'medium' THEN interval '7 days'
    ELSE interval '30 days'
  END < now())::int AS overdue
FROM endpoint_cves ec
JOIN cves c ON c.id = ec.cve_id AND c.tenant_id = ec.tenant_id
WHERE ec.tenant_id = @tenant_id AND ec.status = 'affected'
GROUP BY c.severity
ORDER BY CASE c.severity WHEN 'critical' THEN 1 WHEN 'high' THEN 2 WHEN 'medium' THEN 3 ELSE 4 END;

-- name: GetRiskProjectionData :one
SELECT
  (SELECT count(*) FILTER (WHERE ec.status = 'affected') * 100.0 / NULLIF(count(*), 0) FROM endpoint_cves ec WHERE ec.tenant_id = @tenant_id)::float AS current_risk_pct,
  (SELECT avg(cnt)::float FROM (
    SELECT date_trunc('day', ec.updated_at)::date, count(*)::float AS cnt
    FROM endpoint_cves ec WHERE ec.tenant_id = @tenant_id AND ec.status = 'patched' AND ec.updated_at >= now() - interval '30 days'
    GROUP BY date_trunc('day', ec.updated_at)::date
  ) daily_patches) AS avg_daily_patches,
  (SELECT avg(cnt)::float FROM (
    SELECT date_trunc('day', ec.created_at)::date, count(*)::float AS cnt
    FROM endpoint_cves ec WHERE ec.tenant_id = @tenant_id AND ec.created_at >= now() - interval '30 days'
    GROUP BY date_trunc('day', ec.created_at)::date
  ) daily_new) AS avg_daily_new_cves,
  (SELECT count(*) FROM endpoint_cves ec WHERE ec.tenant_id = @tenant_id AND ec.status = 'affected')::int AS total_affected,
  (SELECT count(*) FROM endpoint_cves ec WHERE ec.tenant_id = @tenant_id)::int AS total_cves;
