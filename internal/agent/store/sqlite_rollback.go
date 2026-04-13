package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// RollbackRecord tracks a package version change that can be rolled back.
type RollbackRecord struct {
	ID           string
	CommandID    string
	PackageName  string
	FromVersion  string
	ToVersion    string
	RolledBackAt *time.Time
	Status       string // pending, completed, failed
}

// RollbackStore provides SQLite operations for rollback records.
type RollbackStore struct {
	db *sql.DB
}

// NewRollbackStore creates a RollbackStore.
func NewRollbackStore(db *sql.DB) *RollbackStore {
	return &RollbackStore{db: db}
}

// Save inserts a new rollback record.
func (s *RollbackStore) Save(ctx context.Context, record *RollbackRecord) error {
	const query = `INSERT INTO rollback_records (id, command_id, package_name, from_version, to_version, rolled_back_at, status)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	var rolledBackAt *string
	if record.RolledBackAt != nil {
		v := record.RolledBackAt.UTC().Format(time.RFC3339)
		rolledBackAt = &v
	}

	_, err := s.db.ExecContext(ctx, query,
		record.ID,
		record.CommandID,
		record.PackageName,
		record.FromVersion,
		record.ToVersion,
		rolledBackAt,
		record.Status,
	)
	if err != nil {
		return fmt.Errorf("save rollback record: %w", err)
	}
	return nil
}

// ListByCommand returns all rollback records for a given command ID.
func (s *RollbackStore) ListByCommand(ctx context.Context, commandID string) ([]*RollbackRecord, error) {
	const query = `SELECT id, command_id, package_name, from_version, to_version, rolled_back_at, status
		FROM rollback_records WHERE command_id = ? ORDER BY id`

	rows, err := s.db.QueryContext(ctx, query, commandID)
	if err != nil {
		return nil, fmt.Errorf("list rollback records by command %s: %w", commandID, err)
	}
	defer rows.Close()

	var records []*RollbackRecord
	for rows.Next() {
		r := &RollbackRecord{}
		var rolledBackAt sql.NullString
		if err := rows.Scan(&r.ID, &r.CommandID, &r.PackageName, &r.FromVersion, &r.ToVersion, &rolledBackAt, &r.Status); err != nil {
			return nil, fmt.Errorf("scan rollback record: %w", err)
		}
		if rolledBackAt.Valid {
			t, err := time.Parse(time.RFC3339, rolledBackAt.String)
			if err == nil {
				r.RolledBackAt = &t
			}
		}
		records = append(records, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rollback records: %w", err)
	}
	return records, nil
}

// MarkCompleted updates a rollback record's status to completed and sets rolled_back_at.
func (s *RollbackStore) MarkCompleted(ctx context.Context, id string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	const query = `UPDATE rollback_records SET status = 'completed', rolled_back_at = ? WHERE id = ?`
	res, err := s.db.ExecContext(ctx, query, now, id)
	if err != nil {
		return fmt.Errorf("mark rollback record completed: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("mark rollback record completed: record %s not found", id)
	}
	return nil
}

// MarkFailed updates a rollback record's status to failed and sets rolled_back_at.
func (s *RollbackStore) MarkFailed(ctx context.Context, id string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	const query = `UPDATE rollback_records SET status = 'failed', rolled_back_at = ? WHERE id = ?`
	res, err := s.db.ExecContext(ctx, query, now, id)
	if err != nil {
		return fmt.Errorf("mark rollback record failed: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("mark rollback record failed: record %s not found", id)
	}
	return nil
}
