# Feature: High Availability & Disaster Recovery

> Status: Proposed | Phase: 3 | License: Enterprise/MSP

---

## Overview

Multi-tier HA/DR architecture supporting single-instance with backups, active-passive failover, and active-active zero-downtime configurations for the Patch Manager.

## Problem Statement

Patch management is infrastructure-critical. Downtime means missed patch windows, stale compliance data, and agents losing connectivity. Enterprise customers require HA guarantees and documented RPO/RTO targets.

---

## HA Architecture Tiers

| Tier | Model | RPO | RTO | License Required |
|------|-------|-----|-----|-----------------|
| **Standard** | Single instance + backups | 24h | 4h | Professional |
| **High Availability** | Active-Passive | 5min | 15min | Enterprise |
| **Maximum Availability** | Active-Active | ~0 | < 1min | Enterprise |

---

## Active-Passive HA

```
┌──────────────────┐        ┌──────────────────┐
│   Primary Site    │        │  Standby Site    │
│                   │        │                  │
│ ┌──────────────┐  │        │ ┌──────────────┐ │
│ │ Patch Manager│  │  ───→  │ │ Patch Manager│ │
│ │ (Active)     │  │ Async  │ │ (Standby)    │ │
│ └──────────────┘  │ Replic │ └──────────────┘ │
│                   │        │                  │
│ ┌──────────────┐  │        │ ┌──────────────┐ │
│ │ PostgreSQL   │──┼────────┼→│ PostgreSQL   │ │
│ │ (Primary)    │  │Streami │ │ (Replica)    │ │
│ └──────────────┘  │ng Repl │ └──────────────┘ │
│                   │        │                  │
│ ┌──────────────┐  │        │ ┌──────────────┐ │
│ │ Redis        │──┼────────┼→│ Redis        │ │
│ │ (Primary)    │  │Sentinel│ │ (Replica)    │ │
│ └──────────────┘  │        │ └──────────────┘ │
│                   │        │                  │
│ ┌──────────────┐  │        │ ┌──────────────┐ │
│ │ MinIO        │──┼────────┼→│ MinIO        │ │
│ │              │  │Bucket  │ │              │ │
│ └──────────────┘  │Repl.   │ └──────────────┘ │
└──────────────────┘        └──────────────────┘
```

**PostgreSQL HA**: Patroni + streaming replication (async for WAN)
**Redis HA**: Redis Sentinel for failover
**MinIO HA**: Bucket replication (active-passive mode)
**Failover**: Automated via health checks; agents reconnect to standby's address via DNS or load balancer

---

## Active-Active HA

For zero-downtime requirements:
- Two or more Patch Manager instances behind a load balancer
- PostgreSQL with Patroni cluster (3+ nodes with quorum)
- Redis Cluster (6+ nodes, 3 primaries + 3 replicas)
- MinIO active-active site replication
- Agents configured with multiple server addresses, automatic failover
- Conflict resolution: last-write-wins for most data; deployment orchestration uses distributed locking (Redis/etcd)

---

## Backup Strategy

| Component | Method | Frequency | Retention |
|-----------|--------|-----------|-----------|
| PostgreSQL | pg_dump + WAL archiving | Hourly (WAL), Daily (full) | 30 days |
| Redis | RDB snapshots + AOF | Hourly | 7 days |
| MinIO | Bucket versioning + cross-site replication | Continuous | 90 days |
| Configuration | Git-backed config export | On every change | Indefinite |
| Audit Logs | Separate archive table + S3 export | Daily | Per policy (1-7 years) |

---

## Integration Points

| Feature | Integration |
|---------|-------------|
| **Multi-Site** | HA complements hub-spoke and federated topologies |
| **Agent** | Agents handle server failover via multi-address config |
| **Observability** | Health checks feed into HA failover decisions |

---

## Code Mapping

| Area | Code Directory |
|------|---------------|
| Health checks | `internal/server/apm/` |
| DB migrations | `internal/server/store/migrations/` |
| Config management | `internal/common/config/` |
