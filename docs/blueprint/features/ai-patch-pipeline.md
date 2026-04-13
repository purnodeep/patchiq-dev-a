# Feature: AI-Powered Automated Patch Pipeline

> Status: Proposed | Phase: 2-3 | License: Premium Add-on (Managed Patch Intelligence)

## Overview

An AI-driven pipeline operated via the Hub Manager that automatically discovers, downloads, analyzes, and tests patches from vendor advisories — reducing the time between CVE disclosure and verified patch availability to hours instead of days.

## Problem Statement

Traditional patch catalog maintenance is manual and slow:
- Vendor releases a patch → Catalog team manually creates an install definition → Tests → Publishes
- This cycle can take days to weeks for less-common applications
- Customers are exposed to known vulnerabilities during this gap
- Maintaining 800+ application definitions is a significant operational burden

## Solution

Automate the patch-to-catalog pipeline using AI and structured vendor feeds, with human oversight as the quality gate.

### Pipeline Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        HUB MANAGER                              │
│                                                                 │
│  ┌──────────┐    ┌──────────────┐    ┌───────────────────────┐  │
│  │ CVE Feed │───>│ AI Crawler / │───>│ Installer Analyzer    │  │
│  │ Monitor  │    │ Downloader   │    │ (type detection,      │  │
│  └──────────┘    └──────────────┘    │  silent switch        │  │
│       │                              │  inference)           │  │
│       │          ┌───────────────┐   └───────────┬───────────┘  │
│       │          │ Install Def   │<──────────────┘              │
│       │          │ Generator     │                              │
│       │          └───────┬───────┘                              │
│       │                  │                                      │
│       │          ┌───────▼───────┐                              │
│       │          │ Sandbox Test  │  (VM per OS: Win/Linux/Mac)  │
│       │          │ Environment   │                              │
│       │          └───────┬───────┘                              │
│       │                  │                                      │
│       │          ┌───────▼───────┐                              │
│       │          │ Verification  │  (version check, service     │
│       │          │ Engine        │   health, no regressions)    │
│       │          └───────┬───────┘                              │
│       │                  │                                      │
│       │          ┌───────▼───────┐                              │
│       │          │ Human Review  │  (PatchIQ team approves)     │
│       │          │ Queue         │                              │
│       │          └───────┬───────┘                              │
│       │                  │                                      │
│       │          ┌───────▼───────┐                              │
│       │          │ Catalog       │  (published to all           │
│       │          │ Publish       │   subscribed clients)        │
│       │          └───────────────┘                              │
└─────────────────────────────────────────────────────────────────┘
```

### Pipeline Stages

#### Stage 1: CVE Feed Monitoring

Monitor structured vulnerability feeds in real-time:

| Source | Type | Coverage |
|--------|------|----------|
| NVD (NIST) | REST API | All CVEs, vendor-neutral |
| Microsoft MSRC | API (CVRF/CSAF) | Windows, Office, .NET, Edge |
| Red Hat OVAL | Structured XML | RHEL, CentOS, Fedora |
| Ubuntu USN | Structured feed | Ubuntu, Debian-based |
| Apple Security Updates | Structured page | macOS, iOS |
| Vendor RSS/Atom feeds | Semi-structured | Third-party applications |

Each CVE entry contains:
- Affected product + versions
- Severity (CVSS score)
- Vendor advisory URL (link to patch info)

#### Stage 2: AI Crawler & Downloader

For each CVE with a vendor advisory link:

1. **Structured feeds** (Microsoft, Red Hat, Ubuntu, Apple): Parse the structured data directly to extract download URLs. No AI needed — deterministic parsing.

2. **Semi-structured pages** (third-party vendors): Use an LLM to:
   - Parse the vendor advisory page
   - Identify the correct download link for the patch
   - Extract version information
   - Determine supported platforms/architectures
   - Extract any documented install instructions

3. **Download** the patch binary and store in Hub MinIO with metadata.

#### Stage 3: Installer Analysis

Determine installer type and generate the installation definition:

| Installer Type | Detection | Silent Switches | Confidence |
|----------------|-----------|-----------------|------------|
| MSI (.msi) | File magic / extension | `msiexec /i <file> /qn /norestart` | High |
| DEB (.deb) | File magic / extension | `dpkg -i <file>` or `apt install ./<file>` | High |
| RPM (.rpm) | File magic / extension | `rpm -U <file>` or `dnf install <file>` | High |
| PKG (.pkg) | File magic / extension | `installer -pkg <file> -target /` | High |
| MSIX (.msix) | File magic / extension | `Add-AppPackage -Path <file>` | High |
| EXE (NSIS) | PE header analysis | `/S` | Medium |
| EXE (Inno Setup) | PE header analysis | `/VERYSILENT /NORESTART` | Medium |
| EXE (InstallShield) | PE header analysis | `/s /v"/qn"` | Medium |
| EXE (Unknown) | AI analyzes vendor docs | Inferred from documentation | Low |

For **Low confidence** installers, the AI:
- Searches vendor documentation for silent install instructions
- Tries common switch patterns in sandbox
- Flags for human review if no working combination found

#### Stage 4: Sandbox Testing

Spin up disposable VMs (one per target OS) and execute the full install flow:

1. **Pre-state capture**: Snapshot installed software, running services, file system state
2. **Execute install**: Run the generated install definition
3. **Verify success**:
   - Exit code is success (0 or expected reboot code)
   - Application version matches expected version
   - No crash logs generated
   - Key services still running (no regressions)
   - Uninstall command works (rollback verification)
4. **Post-state diff**: Compare against pre-state to detect unexpected changes

**Sandbox infrastructure**: Pre-baked VM templates per OS version, managed via libvirt/QEMU or cloud VMs. Reset to snapshot after each test.

#### Stage 5: Human Review & Publish

- AI-generated definitions that pass sandbox testing go to a review queue
- PatchIQ catalog team reviews: install definition, test results, diff report
- Approved patches are published to the catalog
- Subscribed client Patch Managers pull the new catalog entry
- Client admins receive notification of new patch availability

### Client-Side Safety (Double Safety Net)

Even after Hub-side testing, the client's existing deployment safety mechanisms apply:

```
Hub tests patch (sandbox) → Publishes to catalog
    → Client Patch Manager pulls definition
    → Client's canary/test group (Wave 0) installs
    → Monitoring period (configurable, e.g., 24-48 hours)
    → If clean → Roll to production waves (Wave 1, 2, 3...)
    → If issues → Auto-rollback, alert admin
```

### Automation Tiers

| Tier | Behavior | Use Case |
|------|----------|----------|
| **Notify Only** | Hub tests and publishes; client admin manually approves deployment | Security-conscious orgs |
| **Auto-Deploy to Canary** | Automatically deploys to test group; admin approves production rollout | Balanced approach |
| **Full Auto** | Auto-deploys through all waves with configurable wait periods | Orgs prioritizing speed |

Client chooses their automation tier per severity level (e.g., Full Auto for Critical CVEs, Notify Only for Low).

### SLA Impact

| Metric | Without Feature | With Feature |
|--------|----------------|--------------|
| CVE-to-catalog time (structured vendors) | 1-3 days | 2-6 hours |
| CVE-to-catalog time (third-party) | 3-7 days | 6-24 hours |
| Catalog coverage expansion | Manual effort | Continuous, AI-assisted |
| Zero-day response | Manual scramble | Automated pipeline with priority queue |

### Licensing

This feature is offered as **"Managed Patch Intelligence"** — a premium add-on:

- Included in Enterprise and MSP tiers
- Available as paid add-on for Professional tier
- Community tier gets catalog updates on a delayed schedule (e.g., 7-day delay)

### Technical Considerations

- **Rate limiting**: Respect vendor download rate limits and terms of service
- **Binary verification**: Verify downloaded patches against vendor-published checksums/signatures
- **Reproducibility**: All AI-generated definitions are version-controlled and auditable
- **Feedback loop**: If a client reports an issue with an auto-generated definition, it feeds back into the AI model and flags similar definitions for re-review
- **Vendor API keys**: Some vendors (Microsoft MSRC, Red Hat) require API keys — managed at Hub level

---

## Code Mapping

| Area | Code Directory |
|------|---------------|
| CVE feed monitor | `internal/hub/feeds/` |
| AI crawler/downloader | `internal/hub/pipeline/` |
| Installer analyzer | `internal/hub/pipeline/analyzer/` |
| Sandbox orchestration | `internal/hub/pipeline/sandbox/` |
| Catalog publish | `internal/hub/catalog/` |
