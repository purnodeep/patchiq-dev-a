# PatchIQ — Project Structure

> Canonical directory layout for the PatchIQ monorepo. All three platforms (Agent, Patch Manager, Hub Manager) live in a single Go module with shared frontend packages.

---

## Root

```
patchiq/
├── CLAUDE.md                        # AI development guardrails
├── Makefile                         # Build, test, run, generate commands
├── go.mod / go.sum
├── sqlc.yaml                        # sqlc config (references per-platform query/migration paths)
├── buf.gen.yaml                     # Protobuf codegen config (output → gen/)
├── .goreleaser.yaml
├── .golangci.yml
├── .pre-commit-config.yaml
├── .air.toml                        # Go hot-reload: Patch Manager
├── .air.agent.toml                  # Go hot-reload: Agent
├── .air.hub.toml                    # Go hot-reload: Hub Manager
├── .env.example                     # Environment variable template (never commit .env)
├── pnpm-workspace.yaml              # pnpm workspace config (web, web-hub, web-agent, packages/*)
```

---

## Entrypoints

```
cmd/
├── server/main.go                   # Patch Manager entrypoint
├── agent/main.go                    # Agent entrypoint
├── hub/main.go                      # Hub Manager entrypoint
└── tools/
    ├── migrate/main.go              # Goose migration runner CLI
    └── licensegen/main.go           # RSA license generation CLI
```

---

## Backend — `internal/`

### Patch Manager — `internal/server/`

```
internal/server/
├── api/                             # chi HTTP handlers
│   ├── middleware/                   # Auth, tenant context, rate-limit, request logging
│   ├── v1/                          # API v1 handlers (versioned from day 1)
│   └── router.go                    # chi router setup + middleware chain
├── grpc/                            # gRPC server (agent ↔ server communication)
├── engine/                          # Patch engine, policy evaluator, scheduler
├── workflow/                        # Visual workflow execution engine (@xyflow/react backend)
├── auth/                            # App-level fine-grained RBAC (Action+Resource+Scope)
├── iam/                             # Zitadel integration (SSO, OIDC, SAML, LDAP, user sync)
├── license/                         # RSA license validation + feature gating
├── compliance/                      # Compliance engine (HIPAA, SOC2, ISO 27001, FedRAMP)
├── store/                           # Database layer
│   ├── queries/                     # sqlc .sql query files (endpoints.sql, deployments.sql, etc.)
│   ├── sqlcgen/                     # sqlc-generated Go code (committed)
│   ├── migrations/                  # goose .sql migration files (001_init.sql, 002_rls.sql, etc.)
│   └── valkey/                      # Valkey cache + session store
├── events/                          # Watermill event subscribers
│   ├── handlers.go                  # Event handler functions
│   ├── publisher.go                 # Event publishing helpers
│   ├── router.go                    # Watermill router setup + middleware
│   └── topics.go                    # Topic name constants
├── workers/                         # River background jobs
│   ├── patch_scan.go                # Scheduled patch scanning
│   ├── deployment.go                # Deployment orchestration (waves, rollback)
│   ├── report.go                    # PDF report generation via Gotenberg
│   └── registry.go                  # Job type registration
├── notify/                          # Shoutrrr: email, Slack, webhook, Teams
├── mcp/                             # MCP server for AI assistant (Go SDK)
└── apm/                             # OpenTelemetry setup, health checks, readiness probes
```

### Agent — `internal/agent/`

```
internal/agent/
├── inventory/                       # OS-specific inventory collectors (APT, YUM, Windows Update, softwareupdate)
├── patcher/                         # OS-specific patch installers
├── comms/                           # gRPC client + SQLite offline queue
├── updater/                         # Agent self-update (release channel checking)
├── ui/                              # Local agent web UI (stdlib net/http, Go 1.22+ mux)
└── apm/                             # Agent telemetry (lightweight OTel + slog)
```

The agent is intentionally minimal — no Watermill, no River, no PostgreSQL. It uses SQLite for local storage and gRPC streaming for server communication.

### Hub Manager — `internal/hub/`

```
internal/hub/
├── api/                             # chi HTTP handlers (Hub REST API)
│   ├── middleware/
│   ├── v1/
│   └── router.go
├── catalog/                         # Patch catalog management + distribution
├── feeds/                           # NVD, CISA KEV, vendor feed syncing + normalization
├── scanner/                         # OSV scanner integration (osv-scalibr Go library)
├── license/                         # License generation + management + metering
├── telemetry/                       # Client telemetry collection (anonymized, opt-in)
├── msp/                             # MSP portal backend (multi-tenant management)
├── store/                           # Hub database layer
│   ├── queries/                     # sqlc .sql query files
│   ├── sqlcgen/                     # sqlc-generated Go code
│   ├── migrations/                  # goose .sql migration files
│   └── valkey/                      # Valkey cache
├── events/                          # Watermill event subscribers (feed updates, catalog sync)
└── workers/                         # River background jobs (feed refresh, license checks)
```

### Shared — `internal/shared/`

```
internal/shared/
├── domain/                          # Domain event types + audit schema
│   ├── events.go                    # DomainEvent struct, event type constants
│   └── audit.go                     # Audit trail helpers
├── models/                          # Shared domain types (Endpoint, Patch, CVE, Tenant, etc.)
├── tenant/                          # Multi-tenancy: context helpers, RLS wiring, tenant middleware
├── crypto/                          # mTLS cert management, RSA signing, encryption
├── config/                          # Koanf config loading (file + env + flag merge)
└── otel/                            # OpenTelemetry initialization, slog handler setup
```

**Import rules:**
- `server/`, `agent/`, `hub/` may import from `shared/`
- `shared/` must NOT import from `server/`, `agent/`, or `hub/`
- `server/`, `agent/`, `hub/` must NOT import from each other

---

## Protobuf

```
proto/
├── buf.yaml                         # Buf module config + lint rules
└── patchiq/v1/
    ├── agent.proto                   # Agent ↔ Server protocol (enrollment, heartbeat, commands)
    ├── hub.proto                     # Hub ↔ Patch Manager protocol (catalog sync, license)
    └── common.proto                  # Shared message types (Timestamp, Pagination, etc.)

gen/                                  # buf-generated Go code (committed)
└── patchiq/v1/
    ├── agent.pb.go
    ├── agent_grpc.pb.go
    ├── hub.pb.go
    ├── hub_grpc.pb.go
    └── common.pb.go
```

---

## API Specifications

```
api/                                  # OpenAPI specs (source of truth for REST APIs)
├── server.yaml                       # Patch Manager API spec → generates web/src/api/
└── hub.yaml                          # Hub Manager API spec → generates web-hub/src/api/
```

---

## Configuration

```
configs/                              # Default config templates (Koanf)
├── server.yaml                       # Patch Manager defaults
├── agent.yaml                        # Agent defaults
└── hub.yaml                          # Hub Manager defaults
```

These are default configs shipped with binaries. At runtime, Koanf merges: defaults → config file → environment variables → flags.

---

## Frontend — Shared Package

```
packages/
└── ui/                               # @patchiq/ui — shared across all three web apps
    ├── src/
    │   ├── components/               # Shared shadcn/ui components (Button, Dialog, DataTable, etc.)
    │   ├── lib/                      # Shared utilities (cn(), formatDate, etc.)
    │   └── styles/                   # Shared Tailwind theme + base CSS
    ├── package.json                  # Published as @patchiq/ui (workspace link)
    └── tsconfig.json
```

---

## Frontend — Patch Manager UI (`web/`)

```
web/
├── src/
│   ├── components/                   # App-specific components (not shared)
│   ├── pages/                        # Route-level pages
│   ├── flows/                        # @xyflow/react canvases (key differentiator)
│   │   ├── policy-workflow/          # Visual policy builder
│   │   ├── topology-map/            # Network topology view
│   │   └── dependency-graph/        # Patch dependency visualization
│   ├── hooks/                        # Custom React hooks
│   ├── api/                          # Generated API client (from api/server.yaml)
│   ├── store/                        # Zustand stores (client state)
│   ├── types/                        # TypeScript types
│   └── ai/                           # AI assistant chat panel (MCP integration)
├── package.json                      # depends on @patchiq/ui
├── vite.config.ts
├── tailwind.css                      # TW4 CSS-first config (imports @patchiq/ui theme)
└── playwright.config.ts              # E2E test config
```

---

## Frontend — Hub Manager UI (`web-hub/`)

```
web-hub/
├── src/
│   ├── components/                   # Hub-specific components
│   ├── pages/                        # MSP portal, catalog management, license dashboard
│   ├── hooks/
│   ├── api/                          # Generated API client (from api/hub.yaml)
│   ├── store/
│   └── types/
├── package.json                      # depends on @patchiq/ui
├── vite.config.ts
└── tailwind.css
```

---

## Frontend — Agent UI (`web-agent/`)

```
web-agent/
├── src/
│   ├── components/                   # Minimal agent-specific components
│   ├── pages/                        # ~5 pages: status, patches, history, logs, settings
│   └── types/
├── package.json                      # depends on @patchiq/ui
├── vite.config.ts
└── tailwind.css
```

The agent UI is intentionally lightweight — embedded in the agent binary, served via stdlib HTTP.

---

## Report Templates

```
templates/
└── reports/                          # Gotenberg HTML → PDF templates
    ├── compliance-hipaa.html
    ├── compliance-soc2.html
    ├── compliance-iso27001.html
    └── deployment-summary.html
```

---

## Deployment

```
deploy/
├── docker/
│   ├── Dockerfile.server             # Patch Manager (multi-stage, distroless)
│   ├── Dockerfile.agent              # Agent (multi-stage, scratch or distroless)
│   ├── Dockerfile.hub                # Hub Manager
│   ├── docker-compose.yml            # Production stack
│   └── docker-compose.dev.yml        # Dev stack (hot-reload, local DBs)
├── helm/
│   └── patchiq/                      # Helm chart for Kubernetes deployment
├── ova/                              # VM appliance build scripts (air-gapped customers)
└── scripts/
    ├── install-agent.sh              # Linux agent installer
    ├── install-agent.ps1             # Windows agent installer
    └── install-agent-macos.sh        # macOS agent installer
```

---

## GitHub

```
.github/
├── workflows/
│   ├── lint.yml                      # golangci-lint, prettier, eslint, tsc, sloglint, buf lint
│   ├── test-unit.yml                 # go test -race, Vitest
│   ├── test-integration.yml          # testcontainers (PostgreSQL, Valkey, MinIO)
│   ├── build.yml                     # Cross-compile agents, Docker images, frontend build
│   ├── test-e2e.yml                  # Playwright (on merge to main)
│   ├── test-load.yml                 # k6 load tests (weekly / infra changes)
│   └── release.yml                   # GoReleaser (on version tag)
├── PULL_REQUEST_TEMPLATE.md
└── ISSUE_TEMPLATE/
    ├── bug_report.md
    └── feature_request.md
```

---

## Tests

```
test/
├── integration/                      # testcontainers integration test suites
├── e2e/                              # Playwright E2E tests
├── load/                             # k6 load test scripts
└── fixtures/                         # Shared test data
```

Go unit tests live alongside the code they test (standard Go convention). Integration and E2E tests that span multiple packages live here.

---

## Tools

```
tools/
├── agent-simulator/                  # Go program mimicking agent behavior (for load testing)
└── scripts/                          # Build/generation helper scripts (buf generate wrapper, etc.)
```

---

## Claude Code

```
.claude/
├── settings.json                     # Shared permissions + hooks (committed)
└── agents/                           # Custom subagent definitions (if any)
```

---

## Code Mapping Summary

| Platform | Backend | Frontend | Database |
|----------|---------|----------|----------|
| Agent | `internal/agent/` | `web-agent/` | Local SQLite |
| Patch Manager | `internal/server/` | `web/` | `internal/server/store/` |
| Hub Manager | `internal/hub/` | `web-hub/` | `internal/hub/store/` |
| Shared | `internal/shared/` | `packages/ui/` | — |

---

## Key Conventions

1. **Every new table** must include `tenant_id` unless it's in the global list (tenants, patch_catalog, cve_feeds, agent_binaries). See [Architecture Foundations §1](../foundations/architecture-foundations.md).

2. **Every sqlc query file** is named after the resource it queries (e.g., `endpoints.sql`, `deployments.sql`). One file per resource.

3. **Every goose migration** follows sequential numbering with a descriptive name: `001_init_schema.sql`, `002_rls_policies.sql`.

4. **Import direction** is strictly enforced: `shared/ ← server/`, `shared/ ← agent/`, `shared/ ← hub/`. No lateral imports between platforms. Pre-commit hooks can enforce this.

5. **Generated code is committed** (sqlc output, buf output, OpenAPI clients). This avoids requiring codegen tools in every dev environment and makes code review include generated changes.

6. **Frontend workspace** uses pnpm workspaces. `@patchiq/ui` is linked locally — no publishing to npm.
