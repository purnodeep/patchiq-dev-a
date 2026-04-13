-- +goose Up

-- ============================================================
-- M1 Core Loop: new columns on existing tables
-- ============================================================

-- endpoints: richer device metadata
ALTER TABLE endpoints ADD COLUMN IF NOT EXISTS ip_address TEXT;
ALTER TABLE endpoints ADD COLUMN IF NOT EXISTS arch TEXT;
ALTER TABLE endpoints ADD COLUMN IF NOT EXISTS kernel_version TEXT;

-- patches: richer package metadata
ALTER TABLE patches ADD COLUMN IF NOT EXISTS os_distribution TEXT;
ALTER TABLE patches ADD COLUMN IF NOT EXISTS package_url TEXT;
ALTER TABLE patches ADD COLUMN IF NOT EXISTS checksum_sha256 TEXT;
ALTER TABLE patches ADD COLUMN IF NOT EXISTS source_repo TEXT;
ALTER TABLE patches ADD COLUMN IF NOT EXISTS description TEXT;

-- cves: CVSS v3 scoring and exploit tracking
ALTER TABLE cves ADD COLUMN IF NOT EXISTS cvss_v3_score DECIMAL(3,1);
ALTER TABLE cves ADD COLUMN IF NOT EXISTS cvss_v3_vector TEXT;
ALTER TABLE cves ADD COLUMN IF NOT EXISTS cisa_kev_due_date DATE;
ALTER TABLE cves ADD COLUMN IF NOT EXISTS exploit_available BOOLEAN NOT NULL DEFAULT false;

-- deployment_targets: capture command output
ALTER TABLE deployment_targets ADD COLUMN IF NOT EXISTS stdout TEXT;
ALTER TABLE deployment_targets ADD COLUMN IF NOT EXISTS stderr TEXT;
ALTER TABLE deployment_targets ADD COLUMN IF NOT EXISTS exit_code INTEGER;

-- ============================================================
-- M1 Core Loop: new tables
-- ============================================================

-- Per-endpoint software inventory snapshots
CREATE TABLE endpoint_inventories (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    endpoint_id UUID NOT NULL REFERENCES endpoints(id),
    scanned_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    package_count INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_endpoint_inventories_endpoint_scanned
    ON endpoint_inventories(endpoint_id, scanned_at DESC);

-- Individual packages discovered on endpoints
CREATE TABLE endpoint_packages (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id),
    endpoint_id  UUID NOT NULL REFERENCES endpoints(id),
    inventory_id UUID NOT NULL REFERENCES endpoint_inventories(id),
    package_name TEXT NOT NULL,
    version      TEXT NOT NULL,
    arch         TEXT,
    source       TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_endpoint_packages_endpoint_name
    ON endpoint_packages(endpoint_id, package_name);
CREATE UNIQUE INDEX idx_endpoint_packages_unique
    ON endpoint_packages(tenant_id, inventory_id, package_name, version);

-- CVE-to-endpoint vulnerability tracking
CREATE TABLE endpoint_cves (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    endpoint_id UUID NOT NULL REFERENCES endpoints(id),
    cve_id      UUID NOT NULL REFERENCES cves(id),
    status      TEXT NOT NULL DEFAULT 'affected',
    detected_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_endpoint_cves_endpoint_status
    ON endpoint_cves(endpoint_id, status);
CREATE UNIQUE INDEX idx_endpoint_cves_unique
    ON endpoint_cves(tenant_id, endpoint_id, cve_id);

-- CHECK constraints for new tables
ALTER TABLE endpoint_cves ADD CONSTRAINT chk_endpoint_cves_status
    CHECK (status IN ('affected', 'patched', 'mitigated', 'ignored'));

ALTER TABLE cves ADD CONSTRAINT chk_cves_cvss_v3_range
    CHECK (cvss_v3_score IS NULL OR (cvss_v3_score >= 0.0 AND cvss_v3_score <= 10.0));

-- +goose Down

DROP TABLE IF EXISTS endpoint_cves CASCADE;
DROP TABLE IF EXISTS endpoint_packages CASCADE;
DROP TABLE IF EXISTS endpoint_inventories CASCADE;

ALTER TABLE deployment_targets DROP COLUMN IF EXISTS stdout;
ALTER TABLE deployment_targets DROP COLUMN IF EXISTS stderr;
ALTER TABLE deployment_targets DROP COLUMN IF EXISTS exit_code;

ALTER TABLE cves DROP CONSTRAINT IF EXISTS chk_cves_cvss_v3_range;
ALTER TABLE cves DROP COLUMN IF EXISTS cvss_v3_score;
ALTER TABLE cves DROP COLUMN IF EXISTS cvss_v3_vector;
ALTER TABLE cves DROP COLUMN IF EXISTS cisa_kev_due_date;
ALTER TABLE cves DROP COLUMN IF EXISTS exploit_available;

ALTER TABLE patches DROP COLUMN IF EXISTS os_distribution;
ALTER TABLE patches DROP COLUMN IF EXISTS package_url;
ALTER TABLE patches DROP COLUMN IF EXISTS checksum_sha256;
ALTER TABLE patches DROP COLUMN IF EXISTS source_repo;
ALTER TABLE patches DROP COLUMN IF EXISTS description;

ALTER TABLE endpoints DROP COLUMN IF EXISTS ip_address;
ALTER TABLE endpoints DROP COLUMN IF EXISTS arch;
ALTER TABLE endpoints DROP COLUMN IF EXISTS kernel_version;
