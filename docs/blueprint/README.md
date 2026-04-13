# PatchIQ Blueprint

> The complete product specification for PatchIQ — an enterprise patch management platform composed of three interconnected products: Agent, Patch Manager, and Hub Manager.

---

## How This Directory Works

- **`core/`** — Platform identity: what PatchIQ is, how the platforms connect, UI vision, licensing
- **`features/`** — One document per feature, each following `_template.md`. This is where all product behavior is specified.
- **`foundations/`** — Architectural decisions that must be baked in from day 1 (multi-tenancy, events, config hierarchy, etc.)

Related directories:
- **`../adr/`** — Architecture Decision Records (numbered, append-only)
- **`../_revision/`** — Documents extracted from the original blueprint that need revision (dev process, roadmap)
- **`../_archive/`** — Superseded documents kept for reference

---

## Core Documents

| Document | Description |
|----------|-------------|
| [Platform Overview](core/platform-overview.md) | The three platforms (Agent, Patch Manager, Hub Manager), their roles, and how they connect |
| [Tech Stack](core/tech-stack.md) | Complete technology stack with versions, rationale, and ADR cross-references |
| [UI Specification](core/ui-specification.md) | Visual workflow builder, advanced UI elements, frontend tech stack |
| [License Tiers](core/license-tiers.md) | License architecture, tier definitions, feature gating, pricing |
| [Project Structure](core/project-structure.md) | Canonical monorepo directory layout, code mapping, conventions |

---

## Feature Registry

Each feature has its own document in `features/`. Use [`_template.md`](features/_template.md) when adding a new feature.

| Feature | Doc | Code (Backend) | Code (Frontend) | Milestone | Status |
|---------|-----|-----------------|-----------------|-----------|--------|
| AI Assistant | [ai-assistant.md](features/ai-assistant.md) | `internal/server/mcp/` | `web/src/ai/` | M3 | Proposed |
| AI Patch Pipeline | [ai-patch-pipeline.md](features/ai-patch-pipeline.md) | `internal/hub/pipeline/` | — | M3 (structured), M4 (LLM+sandbox) | Proposed |
| Baseline Profiles | [baseline-profiles.md](features/baseline-profiles.md) | `internal/server/baseline/` | `web/src/pages/baselines/` | M3 | Proposed |
| Compliance Engine | [compliance-engine.md](features/compliance-engine.md) | `internal/server/compliance/` | `web/src/pages/compliance/` | M2 (basic), M3 (full) | Proposed |
| Custom RBAC | [rbac.md](features/rbac.md) | `internal/server/auth/` | `web/src/pages/admin/roles/` | M2 | Proposed |
| HA/DR | [ha-dr.md](features/ha-dr.md) | `internal/server/apm/` | — | M4 | Proposed |
| Multi-Site | [multi-site.md](features/multi-site.md) | `internal/server/distribution/` | — | M4 | Proposed |
| Observability | [observability.md](features/observability.md) | `internal/shared/otel/`, `internal/server/apm/` | — | M0 (foundation) | Proposed |
| Support System | [support-system.md](features/support-system.md) | `internal/server/support/` | — | M4 | Proposed |

---

## Foundations

| Document | Description |
|----------|-------------|
| [Architecture Foundations](foundations/architecture-foundations.md) | 7 architectural decisions that must be baked in from Phase 1: multi-tenancy, config hierarchy, event-driven architecture, plugin system, API versioning, agent command protocol, idempotent operations |

---

## Adding a New Feature

1. Copy `features/_template.md` to `features/your-feature-name.md`
2. Fill in all sections, especially **Code Mapping**
3. Add a row to the **Feature Registry** table above
4. Update CLAUDE.md with the new code directory mapping
5. If the feature involves a significant technical decision, create an ADR in `../adr/`

## Updating an Existing Feature

- When changing feature behavior in code, update the corresponding feature doc **in the same PR**
- If the change is significant enough to warrant a new ADR, create one
- Run `/revise-claude-md` periodically to keep CLAUDE.md mappings current

---

## Architecture Decision Records

ADRs live in [`../adr/`](../adr/) and are numbered sequentially. See the [ADR template](../adr/template.md).

| ADR | Decision |
|-----|----------|
| [001](../adr/001-three-separate-platforms.md) | Three separate platforms (Agent, PM, Hub) |
| [002](../adr/002-react-flow-for-workflows.md) | @xyflow/react for visual workflow builder |
| [003](../adr/003-mcp-for-ai-integration.md) | MCP protocol for AI integration |
| [004](../adr/004-custom-rbac-model.md) | Custom RBAC (fine-grained) + Zitadel (coarse) |
| [005](../adr/005-rsa-signed-license-files.md) | RSA-signed JSON license files |
| [006](../adr/006-opentelemetry-and-slog.md) | OpenTelemetry + slog from day 1 |
| [007](../adr/007-hub-spoke-multi-site.md) | Hub-spoke topology with distribution servers |
| [008](../adr/008-patroni-redis-sentinel-ha.md) | Patroni + Valkey Sentinel + MinIO for HA |
| [009](../adr/009-github-actions-goreleaser-helm.md) | GitHub Actions + GoReleaser + Helm |
| [010](../adr/010-anti-slop-development-guardrails.md) | Anti-slop development guardrails |
| [011](../adr/011-valkey-over-redis.md) | Valkey over Redis (BSD-3 licensing) |
| [012](../adr/012-zitadel-for-iam.md) | Zitadel for IAM |
| [013](../adr/013-watermill-event-bus.md) | Watermill for event-driven architecture |
| [014](../adr/014-river-job-queue.md) | River for PostgreSQL job queue |
| [015](../adr/015-go-mcp-sdk.md) | Go MCP SDK (official) over TypeScript |
| [016](../adr/016-frontend-tooling-updates.md) | Frontend tooling (React 19, TW4, Recharts, CM6) |
| [017](../adr/017-chi-pgx-sqlc-goose-koanf.md) | Backend tooling (chi, pgx+sqlc, goose, Koanf) |
| [018](../adr/018-grafana-lgtm-observability.md) | Grafana LGTM stack for observability |
| [019](../adr/019-project-structure.md) | Project structure and layout conventions |
