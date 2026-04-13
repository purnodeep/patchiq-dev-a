package store

import (
	"context"
	"database/sql"
	"testing"

	"github.com/skenzeriq/patchiq/internal/agent/api"
)

func TestListHistoryNewFields(t *testing.T) {
	db := openTestDB(t)
	hs := NewHistoryStore(db)

	dur := 154
	stderr := "E: Package not found"
	_, err := db.Exec(`INSERT INTO patch_history
		(id, patch_name, patch_version, action, result, completed_at, duration_seconds, stderr, attempt, reboot_required)
		VALUES ('h1','openssl','3.0.1','install','failed','2026-03-10T10:00:00Z',?,?,?,1)`,
		dur, stderr, 1)
	if err != nil {
		t.Fatal(err)
	}

	entries, _, _, err := hs.ListHistory(context.Background(), 10, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("want 1, got %d", len(entries))
	}
	e := entries[0]
	if e.DurationSeconds == nil || *e.DurationSeconds != 154 {
		t.Errorf("want duration 154, got %v", e.DurationSeconds)
	}
	if e.Stderr == nil || *e.Stderr != stderr {
		t.Errorf("want stderr, got %v", e.Stderr)
	}
	if e.Attempt != 1 {
		t.Errorf("want attempt 1, got %d", e.Attempt)
	}
}

func TestListHistoryDateRange(t *testing.T) {
	db := openTestDB(t)
	hs := NewHistoryStore(db)

	// Recent entry (within 7d of 2026-03-16)
	_, _ = db.Exec(`INSERT INTO patch_history (id, patch_name, patch_version, action, result, completed_at)
		VALUES ('h1','pkg','1.0','install','success','2026-03-15T10:00:00Z')`)
	// Old entry (more than 7d ago)
	_, _ = db.Exec(`INSERT INTO patch_history (id, patch_name, patch_version, action, result, completed_at)
		VALUES ('h2','pkg','1.0','install','success','2026-01-01T10:00:00Z')`)

	entries, _, _, err := hs.ListHistory(context.Background(), 10, "", "7d")
	if err != nil {
		t.Fatal(err)
	}
	// h2 should be filtered out (older than 7d)
	for _, e := range entries {
		if e.ID == "h2" {
			t.Error("old entry h2 should have been filtered by 7d range")
		}
	}
}

func seedHistory(t *testing.T, db *sql.DB, entries []api.HistoryEntry) {
	t.Helper()
	for _, e := range entries {
		var errMsg *string
		if e.ErrorMessage != nil {
			errMsg = e.ErrorMessage
		}
		_, err := db.Exec(
			`INSERT INTO patch_history (id, patch_name, patch_version, action, result, error_message, completed_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			e.ID, e.PatchName, e.PatchVersion, e.Action, e.Result, errMsg, e.CompletedAt,
		)
		if err != nil {
			t.Fatalf("seed history %s: %v", e.ID, err)
		}
	}
}

func TestHistoryStore_ListHistory_Empty(t *testing.T) {
	db := openTestDB(t)
	s := NewHistoryStore(db)

	entries, next, total, err := s.ListHistory(context.Background(), 50, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
	if next != "" {
		t.Fatalf("expected empty cursor, got %q", next)
	}
	if total != 0 {
		t.Fatalf("expected total 0, got %d", total)
	}
}

func TestHistoryStore_ListHistory_WithData(t *testing.T) {
	db := openTestDB(t)
	s := NewHistoryStore(db)

	errMsg := "timeout"
	seedHistory(t, db, []api.HistoryEntry{
		{ID: "h1", PatchName: "p1", PatchVersion: "1.0", Action: "install", Result: "success", CompletedAt: "2026-01-03T00:00:00Z"},
		{ID: "h2", PatchName: "p2", PatchVersion: "2.0", Action: "rollback", Result: "failed", ErrorMessage: &errMsg, CompletedAt: "2026-01-02T00:00:00Z"},
	})

	entries, _, total, err := s.ListHistory(context.Background(), 50, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if total != 2 {
		t.Fatalf("expected total 2, got %d", total)
	}
	// Ordered by completed_at DESC
	if entries[0].ID != "h1" {
		t.Fatalf("expected first entry h1, got %s", entries[0].ID)
	}
	if entries[1].ErrorMessage == nil || *entries[1].ErrorMessage != "timeout" {
		t.Fatal("expected error_message 'timeout' on second entry")
	}
}

func TestHistoryStore_ListHistory_Pagination(t *testing.T) {
	db := openTestDB(t)
	s := NewHistoryStore(db)

	seedHistory(t, db, []api.HistoryEntry{
		{ID: "h1", PatchName: "p1", PatchVersion: "1.0", Action: "install", Result: "success", CompletedAt: "2026-01-03T00:00:00Z"},
		{ID: "h2", PatchName: "p2", PatchVersion: "1.0", Action: "install", Result: "success", CompletedAt: "2026-01-02T00:00:00Z"},
		{ID: "h3", PatchName: "p3", PatchVersion: "1.0", Action: "install", Result: "success", CompletedAt: "2026-01-01T00:00:00Z"},
	})

	entries, next, total, err := s.ListHistory(context.Background(), 2, "", "")
	if err != nil {
		t.Fatalf("page 1 error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("page 1: expected 2, got %d", len(entries))
	}
	if total != 3 {
		t.Fatalf("page 1: expected total 3, got %d", total)
	}
	if next == "" {
		t.Fatal("page 1: expected non-empty cursor")
	}

	entries2, next2, _, err := s.ListHistory(context.Background(), 2, next, "")
	if err != nil {
		t.Fatalf("page 2 error: %v", err)
	}
	if len(entries2) != 1 {
		t.Fatalf("page 2: expected 1, got %d", len(entries2))
	}
	if next2 != "" {
		t.Fatalf("page 2: expected empty cursor, got %q", next2)
	}
}
