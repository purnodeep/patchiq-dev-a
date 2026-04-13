package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/skenzeriq/patchiq/internal/agent/api"
)

// encodeCursor builds a compound cursor from a timestamp and id.
func encodeCursor(ts, id string) string {
	return ts + "|" + id
}

// decodeCursor splits a compound cursor into timestamp and id parts.
func decodeCursor(cursor string) (ts, id string) {
	if i := strings.LastIndex(cursor, "|"); i >= 0 {
		return cursor[:i], cursor[i+1:]
	}
	return cursor, ""
}

var _ api.PatchStore = (*PatchStore)(nil)

// PatchStore implements api.PatchStore backed by SQLite.
type PatchStore struct {
	db *sql.DB
}

// NewPatchStore creates a PatchStore.
func NewPatchStore(db *sql.DB) *PatchStore {
	return &PatchStore{db: db}
}

// ListPending returns pending patches ordered by queued_at DESC with cursor pagination.
func (s *PatchStore) ListPending(ctx context.Context, limit int, cursor string) ([]api.PendingPatch, string, int64, error) {
	var total int64
	countQuery := `SELECT COUNT(*) FROM pending_patches`
	if err := s.db.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, "", 0, fmt.Errorf("count pending patches: %w", err)
	}

	query := `SELECT id, name, version, severity, status, queued_at,
		size, cvss_score, cve_ids, published_at, source
		FROM pending_patches`
	args := []any{}

	if cursor != "" {
		cursorTS, cursorID := decodeCursor(cursor)
		query += ` WHERE queued_at < ? OR (queued_at = ? AND id < ?)`
		args = append(args, cursorTS, cursorTS, cursorID)
	}

	query += ` ORDER BY queued_at DESC, id DESC LIMIT ?`
	args = append(args, limit+1)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", 0, fmt.Errorf("query pending patches: %w", err)
	}
	defer rows.Close()

	patches := make([]api.PendingPatch, 0)
	for rows.Next() {
		var p api.PendingPatch
		var size, cveIDs, publishedAt, source sql.NullString
		var cvssScore sql.NullFloat64
		if err := rows.Scan(&p.ID, &p.Name, &p.Version, &p.Severity, &p.Status, &p.QueuedAt,
			&size, &cvssScore, &cveIDs, &publishedAt, &source); err != nil {
			return nil, "", 0, fmt.Errorf("scan pending patch: %w", err)
		}
		if size.Valid {
			p.Size = &size.String
		}
		if cvssScore.Valid {
			p.CVSSScore = &cvssScore.Float64
		}
		if cveIDs.Valid && cveIDs.String != "" {
			if err := json.Unmarshal([]byte(cveIDs.String), &p.CVEIDs); err != nil {
				return nil, "", 0, fmt.Errorf("unmarshal cve_ids for patch %s: %w", p.ID, err)
			}
		}
		if p.CVEIDs == nil {
			p.CVEIDs = []string{}
		}
		if publishedAt.Valid {
			p.PublishedAt = &publishedAt.String
		}
		if source.Valid {
			p.Source = &source.String
		}
		patches = append(patches, p)
	}
	if err := rows.Err(); err != nil {
		return nil, "", 0, fmt.Errorf("iterate pending patches: %w", err)
	}

	var nextCursor string
	if len(patches) > limit {
		last := patches[limit-1]
		nextCursor = encodeCursor(last.QueuedAt, last.ID)
		patches = patches[:limit]
	}

	return patches, nextCursor, total, nil
}
