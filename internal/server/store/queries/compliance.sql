-- ============================================================
-- Compliance engine queries (#176)
-- ============================================================

-- ------------------------------------------------------------
-- Tenant Frameworks
-- ------------------------------------------------------------

-- name: CreateTenantFramework :one
INSERT INTO compliance_tenant_frameworks (tenant_id, framework_id, enabled, sla_overrides, scoring_method, at_risk_threshold)
VALUES (@tenant_id, @framework_id, @enabled, @sla_overrides, @scoring_method, @at_risk_threshold)
RETURNING *;

-- name: GetTenantFramework :one
SELECT * FROM compliance_tenant_frameworks
WHERE tenant_id = @tenant_id AND framework_id = @framework_id;

-- name: GetTenantFrameworkByID :one
SELECT * FROM compliance_tenant_frameworks
WHERE id = @id AND tenant_id = @tenant_id;

-- name: ListTenantFrameworks :many
SELECT * FROM compliance_tenant_frameworks
WHERE tenant_id = @tenant_id
ORDER BY framework_id
LIMIT 100;

-- name: ListEnabledTenantFrameworks :many
SELECT * FROM compliance_tenant_frameworks
WHERE tenant_id = @tenant_id AND enabled = true
ORDER BY framework_id
LIMIT 100;

-- name: UpdateTenantFramework :one
UPDATE compliance_tenant_frameworks
SET enabled = @enabled,
    sla_overrides = @sla_overrides,
    scoring_method = @scoring_method,
    at_risk_threshold = @at_risk_threshold,
    updated_at = now()
WHERE id = @id AND tenant_id = @tenant_id
RETURNING *;

-- name: DeleteTenantFramework :exec
DELETE FROM compliance_tenant_frameworks
WHERE id = @id AND tenant_id = @tenant_id;

-- ------------------------------------------------------------
-- Evaluations
-- ------------------------------------------------------------

-- name: InsertEvaluation :one
INSERT INTO compliance_evaluations (
    tenant_id, evaluation_run_id, endpoint_id, cve_id,
    framework_id, control_id, state, sla_deadline_at,
    remediated_at, days_remaining, evaluated_at
)
VALUES (
    @tenant_id, @evaluation_run_id, @endpoint_id, @cve_id,
    @framework_id, @control_id, @state, @sla_deadline_at,
    @remediated_at, @days_remaining, @evaluated_at
)
RETURNING *;

-- name: ListEvaluationsByEndpoint :many
SELECT * FROM compliance_evaluations ce
WHERE ce.tenant_id = @tenant_id
  AND ce.endpoint_id = @endpoint_id
  AND ce.evaluation_run_id = (
      SELECT sub.evaluation_run_id FROM compliance_evaluations sub
      WHERE sub.tenant_id = @tenant_id
        AND sub.framework_id = ce.framework_id
      ORDER BY sub.evaluated_at DESC
      LIMIT 1
  )
ORDER BY ce.framework_id, ce.control_id;

-- name: ListEvaluationsByFramework :many
SELECT * FROM compliance_evaluations ce
WHERE ce.tenant_id = @tenant_id
  AND ce.framework_id = @framework_id
  AND ce.evaluation_run_id = (
      SELECT sub.evaluation_run_id FROM compliance_evaluations sub
      WHERE sub.tenant_id = @tenant_id
        AND sub.framework_id = @framework_id
      ORDER BY sub.evaluated_at DESC
      LIMIT 1
  )
ORDER BY ce.endpoint_id, ce.control_id;

-- name: CountEvaluationsByState :many
SELECT ce.state, count(*)::bigint AS count
FROM compliance_evaluations ce
WHERE ce.tenant_id = @tenant_id
  AND ce.framework_id = @framework_id
  AND ce.evaluation_run_id = (
      SELECT sub.evaluation_run_id FROM compliance_evaluations sub
      WHERE sub.tenant_id = @tenant_id
        AND sub.framework_id = @framework_id
      ORDER BY sub.evaluated_at DESC
      LIMIT 1
  )
GROUP BY ce.state;

-- name: ListSLADeadlinesApproaching :many
SELECT * FROM compliance_evaluations ce
WHERE ce.tenant_id = @tenant_id
  AND ce.state IN ('AT_RISK', 'NON_COMPLIANT')
  AND ce.sla_deadline_at IS NOT NULL
  AND ce.sla_deadline_at <= @deadline_before
  AND ce.evaluation_run_id = (
      SELECT sub.evaluation_run_id FROM compliance_evaluations sub
      WHERE sub.tenant_id = @tenant_id
        AND sub.framework_id = ce.framework_id
      ORDER BY sub.evaluated_at DESC
      LIMIT 1
  )
ORDER BY ce.sla_deadline_at ASC;

-- name: DeleteOldEvaluations :exec
DELETE FROM compliance_evaluations
WHERE tenant_id = @tenant_id
  AND evaluated_at < @before;

-- ------------------------------------------------------------
-- Scores
-- ------------------------------------------------------------

-- name: InsertScore :one
INSERT INTO compliance_scores (
    tenant_id, evaluation_run_id, framework_id, scope_type,
    scope_id, score, total_cves, compliant_cves, at_risk_cves,
    non_compliant_cves, late_remediation_cves, evaluated_at
)
VALUES (
    @tenant_id, @evaluation_run_id, @framework_id, @scope_type,
    @scope_id, @score, @total_cves, @compliant_cves, @at_risk_cves,
    @non_compliant_cves, @late_remediation_cves, @evaluated_at
)
RETURNING *;

-- name: UpdateEndpointScoresForRun :exec
UPDATE compliance_scores
SET score = @score
WHERE tenant_id = @tenant_id
  AND evaluation_run_id = @evaluation_run_id
  AND framework_id = @framework_id
  AND scope_type = 'endpoint';

-- name: UpdateEndpointScoreByID :exec
UPDATE compliance_scores
SET score = @score
WHERE tenant_id = @tenant_id
  AND evaluation_run_id = @evaluation_run_id
  AND framework_id = @framework_id
  AND scope_type = 'endpoint'
  AND scope_id = @scope_id;

-- name: GetLatestScoresByFramework :many
SELECT * FROM compliance_scores cs
WHERE cs.tenant_id = @tenant_id
  AND cs.framework_id = @framework_id
  AND cs.evaluation_run_id = (
      SELECT sub.evaluation_run_id FROM compliance_scores sub
      WHERE sub.tenant_id = @tenant_id AND sub.framework_id = @framework_id
      ORDER BY sub.evaluated_at DESC
      LIMIT 1
  )
ORDER BY cs.scope_type, cs.scope_id;

-- name: GetLatestFrameworkScore :one
SELECT * FROM compliance_scores cs
WHERE cs.tenant_id = @tenant_id
  AND cs.framework_id = @framework_id
  AND cs.scope_type = 'tenant'
  AND cs.evaluation_run_id = (
      SELECT sub.evaluation_run_id FROM compliance_scores sub
      WHERE sub.tenant_id = @tenant_id AND sub.framework_id = @framework_id
      ORDER BY sub.evaluated_at DESC
      LIMIT 1
  );

-- name: ListEndpointScoresByFramework :many
SELECT * FROM compliance_scores cs
WHERE cs.tenant_id = @tenant_id
  AND cs.framework_id = @framework_id
  AND cs.scope_type = 'endpoint'
  AND cs.evaluation_run_id = (
      SELECT sub.evaluation_run_id FROM compliance_scores sub
      WHERE sub.tenant_id = @tenant_id AND sub.framework_id = @framework_id
      ORDER BY sub.evaluated_at DESC
      LIMIT 1
  )
ORDER BY cs.score ASC;

-- name: ListScoreTrend :many
SELECT * FROM (
    SELECT DISTINCT ON (cs.evaluation_run_id)
        cs.id, cs.tenant_id, cs.evaluation_run_id, cs.framework_id,
        cs.scope_type, cs.scope_id, cs.score, cs.total_cves, cs.compliant_cves,
        cs.at_risk_cves, cs.non_compliant_cves, cs.late_remediation_cves, cs.evaluated_at
    FROM compliance_scores cs
    WHERE cs.tenant_id = @tenant_id
      AND cs.framework_id = @framework_id
      AND cs.scope_type = 'tenant'
    ORDER BY cs.evaluation_run_id, cs.evaluated_at DESC
) sub
ORDER BY sub.evaluated_at ASC;

-- name: DeleteOldScores :exec
DELETE FROM compliance_scores
WHERE tenant_id = @tenant_id
  AND evaluated_at < @before;

-- name: GetLastEvaluationTime :one
SELECT MAX(evaluated_at)::timestamptz AS last_evaluated_at
FROM compliance_scores
WHERE tenant_id = @tenant_id;

-- ------------------------------------------------------------
-- Data needed by evaluator (joins across existing tables)
-- ------------------------------------------------------------

-- name: ListAffectedEndpointCVEs :many
SELECT
    ec.id AS endpoint_cve_id,
    ec.endpoint_id,
    ec.cve_id AS cve_ref_id,
    ec.status,
    ec.detected_at,
    ec.resolved_at,
    ec.risk_score,
    c.cve_id AS cve_identifier,
    c.severity,
    c.cvss_v3_score,
    c.published_at,
    e.hostname,
    e.os_family
FROM endpoint_cves ec
JOIN cves c ON ec.cve_id = c.id AND ec.tenant_id = c.tenant_id
JOIN endpoints e ON ec.endpoint_id = e.id AND ec.tenant_id = e.tenant_id
WHERE ec.tenant_id = @tenant_id
  AND ec.status = 'affected'
ORDER BY ec.endpoint_id, c.cvss_v3_score DESC NULLS LAST
LIMIT 50000;

-- name: ListAllTenantIDs :many
SELECT id FROM tenants
LIMIT 10000;

-- ------------------------------------------------------------
-- Control Results
-- ------------------------------------------------------------

-- name: InsertControlResult :one
INSERT INTO compliance_control_results (
    tenant_id, evaluation_run_id, framework_id, control_id,
    category, status, passing_endpoints, total_endpoints,
    remediation_hint, sla_deadline_at, days_overdue, evaluated_at
)
VALUES (
    @tenant_id, @evaluation_run_id, @framework_id, @control_id,
    @category, @status, @passing_endpoints, @total_endpoints,
    @remediation_hint, @sla_deadline_at, @days_overdue, @evaluated_at
)
RETURNING *;

-- name: ListControlResultsByFramework :many
SELECT * FROM compliance_control_results ccr
WHERE ccr.tenant_id = @tenant_id
  AND ccr.framework_id = @framework_id
  AND ccr.evaluation_run_id = (
      SELECT sub.evaluation_run_id FROM compliance_control_results sub
      WHERE sub.tenant_id = @tenant_id AND sub.framework_id = @framework_id
      ORDER BY sub.evaluated_at DESC
      LIMIT 1
  )
ORDER BY ccr.category, ccr.control_id;

-- name: ListOverdueControls :many
WITH latest_runs AS (
    SELECT DISTINCT ON (framework_id) framework_id, evaluation_run_id
    FROM compliance_control_results WHERE tenant_id = @tenant_id
    ORDER BY framework_id, evaluated_at DESC
)
SELECT ccr.* FROM compliance_control_results ccr
JOIN latest_runs lr ON ccr.framework_id = lr.framework_id AND ccr.evaluation_run_id = lr.evaluation_run_id
JOIN compliance_tenant_frameworks ctf ON ctf.tenant_id = ccr.tenant_id AND ctf.framework_id = ccr.framework_id AND ctf.enabled = true
WHERE ccr.tenant_id = @tenant_id
  AND ccr.status IN ('fail', 'partial')
  AND ccr.sla_deadline_at IS NOT NULL
  AND ccr.sla_deadline_at < now()
ORDER BY ccr.days_overdue DESC NULLS LAST
LIMIT sqlc.arg(result_limit)::int OFFSET sqlc.arg(result_offset)::int;

-- name: GetOverallComplianceScore :one
SELECT
    COALESCE(AVG(sub.score), 0.00)::numeric(5,2) AS overall_score,
    COALESCE(SUM(sub.total_cves), 0)::bigint AS total_cves,
    COALESCE(SUM(sub.compliant_cves), 0)::bigint AS compliant_cves,
    COALESCE(SUM(sub.at_risk_cves), 0)::bigint AS at_risk_cves,
    COALESCE(SUM(sub.non_compliant_cves), 0)::bigint AS non_compliant_cves,
    COUNT(*)::bigint AS framework_count
FROM (
    SELECT
        COALESCE(latest.score, 0.00) AS score,
        COALESCE(latest.total_cves, 0) AS total_cves,
        COALESCE(latest.compliant_cves, 0) AS compliant_cves,
        COALESCE(latest.at_risk_cves, 0) AS at_risk_cves,
        COALESCE(latest.non_compliant_cves, 0) AS non_compliant_cves
    FROM compliance_tenant_frameworks ctf
    LEFT JOIN LATERAL (
        SELECT cs.score, cs.total_cves, cs.compliant_cves, cs.at_risk_cves, cs.non_compliant_cves
        FROM compliance_scores cs
        WHERE cs.framework_id = ctf.framework_id
          AND cs.tenant_id = ctf.tenant_id
          AND cs.scope_type = 'tenant'
        ORDER BY cs.evaluated_at DESC
        LIMIT 1
    ) latest ON true
    WHERE ctf.tenant_id = @tenant_id AND ctf.enabled = true
) sub;

-- name: ListNonCompliantEndpointsByFramework :many
SELECT cs.scope_id AS endpoint_id, cs.score, cs.total_cves, cs.compliant_cves,
       cs.at_risk_cves, cs.non_compliant_cves, cs.late_remediation_cves,
       e.hostname, e.os_family
FROM compliance_scores cs
JOIN endpoints e ON cs.scope_id = e.id AND cs.tenant_id = e.tenant_id
WHERE cs.tenant_id = @tenant_id
  AND cs.framework_id = @framework_id
  AND cs.scope_type = 'endpoint'
  AND cs.evaluation_run_id = (
      SELECT sub.evaluation_run_id FROM compliance_scores sub
      WHERE sub.tenant_id = @tenant_id AND sub.framework_id = @framework_id
      ORDER BY sub.evaluated_at DESC
      LIMIT 1
  )
ORDER BY cs.score ASC
LIMIT 100;

-- name: DeleteControlResultsByFramework :exec
DELETE FROM compliance_control_results
WHERE tenant_id = @tenant_id AND framework_id = @framework_id
  AND evaluation_run_id != @current_run_id;

-- name: DeleteOldControlResults :exec
DELETE FROM compliance_control_results
WHERE tenant_id = @tenant_id
  AND evaluated_at < @before;

-- name: GetControlCountsByFramework :many
WITH latest_runs AS (
    SELECT DISTINCT ON (framework_id) framework_id, evaluation_run_id
    FROM compliance_control_results WHERE tenant_id = @tenant_id
    ORDER BY framework_id, evaluated_at DESC
)
SELECT
    ccr.framework_id,
    COUNT(*) FILTER (WHERE ccr.status = 'pass')::integer AS passing_controls,
    COUNT(*) FILTER (WHERE ccr.status = 'fail')::integer AS failing_controls,
    COUNT(*) FILTER (WHERE ccr.status = 'partial')::integer AS partial_controls,
    COUNT(*) FILTER (WHERE ccr.status = 'na')::integer AS na_controls,
    COUNT(*)::integer AS total_controls,
    COUNT(*) FILTER (WHERE ccr.status IN ('fail', 'partial') AND ccr.sla_deadline_at IS NOT NULL AND ccr.sla_deadline_at < now())::integer AS overdue_count,
    COALESCE(MAX(ccr.total_endpoints), 0)::integer AS max_total_endpoints
FROM compliance_control_results ccr
JOIN latest_runs lr ON ccr.framework_id = lr.framework_id AND ccr.evaluation_run_id = lr.evaluation_run_id
WHERE ccr.tenant_id = @tenant_id
GROUP BY ccr.framework_id;

-- name: GetFrameworkScoreSummary :many
SELECT ctf.framework_id, COALESCE(cs.score, 0)::numeric(5,2) AS score
FROM compliance_tenant_frameworks ctf
LEFT JOIN LATERAL (
    SELECT cs2.score
    FROM compliance_scores cs2
    WHERE cs2.tenant_id = ctf.tenant_id
      AND cs2.framework_id = ctf.framework_id
      AND cs2.scope_type = 'tenant'
    ORDER BY cs2.evaluated_at DESC
    LIMIT 1
) cs ON true
WHERE ctf.tenant_id = @tenant_id
  AND ctf.enabled = true;

-- name: GetCompliantEndpointCountsByFramework :many
WITH latest_runs AS (
    SELECT DISTINCT ON (framework_id) framework_id, evaluation_run_id
    FROM compliance_scores WHERE tenant_id = @tenant_id AND scope_type = 'endpoint'
    ORDER BY framework_id, evaluated_at DESC
)
SELECT
    cs.framework_id,
    COUNT(*) FILTER (WHERE cs.score >= 95)::integer AS endpoints_compliant,
    COUNT(*)::integer AS total_endpoints
FROM compliance_scores cs
JOIN latest_runs lr ON cs.framework_id = lr.framework_id AND cs.evaluation_run_id = lr.evaluation_run_id
JOIN compliance_tenant_frameworks ctf ON cs.framework_id = ctf.framework_id AND cs.tenant_id = ctf.tenant_id
WHERE cs.tenant_id = @tenant_id
  AND cs.scope_type = 'endpoint'
  AND ctf.enabled = true
GROUP BY cs.framework_id;

-- ------------------------------------------------------------
-- Data queries for Tier 1 control evaluators
-- ------------------------------------------------------------

-- name: CountActiveEndpoints :one
SELECT COUNT(*) FROM endpoints
WHERE tenant_id = @tenant_id AND status != 'decommissioned';

-- name: CountEndpointsWithRecentInventory :one
SELECT COUNT(DISTINCT e.id) FROM endpoints e
JOIN endpoint_inventories ei ON ei.endpoint_id = e.id AND ei.tenant_id = e.tenant_id
WHERE e.tenant_id = @tenant_id AND e.status != 'decommissioned'
  AND ei.scanned_at > @since;

-- name: CountEndpointsWithRecentHeartbeat :one
SELECT COUNT(*) FROM endpoints
WHERE tenant_id = @tenant_id AND status != 'decommissioned'
  AND last_heartbeat > @since;

-- name: CountEndpointsWithHardwareInfo :one
SELECT COUNT(*) FROM endpoints
WHERE tenant_id = @tenant_id AND status != 'decommissioned'
  AND cpu_model IS NOT NULL AND memory_total_mb > 0;

-- name: CountEndpointsWithKEVVulnerabilities :one
SELECT COUNT(DISTINCT ec.endpoint_id) FROM endpoint_cves ec
JOIN cves c ON c.id = ec.cve_id AND c.tenant_id = ec.tenant_id
WHERE ec.tenant_id = @tenant_id AND ec.status = 'affected'
  AND c.cisa_kev_due_date IS NOT NULL;

-- name: CountEndpointsScannedForCVEs :one
SELECT COUNT(DISTINCT ec.endpoint_id) FROM endpoint_cves ec
WHERE ec.tenant_id = @tenant_id;

-- name: CountEndpointsWithStaleCriticalCVEs :one
SELECT COUNT(DISTINCT ec.endpoint_id) FROM endpoint_cves ec
JOIN cves c ON c.id = ec.cve_id AND c.tenant_id = ec.tenant_id
WHERE ec.tenant_id = @tenant_id AND ec.status = 'affected'
  AND c.severity IN ('critical', 'high')
  AND ec.detected_at < @max_age;

-- name: CountEndpointsWithStaleCriticalOnlyCVEs :one
SELECT COUNT(DISTINCT ec.endpoint_id) FROM endpoint_cves ec
JOIN cves c ON c.id = ec.cve_id AND c.tenant_id = ec.tenant_id
WHERE ec.tenant_id = @tenant_id AND ec.status = 'affected'
  AND c.severity = 'critical'
  AND ec.detected_at < @max_age;

-- name: ListNonDecommissionedEndpointIDs :many
SELECT id FROM endpoints
WHERE tenant_id = @tenant_id AND status != 'decommissioned'
ORDER BY hostname
LIMIT 10000;

-- name: ListEndpointComplianceFlags :many
-- Returns per-endpoint boolean flags for each check type.
-- Used to compute individual endpoint compliance scores.
SELECT
  e.id AS endpoint_id,
  -- asset_inventory: has hardware data
  (e.cpu_model IS NOT NULL AND e.memory_total_mb > 0)::boolean AS has_hardware,
  -- asset_inventory / agent_monitoring: recent heartbeat
  (e.last_heartbeat IS NOT NULL AND e.last_heartbeat > @heartbeat_since)::boolean AS has_recent_heartbeat,
  -- software_inventory: recent inventory scan
  EXISTS(
    SELECT 1 FROM endpoint_inventories ei
    WHERE ei.endpoint_id = e.id AND ei.tenant_id = e.tenant_id
      AND ei.scanned_at > @scan_since
  )::boolean AS has_recent_scan,
  -- vuln_scanning: has CVE scan data
  EXISTS(
    SELECT 1 FROM endpoint_cves ec
    WHERE ec.endpoint_id = e.id AND ec.tenant_id = e.tenant_id
  )::boolean AS has_cve_data,
  -- kev_compliance: no CISA KEV vulnerabilities
  NOT EXISTS(
    SELECT 1 FROM endpoint_cves ec
    JOIN cves c ON c.id = ec.cve_id AND c.tenant_id = ec.tenant_id
    WHERE ec.endpoint_id = e.id AND ec.tenant_id = e.tenant_id
      AND ec.status = 'affected' AND c.cisa_kev_due_date IS NOT NULL
  )::boolean AS kev_clean,
  -- critical_vuln_remediation: no stale critical/high CVEs
  NOT EXISTS(
    SELECT 1 FROM endpoint_cves ec
    JOIN cves c ON c.id = ec.cve_id AND c.tenant_id = ec.tenant_id
    WHERE ec.endpoint_id = e.id AND ec.tenant_id = e.tenant_id
      AND ec.status = 'affected' AND c.severity IN ('critical', 'high')
      AND ec.detected_at < @cve_max_age
  )::boolean AS no_stale_critical
FROM endpoints e
WHERE e.tenant_id = @tenant_id AND e.status != 'decommissioned'
ORDER BY e.hostname
LIMIT 10000;

-- name: GetRecentDeploymentStats :one
SELECT
  COUNT(*)::bigint AS total,
  COUNT(*) FILTER (WHERE status IN ('completed', 'success'))::bigint AS succeeded,
  COUNT(*) FILTER (WHERE status = 'failed')::bigint AS failed
FROM deployments
WHERE tenant_id = @tenant_id AND created_at > @since;
