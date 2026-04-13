package patcher

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"os"
	"testing"
	"time"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	"github.com/skenzeriq/patchiq/internal/agent"
	"github.com/skenzeriq/patchiq/internal/agent/store"
	"google.golang.org/protobuf/proto"
	_ "modernc.org/sqlite"
)

func TestModule_Metadata(t *testing.T) {
	m := New()
	if m.Name() != "patcher" {
		t.Errorf("Name() = %q, want %q", m.Name(), "patcher")
	}
	wantCmds := []string{"install_patch", "rollback_patch"}
	gotCmds := m.SupportedCommands()
	if len(gotCmds) != len(wantCmds) {
		t.Errorf("SupportedCommands() = %v, want %v", gotCmds, wantCmds)
	} else {
		for i, want := range wantCmds {
			if gotCmds[i] != want {
				t.Errorf("SupportedCommands()[%d] = %q, want %q", i, gotCmds[i], want)
			}
		}
	}
	if m.CollectInterval() != 0 {
		t.Errorf("CollectInterval() = %v, want 0", m.CollectInterval())
	}
}

func TestModule_HandleCommand_success(t *testing.T) {
	payload := &pb.InstallPatchPayload{
		Packages: []*pb.PatchTarget{{Name: "curl", Version: "7.88.1"}},
	}
	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}

	mockInst := &mockInstaller{
		name: "apt",
		fn: func(_ context.Context, pkg PatchTarget, _ bool) (InstallResult, error) {
			return InstallResult{Stdout: []byte("installed " + pkg.Name), ExitCode: 0}, nil
		},
	}

	m := newTestModule(mockInst, nil)
	result, err := m.HandleCommand(context.Background(), agent.Command{ID: "cmd-1", Type: "install_patch", Payload: payloadBytes})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ErrorMessage != "" {
		t.Errorf("error message = %q, want empty", result.ErrorMessage)
	}

	var output pb.InstallPatchOutput
	if err := proto.Unmarshal(result.Output, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if len(output.Results) != 1 {
		t.Fatalf("results count = %d, want 1", len(output.Results))
	}
	if !output.Results[0].Succeeded {
		t.Error("expected succeeded = true")
	}
}

func TestModule_HandleCommand_install_failure(t *testing.T) {
	payload := &pb.InstallPatchPayload{
		Packages: []*pb.PatchTarget{{Name: "badpkg", Version: "1.0"}},
	}
	payloadBytes, _ := proto.Marshal(payload)

	mockInst := &mockInstaller{
		name: "apt",
		fn: func(_ context.Context, _ PatchTarget, _ bool) (InstallResult, error) {
			return InstallResult{Stderr: []byte("not found"), ExitCode: 100}, nil
		},
	}

	m := newTestModule(mockInst, nil)
	result, err := m.HandleCommand(context.Background(), agent.Command{ID: "cmd-2", Type: "install_patch", Payload: payloadBytes})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var output pb.InstallPatchOutput
	if err := proto.Unmarshal(result.Output, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if output.Results[0].Succeeded {
		t.Error("expected succeeded = false")
	}
	if output.Results[0].ExitCode != 100 {
		t.Errorf("exit code = %d, want 100", output.Results[0].ExitCode)
	}
}

func TestModule_HandleCommand_prescript_failure(t *testing.T) {
	payload := &pb.InstallPatchPayload{
		Packages:  []*pb.PatchTarget{{Name: "curl", Version: "1.0"}},
		PreScript: "exit 1",
	}
	payloadBytes, _ := proto.Marshal(payload)

	installCalled := false
	mockInst := &mockInstaller{
		name: "apt",
		fn: func(_ context.Context, _ PatchTarget, _ bool) (InstallResult, error) {
			installCalled = true
			return InstallResult{ExitCode: 0}, nil
		},
	}

	mockExec := &mockExecutor{
		fn: func(_ context.Context, name string, args ...string) (ExecResult, error) {
			return ExecResult{ExitCode: 1, Stderr: []byte("pre-script failed")}, nil
		},
	}

	m := newTestModule(mockInst, mockExec)
	result, err := m.HandleCommand(context.Background(), agent.Command{ID: "cmd-3", Type: "install_patch", Payload: payloadBytes})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if installCalled {
		t.Error("install should not be called when pre-script fails")
	}
	if result.ErrorMessage == "" {
		t.Error("expected error message for pre-script failure")
	}
}

func TestModule_HandleCommand_mutex(t *testing.T) {
	payload := &pb.InstallPatchPayload{
		Packages: []*pb.PatchTarget{{Name: "curl", Version: "1.0"}},
	}
	payloadBytes, _ := proto.Marshal(payload)

	started := make(chan struct{})
	block := make(chan struct{})
	mockInst := &mockInstaller{
		name: "apt",
		fn: func(_ context.Context, _ PatchTarget, _ bool) (InstallResult, error) {
			started <- struct{}{}
			<-block
			return InstallResult{ExitCode: 0}, nil
		},
	}

	m := newTestModule(mockInst, nil)

	go func() {
		_, _ = m.HandleCommand(context.Background(), agent.Command{ID: "cmd-a", Type: "install_patch", Payload: payloadBytes})
	}()

	<-started

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := m.HandleCommand(ctx, agent.Command{ID: "cmd-b", Type: "install_patch", Payload: payloadBytes})
	if err == nil {
		t.Error("expected error due to context timeout while waiting for mutex")
	}

	close(block)
}

func TestModule_HandleCommand_unsupported(t *testing.T) {
	m := New()
	_, err := m.HandleCommand(context.Background(), agent.Command{Type: "unknown"})
	if err == nil {
		t.Error("expected error for unsupported command")
	}
}

func TestModule_HandleCommand_invalid_payload(t *testing.T) {
	mockInst := &mockInstaller{name: "apt", fn: func(_ context.Context, _ PatchTarget, _ bool) (InstallResult, error) {
		t.Fatal("should not be called")
		return InstallResult{}, nil
	}}
	m := newTestModule(mockInst, nil)
	result, err := m.HandleCommand(context.Background(), agent.Command{
		ID: "cmd-bad", Type: "install_patch", Payload: []byte("not-a-protobuf"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ErrorMessage == "" {
		t.Error("expected error message for invalid payload")
	}
}

func TestModule_HandleCommand_empty_packages(t *testing.T) {
	payload := &pb.InstallPatchPayload{Packages: []*pb.PatchTarget{}}
	payloadBytes, _ := proto.Marshal(payload)

	installCalled := false
	mockInst := &mockInstaller{name: "apt", fn: func(_ context.Context, _ PatchTarget, _ bool) (InstallResult, error) {
		installCalled = true
		return InstallResult{}, nil
	}}

	m := newTestModule(mockInst, nil)
	result, err := m.HandleCommand(context.Background(), agent.Command{ID: "cmd-empty", Type: "install_patch", Payload: payloadBytes})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if installCalled {
		t.Error("install should not be called for empty package list")
	}
	if result.ErrorMessage != "" {
		t.Errorf("unexpected error message: %s", result.ErrorMessage)
	}
}

func TestModule_HandleCommand_dryrun_propagated(t *testing.T) {
	payload := &pb.InstallPatchPayload{
		Packages: []*pb.PatchTarget{{Name: "curl", Version: "1.0"}},
		DryRun:   true,
	}
	payloadBytes, _ := proto.Marshal(payload)

	var gotDryRun bool
	mockInst := &mockInstaller{name: "apt", fn: func(_ context.Context, _ PatchTarget, dryRun bool) (InstallResult, error) {
		gotDryRun = dryRun
		return InstallResult{ExitCode: 0}, nil
	}}

	m := newTestModule(mockInst, nil)
	_, _ = m.HandleCommand(context.Background(), agent.Command{ID: "cmd-dry", Type: "install_patch", Payload: payloadBytes})
	if !gotDryRun {
		t.Error("expected dry_run=true to be propagated to installer")
	}
}

func TestModule_HandleCommand_postscript_runs_after_failure(t *testing.T) {
	payload := &pb.InstallPatchPayload{
		Packages:   []*pb.PatchTarget{{Name: "badpkg", Version: "1.0"}},
		PostScript: "echo cleanup",
	}
	payloadBytes, _ := proto.Marshal(payload)

	mockInst := &mockInstaller{name: "apt", fn: func(_ context.Context, _ PatchTarget, _ bool) (InstallResult, error) {
		return InstallResult{ExitCode: 1, Stderr: []byte("failed")}, nil
	}}

	postScriptRan := false
	mockExec := &mockExecutor{fn: func(_ context.Context, name string, args ...string) (ExecResult, error) {
		postScriptRan = true
		return ExecResult{Stdout: []byte("cleanup done"), ExitCode: 0}, nil
	}}

	m := newTestModule(mockInst, mockExec)
	_, _ = m.HandleCommand(context.Background(), agent.Command{ID: "cmd-post", Type: "install_patch", Payload: payloadBytes})
	if !postScriptRan {
		t.Error("post-script should run even when install fails")
	}
}

func TestModule_HandleCommand_postscript_failure_sets_error(t *testing.T) {
	payload := &pb.InstallPatchPayload{
		Packages:   []*pb.PatchTarget{{Name: "curl", Version: "1.0"}},
		PostScript: "echo failing",
	}
	payloadBytes, _ := proto.Marshal(payload)

	mockInst := &mockInstaller{name: "apt", fn: func(_ context.Context, _ PatchTarget, _ bool) (InstallResult, error) {
		return InstallResult{ExitCode: 0, Stdout: []byte("ok")}, nil
	}}

	mockExec := &mockExecutor{fn: func(_ context.Context, name string, args ...string) (ExecResult, error) {
		return ExecResult{Stdout: []byte("post failed"), ExitCode: 1}, nil
	}}

	m := newTestModule(mockInst, mockExec)
	result, err := m.HandleCommand(context.Background(), agent.Command{ID: "cmd-postfail", Type: "install_patch", Payload: payloadBytes})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ErrorMessage == "" {
		t.Error("expected error message when post-script fails")
	}
}

func TestModule_HandleCommand_no_installer(t *testing.T) {
	m := newWithMaxFunc(func() int { return 1 })
	m.logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	m.installers = map[string]Installer{}
	payload := &pb.InstallPatchPayload{Packages: []*pb.PatchTarget{{Name: "curl", Version: "1.0"}}}
	payloadBytes, _ := proto.Marshal(payload)

	result, err := m.HandleCommand(context.Background(), agent.Command{ID: "cmd-noinst", Type: "install_patch", Payload: payloadBytes})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ErrorMessage == "" {
		t.Error("expected error message when no installer available")
	}
}

func TestModule_HandleCommand_multiple_packages(t *testing.T) {
	payload := &pb.InstallPatchPayload{
		Packages: []*pb.PatchTarget{
			{Name: "curl", Version: "7.88"},
			{Name: "wget", Version: "1.21"},
			{Name: "openssl", Version: "3.0"},
		},
	}
	payloadBytes, _ := proto.Marshal(payload)

	var installed []string
	mockInst := &mockInstaller{name: "apt", fn: func(_ context.Context, pkg PatchTarget, _ bool) (InstallResult, error) {
		installed = append(installed, pkg.Name)
		return InstallResult{ExitCode: 0}, nil
	}}

	m := newTestModule(mockInst, nil)
	result, err := m.HandleCommand(context.Background(), agent.Command{ID: "cmd-multi", Type: "install_patch", Payload: payloadBytes})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ErrorMessage != "" {
		t.Errorf("unexpected error: %s", result.ErrorMessage)
	}
	if len(installed) != 3 {
		t.Errorf("installed %d packages, want 3", len(installed))
	}

	var output pb.InstallPatchOutput
	if err := proto.Unmarshal(result.Output, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if len(output.Results) != 3 {
		t.Errorf("output results = %d, want 3", len(output.Results))
	}
}

func TestModule_HandleCommand_partial_failure(t *testing.T) {
	payload := &pb.InstallPatchPayload{
		Packages: []*pb.PatchTarget{
			{Name: "good", Version: "1.0"},
			{Name: "bad", Version: "1.0"},
			{Name: "good2", Version: "1.0"},
		},
	}
	payloadBytes, _ := proto.Marshal(payload)

	mockInst := &mockInstaller{name: "apt", fn: func(_ context.Context, pkg PatchTarget, _ bool) (InstallResult, error) {
		if pkg.Name == "bad" {
			return InstallResult{ExitCode: 1, Stderr: []byte("not found")}, nil
		}
		return InstallResult{ExitCode: 0}, nil
	}}

	m := newTestModule(mockInst, nil)
	result, _ := m.HandleCommand(context.Background(), agent.Command{ID: "cmd-partial", Type: "install_patch", Payload: payloadBytes})

	if result.ErrorMessage == "" {
		t.Error("expected error message for partial failure")
	}

	var output pb.InstallPatchOutput
	if err := proto.Unmarshal(result.Output, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if len(output.Results) != 3 {
		t.Fatalf("results = %d, want 3", len(output.Results))
	}
	if !output.Results[0].Succeeded {
		t.Error("first package should succeed")
	}
	if output.Results[1].Succeeded {
		t.Error("second package should fail")
	}
	if !output.Results[2].Succeeded {
		t.Error("third package should succeed (continues after failure)")
	}
}

func TestMarshalOutput_success(t *testing.T) {
	msg := &pb.InstallPatchOutput{
		Results: []*pb.InstallResultDetail{
			{PackageName: "curl", Succeeded: true},
		},
	}
	data, err := marshalOutput(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty output")
	}

	var decoded pb.InstallPatchOutput
	if err := proto.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("round-trip unmarshal failed: %v", err)
	}
	if len(decoded.Results) != 1 || decoded.Results[0].PackageName != "curl" {
		t.Errorf("round-trip mismatch: got %v", decoded.Results)
	}
}

func TestModule_Collect(t *testing.T) {
	m := New()
	items, err := m.Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if items != nil {
		t.Errorf("expected nil, got %v", items)
	}
}

// mockInstaller implements Installer for tests.
type mockInstaller struct {
	name string
	fn   func(ctx context.Context, pkg PatchTarget, dryRun bool) (InstallResult, error)
}

func (m *mockInstaller) Name() string { return m.name }
func (m *mockInstaller) Install(ctx context.Context, pkg PatchTarget, dryRun bool) (InstallResult, error) {
	return m.fn(ctx, pkg, dryRun)
}

// --- Rollback tests ---

// setupRollbackDB creates an in-memory SQLite database with the agent schema applied.
func setupRollbackDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ApplySchema(db); err != nil {
		t.Fatal(err)
	}
	if err := store.ApplyMigrations(db); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// newRollbackCmd builds a rollback_patch agent.Command for the given original commandID.
func newRollbackCmd(commandID string) agent.Command {
	payload, _ := json.Marshal(rollbackPayload{CommandID: commandID})
	return agent.Command{ID: "rb-cmd-1", Type: "rollback_patch", Payload: payload}
}

// newTestModuleWithRollbackStore creates a test module with a rollback store injected.
func newTestModuleWithRollbackStore(inst Installer, rs *store.RollbackStore) *Module {
	m := newTestModule(inst, nil)
	m.rollbackStore = rs
	return m
}

func TestRollback_NilStore(t *testing.T) {
	payload, _ := json.Marshal(rollbackPayload{CommandID: "cmd-x"})
	m := newTestModule(nil, nil)
	// rollbackStore is nil by default in newTestModule.

	result, err := m.HandleCommand(context.Background(), agent.Command{
		ID: "rb-nil", Type: "rollback_patch", Payload: payload,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ErrorMessage == "" {
		t.Error("expected error message when rollback store is nil")
	}
}

func TestRollback_StoreReturnsError(t *testing.T) {
	db := setupRollbackDB(t)
	rs := store.NewRollbackStore(db)

	// Close the DB so all queries fail.
	db.Close()

	mockInst := &mockInstaller{name: "apt", fn: func(_ context.Context, _ PatchTarget, _ bool) (InstallResult, error) {
		return InstallResult{ExitCode: 0}, nil
	}}
	m := newTestModuleWithRollbackStore(mockInst, rs)

	result, err := m.HandleCommand(context.Background(), newRollbackCmd("cmd-closed-db"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ErrorMessage == "" {
		t.Error("expected error message when store query fails")
	}
}

func TestRollback_EmptyRecords(t *testing.T) {
	db := setupRollbackDB(t)
	rs := store.NewRollbackStore(db)

	mockInst := &mockInstaller{name: "apt", fn: func(_ context.Context, _ PatchTarget, _ bool) (InstallResult, error) {
		t.Fatal("install should not be called for empty rollback records")
		return InstallResult{}, nil
	}}
	m := newTestModuleWithRollbackStore(mockInst, rs)

	// No records saved for "cmd-no-records" — expect "not found" error message.
	result, err := m.HandleCommand(context.Background(), newRollbackCmd("cmd-no-records"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ErrorMessage == "" {
		t.Error("expected error message when no rollback records found")
	}
}

func TestRollback_Success(t *testing.T) {
	db := setupRollbackDB(t)
	rs := store.NewRollbackStore(db)
	ctx := context.Background()

	// Pre-populate a rollback record.
	rec := &store.RollbackRecord{
		ID:          "rb-s1",
		CommandID:   "cmd-orig-1",
		PackageName: "curl",
		FromVersion: "7.68.0",
		ToVersion:   "7.88.1",
		Status:      "pending",
	}
	if err := rs.Save(ctx, rec); err != nil {
		t.Fatal(err)
	}

	var downgradedPkg string
	var downgradedVersion string
	mockInst := &mockInstaller{name: "apt", fn: func(_ context.Context, pkg PatchTarget, _ bool) (InstallResult, error) {
		downgradedPkg = pkg.Name
		downgradedVersion = pkg.Version
		return InstallResult{ExitCode: 0}, nil
	}}
	m := newTestModuleWithRollbackStore(mockInst, rs)

	result, err := m.HandleCommand(ctx, newRollbackCmd("cmd-orig-1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ErrorMessage != "" {
		t.Errorf("unexpected error message: %s", result.ErrorMessage)
	}
	if downgradedPkg != "curl" {
		t.Errorf("downgraded package = %q, want %q", downgradedPkg, "curl")
	}
	if downgradedVersion != "7.68.0" {
		t.Errorf("downgraded version = %q, want %q", downgradedVersion, "7.68.0")
	}

	// Verify the record was marked completed in the store.
	records, err := rs.ListByCommand(ctx, "cmd-orig-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].Status != "completed" {
		t.Errorf("record status = %q, want %q", records[0].Status, "completed")
	}
}

func TestRollback_PartialFailure(t *testing.T) {
	db := setupRollbackDB(t)
	rs := store.NewRollbackStore(db)
	ctx := context.Background()

	// Two records: one will succeed, one will fail.
	records := []*store.RollbackRecord{
		{ID: "rb-p1", CommandID: "cmd-partial", PackageName: "curl", FromVersion: "7.68.0", ToVersion: "7.88.1", Status: "pending"},
		{ID: "rb-p2", CommandID: "cmd-partial", PackageName: "wget", FromVersion: "1.20.0", ToVersion: "1.21.0", Status: "pending"},
	}
	for _, r := range records {
		if err := rs.Save(ctx, r); err != nil {
			t.Fatal(err)
		}
	}

	mockInst := &mockInstaller{name: "apt", fn: func(_ context.Context, pkg PatchTarget, _ bool) (InstallResult, error) {
		if pkg.Name == "wget" {
			return InstallResult{ExitCode: 1, Stderr: []byte("downgrade failed")}, nil
		}
		return InstallResult{ExitCode: 0}, nil
	}}
	m := newTestModuleWithRollbackStore(mockInst, rs)

	result, err := m.HandleCommand(ctx, newRollbackCmd("cmd-partial"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ErrorMessage == "" {
		t.Error("expected error message for partial rollback failure")
	}

	var output pb.InstallPatchOutput
	if err := proto.Unmarshal(result.Output, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if len(output.Results) != 2 {
		t.Fatalf("output results = %d, want 2", len(output.Results))
	}

	// curl should have succeeded, wget should have failed.
	curlResult := output.Results[0]
	wgetResult := output.Results[1]
	if curlResult.PackageName != "curl" || !curlResult.Succeeded {
		t.Errorf("curl result: pkg=%q succeeded=%v, want curl/true", curlResult.PackageName, curlResult.Succeeded)
	}
	if wgetResult.PackageName != "wget" || wgetResult.Succeeded {
		t.Errorf("wget result: pkg=%q succeeded=%v, want wget/false", wgetResult.PackageName, wgetResult.Succeeded)
	}

	// Verify store marks: curl→completed, wget→failed.
	stored, err := rs.ListByCommand(ctx, "cmd-partial")
	if err != nil {
		t.Fatal(err)
	}
	statusByPkg := make(map[string]string)
	for _, r := range stored {
		statusByPkg[r.PackageName] = r.Status
	}
	if statusByPkg["curl"] != "completed" {
		t.Errorf("curl record status = %q, want completed", statusByPkg["curl"])
	}
	if statusByPkg["wget"] != "failed" {
		t.Errorf("wget record status = %q, want failed", statusByPkg["wget"])
	}
}

// --- Protobuf rollback tests ---

func TestHandleRollbackProtobuf(t *testing.T) {
	// Protobuf payload with OriginalCommandId — uses rollback store.
	db := setupRollbackDB(t)
	rs := store.NewRollbackStore(db)
	ctx := context.Background()

	// Pre-populate a rollback record for the original command.
	rec := &store.RollbackRecord{
		ID:          "rb-proto-1",
		CommandID:   "orig-cmd-proto",
		PackageName: "nginx",
		FromVersion: "1.18.0",
		ToVersion:   "1.24.0",
		Status:      "pending",
	}
	if err := rs.Save(ctx, rec); err != nil {
		t.Fatal(err)
	}

	var downgradedPkg string
	var downgradedVersion string
	mockInst := &mockInstaller{name: "apt", fn: func(_ context.Context, pkg PatchTarget, _ bool) (InstallResult, error) {
		downgradedPkg = pkg.Name
		downgradedVersion = pkg.Version
		return InstallResult{ExitCode: 0}, nil
	}}
	m := newTestModuleWithRollbackStore(mockInst, rs)

	// Build protobuf payload.
	pbPayload := &pb.RollbackPatchPayload{
		DeploymentId:      "deploy-1",
		OriginalCommandId: "orig-cmd-proto",
	}
	payloadBytes, err := proto.Marshal(pbPayload)
	if err != nil {
		t.Fatal(err)
	}

	result, err := m.HandleCommand(ctx, agent.Command{
		ID: "rb-pb-cmd-1", Type: "rollback_patch", Payload: payloadBytes,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ErrorMessage != "" {
		t.Errorf("unexpected error message: %s", result.ErrorMessage)
	}
	if downgradedPkg != "nginx" {
		t.Errorf("downgraded package = %q, want %q", downgradedPkg, "nginx")
	}
	if downgradedVersion != "1.18.0" {
		t.Errorf("downgraded version = %q, want %q", downgradedVersion, "1.18.0")
	}

	// Verify store record marked completed.
	records, err := rs.ListByCommand(ctx, "orig-cmd-proto")
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].Status != "completed" {
		t.Errorf("record status = %q, want %q", records[0].Status, "completed")
	}
}

func TestHandleRollbackWithRevertTo(t *testing.T) {
	// Protobuf payload with RevertTo targets — installs specified versions directly,
	// no rollback store needed.
	var installed []PatchTarget
	mockInst := &mockInstaller{name: "apt", fn: func(_ context.Context, pkg PatchTarget, _ bool) (InstallResult, error) {
		installed = append(installed, pkg)
		return InstallResult{ExitCode: 0}, nil
	}}
	m := newTestModule(mockInst, nil)
	// No rollback store needed for RevertTo path.

	pbPayload := &pb.RollbackPatchPayload{
		DeploymentId:      "deploy-2",
		OriginalCommandId: "orig-cmd-2",
		RevertTo: []*pb.PatchTarget{
			{Name: "curl", Version: "7.68.0"},
			{Name: "openssl", Version: "1.1.1"},
		},
	}
	payloadBytes, err := proto.Marshal(pbPayload)
	if err != nil {
		t.Fatal(err)
	}

	result, err := m.HandleCommand(context.Background(), agent.Command{
		ID: "rb-revert-1", Type: "rollback_patch", Payload: payloadBytes,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ErrorMessage != "" {
		t.Errorf("unexpected error message: %s", result.ErrorMessage)
	}
	if len(installed) != 2 {
		t.Fatalf("installed %d packages, want 2", len(installed))
	}
	if installed[0].Name != "curl" || installed[0].Version != "7.68.0" {
		t.Errorf("first installed = %+v, want curl@7.68.0", installed[0])
	}
	if installed[1].Name != "openssl" || installed[1].Version != "1.1.1" {
		t.Errorf("second installed = %+v, want openssl@1.1.1", installed[1])
	}

	// Verify output has correct results.
	var output pb.InstallPatchOutput
	if err := proto.Unmarshal(result.Output, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if len(output.Results) != 2 {
		t.Fatalf("output results = %d, want 2", len(output.Results))
	}
	for _, r := range output.Results {
		if !r.Succeeded {
			t.Errorf("package %s should have succeeded", r.PackageName)
		}
	}
}

func TestHandleRollbackProtobufNoRecords(t *testing.T) {
	// Protobuf payload with OriginalCommandId but no matching rollback records.
	db := setupRollbackDB(t)
	rs := store.NewRollbackStore(db)

	mockInst := &mockInstaller{name: "apt", fn: func(_ context.Context, _ PatchTarget, _ bool) (InstallResult, error) {
		t.Fatal("install should not be called when no rollback records found")
		return InstallResult{}, nil
	}}
	m := newTestModuleWithRollbackStore(mockInst, rs)

	pbPayload := &pb.RollbackPatchPayload{
		DeploymentId:      "deploy-3",
		OriginalCommandId: "nonexistent-cmd",
	}
	payloadBytes, err := proto.Marshal(pbPayload)
	if err != nil {
		t.Fatal(err)
	}

	result, err := m.HandleCommand(context.Background(), agent.Command{
		ID: "rb-norec-1", Type: "rollback_patch", Payload: payloadBytes,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ErrorMessage == "" {
		t.Error("expected error message when no rollback records found")
	}
}

// --- Auto-reboot tests ---

func TestHandleInstallPatch_RebootTriggered(t *testing.T) {
	payload := &pb.InstallPatchPayload{
		Packages: []*pb.PatchTarget{{Name: "patch1", Version: "1.0"}},
	}
	payloadBytes, _ := proto.Marshal(payload)

	mockInst := &mockInstaller{
		name: "msi",
		fn: func(_ context.Context, _ PatchTarget, _ bool) (InstallResult, error) {
			return InstallResult{ExitCode: 0, RebootRequired: true}, nil
		},
	}

	var rebootCalled bool
	var gotDelay int32
	m := newTestModule(mockInst, nil)
	m.rebootFunc = func(_ context.Context, delay int32) error {
		rebootCalled = true
		gotDelay = delay
		return nil
	}

	result, err := m.HandleCommand(context.Background(), agent.Command{
		ID: "cmd-reboot", Type: "install_patch", Payload: payloadBytes,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ErrorMessage != "" {
		t.Errorf("unexpected error message: %s", result.ErrorMessage)
	}
	if !rebootCalled {
		t.Error("expected reboot function to be called when RebootRequired=true and rebootFunc is set")
	}
	if gotDelay != 60 {
		t.Errorf("reboot delay = %d, want 60 (default)", gotDelay)
	}
}

func TestHandleInstallPatch_NoRebootWhenFuncNil(t *testing.T) {
	payload := &pb.InstallPatchPayload{
		Packages: []*pb.PatchTarget{{Name: "patch1", Version: "1.0"}},
	}
	payloadBytes, _ := proto.Marshal(payload)

	mockInst := &mockInstaller{
		name: "msi",
		fn: func(_ context.Context, _ PatchTarget, _ bool) (InstallResult, error) {
			return InstallResult{ExitCode: 0, RebootRequired: true}, nil
		},
	}

	m := newTestModule(mockInst, nil)
	// rebootFunc is nil by default — should not panic.

	result, err := m.HandleCommand(context.Background(), agent.Command{
		ID: "cmd-nilreboot", Type: "install_patch", Payload: payloadBytes,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ErrorMessage != "" {
		t.Errorf("unexpected error message: %s", result.ErrorMessage)
	}
}

func TestHandleInstallPatch_NoRebootOnDryRun(t *testing.T) {
	payload := &pb.InstallPatchPayload{
		Packages: []*pb.PatchTarget{{Name: "patch1", Version: "1.0"}},
		DryRun:   true,
	}
	payloadBytes, _ := proto.Marshal(payload)

	mockInst := &mockInstaller{
		name: "msi",
		fn: func(_ context.Context, _ PatchTarget, _ bool) (InstallResult, error) {
			return InstallResult{ExitCode: 0, RebootRequired: true}, nil
		},
	}

	rebootCalled := false
	m := newTestModule(mockInst, nil)
	m.rebootFunc = func(_ context.Context, _ int32) error {
		rebootCalled = true
		return nil
	}

	_, _ = m.HandleCommand(context.Background(), agent.Command{
		ID: "cmd-dryrun-reboot", Type: "install_patch", Payload: payloadBytes,
	})
	if rebootCalled {
		t.Error("reboot should NOT be called during dry-run")
	}
}

func TestHandleInstallPatch_NoRebootWhenNotRequired(t *testing.T) {
	payload := &pb.InstallPatchPayload{
		Packages: []*pb.PatchTarget{{Name: "patch1", Version: "1.0"}},
	}
	payloadBytes, _ := proto.Marshal(payload)

	mockInst := &mockInstaller{
		name: "msi",
		fn: func(_ context.Context, _ PatchTarget, _ bool) (InstallResult, error) {
			return InstallResult{ExitCode: 0, RebootRequired: false}, nil
		},
	}

	rebootCalled := false
	m := newTestModule(mockInst, nil)
	m.rebootFunc = func(_ context.Context, _ int32) error {
		rebootCalled = true
		return nil
	}

	_, _ = m.HandleCommand(context.Background(), agent.Command{
		ID: "cmd-no-reboot-needed", Type: "install_patch", Payload: payloadBytes,
	})
	if rebootCalled {
		t.Error("reboot should NOT be called when no package requires it")
	}
}
