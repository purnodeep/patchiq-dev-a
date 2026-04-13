-- +goose Up

CREATE TABLE report_generations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    report_type     TEXT NOT NULL,
    format          TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending',
    name            TEXT NOT NULL DEFAULT '',
    filters         JSONB NOT NULL DEFAULT '{}',
    file_path       TEXT,
    file_size_bytes BIGINT,
    checksum_sha256 TEXT,
    row_count       INT,
    error_message   TEXT,
    created_by      UUID NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at    TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ NOT NULL,
    CONSTRAINT valid_report_type CHECK (report_type IN ('endpoints', 'patches', 'cves', 'deployments', 'compliance', 'executive')),
    CONSTRAINT valid_format CHECK (format IN ('pdf', 'csv', 'xlsx')),
    CONSTRAINT valid_status CHECK (status IN ('pending', 'generating', 'completed', 'failed'))
);

CREATE INDEX idx_report_generations_tenant ON report_generations(tenant_id);
CREATE INDEX idx_report_generations_expires ON report_generations(expires_at) WHERE status = 'completed';
CREATE INDEX idx_report_generations_tenant_created ON report_generations(tenant_id, created_at DESC);

ALTER TABLE report_generations ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON report_generations
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- +goose Down

DROP TABLE IF EXISTS report_generations;
