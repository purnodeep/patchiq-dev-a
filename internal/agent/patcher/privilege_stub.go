//go:build !windows

package patcher

import "errors"

// errNotAdmin is referenced by cross-platform msix.go tests; keep on all OSes.
var errNotAdmin = errors.New("patcher: administrator privileges required")

// checkAdmin is referenced by cross-platform msix.go; no-op on non-Windows.
var checkAdmin = func() error { return nil }
