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
-- This UUID (00000000-0000-0000-0000-000000000001) is also used as the
-- defaultTenant constant in db_test.go. Changing it requires updating tests.
INSERT INTO tenants (id, name, slug) VALUES
    ('00000000-0000-0000-0000-000000000001', 'Default', 'default')
ON CONFLICT DO NOTHING;

-- ============================================================
-- Tenant-scoped tables (tenant_id first after PK)
-- ============================================================

CREATE TABLE endpoints (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id),
    hostname      TEXT NOT NULL,
    os_family     TEXT NOT NULL,
    os_version    TEXT NOT NULL,
    agent_version TEXT,
    status        TEXT NOT NULL DEFAULT 'pending',
    last_seen     TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_endpoints_tenant ON endpoints(tenant_id);
CREATE INDEX idx_endpoints_tenant_status ON endpoints(tenant_id, status);

CREATE TABLE endpoint_groups (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    name        TEXT NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_endpoint_groups_tenant ON endpoint_groups(tenant_id);

CREATE TABLE endpoint_group_members (
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    endpoint_id UUID NOT NULL REFERENCES endpoints(id),
    group_id    UUID NOT NULL REFERENCES endpoint_groups(id),
    added_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, endpoint_id, group_id)
);

-- M0 simplification: patches and cves are tenant-scoped for now.
-- In M1/M2, global patch_catalog and cve_feeds tables will be added
-- when Hub Manager integration provides shared catalog data.
CREATE TABLE patches (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL REFERENCES tenants(id),
    name       TEXT NOT NULL,
    version    TEXT NOT NULL,
    severity   TEXT NOT NULL DEFAULT 'none',
    os_family  TEXT NOT NULL,
    status     TEXT NOT NULL DEFAULT 'available',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_patches_tenant ON patches(tenant_id);

CREATE TABLE cves (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id),
    cve_id       TEXT NOT NULL,
    severity     TEXT NOT NULL,
    description  TEXT,
    published_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, cve_id)
);

CREATE INDEX idx_cves_tenant ON cves(tenant_id);

CREATE TABLE patch_cves (
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    patch_id  UUID NOT NULL REFERENCES patches(id),
    cve_id    UUID NOT NULL REFERENCES cves(id),
    PRIMARY KEY (tenant_id, patch_id, cve_id)
);

CREATE TABLE policies (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL REFERENCES tenants(id),
    name               TEXT NOT NULL,
    description        TEXT,
    schedule           TEXT,
    maintenance_window TEXT,
    enabled            BOOLEAN NOT NULL DEFAULT true,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_policies_tenant ON policies(tenant_id);

CREATE TABLE policy_groups (
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    policy_id UUID NOT NULL REFERENCES policies(id),
    group_id  UUID NOT NULL REFERENCES endpoint_groups(id),
    PRIMARY KEY (tenant_id, policy_id, group_id)
);

CREATE TABLE deployments (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id),
    policy_id    UUID NOT NULL REFERENCES policies(id),
    status       TEXT NOT NULL DEFAULT 'created',
    created_by   UUID,
    started_at   TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_deployments_tenant ON deployments(tenant_id);
CREATE INDEX idx_deployments_tenant_status ON deployments(tenant_id, status);

CREATE TABLE deployment_targets (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id),
    deployment_id UUID NOT NULL REFERENCES deployments(id),
    endpoint_id   UUID NOT NULL REFERENCES endpoints(id),
    patch_id      UUID NOT NULL REFERENCES patches(id),
    status        TEXT NOT NULL DEFAULT 'pending',
    started_at    TIMESTAMPTZ,
    completed_at  TIMESTAMPTZ,
    error_message TEXT
);

CREATE INDEX idx_deployment_targets_tenant_deployment ON deployment_targets(tenant_id, deployment_id);

CREATE TABLE deployment_waves (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id),
    deployment_id UUID NOT NULL REFERENCES deployments(id),
    wave_number   INTEGER NOT NULL,
    status        TEXT NOT NULL DEFAULT 'pending',
    started_at    TIMESTAMPTZ,
    completed_at  TIMESTAMPTZ,
    UNIQUE (tenant_id, deployment_id, wave_number)
);

CREATE INDEX idx_deployment_waves_tenant_deployment ON deployment_waves(tenant_id, deployment_id);

CREATE TABLE agent_registrations (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL REFERENCES tenants(id),
    endpoint_id        UUID REFERENCES endpoints(id),
    registration_token TEXT NOT NULL UNIQUE,
    status             TEXT NOT NULL DEFAULT 'pending',
    registered_at      TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_agent_registrations_tenant ON agent_registrations(tenant_id);

CREATE TABLE config_overrides (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL REFERENCES tenants(id),
    scope_type TEXT NOT NULL,
    scope_id   UUID NOT NULL,
    module     TEXT NOT NULL,
    config     JSONB NOT NULL,
    updated_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, scope_type, scope_id, module)
);

CREATE INDEX idx_config_overrides_scope ON config_overrides(tenant_id, scope_type, scope_id);

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
DROP TABLE IF EXISTS config_overrides CASCADE;
DROP TABLE IF EXISTS agent_registrations CASCADE;
DROP TABLE IF EXISTS deployment_waves CASCADE;
DROP TABLE IF EXISTS deployment_targets CASCADE;
DROP TABLE IF EXISTS deployments CASCADE;
DROP TABLE IF EXISTS policy_groups CASCADE;
DROP TABLE IF EXISTS policies CASCADE;
DROP TABLE IF EXISTS patch_cves CASCADE;
DROP TABLE IF EXISTS cves CASCADE;
DROP TABLE IF EXISTS patches CASCADE;
DROP TABLE IF EXISTS endpoint_group_members CASCADE;
DROP TABLE IF EXISTS endpoint_groups CASCADE;
DROP TABLE IF EXISTS endpoints CASCADE;
DROP TABLE IF EXISTS tenants CASCADE;
