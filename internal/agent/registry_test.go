package agent_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/skenzeriq/patchiq/internal/agent"
)

type mockModule struct {
	name         string
	version      string
	caps         []string
	commands     []string
	interval     time.Duration
	initCalled   bool
	startCalled  bool
	stopCalled   bool
	initOrder    int
	startOrder   int
	stopOrder    int
	initErr      error
	startErr     error
	collectItems []agent.OutboxItem
	collectErr   error
	orderCounter *int
}

func newMockModule(name string, counter *int) *mockModule {
	return &mockModule{
		name:         name,
		version:      "1.0.0",
		caps:         []string{name},
		commands:     []string{name + "_cmd"},
		interval:     5 * time.Minute,
		orderCounter: counter,
	}
}

func (m *mockModule) Name() string                        { return m.name }
func (m *mockModule) Version() string                     { return m.version }
func (m *mockModule) Capabilities() []string              { return m.caps }
func (m *mockModule) SupportedCommands() []string         { return m.commands }
func (m *mockModule) CollectInterval() time.Duration      { return m.interval }
func (m *mockModule) HealthCheck(_ context.Context) error { return nil }
func (m *mockModule) Collect(_ context.Context) ([]agent.OutboxItem, error) {
	return m.collectItems, m.collectErr
}

func (m *mockModule) Init(_ context.Context, _ agent.ModuleDeps) error {
	m.initCalled = true
	*m.orderCounter++
	m.initOrder = *m.orderCounter
	return m.initErr
}

func (m *mockModule) Start(_ context.Context) error {
	m.startCalled = true
	*m.orderCounter++
	m.startOrder = *m.orderCounter
	return m.startErr
}

func (m *mockModule) Stop(_ context.Context) error {
	m.stopCalled = true
	*m.orderCounter++
	m.stopOrder = *m.orderCounter
	return nil
}

func (m *mockModule) HandleCommand(_ context.Context, _ agent.Command) (agent.Result, error) {
	return agent.Result{Output: []byte("ok")}, nil
}

func TestRegistry_InitStartStop_Order(t *testing.T) {
	counter := 0
	modA := newMockModule("a", &counter)
	modB := newMockModule("b", &counter)

	reg := agent.NewRegistry(slog.Default())
	reg.Register(modA)
	reg.Register(modB)

	ctx := context.Background()
	if err := reg.InitAll(ctx, agent.ModuleDeps{}); err != nil {
		t.Fatalf("InitAll: %v", err)
	}
	if modA.initOrder >= modB.initOrder {
		t.Errorf("expected a.Init before b.Init, got a=%d b=%d", modA.initOrder, modB.initOrder)
	}

	if err := reg.StartAll(ctx); err != nil {
		t.Fatalf("StartAll: %v", err)
	}
	if modA.startOrder >= modB.startOrder {
		t.Errorf("expected a.Start before b.Start, got a=%d b=%d", modA.startOrder, modB.startOrder)
	}

	if err := reg.StopAll(ctx); err != nil {
		t.Fatalf("StopAll: %v", err)
	}
	if modB.stopOrder >= modA.stopOrder {
		t.Errorf("expected b.Stop before a.Stop, got a=%d b=%d", modA.stopOrder, modB.stopOrder)
	}
}

func TestRegistry_InitAll_FailureStopsRemaining(t *testing.T) {
	counter := 0
	modA := newMockModule("a", &counter)
	modA.initErr = errors.New("init failed")
	modB := newMockModule("b", &counter)

	reg := agent.NewRegistry(slog.Default())
	reg.Register(modA)
	reg.Register(modB)

	err := reg.InitAll(context.Background(), agent.ModuleDeps{})
	if err == nil {
		t.Fatal("expected error from InitAll")
	}
	if modB.initCalled {
		t.Error("module b should not have been initialized after a failed")
	}
}

func TestRegistry_Capabilities_Aggregated(t *testing.T) {
	counter := 0
	modA := newMockModule("inventory", &counter)
	modA.caps = []string{"inventory", "scan"}
	modB := newMockModule("patcher", &counter)
	modB.caps = []string{"patching"}

	reg := agent.NewRegistry(slog.Default())
	reg.Register(modA)
	reg.Register(modB)

	caps := reg.Capabilities()
	if len(caps) != 3 {
		t.Errorf("expected 3 capabilities, got %d: %v", len(caps), caps)
	}
}

func TestRegistry_HandleCommand_Dispatches(t *testing.T) {
	counter := 0
	modA := newMockModule("inventory", &counter)
	modA.commands = []string{"run_scan"}

	reg := agent.NewRegistry(slog.Default())
	reg.Register(modA)

	if err := reg.InitAll(context.Background(), agent.ModuleDeps{}); err != nil {
		t.Fatal(err)
	}

	result, err := reg.HandleCommand(context.Background(), agent.Command{ID: "cmd-1", Type: "run_scan"})
	if err != nil {
		t.Fatalf("HandleCommand: %v", err)
	}
	if string(result.Output) != "ok" {
		t.Errorf("expected 'ok', got %q", string(result.Output))
	}
}

func TestRegistry_HandleCommand_UnknownCommand(t *testing.T) {
	reg := agent.NewRegistry(slog.Default())
	_, err := reg.HandleCommand(context.Background(), agent.Command{ID: "cmd-1", Type: "unknown_cmd"})
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
}

func TestRegistry_Register_DuplicateName(t *testing.T) {
	counter := 0
	modA := newMockModule("inventory", &counter)
	modB := newMockModule("inventory", &counter)

	reg := agent.NewRegistry(slog.Default())
	reg.Register(modA)

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on duplicate module name")
		}
	}()
	reg.Register(modB)
}
