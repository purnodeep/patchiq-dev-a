-- +goose Up

-- ============================================================
-- Global tables (no tenant_id, no RLS)
-- ============================================================

CREATE TABLE tenants (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    license_id  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Default tenant for single-tenant deployments.
INSERT INTO tenants (id, name, slug) VALUES
    ('00000000-0000-0000-0000-000000000001', 'Default', 'default')
ON CONFLICT DO NOTHING;

CREATE TABLE patch_catalog (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name         TEXT NOT NULL,
    vendor       TEXT NOT NULL,
    os_family    TEXT NOT NULL,
    version      TEXT NOT NULL,
    severity     TEXT NOT NULL DEFAULT 'none',
    release_date TIMESTAMPTZ,
    description  TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE cve_feeds (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cve_id       TEXT NOT NULL UNIQUE,
    severity     TEXT NOT NULL,
    description  TEXT,
    published_at TIMESTAMPTZ,
    source       TEXT NOT NULL DEFAULT 'nist',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE agent_binaries (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    os_family   TEXT NOT NULL,
    arch        TEXT NOT NULL,
    version     TEXT NOT NULL,
    download_url TEXT NOT NULL,
    checksum    TEXT NOT NULL,
    released_at TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (os_family, arch, version)
);

-- ============================================================
-- Tenant-scoped tables (tenant_id first after PK)
-- ============================================================

CREATE TABLE hub_config (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL REFERENCES tenants(id),
    key        TEXT NOT NULL,
    value      JSONB NOT NULL,
    updated_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, key)
);

CREATE INDEX idx_hub_config_tenant ON hub_config(tenant_id);

-- ============================================================
-- Audit events (partitioned, append-only, ULID PK)
-- ============================================================

CREATE TABLE audit_events (
    id          TEXT NOT NULL,
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    type        TEXT NOT NULL,
    actor_id    TEXT NOT NULL,
    actor_type  TEXT NOT NULL,
    resource    TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    action      TEXT NOT NULL,
    payload     JSONB,
    metadata    JSONB,
    timestamp   TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (id, timestamp)
) PARTITION BY RANGE (timestamp);

-- Monthly partitions for 2026 (hardcoded for M0).
-- Inserts with timestamps outside defined ranges route to the default partition.
-- A future migration should create partitions for subsequent years or adopt pg_partman.
CREATE TABLE audit_events_2026_01 PARTITION OF audit_events FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');
CREATE TABLE audit_events_2026_02 PARTITION OF audit_events FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');
CREATE TABLE audit_events_2026_03 PARTITION OF audit_events FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');
CREATE TABLE audit_events_2026_04 PARTITION OF audit_events FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');
CREATE TABLE audit_events_2026_05 PARTITION OF audit_events FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');
CREATE TABLE audit_events_2026_06 PARTITION OF audit_events FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');
CREATE TABLE audit_events_2026_07 PARTITION OF audit_events FOR VALUES FROM ('2026-07-01') TO ('2026-08-01');
CREATE TABLE audit_events_2026_08 PARTITION OF audit_events FOR VALUES FROM ('2026-08-01') TO ('2026-09-01');
CREATE TABLE audit_events_2026_09 PARTITION OF audit_events FOR VALUES FROM ('2026-09-01') TO ('2026-10-01');
CREATE TABLE audit_events_2026_10 PARTITION OF audit_events FOR VALUES FROM ('2026-10-01') TO ('2026-11-01');
CREATE TABLE audit_events_2026_11 PARTITION OF audit_events FOR VALUES FROM ('2026-11-01') TO ('2026-12-01');
CREATE TABLE audit_events_2026_12 PARTITION OF audit_events FOR VALUES FROM ('2026-12-01') TO ('2027-01-01');
-- Default partition catches out-of-range timestamps (pre-2026 or post-2026).
-- WARNING: Rows landing here indicate missing partitions for new time ranges.
CREATE TABLE audit_events_default PARTITION OF audit_events DEFAULT;

CREATE INDEX idx_audit_tenant_time ON audit_events(tenant_id, timestamp DESC);
CREATE INDEX idx_audit_resource ON audit_events(tenant_id, resource, resource_id);
CREATE INDEX idx_audit_actor ON audit_events(tenant_id, actor_id);
CREATE INDEX idx_audit_type ON audit_events(tenant_id, type);

-- +goose Down

DROP TABLE IF EXISTS audit_events CASCADE;
DROP TABLE IF EXISTS hub_config CASCADE;
DROP TABLE IF EXISTS agent_binaries CASCADE;
DROP TABLE IF EXISTS cve_feeds CASCADE;
DROP TABLE IF EXISTS patch_catalog CASCADE;
DROP TABLE IF EXISTS tenants CASCADE;
