-- name: GetDashboardStats :one
SELECT
    (SELECT count(*) FROM patch_catalog WHERE deleted_at IS NULL) AS total_catalog_entries,
    (SELECT count(*) FROM feed_sources WHERE enabled = true) AS active_feeds,
    (SELECT count(*) FROM clients WHERE status = 'approved') AS connected_clients,
    (SELECT count(*) FROM clients WHERE status = 'pending') AS pending_clients,
    (SELECT count(*) FROM licenses WHERE revoked_at IS NULL AND expires_at > now()) AS active_licenses;

-- name: GetDashboardLicenseBreakdown :many
SELECT
    tier,
    CASE
        WHEN revoked_at IS NOT NULL THEN 'revoked'
        WHEN expires_at <= now() THEN 'expired'
        WHEN expires_at <= now() + INTERVAL '30 days' THEN 'expiring'
        ELSE 'active'
    END AS status,
    count(*)::int AS count,
    COALESCE(sum(max_endpoints), 0)::bigint AS total_endpoints
FROM licenses
GROUP BY tier, status
ORDER BY tier, status;

-- name: GetDashboardCatalogGrowth :many
SELECT
    date_trunc('day', created_at)::date AS day,
    count(*)::int AS entries_added
FROM patch_catalog
WHERE created_at >= now() - make_interval(days => sqlc.arg('days')::int)
  AND deleted_at IS NULL
GROUP BY day
ORDER BY day;

-- name: GetDashboardClientSummary :many
SELECT
    id, hostname, status, endpoint_count, last_sync_at, version, os
FROM clients
WHERE status IN ('approved', 'pending')
ORDER BY hostname;
