# Agent Execution Pipeline & Deployment Architecture — Design Spec

**Date**: 2026-03-23
**Status**: Draft
**Scope**: M2/M3 — Binary distribution, agent commands, tags, deployment wizard, policy wiring, workflow enhancements

---

## Table of Contents

1. [Overview](#1-overview)
2. [Binary Distribution Pipeline](#2-binary-distribution-pipeline)
3. [Agent Execution Commands](#3-agent-execution-commands)
4. [Tags System (replacing Groups)](#4-tags-system-replacing-groups)
5. [Unified Deployment Wizard](#5-unified-deployment-wizard)
6. [Policy → Auto-Deployment Wiring](#6-policy--auto-deployment-wiring)
7. [Workflow Enhancements](#7-workflow-enhancements)
8. [RBAC & License Gating](#8-rbac--license-gating)
9. [Roadmap Items](#9-roadmap-items)

---

## 1. Overview

### Problem

The agent can collect inventory and hardware data, but cannot execute actions — installing patches, rebooting, running scripts. The deployment engine creates commands but the agent only handles `install_patch` and agent-local `rollback_patch`. The binary distribution layer is missing entirely — agents rely on public OS repos instead of our managed Hub catalog. The UI has disconnected deployment flows, dead buttons, and no policy-to-deployment automation.

### Solution

End-to-end pipeline from Hub binary storage through Server distribution to Agent execution, with:
- Full binary distribution (Hub → Server → Agent) using native package repos (Linux) and direct download (Windows/macOS)
- Six server-triggered agent commands with full dependency resolution
- Tags replacing Groups as the universal classification system
- Unified deployment wizard (right-side panel) with rich visuals
- Policy auto-deployment wiring
- Workflow engine enhancements with event-based triggers

### Architecture Overview

```
Hub Manager (SaaS)
├── Feed Sync (6 vendor feeds)
├── Catalog Pipeline (normalize, dedup, CVE link)
├── Binary Fetcher (downloads from vendor repos)
├── MinIO Object Storage (patch binaries)
├── Test Group Approval Workflow
└── Catalog Sync API (metadata + binaries)
         │
         │  gRPC HubService.SyncCatalog + HTTPS binary download
         ▼
Patch Manager Server (On-prem)
├── Catalog Sync Job (pulls metadata + binaries)
├── Binary Cache (local disk)
├── Native Repos: APT (/repo/apt/), YUM (/repo/yum/)
├── File Server: /repo/files/ (MSI, MSU, PKG, CAB)
├── Dependency Resolution (supersedence, prereqs, ordering)
├── Deployment Engine (state machine, waves, commands)
├── Policy Engine (evaluation → auto-deploy)
├── Workflow Engine (DAG orchestration)
└── gRPC SyncInbox (commands → agents)
         │
         │  gRPC bidirectional streams
         ▼
Agent (Endpoint)
├── Binary Download (from Server's repos/file server)
├── Native Install (APT/YUM/DPKG/RPM/MSI/PKG)
├── Command Handlers (install, rollback, reboot, script, scan, config)
├── Result Reporting (outbox → SyncOutbox → Server)
└── Post-reboot Verification
```

---

## 2. Binary Distribution Pipeline

### Design Decision

**Model A — Hub stores, Server caches, Agent downloads from Server.**

Hub downloads patch binaries from vendor sources, stores in MinIO. Server pulls and caches locally during catalog sync. Agent always downloads from Server, never from the internet. This supports air-gapped environments and gives the Server offline self-sufficiency.

### Hub Side

#### Binary Fetcher

New component within the Hub's catalog pipeline. After normalizing a catalog entry, the fetcher:

1. Resolves the vendor download URL from feed metadata (MSRC provides direct links, NVD links to vendor advisories, RedHat OVAL includes RPM URLs, Ubuntu USN includes .deb URLs, Apple provides .pkg URLs)
2. Downloads the binary
3. Computes SHA256 checksum
4. Uploads to MinIO with structured key: `patches/{os_family}/{os_version}/{filename}`
5. Updates catalog entry with `binary_ref` (MinIO object key) and `checksum_sha256`

#### MinIO Storage Layout

```
patches/
├── ubuntu/
│   └── 22.04/
│       ├── curl_7.88.1-10+deb12u5_amd64.deb
│       └── openssl_3.0.13-1ubuntu0.1_amd64.deb
├── rhel/
│   └── 9/
│       ├── curl-7.76.1-26.el9_3.3.x86_64.rpm
│       └── openssl-3.0.7-25.el9_3.x86_64.rpm
├── windows/
│   └── 11/
│       ├── KB5034441-x64.msu
│       └── dotnet-runtime-8.0.1-win-x64.msi
└── macos/
    └── 15/
        └── macOS-Sequoia-15.3-Update.pkg
```

#### Test Group Approval

Before publishing to clients, patches go through a Hub-side approval workflow:
1. Binary fetched → status: `pending_review`
2. Test group endpoints (tagged `wave:test-group`) install and validate
3. Admin approves → status: `approved`
4. Only `approved` patches are included in catalog sync to Patch Manager Servers
5. Admin can reject → status: `rejected` (excluded from sync)

### Server Side

#### Catalog Sync Job (Extended)

The existing `CatalogSyncJob` (River job) is extended:

1. Pull catalog metadata from Hub (existing)
2. For each new/updated catalog entry with a `binary_ref`:
   a. Download binary from Hub's MinIO via HTTPS
   b. Verify SHA256 checksum
   c. Store in local binary cache directory
   d. Update local catalog with `local_binary_path`
3. Regenerate native repository metadata (see below)

#### Native Repository Hosting

**APT Repository** (`/repo/apt/`):

Server hosts a valid APT repository:
```
/repo/apt/
├── dists/
│   └── patchiq/
│       └── main/
│           └── binary-amd64/
│               ├── Packages
│               ├── Packages.gz
│               └── Release
└── pool/
    └── main/
        ├── c/curl/
        │   └── curl_7.88.1-10+deb12u5_amd64.deb
        └── o/openssl/
            └── openssl_3.0.13-1ubuntu0.1_amd64.deb
```

Metadata regenerated after each catalog sync using `dpkg-scanpackages` or a Go-native implementation.

Agent configuration during enrollment adds:
```
# /etc/apt/sources.list.d/patchiq.list
deb [signed-by=/etc/apt/keyrings/patchiq.gpg] https://{server_addr}/repo/apt patchiq main
```

> **Security note**: The Server signs repo metadata with a GPG key generated during initial setup. The public key is distributed to agents during enrollment and installed at `/etc/apt/keyrings/patchiq.gpg`. This ensures agents verify package authenticity. Initial development/testing may use `[trusted=yes]` temporarily, but production deployments MUST use signed repos.

**YUM/DNF Repository** (`/repo/yum/`):

```
/repo/yum/
├── repodata/
│   ├── repomd.xml
│   ├── primary.xml.gz
│   └── filelists.xml.gz
└── packages/
    ├── curl-7.76.1-26.el9_3.3.x86_64.rpm
    └── openssl-3.0.7-25.el9_3.x86_64.rpm
```

Metadata regenerated using `createrepo_c` or a Go-native implementation.

Agent configuration during enrollment adds:
```ini
# /etc/yum.repos.d/patchiq.repo
[patchiq]
name=PatchIQ Managed Patches
baseurl=https://{server_addr}/repo/yum
enabled=1
gpgcheck=1
gpgkey=https://{server_addr}/repo/yum/RPM-GPG-KEY-patchiq
```

> **Security note**: Same GPG key distribution as APT — the Server signs RPM packages and repo metadata. Key is distributed during enrollment.

**File Server** (`/repo/files/`):

For Windows and macOS — direct HTTPS download, no repo metadata needed:
```
/repo/files/
├── windows/
│   ├── KB5034441-x64.msu
│   └── dotnet-runtime-8.0.1-win-x64.msi
└── macos/
    └── macOS-Sequoia-15.3-Update.pkg
```

#### Dependency Metadata

The catalog stores dependency information per platform:

**Linux**: Package dependencies are handled natively by APT/YUM when installing from the PatchIQ repo. The repo metadata includes dependency information automatically.

**Windows**: Supersedence chains and prerequisite KBs stored in catalog:
```json
{
  "kb_id": "KB5034441",
  "prerequisites": ["KB5034123"],
  "supersedes": ["KB5033375"],
  "servicing_stack": "KB5032392",
  "install_order": 2
}
```

The deployment engine uses `install_order` to sequence commands correctly. If prerequisites are missing, it creates install commands for them first.

**macOS**: OS version prerequisites stored in catalog:
```json
{
  "min_os_version": "15.0",
  "requires_restart": true
}
```

Agent checks OS version before attempting install.

### License Gating

- Binary distribution: PROFESSIONAL+ tier
- COMMUNITY tier: catalog metadata only (patch names, CVEs, severity). Admin downloads binaries manually and distributes outside PatchIQ.

---

## 3. Agent Execution Commands

### Command Architecture

Six command types, all server-triggered via the existing pipeline:

```
Server commands table → gRPC SyncInbox → Agent SQLite inbox → CommandProcessor → Module Handler
```

### 3a. install_patch (Updated)

**Current**: Receives package name + version, tells OS package manager to install from public repos.

**Updated**: Receives package info + download reference. Agent installs from Server's managed repo.

Proto changes:
```protobuf
message InstallPatchPayload {
  repeated PatchTarget packages = 1;
  bool dry_run = 2;
  string pre_script = 3;
  string post_script = 4;
  // New fields
  string download_url = 5;           // relative URL to Server repo
  string checksum_sha256 = 6;        // verify binary integrity
  map<string, string> dependencies = 7;  // prerequisite packages
  int32 install_order = 8;           // for Windows supersedence ordering
}
```

Updated flow:
1. Receive `InstallPatchPayload` (now includes `download_url` + `checksum_sha256`)
2. IF Linux: verify Server's repo is configured in sources, `apt-get update` / `yum makecache`
3. IF Windows/macOS: download binary from Server's file server to temp dir, verify SHA256
4. Install via native package manager (existing logic)
5. Record rollback info (existing logic)
6. Report result with `reboot_required` flag (existing)

### 3b. rollback_patch (Server-Triggered)

**Current**: Agent-local only, uses JSON payload, looks up rollback records by original command ID.

**Updated**: Server creates `rollback_patch` commands. New protobuf message.

```protobuf
message RollbackPatchPayload {
  string deployment_id = 1;          // which deployment to roll back
  string original_command_id = 2;    // the install command to reverse
  repeated PatchTarget revert_to = 3;  // explicit target versions (optional)
  bool force_uninstall = 4;          // if no previous version, uninstall entirely
}
```

Three rollback triggers:
1. **Auto**: Wave failure threshold exceeded → deployment engine creates rollback commands for all affected agents
2. **Manual**: Admin clicks "Rollback" on deployment detail → REST API → creates commands
3. **Workflow**: Rollback node fires → creates commands via same mechanism

Rollback scopes:
- **Per-deployment**: Roll back everything in deployment X (all packages, all targets)
- **Per-target**: Roll back deployment X on specific endpoint only
- **Per-package**: Roll back one package on one endpoint

Agent-side: Patcher module handles `rollback_patch` using existing rollback logic, but deserializes from protobuf. Uses `revert_to` versions from server if provided (server has the catalog, knows correct previous version). Falls back to local rollback records if `revert_to` is empty.

### 3c. run_scan (Enhanced)

**Current**: Collects packages, hardware, OS info.

**Updated**: Full endpoint assessment.

```protobuf
message RunScanPayload {
  ScanType scan_type = 1;
  repeated string check_categories = 2;  // ["packages", "services", "security"]
}

enum ScanType {
  SCAN_TYPE_UNSPECIFIED = 0;
  SCAN_TYPE_FULL = 1;         // everything
  SCAN_TYPE_QUICK = 2;        // packages + OS info only
  SCAN_TYPE_TARGETED = 3;     // specific categories only
}
```

Full scan collects:
- Installed packages (APT/YUM/Homebrew/WUA/Snap) — existing
- Hardware info (CPU, RAM, disk, model) — existing
- OS details (version, kernel, arch) — existing
- Running services (systemd/launchd/Windows SCM) — existing
- System metrics (CPU/RAM/disk usage) — existing
- Software inventory (all installed applications) — enhanced
- Security posture (firewall, disk encryption, antivirus) — M3 built-in collectors

Post-scan server-side:
- `EndpointMatchJob` runs on every scan result (existing, confirmed for on-demand scans)
- Correlates installed packages against catalog + CVE database
- Updates endpoint's patch status, CVE exposure, compliance score

### 3d. reboot (New)

```protobuf
message RebootPayload {
  RebootMode mode = 1;
  int32 grace_period_seconds = 2;     // wait before force (graceful mode)
  string message = 3;                 // displayed to user if interactive
  bool post_reboot_scan = 4;          // auto-run inventory scan after reboot
}

enum RebootMode {
  REBOOT_MODE_UNSPECIFIED = 0;
  REBOOT_MODE_IMMEDIATE = 1;          // force reboot now
  REBOOT_MODE_GRACEFUL = 2;           // notify user, wait grace_period, then force
  REBOOT_MODE_DEFERRED = 3;           // schedule for next maintenance window
}
```

New `system` module on the agent:
- `SupportedCommands()` → `["reboot", "update_config"]`
- OS-specific reboot: `shutdown -r now` (Linux), `shutdown -r +0` (macOS), `shutdown /r /t 0` (Windows)
- Graceful mode: writes "reboot pending" flag, waits grace period, then forces
- Deferred mode: stores reboot request in SQLite, executes during maintenance window
- `post_reboot_scan`: agent writes a `reboot_pending` flag to SQLite before rebooting. On next daemon startup (Linux: systemd restarts the service, macOS: launchd restarts, Windows: Windows Service `OnStart`), the agent checks for this flag, runs a full inventory scan, reports "reboot completed" to outbox, and clears the flag. No OS-level startup hooks needed — the agent daemon already starts on boot via its service registration.

### 3e. run_script (New)

```protobuf
message RunScriptPayload {
  string script_id = 1;              // reference to script library (optional)
  string inline_script = 2;          // script body (if not from library)
  ScriptType script_type = 3;        // shell, powershell, python
  int32 timeout_seconds = 4;         // max execution time (default: 300)
  int32 max_output_bytes = 5;        // truncate output (default: 1MB)
  map<string, string> env = 6;       // environment variables
  bool capture_exit_code = 7;        // include exit code in result
}

enum ScriptType {
  SCRIPT_TYPE_UNSPECIFIED = 0;
  SCRIPT_TYPE_SHELL = 1;             // bash/zsh on Linux/macOS
  SCRIPT_TYPE_POWERSHELL = 2;        // Windows PowerShell
  SCRIPT_TYPE_PYTHON = 3;            // cross-platform
}
```

New `executor` module on the agent:
- `SupportedCommands()` → `["run_script"]`
- Writes script to temp file, executes via appropriate interpreter
- Captures stdout, stderr, exit code
- Enforces timeout and output size limits
- Cleans up temp file after execution
- Runs as agent's service user (root/SYSTEM)

### 3f. update_config (New)

```protobuf
message UpdateConfigPayload {
  map<string, string> settings = 1;  // key-value settings to apply
  bool restart_required = 2;         // hint: agent should restart after applying
}
```

Handled by the `system` module:
- Writes settings to SQLite settings table (existing `SettingsStore`)
- Settings watcher picks up changes on next poll cycle (existing)
- Covers: log level, scan interval, heartbeat interval, bandwidth limit, max concurrent installs, offline mode

### Module Registry Update

```go
// cmd/agent/main.go
registry.Register(inventory.New())                          // existing: run_scan
registry.Register(patcher.NewWithMaxConcurrentFunc(...))    // existing: install_patch, rollback_patch
registry.Register(system.New())                             // NEW: reboot, update_config
registry.Register(executor.New())                           // NEW: run_script
```

---

## 4. Tags System (replacing Groups)

### Design Decision

Replace Groups with a Tags system — key:value pairs with boolean expressions for universal endpoint classification.

### Data Model

```sql
-- Tag definitions
CREATE TABLE tags (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id   UUID NOT NULL REFERENCES tenants(id),
  key         TEXT NOT NULL,
  value       TEXT NOT NULL,
  color       TEXT,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, key, value)
);

-- Endpoint-to-tag assignments
CREATE TABLE endpoint_tags (
  endpoint_id UUID NOT NULL REFERENCES endpoints(id),
  tag_id      UUID NOT NULL REFERENCES tags(id),
  tenant_id   UUID NOT NULL REFERENCES tenants(id),
  source      TEXT NOT NULL,  -- 'manual', 'auto_rule', 'agent_reported'
  assigned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (endpoint_id, tag_id)
);

-- Auto-assignment rules
CREATE TABLE tag_rules (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id       UUID NOT NULL REFERENCES tenants(id),
  name            TEXT NOT NULL,
  condition       JSONB NOT NULL,
  tags_to_apply   UUID[] NOT NULL,
  enabled         BOOLEAN NOT NULL DEFAULT true,
  priority        INT NOT NULL DEFAULT 0,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

All tables are tenant-scoped with RLS policies.

### Tag Expressions

Structured JSON for targeting endpoints across all features:

```json
{
  "op": "AND",
  "conditions": [
    {"op": "OR", "conditions": [
      {"tag": "env", "value": "production"},
      {"tag": "env", "value": "staging"}
    ]},
    {"tag": "compliance", "value": "pci-dss"},
    {"op": "NOT", "conditions": [
      {"tag": "wave", "value": "canary"}
    ]}
  ]
}
```

Simple expressions:
- `env:production` → `{"tag": "env", "value": "production"}`
- `env:production AND os:linux` → `{"op": "AND", "conditions": [...]}`
- `(env:production OR env:staging) AND NOT wave:canary` → nested

Server evaluates tag expressions by querying `endpoint_tags` to resolve endpoint ID sets.

### Auto-Assignment Rules

Rules engine evaluates conditions against endpoint properties:

```json
{
  "field": "os_family",
  "op": "eq",
  "value": "linux"
}
```

Supported fields: `os_family`, `os_version`, `hostname` (regex), `ip_address` (CIDR), `hardware_model`, `agent_version`, any inventory-reported property.

Rules run on:
1. Endpoint enrollment (new endpoint)
2. After every inventory scan (state may have changed)
3. Manual trigger (admin re-evaluates all rules)

### Migration from Groups

- Existing groups become tags: group "Production Servers" → tag `group:production-servers`
- Existing policy `group_ids` → tag expressions: `group:production-servers`
- One-time migration script, backward compatible
- Groups page in UI becomes Tags management page

### Where Tags Are Used

| Feature | Current (Groups) | New (Tags) |
|---------|-----------------|------------|
| Policy targeting | `group_ids: [uuid]` | `target_expression: {tag expression}` |
| Deployment waves | Wave N = % of endpoints | Wave 1 = `wave:canary`, Wave 2 = `NOT wave:canary` |
| Compliance scope | Hardcoded frameworks | `compliance:pci-dss`, `compliance:hipaa` |
| Alert routing | N/A | `priority:critical` → PagerDuty |
| Dashboard filtering | N/A | Filter any view by tag expression |
| Test group (Hub) | Separate concept | `wave:test-group` on Hub's test endpoints |

---

## 5. Unified Deployment Wizard

### Design Decision

Kill the separate `DeploymentModal` (Patches page, `usePatchDeploy` API). All deployment creation goes through one unified wizard — a right-side slide-in panel (`Sheet` component, `side="right"`, ~480-520px width).

### Entry Points

| Entry Point | Opens At | Pre-filled |
|-------------|----------|-----------|
| Patches → "Deploy" on row | Step 1 | Patch pre-selected |
| Patches → bulk select → "Deploy Selected" | Step 1 | Multiple patches pre-selected |
| Policy detail → "Deploy Now" | Step 2 | Policy's patches + target expression |
| Deployments → "Create Deployment" | Step 1 | Blank |
| Workflow → "Execute" | Step 3 | Workflow's wave/strategy config |
| Dashboard → critical patch widget | Step 1 | Patch pre-selected |

### Step 1: Source

"What to deploy?"

- **From catalog**: Patch picker table with severity/CVE/OS filters, shows affected endpoint count per patch
- **From policy**: Policy dropdown with live preview — "This policy matches 14 patches across 38 endpoints"
- **Ad-hoc**: Manual package name + version entry (for custom/third-party software not in catalog)

### Step 2: Targets

"Where to deploy?"

- **Tag expression builder**: Visual builder with dropdowns for tag keys, autocomplete for values, AND/OR/NOT toggle buttons. Colored tag chips.
- **Live resolution**: Endpoint count animates as expression changes
- **Visuals**: Mini OS breakdown donut chart, endpoint list preview (first 5 + "and 37 more")
- **Options**: Exclude endpoints with pending deployments, respect maintenance windows

### Step 3: Strategy

"How to deploy?"

- **Waves**: Per-wave config with tag expression targeting, percentage, success threshold. Visual: horizontal wave timeline with sized endpoint circles.
- **Schedule**: Deploy now / future datetime / next maintenance window
- **Rollback**: Auto-rollback threshold slider with red zone indicator
- **Reboot**: Mode selection as icon cards (immediate/graceful/deferred) + grace period
- **Orchestration**: Optional workflow template dropdown with inline mini-DAG preview

### Step 4: Review & Confirm

- **Summary card**: "14 patches → 42 endpoints in 3 waves" with severity dots
- **Patches table**: Compact, severity badges, CVE count chips
- **Wave pipeline visualization**: Matches deployment detail page's `WavePipelineLanes` component
- **Workflow DAG preview**: If workflow template attached, full DAG with node labels
- **Timeline bar**: Estimated deployment duration based on wave delays
- **Two-click confirmation**: Warning banner + Deploy button with "Deploy & Watch" option

### Navigation

Step indicators at top as clickable pills. Any completed step is revisitable. Data persists across steps. Back-and-forth freely.

```
[ 1 Source ✓ ]  [ 2 Targets ✓ ]  [ 3 Strategy ● ]  [ 4 Review ]
```

### Contextual Intelligence (Patches Page)

Each patch row shows:
- "38 endpoints missing this patch"
- "Covered by Policy: Linux Critical Patches" (if applicable)
- "Deploy" opens wizard pre-filled; if policy covers it, secondary label: "Already in policy — deploy manually anyway?"

### API Changes

- Remove `usePatchDeploy` / its backend endpoint
- `POST /api/v1/deployments` becomes the single creation endpoint
- Extended request body: `source_type`, `target_expression`, `wave_config` (per-wave tag expressions), `rollback_config`, `reboot_config`, `workflow_template_id`

---

## 6. Policy → Auto-Deployment Wiring

### Policy Modes

| Mode | Behavior |
|------|----------|
| **Advisory** | Evaluate only. Shows matched patches in UI. No deployment. |
| **Manual** | Evaluate + "Deploy Now" opens wizard pre-filled at Step 2. |
| **Automatic** | Evaluate + auto-deploy using policy's deployment config. |

### Automatic Policy Flow

```
PolicySchedulerJob (River, per policy's cron schedule)
    │
    ▼
Evaluate policy (existing evaluator)
    │ matched patches × matched endpoints
    │
    ▼
Any new patches? (diff against last evaluation)
    │
    ├── No  → log, done
    │
    └── Yes → Create deployment:
              source: policy reference
              patches: evaluation result
              targets: policy's tag expression
              waves: policy's deployment_config
              rollback: policy's rollback_config
              reboot: policy's reboot_config
              workflow: policy's workflow_template_id
              │
              ▼
         Deployment enters normal pipeline
         (CREATED → RUNNING → waves → commands → agents)
              │
              ▼
         Emit: policy.auto_deployed
         Notify via policy's notification config
```

### Policy Deployment Config

New fields on the policy entity:

```json
{
  "deployment_config": {
    "wave_config": [
      {"tag_expression": {"tag": "wave", "value": "canary"}, "max_targets": 0, "success_threshold": 100},
      {"tag_expression": {"tag": "env", "value": "staging"}, "max_targets": 0, "success_threshold": 95},
      {"tag_expression": null, "max_targets": 0, "success_threshold": 90}
    ],
    "rollback_threshold": 20,
    "reboot_mode": "graceful",
    "reboot_grace_period": 900,
    "max_concurrent": 10,
    "workflow_template_id": null
  }
}
```

**Wave targeting semantics**: Each wave uses a `tag_expression` to select its endpoints. If `tag_expression` is null, the wave targets all remaining endpoints not matched by prior waves. `max_targets` (0 = unlimited) optionally caps the number of endpoints per wave. `success_threshold` is the minimum percentage of successful installs required before the next wave proceeds. This replaces the old percentage-based wave model entirely — wave size is determined by how many endpoints match the tag expression, not by a percentage of the total fleet.
```

### Policy Detail Page Updates

- **Deploy Now** (Manual): Opens wizard panel at Step 2, pre-filled with policy config
- **Deploy Now** (Automatic): Not shown. Shows "Next scheduled evaluation: [datetime]" instead
- **Evaluate Now** (all modes): Runs evaluation, shows matched patches × endpoints preview
- **Deployment History tab**: Each row links to `/deployments/:id`
- **New Deployment Config tab**: Configure waves, rollback, reboot defaults

---

## 7. Workflow Enhancements

### New Node Types

| Node | Purpose |
|------|---------|
| `reboot` | Send reboot command to targeted endpoints after a wave completes |
| `scan` | Trigger on-demand scan (post-deploy verification) |
| `tag_gate` | Evaluate tag expression — proceed only if condition met |
| `compliance_check` | Run compliance evaluation, gate on pass/fail. M2 stub: always returns pass and logs "full compliance evaluation requires M3". M3: wired to compliance engine. |

### Updated Existing Nodes

| Node | Change |
|------|--------|
| `deployment_wave` | Binds to tag expression for targeting. Config: patch source, tag expression, success threshold, max concurrent, reboot config |
| `script` | Payload defined — uses `RunScriptPayload`. Config: inline script or library ref, script type, timeout, expected exit code |
| `trigger` | New types: `on_policy_evaluation`, `on_cve_published`, `on_scan_complete`, `on_tag_assigned`, `on_compliance_drift` |
| `rollback` | Creates server-triggered `rollback_patch` commands. Config: target wave, force uninstall toggle |
| `notification` | Contextual body — deployment summary, wave results, compliance report |

### Workflow Templates

Reusable orchestration patterns:
- "Standard 3-wave rollout" (canary → staging → production with approval gates)
- "Emergency hotfix" (skip canary, deploy all, auto-reboot)
- "Compliance remediation" (scan → evaluate → deploy → re-scan → verify)

Templates selectable in:
- Deployment wizard Step 3
- Policy deployment config

### Trigger Events

Workflow triggers subscribe to Watermill domain events:

| Event | Trigger |
|-------|---------|
| `policy.evaluated` | "When this policy finds new patches" |
| `cve.published` | "When a critical CVE is published" |
| `scan.completed` | "When an endpoint scan finishes" |
| `tag.assigned` | "When an endpoint gets tagged" |
| `compliance.drift` | "When compliance score drops below threshold" |
| `deployment.failed` | "When a deployment fails" |
| `deployment.completed` | "When a deployment completes" |
| `schedule.cron` | "On a time schedule" (existing) |
| `manual` | "Admin clicks Execute" (existing) |

### Extensibility

**Event side**: Watermill uses topic strings. Adding a new trigger = subscribing to a new topic. No schema changes. M3 features (compliance v2, remote access) will emit their own domain events and workflows hook in automatically.

**Node handler side**: Executor uses `map[NodeType]NodeHandler` registry. Adding a new node type = define handler + register + add UI palette icon. Engine doesn't change.

---

## 8. RBAC & License Gating

### License Tiers

| Feature | COMMUNITY | PROFESSIONAL | ENTERPRISE |
|---------|------|----------|------------|
| Catalog metadata (patches, CVEs) | Yes | Yes | Yes |
| Binary distribution | No | Yes | Yes |
| Manual deployments | Yes | Yes | Yes |
| Wave configuration | No | Yes | Yes |
| Policy (Advisory mode) | Yes | Yes | Yes |
| Policy (Manual mode) | No | Yes | Yes |
| Policy (Automatic mode) | No | No | Yes |
| Tag auto-assignment rules | No | Yes | Yes |
| Workflow templates | No | Yes | Yes |
| Event-based workflow triggers | No | No | Yes |
| Custom script execution | No | No | Yes |

### RBAC Permissions

| Permission | Scope |
|-----------|-------|
| `tags:read` | View tags and tag assignments |
| `tags:manage` | Create, edit, delete tags and auto-assignment rules |
| `deployments:create` | Create deployments via wizard |
| `deployments:manage` | Cancel, retry, rollback deployments |
| `policies:deploy` | Trigger manual deployments from policies |
| `policies:manage` | Configure automatic deployment settings |
| `workflows:manage` | Create, edit, delete workflows |
| `workflows:execute` | Execute workflows manually |
| `catalog:manage` | Manage catalog sync, binary distribution |
| `scripts:manage` | Create, edit, delete scripts in script library |

---

## 9. Roadmap Items

The following items are acknowledged in this design but out of scope. Architecture decisions in this spec are designed to accommodate them without rework.

Detailed feature documents: `docs/blueprint/roadmap/`

| Item | Foundation Built Here |
|------|----------------------|
| Compliance Engine v2 | `compliance_check` workflow node, `run_script` command, tag-based scoping |
| Extended Agent Collectors | Module registry pattern, `run_scan` scan types |
| Development Compliance | `run_script` command, script library, tag expressions |
| RDP / Remote Access | Agent `system` module, RBAC permission model |
| Alert Pipelines via Workflows | Workflow event triggers, `notification` node, tag-based routing |
| Script-based Custom Collectors | `executor` module, `RunScriptPayload`, script library |

---

## Appendix: Files Changed

### Proto
- `proto/patchiq/v1/common.proto` — new messages: `RollbackPatchPayload`, `RebootPayload`, `RunScriptPayload`, `UpdateConfigPayload`, `RunScanPayload`. Updated: `InstallPatchPayload`. New enums: `RebootMode`, `ScriptType`, `ScanType`.

### Agent (new files)
- `internal/agent/system/system.go` — system module (reboot, update_config)
- `internal/agent/executor/executor.go` — executor module (run_script)

### Agent (modified)
- `internal/agent/patcher/patcher.go` — rollback_patch from protobuf, binary download step in install_patch
- `internal/agent/inventory/collector.go` — enhanced scan with scan types
- `cmd/agent/main.go` — register system + executor modules

### Server (new files)
- `internal/server/repo/` — APT/YUM repo hosting, file server
- `internal/server/store/queries/tags.sql` — tag queries
- `internal/server/store/queries/tag_rules.sql` — auto-assignment queries
- `internal/server/api/v1/tags.go` — tags REST handlers
- `internal/server/api/v1/tag_rules.go` — tag rules REST handlers
- `internal/server/store/migrations/023_tags.sql` — tags, endpoint_tags, tag_rules schema
- `internal/server/store/migrations/024_groups_to_tags_migration.sql` — data migration: groups → tags, policy group_ids → tag expressions

### Server (modified)
- `internal/server/deployment/wave_dispatcher.go` — rollback command creation, tag-based wave targeting
- `internal/server/deployment/statemachine.go` — rollback command dispatch
- `internal/server/policy/evaluator.go` — tag expression evaluation
- `internal/server/workflow/handlers/` — updated script, rollback, new reboot/scan/tag_gate/compliance_check handlers
- `internal/server/workers/catalog_sync.go` — binary download during sync
- `internal/server/grpc/sync_inbox.go` — new command type mappings

### Hub (modified)
- `internal/hub/catalog/` — binary fetcher, MinIO upload
- `internal/hub/store/queries/` — binary_ref, checksum fields
- `internal/hub/store/migrations/007_binary_distribution.sql` — binary_ref, checksum_sha256 columns on catalog

### Frontend (new)
- `web/src/components/DeploymentWizard/` — unified 4-step wizard panel
- `web/src/pages/tags/` — tags management page
- `web/src/components/TagExpressionBuilder/` — visual tag expression builder

### Frontend (modified)
- `web/src/pages/patches/` — remove DeploymentModal, add contextual deploy actions
- `web/src/pages/deployments/` — remove CreateDeploymentDialog, use wizard
- `web/src/pages/policies/` — wire dead buttons, add deployment config tab
- `web/src/pages/workflows/` — new node types in palette, trigger config forms
- `web/src/app/layout/AppSidebar.tsx` — add Tags, remove Groups
