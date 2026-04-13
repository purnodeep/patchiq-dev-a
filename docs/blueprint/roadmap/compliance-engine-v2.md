# Compliance Engine v2

**Status**: Planned
**Milestone**: M3
**Dependencies**: Tags system, `run_script` command, workflow event triggers

---

## Vision

Move beyond patch compliance to full **endpoint compliance**. Clients want to define, monitor, and audit arbitrary compliance rules — not just "is this patch installed?" but "does this endpoint meet our security, operational, and regulatory requirements?"

## User Value

- Custom compliance frameworks alongside preset ones (CIS, PCI-DSS, HIPAA, NIST, ISO 27001, SOC 2)
- Endpoint health monitoring: antivirus presence, firewall state, disk encryption, VPN/proxy connectivity, USB policy, screen lock timeout
- Continuous monitoring with drift detection and instant notification
- Audit-ready compliance reports with evidence trails
- Governance layer for enterprise clients — prove endpoints meet contractual obligations

## Architecture

### Compliance Rules

A compliance rule is a named check with:
- **Check type**: built-in (antivirus, firewall, encryption, etc.) or script-based (custom)
- **Expected result**: boolean (pass/fail), threshold (value >= N), or regex match
- **Tag scope**: which endpoints this rule applies to (tag expression)
- **Severity**: critical, high, medium, low (drives SLA and alerting)
- **Remediation**: optional — link to a deployment, script, or workflow that fixes non-compliance

### Compliance Frameworks (Custom)

Admins compose frameworks from rules:
```
Framework: "SOC 2 Endpoint Baseline"
├── Rule: Disk encryption enabled (built-in check)
├── Rule: Antivirus installed and updated (built-in check)
├── Rule: Screen lock < 5 min (built-in check)
├── Rule: VPN connected during business hours (script-based)
├── Rule: No unauthorized software (script-based)
└── Rule: All critical patches applied within 72h (patch compliance)
```

### Data Flow

```
Agent (built-in + script collectors)
    │ scan results
    ▼
Server (compliance evaluation engine)
    │ rule matching per endpoint
    ▼
Compliance score per endpoint per framework
    │
    ├── Dashboard: compliance gauges, drift alerts
    ├── Workflow trigger: compliance.drift event
    └── Audit log: compliance state changes
```

### Workflow Integration

- `compliance_check` workflow node gates deployments on compliance state
- `compliance.drift` event triggers remediation workflows
- Notification node includes compliance report in alert body

## Foundations Built in M2

- **Tags**: Compliance scope uses tag expressions (`compliance:pci-dss`)
- **`run_script` command**: Script-based compliance checks run on agent
- **Workflow triggers**: `compliance.drift` event type already defined
- **`compliance_check` workflow node**: Defined in M2 spec as placeholder
- **Module registry**: New built-in collectors register as modules

## License Gating

- Preset compliance frameworks: PROFESSIONAL+
- Custom compliance frameworks: ENTERPRISE
- Script-based compliance checks: ENTERPRISE
- Continuous monitoring with drift alerts: ENTERPRISE
