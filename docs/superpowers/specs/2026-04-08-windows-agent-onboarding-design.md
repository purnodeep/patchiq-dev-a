# Windows Agent Onboarding — Design

**Date:** 2026-04-08
**Branch:** `feat/agent-msi-onboarding`
**Author:** rishab + Claude
**Status:** Draft, awaiting review
**Scope:** Windows only. (Linux already implemented in a parallel design.)

## Goal

A Windows operator can install the PatchIQ agent on a new endpoint with this exact flow:

1. Open Agent Downloads page in PM UI
2. Click "Generate registration token" → copy short token
3. Click "Download Windows agent" → get `patchiq-agent.exe`
4. On the target Windows box: right-click `patchiq-agent.exe` → **Run as administrator**
5. A wizard appears (the existing Bubble Tea TUI)
6. Operator pastes token, presses Enter
7. Wizard enrolls → writes config → installs Windows service → starts service → done
8. Endpoint appears in PM UI `/endpoints` within seconds, runs forever as a background service, survives reboot

**No SSH tunnel. No env vars. No `--server` flag. No PowerShell ceremony. No MSI. No new build tooling.**

This is a direct Windows mirror of the already-implemented Linux design.

## Non-goals (out of scope for this spec)

- MSI packaging (deferred — current design ships a bare `.exe`)
- Code signing / SmartScreen suppression (follow-up once a cert is procured)
- Linux and macOS parity (Linux already done; macOS is a separate spec)
- TLS / mTLS for the gRPC channel (PIQ-116, separate work; nginx `stream` mode is the prerequisite and is included here)
- Auto-update of installed agents
- Migrating sandy's existing nginx `:3003 http2` block to `stream` mode (small follow-up chore PR)

## Existing scaffolding (already in repo, do not rebuild)

- `cmd/agent/cli/install.go` — full `install` subcommand: flag parsing, headless enrollment via gRPC, writes `agent.yaml`, supports `PATCHIQ_ENROLLMENT_TOKEN` env var to keep token off the process command line
- `cmd/agent/cli/install_tui.go` — Bubble Tea wizard already exists (steps not yet read in detail; will be extended)
- `cmd/agent/cli/service_windows.go` — **production-complete**. `serviceInstall()`, `serviceUninstall()`, `serviceStart()`, `serviceStop()`, `serviceStatus()` all working via `golang.org/x/sys/windows/svc/mgr`. Includes restart-on-failure recovery actions. Service name `PatchIQAgent`, display name `PatchIQ Agent`, auto-start on boot.
- `cmd/agent/cli/config_windows.go` — resolves Windows config dir to `C:\ProgramData\PatchIQ\`
- `deploy/nginx/patchiq.conf` — public reverse proxy config (sandy's offset, h2c gRPC on `:3003`). Confirmed reachable from WAN at `122.176.60.111`. Router forwards ports `3000-3199` only.
- `internal/agent/comms/` — gRPC enrollment + heartbeat + outbox/inbox sync, all working

## Design — what to build

### 1. Reverse-proxy entry point (one-time infra)

Add a new `stream` block to `deploy/nginx/patchiq.conf` for the rishab gRPC route, plus an `events` block alongside the existing `http` block (nginx requires `stream` and `http` in separate top-level contexts).

```nginx
# At the top level of nginx.conf, alongside the existing http {} block.
# stream {} must be a sibling of http {}, not nested inside it.
stream {
    server {
        listen 3013;
        proxy_pass 127.0.0.1:50351;   # rishab's server gRPC
        proxy_timeout 86400s;          # match long-lived sync streams
    }
}
```

Why `stream` (TCP passthrough) and not `http2` + `grpc_pass`:
- Future-proofs for mTLS (PIQ-116) — nginx never opens the gRPC stream, so the TLS handshake survives end-to-end when we add it
- Simpler config, no gRPC framing logic in the proxy
- Matches the Linux design's choice

The agent will dial `122.176.60.111:3013` (or whatever public hostname maps there) to reach this server.

### 2. Bake server address into the binary

New file:

```go
// cmd/agent/cli/defaults.go
package cli

// DefaultServerAddress is the Patch Manager gRPC address baked into the
// binary at build time via -ldflags. Empty in dev builds; set to the
// public address (e.g. "patchiq.skenzer.com:3013") in release builds.
var DefaultServerAddress = ""
```

Build command (used in CI for the release binary):

```
go build -ldflags "-X github.com/skenzeriq/patchiq/cmd/agent/cli.DefaultServerAddress=patchiq.skenzer.com:3013" \
    -o patchiq-agent.exe ./cmd/agent
```

Server address resolution order in `cmd/agent/cli/install.go::validateInstallOpts`:
1. Explicit `--server` flag
2. `PATCHIQ_AGENT_SERVER_ADDRESS` env var (already supported by daemon path)
3. `DefaultServerAddress` (baked in)
4. Error: "no server address — must build with -ldflags or pass --server"

The headless mode error message changes from "must pass --server" to "no server address available" when `DefaultServerAddress` is empty AND no flag/env supplied. Interactive (TUI) mode falls through to a manual server-input step (existing TUI step) only when nothing is baked in.

### 3. Auto-launch wizard on first run

Modify `cmd/agent/main.go` startup logic:

- Resolve config file path using existing `config_windows.go` / `config_unix.go`
- Check: does the config file exist?
  - **No** → dispatch to `cli.RunInstall([]string{})` (interactive wizard with no flags)
  - **Yes** → run as daemon (current behavior)

When invoked with explicit subcommands (`install`, `service`, `status`, `scan`), preserve current behavior — only the no-args / no-config branch triggers the auto-launch.

This means: if a fresh `patchiq-agent.exe` is double-clicked on a box that has never had it installed, the wizard appears immediately. If the same binary is launched later (or restarted by the Windows service manager), it sees the config and runs as a daemon.

### 4. Wizard installs the service after enrollment

Extend `cmd/agent/cli/install_tui.go`. After the existing `stepWritingConfig` succeeds:

- New step `stepInstallingService` — calls `serviceInstall()` from `service_windows.go` (already exists, copy the call site, no new code in `service_windows.go`)
- New step `stepStartingService` — calls `serviceStart()` from `service_windows.go`
- New step `stepDone` — shows `agent_id`, hostname, "Endpoint registered. View it at https://<server>/endpoints"

Skip `stepServerInput` entirely when `DefaultServerAddress != ""` — jump straight to `stepTokenInput`.

**Admin elevation check at the very top of `RunInstall`:** on Windows, call a helper that checks if the current process has administrator rights (using `golang.org/x/sys/windows`). If not, print:

```
This installer must be run as Administrator.
Right-click patchiq-agent.exe and select "Run as administrator".
```

…and exit with `ExitError`. Refuse to enter any partial state. Symmetric with the Linux design's `os.Geteuid() != 0` check.

### 5. Daemon must skip enrollment when already enrolled

The wizard performs enrollment and writes `agent_id` into `agent.yaml`. When the Windows service starts the same binary as a background process, it reads the config, sees `agent_id` is set, and **must skip the enrollment RPC and go straight to heartbeat**.

This is a verification + tiny fix item, not a major rebuild:
- Read `internal/agent/comms/` to confirm the daemon path checks `state.AgentID()` (or equivalent) before calling `Enroll`
- If yes, no change needed
- If no, add the check — refuse to re-enroll if `agent_id` is already populated; just start heartbeat with the existing ID

### 6. CI build for the Windows release binary

Extend `.github/workflows/release.yml` (or wherever release builds happen) to add a Windows build matrix entry:

```yaml
- name: Build Windows agent
  env:
    GOOS: windows
    GOARCH: amd64
    PATCHIQ_PUBLIC_SERVER_ADDR: ${{ secrets.PATCHIQ_PUBLIC_SERVER_ADDR }}
  run: |
    go build -ldflags "-X github.com/skenzeriq/patchiq/cmd/agent/cli.DefaultServerAddress=${PATCHIQ_PUBLIC_SERVER_ADDR}" \
      -o dist/patchiq-agent.exe ./cmd/agent
```

Then upload `dist/patchiq-agent.exe` to object storage (MinIO) and insert/update a row in `agent_binaries` for `os_family=windows, arch=amd64`.

A new `Makefile` target `build-agent-windows` runs the same command locally for ad-hoc release builds.

### 7. Agent Downloads page (web/)

Extend `web/src/pages/agent-downloads/AgentDownloadsPage.tsx`:

- Already lists binaries via `useAgentBinaries()` hook (verified earlier this session)
- Already generates registration tokens via `useCreateRegistration()`
- For the Windows binary card, render install instructions:

> 1. Download `patchiq-agent.exe`
> 2. Right-click the file → **Run as administrator**
> 3. When prompted, paste this token: `K7M-3PQ-9XR` *(copy button)*
> 4. The agent will install itself as a Windows service and connect automatically.

No new API endpoints. No `installer_type` column needed (single binary path).

## File-change list

| Area | File | Action |
|---|---|---|
| Infra | `deploy/nginx/patchiq.conf` | Add `stream {}` top-level block + server block for rishab gRPC route on `:3013 → :50351` |
| Agent | `cmd/agent/cli/defaults.go` | NEW — `DefaultServerAddress` ldflags var |
| Agent | `cmd/agent/cli/install.go` | Wire `DefaultServerAddress` into `validateInstallOpts` resolution chain; add Windows admin elevation check at top of `RunInstall` |
| Agent | `cmd/agent/cli/install_tui.go` | Add `stepInstallingService` + `stepStartingService` + `stepDone`; skip `stepServerInput` when default is baked in |
| Agent | `cmd/agent/cli/elevation_windows.go` | NEW — `isAdmin() bool` using `golang.org/x/sys/windows` |
| Agent | `cmd/agent/cli/elevation_unix.go` | NEW — stub returning `true` (other OSes handle their own elevation) |
| Agent | `cmd/agent/main.go` | If no config file at resolved path → dispatch to `cli.RunInstall([]string{})` |
| Agent | `internal/agent/comms/` (path TBD after read) | Verify daemon skips `Enroll` when `agent_id` already in state; tiny fix if it doesn't |
| CI | `.github/workflows/release.yml` | Add Windows build step with `-ldflags` baking `DefaultServerAddress`, upload to object storage, insert into `agent_binaries` |
| CI | `Makefile` | NEW target `build-agent-windows` for local ad-hoc builds |
| Secrets | GitHub Actions repo secret `PATCHIQ_PUBLIC_SERVER_ADDR` | NEW — public gRPC address used at build time |
| UI | `web/src/pages/agent-downloads/AgentDownloadsPage.tsx` | Update Windows card with new operator instructions; remove any stale env-var / SSH-tunnel guidance |
| Tests | `cmd/agent/cli/install_test.go` | Extend table-driven tests for `DefaultServerAddress` resolution order |
| Tests | `cmd/agent/cli/install_tui_test.go` (if exists) | Cover the new service-install steps and the skip-server-input branch |
| Docs | `docs/agent-onboarding-windows.md` | NEW — operator-facing one-page runbook |

## Testing strategy

Per CLAUDE.md anti-slop rule #2 (no code without a failing test first), each implementation task in the plan will:

1. Write a failing table-driven test for the unit under change
2. Make it pass with minimal code
3. Refactor

Specific test coverage targets:

- **`defaults_test.go`** — verify the default value resolution chain (flag > env > ldflags > error). Use a build-tag-free helper that takes `DefaultServerAddress` as a parameter to keep tests deterministic.
- **`install_test.go`** — extend existing tests with cases where `--server` is omitted but `DefaultServerAddress` is set; assert no error.
- **`install_tui_test.go`** — assert the TUI skips `stepServerInput` when default is non-empty; assert post-enrollment steps are sequenced correctly.
- **`elevation_windows_test.go`** — gated by `//go:build windows`; covers the admin-check helper. CI on Linux skips it cleanly.
- **`main_test.go`** — table-driven: given (config exists / config missing) × (args / no args), assert the correct dispatch (daemon vs install wizard).
- **Integration check (manual, pre-merge):** build the binary with a baked-in test address, run on the actual Windows test box (DESKTOP-629B940), confirm the wizard flow end-to-end and that the service survives a reboot.

## Risks

- **Daemon enrollment skip not yet verified.** If `internal/agent/comms/` doesn't already check for an existing `agent_id` before calling `Enroll`, the daemon would re-enroll on every service start, creating duplicate endpoints. Must verify in the first task of the plan and add the check if missing. Estimated additional work if needed: ~30 minutes.
- **Bubble Tea TUI behavior in a non-interactive Windows console.** When the operator double-clicks `patchiq-agent.exe`, Windows opens a console window. Bubble Tea typically works fine in `cmd.exe` and Windows Terminal, but edge cases with raw mode and terminal size detection on first launch can cause flicker or input issues. Mitigation: test on the actual Windows box before declaring done; have a fallback `--no-tui` headless mode (already exists in the install command).
- **`PATCHIQ_PUBLIC_SERVER_ADDR` secret management.** The CI secret needs to be set in the GitHub Actions repo settings before the first release build. Coordinate with whoever has admin on the repo.
- **Single public gRPC port for all developers' servers.** This design adds rishab's port (`:3013 → :50351`). Sandy's existing `:3003 → :50151` block is left alone. When more developers need public access, the pattern is clear (add a new server block, pick a free port in `3000-3199`). Document this in the runbook.
- **No uninstaller in the operator's eyes.** Without an MSI, there's no Add/Remove Programs entry. To uninstall, the operator must run `patchiq-agent.exe service uninstall` (already supported in `service_windows.go`) and then delete the binary + `C:\ProgramData\PatchIQ\` manually. Document this in the runbook. Acceptable for beta; revisit if customer feedback complains.
- **SmartScreen warning on first run.** Unsigned `.exe` triggers Windows Defender SmartScreen "Unknown publisher" warning. Operator clicks "More info → Run anyway." Document this in the runbook with a screenshot. Code signing is a follow-up PR once a cert is procured.

## Open questions to resolve before implementation

None blocking. The daemon-enrollment-skip verification is the only "go look at the code" item, and it's a 5-minute read at the start of the plan.

## What success looks like

A new Windows machine joins PatchIQ in under 60 seconds without ever opening a terminal, without typing any flag, without copying any URL — just download, right-click, paste token, watch it appear in the dashboard.
