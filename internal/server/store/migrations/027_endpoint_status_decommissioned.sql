-- +goose Up
UPDATE endpoints SET status = 'offline' WHERE status = 'stale';
ALTER TABLE endpoints DROP CONSTRAINT IF EXISTS chk_endpoints_status;
ALTER TABLE endpoints ADD CONSTRAINT chk_endpoints_status
    CHECK (status IN ('pending', 'online', 'offline', 'decommissioned'));

-- +goose Down
ALTER TABLE endpoints DROP CONSTRAINT IF EXISTS chk_endpoints_status;
ALTER TABLE endpoints ADD CONSTRAINT chk_endpoints_status
    CHECK (status IN ('pending', 'online', 'offline', 'stale'));
