package inventory

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	"github.com/skenzeriq/patchiq/internal/agent"
	"google.golang.org/protobuf/proto"
)

// fakeExtCollector is a minimal packageCollector that also implements
// extendedCollector, so tests that exercise the cacheSaver path produce
// non-empty JSON from ExtendedPackagesJSON.
type fakeExtCollector struct{}

func (f *fakeExtCollector) Name() string { return "fake" }
func (f *fakeExtCollector) Collect(_ context.Context) ([]*pb.PackageInfo, error) {
	return []*pb.PackageInfo{{Name: "fake-pkg", Version: "1.0.0"}}, nil
}
func (f *fakeExtCollector) ExtendedPackages() []ExtendedPackageInfo {
	return []ExtendedPackageInfo{{Name: "fake-pkg", Version: "1.0.0"}}
}

// fakeOutbox captures Add calls for assertions.
type fakeOutbox struct {
	mu    sync.Mutex
	calls []fakeOutboxCall
}

type fakeOutboxCall struct {
	messageType string
	payload     []byte
}

func (f *fakeOutbox) Add(_ context.Context, messageType string, payload []byte) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := make([]byte, len(payload))
	copy(cp, payload)
	f.calls = append(f.calls, fakeOutboxCall{messageType: messageType, payload: cp})
	return int64(len(f.calls)), nil
}

func (f *fakeOutbox) Calls() []fakeOutboxCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]fakeOutboxCall, len(f.calls))
	copy(out, f.calls)
	return out
}

func TestInventoryModule_Identity(t *testing.T) {
	m := New()

	if m.Name() != "inventory" {
		t.Errorf("expected name 'inventory', got %q", m.Name())
	}
	if m.Version() != "0.2.0" {
		t.Errorf("expected version '0.2.0', got %q", m.Version())
	}

	caps := m.Capabilities()
	if len(caps) != 1 || caps[0] != "inventory" {
		t.Errorf("expected capabilities [inventory], got %v", caps)
	}

	cmds := m.SupportedCommands()
	if len(cmds) != 1 || cmds[0] != "run_scan" {
		t.Errorf("expected commands [run_scan], got %v", cmds)
	}
}

func TestInventoryModule_Lifecycle(t *testing.T) {
	m := New()
	ctx := context.Background()

	if err := m.Init(ctx, agent.ModuleDeps{}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := m.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}

func TestInventoryModule_Collect_ReturnsInventoryReport(t *testing.T) {
	m := New()
	ctx := context.Background()
	m.Init(ctx, agent.ModuleDeps{}) //nolint:errcheck

	items, err := m.Collect(ctx)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 outbox item, got %d", len(items))
	}
	if items[0].MessageType != "inventory" {
		t.Errorf("expected message type 'inventory', got %q", items[0].MessageType)
	}
	if len(items[0].Payload) == 0 {
		t.Error("expected non-empty payload")
	}
}

func TestInventoryModule_HandleCommand_RunScan(t *testing.T) {
	m := New()
	ctx := context.Background()
	m.Init(ctx, agent.ModuleDeps{}) //nolint:errcheck
	m.SetOutbox(&fakeOutbox{})

	result, err := m.HandleCommand(ctx, agent.Command{ID: "cmd-1", Type: "run_scan"})
	if err != nil {
		t.Fatalf("HandleCommand: %v", err)
	}
	if !bytes.Contains(result.Output, []byte("scan completed")) {
		t.Errorf("expected output to contain 'scan completed', got %q", string(result.Output))
	}
}

func TestInventoryModule_HandleCommand_UnknownType(t *testing.T) {
	m := New()
	ctx := context.Background()
	m.Init(ctx, agent.ModuleDeps{}) //nolint:errcheck

	_, err := m.HandleCommand(ctx, agent.Command{ID: "cmd-1", Type: "unknown"})
	if err == nil {
		t.Error("expected error for unknown command type")
	}
}

func TestInventoryModule_Collect_IncludesPackages(t *testing.T) {
	m := newModuleWithCollectors([]packageCollector{
		&aptCollector{statusPath: "testdata/dpkg_status_basic"},
	})
	ctx := context.Background()
	if err := m.Init(ctx, agent.ModuleDeps{}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	items, err := m.Collect(ctx)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	var report pb.InventoryReport
	if err := proto.Unmarshal(items[0].Payload, &report); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(report.InstalledPackages) != 4 {
		t.Errorf("expected 4 packages, got %d", len(report.InstalledPackages))
	}
}

func TestInventoryModule_Collect_PartialOnCollectorError(t *testing.T) {
	m := newModuleWithCollectors([]packageCollector{
		&aptCollector{statusPath: "testdata/dpkg_status_basic"},
		&aptCollector{statusPath: "testdata/nonexistent"}, // will error
	})
	ctx := context.Background()
	if err := m.Init(ctx, agent.ModuleDeps{}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	items, err := m.Collect(ctx)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	var report pb.InventoryReport
	if err := proto.Unmarshal(items[0].Payload, &report); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(report.InstalledPackages) != 4 {
		t.Errorf("expected 4 packages, got %d", len(report.InstalledPackages))
	}
	if len(report.CollectionErrors) != 1 {
		t.Errorf("expected 1 collection error, got %d", len(report.CollectionErrors))
	}
}

func TestInventoryModule_CollectInterval_Default24h(t *testing.T) {
	m := New()
	if m.CollectInterval() != 24*time.Hour {
		t.Errorf("expected 24h, got %v", m.CollectInterval())
	}
}

func TestHandleRunScanWithPayload(t *testing.T) {
	m := New()
	ctx := context.Background()
	m.Init(ctx, agent.ModuleDeps{}) //nolint:errcheck
	m.SetOutbox(&fakeOutbox{})

	payload := &pb.RunScanPayload{
		ScanType:        pb.ScanType_SCAN_TYPE_QUICK,
		CheckCategories: []string{"os_packages", "security_updates"},
	}
	raw, err := proto.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	result, err := m.HandleCommand(ctx, agent.Command{
		ID:      "cmd-scan-1",
		Type:    "run_scan",
		Payload: raw,
	})
	if err != nil {
		t.Fatalf("HandleCommand: %v", err)
	}
	if !bytes.Contains(result.Output, []byte("scan completed")) {
		t.Errorf("expected 'scan completed' in output, got %q", string(result.Output))
	}
}

func TestHandleRunScanEmptyPayload(t *testing.T) {
	m := New()
	ctx := context.Background()
	m.Init(ctx, agent.ModuleDeps{}) //nolint:errcheck
	m.SetOutbox(&fakeOutbox{})

	// nil payload — backward compatible, should still produce output.
	result, err := m.HandleCommand(ctx, agent.Command{
		ID:      "cmd-scan-2",
		Type:    "run_scan",
		Payload: nil,
	})
	if err != nil {
		t.Fatalf("HandleCommand with nil payload: %v", err)
	}
	if !bytes.Contains(result.Output, []byte("scan completed")) {
		t.Errorf("expected 'scan completed' in output, got %q", string(result.Output))
	}

	// Also test empty (zero-length) payload.
	result2, err := m.HandleCommand(ctx, agent.Command{
		ID:      "cmd-scan-3",
		Type:    "run_scan",
		Payload: []byte{},
	})
	if err != nil {
		t.Fatalf("HandleCommand with empty payload: %v", err)
	}
	if !bytes.Contains(result2.Output, []byte("scan completed")) {
		t.Errorf("expected 'scan completed' in output, got %q", string(result2.Output))
	}
}

func TestHandleRunScanInvalidPayload(t *testing.T) {
	m := New()
	ctx := context.Background()
	m.Init(ctx, agent.ModuleDeps{}) //nolint:errcheck
	m.SetOutbox(&fakeOutbox{})

	// Garbage payload — should log warning and still produce output (not error).
	result, err := m.HandleCommand(ctx, agent.Command{
		ID:      "cmd-scan-4",
		Type:    "run_scan",
		Payload: []byte("this-is-not-valid-protobuf-data!!!"),
	})
	if err != nil {
		t.Fatalf("HandleCommand with invalid payload should not error, got: %v", err)
	}
	if !bytes.Contains(result.Output, []byte("scan completed")) {
		t.Errorf("expected 'scan completed' in output, got %q", string(result.Output))
	}
}

func TestHandleRunScan_WritesInventoryToOutbox(t *testing.T) {
	m := newModuleWithCollectors([]packageCollector{
		&aptCollector{statusPath: "testdata/dpkg_status_basic"},
	})
	ctx := context.Background()
	if err := m.Init(ctx, agent.ModuleDeps{}); err != nil {
		t.Fatalf("Init: %v", err)
	}

	ob := &fakeOutbox{}
	m.SetOutbox(ob)

	result, err := m.HandleCommand(ctx, agent.Command{ID: "cmd-1", Type: "run_scan"})
	if err != nil {
		t.Fatalf("HandleCommand: %v", err)
	}
	if !bytes.Contains(result.Output, []byte("scan completed")) {
		t.Errorf("expected 'scan completed' in output, got %q", string(result.Output))
	}

	calls := ob.Calls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 outbox call, got %d", len(calls))
	}
	if calls[0].messageType != "inventory" {
		t.Errorf("expected message type 'inventory', got %q", calls[0].messageType)
	}
	if len(calls[0].payload) == 0 {
		t.Error("expected non-empty payload in outbox call")
	}

	// Payload must be a valid InventoryReport.
	var report pb.InventoryReport
	if err := proto.Unmarshal(calls[0].payload, &report); err != nil {
		t.Fatalf("unmarshal outbox payload as InventoryReport: %v", err)
	}
	if report.EndpointInfo == nil {
		t.Error("expected EndpointInfo in outbox payload")
	}
}

func TestHandleRunScan_CallsCacheSaver(t *testing.T) {
	m := newModuleWithCollectors([]packageCollector{&fakeExtCollector{}})
	ctx := context.Background()
	if err := m.Init(ctx, agent.ModuleDeps{}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	m.SetOutbox(&fakeOutbox{})

	var (
		mu          sync.Mutex
		cacheCalled bool
	)
	m.SetCacheSaver(func(_ context.Context, _ []byte) error {
		mu.Lock()
		defer mu.Unlock()
		cacheCalled = true
		return nil
	})

	if _, err := m.HandleCommand(ctx, agent.Command{ID: "cmd-1", Type: "run_scan"}); err != nil {
		t.Fatalf("HandleCommand: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if !cacheCalled {
		t.Error("expected cacheSaver to be called")
	}
}

func TestHandleRunScan_NoOutbox_ReturnsError(t *testing.T) {
	m := newModuleWithCollectors([]packageCollector{
		&aptCollector{statusPath: "testdata/dpkg_status_basic"},
	})
	ctx := context.Background()
	if err := m.Init(ctx, agent.ModuleDeps{}); err != nil {
		t.Fatalf("Init: %v", err)
	}

	// No SetOutbox call — run_scan MUST return an error so the caller sees
	// the misconfiguration rather than silently dropping inventory.
	_, err := m.HandleCommand(ctx, agent.Command{ID: "cmd-1", Type: "run_scan"})
	if err == nil {
		t.Fatal("HandleCommand with no outbox: expected error, got nil")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("outbox not configured")) {
		t.Errorf("error = %q, want to mention 'outbox not configured'", err)
	}
}
