package store

import (
	"context"
	"database/sql"
	"testing"

	"github.com/skenzeriq/patchiq/internal/agent/api"
	_ "modernc.org/sqlite"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:?_pragma=journal_mode(wal)&_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	db.SetMaxOpenConns(1)
	if err := ApplySchema(db); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	if err := ApplyMigrations(db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func seedPatches(t *testing.T, db *sql.DB, patches []api.PendingPatch) {
	t.Helper()
	for _, p := range patches {
		_, err := db.Exec(
			`INSERT INTO pending_patches (id, name, version, severity, status, queued_at) VALUES (?, ?, ?, ?, ?, ?)`,
			p.ID, p.Name, p.Version, p.Severity, p.Status, p.QueuedAt,
		)
		if err != nil {
			t.Fatalf("seed patch %s: %v", p.ID, err)
		}
	}
}

func TestPatchStore_ListPending_Empty(t *testing.T) {
	db := openTestDB(t)
	s := NewPatchStore(db)

	patches, next, total, err := s.ListPending(context.Background(), 50, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(patches) != 0 {
		t.Fatalf("expected 0 patches, got %d", len(patches))
	}
	if next != "" {
		t.Fatalf("expected empty cursor, got %q", next)
	}
	if total != 0 {
		t.Fatalf("expected total 0, got %d", total)
	}
}

func TestPatchStore_ListPending_WithData(t *testing.T) {
	db := openTestDB(t)
	s := NewPatchStore(db)

	seedPatches(t, db, []api.PendingPatch{
		{ID: "p1", Name: "patch-a", Version: "1.0", Severity: "high", Status: "queued", QueuedAt: "2026-01-03T00:00:00Z"},
		{ID: "p2", Name: "patch-b", Version: "2.0", Severity: "critical", Status: "downloading", QueuedAt: "2026-01-02T00:00:00Z"},
		{ID: "p3", Name: "patch-c", Version: "3.0", Severity: "low", Status: "queued", QueuedAt: "2026-01-01T00:00:00Z"},
	})

	patches, next, total, err := s.ListPending(context.Background(), 50, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(patches) != 3 {
		t.Fatalf("expected 3 patches, got %d", len(patches))
	}
	if total != 3 {
		t.Fatalf("expected total 3, got %d", total)
	}
	if next != "" {
		t.Fatalf("expected empty cursor on last page, got %q", next)
	}
	// Ordered by queued_at DESC
	if patches[0].ID != "p1" {
		t.Fatalf("expected first patch p1, got %s", patches[0].ID)
	}
}

func TestPatchStore_ListPending_DuplicateTimestamps(t *testing.T) {
	db := openTestDB(t)
	s := NewPatchStore(db)

	// All three patches have the same queued_at — cursor must use compound key
	seedPatches(t, db, []api.PendingPatch{
		{ID: "p-a", Name: "a", Version: "1.0", Severity: "high", Status: "queued", QueuedAt: "2026-01-01T00:00:00Z"},
		{ID: "p-b", Name: "b", Version: "1.0", Severity: "low", Status: "queued", QueuedAt: "2026-01-01T00:00:00Z"},
		{ID: "p-c", Name: "c", Version: "1.0", Severity: "medium", Status: "queued", QueuedAt: "2026-01-01T00:00:00Z"},
	})

	seen := map[string]bool{}

	// Page through with limit=1 to exercise tie-breaking
	patches, next, _, err := s.ListPending(context.Background(), 1, "")
	if err != nil {
		t.Fatalf("page 1 error: %v", err)
	}
	if len(patches) != 1 {
		t.Fatalf("page 1: expected 1, got %d", len(patches))
	}
	seen[patches[0].ID] = true

	for next != "" {
		patches, next, _, err = s.ListPending(context.Background(), 1, next)
		if err != nil {
			t.Fatalf("error during pagination: %v", err)
		}
		for _, p := range patches {
			if seen[p.ID] {
				t.Fatalf("duplicate patch %s seen across pages", p.ID)
			}
			seen[p.ID] = true
		}
	}

	if len(seen) != 3 {
		t.Fatalf("expected to see 3 unique patches, got %d: %v", len(seen), seen)
	}
}

func TestListPendingWithNewFields(t *testing.T) {
	db := openTestDB(t)
	// Apply migrations to add new columns
	if err := ApplyMigrations(db); err != nil {
		t.Fatal(err)
	}
	ps := NewPatchStore(db)

	cvss := 9.8
	size := "45 MB"
	cves := `["CVE-2024-1234","CVE-2024-5678"]`
	source := "apt"
	_, err := db.Exec(`INSERT INTO pending_patches (id, name, version, severity, status, queued_at, cvss_score, size, cve_ids, source)
		VALUES ('p1', 'openssl', '3.0.1', 'critical', 'queued', '2026-03-10T10:00:00Z', ?, ?, ?, ?)`,
		cvss, size, cves, source)
	if err != nil {
		t.Fatal(err)
	}

	patches, _, _, err := ps.ListPending(context.Background(), 10, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(patches) != 1 {
		t.Fatalf("want 1 patch, got %d", len(patches))
	}
	p := patches[0]
	if p.CVSSScore == nil || *p.CVSSScore != 9.8 {
		t.Errorf("want cvss 9.8, got %v", p.CVSSScore)
	}
	if p.Size == nil || *p.Size != "45 MB" {
		t.Errorf("want size '45 MB', got %v", p.Size)
	}
	if len(p.CVEIDs) != 2 || p.CVEIDs[0] != "CVE-2024-1234" {
		t.Errorf("want CVE-2024-1234, got %v", p.CVEIDs)
	}
	if p.Source == nil || *p.Source != "apt" {
		t.Errorf("want source 'apt', got %v", p.Source)
	}
}

func TestListPendingNullFields(t *testing.T) {
	db := openTestDB(t)
	if err := ApplyMigrations(db); err != nil {
		t.Fatal(err)
	}
	ps := NewPatchStore(db)

	_, err := db.Exec(`INSERT INTO pending_patches (id, name, version, severity, status, queued_at)
		VALUES ('p2', 'curl', '7.88', 'medium', 'queued', '2026-03-10T09:00:00Z')`)
	if err != nil {
		t.Fatal(err)
	}

	patches, _, _, err := ps.ListPending(context.Background(), 10, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(patches) != 1 {
		t.Fatalf("want 1 patch, got %d", len(patches))
	}
	p := patches[0]
	if p.CVSSScore != nil {
		t.Errorf("want nil cvss, got %v", p.CVSSScore)
	}
	if p.CVEIDs == nil {
		t.Error("want empty slice, got nil CVEIDs")
	}
}

func TestPatchStore_ListPending_Pagination(t *testing.T) {
	db := openTestDB(t)
	s := NewPatchStore(db)

	seedPatches(t, db, []api.PendingPatch{
		{ID: "p1", Name: "a", Version: "1.0", Severity: "high", Status: "queued", QueuedAt: "2026-01-03T00:00:00Z"},
		{ID: "p2", Name: "b", Version: "1.0", Severity: "low", Status: "queued", QueuedAt: "2026-01-02T00:00:00Z"},
		{ID: "p3", Name: "c", Version: "1.0", Severity: "medium", Status: "queued", QueuedAt: "2026-01-01T00:00:00Z"},
	})

	// Page 1: limit 2
	patches, next, total, err := s.ListPending(context.Background(), 2, "")
	if err != nil {
		t.Fatalf("page 1 error: %v", err)
	}
	if len(patches) != 2 {
		t.Fatalf("page 1: expected 2, got %d", len(patches))
	}
	if total != 3 {
		t.Fatalf("page 1: expected total 3, got %d", total)
	}
	if next == "" {
		t.Fatal("page 1: expected non-empty cursor")
	}

	// Page 2: use cursor
	patches2, next2, total2, err := s.ListPending(context.Background(), 2, next)
	if err != nil {
		t.Fatalf("page 2 error: %v", err)
	}
	if len(patches2) != 1 {
		t.Fatalf("page 2: expected 1, got %d", len(patches2))
	}
	if total2 != 3 {
		t.Fatalf("page 2: expected total 3, got %d", total2)
	}
	if next2 != "" {
		t.Fatalf("page 2: expected empty cursor, got %q", next2)
	}
}
