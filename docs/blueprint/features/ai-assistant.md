# Feature: AI Assistant (MCP Integration)

> Status: In Progress | Phase: 2 | License: Professional (read-only), Enterprise/MSP (full)

---

## Overview

An LLM-powered assistant embedded in the Patch Manager UI that operates the platform through natural language. The user tells the assistant what they want done, and the assistant executes actions via MCP tools — with human-in-the-loop confirmation for destructive operations.

## Problem Statement

Enterprise patch management involves complex multi-step operations (find affected endpoints, create policies, schedule deployments, verify compliance). Admins currently navigate multiple screens and fill out forms. A conversational interface reduces this to a single natural-language request.

---

## Architecture

```
┌──────────────────────────────────────────┐
│           Patch Manager Frontend          │
│  ┌─────────────────────────────────┐     │
│  │       AI Chat Panel             │     │
│  │  "Deploy critical patches to    │     │
│  │   all prod Linux servers this   │     │
│  │   weekend"                      │     │
│  └──────────────┬──────────────────┘     │
└─────────────────┼────────────────────────┘
                  │ WebSocket / SSE
┌─────────────────┼────────────────────────┐
│          AI Backend Service              │
│  ┌──────────────┴──────────────────┐     │
│  │       MCP Server (Go/TS)        │     │
│  │                                 │     │
│  │  Tools:                         │     │
│  │  - list_endpoints               │     │
│  │  - get_endpoint_detail          │     │
│  │  - list_patches                 │     │
│  │  - create_policy [destructive]  │     │
│  │  - create_deployment [destruct] │     │
│  │  - get_compliance_report        │     │
│  │  - search_cves                  │     │
│  │  - approve_deployment [destruct]│     │
│  │  - run_scan                     │     │
│  │  - create_group                 │     │
│  │  - get_audit_log                │     │
│  │                                 │     │
│  │  Resources:                     │     │
│  │  - endpoint://                  │     │
│  │  - policy://                    │     │
│  │  - deployment://                │     │
│  │  - compliance://                │     │
│  └─────────────────────────────────┘     │
│                  │                        │
│           Anthropic Claude API            │
└──────────────────────────────────────────┘
```

---

## MCP Tool Design

Each tool exposes PatchIQ functionality to the LLM with proper annotations:

| Tool | readOnlyHint | destructiveHint | Description |
|------|-------------|----------------|-------------|
| `list_endpoints` | true | false | Search/filter endpoints by OS, group, tags, compliance status |
| `get_endpoint_detail` | true | false | Full detail of a specific endpoint |
| `list_available_patches` | true | false | Patches available for deployment |
| `search_cves` | true | false | Search CVEs by ID, severity, affected package |
| `get_compliance_report` | true | false | Generate compliance summary for a scope |
| `get_audit_log` | true | false | Query audit trail |
| `create_policy` | false | false | Create a new patch policy (requires user confirmation) |
| `create_deployment` | false | true | Deploy patches to endpoints (requires user approval) |
| `approve_deployment` | false | true | Approve a pending deployment (requires user approval) |
| `cancel_deployment` | false | true | Cancel an in-progress deployment |
| `trigger_scan` | false | false | Trigger an inventory scan on endpoints |
| `create_group` | false | false | Create an endpoint group |
| `modify_policy` | false | false | Update an existing policy |

---

## Human-in-the-Loop Pattern

The AI assistant follows a strict confirmation model:

1. **Read-only actions**: Execute immediately, display results
2. **Non-destructive writes** (create group, trigger scan): Show what will happen, ask "Should I proceed?"
3. **Destructive actions** (deploy patches, cancel deployment): Show detailed preview with affected endpoints count, require explicit "Yes, deploy" confirmation
4. **Sensitive actions** (modify RBAC, change compliance settings): Require authentication re-confirmation + explicit approval

**Example conversation:**

```
User: "Patch all critical CVEs on production Linux servers this weekend"

AI: I found 3 critical CVEs affecting 247 production Linux servers:
    - CVE-2025-1234 (OpenSSL 3.x) — 247 servers
    - CVE-2025-5678 (kernel) — 183 servers
    - CVE-2025-9012 (glibc) — 92 servers

    I'll create a deployment with:
    - Maintenance window: Saturday 2am–6am
    - Strategy: 10% canary → 4h wait → remaining 90%
    - Auto-rollback if >5% failure rate

    [Create Deployment] [Modify] [Cancel]

User: clicks [Create Deployment]

AI: Deployment DEP-2025-0142 created. 247 endpoints scheduled
    for Saturday 02:00. You'll be notified on progress.
```

---

## MCP Implementation

- **Transport**: HTTP with SSE (Streamable HTTP) for the MCP server, embedded in the Patch Manager backend
- **Language**: TypeScript MCP server (using official `@modelcontextprotocol/sdk`) wrapping Go API calls
- **LLM**: Claude API (latest Sonnet for speed, Opus for complex reasoning)
- **Context**: MCP Resources provide the LLM with current state (endpoint counts, pending deployments, compliance scores) without needing to call tools first
- **Session**: Each user gets an isolated MCP session with their RBAC permissions applied to all tool calls

---

## Integration Points

| Feature | Integration |
|---------|-------------|
| **RBAC** | AI tool calls are scoped to the user's permissions |
| **Audit Trail** | All AI-initiated actions are logged with `actor: ai_assistant, confirmed_by: <user>` |
| **Compliance** | AI can generate compliance reports and explain compliance status |
| **Deployment Engine** | AI creates deployments through the same API as the UI |

---

## License Gating

| Tier | AI Capability |
|------|--------------|
| Community | No AI assistant |
| Professional | Read-only (list, search, report) |
| Enterprise | Full (read + write + destructive with confirmation) |
| MSP | Full + cross-tenant queries |

---

## Code Mapping

| Area | Code Directory |
|------|---------------|
| MCP server | `internal/server/mcp/` |
| AI chat panel (frontend) | `web/src/ai/` |
