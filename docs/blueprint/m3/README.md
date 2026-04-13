# M3 — Intelligence + Hub

> **Goal**: Make the platform smart, reliable, and enterprise-grade. Three waves: polish the foundation, add automation, then layer AI on top.

## Waves

### Wave 1: Foundation Polish (unblocks everything)
| Feature | Status | Doc |
|---------|--------|-----|
| [UI Design Overhaul](01-ui-design-overhaul.md) | **Next Up** | Enterprise theming, component consistency, navigation redesign |
| [Hub Manager E2E](02-hub-manager-e2e.md) | Planned | Reliable feeds, catalog publish, Hub→PM sync for real |
| [Full Compliance Engine](03-full-compliance-engine.md) | Planned | 6 frameworks E2E, exceptions, evidence reports, custom frameworks |
| [Extended Agent Collectors](04-extended-agent-collectors.md) | Planned | Security, apps, energy, network, certificates |

### Wave 2: Automation & Extensibility
| Feature | Status | Doc |
|---------|--------|-----|
| [Alert Pipelines](05-alert-pipelines.md) | Planned | Workflow-driven routing, escalation, PagerDuty/Teams/OpsGenie |
| [Script-based Collectors](06-script-based-collectors.md) | Planned | Script library, scheduled collection, compliance integration |
| [Approval Workflows](07-approval-workflows.md) | Planned | Approval chains, ITSM webhooks, email approve/reject |
| [3rd-Party App Patching](08-3rd-party-app-patching.md) | Planned | Top 25 apps, detection rules, silent install definitions |
| [Baseline Profiles](09-baseline-profiles.md) | Planned | Desired-state, golden endpoint capture, drift detection, auto-remediate |

### Wave 3: AI
| Feature | Status | Doc |
|---------|--------|-----|
| [AI Assistant (MCP + Claude)](10-ai-assistant.md) | Planned | 13 tools, human-in-the-loop, RBAC, chat panel, context awareness |
| [MSP Portal Foundations](11-msp-portal-foundations.md) | Planned | Multi-tenant management, per-tenant policies, cross-tenant dashboards |

## Work Streams (3 devs)

| Dev | Track | Focus |
|-----|-------|-------|
| Dev 1 | Platform | Hub E2E (feed reliability, sync pipeline), extended collectors (agent-side), 3rd-party app patching, MCP server + AI tools |
| Dev 2 | Engine | Full compliance engine (evaluator, scoring, exceptions, evidence, custom frameworks), approval workflows, baseline profiles (enforcement, drift), structured patch pipeline |
| Dev 3 | Surface | UI design overhaul (all 3 apps), alert pipeline UI, script library UI, compliance dashboard (heatmap, scorecards, reports), AI chat panel, MSP portal foundations |

## Exit Criteria

- [ ] All 3 UIs have consistent enterprise-grade theming, navigation, and component patterns
- [ ] Hub feeds sync reliably; catalog publishes to PM automatically; 0% → 100% sync rate
- [ ] Compliance dashboard shows scores for 6+ frameworks with drill-down to per-endpoint detail
- [ ] Exception workflow: request → justify → approve → track expiry
- [ ] Evidence report (PDF) passes a mock audit review
- [ ] Alert pipeline routes critical CVE alerts to PagerDuty within 60 seconds
- [ ] Baseline profile assigned; drift detected and auto-remediated via wave deployment
- [ ] Admin asks AI "Which endpoints have critical unpatched CVEs?" → accurate answer
- [ ] 3rd-party apps (Chrome, Firefox, Adobe) detected and patchable on all 3 OS
- [ ] All existing CI checks pass
