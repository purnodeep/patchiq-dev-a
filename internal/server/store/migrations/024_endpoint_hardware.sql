-- +goose Up
-- ============================================================
-- Hardware & agent metadata on endpoints
-- ============================================================
ALTER TABLE endpoints ADD COLUMN IF NOT EXISTS cpu_model TEXT;
ALTER TABLE endpoints ADD COLUMN IF NOT EXISTS cpu_cores INTEGER;
ALTER TABLE endpoints ADD COLUMN IF NOT EXISTS memory_total_mb BIGINT;
ALTER TABLE endpoints ADD COLUMN IF NOT EXISTS memory_used_mb BIGINT;
ALTER TABLE endpoints ADD COLUMN IF NOT EXISTS disk_total_gb BIGINT;
ALTER TABLE endpoints ADD COLUMN IF NOT EXISTS disk_used_gb BIGINT;
ALTER TABLE endpoints ADD COLUMN IF NOT EXISTS gpu_model TEXT;
ALTER TABLE endpoints ADD COLUMN IF NOT EXISTS uptime_seconds BIGINT;
ALTER TABLE endpoints ADD COLUMN IF NOT EXISTS enrolled_at TIMESTAMPTZ;
ALTER TABLE endpoints ADD COLUMN IF NOT EXISTS last_heartbeat TIMESTAMPTZ; -- gRPC heartbeat specifically (last_seen = any activity)
ALTER TABLE endpoints ADD COLUMN IF NOT EXISTS cert_expiry TIMESTAMPTZ;

-- ============================================================
-- Network interfaces per endpoint
-- ============================================================
CREATE TABLE endpoint_network_interfaces (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    endpoint_id UUID NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    ip_address  TEXT,
    mac_address TEXT,
    status      TEXT NOT NULL DEFAULT 'up',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_eni_status CHECK (status IN ('up', 'down'))
);

CREATE INDEX idx_eni_tenant_endpoint ON endpoint_network_interfaces(tenant_id, endpoint_id);

ALTER TABLE endpoint_network_interfaces ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON endpoint_network_interfaces
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
ALTER TABLE endpoint_network_interfaces FORCE ROW LEVEL SECURITY;

GRANT SELECT, INSERT, UPDATE, DELETE ON endpoint_network_interfaces TO patchiq_app;

-- +goose Down
-- ============================================================
-- Drop network interfaces table
-- ============================================================
REVOKE ALL ON endpoint_network_interfaces FROM patchiq_app;
DROP TABLE IF EXISTS endpoint_network_interfaces CASCADE;

-- ============================================================
-- Remove hardware & agent metadata columns
-- ============================================================
ALTER TABLE endpoints DROP COLUMN IF EXISTS cpu_model;
ALTER TABLE endpoints DROP COLUMN IF EXISTS cpu_cores;
ALTER TABLE endpoints DROP COLUMN IF EXISTS memory_total_mb;
ALTER TABLE endpoints DROP COLUMN IF EXISTS memory_used_mb;
ALTER TABLE endpoints DROP COLUMN IF EXISTS disk_total_gb;
ALTER TABLE endpoints DROP COLUMN IF EXISTS disk_used_gb;
ALTER TABLE endpoints DROP COLUMN IF EXISTS gpu_model;
ALTER TABLE endpoints DROP COLUMN IF EXISTS uptime_seconds;
ALTER TABLE endpoints DROP COLUMN IF EXISTS enrolled_at;
ALTER TABLE endpoints DROP COLUMN IF EXISTS last_heartbeat;
ALTER TABLE endpoints DROP COLUMN IF EXISTS cert_expiry;
