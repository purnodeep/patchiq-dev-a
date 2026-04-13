package v1

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/user"
)

// scanUUID parses a string UUID into the pgtype representation used by sqlc.
func scanUUID(s string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(s)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}, nil
}

// uuidToString converts a pgtype.UUID to its canonical string form.
// Returns "" for invalid (null) UUIDs.
func uuidToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return uuid.UUID(u.Bytes).String()
}

// textFromString wraps a Go string into a pgtype.Text.
// An empty string produces a null pgtype.Text.
func textFromString(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

// isNotFound reports whether err represents a "no rows" result from pgx.
func isNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

// isUniqueViolation reports whether err is a PostgreSQL unique constraint violation (23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// emitEvent publishes a domain event on the bus, extracting the real user ID
// from context. Falls back to "system" if no user ID is present.
// Errors are logged but never propagated — event emission must not fail requests.
// TODO(#177): Consider an outbox pattern to guarantee event delivery for audit compliance.
func emitEvent(ctx context.Context, bus domain.EventBus, eventType, resource, resourceID, tenantID string, payload any) {
	if bus == nil {
		slog.ErrorContext(ctx, "event bus is nil — domain event not emitted",
			"event_type", eventType, "resource", resource, "resource_id", resourceID)
		return
	}
	actorID := "system"
	actorType := domain.ActorSystem
	if uid, ok := user.UserIDFromContext(ctx); ok && uid != "" {
		actorID = uid
		actorType = domain.ActorUser
	}
	event := domain.DomainEvent{
		ID:         domain.NewEventID(),
		Type:       eventType,
		TenantID:   tenantID,
		ActorID:    actorID,
		ActorType:  actorType,
		Resource:   resource,
		ResourceID: resourceID,
		Action:     eventType,
		Payload:    payload,
		Timestamp:  time.Now(),
	}
	if err := bus.Emit(ctx, event); err != nil {
		slog.ErrorContext(ctx, "emit domain event failed",
			"event_type", eventType, "resource", resource,
			"resource_id", resourceID, "tenant_id", tenantID, "error", err)
	}
}

// nullableText converts a pgtype.Text to *string, returning nil if not valid.
func nullableText(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}
