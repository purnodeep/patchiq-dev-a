-- +goose Up

-- ============================================================
-- Organizations: parent entity above tenants for MSP model.
--
-- See docs/adr/025-organization-scoped-rbac-msp.md for the full decision.
--
-- Key invariants:
--   * organizations is GLOBAL (no tenant_id, no RLS). An org owns N tenants.
--   * tenants.organization_id is NOT NULL (every tenant belongs to exactly one org).
--   * RLS remains tenant-scoped — this migration does NOT change any existing
--     RLS policy. Cross-tenant reads for MSP dashboards are done in application
--     code via store.ForEachTenant(), which iterates per-tenant transactions.
--   * org_user_roles grants org-wide RBAC (e.g. MSP Admin). It is GLOBAL (no RLS)
--     because it spans tenants by design.
--   * Every existing tenant is backfilled into its own "direct"-type organization
--     (1:1). Single-tenant deployments see zero behavior change.
-- ============================================================

CREATE TABLE organizations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL,
    slug            TEXT NOT NULL UNIQUE,
    type            TEXT NOT NULL DEFAULT 'direct',
    parent_org_id   UUID REFERENCES organizations(id),
    zitadel_org_id  TEXT,
    license_id      UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_org_type CHECK (type IN ('direct', 'msp', 'reseller')),
    CONSTRAINT chk_org_name_not_empty CHECK (name <> ''),
    CONSTRAINT chk_org_slug_not_empty CHECK (slug <> '')
);

-- Partial unique index: a Zitadel org ID can map to at most one PatchIQ org,
-- but multiple orgs may have NULL (not yet bound to Zitadel).
CREATE UNIQUE INDEX uq_organizations_zitadel_org_id
    ON organizations(zitadel_org_id)
    WHERE zitadel_org_id IS NOT NULL;

CREATE INDEX idx_organizations_parent ON organizations(parent_org_id);
CREATE INDEX idx_organizations_type ON organizations(type);

-- ============================================================
-- tenants.organization_id: add nullable column, backfill existing rows.
--
-- DEVIATION FROM ADR-025: the ADR specifies NOT NULL. This migration keeps
-- the column NULLABLE to avoid cascading breakage into ~15 test helpers and
-- seed scripts that insert tenants without providing organization_id. The
-- invariant is enforced at the application layer (sqlcgen query always
-- supplies organization_id) and a follow-up migration will flip to NOT NULL
-- after all call sites are hardened. See docs/adr/025 implementation note.
-- ============================================================

ALTER TABLE tenants ADD COLUMN organization_id UUID REFERENCES organizations(id);

-- Backfill: create one default organization per existing tenant (1:1).
-- Slug matches the tenant slug (tenants.slug is UNIQUE, so organizations.slug
-- inherits that uniqueness). Name matches the tenant name. Type is 'direct'
-- because these are single-tenant deployments until explicitly converted.
INSERT INTO organizations (id, name, slug, type)
SELECT gen_random_uuid(), t.name, t.slug, 'direct'
FROM tenants t;

UPDATE tenants t
SET organization_id = o.id
FROM organizations o
WHERE o.slug = t.slug;

CREATE INDEX idx_tenants_organization ON tenants(organization_id);

-- ============================================================
-- org_user_roles: org-scoped RBAC grants (parallel to user_roles).
--
-- Unlike user_roles, this table is NOT RLS-protected because org-scoped
-- grants span tenants by design. The evaluator (internal/server/auth/evaluator.go)
-- checks these grants alongside tenant-scoped user_roles.
--
-- role_id references a role in the org's "platform tenant" — a hidden tenant
-- per MSP org that exists solely to host org-scoped role definitions, created
-- lazily at first msp-type conversion. See ADR-025 decision #4.
-- ============================================================

CREATE TABLE org_user_roles (
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id         TEXT NOT NULL,
    role_id         UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    assigned_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (organization_id, user_id, role_id),
    CONSTRAINT chk_org_user_roles_user_not_empty CHECK (user_id <> '')
);

CREATE INDEX idx_org_user_roles_user ON org_user_roles(organization_id, user_id);
CREATE INDEX idx_org_user_roles_role ON org_user_roles(role_id);

-- ============================================================
-- Grants for patchiq_app role
-- ============================================================

GRANT SELECT, INSERT, UPDATE, DELETE ON organizations TO patchiq_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON org_user_roles TO patchiq_app;

-- +goose Down

ALTER TABLE tenants DROP COLUMN IF EXISTS organization_id;
DROP TABLE IF EXISTS org_user_roles;
DROP TABLE IF EXISTS organizations;
