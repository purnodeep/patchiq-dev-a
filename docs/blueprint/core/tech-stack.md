# PatchIQ — Tech Stack

> Complete technology stack for the PatchIQ platform. Each significant decision has a corresponding ADR in `docs/adr/`.

---

## Backend

| Category | Technology | Version | ADR | Notes |
|----------|-----------|---------|-----|-------|
| Language | Go | 1.23+ | — | All three platforms (Agent, Patch Manager, Hub Manager) |
| HTTP Router | chi (Manager/Hub), stdlib net/http (Agent) | chi v5.2.x | [ADR-017](../../adr/017-chi-pgx-sqlc-goose-koanf.md) | Zero-dependency, net/http-native. Agent uses stdlib's Go 1.22+ enhanced mux. |
| SQL Driver | pgx | v5.8.x | [ADR-017](../../adr/017-chi-pgx-sqlc-goose-koanf.md) | Fastest pure Go PostgreSQL driver. Native RLS support, LISTEN/NOTIFY, COPY. |
| SQL Codegen | sqlc | v1.30.x | [ADR-017](../../adr/017-chi-pgx-sqlc-goose-koanf.md) | Type-safe Go code from SQL queries. Generates pgx-native code. |
| Database | PostgreSQL (with RLS) | 16 | [ADR-001](../../adr/001-three-separate-platforms.md) | Row-Level Security for multi-tenancy. |
| Migrations | goose (+Atlas for drift detection) | goose v3.26.x | [ADR-017](../../adr/017-chi-pgx-sqlc-goose-koanf.md) | Supports Go-coded migrations. Embeddable as library for on-prem startup. |
| Cache / KV | Valkey | 9.0 | [ADR-011](../../adr/011-valkey-over-redis.md) | BSD-3 license (Linux Foundation fork of Redis). Drop-in compatible. |
| Event Bus | Watermill | v1.5.x | [ADR-013](../../adr/013-watermill-event-bus.md) | PostgreSQL backend (on-prem), NATS JetStream (SaaS). Same handler code, swappable backends. |
| Job Queue | River | v0.26.x | [ADR-014](../../adr/014-river-job-queue.md) | PostgreSQL-backed. Transactional job enqueuing (ACID with app transactions). |
| Search | PostgreSQL FTS (pg_trgm + tsvector) | built-in | — | Sufficient for ~50K patch catalog entries. No extra infrastructure. |
| Object Storage | MinIO | latest | — | S3-compatible. Works air-gapped. Shared by LGTM observability stack. |
| Agent Communication | gRPC + mTLS | latest | [ADR-001](../../adr/001-three-separate-platforms.md) | Bidirectional streaming, mutual TLS for zero-trust. |
| Agent Local Storage | SQLite | latest | — | Offline queue for inventory and patch results. |
| Configuration | Koanf | v2.x | [ADR-017](../../adr/017-chi-pgx-sqlc-goose-koanf.md) | Modular, preserves key casing. Lighter than Viper (313% smaller binary). |
| Notifications | Shoutrrr | latest | — | Go library. Unified interface for Slack, email, webhook, Teams. Upgrade to Novu if orchestration needed. |
| PDF Reports | Gotenberg | v8 | — | Containerized HTML-to-PDF via Chromium. For compliance reports. |

---

## IAM & Authentication

| Category | Technology | Version | ADR | Notes |
|----------|-----------|---------|-----|-------|
| Identity Platform | Zitadel | v4.11.x | [ADR-012](../../adr/012-zitadel-for-iam.md) | Go-native, single binary, Apache 2.0. Native multi-tenancy. Event-sourced audit trail. |
| Protocols | SAML, OIDC, LDAP/AD | via Zitadel | [ADR-012](../../adr/012-zitadel-for-iam.md) | Enterprise SSO and directory sync. |
| Go SDK | zitadel-go | latest | [ADR-012](../../adr/012-zitadel-for-iam.md) | Official Go SDK for admin API and token validation. |
| RBAC Split | Zitadel (coarse) + App (fine-grained) | — | [ADR-004](../../adr/004-custom-rbac-model.md) | Zitadel: users, orgs, coarse roles. App: Action+Resource+Scope permissions. |

---

## Frontend

| Category | Technology | Version | ADR | Notes |
|----------|-----------|---------|-----|-------|
| Framework | React + TypeScript (strict) | 19.2.4 | [ADR-016](../../adr/016-frontend-tooling-updates.md) | `use()`, `useActionState`, `useOptimistic` useful for SPA dashboards. |
| Build Tool | Vite | 7.3.1 | — | Vite 8 (Rolldown-based) available as beta; wait for stable. |
| CSS Framework | Tailwind CSS | 4.2.1 | [ADR-016](../../adr/016-frontend-tooling-updates.md) | Rust-based Oxide engine. CSS-first config. No `tailwind.config.js`. |
| UI Components | shadcn/ui (Radix + Tailwind) | latest (copy-paste) | — | Full ownership of component code. |
| State (client) | Zustand | 5.0.11 | — | Global app state: user, sidebar, filters, notification queue. |
| State (server) | TanStack Query | 5.90.21 | — | Caching, background refetch, optimistic updates, pagination. |
| Data Tables | TanStack Table | 8.21.3 | — | Headless: sorting, filtering, pagination, column pinning, virtualization. |
| Forms | react-hook-form + zod + @hookform/resolvers | 7.71.2 + 4.3.6 + 5.2.2 | — | Zod 4 (major upgrade from 3.x). Standard Schema support. |
| Charts | Recharts (+@nivo/heatmap for compliance) | 3.7.0 | [ADR-016](../../adr/016-frontend-tooling-updates.md) | React-native API. Faster to ship than visx. Nivo for specialized heatmaps. |
| Workflow Builder | @xyflow/react + ELK.js | 12.10.1 | [ADR-002](../../adr/002-react-flow-for-workflows.md) | Renamed from `reactflow`. Key differentiator — no competitor has this. |
| Code Editor | CodeMirror 6 | @codemirror/view 6.x | [ADR-016](../../adr/016-frontend-tooling-updates.md) | 300KB vs Monaco's 5MB. Sufficient for bash/PowerShell script editing. |
| Terminal Emulator | @xterm/xterm | 6.0.0 | — | Scoped package (old `xterm` deprecated). Live agent output streaming. |
| Animation | Tailwind CSS transitions | built-in | [ADR-016](../../adr/016-frontend-tooling-updates.md) | CSS transitions cover 90% of B2B dashboard needs. Add `motion` only if needed later. |
| Deployment Timeline | Recharts custom or SVAR React Gantt (MIT) | — | — | react-chrono is display-only. Recharts bar chart for read-only timelines. |

---

## AI

| Category | Technology | Version | ADR | Notes |
|----------|-----------|---------|-----|-------|
| Protocol | MCP (Model Context Protocol) | latest spec | [ADR-003](../../adr/003-mcp-for-ai-integration.md) | Standard protocol. Human-in-the-loop. Tool annotations for safety. |
| SDK | modelcontextprotocol/go-sdk (official) | pre-1.0 | [ADR-015](../../adr/015-go-mcp-sdk.md) | Official Go SDK. Eliminates TypeScript↔Go language boundary. Backed by Google + MCP org. |
| LLM | Claude API | latest | [ADR-003](../../adr/003-mcp-for-ai-integration.md) | Best reasoning model for complex operations. |

---

## Vulnerability Data

| Category | Technology | Version | ADR | Notes |
|----------|-----------|---------|-----|-------|
| CVE Database | OSV.dev + NVD (supplementary) | — | — | Google-backed. Aggregates NVD, GitHub Advisories, vendor feeds. |
| Scanner Library | osv-scanner v2 (Go) | v2.x | — | Importable as Go library via OSV-Scalibr. |
| Additional Sources | Trivy (container scanning), CISA KEV | — | — | Supplementary for container and known-exploited vulnerability coverage. |

---

## Observability

| Category | Technology | Version | ADR | Notes |
|----------|-----------|---------|-----|-------|
| Instrumentation | OpenTelemetry | latest | [ADR-006](../../adr/006-opentelemetry-and-slog.md) | Vendor-neutral. Instruments all three platforms from day 1. |
| Structured Logging | slog (Go stdlib) | stdlib | [ADR-006](../../adr/006-opentelemetry-and-slog.md) | JSON handler. Every log includes trace_id from context. |
| Metrics Backend | Grafana Mimir (Prometheus-compatible) | latest | [ADR-018](../../adr/018-grafana-lgtm-observability.md) | Self-hostable. Stores in S3/MinIO. Drop-in Prometheus replacement. |
| Tracing Backend | Grafana Tempo (Jaeger-compatible) | latest | [ADR-018](../../adr/018-grafana-lgtm-observability.md) | Self-hostable. Stores in S3/MinIO. Accepts OTel traces natively. |
| Log Aggregation | Grafana Loki | latest | [ADR-018](../../adr/018-grafana-lgtm-observability.md) | Self-hostable. Stores in S3/MinIO. Label-based log indexing. |
| Dashboards | Grafana | latest | [ADR-018](../../adr/018-grafana-lgtm-observability.md) | Unified UI for metrics, traces, and logs. |
| Deployment | Self-hosted (all AGPL-3.0) or Grafana Cloud | — | [ADR-018](../../adr/018-grafana-lgtm-observability.md) | Grafana Cloud free tier for early SaaS development. Migrate to self-hosted when scale/cost warrants. |

---

## HA / DR

| Category | Technology | Version | ADR | Notes |
|----------|-----------|---------|-----|-------|
| PostgreSQL HA | Patroni | latest | [ADR-008](../../adr/008-patroni-redis-sentinel-ha.md) | Streaming replication with automatic failover. |
| Valkey HA | Valkey Sentinel / Valkey Cluster | 9.0 | [ADR-008](../../adr/008-patroni-redis-sentinel-ha.md) | Sentinel for active-passive, Cluster for active-active. |
| Object Storage HA | MinIO bucket replication | latest | [ADR-008](../../adr/008-patroni-redis-sentinel-ha.md) | Cross-site replication for patch binaries and support bundles. |

---

## CI/CD & Tooling

| Category | Technology | Version | ADR | Notes |
|----------|-----------|---------|-----|-------|
| CI | GitHub Actions | — | [ADR-009](../../adr/009-github-actions-goreleaser-helm.md) | Lint → Build → Test → Release pipeline. |
| Release | GoReleaser | latest | [ADR-009](../../adr/009-github-actions-goreleaser-helm.md) | Cross-compiles 6 agent binaries + Docker images. Checksums + signing. |
| K8s Deployment | Helm | latest | [ADR-009](../../adr/009-github-actions-goreleaser-helm.md) | Helm chart for Kubernetes customers. |
| Protobuf | buf | latest | — | Protobuf linting, code generation, breaking change detection. |
| Go Hot-Reload | air | latest | — | Development-only. Rebuilds on file change. |
| Anti-Slop | CLAUDE.md + pre-commit hooks + ADRs + PR template | — | [ADR-010](../../adr/010-anti-slop-development-guardrails.md) | Multi-layer guardrails for AI-assisted development. |

---

## Key Technical Decisions Summary

| # | Decision | ADR |
|---|----------|-----|
| 1 | Three separate platforms (Agent, PM, Hub) | [ADR-001](../../adr/001-three-separate-platforms.md) |
| 2 | @xyflow/react for visual workflow builder | [ADR-002](../../adr/002-react-flow-for-workflows.md) |
| 3 | MCP protocol + Claude API for AI | [ADR-003](../../adr/003-mcp-for-ai-integration.md) |
| 4 | Custom RBAC (fine-grained) + Zitadel (coarse) | [ADR-004](../../adr/004-custom-rbac-model.md) |
| 5 | RSA-signed JSON license files | [ADR-005](../../adr/005-rsa-signed-license-files.md) |
| 6 | OpenTelemetry + slog from day 1 | [ADR-006](../../adr/006-opentelemetry-and-slog.md) |
| 7 | Hub-spoke multi-site topology | [ADR-007](../../adr/007-hub-spoke-multi-site.md) |
| 8 | Patroni + Valkey Sentinel + MinIO for HA | [ADR-008](../../adr/008-patroni-redis-sentinel-ha.md) |
| 9 | GitHub Actions + GoReleaser + Helm | [ADR-009](../../adr/009-github-actions-goreleaser-helm.md) |
| 10 | Anti-slop development guardrails | [ADR-010](../../adr/010-anti-slop-development-guardrails.md) |
| 11 | Valkey over Redis (BSD-3 licensing) | [ADR-011](../../adr/011-valkey-over-redis.md) |
| 12 | Zitadel for IAM | [ADR-012](../../adr/012-zitadel-for-iam.md) |
| 13 | Watermill for event-driven architecture | [ADR-013](../../adr/013-watermill-event-bus.md) |
| 14 | River for PostgreSQL job queue | [ADR-014](../../adr/014-river-job-queue.md) |
| 15 | Go MCP SDK (official) over TypeScript | [ADR-015](../../adr/015-go-mcp-sdk.md) |
| 16 | Frontend tooling updates (React 19, TW4, Recharts, CM6) | [ADR-016](../../adr/016-frontend-tooling-updates.md) |
| 17 | Backend tooling (chi, pgx+sqlc, goose, Koanf) | [ADR-017](../../adr/017-chi-pgx-sqlc-goose-koanf.md) |
| 18 | Grafana LGTM stack for observability backends | [ADR-018](../../adr/018-grafana-lgtm-observability.md) |
| 19 | Project structure and layout conventions | [ADR-019](../../adr/019-project-structure.md) |
