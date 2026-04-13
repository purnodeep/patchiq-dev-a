-- +goose Up
-- ============================================================
-- Deep hardware details and software summary as JSONB columns
-- ============================================================
ALTER TABLE endpoints ADD COLUMN IF NOT EXISTS hardware_details JSONB DEFAULT '{}';
ALTER TABLE endpoints ADD COLUMN IF NOT EXISTS software_summary JSONB DEFAULT '{}';

-- Deduplicate existing network interfaces before adding unique constraint.
DELETE FROM endpoint_network_interfaces a
    USING endpoint_network_interfaces b
    WHERE a.id > b.id
      AND a.tenant_id = b.tenant_id
      AND a.endpoint_id = b.endpoint_id
      AND a.name = b.name;

-- Unique constraint for network interface upserts (tenant + endpoint + iface name).
CREATE UNIQUE INDEX IF NOT EXISTS uq_eni_tenant_endpoint_name
    ON endpoint_network_interfaces(tenant_id, endpoint_id, name);

-- +goose Down
DROP INDEX IF EXISTS uq_eni_tenant_endpoint_name;
ALTER TABLE endpoints DROP COLUMN IF EXISTS hardware_details;
ALTER TABLE endpoints DROP COLUMN IF EXISTS software_summary;
