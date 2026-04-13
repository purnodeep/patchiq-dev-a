-- +goose NO TRANSACTION
-- +goose Up
-- Hot-path indexes for tables frequently queried under load.

-- deployment_targets: queried by endpoint, patch, and status within deployments.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_deployment_targets_endpoint_tenant
    ON deployment_targets(endpoint_id, tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_deployment_targets_patch_tenant
    ON deployment_targets(patch_id, tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_deployment_targets_deployment_status
    ON deployment_targets(deployment_id, status);

-- endpoint_cves: queried by cve_id for vulnerability correlation.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_endpoint_cves_cve_tenant
    ON endpoint_cves(cve_id, tenant_id);

-- cves: filtered by severity per tenant.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_cves_tenant_severity
    ON cves(tenant_id, severity);

-- patches: filtered by os_family and severity per tenant.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_patches_tenant_os_family
    ON patches(tenant_id, os_family);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_patches_tenant_severity
    ON patches(tenant_id, severity);

-- +goose Down
DROP INDEX IF EXISTS idx_patches_tenant_severity;
DROP INDEX IF EXISTS idx_patches_tenant_os_family;
DROP INDEX IF EXISTS idx_cves_tenant_severity;
DROP INDEX IF EXISTS idx_endpoint_cves_cve_tenant;
DROP INDEX IF EXISTS idx_deployment_targets_deployment_status;
DROP INDEX IF EXISTS idx_deployment_targets_patch_tenant;
DROP INDEX IF EXISTS idx_deployment_targets_endpoint_tenant;
