-- +goose Up

-- user_identities links external IdP users (Zitadel) to PatchIQ.
CREATE TABLE user_identities (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    external_id     TEXT NOT NULL,
    provider        TEXT NOT NULL DEFAULT 'zitadel',
    email           TEXT NOT NULL,
    display_name    TEXT NOT NULL DEFAULT '',
    provisioned_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_login_at   TIMESTAMPTZ,
    disabled        BOOLEAN NOT NULL DEFAULT false,
    UNIQUE (tenant_id, external_id, provider)
);

ALTER TABLE user_identities ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON user_identities
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
ALTER TABLE user_identities FORCE ROW LEVEL SECURITY;
GRANT SELECT, INSERT, UPDATE, DELETE ON user_identities TO patchiq_app;

-- role_mappings maps external IdP roles to PatchIQ roles (per tenant).
CREATE TABLE role_mappings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    external_role   TEXT NOT NULL,
    patchiq_role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, external_role)
);

ALTER TABLE role_mappings ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON role_mappings
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
ALTER TABLE role_mappings FORCE ROW LEVEL SECURITY;
GRANT SELECT, INSERT, UPDATE, DELETE ON role_mappings TO patchiq_app;

-- iam_settings stores per-tenant IAM configuration.
CREATE TABLE iam_settings (
    tenant_id           UUID NOT NULL REFERENCES tenants(id) PRIMARY KEY,
    zitadel_org_id      TEXT NOT NULL DEFAULT '',
    default_role_id     UUID REFERENCES roles(id),
    user_sync_enabled   BOOLEAN NOT NULL DEFAULT true,
    user_sync_interval  INTEGER NOT NULL DEFAULT 15,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE iam_settings ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON iam_settings
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
ALTER TABLE iam_settings FORCE ROW LEVEL SECURITY;
GRANT SELECT, INSERT, UPDATE ON iam_settings TO patchiq_app;

-- +goose Down
DROP TABLE IF EXISTS iam_settings;
DROP TABLE IF EXISTS role_mappings;
DROP TABLE IF EXISTS user_identities;
