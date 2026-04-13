-- +goose Up

-- ============================================================
-- RLS policies for M1 core loop tables
-- ============================================================

ALTER TABLE endpoint_inventories ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON endpoint_inventories
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
ALTER TABLE endpoint_inventories FORCE ROW LEVEL SECURITY;

ALTER TABLE endpoint_packages ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON endpoint_packages
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
ALTER TABLE endpoint_packages FORCE ROW LEVEL SECURITY;

ALTER TABLE endpoint_cves ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON endpoint_cves
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
ALTER TABLE endpoint_cves FORCE ROW LEVEL SECURITY;

-- Restrict audit on new M1 partitions (append-only enforcement for patchiq_app
-- is handled by DEFAULT PRIVILEGES from 002, but explicit REVOKE on any new
-- audit partitions created after 002 would go here if needed).

-- +goose Down

-- +goose StatementBegin
DO $$
DECLARE
    tbl TEXT;
BEGIN
    FOR tbl IN
        SELECT unnest(ARRAY[
            'endpoint_inventories', 'endpoint_packages', 'endpoint_cves'
        ])
    LOOP
        EXECUTE format('ALTER TABLE %I NO FORCE ROW LEVEL SECURITY', tbl);
        EXECUTE format('ALTER TABLE %I DISABLE ROW LEVEL SECURITY', tbl);
        EXECUTE format('DROP POLICY IF EXISTS tenant_isolation ON %I', tbl);
    END LOOP;
END
$$;
-- +goose StatementEnd
