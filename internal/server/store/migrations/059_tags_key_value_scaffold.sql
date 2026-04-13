-- +goose Up

-- ============================================================
-- Phase 1 (additive) of the tags-replace-groups migration.
-- See docs/plans/tags-replace-groups.md.
--
-- This migration is strictly ADDITIVE:
--   - Adds `key` and `value` columns to `tags` (default '' so existing
--     rows remain valid; the old `name` column stays in place).
--   - Introduces `tag_keys` metadata table.
--   - Introduces `policy_tag_selectors` (JSONB AST per policy).
--   - Adds a compound index on `endpoint_tags` for the selector hot path.
--
-- It does NOT drop endpoint_groups, policy_groups, or endpoint_group_members,
-- and does NOT drop `tags.name`. Those destructive changes land in a
-- follow-up migration (060) after the Go layer has been migrated to the
-- new schema in Phase 2.
--
-- Intentionally leaves existing tag rows untouched: their `key`/`value`
-- default to empty strings and will be rebuilt once the Phase 2 handler
-- ships. Parallel devs see only new objects; no existing queries break.
-- ============================================================

-- ---------------------------------------------------------------------------
-- 1. Promote tags to key=value (additive — keep `name` for now).
-- ---------------------------------------------------------------------------
ALTER TABLE tags ADD COLUMN IF NOT EXISTS key   TEXT NOT NULL DEFAULT '';
ALTER TABLE tags ADD COLUMN IF NOT EXISTS value TEXT NOT NULL DEFAULT '';

-- Mirror the lowercase invariant from tag_keys.key onto tags.key so the
-- two tables cannot drift: without this a tenant could assign a tag with
-- key='ENV' while the catalog row is 'env', producing two distinct keys
-- that the compiler would match via lower() but the handler layer would
-- treat as separate. Empty-string defaults trivially satisfy the check.
ALTER TABLE tags DROP CONSTRAINT IF EXISTS chk_tags_key_lowercase;
ALTER TABLE tags ADD CONSTRAINT chk_tags_key_lowercase CHECK (key = lower(key));

-- Supporting index for key-first lookups. The unique index on
-- (tenant_id, lower(key), lower(value)) cannot be created while existing
-- rows all share empty key/value — it lands in migration 060.
CREATE INDEX IF NOT EXISTS tags_tenant_key_idx
    ON tags (tenant_id, lower(key));

-- ---------------------------------------------------------------------------
-- 2. tag_keys: metadata catalog for known keys (e.g. env, os, region).
-- The `exclusive` flag means an endpoint may carry at most one value for
-- this key. Enforced in application code (AssignTag handler), not the DB,
-- because a CHECK or trigger on a multi-row invariant is brittle.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS tag_keys (
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    key         TEXT NOT NULL,
    description TEXT,
    exclusive   BOOLEAN NOT NULL DEFAULT false,
    value_type  TEXT NOT NULL DEFAULT 'string',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, key),
    CONSTRAINT chk_tag_keys_value_type CHECK (value_type IN ('string', 'enum')),
    -- Keys are stored lowercased so that the PK (tenant_id, key) is the
    -- canonical identity. The compile layer uses lower(t.key) everywhere;
    -- without this CHECK a tenant could insert both "env" and "ENV" and
    -- the AssignTag handler would have to pick a winner at runtime.
    CONSTRAINT chk_tag_keys_key_lowercase CHECK (key = lower(key))
);

CREATE INDEX IF NOT EXISTS idx_tag_keys_tenant ON tag_keys(tenant_id);

ALTER TABLE tag_keys ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON tag_keys;
CREATE POLICY tenant_isolation ON tag_keys
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
ALTER TABLE tag_keys FORCE ROW LEVEL SECURITY;
GRANT SELECT, INSERT, UPDATE, DELETE ON tag_keys TO patchiq_app;

-- ---------------------------------------------------------------------------
-- 3. policy_tag_selectors: one JSONB AST per policy (or none = match all).
-- Will replace policy_groups as the sole policy targeting mechanism in
-- Phase 2. Until then, both tables coexist.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS policy_tag_selectors (
    policy_id  UUID PRIMARY KEY REFERENCES policies(id) ON DELETE CASCADE,
    tenant_id  UUID NOT NULL REFERENCES tenants(id),
    expression JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_policy_tag_selectors_tenant
    ON policy_tag_selectors(tenant_id);
CREATE INDEX IF NOT EXISTS idx_policy_tag_selectors_expr_gin
    ON policy_tag_selectors USING gin (expression);

ALTER TABLE policy_tag_selectors ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON policy_tag_selectors;
CREATE POLICY tenant_isolation ON policy_tag_selectors
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
ALTER TABLE policy_tag_selectors FORCE ROW LEVEL SECURITY;
GRANT SELECT, INSERT, UPDATE, DELETE ON policy_tag_selectors TO patchiq_app;

-- ---------------------------------------------------------------------------
-- 4. Compound index for the selector-resolution hot path.
-- endpoint_tags already has (tag_id, tenant_id) from migration 039, but the
-- compiler-generated EXISTS subqueries filter by (endpoint_id, tag_id) — add
-- a covering index so the planner does not fall back to a seq scan.
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_endpoint_tags_endpoint_tag
    ON endpoint_tags(endpoint_id, tag_id);

-- +goose Down

DROP INDEX IF EXISTS idx_endpoint_tags_endpoint_tag;

DROP POLICY IF EXISTS tenant_isolation ON policy_tag_selectors;
DROP INDEX IF EXISTS idx_policy_tag_selectors_expr_gin;
DROP INDEX IF EXISTS idx_policy_tag_selectors_tenant;
DROP TABLE IF EXISTS policy_tag_selectors;

DROP POLICY IF EXISTS tenant_isolation ON tag_keys;
DROP INDEX IF EXISTS idx_tag_keys_tenant;
DROP TABLE IF EXISTS tag_keys;

DROP INDEX IF EXISTS tags_tenant_key_idx;
ALTER TABLE tags DROP CONSTRAINT IF EXISTS chk_tags_key_lowercase;
ALTER TABLE tags DROP COLUMN IF EXISTS value;
ALTER TABLE tags DROP COLUMN IF EXISTS key;
