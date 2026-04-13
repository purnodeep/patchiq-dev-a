# ADR-017: Backend Tooling (chi, pgx+sqlc, goose, Koanf)

## Status

Accepted

## Context

The original blueprint specified Go as the backend language but did not specify the HTTP router, SQL library, migration tool, or configuration library. These foundational choices affect every part of the codebase and should be decided before scaffolding.

## Decision

### HTTP Router: chi v5.2.x

Use `go-chi/chi` for Patch Manager and Hub Manager REST APIs. Use raw `net/http` with Go 1.22+ enhanced mux for the Agent (minimal binary size).

**Why chi**: Zero external dependencies, built directly on `net/http`, uses standard `http.Handler`. Handlers are portable to stdlib. In production at Cloudflare, Heroku, 99Designs.

### SQL: pgx v5.8.x + sqlc v1.30.x

Use `pgx` as the PostgreSQL driver and `sqlc` for type-safe Go code generation from SQL queries.

**Why pgx**: Fastest pure Go PostgreSQL driver. Exposes PostgreSQL-specific features critical for PatchIQ: `LISTEN/NOTIFY` (event-driven agent status), `COPY` (bulk patch catalog imports), native RLS support (`SET app.tenant_id` per connection).

**Why sqlc**: Write SQL, get type-safe Go functions. Generates pgx-native code. Ideal for complex multi-tenant queries where an ORM would fight against RLS and custom SQL.

### Migrations: goose v3.26.x (+Atlas for drift detection)

Use `goose` for database migrations with `Atlas` as a CI complement for schema drift detection.

**Why goose**: Supports migrations written in Go code (not just SQL) — useful for data migrations. Embeddable as a Go library — runs at application startup, critical for on-prem Patch Manager where customers deploy without CI/CD. v3.26.0 adds `slog.Logger` support.

**Why Atlas as complement**: Detects when someone manually alters a production database (common in enterprise on-prem). Plans corrective migrations.

### Configuration: Koanf v2.x

Use `koanf` for application configuration.

**Why Koanf over Viper**: 313% smaller binary impact (important for lightweight agent). Preserves JSON/YAML/TOML key casing (Viper forces lowercase, breaking specs). Modular dependencies — install only what you need. Supports all the same sources: files, env vars, flags, remote config.

## Consequences

- **Positive**: All choices are stdlib-compatible or minimal-dependency; pgx+sqlc together provide type safety without ORM overhead; goose embeddability enables clean on-prem deployments; Koanf reduces agent binary size
- **Negative**: sqlc requires SQL-first development (no ORM migrations); team must maintain SQL queries alongside Go code; chi lacks some convenience features of Gin (but gains stdlib compatibility); Koanf is less well-known than Viper

## Alternatives Considered

- **Gin** (HTTP): Custom context, vendor lock-in — rejected for stdlib incompatibility
- **GORM** (SQL): Full ORM — rejected for 2-3x performance overhead, fights against RLS and complex SQL
- **sqlx** (SQL): Thin wrapper — rejected because sqlc provides stronger type safety via code generation
- **golang-migrate** (migrations): File-based — rejected because it lacks Go-coded migrations and library embedding
- **Viper** (config): De facto standard — rejected for binary bloat and forced lowercase keys
