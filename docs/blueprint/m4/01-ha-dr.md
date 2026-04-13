# HA/DR

**Status**: Planned
**Milestone**: M4
**Dependencies**: M1 Core Loop, M2 Usable Product, PostgreSQL 16, Valkey, MinIO

---

## Vision

Provide enterprise-grade high availability and disaster recovery so that PatchIQ continues operating through infrastructure failures with defined RPO/RTO guarantees.

## Deliverables

### Active-Passive (RPO 5min / RTO 15min)
- [ ] Patroni setup for PostgreSQL streaming replication (primary + 1 replica)
- [ ] Valkey Sentinel for automatic failover (1 primary + 2 sentinels)
- [ ] MinIO replication across two nodes
- [ ] Health monitoring dashboard (replication lag, sentinel status, MinIO sync)
- [ ] Automated backup schedule with retention policy (daily snapshots, 30-day retention)
- [ ] Restore procedure documented and tested via runbook

### Active-Active (RPO ~0 / RTO <1min)
- [ ] Patroni cluster with 3+ nodes and synchronous replication
- [ ] Valkey Cluster (6+ nodes, 3 primary + 3 replica shards)
- [ ] MinIO active-active with bidirectional replication and conflict resolution
- [ ] Connection pooling (PgBouncer) aware of topology changes
- [ ] Health monitoring dashboard extended for cluster mode

### Operations
- [ ] DR runbook: step-by-step failover and recovery procedures
- [ ] Failover testing suite: inject failures and verify automatic recovery
- [ ] Alerting integration: notify on replication lag > threshold, sentinel elections
- [ ] Backup verification job: periodic restore-and-verify cycle
- [ ] RTO/RPO metrics exposed in observability stack (Grafana dashboard)

## License Gating

- Active-Passive HA: ENTERPRISE
- Active-Active HA: ENTERPRISE
- Automated backup/DR runbook: ENTERPRISE
