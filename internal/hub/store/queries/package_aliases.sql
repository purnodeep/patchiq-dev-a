-- name: GetPackageAlias :one
SELECT * FROM package_aliases
WHERE feed_product = $1 AND os_family = $2 AND os_distribution = $3;

-- name: UpsertPackageAlias :one
INSERT INTO package_aliases (feed_product, os_family, os_distribution, os_package_name, confidence)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (feed_product, os_family, os_distribution) DO UPDATE SET
    os_package_name = EXCLUDED.os_package_name,
    confidence = EXCLUDED.confidence,
    updated_at = now()
RETURNING *;

-- name: ListPackageAliases :many
SELECT * FROM package_aliases
ORDER BY feed_product, os_family, os_distribution
LIMIT $1 OFFSET $2;

-- name: ListPackageAliasesByProduct :many
SELECT * FROM package_aliases
WHERE feed_product = $1
ORDER BY os_family, os_distribution
LIMIT 1000;

-- name: UpdatePackageAliasById :one
UPDATE package_aliases
SET feed_product = $2,
    os_family = $3,
    os_distribution = $4,
    os_package_name = $5,
    confidence = $6,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeletePackageAlias :exec
DELETE FROM package_aliases WHERE id = $1;
