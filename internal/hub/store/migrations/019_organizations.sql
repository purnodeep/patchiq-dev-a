-- +goose Up

-- ============================================================
-- Organizations (hub side): parent entity above tenants for the MSP model.
-- Mirrors the server-side organizations table (migration 059) with the
-- addition that hub licenses can be issued at the organization level as
-- umbrella licenses.
--
-- Global table (no tenant_id, no RLS). See docs/adr/025.
-- ============================================================

CREATE TABLE organizations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL,
    slug            TEXT NOT NULL UNIQUE,
    type            TEXT NOT NULL DEFAULT 'direct',
    parent_org_id   UUID REFERENCES organizations(id),
    zitadel_org_id  TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_hub_org_type CHECK (type IN ('direct', 'msp', 'reseller')),
    CONSTRAINT chk_hub_org_name_not_empty CHECK (name <> ''),
    CONSTRAINT chk_hub_org_slug_not_empty CHECK (slug <> '')
);

CREATE UNIQUE INDEX uq_hub_organizations_zitadel_org_id
    ON organizations(zitadel_org_id)
    WHERE zitadel_org_id IS NOT NULL;

CREATE INDEX idx_hub_organizations_type ON organizations(type);

-- ============================================================
-- tenants.organization_id: nullable FK + backfill per existing tenant.
-- DEVIATION FROM ADR: NULLABLE (matches server migration 059). Tightened
-- in a follow-up once all hub call sites are updated.
-- ============================================================

ALTER TABLE tenants ADD COLUMN organization_id UUID REFERENCES organizations(id);

INSERT INTO organizations (id, name, slug, type)
SELECT gen_random_uuid(), t.name, t.slug, 'direct'
FROM tenants t;

UPDATE tenants t
SET organization_id = o.id
FROM organizations o
WHERE o.slug = t.slug;

CREATE INDEX idx_hub_tenants_organization ON tenants(organization_id);

-- ============================================================
-- licenses.organization_id: umbrella licensing.
-- When set, this license's max_endpoints is enforced as the sum of
-- endpoint_count across all clients belonging to tenants in that
-- organization. Enforcement lives in internal/hub/license.
-- ============================================================

ALTER TABLE licenses ADD COLUMN organization_id UUID REFERENCES organizations(id);

CREATE INDEX idx_licenses_organization ON licenses(organization_id) WHERE organization_id IS NOT NULL;

-- ============================================================
-- Grants for hub_app role
-- ============================================================

GRANT SELECT, INSERT, UPDATE, DELETE ON organizations TO hub_app;

-- +goose Down

ALTER TABLE licenses DROP COLUMN IF EXISTS organization_id;
ALTER TABLE tenants DROP COLUMN IF EXISTS organization_id;
DROP TABLE IF EXISTS organizations;
