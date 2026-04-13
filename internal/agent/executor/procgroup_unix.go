//go:build !windows

package executor

import (
	"os/exec"
	"syscall"
)

// setProcGroup places the command in its own process group so all descendants
// can be killed together on timeout.
func setProcGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// cancelFunc returns a function that kills the entire process group.
func cancelFunc(cmd *exec.Cmd) func() error {
	return func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
