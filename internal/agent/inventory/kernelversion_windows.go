//go:build windows

package inventory

// collectKernelVersion returns the Windows kernel version.
// Windows hardware collection is not yet fully implemented; returns empty string.
func collectKernelVersion() string {
	return ""
}
