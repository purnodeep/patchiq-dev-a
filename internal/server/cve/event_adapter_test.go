package cve

import (
	"context"
	"testing"

	"github.com/skenzeriq/patchiq/internal/shared/domain"
)

type mockEventBus struct {
	emitted []domain.DomainEvent
}

func (m *mockEventBus) Emit(_ context.Context, event domain.DomainEvent) error {
	m.emitted = append(m.emitted, event)
	return nil
}

func (m *mockEventBus) Subscribe(_ string, _ domain.EventHandler) error { return nil }

func (m *mockEventBus) Close() error { return nil }

func TestEventAdapter_EmitCVEDiscovered(t *testing.T) {
	bus := &mockEventBus{}
	adapter := NewEventAdapter(bus)
	err := adapter.EmitCVEDiscovered(context.Background(), "tenant-1", "db-id-1", "CVE-2024-1234", "CRITICAL", 9.8)
	if err != nil {
		t.Fatalf("EmitCVEDiscovered: %v", err)
	}
	if len(bus.emitted) != 1 {
		t.Fatalf("expected 1 event, got %d", len(bus.emitted))
	}
	if bus.emitted[0].Type != "cve.discovered" {
		t.Errorf("Type = %q, want cve.discovered", bus.emitted[0].Type)
	}
}

func TestEventAdapter_EmitCVELinkedToEndpoint(t *testing.T) {
	bus := &mockEventBus{}
	adapter := NewEventAdapter(bus)
	err := adapter.EmitCVELinkedToEndpoint(context.Background(), "tenant-1", "endpoint-1", "CVE-2024-1234", 9.8)
	if err != nil {
		t.Fatalf("EmitCVELinkedToEndpoint: %v", err)
	}
	if len(bus.emitted) != 1 {
		t.Fatalf("expected 1 event, got %d", len(bus.emitted))
	}
	if bus.emitted[0].Type != "cve.linked_to_endpoint" {
		t.Errorf("Type = %q, want cve.linked_to_endpoint", bus.emitted[0].Type)
	}
}

func TestEventAdapter_EmitCVERemediationAvailable(t *testing.T) {
	bus := &mockEventBus{}
	adapter := NewEventAdapter(bus)
	err := adapter.EmitCVERemediationAvailable(context.Background(), "tenant-1", "CVE-2024-1234", "patch-1", "openssl")
	if err != nil {
		t.Fatalf("EmitCVERemediationAvailable: %v", err)
	}
	if len(bus.emitted) != 1 {
		t.Fatalf("expected 1 event, got %d", len(bus.emitted))
	}
	if bus.emitted[0].Type != "cve.remediation_available" {
		t.Errorf("Type = %q, want cve.remediation_available", bus.emitted[0].Type)
	}
}
