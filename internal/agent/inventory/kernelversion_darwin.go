//go:build darwin

package inventory

import (
	"os/exec"
	"strings"
)

// collectKernelVersion returns the Darwin kernel version via uname -r,
// e.g. "25.3.0".
func collectKernelVersion() string {
	out, err := exec.Command("uname", "-r").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
