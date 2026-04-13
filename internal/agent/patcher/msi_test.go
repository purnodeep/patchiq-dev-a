package patcher

import (
	"context"
	"strings"
	"testing"
)

func TestMSIInstaller_Name(t *testing.T) {
	inst := &msiInstaller{executor: &mockExecutor{fn: func(_ context.Context, _ string, _ ...string) (ExecResult, error) {
		return ExecResult{}, nil
	}}}
	if got := inst.Name(); got != "msi" {
		t.Errorf("Name() = %q, want %q", got, "msi")
	}
}

func TestMSIInstaller_Install_Args(t *testing.T) {
	var gotName string
	var gotArgs []string
	mock := &mockExecutor{
		fn: func(_ context.Context, name string, args ...string) (ExecResult, error) {
			gotName = name
			gotArgs = args
			return ExecResult{ExitCode: 0}, nil
		},
	}

	inst := &msiInstaller{executor: mock}
	_, err := inst.Install(context.Background(), PatchTarget{Name: `C:\patches\update.msi`, Version: "1.0"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotName != "msiexec" {
		t.Errorf("command = %q, want %q", gotName, "msiexec")
	}
	// Verify required args are present: /i <path> /quiet /norestart /l*v <logpath>
	if len(gotArgs) < 6 {
		t.Fatalf("args = %v, want at least 6 args", gotArgs)
	}
	if gotArgs[0] != "/i" {
		t.Errorf("arg[0] = %q, want /i", gotArgs[0])
	}
	if gotArgs[1] != `C:\patches\update.msi` {
		t.Errorf("arg[1] = %q, want package path", gotArgs[1])
	}
	if gotArgs[2] != "/quiet" {
		t.Errorf("arg[2] = %q, want /quiet", gotArgs[2])
	}
	if gotArgs[3] != "/norestart" {
		t.Errorf("arg[3] = %q, want /norestart", gotArgs[3])
	}
	if gotArgs[4] != "/l*v" {
		t.Errorf("arg[4] = %q, want /l*v", gotArgs[4])
	}
	if !strings.Contains(gotArgs[5], "patchiq-msi-") {
		t.Errorf("arg[5] = %q, want log path containing 'patchiq-msi-'", gotArgs[5])
	}
}

func TestMSIInstaller_Install_DryRun(t *testing.T) {
	mock := &mockExecutor{
		fn: func(_ context.Context, _ string, _ ...string) (ExecResult, error) {
			t.Fatal("executor should not be called during dry-run")
			return ExecResult{}, nil
		},
	}

	inst := &msiInstaller{executor: mock}
	result, err := inst.Install(context.Background(), PatchTarget{Name: `C:\patches\update.msi`}, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(result.Stdout), "dry-run") {
		t.Errorf("expected dry-run message in stdout, got %q", string(result.Stdout))
	}
}

func TestMSIInstaller_ExitCodeMapping(t *testing.T) {
	tests := []struct {
		name       string
		exitCode   int
		wantReboot bool
	}{
		{"success", 0, false},
		{"reboot initiated", 1641, true},
		{"reboot required", 3010, true},
		{"user cancelled", 1602, false},
		{"install in progress", 1618, false},
		{"generic failure", 1603, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockExecutor{
				fn: func(_ context.Context, _ string, _ ...string) (ExecResult, error) {
					return ExecResult{ExitCode: tt.exitCode}, nil
				},
			}

			inst := &msiInstaller{executor: mock}
			result, err := inst.Install(context.Background(), PatchTarget{Name: `C:\test.msi`}, false)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.RebootRequired != tt.wantReboot {
				t.Errorf("RebootRequired = %v, want %v", result.RebootRequired, tt.wantReboot)
			}
			if result.ExitCode != tt.exitCode {
				t.Errorf("ExitCode = %d, want %d", result.ExitCode, tt.exitCode)
			}
		})
	}
}
