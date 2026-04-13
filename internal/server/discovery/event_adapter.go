package discovery

import (
	"context"

	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
)

// EventAdapter implements EventEmitter using the domain EventBus.
type EventAdapter struct {
	bus domain.EventBus
}

// NewEventAdapter creates an EventAdapter backed by the given event bus.
func NewEventAdapter(bus domain.EventBus) *EventAdapter {
	return &EventAdapter{bus: bus}
}

// EmitPatchDiscovered emits a patch.discovered domain event.
func (a *EventAdapter) EmitPatchDiscovered(ctx context.Context, tenantID, patchID, patchName, version, sourceRepo string) error {
	payload := map[string]any{
		"patch_name":  patchName,
		"version":     version,
		"source_repo": sourceRepo,
	}
	event := domain.NewSystemEvent(events.PatchDiscovered, tenantID, "patch", patchID, "discovered", payload)
	return a.bus.Emit(ctx, event)
}

// EmitRepositorySynced emits a repository.synced domain event.
func (a *EventAdapter) EmitRepositorySynced(ctx context.Context, tenantID, repoName string, patchCount int) error {
	payload := map[string]any{
		"repo_name":   repoName,
		"patch_count": patchCount,
	}
	event := domain.NewSystemEvent(events.RepositorySynced, tenantID, "repository", repoName, "synced", payload)
	return a.bus.Emit(ctx, event)
}
