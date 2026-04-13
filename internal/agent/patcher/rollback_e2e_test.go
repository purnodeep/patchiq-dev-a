package patcher

import (
	"context"
	"database/sql"
	"testing"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	"github.com/skenzeriq/patchiq/internal/agent"
	"github.com/skenzeriq/patchiq/internal/agent/store"
	"google.golang.org/protobuf/proto"
	_ "modernc.org/sqlite"
)

// mockInstallerWithVersion implements both Installer and VersionQuerier,
// allowing tests to control pre-install version queries and install behavior.
type mockInstallerWithVersion struct {
	name         string
	installFn    func(ctx context.Context, pkg PatchTarget, dryRun bool) (InstallResult, error)
	getVersionFn func(ctx context.Context, packageName string) (string, error)
}

func (m *mockInstallerWithVersion) Name() string { return m.name }
func (m *mockInstallerWithVersion) Install(ctx context.Context, pkg PatchTarget, dryRun bool) (InstallResult, error) {
	return m.installFn(ctx, pkg, dryRun)
}
func (m *mockInstallerWithVersion) GetCurrentVersion(ctx context.Context, packageName string) (string, error) {
	return m.getVersionFn(ctx, packageName)
}

// Compile-time check that mockInstallerWithVersion satisfies both interfaces.
var (
	_ Installer      = (*mockInstallerWithVersion)(nil)
	_ VersionQuerier = (*mockInstallerWithVersion)(nil)
)

// setupE2EDB creates an in-memory SQLite database with the agent schema for E2E tests.
func setupE2EDB(t *testing.T) *sql.DB {
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

// newTestModuleWithDeps creates a patcher module with a rollback store and a
// version-aware mock installer for E2E flow tests.
func newTestModuleWithDeps(inst Installer, rs *store.RollbackStore) *Module {
	m := newTestModule(inst, nil)
	m.rollbackStore = rs
	return m
}

// TestRollbackE2E_InstallThenRollback exercises the full rollback lifecycle:
//
//  1. Send install_patch command with a package that has a known pre-install version.
//  2. Verify the install succeeds and a rollback record is persisted with correct
//     from_version / to_version.
//  3. Send rollback_patch command referencing the original install command.
//  4. Verify the rollback installs the previous (from) version.
//  5. Verify the rollback record is marked completed.
//  6. Verify both command results carry the correct protobuf output.
func TestRollbackE2E_InstallThenRollback(t *testing.T) {
	db := setupE2EDB(t)
	rs := store.NewRollbackStore(db)
	ctx := context.Background()

	const (
		preInstallVersion = "7.68.0"
		newVersion        = "7.88.1"
		packageName       = "curl"
		installCmdID      = "cmd-install-e2e"
	)

	// Track what the installer was asked to install, in order.
	var installedTargets []PatchTarget
	mockInst := &mockInstallerWithVersion{
		name: "apt",
		installFn: func(_ context.Context, pkg PatchTarget, _ bool) (InstallResult, error) {
			installedTargets = append(installedTargets, pkg)
			return InstallResult{Stdout: []byte("ok"), ExitCode: 0}, nil
		},
		getVersionFn: func(_ context.Context, pkgName string) (string, error) {
			if pkgName == packageName {
				return preInstallVersion, nil
			}
			return "", errNotFound
		},
	}

	m := newTestModuleWithDeps(mockInst, rs)

	// --- Step 1: install_patch ---
	installPayload := &pb.InstallPatchPayload{
		Packages: []*pb.PatchTarget{{Name: packageName, Version: newVersion}},
	}
	installBytes, err := proto.Marshal(installPayload)
	if err != nil {
		t.Fatal(err)
	}

	installResult, err := m.HandleCommand(ctx, agent.Command{
		ID:      installCmdID,
		Type:    "install_patch",
		Payload: installBytes,
	})
	if err != nil {
		t.Fatalf("install_patch HandleCommand error: %v", err)
	}
	if installResult.ErrorMessage != "" {
		t.Fatalf("install_patch reported failure: %s", installResult.ErrorMessage)
	}

	// Verify install output.
	var installOutput pb.InstallPatchOutput
	if err := proto.Unmarshal(installResult.Output, &installOutput); err != nil {
		t.Fatalf("unmarshal install output: %v", err)
	}
	if len(installOutput.Results) != 1 {
		t.Fatalf("install output results = %d, want 1", len(installOutput.Results))
	}
	if !installOutput.Results[0].Succeeded {
		t.Error("expected install to succeed")
	}
	if installOutput.Results[0].PackageName != packageName {
		t.Errorf("install package name = %q, want %q", installOutput.Results[0].PackageName, packageName)
	}

	// --- Step 2: verify rollback record was saved ---
	records, err := rs.ListByCommand(ctx, installCmdID)
	if err != nil {
		t.Fatalf("list rollback records: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("rollback records = %d, want 1", len(records))
	}
	rec := records[0]
	if rec.PackageName != packageName {
		t.Errorf("rollback record package = %q, want %q", rec.PackageName, packageName)
	}
	if rec.FromVersion != preInstallVersion {
		t.Errorf("rollback record from_version = %q, want %q", rec.FromVersion, preInstallVersion)
	}
	if rec.ToVersion != newVersion {
		t.Errorf("rollback record to_version = %q, want %q", rec.ToVersion, newVersion)
	}
	if rec.Status != "pending" {
		t.Errorf("rollback record status = %q, want %q", rec.Status, "pending")
	}

	// --- Step 3: rollback_patch ---
	rollbackPayloadBytes, err := proto.Marshal(&pb.RollbackPatchPayload{
		OriginalCommandId: installCmdID,
	})
	if err != nil {
		t.Fatal(err)
	}

	rollbackResult, err := m.HandleCommand(ctx, agent.Command{
		ID:      "cmd-rollback-e2e",
		Type:    "rollback_patch",
		Payload: rollbackPayloadBytes,
	})
	if err != nil {
		t.Fatalf("rollback_patch HandleCommand error: %v", err)
	}
	if rollbackResult.ErrorMessage != "" {
		t.Fatalf("rollback_patch reported failure: %s", rollbackResult.ErrorMessage)
	}

	// --- Step 4: verify the rollback installed the previous version ---
	if len(installedTargets) != 2 {
		t.Fatalf("total install calls = %d, want 2 (1 install + 1 rollback)", len(installedTargets))
	}
	rollbackTarget := installedTargets[1]
	if rollbackTarget.Name != packageName {
		t.Errorf("rollback target name = %q, want %q", rollbackTarget.Name, packageName)
	}
	if rollbackTarget.Version != preInstallVersion {
		t.Errorf("rollback target version = %q, want %q", rollbackTarget.Version, preInstallVersion)
	}

	// --- Step 5: verify rollback record is now completed ---
	records, err = rs.ListByCommand(ctx, installCmdID)
	if err != nil {
		t.Fatalf("list rollback records after rollback: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("rollback records after rollback = %d, want 1", len(records))
	}
	if records[0].Status != "completed" {
		t.Errorf("rollback record status after rollback = %q, want %q", records[0].Status, "completed")
	}
	if records[0].RolledBackAt == nil {
		t.Error("expected rolled_back_at to be set after rollback")
	}

	// --- Step 6: verify rollback result output ---
	var rollbackOutput pb.InstallPatchOutput
	if err := proto.Unmarshal(rollbackResult.Output, &rollbackOutput); err != nil {
		t.Fatalf("unmarshal rollback output: %v", err)
	}
	if len(rollbackOutput.Results) != 1 {
		t.Fatalf("rollback output results = %d, want 1", len(rollbackOutput.Results))
	}
	rbDetail := rollbackOutput.Results[0]
	if !rbDetail.Succeeded {
		t.Error("expected rollback to succeed")
	}
	if rbDetail.PackageName != packageName {
		t.Errorf("rollback output package = %q, want %q", rbDetail.PackageName, packageName)
	}
	if rbDetail.Version != preInstallVersion {
		t.Errorf("rollback output version = %q, want %q (should be the reverted-to version)", rbDetail.Version, preInstallVersion)
	}
}

// TestRollbackE2E_MultiplePackages tests install and rollback of multiple
// packages in a single command, verifying each gets its own rollback record
// and each is reverted independently.
func TestRollbackE2E_MultiplePackages(t *testing.T) {
	db := setupE2EDB(t)
	rs := store.NewRollbackStore(db)
	ctx := context.Background()

	const installCmdID = "cmd-multi-install-e2e"

	preVersions := map[string]string{
		"curl":    "7.68.0",
		"openssl": "1.1.1",
		"wget":    "1.20.0",
	}

	var installedTargets []PatchTarget
	mockInst := &mockInstallerWithVersion{
		name: "apt",
		installFn: func(_ context.Context, pkg PatchTarget, _ bool) (InstallResult, error) {
			installedTargets = append(installedTargets, pkg)
			return InstallResult{Stdout: []byte("ok"), ExitCode: 0}, nil
		},
		getVersionFn: func(_ context.Context, pkgName string) (string, error) {
			if v, ok := preVersions[pkgName]; ok {
				return v, nil
			}
			return "", errNotFound
		},
	}

	m := newTestModuleWithDeps(mockInst, rs)

	// Install three packages.
	installPayload := &pb.InstallPatchPayload{
		Packages: []*pb.PatchTarget{
			{Name: "curl", Version: "7.88.1"},
			{Name: "openssl", Version: "3.0.0"},
			{Name: "wget", Version: "1.21.0"},
		},
	}
	installBytes, _ := proto.Marshal(installPayload)

	installResult, err := m.HandleCommand(ctx, agent.Command{
		ID: installCmdID, Type: "install_patch", Payload: installBytes,
	})
	if err != nil {
		t.Fatalf("install error: %v", err)
	}
	if installResult.ErrorMessage != "" {
		t.Fatalf("install failure: %s", installResult.ErrorMessage)
	}

	// Verify 3 rollback records created.
	records, err := rs.ListByCommand(ctx, installCmdID)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 3 {
		t.Fatalf("rollback records = %d, want 3", len(records))
	}
	for _, r := range records {
		if r.Status != "pending" {
			t.Errorf("record %s status = %q, want pending", r.PackageName, r.Status)
		}
		expectedFrom, ok := preVersions[r.PackageName]
		if !ok {
			t.Errorf("unexpected package in rollback records: %s", r.PackageName)
			continue
		}
		if r.FromVersion != expectedFrom {
			t.Errorf("record %s from_version = %q, want %q", r.PackageName, r.FromVersion, expectedFrom)
		}
	}

	// Reset tracking.
	installedTargets = nil

	// Rollback all packages for the original command.
	rollbackBytes, _ := proto.Marshal(&pb.RollbackPatchPayload{
		OriginalCommandId: installCmdID,
	})
	rollbackResult, err := m.HandleCommand(ctx, agent.Command{
		ID: "cmd-multi-rollback-e2e", Type: "rollback_patch", Payload: rollbackBytes,
	})
	if err != nil {
		t.Fatalf("rollback error: %v", err)
	}
	if rollbackResult.ErrorMessage != "" {
		t.Fatalf("rollback failure: %s", rollbackResult.ErrorMessage)
	}

	// Verify all 3 packages were reverted to their previous versions.
	if len(installedTargets) != 3 {
		t.Fatalf("rollback install calls = %d, want 3", len(installedTargets))
	}
	for _, target := range installedTargets {
		expectedVersion, ok := preVersions[target.Name]
		if !ok {
			t.Errorf("unexpected rollback target: %s", target.Name)
			continue
		}
		if target.Version != expectedVersion {
			t.Errorf("rollback %s version = %q, want %q", target.Name, target.Version, expectedVersion)
		}
	}

	// All records should now be completed.
	records, err = rs.ListByCommand(ctx, installCmdID)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range records {
		if r.Status != "completed" {
			t.Errorf("record %s status = %q after rollback, want completed", r.PackageName, r.Status)
		}
	}

	// Verify rollback output has 3 successful results.
	var output pb.InstallPatchOutput
	if err := proto.Unmarshal(rollbackResult.Output, &output); err != nil {
		t.Fatalf("unmarshal rollback output: %v", err)
	}
	if len(output.Results) != 3 {
		t.Fatalf("rollback output results = %d, want 3", len(output.Results))
	}
	for _, r := range output.Results {
		if !r.Succeeded {
			t.Errorf("rollback result for %s should have succeeded", r.PackageName)
		}
	}
}

// TestRollbackE2E_InstallFailure_NoRollbackRecord verifies that a failed
// install does not create a rollback record, so a subsequent rollback
// correctly reports no records found.
func TestRollbackE2E_InstallFailure_NoRollbackRecord(t *testing.T) {
	db := setupE2EDB(t)
	rs := store.NewRollbackStore(db)
	ctx := context.Background()

	const installCmdID = "cmd-fail-install-e2e"

	mockInst := &mockInstallerWithVersion{
		name: "apt",
		installFn: func(_ context.Context, _ PatchTarget, _ bool) (InstallResult, error) {
			return InstallResult{Stderr: []byte("dependency error"), ExitCode: 1}, nil
		},
		getVersionFn: func(_ context.Context, _ string) (string, error) {
			return "1.0.0", nil
		},
	}

	m := newTestModuleWithDeps(mockInst, rs)

	installPayload := &pb.InstallPatchPayload{
		Packages: []*pb.PatchTarget{{Name: "badpkg", Version: "2.0.0"}},
	}
	installBytes, _ := proto.Marshal(installPayload)

	installResult, err := m.HandleCommand(ctx, agent.Command{
		ID: installCmdID, Type: "install_patch", Payload: installBytes,
	})
	if err != nil {
		t.Fatalf("install error: %v", err)
	}
	// Install should report failure.
	if installResult.ErrorMessage == "" {
		t.Error("expected error message for failed install")
	}

	// No rollback records should exist for the failed install.
	records, err := rs.ListByCommand(ctx, installCmdID)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 0 {
		t.Fatalf("rollback records = %d, want 0 (install failed)", len(records))
	}

	// Attempting rollback should report no records found.
	rollbackBytes, _ := proto.Marshal(&pb.RollbackPatchPayload{
		OriginalCommandId: installCmdID,
	})
	rollbackResult, err := m.HandleCommand(ctx, agent.Command{
		ID: "cmd-fail-rollback-e2e", Type: "rollback_patch", Payload: rollbackBytes,
	})
	if err != nil {
		t.Fatalf("rollback error: %v", err)
	}
	if rollbackResult.ErrorMessage == "" {
		t.Error("expected error message when no rollback records found for failed install")
	}
}

// TestRollbackE2E_DryRunSkipsRollbackRecord verifies that dry-run installs
// do not create rollback records.
func TestRollbackE2E_DryRunSkipsRollbackRecord(t *testing.T) {
	db := setupE2EDB(t)
	rs := store.NewRollbackStore(db)
	ctx := context.Background()

	const installCmdID = "cmd-dryrun-e2e"

	mockInst := &mockInstallerWithVersion{
		name: "apt",
		installFn: func(_ context.Context, _ PatchTarget, _ bool) (InstallResult, error) {
			return InstallResult{ExitCode: 0}, nil
		},
		getVersionFn: func(_ context.Context, _ string) (string, error) {
			return "1.0.0", nil
		},
	}

	m := newTestModuleWithDeps(mockInst, rs)

	installPayload := &pb.InstallPatchPayload{
		Packages: []*pb.PatchTarget{{Name: "curl", Version: "2.0.0"}},
		DryRun:   true,
	}
	installBytes, _ := proto.Marshal(installPayload)

	installResult, err := m.HandleCommand(ctx, agent.Command{
		ID: installCmdID, Type: "install_patch", Payload: installBytes,
	})
	if err != nil {
		t.Fatalf("dry-run install error: %v", err)
	}
	if installResult.ErrorMessage != "" {
		t.Fatalf("dry-run install failure: %s", installResult.ErrorMessage)
	}

	// Dry run should not create rollback records.
	records, err := rs.ListByCommand(ctx, installCmdID)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 0 {
		t.Errorf("rollback records = %d, want 0 (dry run)", len(records))
	}
}

// TestRollbackE2E_DoubleRollbackSkipsCompleted verifies that rolling back the
// same command twice does not re-process already-completed records.
func TestRollbackE2E_DoubleRollbackSkipsCompleted(t *testing.T) {
	db := setupE2EDB(t)
	rs := store.NewRollbackStore(db)
	ctx := context.Background()

	const installCmdID = "cmd-double-rb-e2e"

	installCount := 0
	mockInst := &mockInstallerWithVersion{
		name: "apt",
		installFn: func(_ context.Context, _ PatchTarget, _ bool) (InstallResult, error) {
			installCount++
			return InstallResult{ExitCode: 0}, nil
		},
		getVersionFn: func(_ context.Context, _ string) (string, error) {
			return "1.0.0", nil
		},
	}

	m := newTestModuleWithDeps(mockInst, rs)

	// Install.
	installBytes, _ := proto.Marshal(&pb.InstallPatchPayload{
		Packages: []*pb.PatchTarget{{Name: "curl", Version: "2.0.0"}},
	})
	if _, err := m.HandleCommand(ctx, agent.Command{
		ID: installCmdID, Type: "install_patch", Payload: installBytes,
	}); err != nil {
		t.Fatal(err)
	}
	// installCount == 1 (the install itself).

	// First rollback.
	rollbackBytes, _ := proto.Marshal(&pb.RollbackPatchPayload{
		OriginalCommandId: installCmdID,
	})
	result1, err := m.HandleCommand(ctx, agent.Command{
		ID: "rb-1-e2e", Type: "rollback_patch", Payload: rollbackBytes,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result1.ErrorMessage != "" {
		t.Fatalf("first rollback failure: %s", result1.ErrorMessage)
	}
	// installCount == 2 (install + rollback).

	firstRollbackCount := installCount

	// Second rollback of the same command — records are already completed.
	result2, err := m.HandleCommand(ctx, agent.Command{
		ID: "rb-2-e2e", Type: "rollback_patch", Payload: rollbackBytes,
	})
	if err != nil {
		t.Fatal(err)
	}

	// The installer should not have been called again since the record is completed.
	if installCount != firstRollbackCount {
		t.Errorf("installer was called %d times after double rollback, want %d (no additional calls)",
			installCount, firstRollbackCount)
	}

	// Second rollback should still return output (empty results since all skipped).
	var output pb.InstallPatchOutput
	if err := proto.Unmarshal(result2.Output, &output); err != nil {
		t.Fatalf("unmarshal second rollback output: %v", err)
	}
	if len(output.Results) != 0 {
		t.Errorf("second rollback results = %d, want 0 (all already completed)", len(output.Results))
	}
}
