-- name: UpsertPolicyTagSelector :one
INSERT INTO policy_tag_selectors (policy_id, tenant_id, expression)
VALUES (@policy_id, @tenant_id, @expression)
ON CONFLICT (policy_id) DO UPDATE SET
    expression = EXCLUDED.expression,
    updated_at = now()
RETURNING *;

-- name: GetPolicyTagSelector :one
SELECT * FROM policy_tag_selectors
WHERE policy_id = @policy_id AND tenant_id = @tenant_id;

-- name: DeletePolicyTagSelector :exec
DELETE FROM policy_tag_selectors
WHERE policy_id = @policy_id AND tenant_id = @tenant_id;

-- name: ListPolicyTagSelectorsForTenant :many
SELECT * FROM policy_tag_selectors
WHERE tenant_id = @tenant_id
ORDER BY updated_at DESC
LIMIT 1000;
