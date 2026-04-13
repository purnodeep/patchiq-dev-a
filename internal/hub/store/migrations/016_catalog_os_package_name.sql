-- +goose Up
ALTER TABLE patch_catalog ADD COLUMN os_package_name TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE patch_catalog DROP COLUMN os_package_name;
