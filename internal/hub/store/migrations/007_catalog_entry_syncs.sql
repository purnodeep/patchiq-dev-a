-- +goose Up
CREATE TABLE catalog_entry_syncs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    catalog_id  UUID NOT NULL REFERENCES patch_catalog(id) ON DELETE CASCADE,
    client_id   UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    status      TEXT NOT NULL DEFAULT 'pending',
    synced_at   TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(catalog_id, client_id)
);
CREATE INDEX idx_catalog_entry_syncs_catalog ON catalog_entry_syncs(catalog_id);
CREATE INDEX idx_catalog_entry_syncs_client ON catalog_entry_syncs(client_id);

-- +goose Down
DROP TABLE IF EXISTS catalog_entry_syncs;
