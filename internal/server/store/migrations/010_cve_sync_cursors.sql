-- +goose Up

-- Sync cursor tracking per tenant
CREATE TABLE cve_sync_cursors (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    source      TEXT NOT NULL,
    last_synced TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, source)
);

CREATE INDEX idx_cve_sync_cursors_tenant ON cve_sync_cursors(tenant_id);

ALTER TABLE cve_sync_cursors ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON cve_sync_cursors
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- Risk score on endpoint_cves
ALTER TABLE endpoint_cves ADD COLUMN IF NOT EXISTS risk_score DECIMAL(4,2);
ALTER TABLE endpoint_cves ADD CONSTRAINT chk_endpoint_cves_risk_score
    CHECK (risk_score IS NULL OR (risk_score >= 0.0 AND risk_score <= 10.0));

-- NVD last modified timestamp for incremental sync
ALTER TABLE cves ADD COLUMN IF NOT EXISTS nvd_last_modified TIMESTAMPTZ;

-- +goose Down

ALTER TABLE cves DROP COLUMN IF EXISTS nvd_last_modified;
ALTER TABLE endpoint_cves DROP CONSTRAINT IF EXISTS chk_endpoint_cves_risk_score;
ALTER TABLE endpoint_cves DROP COLUMN IF EXISTS risk_score;
DROP TABLE IF EXISTS cve_sync_cursors CASCADE;
