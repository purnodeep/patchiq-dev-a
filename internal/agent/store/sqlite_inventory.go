package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// SaveInventoryCache upserts the latest inventory snapshot into the local cache.
func SaveInventoryCache(ctx context.Context, db *sql.DB, packagesJSON []byte) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO inventory_cache (id, packages_json, collected_at) VALUES (1, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET packages_json = excluded.packages_json, collected_at = excluded.collected_at`,
		string(packagesJSON), time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("save inventory cache: %w", err)
	}
	return nil
}

// LoadInventoryCache reads the latest cached inventory snapshot.
// Returns nil, zero time, nil if no cache exists.
func LoadInventoryCache(ctx context.Context, db *sql.DB) ([]byte, time.Time, error) {
	var pkgsJSON string
	var collectedAtStr string
	err := db.QueryRowContext(ctx, `SELECT packages_json, collected_at FROM inventory_cache WHERE id = 1`).
		Scan(&pkgsJSON, &collectedAtStr)
	if err == sql.ErrNoRows {
		return nil, time.Time{}, nil
	}
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("load inventory cache: %w", err)
	}

	collectedAt, parseErr := time.Parse(time.RFC3339, collectedAtStr)
	if parseErr != nil {
		return nil, time.Time{}, fmt.Errorf("load inventory cache: parse collected_at %q: %w", collectedAtStr, parseErr)
	}
	return []byte(pkgsJSON), collectedAt, nil
}
