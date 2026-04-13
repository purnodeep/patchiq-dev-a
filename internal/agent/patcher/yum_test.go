package patcher

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
)

var testLogger = slog.New(slog.NewTextHandler(os.Stderr, nil))

func TestYumInstaller_Install(t *testing.T) {
	tests := []struct {
		name       string
		binary     string
		pkg        PatchTarget
		dryRun     bool
		execResult ExecResult
		execErr    error
		wantArgs   []string
		wantErr    bool
	}{
		{
			name:       "yum install success",
			binary:     "yum",
			pkg:        PatchTarget{Name: "openssl", Version: "3.0.7-24.el9"},
			execResult: ExecResult{Stdout: []byte("installed"), ExitCode: 0},
			wantArgs:   []string{"yum", "install", "-y", "openssl-3.0.7-24.el9"},
		},
		{
			name:       "dnf install success",
			binary:     "dnf",
			pkg:        PatchTarget{Name: "curl", Version: "8.0.1-1.el9"},
			execResult: ExecResult{Stdout: []byte("installed"), ExitCode: 0},
			wantArgs:   []string{"dnf", "install", "-y", "curl-8.0.1-1.el9"},
		},
		{
			name:       "yum dry run",
			binary:     "yum",
			pkg:        PatchTarget{Name: "openssl", Version: "3.0.7"},
			dryRun:     true,
			execResult: ExecResult{Stdout: []byte("simulated"), ExitCode: 0},
			wantArgs:   []string{"yum", "install", "--assumeno", "openssl-3.0.7"},
		},
		{
			name:       "install failure",
			binary:     "dnf",
			pkg:        PatchTarget{Name: "foo", Version: "1.0"},
			execResult: ExecResult{Stderr: []byte("No match"), ExitCode: 1},
			wantArgs:   []string{"dnf", "install", "-y", "foo-1.0"},
		},
		{
			name:    "executor error",
			binary:  "yum",
			pkg:     PatchTarget{Name: "curl", Version: "1.0"},
			execErr: fmt.Errorf("context cancelled"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotArgs []string
			callCount := 0
			mock := &mockExecutor{
				fn: func(_ context.Context, name string, args ...string) (ExecResult, error) {
					callCount++
					if callCount == 1 {
						gotArgs = append([]string{name}, args...)
						return tt.execResult, tt.execErr
					}
					// needs-restarting: exit 0 = no reboot
					return ExecResult{ExitCode: 0}, nil
				},
			}

			installer := &yumInstaller{executor: mock, logger: testLogger, binary: tt.binary}
			result, err := installer.Install(context.Background(), tt.pkg, tt.dryRun)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(gotArgs) != len(tt.wantArgs) {
				t.Fatalf("args = %v, want %v", gotArgs, tt.wantArgs)
			}
			for i, arg := range tt.wantArgs {
				if gotArgs[i] != arg {
					t.Errorf("arg[%d] = %q, want %q", i, gotArgs[i], arg)
				}
			}

			if result.ExitCode != tt.execResult.ExitCode {
				t.Errorf("exit code = %d, want %d", result.ExitCode, tt.execResult.ExitCode)
			}
		})
	}
}

func TestYumInstaller_RebootRequired(t *testing.T) {
	callCount := 0
	mock := &mockExecutor{
		fn: func(_ context.Context, name string, args ...string) (ExecResult, error) {
			callCount++
			if callCount == 1 {
				return ExecResult{ExitCode: 0}, nil
			}
			// needs-restarting -r returns exit code 1 = reboot needed
			return ExecResult{ExitCode: 1}, nil
		},
	}

	installer := &yumInstaller{executor: mock, logger: testLogger, binary: "dnf"}
	result, err := installer.Install(context.Background(), PatchTarget{Name: "kernel", Version: "6.1"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.RebootRequired {
		t.Error("expected reboot required = true")
	}
}

func TestYumInstaller_NoRebootCheckOnFailure(t *testing.T) {
	callCount := 0
	mock := &mockExecutor{
		fn: func(_ context.Context, name string, args ...string) (ExecResult, error) {
			callCount++
			return ExecResult{ExitCode: 1}, nil
		},
	}

	installer := &yumInstaller{executor: mock, logger: testLogger, binary: "dnf"}
	result, err := installer.Install(context.Background(), PatchTarget{Name: "badpkg", Version: "1.0"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 call (no needs-restarting check), got %d", callCount)
	}
	if result.RebootRequired {
		t.Error("expected reboot required = false on failed install")
	}
}

func TestYumInstaller_Name(t *testing.T) {
	tests := []struct {
		binary string
		want   string
	}{
		{"yum", "yum"},
		{"dnf", "dnf"},
	}
	for _, tt := range tests {
		t.Run(tt.binary, func(t *testing.T) {
			installer := &yumInstaller{binary: tt.binary}
			if got := installer.Name(); got != tt.want {
				t.Errorf("Name() = %q, want %q", got, tt.want)
			}
		})
	}
}
