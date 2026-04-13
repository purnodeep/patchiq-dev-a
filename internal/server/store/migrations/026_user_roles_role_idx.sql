-- +goose Up
CREATE INDEX IF NOT EXISTS idx_user_roles_role ON user_roles(tenant_id, role_id);

-- +goose Down
DROP INDEX IF EXISTS idx_user_roles_role;
