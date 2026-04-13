# ADR-023: Domain Events with ULID and Append-Only Audit Trail

## Status

Accepted

## Context

PatchIQ requires that every write operation emits a domain event (see `CLAUDE.md` — "Every write operation MUST emit a domain event"). These events serve two purposes: (1) driving reactive workflows (deployment state transitions, policy evaluation, vulnerability scanning) via Watermill pub/sub, and (2) maintaining an immutable audit trail for compliance. The audit trail must be tamper-resistant, efficiently queryable by time range, and support high insert throughput without contention.

## Decision

### ULID for Event IDs

Use ULIDs (Universally Unique Lexicographically Sortable Identifiers) as event IDs (`internal/shared/domain/events.go:42-44`). Generated via `oklog/ulid/v2` with `crypto/rand` as entropy source:

```go
func NewEventID() string {
    return ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String()
}
```

ULIDs encode a 48-bit millisecond timestamp in the first 10 characters, followed by 80 bits of randomness. This makes them time-ordered (sortable by creation time without a separate column), globally unique (no coordination needed between server instances), and string-safe (26 Crockford Base32 characters, no special characters).

### DomainEvent Envelope

The `DomainEvent` struct (`internal/shared/domain/events.go:27-39`) is the canonical event envelope, mapping 1:1 to the `audit_events` table:

- `ID` (ULID), `Type` (format: `resource.action`, e.g., `endpoint.enrolled`), `TenantID`
- `ActorID`, `ActorType` (one of `user`, `system`, `ai_assistant`)
- `Resource`, `ResourceID`, `Action` — the entity affected
- `Payload` (`any`) — event-specific data, serialized as JSON
- `Metadata` (`EventMeta`) — request-scoped context: `trace_id`, `request_id`, `ip_address`, `user_agent`
- `Timestamp` — UTC, set at creation time

Convenience constructors: `NewAuditEvent()` for user-attributed events, `NewSystemEvent()` for system-attributed events (`internal/shared/domain/audit.go`).

### Event Type Registry

All event types are declared as constants in `internal/server/events/topics.go` using `resource.action` format (e.g., `endpoint.created`, `deployment.started`, `cve.discovered`). The `AllTopics()` function returns the complete registry. `WatermillEventBus.Emit()` rejects events with unregistered types to prevent silent audit gaps (`internal/server/events/publisher.go:51-53`).

### EventBus Interface

The `domain.EventBus` interface (`internal/shared/domain/bus.go`) defines three operations: `Emit`, `Subscribe`, `Close`. Subscribe supports wildcard patterns: `"*"` (all events), `"deployment.*"` (prefix match), `"deployment.created"` (exact match). The `WatermillEventBus` implementation resolves patterns against the topic registry via `MatchingTopics()`.

### Append-Only Audit Table

The `audit_events` table (`internal/server/store/migrations/001_init_schema.sql:193-206`) is partitioned by `RANGE (timestamp)` with monthly partitions. The `patchiq_app` role has `INSERT` and `SELECT` only — `UPDATE` and `DELETE` are explicitly revoked on the parent table and all 12 monthly partitions for 2026 (`002_rls_policies.sql:20-32`). A default partition catches timestamps outside defined ranges.

Indexes support the primary query patterns: `(tenant_id, timestamp DESC)` for time-range queries, `(tenant_id, resource, resource_id)` for entity history, `(tenant_id, actor_id)` for user activity, `(tenant_id, type)` for event type filtering.

### Watermill Integration

`WatermillEventBus` (`internal/server/events/publisher.go`) publishes events to Watermill topics matching the event type. Events are JSON-serialized with `tenant_id` and `event_type` in Watermill message metadata. The Watermill SQL backend (`internal/server/events/watermill.go`) uses PostgreSQL with `DefaultPostgreSQLSchema` and auto-initialized schema.

## Consequences

- **Positive**: ULIDs provide time-ordering without composite indexes; append-only enforcement at the database role level prevents tampering even by application bugs; monthly partitioning enables efficient time-range queries and easy data retention management; topic registry prevents unregistered events from silently bypassing audit; wildcard subscription simplifies audit handler (subscribe to `"*"`)
- **Negative**: ULID string representation (26 chars) uses more storage than binary UUID (16 bytes); monthly partitions must be created proactively (currently hardcoded for 2026 — requires future migration or pg_partman adoption); `AllTopics()` registry must be manually maintained when adding new event types; `MustNew` panics on entropy exhaustion (mitigated by `crypto/rand`)

## Alternatives Considered

- **UUIDv7**: Time-ordered UUID — rejected because less human-readable than ULID, and the Go ecosystem has stronger ULID library support (`oklog/ulid` vs draft UUIDv7 implementations at time of decision)
- **Snowflake IDs**: Twitter-style — rejected because requires machine ID coordination, adding operational complexity for distributed deployments
- **Auto-increment integer IDs**: Simplest — rejected because not globally unique across server instances, no embedded timestamp, reveals sequence information
- **Mutable audit table**: Simpler permissions — rejected because compliance requirements demand tamper-evident audit trails; UPDATE/DELETE on audit data is a security risk
- **Separate audit database**: Strongest isolation — rejected as premature; partitioned table with role-level restrictions provides sufficient protection for current requirements
