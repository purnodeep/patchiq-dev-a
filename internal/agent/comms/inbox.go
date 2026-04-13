package comms

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// InboxItem represents a command received from the server.
type InboxItem struct {
	ID          string
	CommandType string
	Payload     []byte
	Priority    int
	ReceivedAt  string
	Status      string
}

// Inbox manages server commands stored locally in SQLite.
type Inbox struct {
	db *sql.DB
}

// NewInbox creates an Inbox backed by the given SQLite database.
func NewInbox(db *sql.DB) *Inbox {
	return &Inbox{db: db}
}

// Store inserts a command into the inbox. Idempotent on ID.
func (i *Inbox) Store(ctx context.Context, item InboxItem) error {
	_, err := i.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO inbox (id, command_type, payload, priority, received_at, status)
		 VALUES (?, ?, ?, ?, ?, 'pending')`,
		item.ID, item.CommandType, item.Payload, item.Priority,
		item.ReceivedAt,
	)
	if err != nil {
		return fmt.Errorf("inbox store %s: %w", item.ID, err)
	}
	return nil
}

// Pending returns pending commands ordered by priority (highest first).
func (i *Inbox) Pending(ctx context.Context, limit int) ([]InboxItem, error) {
	rows, err := i.db.QueryContext(ctx,
		`SELECT id, command_type, payload, priority, received_at, status
		 FROM inbox WHERE status = 'pending' ORDER BY priority DESC, received_at ASC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("inbox pending: %w", err)
	}
	defer rows.Close()

	var items []InboxItem
	for rows.Next() {
		var item InboxItem
		if err := rows.Scan(&item.ID, &item.CommandType, &item.Payload, &item.Priority, &item.ReceivedAt, &item.Status); err != nil {
			return nil, fmt.Errorf("inbox scan: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// MarkCompleted marks a command as completed with the given result.
func (i *Inbox) MarkCompleted(ctx context.Context, id string, result []byte) error {
	_, err := i.db.ExecContext(ctx,
		`UPDATE inbox SET status = 'completed', result = ?, completed_at = ? WHERE id = ?`,
		result, time.Now().UTC().Format(time.RFC3339Nano), id,
	)
	if err != nil {
		return fmt.Errorf("inbox mark completed %s: %w", id, err)
	}
	return nil
}

// MarkFailed marks a command as failed.
func (i *Inbox) MarkFailed(ctx context.Context, id string, errMsg string) error {
	_, err := i.db.ExecContext(ctx,
		`UPDATE inbox SET status = 'failed', result = ?, completed_at = ? WHERE id = ?`,
		[]byte(errMsg), time.Now().UTC().Format(time.RFC3339Nano), id,
	)
	if err != nil {
		return fmt.Errorf("inbox mark failed %s: %w", id, err)
	}
	return nil
}
