package agent_test

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/skenzeriq/patchiq/internal/agent"
)

// runnerMockModule is a test double whose Collect behavior can change per call.
type runnerMockModule struct {
	name     string
	interval time.Duration

	mu       sync.Mutex
	callNum  int
	collectF func(callNum int) ([]agent.OutboxItem, error)
}

func (m *runnerMockModule) Name() string                { return m.name }
func (m *runnerMockModule) Version() string             { return "1.0.0" }
func (m *runnerMockModule) Capabilities() []string      { return nil }
func (m *runnerMockModule) SupportedCommands() []string { return nil }
func (m *runnerMockModule) CollectInterval() time.Duration {
	return m.interval
}

func (m *runnerMockModule) Init(_ context.Context, _ agent.ModuleDeps) error { return nil }
func (m *runnerMockModule) Start(_ context.Context) error                    { return nil }
func (m *runnerMockModule) Stop(_ context.Context) error                     { return nil }
func (m *runnerMockModule) HealthCheck(_ context.Context) error              { return nil }

func (m *runnerMockModule) HandleCommand(_ context.Context, _ agent.Command) (agent.Result, error) {
	return agent.Result{}, nil
}

func (m *runnerMockModule) Collect(_ context.Context) ([]agent.OutboxItem, error) {
	m.mu.Lock()
	m.callNum++
	n := m.callNum
	m.mu.Unlock()
	return m.collectF(n)
}

// mockOutboxWriter records all Add calls.
type mockOutboxWriter struct {
	mu    sync.Mutex
	items []outboxEntry
	seq   atomic.Int64
}

type outboxEntry struct {
	messageType string
	payload     []byte
}

func (w *mockOutboxWriter) Add(_ context.Context, messageType string, payload []byte) (int64, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.items = append(w.items, outboxEntry{messageType: messageType, payload: payload})
	return w.seq.Add(1), nil
}

func (w *mockOutboxWriter) count() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.items)
}

func TestCollectionRunner_RunsModuleAndWritesToOutbox(t *testing.T) {
	t.Parallel()

	mod := &runnerMockModule{
		name:     "test-mod",
		interval: 50 * time.Millisecond,
		collectF: func(_ int) ([]agent.OutboxItem, error) {
			return []agent.OutboxItem{
				{MessageType: "inventory.packages", Payload: []byte(`{"pkg":"vim"}`)},
			}, nil
		},
	}

	outbox := &mockOutboxWriter{}
	runner := agent.NewCollectionRunner([]agent.Module{mod}, outbox, slog.Default())

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	runner.Run(ctx)

	if got := outbox.count(); got < 1 {
		t.Errorf("expected at least 1 outbox item, got %d", got)
	}
}

func TestCollectionRunner_ContinuesOnCollectError(t *testing.T) {
	t.Parallel()

	mod := &runnerMockModule{
		name:     "flaky-mod",
		interval: 50 * time.Millisecond,
		collectF: func(callNum int) ([]agent.OutboxItem, error) {
			if callNum == 1 {
				return nil, errors.New("transient failure")
			}
			return []agent.OutboxItem{
				{MessageType: "inventory.packages", Payload: []byte(`{"pkg":"curl"}`)},
			}, nil
		},
	}

	outbox := &mockOutboxWriter{}
	runner := agent.NewCollectionRunner([]agent.Module{mod}, outbox, slog.Default())

	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	runner.Run(ctx)

	if got := outbox.count(); got < 1 {
		t.Errorf("expected at least 1 outbox item after recovery, got %d", got)
	}
}

func TestCollectionRunner_MultipleModules(t *testing.T) {
	t.Parallel()

	modA := &runnerMockModule{
		name:     "mod-a",
		interval: 50 * time.Millisecond,
		collectF: func(_ int) ([]agent.OutboxItem, error) {
			return []agent.OutboxItem{
				{MessageType: "type-a", Payload: []byte(`a`)},
			}, nil
		},
	}
	modB := &runnerMockModule{
		name:     "mod-b",
		interval: 50 * time.Millisecond,
		collectF: func(_ int) ([]agent.OutboxItem, error) {
			return []agent.OutboxItem{
				{MessageType: "type-b", Payload: []byte(`b`)},
			}, nil
		},
	}

	outbox := &mockOutboxWriter{}
	runner := agent.NewCollectionRunner(
		[]agent.Module{modA, modB}, outbox, slog.Default(),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	runner.Run(ctx)

	outbox.mu.Lock()
	defer outbox.mu.Unlock()

	hasA, hasB := false, false
	for _, item := range outbox.items {
		if item.messageType == "type-a" {
			hasA = true
		}
		if item.messageType == "type-b" {
			hasB = true
		}
	}
	if !hasA {
		t.Error("expected outbox to contain items from mod-a")
	}
	if !hasB {
		t.Error("expected outbox to contain items from mod-b")
	}
}

func TestCollectionRunner_CollectNow_Found(t *testing.T) {
	t.Parallel()

	mod := &runnerMockModule{
		name:     "inventory",
		interval: 1 * time.Hour,
		collectF: func(_ int) ([]agent.OutboxItem, error) {
			return []agent.OutboxItem{
				{MessageType: "inventory.packages", Payload: []byte(`{"pkg":"vim"}`)},
			}, nil
		},
	}

	outbox := &mockOutboxWriter{}
	runner := agent.NewCollectionRunner([]agent.Module{mod}, outbox, slog.Default())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := runner.CollectNow(ctx, "inventory"); err != nil {
		t.Fatalf("CollectNow returned error: %v", err)
	}

	mod.mu.Lock()
	calls := mod.callNum
	mod.mu.Unlock()
	if calls != 1 {
		t.Errorf("expected Collect to be called 1 time, got %d", calls)
	}
	if got := outbox.count(); got != 1 {
		t.Errorf("expected 1 outbox item, got %d", got)
	}
}

func TestCollectionRunner_CollectNow_NotFound(t *testing.T) {
	t.Parallel()

	mod := &runnerMockModule{
		name:     "inventory",
		interval: 1 * time.Hour,
		collectF: func(_ int) ([]agent.OutboxItem, error) {
			return nil, nil
		},
	}

	outbox := &mockOutboxWriter{}
	runner := agent.NewCollectionRunner([]agent.Module{mod}, outbox, slog.Default())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := runner.CollectNow(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown module, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected error to contain 'not found', got %v", err)
	}

	mod.mu.Lock()
	calls := mod.callNum
	mod.mu.Unlock()
	if calls != 0 {
		t.Errorf("expected Collect to NOT be called, got %d", calls)
	}
}
