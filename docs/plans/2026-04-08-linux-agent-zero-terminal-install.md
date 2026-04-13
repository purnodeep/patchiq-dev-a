# Linux Agent Zero-Terminal Install

**Status**: Implemented
**Date**: 2026-04-08
**Scope**: Linux only (amd64 + arm64)

## Problem

Enterprise POC deployments require IT staff to install agents on employee workstations. The existing enrollment flow requires opening a terminal, extracting a tarball, and running `./patchiq-agent install --server <url> --token <token>`. For non-technical IT staff or environments where "open a terminal" is an unfamiliar step, this creates friction that slows POC adoption. A double-click GUI install — similar to what users expect on Windows/macOS — removes that friction entirely.

## Solution

Native GTK dialogs via **zenity**, a lightweight CLI tool pre-installed on most Linux desktop distributions (GNOME, XFCE, Cinnamon, MATE). The agent binary shells out to `zenity` for each dialog step — no CGo, no new Go dependencies, no embedded GUI toolkit. The agent binary stays minimal (~15MB, static).

When zenity or a display server is unavailable (headless servers, minimal installs), the agent falls through to the existing Bubble Tea TUI or daemon mode. Zero regression for existing workflows.

## End-user flow

1. Admin downloads the Linux agent tarball from the Patch Manager web UI (Agent Downloads page).
2. The tarball is served through a repack endpoint that injects `server.txt` containing the Patch Manager's URL.
3. User extracts the tarball (right-click "Extract Here" or `tar xzf`).
4. User double-clicks `Install PatchIQ Agent` (the `.desktop` file).
5. The OS prompts for admin credentials via polkit (`pkexec`).
6. A zenity entry dialog appears, pre-filled with the server URL from `server.txt`. User pastes the enrollment token.
7. A pulsating progress dialog shows while gRPC enrollment executes.
8. On success: an info dialog confirms enrollment. On failure: an error dialog with retry (up to 3 attempts).

## Architecture

### Agent first-run detection (`cmd/agent/main.go`)

`isEnrolled(dataDir)` checks the SQLite `agent_state` table for an `agent_id` row. When all of the following are true, the GUI wizard launches:
- No CLI subcommand passed (`len(os.Args) == 1`)
- `DISPLAY` environment variable is set
- `HasZenity()` returns true (checks `$PATH`)
- Agent is not already enrolled

If already enrolled, an info dialog is shown and the process exits. If DISPLAY or zenity is missing, the agent falls through to the existing TUI / daemon path.

### Zenity GUI wizard (`cmd/agent/cli/install_gui_linux.go`)

Build-tagged `//go:build linux`. Implements the dialog flow using `exec.Command("zenity", ...)`:

- **Entry dialog**: `--entry` with server URL + token fields. Pre-fills server URL from `server.txt` in the binary's directory.
- **Progress dialog**: `--progress --pulsate` shown during enrollment.
- **Success dialog**: `--info` with confirmation message.
- **Error dialog**: `--error` with failure details and retry logic (up to 3 attempts).

A `zenityRunner` interface wraps `exec.Command` calls, enabling test injection without spawning real zenity processes.

A stub file exists for non-Linux platforms to satisfy the compiler.

### Shared enrollment helper (`cmd/agent/cli/install_shared.go`)

`performEnroll(ctx, opts, logStatus)` extracts the common enrollment logic used by both the TUI (`install_tui.go`) and GUI wizards:

1. Establish gRPC connection to the server.
2. Call `Enroll` RPC with endpoint metadata.
3. Persist mTLS certificates and agent state to SQLite.
4. Report status via the `logStatus` callback (adapts to TUI or GUI progress display).

Previously this logic was inlined in the TUI code. Extracting it eliminated duplication and ensured both install paths use identical enrollment logic.

### Server-side tarball repack (`internal/server/api/v1/agent_binaries.go`)

`GET /api/v1/agent-binaries/{filename}/download` — streams Linux `.tar.gz` files through a gzip/tar pipeline that appends a `server.txt` file containing the requesting server's URL. Non-Linux binaries (Windows `.zip`, macOS `.tar.gz`) are served as-is without modification.

Wired in `router.go` with `endpoints:read` RBAC permission.

This approach means `server.txt` is never baked into the build artifact. The same build artifact works for any Patch Manager instance — the URL is injected at download time.

### Packaging (`Makefile` + `cmd/agent/dist/`)

`make build-agent-linux` produces `dist/agents/patchiq-agent-linux-{amd64,arm64}.tar.gz` containing:

- `patchiq-agent` — the agent binary
- `install.desktop` — freedesktop `.desktop` file with `Terminal=false`, `Exec=pkexec env DISPLAY=... XAUTHORITY=... ./patchiq-agent install`
- `README.txt` — brief install instructions

No `server.txt` at build time. It is injected per-download by the repack endpoint.

### Frontend (`web/src/pages/agent-downloads/AgentDownloadsPage.tsx`)

Linux download links route through the repack endpoint (`/api/v1/agent-binaries/{filename}/download`). The Linux instructions show 3 GUI steps: "Extract the archive", "Double-click Install PatchIQ Agent", "Paste your enrollment token". macOS and Windows instructions are unchanged.

## Key design decisions

**zenity over Fyne/webview**: Fyne adds ~15MB to the binary and requires CGo, violating the "agent is minimal" constraint. Webview requires CGo plus a WebKitGTK runtime dependency. zenity is pre-installed on virtually all Linux desktop environments, adds zero bytes to the binary, and requires zero CGo.

**server.txt injection per-download**: Baking the server URL at build time would require separate builds per Patch Manager instance. Injecting at download time means one build artifact works everywhere. The tarball repack is a streaming operation with negligible overhead.

**performEnroll shared helper**: The TUI and GUI wizards perform identical enrollment logic. Extracting it into a shared function eliminates duplication and ensures bug fixes apply to both paths. The `logStatus` callback pattern lets each UI update its own progress display.

**install.desktop with pkexec + DISPLAY passthrough**: Agent installation requires root (writing to `/opt`, installing systemd units). `pkexec` provides a native polkit dialog for privilege escalation. `Terminal=false` ensures no terminal window appears. DISPLAY and XAUTHORITY are explicitly passed in the `Exec=` line because pkexec strips environment variables by default.

**Fallback to TUI when no DISPLAY / no zenity**: Headless servers and minimal installs do not have zenity or a display server. The detection logic (`DISPLAY` set + `HasZenity()`) ensures zero regression — these environments continue using the existing TUI or `--server`/`--token` CLI flags.

## Alternatives considered

**Fyne (Go native GUI toolkit)**: Adds ~15MB to binary size, requires CGo for OpenGL/X11 bindings, and pulls in a large dependency tree. Violates the "agent is minimal, offline-first, no heavy dependencies" principle.

**Webview (embedded browser)**: Requires CGo and a WebKitGTK runtime dependency. Would provide a richer UI but at the cost of a fragile runtime dependency chain on Linux. Not justified for a one-time enrollment wizard.

**Bubble Tea TUI with Terminal=true .desktop file**: This was the lowest-effort option but defeats the purpose. Opening a terminal window is visually identical to "open a terminal and run a command" from the user's perspective. Does not achieve the zero-terminal goal.

## Known limitations

- **zenity not on minimal/headless installs**: Fallback to TUI handles this. No data loss or broken flow.
- **pkexec strips environment variables**: Workaround is explicit `DISPLAY` and `XAUTHORITY` passthrough in the `Exec=` line of `install.desktop`.
- **`%k` field code not universally supported**: Some file managers do not expand `%k` (path to the `.desktop` file). The desktop file uses explicit relative paths as a fallback.
- **systemd service install not yet wired into performEnroll**: After enrollment succeeds, the user must manually start the agent service. This will be addressed in future work.

## Test coverage

| File | Covers |
|------|--------|
| `cmd/agent/cli/install_gui_linux_test.go` | Zenity dialog flow, retry logic, server.txt pre-fill, zenityRunner injection, error handling |
| `cmd/agent/cli/install_shared_test.go` | performEnroll helper, gRPC enrollment, cert persistence, logStatus callback, error paths |
| `cmd/agent/cli/install_test.go` | Existing TUI install tests, now also validates shared helper integration |
| `internal/server/api/v1/agent_binaries_test.go` | Tarball repack (server.txt injection), non-Linux passthrough, RBAC, streaming behavior |

## Future work

- Wire systemd service install and start into `performEnroll` (post-enrollment auto-start).
- Config field for binary cache directory in `cmd/server/main.go` (currently uses a fixed path).
- macOS `.command` script / Windows UAC equivalent for zero-terminal install on other platforms (separate effort).
- `.deb` / `.rpm` packaging for proper distro integration and package-manager-based deployment.
