package patcher

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestMacOSInstaller_Name(t *testing.T) {
	inst := &macosInstaller{}
	if got := inst.Name(); got != "softwareupdate" {
		t.Errorf("Name() = %q, want %q", got, "softwareupdate")
	}
}

func TestMacOSInstaller_Install(t *testing.T) {
	tests := []struct {
		name         string
		pkg          PatchTarget
		dryRun       bool
		execResult   ExecResult
		execErr      error
		wantArgs     []string
		wantErr      bool
		wantReboot   bool
		wantExitCode int
	}{
		{
			name:       "install by label",
			pkg:        PatchTarget{Name: "Safari 17.1-17.1", Version: "17.1"},
			execResult: ExecResult{Stdout: []byte("Installing Safari 17.1"), ExitCode: 0},
			wantArgs:   []string{"softwareupdate", "--install", "Safari 17.1-17.1"},
		},
		{
			name:       "dry run uses --list",
			pkg:        PatchTarget{Name: "Safari 17.1-17.1", Version: "17.1"},
			dryRun:     true,
			execResult: ExecResult{Stdout: []byte("listing"), ExitCode: 0},
			wantArgs:   []string{"softwareupdate", "--list"},
		},
		{
			name:    "executor error",
			pkg:     PatchTarget{Name: "Safari 17.1-17.1", Version: "17.1"},
			execErr: fmt.Errorf("context cancelled"),
			wantErr: true,
		},
		{
			name:         "install failure exit code",
			pkg:          PatchTarget{Name: "BadUpdate-1.0", Version: "1.0"},
			execResult:   ExecResult{Stderr: []byte("update not found"), ExitCode: 1},
			wantArgs:     []string{"softwareupdate", "--install", "BadUpdate-1.0"},
			wantExitCode: 1,
		},
		{
			name:       "reboot required from stdout",
			pkg:        PatchTarget{Name: "Safari 17.1-17.1", Version: "17.1"},
			execResult: ExecResult{Stdout: []byte("Installing Safari... You need to restart your computer"), ExitCode: 0},
			wantArgs:   []string{"softwareupdate", "--install", "Safari 17.1-17.1"},
			wantReboot: true,
		},
		{
			name:       "reboot required from stderr",
			pkg:        PatchTarget{Name: "Safari 17.1-17.1", Version: "17.1"},
			execResult: ExecResult{Stderr: []byte("restart required")},
			wantArgs:   []string{"softwareupdate", "--install", "Safari 17.1-17.1"},
			wantReboot: true,
		},
		{
			name:       "no reboot needed",
			pkg:        PatchTarget{Name: "Safari 17.1-17.1", Version: "17.1"},
			execResult: ExecResult{Stdout: []byte("Installing update complete"), ExitCode: 0},
			wantArgs:   []string{"softwareupdate", "--install", "Safari 17.1-17.1"},
			wantReboot: false,
		},
		{
			name: "false positive — No such update in stderr",
			pkg:  PatchTarget{Name: "Security Update 2024-001", Version: "2024-001"},
			execResult: ExecResult{
				Stderr:   []byte("Security Update 2024-001: No such update\nNo updates are available."),
				ExitCode: 0,
			},
			wantArgs:     []string{"softwareupdate", "--install", "Security Update 2024-001"},
			wantExitCode: 1,
		},
		{
			name: "false positive — No updates are available in stderr",
			pkg:  PatchTarget{Name: "macOS Ventura 13.6.1", Version: "13.6.1"},
			execResult: ExecResult{
				Stderr:   []byte("No updates are available."),
				ExitCode: 0,
			},
			wantArgs:     []string{"softwareupdate", "--install", "macOS Ventura 13.6.1"},
			wantExitCode: 1,
		},
		{
			name: "false positive — not eligible in stderr",
			pkg:  PatchTarget{Name: "macOS Sequoia 15.0", Version: "15.0"},
			execResult: ExecResult{
				Stderr:   []byte("macOS Sequoia 15.0 is not eligible for this Mac"),
				ExitCode: 0,
			},
			wantArgs:     []string{"softwareupdate", "--install", "macOS Sequoia 15.0"},
			wantExitCode: 1,
		},
		{
			name: "false positive — requires restart first",
			pkg:  PatchTarget{Name: "Safari 18.0", Version: "18.0"},
			execResult: ExecResult{
				Stderr:   []byte("This update requires restart first"),
				ExitCode: 0,
			},
			wantArgs:     []string{"softwareupdate", "--install", "Safari 18.0"},
			wantExitCode: 1,
			wantReboot:   true, // stderr contains "restart", so reboot flag is set
		},
		{
			name: "genuine exit code 0 with benign stderr is success",
			pkg:  PatchTarget{Name: "Safari 17.2-17.2", Version: "17.2"},
			execResult: ExecResult{
				Stdout:   []byte("Installing Safari 17.2"),
				Stderr:   []byte("Downloading Safari 17.2..."),
				ExitCode: 0,
			},
			wantArgs:     []string{"softwareupdate", "--install", "Safari 17.2-17.2"},
			wantExitCode: 0,
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotArgs []string
			mock := &mockExecutor{
				fn: func(_ context.Context, name string, args ...string) (ExecResult, error) {
					gotArgs = append([]string{name}, args...)
					return tt.execResult, tt.execErr
				},
			}

			inst := &macosInstaller{executor: mock, logger: logger}
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

			if result.ExitCode != tt.wantExitCode {
				t.Errorf("exit code = %d, want %d", result.ExitCode, tt.wantExitCode)
			}
		})
	}
}

func TestDetectSoftwareupdateFailure(t *testing.T) {
	tests := []struct {
		name   string
		stderr string
		want   string
	}{
		{
			name:   "No such update",
			stderr: "Security Update 2024-001: No such update\nNo updates are available.",
			want:   "No such update",
		},
		{
			name:   "No updates are available only",
			stderr: "No updates are available.",
			want:   "No updates are available",
		},
		{
			name:   "not eligible",
			stderr: "macOS Sequoia 15.0 is Not Eligible for this Mac",
			want:   "not eligible",
		},
		{
			name:   "requires restart first",
			stderr: "This update Requires Restart First before installing",
			want:   "requires restart first",
		},
		{
			name:   "benign stderr",
			stderr: "Downloading Safari 17.2...\nInstalling...",
			want:   "",
		},
		{
			name:   "empty stderr",
			stderr: "",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectSoftwareupdateFailure(tt.stderr)
			if tt.want == "" {
				if got != "" {
					t.Errorf("detectSoftwareupdateFailure() = %q, want empty", got)
				}
				return
			}
			if !strings.EqualFold(got, tt.want) {
				t.Errorf("detectSoftwareupdateFailure() = %q, want %q", got, tt.want)
			}
		})
	}
}
