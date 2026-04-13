package store_test

import (
	"database/sql"
	"testing"

	"github.com/skenzeriq/patchiq/internal/agent/store"
	_ "modernc.org/sqlite"
)

func TestApplyMigrations(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Apply base schema first (creates the actual tables ApplyMigrations will ALTER)
	if err := store.ApplySchema(db); err != nil {
		t.Fatal(err)
	}

	// First apply: should add all new columns
	if err := store.ApplyMigrations(db); err != nil {
		t.Fatalf("first apply: %v", err)
	}

	// Second apply: must be idempotent (no error on duplicate column)
	if err := store.ApplyMigrations(db); err != nil {
		t.Fatalf("second apply (idempotent): %v", err)
	}

	// Verify columns exist by inserting with new fields
	_, err = db.Exec(`INSERT INTO pending_patches (id, name, version, severity, status, queued_at, size, cvss_score)
		VALUES ('test1', 'pkg', '1.0', 'high', 'queued', '2026-03-16T00:00:00Z', '10 MB', 9.8)`)
	if err != nil {
		t.Fatalf("insert with new patch columns: %v", err)
	}

	_, err = db.Exec(`INSERT INTO patch_history (id, patch_name, patch_version, action, result, completed_at, duration_seconds, attempt)
		VALUES ('hist1', 'pkg', '1.0', 'install', 'success', '2026-03-16T00:00:00Z', 87, 1)`)
	if err != nil {
		t.Fatalf("insert with new history columns: %v", err)
	}
}
