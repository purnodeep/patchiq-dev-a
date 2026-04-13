package patcher

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
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

// osExecutor implements CommandExecutor using os/exec.
type osExecutor struct{}

func (e *osExecutor) Execute(ctx context.Context, name string, args ...string) (ExecResult, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := ExecResult{
		Stdout: stdout.Bytes(),
		Stderr: stderr.Bytes(),
	}

	if err != nil {
		if ctx.Err() != nil {
			return result, fmt.Errorf("execute %s: %w", name, ctx.Err())
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
			return result, nil
		}
		return result, fmt.Errorf("execute %s: %w", name, err)
	}

	return result, nil
}
