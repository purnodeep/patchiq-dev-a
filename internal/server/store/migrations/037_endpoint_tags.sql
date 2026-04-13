-- +goose Up
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

-- +goose Down
DROP TABLE IF EXISTS endpoint_tags;
DROP TABLE IF EXISTS tags;
