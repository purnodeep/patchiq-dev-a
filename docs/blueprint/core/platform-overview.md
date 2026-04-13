# PatchIQ — Platform Overview

> The three interconnected platforms that make up PatchIQ.

---

## The Three Platforms

PatchIQ is composed of three distinct but interconnected products:

```
┌─────────────────────────────────────────────────────────────────────┐
│                        PatchIQ Hub Manager                          │
│              (Operated by PatchIQ / MSPs)                           │
│   Software catalog, patch feeds, license management, analytics      │
└──────────────────────────────┬──────────────────────────────────────┘
                               │ Secure API (patch metadata, catalog sync)
                               │
┌──────────────────────────────┴──────────────────────────────────────┐
│                       PatchIQ Patch Manager                         │
│              (Deployed at Client Sites)                             │
│   Policies, deployments, compliance, RBAC, workflows, AI assistant  │
└───────────────────────┬─────────────────────────────────────────────┘
                        │ gRPC + mTLS
          ┌─────────────┼─────────────┐
          │             │             │
     ┌────┴────┐  ┌────┴────┐  ┌────┴────┐
     │ PatchIQ │  │ PatchIQ │  │ PatchIQ │
     │ Agent   │  │ Agent   │  │ Agent   │
     │ (Win)   │  │ (Linux) │  │ (macOS) │
     └─────────┘  └─────────┘  └─────────┘
```

---

## 1. PatchIQ Agent

**What it is:** A lightweight Go binary that lives on every managed endpoint.

**Key characteristics:**
- Single binary, < 30MB disk, < 50MB RAM at idle, < 1% CPU
- Lightweight local UI (system tray on Windows/macOS, web UI on Linux) for endpoint-level monitoring
- Communicates with Patch Manager via gRPC + mTLS
- Offline-resilient: queues inventory and patch results in local SQLite, syncs on reconnect
- Self-updating: checks for new agent versions and updates without human intervention
- OS-specific patch detection: Windows Update API, APT/YUM/DNF, softwareupdate/Homebrew

**Agent local UI shows:**
- Agent health status and connection state
- Pending patches and scheduled maintenance windows
- Last scan results and installed packages
- Patch installation history with logs
- Manual scan trigger and reboot scheduling

---

## 2. PatchIQ Patch Manager

**What it is:** The main management console deployed at the client's infrastructure. This is where admins live day-to-day.

**Key responsibilities:**
- Visual workflow builder for policies and deployments (React Flow)
- Endpoint inventory and group management
- Patch discovery, CVE correlation, risk scoring
- Deployment orchestration with waves/rings/rollback
- Compliance reporting (HIPAA, SOC2, ISO 27001, FedRAMP)
- Approval workflows with ITSM integration
- Custom RBAC with granular permissions
- AI assistant (MCP-powered) for natural language operations
- Audit trail for every action
- Notifications (email, Slack, webhook, PagerDuty)

---

## 3. PatchIQ Hub Manager

**What it is:** The central platform operated by PatchIQ (us) or by MSPs. This is the upstream intelligence layer.

**Key responsibilities:**
- **Software Catalog Management**: Maintain a curated catalog of patches across all OS vendors, third-party apps, and custom packages
- **Patch Feed Aggregation**: Pull from NVD, CISA KEV, vendor-specific feeds (Microsoft, Canonical, Red Hat, Apple), and normalize into a unified schema
- **License Management**: Generate, validate, and manage client licenses; feature gating; usage metering
- **Client Analytics**: Aggregated (anonymized) telemetry from consenting clients for product improvement
- **Update Distribution**: Serve agent binaries, Patch Manager updates, and patch metadata to client instances
- **MSP Portal**: Multi-tenant management view for MSPs managing multiple client deployments
- **Support Integration**: Client health monitoring, support ticket routing, remote diagnostics

**Hub-to-Patch Manager sync:**
- Patch metadata (new patches, CVE mappings, severity updates) synced on schedule or on-demand
- Agent binary updates published through release channels (stable, beta, canary)
- License validation (online check with offline grace period)
- Optional: anonymized telemetry upload (opt-in, configurable)

---

## Code Mapping

| Platform | Backend | Frontend |
|----------|---------|----------|
| Agent | `internal/agent/` | `web-agent/` |
| Patch Manager | `internal/server/` | `web/` |
| Hub Manager | `internal/hub/` | `web-hub/` |
