package notify

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
)

// DBHistoryRecorder records notification history entries using sqlc.
type DBHistoryRecorder struct {
	pool *pgxpool.Pool
}

func NewDBHistoryRecorder(pool *pgxpool.Pool) *DBHistoryRecorder {
	return &DBHistoryRecorder{pool: pool}
}

func (r *DBHistoryRecorder) Record(ctx context.Context, rec HistoryRecord) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("record notification history: begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx,
		"SELECT set_config('app.current_tenant_id', $1, true)", rec.TenantID,
	); err != nil {
		return fmt.Errorf("record notification history: set tenant: %w", err)
	}

	var tenantUUID pgtype.UUID
	if parseErr := tenantUUID.Scan(rec.TenantID); parseErr != nil {
		return fmt.Errorf("record notification history: parse tenant_id: %w", parseErr)
	}

	var channelUUID pgtype.UUID
	if rec.ChannelID != "" {
		parsed, parseErr := uuid.Parse(rec.ChannelID)
		if parseErr != nil {
			return fmt.Errorf("record notification history: parse channel_id: %w", parseErr)
		}
		channelUUID = pgtype.UUID{Bytes: parsed, Valid: true}
	}

	var errMsg pgtype.Text
	if rec.ErrorMessage != "" {
		errMsg = pgtype.Text{String: rec.ErrorMessage, Valid: true}
	}

	var channelType pgtype.Text
	if rec.ChannelType != "" {
		channelType = pgtype.Text{String: rec.ChannelType, Valid: true}
	}

	q := sqlcgen.New(tx)
	if insertErr := q.InsertNotificationHistory(ctx, sqlcgen.InsertNotificationHistoryParams{
		ID:           rec.ID,
		TenantID:     tenantUUID,
		TriggerType:  rec.TriggerType,
		ChannelID:    channelUUID,
		ChannelType:  channelType,
		Recipient:    rec.Recipient,
		Subject:      rec.Subject,
		UserID:       rec.UserID,
		Status:       rec.Status,
		Payload:      rec.Payload,
		ErrorMessage: errMsg,
	}); insertErr != nil {
		return fmt.Errorf("record notification history: insert: %w", insertErr)
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		return fmt.Errorf("record notification history: commit: %w", commitErr)
	}
	return nil
}
