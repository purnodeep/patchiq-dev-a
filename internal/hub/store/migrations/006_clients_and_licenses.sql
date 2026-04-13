-- +goose Up

-- ============================================================
-- clients table (tenant-scoped)
-- ============================================================

CREATE TABLE clients (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    hostname        TEXT NOT NULL,
    version         TEXT,
    os              TEXT,
    endpoint_count  INT NOT NULL DEFAULT 0,
    contact_email   TEXT,
    status          TEXT NOT NULL DEFAULT 'pending',
    api_key_hash    TEXT,
    bootstrap_token TEXT NOT NULL,
    sync_interval   INT NOT NULL DEFAULT 21600,
    last_sync_at    TIMESTAMPTZ,
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_client_status CHECK (status IN ('pending', 'approved', 'declined', 'suspended'))
);

CREATE INDEX idx_clients_tenant_status ON clients(tenant_id, status);
CREATE INDEX idx_clients_bootstrap_token ON clients(bootstrap_token) WHERE bootstrap_token IS NOT NULL;
CREATE UNIQUE INDEX idx_clients_api_key_hash ON clients(api_key_hash) WHERE api_key_hash IS NOT NULL;

-- RLS
ALTER TABLE clients ENABLE ROW LEVEL SECURITY;
ALTER TABLE clients FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON clients
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);

GRANT SELECT, INSERT, UPDATE, DELETE ON clients TO hub_app;

-- ============================================================
-- licenses table (tenant-scoped)
-- ============================================================

CREATE TABLE licenses (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL REFERENCES tenants(id),
    client_id      UUID REFERENCES clients(id) ON DELETE SET NULL,
    license_key    TEXT NOT NULL,
    tier           TEXT NOT NULL,
    max_endpoints  INT NOT NULL,
    issued_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at     TIMESTAMPTZ NOT NULL,
    revoked_at     TIMESTAMPTZ,
    customer_name  TEXT NOT NULL,
    customer_email TEXT,
    notes          TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_license_tier CHECK (tier IN ('community', 'professional', 'enterprise', 'msp'))
);

CREATE INDEX idx_licenses_tenant ON licenses(tenant_id);
CREATE INDEX idx_licenses_client ON licenses(client_id) WHERE client_id IS NOT NULL;
CREATE INDEX idx_licenses_tenant_tier ON licenses(tenant_id, tier);

-- RLS
ALTER TABLE licenses ENABLE ROW LEVEL SECURITY;
ALTER TABLE licenses FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON licenses
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);

GRANT SELECT, INSERT, UPDATE, DELETE ON licenses TO hub_app;

-- +goose Down

DROP TABLE IF EXISTS licenses CASCADE;
DROP TABLE IF EXISTS clients CASCADE;
