-- +goose Up
CREATE UNIQUE INDEX IF NOT EXISTS idx_patches_tenant_name_version_os
    ON patches(tenant_id, name, version, os_family);

-- +goose Down
DROP INDEX IF EXISTS idx_patches_tenant_name_version_os;
