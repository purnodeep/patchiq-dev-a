# Compliance Engine — Design & Guidelines

> Status: Phase 1 (basic), Phase 3 (full) | License: Professional and above

## Overview

The PatchIQ Compliance Engine continuously measures patching practices against regulatory framework requirements, computes compliance scores, and generates audit-ready evidence reports. It does not certify compliance — it produces the evidence an auditor needs to make that determination.

**Compliance answers one question:** "Can we prove to an auditor that our patching process meets the requirements of framework X?"

This is distinct from vulnerability management:
- **Vulnerability management**: What's broken and how do we fix it?
- **Compliance**: Was it fixed fast enough, and was the right process followed?

---

## 1. Supported Frameworks

### 1.1 Framework-to-Control Mapping

Each compliance framework is a large document. PatchIQ only covers the **patch management and vulnerability remediation controls** within each framework — not the entire framework.

| Framework | Relevant Controls | Scope |
|-----------|------------------|-------|
| **NIST 800-53 / FedRAMP** | SI-2 (Flaw Remediation), CM-3 (Configuration Change Control), RA-5 (Vulnerability Scanning) | US federal agencies, FedRAMP-authorized cloud services, government contractors |
| **SOC 2 (Type II)** | CC7.1 (System Monitoring), CC8.1 (Change Management) | SaaS companies, service providers, any org undergoing SOC 2 audit |
| **HIPAA** | §164.308(a)(5)(ii)(B) (Security Updates), §164.312(a)(1) (Access Controls) | Healthcare providers, health plans, business associates handling PHI |
| **PCI DSS v4.0** | Req 6.3.3 (Security Patches), Req 11.3.1 (Vulnerability Scanning) | Any organization that processes, stores, or transmits credit card data |
| **ISO 27001:2022** | A.8.8 (Management of Technical Vulnerabilities), A.8.32 (Change Management) | International standard, any organization seeking ISO certification |
| **CIS Controls v8** | Control 7 (Continuous Vulnerability Management) | Organizations adopting CIS benchmarks as a security baseline |
| **DISA STIG** | V-patch requirements per STIG checklist | US Department of Defense systems |
| **Cyber Essentials** | Patch Management requirement | UK government suppliers, UK-based organizations |

### 1.2 What Each Framework Requires (Patch Management Scope)

#### NIST 800-53 / FedRAMP (Most Prescriptive)

```
Control: SI-2 — Flaw Remediation

Requirements:
  a) Identify, report, and correct system flaws
  b) Test patches before installation
  c) Install security-relevant patches within defined timelines
  d) Incorporate flaw remediation into configuration management

Patch SLA Timelines (per FedRAMP guidance):
  Critical (CVSS 9.0-10.0):  15 calendar days
  High (CVSS 7.0-8.9):       30 calendar days
  Moderate (CVSS 4.0-6.9):   90 calendar days
  Low (CVSS 0.1-3.9):        No fixed timeline (next maintenance cycle)

Control: RA-5 — Vulnerability Scanning
  - Scan at least every 72 hours (for high-impact systems)
  - Scan after new vulnerabilities are identified
  - Share scan results with relevant personnel

Control: CM-3 — Configuration Change Control
  - Document and approve changes (patches)
  - Audit and review changes
  - Retain records of changes
```

#### SOC 2 (Type II)

```
Control: CC7.1 — System Monitoring
  - Monitor systems for vulnerabilities
  - Evaluate and respond to identified vulnerabilities

Control: CC8.1 — Change Management
  - Changes (including patches) follow a documented process
  - Changes are tested before production deployment
  - Changes are authorized by appropriate personnel
  - Rollback procedures exist
  - Changes are logged and auditable

Patch SLA Timelines (not prescribed — org defines its own):
  Typical industry practice:
    Critical: 14-30 days
    High: 30-60 days
    Moderate: 60-90 days
    Low: Next maintenance cycle

Evidence required:
  - Change management policy documentation
  - Patch testing records
  - Approval records with timestamps
  - Deployment logs showing rollback capability
  - Monitoring dashboards showing vulnerability trends
```

#### HIPAA

```
Control: §164.308(a)(5)(ii)(B) — Security Updates and Patches
  - Procedures for guarding against, detecting, and reporting
    malicious software
  - Procedures for installing security-relevant patches

Patch SLA Timelines (not prescribed — "reasonable and appropriate"):
  Guidance from HHS:
    Critical: 30 days (industry consensus)
    High: 60 days
    Others: Risk-based timeline

Evidence required:
  - Written patch management policy
  - Risk assessment showing patch prioritization methodology
  - Patch deployment records with dates
  - Exception documentation for delayed patches
  - Periodic review of patching effectiveness
```

#### PCI DSS v4.0

```
Requirement 6.3.3:
  - Install critical and high security patches within one month
    of release
  - All other patches within an appropriate time frame determined
    by risk analysis

Requirement 11.3.1:
  - Perform internal vulnerability scans at least quarterly
  - Re-scan to verify remediation

Patch SLA Timelines:
  Critical/High: 30 calendar days from vendor release
  Medium/Low: Risk-based, documented rationale required

Evidence required:
  - Patch installation records with dates
  - Vulnerability scan reports (before and after patching)
  - Risk ranking methodology documentation
  - Exception process documentation
```

#### ISO 27001:2022

```
Control A.8.8 — Management of Technical Vulnerabilities:
  - Obtain information about technical vulnerabilities in a timely fashion
  - Evaluate exposure to vulnerabilities
  - Take appropriate measures to address the risk
  - Verify effectiveness of remediation

Control A.8.32 — Change Management:
  - Changes must follow a formal change management process

Patch SLA Timelines (not prescribed — risk-based):
  Organization defines its own based on risk assessment
  Must be documented in the vulnerability management policy

Evidence required:
  - Vulnerability management procedure document
  - Risk assessment methodology
  - Patch deployment records
  - Verification that patches were applied successfully
  - Change management records
```

#### CIS Controls v8

```
Control 7 — Continuous Vulnerability Management:
  7.1 — Establish and maintain a vulnerability management process
  7.2 — Establish and maintain a remediation process
  7.3 — Perform automated operating system patch management
  7.4 — Perform automated application patch management
  7.5 — Perform automated vulnerability scans of internal assets
  7.6 — Perform automated vulnerability scans of externally-exposed assets
  7.7 — Remediate detected vulnerabilities

Patch SLA Timelines:
  Per 7.2: Organization defines based on risk
  CIS recommends: Critical within 14 days, others risk-based

Evidence required:
  - Documented vulnerability management process
  - Automated scan results
  - Remediation records with timelines
  - Trend data showing improvement over time
```

---

## 2. Compliance Engine Architecture

### 2.1 Components

```
┌──────────────────────────────────────────────────────────────────┐
│                      Compliance Engine                            │
│                                                                   │
│  ┌────────────────────┐                                           │
│  │ Framework Registry │  Pre-loaded framework profiles with       │
│  │                    │  control definitions and SLA timelines    │
│  └────────┬───────────┘                                           │
│           │                                                       │
│  ┌────────▼───────────┐     ┌──────────────────────────────────┐ │
│  │ Policy Binder      │     │ Data Sources                      │ │
│  │                    │     │                                    │ │
│  │ Maps client's      │     │ ┌────────────────────────────┐   │ │
│  │ frameworks to      │     │ │ CVE Feed                    │   │ │
│  │ endpoint groups    │◄────│ │ (NVD, CISA KEV, vendor)     │   │ │
│  │                    │     │ │ - CVE ID, severity (CVSS)   │   │ │
│  └────────┬───────────┘     │ │ - publish date (SLA clock)  │   │ │
│           │                 │ │ - affected products/versions │   │ │
│  ┌────────▼───────────┐     │ └────────────────────────────┘   │ │
│  │ Compliance         │     │ ┌────────────────────────────┐   │ │
│  │ Evaluator          │     │ │ Endpoint Inventory          │   │ │
│  │                    │◄────│ │ - installed packages        │   │ │
│  │ Per endpoint:      │     │ │ - versions                  │   │ │
│  │ - SLA check        │     │ │ - last scan time            │   │ │
│  │ - Process check    │     │ └────────────────────────────┘   │ │
│  │ - Score compute    │     │ ┌────────────────────────────┐   │ │
│  │                    │◄────│ │ Deployment History           │   │ │
│  └────────┬───────────┘     │ │ - when patched              │   │ │
│           │                 │ │ - elapsed time from CVE     │   │ │
│  ┌────────▼───────────┐     │ │ - was it wave-deployed?     │   │ │
│  │ Exception Manager  │     │ │ - was rollback available?   │   │ │
│  │                    │     │ └────────────────────────────┘   │ │
│  │ - Documented       │     │ ┌────────────────────────────┐   │ │
│  │   justification    │◄────│ │ Audit Trail                 │   │ │
│  │ - Compensating     │     │ │ - who approved              │   │ │
│  │   controls         │     │ │ - when approved             │   │ │
│  │ - Expiry date      │     │ │ - workflow followed?        │   │ │
│  │ - Review schedule  │     │ │ - testing completed?        │   │ │
│  └────────┬───────────┘     │ └────────────────────────────┘   │ │
│           │                 └──────────────────────────────────┘ │
│  ┌────────▼───────────┐                                          │
│  │ Report Generator   │                                          │
│  │                    │                                          │
│  │ - Executive summary│                                          │
│  │ - Per-control      │                                          │
│  │   evidence         │                                          │
│  │ - Exception log    │                                          │
│  │ - Trend analysis   │                                          │
│  │ - Export: PDF, CSV │                                          │
│  └────────────────────┘                                          │
└──────────────────────────────────────────────────────────────────┘
```

### 2.2 Evaluation Flow

For every endpoint in a compliance-bound group, the engine runs this evaluation:

```
For each endpoint in scope:
    For each CVE affecting this endpoint:
        1. DETERMINE SLA DEADLINE
           - CVE publish date + framework SLA for this severity
           - Example: CVE published Jan 1, severity Critical, FedRAMP
             → Deadline: Jan 16 (15 days)

        2. CHECK REMEDIATION STATUS
           - Is the vulnerable package still installed at the affected version?
           - If patched: when was the patch deployed?
           - Elapsed time = patch_date - cve_publish_date

        3. EVALUATE COMPLIANCE
           - If patched within SLA → COMPLIANT for this CVE
           - If patched but exceeded SLA → LATE REMEDIATION (violation)
           - If not patched and within SLA → AT RISK (approaching deadline)
           - If not patched and past SLA → NON-COMPLIANT (violation)
           - If exception exists → EXCEPTED (documented, with justification)

        4. CHECK PROCESS ADHERENCE (framework-specific)
           - Was the deployment approved by an authorized user? (CM-3, CC8.1)
           - Was a canary/test wave used before production? (SI-2b, CC8.1)
           - Was rollback configured? (CC8.1)
           - Was the scan performed within required frequency? (RA-5)

    Compute endpoint compliance score:
        score = (compliant_cves + excepted_cves) / total_applicable_cves * 100

Compute group compliance score:
    score = average of endpoint scores (or: % of endpoints fully compliant)
```

### 2.3 Compliance States

Each CVE-endpoint pair has one of these states:

| State | Meaning | Color | Action Required |
|-------|---------|-------|-----------------|
| **COMPLIANT** | Patched within SLA, proper process followed | Green | None |
| **AT RISK** | Not yet patched, SLA deadline approaching (configurable threshold, e.g., 75% of window elapsed) | Yellow | Prioritize patching |
| **NON-COMPLIANT** | SLA deadline exceeded, not patched | Red | Immediate action + exception documentation |
| **LATE REMEDIATION** | Patched, but after SLA deadline | Orange | Document for audit, review root cause |
| **EXCEPTED** | Not patched, but has documented exception with justification and compensating controls | Blue | Review on exception expiry date |
| **NOT APPLICABLE** | CVE doesn't affect this endpoint (false positive, or compensating control eliminates the risk) | Gray | None |

---

## 3. Framework Profiles (Shipped With Product)

### 3.1 Profile Schema

Each framework profile is a structured definition shipped with the product and versioned with releases:

```json
{
  "framework_id": "nist_800_53_r5",
  "framework_name": "NIST 800-53 Rev. 5",
  "version": "2026.1",
  "description": "NIST Special Publication 800-53 Revision 5 — Security and Privacy Controls",
  "applicable_industries": ["government", "defense", "federal_contractors"],
  "controls": [
    {
      "control_id": "SI-2",
      "control_name": "Flaw Remediation",
      "description": "Identify, report, and correct system flaws. Install security-relevant patches within defined timelines.",
      "requirements": {
        "patch_sla": {
          "critical": { "days": 15, "severity_range": [9.0, 10.0] },
          "high":     { "days": 30, "severity_range": [7.0, 8.9] },
          "moderate": { "days": 90, "severity_range": [4.0, 6.9] },
          "low":      { "days": null, "severity_range": [0.1, 3.9], "note": "Next maintenance cycle" }
        },
        "require_testing_before_production": true,
        "require_rollback_capability": true,
        "require_approval": true,
        "flaw_reporting": true
      },
      "evidence_types": [
        "patch_timeline_report",
        "testing_records",
        "approval_records",
        "rollback_configuration"
      ]
    },
    {
      "control_id": "RA-5",
      "control_name": "Vulnerability Monitoring and Scanning",
      "description": "Monitor and scan for vulnerabilities in the system and hosted applications.",
      "requirements": {
        "scan_frequency_hours": 72,
        "scan_after_new_cve": true,
        "share_results_with_personnel": true
      },
      "evidence_types": [
        "scan_frequency_report",
        "scan_results_distribution"
      ]
    },
    {
      "control_id": "CM-3",
      "control_name": "Configuration Change Control",
      "description": "Document, approve, audit, and review changes to the system.",
      "requirements": {
        "require_approval": true,
        "require_audit_trail": true,
        "require_change_documentation": true,
        "retain_records": true
      },
      "evidence_types": [
        "change_management_policy",
        "approval_records",
        "audit_trail_export"
      ]
    }
  ]
}
```

```json
{
  "framework_id": "pci_dss_v4",
  "framework_name": "PCI DSS v4.0",
  "version": "2026.1",
  "description": "Payment Card Industry Data Security Standard Version 4.0",
  "applicable_industries": ["retail", "finance", "e-commerce", "payment_processing"],
  "controls": [
    {
      "control_id": "6.3.3",
      "control_name": "Security Patches",
      "description": "Install critical and high security patches within one month of release.",
      "requirements": {
        "patch_sla": {
          "critical": { "days": 30, "severity_range": [9.0, 10.0] },
          "high":     { "days": 30, "severity_range": [7.0, 8.9] },
          "moderate": { "days": null, "severity_range": [4.0, 6.9], "note": "Risk-based timeline, documented" },
          "low":      { "days": null, "severity_range": [0.1, 3.9], "note": "Risk-based timeline, documented" }
        },
        "require_risk_ranking_methodology": true
      },
      "evidence_types": [
        "patch_timeline_report",
        "risk_ranking_documentation"
      ]
    },
    {
      "control_id": "11.3.1",
      "control_name": "Vulnerability Scanning",
      "description": "Perform internal vulnerability scans at least quarterly. Re-scan to verify remediation.",
      "requirements": {
        "scan_frequency_days": 90,
        "rescan_after_remediation": true
      },
      "evidence_types": [
        "quarterly_scan_report",
        "rescan_verification"
      ]
    }
  ]
}
```

### 3.2 Profiles We Ship

| Framework | Controls Covered | SLA Timelines Defined | Status |
|-----------|-----------------|----------------------|--------|
| NIST 800-53 Rev. 5 / FedRAMP | SI-2, RA-5, CM-3 | Yes (prescriptive) | Phase 3 |
| SOC 2 Type II | CC7.1, CC8.1 | Org-defined (we provide templates) | Phase 3 |
| HIPAA | §164.308(a)(5), §164.312(a)(1) | Industry consensus defaults | Phase 3 |
| PCI DSS v4.0 | 6.3.3, 11.3.1 | Yes (prescriptive) | Phase 3 |
| ISO 27001:2022 | A.8.8, A.8.32 | Org-defined (we provide templates) | Phase 3 |
| CIS Controls v8 | 7.1-7.7 | CIS recommendations as defaults | Phase 3 |
| DISA STIG | Per-STIG checklist items | Yes (prescriptive) | Phase 4 |
| Cyber Essentials | Patch requirement | 14-day critical patch window | Phase 4 |
| **Custom** | Admin-defined | Admin-defined | Phase 3 |

### 3.3 Custom Frameworks

Clients can create custom compliance profiles for internal policies that don't map to a standard framework:

```json
{
  "framework_id": "custom_acme_internal",
  "framework_name": "Acme Corp Internal Security Policy",
  "controls": [
    {
      "control_id": "ACME-PATCH-001",
      "control_name": "Critical Patch Response",
      "requirements": {
        "patch_sla": {
          "critical": { "days": 7 },
          "high": { "days": 14 }
        },
        "require_approval": true,
        "require_testing_before_production": true
      }
    }
  ]
}
```

This allows organizations with stricter internal policies (e.g., "critical patches in 7 days, not 15") to track compliance against their own standards.

---

## 4. Compliance Scoring

### 4.1 Per-Endpoint Score

```
Endpoint Compliance Score (0-100):

  Inputs:
    total_applicable_cves    = CVEs that affect this endpoint
    compliant_cves           = patched within SLA with proper process
    excepted_cves            = documented exception, approved
    late_remediation_cves    = patched but after SLA
    non_compliant_cves       = not patched, past SLA

  Score Calculation:
    fully_resolved = compliant_cves + excepted_cves
    score = (fully_resolved / total_applicable_cves) * 100

    Deductions for process violations:
      - Patch deployed without approval: -5 per instance
      - Patch deployed without testing/canary: -3 per instance
      - Scan not performed within required frequency: -10

  Status:
    score >= 95  → COMPLIANT (green)
    score >= 80  → MOSTLY COMPLIANT (yellow)
    score < 80   → NON-COMPLIANT (red)
```

### 4.2 Group Score

```
Group Compliance Score:

  Method 1 (Strictest — used for FedRAMP/DISA):
    score = percentage of endpoints that are individually COMPLIANT (score >= 95)
    "98% of endpoints are fully compliant"

  Method 2 (Average — used for SOC 2/ISO):
    score = average of all endpoint scores in the group
    "Average compliance score: 92%"

  Method 3 (Worst-case — used for PCI DSS):
    score = lowest endpoint score in the group
    "Compliance limited by worst endpoint: 78%"

  The scoring method is configurable per framework.
```

### 4.3 Organization Score

```
Organization Compliance Score (across all frameworks):

  Per framework: use the group scoring method defined above
  Organization-level: report per-framework scores independently

  Dashboard shows:
    NIST 800-53:  94% ██████████████████░░  (23 endpoints at risk)
    SOC 2:        98% ████████████████████  (3 endpoints at risk)
    PCI DSS:      87% █████████████████░░░  (41 endpoints non-compliant)
    HIPAA:       100% ████████████████████  (fully compliant)
```

---

## 5. Exception Management

### 5.1 Why Exceptions Exist

Not every vulnerability can be patched within the SLA window. Valid reasons:
- Patch not yet released by vendor (zero-day with no fix)
- Patch breaks a critical business application (compatibility issue)
- System cannot be rebooted during current business period
- Compensating control mitigates the risk (e.g., network isolation, WAF rule)
- End-of-life system awaiting decommission

Auditors **expect** exceptions. What they don't accept is untracked, undocumented deviations.

### 5.2 Exception Workflow

```
Admin identifies a CVE that cannot be patched within SLA
    │
    ▼
Admin creates an exception request in PatchIQ:
    - CVE ID(s)
    - Affected endpoints/groups
    - Reason category (vendor_delay, compatibility, business_constraint, compensating_control)
    - Detailed justification (free text)
    - Compensating controls applied (if any)
    - Requested exception duration
    │
    ▼
Exception requires approval (configurable approver role):
    - Security officer, CISO, or compliance manager
    - Approval with optional conditions
    │
    ▼
Exception is active:
    - Affected CVE-endpoint pairs move to EXCEPTED status
    - Exception is tracked in compliance reports as a documented deviation
    - Exception has an expiry date (must be re-reviewed)
    │
    ▼
On expiry:
    - If vulnerability is still present → exception must be renewed or remediated
    - Notification sent to admin and approver before expiry (7 days, 3 days, 1 day)
    - If not renewed → reverts to NON-COMPLIANT status
```

### 5.3 Exception Record

```json
{
  "exception_id": "EXC-2026-0042",
  "tenant_id": "tenant-acme",
  "cve_ids": ["CVE-2026-1234"],
  "endpoint_scope": "group:legacy-servers",
  "endpoint_count": 12,
  "framework_id": "nist_800_53_r5",
  "control_id": "SI-2",
  "reason_category": "compatibility",
  "justification": "Patch for OpenSSL 3.2.1 breaks the legacy ERP application (SAP R/3). Vendor ticket SAP-INC-98765 opened. ETA for compatible patch: March 15, 2026.",
  "compensating_controls": [
    "Network segmentation: legacy servers isolated in VLAN 42",
    "WAF rule deployed to block known exploit vectors",
    "Enhanced monitoring: IDS signatures updated for CVE-2026-1234"
  ],
  "requested_by": "admin@acme.com",
  "requested_at": "2026-02-01T10:00:00Z",
  "approved_by": "ciso@acme.com",
  "approved_at": "2026-02-01T14:30:00Z",
  "approval_conditions": "Re-evaluate on March 1 when SAP releases hotfix.",
  "expires_at": "2026-03-15T00:00:00Z",
  "review_schedule": "biweekly",
  "status": "active"
}
```

---

## 6. Evidence Reports

### 6.1 Report Types

| Report | Audience | Content | Frequency |
|--------|----------|---------|-----------|
| **Executive Compliance Summary** | CISO, VP of IT | Per-framework compliance percentage, trend over 90 days, top risks, exceptions count | Monthly / Quarterly |
| **Control Evidence Report** | Auditor | Per-control detail with evidence artifacts (see below) | On-demand (audit preparation) |
| **Patch Timeline Report** | Auditor, IT Manager | Every CVE → patch deployment with elapsed time vs. SLA deadline | On-demand |
| **Exception Report** | Compliance Manager | All active/expired exceptions with justifications and compensating controls | Monthly |
| **Scan Coverage Report** | Auditor | Proof that scans are running at required frequency, with no gaps | On-demand |
| **Process Adherence Report** | Auditor | Proof that change management process was followed (approvals, testing, rollback) | On-demand |
| **Drift Report** | IT Manager | Endpoints that were compliant but drifted out of compliance (new CVEs, missed scans) | Weekly |
| **Trend Report** | CISO | Compliance score over time, mean-time-to-remediate trends, improvement tracking | Monthly / Quarterly |

### 6.2 Control Evidence Report (Auditor-Facing)

This is the primary deliverable for an audit. For each control in the selected framework, the report provides:

```
═══════════════════════════════════════════════════════════════
CONTROL EVIDENCE REPORT
Framework: NIST 800-53 Rev. 5
Scope: All production endpoints (342 endpoints)
Period: January 1, 2026 — March 31, 2026 (Q1)
Generated: April 2, 2026
═══════════════════════════════════════════════════════════════

───────────────────────────────────────────────────────────────
CONTROL: SI-2 — Flaw Remediation
Status: COMPLIANT (with exceptions)
───────────────────────────────────────────────────────────────

REQUIREMENT: Patch critical vulnerabilities within 15 days

  Summary:
    Total critical CVEs in period:         47
    Patched within 15-day SLA:             43 (91.5%)
    Patched late (exceeded SLA):            1 (2.1%)
    Active exceptions (documented):         2 (4.3%)
    Still open (within SLA window):         1 (2.1%)

  Late Remediation Detail:
    CVE-2026-3456 (OpenSSL buffer overflow, CVSS 9.8)
      SLA deadline: February 12, 2026
      Patched on:   February 14, 2026 (2 days late)
      Root cause:   Emergency change freeze during system migration
      Corrective action: Change freeze policy updated to exclude
                         critical security patches

  Active Exceptions:
    EXC-2026-0042: CVE-2026-1234 on legacy-servers (12 endpoints)
      Reason: Vendor compatibility issue (SAP)
      Compensating controls: Network segmentation, WAF, IDS
      Expires: March 15, 2026
      Approved by: CISO on February 1, 2026

    EXC-2026-0051: CVE-2026-5678 on embedded-devices (3 endpoints)
      Reason: Vendor has not released patch (zero-day)
      Compensating controls: Network isolation, enhanced monitoring
      Expires: April 30, 2026
      Approved by: CISO on March 1, 2026

REQUIREMENT: Test patches before production deployment

  Evidence:
    All 43 patch deployments used wave-based deployment:
      Wave 0 (canary, 5% of endpoints): 43/43 deployments ✓
      Minimum soak time before production: 4 hours (configured)
      Rollback triggered: 2 deployments (resolved, re-deployed)

    Deployment records with wave evidence:
      [Attached: deployment_logs_q1_2026.csv — 43 entries]

REQUIREMENT: Approval required for patch deployments

  Evidence:
    All 43 deployments approved by authorized personnel:
      Approved by IT Manager role: 38 deployments
      Approved by CISO role: 5 deployments (critical severity)
      Average approval turnaround: 2.3 hours

    Approval records with timestamps:
      [Attached: approval_audit_trail_q1_2026.csv — 43 entries]

───────────────────────────────────────────────────────────────
CONTROL: RA-5 — Vulnerability Scanning
Status: COMPLIANT
───────────────────────────────────────────────────────────────

REQUIREMENT: Scan at least every 72 hours

  Evidence:
    Scan frequency across all 342 endpoints:
      Average scan interval: 24 hours (configured for daily)
      Maximum gap between scans: 26 hours (Feb 15, planned maintenance)
      Endpoints with scan gaps > 72h: 0

    Scan execution log:
      [Attached: scan_schedule_q1_2026.csv — 90 days × 342 endpoints]

REQUIREMENT: Scan after new vulnerabilities are identified

  Evidence:
    Ad-hoc scans triggered after CISA KEV additions: 12 instances
    Average time from KEV addition to scan completion: 45 minutes

───────────────────────────────────────────────────────────────
CONTROL: CM-3 — Configuration Change Control
Status: COMPLIANT
───────────────────────────────────────────────────────────────

REQUIREMENT: Document and approve all changes

  Evidence:
    Total patch-related changes in period: 43 deployments
    All changes have:
      ✓ Deployment record with description (43/43)
      ✓ Approval by authorized role (43/43)
      ✓ Audit trail entry with actor and timestamp (43/43)
      ✓ Rollback configuration (43/43)

    Full audit trail export:
      [Attached: change_audit_trail_q1_2026.csv]

═══════════════════════════════════════════════════════════════
APPENDIX A: Methodology
  - Compliance evaluated against NIST SP 800-53 Rev. 5
  - CVSS v3.1 scores from NVD used for severity classification
  - SLA timelines per FedRAMP Low/Moderate/High baseline guidance
  - Scan frequency measured as time between consecutive completed scans
  - All timestamps in UTC

APPENDIX B: Attached Evidence Files
  1. deployment_logs_q1_2026.csv
  2. approval_audit_trail_q1_2026.csv
  3. scan_schedule_q1_2026.csv
  4. change_audit_trail_q1_2026.csv
  5. exception_register_q1_2026.csv
  6. patch_policy_document.pdf (exported from PatchIQ workflow)
═══════════════════════════════════════════════════════════════
```

### 6.3 Export Formats

| Format | Use Case |
|--------|----------|
| **PDF** | Executive summaries, audit submissions |
| **CSV** | Raw data for auditor analysis, import into GRC tools |
| **JSON** | API consumption, integration with SIEM/GRC platforms |
| **HTML** | In-app viewing with interactive drill-down |

---

## 7. Compliance Dashboard (UI)

### 7.1 Dashboard Elements

```
┌──────────────────────────────────────────────────────────────────┐
│  COMPLIANCE OVERVIEW                                 [Export ▼]  │
│                                                                  │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌────────────┐   │
│  │ NIST 800-53│ │   SOC 2    │ │  PCI DSS   │ │   HIPAA    │   │
│  │   94%      │ │   98%      │ │   87%      │ │   100%     │   │
│  │  ██████░░  │ │  ████████  │ │  █████░░░  │ │  ████████  │   │
│  │  23 at risk│ │  3 at risk │ │  41 issues │ │  compliant │   │
│  └────────────┘ └────────────┘ └────────────┘ └────────────┘   │
│                                                                  │
│  ┌───────────────────────────────────────────────────────────┐   │
│  │ COMPLIANCE HEATMAP                                         │   │
│  │                NIST    SOC2    PCI     HIPAA    ISO        │   │
│  │ Prod Servers   ██ 95%  ██ 99%  ██ 90%  ██ 100%  ██ 96%   │   │
│  │ Dev Servers    ██ 88%  ██ 92%  ░░ N/A   ░░ N/A   ██ 91%   │   │
│  │ Workstations   ██ 97%  ██ 98%  ██ 85%  ██ 100%  ██ 94%   │   │
│  │ Legacy Systems ░░ 72%  ░░ 78%  ░░ 65%  ░░ 80%   ░░ 70%   │   │
│  └───────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ┌──────────────────────────┐ ┌──────────────────────────────┐  │
│  │ SLA DEADLINE TRACKER      │ │ COMPLIANCE TREND (90 DAYS)   │  │
│  │                           │ │                              │  │
│  │ ⚠ 5 CVEs due in <3 days  │ │  100%─┐     ╱─────╲╱──      │  │
│  │ ⚠ 12 CVEs due in <7 days │ │   90%─┤────╱───────────      │  │
│  │ ● 28 CVEs due in <30 days│ │   80%─┤───╱─ ─ ─ ─ ─ ─      │  │
│  │                           │ │       └───┬───┬───┬───┬──    │  │
│  │ [View details →]          │ │        Dec  Jan  Feb  Mar    │  │
│  └──────────────────────────┘ └──────────────────────────────┘  │
│                                                                  │
│  ┌───────────────────────────────────────────────────────────┐   │
│  │ ACTIVE EXCEPTIONS (4)                                      │   │
│  │                                                            │   │
│  │ EXC-0042  CVE-2026-1234  legacy-servers  Expires Mar 15  │   │
│  │ EXC-0051  CVE-2026-5678  embedded-devs   Expires Apr 30  │   │
│  │ EXC-0053  CVE-2026-7890  db-cluster      Expires Mar 1   │   │
│  │ EXC-0055  CVE-2026-2345  iot-devices     Expires May 15  │   │
│  └───────────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────────┘
```

### 7.2 Drill-Down Views

| View | What It Shows |
|------|--------------|
| **Framework detail** | Click a framework card → all controls, per-control compliance %, evidence status |
| **Control detail** | Click a control → all CVEs evaluated against this control, per-CVE status, SLA timeline visualization |
| **Endpoint compliance** | Click an endpoint → all applicable CVEs, compliance state for each, remediation timeline |
| **CVE compliance** | Click a CVE → all affected endpoints, which are compliant/non-compliant, SLA deadline, deployment status |
| **Exception detail** | Click an exception → full justification, compensating controls, approval chain, expiry countdown |

---

## 8. Integration Points

### 8.1 Depends On (From Existing Architecture)

| Component | What Compliance Uses |
|-----------|---------------------|
| **CVE Feed** (Hub) | CVE IDs, CVSS scores, publish dates — starts the SLA clock |
| **Endpoint Inventory** (Agent) | Installed packages + versions — determines which CVEs apply |
| **Deployment History** (Engine) | When patches were deployed, to which endpoints — proves remediation timing |
| **Audit Trail** (Event System) | Who approved, when, what workflow was used — proves process adherence |
| **Scan Scheduler** (Agent) | Scan execution timestamps — proves scanning frequency |
| **RBAC** | Controls who can create exceptions, who can approve them, who can view reports |
| **Config Hierarchy** | Framework assignments per group, SLA overrides per tenant |
| **License Gating** | Compliance reports are gated by license tier |

### 8.2 External Integrations

| Integration | Direction | Purpose |
|-------------|-----------|---------|
| **GRC Platforms** (ServiceNow GRC, Archer, ZenGRC) | Export | Push compliance data into the organization's GRC tool via API |
| **SIEM** (Splunk, Elastic, Sentinel) | Export | Forward compliance events and violations for security correlation |
| **Ticketing** (Jira, ServiceNow ITSM) | Bidirectional | Auto-create tickets for non-compliant CVEs; update compliance status when ticket is resolved |
| **Vulnerability Scanners** (Qualys, Tenable, Rapid7) | Import | Ingest vulnerability scan results to enrich PatchIQ's CVE matching (covers apps PatchIQ doesn't scan natively) |
| **Email/Slack/Webhook** | Export | Compliance status alerts, SLA deadline warnings, exception expiry notifications |

---

## 9. Compliance by License Tier

| Capability | Community | Professional | Enterprise | MSP |
|-----------|-----------|-------------|-----------|-----|
| Basic CVE matching | Yes | Yes | Yes | Yes |
| Compliance dashboard | CVE counts only | 1 framework | All frameworks | All + per-tenant |
| SLA tracking | No | Yes | Yes | Yes |
| Exception management | No | Basic (no approval workflow) | Full workflow | Full + cross-tenant |
| Evidence reports | No | Summary only | Full audit-ready reports | Full + white-label |
| Custom frameworks | No | No | Yes | Yes |
| Scheduled reports | No | Monthly | Any frequency | Any + per-tenant |
| GRC integration | No | No | Yes | Yes |
| Report export (PDF/CSV) | No | CSV only | All formats | All formats |

---

## 10. Implementation Phases

### Phase 1 (Months 1-4): Foundation

- CVE feed ingestion from NVD (basic)
- CVE-to-inventory matching (which endpoints are affected)
- Simple vulnerability list in the UI (no framework mapping yet)
- Audit trail captures all deployment events (no compliance evaluation yet)

### Phase 3 (Months 9-12): Full Compliance Engine

- Framework profile registry with NIST, SOC 2, HIPAA, PCI DSS, ISO 27001, CIS
- SLA deadline tracking with configurable timelines
- Compliance evaluator (per-endpoint, per-group scoring)
- Exception management with approval workflow
- Full evidence report generation (PDF, CSV, JSON)
- Compliance dashboard with heatmap, trends, drill-down
- Scan frequency monitoring and reporting
- Process adherence checks (was approval/testing/rollback used?)
- Custom framework support
- Scheduled report delivery

### Phase 4 (Months 13-16): Advanced

- DISA STIG profiles
- Cyber Essentials profile
- GRC platform integration (ServiceNow, Archer)
- Vulnerability scanner import (Qualys, Tenable)
- Predictive compliance (forecast SLA breaches before they happen)
- Compliance-driven auto-prioritization (automatically prioritize patches that affect compliance score)

---

## 11. Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Ship framework profiles with the product | Pre-built JSON profiles, versioned with releases | Customers shouldn't have to interpret framework documents; we've done the mapping for them |
| Support custom frameworks | Admin-defined profiles using the same schema | Organizations often have internal policies stricter than external frameworks |
| Exception management with approval workflow | Structured exceptions, not just "snooze" | Auditors require documented justifications and compensating controls, not arbitrary dismissals |
| Compliance scoring uses multiple methods | Strictest/Average/Worst-case configurable per framework | Different frameworks have different audit philosophies; one scoring model doesn't fit all |
| Evidence reports are generated, not live views | Point-in-time PDF/CSV snapshots | Auditors need a fixed artifact they can file, not a changing dashboard |
| Phase 1 captures audit trail even without compliance engine | Events from day 1, compliance evaluation from Phase 3 | The data must exist before you can report on it; retroactive audit trails are worthless |

---

## Code Mapping

| Area | Code Directory |
|------|---------------|
| Compliance engine backend | `internal/server/compliance/` |
| Framework profiles (shipped) | `internal/server/compliance/frameworks/` |
| Compliance dashboard UI | `web/src/pages/compliance/` |
| Exception management UI | `web/src/pages/compliance/exceptions/` |
