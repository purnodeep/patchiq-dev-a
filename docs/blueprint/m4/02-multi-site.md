# Multi-Site Distribution

**Status**: Planned
**Milestone**: M4
**Dependencies**: M1 Core Loop, M2 Usable Product, Hub-Server sync (M1), Tag system (M2)

---

## Vision

Enable PatchIQ to serve geographically distributed organizations by placing distribution servers closer to agents, reducing WAN traffic and improving reliability at remote sites.

## Deliverables

### Hub-and-Spoke Topology
- [ ] Distribution Server (DS) role: lightweight Patch Manager node that caches content and proxies agent comms
- [ ] Each DS supports up to 5,000 agents
- [ ] Parent PM pushes policies and patch content to DS nodes
- [ ] DS registers with parent PM; health and sync status visible in topology UI
- [ ] Agent enrollment targets nearest DS (by tag or manual assignment)

### DMZ / Mixed-Zone Mode
- [ ] Proxy node sits in DMZ; forwards agent traffic to core PM on internal network
- [ ] No patch content stored in DMZ node (pass-through only)
- [ ] mTLS preserved end-to-end through proxy
- [ ] Firewall rule documentation for DMZ deployment

### Federated Multi-HQ
- [ ] Independent Patch Managers per HQ, each with full autonomy
- [ ] Hub provides unified cross-PM reporting (compliance, patch status, CVE exposure)
- [ ] No direct PM-to-PM communication; Hub is the aggregation point

### Operations
- [ ] Bandwidth throttling per DS link (MB/s cap, scheduled windows)
- [ ] Content caching: patch binaries cached at DS, served locally to agents
- [ ] Topology management UI: add/remove DS nodes, view sync status, replication health
- [ ] Alerting: DS offline, sync lag > threshold, capacity warnings

## License Gating

- Hub-and-Spoke / DS nodes: ENTERPRISE
- DMZ proxy mode: ENTERPRISE
- Federated Multi-HQ reporting: ENTERPRISE
