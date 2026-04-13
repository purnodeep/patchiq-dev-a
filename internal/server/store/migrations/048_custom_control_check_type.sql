-- +goose Up
ALTER TABLE custom_compliance_controls ADD COLUMN check_type TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE custom_compliance_controls DROP COLUMN check_type;
