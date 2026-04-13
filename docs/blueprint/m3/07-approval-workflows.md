# Approval Workflows

**Status**: Planned
**Wave**: 2 — Automation & Extensibility
**Dependencies**: Workflow builder, notification system

---

## Vision

Configurable approval chains that gate deployments, policy changes, and exception requests. Visual workflow builder integration makes approval logic composable with deployment logic.

## Deliverables

### Approval Chains
- [ ] 1-level approval: single approver
- [ ] 2-level approval: team lead → manager
- [ ] Group approval: any N of M approvers (quorum-based)
- [ ] Approval timeout: auto-reject or escalate after configurable duration

### Workflow Integration
- [ ] Approval node in workflow builder palette (already exists as placeholder)
- [ ] Approval node blocks workflow execution until approved
- [ ] Approval node shows pending count in workflow execution view
- [ ] Multiple approval nodes in a single workflow (e.g., security review → change board)

### ITSM Integration
- [ ] Webhook-based integration for ticketing systems (ServiceNow, Jira Service Management)
- [ ] Auto-create ticket on approval request, close on approve/reject
- [ ] Bidirectional sync: ticket status updates reflected in PatchIQ

### Email Approval
- [ ] Approve/reject via email link (signed URL, time-limited)
- [ ] Email includes deployment summary, affected endpoints, risk context
- [ ] Audit trail: who approved, when, via what channel

## License Gating
- 1-level approval: PROFESSIONAL+
- Multi-level approval: ENTERPRISE
- ITSM integration: ENTERPRISE
- Email approval: PROFESSIONAL+
