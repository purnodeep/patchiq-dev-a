# ADR-019: Project Structure and Layout Conventions

## Status

Accepted

## Context

The original project structure was extracted from BLUEPRINT-V1 and predates 8 technology decisions (ADR-011 through ADR-018). It contains stale references (Redis instead of Valkey, golang-migrate instead of goose), misattributes responsibilities (auth/ described as handling SSO/MFA that Zitadel now owns), and is missing infrastructure for key architectural patterns (Watermill events, River jobs, multi-tenancy, domain events).

The structure needs to be revised to align with the finalized tech stack and the 7 foundational architecture patterns documented in `architecture-foundations.md`.

Key decisions that needed resolution:

1. **sqlc directory layout** — where query files, generated code, and migrations co-exist
2. **Shared package naming** — `internal/common/` vs alternatives
3. **Async processing layout** — where Watermill event handlers and River job workers live
4. **Frontend code sharing** — shared UI package vs fully independent apps

## Decision

### 1. sqlc: Co-located with store

Query `.sql` files live in `internal/{platform}/store/queries/`. sqlc generates Go code into `internal/{platform}/store/sqlcgen/`. Goose migration files live in `internal/{platform}/store/migrations/`. The `sqlc.yaml` config at the repo root references all platform paths.

**Rationale:** Everything database-related is in one place per platform. Developers working on a query can see the schema (migrations), the query (queries/), and the generated code (sqlcgen/) in the same tree. The sqlc config at root provides a single point of truth for all codegen settings.

### 2. Shared package: `internal/shared/`

Renamed from `internal/common/` to `internal/shared/`. Sub-packages: `domain/` (event types, audit), `models/` (domain types), `tenant/` (multi-tenancy), `crypto/` (mTLS, signing), `config/` (Koanf loading), `otel/` (OpenTelemetry setup).

**Rationale:** "shared" explicitly communicates cross-platform intent. The sub-packages are disciplined — each has a clear responsibility, not a dumping ground.

### 3. Async processing: Dedicated directories

Each platform that uses async processing (server, hub) gets dedicated `events/` and `workers/` directories for Watermill subscribers and River jobs respectively. The agent has neither — it's intentionally lightweight.

**Rationale:** Separating async handlers from synchronous API handlers makes the codebase navigable. When debugging a background job failure, you go to `workers/`. When tracing an event, you go to `events/`. Co-locating with domain packages was considered but rejected because event handlers often span multiple domains and benefit from centralized router configuration.

### 4. Frontend: Shared `packages/ui/` via pnpm workspaces

A `packages/ui/` directory contains shared shadcn/ui components, Tailwind theme config, and common utilities. The three web apps (`web/`, `web-hub/`, `web-agent/`) depend on `@patchiq/ui` as a workspace link. pnpm workspaces manage the monorepo.

**Rationale:** The Patch Manager and Hub Manager share significant UI surface area (data tables, form patterns, navigation). Without a shared package, components would drift between apps. shadcn/ui's copy-paste model means shared components are still fully owned — the shared package is just where the canonical copies live.

### 5. Additional structural decisions

- **`internal/server/iam/`** — New directory for Zitadel integration (SSO, OIDC, SAML, LDAP, user sync). Separated from `auth/` which handles app-level fine-grained RBAC. This reflects the split in ADR-004 and ADR-012.
- **`internal/server/compliance/`** — Compliance engine gets its own directory (not buried in engine/).
- **`api/`** at repo root — OpenAPI specs for server and hub REST APIs. Frontend clients are generated from these.
- **`configs/`** at repo root — Default Koanf config files per platform.
- **`gen/`** at repo root — buf-generated protobuf Go code.
- **`templates/reports/`** — Gotenberg HTML templates for PDF report generation.
- **API versioning** — `api/v1/` subdirectory in handler packages from day 1.
- **`proto/buf.yaml`** — buf module config inside proto/, `buf.gen.yaml` at root.

## Consequences

- **Positive**: Structure reflects actual tech stack decisions; clear separation between sync (api/) and async (events/, workers/) code paths; shared UI prevents component drift; import rules (shared ← platforms) are enforceable via linting; every infrastructure concern (tenancy, events, config) has a home from day 1
- **Negative**: More directories than the original structure (higher initial scaffolding effort); pnpm workspace adds frontend build complexity; sqlcgen/ directories mean generated code shows up in diffs; separate iam/ and auth/ directories require developers to understand the split

## Alternatives Considered

- **sqlc queries alongside migrations**: Rejected because mixing query intent (reads/writes) with schema evolution (migrations) in the same directory is confusing
- **Keep `internal/common/`**: Rejected because "common" is a Go naming smell and "shared" better communicates intent
- **Inline events/workers with domain packages**: Rejected because event handlers span domains and centralized Watermill router setup is cleaner in one place
- **Fully independent frontend apps**: Rejected because component drift between Patch Manager and Hub Manager UIs is inevitable without a shared package
- **Turborepo for frontend monorepo**: Considered but deferred — pnpm workspaces alone are sufficient until build times become a problem
