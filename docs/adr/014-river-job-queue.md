# ADR-014: River for PostgreSQL-Backed Job Queue

## Status

Accepted

## Context

PatchIQ needs reliable job scheduling for: patch deployments at maintenance windows, compliance scans (daily/weekly/monthly), CVE feed sync jobs, report generation, and agent self-update rollouts. Jobs must be durable (survive server restarts), support scheduling (cron-like), and provide visibility (status, retries, dead-letter queue).

The critical requirement: when a patch deployment is created in the database, the corresponding job must be enqueued atomically in the same transaction. Without this, a crash between "insert deployment" and "enqueue job" creates orphaned records.

## Decision

Use River (v0.26.x) as the primary job queue, backed by PostgreSQL. River supports transactional job enqueuing — jobs are inserted in the same database transaction as application data, providing ACID guarantees.

For high-throughput fire-and-forget tasks in Hub Manager SaaS (telemetry ingestion, metric aggregation), add Asynq backed by Valkey if PostgreSQL-based queuing becomes a bottleneck.

## Consequences

- **Positive**: Transactional enqueuing eliminates orphaned jobs; no additional infrastructure (uses existing PostgreSQL); built-in periodic/cron scheduling; web UI (riverui) for monitoring; Go-native; supports job priorities, retries, and dead-letter queue
- **Negative**: PostgreSQL-based queuing has lower throughput than dedicated message brokers; River is pre-1.0 (MPL-2.0 license); monitoring requires the riverui add-on

## Alternatives Considered

- **Asynq**: Redis/Valkey-backed, 12.7k stars — kept as secondary option for high-throughput Hub Manager tasks; not primary because lacks transactional enqueuing with application database
- **Temporal**: Full workflow engine — rejected because massive operational complexity (requires its own server cluster); overkill for scheduling jobs; will evaluate if multi-step durable workflows become a requirement
- **Custom cron + goroutines**: Simplest — rejected because no persistence, no retry, no dead-letter queue, no visibility
