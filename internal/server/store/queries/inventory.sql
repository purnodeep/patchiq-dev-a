-- name: CreateEndpointInventory :one
INSERT INTO endpoint_inventories (tenant_id, endpoint_id, scanned_at, package_count, collection_errors)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetEndpointInventoryByID :one
SELECT * FROM endpoint_inventories WHERE id = $1 AND tenant_id = $2;

-- name: ListEndpointInventories :many
SELECT * FROM endpoint_inventories
WHERE endpoint_id = $1 AND tenant_id = $2
ORDER BY scanned_at DESC
LIMIT 1000;

-- name: GetLatestEndpointInventory :one
SELECT * FROM endpoint_inventories
WHERE endpoint_id = $1 AND tenant_id = $2
ORDER BY scanned_at DESC
LIMIT 1;

-- name: CreateEndpointPackage :one
INSERT INTO endpoint_packages (tenant_id, endpoint_id, inventory_id, package_name, version, arch, source, release, kb_article, severity, install_date, category, publisher)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
RETURNING *;

-- name: ListEndpointPackages :many
SELECT * FROM endpoint_packages
WHERE inventory_id = $1 AND tenant_id = $2
ORDER BY package_name
LIMIT 5000;

-- name: ListEndpointPackagesByEndpoint :many
SELECT ep.id, ep.package_name, ep.version, ep.arch, ep.source, ep.release, ep.created_at
FROM endpoint_packages ep
JOIN endpoint_inventories ei ON ep.inventory_id = ei.id
WHERE ep.endpoint_id = $1 AND ep.tenant_id = $2
  AND ei.id = (
    SELECT id FROM endpoint_inventories
    WHERE endpoint_id = $1 AND tenant_id = $2
    ORDER BY scanned_at DESC LIMIT 1
  )
ORDER BY ep.package_name
LIMIT 5000;
