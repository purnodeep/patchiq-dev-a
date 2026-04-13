-- +goose Up

-- ============================================================
-- Hub sync state: tracks PM-to-Hub catalog synchronization
-- Issue #180
-- ============================================================

CREATE TABLE hub_sync_state (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL REFERENCES tenants(id),
    hub_url          TEXT NOT NULL,
    api_key          TEXT NOT NULL,
    last_sync_at     TIMESTAMPTZ,
    next_sync_at     TIMESTAMPTZ,
    sync_interval    INT NOT NULL DEFAULT 21600,
    entries_received INT NOT NULL DEFAULT 0,
    last_entry_count INT NOT NULL DEFAULT 0,
    last_error       TEXT,
    status           TEXT NOT NULL DEFAULT 'idle',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE hub_sync_state ADD CONSTRAINT chk_hub_sync_status
    CHECK (status IN ('idle', 'syncing', 'error'));

CREATE UNIQUE INDEX idx_hub_sync_state_tenant ON hub_sync_state (tenant_id);

-- RLS
ALTER TABLE hub_sync_state ENABLE ROW LEVEL SECURITY;
ALTER TABLE hub_sync_state FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_hub_sync_state ON hub_sync_state
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);

GRANT SELECT, INSERT, UPDATE, DELETE ON hub_sync_state TO patchiq_app;

-- +goose Down
DROP TABLE IF EXISTS hub_sync_state;
