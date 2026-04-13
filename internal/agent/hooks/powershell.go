package hooks

import (
	"context"
	"time"
)

// ExecResult holds the output of a command execution.
type ExecResult struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

// CommandExecutor abstracts command execution for testability.
type CommandExecutor interface {
	Execute(ctx context.Context, name string, args ...string) (ExecResult, error)
}

// PowerShellExecutor runs PowerShell scripts with timeout support.
type PowerShellExecutor struct {
	executor CommandExecutor
}

// NewPowerShellExecutor creates a new PowerShell hook executor.
func NewPowerShellExecutor(executor CommandExecutor) *PowerShellExecutor {
	return &PowerShellExecutor{executor: executor}
}

// Run executes a PowerShell script with the given arguments and timeout.
func (e *PowerShellExecutor) Run(ctx context.Context, scriptPath string, scriptArgs []string, timeout time.Duration) (ExecResult, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	args := []string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-File", scriptPath}
	args = append(args, scriptArgs...)

	return e.executor.Execute(ctx, "powershell.exe", args...)
}
