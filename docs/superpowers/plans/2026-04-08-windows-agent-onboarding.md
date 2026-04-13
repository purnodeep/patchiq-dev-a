# Windows Agent Onboarding Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Windows operator downloads `patchiq-agent.exe`, right-clicks "Run as administrator", pastes a token in a TUI wizard, and the agent installs itself as a Windows service that survives reboot.

**Architecture:** Mirror of the existing Linux design. Server address baked into the binary at build time via `-ldflags`. The same binary auto-detects: if no config file exists → run install wizard; if config exists → run as daemon. The wizard reuses the existing Bubble Tea TUI in `install_tui.go`, extends it with post-enrollment service-install steps, and calls the already-complete `service_windows.go` helpers. No MSI, no new build tooling.

**Tech Stack:** Go 1.25, Bubble Tea v2 (charm.land/bubbletea/v2), `golang.org/x/sys/windows`, nginx `stream` module, GitHub Actions, React 19 + Vite for the Agent Downloads page update.

**Spec:** [`docs/superpowers/specs/2026-04-08-windows-agent-onboarding-design.md`](../specs/2026-04-08-windows-agent-onboarding-design.md)

---

## File Structure

| File | Status | Responsibility |
|---|---|---|
| `cmd/agent/cli/defaults.go` | NEW | Single ldflags-overridable `DefaultServerAddress` var |
| `cmd/agent/cli/defaults_test.go` | NEW | Resolution-order tests |
| `cmd/agent/cli/elevation_windows.go` | NEW | `isAdmin()` for Windows via syscall |
| `cmd/agent/cli/elevation_other.go` | NEW | Stub `isAdmin() = true` for non-Windows builds |
| `cmd/agent/cli/install.go` | MODIFY | Use `DefaultServerAddress` in resolution chain; call `isAdmin()` at top of `RunInstall` |
| `cmd/agent/cli/install_tui.go` | MODIFY | Skip `stepServerInput` when default baked in; add `stepInstallingService`, `stepStartingService` after enrollment; call `serviceInstall()` and `serviceStart()` from `service_windows.go` |
| `cmd/agent/cli/install_tui_other.go` | NEW | Stub `installAndStartService() = nil` for non-Windows builds (so the TUI compiles cross-platform) |
| `cmd/agent/cli/install_tui_windows.go` | NEW | Real `installAndStartService()` calling existing `serviceInstall()` + `serviceStart()` |
| `cmd/agent/main.go` | MODIFY | If no subcommand AND no config file at resolved path → dispatch to `cli.RunInstall([]string{})` |
| `deploy/nginx/patchiq.conf` | MODIFY | Add `stream {}` top-level block + server block for rishab gRPC route |
| `Makefile` | MODIFY | New `build-agent-windows` target |
| `.github/workflows/release.yml` | MODIFY | Add Windows build matrix entry with ldflags |
| `web/src/pages/agent-downloads/AgentDownloadsPage.tsx` | MODIFY | Update Windows install instructions text |
| `docs/agent-onboarding-windows.md` | NEW | Operator-facing one-page runbook |

---

## Task Ordering Rationale

Tasks are ordered so each one produces a working, committable change that doesn't break the build. Earlier tasks have zero external dependencies. Later tasks depend on earlier ones. CI, web UI, and docs come last because they don't gate the agent-side work.

---

## Task 1: Add `DefaultServerAddress` ldflags variable

**Files:**
- Create: `cmd/agent/cli/defaults.go`
- Test: `cmd/agent/cli/defaults_test.go`

- [ ] **Step 1: Write the failing test**

```go
// cmd/agent/cli/defaults_test.go
package cli

import "testing"

func TestDefaultServerAddress_DefaultsToEmpty(t *testing.T) {
	// In a development build (no -ldflags injection), DefaultServerAddress
	// must be empty so install validation forces an explicit --server.
	if DefaultServerAddress != "" {
		t.Errorf("DefaultServerAddress should be empty in dev builds, got %q", DefaultServerAddress)
	}
}

func TestResolveServerAddress(t *testing.T) {
	tests := []struct {
		name     string
		flag     string
		envVal   string
		baked    string
		want     string
		wantErr  bool
	}{
		{name: "flag wins", flag: "flag.example:50051", envVal: "env.example:50051", baked: "baked.example:50051", want: "flag.example:50051"},
		{name: "env wins over baked", flag: "", envVal: "env.example:50051", baked: "baked.example:50051", want: "env.example:50051"},
		{name: "baked when no flag or env", flag: "", envVal: "", baked: "baked.example:50051", want: "baked.example:50051"},
		{name: "error when nothing set", flag: "", envVal: "", baked: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveServerAddress(tt.flag, tt.envVal, tt.baked)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run test, expect FAIL**

Run: `go test ./cmd/agent/cli/ -run 'TestDefaultServerAddress|TestResolveServerAddress' -v`
Expected: FAIL — `undefined: DefaultServerAddress` and `undefined: resolveServerAddress`

- [ ] **Step 3: Create `defaults.go` with minimal implementation**

```go
// cmd/agent/cli/defaults.go
package cli

import "fmt"

// DefaultServerAddress is the Patch Manager gRPC address baked into the
// binary at build time via -ldflags. Empty in development builds; set to
// the public address (e.g. "patchiq.skenzer.com:3013") in release builds.
//
// Override at build time:
//
//	go build -ldflags "-X github.com/skenzeriq/patchiq/cmd/agent/cli.DefaultServerAddress=patchiq.example.com:3013" ./cmd/agent
var DefaultServerAddress = ""

// resolveServerAddress picks the server address using the precedence:
// explicit flag > environment variable > ldflags-baked default.
// Returns an error if all three are empty.
func resolveServerAddress(flag, env, baked string) (string, error) {
	if flag != "" {
		return flag, nil
	}
	if env != "" {
		return env, nil
	}
	if baked != "" {
		return baked, nil
	}
	return "", fmt.Errorf("no server address: pass --server, set PATCHIQ_AGENT_SERVER_ADDRESS, or build with -ldflags DefaultServerAddress")
}
```

- [ ] **Step 4: Run test, expect PASS**

Run: `go test ./cmd/agent/cli/ -run 'TestDefaultServerAddress|TestResolveServerAddress' -v`
Expected: PASS (5 subtests, including the empty-default check)

- [ ] **Step 5: Commit**

```bash
git add cmd/agent/cli/defaults.go cmd/agent/cli/defaults_test.go
git commit -m "feat(agent): add DefaultServerAddress ldflags var with resolution helper"
```

---

## Task 2: Wire `resolveServerAddress` into install opts

**Files:**
- Modify: `cmd/agent/cli/install.go` (functions `parseInstallFlags` and `validateInstallOpts`)
- Test: `cmd/agent/cli/install_test.go` (new test cases)

- [ ] **Step 1: Read the existing test file to understand the test pattern**

Run: `cat cmd/agent/cli/install_test.go | head -80`
Note the existing table-driven test format and reuse it.

- [ ] **Step 2: Write a failing test for the new behavior**

Append to `cmd/agent/cli/install_test.go`:

```go
func TestValidateInstallOpts_UsesDefaultServerAddress(t *testing.T) {
	// Save & restore the package-level baked default.
	orig := DefaultServerAddress
	t.Cleanup(func() { DefaultServerAddress = orig })

	tests := []struct {
		name        string
		baked       string
		serverFlag  string
		envServer   string
		nonInteract bool
		token       string
		wantErr     bool
		wantServer  string
	}{
		{
			name:        "headless: baked default fills in missing --server",
			baked:       "patchiq.example.com:3013",
			serverFlag:  "",
			nonInteract: true,
			token:       "tok123",
			wantErr:     false,
			wantServer:  "patchiq.example.com:3013",
		},
		{
			name:        "headless: explicit --server overrides baked",
			baked:       "patchiq.example.com:3013",
			serverFlag:  "other.example:50051",
			nonInteract: true,
			token:       "tok123",
			wantErr:     false,
			wantServer:  "other.example:50051",
		},
		{
			name:        "headless: no flag, no env, no baked → error",
			baked:       "",
			serverFlag:  "",
			nonInteract: true,
			token:       "tok123",
			wantErr:     true,
		},
		{
			name:        "headless: env var fills in missing --server",
			baked:       "",
			envServer:   "env.example:50051",
			serverFlag:  "",
			nonInteract: true,
			token:       "tok123",
			wantErr:     false,
			wantServer:  "env.example:50051",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			DefaultServerAddress = tt.baked
			if tt.envServer != "" {
				t.Setenv("PATCHIQ_AGENT_SERVER_ADDRESS", tt.envServer)
			}
			opts := installOpts{
				server:         tt.serverFlag,
				token:          tt.token,
				nonInteractive: tt.nonInteract,
			}
			resolved, err := validateInstallOpts(opts)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (resolved=%+v)", resolved)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resolved.server != tt.wantServer {
				t.Errorf("server: got %q, want %q", resolved.server, tt.wantServer)
			}
		})
	}
}
```

- [ ] **Step 3: Run test, expect FAIL**

Run: `go test ./cmd/agent/cli/ -run TestValidateInstallOpts_UsesDefaultServerAddress -v`
Expected: FAIL — `validateInstallOpts` currently returns `error`, not `(installOpts, error)`. Compile error.

- [ ] **Step 4: Update `validateInstallOpts` to return resolved opts**

In `cmd/agent/cli/install.go`, replace the existing `validateInstallOpts` function (around line 76) with:

```go
// validateInstallOpts checks invariants for the install command and resolves
// the server address using flag > env > ldflags-baked default precedence.
// Returns the resolved opts (with server address filled in) and an error if
// any required field is missing.
func validateInstallOpts(opts installOpts) (installOpts, error) {
	if opts.nonInteractive {
		if opts.token == "" {
			return opts, fmt.Errorf("install: --token is required in non-interactive mode")
		}
		envServer := os.Getenv("PATCHIQ_AGENT_SERVER_ADDRESS")
		resolved, err := resolveServerAddress(opts.server, envServer, DefaultServerAddress)
		if err != nil {
			return opts, fmt.Errorf("install: %w", err)
		}
		opts.server = resolved
	}
	return opts, nil
}
```

Then update the call site in `RunInstall` (around line 113):

```go
	resolved, err := validateInstallOpts(opts)
	if err != nil {
		slog.Error("install: validation failed", "error", err)
		return ExitError
	}
	opts = resolved
```

- [ ] **Step 5: Run all install tests, expect PASS**

Run: `go test ./cmd/agent/cli/ -run TestValidateInstallOpts -v`
Expected: PASS, including any pre-existing tests for `validateInstallOpts`.

- [ ] **Step 6: Run the full cli package tests to confirm no regression**

Run: `go test ./cmd/agent/cli/ -race -v`
Expected: all PASS.

- [ ] **Step 7: Commit**

```bash
git add cmd/agent/cli/install.go cmd/agent/cli/install_test.go
git commit -m "feat(agent): resolve install server address from flag/env/ldflags chain"
```

---

## Task 3: Add admin-elevation check (Windows + stub for other OSes)

**Files:**
- Create: `cmd/agent/cli/elevation_windows.go`
- Create: `cmd/agent/cli/elevation_other.go`

- [ ] **Step 1: Create the Windows implementation**

```go
// cmd/agent/cli/elevation_windows.go
//go:build windows

package cli

import (
	"golang.org/x/sys/windows"
)

// isAdmin reports whether the current process is running with administrator
// privileges. Required to install a Windows service.
func isAdmin() bool {
	var sid *windows.SID
	// S-1-5-32-544 is the well-known SID for the local Administrators group.
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid,
	)
	if err != nil {
		return false
	}
	defer windows.FreeSid(sid)

	token := windows.Token(0) // current process token
	member, err := token.IsMember(sid)
	if err != nil {
		return false
	}
	return member
}
```

- [ ] **Step 2: Create the non-Windows stub**

```go
// cmd/agent/cli/elevation_other.go
//go:build !windows

package cli

// isAdmin always returns true on non-Windows platforms. Linux and macOS
// install paths handle their own privilege checks (geteuid).
func isAdmin() bool {
	return true
}
```

- [ ] **Step 3: Verify it compiles on Linux**

Run: `go build ./cmd/agent/...`
Expected: success, no errors.

- [ ] **Step 4: Verify it cross-compiles for Windows**

Run: `GOOS=windows GOARCH=amd64 go build ./cmd/agent/...`
Expected: success, no errors.

- [ ] **Step 5: Commit**

```bash
git add cmd/agent/cli/elevation_windows.go cmd/agent/cli/elevation_other.go
git commit -m "feat(agent): add isAdmin() helper for Windows elevation check"
```

---

## Task 4: Wire elevation check into `RunInstall`

**Files:**
- Modify: `cmd/agent/cli/install.go` (function `RunInstall`)

- [ ] **Step 1: Add the check at the top of `RunInstall`**

In `cmd/agent/cli/install.go`, in `RunInstall` (around line 106), add the check as the very first thing inside the function, before flag parsing:

```go
func RunInstall(args []string) int {
	if !isAdmin() {
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "  PatchIQ Agent installer must be run as Administrator.")
		fmt.Fprintln(os.Stderr, "  Right-click patchiq-agent.exe and select \"Run as administrator\".")
		fmt.Fprintln(os.Stderr, "")
		return ExitError
	}

	opts, err := parseInstallFlags(args)
	// ... existing code unchanged
```

- [ ] **Step 2: Verify build**

Run: `go build ./cmd/agent/...`
Expected: success.

- [ ] **Step 3: Run install tests to verify nothing regresses**

Run: `go test ./cmd/agent/cli/ -race -v`
Expected: PASS. (On Linux, `isAdmin()` returns true, so no test changes needed.)

- [ ] **Step 4: Commit**

```bash
git add cmd/agent/cli/install.go
git commit -m "feat(agent): require admin privileges to run install subcommand"
```

---

## Task 5: Add `installAndStartService` helper (Windows + stub)

**Files:**
- Create: `cmd/agent/cli/install_tui_windows.go`
- Create: `cmd/agent/cli/install_tui_other.go`

These wrap the existing `serviceInstall()` and `serviceStart()` from `service_windows.go` so the TUI Update loop can call a single platform-agnostic function.

- [ ] **Step 1: Create the Windows wrapper**

```go
// cmd/agent/cli/install_tui_windows.go
//go:build windows

package cli

import "fmt"

// installAndStartService registers the agent as a Windows service and starts it.
// Called by the install TUI after successful enrollment.
func installAndStartService() error {
	if rc := serviceInstall(); rc != ExitOK {
		return fmt.Errorf("install service: serviceInstall returned exit code %d", rc)
	}
	if rc := serviceStart(); rc != ExitOK {
		return fmt.Errorf("start service: serviceStart returned exit code %d", rc)
	}
	return nil
}
```

- [ ] **Step 2: Create the non-Windows stub**

```go
// cmd/agent/cli/install_tui_other.go
//go:build !windows

package cli

// installAndStartService is a no-op on non-Windows platforms. The Linux install
// path handles systemd registration separately.
func installAndStartService() error {
	return nil
}
```

- [ ] **Step 3: Verify cross-compile**

Run: `go build ./cmd/agent/... && GOOS=windows GOARCH=amd64 go build ./cmd/agent/...`
Expected: both succeed.

- [ ] **Step 4: Commit**

```bash
git add cmd/agent/cli/install_tui_windows.go cmd/agent/cli/install_tui_other.go
git commit -m "feat(agent): add installAndStartService wrapper for TUI service step"
```

---

## Task 6: Extend install TUI with service-install steps and skip-server-input

**Files:**
- Modify: `cmd/agent/cli/install_tui.go`

This is the largest single edit in the plan. Read the current file once before starting (already done in brainstorming).

- [ ] **Step 1: Add new step constants**

In `install_tui.go`, replace the existing `installStep` constants block (lines 19-25) with:

```go
const (
	stepServerInput installStep = iota
	stepTokenInput
	stepConnecting
	stepInstallingService
	stepDone
	stepError
)
```

(Note: `installAndStartService` does install + start in one synchronous call, so a separate `stepStartingService` would never be observed by the user. Keep the step list lean.)

- [ ] **Step 2: Add a new message type for service-install completion**

Add this near `enrollResultMsg` (line 27):

```go
// serviceResultMsg carries the result of installing/starting the Windows service.
type serviceResultMsg struct {
	err error
}
```

- [ ] **Step 3: Skip `stepServerInput` when `DefaultServerAddress` is set**

Replace the existing `newInstallModel` function (line 50) with:

```go
func newInstallModel(opts installOpts) installModel {
	si := textinput.New()
	si.Placeholder = "localhost:50051"
	si.CharLimit = 256
	si.SetWidth(40)

	ti := textinput.New()
	ti.Placeholder = "paste enrollment token"
	ti.CharLimit = 512
	ti.SetWidth(40)

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	// If a default server address is baked in (release builds), skip the
	// server-input step entirely and jump straight to the token field.
	startStep := stepServerInput
	if DefaultServerAddress != "" && opts.server == "" {
		opts.server = DefaultServerAddress
		startStep = stepTokenInput
		ti.Focus()
	} else {
		si.Focus()
	}

	return installModel{
		step:        startStep,
		serverInput: si,
		tokenInput:  ti,
		spinner:     sp,
		opts:        opts,
	}
}
```

- [ ] **Step 4: Handle `enrollResultMsg` to advance to service-install instead of stepDone**

In the `Update` method, replace the `enrollResultMsg` case (lines 92-100) with:

```go
	case enrollResultMsg:
		if msg.err != nil {
			m.step = stepError
			m.err = msg.err
			return m, tea.Quit
		}
		m.agentID = msg.agentID
		m.step = stepInstallingService
		return m, tea.Batch(m.spinner.Tick, m.doInstallService())
```

- [ ] **Step 5: Add `serviceResultMsg` handler**

Add this case to the `Update` method's switch (after the `enrollResultMsg` case):

```go
	case serviceResultMsg:
		if msg.err != nil {
			m.step = stepError
			m.err = msg.err
			return m, tea.Quit
		}
		m.step = stepDone
		return m, tea.Quit
```

- [ ] **Step 6: Add the `doInstallService` command**

Add this method after `doEnroll` (around line 195):

```go
// doInstallService returns a tea.Cmd that registers the agent as a Windows
// service and starts it. On non-Windows platforms it is a no-op (see
// installAndStartService stub) so the TUI flow stays uniform.
func (m installModel) doInstallService() tea.Cmd {
	return func() tea.Msg {
		if err := installAndStartService(); err != nil {
			return serviceResultMsg{err: fmt.Errorf("install: %w", err)}
		}
		return serviceResultMsg{}
	}
}
```

- [ ] **Step 7: Render the new step views**

In the `View` method, add cases for the new steps before `stepDone`:

```go
	case stepInstallingService:
		s = fmt.Sprintf(
			"%s\n\n%s Installing PatchIQ as a Windows service...\n",
			title, m.spinner.View(),
		)
```

- [ ] **Step 8: Update the `stepDone` view to mention the service**

Replace the existing `stepDone` case in `View` (lines 221-232) with:

```go
	case stepDone:
		configPath := m.opts.configPath
		if configPath == "" {
			configPath = defaultConfigPath
		}
		s = fmt.Sprintf(
			"%s\n\n%s\n  Agent ID:    %s\n  Config:      %s\n  Service:     PatchIQAgent (running)\n\n%s\n",
			title,
			successStyle.Render("Setup complete!"),
			m.agentID,
			configPath,
			dimStyle.Render("The agent is now running as a background service and will start automatically on boot."),
		)
```

- [ ] **Step 9: Verify cross-compile**

Run: `go build ./cmd/agent/... && GOOS=windows GOARCH=amd64 go build ./cmd/agent/...`
Expected: both succeed.

- [ ] **Step 10: Run any existing TUI tests**

Run: `go test ./cmd/agent/cli/ -race -v`
Expected: PASS.

- [ ] **Step 11: Commit**

```bash
git add cmd/agent/cli/install_tui.go
git commit -m "feat(agent): TUI installs+starts windows service after enrollment, skips server step when baked"
```

---

## Task 7: Auto-launch wizard on first run when no config exists

**Files:**
- Modify: `cmd/agent/main.go`

- [ ] **Step 1: Read the existing subcommand dispatch in `main()` to understand the entry shape**

Already reviewed. The current `main()` (line 38) dispatches subcommands first, then falls through to daemon. We add a check between the two.

- [ ] **Step 2: Add a helper to resolve the config path the same way `loadConfig` does**

`loadConfig` calls `cli.LoadAgentConfig(configPath)` which uses `defaultConfigPath` (per-OS) when `configPath` is empty. We need to peek at whether a file exists at that resolved path *without* loading it (and without exiting on error). Add a helper near `parseConfigFlag`:

```go
// configFileExists reports whether a config file exists at the resolved path.
// Used by the no-args first-run logic to decide between launching the install
// wizard and starting the daemon. Never errors — a missing file is the whole point.
func configFileExists(configPath string) bool {
	if configPath == "" {
		configPath = cli.DefaultConfigPath()
	}
	_, err := os.Stat(configPath)
	return err == nil
}
```

- [ ] **Step 3: Expose `DefaultConfigPath` from the cli package**

The constant `defaultConfigPath` already exists in `cli/config.go` (lowercase, unexported). Add an exported accessor in the same file:

```go
// DefaultConfigPath returns the default config file path for the current OS.
// Exposed for callers (main.go) that need to check existence without loading.
func DefaultConfigPath() string {
	return defaultConfigPath
}
```

- [ ] **Step 4: Wire the first-run dispatch into `main()`**

Replace the current `main()` function (lines 38-72) with:

```go
func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "install":
			os.Exit(cli.RunInstall(os.Args[2:]))
		case "status":
			os.Exit(cli.RunStatus(os.Args[2:]))
		case "scan":
			os.Exit(cli.RunScan(os.Args[2:]))
		case "service":
			os.Exit(cli.RunService(os.Args[2:]))
		case "--help", "-h", "help":
			cli.Usage()
			os.Exit(0)
		}
	}

	// Parse --config flag from os.Args for the daemon path.
	configPath := parseConfigFlag(os.Args[1:])

	// First-run auto-launch: if invoked with no subcommand and no config file
	// exists, run the install wizard interactively. After it completes the user
	// closes the window; the Windows service (registered by the wizard) will
	// be the daemon process from then on.
	if len(os.Args) == 1 && !configFileExists(configPath) {
		os.Exit(cli.RunInstall([]string{}))
	}

	// Check if running as Windows service
	if isWindowsService() {
		runAsWindowsService(configPath)
		return
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := runDaemon(ctx, cancel, configPath); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 5: Verify cross-compile**

Run: `go build ./cmd/agent/... && GOOS=windows GOARCH=amd64 go build ./cmd/agent/...`
Expected: both succeed.

- [ ] **Step 6: Run agent tests**

Run: `go test ./cmd/agent/... -race -v`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add cmd/agent/main.go cmd/agent/cli/config.go
git commit -m "feat(agent): auto-launch install wizard on first run when no config exists"
```

---

## Task 8: nginx `stream` block for rishab gRPC route

**Files:**
- Modify: `deploy/nginx/patchiq.conf`

- [ ] **Step 1: Add the stream block at the top of the file (sibling of `http{}`)**

Open `deploy/nginx/patchiq.conf` and add this block immediately after the `events {}` block (line 14), before the existing `http {}` block:

```nginx
# ───────── Agent gRPC TCP passthrough on :3013 ─────────
# Pure TCP passthrough (not L7 gRPC). Works for h2c today and is mTLS-ready
# for PIQ-116 — when TLS is enabled, the handshake survives end-to-end
# because nginx never opens the gRPC stream.
#
# Port 3013 chosen because it's in the router-forwarded range (3000-3199).
# Upstream is rishab's server gRPC port (50051 + 300 offset = 50351).
stream {
    server {
        listen 3013;
        proxy_pass 127.0.0.1:50351;
        proxy_timeout 86400s;  # match the 24h timeout used for long-lived sync streams
    }
}
```

- [ ] **Step 2: Validate the nginx config syntax**

Run: `docker run --rm -v "$(pwd)/deploy/nginx/patchiq.conf:/etc/nginx/nginx.conf:ro" nginx:alpine nginx -t`
Expected: `nginx: configuration file /etc/nginx/nginx.conf test is successful`

- [ ] **Step 3: Commit**

```bash
git add deploy/nginx/patchiq.conf
git commit -m "feat(deploy): add nginx stream block for rishab agent gRPC on :3013"
```

---

## Task 9: Add `build-agent-windows` Makefile target

**Files:**
- Modify: `Makefile`

- [ ] **Step 1: Read the existing `build-agents` target to match style**

Run: `grep -A 10 '^build-agents' Makefile`
Note the existing pattern for cross-compiling agents.

- [ ] **Step 2: Add the new target**

Append to the build section of `Makefile`:

```makefile
# Build the Windows agent with the public server address baked in via -ldflags.
# Override SERVER_ADDR on the command line for ad-hoc release builds:
#   make build-agent-windows SERVER_ADDR=patchiq.example.com:3013
SERVER_ADDR ?=
build-agent-windows:
	@if [ -z "$(SERVER_ADDR)" ]; then \
		echo "ERROR: SERVER_ADDR is required. Example:"; \
		echo "  make build-agent-windows SERVER_ADDR=patchiq.example.com:3013"; \
		exit 1; \
	fi
	GOOS=windows GOARCH=amd64 go build \
		-ldflags "-X github.com/skenzeriq/patchiq/cmd/agent/cli.DefaultServerAddress=$(SERVER_ADDR)" \
		-o bin/patchiq-agent.exe ./cmd/agent
	@echo "Built bin/patchiq-agent.exe with server address $(SERVER_ADDR)"
.PHONY: build-agent-windows
```

- [ ] **Step 3: Test the failure path (no SERVER_ADDR)**

Run: `make build-agent-windows`
Expected: fails with the SERVER_ADDR error message and exit code 1.

- [ ] **Step 4: Test the success path with a dummy address**

Run: `make build-agent-windows SERVER_ADDR=test.example:3013`
Expected: produces `bin/patchiq-agent.exe`. Verify the address is baked in:

```bash
strings bin/patchiq-agent.exe | grep test.example
```
Expected: prints `test.example:3013`.

- [ ] **Step 5: Clean up the test artifact**

```bash
rm bin/patchiq-agent.exe
```

- [ ] **Step 6: Commit**

```bash
git add Makefile
git commit -m "build: add make target build-agent-windows with ldflags server address"
```

---

## Task 10: CI release build for Windows agent

**Files:**
- Modify: `.github/workflows/release.yml`

This task touches a protected file per CLAUDE.md (`.github/workflows/`), so it requires core dev review on the PR.

- [ ] **Step 1: Read the existing release workflow**

Run: `cat .github/workflows/release.yml`
Identify where existing agent builds happen, the matrix strategy if any, and how artifacts are uploaded.

- [ ] **Step 2: Add a Windows build job (or matrix entry)**

If the workflow uses a matrix, add `windows-amd64` to the matrix list. Otherwise add a new job:

```yaml
  build-agent-windows:
    runs-on: self-hosted
    needs: [test]   # match the existing dependency pattern
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - name: Build Windows agent with baked server address
        env:
          SERVER_ADDR: ${{ secrets.PATCHIQ_PUBLIC_SERVER_ADDR }}
        run: |
          if [ -z "$SERVER_ADDR" ]; then
            echo "ERROR: PATCHIQ_PUBLIC_SERVER_ADDR secret not set in repo settings"
            exit 1
          fi
          make build-agent-windows SERVER_ADDR="$SERVER_ADDR"

      - name: Upload Windows agent artifact
        uses: actions/upload-artifact@v4
        with:
          name: patchiq-agent-windows-amd64
          path: bin/patchiq-agent.exe
```

- [ ] **Step 3: Add the required secret to the repo (manual, document in commit body)**

This is an out-of-band action by a repo admin: in GitHub repo Settings → Secrets and variables → Actions, add `PATCHIQ_PUBLIC_SERVER_ADDR` with the public address (e.g. `patchiq.skenzer.com:3013`). The plan only documents the requirement; the human must perform this step.

- [ ] **Step 4: Validate the YAML syntax locally**

Run: `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/release.yml'))"`
Expected: no output (valid YAML). If python3-yaml isn't installed, use `actionlint` if available, or skip and rely on GitHub's validation on push.

- [ ] **Step 5: Commit**

```bash
git add .github/workflows/release.yml
git commit -m "ci: build windows agent with baked server address in release workflow

Requires repo admin to set PATCHIQ_PUBLIC_SERVER_ADDR secret before
the next release. Touches a protected file — needs core dev review."
```

---

## Task 11: Update Agent Downloads page (Windows instructions)

**Files:**
- Modify: `web/src/pages/agent-downloads/AgentDownloadsPage.tsx`

- [ ] **Step 1: Locate and update `buildInstallCommand` for the Windows branch**

The current page builds an install command at `web/src/pages/agent-downloads/AgentDownloadsPage.tsx:41-50`. The Windows branch (line 43-44) currently produces:

```ts
if (binary.os === 'windows') {
  return `.\\${filename} install --server ${serverUrl}`;
}
```

This is wrong for the new flow because (a) the server address is now baked into the binary at build time, (b) the operator never opens a terminal for Windows — they double-click. Replace lines 43-44 with:

```ts
if (binary.os === 'windows') {
  // Windows: server address is baked into the binary at build time.
  // Operator double-clicks the .exe and pastes the token in the wizard.
  return `# Right-click ${filename} → "Run as administrator"\n# Paste your registration token in the wizard.`;
}
```

(The leading `# ` lines render as a comment in the existing copy-to-clipboard UI, which doubles as user-visible instructions. The Linux/macOS branches stay unchanged because they still use `--server`.)

- [ ] **Step 2: Update any Windows-specific prose in the page body**

Run: `grep -n 'windows\|Windows' web/src/pages/agent-downloads/AgentDownloadsPage.tsx`
For any text that says "PowerShell", "env vars", "SSH tunnel", or references the old `install --server <url>` pattern in the Windows context, replace with: "Right-click and Run as administrator. Paste the token when the wizard appears. The agent installs itself as a Windows service automatically." If no such prose exists (likely — instructions are generated from `buildInstallCommand`), skip this step.

- [ ] **Step 3: Run typecheck**

Run: `cd web && pnpm tsc --noEmit`
Expected: no errors.

- [ ] **Step 4: Run the page's tests if they exist**

Run: `cd web && pnpm vitest run src/__tests__/pages/agent-downloads/`
Expected: PASS, or "no test files found" (then skip).

- [ ] **Step 5: Visually verify in dev**

(Manual step) — `make dev` is already running. Open `http://localhost:3301/agent-downloads` and confirm the Windows card shows the new four-step flow.

- [ ] **Step 6: Commit**

```bash
git add web/src/pages/agent-downloads/AgentDownloadsPage.tsx
git commit -m "feat(web): update agent-downloads page with new windows install flow"
```

---

## Task 12: Operator-facing Windows runbook

**Files:**
- Create: `docs/agent-onboarding-windows.md`

- [ ] **Step 1: Write the runbook**

```markdown
# Onboarding a Windows Endpoint

This runbook covers installing the PatchIQ agent on a Windows machine
(Windows 10, Windows 11, or Windows Server 2019+).

## Requirements

- Administrator account on the target machine
- Network access from the target machine to your PatchIQ server
  (the public address is baked into the binary you download)
- ~50 MB free disk space

## Steps

1. **Generate a registration token.**
   In the PatchIQ web UI, open **Agent Downloads**, click
   **Generate registration token**, and copy the token shown
   (it looks like `K7M-3PQ-9XR`).

2. **Download `patchiq-agent.exe`.**
   On the same Agent Downloads page, click **Download Windows agent**.

3. **Copy the file to the target machine.**
   Any method works — USB, network share, RDP file transfer.

4. **Right-click `patchiq-agent.exe` → Run as administrator.**
   Windows will prompt for elevation (UAC). Click **Yes**.

   On unsigned builds, Windows SmartScreen may show "Windows protected
   your PC". Click **More info** then **Run anyway**.

5. **Paste the token in the wizard.**
   A small terminal window appears with the PatchIQ Agent Setup wizard.
   Paste your token and press Enter.

6. **Wait for the wizard to finish.**
   The wizard will:
   - Connect to the PatchIQ server
   - Register this endpoint
   - Install the `PatchIQAgent` Windows service
   - Start the service

   When you see "Setup complete!", close the window. The service is now
   running in the background and will start automatically on every boot.

7. **Verify in PatchIQ.**
   In the web UI, open **Endpoints**. Your new machine should appear
   within ~30 seconds, marked online.

## Uninstalling

There is no Add/Remove Programs entry in the current version. To remove
the agent:

1. Open an Administrator PowerShell.
2. Run: `& "C:\Path\To\patchiq-agent.exe" service uninstall`
3. Delete `C:\Program Files\PatchIQ\` (or wherever you placed the binary).
4. Delete `C:\ProgramData\PatchIQ\` (config + local database).

## Troubleshooting

- **"Must be run as Administrator"** — close the wizard, right-click the
  binary again and choose "Run as administrator".
- **"connection refused" / "no route to host"** — your machine cannot
  reach the PatchIQ server. Verify with `Test-NetConnection
  <server-address> -Port <port>`. Check your firewall and proxy.
- **Wizard appears but token is rejected** — the token may have expired
  (24h lifetime) or already been used. Generate a fresh one in the UI.
- **Endpoint doesn't appear in UI after 1 minute** — open
  `C:\ProgramData\PatchIQ\agent.log` and look for errors. The most
  common cause is a network issue between the machine and the server.
```

- [ ] **Step 2: Commit**

```bash
git add docs/agent-onboarding-windows.md
git commit -m "docs: add windows agent onboarding runbook"
```

---

## Final verification (before requesting code review)

- [ ] **Step 1: Run the full Go test suite with race detector**

Run: `make test`
Expected: all PASS.

- [ ] **Step 2: Run the linter**

Run: `make lint`
Expected: no new findings.

- [ ] **Step 3: Build all three binaries (server, hub, agent) for sanity**

Run: `make build`
Expected: success.

- [ ] **Step 4: Cross-compile the Windows agent with a real test address**

Run: `make build-agent-windows SERVER_ADDR=test.local:3013`
Expected: produces `bin/patchiq-agent.exe`.

- [ ] **Step 5: Frontend typecheck**

Run: `cd web && pnpm tsc --noEmit`
Expected: no errors.

- [ ] **Step 6: Manual end-to-end test on the actual Windows box (DESKTOP-629B940)**

This is the only step that cannot be automated:

1. Copy `bin/patchiq-agent.exe` (built with `SERVER_ADDR` matching your dev nginx port) to the Windows box.
2. Generate a token in the dev PM UI at `http://localhost:3301/agent-downloads`.
3. Right-click the binary → Run as administrator.
4. Paste the token in the wizard.
5. Watch for "Setup complete!".
6. In PM UI, confirm the endpoint appears under `/endpoints`.
7. Reboot the Windows box.
8. Confirm the endpoint reconnects automatically without any user action.

If any step fails, debug before requesting review.

- [ ] **Step 7: Run code review**

Run: `/review-pr all parallel`
Fix any Critical or Important issues.

- [ ] **Step 8: Open PR**

Run: `/commit-push-pr`
(Per user's saved feedback memory: only push and open PR after explicit user instruction.)

---

## Out of scope (explicit deferrals)

These are documented in the spec but **not** in this plan. Each is a separate future PR:

- MSI packaging (WiX v4)
- Code-signing the binary (procurement of signing cert)
- macOS `.pkg` and Linux `.deb`/`.rpm` parity
- Migrating sandy's existing nginx `:3003 http2` block to `stream` mode
- Auto-update of installed agents
- TLS/mTLS for the gRPC channel (PIQ-116)
- Adding "Add/Remove Programs" entry on Windows (depends on MSI)
