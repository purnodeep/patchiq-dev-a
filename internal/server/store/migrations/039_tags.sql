-- +goose Up

-- ============================================================
-- Replace simple name-based tags (migration 037) with
-- key-value tags, updated endpoint_tags, and tag_rules.
-- ============================================================

-- Drop old tables from migration 037.
DROP POLICY IF EXISTS endpoint_tags_tenant_isolation ON endpoint_tags;
DROP TABLE IF EXISTS endpoint_tags;
DROP POLICY IF EXISTS tags_tenant_isolation ON tags;
DROP TABLE IF EXISTS tags;

-- Tag definitions: key-value pairs for universal endpoint classification.
CREATE TABLE tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    color TEXT,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, key, value)
);

CREATE INDEX idx_tags_tenant_key ON tags(tenant_id, key);

ALTER TABLE tags ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON tags
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
ALTER TABLE tags FORCE ROW LEVEL SECURITY;
GRANT SELECT, INSERT, UPDATE, DELETE ON tags TO patchiq_app;

-- Endpoint-to-tag assignments (join table).
CREATE TABLE endpoint_tags (
    endpoint_id UUID NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    source TEXT NOT NULL DEFAULT 'manual',
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (endpoint_id, tag_id)
);

CREATE INDEX idx_endpoint_tags_tag ON endpoint_tags(tag_id, tenant_id);
CREATE INDEX idx_endpoint_tags_tenant ON endpoint_tags(tenant_id);

ALTER TABLE endpoint_tags ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON endpoint_tags
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
ALTER TABLE endpoint_tags FORCE ROW LEVEL SECURITY;
GRANT SELECT, INSERT, UPDATE, DELETE ON endpoint_tags TO patchiq_app;

-- Auto-assignment rules: automatically tag endpoints based on conditions.
CREATE TABLE tag_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name TEXT NOT NULL,
    description TEXT,
    condition JSONB NOT NULL,
    tags_to_apply UUID[] NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    priority INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_tag_rules_tenant ON tag_rules(tenant_id);

ALTER TABLE tag_rules ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON tag_rules
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
ALTER TABLE tag_rules FORCE ROW LEVEL SECURITY;
GRANT SELECT, INSERT, UPDATE, DELETE ON tag_rules TO patchiq_app;

-- +goose Down

DROP POLICY IF EXISTS tenant_isolation ON tag_rules;
DROP TABLE IF EXISTS tag_rules;
DROP POLICY IF EXISTS tenant_isolation ON endpoint_tags;
DROP TABLE IF EXISTS endpoint_tags;
DROP POLICY IF EXISTS tenant_isolation ON tags;
DROP TABLE IF EXISTS tags;

-- Restore original tables from migration 037.
CREATE TABLE tags (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    name        VARCHAR(100) NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_tags_tenant_name ON tags (tenant_id, lower(name));
CREATE INDEX idx_tags_tenant_id ON tags (tenant_id);

CREATE TABLE endpoint_tags (
    tag_id      UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    endpoint_id UUID NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tag_id, endpoint_id)
);

CREATE INDEX idx_endpoint_tags_endpoint ON endpoint_tags (endpoint_id);
CREATE INDEX idx_endpoint_tags_tenant ON endpoint_tags (tenant_id);

ALTER TABLE tags ENABLE ROW LEVEL SECURITY;
CREATE POLICY tags_tenant_isolation ON tags
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));

ALTER TABLE endpoint_tags ENABLE ROW LEVEL SECURITY;
CREATE POLICY endpoint_tags_tenant_isolation ON endpoint_tags
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));
