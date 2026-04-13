//go:build windows && integration

package integration

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestWindowsAgent_ExecutorCrossPlatform verifies that os/exec works correctly
// on Windows with native commands (cmd.exe, powershell.exe).
func TestWindowsAgent_ExecutorCrossPlatform(t *testing.T) {
	t.Run("echo", func(t *testing.T) {
		out, err := exec.CommandContext(context.Background(), "cmd.exe", "/C", "echo hello").Output()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := strings.TrimSpace(string(out))
		if got != "hello" {
			t.Errorf("stdout = %q, want %q", got, "hello")
		}
	})

	t.Run("exit_code", func(t *testing.T) {
		err := exec.CommandContext(context.Background(), "cmd.exe", "/C", "exit /b 42").Run()
		if err == nil {
			t.Fatal("expected error for non-zero exit")
		}
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("expected *exec.ExitError, got %T", err)
		}
		if exitErr.ExitCode() != 42 {
			t.Errorf("exit code = %d, want 42", exitErr.ExitCode())
		}
	})

	t.Run("context_timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		err := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-Command", "Start-Sleep -Seconds 30").Run()
		if err == nil {
			t.Fatal("expected error for timed-out command")
		}
	})
}

// TestWindowsAgent_PowerShellAvailable verifies PowerShell is available.
func TestWindowsAgent_PowerShellAvailable(t *testing.T) {
	path, err := exec.LookPath("powershell.exe")
	if err != nil {
		t.Fatalf("powershell.exe not found: %v", err)
	}
	t.Logf("powershell.exe at: %s", path)
}

// TestWindowsAgent_MsiexecAvailable verifies msiexec is available.
func TestWindowsAgent_MsiexecAvailable(t *testing.T) {
	path, err := exec.LookPath("msiexec")
	if err != nil {
		t.Fatalf("msiexec not found: %v", err)
	}
	t.Logf("msiexec at: %s", path)
}

// TestWindowsAgent_WUAServiceQueryable verifies Windows Update service can be queried.
func TestWindowsAgent_WUAServiceQueryable(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("skipping real WUA test in CI")
	}

	out, err := exec.CommandContext(context.Background(),
		"powershell.exe", "-NoProfile", "-Command",
		"Get-Service wuauserv | Select-Object Status | ConvertTo-Json").Output()
	if err != nil {
		t.Fatalf("failed to query wuauserv: %v", err)
	}
	t.Logf("wuauserv status: %s", strings.TrimSpace(string(out)))
}

// TestWindowsAgent_PendingRebootCheck verifies pending reboot registry check.
func TestWindowsAgent_PendingRebootCheck(t *testing.T) {
	out, err := exec.CommandContext(context.Background(),
		"powershell.exe", "-NoProfile", "-Command",
		`Test-Path "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update\RebootRequired"`).Output()
	if err != nil {
		t.Logf("reboot check failed (expected on non-admin): %v", err)
		return
	}
	result := strings.TrimSpace(string(out))
	t.Logf("reboot pending: %s", result)
}
