package patcher

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"
	"time"
)

// testShellCmd returns platform-appropriate shell command args.
func testShellCmd(script string) (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd.exe", []string{"/C", script}
	}
	return "sh", []string{"-c", script}
}

// testSleepCmd returns a long-running command for context cancellation tests.
func testSleepCmd() (string, []string) {
	if runtime.GOOS == "windows" {
		return "powershell.exe", []string{"-NoProfile", "-Command", "Start-Sleep -Seconds 30"}
	}
	return "sleep", []string{"10"}
}

// trimOutput normalizes line endings and trailing whitespace for cross-platform comparison.
// Windows cmd.exe echo adds trailing spaces before redirects (e.g. "echo fail 1>&2" outputs "fail ").
func trimOutput(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, "\r\n", "\n"))
}

func TestOSExecutor_Execute_echo(t *testing.T) {
	name, args := testShellCmd("echo hello")
	exec := &osExecutor{}
	result, err := exec.Execute(context.Background(), name, args...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0", result.ExitCode)
	}
	got := trimOutput(string(result.Stdout))
	if got != "hello" {
		t.Errorf("stdout = %q, want %q", got, "hello")
	}
}

func TestOSExecutor_Execute_failure(t *testing.T) {
	var script string
	if runtime.GOOS == "windows" {
		script = "echo fail 1>&2 & exit /b 42"
	} else {
		script = "echo fail >&2; exit 42"
	}
	name, args := testShellCmd(script)

	exec := &osExecutor{}
	result, err := exec.Execute(context.Background(), name, args...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 42 {
		t.Errorf("exit code = %d, want 42", result.ExitCode)
	}
	got := trimOutput(string(result.Stderr))
	if got != "fail" {
		t.Errorf("stderr = %q, want %q", got, "fail")
	}
}

func TestOSExecutor_Execute_context_cancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	name, args := testSleepCmd()
	exec := &osExecutor{}
	_, err := exec.Execute(ctx, name, args...)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want wrapping context.Canceled", err)
	}
}

func TestOSExecutor_Execute_context_deadline_exceeded(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	name, args := testSleepCmd()
	exec := &osExecutor{}
	_, err := exec.Execute(ctx, name, args...)
	if err == nil {
		t.Fatal("expected error for deadline exceeded context")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("error = %v, want wrapping context.DeadlineExceeded", err)
	}
}

func TestOSExecutor_Execute_failure_without_context_cancellation(t *testing.T) {
	var script string
	if runtime.GOOS == "windows" {
		script = "exit /b 1"
	} else {
		script = "exit 1"
	}
	name, args := testShellCmd(script)

	exec := &osExecutor{}
	result, err := exec.Execute(context.Background(), name, args...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 1 {
		t.Errorf("exit code = %d, want 1", result.ExitCode)
	}
}

func TestOSExecutor_Execute_not_found(t *testing.T) {
	exec := &osExecutor{}
	_, err := exec.Execute(context.Background(), "nonexistent-binary-xyz")
	if err == nil {
		t.Fatal("expected error for missing binary")
	}
}

func TestOSExecutor_Execute_mixed_output(t *testing.T) {
	var script string
	if runtime.GOOS == "windows" {
		script = "echo out & echo err 1>&2"
	} else {
		script = "echo out; echo err >&2"
	}
	name, args := testShellCmd(script)

	exec := &osExecutor{}
	result, err := exec.Execute(context.Background(), name, args...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if trimOutput(string(result.Stdout)) != "out" {
		t.Errorf("stdout = %q, want %q", string(result.Stdout), "out")
	}
	if trimOutput(string(result.Stderr)) != "err" {
		t.Errorf("stderr = %q, want %q", string(result.Stderr), "err")
	}
	if result.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0", result.ExitCode)
	}
}
