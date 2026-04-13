-- name: CreatePatch :one
INSERT INTO patches (tenant_id, name, version, severity, os_family, status,
    os_distribution, package_url, checksum_sha256, source_repo, description)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: GetPatchByID :one
SELECT * FROM patches WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL;

-- name: ListPatchesByTenant :many
SELECT * FROM patches WHERE tenant_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT 1000;

-- name: UpdatePatch :one
UPDATE patches
SET name = $2, version = $3, severity = $4, status = $5,
    os_distribution = $6, package_url = $7, checksum_sha256 = $8,
    source_repo = $9, description = $10, updated_at = now()
WHERE id = $1 AND tenant_id = $11
RETURNING *;

-- name: CreateCVE :one
INSERT INTO cves (tenant_id, cve_id, severity, description, published_at,
    cvss_v3_score, cvss_v3_vector, cisa_kev_due_date, exploit_available,
    attack_vector, external_references, cwe_id, source)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
RETURNING *;

-- name: GetCVEByID :one
SELECT * FROM cves WHERE id = $1 AND tenant_id = $2;

-- name: GetCVEByCVEID :one
SELECT * FROM cves WHERE cve_id = $1 AND tenant_id = $2;

-- name: ListCVEsByTenant :many
SELECT * FROM cves WHERE tenant_id = $1 ORDER BY published_at DESC NULLS LAST LIMIT 1000;

-- name: LinkPatchCVE :exec
INSERT INTO patch_cves (tenant_id, patch_id, cve_id, version_end_excluding, version_end_including)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (tenant_id, patch_id, cve_id)
DO UPDATE SET
    version_end_excluding = EXCLUDED.version_end_excluding,
    version_end_including = EXCLUDED.version_end_including;

-- name: ListCVEsForPatch :many
SELECT c.* FROM cves c
JOIN patch_cves pc ON c.id = pc.cve_id AND c.tenant_id = pc.tenant_id
WHERE pc.patch_id = $1 AND pc.tenant_id = $2
ORDER BY c.severity
LIMIT 1000;

-- name: UpsertDiscoveredPatch :one
INSERT INTO patches (tenant_id, name, version, severity, os_family, status,
    os_distribution, package_url, checksum_sha256, source_repo, description, package_name,
    released_at, installer_type, silent_args, hub_catalog_id)
VALUES ($1, $2, $3, $4, $5, 'available', $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
ON CONFLICT (tenant_id, name, version, os_family)
DO UPDATE SET
    severity = EXCLUDED.severity,
    os_distribution = EXCLUDED.os_distribution,
    package_url = EXCLUDED.package_url,
    checksum_sha256 = EXCLUDED.checksum_sha256,
    source_repo = EXCLUDED.source_repo,
    description = EXCLUDED.description,
    package_name = EXCLUDED.package_name,
    released_at = EXCLUDED.released_at,
    installer_type = EXCLUDED.installer_type,
    silent_args = EXCLUDED.silent_args,
    hub_catalog_id = EXCLUDED.hub_catalog_id,
    deleted_at = NULL,
    updated_at = now()
RETURNING *;

-- name: SoftDeletePatchesByHubIDs :execrows
UPDATE patches SET deleted_at = now(), updated_at = now()
WHERE tenant_id = $1 AND hub_catalog_id = ANY(@hub_ids::uuid[]) AND deleted_at IS NULL;

-- name: UpsertCVE :one
INSERT INTO cves (tenant_id, cve_id, severity, description, published_at,
    cvss_v3_score, cvss_v3_vector, cisa_kev_due_date, exploit_available, nvd_last_modified,
    attack_vector, external_references, cwe_id, source)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
ON CONFLICT (tenant_id, cve_id)
DO UPDATE SET
    severity = EXCLUDED.severity,
    description = EXCLUDED.description,
    published_at = EXCLUDED.published_at,
    cvss_v3_score = EXCLUDED.cvss_v3_score,
    cvss_v3_vector = EXCLUDED.cvss_v3_vector,
    cisa_kev_due_date = EXCLUDED.cisa_kev_due_date,
    exploit_available = EXCLUDED.exploit_available,
    nvd_last_modified = EXCLUDED.nvd_last_modified,
    attack_vector = EXCLUDED.attack_vector,
    external_references = EXCLUDED.external_references,
    cwe_id = EXCLUDED.cwe_id,
    source = EXCLUDED.source,
    updated_at = now()
RETURNING *;

-- name: UpsertAgentCVE :one
INSERT INTO cves (tenant_id, cve_id, severity, description, cvss_v3_score, source)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (tenant_id, cve_id)
DO UPDATE SET
    severity = CASE WHEN cves.source = 'nvd' THEN cves.severity ELSE EXCLUDED.severity END,
    description = CASE WHEN COALESCE(cves.description, '') != '' THEN cves.description ELSE EXCLUDED.description END,
    cvss_v3_score = CASE WHEN cves.source = 'nvd' THEN cves.cvss_v3_score ELSE EXCLUDED.cvss_v3_score END,
    source = CASE WHEN cves.source = 'nvd' THEN cves.source ELSE EXCLUDED.source END,
    updated_at = now()
RETURNING *;

-- name: ListPatchesFiltered :many
WITH patch_cve_stats AS (
    SELECT pc.patch_id,
           count(*)::int AS cve_count,
           COALESCE(MAX(c.cvss_v3_score), 0)::float8 AS highest_cvss_score
    FROM patch_cves pc
    JOIN cves c ON c.id = pc.cve_id AND c.tenant_id = pc.tenant_id
    WHERE pc.tenant_id = @tenant_id
    GROUP BY pc.patch_id
),
patch_remediation AS (
    SELECT pc.patch_id,
           CASE WHEN count(*) = 0 THEN 0
                ELSE (count(*) FILTER (WHERE ec.status = 'patched') * 100 / count(*))::int
           END AS remediation_pct,
           count(DISTINCT ec.endpoint_id) FILTER (WHERE ec.status = 'patched')::int AS endpoints_deployed_count,
           count(DISTINCT ec.endpoint_id)::int AS affected_endpoint_count
    FROM patch_cves pc
    JOIN endpoint_cves ec ON ec.cve_id = pc.cve_id AND ec.tenant_id = pc.tenant_id
    WHERE pc.tenant_id = @tenant_id
    GROUP BY pc.patch_id
)
SELECT p.*,
       COALESCE(pcs.cve_count, 0)::int AS cve_count,
       COALESCE(pcs.highest_cvss_score, 0)::float8 AS highest_cvss_score,
       COALESCE(pr.remediation_pct, 0)::int AS remediation_pct,
       COALESCE(pr.endpoints_deployed_count, 0)::int AS endpoints_deployed_count,
       COALESCE(pr.affected_endpoint_count, 0)::int AS affected_endpoint_count
FROM patches p
LEFT JOIN patch_cve_stats pcs ON pcs.patch_id = p.id
LEFT JOIN patch_remediation pr ON pr.patch_id = p.id
WHERE p.tenant_id = @tenant_id
  AND p.deleted_at IS NULL
  AND (@severity::text = '' OR p.severity = @severity)
  AND (@os_family::text = '' OR p.os_family = @os_family)
  AND (@os_distribution::text = '' OR p.os_distribution = @os_distribution)
  AND (@status::text = '' OR p.status = @status)
  AND (@search::text = '' OR p.name ILIKE '%' || @search || '%')
  AND (
    @cursor_created_at::timestamptz IS NULL
    OR (p.created_at, p.id) < (@cursor_created_at, @cursor_id::uuid)
  )
ORDER BY
  CASE WHEN @sort_by::text = 'name'     AND @sort_dir::text = 'asc'  THEN p.name END ASC NULLS LAST,
  CASE WHEN @sort_by::text = 'name'     AND @sort_dir::text = 'desc' THEN p.name END DESC NULLS LAST,
  CASE WHEN @sort_by::text = 'severity' AND @sort_dir::text = 'asc'  THEN CASE p.severity WHEN 'critical' THEN 1 WHEN 'high' THEN 2 WHEN 'medium' THEN 3 WHEN 'low' THEN 4 ELSE 5 END END ASC,
  CASE WHEN @sort_by::text = 'severity' AND @sort_dir::text = 'desc' THEN CASE p.severity WHEN 'critical' THEN 1 WHEN 'high' THEN 2 WHEN 'medium' THEN 3 WHEN 'low' THEN 4 ELSE 5 END END DESC,
  CASE WHEN @sort_by::text = 'cvss'     AND @sort_dir::text = 'asc'  THEN COALESCE(pcs.highest_cvss_score, 0) END ASC,
  CASE WHEN @sort_by::text = 'cvss'     AND @sort_dir::text = 'desc' THEN COALESCE(pcs.highest_cvss_score, 0) END DESC,
  CASE WHEN @sort_by::text = 'cves'     AND @sort_dir::text = 'asc'  THEN COALESCE(pcs.cve_count, 0) END ASC,
  CASE WHEN @sort_by::text = 'cves'     AND @sort_dir::text = 'desc' THEN COALESCE(pcs.cve_count, 0) END DESC,
  CASE WHEN @sort_by::text = 'affected' AND @sort_dir::text = 'asc'  THEN COALESCE(pr.affected_endpoint_count, 0) END ASC,
  CASE WHEN @sort_by::text = 'affected' AND @sort_dir::text = 'desc' THEN COALESCE(pr.affected_endpoint_count, 0) END DESC,
  p.created_at DESC, p.id DESC
LIMIT @page_limit;

-- name: CountPatchesFiltered :one
SELECT count(*) FROM patches p
WHERE p.tenant_id = @tenant_id
  AND p.deleted_at IS NULL
  AND (@severity::text = '' OR p.severity = @severity)
  AND (@os_family::text = '' OR p.os_family = @os_family)
  AND (@os_distribution::text = '' OR p.os_distribution = @os_distribution)
  AND (@status::text = '' OR p.status = @status)
  AND (@search::text = '' OR p.name ILIKE '%' || @search || '%');

-- name: GetPatchRemediation :one
SELECT
  count(*) FILTER (WHERE ec.status != 'patched')::int AS endpoints_affected,
  count(*) FILTER (WHERE ec.status = 'patched')::int AS endpoints_patched,
  count(*) FILTER (WHERE ec.status NOT IN ('patched', 'mitigated', 'ignored'))::int AS endpoints_pending,
  count(*) FILTER (WHERE ec.status = 'mitigated')::int AS endpoints_failed
FROM endpoint_cves ec
WHERE ec.tenant_id = @tenant_id
  AND ec.cve_id IN (
    SELECT pc.cve_id FROM patch_cves pc
    WHERE pc.patch_id = @patch_id AND pc.tenant_id = @tenant_id
  );

-- name: CountAffectedEndpointsForPatch :one
SELECT count(*) FROM (
  SELECT DISTINCT ec.endpoint_id FROM endpoint_cves ec
  WHERE ec.tenant_id = @tenant_id
    AND ec.cve_id IN (
      SELECT pc.cve_id FROM patch_cves pc
      WHERE pc.patch_id = @patch_id AND pc.tenant_id = @tenant_id
    )
) sub;

-- name: ListDeploymentsForPatch :many
SELECT DISTINCT d.id, d.status, d.started_at, d.completed_at, d.created_at,
       d.total_targets, d.success_count, d.failed_count
FROM deployments d
JOIN deployment_targets dt ON d.id = dt.deployment_id AND d.tenant_id = dt.tenant_id
WHERE dt.patch_id = @patch_id AND dt.tenant_id = @tenant_id
ORDER BY d.created_at DESC
LIMIT 20;

-- name: ListCVEsFiltered :many
SELECT c.*,
       (SELECT count(*) FROM endpoint_cves ec WHERE ec.cve_id = c.id AND ec.tenant_id = c.tenant_id)::int AS affected_endpoint_count,
       EXISTS(SELECT 1 FROM patch_cves pc WHERE pc.cve_id = c.id AND pc.tenant_id = c.tenant_id) AS patch_available,
       (SELECT count(*)::int FROM patch_cves pc WHERE pc.cve_id = c.id AND pc.tenant_id = c.tenant_id) AS patch_count
FROM cves c
WHERE c.tenant_id = @tenant_id
  AND (@severity::text = '' OR c.severity = @severity)
  AND (@cisa_kev::text = '' OR c.cisa_kev_due_date IS NOT NULL)
  AND (@exploit_available::text = '' OR c.exploit_available = (@exploit_available = 'true'))
  AND (@attack_vector::text = '' OR c.attack_vector = @attack_vector)
  AND (@search::text = '' OR c.cve_id ILIKE '%' || @search || '%')
  AND (@published_after::timestamptz IS NULL OR c.published_at >= @published_after)
  AND (@has_patch::text = '' OR (@has_patch = 'true' AND EXISTS(SELECT 1 FROM patch_cves pc2 WHERE pc2.cve_id = c.id AND pc2.tenant_id = c.tenant_id)) OR (@has_patch = 'false' AND NOT EXISTS(SELECT 1 FROM patch_cves pc3 WHERE pc3.cve_id = c.id AND pc3.tenant_id = c.tenant_id)))
  AND (
    @cursor_created_at::timestamptz IS NULL
    OR (c.created_at, c.id) < (@cursor_created_at, @cursor_id::uuid)
  )
ORDER BY c.created_at DESC, c.id DESC
LIMIT @page_limit;

-- name: CountCVEsFiltered :one
SELECT count(*) FROM cves c
WHERE c.tenant_id = @tenant_id
  AND (@severity::text = '' OR c.severity = @severity)
  AND (@cisa_kev::text = '' OR c.cisa_kev_due_date IS NOT NULL)
  AND (@exploit_available::text = '' OR c.exploit_available = (@exploit_available = 'true'))
  AND (@attack_vector::text = '' OR c.attack_vector = @attack_vector)
  AND (@search::text = '' OR c.cve_id ILIKE '%' || @search || '%')
  AND (@published_after::timestamptz IS NULL OR c.published_at >= @published_after)
  AND (@has_patch::text = '' OR (@has_patch = 'true' AND EXISTS(SELECT 1 FROM patch_cves pc2 WHERE pc2.cve_id = c.id AND pc2.tenant_id = c.tenant_id)) OR (@has_patch = 'false' AND NOT EXISTS(SELECT 1 FROM patch_cves pc3 WHERE pc3.cve_id = c.id AND pc3.tenant_id = c.tenant_id)));

-- name: ListAffectedEndpointsForCVE :many
-- group_names was a comma-joined string of group display names; in the
-- key=value world we return a comma-joined "key=value" list for the same
-- rendering slot. Column name preserved to keep the consumer DTO stable.
SELECT e.id, e.hostname, e.os_family, e.os_version, e.ip_address,
       ec.status, ec.detected_at,
       e.agent_version, e.last_seen,
       tg.group_names
FROM endpoint_cves ec
JOIN endpoints e ON ec.endpoint_id = e.id AND ec.tenant_id = e.tenant_id
LEFT JOIN LATERAL (
    SELECT string_agg(t.key || '=' || t.value, ', ' ORDER BY t.key, t.value) AS group_names
    FROM endpoint_tags et
    JOIN tags t ON t.id = et.tag_id AND t.tenant_id = et.tenant_id
    WHERE et.endpoint_id = e.id AND et.tenant_id = e.tenant_id
) tg ON true
WHERE ec.cve_id = @cve_id AND ec.tenant_id = @tenant_id
ORDER BY ec.detected_at DESC
LIMIT 50;

-- name: CountAffectedEndpointsForCVE :one
SELECT count(*) FROM endpoint_cves
WHERE cve_id = @cve_id AND tenant_id = @tenant_id;

-- name: ListPatchesForCVEDetail :many
WITH patch_stats AS (
    SELECT pc2.patch_id,
           count(DISTINCT ec2.endpoint_id)::int AS endpoints_covered,
           count(DISTINCT ec2.endpoint_id) FILTER (WHERE ec2.status = 'patched')::int AS endpoints_patched
    FROM patch_cves pc2
    JOIN endpoint_cves ec2 ON ec2.cve_id = pc2.cve_id AND ec2.tenant_id = pc2.tenant_id
    WHERE pc2.tenant_id = @tenant_id
    GROUP BY pc2.patch_id
)
SELECT p.id, p.name, p.version, p.severity, p.os_family, p.created_at,
       COALESCE(ps.endpoints_covered, 0) AS endpoints_covered,
       COALESCE(ps.endpoints_patched, 0) AS endpoints_patched
FROM patches p
JOIN patch_cves pc ON p.id = pc.patch_id AND p.tenant_id = pc.tenant_id
LEFT JOIN patch_stats ps ON ps.patch_id = p.id
WHERE pc.cve_id = @cve_id AND pc.tenant_id = @tenant_id
  AND p.deleted_at IS NULL
LIMIT 100;

-- name: CountCVEsBySeverity :many
SELECT severity, count(*)::int AS count
FROM cves
WHERE tenant_id = @tenant_id
GROUP BY severity
LIMIT 20;

-- name: CountCVEsKEV :one
SELECT count(*)::int AS count
FROM cves
WHERE tenant_id = @tenant_id AND cisa_kev_due_date IS NOT NULL;

-- name: CountCVEsExploit :one
SELECT count(*)::int AS count
FROM cves
WHERE tenant_id = @tenant_id AND exploit_available = true;

-- name: ListRelatedCVEsForCVE :many
SELECT DISTINCT c.id, c.cve_id, c.severity, c.cvss_v3_score
FROM patch_cves pc1
JOIN patch_cves pc2 ON pc1.patch_id = pc2.patch_id AND pc1.tenant_id = pc2.tenant_id
JOIN cves c ON pc2.cve_id = c.id AND pc2.tenant_id = c.tenant_id
WHERE pc1.cve_id = @cve_id AND pc1.tenant_id = @tenant_id
  AND pc2.cve_id != @cve_id
ORDER BY c.cvss_v3_score DESC NULLS LAST
LIMIT 10;

-- name: CountPatchesBySeverity :many
SELECT p.severity, count(*)::int AS count
FROM patches p
WHERE p.tenant_id = @tenant_id
  AND p.deleted_at IS NULL
  AND (@os_family::text = '' OR p.os_family = @os_family)
  AND (@os_distribution::text = '' OR p.os_distribution = @os_distribution)
  AND (@status::text = '' OR p.status = @status)
  AND (@search::text = '' OR p.name ILIKE '%' || @search || '%')
GROUP BY p.severity
LIMIT 20;

-- name: ListAffectedEndpointsForPatch :many
SELECT DISTINCT
    e.id,
    e.hostname,
    e.os_family,
    e.agent_version,
    e.status,
    e.last_seen,
    (
        SELECT MAX(dt2.completed_at)
        FROM deployment_targets dt2
        JOIN deployments d2 ON dt2.deployment_id = d2.id AND dt2.tenant_id = d2.tenant_id
        WHERE dt2.endpoint_id = e.id
          AND dt2.tenant_id = e.tenant_id
          AND dt2.patch_id = @patch_id
          AND d2.status = 'success'
    ) AS last_deployed_at,
    CASE
        WHEN NOT EXISTS (
            SELECT 1 FROM endpoint_cves ec2
            JOIN patch_cves pc2 ON ec2.cve_id = pc2.cve_id AND ec2.tenant_id = pc2.tenant_id
            WHERE pc2.patch_id = @patch_id AND ec2.endpoint_id = e.id AND ec2.tenant_id = e.tenant_id
              AND ec2.status != 'patched'
        ) THEN 'deployed'
        WHEN EXISTS (
            SELECT 1 FROM endpoint_cves ec2
            JOIN patch_cves pc2 ON ec2.cve_id = pc2.cve_id AND ec2.tenant_id = pc2.tenant_id
            WHERE pc2.patch_id = @patch_id AND ec2.endpoint_id = e.id AND ec2.tenant_id = e.tenant_id
              AND ec2.status = 'mitigated'
        ) THEN 'failed'
        ELSE 'pending'
    END AS patch_status
FROM endpoints e
JOIN endpoint_cves ec ON e.id = ec.endpoint_id AND e.tenant_id = ec.tenant_id
JOIN patch_cves pc ON ec.cve_id = pc.cve_id AND ec.tenant_id = pc.tenant_id
WHERE pc.patch_id = @patch_id AND pc.tenant_id = @tenant_id
ORDER BY e.hostname
LIMIT 50;

-- name: ListDeploymentHistoryForPatch :many
SELECT DISTINCT
    d.id,
    d.status,
    d.created_by,
    d.started_at,
    d.completed_at,
    d.total_targets,
    d.success_count,
    d.failed_count,
    d.created_at
FROM deployments d
JOIN deployment_targets dt ON d.id = dt.deployment_id AND d.tenant_id = dt.tenant_id
WHERE dt.patch_id = @patch_id AND dt.tenant_id = @tenant_id
ORDER BY d.created_at DESC
LIMIT 20;

-- name: GetPatchHighestCVSS :one
SELECT COALESCE(MAX(c.cvss_v3_score), 0)::float8 AS highest_cvss
FROM cves c
JOIN patch_cves pc ON c.id = pc.cve_id AND c.tenant_id = pc.tenant_id
WHERE pc.patch_id = @patch_id AND pc.tenant_id = @tenant_id;
