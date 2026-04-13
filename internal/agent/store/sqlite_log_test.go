package store

import (
	"context"
	"database/sql"
	"testing"

	"github.com/skenzeriq/patchiq/internal/agent/api"
)

func seedLogs(t *testing.T, db *sql.DB, entries []api.LogEntry) {
	t.Helper()
	for _, e := range entries {
		_, err := db.Exec(
			`INSERT INTO agent_logs (id, level, message, source, timestamp) VALUES (?, ?, ?, ?, ?)`,
			e.ID, e.Level, e.Message, e.Source, e.Timestamp,
		)
		if err != nil {
			t.Fatalf("seed log %s: %v", e.ID, err)
		}
	}
}

func TestLogStore_ListLogs_Empty(t *testing.T) {
	db := openTestDB(t)
	s := NewLogStore(db)

	entries, next, total, err := s.ListLogs(context.Background(), 50, "", "")
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

func TestLogStore_ListLogs_WithData(t *testing.T) {
	db := openTestDB(t)
	s := NewLogStore(db)

	src := "agent"
	seedLogs(t, db, []api.LogEntry{
		{ID: "l1", Level: "error", Message: "disk full", Source: &src, Timestamp: "2026-01-03T00:00:00Z"},
		{ID: "l2", Level: "info", Message: "started", Timestamp: "2026-01-02T00:00:00Z"},
		{ID: "l3", Level: "warn", Message: "slow query", Timestamp: "2026-01-01T00:00:00Z"},
	})

	entries, _, total, err := s.ListLogs(context.Background(), 50, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if total != 3 {
		t.Fatalf("expected total 3, got %d", total)
	}
	// Ordered by timestamp DESC
	if entries[0].ID != "l1" {
		t.Fatalf("expected first entry l1, got %s", entries[0].ID)
	}
}

func TestLogStore_ListLogs_LevelFilter(t *testing.T) {
	db := openTestDB(t)
	s := NewLogStore(db)

	seedLogs(t, db, []api.LogEntry{
		{ID: "l1", Level: "error", Message: "disk full", Timestamp: "2026-01-03T00:00:00Z"},
		{ID: "l2", Level: "info", Message: "started", Timestamp: "2026-01-02T00:00:00Z"},
		{ID: "l3", Level: "error", Message: "oom", Timestamp: "2026-01-01T00:00:00Z"},
	})

	entries, _, total, err := s.ListLogs(context.Background(), 50, "", "error")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 error entries, got %d", len(entries))
	}
	if total != 2 {
		t.Fatalf("expected total 2, got %d", total)
	}
}

func TestLogStore_ListLogs_LevelFilterWithPagination(t *testing.T) {
	db := openTestDB(t)
	s := NewLogStore(db)

	seedLogs(t, db, []api.LogEntry{
		{ID: "l1", Level: "error", Message: "err1", Timestamp: "2026-01-04T00:00:00Z"},
		{ID: "l2", Level: "info", Message: "info1", Timestamp: "2026-01-03T00:00:00Z"},
		{ID: "l3", Level: "error", Message: "err2", Timestamp: "2026-01-02T00:00:00Z"},
		{ID: "l4", Level: "error", Message: "err3", Timestamp: "2026-01-01T00:00:00Z"},
	})

	// Page 1: limit 1, filter error
	entries, next, total, err := s.ListLogs(context.Background(), 1, "", "error")
	if err != nil {
		t.Fatalf("page 1 error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("page 1: expected 1, got %d", len(entries))
	}
	if total != 3 {
		t.Fatalf("page 1: expected total 3 (error only), got %d", total)
	}
	if entries[0].ID != "l1" {
		t.Fatalf("page 1: expected l1, got %s", entries[0].ID)
	}
	if next == "" {
		t.Fatal("page 1: expected non-empty cursor")
	}

	// Page 2
	entries2, next2, _, err := s.ListLogs(context.Background(), 1, next, "error")
	if err != nil {
		t.Fatalf("page 2 error: %v", err)
	}
	if len(entries2) != 1 {
		t.Fatalf("page 2: expected 1, got %d", len(entries2))
	}
	if entries2[0].ID != "l3" {
		t.Fatalf("page 2: expected l3, got %s", entries2[0].ID)
	}

	// Page 3
	entries3, next3, _, err := s.ListLogs(context.Background(), 1, next2, "error")
	if err != nil {
		t.Fatalf("page 3 error: %v", err)
	}
	if len(entries3) != 1 {
		t.Fatalf("page 3: expected 1, got %d", len(entries3))
	}
	if entries3[0].ID != "l4" {
		t.Fatalf("page 3: expected l4, got %s", entries3[0].ID)
	}
	if next3 != "" {
		t.Fatalf("page 3: expected empty cursor, got %q", next3)
	}
}

func TestLogStore_ListLogs_Pagination(t *testing.T) {
	db := openTestDB(t)
	s := NewLogStore(db)

	seedLogs(t, db, []api.LogEntry{
		{ID: "l1", Level: "info", Message: "a", Timestamp: "2026-01-03T00:00:00Z"},
		{ID: "l2", Level: "info", Message: "b", Timestamp: "2026-01-02T00:00:00Z"},
		{ID: "l3", Level: "info", Message: "c", Timestamp: "2026-01-01T00:00:00Z"},
	})

	entries, next, total, err := s.ListLogs(context.Background(), 2, "", "")
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

	entries2, next2, _, err := s.ListLogs(context.Background(), 2, next, "")
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
