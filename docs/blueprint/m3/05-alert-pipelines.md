# Alert Pipelines

**Status**: Planned
**Wave**: 2 — Automation & Extensibility
**Dependencies**: Workflow event triggers, notification node, tags system

---

## Vision

Configurable, intelligent alert pipelines powered by the workflow engine. Alerts triggered by domain events, enriched with context, routed by tag-based rules, escalated through configurable chains.

## Deliverables

### Alert-Specific Workflow Nodes
- [ ] `enrich`: fetch context data (deployment details, endpoint info, CVE data)
- [ ] `deduplicate`: suppress if identical alert fired within N minutes
- [ ] `throttle`: rate-limit per source/type (max N per hour)
- [ ] `escalate`: wait for acknowledgment, escalate on timeout
- [ ] `correlate`: group related alerts (50 endpoints failing = 1 alert)

### Tag-Based Routing
- [ ] Route rules using tag expressions → channel mapping
- [ ] Multiple routes can match — alert goes to all matching channels
- [ ] Route priority and fallback chains

### New Notification Channels
- [ ] PagerDuty integration
- [ ] Microsoft Teams integration
- [ ] OpsGenie integration

### Escalation Chains
- [ ] Configurable timeout-based escalation (15min → team lead, 30min → VP)
- [ ] Acknowledgment tracking via email/webhook callback
- [ ] Escalation history in audit log

### Alert Correlation
- [ ] Group related alerts by source, type, or time window
- [ ] Summary alert with affected count and drill-down link
- [ ] Configurable correlation rules

## License Gating
- Basic notifications: all tiers
- Alert pipelines (workflow-based routing): PROFESSIONAL+
- Escalation chains: ENTERPRISE
- Alert correlation/deduplication: ENTERPRISE
- PagerDuty/OpsGenie: ENTERPRISE
