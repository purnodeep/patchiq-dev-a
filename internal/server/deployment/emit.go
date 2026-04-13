package deployment

import (
	"context"
	"log/slog"

	"github.com/skenzeriq/patchiq/internal/shared/domain"
)

// EventEmitter is the subset of domain.EventBus needed for emitting events.
type EventEmitter interface {
	Emit(ctx context.Context, event domain.DomainEvent) error
}

// EmitBestEffort emits domain events after a successful commit. Failures are
// logged but not returned because the DB state is authoritative — returning an
// error would cause callers to retry an already-committed write.
// TODO(PIQ-145): add event emission failure counter (otel/prometheus) to surface silent drops.
func EmitBestEffort(ctx context.Context, bus EventEmitter, events []domain.DomainEvent) {
	if bus == nil {
		slog.ErrorContext(ctx, "EmitBestEffort called with nil bus, dropping events", "count", len(events))
		return
	}
	for _, evt := range events {
		if err := bus.Emit(ctx, evt); err != nil {
			slog.ErrorContext(ctx, "emit event failed (best-effort)",
				"event_type", evt.Type, "resource_id", evt.ResourceID, "error", err)
		}
	}
}
