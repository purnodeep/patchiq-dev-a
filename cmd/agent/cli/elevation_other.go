//go:build !windows

package cli

// isAdmin always returns true on non-Windows platforms. Linux and macOS
// install paths handle their own privilege checks (geteuid).
func isAdmin() bool {
	return true
}

// IsAdmin is the exported wrapper (always true on non-Windows).
func IsAdmin() bool {
	return isAdmin()
}

// RelaunchAsAdmin is a no-op on non-Windows platforms. Linux and macOS
// install paths use sudo or pkexec at invocation time rather than
// Windows-style UAC self-elevation. Returns nil unconditionally so the
// caller can treat this as "already elevated, proceed".
func RelaunchAsAdmin() error {
	return nil
}
