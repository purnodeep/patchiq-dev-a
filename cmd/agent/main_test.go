package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/skenzeriq/patchiq/internal/agent/comms"
	"github.com/skenzeriq/patchiq/internal/agent/store"
)

func TestIsEnrolled(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(t *testing.T) (dataDir string)
		dbFile string
		want   bool
	}{
		{
			name: "dataDir does not exist",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent")
			},
			dbFile: "agent.db",
			want:   false,
		},
		{
			name: "DB file does not exist",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			dbFile: "agent.db",
			want:   false,
		},
		{
			name: "DB exists but no agent_state table",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				dbPath := filepath.Join(dir, "agent.db")
				db, err := sql.Open("sqlite", dbPath)
				if err != nil {
					t.Fatalf("open db: %v", err)
				}
				// Create a different table, not agent_state.
				_, err = db.Exec("CREATE TABLE other (key TEXT PRIMARY KEY)")
				if err != nil {
					t.Fatalf("create table: %v", err)
				}
				db.Close()
				return dir
			},
			dbFile: "agent.db",
			want:   false,
		},
		{
			name: "agent_state exists but row absent",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				dbPath := filepath.Join(dir, "agent.db")
				db, err := comms.OpenDB(dbPath)
				if err != nil {
					t.Fatalf("open db: %v", err)
				}
				db.Close()
				return dir
			},
			dbFile: "agent.db",
			want:   false,
		},
		{
			name: "agent_id row present but empty value",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				dbPath := filepath.Join(dir, "agent.db")
				db, err := comms.OpenDB(dbPath)
				if err != nil {
					t.Fatalf("open db: %v", err)
				}
				_, err = db.Exec("INSERT INTO agent_state (key, value) VALUES ('agent_id', '')")
				if err != nil {
					t.Fatalf("insert: %v", err)
				}
				db.Close()
				return dir
			},
			dbFile: "agent.db",
			want:   false,
		},
		{
			name: "agent_id row present and non-empty",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				dbPath := filepath.Join(dir, "agent.db")
				db, err := comms.OpenDB(dbPath)
				if err != nil {
					t.Fatalf("open db: %v", err)
				}
				_, err = db.Exec("INSERT INTO agent_state (key, value) VALUES ('agent_id', 'agent-001')")
				if err != nil {
					t.Fatalf("insert: %v", err)
				}
				db.Close()
				return dir
			},
			dbFile: "agent.db",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dataDir := tt.setup(t)
			got := isEnrolled(dataDir, tt.dbFile)
			if got != tt.want {
				t.Errorf("isEnrolled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClearSeedData_PreservesProductionRows(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "agent.db")
	db, err := comms.OpenDB(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := store.ApplySchema(db); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	if err := store.ApplyMigrations(db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	// Insert seed rows (match seed.go format: l001, h001)
	_, err = db.Exec(`INSERT INTO agent_logs (id, level, message, source, timestamp) VALUES ('l001', 'info', 'seed log', 'test', '2026-01-01T00:00:00Z')`)
	if err != nil {
		t.Fatalf("insert seed log: %v", err)
	}
	_, err = db.Exec(`INSERT INTO patch_history (id, patch_name, patch_version, action, result, completed_at) VALUES ('h001', 'openssl', '3.0.7', 'install', 'success', '2026-01-01T00:00:00Z')`)
	if err != nil {
		t.Fatalf("insert seed history: %v", err)
	}

	// Insert production rows (match generateLogID format: log-...)
	_, err = db.Exec(`INSERT INTO agent_logs (id, level, message, source, timestamp) VALUES ('log-1234-abcd', 'info', 'production log', 'main', '2026-01-02T00:00:00Z')`)
	if err != nil {
		t.Fatalf("insert prod log: %v", err)
	}
	_, err = db.Exec(`INSERT INTO patch_history (id, patch_name, patch_version, action, result, completed_at) VALUES ('hist-5678-efgh', 'curl', '7.88', 'install', 'success', '2026-01-02T00:00:00Z')`)
	if err != nil {
		t.Fatalf("insert prod history: %v", err)
	}

	// Run the seed cleanup
	clearSeedData(context.Background(), db, slog.Default())

	// Verify seed rows are deleted
	var logCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM agent_logs WHERE id = 'l001'`).Scan(&logCount); err != nil {
		t.Fatalf("count seed logs: %v", err)
	}
	if logCount != 0 {
		t.Errorf("seed log l001 should be deleted, but still exists")
	}

	var histCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM patch_history WHERE id = 'h001'`).Scan(&histCount); err != nil {
		t.Fatalf("count seed history: %v", err)
	}
	if histCount != 0 {
		t.Errorf("seed history h001 should be deleted, but still exists")
	}

	// Verify production rows survive
	var prodLogCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM agent_logs WHERE id = 'log-1234-abcd'`).Scan(&prodLogCount); err != nil {
		t.Fatalf("count prod logs: %v", err)
	}
	if prodLogCount != 1 {
		t.Errorf("production log 'log-1234-abcd' should survive seed cleanup, got count=%d", prodLogCount)
	}

	var prodHistCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM patch_history WHERE id = 'hist-5678-efgh'`).Scan(&prodHistCount); err != nil {
		t.Fatalf("count prod history: %v", err)
	}
	if prodHistCount != 1 {
		t.Errorf("production history 'hist-5678-efgh' should survive seed cleanup, got count=%d", prodHistCount)
	}
}

func TestIsEnrolled_OpenError(t *testing.T) {
	// Create a file that is not a valid SQLite database.
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "agent.db")
	if err := os.WriteFile(dbPath, []byte("not a database"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	got := isEnrolled(dir, "agent.db")
	if got {
		t.Error("isEnrolled() = true for corrupt db, want false")
	}
}
