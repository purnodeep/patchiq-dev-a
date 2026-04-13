-- +goose Up

-- ============================================================
-- Application role with restricted audit_events permissions
-- ============================================================

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'hub_app') THEN
        CREATE ROLE hub_app LOGIN;
    END IF;
END
$$;
-- +goose StatementEnd

-- Grant CRUD on all tables that exist at migration time.
-- Tables created after this migration require DEFAULT PRIVILEGES (see below).
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO hub_app;

-- Restrict audit_events: INSERT + SELECT only (append-only)
REVOKE UPDATE, DELETE ON audit_events FROM hub_app;
REVOKE UPDATE, DELETE ON audit_events_2026_01 FROM hub_app;
REVOKE UPDATE, DELETE ON audit_events_2026_02 FROM hub_app;
REVOKE UPDATE, DELETE ON audit_events_2026_03 FROM hub_app;
REVOKE UPDATE, DELETE ON audit_events_2026_04 FROM hub_app;
REVOKE UPDATE, DELETE ON audit_events_2026_05 FROM hub_app;
REVOKE UPDATE, DELETE ON audit_events_2026_06 FROM hub_app;
REVOKE UPDATE, DELETE ON audit_events_2026_07 FROM hub_app;
REVOKE UPDATE, DELETE ON audit_events_2026_08 FROM hub_app;
REVOKE UPDATE, DELETE ON audit_events_2026_09 FROM hub_app;
REVOKE UPDATE, DELETE ON audit_events_2026_10 FROM hub_app;
REVOKE UPDATE, DELETE ON audit_events_2026_11 FROM hub_app;
REVOKE UPDATE, DELETE ON audit_events_2026_12 FROM hub_app;
REVOKE UPDATE, DELETE ON audit_events_default FROM hub_app;
-- NOTE: Future partitions created by later migrations must also REVOKE UPDATE, DELETE for hub_app.

-- Grant on future tables too
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO hub_app;

GRANT USAGE ON ALL SEQUENCES IN SCHEMA public TO hub_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT USAGE ON SEQUENCES TO hub_app;

-- ============================================================
-- Row-Level Security on tenant-scoped tables
-- ============================================================

ALTER TABLE hub_config ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON hub_config
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);

ALTER TABLE audit_events ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON audit_events
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);

-- ============================================================
-- Force RLS even for the table owner (migration superuser).
-- Without FORCE, the owner bypasses ENABLE ROW LEVEL SECURITY entirely.
-- hub_app is a non-owner role already covered by ENABLE alone,
-- but FORCE protects against accidental owner-role access in production.
-- ============================================================

ALTER TABLE hub_config FORCE ROW LEVEL SECURITY;
ALTER TABLE audit_events FORCE ROW LEVEL SECURITY;

-- +goose Down

-- +goose StatementBegin
DO $$
DECLARE
    tbl TEXT;
BEGIN
    FOR tbl IN
        SELECT unnest(ARRAY['hub_config', 'audit_events'])
    LOOP
        EXECUTE format('ALTER TABLE %I NO FORCE ROW LEVEL SECURITY', tbl);
        EXECUTE format('ALTER TABLE %I DISABLE ROW LEVEL SECURITY', tbl);
        EXECUTE format('DROP POLICY IF EXISTS tenant_isolation ON %I', tbl);
    END LOOP;
END
$$;
-- +goose StatementEnd

REVOKE ALL ON ALL TABLES IN SCHEMA public FROM hub_app;
DROP ROLE IF EXISTS hub_app;
