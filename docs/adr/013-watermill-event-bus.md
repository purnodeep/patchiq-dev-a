# ADR-013: Watermill for Event-Driven Architecture

## Status

Accepted

## Context

PatchIQ's architecture foundations (see `docs/blueprint/foundations/architecture-foundations.md`) require event-driven communication for: agent status changes, patch deployment progress, policy triggers, audit logging, and compliance scan results. The challenge is that on-prem Patch Manager must minimize infrastructure requirements (customers don't want to manage NATS/Kafka clusters), while Hub Manager SaaS needs high-throughput messaging for thousands of connected agents.

## Decision

Use Watermill (v1.5.x) as the event bus abstraction layer with deployment-specific backends:

- **On-prem Patch Manager**: Watermill with PostgreSQL backend. Events stored in PostgreSQL tables. Zero additional infrastructure — customers already have PostgreSQL.
- **Hub Manager SaaS**: Watermill with NATS JetStream backend. High throughput for thousands of agents streaming status updates.
- **Agent**: Raw PostgreSQL LISTEN/NOTIFY via pgx for lightweight, real-time command delivery from Manager to Agent. No library overhead.

The key: same Watermill publisher/subscriber interface across components, different backends per deployment. Event handlers for deployment progress, policy triggers, and audit logging are written once and work on both PostgreSQL (on-prem) and NATS (SaaS).

## Consequences

- **Positive**: Single event handler codebase for on-prem and SaaS; PostgreSQL backend means no extra infra for on-prem; NATS JetStream handles SaaS scale; Watermill provides poison queues, retries, and exactly-once semantics; ~9.2k GitHub stars, actively maintained by Three Dots Labs
- **Negative**: Abstraction layer adds a dependency; backend differences may surface in edge cases (message ordering, delivery guarantees); team must understand both PostgreSQL and NATS backends; Watermill's PostgreSQL backend has lower throughput than NATS

## Alternatives Considered

- **PostgreSQL LISTEN/NOTIFY only**: Simplest — rejected because fire-and-forget (no persistence, no retry, no dead-letter queue); messages lost if subscriber offline
- **NATS JetStream everywhere**: Single backend — rejected because adds NATS as required on-prem infrastructure; customers may push back on deploying another service
- **Redis/Valkey Streams**: Good performance — rejected because adds Valkey as required event infrastructure separate from its caching role; Watermill with PostgreSQL is simpler for on-prem
- **Kafka**: Enterprise standard — rejected because massive operational overhead for on-prem deployments; overkill for PatchIQ's event volume
