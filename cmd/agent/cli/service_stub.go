//go:build !linux && !windows

package cli

import (
	"fmt"
	"os"
)

// RunService is a stub for unsupported platforms.
func RunService(_ []string) int {
	fmt.Fprintln(os.Stderr, "service management is not available on this platform")
	return ExitError
}
