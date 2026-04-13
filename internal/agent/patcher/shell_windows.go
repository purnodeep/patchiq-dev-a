//go:build windows

package patcher

// scriptShellArgs returns the shell command and arguments for executing
// pre/post scripts on Windows via PowerShell.
func scriptShellArgs(script string) (string, []string) {
	return "powershell.exe", []string{"-NoProfile", "-NonInteractive", "-Command", script}
}
