package comms

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// PendingItem represents a message waiting to be sent to the server.
type PendingItem struct {
	ID          int64
	MessageType string
	Payload     []byte
	CreatedAt   string
	Attempts    int
	LastError   string
}

// Outbox manages the agent's outbound message queue in SQLite.
type Outbox struct {
	db *sql.DB
}

// NewOutbox creates an Outbox backed by the given SQLite database.
func NewOutbox(db *sql.DB) *Outbox {
	return &Outbox{db: db}
}

// Add inserts a new message into the outbox with status 'pending'.
func (o *Outbox) Add(ctx context.Context, messageType string, payload []byte) (int64, error) {
	result, err := o.db.ExecContext(ctx,
		`INSERT INTO outbox (message_type, payload, created_at, status) VALUES (?, ?, ?, 'pending')`,
		messageType, payload, time.Now().UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return 0, fmt.Errorf("outbox add: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("outbox last insert id: %w", err)
	}
	return id, nil
}

// Pending returns up to limit messages with status 'pending', ordered oldest first.
func (o *Outbox) Pending(ctx context.Context, limit int) ([]PendingItem, error) {
	rows, err := o.db.QueryContext(ctx,
		`SELECT id, message_type, payload, created_at, attempts, COALESCE(last_error, '')
		 FROM outbox WHERE status = 'pending' ORDER BY created_at ASC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("outbox pending: %w", err)
	}
	defer rows.Close()

	var items []PendingItem
	for rows.Next() {
		var item PendingItem
		if err := rows.Scan(&item.ID, &item.MessageType, &item.Payload, &item.CreatedAt, &item.Attempts, &item.LastError); err != nil {
			return nil, fmt.Errorf("outbox scan: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// PendingCount returns the number of pending messages without loading payloads.
func (o *Outbox) PendingCount(ctx context.Context) (int64, error) {
	var count int64
	err := o.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM outbox WHERE status = 'pending'`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("outbox pending count: %w", err)
	}
	return count, nil
}

// MarkSent marks a message as sent.
func (o *Outbox) MarkSent(ctx context.Context, id int64) error {
	_, err := o.db.ExecContext(ctx, `UPDATE outbox SET status = 'sent' WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("outbox mark sent %d: %w", id, err)
	}
	return nil
}

// MarkFailed marks a message as permanently failed.
func (o *Outbox) MarkFailed(ctx context.Context, id int64, reason string) error {
	_, err := o.db.ExecContext(ctx,
		`UPDATE outbox SET status = 'failed', last_error = ? WHERE id = ?`,
		reason, id,
	)
	if err != nil {
		return fmt.Errorf("outbox mark failed %d: %w", id, err)
	}
	return nil
}

// IncrementAttempts bumps the attempt counter. Item stays 'pending'.
func (o *Outbox) IncrementAttempts(ctx context.Context, id int64, lastError string) error {
	_, err := o.db.ExecContext(ctx,
		`UPDATE outbox SET attempts = attempts + 1, last_error = ? WHERE id = ?`,
		lastError, id,
	)
	if err != nil {
		return fmt.Errorf("outbox increment attempts %d: %w", id, err)
	}
	return nil
}
