//go:build linux

package cli

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// zenityRunner abstracts zenity subprocess execution for testability.
type zenityRunner interface {
	// Run executes zenity with the given arguments and returns stdout.
	// A non-nil error with exit code 1 means the user cancelled the dialog.
	Run(args ...string) (stdout string, err error)
	// Start begins a zenity process and returns its stdin pipe for writing.
	Start(args ...string) (stdin io.WriteCloser, cancel func(), err error)
}

// defaultZenityRunner implements zenityRunner using real exec.Command.
type defaultZenityRunner struct{}

func (d defaultZenityRunner) Run(args ...string) (string, error) {
	cmd := exec.Command("zenity", args...)
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

func (d defaultZenityRunner) Start(args ...string) (io.WriteCloser, func(), error) {
	cmd := exec.Command("zenity", args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("zenity stdin pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("zenity start: %w", err)
	}
	cancel := func() {
		stdin.Close()
		_ = cmd.Wait()
	}
	return stdin, cancel, nil
}

// HasZenity returns true if the zenity binary is in PATH.
func HasZenity() bool {
	_, err := exec.LookPath("zenity")
	return err == nil
}

// readServerTxt reads the file "server.txt" next to the given executable path.
// Returns trimmed content, or empty string on any error.
func readServerTxt(exePath string) string {
	dir := filepath.Dir(exePath)
	data, err := os.ReadFile(filepath.Join(dir, "server.txt"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// guiInstaller encapsulates the GUI enrollment flow with injectable dependencies.
type guiInstaller struct {
	runner zenityRunner
	enroll enrollFunc
}

// RunGUIInstall executes the full GUI enrollment flow. Returns an exit code.
// If not running as root, it re-launches itself via pkexec — the OS shows a
// native password dialog, user enters password and clicks OK, then the wizard
// runs with full admin permissions to create /etc/patchiq, install to
// /usr/local/bin, and set up a systemd system service.
func RunGUIInstall(_ []string) int {
	if os.Getuid() != 0 {
		return relaunchAsRoot()
	}

	installer := guiInstaller{
		runner: defaultZenityRunner{},
		enroll: performEnroll,
	}
	return installer.run()
}

// relaunchAsRoot re-executes this binary via pkexec. pkexec shows the native
// polkit password dialog — user enters their password and clicks Authenticate.
// The binary then runs as root with full permissions.
func relaunchAsRoot() int {
	exe, err := os.Executable()
	if err != nil {
		slog.Error("gui install: cannot find own executable", "error", err)
		return 1
	}
	exe, _ = filepath.EvalSymlinks(exe)

	display := os.Getenv("DISPLAY")
	xauth := os.Getenv("XAUTHORITY")
	if xauth == "" {
		if home, err := os.UserHomeDir(); err == nil {
			xauth = filepath.Join(home, ".Xauthority")
		}
	}

	cmd := exec.Command("pkexec", "env",
		"DISPLAY="+display,
		"XAUTHORITY="+xauth,
		exe,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		slog.Error("gui install: pkexec failed", "error", err)
		return 1
	}
	return 0
}

func (g *guiInstaller) run() int {
	// Read default server URL from server.txt next to binary.
	exePath, _ := os.Executable()
	defaultServer := readServerTxt(exePath)

	const maxAttempts = 3
	for attempt := range maxAttempts {
		server, token, ok := g.promptCredentials(defaultServer)
		if !ok {
			return 1
		}

		// Show progress dialog.
		stdin, cancelProgress, err := g.runner.Start(
			"--progress", "--pulsate", "--no-cancel", "--auto-close",
			"--title=PatchIQ Agent", "--text=Enrolling...",
		)
		if err != nil {
			slog.Error("gui install: failed to start progress dialog", "error", err)
			return 1
		}

		logStatus := func(msg string) {
			fmt.Fprintf(stdin, "# %s\n", msg)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		opts := installOpts{
			server: server,
			token:  token,
		}
		agentID, enrollErr := g.enroll(ctx, opts, logStatus)
		cancel()
		cancelProgress()

		if enrollErr == nil {
			hostname, _ := os.Hostname()
			_, _ = g.runner.Run(
				"--info", "--title=PatchIQ Agent",
				fmt.Sprintf("--text=Agent enrolled as %s\nAgent ID: %s\nService installed and running.", hostname, agentID),
			)
			return 0
		}

		slog.Error("gui install: enrollment failed", "error", enrollErr, "attempt", attempt+1)

		if attempt < maxAttempts-1 {
			_, _ = g.runner.Run(
				"--error", "--title=PatchIQ Agent",
				fmt.Sprintf("--text=Enrollment failed: %v\n\nRetrying (%d/%d)...", enrollErr, attempt+1, maxAttempts),
			)
		} else {
			_, _ = g.runner.Run(
				"--error", "--title=PatchIQ Agent",
				fmt.Sprintf("--text=Enrollment failed after %d attempts: %v", maxAttempts, enrollErr),
			)
		}
	}

	return 1
}

// promptCredentials shows zenity dialogs for server URL and token.
// If defaultServer is set (from server.txt), skips the server address dialog
// and only asks for the token — one field, one click.
// Returns server, token, ok. ok is false if the user cancelled.
func (g *guiInstaller) promptCredentials(defaultServer string) (server, token string, ok bool) {
	var err error

	// Only ask for server address if server.txt was missing/empty.
	if defaultServer != "" {
		server = defaultServer
	} else {
		server, err = g.runner.Run(
			"--entry", "--title=PatchIQ Agent Setup",
			"--text=Patch Manager server address:",
			"--entry-text=10.0.5.13:50451",
		)
		if err != nil {
			return "", "", false
		}
		if server == "" {
			return "", "", false
		}
	}

	for {
		token, err = g.runner.Run(
			"--entry", "--title=PatchIQ Agent Setup",
			"--text=Paste your registration token:",
		)
		if err != nil {
			return "", "", false
		}
		if token != "" {
			break
		}
		_, _ = g.runner.Run(
			"--error", "--title=PatchIQ Agent",
			"--text=Registration token is required.",
		)
	}

	return server, token, true
}

// ShowAlreadyEnrolledDialog displays an info dialog and returns 0.
func ShowAlreadyEnrolledDialog() int {
	_ = exec.Command("zenity",
		"--info", "--title=PatchIQ Agent",
		"--text=Agent is already enrolled and running on this machine.",
	).Run()
	return 0
}
