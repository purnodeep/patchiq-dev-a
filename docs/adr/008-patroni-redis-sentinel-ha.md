# ADR-008: Patroni + Valkey Sentinel + MinIO Replication for HA

## Status

Accepted (Updated — Redis replaced with Valkey per ADR-011)

## Context

Enterprise customers require HA guarantees with documented RPO/RTO targets. We need HA for PostgreSQL, Valkey (cache/KV), and object storage.

## Decision

Use Patroni for PostgreSQL HA (streaming replication), Valkey Sentinel for Valkey failover, and MinIO bucket replication for object storage.

## Consequences

- **Positive**: Industry-standard components; no exotic dependencies; well-documented; battle-tested at scale; multiple HA tiers (standard → active-passive → active-active); Valkey Sentinel is API-compatible with Redis Sentinel
- **Negative**: Patroni requires etcd/ZooKeeper for consensus; Valkey Sentinel has known split-brain edge cases (same as Redis Sentinel); active-active adds distributed locking complexity

## Alternatives Considered

- **CockroachDB**: Distributed SQL, built-in HA — rejected because PostgreSQL ecosystem is richer; team expertise; PG RLS for multi-tenancy
- **Valkey Cluster** (instead of Sentinel): Better for active-active — accepted for active-active tier, Sentinel for active-passive
- **S3-compatible cloud storage**: Use AWS S3 — rejected because customers may be on-prem/air-gapped
