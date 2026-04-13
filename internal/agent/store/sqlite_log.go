package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/skenzeriq/patchiq/internal/agent/api"
)

var (
	_ api.LogStore  = (*LogStore)(nil)
	_ api.LogWriter = (*LogStore)(nil)
)

// LogStore implements api.LogStore backed by SQLite.
type LogStore struct {
	db *sql.DB
}

// NewLogStore creates a LogStore.
func NewLogStore(db *sql.DB) *LogStore {
	return &LogStore{db: db}
}

// ListLogs returns log entries ordered by timestamp DESC with cursor pagination and optional level filter.
func (s *LogStore) ListLogs(ctx context.Context, limit int, cursor string, level string) ([]api.LogEntry, string, int64, error) {
	// Count with optional level filter
	countQuery := `SELECT COUNT(*) FROM agent_logs`
	countArgs := []any{}
	if level != "" {
		countQuery += ` WHERE level = ?`
		countArgs = append(countArgs, level)
	}

	var total int64
	if err := s.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, "", 0, fmt.Errorf("count agent logs: %w", err)
	}

	query := `SELECT id, level, message, source, timestamp FROM agent_logs`
	args := []any{}
	conditions := []string{}

	if level != "" {
		conditions = append(conditions, `level = ?`)
		args = append(args, level)
	}
	if cursor != "" {
		cursorTS, cursorID := decodeCursor(cursor)
		conditions = append(conditions, `(timestamp < ? OR (timestamp = ? AND id < ?))`)
		args = append(args, cursorTS, cursorTS, cursorID)
	}

	if len(conditions) > 0 {
		query += ` WHERE ` + strings.Join(conditions, ` AND `)
	}

	query += ` ORDER BY timestamp DESC, id DESC LIMIT ?`
	args = append(args, limit+1)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", 0, fmt.Errorf("query agent logs: %w", err)
	}
	defer rows.Close()

	entries := make([]api.LogEntry, 0)
	for rows.Next() {
		var e api.LogEntry
		if err := rows.Scan(&e.ID, &e.Level, &e.Message, &e.Source, &e.Timestamp); err != nil {
			return nil, "", 0, fmt.Errorf("scan log entry: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, "", 0, fmt.Errorf("iterate agent logs: %w", err)
	}

	var nextCursor string
	if len(entries) > limit {
		last := entries[limit-1]
		nextCursor = encodeCursor(last.Timestamp, last.ID)
		entries = entries[:limit]
	}

	return entries, nextCursor, total, nil
}

// WriteLog inserts a log entry into the agent_logs table.
func (s *LogStore) WriteLog(ctx context.Context, level, message, source string) error {
	id := generateLogID()
	ts := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO agent_logs (id, level, message, source, timestamp) VALUES (?, ?, ?, ?, ?)`,
		id, level, message, source, ts,
	)
	if err != nil {
		return fmt.Errorf("write agent log: %w", err)
	}
	return nil
}

// generateLogID returns a unique log entry ID using timestamp + random suffix.
func generateLogID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("log-%d-%s", time.Now().UnixNano(), hex.EncodeToString(b))
}
