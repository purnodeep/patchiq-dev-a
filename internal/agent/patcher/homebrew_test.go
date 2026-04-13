package patcher

import (
	"context"
	"fmt"
	"testing"
)

func TestHomebrewInstaller_Name(t *testing.T) {
	inst := &homebrewInstaller{}
	if got := inst.Name(); got != "homebrew" {
		t.Errorf("Name() = %q, want %q", got, "homebrew")
	}
}

func TestHomebrewInstaller_Install(t *testing.T) {
	tests := []struct {
		name       string
		pkg        PatchTarget
		dryRun     bool
		execResult ExecResult
		execErr    error
		wantArgs   []string
		wantErr    bool
	}{
		{
			name:       "upgrade success",
			pkg:        PatchTarget{Name: "git", Version: "2.43.1"},
			execResult: ExecResult{Stdout: []byte("Upgrading git"), ExitCode: 0},
			wantArgs:   []string{"brew", "upgrade", "git"},
		},
		{
			name:       "upgrade with specific version uses brew upgrade",
			pkg:        PatchTarget{Name: "curl", Version: "8.1.0"},
			execResult: ExecResult{Stdout: []byte("Upgrading curl"), ExitCode: 0},
			wantArgs:   []string{"brew", "upgrade", "curl"},
		},
		{
			name:       "dry run",
			pkg:        PatchTarget{Name: "git", Version: "2.43.1"},
			dryRun:     true,
			execResult: ExecResult{Stdout: []byte("Would upgrade"), ExitCode: 0},
			wantArgs:   []string{"brew", "upgrade", "--dry-run", "git"},
		},
		{
			name:    "executor error",
			pkg:     PatchTarget{Name: "git", Version: "2.43.1"},
			execErr: fmt.Errorf("brew not found"),
			wantErr: true,
		},
		{
			name:       "upgrade failure",
			pkg:        PatchTarget{Name: "badpkg", Version: "1.0"},
			execResult: ExecResult{Stderr: []byte("No available formula"), ExitCode: 1},
			wantArgs:   []string{"brew", "upgrade", "badpkg"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotArgs []string
			mock := &mockExecutor{
				fn: func(_ context.Context, name string, args ...string) (ExecResult, error) {
					gotArgs = append([]string{name}, args...)
					return tt.execResult, tt.execErr
				},
			}

			inst := &homebrewInstaller{executor: mock}
			result, err := inst.Install(context.Background(), tt.pkg, tt.dryRun)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantArgs != nil {
				if len(gotArgs) != len(tt.wantArgs) {
					t.Fatalf("args = %v, want %v", gotArgs, tt.wantArgs)
				}
				for i, arg := range tt.wantArgs {
					if gotArgs[i] != arg {
						t.Errorf("arg[%d] = %q, want %q", i, gotArgs[i], arg)
					}
				}
			}

			if result.ExitCode != tt.execResult.ExitCode {
				t.Errorf("exit code = %d, want %d", result.ExitCode, tt.execResult.ExitCode)
			}

			if result.RebootRequired {
				t.Error("expected RebootRequired = false for homebrew")
			}
		})
	}
}
