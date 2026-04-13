-- +goose Up

ALTER TABLE policies
    ADD COLUMN selection_mode      TEXT NOT NULL DEFAULT 'all_available',
    ADD COLUMN min_severity        TEXT,
    ADD COLUMN cve_ids             TEXT[],
    ADD COLUMN package_regex       TEXT,
    ADD COLUMN exclude_packages    TEXT[],
    ADD COLUMN schedule_type       TEXT NOT NULL DEFAULT 'manual',
    ADD COLUMN schedule_cron       TEXT,
    ADD COLUMN mw_start            TIME,
    ADD COLUMN mw_end              TIME,
    ADD COLUMN deployment_strategy TEXT NOT NULL DEFAULT 'all_at_once',
    ADD COLUMN deleted_at          TIMESTAMPTZ;

ALTER TABLE policies
    DROP COLUMN schedule,
    DROP COLUMN maintenance_window;

ALTER TABLE policies ADD CONSTRAINT chk_selection_mode
    CHECK (selection_mode IN ('all_available', 'by_severity', 'by_cve_list', 'by_regex'));

ALTER TABLE policies ADD CONSTRAINT chk_schedule_type
    CHECK (schedule_type IN ('manual', 'recurring'));

ALTER TABLE policies ADD CONSTRAINT chk_deployment_strategy
    CHECK (deployment_strategy IN ('all_at_once', 'rolling'));

-- +goose Down

ALTER TABLE policies DROP CONSTRAINT IF EXISTS chk_deployment_strategy;
ALTER TABLE policies DROP CONSTRAINT IF EXISTS chk_schedule_type;
ALTER TABLE policies DROP CONSTRAINT IF EXISTS chk_selection_mode;

ALTER TABLE policies
    ADD COLUMN schedule TEXT,
    ADD COLUMN maintenance_window TEXT;

ALTER TABLE policies
    DROP COLUMN IF EXISTS selection_mode,
    DROP COLUMN IF EXISTS min_severity,
    DROP COLUMN IF EXISTS cve_ids,
    DROP COLUMN IF EXISTS package_regex,
    DROP COLUMN IF EXISTS exclude_packages,
    DROP COLUMN IF EXISTS schedule_type,
    DROP COLUMN IF EXISTS schedule_cron,
    DROP COLUMN IF EXISTS mw_start,
    DROP COLUMN IF EXISTS mw_end,
    DROP COLUMN IF EXISTS deployment_strategy,
    DROP COLUMN IF EXISTS deleted_at;
