package store

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestInventoryCache_SaveAndLoad(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := ApplySchema(db); err != nil {
		t.Fatal(err)
	}
	if err := ApplyMigrations(db); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	pkgsJSON := []byte(`[{"name":"vscode","version":"1.85"}]`)
	if err := SaveInventoryCache(ctx, db, pkgsJSON); err != nil {
		t.Fatalf("SaveInventoryCache: %v", err)
	}

	loaded, collectedAt, err := LoadInventoryCache(ctx, db)
	if err != nil {
		t.Fatalf("LoadInventoryCache: %v", err)
	}
	if string(loaded) != string(pkgsJSON) {
		t.Errorf("loaded = %s, want %s", loaded, pkgsJSON)
	}
	if collectedAt.IsZero() {
		t.Error("collected_at should not be zero")
	}
}

func TestInventoryCache_EmptyOnFreshDB(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := ApplySchema(db); err != nil {
		t.Fatal(err)
	}
	if err := ApplyMigrations(db); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	loaded, _, err := LoadInventoryCache(ctx, db)
	if err != nil {
		t.Fatalf("LoadInventoryCache: %v", err)
	}
	if loaded != nil {
		t.Errorf("expected nil on fresh db, got %s", loaded)
	}
}

func TestInventoryCache_Upsert(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := ApplySchema(db); err != nil {
		t.Fatal(err)
	}
	if err := ApplyMigrations(db); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	// Save first version.
	if err := SaveInventoryCache(ctx, db, []byte(`[{"name":"old"}]`)); err != nil {
		t.Fatal(err)
	}
	// Overwrite.
	if err := SaveInventoryCache(ctx, db, []byte(`[{"name":"new"}]`)); err != nil {
		t.Fatal(err)
	}

	loaded, _, err := LoadInventoryCache(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if string(loaded) != `[{"name":"new"}]` {
		t.Errorf("expected new, got %s", loaded)
	}
}
