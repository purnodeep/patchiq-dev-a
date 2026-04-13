-- +goose Up

-- ============================================================
-- Platform tenant link for MSP/reseller organizations.
--
-- Background: org-scoped role definitions (MSP Admin, MSP Technician,
-- MSP Auditor) must live in SOME tenant because roles.tenant_id is NOT NULL
-- and org_user_roles.role_id references roles.id. We host them in a hidden
-- "platform tenant" per MSP/reseller org. Direct-type orgs do not have a
-- platform tenant — their working tenant is the only tenant.
--
-- See ADR-025 decision #4.
-- ============================================================

ALTER TABLE organizations
    ADD COLUMN platform_tenant_id UUID REFERENCES tenants(id);

CREATE INDEX idx_organizations_platform_tenant
    ON organizations(platform_tenant_id);

-- +goose Down

-- Delete the hidden platform tenant rows BEFORE dropping the link column.
-- If we only dropped the column, the tenants would remain attached to their
-- orgs via tenants.organization_id and silently start showing up in
-- ListClientTenantsByOrganization (which will no longer be able to filter
-- them). Cascade through role_permissions and roles via the FK chain.
DELETE FROM tenants
 WHERE id IN (
     SELECT platform_tenant_id
       FROM organizations
      WHERE platform_tenant_id IS NOT NULL
 );

DROP INDEX IF EXISTS idx_organizations_platform_tenant;
ALTER TABLE organizations DROP COLUMN IF EXISTS platform_tenant_id;
