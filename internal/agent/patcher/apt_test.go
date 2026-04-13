package patcher

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
)

func TestAPTInstaller_Install(t *testing.T) {
	tests := []struct {
		name         string
		pkg          PatchTarget
		dryRun       bool
		execResults  []ExecResult // one per call (update, install, optional fallback)
		execErrs     []error
		wantAllCalls [][]string // all expected calls in order
		wantExitCode int
		wantErr      bool
	}{
		{
			name: "install success",
			pkg:  PatchTarget{Name: "curl", Version: "7.88.1-10+deb12u5"},
			execResults: []ExecResult{
				{ExitCode: 0}, // apt-get update
				{Stdout: []byte("installed"), ExitCode: 0}, // apt-get install (versioned)
			},
			execErrs: []error{nil, nil},
			wantAllCalls: [][]string{
				{"apt-get", "update"},
				{"apt-get", "install", "-y", "curl=7.88.1-10+deb12u5"},
			},
			wantExitCode: 0,
		},
		{
			name: "install failure with fallback",
			pkg:  PatchTarget{Name: "curl", Version: "7.88.1-10+deb12u5"},
			execResults: []ExecResult{
				{ExitCode: 0}, // apt-get update
				{Stderr: []byte("not found"), ExitCode: 100}, // versioned install fails
				{Stderr: []byte("not found"), ExitCode: 100}, // fallback also fails
			},
			execErrs: []error{nil, nil, nil},
			wantAllCalls: [][]string{
				{"apt-get", "update"},
				{"apt-get", "install", "-y", "curl=7.88.1-10+deb12u5"},
				{"apt-get", "install", "-y", "curl"},
			},
			wantExitCode: 100,
		},
		{
			name:   "dry run",
			pkg:    PatchTarget{Name: "curl", Version: "7.88.1-10+deb12u5"},
			dryRun: true,
			execResults: []ExecResult{
				{ExitCode: 0}, // apt-get update
				{Stdout: []byte("simulated"), ExitCode: 0}, // dry run
			},
			execErrs: []error{nil, nil},
			wantAllCalls: [][]string{
				{"apt-get", "update"},
				{"apt-get", "install", "--dry-run", "curl=7.88.1-10+deb12u5"},
			},
			wantExitCode: 0,
		},
		{
			name: "executor error",
			pkg:  PatchTarget{Name: "curl", Version: "1.0"},
			execResults: []ExecResult{
				{ExitCode: 0}, // apt-get update
				{},            // install (error)
			},
			execErrs: []error{nil, fmt.Errorf("context cancelled")},
			wantErr:  true,
		},
		{
			name: "install without version",
			pkg:  PatchTarget{Name: "curl"},
			execResults: []ExecResult{
				{ExitCode: 0}, // apt-get update
				{Stdout: []byte("installed"), ExitCode: 0}, // install (no version)
			},
			execErrs: []error{nil, nil},
			wantAllCalls: [][]string{
				{"apt-get", "update"},
				{"apt-get", "install", "-y", "curl"},
			},
			wantExitCode: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var allCalls [][]string
			callIdx := 0
			mock := &mockExecutor{
				fn: func(_ context.Context, name string, args ...string) (ExecResult, error) {
					call := append([]string{name}, args...)
					allCalls = append(allCalls, call)
					idx := callIdx
					callIdx++
					if idx < len(tt.execResults) {
						var execErr error
						if idx < len(tt.execErrs) {
							execErr = tt.execErrs[idx]
						}
						return tt.execResults[idx], execErr
					}
					return ExecResult{}, nil
				},
			}

			installer := &aptInstaller{
				executor:           mock,
				logger:             slog.Default(),
				rebootRequiredPath: "/nonexistent/reboot-required",
			}
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

			if tt.wantAllCalls != nil {
				if len(allCalls) != len(tt.wantAllCalls) {
					t.Fatalf("got %d calls, want %d:\n  got:  %v\n  want: %v", len(allCalls), len(tt.wantAllCalls), allCalls, tt.wantAllCalls)
				}
				for i, wantCall := range tt.wantAllCalls {
					gotCall := allCalls[i]
					if len(gotCall) != len(wantCall) {
						t.Errorf("call[%d] = %v, want %v", i, gotCall, wantCall)
						continue
					}
					for j, arg := range wantCall {
						if gotCall[j] != arg {
							t.Errorf("call[%d] arg[%d] = %q, want %q", i, j, gotCall[j], arg)
						}
					}
				}
			}

			if result.ExitCode != tt.wantExitCode {
				t.Errorf("exit code = %d, want %d", result.ExitCode, tt.wantExitCode)
			}
		})
	}
}

func TestAPTInstaller_RebootRequired(t *testing.T) {
	tmp := t.TempDir() + "/reboot-required"
	if err := os.WriteFile(tmp, []byte("*** System restart required ***"), 0644); err != nil {
		t.Fatal(err)
	}

	mock := &mockExecutor{
		fn: func(_ context.Context, _ string, _ ...string) (ExecResult, error) {
			return ExecResult{ExitCode: 0}, nil
		},
	}

	installer := &aptInstaller{
		executor:           mock,
		logger:             slog.Default(),
		rebootRequiredPath: tmp,
	}
	result, err := installer.Install(context.Background(), PatchTarget{Name: "linux-image", Version: "6.1"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.RebootRequired {
		t.Error("expected reboot required = true")
	}
}

// mockExecutor implements CommandExecutor for tests.
type mockExecutor struct {
	fn func(ctx context.Context, name string, args ...string) (ExecResult, error)
}

func (m *mockExecutor) Execute(ctx context.Context, name string, args ...string) (ExecResult, error) {
	return m.fn(ctx, name, args...)
}
