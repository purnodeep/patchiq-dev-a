package store

import (
	"database/sql"
	_ "embed"
	"fmt"
	"strings"
)

//go:embed schema.sql
var schemaSQL string

// addColumnIfNotExists runs ALTER TABLE ADD COLUMN, ignoring "duplicate column name" errors.
// SQLite does not support IF NOT EXISTS on ALTER TABLE ADD COLUMN in all versions,
// so we catch the error string instead.
func addColumnIfNotExists(db *sql.DB, table, column, colDef string) error {
	_, err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, colDef))
	if err != nil && strings.Contains(err.Error(), "duplicate column name") {
		return nil
	}
	return err
}

// ApplyMigrations adds new columns to existing agent tables without breaking existing DBs.
// Safe to call multiple times (idempotent). Must be called after ApplySchema in main.go.
func ApplyMigrations(db *sql.DB) error {
	patchCols := []struct{ name, def string }{
		{"size", "TEXT"},
		{"cvss_score", "REAL"},
		{"cve_ids", "TEXT"},
		{"published_at", "TEXT"},
		{"source", "TEXT"},
	}
	for _, c := range patchCols {
		if err := addColumnIfNotExists(db, "pending_patches", c.name, c.def); err != nil {
			return fmt.Errorf("migrate pending_patches.%s: %w", c.name, err)
		}
	}

	historyCols := []struct{ name, def string }{
		{"duration_seconds", "INTEGER"},
		{"size", "TEXT"},
		{"reboot_required", "INTEGER DEFAULT 0"},
		{"stdout", "TEXT"},
		{"stderr", "TEXT"},
		{"exit_code", "INTEGER"},
		{"attempt", "INTEGER DEFAULT 1"},
	}
	for _, c := range historyCols {
		if err := addColumnIfNotExists(db, "patch_history", c.name, c.def); err != nil {
			return fmt.Errorf("migrate patch_history.%s: %w", c.name, err)
		}
	}

	// inventory_cache table for persisting inventory across restarts.
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS inventory_cache (
		id            INTEGER PRIMARY KEY CHECK (id = 1),
		packages_json TEXT NOT NULL,
		collected_at  TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("create inventory_cache: %w", err)
	}

	return nil
}

// ApplySchema applies the store schema to an existing database connection.
// The agent shares a single SQLite file; comms.OpenDB creates and opens it,
// and this function adds the store tables on top.
func ApplySchema(db *sql.DB) error {
	if _, err := db.Exec(schemaSQL); err != nil {
		return fmt.Errorf("exec store schema: %w", err)
	}
	return nil
}
