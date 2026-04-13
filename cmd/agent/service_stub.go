//go:build !windows

package main

func isWindowsService() bool {
	return false
}

func runAsWindowsService(_ string) {
	// unreachable on non-Windows
}
