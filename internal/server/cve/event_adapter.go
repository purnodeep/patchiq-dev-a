package cve

import (
	"context"
	"fmt"
	"time"

	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
)

// EventAdapter emits CVE-related domain events via the shared EventBus.
// TODO(PIQ-107): Task 10 will flesh out this implementation.
type EventAdapter struct {
	bus domain.EventBus
}

// NewEventAdapter creates an EventAdapter backed by the given EventBus.
func NewEventAdapter(bus domain.EventBus) *EventAdapter {
	return &EventAdapter{bus: bus}
}

// EmitCVEDiscovered emits a cve.discovered event.
func (a *EventAdapter) EmitCVEDiscovered(ctx context.Context, tenantID, cveDBID, cveID, severity string, cvss float64) error {
	event := domain.DomainEvent{
		ID:         domain.NewEventID(),
		Type:       events.CVEDiscovered,
		TenantID:   tenantID,
		ActorID:    "system",
		ActorType:  domain.ActorSystem,
		Resource:   "cve",
		ResourceID: cveDBID,
		Action:     "discovered",
		Payload: map[string]any{
			"cve_id":   cveID,
			"severity": severity,
			"cvss":     cvss,
		},
		Timestamp: time.Now(),
	}
	if err := a.bus.Emit(ctx, event); err != nil {
		return fmt.Errorf("emit cve.discovered: %w", err)
	}
	return nil
}

// EmitCVELinkedToEndpoint emits a cve.linked_to_endpoint event.
func (a *EventAdapter) EmitCVELinkedToEndpoint(ctx context.Context, tenantID, endpointID, cveID string, riskScore float64) error {
	event := domain.DomainEvent{
		ID:         domain.NewEventID(),
		Type:       events.CVELinkedToEndpoint,
		TenantID:   tenantID,
		ActorID:    "system",
		ActorType:  domain.ActorSystem,
		Resource:   "endpoint_cve",
		ResourceID: endpointID,
		Action:     "linked",
		Payload: map[string]any{
			"cve_id":      cveID,
			"endpoint_id": endpointID,
			"risk_score":  riskScore,
		},
		Timestamp: time.Now(),
	}
	if err := a.bus.Emit(ctx, event); err != nil {
		return fmt.Errorf("emit cve.linked_to_endpoint: %w", err)
	}
	return nil
}

// EmitCVERemediationAvailable emits a cve.remediation_available event.
func (a *EventAdapter) EmitCVERemediationAvailable(ctx context.Context, tenantID, cveID, patchID, packageName string) error {
	event := domain.DomainEvent{
		ID:         domain.NewEventID(),
		Type:       events.CVERemediationAvailable,
		TenantID:   tenantID,
		ActorID:    "system",
		ActorType:  domain.ActorSystem,
		Resource:   "cve",
		ResourceID: cveID,
		Action:     "remediation_available",
		Payload: map[string]any{
			"cve_id":       cveID,
			"patch_id":     patchID,
			"package_name": packageName,
		},
		Timestamp: time.Now(),
	}
	if err := a.bus.Emit(ctx, event); err != nil {
		return fmt.Errorf("emit cve.remediation_available: %w", err)
	}
	return nil
}
