//go:build windows

package cli

import (
	"testing"
)

func TestRunService_InvalidAction(t *testing.T) {
	code := RunService([]string{"invalid-action"})
	if code != ExitError {
		t.Errorf("RunService('invalid-action') = %d, want %d", code, ExitError)
	}
}

func TestRunService_NoArgs(t *testing.T) {
	code := RunService(nil)
	if code != ExitError {
		t.Errorf("RunService(nil) = %d, want %d", code, ExitError)
	}
}

func TestRunService_EmptyArgs(t *testing.T) {
	code := RunService([]string{})
	if code != ExitError {
		t.Errorf("RunService([]) = %d, want %d", code, ExitError)
	}
}
