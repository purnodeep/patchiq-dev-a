//go:build !windows

package patcher

// scriptShellArgs returns the shell command and arguments for executing
// pre/post scripts on Unix.
func scriptShellArgs(script string) (string, []string) {
	return "sh", []string{"-c", script}
}
