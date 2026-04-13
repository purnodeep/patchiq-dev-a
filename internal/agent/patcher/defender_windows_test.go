//go:build windows

package patcher

import (
	"context"
	"testing"
)

func TestDefenderAddExclusion_Success(t *testing.T) {
	var executedCmd string
	mock := &mockExecutor{
		fn: func(_ context.Context, name string, args ...string) (ExecResult, error) {
			executedCmd = name
			return ExecResult{ExitCode: 0}, nil
		},
	}
	err := defenderAddExclusion(context.Background(), mock, `C:\temp\patch.msi`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if executedCmd != "powershell.exe" {
		t.Errorf("expected powershell.exe, got %q", executedCmd)
	}
}

func TestDefenderAddExclusion_NonZeroExit(t *testing.T) {
	mock := &mockExecutor{
		fn: func(_ context.Context, _ string, _ ...string) (ExecResult, error) {
			return ExecResult{ExitCode: 1, Stderr: []byte("access denied")}, nil
		},
	}
	// Non-zero exit is non-fatal — should return nil.
	err := defenderAddExclusion(context.Background(), mock, `C:\temp\patch.msi`)
	if err != nil {
		t.Fatalf("expected nil for non-zero exit (non-fatal), got: %v", err)
	}
}

func TestDefenderRemoveExclusion_Success(t *testing.T) {
	mock := &mockExecutor{
		fn: func(_ context.Context, _ string, _ ...string) (ExecResult, error) {
			return ExecResult{ExitCode: 0}, nil
		},
	}
	err := defenderRemoveExclusion(context.Background(), mock, `C:\temp\patch.msi`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
