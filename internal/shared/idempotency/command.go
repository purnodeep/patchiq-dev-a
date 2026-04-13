package idempotency

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// CommandResult holds the serialized output of an executed agent command.
type CommandResult struct {
	Data  []byte `json:"data"`
	Error string `json:"error,omitempty"`
}

// CommandStore persists command execution records so duplicate deliveries can
// be detected and short-circuited.
type CommandStore interface {
	HasExecuted(ctx context.Context, commandID string) (CommandResult, bool, error)
	MarkExecuted(ctx context.Context, commandID string, result CommandResult) error
}

// Deduplicator ensures each command identified by a unique commandID is
// executed exactly once, even when the same command is delivered multiple
// times by the gRPC streaming layer.
type Deduplicator struct {
	store CommandStore
}

// NewDeduplicator returns a Deduplicator backed by the provided store.
func NewDeduplicator(store CommandStore) *Deduplicator {
	return &Deduplicator{store: store}
}

// Execute runs handler for commandID unless a cached result already exists in
// the store. When the handler returns an error the result is still persisted
// (with Error set) so subsequent duplicate calls return the cached result
// without re-executing the handler.
func (d *Deduplicator) Execute(ctx context.Context, commandID string, handler func(ctx context.Context) (CommandResult, error)) (CommandResult, error) {
	cached, found, err := d.store.HasExecuted(ctx, commandID)
	if err != nil {
		return CommandResult{}, fmt.Errorf("check command %q: %w", commandID, err)
	}
	if found {
		slog.InfoContext(ctx, "command already executed", "command_id", commandID)
		return cached, nil
	}

	result, handlerErr := handler(ctx)
	if handlerErr != nil {
		result.Error = handlerErr.Error()
	}

	if markErr := d.store.MarkExecuted(ctx, commandID, result); markErr != nil {
		return CommandResult{}, fmt.Errorf("persist command result %q: %w", commandID, markErr)
	}

	if handlerErr != nil {
		return result, handlerErr
	}
	return result, nil
}

// MemoryCommandStore is an in-memory CommandStore suitable for tests and the
// agent's ephemeral use-case where persistence across restarts is not required.
type MemoryCommandStore struct {
	mu    sync.RWMutex
	items map[string]CommandResult
}

// NewMemoryCommandStore returns an initialised MemoryCommandStore.
func NewMemoryCommandStore() *MemoryCommandStore {
	return &MemoryCommandStore{items: make(map[string]CommandResult)}
}

// HasExecuted returns the stored result for commandID if it exists.
func (m *MemoryCommandStore) HasExecuted(_ context.Context, commandID string) (CommandResult, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.items[commandID]
	return r, ok, nil
}

// MarkExecuted stores result for commandID.
func (m *MemoryCommandStore) MarkExecuted(_ context.Context, commandID string, result CommandResult) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[commandID] = result
	return nil
}
