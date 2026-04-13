-- +goose Up

-- ============================================================
-- CHECK constraints for type safety and invariant enforcement
-- ============================================================

-- audit_events constraints
ALTER TABLE audit_events ADD CONSTRAINT chk_audit_actor_type
    CHECK (actor_type IN ('user', 'system', 'ai_assistant'));

ALTER TABLE audit_events ADD CONSTRAINT chk_audit_id_length
    CHECK (length(id) = 26);

-- patch_catalog constraints
ALTER TABLE patch_catalog ADD CONSTRAINT chk_catalog_severity
    CHECK (severity IN ('critical', 'high', 'medium', 'low', 'none'));

-- cve_feeds constraints
ALTER TABLE cve_feeds ADD CONSTRAINT chk_cve_severity
    CHECK (severity IN ('critical', 'high', 'medium', 'low', 'none'));

-- +goose Down

ALTER TABLE audit_events DROP CONSTRAINT IF EXISTS chk_audit_actor_type;
ALTER TABLE audit_events DROP CONSTRAINT IF EXISTS chk_audit_id_length;
ALTER TABLE patch_catalog DROP CONSTRAINT IF EXISTS chk_catalog_severity;
ALTER TABLE cve_feeds DROP CONSTRAINT IF EXISTS chk_cve_severity;
