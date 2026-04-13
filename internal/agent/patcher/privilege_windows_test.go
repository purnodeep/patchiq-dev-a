//go:build windows

package patcher

import (
	"errors"
	"testing"
)

func TestCheckAdminPrivilege_ReportsCorrectly(t *testing.T) {
	// This test runs on the actual system — just verify it returns
	// a consistent result (either nil or errNotAdmin, no panics).
	err := checkAdminPrivilege()
	if err != nil && !errors.Is(err, errNotAdmin) {
		t.Errorf("unexpected error type: %v", err)
	}
	t.Logf("checkAdminPrivilege() returned: %v", err)
}

func TestCheckWUAServiceRunning(t *testing.T) {
	// On a normal Windows machine the WUA service should be queryable.
	// We don't assert running/stopped since CI may differ, just no panic.
	err := checkWUAServiceRunning()
	t.Logf("checkWUAServiceRunning() returned: %v", err)
}
