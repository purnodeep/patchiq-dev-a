-- +goose Up
ALTER TABLE patches ADD COLUMN IF NOT EXISTS installer_type TEXT NOT NULL DEFAULT '';
ALTER TABLE patches ADD COLUMN IF NOT EXISTS silent_args TEXT NOT NULL DEFAULT '';

-- Backfill existing Windows patches with heuristic installer_type
UPDATE patches SET installer_type = CASE
    WHEN os_family != 'windows' THEN installer_type
    WHEN name ~* '(^KB|Cumulative Update|Security Update|Servicing Stack|\.NET Framework)' THEN 'wua'
    WHEN package_url IS NOT NULL AND package_url LIKE '%.msi' THEN 'msi'
    WHEN package_url IS NOT NULL AND package_url LIKE '%.msix' THEN 'msix'
    WHEN package_url IS NOT NULL AND package_url LIKE '%.appx' THEN 'msix'
    WHEN package_url IS NOT NULL AND package_url LIKE '%.exe' THEN 'exe'
    ELSE 'wua'
END
WHERE os_family = 'windows' AND installer_type = '';

-- +goose Down
ALTER TABLE patches DROP COLUMN IF EXISTS installer_type;
ALTER TABLE patches DROP COLUMN IF EXISTS silent_args;
