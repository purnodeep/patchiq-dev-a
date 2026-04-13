//go:build windows

package agent

import "golang.org/x/sys/windows"

// IsRoot reports whether the current process is running with admin (elevated) privileges.
func IsRoot() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}
