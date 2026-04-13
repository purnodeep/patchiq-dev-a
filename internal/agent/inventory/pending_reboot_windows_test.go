//go:build windows

package inventory

import (
	"context"
	"log/slog"
	"os"
	"testing"
)

type mockRebootChecker struct {
	existingKeys map[string]bool
}

func (m *mockRebootChecker) KeyExists(path string) bool {
	return m.existingKeys[path]
}

func TestPendingRebootCollector_Name(t *testing.T) {
	c := &pendingRebootCollector{}
	if c.Name() != "pending_reboot" {
		t.Errorf("Name() = %q, want %q", c.Name(), "pending_reboot")
	}
}

func TestPendingRebootCollector_RebootPending(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	checker := &mockRebootChecker{
		existingKeys: map[string]bool{
			`SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update\RebootRequired`: true,
		},
	}
	c := &pendingRebootCollector{checker: checker, logger: logger}
	pkgs, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("got %d packages, want 1", len(pkgs))
	}
	if pkgs[0].Name != "REBOOT_PENDING" {
		t.Errorf("name = %q, want %q", pkgs[0].Name, "REBOOT_PENDING")
	}
	if pkgs[0].Source != "system" {
		t.Errorf("source = %q, want %q", pkgs[0].Source, "system")
	}
	if pkgs[0].Status != "pending" {
		t.Errorf("status = %q, want %q", pkgs[0].Status, "pending")
	}
}

func TestPendingRebootCollector_NoReboot(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	checker := &mockRebootChecker{existingKeys: map[string]bool{}}
	c := &pendingRebootCollector{checker: checker, logger: logger}
	pkgs, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 0 {
		t.Errorf("got %d packages, want 0", len(pkgs))
	}
}

func TestPendingRebootCollector_IncludesCategory(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	checker := &mockRebootChecker{
		existingKeys: map[string]bool{
			`SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update\RebootRequired`: true,
		},
	}
	c := &pendingRebootCollector{checker: checker, logger: logger}
	pkgs, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("expected 1, got %d", len(pkgs))
	}
	if pkgs[0].Category != "WindowsUpdate" {
		t.Errorf("Category = %q, want WindowsUpdate", pkgs[0].Category)
	}
}

func TestPendingRebootCollector_MultipleSources(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	checker := &mockRebootChecker{
		existingKeys: map[string]bool{
			`SOFTWARE\Microsoft\Windows\CurrentVersion\Component Based Servicing\RebootPending`: true,
			`SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update\RebootRequired`: true,
		},
	}
	c := &pendingRebootCollector{checker: checker, logger: logger}
	pkgs, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(pkgs) != 2 {
		t.Fatalf("expected 2, got %d", len(pkgs))
	}
}
