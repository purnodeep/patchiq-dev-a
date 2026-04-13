# Kubernetes + Air-Gapped

**Status**: Planned
**Milestone**: M4
**Dependencies**: M1 Docker Compose deployment, M2 stable APIs, HA/DR (01-ha-dr.md) for HA mode

---

## Vision

Make PatchIQ deployable on Kubernetes for cloud-native customers and as a self-contained OVA appliance for air-gapped environments, with load-tested scalability at 5,000+ agents.

## Deliverables

### Helm Chart
- [ ] Single Helm chart deploying all 3 platforms (server, hub, agent daemonset optional)
- [ ] Values schema: replicas, resource requests/limits, ingress, TLS, storage class, external DB config
- [ ] Sub-charts: PostgreSQL (Bitnami), Valkey, MinIO (optional — external supported)
- [ ] Published to OCI registry and documented in chart repo
- [ ] Upgrade path documented (Helm diff + rollback procedure)

### Horizontal Scaling
- [ ] Server: stateless HTTP handlers; HPA on CPU/request rate
- [ ] Agent routing: consistent-hash load balancer ensures agent reconnects to same replica (gRPC affinity)
- [ ] Job queue (River): workers scale independently via replica count
- [ ] Session store in Valkey (not in-process) to support multiple server replicas

### Air-Gapped OVA Appliance
- [ ] OVA packages: Ubuntu 24.04 LTS + Docker + all service images + initial catalog snapshot
- [ ] First-boot wizard: network config, admin password, license activation (offline token)
- [ ] Self-update mechanism disabled by default; manual update via uploaded OVA diff package
- [ ] Offline catalog update: import `.piq-catalog` bundle from USB or network share
- [ ] Appliance hardening checklist (CIS L1 applied at build time)

### Load Testing
- [ ] Agent simulator: k6 extension that speaks the PatchIQ gRPC protocol
- [ ] Test scenario: 5,000 concurrent agents — enroll, heartbeat, sync outbox, sync inbox
- [ ] Targets: p99 heartbeat latency < 500ms, zero dropped messages at sustained load
- [ ] Results published as benchmark artifact in CI on release branches
- [ ] Identified bottlenecks addressed before M4 GA

## License Gating

- Kubernetes Helm chart: ENTERPRISE
- Air-gapped OVA appliance: ENTERPRISE
- Horizontal scaling: ENTERPRISE
