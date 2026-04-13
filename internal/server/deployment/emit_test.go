package deployment_test

import (
	"context"
	"errors"
	"testing"

	"github.com/skenzeriq/patchiq/internal/server/deployment"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
)

func TestEmitBestEffort_EmitsAllEvents(t *testing.T) {
	t.Parallel()
	bus := &fakeEventBus{}
	events := []domain.DomainEvent{
		domain.NewSystemEvent("test.one", "t1", "res", "r1", "action", nil),
		domain.NewSystemEvent("test.two", "t1", "res", "r2", "action", nil),
	}

	deployment.EmitBestEffort(context.Background(), bus, events)

	if len(bus.events) != 2 {
		t.Fatalf("expected 2 emitted events, got %d", len(bus.events))
	}
	if bus.events[0].Type != "test.one" {
		t.Errorf("first event type = %q, want %q", bus.events[0].Type, "test.one")
	}
	if bus.events[1].Type != "test.two" {
		t.Errorf("second event type = %q, want %q", bus.events[1].Type, "test.two")
	}
}

func TestEmitBestEffort_EmptySlice_NoOp(t *testing.T) {
	t.Parallel()
	bus := &fakeEventBus{}

	deployment.EmitBestEffort(context.Background(), bus, nil)

	if len(bus.events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(bus.events))
	}
}

func TestEmitBestEffort_NilBus_DoesNotPanic(t *testing.T) {
	t.Parallel()
	events := []domain.DomainEvent{
		domain.NewSystemEvent("test.one", "t1", "res", "r1", "action", nil),
	}

	// Should not panic with nil bus — logs and returns.
	deployment.EmitBestEffort(context.Background(), nil, events)
}

func TestEmitBestEffort_EmitFailure_ContinuesAndDoesNotPanic(t *testing.T) {
	t.Parallel()
	bus := &fakeEventBus{emitErr: errors.New("bus down")}
	events := []domain.DomainEvent{
		domain.NewSystemEvent("test.one", "t1", "res", "r1", "action", nil),
		domain.NewSystemEvent("test.two", "t1", "res", "r2", "action", nil),
	}

	// Should not panic and should attempt all events despite errors.
	deployment.EmitBestEffort(context.Background(), bus, events)

	// fakeEventBus still appends even when returning error.
	if len(bus.events) != 2 {
		t.Fatalf("expected 2 emit attempts, got %d", len(bus.events))
	}
}
