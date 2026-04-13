-- PatchIQ Agent local database schema

CREATE TABLE IF NOT EXISTS outbox (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    message_type TEXT NOT NULL,
    payload      BLOB NOT NULL,
    created_at   TEXT NOT NULL,
    attempts     INTEGER DEFAULT 0,
    last_error   TEXT,
    status       TEXT DEFAULT 'pending'
);

CREATE INDEX IF NOT EXISTS idx_outbox_status ON outbox(status, created_at);

CREATE TABLE IF NOT EXISTS inbox (
    id           TEXT PRIMARY KEY,
    command_type TEXT NOT NULL,
    payload      BLOB,
    priority     INTEGER DEFAULT 0,
    received_at  TEXT NOT NULL,
    execute_at   TEXT,
    status       TEXT DEFAULT 'pending',
    result       BLOB,
    completed_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_inbox_status ON inbox(status, priority DESC, execute_at);

CREATE TABLE IF NOT EXISTS local_inventory (
    package_name TEXT NOT NULL,
    version      TEXT NOT NULL,
    os_family    TEXT NOT NULL,
    source       TEXT,
    installed_at TEXT,
    scanned_at   TEXT NOT NULL,
    PRIMARY KEY (package_name, os_family)
);

CREATE TABLE IF NOT EXISTS agent_state (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
