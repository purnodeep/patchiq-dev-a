# ADR-021: Agent Offline-First Architecture (SQLite + Outbox/Inbox)

## Status

Accepted

## Context

PatchIQ agents run on endpoints (laptops, servers, VMs) that frequently lose connectivity to the Patch Manager server — network outages, VPN disconnects, laptop lids closed. The agent must continue collecting inventory data, executing commands, and reporting results regardless of connectivity. When the connection resumes, all buffered data must reach the server without loss or duplication.

The agent binary must remain minimal: no PostgreSQL, no Watermill, no heavy dependencies (see `CLAUDE.md` — "Agent is minimal: no Watermill, no River, no PostgreSQL. SQLite + gRPC only"). The local data store must survive unclean process termination (kill -9, power loss) and work on every OS PatchIQ supports.

## Decision

### SQLite as the Agent Database

Use SQLite (via `modernc.org/sqlite`, a pure-Go CGo-free port) as the agent's local database (`internal/agent/comms/db.go`). Configuration:

- **WAL mode**: `_pragma=journal_mode(wal)` — concurrent reads during writes, crash-safe.
- **Busy timeout**: `_pragma=busy_timeout(5000)` — 5-second wait on lock contention instead of immediate `SQLITE_BUSY`.
- **Single connection**: `db.SetMaxOpenConns(1)` — SQLite handles one writer at a time; a single connection avoids lock contention entirely.

The schema (`internal/agent/comms/schema.sql`) defines four tables: `outbox`, `inbox`, `local_inventory`, and `agent_state`.

### Outbox Pattern (Store-and-Forward)

The outbox table (`internal/agent/comms/outbox.go`) stores messages destined for the server. Every agent-generated message (inventory reports, command results, events) is written to the outbox first, then drained to the server by `SyncRunner`.

- Messages are inserted with status `pending` and a `created_at` timestamp.
- `SyncRunner` (`internal/agent/comms/sync.go`) drains up to **100 items per batch** (`BatchSize` default), ordered oldest-first.
- Sync runs immediately on connect (to drain items queued while offline), then every `SyncInterval` (default 30 seconds). Extra syncs can be triggered via `SyncRunner.Trigger()`, which coalesces multiple trigger requests.
- Each item is sent over a gRPC `SyncOutbox` bidirectional stream. The server responds with an `OutboxAck` per message.
- On acceptance (`RejectionCode == UNSPECIFIED`): mark `sent`.
- On transient rejection (`SERVER_OVERLOADED`): increment `attempts`, stop the batch, retry next tick.
- On permanent rejection (all other codes): mark `failed`.

### Dead-Lettering

Items that exceed `MaxAttempts` (default 5) are marked `failed` without being sent, logged as dead-lettered (`internal/agent/comms/sync.go:192-198`). This prevents a single poison message from blocking the entire outbox queue.

### Inbox Pattern (Command Reception)

The inbox table (`internal/agent/comms/inbox.go`) stores commands received from the server. Insertion is idempotent (`INSERT OR IGNORE`) on the server-assigned command ID. Pending commands are processed by priority (highest first), then by receive time.

### Exponential Backoff for Reconnection

`ReconnectConfig` (`internal/agent/comms/client.go`) controls retry behavior for gRPC connection, enrollment, and heartbeat failures:

- **Initial delay**: 1 second
- **Max delay**: 5 minutes
- **Multiplier**: 2.0
- **Jitter factor**: 0.2 (+-20% randomization to avoid thundering herd)

### Eventual Consistency Guarantees

The agent guarantees at-least-once delivery: messages persist in SQLite until the server acknowledges them. Duplicate delivery is possible (crash after server ack, before local `MarkSent`), and the server must handle idempotency. The outbox is FIFO-ordered by `created_at`, preserving causal ordering within a single agent.

## Consequences

- **Positive**: Agents operate fully offline; zero data loss on connectivity interruption; SQLite is zero-config, embedded, and battle-tested; single-connection model eliminates concurrency bugs; WAL mode provides crash safety; dead-lettering prevents queue stalls; pure-Go SQLite avoids CGo cross-compilation issues
- **Negative**: Single-connection limits write throughput (acceptable for agent workload); at-least-once semantics require server-side idempotency; dead-lettered messages are silently dropped after max attempts (operator must monitor logs); batch size of 100 may need tuning for high-volume agents

## Alternatives Considered

- **BoltDB/bbolt**: Key-value store — rejected because it lacks relational queries needed for inbox priority ordering and outbox status filtering; no SQL interface for debugging
- **Embedded PostgreSQL**: Full RDBMS — rejected because massive binary size increase, complex startup, overkill for agent's simple schema
- **File-based queue (JSON files)**: Simplest — rejected because no atomic writes, no crash recovery, no indexing for status-based queries
- **In-memory only**: Fastest — rejected because all buffered data lost on agent restart or crash; fundamentally incompatible with offline-first requirement
