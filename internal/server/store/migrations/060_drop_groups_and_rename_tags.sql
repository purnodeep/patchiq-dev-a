-- +goose Up

-- ============================================================
-- Phase 2 of tags-replace-groups: destructive wave.
-- See docs/plans/tags-replace-groups.md.
--
-- After migration 059 scaffolded the new targeting primitives, this
-- migration removes the legacy groups feature entirely and finalises the
-- tags schema to be key=value-only.
--
-- Data policy: existing tag rows are wiped. Users re-tag endpoints using
-- the new key=value UI (decision made 2026-04-11, pre-client-deploy).
-- Groups have no migration path — those tables are dropped outright.
-- ============================================================

-- ---------------------------------------------------------------------------
-- 1. Wipe stale tag data before we change the shape of `tags` and before
--    we drop the `name` column that migration 047 briefly re-introduced.
-- ---------------------------------------------------------------------------
DELETE FROM endpoint_tags;
DELETE FROM tag_rules;
DELETE FROM tags;

-- ---------------------------------------------------------------------------
-- 2. Finalise the `tags` schema.
--
-- Migration 059 added `key`/`value` as NOT NULL DEFAULT ''; now that every
-- row is gone we can drop the defaults, drop the legacy `name` column, add
-- the unique index, and tighten the same `chk_tags_key_lowercase` symmetry
-- the tag_keys catalog already enforces.
-- ---------------------------------------------------------------------------
DROP INDEX IF EXISTS idx_tags_tenant_name;
ALTER TABLE tags DROP COLUMN IF EXISTS name;

ALTER TABLE tags ALTER COLUMN key DROP DEFAULT;
ALTER TABLE tags ALTER COLUMN value DROP DEFAULT;

-- Note: chk_tags_key_lowercase was already added in migration 059 as part
-- of the Phase 1 hardening pass. It is intentionally not re-added here.

-- Unique per (tenant, key, value). Both key and value are lowered in the
-- index expression so "Prod" and "prod" collide under the same `env` key.
-- Matches the semantics of the chk_tag_keys_key_lowercase CHECK.
CREATE UNIQUE INDEX tags_tenant_key_value_uq
    ON tags (tenant_id, lower(key), lower(value));

-- ---------------------------------------------------------------------------
-- 3. Tighten CHECK constraints: remove 'group' from scope_type enums.
-- Delete any stale rows first so the constraint re-add does not fail.
-- ---------------------------------------------------------------------------
DELETE FROM config_overrides WHERE scope_type = 'group';
ALTER TABLE config_overrides DROP CONSTRAINT IF EXISTS chk_config_scope_type;
ALTER TABLE config_overrides ADD CONSTRAINT chk_config_scope_type
    CHECK (scope_type IN ('tenant', 'tag', 'endpoint'));

DELETE FROM compliance_scores WHERE scope_type = 'group';
ALTER TABLE compliance_scores DROP CONSTRAINT IF EXISTS chk_cs_scope_type;
ALTER TABLE compliance_scores ADD CONSTRAINT chk_cs_scope_type
    CHECK (scope_type IN ('endpoint', 'tag', 'tenant'));

-- ---------------------------------------------------------------------------
-- 4. Drop groups entirely. CASCADE clears RLS policies and FKs.
-- ---------------------------------------------------------------------------
DROP TABLE IF EXISTS policy_groups           CASCADE;
DROP TABLE IF EXISTS endpoint_group_members  CASCADE;
DROP TABLE IF EXISTS endpoint_groups         CASCADE;

-- +goose Down

-- Best-effort down: restore empty group tables, revert tag shape.
-- Data is not recoverable — this is a one-way migration in practice.

DROP INDEX IF EXISTS tags_tenant_key_value_uq;
ALTER TABLE tags DROP CONSTRAINT IF EXISTS chk_tags_key_lowercase;

-- Restore the name column as empty string so existing rows (if any) remain
-- loadable by the migration 047 handler shape.
ALTER TABLE tags ADD COLUMN IF NOT EXISTS name TEXT NOT NULL DEFAULT '';
CREATE UNIQUE INDEX IF NOT EXISTS idx_tags_tenant_name ON tags (tenant_id, lower(name));

-- Restore scope_type 'group' on the CHECK constraints.
ALTER TABLE config_overrides DROP CONSTRAINT IF EXISTS chk_config_scope_type;
ALTER TABLE config_overrides ADD CONSTRAINT chk_config_scope_type
    CHECK (scope_type IN ('tenant', 'group', 'endpoint'));

ALTER TABLE compliance_scores DROP CONSTRAINT IF EXISTS chk_cs_scope_type;
ALTER TABLE compliance_scores ADD CONSTRAINT chk_cs_scope_type
    CHECK (scope_type IN ('endpoint', 'group', 'tenant'));

-- Recreate empty group tables.
CREATE TABLE IF NOT EXISTS endpoint_groups (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    name        TEXT NOT NULL,
    description TEXT,
    deleted_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_endpoint_groups_tenant ON endpoint_groups(tenant_id);
ALTER TABLE endpoint_groups ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON endpoint_groups;
CREATE POLICY tenant_isolation ON endpoint_groups
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
ALTER TABLE endpoint_groups FORCE ROW LEVEL SECURITY;
GRANT SELECT, INSERT, UPDATE, DELETE ON endpoint_groups TO patchiq_app;

CREATE TABLE IF NOT EXISTS endpoint_group_members (
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    endpoint_id UUID NOT NULL REFERENCES endpoints(id),
    group_id    UUID NOT NULL REFERENCES endpoint_groups(id),
    added_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, endpoint_id, group_id)
);
ALTER TABLE endpoint_group_members ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON endpoint_group_members;
CREATE POLICY tenant_isolation ON endpoint_group_members
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
ALTER TABLE endpoint_group_members FORCE ROW LEVEL SECURITY;
GRANT SELECT, INSERT, UPDATE, DELETE ON endpoint_group_members TO patchiq_app;

CREATE TABLE IF NOT EXISTS policy_groups (
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    policy_id UUID NOT NULL REFERENCES policies(id),
    group_id  UUID NOT NULL REFERENCES endpoint_groups(id),
    PRIMARY KEY (tenant_id, policy_id, group_id)
);
ALTER TABLE policy_groups ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON policy_groups;
CREATE POLICY tenant_isolation ON policy_groups
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
ALTER TABLE policy_groups FORCE ROW LEVEL SECURITY;
GRANT SELECT, INSERT, UPDATE, DELETE ON policy_groups TO patchiq_app;
