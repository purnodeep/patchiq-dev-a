-- +goose Up

-- ============================================================
-- Invitations table for invite-based user registration
-- ============================================================

CREATE TABLE invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    code UUID NOT NULL DEFAULT gen_random_uuid(),
    email TEXT NOT NULL,
    role_id UUID NOT NULL REFERENCES roles(id),
    invited_by TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    expires_at TIMESTAMPTZ NOT NULL DEFAULT now() + interval '7 days',
    claimed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_invitations_code ON invitations(code);
CREATE INDEX idx_invitations_tenant_status ON invitations(tenant_id, status);

-- ============================================================
-- RLS policies
-- ============================================================

ALTER TABLE invitations ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON invitations
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
ALTER TABLE invitations FORCE ROW LEVEL SECURITY;

-- ============================================================
-- Grants for patchiq_app role
-- ============================================================

GRANT SELECT, INSERT, UPDATE, DELETE ON invitations TO patchiq_app;

-- +goose Down

DROP POLICY IF EXISTS tenant_isolation ON invitations;
DROP TABLE IF EXISTS invitations;
