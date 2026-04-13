//go:build windows

package cli

import (
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/windows"
)

// isAdmin reports whether the current process is running with administrator
// privileges. Required to install a Windows service.
func isAdmin() bool {
	var sid *windows.SID
	// S-1-5-32-544 is the well-known SID for the local Administrators group.
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid,
	)
	if err != nil {
		return false
	}
	defer windows.FreeSid(sid)

	token := windows.Token(0) // current process token
	member, err := token.IsMember(sid)
	if err != nil {
		return false
	}
	return member
}

// IsAdmin is the exported wrapper for the internal admin check.
// Allows callers outside this package (e.g. cmd/agent/main.go) to
// reuse the same logic without duplicating the Win32 SID dance.
func IsAdmin() bool {
	return isAdmin()
}

// RelaunchAsAdmin re-invokes the current executable with administrator
// privileges via Windows ShellExecute("runas"). Windows shows the UAC
// prompt to the user; if approved, a new elevated instance of this exe
// launches in a fresh console window. The caller (the non-elevated
// parent) should os.Exit(0) immediately after a successful relaunch.
//
// Returns an error if the ShellExecute call itself fails (e.g. user
// clicks "No" on UAC — windows reports ERROR_CANCELLED).
func RelaunchAsAdmin() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path for elevation: %w", err)
	}

	verb, err := syscall.UTF16PtrFromString("runas")
	if err != nil {
		return fmt.Errorf("encode verb: %w", err)
	}
	file, err := syscall.UTF16PtrFromString(exe)
	if err != nil {
		return fmt.Errorf("encode exe path: %w", err)
	}

	// SW_SHOWNORMAL = 1 — open the new elevated process in a normal
	// window so the operator sees the TUI wizard console.
	const swShowNormal = 1
	if err := windows.ShellExecute(0, verb, file, nil, nil, swShowNormal); err != nil {
		return fmt.Errorf("shell execute runas: %w", err)
	}
	return nil
}
