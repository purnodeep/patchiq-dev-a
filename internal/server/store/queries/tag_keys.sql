-- name: UpsertTagKey :one
INSERT INTO tag_keys (tenant_id, key, description, exclusive, value_type)
VALUES (@tenant_id, @key, @description, @exclusive, @value_type)
ON CONFLICT (tenant_id, key) DO UPDATE SET
    description = EXCLUDED.description,
    exclusive   = EXCLUDED.exclusive,
    value_type  = EXCLUDED.value_type,
    updated_at  = now()
RETURNING *;

-- name: GetTagKey :one
SELECT * FROM tag_keys
WHERE tenant_id = @tenant_id AND key = @key;

-- name: ListTagKeys :many
SELECT * FROM tag_keys
WHERE tenant_id = @tenant_id
ORDER BY key
LIMIT 1000;

-- name: DeleteTagKey :exec
DELETE FROM tag_keys
WHERE tenant_id = @tenant_id AND key = @key;

-- name: IsKeyExclusive :one
SELECT exclusive FROM tag_keys
WHERE tenant_id = @tenant_id AND key = @key;
