package patcher

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestMSIXInstaller_Name(t *testing.T) {
	inst := &msixInstaller{executor: &mockExecutor{fn: func(_ context.Context, _ string, _ ...string) (ExecResult, error) {
		return ExecResult{}, nil
	}}}
	if got := inst.Name(); got != "msix" {
		t.Errorf("Name() = %q, want %q", got, "msix")
	}
}

func TestMSIXInstaller_Install_Success(t *testing.T) {
	origCheck := checkAdmin
	checkAdmin = func() error { return nil }
	defer func() { checkAdmin = origCheck }()

	var gotName string
	var gotArgs []string
	mock := &mockExecutor{
		fn: func(_ context.Context, name string, args ...string) (ExecResult, error) {
			gotName = name
			gotArgs = args
			return ExecResult{ExitCode: 0, Stdout: []byte("ok")}, nil
		},
	}

	inst := &msixInstaller{executor: mock}
	result, err := inst.Install(context.Background(), PatchTarget{Name: `C:\patches\app.msix`}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}

	if gotName != "powershell.exe" {
		t.Errorf("command = %q, want %q", gotName, "powershell.exe")
	}

	// Path is single-quoted to prevent PowerShell metacharacter injection.
	wantArgs := []string{"-NoProfile", "-Command", `Add-AppxPackage -Path 'C:\patches\app.msix'`}
	if len(gotArgs) != len(wantArgs) {
		t.Fatalf("args = %v, want %v", gotArgs, wantArgs)
	}
	for i, arg := range wantArgs {
		if gotArgs[i] != arg {
			t.Errorf("arg[%d] = %q, want %q", i, gotArgs[i], arg)
		}
	}
}

func TestMSIXInstaller_Install_NoInjection(t *testing.T) {
	origCheck := checkAdmin
	checkAdmin = func() error { return nil }
	defer func() { checkAdmin = origCheck }()

	var gotArgs []string
	mock := &mockExecutor{
		fn: func(_ context.Context, _ string, args ...string) (ExecResult, error) {
			gotArgs = args
			return ExecResult{ExitCode: 0}, nil
		},
	}

	inst := &msixInstaller{executor: mock}
	// A malicious package name with single quotes trying to break out of the string literal.
	_, err := inst.Install(context.Background(), PatchTarget{Name: "'; Remove-Item -Recurse C:\\; '"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The -Command arg must have single quotes escaped (' → '') so the malicious
	// payload stays inside the string literal and is not interpreted as code.
	cmdArg := gotArgs[len(gotArgs)-1]
	if !strings.Contains(cmdArg, "''") {
		t.Errorf("expected escaped single quotes ('') in command arg, got %q", cmdArg)
	}
	if strings.Contains(cmdArg, "Remove-Item") && !strings.Contains(cmdArg, "''") {
		t.Error("injection payload not properly escaped — single quote breakout possible")
	}
}

func TestMSIXInstaller_Install_DryRun(t *testing.T) {
	mock := &mockExecutor{
		fn: func(_ context.Context, _ string, _ ...string) (ExecResult, error) {
			t.Fatal("executor should not be called during dry-run")
			return ExecResult{}, nil
		},
	}

	inst := &msixInstaller{executor: mock}
	result, err := inst.Install(context.Background(), PatchTarget{Name: `C:\patches\app.msix`}, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(result.Stdout), "dry-run") {
		t.Errorf("expected dry-run message in stdout, got %q", string(result.Stdout))
	}
}

func TestMSIXInstaller_Install_Failure(t *testing.T) {
	origCheck := checkAdmin
	checkAdmin = func() error { return nil }
	defer func() { checkAdmin = origCheck }()

	mock := &mockExecutor{
		fn: func(_ context.Context, _ string, _ ...string) (ExecResult, error) {
			return ExecResult{ExitCode: 1, Stderr: []byte("deployment failed")}, nil
		},
	}

	inst := &msixInstaller{executor: mock}
	result, err := inst.Install(context.Background(), PatchTarget{Name: `C:\app.msix`}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 1 {
		t.Errorf("ExitCode = %d, want 1", result.ExitCode)
	}
}

func TestMSIXInstaller_Install_ElevationRequired(t *testing.T) {
	origCheck := checkAdmin
	checkAdmin = func() error { return errNotAdmin }
	defer func() { checkAdmin = origCheck }()

	inst := &msixInstaller{executor: &mockExecutor{
		fn: func(_ context.Context, _ string, _ ...string) (ExecResult, error) {
			t.Fatal("executor should not be called when elevation check fails")
			return ExecResult{}, nil
		},
	}}
	_, err := inst.Install(context.Background(), PatchTarget{Name: "test.msix"}, false)
	if err == nil {
		t.Fatal("expected error when admin check fails")
	}
	if !errors.Is(err, errNotAdmin) {
		t.Errorf("error = %v, want wrapping errNotAdmin", err)
	}
}

func TestMSIXInstaller_Install_ElevationSuccess(t *testing.T) {
	origCheck := checkAdmin
	checkAdmin = func() error { return nil }
	defer func() { checkAdmin = origCheck }()

	inst := &msixInstaller{executor: &mockExecutor{
		fn: func(_ context.Context, name string, args ...string) (ExecResult, error) {
			return ExecResult{Stdout: []byte("ok"), ExitCode: 0}, nil
		},
	}}
	result, err := inst.Install(context.Background(), PatchTarget{Name: "test.msix"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0", result.ExitCode)
	}
}
