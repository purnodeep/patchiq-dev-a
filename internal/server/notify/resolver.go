package notify

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/crypto"
)

// ResolvedTarget represents a user+channel pair for notification delivery.
type ResolvedTarget struct {
	UserID      string
	ChannelID   string
	ChannelType string // e.g. "email", "slack", "webhook"
	Recipient   string // decoded recipient address (ShoutrrrURL used for delivery)
	ShoutrrrURL string
}

// DBPreferenceResolver resolves notification preferences from the database,
// returning the set of user+channel targets that should receive a notification
// for a given trigger type.
type DBPreferenceResolver struct {
	pool      *pgxpool.Pool
	cryptoKey []byte
}

// NewDBPreferenceResolver creates a resolver backed by PostgreSQL.
func NewDBPreferenceResolver(pool *pgxpool.Pool, cryptoKey []byte) *DBPreferenceResolver {
	return &DBPreferenceResolver{pool: pool, cryptoKey: cryptoKey}
}

// ResolveTargets queries enabled notification preferences for the given tenant
// and trigger type, decrypts channel configs, and returns resolved targets.
func (r *DBPreferenceResolver) ResolveTargets(ctx context.Context, tenantID, triggerType string) ([]ResolvedTarget, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve targets: begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx,
		"SELECT set_config('app.current_tenant_id', $1, true)", tenantID,
	); err != nil {
		return nil, fmt.Errorf("resolve targets: set tenant: %w", err)
	}

	var tenantUUID pgtype.UUID
	if parseErr := tenantUUID.Scan(tenantID); parseErr != nil {
		return nil, fmt.Errorf("resolve targets: parse tenant_id: %w", parseErr)
	}

	q := sqlcgen.New(tx)
	rows, err := q.ListEnabledPreferencesForTrigger(ctx, sqlcgen.ListEnabledPreferencesForTriggerParams{
		TenantID:    tenantUUID,
		TriggerType: triggerType,
	})
	if err != nil {
		return nil, fmt.Errorf("resolve targets: query preferences: %w", err)
	}

	var targets []ResolvedTarget
	for _, row := range rows {
		decrypted, decErr := crypto.Decrypt(r.cryptoKey, row.ConfigEncrypted)
		if decErr != nil {
			slog.ErrorContext(ctx, "decrypt channel config failed",
				"user_id", row.UserID, "channel_type", row.ChannelType, "error", decErr)
			continue
		}
		targets = append(targets, ResolvedTarget{
			UserID:      row.UserID,
			ChannelID:   uuid.UUID(row.ID.Bytes).String(),
			ChannelType: row.ChannelType,
			Recipient:   row.ChannelType, // TODO(PIQ-244): decode recipient from config when crypto layer supports it
			ShoutrrrURL: string(decrypted),
		})
	}

	return targets, nil
}
