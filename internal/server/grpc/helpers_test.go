package grpc_test

import (
	"context"
	"sync"
	"testing"

	"github.com/skenzeriq/patchiq/internal/server/store"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
)

// testStore returns a Store with a nil pool for unit tests that only
// exercise pre-DB code paths.
func testStore(_ testing.TB) *store.Store { return store.NewStoreForTest() }

// noopEventBus is a test event bus that accepts and discards all events.
type noopEventBus struct{}

func (noopEventBus) Emit(context.Context, domain.DomainEvent) error { return nil }
func (noopEventBus) Subscribe(string, domain.EventHandler) error    { return nil }
func (noopEventBus) Close() error                                   { return nil }

// failingEventBus is a test event bus that returns an error on every Emit call.
type failingEventBus struct {
	err error
}

func (f failingEventBus) Emit(context.Context, domain.DomainEvent) error { return f.err }
func (failingEventBus) Subscribe(string, domain.EventHandler) error      { return nil }
func (failingEventBus) Close() error                                     { return nil }

// capturingEventBus records all emitted events for assertions.
type capturingEventBus struct {
	mu     sync.Mutex
	events []domain.DomainEvent
}

func (c *capturingEventBus) Emit(_ context.Context, evt domain.DomainEvent) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, evt)
	return nil
}
func (c *capturingEventBus) Subscribe(string, domain.EventHandler) error { return nil }
func (c *capturingEventBus) Close() error                                { return nil }

func (c *capturingEventBus) Events() []domain.DomainEvent {
	c.mu.Lock()
	defer c.mu.Unlock()
	copied := make([]domain.DomainEvent, len(c.events))
	copy(copied, c.events)
	return copied
}
