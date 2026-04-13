//go:build windows

package executor

import "os/exec"

// setProcGroup is a no-op on Windows; job objects would be needed for full
// descendant kill, which is out of scope for this fix.
func setProcGroup(_ *exec.Cmd) {}

// cancelFunc returns a function that kills the direct process on Windows.
func cancelFunc(cmd *exec.Cmd) func() error {
	return func() error {
		return cmd.Process.Kill()
	}
}
