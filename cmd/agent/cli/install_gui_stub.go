//go:build !linux && !windows

package cli

// HasZenity returns false on non-Linux platforms.
func HasZenity() bool { return false }

// RunGUIInstall falls back to the standard install on non-Linux platforms.
func RunGUIInstall(args []string) int { return RunInstall(args) }

// ShowAlreadyEnrolledDialog is a no-op on non-Linux platforms.
func ShowAlreadyEnrolledDialog() int { return 0 }
