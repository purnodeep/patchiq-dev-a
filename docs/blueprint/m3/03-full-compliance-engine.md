# Full Compliance Engine

**Status**: Planned
**Wave**: 1 — Foundation Polish
**Priority**: High — enterprise clients require audit-ready compliance

---

## Problem

M2 built compliance scaffolding (6 preset frameworks, basic scoring, evaluation trigger) but it's not production-ready:

- Compliance scores are framework-level only — no per-endpoint compliance tracking
- No exception management (compensating controls, approval workflow, expiry)
- No evidence reports (PDF/CSV for auditors)
- No custom framework builder (org-specific controls)
- Per-CVE state machine not implemented (only pass/fail)
- Process adherence checks missing (scan frequency, wave testing, rollback capability)
- No scheduled compliance reports

## Goals

1. **Per-endpoint compliance** — each endpoint has a compliance score per framework
2. **Per-CVE state machine** — COMPLIANT / AT RISK / NON-COMPLIANT / LATE REMEDIATION / EXCEPTED / NOT APPLICABLE
3. **3 scoring methods** — strictest (FedRAMP), average (SOC2/ISO), worst-case (PCI DSS)
4. **Exception management** — structured justifications, compensating controls, approval workflow, expiry tracking
5. **Evidence reports** — PDF, CSV, JSON, HTML per-control, per-endpoint, per-organization
6. **Custom framework builder** — define org-specific controls and SLAs
7. **Scheduled reports** — daily/weekly/monthly compliance snapshots
8. **Process adherence** — verify scan frequency, wave testing, rollback capability, approval gates

## Deliverables

### Per-Endpoint Compliance
- [ ] Compliance evaluator runs per endpoint, not just tenant-wide
- [ ] Endpoint detail Compliance tab shows per-framework scores
- [ ] Dashboard shows compliance distribution heatmap (endpoints × frameworks)
- [ ] Non-compliant endpoint drill-down with remediation guidance

### Per-CVE State Machine
- [ ] 6 states: COMPLIANT, AT_RISK, NON_COMPLIANT, LATE_REMEDIATION, EXCEPTED, NOT_APPLICABLE
- [ ] State transitions tracked with timestamps and actor
- [ ] SLA tracking per state (e.g., AT_RISK for >72h → NON_COMPLIANT)
- [ ] State history visible in CVE detail and audit log

### Scoring Methods
- [ ] Strictest (FedRAMP): overall score = lowest framework score
- [ ] Average (SOC2/ISO): overall score = mean of framework scores
- [ ] Worst-case (PCI DSS): one NON_COMPLIANT control = entire framework fails
- [ ] Per-tenant scoring method configuration

### Exception Management
- [ ] Exception request form: CVE, justification, compensating control, requested duration
- [ ] Approval workflow: request → review → approve/reject (uses Approval Workflow feature)
- [ ] Exception tracking: approved exceptions visible in compliance dashboard
- [ ] Exception expiry: auto-transition back to NON_COMPLIANT when exception expires
- [ ] Audit trail: all exception actions logged

### Evidence Reports
- [ ] PDF report generation (Gotenberg HTML → PDF)
- [ ] Per-control evidence: what was checked, result, timestamp, remediation status
- [ ] Per-endpoint report: all frameworks, all controls, compliance history
- [ ] Organization-level report: executive summary, framework scores, trend charts
- [ ] Scheduled delivery: daily/weekly/monthly to configured email addresses
- [ ] Report templates: SOC 2 auditor format, PCI DSS ROC format, generic

### Custom Framework Builder
- [ ] Create custom framework with name, description, version
- [ ] Add controls: name, description, check type (built-in/script/patch), severity, SLA
- [ ] Map controls to built-in checks or script-based collectors
- [ ] Enable/disable controls per framework
- [ ] Framework versioning and diff display

## Dependencies
- Extended Agent Collectors (for built-in security checks)
- Script-based Collectors (for custom compliance rules)
- Approval Workflows (for exception management)

## License Gating
- Preset frameworks (6): PROFESSIONAL+
- Custom frameworks: ENTERPRISE
- Exception management: ENTERPRISE
- Evidence reports (PDF): ENTERPRISE
- Scheduled reports: ENTERPRISE
