# Alert Pipelines

**Status**: Planned
**Milestone**: M3
**Dependencies**: Workflow event triggers, notification node, tags system

---

## Vision

Transform alerting from simple notifications into configurable, intelligent pipelines powered by the workflow engine. Alerts are triggered by domain events, enriched with context, routed by tag-based rules, and escalated through configurable chains — all defined visually in the workflow builder.

## User Value

- **Intelligent routing**: Route alerts based on endpoint tags — `priority:critical` → PagerDuty, `env:dev` → Slack, `compliance:hipaa` → email to compliance officer
- **Escalation chains**: If not acknowledged in 15 minutes, escalate to team lead. If still unacked in 30 minutes, escalate to VP.
- **Alert suppression**: Deduplicate, throttle, and correlate related alerts to prevent alert fatigue
- **Rich context**: Alerts include deployment summary, affected endpoints, CVE details, compliance report — not just "something happened"
- **Custom triggers**: Any domain event can trigger an alert pipeline — not just deployments, but compliance drift, scan results, CVE publications, policy violations

## Architecture

### Alert Pipeline as a Workflow

An alert pipeline is a workflow with specialized nodes:

```
Trigger (event) → Enrich (add context) → Route (by tags) → Notify (channel) → Escalate (if unacked)
```

Example workflow DAG:
```
[on: deployment.failed]
    │
    ▼
[Enrich: fetch deployment details, affected endpoints, patch info]
    │
    ▼
[Decision: is env:production?]
    ├── Yes → [Notify: PagerDuty] → [Gate: acked within 15min?]
    │                                    ├── No → [Notify: VP email]
    │                                    └── Yes → [Complete]
    └── No  → [Notify: Slack #deployments] → [Complete]
```

### Alert-Specific Workflow Nodes

| Node | Purpose |
|------|---------|
| `enrich` | Fetch context data (deployment details, endpoint info, CVE data) and add to workflow context |
| `deduplicate` | Suppress alert if identical alert fired within N minutes |
| `throttle` | Rate-limit alerts per source/type (max N per hour) |
| `escalate` | Wait for acknowledgment, escalate if timeout reached |
| `correlate` | Group related alerts (e.g., 50 endpoints failing same patch = 1 alert, not 50) |

### Alert Channels

Extends existing Shoutrrr notification system:
- Email (existing)
- Slack (existing)
- Discord (existing)
- Webhook (existing)
- PagerDuty (new)
- Microsoft Teams (new)
- OpsGenie (new)

### Tag-Based Routing

Alert routing rules use tag expressions:
```json
{
  "route": {
    "match": {"tag": "priority", "value": "critical"},
    "channel": "pagerduty",
    "config": {"service_key": "..."}
  }
}
```

Multiple routes can match — alert goes to all matching channels.

## Foundations Built in M2

- **Workflow engine**: Alert pipelines are workflows — same DAG executor, same UI builder
- **Event triggers**: `deployment.failed`, `compliance.drift`, `cve.published` etc. already defined
- **Notification node**: Existing Shoutrrr integration, extended with contextual body
- **Tags**: Alert routing uses same tag expression system as everything else
- **Watermill event bus**: All domain events already flowing through pub/sub

## License Gating

- Basic notifications (email on deployment complete): all tiers
- Alert pipelines (workflow-based routing): PROFESSIONAL+
- Escalation chains: ENTERPRISE
- Alert correlation and deduplication: ENTERPRISE
- PagerDuty/OpsGenie integration: ENTERPRISE
