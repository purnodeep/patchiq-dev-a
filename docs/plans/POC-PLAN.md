# PatchIQ — POC Readiness Plan

> **Goal**: Take PatchIQ from "features exist" to "everything works E2E" for client POC deployment.
>
> **POC definition**: Deploy on client infrastructure, client uses for 2 weeks, zero critical bugs.
>
> **Demo definition**: Scripted walkthrough of 7 core flows, everything polished on the happy path.
>
> **Created**: 2026-03-30 | **Branch**: dev-b

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Current State Audit](#2-current-state-audit)
3. [E2E Flow Verification & Fixes](#3-e2e-flow-verification--fixes)
4. [Critical Missing Pieces](#4-critical-missing-pieces)
5. [Platform Polish](#5-platform-polish)
6. [Windows Agent Completion](#6-windows-agent-completion)
7. [Deployment Packaging](#7-deployment-packaging)
8. [QA & Hardening](#8-qa--hardening)
9. [Work Item Index](#9-work-item-index)

---

## 1. Executive Summary

### What We Have

PatchIQ is a 3-tier enterprise patch management platform (Hub → Server → Agent) with:
- **607 Go files**, **6,000 TypeScript files**, **45 server migrations**, **11 hub migrations**
- All backend handlers implemented (30+ handler files, 72 API endpoints)
- All 13 background workers wired and running
- gRPC bidirectional sync operational
- Deployment state machine with waves, rollback, timeout handling
- 6 vulnerability feeds aggregating real data
- 6 compliance frameworks with real evaluator
- 3 frontend apps with 51 routes total
- Design system with light/dark mode

### What's Wrong

Nothing works perfectly end-to-end. Individual components exist but the integration between them has gaps. Specific problems:

- **Settings page**: IAM card is cosmetic (read-only with fake save), integrations card is read-only, license card is read-only. Only general settings actually save.
- **Compliance exports**: UI buttons exist, zero backend implementation.
- **Hub client/license pages**: 8 TODOs with placeholder chart data.
- **Hub auth**: Still using mock authentication (PIQ-12).
- **No Alerts/Events page**: Client has no unified view of system activity.
- **Workflow compliance node**: Stub that always returns "pass."
- **Windows agent**: WUA collector exists but isn't registered; WUA patcher doesn't exist.
- **No production deployment packaging**: Only dev docker-compose exists.

### Target State

Every feature either works flawlessly or is explicitly removed from the UI. No fake buttons, no placeholder data, no stubs that silently pass. A client can use every visible feature for 2 weeks without hitting a white screen or a dead button.

---

## 2. Current State Audit

### 2.1 Server Backend — Status: SOLID

| Component | Status | Notes |
|-----------|--------|-------|
| API Handlers | **100%** | All CRUD operations complete, 30+ handler files |
| Init Sequence | **100%** | All services wired in correct order |
| Deployment Pipeline | **100%** | State machine, waves, rollback, timeout, scheduling |
| CVE Sync & Correlation | **100%** | NVD sync, patch correlation, endpoint matching |
| Compliance Evaluation | **100%** | 6 frameworks, custom frameworks, evaluator, scorer |
| Policy Auto-Deploy | **100%** | Scheduler + evaluator + schedule checker |
| Workflow Execution | **95%** | ComplianceCheckHandler is M2 stub (always passes) |
| Notifications | **100%** | Shoutrrr integration, 11 event subscriptions |
| Auth (Zitadel OIDC) | **100%** | SSO, direct login, invite, JWT middleware |
| Background Workers | **100%** | 13 workers registered, 8 periodic jobs |
| gRPC Sync | **100%** | Enroll, Heartbeat, SyncOutbox, SyncInbox |
| Event Bus | **100%** | Watermill + PostgreSQL, audit subscriber on wildcard |

**Known backend issues (non-critical):**
- Deleted patch sync incomplete (no soft-delete cleanup from Hub)
- Event emission not transactional (audit events can silently drop)
- TOCTOU race in workflow reads (minor concurrency issue)
- No metrics for dropped events (observability gap)

### 2.2 Hub Backend — Status: SOLID

| Component | Status | Notes |
|-----------|--------|-------|
| API Handlers | **100%** | 9 handlers, 30+ endpoints |
| Feed Sync | **100%** | All 6 feeds (NVD, CISA KEV, MSRC, RedHat, Ubuntu, Apple) |
| Catalog Pipeline | **100%** | Normalization, CVE linking, deduplication |
| Hub→Server Sync | **100%** | Delta sync with bearer token auth |
| License Management | **80%** | CRUD works, RSA signing uses placeholder keys |
| Binary Storage | **80%** | MinIO download works, upload endpoint missing |
| Client Management | **100%** | Register, approve, decline, suspend, delete |
| Workers | **100%** | 6 periodic feed sync jobs |

**Known hub issues:**
- Idempotency store is in-memory (should be Valkey)
- Settings uses tenant_id for UpdatedBy instead of user_id
- Binary upload endpoint not implemented (manual SQL needed)

### 2.3 Agent — Status: GOOD (Windows gaps)

| Component | Status | Notes |
|-----------|--------|-------|
| Linux Patching (APT/YUM/DNF) | **100%** | Fully functional |
| macOS Patching (Homebrew/softwareupdate) | **100%** | Fully functional |
| Windows Patching (MSI/MSIX) | **70%** | Works for MSI/MSIX only |
| Windows Patching (WUA) | **0%** | Collector exists but unregistered; installer missing |
| gRPC Communications | **100%** | Enrollment, heartbeat, outbox, inbox |
| Command Processing | **100%** | All 6 command types handled |
| HTTP API | **100%** | 12 endpoints for local UI |
| SQLite Store | **100%** | 6 tables with proper indexing |
| Dynamic Settings | **100%** | In-memory cache with 30s refresh |
| Rollback | **100%** | 3-mode support |
| Script Execution | **100%** | sh, PowerShell, Python3 |

### 2.4 Frontend — web/ (Patch Manager) — Status: FEATURE COMPLETE

| Page | List | Detail | Create | Edit | Delete | Actions | Status |
|------|------|--------|--------|------|--------|---------|--------|
| Dashboard | — | Real data | — | — | — | Stat cards, risk landscape, compliance rings, deployment pipeline, activity feed | **Complete** |
| Endpoints | Yes | Yes (8 tabs) | — | — | Decommission | Deploy, Scan, CSV export | **Complete** |
| Tags | Yes | — | Yes | Yes | Yes | Search, assign | **Complete** |
| Patches | Yes | Yes | — | — | — | Deploy via wizard | **Complete** |
| CVEs | Yes | Yes | — | — | — | Deploy fix action | **Complete** |
| Policies | Yes | Yes | Yes | Yes | Yes | Toggle, Evaluate, Bulk actions | **Complete** |
| Deployments | Yes | Yes | Wizard | — | — | Cancel, Retry, Rollback | **Complete** |
| Workflows | Yes | — | Editor | Editor | — | Publish, Execute, Approve/Reject | **Complete** |
| Compliance | Yes | Yes (5 tabs) | — | — | — | Enable/Disable, Evaluate, Custom CRUD | **Complete** |
| Audit | Yes | — | — | — | — | Filter, Export CSV/JSON, Timeline view | **Complete** |
| Notifications | — | Yes | — | — | — | Preferences, History, Channel config | **Complete** |
| Settings | — | Yes | — | — | — | General saves; IAM/Integrations/License cosmetic | **Partial** |
| Admin: Roles | Yes | Yes | Yes | Yes | — | Permission management | **Complete** |
| Admin: User Roles | Yes | — | — | — | — | Assign/Revoke | **Complete** |
| Agent Downloads | — | Yes | — | — | — | Platform binaries, install commands | **Complete** |
| Groups | Yes | — | Yes | Yes | Yes | CRUD dialogs | **Complete** |
| **Alerts/Events** | — | — | — | — | — | — | **MISSING** |

### 2.5 Frontend — web-hub/ (Hub Manager) — Status: MOSTLY COMPLETE

| Page | Status | Issues |
|------|--------|--------|
| Dashboard | **Complete** | Fleet topology, stat cards, catalog growth, activity |
| Catalog | **Complete** | List, detail, search/filter |
| Feeds | **Complete** | List, detail, sync trigger, history |
| Licenses | **Mostly** | List, detail, assign, revoke. **3 TODOs**: renewal flow, usage history chart, audit trail (PIQ-247) |
| Clients | **Mostly** | List, detail, approve/decline/suspend. **5 TODOs**: placeholder charts for endpoint history, sync data, OS distribution (PIQ-247) |
| Settings | **Complete** | General, IAM (editable), API/Webhooks, Feed Config — all functional |
| Deployments | **Minimal** | Listed in routes but minimal implementation |
| **Auth** | **Mock** | PIQ-12: Still using mock authentication, no Zitadel OIDC |

### 2.6 Frontend — web-agent/ (Agent) — Status: COMPLETE

All 9 pages fully functional. No TODOs. Rich detail views with real-time metrics, hardware specs, service monitoring, structured logs.

### 2.7 Settings Deep-Dive

The settings page was specifically called out. Here's the exact status:

| Settings Area | App | Functional? | What Works | What Doesn't |
|---------------|-----|-------------|------------|--------------|
| General (org, timezone, scan interval) | web | **Yes** | All 4 fields save to backend with validation | — |
| IAM/SSO | web | **Cosmetic** | Test Connection works and saves result | Save button shows toast "managed through Zitadel console" — fields are read-only |
| Integrations | web | **Cosmetic** | Test buttons work, shows channel status | Save button shows toast "managed from Notifications page" — can't edit here |
| License | web | **Cosmetic** | Displays tier, features, expiry, usage | "Manage License" shows toast "available in enterprise tiers" |
| Appearance | web | **Yes** | Theme toggle (light/dark) saves to localStorage | — |
| General | web-hub | **Yes** | Hub name, sync interval, region, auto-publish, timezone | — |
| IAM | web-hub | **Yes** | SSO URL, client ID/secret, redirect URI, role mappings — all editable and save | Test connection is client-side simulated (not real) |
| API/Webhooks | web-hub | **Mostly** | Webhook URL + event subscriptions save | "Send Test Event" and "Rotate API Key" are cosmetic |
| Feed Config | web-hub | **Yes** | Per-feed enable/disable, sync interval, manual retry | — |
| All settings | web-agent | **Yes** | Heartbeat, bandwidth, scan interval, log level, retention — all save | — |

### 2.8 Compliance Deep-Dive

| Feature | Status | Detail |
|---------|--------|--------|
| Evaluator engine | **Real** | Full state machine: COMPLIANT / AT_RISK / NON_COMPLIANT / LATE_REMEDIATION |
| 6 built-in frameworks | **Real** | NIST (3 controls), PCI-DSS (2), CIS (10), HIPAA (10), ISO 27001 (8), SOC 2 (8) |
| Custom frameworks | **Real** | CRUD with controls, migration 043 |
| API (12+ endpoints) | **Real** | Summary, scores, overdue, frameworks, controls, trends, evaluate |
| Compliance overview page | **Real** | Ring gauges, framework cards, overdue controls, trend chart |
| Framework detail (5 tabs) | **Real** | Overview, Controls, Endpoints, SLA Tracking, Evidence & Reports |
| Trend data | **Real** | Historical score points via API |
| Dashboard card | **Real** | Compliance rate from SQL |
| **Export/Report** | **Shell** | UI buttons exist (PDF, CSV, JSON, Executive Summary) — zero backend implementation |
| **Workflow compliance node** | **Stub** | Always returns "pass" — silently fakes compliance validation |

---

## 3. E2E Flow Verification & Fixes

These are the 7 flows that must work perfectly. Each flow needs to be verified with real data, and every gap fixed.

### Flow 1: Agent Lifecycle
**Path**: Agent enrollment → inventory scan → data appears in PM UI

**Verify**:
- [ ] Fresh agent can enroll with registration token
- [ ] Agent sends heartbeat, status shows "Connected" in PM
- [ ] Agent runs inventory scan on enrollment
- [ ] Inventory data (packages, hardware, services) visible in endpoint detail tabs
- [ ] Agent reconnects gracefully after server restart
- [ ] Agent handles offline mode (queues outbox, syncs when reconnected)
- [ ] Multiple agents can enroll and appear in endpoints list
- [ ] Endpoint detail shows correct OS, hostname, IP, last seen

**Known risks**: Agent reconnection UX not verified; what does UI show during disconnect?

### Flow 2: Patching Pipeline
**Path**: Create deployment → wave dispatch → agent installs → results reported → deployment completes

**Verify**:
- [ ] Deployment wizard: select patches → select targets → configure waves → schedule → create
- [ ] Deployment appears in list with "Pending" status
- [ ] Wave dispatcher picks up deployment, transitions to "Running"
- [ ] Agent receives install_patch command via SyncInbox
- [ ] Agent installs patch (APT on Linux verified)
- [ ] Agent reports result via SyncOutbox
- [ ] Server processes result, updates target status
- [ ] Wave completion evaluated, next wave triggered
- [ ] Deployment detail page shows per-target results (success/failed/pending)
- [ ] Deployment completes with correct final status
- [ ] Cancel deployment mid-execution works
- [ ] Retry failed deployment works
- [ ] Rollback deployment works (agent receives rollback command)
- [ ] Deployment timeout fires after deadline (5 min default)
- [ ] Post-deployment scan scheduler triggers re-scan

**Known risks**: OpenAPI spec missing wave_config and scheduling fields (PIQ-239); frontend extends types manually.

### Flow 3: Vulnerability Management
**Path**: CVE feeds → correlation → endpoint matching → risk scoring → deploy fix → CVE resolved

**Verify**:
- [ ] Hub NVD feed syncs CVEs (verify with real NVD data, check rate limiting)
- [ ] CVE-patch correlation links CVEs to patches by package name
- [ ] Inventory scan triggers endpoint CVE matching
- [ ] CVEs page shows real CVEs with CVSS scores and KEV flagging
- [ ] CVE detail page shows affected endpoints
- [ ] Endpoint detail "CVE Exposure" tab shows matched CVEs
- [ ] "Deploy Fix" action from CVE page creates a deployment targeting affected endpoints
- [ ] After successful patch deployment, CVE is marked as remediated
- [ ] Dashboard "Top Vulnerabilities" card shows real data
- [ ] Risk scores update after patching

**Known risks**: NVD API key needed for production (currently rate-limited, 429 errors likely).

### Flow 4: Policy & Auto-Deploy
**Path**: Create policy → evaluate → match endpoints/patches → trigger deployment (manual or auto)

**Verify**:
- [ ] Create policy with severity filter and tag-based targeting
- [ ] Toggle policy enabled/disabled
- [ ] "Evaluate" action runs and shows matched endpoints + patches count
- [ ] Manual mode: evaluation pre-fills deployment wizard
- [ ] Automatic mode: ScheduleCheckerJob creates deployments automatically
- [ ] Advisory mode: evaluation only, no deployment created
- [ ] Policy detail page shows evaluation results
- [ ] Bulk actions work (enable/disable multiple policies)
- [ ] Policy with tag expression targeting works (e.g., `os:linux AND env:production`)

### Flow 5: Compliance
**Path**: Enable framework → evaluate → scores → control results → trends

**Verify**:
- [ ] Enable a framework (e.g., NIST 800-53) from compliance page
- [ ] Trigger manual evaluation via "Evaluate Now"
- [ ] Compliance score appears and is non-zero (reflects real endpoint state)
- [ ] Framework detail: Overview tab shows trend chart with real data points
- [ ] Framework detail: Controls tab shows per-control pass/fail with SLA deadlines
- [ ] Framework detail: Endpoints tab shows non-compliant endpoints
- [ ] Framework detail: SLA Tracking tab shows overdue controls with timers
- [ ] Overdue controls appear in main compliance page table
- [ ] Dashboard compliance card shows real rate
- [ ] Custom framework: create → add controls → evaluate → see results
- [ ] Disable framework removes it from active list
- [ ] Compliance periodic worker (every 6h) runs and updates scores

**Known gaps**:
- [ ] **FIX**: Export/Report buttons (4 formats) have zero backend — either implement or remove from UI
- [ ] **FIX**: Workflow ComplianceCheckHandler stub — either wire to real evaluator or remove node from workflow templates

### Flow 6: Hub → PM Data Pipeline
**Path**: Hub feeds aggregate → catalog built → synced to PM → patches available for deployment

**Verify**:
- [ ] All 6 Hub feeds sync on schedule (NVD 6h, others 12h)
- [ ] Feed detail page shows sync history with entry counts
- [ ] Catalog page shows aggregated patches with correct metadata
- [ ] Hub→PM catalog sync delivers patches (verify delta sync with `since` parameter)
- [ ] PM patches page shows synced patches
- [ ] Manual "Sync Now" from PM triggers catalog pull
- [ ] New feed entries appear in PM after next sync cycle

### Flow 7: Notifications
**Path**: Configure channel → event triggers → notification sent → history recorded

**Verify**:
- [ ] Create notification channel (email, Slack, Discord, or webhook)
- [ ] Test notification sends successfully
- [ ] Configure notification preferences (which events trigger which channels)
- [ ] Trigger an event (e.g., deployment completed) and verify notification fires
- [ ] Notification history page shows sent notifications with status
- [ ] All 11 subscribed events actually trigger notifications:
  - DeploymentStarted, DeploymentCompleted, DeploymentFailed, DeploymentRollbackTriggered
  - ComplianceThresholdBreach, ComplianceEvaluationCompleted
  - AgentDisconnected
  - CVEDiscovered, CVERemediationAvailable
  - CatalogSyncFailed
  - LicenseExpiring

---

## 4. Critical Missing Pieces

### 4.1 Alerts/Events Page (NEW)

**What**: A new page in web/ that shows a unified, real-time feed of all system events.

**Why**: Clients need a single place to see "what's happening." Audit log is historical/forensic. Alerts page is operational — "what needs my attention now."

**Requirements**:
- [ ] New route: `/alerts`
- [ ] Sidebar nav entry between Notifications and Settings
- [ ] Real-time event feed from domain events (filterable by severity: critical, warning, info)
- [ ] Event categories: deployments, agents, CVEs, compliance, system
- [ ] Each alert shows: timestamp, severity icon, title, description, source entity link
- [ ] Filter by: severity, category, date range, search text
- [ ] Mark as read/acknowledged
- [ ] Badge count in sidebar showing unread critical/warning alerts
- [ ] Auto-refresh (polling or SSE)

**Backend needs**:
- [ ] New API endpoint: `GET /api/v1/alerts` (query domain events filtered by severity/category with unread tracking)
- [ ] New table or view: `alerts` (materialized from domain events with read/acknowledged state)
- [ ] Consider: SSE endpoint for real-time push (optional, polling is fine for POC)

### 4.2 Settings Page Overhaul

**Problem**: IAM, Integrations, and License cards in web/ are cosmetic. Buttons exist that do nothing useful.

**Fix options** (per card):

**IAM Card**:
- [ ] Option A: Make fields editable and wire PUT `/api/v1/settings/iam` (backend exists but UI doesn't call it)
- [ ] Option B: Remove the fake "Save" button, keep as read-only status display with link to Zitadel console
- [ ] Recommendation: **Option B for POC** — Zitadel has its own admin UI; duplicating config here is error-prone

**Integrations Card**:
- [ ] Option A: Add inline channel config forms (currently redirects to Notifications page)
- [ ] Option B: Replace "Save Integrations" toast with a clear link/button to Notifications page
- [ ] Recommendation: **Option B for POC** — avoid duplicate config surfaces

**License Card**:
- [ ] Option A: Implement license upload/activation flow
- [ ] Option B: Keep read-only display, remove fake "Manage License" button
- [ ] Recommendation: **Option B for POC** — license is configured server-side for client deployments

**General rule**: If a button doesn't do anything, remove it. No fake buttons in POC.

### 4.3 Hub Placeholder Data (PIQ-247)

8 TODOs across web-hub/ using fake chart data. Each needs real backend data or removal.

**Clients pages (5 TODOs)**:
- [ ] ClientDetailPage: Endpoint count history chart → need backend endpoint tracking per client over time, or remove chart
- [ ] ClientDetailPage: Sync success/failure data → wire to feed_sync_history table, or remove chart
- [ ] ClientDetailPage: Per-endpoint compliance/patch charts → need aggregation endpoint, or remove charts
- [ ] ClientsPage: Real sync history rows → wire to existing sync data
- [ ] ClientsPage: Real per-endpoint OS data → wire to client endpoint inventory

**Licenses pages (3 TODOs)**:
- [ ] LicenseDetailPage: License renewal flow → remove "Renew" button for POC (server-side operation)
- [ ] LicenseDetailPage: Usage history chart → need time-series endpoint data, or remove chart
- [ ] LicenseDetailPage: Full audit trail → wire to audit_events filtered by license entity

**Recommendation**: For each, either wire to real data (if backend data exists in some form) or remove the chart/section entirely. No placeholders in POC.

### 4.4 Hub Authentication (PIQ-12)

**Current**: web-hub/AuthContext.tsx uses mock user — no login, no session, no auth.

**Fix**:
- [ ] Wire Zitadel OIDC same as web/ (the server-side auth code is the same pattern)
- [ ] Or implement simple API key auth if Hub is internal-only for POC
- [ ] Hub settings IAM page already has fields for SSO config — these need to work end-to-end

### 4.5 OpenAPI Spec Gap (PIQ-239)

**Current**: `wave_config`, `max_concurrent`, `scheduled_at` fields missing from server.yaml. Frontend manually extends the generated types.

**Fix**:
- [ ] Add fields to api/server.yaml deployment schemas
- [ ] Regenerate types: `make api-client`
- [ ] Remove manual type extensions from frontend

### 4.6 Compliance Exports

**Current**: 4 export buttons in UI (Full Report PDF, Control Evidence CSV, Audit Trail JSON, Executive Summary PDF). Zero backend.

**Fix options**:
- [ ] Option A: Implement backend export endpoints (at least CSV/JSON — PDF is complex)
- [ ] Option B: Remove export buttons from UI for POC
- [ ] Recommendation: **Implement CSV and JSON exports** (straightforward — query controls + scores, format as CSV/JSON). Remove PDF buttons or mark as "Coming Soon."

### 4.7 Workflow Compliance Node

**Current**: `ComplianceCheckHandler` always returns pass. Workflows with compliance gates silently skip real validation.

**Fix**:
- [ ] Option A: Wire handler to real compliance evaluator (check framework score, fail if below threshold)
- [ ] Option B: Remove compliance_check from available workflow nodes
- [ ] Recommendation: **Option A** — the evaluator already exists; handler just needs to call it with the framework ID and threshold from node config.

---

## 5. Platform Polish

### 5.1 Error Boundaries

**Problem**: One uncaught JS error in any React component → white screen of death for entire app.

**Fix**:
- [ ] Add React Error Boundary wrapper to each app's root layout
- [ ] Add per-page error boundaries for graceful degradation
- [ ] Error fallback UI: "Something went wrong" with retry button and error details (dev mode)
- [ ] Apply to all 3 apps: web/, web-hub/, web-agent/

### 5.2 Empty State Audit

**Problem**: Pages with no data may show blank white space instead of helpful guidance.

**Fix**:
- [ ] Audit every list page: what renders when the query returns 0 results?
- [ ] Every empty state must show: icon, message, primary action (e.g., "No endpoints yet → Register your first endpoint")
- [ ] Key pages to check: Endpoints, Deployments, Policies, Workflows, Compliance, Alerts, Tags
- [ ] EmptyState component exists in packages/ui/ — ensure it's used consistently

### 5.3 Loading State Audit

**Problem**: Pages may show hanging spinners or no loading indication.

**Fix**:
- [ ] Audit every page: does it show skeleton loaders while data fetches?
- [ ] SkeletonCard component exists in packages/ui/ — ensure consistent use
- [ ] No infinite spinners — add timeout with error state after 30s
- [ ] Check: what happens when backend is down? Each page should show ErrorState, not spinner

### 5.4 Dashboard Verification

- [ ] Verify all dashboard stat cards show real computed data (not hardcoded)
- [ ] Verify charts render with real data (deployment pipeline, patch velocity, compliance rings)
- [ ] Verify activity feed shows real recent events
- [ ] Verify risk landscape shows real endpoint risk distribution
- [ ] Verify top vulnerabilities shows real CVEs

### 5.5 Audit Log Verification

- [ ] Verify CSV export produces valid CSV with all columns
- [ ] Verify JSON export produces valid JSON
- [ ] Verify filters work: event type, actor, resource type, date range
- [ ] Verify timeline view renders events in correct chronological order
- [ ] Verify audit events are generated for all write operations (domain events rule)

---

## 6. Windows Agent Completion

**Scope decision needed**: Is Windows in scope for the POC? If the client runs Windows endpoints, this is critical. If Linux-only POC, skip entirely.

### If Windows is in scope:

**6.1 WUA Inventory Collector**:
- [ ] Register WUA collector in `internal/agent/inventory/detect_windows.go`
- [ ] Verify it detects available Windows Updates
- [ ] Verify inventory data flows to server via SyncOutbox

**6.2 WUA Patcher Installer**:
- [ ] Implement `wuaInstaller` in `internal/agent/patcher/`
- [ ] Use Windows Update Agent COM API (IUpdateSearcher, IUpdateDownloader, IUpdateInstaller)
- [ ] Handle reboot requirements
- [ ] Test with real Windows Updates

**6.3 Windows Agent E2E**:
- [ ] Agent enrollment on Windows
- [ ] Inventory scan captures Windows packages + hotfixes + WUA updates
- [ ] Deployment installs Windows Update via WUA
- [ ] Rollback support for Windows patches

---

## 7. Deployment Packaging

### 7.1 Production Docker Compose

**Current**: Only `docker-compose.dev.yml` exists with dev-specific config (debug ports, hot-reload, liberal CORS).

**Need**:
- [ ] `docker-compose.prod.yml` with:
  - Server, Hub, PostgreSQL, Valkey, Zitadel, MinIO containers
  - Production-safe defaults (no debug, restricted CORS, TLS)
  - Health checks for all services
  - Persistent volumes for all stateful services
  - Resource limits
  - Restart policies
  - Log rotation
- [ ] `.env.prod.example` with all required environment variables documented

### 7.2 Dockerfiles

**Current**: `deploy/docker/Dockerfile.server` and `Dockerfile.hub` exist with multi-stage builds and distroless base.

**Need**:
- [ ] Add HEALTHCHECK directives to both Dockerfiles
- [ ] Verify production builds work: `docker build -f deploy/docker/Dockerfile.server .`
- [ ] Agent Dockerfile (if distributing as container — may not be needed if distributing as binary)

### 7.3 Agent Installers

**Current**: `make build-agents` cross-compiles binaries.

**Need**:
- [ ] `.deb` package for Debian/Ubuntu (with systemd service file)
- [ ] `.rpm` package for RHEL/CentOS (with systemd service file)
- [ ] `.msi` installer for Windows (if in scope)
- [ ] Install script that: downloads binary, creates config, registers systemd service, starts agent
- [ ] The Agent Downloads page in web/ should serve these packages

### 7.4 Secrets Management

**Current**: Credentials in YAML config files and env vars.

**Need**:
- [ ] Document which secrets exist (DB password, Zitadel client secret, MinIO credentials, Hub sync API key, notification channel credentials, RSA keys)
- [ ] Ensure all secrets can be set via environment variables (no plaintext in config files)
- [ ] `.env.prod.example` with clear documentation of each secret
- [ ] AES encryption at rest for stored credentials (notification channels — already exists via shared/crypto)

### 7.5 Setup Documentation

**Need**:
- [ ] `docs/deployment-guide.md` covering:
  - Prerequisites (Docker, network requirements, DNS)
  - Initial setup (docker-compose up, run migrations, create initial tenant, configure Zitadel)
  - Agent enrollment (generate token, install agent, verify connection)
  - Hub configuration (enable feeds, initial sync)
  - First deployment walkthrough
  - Troubleshooting guide (common errors, log locations, health checks)

### 7.6 Monitoring & Observability

**Current**: OpenTelemetry infrastructure exists. Grafana OTEL-LGTM stack in dev compose.

**Need**:
- [ ] Grafana dashboards for:
  - Platform health (HTTP latency, error rates, DB connection pool)
  - Agent fleet (connected/disconnected agents, heartbeat status)
  - Deployment pipeline (active deployments, success/failure rates)
  - Feed sync status (last sync time, error counts)
- [ ] Alert rules for critical conditions:
  - All agents disconnected
  - Database connection failures
  - Feed sync failures > 3 consecutive
  - Deployment failure rate > 50%

### 7.7 Backup & Recovery

**Need**:
- [ ] PostgreSQL backup script (pg_dump with compression)
- [ ] Backup schedule recommendation (daily full, hourly WAL)
- [ ] Restore procedure documented and tested
- [ ] MinIO backup (mc mirror or S3 replication)

---

## 8. QA & Hardening

> This section should be executed AFTER sections 3-5 are complete.

### 8.1 Playwright E2E Test Suite

One test per core flow:
- [ ] Test 1: Agent enrollment → appears in endpoints list
- [ ] Test 2: Create deployment → verify status progression → completion
- [ ] Test 3: CVE list → deploy fix → verify remediation
- [ ] Test 4: Create policy → evaluate → verify matches
- [ ] Test 5: Compliance → enable framework → evaluate → verify scores
- [ ] Test 6: Notifications → create channel → test → verify history
- [ ] Test 7: Settings → update general settings → verify persistence

### 8.2 Error Handling Audit

- [ ] No uncaught exceptions in browser console during normal use
- [ ] Every API error returns structured JSON with context (not generic 500)
- [ ] Every failed mutation shows a user-visible error toast
- [ ] Backend: grep for suppressed errors (empty catch blocks, ignored error returns)
- [ ] Frontend: grep for `catch {}` or `catch (_)` blocks that silently swallow errors

### 8.3 NVD API Key

- [ ] Obtain NVD API key from https://nvd.nist.gov/developers/request-an-api-key
- [ ] Configure in Hub config (`feeds.nvd.api_key`)
- [ ] Verify rate limit increases from 5/30s to 50/30s
- [ ] Test full NVD sync completes without 429 errors

### 8.4 Performance Baseline

- [ ] Test with 50+ endpoints (agents or simulated)
- [ ] Test concurrent deployments (5+ active deployments)
- [ ] Verify dashboard renders within 3 seconds with real data
- [ ] Verify endpoints list page handles 100+ endpoints without UI lag
- [ ] Verify audit log handles 10,000+ events without pagination issues

---

## 9. Work Item Index

Quick reference of all work items, organized by type.

### Verification Tasks (confirm existing features work E2E)

| ID | Item | Section |
|----|------|---------|
| V1 | Agent enrollment → inventory scan → UI display | 3.1 |
| V2 | Deployment creation → wave dispatch → agent install → result → completion | 3.2 |
| V3 | CVE feeds → correlation → endpoint matching → deploy fix → remediation | 3.3 |
| V4 | Policy create → evaluate → match → auto-deploy | 3.4 |
| V5 | Compliance enable → evaluate → scores → controls → trends | 3.5 |
| V6 | Hub feeds → catalog sync → PM patches available | 3.6 |
| V7 | Notification channel → event trigger → delivery → history | 3.7 |

### Build Tasks (new code needed)

| ID | Item | Section | Size |
|----|------|---------|------|
| B1 | Alerts/Events page (new route, API, table/view) | 4.1 | Large |
| B2 | Compliance CSV/JSON export endpoints | 4.6 | Medium |
| B3 | Wire workflow ComplianceCheckHandler to real evaluator | 4.7 | Small |
| B4 | Hub Zitadel OIDC auth (PIQ-12) | 4.4 | Medium |
| B5 | Production docker-compose.yml | 7.1 | Medium |
| B6 | Agent .deb/.rpm packages | 7.3 | Medium |
| B7 | Deployment guide documentation | 7.5 | Medium |
| B8 | Grafana monitoring dashboards | 7.6 | Medium |
| B9 | PostgreSQL backup/restore scripts | 7.7 | Small |
| B10 | Error boundaries (all 3 apps) | 5.1 | Small |
| B11 | WUA inventory collector registration (Windows) | 6.1 | Small |
| B12 | WUA patcher installer (Windows) | 6.2 | Large |

### Fix Tasks (existing code needs correction)

| ID | Item | Section | Size |
|----|------|---------|------|
| F1 | Settings page: remove fake buttons or wire to real actions | 4.2 | Small |
| F2 | Hub placeholder data: wire real data or remove charts (8 TODOs) | 4.3 | Medium |
| F3 | OpenAPI spec: add wave_config/scheduling fields (PIQ-239) | 4.5 | Small |
| F4 | Remove compliance PDF export buttons (or mark "Coming Soon") | 4.6 | Small |
| F5 | Dockerfiles: add HEALTHCHECK directives | 7.2 | Small |
| F6 | Secrets: ensure all configurable via env vars | 7.4 | Small |

### Polish Tasks (UX quality)

| ID | Item | Section | Size |
|----|------|---------|------|
| P1 | Empty state audit (all list pages) | 5.2 | Medium |
| P2 | Loading state audit (all pages) | 5.3 | Medium |
| P3 | Dashboard real data verification | 5.4 | Small |
| P4 | Audit log export verification | 5.5 | Small |

### QA Tasks (after all above complete)

| ID | Item | Section | Size |
|----|------|---------|------|
| Q1 | Playwright E2E test suite (7 flows) | 8.1 | Large |
| Q2 | Error handling audit | 8.2 | Medium |
| Q3 | NVD API key setup | 8.3 | Small |
| Q4 | Performance baseline (50+ agents) | 8.4 | Medium |

### Totals

| Type | Count | Small | Medium | Large |
|------|-------|-------|--------|-------|
| Verification | 7 | — | — | — |
| Build | 12 | 3 | 7 | 2 |
| Fix | 6 | 4 | 2 | 0 |
| Polish | 4 | 2 | 2 | 0 |
| QA | 4 | 1 | 2 | 1 |
| **Total** | **33** | **10** | **13** | **3** |

### Dependency Map

```
                    ┌─── V1-V7 (Verification) ──────┐
                    │                                │
Start ──────────────┼─── B1-B10, F1-F6 (Build/Fix) ─┼──► Q1-Q4 (QA) ──► POC Ready
                    │                                │
                    ├─── P1-P4 (Polish) ─────────────┘
                    │
                    └─── B5-B9 (Packaging) ──────────────► Deploy to Client

Windows (B11-B12): Independent. Only if Windows is in POC scope.
```

All work in the left column can run in parallel. QA and deployment are the gates.
