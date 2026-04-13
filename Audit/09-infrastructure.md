# Infrastructure Audit — 2026-04-09

Audited: Docker, deploy configs, app configs, scripts, CI/CD workflows, Makefile, proto/buf, OpenAPI specs, go.mod, package.json, root-level files.

---

## 1. Stale/Artifact Files in Project Root

### 1A. `prototype-ui/` directory committed to git — **Important**
- **Path**: `/prototype-ui/` (5.6 MB, 130+ tracked files)
- **Issue**: Entire prototype UI with HTML mockups, a separate `package-lock.json` (5917 lines), WhatsApp screenshots (`prototype-ui/updated-version/WhatsApp Image *.jpeg`), and `tsconfig.app.tsbuildinfo` are committed to the repo. This was an early design reference (commit `537665d`) and is no longer relevant — the real apps (`web/`, `web-hub/`, `web-agent/`) are fully built.
- **Impact**: Bloats repo, confuses contributors, binary images inflate git history.
- **Fix**: Remove the directory and add it to `.gitignore`. If historical reference is needed, keep a link to the commit SHA in docs.

### 1B. Snapshot/debug files committed to root — **Minor**
- **Paths**: `agent-overview-snapshot.md`, `hardware-snapshot.md`, `compliance-page.yml`, `policy-detail-snap.md`
- **Issue**: These look like point-in-time debug/design snapshots committed to the repo root. They are tracked in git and serve no runtime or CI purpose.
- **Fix**: Move to `docs/archive/` or remove entirely.

---

## 2. Docker Issues

### 2A. Postgres password mismatch between docker-compose and app configs — **Critical**
- **docker-compose.dev.yml** line 9: `POSTGRES_PASSWORD: patchiq`
- **configs/server.yaml** line 24: `postgres://patchiq:patchiq@localhost:5432/...` (password: `patchiq`)
- **configs/hub.yaml** line 21: `postgres://patchiq:patchiq_dev@localhost:5432/...` (password: `patchiq_dev`)
- **scripts/dev-env.sh** line 101: `postgres://patchiq:patchiq_dev@localhost:...` (password: `patchiq_dev`)
- **Issue**: The Postgres container sets password to `patchiq`, but `hub.yaml` and `dev-env.sh` use `patchiq_dev`. This works only because the env var overlay from `.env` overrides `hub.yaml` at runtime, but `server.yaml` hardcodes the wrong password (`patchiq` instead of `patchiq_dev`). If anyone runs the server with just `server.yaml` and no `.env`, it would connect with the Docker-set password by coincidence — but it is still inconsistent.
- **Fix**: Align all configs to use the same password (`patchiq_dev` in dev-env.sh is the canonical source, so update `docker-compose.dev.yml` line 9 to `POSTGRES_PASSWORD: patchiq_dev` and `server.yaml` lines 24/29 to use `patchiq_dev`).

### 2B. No Valkey config section in `configs/server.yaml` — **Important**
- **Issue**: `ServerConfig` struct has a `Valkey ValkeySettings` field (loader.go:17), and `dev-env.sh` generates `PATCHIQ_VALKEY_URL`, but `configs/server.yaml` has no `valkey:` section at all. The config works only via env var overlay. If a developer examines server.yaml to understand the config, they will not know Valkey is required.
- **Fix**: Add `valkey: url: "redis://localhost:6379"` to `configs/server.yaml`.

### 2C. No OTel endpoint in hub.yaml — **Minor**
- **Issue**: `configs/hub.yaml` line 34-35 has `otel: endpoint: ""` (empty string), but `HubConfig` includes `OTel OTelSettings`. Hub telemetry is silently disabled unless overridden. The `dev-env.sh` does NOT generate a `PATCHIQ_HUB_OTEL_ENDPOINT` var, so hub never sends telemetry in dev.
- **Fix**: Add `PATCHIQ_HUB_OTEL_ENDPOINT=localhost:${OTLP_GRPC_PORT}` to `dev-env.sh` and set `otel.endpoint` in `hub.yaml` to match.

### 2D. No Dockerfile for agent — **Minor** (by design)
- The agent is a native binary, not containerized. This is intentional per the architecture (agents run on endpoints). Not a bug, but noted for completeness since `goreleaser` builds agent binaries cross-platform.

---

## 3. Config Inconsistencies

### 3A. CORS origins in server.yaml include stale entries — **Minor**
- **Path**: `configs/server.yaml` lines 11-21
- **Issue**: The `cors_origins` list includes `http://ssh.skenzer.com:3001` (an external domain), ports 4000/4100 (no known service), and 5173-5175 (Vite default ports that are not used — the actual Vite dev ports are 3001-3003). These are stale from earlier development.
- **Fix**: Remove stale origins. The env var `PATCHIQ_SERVER_CORS_ORIGINS` from `dev-env.sh` correctly generates per-user origins, so the yaml list should only contain the base-offset defaults.

### 3B. Nginx config hardcoded to sandy's ports — **Minor**
- **Path**: `deploy/nginx/patchiq.conf`
- **Issue**: All upstream ports are hardcoded to offset +100 (sandy): `:3101`, `:3102`, `:3103`, `:8180`, `:8182`, `:8190`, `:50151`. This is a single-user deploy config, not parameterized. Anyone else deploying with this config will point at the wrong services.
- **Fix**: Document that this is sandy-specific, or parameterize with env vars/templating.

### 3C. `server.yaml` hardcodes heramb's database name — **Minor**
- **Path**: `configs/server.yaml` line 24: `patchiq_dev_heramb`
- **Issue**: The base config references heramb's personal database. All other developers rely on env var overlay to fix this, but the file itself is misleading.
- **Fix**: Change to a generic `patchiq_dev` or add a comment stating env vars override this.

---

## 4. Script Issues

### 4A. `dev-cleanup.sh` uses wrong repo path — **Important**
- **Path**: `scripts/dev-cleanup.sh` lines 19, 28
- **Issue**: Hardcoded paths reference `/home/{user}/skenzeriq/patchiq`, but the actual repo path is `/home/{user}/patchiq-dev-a` (or similar per-worktree). The cleanup script will never find any repos and silently does nothing.
- **Fix**: Update paths to match actual repo locations. Consider making the repo path configurable or auto-detected.

### 4B. Duplicate install scripts in `scripts/` and `deploy/scripts/` — **Important**
- **Paths**: `scripts/install-agent-macos.sh` vs `deploy/scripts/install-agent-macos.sh` (351 vs unknown lines, already diverged in usage comments)
- **Issue**: Two copies of macOS agent installer exist in different directories and have already diverged. Neither is referenced by Makefile or CI. Similarly, `scripts/install-agent-windows.ps1` duplicates `deploy/scripts/install-agent.ps1`.
- **Fix**: Consolidate to one canonical location (`deploy/scripts/` for deploy artifacts, or `scripts/` for dev scripts). Remove the duplicate.

### 4C. Install/uninstall scripts not referenced anywhere — **Minor**
- **Paths**: `scripts/install-agent-linux.sh`, `scripts/install-agent-macos.sh`, `scripts/install-agent-windows.ps1`, `scripts/uninstall-agent-*.sh/.ps1`, `deploy/scripts/install-agent.ps1`, `deploy/scripts/install-agent-macos.sh`
- **Issue**: None of these are referenced by the Makefile or any CI workflow. They are standalone scripts for field deployment, which is fine, but their discoverability is low and there is no automated testing or validation.
- **Fix**: Add a Makefile target `install-scripts-lint` or at minimum document them in a deployment guide.

---

## 5. CI/CD Gaps

### 5A. `release.yml` missing concurrency group — **Important**
- **Path**: `.github/workflows/release.yml`
- **Issue**: All other workflows (`lint.yml`, `test-unit.yml`, `build.yml`) use `concurrency: group: self-hosted-ci` to prevent resource starvation on the shared self-hosted runner. `release.yml` does NOT have this, meaning a release triggered during a CI run could starve the machine.
- **Fix**: Add `concurrency: group: self-hosted-ci` and `cancel-in-progress: false` to `release.yml`.

### 5B. Release workflow does not verify `web-agent/dist` — **Important**
- **Path**: `.github/workflows/release.yml` lines 51-63
- **Issue**: The "Verify frontend assets" step checks `web/dist` and `web-hub/dist`, but NOT `web-agent/dist`. If the agent UI build fails silently, the release proceeds without it.
- **Fix**: Add `web-agent/dist` to the verification loop.

### 5C. No Windows agent build in CI `build.yml` matrix — **Minor**
- **Path**: `.github/workflows/build.yml` lines 19-38
- **Issue**: The build matrix includes linux/amd64, linux/arm64, darwin/arm64, darwin/amd64 for the agent, but NOT windows/amd64. The Makefile `ci-go-build` target (line 229) does cross-compile for Windows, but the CI workflow does not.
- **Fix**: Add `{binary: agent, goos: windows, goarch: amd64}` to the build matrix.

### 5D. CI workflows only trigger on `main` branch — **Minor**
- **Path**: All workflow `on:` triggers: `push: branches: [main]` / `pull_request: branches: [main]`
- **Issue**: Per CLAUDE.md, the branch hierarchy is `dev-* -> dev -> main -> production`. PRs targeting `dev` will not trigger any CI checks. Developers must rely on `make ci` locally or the pre-push hook.
- **Fix**: Consider adding `dev` to the `pull_request.branches` list, or document that CI is local-only for dev branches.

---

## 6. Makefile Issues

### 6A. `seed-agent` target is fragile — **Minor**
- **Path**: `Makefile` lines 126-127
- **Issue**: The `seed-agent` target uses a chain of fallbacks with `||`, background process (`&sleep 2 && kill %%`), and `2>/dev/null` suppression. The `&sleep 2` is missing a space/semicolon before `sleep`, which means it backgrounds the `go run` and immediately runs `sleep 2` — this is bash syntax that works but is confusing. The `kill %%` is also non-standard.
- **Fix**: Rewrite as a cleaner script or use a dedicated seed subcommand in the agent binary.

### 6B. `api-client` target says "deferred to M1" — **Minor**
- **Path**: `Makefile` line 146
- **Issue**: The message says "api-client generation not yet configured (deferred to M1)" but M1 is complete per CLAUDE.md. The target only validates OpenAPI specs, it does not generate clients.
- **Fix**: Update the message or implement client generation.

---

## 7. Dependency Health

### 7A. Go module looks healthy — **No issues**
- Go 1.25.0 with recent dependencies. No obviously stale versions. `go-ole`, `wmi`, `gopsutil` are Windows-specific agent deps (legitimate). All indirect deps look reasonable.

### 7B. Root `package.json` engines spec is loose — **Minor**
- **Path**: `package.json` line 15: `"pnpm": ">=9"`
- **Issue**: CI uses `pnpm@10` (all workflows), but the engine spec allows pnpm 9. This could cause lockfile incompatibilities if a developer uses pnpm 9 locally.
- **Fix**: Update to `"pnpm": ">=10"` to match CI.

---

## 8. Proto/API Spec Issues

### 8A. buf version mismatch between Makefile and CI — **Minor**
- **Makefile** `proto-tools` target (line 135): installs `buf@v1.66.0`
- **CI** `codegen-check` job (lint.yml line 109): installs `buf@v1.66.1`
- **Fix**: Align to same version (v1.66.1).

---

## 9. goreleaser Issues

### 9A. goreleaser Dockerfile references may conflict with multi-stage builds — **Minor**
- **Path**: `.goreleaser.yaml` lines 37-56
- **Issue**: The goreleaser docker builds reference `deploy/docker/Dockerfile.server` and `Dockerfile.hub`, but these Dockerfiles are multi-stage (frontend + Go build + distroless). goreleaser already builds the Go binary separately, so the Dockerfile's Go build stage will duplicate work. This may cause the Docker image to use the Dockerfile-built binary rather than goreleaser's binary (with proper ldflags/version info).
- **Fix**: Create separate production Dockerfiles that COPY the pre-built binary from goreleaser, or use goreleaser's `extra_files` + a simpler Dockerfile.

---

## Summary Table

| # | Finding | Severity | Category |
|---|---------|----------|----------|
| 2A | Postgres password mismatch across configs | Critical | Config |
| 1A | `prototype-ui/` (5.6MB, WhatsApp images) committed | Important | Stale files |
| 2B | No Valkey config section in server.yaml | Important | Config |
| 4A | dev-cleanup.sh uses wrong repo paths | Important | Scripts |
| 4B | Duplicate install scripts (scripts/ vs deploy/scripts/) | Important | Scripts |
| 5A | release.yml missing concurrency group | Important | CI |
| 5B | Release does not verify web-agent/dist | Important | CI |
| 1B | Snapshot/debug files in project root | Minor | Stale files |
| 2C | Hub OTel endpoint empty, no env var generated | Minor | Config |
| 3A | Stale CORS origins in server.yaml | Minor | Config |
| 3B | Nginx config hardcoded to sandy's ports | Minor | Config |
| 3C | server.yaml hardcodes heramb's database | Minor | Config |
| 4C | Install/uninstall scripts unreferenced | Minor | Scripts |
| 5C | No Windows agent build in CI matrix | Minor | CI |
| 5D | CI only triggers on main branch | Minor | CI |
| 6A | seed-agent target fragile syntax | Minor | Makefile |
| 6B | api-client message says "deferred to M1" (M1 is done) | Minor | Makefile |
| 7B | pnpm engine spec allows v9, CI uses v10 | Minor | Deps |
| 8A | buf version mismatch (v1.66.0 vs v1.66.1) | Minor | Proto |
| 9A | goreleaser may duplicate Docker build work | Minor | Release |

**Critical: 1 | Important: 5 | Minor: 13**
