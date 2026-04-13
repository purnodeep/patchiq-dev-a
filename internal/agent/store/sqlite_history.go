package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/skenzeriq/patchiq/internal/agent"
	"github.com/skenzeriq/patchiq/internal/agent/api"
)

var _ api.HistoryStore = (*HistoryStore)(nil)

// HistoryStore implements api.HistoryStore backed by SQLite.
type HistoryStore struct {
	db *sql.DB
}

// NewHistoryStore creates a HistoryStore.
func NewHistoryStore(db *sql.DB) *HistoryStore {
	return &HistoryStore{db: db}
}

// InsertHistoryRecord inserts a history record from command execution.
func (s *HistoryStore) InsertHistoryRecord(ctx context.Context, rec agent.HistoryRecord) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO patch_history
			(id, patch_name, patch_version, action, result, error_message, completed_at,
			 duration_seconds, reboot_required, stdout, stderr, exit_code, attempt)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1)`,
		rec.ID, rec.PatchName, rec.PatchVersion, rec.Action, rec.Result,
		rec.ErrorMessage, rec.CompletedAt,
		rec.DurationSeconds, boolToInt(rec.RebootRequired),
		nullString(rec.Stdout), nullString(rec.Stderr), rec.ExitCode,
	)
	if err != nil {
		return fmt.Errorf("insert history record: %w", err)
	}
	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// InsertHistory inserts a single patch history entry.
func (s *HistoryStore) InsertHistory(ctx context.Context, entry api.HistoryEntry) error {
	var dur, exitCode sql.NullInt64
	if entry.DurationSeconds != nil {
		dur = sql.NullInt64{Int64: int64(*entry.DurationSeconds), Valid: true}
	}
	if entry.ExitCode != nil {
		exitCode = sql.NullInt64{Int64: int64(*entry.ExitCode), Valid: true}
	}
	var stdout, stderr sql.NullString
	if entry.Stdout != nil {
		stdout = sql.NullString{String: *entry.Stdout, Valid: true}
	}
	if entry.Stderr != nil {
		stderr = sql.NullString{String: *entry.Stderr, Valid: true}
	}
	var reboot int64
	if entry.RebootRequired {
		reboot = 1
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO patch_history
			(id, patch_name, patch_version, action, result, error_message, completed_at,
			 duration_seconds, reboot_required, stdout, stderr, exit_code, attempt)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ID, entry.PatchName, entry.PatchVersion, entry.Action, entry.Result,
		entry.ErrorMessage, entry.CompletedAt,
		dur, reboot, stdout, stderr, exitCode, entry.Attempt,
	)
	if err != nil {
		return fmt.Errorf("insert patch history: %w", err)
	}
	return nil
}

// ListHistory returns patch history ordered by completed_at DESC with cursor pagination.
func (s *HistoryStore) ListHistory(ctx context.Context, limit int, cursor string, dateRange string) ([]api.HistoryEntry, string, int64, error) {
	// Build date filter
	var sinceStr string
	if dateRange != "" {
		var d time.Duration
		switch dateRange {
		case "24h":
			d = 24 * time.Hour
		case "7d":
			d = 7 * 24 * time.Hour
		case "30d":
			d = 30 * 24 * time.Hour
		}
		if d > 0 {
			sinceStr = time.Now().UTC().Add(-d).Format(time.RFC3339)
		}
	}

	// Count query
	countQuery := `SELECT COUNT(*) FROM patch_history`
	countArgs := []any{}
	if sinceStr != "" {
		countQuery += ` WHERE completed_at >= ?`
		countArgs = append(countArgs, sinceStr)
	}
	var total int64
	if err := s.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, "", 0, fmt.Errorf("count patch history: %w", err)
	}

	// Build main query
	query := `SELECT id, patch_name, patch_version, action, result, error_message, completed_at,
		duration_seconds, size, reboot_required, stdout, stderr, exit_code, attempt
		FROM patch_history`
	args := []any{}
	where := []string{}

	if sinceStr != "" {
		where = append(where, "completed_at >= ?")
		args = append(args, sinceStr)
	}
	if cursor != "" {
		cursorTS, cursorID := decodeCursor(cursor)
		where = append(where, "(completed_at < ? OR (completed_at = ? AND id < ?))")
		args = append(args, cursorTS, cursorTS, cursorID)
	}
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY completed_at DESC, id DESC LIMIT ?"
	args = append(args, limit+1)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", 0, fmt.Errorf("query patch history: %w", err)
	}
	defer rows.Close()

	entries := make([]api.HistoryEntry, 0)
	for rows.Next() {
		var e api.HistoryEntry
		var dur, exitCode, reboot, attempt sql.NullInt64
		var size, stdout, stderr sql.NullString
		if err := rows.Scan(&e.ID, &e.PatchName, &e.PatchVersion, &e.Action, &e.Result,
			&e.ErrorMessage, &e.CompletedAt,
			&dur, &size, &reboot, &stdout, &stderr, &exitCode, &attempt); err != nil {
			return nil, "", 0, fmt.Errorf("scan history entry: %w", err)
		}
		if dur.Valid {
			v := int(dur.Int64)
			e.DurationSeconds = &v
		}
		if size.Valid {
			e.Size = &size.String
		}
		e.RebootRequired = reboot.Valid && reboot.Int64 == 1
		if stdout.Valid {
			e.Stdout = &stdout.String
		}
		if stderr.Valid {
			e.Stderr = &stderr.String
		}
		if exitCode.Valid {
			v := int(exitCode.Int64)
			e.ExitCode = &v
		}
		e.Attempt = 1
		if attempt.Valid && attempt.Int64 > 0 {
			e.Attempt = int(attempt.Int64)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, "", 0, fmt.Errorf("iterate patch history: %w", err)
	}

	var nextCursor string
	if len(entries) > limit {
		last := entries[limit-1]
		nextCursor = encodeCursor(last.CompletedAt, last.ID)
		entries = entries[:limit]
	}
	return entries, nextCursor, total, nil
}
