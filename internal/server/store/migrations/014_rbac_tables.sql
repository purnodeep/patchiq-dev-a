-- +goose Up

-- ============================================================
-- RBAC tables: roles, role_permissions, user_roles
-- ============================================================

CREATE TABLE roles (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    name            TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    parent_role_id  UUID REFERENCES roles(id),
    is_system       BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, name),
    CONSTRAINT chk_roles_name_not_empty CHECK (name <> '')
);

CREATE INDEX idx_roles_tenant ON roles(tenant_id);
CREATE INDEX idx_roles_parent ON roles(parent_role_id);

CREATE TABLE role_permissions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    role_id     UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    resource    TEXT NOT NULL,
    action      TEXT NOT NULL,
    scope       TEXT NOT NULL,
    UNIQUE (tenant_id, role_id, resource, action, scope),
    CONSTRAINT chk_rp_resource_not_empty CHECK (resource <> ''),
    CONSTRAINT chk_rp_action_not_empty CHECK (action <> ''),
    CONSTRAINT chk_rp_scope_not_empty CHECK (scope <> '')
);

CREATE INDEX idx_role_permissions_role ON role_permissions(role_id);
CREATE INDEX idx_role_permissions_resource ON role_permissions(tenant_id, resource);

CREATE TABLE user_roles (
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    user_id     TEXT NOT NULL,
    role_id     UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, user_id, role_id)
);

CREATE INDEX idx_user_roles_user ON user_roles(tenant_id, user_id);

-- ============================================================
-- RLS policies
-- ============================================================

ALTER TABLE roles ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON roles
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
ALTER TABLE roles FORCE ROW LEVEL SECURITY;

ALTER TABLE role_permissions ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON role_permissions
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
ALTER TABLE role_permissions FORCE ROW LEVEL SECURITY;

ALTER TABLE user_roles ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON user_roles
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
ALTER TABLE user_roles FORCE ROW LEVEL SECURITY;

-- ============================================================
-- Grants for patchiq_app role
-- ============================================================

GRANT SELECT, INSERT, UPDATE, DELETE ON roles TO patchiq_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON role_permissions TO patchiq_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON user_roles TO patchiq_app;

-- +goose Down

DROP TABLE IF EXISTS user_roles CASCADE;
DROP TABLE IF EXISTS role_permissions CASCADE;
DROP TABLE IF EXISTS roles CASCADE;
