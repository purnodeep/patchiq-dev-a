package store_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/skenzeriq/patchiq/internal/agent/store"
	_ "modernc.org/sqlite"
)

func setupRollbackTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ApplySchema(db); err != nil {
		t.Fatal(err)
	}
	if err := store.ApplyMigrations(db); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestRollbackStore_Save_and_ListByCommand(t *testing.T) {
	db := setupRollbackTestDB(t)
	s := store.NewRollbackStore(db)
	ctx := context.Background()

	r1 := &store.RollbackRecord{
		ID:          "rb-001",
		CommandID:   "cmd-100",
		PackageName: "curl",
		FromVersion: "7.68.0",
		ToVersion:   "7.88.1",
		Status:      "pending",
	}
	r2 := &store.RollbackRecord{
		ID:          "rb-002",
		CommandID:   "cmd-100",
		PackageName: "wget",
		FromVersion: "1.20",
		ToVersion:   "1.21",
		Status:      "pending",
	}
	r3 := &store.RollbackRecord{
		ID:          "rb-003",
		CommandID:   "cmd-200",
		PackageName: "openssl",
		FromVersion: "1.1.1",
		ToVersion:   "3.0.0",
		Status:      "pending",
	}

	for _, r := range []*store.RollbackRecord{r1, r2, r3} {
		if err := s.Save(ctx, r); err != nil {
			t.Fatalf("save %s: %v", r.ID, err)
		}
	}

	// List by cmd-100 should return 2 records.
	records, err := s.ListByCommand(ctx, "cmd-100")
	if err != nil {
		t.Fatalf("list by command: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("got %d records, want 2", len(records))
	}
	if records[0].PackageName != "curl" || records[1].PackageName != "wget" {
		t.Errorf("unexpected packages: %s, %s", records[0].PackageName, records[1].PackageName)
	}

	// List by cmd-200 should return 1 record.
	records, err = s.ListByCommand(ctx, "cmd-200")
	if err != nil {
		t.Fatalf("list by command: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}

	// List by nonexistent command should return empty.
	records, err = s.ListByCommand(ctx, "cmd-999")
	if err != nil {
		t.Fatalf("list by command: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("got %d records, want 0", len(records))
	}
}

func TestRollbackStore_MarkCompleted(t *testing.T) {
	db := setupRollbackTestDB(t)
	s := store.NewRollbackStore(db)
	ctx := context.Background()

	r := &store.RollbackRecord{
		ID:          "rb-010",
		CommandID:   "cmd-10",
		PackageName: "curl",
		FromVersion: "7.68.0",
		ToVersion:   "7.88.1",
		Status:      "pending",
	}
	if err := s.Save(ctx, r); err != nil {
		t.Fatal(err)
	}

	if err := s.MarkCompleted(ctx, "rb-010"); err != nil {
		t.Fatalf("mark completed: %v", err)
	}

	records, _ := s.ListByCommand(ctx, "cmd-10")
	if len(records) != 1 {
		t.Fatal("expected 1 record")
	}
	if records[0].Status != "completed" {
		t.Errorf("status = %q, want completed", records[0].Status)
	}
	if records[0].RolledBackAt == nil {
		t.Error("expected rolled_back_at to be set")
	}
}

func TestRollbackStore_MarkFailed(t *testing.T) {
	db := setupRollbackTestDB(t)
	s := store.NewRollbackStore(db)
	ctx := context.Background()

	r := &store.RollbackRecord{
		ID:          "rb-020",
		CommandID:   "cmd-20",
		PackageName: "wget",
		FromVersion: "1.20",
		ToVersion:   "1.21",
		Status:      "pending",
	}
	if err := s.Save(ctx, r); err != nil {
		t.Fatal(err)
	}

	if err := s.MarkFailed(ctx, "rb-020"); err != nil {
		t.Fatalf("mark failed: %v", err)
	}

	records, _ := s.ListByCommand(ctx, "cmd-20")
	if records[0].Status != "failed" {
		t.Errorf("status = %q, want failed", records[0].Status)
	}
	if records[0].RolledBackAt == nil {
		t.Error("expected rolled_back_at to be set")
	}
}

func TestRollbackStore_MarkCompleted_not_found(t *testing.T) {
	db := setupRollbackTestDB(t)
	s := store.NewRollbackStore(db)

	err := s.MarkCompleted(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent record")
	}
}

func TestRollbackStore_MarkFailed_not_found(t *testing.T) {
	db := setupRollbackTestDB(t)
	s := store.NewRollbackStore(db)

	err := s.MarkFailed(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent record")
	}
}
