-- +goose Up

-- Track each client catalog sync call for history + endpoint trends
CREATE TABLE client_sync_history (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES tenants(id),
    client_id         UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    started_at        TIMESTAMPTZ NOT NULL,
    finished_at       TIMESTAMPTZ,
    duration_ms       INT,
    entries_delivered  INT NOT NULL DEFAULT 0,
    deletes_delivered  INT NOT NULL DEFAULT 0,
    endpoint_count    INT NOT NULL DEFAULT 0,
    status            TEXT NOT NULL DEFAULT 'success',
    error_message     TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_sync_status CHECK (status IN ('success', 'failed'))
);

CREATE INDEX idx_client_sync_history_lookup
    ON client_sync_history (tenant_id, client_id, started_at DESC);

ALTER TABLE client_sync_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE client_sync_history FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON client_sync_history
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);

GRANT SELECT, INSERT, UPDATE, DELETE ON client_sync_history TO hub_app;

-- Add summary columns to clients for PM-reported data
ALTER TABLE clients
    ADD COLUMN os_summary              JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN endpoint_status_summary JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN compliance_summary      JSONB NOT NULL DEFAULT '{}';

-- +goose Down

ALTER TABLE clients
    DROP COLUMN IF EXISTS compliance_summary,
    DROP COLUMN IF EXISTS endpoint_status_summary,
    DROP COLUMN IF EXISTS os_summary;

DROP TABLE IF EXISTS client_sync_history;
