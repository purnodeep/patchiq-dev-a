package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/skenzeriq/patchiq/internal/hub/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
)

// AuditSubscriber writes domain events to the audit_events table.
type AuditSubscriber struct {
	pool *pgxpool.Pool
	log  *slog.Logger
}

// NewAuditSubscriber creates a subscriber that persists events to audit_events.
func NewAuditSubscriber(pool *pgxpool.Pool, logger *slog.Logger) *AuditSubscriber {
	return &AuditSubscriber{pool: pool, log: logger}
}

// Handle persists a domain event to the audit_events table.
// It sets the tenant context on the transaction for RLS compliance.
func (s *AuditSubscriber) Handle(ctx context.Context, event domain.DomainEvent) error {
	// Skip audit for events without a tenant ID (Hub-internal operations like binary fetches).
	if event.TenantID == "" {
		slog.DebugContext(ctx, "audit subscriber: skipping event with empty tenant ID",
			"event_type", event.Type, "event_id", event.ID)
		return nil
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin audit tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Set tenant context for RLS.
	if _, err := tx.Exec(ctx,
		"SELECT set_config('app.current_tenant_id', $1, true)", event.TenantID,
	); err != nil {
		return fmt.Errorf("set tenant context for audit: %w", err)
	}

	// Marshal payload and metadata to JSON for JSONB columns.
	var payloadBytes []byte
	if event.Payload != nil {
		payloadBytes, err = json.Marshal(event.Payload)
		if err != nil {
			return fmt.Errorf("marshal audit payload: %w", err)
		}
	}

	metaBytes, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("marshal audit metadata: %w", err)
	}

	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(event.TenantID); err != nil {
		return fmt.Errorf("parse tenant UUID %q: %w", event.TenantID, err)
	}

	var ts pgtype.Timestamptz
	ts.Time = event.Timestamp
	ts.Valid = true

	queries := sqlcgen.New(tx)
	if err := queries.InsertAuditEvent(ctx, sqlcgen.InsertAuditEventParams{
		ID:         event.ID,
		Type:       event.Type,
		TenantID:   tenantUUID,
		ActorID:    event.ActorID,
		ActorType:  event.ActorType,
		Resource:   event.Resource,
		ResourceID: event.ResourceID,
		Action:     event.Action,
		Payload:    payloadBytes,
		Metadata:   metaBytes,
		Timestamp:  ts,
	}); err != nil {
		return fmt.Errorf("insert audit event %s: %w", event.ID, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit audit tx: %w", err)
	}

	s.log.DebugContext(ctx, "audit event persisted",
		"event_id", event.ID,
		"event_type", event.Type,
		"tenant_id", event.TenantID,
	)
	return nil
}
