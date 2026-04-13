# Support System

**Status**: Planned
**Milestone**: M4
**Dependencies**: M2 Hub telemetry sync, M3 MSP foundations, M2 agent heartbeat and inventory

---

## Vision

Give MSP operators and internal support teams proactive visibility into client health and the tools to diagnose and route issues without requiring direct access to client infrastructure.

## Deliverables

### Client Health Scoring
- [ ] Hub aggregates telemetry per managed PM: compliance score, patch currency, agent connectivity rate, CVE exposure
- [ ] Composite health score (0–100) computed per client on a rolling 24-hour window
- [ ] Score trend chart: 30-day history per client
- [ ] Health score thresholds: HEALTHY / AT RISK / CRITICAL with configurable cutoffs
- [ ] Alerting: notify MSP admin when client drops below AT RISK threshold

### Remote Diagnostics
- [ ] Support bundle collection: agent gathers logs, config snapshot, recent event history, system info
- [ ] Bundle requested from Hub or Patch Manager UI; agent streams bundle as encrypted ZIP
- [ ] Bundle stored in MinIO with TTL (7 days default); download link shared with support ticket
- [ ] Automated diagnostic checks included in bundle: connectivity test, disk space, service status

### Ticket Routing
- [ ] Issue classifier: categorizes support events by type (connectivity, compliance failure, deployment error, CVE alert)
- [ ] Routing rules: map issue type + client tier to support queue (L1 / L2 / escalation)
- [ ] Integration hooks: route to external ticketing systems (Jira, Zendesk, ServiceNow) via webhook
- [ ] Auto-ticket creation on critical health score drop or SLA breach

### Support Dashboard (Hub)
- [ ] Fleet health overview: all clients ranked by health score
- [ ] Open issues panel: unresolved alerts across all clients with age and severity
- [ ] Diagnostic bundle history: recent bundles per client, download, expiry indicator
- [ ] Support metrics: MTTR per issue type, ticket volume trend, SLA compliance rate

## License Gating

- Client health scoring: ENTERPRISE
- Remote diagnostics: ENTERPRISE
- Ticket routing + external integrations: ENTERPRISE
- Support dashboard: ENTERPRISE
