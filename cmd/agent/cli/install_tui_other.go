//go:build !linux && !windows

package cli

import "fmt"

// installAndStartService on unsupported platforms (currently macOS and others)
// returns an error so the install wizard surfaces a clear failure instead of
// silently claiming success. When LaunchdService lands, add a darwin build
// variant analogous to install_tui_linux.go.
func installAndStartService() error {
	return fmt.Errorf("install service: not supported on this platform yet; run 'patchiq-agent service install' manually once support is added")
}
