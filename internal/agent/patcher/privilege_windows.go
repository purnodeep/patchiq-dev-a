//go:build windows

package patcher

import (
	"errors"
	"fmt"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

var (
	errNotAdmin          = errors.New("patcher: administrator privileges required")
	errWUAServiceStopped = errors.New("patcher: Windows Update service (wuauserv) is not running")
)

// checkAdmin and checkWUAService are package-level function variables
// so tests can override them.
var checkAdmin = checkAdminPrivilege
var checkWUAService = checkWUAServiceRunning

func checkAdminPrivilege() error {
	token := windows.Token(0)
	if !token.IsElevated() {
		return errNotAdmin
	}
	return nil
}

func checkWUAServiceRunning() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("patcher: connect to service manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService("wuauserv")
	if err != nil {
		return fmt.Errorf("patcher: open wuauserv service: %w", err)
	}
	defer s.Close()

	status, err := s.Query()
	if err != nil {
		return fmt.Errorf("patcher: query wuauserv status: %w", err)
	}

	if status.State != svc.Running {
		return errWUAServiceStopped
	}
	return nil
}
