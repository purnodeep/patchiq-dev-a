-- PatchIQ Agent store schema: patches, history, logs

CREATE TABLE IF NOT EXISTS pending_patches (
    id        TEXT PRIMARY KEY,
    name      TEXT NOT NULL,
    version   TEXT NOT NULL,
    severity  TEXT NOT NULL DEFAULT 'none',
    status    TEXT NOT NULL DEFAULT 'queued',
    queued_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_pending_patches_queued ON pending_patches(queued_at DESC);

CREATE TABLE IF NOT EXISTS patch_history (
    id            TEXT PRIMARY KEY,
    patch_name    TEXT NOT NULL,
    patch_version TEXT NOT NULL,
    action        TEXT NOT NULL,
    result        TEXT NOT NULL,
    error_message TEXT,
    completed_at  TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_patch_history_completed ON patch_history(completed_at DESC);

CREATE TABLE IF NOT EXISTS agent_logs (
    id        TEXT PRIMARY KEY,
    level     TEXT NOT NULL,
    message   TEXT NOT NULL,
    source    TEXT,
    timestamp TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_agent_logs_timestamp ON agent_logs(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_agent_logs_level ON agent_logs(level, timestamp DESC);

CREATE TABLE IF NOT EXISTS rollback_records (
    id TEXT PRIMARY KEY,
    command_id TEXT NOT NULL,
    package_name TEXT NOT NULL,
    from_version TEXT NOT NULL,
    to_version TEXT NOT NULL,
    rolled_back_at TEXT,
    status TEXT NOT NULL DEFAULT 'pending'
);

CREATE INDEX IF NOT EXISTS idx_rollback_records_command ON rollback_records(command_id);

CREATE TABLE IF NOT EXISTS inventory_cache (
    id            INTEGER PRIMARY KEY CHECK (id = 1),
    packages_json TEXT NOT NULL,
    collected_at  TEXT NOT NULL
);
