# PatchIQ — Claude Code Conventions

> Primary AI guardrail. Every Claude Code session reads this file.
> ADR: [010-anti-slop-development-guardrails](docs/adr/010-anti-slop-development-guardrails.md)

---

## Development Workflow

### Three Tracks

Every task follows one of three tracks. The GitHub issue specifies which.

**Standard Track** — features, enhancements, refactors:
1. `/brainstorm` → design doc in `docs/plans/`
2. `/write-plan` → granular TDD task list
3. Worktree creation (auto via `using-git-worktrees` skill)
4. TDD implementation (auto via `test-driven-development` skill)
5. `/review-pr all parallel` → fix all Critical/Important issues
6. `/commit-push-pr` → GitHub PR
7. Human review → squash merge

**Fix Track** — bugs, incidents:
1. Reproduce the bug
2. Systematic debugging (auto via `systematic-debugging` skill)
3. Write failing test → fix → verify
4. `/review-pr all parallel`
5. `/commit-push-pr`
6. Human review → squash merge

**Quick Track** — docs, chores, scaffolding, <30min work:
1. Worktree creation (auto via skill)
2. Do the work
3. `/review-pr` (targeted, e.g. `/review-pr code errors`)
4. `/commit-push-pr`
5. Human review → squash merge

### Decision Tree

```
Task arrives →
├── New feature / enhancement / refactor? → Standard Track
├── Bug fix / incident?                   → Fix Track
└── Docs / chore / scaffolding / <30min?  → Quick Track
```

### Slash Commands (complete list)

| Command | Plugin | What It Does |
|---------|--------|-------------|
| `/brainstorm` | superpowers | Design exploration → approach selection → design doc |
| `/write-plan` | superpowers | Granular TDD implementation plan from design |
| `/review-pr` | pr-review-toolkit | 6-agent code review (code, errors, types, tests, comments, simplify) |
| `/commit` | commit-commands | Stage + commit with proper message |
| `/commit-push-pr` | commit-commands | Commit + push + create PR in one flow |
| `/clean_gone` | commit-commands | Remove merged local branches + worktrees |
| `/revise-claude-md` | claude-md-management | Update CLAUDE.md with session learnings |

### Auto-Activated Skills (do not invoke manually)

| Skill | Activates When | Enforces |
|-------|---------------|----------|
| `brainstorming` | `/brainstorm` invoked | No code before design approval |
| `writing-plans` | `/write-plan` invoked | Bite-sized TDD tasks with file paths |
| `test-driven-development` | Before any implementation | No production code without failing test |
| `systematic-debugging` | Before any bug fix | 4-phase root cause analysis |
| `verification-before-completion` | Before claiming done | Run tests, read output, evidence required |
| `using-git-worktrees` | Starting feature work | Isolated worktree in `.worktrees/` |
| `finishing-a-development-branch` | All tasks complete | Test → merge/PR/keep/discard options |
| `requesting-code-review` | After completing work | Dispatch review, fix Critical/Important |
| `receiving-code-review` | When getting feedback | Verify claims before implementing suggestions |

### Mandatory Tool Usage (NEVER bypass)

NEVER do the manual equivalent. Use the correct command/skill.

| Action | NEVER do this | ALWAYS do this |
|--------|--------------|----------------|
| Review a PR | Read diff + comment manually | `/review-pr` (even for small PRs: `/review-pr code errors`) |
| Commit | `git commit` | `/commit` |
| Ship (push + PR) | `git push` / `gh pr create` | `/commit-push-pr` |
| Merge a PR | Local `git merge` + push | GitHub PR merge button (required — see Merge Rules below) |
| PR base branch | Stack onto another dev's `dev-a-*` | Always target `dev-a` / `dev-b` / `dev-c` directly |
| Create branch | `git checkout -b` | Worktree via skill |
| Design a feature | Start coding | `/brainstorm` first (Standard Track) |
| Plan implementation | Jump to code | `/write-plan` first (Standard Track) |
| Fix a bug | Guess at fixes | `systematic-debugging` skill → failing test → fix |

### Merge Rules (NEVER bypass)

1. **Never merge locally.** Always use the GitHub PR merge button. Local `git merge` + push bypasses CI gates and leaves the GitHub PR stuck in `OPEN` or `CLOSED` state, so history and reality diverge. If you need to merge `dev-a` into a feature branch to resolve conflicts, that's fine — but the final landing of a PR into `dev-a` must go through the button.
2. **Never stack PRs on other developers' branches.** A PR's base branch must be `dev-a`, `dev-b`, or `dev-c` — never `dev-a-rishab`, `dev-a-danish`, or another feature branch. If your work genuinely depends on an unmerged PR, wait for it to land, then rebase. Stacking hides dependencies and produces unreviewable histories (see PR #352 ↔ PR #355 incident on dev-a, 2026-04).
3. **Never push to `dev-a`/`dev-b`/`dev-c`/`dev`/`main`/`production` directly.** All changes land through PRs. If you need a hotfix, open a PR — even a trivial one.

Before starting any issue: read the issue's **Workflow** and **Dependencies** sections. Follow steps in order. If blocked, do not start.

**Cross-cutting changes**: When modifying one platform (server/hub/agent), check if others need the same change (config fields, middleware, init sequence).

---

## Anti-Slop Rules

1. **No feature work without design.** Run `/brainstorm` first.
2. **No code without a failing test first.** TDD is mandatory for features and bug fixes.
3. **No completion claims without evidence.** Run the tests. Read the output.
4. **No speculative debugging.** If 3+ fix attempts fail, STOP — the problem is architectural.
5. **No unnecessary abstractions.** Three similar lines > premature helper function. Abstract at 3+ callers.
6. **No over-engineering.** Do exactly what was asked.
7. **No orphaned imports.** Add import → use it. Remove code → remove its imports.
8. **No generic error messages.** Every error includes context about what was attempted.
9. **No fmt.Println or log.Println.** Use `slog`. Always.
10. **No TODO/FIXME without issue reference.** `// TODO(PIQ-42): implement caching` is fine. `// TODO: fix later` is not.
11. **No working on blocked issues.** Check the Dependencies section. If a dependency isn't merged, don't start the issue.

---

## Git Conventions

**Branch naming**: `{type}/{short-description}` (e.g., `feat/workflow-builder`)
**Commit format**: `type(scope): description` + optional body + `Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>`

**Branch hierarchy**:
```
production  ← always safe, ready to host on client infra
  ↑ merge
main        ← stable beta, tested
  ↑ merge
dev         ← stable after QA and verification
  ↑ merge
dev-a/b/c   ← active development (per-developer or per-feature)
```

**Rules**:
- All development happens on `dev-*` branches (e.g., `dev-b`, `dev-a-danish`)
- Merge to `dev` only after QA and verification pass
- Merge to `main` only when `dev` is stable
- Merge to `production` only for client-facing releases
- No direct pushes to `dev`, `main`, or `production`
- Squash merge when promoting between tiers

---

## Project Identity

PatchIQ is an enterprise patch management platform with three tiers:

```
Hub Manager (SaaS) ──REST API──> Patch Manager (On-prem) ──gRPC+mTLS──> Agents (Endpoints)
```

| Platform | Backend | Frontend | Database | Base Ports |
|----------|---------|----------|----------|------------|
| Patch Manager | `internal/server/` | `web/` | PostgreSQL 16 | HTTP :8080, gRPC :50051, UI :3001 |
| Hub Manager | `internal/hub/` | `web-hub/` | PostgreSQL 16 | HTTP :8082, UI :3002 |
| Agent | `internal/agent/` | `web-agent/` | Local SQLite | HTTP :8090, UI :3003 |
| Shared | `internal/shared/` | `packages/ui/` | — | — |

> **Ports are per-user.** Base ports shown above are for heramb (offset +0). Each developer gets a +100 offset via `.env`. See [Shared Dev Server](#shared-dev-server) section. Run `make dev-ports` to see all user ports.

### Current State

**Milestone status**: M0 (Skeleton) complete. M1 (Core Loop) complete. M2 (Usable Product) complete. **Platform maturation in progress** — making every feature work E2E for client POC deployment.

**Current focus**: No new features. Every existing feature must work flawlessly end-to-end. The beta deploys on client infrastructure — bugs directly impact negotiation leverage.

See `docs/roadmap.md` for full M0-M4 milestone details.

---

## Import Rules (Hard Constraints)

```
shared/ <--- server/     (server imports shared)
shared/ <--- agent/      (agent imports shared)
shared/ <--- hub/        (hub imports shared)

server/ -X-> agent/      (NEVER)
server/ -X-> hub/        (NEVER)
agent/  -X-> server/     (NEVER)
agent/  -X-> hub/        (NEVER)
hub/    -X-> server/     (NEVER)
hub/    -X-> agent/      (NEVER)
```

---

## Backend Architecture

### Tech Stack

Go 1.25.0 | chi/v5 (HTTP) | pgx/v5 + sqlc (database) | goose (migrations) | koanf (config) | Watermill (events) | River (job queue) | gRPC + protobuf (agent comms) | modernc.org/sqlite (agent) | Valkey (cache/sessions) | OpenTelemetry + slog (observability) | Zitadel (IAM/OIDC) | Shoutrrr (notifications)

### Platform Init Sequence

All platforms follow: config → logger → otel → signal context → database → event bus → services → HTTP/gRPC servers.

**Server** (`cmd/server/main.go`): Config → Logger → OTel → Signal → PostgreSQL → Watermill → River (12 workers) → Discovery/CVE/Deployment/Compliance/Notification engines → gRPC :50051 → HTTP :8080

**Hub** (`cmd/hub/main.go`): Config → Logger → OTel → Signal → PostgreSQL → Watermill → Feed Registry (6 feeds) → River → HTTP :8082

**Agent** (`cmd/agent/main.go`): Config → Logger → OTel → SQLite → Module Registry (inventory, patcher) → gRPC client → HTTP :8090. Supports CLI subcommands: `install`, `status`, `scan`, `service`.

### Server Services (`internal/server/`)

| Package | Purpose | River Jobs |
|---------|---------|------------|
| `api/v1/` | REST handlers: endpoints, groups, patches, cves, deployments, schedules, policies, workflows, workflow_executions, compliance, audit, notifications, discovery, roles, user_roles, hub_sync, license, dashboard, health, auth, settings_iam | — |
| `grpc/` | AgentService: Enroll, Heartbeat, SyncOutbox, SyncInbox | — |
| `deployment/` | State machine, wave dispatcher, schedule checker, timeout checker, evaluator | TimeoutJob (5min), ScanJob (24h), WaveDispatcherJob (30s), ScheduleCheckerJob (1min) |
| `cve/` | NVD sync, CISA KEV, CVE-patch correlation, CVSS scoring | NVDSyncJob (24h), EndpointMatchJob (on scan) |
| `discovery/` | Patch repository scanning | DiscoveryJob (60min) |
| `compliance/` | Framework evaluation (CIS, PCI-DSS, HIPAA, NIST, ISO 27001, SOC 2) | ComplianceEvalJob (6h) |
| `notify/` | Shoutrrr (email, Slack, Discord, webhook) | Async delivery |
| `policy/` | Policy engine, endpoint-to-policy matching | — |
| `workflow/` | Workflow DAG execution | — |
| `auth/` | Zitadel OIDC integration, session management | — |
| `workers/` | Audit retention (partition pruning), catalog sync | AuditRetentionJob (24h), CatalogSyncJob (on event) |
| `mcp/` | Model Context Protocol (AI integration) | — |
| `store/` | PostgreSQL: 45 migrations, 28 sqlc query files | — |

### Hub Services (`internal/hub/`)

| Package | Purpose |
|---------|---------|
| `api/v1/` | REST handlers: catalog, feeds, clients, licenses, sync, dashboard, health |
| `feeds/` | 6 feed implementations: NVD (6h), CISA KEV (12h), MSRC (12h), RedHat OVAL (12h), Ubuntu USN (12h), Apple (12h) |
| `catalog/` | Normalization pipeline: RawEntry → patch catalog, CVE linking, deduplication |
| `workers/` | FeedSyncJob per feed on periodic schedule |
| `license/` | License issuance and management |
| `store/` | PostgreSQL: 11 migrations, 13 sqlc query files |

### Agent Architecture (`internal/agent/`)

Offline-first, minimal dependencies. Module registry pattern.

| Package | Purpose |
|---------|---------|
| `inventory/` | OS-specific collectors: APT, YUM, WUA, Hotfix, Homebrew, macOS softwareupdate |
| `patcher/` | OS-specific executors: APT, YUM, WUA, MSI, MSIX, Homebrew, macOS, generic installer |
| `comms/` | gRPC client: Enrollment → Heartbeat → SyncOutbox → SyncInbox |
| `api/` | Local HTTP: status, patches, history, logs |
| `store/` | SQLite: patches, history, logs, outbox, inbox |

### Shared Packages (`internal/shared/`)

| Package | Purpose |
|---------|---------|
| `tenant/` | Context injection, HTTP middleware (`X-Tenant-ID` header) |
| `user/` | User context injection (`X-User-ID` header) |
| `domain/` | DomainEvent envelope (ULID IDs), EventBus interface, audit schema |
| `idempotency/` | HTTP middleware, Valkey/in-memory cache (24h TTL) |
| `config/` | YAML loader (koanf), config hierarchy (tenant → group → endpoint) |
| `crypto/` | AES (notification credentials), RSA (mTLS certs) |
| `otel/` | OTLP gRPC exporters, HTTP/gRPC middleware, slog handler with trace_id |
| `license/` | Tiers (FREE, STANDARD, ENTERPRISE), validation |
| `models/` | Shared domain types |
| `protocol/` | gRPC + protobuf enum helpers |

### gRPC Services (`proto/patchiq/v1/`)

**AgentService** (server ↔ agent):
- `Enroll(EnrollRequest) → EnrollResponse`
- `Heartbeat(stream HeartbeatRequest) → stream HeartbeatResponse`
- `SyncOutbox(stream OutboxMessage) → stream OutboxAck`
- `SyncInbox(InboxRequest) → stream CommandRequest`

**HubService** (hub ↔ server):
- `SyncCatalog(SyncCatalogRequest) → SyncCatalogResponse`
- `ValidateLicense(ValidateLicenseRequest) → ValidateLicenseResponse`

### Middleware Chain (HTTP)

Server/Hub chi middleware order: OTel → Request ID → Real IP → slog logger → Panic Recovery → CORS → JWT (Zitadel, optional) → Tenant → User ID → Idempotency → RBAC (per handler).

### Domain Events

60+ event types across: Endpoint, Heartbeat, Inventory, Command, Group, Policy, Deployment (incl. waves, rollback), Schedule, Patch, Catalog, CVE, Compliance, Role/IAM, Notification, License, Audit, Workflow. Event bus: Watermill + PostgreSQL transport. Audit subscriber captures all events (`*` wildcard). Topics defined in `internal/server/events/topics.go`.

---

## Frontend Architecture

### Tech Stack

React 19 | TypeScript 5.7 strict | Vite 6.2 | Tailwind CSS 4.2 | pnpm workspaces | Radix UI (via @patchiq/ui) | TanStack Query 5 (server state) | TanStack Table 8 | react-hook-form 7 + Zod 4 | react-router 7 | openapi-fetch + openapi-typescript (typed API clients) | Vitest 3.2 | @testing-library/react 16

### App Structure (all three follow same pattern)

```
src/
├── api/
│   ├── client.ts          # openapi-fetch client, tenant headers
│   ├── types.ts           # Generated from OpenAPI spec (do not hand-edit)
│   └── hooks/             # TanStack Query hooks, one per resource
├── app/
│   ├── routes.tsx          # React Router v7 config
│   ├── auth/AuthContext.tsx # Auth context (Zitadel OIDC / dev stubs)
│   └── layout/             # AppLayout, AppSidebar, TopBar
├── pages/                  # One folder per route
├── components/             # Shared page-level components
├── flows/                  # Complex interactive UIs (workflow builder)
├── types/                  # Domain type definitions
├── lib/                    # Utilities
└── __tests__/              # Vitest tests mirroring src/ structure
```

### Per-App Details

**web/** (Patch Manager — richest UI):
- 31 routes: dashboard, endpoints, tags, patches, cves, policies, deployments, workflows, compliance, audit, notifications, settings (license, IAM), admin/roles, admin/users/roles, agent-downloads
- 23 API hooks in `api/hooks/`
- Workflow builder: @xyflow/react (DAG canvas) + elkjs (layout) + CodeMirror (script editor)
- Charts: Recharts
- Generated types: 5831 lines

**web-hub/** (Hub Manager):
- 11 routes: dashboard, catalog, feeds, licenses, clients, deployments, settings
- 6 API hooks
- Generated types: 308 lines

**web-agent/** (Agent):
- 9 routes: status, pending, hardware, software, services, history, logs, settings
- 9 API hooks
- Generated types: 371 lines

### Shared UI (`packages/ui/`)

32 components total. 10 custom (DotMosaic, EmptyState, ErrorState, MonoTag, PageHeader, RingGauge, SeverityText, SkeletonCard, StatCard, ThemeConfigurator) + 22 shadcn/ui primitives (AlertBanner, Avatar, Badge, Button, Card, Collapsible, DataTable, Dialog, DropdownMenu, Input, Progress, RingChart, Select, Separator, Sheet, Sidebar, Skeleton, Sonner, SparklineChart, Switch, Tabs, Tooltip). Tailwind v4 with `@theme` CSS variables for light/dark mode. `cn()` utility from clsx + tailwind-merge.

### Key Frontend Patterns

- **API hooks**: All data fetching via custom hooks in `api/hooks/`, never call `api.GET` directly from components
- **Forms**: react-hook-form + zodResolver schema → register fields → validate → mutate
- **Tables**: TanStack Table with column helpers, wrapped in shared DataTable
- **State**: TanStack Query for server state, React Context for auth. No Zustand/Redux currently in use
- **Dev proxy**: Vite proxies `/api` to backend (ports from env vars, defaults: web→:8080, web-hub→:8082, web-agent→:8090)

---

## Database Conventions

- **Every tenant-scoped table**: `tenant_id UUID NOT NULL` as first column after PK + RLS policy.
- **Every `BeginTx()`**: must call `SET LOCAL app.current_tenant_id = $tenant_id`.
- **Global tables** (no tenant_id): `tenants`, `patch_catalog`, `cve_feeds`, `agent_binaries`.
- **Migrations**: goose, sequential numbering. Server: 45 migrations in `internal/server/store/migrations/`. Hub: 11 migrations in `internal/hub/store/migrations/`.
- **Queries**: sqlc, one `.sql` file per resource in `internal/{server,hub}/store/queries/`.
- **Generated code**: `internal/{server,hub}/store/sqlcgen/` — committed to git. Regenerate: `make sqlc`.
- If unsure whether a table is tenant-scoped, it is. Default to adding `tenant_id`.

## Domain Events (Non-negotiable)

Every write operation MUST emit a domain event. Events go to the audit table (append-only, ULID IDs, partitioned monthly). Watermill handles pub/sub. If your code writes to the DB without emitting an event, it is a bug.

---

## Code Conventions

### Go
- `slog` for all logging. Every log line includes `trace_id` from context.
- Error handling: `fmt.Errorf("enroll endpoint: %w", err)`. No naked returns. No suppressed errors.
- Table-driven tests. `go test -race` always.
- Package names: lowercase, single word. Interface names: behavior name (no `I` prefix).
- Handlers in `internal/{server,hub}/api/v1/`, one file per resource.
- Agent is minimal: no Watermill, no River, no PostgreSQL. SQLite + gRPC only.

### Frontend
- TypeScript strict mode. No `any` without justifying comment.
- `@patchiq/ui` for shared components. Don't duplicate shadcn/ui in app code.
- TanStack Query for server state. React Context for auth.
- react-hook-form + Zod for every form.
- File org: `pages/`, `components/`, `hooks/`, `api/`, `types/`.
- openapi-fetch for type-safe API calls. Never use raw fetch.

---

## Common Operations

```bash
# Development
make dev              # Boot dev environment (Docker + hot-reload all 3 platforms)
make dev-down         # Stop Docker and hot-reload
make dev-env          # Regenerate .env for current $USER
make dev-ports        # Print port table for all developers

# Build
make build            # Build all 3 Go binaries
make build-agents     # Cross-compile agents for all platforms (linux/windows/macOS)

# Test
make test             # Go tests with race detector
make test-integration # Integration tests (15min timeout, testcontainers)

# Lint
make lint             # Go lint (golangci-lint v2)
make lint-frontend    # TypeScript + Prettier + ESLint
make lint-all         # Go + frontend + protobuf lint

# Database
make migrate          # Run server migrations
make migrate-hub      # Run hub migrations
make migrate-status   # Show migration status
make seed             # Migrate + load seed data
make seed-demo        # Migrate + load demo seed data
make seed-clean       # Migrate + load clean seed (no demo data)
make seed-hub         # Load hub seed data
make seed-agent       # Seed agent locally

# Code generation
make sqlc             # Regenerate sqlc
make proto            # Lint + regenerate protobuf (buf)
make api-client       # Validate OpenAPI specs

# CI (local)
make ci               # Fast local CI (codegen → lint → test → build)
make ci-full          # Full local CI (+ integration tests + Docker builds)
make ci-quick         # Quick CI (changed packages only vs main)

# Utilities
make fmt              # Format Go code
make tidy             # Clean go.mod/go.sum
make clean            # Remove bins, tear down Docker
make setup-hooks      # Configure git hooks
```

---

## Shared Dev Server

All 4 developers work on one Linux server (32 cores, 64GB RAM, 1TB disk). Each developer gets isolated ports, containers, and databases via per-user `.env` files.

### Per-User Isolation

`scripts/dev-env.sh` generates `.env` (gitignored) with a port offset per user. **First `make dev` auto-generates it from `$USER`.** To regenerate: `make dev-env` or `./scripts/dev-env.sh <username>`.

| User | Offset | Server | Hub | Web UI | Postgres |
|------|--------|--------|-----|--------|----------|
| heramb | +0 | :8080 | :8082 | :3001 | :5432 |
| sandy | +100 | :8180 | :8182 | :3101 | :5532 |
| danish | +200 | :8280 | :8282 | :3201 | :5632 |
| rishab | +300 | :8380 | :8382 | :3301 | :5732 |

Each user gets: separate `COMPOSE_PROJECT_NAME` (isolated Docker containers/volumes), separate databases (`patchiq_dev_<user>`, `patchiq_hub_dev_<user>`), and separate host port bindings. All Go config env vars (`PATCHIQ_*`) and Vite proxy ports are set automatically.

**When invoking `scripts/dev-env.sh`, always pass the Linux username of the developer you're generating for.** The script maps usernames to port offsets. New developers must be added to `get_offset()` in the script.

### Resource Limits

Docker containers have CPU (1–2 cores), memory (512M–2G), and log limits (10MB × 3 files) per container. App-level log rotation via lumberjack (100MB × 5 files per service, opt-in via `log.file` config). Cleanup: `scripts/dev-cleanup.sh` (cron-ready, prunes Docker/Go cache/stale worktrees).

### Infrastructure Services

`docker-compose.dev.yml` (all ports via `.env` interpolation):
- **PostgreSQL 16** — per-user databases, all on one instance
- **Zitadel v2.71.6** — OIDC/IAM provider
- **Valkey 9** — cache/session store (Redis-compatible)
- **MinIO** — S3-compatible object storage
- **Grafana OTEL-LGTM** — traces, metrics, logs

Config files: `configs/server.yaml`, `configs/hub.yaml`, `configs/agent.yaml` (base config; env vars from `.env` override at runtime)

### CI / GitHub Actions

Self-hosted runner on this machine (`/home/heramb/actions-runner/`, systemd service: `actions.runner.herambskanda-patchiq.patchiq-local`). Workflows: `lint.yml`, `test-unit.yml`, `build.yml`, `release.yml` — all share concurrency group `self-hosted-ci` with `cancel-in-progress: false`, so CI jobs **queue sequentially** across all PRs. This prevents resource starvation on the shared server. Use `make ci-quick` for fast local pre-push validation.

---

## Key Documentation

| Document | Purpose |
|----------|---------|
| `docs/roadmap.md` | Milestone roadmap (M0-M4) |
| `docs/DEVELOPMENT-PROCESS.md` | Full workflow, tracks, commands, review process |
| `docs/PM.md` | Issue writing guide, sprint rhythm, PR review |
| `docs/PROJECT.md` | Plugin stack (8 always-on + 1 optional) + developer setup |
| `docs/blueprint/core/` | Tech stack, platform overview, project structure, UI spec, license tiers |
| `docs/blueprint/features/` | AI assistant, compliance engine, RBAC, HA/DR, multi-site, observability |
| `docs/blueprint/foundations/` | 7 architectural foundations (multi-tenancy, events, offline-first, etc.) |
| `docs/adr/` | 25 Architecture Decision Records (001-024 + template) |
| `docs/plans/` | 15 design docs and implementation plans |

## Generated Code (do not hand-edit)

`internal/{server,hub}/store/sqlcgen/` (sqlc), `gen/patchiq/v1/` (protobuf), `web/src/api/types.ts`, `web-hub/src/api/types.ts`, `web-agent/src/api/types.ts` (OpenAPI).

## Protected Files (require core dev review)

`internal/shared/tenant/`, `internal/shared/domain/`, `internal/shared/crypto/`, `internal/server/store/migrations/`, `internal/hub/store/migrations/`, `proto/patchiq/v1/`, `.github/workflows/`, `CLAUDE.md`, `.claude/settings.json`.

## Team

| Name | Role | Level | Scope |
|------|------|-------|-------|
| Heramb | PM + Dev 1 | core | Everything. Final reviewer. |
| Sandy | Dev 2 | core | Full-stack. Foundation work. |
| Rishab | Dev 3 | intern | risk:low tasks only. Core dev reviews PRs. |
| Danish | Dev 4 | intern | risk:low tasks only. Core dev reviews PRs. |
