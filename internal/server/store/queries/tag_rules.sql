-- name: CreateTagRule :one
INSERT INTO tag_rules (tenant_id, name, description, condition, tags_to_apply, enabled, priority)
VALUES (@tenant_id, @name, @description, @condition, @tags_to_apply, @enabled, @priority)
RETURNING *;

-- name: GetTagRuleByID :one
SELECT * FROM tag_rules WHERE id = @id AND tenant_id = @tenant_id;

-- name: ListTagRules :many
SELECT * FROM tag_rules WHERE tenant_id = @tenant_id ORDER BY priority DESC, name
LIMIT 100;

-- name: ListEnabledTagRules :many
SELECT * FROM tag_rules WHERE tenant_id = @tenant_id AND enabled = true ORDER BY priority DESC
LIMIT 100;

-- name: UpdateTagRule :one
UPDATE tag_rules SET
    name = @name,
    description = @description,
    condition = @condition,
    tags_to_apply = @tags_to_apply,
    enabled = @enabled,
    priority = @priority,
    updated_at = now()
WHERE id = @id AND tenant_id = @tenant_id
RETURNING *;

-- name: DeleteTagRule :exec
DELETE FROM tag_rules WHERE id = @id AND tenant_id = @tenant_id;
