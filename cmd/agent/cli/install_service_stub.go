//go:build !linux

package cli

// installService is a no-op on platforms without service management.
func installService(_ string, _ func(string)) error {
	return nil
}
