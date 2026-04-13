-- +goose Up

-- Store the actual installable package name (e.g. "curl", "openssl") separately
-- from the advisory/patch name (e.g. "RHSA-2024:0893", "USN-8107-1").
-- When deploying, the wave dispatcher uses package_name for the install payload
-- so agents can run the correct package manager command.
ALTER TABLE patches ADD COLUMN package_name TEXT NOT NULL DEFAULT '';

-- +goose Down

ALTER TABLE patches DROP COLUMN IF EXISTS package_name;
