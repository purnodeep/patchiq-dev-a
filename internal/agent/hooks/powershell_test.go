package hooks

import (
	"context"
	"testing"
	"time"
)

// mockExecutor implements CommandExecutor for tests.
type mockExecutor struct {
	fn func(ctx context.Context, name string, args ...string) (ExecResult, error)
}

func (m *mockExecutor) Execute(ctx context.Context, name string, args ...string) (ExecResult, error) {
	return m.fn(ctx, name, args...)
}

func TestPowerShellExecutor_Run_CommandConstruction(t *testing.T) {
	var gotName string
	var gotArgs []string
	mock := &mockExecutor{
		fn: func(_ context.Context, name string, args ...string) (ExecResult, error) {
			gotName = name
			gotArgs = args
			return ExecResult{Stdout: []byte("done"), ExitCode: 0}, nil
		},
	}

	exec := NewPowerShellExecutor(mock)
	result, err := exec.Run(context.Background(), `C:\scripts\pre.ps1`, []string{"-Param1", "value"}, 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotName != "powershell.exe" {
		t.Errorf("command = %q, want %q", gotName, "powershell.exe")
	}

	wantArgs := []string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-File", `C:\scripts\pre.ps1`, "-Param1", "value"}
	if len(gotArgs) != len(wantArgs) {
		t.Fatalf("args = %v, want %v", gotArgs, wantArgs)
	}
	for i, arg := range wantArgs {
		if gotArgs[i] != arg {
			t.Errorf("arg[%d] = %q, want %q", i, gotArgs[i], arg)
		}
	}

	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
}

func TestPowerShellExecutor_Run_NoArgs(t *testing.T) {
	var gotArgs []string
	mock := &mockExecutor{
		fn: func(_ context.Context, _ string, args ...string) (ExecResult, error) {
			gotArgs = args
			return ExecResult{ExitCode: 0}, nil
		},
	}

	exec := NewPowerShellExecutor(mock)
	_, err := exec.Run(context.Background(), `C:\scripts\hook.ps1`, nil, 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantArgs := []string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-File", `C:\scripts\hook.ps1`}
	if len(gotArgs) != len(wantArgs) {
		t.Fatalf("args = %v, want %v", gotArgs, wantArgs)
	}
}

func TestPowerShellExecutor_Run_ScriptFailure(t *testing.T) {
	mock := &mockExecutor{
		fn: func(_ context.Context, _ string, _ ...string) (ExecResult, error) {
			return ExecResult{ExitCode: 1, Stderr: []byte("script error")}, nil
		},
	}

	exec := NewPowerShellExecutor(mock)
	result, err := exec.Run(context.Background(), `C:\scripts\bad.ps1`, nil, 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 1 {
		t.Errorf("ExitCode = %d, want 1", result.ExitCode)
	}
	if string(result.Stderr) != "script error" {
		t.Errorf("Stderr = %q, want %q", result.Stderr, "script error")
	}
}

func TestPowerShellExecutor_Run_Timeout(t *testing.T) {
	mock := &mockExecutor{
		fn: func(ctx context.Context, _ string, _ ...string) (ExecResult, error) {
			<-ctx.Done()
			return ExecResult{}, ctx.Err()
		},
	}

	exec := NewPowerShellExecutor(mock)
	_, err := exec.Run(context.Background(), `C:\scripts\slow.ps1`, nil, 1*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}
