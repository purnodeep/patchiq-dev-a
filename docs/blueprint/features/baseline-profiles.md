# Feature: Baseline Profiles (Desired State Management)

> Status: Proposed | Phase: 2-3 | License: Professional and above

## Overview

Capture the complete software state of a "golden" endpoint as a reusable profile, then enforce that profile across any number of target endpoints — automatically installing, upgrading, or flagging differences to bring them into alignment.

## Problem Statement

Enterprises struggle with endpoint consistency:
- New employee onboarding requires IT to manually install 15-30 applications
- Lab/classroom environments drift from their intended configuration over time
- Compliance audits require proof that all endpoints in a group match an approved baseline
- After incidents, rebuilding endpoints to a known-good state is slow and error-prone

## Solution

**Baseline Profiles** — a declarative model where admins define what an endpoint should look like, and the agent enforces it.

### Core Concept

```
┌──────────────────────┐         ┌──────────────────────┐
│   Golden Endpoint    │         │   Baseline Profile   │
│                      │ Capture │                      │
│  OS: Ubuntu 24.04    │────────>│  OS: Ubuntu 24.04    │
│  Patch Level: 2026-02│         │  Patch Level: 2026-02│
│  Chrome: 130.0       │         │  Chrome: 130.0       │
│  VS Code: 1.95       │         │  VS Code: 1.95       │
│  Docker: 27.1        │         │  Docker: 27.1        │
│  Node: 22.1          │         │  Node: 22.1          │
│  ...                 │         │  ...                 │
└──────────────────────┘         └──────────┬───────────┘
                                            │
                                            │ Enforce
                                            │
                    ┌───────────────────────┬┴──────────────────────┐
                    │                       │                       │
                    ▼                       ▼                       ▼
          ┌─────────────────┐   ┌─────────────────┐   ┌─────────────────┐
          │  Target EP #1   │   │  Target EP #2   │   │  Target EP #3   │
          │                 │   │                 │   │                 │
          │  Chrome: 128    │   │  Chrome: 130 ✓  │   │  Chrome: N/A    │
          │  → upgrade 130  │   │  → skip         │   │  → install 130  │
          │                 │   │                 │   │                 │
          │  VS Code: 1.95 ✓│   │  VS Code: 1.90  │   │  VS Code: 1.95 ✓│
          │  → skip         │   │  → upgrade 1.95 │   │  → skip         │
          └─────────────────┘   └─────────────────┘   └─────────────────┘
```

### Profile Structure

A Baseline Profile is a JSON document stored in the Patch Manager database:

```json
{
  "profile_id": "prof_abc123",
  "name": "Engineering Workstation v3",
  "description": "Standard setup for engineering department",
  "created_from": "agent_id_golden_endpoint",
  "created_at": "2026-02-26T10:00:00Z",
  "created_by": "admin@company.com",
  "os": {
    "family": "linux",
    "distribution": "ubuntu",
    "version": "24.04",
    "patch_level": "2026-02-15",
    "enforcement": "warn"
  },
  "applications": [
    {
      "catalog_id": "app_chrome",
      "name": "Google Chrome",
      "version": "130.0.6723.91",
      "enforcement": "exact",
      "required": true
    },
    {
      "catalog_id": "app_vscode",
      "name": "Visual Studio Code",
      "version": "1.95.0",
      "enforcement": "minimum",
      "required": true
    },
    {
      "catalog_id": "app_docker",
      "name": "Docker Engine",
      "version": "27.1.0",
      "enforcement": "minimum",
      "required": true
    },
    {
      "catalog_id": "app_slack",
      "name": "Slack",
      "version": "4.40.0",
      "enforcement": "minimum",
      "required": false
    }
  ],
  "deny_list": [
    {
      "catalog_id": "app_torrent_client",
      "name": "BitTorrent",
      "action": "warn"
    }
  ]
}
```

### Profile Fields Explained

#### OS Section

| Field | Description |
|-------|-------------|
| `family` | linux, windows, darwin |
| `distribution` | ubuntu, rhel, windows-11, macos-ventura, etc. |
| `version` | Major OS version |
| `patch_level` | Date or KB/patch identifier for OS patch baseline |
| `enforcement` | `enforce` = block if mismatch, `warn` = alert only |

#### Application Entries

| Field | Description |
|-------|-------------|
| `catalog_id` | Reference to Hub catalog application |
| `version` | Target version |
| `enforcement` | Version matching strategy (see below) |
| `required` | `true` = must be installed, `false` = install if missing but don't enforce |

#### Version Enforcement Modes

| Mode | Behavior |
|------|----------|
| `exact` | Must be exactly this version. Downgrades if newer. Use for apps where specific versions are certified. |
| `minimum` | This version or newer. Upgrades if older, leaves alone if newer. Most common mode. |
| `range` | Within a version range (e.g., `>=22.0 <23.0`). For major-version-locked apps. |
| `latest` | Always upgrade to the latest version in the catalog. |
| `pinned` | Install this version, never auto-update. For compliance-frozen apps. |

#### Deny List

Applications that should NOT be on endpoints matching this profile. Actions:
- `warn` — Alert admin, don't remove
- `block` — Prevent the app from running (if agent supports application control)
- `remove` — Automatically uninstall (requires explicit admin opt-in)

### Workflow

#### 1. Profile Creation

Three methods:

**a) Capture from Golden Endpoint**
```
Admin selects a "golden" endpoint in the UI
    → Agent runs full inventory scan
    → Returns installed OS + all applications with versions
    → Admin reviews and curates the list (remove personal apps, etc.)
    → Saves as a named Baseline Profile
```

**b) Manual Composition**
```
Admin creates a new profile in the UI
    → Searches catalog for applications
    → Adds applications with desired versions and enforcement modes
    → Sets OS requirements
    → Saves profile
```

**c) Import / Clone**
```
Admin imports a profile from another Patch Manager instance
    → Or clones an existing profile and modifies it
    → Useful for multi-site consistency
```

#### 2. Profile Assignment

Profiles are assigned to **endpoint groups** (not individual endpoints):

- "All Engineering Laptops" → Engineering Workstation v3
- "Finance Desktops" → Finance Desktop v2
- "Lab Room 101" → Lab Standard v1

An endpoint can have **one active profile**. If an endpoint moves groups, its profile changes.

#### 3. Compliance Check (Drift Detection)

The agent periodically compares its current state against the assigned profile:

```
Agent scans local inventory
    → Compares each application against profile
    → Generates a compliance report:

    Profile: Engineering Workstation v3
    Status: NON-COMPLIANT (3 issues)

    ✓ Chrome 130.0.6723.91    — matches (exact)
    ✗ VS Code 1.90.0          — below minimum 1.95.0 (upgrade needed)
    ✗ Docker: not installed    — required, missing (install needed)
    ✗ BitTorrent detected      — on deny list (warn)
    ✓ Node 22.3.0             — above minimum 22.1.0 (ok)
    ✓ Slack 4.41.0            — above minimum 4.40.0 (ok)
```

#### 4. Enforcement / Remediation

Based on admin configuration, the system can:

| Mode | Behavior |
|------|----------|
| **Monitor Only** | Report drift, take no action. For audit/visibility. |
| **Notify & Recommend** | Report drift + generate a remediation plan for admin approval. |
| **Auto-Remediate** | Automatically install/upgrade to match profile. Uses wave deployment for safety. |

Auto-remediation follows the same wave/canary deployment as regular patching — not a bulk push.

#### 5. Continuous Monitoring

```
┌─────────────────────────────────────────────────┐
│              Compliance Dashboard                │
│                                                  │
│  Profile: Engineering Workstation v3             │
│  Assigned: 342 endpoints                        │
│                                                  │
│  ██████████████████████████░░░░  87% compliant   │
│                                                  │
│  Compliant:     298 endpoints                    │
│  Drifted:        31 endpoints (auto-remediating) │
│  Non-compliant:  13 endpoints (needs attention)  │
│                                                  │
│  Top Issues:                                     │
│  - VS Code outdated (22 endpoints)               │
│  - Docker missing (8 endpoints)                  │
│  - Unauthorized software detected (5 endpoints)  │
└─────────────────────────────────────────────────┘
```

### Handling Edge Cases

| Scenario | Behavior |
|----------|----------|
| **Target has no OS** | Out of scope. Profile requires a running OS with an agent. Admin must image the machine first. |
| **Target has different OS version** | If `enforcement: warn` — report mismatch. If `enforcement: enforce` — block profile application and alert admin. OS upgrades are not auto-applied (too risky). |
| **Target has different OS family** | Profile is OS-specific. A Linux profile cannot be applied to a Windows endpoint. System rejects the assignment. |
| **Target has newer app version** | Depends on enforcement mode. `minimum` — no action. `exact` — downgrade. `pinned` — downgrade. `latest` — no action. |
| **App not in catalog** | Cannot enforce apps that aren't in the Hub catalog. Flag as "unmanaged" in the profile. |
| **Endpoint offline** | Profile is cached locally by the agent. Enforcement runs on next check-in. Compliance status shows "last checked: X hours ago." |
| **Conflicting profiles** | One profile per endpoint. If group membership overlaps, admin must resolve the conflict in the UI. |

### Profile Versioning

Profiles are versioned. When an admin updates a profile:

```
Engineering Workstation v3.1
  Changed: VS Code 1.95 → 1.96
  Changed: Docker 27.1 → 27.2
  Added: kubectl 1.31
```

- Previous version is archived (not deleted)
- All assigned endpoints gradually transition to the new version via wave deployment
- Admin can roll back to a previous profile version if issues arise

### Integration with Other Features

| Feature | Integration |
|---------|-------------|
| **Wave Deployment** | Profile changes roll out through waves, not all-at-once |
| **AI Patch Pipeline** | When the AI pipeline publishes a new version, profiles using `latest` enforcement auto-update |
| **Compliance Reporting** | Profile compliance feeds into overall compliance dashboards and reports |
| **RBAC** | Profile creation/assignment restricted by role and scope |
| **Audit Log** | All profile changes, assignments, and enforcements are logged |

### Use Cases

1. **Employee Onboarding**: IT assigns the department profile to the new laptop. Agent installs everything automatically. Employee is productive in minutes, not hours.

2. **Compliance Baselines**: Security team defines a hardened baseline. All endpoints in scope are continuously monitored. Audit reports show compliance percentage over time.

3. **Lab/Classroom Reset**: After each semester/session, re-enforce the lab profile. All machines return to the standard state without reimaging.

4. **Incident Recovery**: If an endpoint is compromised, wipe and reinstall OS, then apply the baseline profile. Agent rebuilds the software stack automatically.

5. **Multi-Site Consistency**: HQ defines profiles, all branch offices enforce the same standards. Distribution servers ensure fast local installs.

### Licensing

| Tier | Capability |
|------|------------|
| Community | 1 profile, monitor-only, up to 25 endpoints |
| Professional | 10 profiles, notify & recommend, unlimited endpoints |
| Enterprise | Unlimited profiles, full auto-remediation, profile versioning |
| MSP | Per-tenant profiles, cross-tenant templates, bulk assignment |

---

## Code Mapping

| Area | Code Directory |
|------|---------------|
| Baseline engine backend | `internal/server/baseline/` |
| Profile management UI | `web/src/pages/baselines/` |
| Agent drift detection | `internal/agent/baseline/` |
