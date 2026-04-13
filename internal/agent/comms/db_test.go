package comms_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/skenzeriq/patchiq/internal/agent/comms"
)

func TestOpenDB_CreatesSchemaAndTables(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "agent.db")

	db, err := comms.OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer db.Close()

	tables := []string{"outbox", "inbox", "local_inventory", "agent_state"}
	for _, table := range tables {
		var name string
		err := db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", table, err)
		}
	}
}

func TestOpenDB_CreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "subdir", "nested", "agent.db")

	db, err := comms.OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer db.Close()

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("database file was not created")
	}
}
