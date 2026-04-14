# Patch Manager - Complete Technical Reference

> One-stop technical document covering every aspect of PatchIQ's Patch Manager system: architecture, data flow, deployment engine, agent installation, APIs, database schema, event system, and operational details.

---

## Table of Contents

1. [System Architecture Overview](#1-system-architecture-overview)
2. [Three-Tier Platform Topology](#2-three-tier-platform-topology)
3. [Patch Lifecycle - End-to-End Flow](#3-patch-lifecycle---end-to-end-flow)
4. [Hub Manager - Patch Catalog & Feed Ingestion](#4-hub-manager---patch-catalog--feed-ingestion)
5. [Patch Manager Server - Core Backend](#5-patch-manager-server---core-backend)
6. [Deployment Engine - Deep Dive](#6-deployment-engine---deep-dive)
7. [State Machine - Deployment States & Transitions](#7-state-machine---deployment-states--transitions)
8. [Wave-Based Deployment - Phased Rollout](#8-wave-based-deployment---phased-rollout)
9. [Agent Patcher Module - Installation Execution](#9-agent-patcher-module---installation-execution)
10. [Policy Engine - Automated Deployment](#10-policy-engine---automated-deployment)
11. [CVE & Vulnerability Integration](#11-cve--vulnerability-integration)
12. [REST API Reference](#12-rest-api-reference)
13. [gRPC Protocol - Agent Communication](#13-grpc-protocol---agent-communication)
14. [Database Schema](#14-database-schema)
15. [Domain Events](#15-domain-events)
16. [Background Jobs (River)](#16-background-jobs-river)
17. [Frontend (React UI)](#17-frontend-react-ui)
18. [Configuration & Environment](#18-configuration--environment)
19. [Operational Parameters & Limits](#19-operational-parameters--limits)
20. [Key File Index](#20-key-file-index)

---

## 1. System Architecture Overview

PatchIQ is an enterprise patch management platform built on a three-tier Hub-Spoke architecture. The Patch Manager is the central on-premises component that orchestrates all patch operations for endpoints within an organization.

```
+---------------------+          REST API          +------------------------+        gRPC + mTLS        +------------------+
|   Hub Manager       | ───────────────────────>   |    Patch Manager       | ──────────────────────>   |    Agent         |
|   (SaaS / Central)  |   Catalog Sync, License    |    (On-Prem Server)    |   Commands, Results      |    (Endpoint)    |
|                     | <───────────────────────   |                        | <──────────────────────   |                  |
+---------------------+                           +------------------------+                          +------------------+
        |                                                   |                                                  |
   PostgreSQL 16                                       PostgreSQL 16                                      SQLite (local)
   MinIO (S3)                                          Valkey (cache)                                     Offline-first
   6 Feed Sources                                      River (jobs)
                                                       Watermill (events)
                                                       Zitadel (IAM)
```

### Technology Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.25.0 |
| HTTP Router | chi/v5 |
| Database | PostgreSQL 16 (pgx/v5 + sqlc) |
| Migrations | goose |
| Job Queue | River (12 workers) |
| Event Bus | Watermill (PostgreSQL transport) |
| Agent Protocol | gRPC + protobuf |
| Agent DB | modernc.org/sqlite |
| Cache | Valkey (Redis-compatible) |
| Object Storage | MinIO (S3-compatible) |
| IAM | Zitadel (OIDC) |
| Config | koanf (YAML + env vars) |
| Observability | OpenTelemetry + slog |
| Notifications | Shoutrrr (email, Slack, Discord, webhooks) |

---

## 2. Three-Tier Platform Topology

### Hub Manager (SaaS / Central)

**Purpose:** Global patch catalog, CVE feed aggregation, license management.

- **Backend:** `internal/hub/`
- **Frontend:** `web-hub/`
- **Database:** PostgreSQL 16 (global `patch_catalog` table, NOT tenant-scoped)
- **Ports:** HTTP :8082, UI :3002

The Hub aggregates vulnerability data from 6 external feeds, normalizes them into a unified patch catalog, and distributes catalog updates to connected Patch Manager instances.

### Patch Manager (On-Premises Server)

**Purpose:** Central management server for an organization's patch operations.

- **Backend:** `internal/server/`
- **Frontend:** `web/`
- **Database:** PostgreSQL 16 (tenant-scoped via RLS)
- **Ports:** HTTP :8080, gRPC :50051, UI :3001

The Patch Manager is the core orchestration layer. It:
- Syncs the patch catalog from the Hub
- Manages endpoints and their inventory
- Evaluates policies to determine which patches apply to which endpoints
- Orchestrates deployments with wave-based rollout
- Collects results from agents and updates deployment state
- Provides the primary UI for administrators

### Agent (Endpoint)

**Purpose:** Lightweight daemon running on each managed endpoint.

- **Backend:** `internal/agent/`
- **Frontend:** `web-agent/`
- **Database:** Local SQLite
- **Ports:** HTTP :8090, UI :3003

The Agent is offline-first and minimal. It:
- Enrolls with the Patch Manager via gRPC
- Sends periodic heartbeats
- Collects package inventory (installed software)
- Receives and executes patch install/rollback commands
- Reports results back to the server

---

## 3. Patch Lifecycle - End-to-End Flow

This is the complete flow from a vulnerability being discovered to a patch being installed on an endpoint.

```
                     FEED INGESTION (Hub)
                            │
    ┌──────────┬────────────┼────────────┬──────────┬──────────┐
    │          │            │            │          │          │
   NVD    CISA KEV       MSRC      RedHat OVAL  Ubuntu USN  Apple
  (6h)     (12h)        (12h)       (12h)        (12h)      (12h)
    │          │            │            │          │          │
    └──────────┴────────────┼────────────┴──────────┴──────────┘
                            │
                   Normalization Pipeline
                   (hub/catalog/pipeline.go)
                            │
                    ┌───────▼────────┐
                    │  patch_catalog  │  (Hub PostgreSQL - global)
                    │  + CVE links    │
                    └───────┬────────┘
                            │
                   Binary Fetching (MSU, APT, YUM, Apple)
                   Stored in MinIO (S3)
                            │
              ══════════════╪═══════════════
              CATALOG SYNC (Hub → Server)
              CatalogSyncJob (River, on event)
              ══════════════╪═══════════════
                            │
                    ┌───────▼────────┐
                    │    patches      │  (Server PostgreSQL - tenant-scoped)
                    │  + patch_cves   │
                    └───────┬────────┘
                            │
            ┌───────────────┼───────────────┐
            │               │               │
     Policy Evaluation  Quick Deploy   Scheduled Deploy
      (auto, 6h cycle)   (manual)     (cron-based)
            │               │               │
            └───────────────┼───────────────┘
                            │
                    ┌───────▼────────┐
                    │   deployment    │  status: created
                    │ + waves         │
                    │ + targets       │
                    └───────┬────────┘
                            │
                   Executor (River job)
                   CREATED → RUNNING
                   Activates Wave 1
                            │
                   Wave Dispatcher (30s cycle)
                            │
                    ┌───────▼────────┐
                    │   commands      │  type: install_patch
                    │   (pending)     │  payload: InstallPatchPayload
                    └───────┬────────┘
                            │
              ══════════════╪═══════════════
              gRPC: SyncInbox (server → agent)
              ══════════════╪═══════════════
                            │
                    ┌───────▼────────┐
                    │  Agent Patcher  │
                    │  Module         │
                    │                 │
                    │  1. Download    │  (binary from /repo/files/)
                    │  2. Pre-script  │
                    │  3. Install     │  (OS-specific installer)
                    │  4. Post-script │
                    │  5. Rollback    │  (save record for undo)
                    │     tracking    │
                    └───────┬────────┘
                            │
              ══════════════╪═══════════════
              gRPC: SyncOutbox (agent → server)
              CommandResponse with results
              ══════════════╪═══════════════
                            │
                   Result Handler
                   (deployment/results.go)
                            │
              ┌─────────────┼─────────────┐
              │             │             │
        Update Command  Update Target  Increment Wave
         Status          Status         & Deployment
         (succeeded/     (succeeded/    Counters
          failed)         failed)
                            │
                   Wave Completion Check
                   (evaluator logic)
                            │
              ┌─────────────┼─────────────┐
              │             │             │
        Success Rate    Failure Rate   All Targets
        >= threshold    > error_max    Terminal
              │             │             │
        Advance to     Trigger         Complete
        Next Wave      Rollback        Deployment
              │             │             │
              └─────────────┼─────────────┘
                            │
                   Notification System
                   (email, Slack, Discord, webhook)
```

---

## 4. Hub Manager - Patch Catalog & Feed Ingestion

### External Feed Sources

The Hub ingests vulnerability and patch data from 6 external sources, each on its own sync cycle:

| Feed | Source | Sync Interval | Data Type |
|------|--------|--------------|-----------|
| NVD | National Vulnerability Database | 6 hours | CVE records, CVSS scores |
| CISA KEV | Known Exploited Vulnerabilities | 12 hours | Actively exploited CVEs |
| MSRC | Microsoft Security Response Center | 12 hours | Windows security updates |
| RedHat OVAL | Red Hat Security Advisories | 12 hours | RHEL/CentOS advisories |
| Ubuntu USN | Ubuntu Security Notices | 12 hours | Debian/Ubuntu advisories |
| Apple | Apple Security Updates | 12 hours | macOS/iOS updates |

**Implementation:** Each feed has a dedicated Go implementation in `internal/hub/feeds/`:
- `nvd.go`, `cisa_kev.go`, `msrc.go`, `redhat.go`, `ubuntu.go`, `apple.go`
- Each implements a common `Feed` interface
- Sync is driven by `FeedSyncJob` River workers on periodic schedules

### Normalization Pipeline

**File:** `internal/hub/catalog/pipeline.go`

Raw feed entries are processed through a normalization pipeline:

```
RawFeedEntry → OS Detection → Package Name Normalization → CVE Linking → Deduplication → PatchCatalog Record
```

Key operations:
1. **OS detection** — Maps advisory metadata to standardized os_family values (linux-debian, linux-rhel, windows, macos, etc.)
2. **Package name normalization** — Extracts actual package name from advisory name
3. **CVE linking** — Associates CVEs with version ranges (`version_end_excluding`, `version_end_including`)
4. **Deduplication** — Prevents duplicate patches from multiple overlapping feeds

### Binary Fetching & Storage

For platforms requiring binary downloads (Windows, macOS), the Hub fetches patch binaries:

| Fetcher | File | Platforms |
|---------|------|-----------|
| MSU fetcher | `catalog/fetcher_msu.go` | Windows .msu files from Microsoft |
| APT fetcher | `catalog/fetcher_apt.go` | Debian/Ubuntu .deb packages |
| YUM fetcher | `catalog/fetcher_yum.go` | RHEL/CentOS .rpm packages |
| Apple fetcher | `catalog/fetcher_apple.go` | macOS installer packages |

**Storage:** MinIO (S3-compatible) — `internal/hub/catalog/minio*.go`
- Binaries are stored in S3 buckets organized by OS family
- Agents download from the Patch Manager's binary file server: `GET /repo/files/{os}/{patch_file}`

### Hub Database Schema

**File:** `internal/hub/store/migrations/001_init_schema.sql`

```sql
-- Global catalog (NOT tenant-scoped)
TABLE patch_catalog (
    id              UUID PRIMARY KEY,
    name            TEXT NOT NULL,
    version         TEXT NOT NULL,
    severity        TEXT NOT NULL,         -- critical, high, medium, low
    os_family       TEXT NOT NULL,         -- windows, linux-debian, linux-rhel, macos, etc.
    status          TEXT NOT NULL,         -- available, superseded, recalled
    os_distribution TEXT,
    package_url     TEXT,                  -- Binary download URL or S3 path
    checksum_sha256 TEXT,
    source_repo     TEXT,
    description     TEXT,
    release_date    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

TABLE patch_catalog_cves (
    patch_id              UUID REFERENCES patch_catalog(id),
    cve_id                UUID REFERENCES cves(id),
    version_end_excluding TEXT,           -- CVE affects versions < this
    version_end_including TEXT            -- CVE affects versions <= this
);
```

---

## 5. Patch Manager Server - Core Backend

### Server Initialization

**File:** `cmd/server/main.go`

Boot sequence:
```
Config → Logger → OTel → Signal Context → PostgreSQL → Watermill → River (12 workers)
→ Discovery/CVE/Deployment/Compliance/Notification Engines → gRPC :50051 → HTTP :8080
```

### Catalog Sync (Hub → Server)

The `CatalogSyncJob` (River worker) pulls new/updated patches from the Hub:

1. Server calls Hub's REST API to fetch unsynced catalog entries
2. For each entry, calls `UpsertDiscoveredPatch()` which inserts or updates the `patches` table
3. Links CVEs via `LinkPatchCVE()`
4. Emits `PatchDiscovered` event per new patch
5. Patches that the Hub has removed are soft-deleted via `SoftDeletePatchesByHubIDs()` (sets `deleted_at`)

### Patch Discovery Engine

**File:** `internal/server/discovery/job.go`

- **Cycle:** Every 60 minutes (DiscoveryJob)
- Scans configured patch repositories (APT, YUM, Microsoft, etc.)
- Upserts discovered patches into the server database
- Emits `RepositorySynced` event after each source completes

### Key Server Packages

| Package | Purpose | Key Files |
|---------|---------|-----------|
| `api/v1/` | REST handlers | `patches.go`, `deployments.go`, `deployment_schedules.go`, `policies.go`, `cves.go` |
| `deployment/` | Deployment orchestration | `statemachine.go`, `engine.go`, `wave_dispatcher.go`, `waves.go`, `evaluator.go`, `results.go`, `timeout.go`, `schedule_checker.go`, `emit.go` |
| `grpc/` | Agent communication | AgentService: Enroll, Heartbeat, SyncOutbox, SyncInbox |
| `cve/` | CVE sync & correlation | `job.go` (NVDSyncJob, EndpointMatchJob) |
| `discovery/` | Patch repository scanning | `job.go` (DiscoveryJob, 60min cycle) |
| `policy/` | Policy engine | `evaluator.go`, `scheduler.go`, `worker.go` |
| `compliance/` | Framework evaluation | CIS, PCI-DSS, HIPAA, NIST, ISO 27001, SOC 2 |
| `notify/` | Notifications | Shoutrrr (email, Slack, Discord, webhook) |
| `workflow/` | Workflow DAG execution | Custom workflow builder |
| `store/` | PostgreSQL access | 45 migrations, 28 sqlc query files |

---

## 6. Deployment Engine - Deep Dive

The deployment engine is the core of the Patch Manager. It orchestrates the entire lifecycle of deploying patches to endpoints.

### Deployment Creation Paths

There are 4 ways a deployment can be created:

#### 1. Quick Deploy (Manual, Immediate)
**Endpoint:** `POST /api/v1/patches/{id}/deploy`
**Handler:** `PatchHandler.QuickDeploy()` in `internal/server/api/v1/patches.go`

Quick deploy creates a deployment for a single patch targeting selected endpoints:

```go
// Inside a single database transaction:
1. Fetch the patch by ID
2. Parse request body (name, description, endpoint_ids/filter)
3. List all active endpoints for the tenant
4. Filter endpoints (by specific IDs or by OS family)
5. Create Deployment record (status: "created")
6. Create single DeploymentWave (100%, no delay)
7. Bulk-insert DeploymentTargets (one per endpoint)
8. Set wave target_count and deployment total_targets
9. Commit transaction
10. Emit DeploymentCreated event
```

The request body supports two targeting modes:
- **By endpoint IDs:** `endpoint_ids: ["uuid1", "uuid2"]` — targets specific endpoints
- **By OS family filter:** `endpoint_filter: "windows"` or `"linux"` — targets all matching OS endpoints

#### 2. Deploy Critical (Manual, Multi-Patch)
**Endpoint:** `POST /api/v1/endpoints/{id}/deploy-critical`
**Handler:** `PatchHandler.DeployCritical()` in `internal/server/api/v1/patches.go`

Creates ONE deployment targeting a single endpoint with multiple patches (instead of N separate deployments):

```go
// Request: { patch_ids: ["uuid1", "uuid2", "uuid3"], name: "Critical patches" }
// Creates: 1 deployment, 1 wave, N targets (one per patch)
```

#### 3. Policy-Driven Deployment (Automated)
**Trigger:** `PolicyEvalJob` (6-hour cycle)
**Files:** `internal/server/policy/evaluator.go`, `scheduler.go`, `worker.go`

The policy engine evaluates all enabled policies:
1. Resolve endpoint set via tag selectors
2. Match patches by severity filter and OS family
3. Create endpoint+patch target pairs
4. If auto-deploy is enabled, create a deployment automatically

#### 4. Scheduled Deployment (Cron-Based)
**Endpoint:** `POST /api/v1/deployment-schedules`
**Checker:** `ScheduleCheckerJob` (1-minute cycle)

Scheduled deployments use cron expressions:
1. Admin creates a schedule with cron expression and wave config
2. `ScheduleChecker.Check()` runs every minute
3. Finds due schedules via `ListDueSchedules()`
4. Checks no active deployment exists for this schedule's policy
5. Creates deployment with `CreateDeploymentWithWaveConfig()`
6. Computes next run time from cron expression
7. Updates schedule's `next_run_at`

### Deployment Executor

**File:** `internal/server/deployment/engine.go`

The Executor is a River worker that transitions a deployment from CREATED to RUNNING:

```go
func (e *Executor) Execute(ctx context.Context, deployID, tenantID pgtype.UUID) error {
    // 1. Transition CREATED → RUNNING via StateMachine
    _, startEvents, _ := e.sm.StartDeployment(ctx, e.q, deployID, tenantID)
    EmitBestEffort(ctx, e.eventBus, startEvents)

    // 2. Get wave 1 (first pending wave)
    wave, _ := e.q.GetCurrentWave(ctx, ...)

    // 3. Set wave 1's eligible_at to NOW (making it immediately dispatchable)
    e.q.SetWaveEligibleAt(ctx, sqlcgen.SetWaveEligibleAtParams{
        ID: wave.ID,
        EligibleAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
    })
}
```

This is a River job on the `"critical"` queue, ensuring deployments start promptly.

---

## 7. State Machine - Deployment States & Transitions

**File:** `internal/server/deployment/statemachine.go`

The `StateMachine` manages deployment lifecycle transitions. All transitions are enforced by both the Go code and database CHECK constraints.

### State Diagram

```
                                  ┌──────────────┐
                      ┌──────────>│  cancelled    │
                      │           └──────────────┘
                      │                 ▲
    ┌─────────┐  Start  ┌─────────┐  Cancel  
    │scheduled│──────>│ created │──────────────────────────┐
    └─────────┘       └─────────┘                          │
         │                │                                │
    Activate         Start │                               │
    (scheduled→      (created→running)                     │
     created)             │                                │
                    ┌─────▼─────┐  Cancel                  │
                    │  running   │─────────────────────>────┘
                    └─────┬─────┘
                          │
           ┌──────────────┼──────────────┐
           │              │              │
      Complete        Fail          Rollback
      (running→       (running→     (running→
       completed)      failed)       rolling_back)
           │              │              │
    ┌──────▼──────┐ ┌─────▼─────┐ ┌─────▼──────────┐
    │  completed   │ │  failed   │ │  rolling_back   │
    └─────────────┘ └─────┬─────┘ └──────┬──────────┘
                          │              │
                     Retry          ┌────┴────┐
                     (failed→       │         │
                      running)   Success   Failure
                                    │         │
                             ┌──────▼──┐ ┌────▼───────────┐
                             │rolled_back│ │rollback_failed │
                             └──────────┘ └───────────────┘
```

### Valid Transitions (Enforced by DB CHECK Constraints)

| From State | To State | Method | Triggers |
|------------|----------|--------|----------|
| `scheduled` | `created` | `ActivateScheduled()` | ScheduleChecker finds due schedule |
| `created` | `running` | `StartDeployment()` | Executor River job fires |
| `created` | `cancelled` | `CancelDeployment()` | User cancels via API |
| `running` | `completed` | `CompleteDeployment()` | All waves succeed |
| `running` | `failed` | `FailDeployment()` | Failure threshold exceeded |
| `running` | `cancelled` | `CancelDeployment()` | User cancels via API |
| `running` | `rolling_back` | `RollbackDeployment()` | User or auto-rollback triggers |
| `rolling_back` | `rolled_back` | `RollbackDeployment()` | Rollback commands succeed |
| `rolling_back` | `rollback_failed` | `RollbackDeployment()` | Rollback command cancellation fails |
| `failed` | `running` | `RetryDeployment()` | User retries failed targets |

### StateMachine Implementation Details

Each transition method:
1. Updates the deployment record in PostgreSQL
2. Returns domain events (but does NOT emit them)
3. The caller emits events AFTER successful transaction commit
4. This prevents "phantom events" — events emitted for rolled-back transactions

```go
// Example: CancelDeployment does 3 DB operations in one transaction
func (sm *StateMachine) CancelDeployment(ctx, q, deployID, tenantID) {
    q.SetDeploymentCancelled(...)         // Update deployment status
    q.CancelDeploymentTargets(...)        // Cancel all pending targets
    q.CancelCommandsByDeployment(...)     // Cancel all pending commands
    return deployment, []DomainEvent{DeploymentCancelled}, nil
}
```

### Rollback Flow (Detailed)

```go
func (sm *StateMachine) RollbackDeployment(ctx, q, deployID, tenantID) {
    // Step 1: RUNNING → ROLLING_BACK
    q.SetDeploymentRollingBack(...)
    events = [DeploymentRollbackTriggered]

    // Step 2: Cancel remaining waves
    q.CancelRemainingWaves(...)

    // Step 3: Cancel wave targets
    q.CancelWaveTargets(...)

    // Step 4: Cancel pending commands
    err := q.CancelCommandsByDeployment(...)
    if err != nil {
        // If cancellation fails → ROLLBACK_FAILED
        q.SetDeploymentRollbackFailed(...)
        events += [DeploymentRollbackFailed]
        return
    }

    // Step 5: ROLLING_BACK → ROLLED_BACK
    q.SetDeploymentRolledBack(...)
    events += [DeploymentRolledBack]
}
```

### Retry Flow

```go
func (sm *StateMachine) RetryDeployment(ctx, q, deployID, tenantID) {
    // FAILED → RUNNING
    q.SetDeploymentRetrying(...)

    // Reset all failed targets back to pending
    affected, _ := q.RetryFailedTargets(...)
    if affected == 0 {
        return error("no failed targets found to retry")
    }

    return deployment, [DeploymentRetryTriggered], nil
}
```

---

## 8. Wave-Based Deployment - Phased Rollout

### WaveConfig Structure

**File:** `internal/server/deployment/waves.go`

```go
type WaveConfig struct {
    Percentage       int     `json:"percentage"`        // % of total targets in this wave
    SuccessThreshold float64 `json:"success_threshold"` // Required success rate (0.0-1.0)
    ErrorRateMax     float64 `json:"error_rate_max"`    // Max failure rate before rollback
    DelayMinutes     int     `json:"delay_minutes"`     // Wait time after previous wave
}
```

**Example multi-wave config:**
```json
[
    {"percentage": 10, "success_threshold": 0.95, "error_rate_max": 0.10, "delay_minutes": 0},
    {"percentage": 30, "success_threshold": 0.90, "error_rate_max": 0.15, "delay_minutes": 30},
    {"percentage": 60, "success_threshold": 0.80, "error_rate_max": 0.20, "delay_minutes": 60}
]
```

This deploys to 10% first, waits for 95% success, then 30 minutes later deploys to 30%, then 60 minutes later to the remaining 60%.

**Default (Quick Deploy):** Single wave, 100% of targets, 80% success threshold, 20% max error rate, no delay.

### Target Assignment

```go
// AssignTargetsToWaves distributes targetCount across waves by percentage.
// The last wave gets any remainder to ensure all targets are assigned.
func AssignTargetsToWaves(waves []WaveConfig, targetCount int) []int {
    // Example: 100 targets with [10%, 30%, 60%] → [10, 30, 60]
    // Example: 7 targets with [10%, 30%, 60%] → [0, 2, 5]
}
```

### Wave Dispatcher

**File:** `internal/server/deployment/wave_dispatcher.go`
**Cycle:** Every 30 seconds (River periodic job on `"critical"` queue)

The Wave Dispatcher is the heartbeat of the deployment engine. Every 30 seconds, it:

```
1. List all tenant IDs with running deployments
   └── For each tenant:
       2. List running deployments
          └── For each deployment:
              3. Get current wave (first pending or running wave)
              4. If wave is PENDING and eligible_at has passed:
                 → Transition to RUNNING
                 → Emit DeploymentWaveStarted
              5. If wave is RUNNING:
                 a. Dispatch pending targets (create commands)
                    - Respect max_concurrent throttle
                    - Check maintenance windows
                    - Build InstallPatchPayload (package name, version, download URL, checksum, silent args)
                    - Create Command record
                    - Mark target as "sent"
                    - Emit CommandDispatched + DeploymentTargetSent events
                 b. Check wave completion
                    - If active/pending targets remain → skip
                    - If failure_rate > error_rate_max → fail wave + trigger rollback
                    - If success_rate >= success_threshold → complete wave + advance
              6. If no active waves remain → Complete deployment
```

### Maintenance Window Enforcement

Before dispatching to an endpoint, the wave dispatcher checks its maintenance window:

```go
mwData, _ := q.GetEndpointMaintenanceWindow(ctx, endpointID, tenantID)
mw, _ := ParseMaintenanceWindow(mwData)
if !IsInMaintenanceWindow(mw, time.Now()) {
    continue  // Skip this endpoint for now
}
```

Endpoints outside their maintenance window are skipped but NOT failed — they'll be dispatched in a future cycle when the window opens.

### Max Concurrent Throttle

Deployments can set `max_concurrent` to limit how many targets are actively being patched at once:

```go
activeCount, _ := q.CountActiveTargets(ctx, deploymentID, tenantID)
if maxConcurrent > 0 && activeCount >= maxConcurrent {
    break  // Stop dispatching more targets
}
```

### Wave Completion Evaluation

```go
func checkWaveCompletion(wave) {
    failureRate := float64(wave.FailedCount) / float64(wave.TargetCount)

    if failureRate > errorRateMax {
        // FAILURE: Too many targets failed
        → SetWaveFailed()
        → sm.RollbackDeployment()  // Auto-rollback
        → Emit DeploymentWaveFailed + DeploymentRollbackTriggered
        return
    }

    successRate := float64(wave.SuccessCount) / float64(wave.TargetCount)

    if successRate >= successThreshold {
        // SUCCESS: Wave met its success criteria
        → SetWaveCompleted()
        → Emit DeploymentWaveCompleted
        → advanceToNextWave() or CompleteDeployment()
    }

    // Otherwise: still in progress (active/pending targets remain)
}
```

### Wave Advancement

```go
func advanceToNextWave(completedWave) {
    // Find next pending wave after the completed one
    nextWave := findNextPendingWave(waves, completedWave.WaveNumber)

    if nextWave == nil {
        // No more waves — deployment complete
        sm.CompleteDeployment()
        return
    }

    // Schedule next wave with delay
    eligibleAt := time.Now().Add(completedWave.DelayAfterMinutes * time.Minute)
    q.SetWaveEligibleAt(nextWave.ID, eligibleAt)
    // Wave dispatcher will pick it up in a future 30s cycle
}
```

### Command Payload Construction

When dispatching a target, the wave dispatcher builds the protobuf payload:

```go
installPayload := &pb.InstallPatchPayload{
    Packages: []*pb.PatchTarget{{
        Name:    patch.PackageName,          // e.g., "curl" or "KB5034441"
        Version: patch.Version,              // e.g., "7.88.1-10+deb12u5"
        Source:  installerTypeOrFallback(),  // e.g., "apt", "yum", "msi", "wua"
    }},
    DownloadUrl:    "/repo/files/windows/KB5034441.msu",  // If binary available
    ChecksumSha256: "abc123...",                           // For integrity verification
    SilentArgs:     "/quiet /norestart",                   // For EXE installers
}
```

The `Source` field determines which installer the agent uses. Mapping:

| OS Family | Source | Agent Installer |
|-----------|--------|-----------------|
| linux-debian, linux-ubuntu | `apt` | APT installer |
| linux-rhel, linux-centos, linux-fedora | `yum` | YUM installer |
| macos, darwin | `homebrew` | Homebrew installer |
| windows | `msi` (legacy) or explicit `wua`, `msi`, `msix`, `exe` | Windows-specific |

---

## 9. Agent Patcher Module - Installation Execution

**File:** `internal/agent/patcher/patcher.go`

The Patcher Module is the agent-side component that actually installs patches on endpoints.

### Module Interface

```go
type Module struct {
    Name()              → "patcher"
    Version()           → "0.1.0"
    Capabilities()      → ["patch_installation"]
    SupportedCommands() → ["install_patch", "rollback_patch"]
}
```

### Concurrency Control

```go
// Dynamic semaphore: reads max_concurrent_installs from settings at acquire time
sem := newDynamicSem(func() int { return settings.MaxConcurrentInstalls })

// Before every install or rollback:
sem.Acquire()   // Blocks if at capacity
defer sem.Release()
```

Default: 1 concurrent installation. Configurable via `patcher.max_concurrent_installs` setting.

### Install Patch Flow (`handleInstallPatch`)

```
1. Acquire semaphore (respects max_concurrent_installs)
2. Apply timeout (default: 30 minutes)
3. Deserialize InstallPatchPayload (protobuf)
4. Download binary (if download_url provided)
   └── GET {server_http_url}{download_url}
   └── SHA256 checksum verification
   └── Override package name with local file path
5. Validate packages (name not empty)
6. Capture pre-install versions (for rollback tracking)
7. Execute pre-script (if provided)
   └── If pre-script fails (exit code != 0) → return failure
8. Install packages (one by one)
   └── Resolve OS-specific installer (by source or auto-detect)
   └── For EXE installers: inject silent_args from payload
   └── Call installer.Install(ctx, target, dryRun)
   └── Record per-package results (exit code, stdout, stderr, reboot_required)
9. Save rollback records (for each successful install)
   └── RollbackRecord: {command_id, package_name, from_version, to_version, status: "pending"}
10. Execute post-script (runs regardless of install outcome)
11. Auto-reboot (if any package requires reboot and auto-reboot is enabled)
    └── Default 60-second grace period
12. Marshal output as protobuf InstallPatchOutput
13. Return Result with aggregated success/failure
```

### OS-Specific Installers

The patcher auto-detects available installers at init time:

| Installer | File | Platform | Command |
|-----------|------|----------|---------|
| APT | `apt.go` | Debian/Ubuntu | `apt-get install -y {package}={version}` |
| YUM | `yum.go` | RHEL/CentOS | `yum install -y {package}-{version}` |
| WUA | `wua.go` | Windows | Windows Update Agent API |
| MSI | `msi_windows.go` | Windows | `msiexec /i {file} /quiet /norestart` |
| MSIX | `msix_windows.go` | Windows | `Add-AppxPackage` PowerShell |
| EXE | `exe_windows.go` | Windows | `{file} {silent_args}` |
| Homebrew | `homebrew.go` | macOS | `brew install {package}@{version}` |
| macOS SoftwareUpdate | `softwareupdate.go` | macOS | `softwareupdate --install {package}` |

### Rollback Support (3 Modes)

**File:** `internal/agent/patcher/patcher.go` — `handleRollback()`

The rollback handler supports three modes, tried in order:

#### Mode 1: Server-Specified Revert Targets
```protobuf
RollbackPatchPayload {
    revert_to: [PatchTarget{name: "curl", version: "7.88.0", source: "apt"}]
}
```
Server knows the exact version to revert to. Agent installs the specified version directly.

#### Mode 2: Protobuf with OriginalCommandId
```protobuf
RollbackPatchPayload {
    original_command_id: "01HXYZ..."
}
```
Agent looks up local rollback records by command ID, finds `from_version`, and installs that version.

#### Mode 3: JSON Payload (Backward Compatibility)
```json
{"command_id": "01HXYZ..."}
```
Legacy JSON format — same behavior as Mode 2.

### Rollback Record Storage

```go
type RollbackRecord struct {
    ID          string  // ULID
    CommandID   string  // Original install command ID
    PackageName string  // e.g., "curl"
    FromVersion string  // Version before install (empty if new install)
    ToVersion   string  // Version after install
    Status      string  // "pending", "completed", "failed"
}
```

Stored in local SQLite. Records are created after each successful package install and consumed during rollback.

### Binary Download Flow

```go
// 1. Construct full URL
fullURL := serverHTTPURL + payload.DownloadUrl
// e.g., "http://192.168.1.17:8180/repo/files/windows/KB5034441.msu"

// 2. Download to temp directory
localPath, err := downloader.Download(ctx, fullURL, payload.ChecksumSha256)
// Downloads to: {data_dir}/patch-downloads/ or {os_temp}/patchiq-patch-downloads/

// 3. SHA256 verification
// If checksum doesn't match → return error

// 4. Override package names with local file path
for _, pkg := range payload.Packages {
    pkg.Name = localPath  // Installer will use the downloaded file
}

// 5. Cleanup after install
defer os.Remove(localPath)
```

### Agent Inventory Collection

The inventory module collects installed package data from endpoints:

| Collector | File | Platform | Data Source |
|-----------|------|----------|-------------|
| APT | `inventory/apt.go` | Debian/Ubuntu | `dpkg` database |
| YUM | `inventory/yum.go` | RHEL/CentOS | `rpm` database |
| WUA | `inventory/wua.go` | Windows | Windows Update Agent |
| Hotfix | `inventory/hotfix.go` | Windows | KB articles (Get-HotFix) |
| Homebrew | `inventory/homebrew.go` | macOS | Homebrew package list |
| SoftwareUpdate | `inventory/softwareupdate.go` | macOS | macOS native updates |

Inventory data is sent to the server via `SyncOutbox` gRPC stream and stored in `endpoint_packages` and `endpoint_inventories` tables. This data is used for CVE correlation and patch applicability matching.

---

## 10. Policy Engine - Automated Deployment

**Files:** `internal/server/deployment/evaluator.go`, `internal/server/policy/`

### Policy Evaluation

The `Evaluator` resolves a policy into deployment targets:

```go
func (e *Evaluator) Evaluate(ctx, q, policyID, tenantID) (*EvalResult, error) {
    // 1. Fetch policy (must be enabled)
    policy := q.GetPolicyByID(policyID, tenantID)

    // 2. Resolve endpoint set via tag selector
    endpointIDs := e.resolver.ResolveForPolicy(ctx, tenantStr, policyStr)
    // Uses targeting.Resolver which evaluates tag selector expressions

    // 3. Hydrate endpoint data (ID, hostname, os_family, status)
    endpoints := q.ListEndpointsByIDs(tenantID, endpointIDs)

    // 4. Build severity filter
    severityFilter := BuildSeverityFilter(policy)
    // by_severity mode: ["critical"] if min_severity="critical"
    //                   ["critical", "high"] if min_severity="high"
    //                   etc.

    // 5. Find matching patches
    patches := q.ListPatchesForPolicyFilters(tenantID, severityFilter, osFamilies)

    // 6. Cross-match endpoints × patches by OS family
    targets := matchTargets(endpoints, patches)
    // Each endpoint gets every patch for its OS family

    return &EvalResult{Policy, Endpoints, Patches, Targets}
}
```

### Severity Filter Logic

```go
func BuildSeverityFilter(policy) []string {
    // Priority 1: Use policy's explicit severity_filter if set
    if len(policy.SeverityFilter) > 0 {
        return policy.SeverityFilter
    }

    // Priority 2: Build from selection_mode + min_severity
    if policy.SelectionMode != "by_severity" {
        return nil  // all_available or by_cve_list — no severity filtering
    }

    // Rank: low=1, medium=2, high=3, critical=4
    // min_severity="high" → returns ["high", "critical"]
    // min_severity="medium" → returns ["medium", "high", "critical"]
}
```

### Policy Selection Modes

| Mode | Behavior |
|------|----------|
| `all_available` | All patches matching the endpoint's OS |
| `by_severity` | Only patches at or above `min_severity` |
| `by_cve_list` | Only patches linked to specific CVEs |

---

## 11. CVE & Vulnerability Integration

**Files:** `internal/server/cve/job.go`, `internal/server/api/v1/cves.go`

### CVE Sync Jobs

| Job | Cycle | Purpose |
|-----|-------|---------|
| `NVDSyncJob` | 24 hours | Fetches NVD database, updates CVE records |
| `EndpointMatchJob` | On scan | Correlates endpoint packages to CVEs |

### CVE-Patch-Endpoint Correlation

```
CVE (cves table)
  ↓ patch_cves (many-to-many with version ranges)
Patch (patches table)
  ↓ deployment_targets (which endpoints get this patch)
Endpoint (endpoints table)
  ↓ endpoint_cves (which CVEs affect this endpoint)
  ↓ endpoint_packages (what's installed)
```

The `EndpointMatchJob` runs after each inventory scan and:
1. Compares installed package versions against CVE version ranges
2. Creates/updates `endpoint_cves` records
3. Calculates per-patch remediation status

### Patch Remediation Status

```sql
-- GetPatchRemediationStatus query
endpoints_affected: count of endpoints with vulnerable version
endpoints_patched:  count of endpoints with fixed version installed
endpoints_pending:  count of endpoints with pending deployment
endpoints_failed:   count of endpoints where deployment failed
```

---

## 12. REST API Reference

### Patch Endpoints

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| `GET` | `/api/v1/patches` | `PatchHandler.List()` | List patches with pagination & filters |
| `GET` | `/api/v1/patches/{id}` | `PatchHandler.Get()` | Detailed patch view |
| `POST` | `/api/v1/patches/{id}/deploy` | `PatchHandler.QuickDeploy()` | Immediate patch deployment |
| `POST` | `/api/v1/endpoints/{id}/deploy-critical` | `PatchHandler.DeployCritical()` | Multi-patch deployment to one endpoint |

### Deployment Endpoints

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| `GET` | `/api/v1/deployments` | `DeploymentHandler.List()` | List deployments with pagination |
| `GET` | `/api/v1/deployments/{id}` | `DeploymentHandler.Get()` | Deployment details |
| `POST` | `/api/v1/deployments` | `DeploymentHandler.Create()` | Create manual deployment with wave config |
| `POST` | `/api/v1/deployments/{id}/cancel` | `DeploymentHandler.Cancel()` | Cancel deployment |
| `POST` | `/api/v1/deployments/{id}/retry` | `DeploymentHandler.Retry()` | Retry failed targets |
| `POST` | `/api/v1/deployments/{id}/rollback` | `DeploymentHandler.Rollback()` | Rollback deployment |

### Schedule Endpoints

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| `POST` | `/api/v1/deployment-schedules` | `ScheduleHandler.Create()` | Create scheduled deployment |
| `PUT` | `/api/v1/deployment-schedules/{id}` | `ScheduleHandler.Update()` | Update schedule |

### Patch List Filters

| Parameter | Type | Description |
|-----------|------|-------------|
| `cursor` | string | Pagination cursor (encoded timestamp+UUID) |
| `limit` | int | Page size |
| `severity` | string | Filter by severity (critical, high, medium, low) |
| `os_family` | string | Filter by OS family |
| `status` | string | Filter by patch status |
| `search` | string | Free-text search on name/description |
| `sort_by` | string | Sort field |

### Patch Detail Response Shape

```json
{
    "id": "uuid",
    "tenant_id": "uuid",
    "name": "curl",
    "version": "7.88.1-10+deb12u5",
    "severity": "critical",
    "os_family": "linux-debian",
    "status": "available",
    "os_distribution": "debian-12",
    "package_url": "s3://patches/linux-debian/curl-7.88.1.deb",
    "checksum_sha256": "abc123...",
    "source_repo": "debian-security",
    "description": "Security update for curl",
    "created_at": "2025-01-15T10:00:00Z",
    "updated_at": "2025-01-15T10:00:00Z",
    "released_at": "2025-01-14T00:00:00Z",
    "file_size": null,
    "highest_cvss_score": 9.8,
    "avg_install_time_ms": 45000,
    "cves": [
        {
            "id": "uuid",
            "cve_id": "CVE-2024-1234",
            "severity": "critical",
            "cvss_v3_score": "9.8",
            "cvss_v3_vector": "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
            "attack_vector": "Network",
            "published_at": "2024-12-01T00:00:00Z",
            "exploit_available": true,
            "cisa_kev": true,
            "description": "Buffer overflow in curl..."
        }
    ],
    "remediation": {
        "endpoints_affected": 150,
        "endpoints_patched": 80,
        "endpoints_pending": 50,
        "endpoints_failed": 20
    },
    "affected_endpoints": {
        "total": 150,
        "has_more": true,
        "items": [
            {
                "id": "uuid",
                "hostname": "web-server-01",
                "os_family": "linux-debian",
                "agent_version": "0.1.0",
                "status": "active",
                "patch_status": "pending",
                "last_deployed_at": null
            }
        ]
    },
    "deployment_history": [
        {
            "id": "uuid",
            "status": "completed",
            "triggered_by": "user-uuid",
            "started_at": "2025-01-15T10:05:00Z",
            "completed_at": "2025-01-15T10:45:00Z",
            "total_targets": 50,
            "success_count": 48,
            "failed_count": 2
        }
    ]
}
```

---

## 13. gRPC Protocol - Agent Communication

**Proto File:** `proto/patchiq/v1/agent.proto`

### AgentService RPCs

| RPC | Direction | Purpose |
|-----|-----------|---------|
| `Enroll` | Agent → Server | One-time agent registration |
| `Heartbeat` | Bidirectional stream | Periodic health check |
| `SyncOutbox` | Agent → Server (stream) | Agent sends command results, inventory |
| `SyncInbox` | Server → Agent (stream) | Server sends commands (install_patch, etc.) |

### Command Types

```protobuf
enum CommandType {
    COMMAND_TYPE_INSTALL_PATCH  = 1;   // Install patch on endpoint
    COMMAND_TYPE_RUN_SCAN      = 2;   // Trigger inventory scan
    COMMAND_TYPE_UPDATE_CONFIG = 3;   // Push configuration
    COMMAND_TYPE_REBOOT        = 4;   // Reboot endpoint
    COMMAND_TYPE_RUN_SCRIPT    = 5;   // Execute arbitrary script
    COMMAND_TYPE_ROLLBACK_PATCH = 6;  // Rollback installed patch
}
```

### InstallPatchPayload (Protobuf)

```protobuf
message InstallPatchPayload {
    repeated PatchTarget packages = 1;    // Packages to install
    bool dry_run = 2;                      // Simulate without installing
    string pre_script = 3;                 // Shell script to run before install
    string post_script = 4;                // Shell script to run after install
    string download_url = 5;               // Binary repo path (e.g., /repo/files/windows/KB.msu)
    string checksum_sha256 = 6;            // SHA256 for integrity verification
    string silent_args = 7;                // Silent install flags (/S, /quiet, etc.)
    bool auto_reboot = 8;                  // Trigger reboot after install
    int32 reboot_delay_seconds = 9;        // Grace period before reboot
}

message PatchTarget {
    string name = 1;      // Package name (e.g., "curl", "KB5034441")
    string version = 2;   // Target version
    string source = 3;    // Installer type (apt, yum, wua, msi, exe, homebrew)
}
```

### RollbackPatchPayload (Protobuf)

```protobuf
message RollbackPatchPayload {
    repeated PatchTarget revert_to = 1;      // Mode 1: explicit version targets
    string original_command_id = 2;          // Mode 2: look up from rollback store
    string deployment_id = 3;               // For logging/tracking
    bool auto_reboot = 4;
    int32 reboot_delay_seconds = 5;
}
```

### InstallPatchOutput (Protobuf)

```protobuf
message InstallPatchOutput {
    repeated InstallResultDetail results = 1;
    bool dry_run = 2;
    string pre_script_output = 3;
    string post_script_output = 4;
}

message InstallResultDetail {
    string package_name = 1;
    string version = 2;
    int32 exit_code = 3;
    string stdout = 4;
    string stderr = 5;
    bool reboot_required = 6;
    bool succeeded = 7;
}
```

---

## 14. Database Schema

### Server-Side Tables (Patch-Related)

**Migration 004:** Core patch tables

```sql
-- Tenant-scoped patch catalog
CREATE TABLE patches (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL REFERENCES tenants(id),
    name             TEXT NOT NULL,
    version          TEXT NOT NULL,
    severity         TEXT NOT NULL,              -- critical, high, medium, low
    os_family        TEXT NOT NULL,              -- windows, linux-debian, linux-rhel, macos
    status           TEXT NOT NULL,              -- available, superseded, recalled
    os_distribution  TEXT,
    package_url      TEXT,                       -- S3 path or external URL
    checksum_sha256  TEXT,
    source_repo      TEXT,
    description      TEXT,
    package_name     TEXT,                       -- Actual package name (e.g., "curl" vs advisory name)
    released_at      TIMESTAMPTZ,
    installer_type   TEXT,                       -- apt, yum, wua, msi, exe, homebrew
    silent_args      TEXT,                       -- Silent install arguments
    hub_catalog_id   UUID,                       -- Reference to hub's patch_catalog.id
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at       TIMESTAMPTZ,                -- Soft delete (migration 055)
    UNIQUE(tenant_id, name, version, os_family)  -- Upsert constraint (migration 006)
);

-- CVE associations with version ranges
CREATE TABLE patch_cves (
    patch_id              UUID REFERENCES patches(id),
    cve_id                UUID REFERENCES cves(id),
    version_end_excluding TEXT,          -- CVE affects versions < this
    version_end_including TEXT,          -- CVE affects versions <= this
    PRIMARY KEY (patch_id, cve_id)
);

-- Which CVEs affect which endpoints
CREATE TABLE endpoint_cves (
    endpoint_id UUID REFERENCES endpoints(id),
    cve_id      UUID REFERENCES cves(id),
    tenant_id   UUID NOT NULL,
    status      TEXT NOT NULL,           -- vulnerable, patched, pending
    PRIMARY KEY (endpoint_id, cve_id)
);

-- Installed packages per endpoint
CREATE TABLE endpoint_packages (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id UUID REFERENCES endpoints(id),
    tenant_id   UUID NOT NULL,
    name        TEXT NOT NULL,
    version     TEXT NOT NULL,
    source      TEXT,                    -- apt, yum, wua, etc.
    UNIQUE(endpoint_id, name, source)
);
```

**Migration 011:** Deployment engine tables

```sql
CREATE TABLE deployments (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL REFERENCES tenants(id),
    policy_id      UUID REFERENCES policies(id),
    patch_id       UUID REFERENCES patches(id),
    name           TEXT,
    status         TEXT NOT NULL,         -- scheduled, created, running, completed, failed,
                                         -- cancelled, rolling_back, rolled_back, rollback_failed
    wave_config    JSONB,                -- Array of WaveConfig objects
    max_concurrent INTEGER,              -- Max concurrent target installs
    total_targets  INTEGER DEFAULT 0,
    success_count  INTEGER DEFAULT 0,
    failed_count   INTEGER DEFAULT 0,
    created_by     UUID,
    started_at     TIMESTAMPTZ,
    completed_at   TIMESTAMPTZ,
    scheduled_at   TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE deployment_waves (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    deployment_id       UUID NOT NULL REFERENCES deployments(id),
    wave_number         INTEGER NOT NULL,
    status              TEXT NOT NULL,       -- pending, running, completed, failed, cancelled
    percentage          INTEGER NOT NULL,
    target_count        INTEGER DEFAULT 0,
    success_count       INTEGER DEFAULT 0,
    failed_count        INTEGER DEFAULT 0,
    success_threshold   NUMERIC,            -- Required success rate (0.0-1.0)
    error_rate_max      NUMERIC,            -- Max failure rate before rollback
    delay_after_minutes INTEGER DEFAULT 0,
    eligible_at         TIMESTAMPTZ,        -- When this wave becomes dispatchable
    started_at          TIMESTAMPTZ,
    completed_at        TIMESTAMPTZ,
    UNIQUE(deployment_id, wave_number)
);

CREATE TABLE deployment_targets (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL,
    deployment_id  UUID NOT NULL REFERENCES deployments(id),
    wave_id        UUID REFERENCES deployment_waves(id),
    endpoint_id    UUID NOT NULL REFERENCES endpoints(id),
    patch_id       UUID NOT NULL REFERENCES patches(id),
    status         TEXT NOT NULL,           -- pending, sent, succeeded, failed, cancelled
    started_at     TIMESTAMPTZ,
    completed_at   TIMESTAMPTZ,
    error_message  TEXT,
    stdout         TEXT,
    stderr         TEXT,
    exit_code      INTEGER
);

CREATE TABLE commands (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL,
    agent_id       UUID NOT NULL,          -- Target endpoint
    deployment_id  UUID REFERENCES deployments(id),
    target_id      UUID REFERENCES deployment_targets(id),
    type           TEXT NOT NULL,           -- install_patch, rollback_patch, run_scan, etc.
    payload        BYTEA NOT NULL,          -- Protobuf-encoded command payload
    priority       INTEGER DEFAULT 0,
    status         TEXT NOT NULL,           -- pending, sent, succeeded, failed, cancelled
    deadline       TIMESTAMPTZ,            -- Command timeout deadline
    completed_at   TIMESTAMPTZ,
    error_message  TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE deployment_schedules (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    policy_id       UUID NOT NULL REFERENCES policies(id),
    cron_expression TEXT NOT NULL,          -- 5-field cron (minute hour dom month dow)
    wave_config     JSONB,
    max_concurrent  INTEGER,
    created_by      UUID,
    next_run_at     TIMESTAMPTZ,
    enabled         BOOLEAN DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### Key Migration Timeline

| Migration | Change |
|-----------|--------|
| 004 | Core tables: patches, endpoint_cves, endpoint_packages |
| 006 | Unique constraint on (tenant_id, name, version, os_family) |
| 011 | Deployment engine: deployments, waves, targets, commands |
| 017 | Deployment waves enhancements |
| 032 | Added `released_at` to patches |
| 034 | Quick deploy support |
| 042 | Package name normalization (added `package_name` column) |
| 044 | CVE version ranges (version_end_excluding, version_end_including) |
| 055 | Patch soft delete (added `deleted_at` column) |

### Key SQL Queries

**File:** `internal/server/store/queries/patches.sql` (40+ queries)

| Query | Purpose |
|-------|---------|
| `UpsertDiscoveredPatch` | Insert/update patch on name+version+os_family |
| `GetPatchByID` | Fetch single patch |
| `ListPatchesFiltered` | Paginated list with severity/os/status/search filters |
| `CountPatchesFiltered` | Total count for pagination |
| `CountPatchesBySeverity` | Aggregate counts by severity |
| `GetPatchRemediation` | Count affected/patched/pending/failed endpoints |
| `ListAffectedEndpointsForPatch` | Which endpoints need this patch |
| `ListDeploymentsForPatch` | Active deployments for this patch |
| `ListDeploymentHistoryForPatch` | Past deployment attempts |
| `GetPatchHighestCVSS` | Max CVSS score among linked CVEs |
| `LinkPatchCVE` | Associate CVE to patch |
| `SoftDeletePatchesByHubIDs` | Mark patches deleted when Hub removes them |
| `ListPatchesForPolicyFilters` | Patches matching policy severity + OS filters |

### Multi-Tenancy & RLS

- **Every tenant-scoped table** has `tenant_id UUID NOT NULL` as the first column after PK
- PostgreSQL Row Level Security (RLS) policies enforce tenant isolation
- **Every `BeginTx()`** must call `SET LOCAL app.current_tenant_id = $tenant_id`
- The deployment engine uses `TxFactory` patterns to ensure tenant-scoped transactions:
  ```go
  type WaveDispatcherTxFactory func(ctx context.Context, tenantID string) (
      WaveDispatcherQuerier, func() error, func() error, error)
  ```

---

## 15. Domain Events

**File:** `internal/server/events/topics.go`

PatchIQ uses a Watermill-based event bus with PostgreSQL transport. Every write operation MUST emit a domain event.

### Patch-Specific Events

| Event | Trigger |
|-------|---------|
| `patch.discovered` | New patch synced from Hub or discovered locally |
| `repository.synced` | Patch repository scan completed |
| `catalog.synced` | Hub catalog sync completed |
| `catalog.sync_started` | Hub catalog sync began |
| `catalog.sync_failed` | Hub catalog sync failed |

### Deployment Events

| Event | Trigger |
|-------|---------|
| `deployment.created` | New deployment record created |
| `deployment.started` | CREATED → RUNNING transition |
| `deployment.completed` | All waves succeeded, deployment done |
| `deployment.failed` | Failure threshold exceeded |
| `deployment.cancelled` | User cancelled |
| `deployment.wave_started` | Wave transitioned to RUNNING |
| `deployment.wave_completed` | Wave met success threshold |
| `deployment.wave_failed` | Wave exceeded error rate max |
| `deployment.rollback_triggered` | RUNNING → ROLLING_BACK |
| `deployment.rolled_back` | Rollback completed successfully |
| `deployment.rollback_failed` | Rollback could not complete |
| `deployment.retry_triggered` | FAILED → RUNNING (retry) |
| `deployment.endpoint_completed` | Single target finished |
| `deployment_target.sent` | Command dispatched to agent |
| `deployment_target.timed_out` | Command exceeded deadline |

### Command Events

| Event | Trigger |
|-------|---------|
| `command.dispatched` | Command created for agent |
| `command.timed_out` | Command exceeded its deadline |
| `command.result.received` | Agent sent back result |

### Event Emission Pattern

Events are NEVER emitted inside transactions. The pattern is:

```go
// 1. Perform DB operations, collect events
deployment, events, err := sm.StartDeployment(ctx, q, deployID, tenantID)

// 2. Commit transaction
if err := tx.Commit(); err != nil { ... }

// 3. Emit events AFTER commit (best-effort)
EmitBestEffort(ctx, eventBus, events)
```

`EmitBestEffort` logs failures but does NOT return errors — the DB state is authoritative.

### Audit Subscriber

A wildcard subscriber (`*`) captures ALL events into the audit table:
- Append-only, ULID IDs
- Partitioned monthly
- Used for compliance reporting and audit trails

---

## 16. Background Jobs (River)

River is the job queue used for all background processing. The server runs 12 workers.

### Deployment-Related Jobs

| Job | Kind | Queue | Interval | Purpose |
|-----|------|-------|----------|---------|
| `ExecutorWorker` | `deployment_executor` | `critical` | On-demand | Transitions CREATED → RUNNING, activates wave 1 |
| `WaveDispatcherWorker` | `wave_dispatcher` | `critical` | 30 seconds | Dispatches targets, checks wave completion |
| `ScheduleCheckerWorker` | `schedule_checker` | `critical` | 1 minute | Creates deployments from due schedules |
| `TimeoutWorker` | `deployment_timeout_checker` | `critical` | 5 minutes | Marks timed-out commands as failed |

### Other Relevant Jobs

| Job | Interval | Purpose |
|-----|----------|---------|
| `DiscoveryJob` | 60 minutes | Scans patch repositories |
| `NVDSyncJob` | 24 hours | Syncs NVD CVE database |
| `EndpointMatchJob` | On scan | CVE-endpoint correlation |
| `ComplianceEvalJob` | 6 hours | Framework compliance evaluation |
| `CatalogSyncJob` | On event | Syncs patches from Hub |
| `AuditRetentionJob` | 24 hours | Prunes old audit partitions |
| `PolicyEvalJob` | Periodic | Policy evaluation cycle |
| `FeedSyncJob` (Hub) | Per-feed | 6-12 hour feed sync |

### Timeout Checker Details

**File:** `internal/server/deployment/timeout.go`

Runs every 5 minutes. For each timed-out command:

```go
func processTimedOutCommand(cmd) {
    // 1. Mark command as failed
    q.UpdateCommandStatus(cmd.ID, "failed", "command timed out")

    // 2. Mark linked deployment target as failed
    if cmd.TargetID.Valid {
        q.UpdateDeploymentTargetStatus(cmd.TargetID, "failed", "command timed out")
    }

    // 3. Increment deployment failure counters
    if cmd.DeploymentID.Valid {
        d := q.IncrementDeploymentCounters(cmd.DeploymentID, isSuccess: false)
        // Check if deployment should fail or complete
        checkDeploymentThreshold(sm, d, deploymentID, tenantID)
    }

    // 4. Emit events
    EmitBestEffort(events: [CommandTimedOut, DeploymentTargetTimedOut])
}
```

### Result Handler Details

**File:** `internal/server/deployment/results.go`

Processes command results received from agents via gRPC `SyncOutbox`:

```go
func HandleResult(commandID, tenantID, succeeded, stdout, stderr, errMsg, exitCode) {
    // 1. Look up command
    cmd := q.GetCommandByID(commandID)

    // 2. Update command status (succeeded/failed)
    q.UpdateCommandStatus(commandID, status, completedAt, errorMessage)

    // 3. Update deployment target (if linked)
    if cmd.TargetID.Valid {
        q.UpdateDeploymentTargetStatus(targetID, status, startedAt, completedAt,
            errorMessage, stdout, stderr, exitCode)
        // Also increment wave counters
        q.IncrementWaveCounters(waveID, isSuccess)
    }

    // 4. Increment deployment counters + check completion/failure
    if cmd.DeploymentID.Valid {
        d := q.IncrementDeploymentCounters(deploymentID, isSuccess)
        checkDeploymentThreshold(sm, d, deploymentID, tenantID)
    }

    // 5. Commit + emit events
}
```

---

## 17. Frontend (React UI)

### Patch Manager UI (`web/`)

The richest UI of the three platforms, with 31 routes.

#### Patch Pages

| File | Route | Purpose |
|------|-------|---------|
| `pages/patches/PatchesPage.tsx` | `/patches` | Main patch list with filters & search |
| `pages/patches/PatchDetailPage.tsx` | `/patches/{id}` | Detailed view: CVEs, affected endpoints, deployment history |
| `pages/patches/PatchExpandedRow.tsx` | — | Inline expansion in table rows |
| `pages/patches/DeploymentModal.tsx` | — | Manual deployment creation dialog |
| `pages/patches/PatchDeploymentDialog.tsx` | — | Quick-deploy dialog |
| `pages/patches/columns.tsx` | — | TanStack Table column definitions |

#### API Hooks

**File:** `web/src/api/hooks/usePatches.ts`

```typescript
// List patches with cursor-based pagination
usePatches({
    cursor, limit, severity, os_family, status, search, sort_by
}) → { data, isLoading, error }
// 60-second cache/refetch interval

// Get patch detail
usePatch(id) → { data: PatchDetail }
```

**File:** `web/src/api/hooks/usePatchDeploy.ts`

```typescript
// Trigger patch deployment
usePatchDeploy() → mutation({
    patchId, name, description, config_type, scope, target_endpoints, endpoint_ids
})
// Auto-invalidates deployments/endpoints queries on success
```

#### TypeScript Types

**File:** `web/src/types/patches.ts`

```typescript
interface PatchListItem {
    id: string
    name: string
    version: string
    severity: string           // critical, high, medium, low
    os_family: string
    status: string
    created_at: string
    released_at: string
    os_distribution?: string
    description?: string
    cve_count: number
    highest_cvss_score: number
    remediation_pct: number
    endpoints_deployed_count: number
    affected_endpoint_count: number
}

interface PatchDetail {
    // All PatchListItem fields plus:
    cves: PatchCVE[]
    remediation: { endpoints_affected, patched, pending, failed }
    affected_endpoints: { total, items: AffectedEndpoint[], has_more }
    deployment_history: DeploymentHistoryItem[]
    highest_cvss_score: number
    avg_install_time_ms?: number
}
```

### Tech Stack

- **React 19** + **TypeScript 5.7** strict
- **Vite 6.2** (dev server, proxies `/api` to Go backend)
- **TanStack Query 5** (server state, 60s refetch)
- **TanStack Table 8** (data tables with sorting/filtering)
- **react-hook-form 7** + **Zod 4** (form validation)
- **openapi-fetch** + **openapi-typescript** (type-safe API client)
- **Tailwind CSS 4.2** + **Radix UI** (via @patchiq/ui)
- **@xyflow/react** + **elkjs** (workflow builder DAG canvas)
- **Recharts** (charts and visualizations)

---

## 18. Configuration & Environment

### Configuration Files

| File | Platform | Purpose |
|------|----------|---------|
| `configs/server.yaml` | Patch Manager | Discovery schedule, CVE sync, deployment defaults |
| `configs/hub.yaml` | Hub Manager | Feed sources, catalog sync intervals |
| `configs/agent.yaml` | Agent | Installer paths, polling intervals, server URL |
| `.env` | All | Per-developer port offsets, database URLs |

### Key Environment Variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `PATCHIQ_DATABASE_URL` | PostgreSQL connection string | — |
| `PATCHIQ_GRPC_PORT` | gRPC server port | `:50051` |
| `PATCHIQ_HTTP_PORT` | HTTP API port | `:8080` |
| `PATCHIQ_LOG_LEVEL` | Log verbosity | `info` |
| `PATCHIQ_CONFIG_PATH` | YAML config file path | `configs/server.yaml` |

### Agent Configuration

| Config Key | Purpose | Default |
|------------|---------|---------|
| `server.http_url` | Patch Manager HTTP URL for binary downloads | — |
| `patcher.command_timeout` | Max time per install command | 30 minutes |
| `patcher.max_concurrent_installs` | Max parallel installations | 1 |
| `data_dir` | Local data directory | system temp |

---

## 19. Operational Parameters & Limits

| Parameter | Value | Configurable | Location |
|-----------|-------|-------------|----------|
| **Max patch install timeout** | 30 minutes | Yes | `patcher.command_timeout` |
| **Max concurrent installs per agent** | 1 | Yes (runtime) | `patcher.max_concurrent_installs` |
| **Wave dispatch interval** | 30 seconds | No | Hardcoded in River periodic job |
| **Deployment timeout check interval** | 5 minutes | No | Hardcoded in River periodic job |
| **Schedule check interval** | 1 minute | No | Hardcoded in River periodic job |
| **Policy evaluation cycle** | 6 hours | Configurable | Policy scheduler config |
| **Discovery job cycle** | 60 minutes | Configurable | Server config |
| **NVD sync cycle** | 24 hours | Configurable | CVE job config |
| **Hub feed sync (NVD)** | 6 hours | Configurable | Hub config |
| **Hub feed sync (others)** | 12 hours | Configurable | Hub config |
| **Default success threshold** | 80% | Per-wave | WaveConfig |
| **Default failure threshold** | 20% | Per-wave | WaveConfig (`DefaultFailureThreshold`) |
| **Auto-reboot grace period** | 60 seconds | Per-payload | `reboot_delay_seconds` |
| **Command deadline** | Set at dispatch time | Per-command | `commandTimeout` param |
| **River workers** | 12 | Configurable | Server startup config |
| **Affected endpoints page limit** | 50 | Hardcoded | `PatchHandler.Get()` |
| **API hook refetch interval** | 60 seconds | Hardcoded | `usePatches.ts` |
| **Audit partitions** | Monthly | Automatic | `AuditRetentionJob` |
| **Binary download timeout** | 10 minutes | Hardcoded | `http.Client{Timeout: 10 * time.Minute}` |

---

## 20. Key File Index

### Backend - Patch Management

| File | Purpose |
|------|---------|
| `internal/server/api/v1/patches.go` | REST API: List, Get, QuickDeploy, DeployCritical |
| `internal/server/deployment/statemachine.go` | Deployment state transitions |
| `internal/server/deployment/engine.go` | Executor: CREATED → RUNNING |
| `internal/server/deployment/wave_dispatcher.go` | Wave dispatch loop (30s cycle) |
| `internal/server/deployment/waves.go` | WaveConfig parsing & target assignment |
| `internal/server/deployment/evaluator.go` | Policy → endpoint+patch matching |
| `internal/server/deployment/results.go` | Process agent command results |
| `internal/server/deployment/timeout.go` | Timeout checker (5min cycle) |
| `internal/server/deployment/schedule_checker.go` | Scheduled deployment checker (1min) |
| `internal/server/deployment/emit.go` | Best-effort event emission |
| `internal/server/api/v1/deployments.go` | REST API: deployments CRUD |
| `internal/server/api/v1/deployment_schedules.go` | REST API: deployment schedules |
| `internal/server/api/v1/policies.go` | REST API: policies |
| `internal/server/api/v1/cves.go` | REST API: CVE listing |
| `internal/server/cve/job.go` | CVE sync & endpoint matching |
| `internal/server/discovery/job.go` | Patch repository scanning |
| `internal/server/policy/evaluator.go` | Policy evaluation engine |
| `internal/server/policy/scheduler.go` | Policy evaluation scheduling |
| `internal/server/events/topics.go` | All domain event types |
| `internal/server/store/queries/patches.sql` | 40+ patch SQL queries |
| `internal/server/store/migrations/004_m1_core_tables.sql` | Core patch schema |
| `internal/server/store/migrations/011_deployment_engine.sql` | Deployment schema |

### Backend - Agent

| File | Purpose |
|------|---------|
| `internal/agent/patcher/patcher.go` | Patcher module: install & rollback |
| `internal/agent/patcher/executor.go` | Command execution abstraction |
| `internal/agent/patcher/download.go` | Binary download + checksum verification |
| `internal/agent/patcher/exe_windows.go` | Windows EXE installer |
| `internal/agent/patcher/privilege_windows.go` | Windows privilege elevation |
| `internal/agent/inventory/apt.go` | Debian/Ubuntu inventory collector |
| `internal/agent/inventory/yum.go` | RHEL/CentOS inventory collector |
| `internal/agent/inventory/wua.go` | Windows Update Agent collector |
| `internal/agent/inventory/hotfix.go` | Windows KB article collector |
| `internal/agent/inventory/homebrew.go` | macOS Homebrew collector |
| `internal/agent/api/patches.go` | Agent local HTTP API |
| `internal/agent/store/` | SQLite storage |

### Backend - Hub

| File | Purpose |
|------|---------|
| `internal/hub/feeds/nvd.go` | NVD feed (6h sync) |
| `internal/hub/feeds/cisa_kev.go` | CISA KEV feed (12h sync) |
| `internal/hub/feeds/msrc.go` | MSRC feed (12h sync) |
| `internal/hub/feeds/redhat.go` | RedHat OVAL feed (12h sync) |
| `internal/hub/feeds/ubuntu.go` | Ubuntu USN feed (12h sync) |
| `internal/hub/feeds/apple.go` | Apple feed (12h sync) |
| `internal/hub/catalog/pipeline.go` | Normalization pipeline |
| `internal/hub/catalog/minio*.go` | S3-compatible binary storage |
| `internal/hub/store/queries/patch_catalog.sql` | Hub catalog queries |

### Protocol

| File | Purpose |
|------|---------|
| `proto/patchiq/v1/common.proto` | Shared types: CommandType, InstallPatchPayload, etc. |
| `proto/patchiq/v1/agent.proto` | AgentService: Enroll, Heartbeat, SyncOutbox, SyncInbox |
| `gen/patchiq/v1/` | Generated Go protobuf code |

### Frontend

| File | Purpose |
|------|---------|
| `web/src/pages/patches/PatchesPage.tsx` | Patch list page |
| `web/src/pages/patches/PatchDetailPage.tsx` | Patch detail page |
| `web/src/pages/patches/DeploymentModal.tsx` | Deployment creation dialog |
| `web/src/api/hooks/usePatches.ts` | Patch data fetching hooks |
| `web/src/api/hooks/usePatchDeploy.ts` | Deployment mutation hook |
| `web/src/types/patches.ts` | TypeScript type definitions |

### Documentation

| File | Purpose |
|------|---------|
| `docs/adr/007-hub-spoke-multi-site.md` | Hub-Spoke architecture decision |
| `docs/adr/024-grpc-bidirectional-streaming.md` | gRPC agent protocol decision |
| `docs/adr/014-river-job-queue.md` | River job queue decision |
| `docs/adr/013-watermill-event-bus.md` | Watermill event bus decision |
| `docs/blueprint/platform-overview.md` | Platform overview |

---

## Appendix: Quick Reference Cheat Sheet

### "How does a patch get from the internet to an endpoint?"

```
NVD/CISA/MSRC/Vendor → Hub Feeds (6-12h) → Hub patch_catalog
→ CatalogSync → Server patches table → Policy Eval or Manual Deploy
→ Deployment + Waves + Targets → Wave Dispatcher (30s) → Commands
→ gRPC SyncInbox → Agent Patcher → OS Installer → Result
→ gRPC SyncOutbox → ResultHandler → Wave/Deployment counters
→ Wave completion check → Next wave or Complete/Rollback
```

### "What happens when I click Quick Deploy?"

```
POST /api/v1/patches/{id}/deploy
→ PatchHandler.QuickDeploy()
→ [Transaction: Deployment + Wave + Targets]
→ Emit deployment.created
→ Executor River job fires (critical queue)
→ CREATED → RUNNING, activate wave 1
→ WaveDispatcher picks up in ≤30s
→ Creates install_patch commands
→ Agent receives via SyncInbox
→ Patcher module installs
→ Agent reports via SyncOutbox
→ ResultHandler updates counters
→ Wave completion → deployment.completed
```

### "What are all the deployment statuses?"

`scheduled` → `created` → `running` → `completed` | `failed` | `cancelled` | `rolling_back` → `rolled_back` | `rollback_failed`

### "What can trigger a rollback?"

1. **Automatic:** Wave failure rate exceeds `error_rate_max` threshold
2. **Manual:** User calls `POST /api/v1/deployments/{id}/rollback`
