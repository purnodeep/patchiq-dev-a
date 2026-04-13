-- name: CreateTag :one
INSERT INTO tags (tenant_id, key, value, description)
VALUES (@tenant_id, @key, @value, @description)
RETURNING *;

-- name: GetTagByID :one
SELECT * FROM tags WHERE id = @id AND tenant_id = @tenant_id;

-- name: GetTagByKeyValue :one
SELECT * FROM tags
WHERE tenant_id = @tenant_id
  AND lower(key) = lower(@key)
  AND lower(value) = lower(@value);

-- name: ListTags :many
SELECT t.*,
    (SELECT count(*) FROM endpoint_tags et WHERE et.tag_id = t.id AND et.tenant_id = t.tenant_id)::bigint AS endpoint_count
FROM tags t
WHERE t.tenant_id = @tenant_id
  AND (@key_filter::text = '' OR lower(t.key) = lower(@key_filter))
ORDER BY t.key, t.value
LIMIT 1000;

-- name: ListDistinctTagKeys :many
SELECT lower(key) AS key, count(*)::bigint AS value_count
FROM tags
WHERE tenant_id = @tenant_id
GROUP BY lower(key)
ORDER BY lower(key);

-- name: UpdateTag :one
UPDATE tags SET
    description = @description,
    updated_at = now()
WHERE id = @id AND tenant_id = @tenant_id
RETURNING *;

-- name: DeleteTag :exec
DELETE FROM tags WHERE id = @id AND tenant_id = @tenant_id;

-- name: AssignTagToEndpoint :exec
INSERT INTO endpoint_tags (endpoint_id, tag_id, tenant_id, source)
VALUES (@endpoint_id, @tag_id, @tenant_id, @source)
ON CONFLICT (endpoint_id, tag_id) DO NOTHING;

-- name: RemoveTagFromEndpoint :exec
DELETE FROM endpoint_tags
WHERE endpoint_id = @endpoint_id AND tag_id = @tag_id AND tenant_id = @tenant_id;

-- name: RemoveEndpointTagsByKey :exec
-- Used by AssignTag for keys flagged as exclusive in tag_keys: strips any
-- existing values for `key` from each endpoint before assigning the new
-- tag, so an endpoint never carries two values for a single-valued key.
DELETE FROM endpoint_tags et
USING tags t
WHERE et.tag_id = t.id
  AND et.tenant_id = @tenant_id
  AND et.endpoint_id = ANY(@endpoint_ids::UUID[])
  AND lower(t.key) = lower(@key);

-- name: ListTagsForEndpoint :many
SELECT t.* FROM tags t
JOIN endpoint_tags et ON et.tag_id = t.id
WHERE et.endpoint_id = @endpoint_id AND et.tenant_id = @tenant_id
ORDER BY t.key, t.value
LIMIT 100;

-- name: ListEndpointsByTag :many
SELECT et.endpoint_id FROM endpoint_tags et
WHERE et.tag_id = @tag_id AND et.tenant_id = @tenant_id
LIMIT 5000;

-- name: CountEndpointsByTag :one
SELECT COUNT(*) FROM endpoint_tags et
WHERE et.tag_id = @tag_id AND et.tenant_id = @tenant_id;

-- name: BulkAssignTag :exec
INSERT INTO endpoint_tags (endpoint_id, tag_id, tenant_id, source)
SELECT unnest(@endpoint_ids::UUID[]), @tag_id, @tenant_id, @source
ON CONFLICT (endpoint_id, tag_id) DO NOTHING;
