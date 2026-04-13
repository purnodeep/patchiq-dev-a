-- +goose Up

-- ============================================================
-- Application role with restricted audit_events permissions
-- ============================================================

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'patchiq_app') THEN
        CREATE ROLE patchiq_app LOGIN;
    END IF;
END
$$;
-- +goose StatementEnd

-- Grant CRUD on all tables that exist at migration time.
-- Tables created after this migration require DEFAULT PRIVILEGES (see below).
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO patchiq_app;

-- Restrict audit_events: INSERT + SELECT only (append-only)
REVOKE UPDATE, DELETE ON audit_events FROM patchiq_app;
REVOKE UPDATE, DELETE ON audit_events_2026_01 FROM patchiq_app;
REVOKE UPDATE, DELETE ON audit_events_2026_02 FROM patchiq_app;
REVOKE UPDATE, DELETE ON audit_events_2026_03 FROM patchiq_app;
REVOKE UPDATE, DELETE ON audit_events_2026_04 FROM patchiq_app;
REVOKE UPDATE, DELETE ON audit_events_2026_05 FROM patchiq_app;
REVOKE UPDATE, DELETE ON audit_events_2026_06 FROM patchiq_app;
REVOKE UPDATE, DELETE ON audit_events_2026_07 FROM patchiq_app;
REVOKE UPDATE, DELETE ON audit_events_2026_08 FROM patchiq_app;
REVOKE UPDATE, DELETE ON audit_events_2026_09 FROM patchiq_app;
REVOKE UPDATE, DELETE ON audit_events_2026_10 FROM patchiq_app;
REVOKE UPDATE, DELETE ON audit_events_2026_11 FROM patchiq_app;
REVOKE UPDATE, DELETE ON audit_events_2026_12 FROM patchiq_app;
-- NOTE: Future partitions created by later migrations must also REVOKE UPDATE, DELETE for patchiq_app.

-- Grant on future tables too
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO patchiq_app;

-- Grant usage on sequences
GRANT USAGE ON ALL SEQUENCES IN SCHEMA public TO patchiq_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT USAGE ON SEQUENCES TO patchiq_app;

-- ============================================================
-- Row-Level Security on all tenant-scoped tables
-- ============================================================

ALTER TABLE endpoints ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON endpoints
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);

ALTER TABLE endpoint_groups ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON endpoint_groups
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);

ALTER TABLE endpoint_group_members ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON endpoint_group_members
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);

ALTER TABLE patches ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON patches
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);

ALTER TABLE cves ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON cves
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);

ALTER TABLE patch_cves ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON patch_cves
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);

ALTER TABLE policies ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON policies
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);

ALTER TABLE policy_groups ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON policy_groups
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);

ALTER TABLE deployments ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON deployments
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);

ALTER TABLE deployment_targets ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON deployment_targets
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);

ALTER TABLE deployment_waves ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON deployment_waves
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);

ALTER TABLE agent_registrations ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON agent_registrations
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);

ALTER TABLE config_overrides ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON config_overrides
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);

ALTER TABLE audit_events ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON audit_events
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);

-- ============================================================
-- Force RLS even for the table owner (migration superuser).
-- Without FORCE, the owner bypasses ENABLE ROW LEVEL SECURITY entirely.
-- patchiq_app is a non-owner role already covered by ENABLE alone,
-- but FORCE protects against accidental owner-role access in production.
-- ============================================================

ALTER TABLE endpoints FORCE ROW LEVEL SECURITY;
ALTER TABLE endpoint_groups FORCE ROW LEVEL SECURITY;
ALTER TABLE endpoint_group_members FORCE ROW LEVEL SECURITY;
ALTER TABLE patches FORCE ROW LEVEL SECURITY;
ALTER TABLE cves FORCE ROW LEVEL SECURITY;
ALTER TABLE patch_cves FORCE ROW LEVEL SECURITY;
ALTER TABLE policies FORCE ROW LEVEL SECURITY;
ALTER TABLE policy_groups FORCE ROW LEVEL SECURITY;
ALTER TABLE deployments FORCE ROW LEVEL SECURITY;
ALTER TABLE deployment_targets FORCE ROW LEVEL SECURITY;
ALTER TABLE deployment_waves FORCE ROW LEVEL SECURITY;
ALTER TABLE agent_registrations FORCE ROW LEVEL SECURITY;
ALTER TABLE config_overrides FORCE ROW LEVEL SECURITY;
ALTER TABLE audit_events FORCE ROW LEVEL SECURITY;

-- +goose Down

-- Drop all RLS policies
-- +goose StatementBegin
DO $$
DECLARE
    tbl TEXT;
BEGIN
    FOR tbl IN
        SELECT unnest(ARRAY[
            'endpoints', 'endpoint_groups', 'endpoint_group_members',
            'patches', 'cves', 'patch_cves',
            'policies', 'policy_groups',
            'deployments', 'deployment_targets', 'deployment_waves',
            'agent_registrations', 'config_overrides', 'audit_events'
        ])
    LOOP
        EXECUTE format('ALTER TABLE %I NO FORCE ROW LEVEL SECURITY', tbl);
        EXECUTE format('ALTER TABLE %I DISABLE ROW LEVEL SECURITY', tbl);
        EXECUTE format('DROP POLICY IF EXISTS tenant_isolation ON %I', tbl);
    END LOOP;
END
$$;
-- +goose StatementEnd

-- Revoke and drop app role
REVOKE ALL ON ALL TABLES IN SCHEMA public FROM patchiq_app;
DROP ROLE IF EXISTS patchiq_app;
