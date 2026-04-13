//go:build !windows

package agent

import "os"

// IsRoot reports whether the current process is running with root/admin privileges.
func IsRoot() bool {
	return os.Geteuid() == 0
}
