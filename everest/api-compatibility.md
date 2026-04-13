# Zirozen API Compatibility Analysis

**Source**: `zirozen-api-v1.0.pdf` (Zirozen Patch Management REST API Document v1.2, dated 17-03-2024)
**Analysis date**: 2026-04-09

---

## Zirozen Endpoints (7 total)

| # | Endpoint | Method | Purpose |
|---|----------|--------|---------|
| 1 | `/api/oauth/token` | POST | Username/password auth -> JWT (access-token + refresh-token) |
| 2 | `/api/patch/agent/search/byUUID` | POST | Lookup endpoint by system UUID -> returns numeric `assetId` |
| 3 | `/api/patch/patch/search` | POST | Search/list all discovered patches with qualification filters |
| 4 | `/api/patch/asset-patch-relation/search` | POST | Get installed or missing patches for a specific endpoint |
| 5 | `/api/patch/asset/execute-scan-patch/{assetID}` | POST | Trigger patch scan on an endpoint |
| 6 | `/api/patch/deployment` | POST | Create a patch deployment for installation |
| 7 | `/api/patch/deployment/search` | POST | Search/list deployment status |

---

## Zirozen API Patterns

### Authentication
- `POST /api/oauth/token` with `{ "username": "...", "password": "..." }`
- Returns `{ "access-token": "JWT...", "refresh-token": "JWT..." }`
- All subsequent requests pass token in `Authorization` header
- RS512 signed JWTs with username, id, iat, exp, iss("Zirozen"), sub("API v1")

### Request Format
- **All endpoints use POST**, even for read operations
- Filter/search uses a `qualification[]` array in the request body:
  ```json
  {
    "offset": 0,
    "size": 20,
    "qualification": [
      {
        "operator": "equals",
        "column": "osPlatform",
        "value": "windows",
        "condition": "and"
      }
    ]
  }
  ```
- Supported operators: `equals` (others likely: `contains`, `greaterThan`, `lessThan`)
- Supported columns: `osPlatform`, `assetId`, `patchState`, `refModel`
- `condition` field: `and` / `or` for combining multiple qualifications

### Response Format
```json
{
  "result": [ ... ],
  "totalCount": 21
}
```
Single results: `{ "result": { ... } }` or `{ "result": "success" }`
Errors: `{ "result": "Invalid username/password" }`

### IDs
- **Numeric** sequential IDs everywhere (`assetId: 67`, `patchId: 19`, `id: 18`)
- Patches have a human-readable `name` field like `"ZPH-W-0022"` (Zirozen Patch - Windows - #22)

### Timestamps
- Unix milliseconds (e.g., `1741194338134`)
- Zero means "not set" (e.g., `"approvedOn": 0`)

---

## Field-by-Field Mapping

### Endpoint/Asset Fields

| Zirozen Field | PatchIQ Equivalent | Notes |
|--------------|-------------------|-------|
| `assetId` (numeric) | `id` (UUID) | Need numeric ID mapping |
| `system_uuid` | Agent enrollment UUID | Available from `agent_state` / enrollment |
| `platform` | `os_family` | Values: "windows" / "mac" / "ubuntu" vs "windows" / "darwin" / "debian" |
| `platform_version` | `os_version` | Direct map |
| `arch` | `arch` | Direct map |
| `agent_version` | `agent_version` | Direct map |
| `code_name` | `os_version` (partial) | PatchIQ doesn't store marketing name separately |
| `physical_memory` | `memory_total_mb` | Zirozen: bytes, PatchIQ: MB |
| `total_disk_space` | `disk_total_gb` | Zirozen: bytes, PatchIQ: GB |
| `used_disk_space` | `disk_used_gb` | Unit conversion needed |
| `used_memory` | `memory_used_mb` | Unit conversion needed |
| `uptime` | Not directly exposed in list API | Available from heartbeat data |
| `risk_score` | Not implemented | Would need to compute from CVE/compliance data |
| `build` | `kernel_version` (partial) | Windows build number vs kernel |
| `bssid` | Not collected | WiFi BSSID — not relevant for patch management |
| `archived` | `status: "decommissioned"` | Different representation |

### Patch Fields

| Zirozen Field | PatchIQ Equivalent | Notes |
|--------------|-------------------|-------|
| `id` (numeric) | `id` (UUID) | Need numeric ID mapping |
| `name` ("ZPH-W-0022") | `name` | PatchIQ uses package name, not sequential ID |
| `title` | `description` (partial) | PatchIQ stores description, not separate title |
| `osPlatform` | `os_family` | Value mapping: "windows"/"mac"/"ubuntu" vs "windows"/"darwin"/"debian" |
| `osArch` | Not stored per-patch | PatchIQ tracks arch per endpoint, not per patch |
| `patchSeverity` | `severity` | Same values: "critical"/"high"/"medium"/"low" |
| `patchApprovalStatus` | `status` | Zirozen: "approved"/"not_approved", PatchIQ: "available"/"superseded"/"recalled" |
| `patchTestStatus` | Not implemented | Zirozen has "not_tested"/"tested"/"failed" |
| `downloadStatus` | Not exposed | PatchIQ handles downloads differently (hub catalog) |
| `downloadSize` | Not stored per-patch | Available from hub catalog entry |
| `cveNumber` | Related CVEs (separate records) | Zirozen: comma-separated string. PatchIQ: separate `cve_patches` join table |
| `kbId` | Not stored natively | Windows KB article ID — could extract from patch metadata |
| `patchUpdateCategory` | Not stored | "security updates" / "critical updates" etc. |
| `supportUrl` | Not stored | Microsoft support URL |
| `rebootBehaviour` | Not stored | "may_be" / "required" / "not_required" |
| `isUninstallable` | Not stored | Boolean |
| `hasSupersededUpdates` | Not stored directly | Could derive from patch relationships |
| `isSuperseded` | `status: "superseded"` | Different representation |
| `downloadFileDetails` | Hub catalog binary info | Available from hub's `patch_catalog` + MinIO |
| `uuid` | Not stored per-patch | Zirozen assigns UUID to each patch independently |
| `source` | Not stored | "scanning" / "manual" |
| `releaseDate` | `released_at` | Zirozen: unix millis, PatchIQ: RFC3339 |
| `missingEndpoints` | `affected_endpoint_count` | Direct semantic match |
| `installedEndpoints` | `endpoints_deployed_count` | Direct semantic match |
| `ignoredEndpoints` | Not tracked | PatchIQ doesn't track ignored status |
| `isThirdParty` | Not tracked | PatchIQ doesn't distinguish 1st vs 3rd party |
| `installCommand` | Not exposed via API | Agent handles internally |
| `affectedProducts` | Not stored as IDs | PatchIQ uses os_family + os_distribution |
| `tags` | Tags system exists | Different structure |
| `bulletinId` | Not stored | Microsoft bulletin ID |

### Deployment Fields

| Zirozen Field | PatchIQ Equivalent | Notes |
|--------------|-------------------|-------|
| `id` (numeric) | `id` (UUID) | Need numeric ID mapping |
| `name` ("ADR-018") | `name` | PatchIQ uses descriptive names |
| `refModel` | Always "Patch" | PatchIQ only deploys patches |
| `refIds` (patch IDs) | Derived from policy | Zirozen specifies patches directly; PatchIQ uses policy-based selection |
| `assets` (endpoint IDs) | Derived from policy targets | Same pattern as refIds |
| `deploymentType` | Always "install" currently | Zirozen: "install"/"uninstall" |
| `deploymentStage` | `status` | Zirozen: "initiated"/"in_progress"/"completed", PatchIQ: "created"/"running"/"completed"/"failed"/etc. |
| `scope` | Not directly mapped | Zirozen: 2=specific assets, 4=all assets |
| `deploymentPolicyId` | `policy_id` | Direct map (but IDs differ) |
| `notifyEmailIds` | Notification channels | PatchIQ uses channel-based notifications, not direct email |
| `retryCount` | `max_retries` (in policy) | Handled at policy level in PatchIQ |
| `totalTaskCount` | `target_count` | Direct map |
| `completedTaskCount` | `completed_count` | Direct map |
| `successTaskCount` | `success_count` | Direct map |
| `failedTaskCount` | `failed_count` | Direct map |
| `pendingTaskCount` | Derived: `target_count - completed_count` | Can compute |
| `isPkgSelectAsBundle` | Not applicable | PatchIQ doesn't bundle patches |
| `isSelfServiceDeployment` | Not implemented | |
| `origin` | Not stored | "manual" / "automated" |
| `isRecurringDeployment` | Deployment schedules | PatchIQ uses separate schedule entities |
| `computerGroupIds` | Policy targets (tag-based) | Different targeting model |

### Asset-Patch Relation Fields

| Zirozen Field | PatchIQ Equivalent | Notes |
|--------------|-------------------|-------|
| `patchId` | Patch UUID | Numeric ID mapping |
| `assetId` | Endpoint UUID | Numeric ID mapping |
| `patchState` | Derived from inventory | "installed" / "missing" / "ignored" |
| `isOld` | Not tracked | |
| `isDeclined` | Not tracked | |
| `exceptionType` | Not tracked | Zirozen has patch exceptions |
| `exceptionReason` | Not tracked | |
| Nested `endpoint` object | Separate API call in PatchIQ | Need to join data |
| Nested `patch` object | Separate API call in PatchIQ | Need to join data |

---

## Platform Value Mapping

| Concept | Zirozen Values | PatchIQ Values |
|---------|---------------|----------------|
| OS Platform | `windows`, `mac`, `ubuntu` | `windows`, `darwin`, `debian`, `rhel`, `centos`, etc. |
| Severity | `critical`, `high`, `medium`, `low` | `critical`, `high`, `medium`, `low`, `none` |
| Patch Status | `approved`, `not_approved`, `draft` | `available`, `superseded`, `recalled` |
| Deployment Stage | `initiated`, `in_progress`, `completed`, `failed` | `created`, `scheduled`, `running`, `completed`, `failed`, `cancelled`, `rolling_back`, `rolled_back`, `rollback_failed` |

---

## Data Gaps (Fields Zirozen has that PatchIQ doesn't)

These fields exist in Zirozen responses but have no PatchIQ equivalent:

| Field | Impact | Recommendation |
|-------|--------|---------------|
| `kbId` (KB article number) | Medium — Windows patch identification | Store in patch metadata or derive from patch name |
| `patchTestStatus` | Low — testing workflow | Return "not_tested" as default |
| `risk_score` per endpoint | Medium — used by client dashboards | Compute from CVE count * severity weights |
| `downloadFileDetails[]` | Low — client likely doesn't use this | Omit or return empty array |
| `patchApprovalStatus` | Medium — approval workflow | Map PatchIQ status: available="approved", recalled="not_approved" |
| `rebootBehaviour` | Low | Return "may_be" as default |
| `isThirdParty` | Low | Return `false` as default |
| `ignoredEndpoints` count | Low | Return `0` |
| `exceptionType`/`exceptionReason` | Low | Return empty |
| `bulletinId` | Low | Return empty |

---

## Feasibility Summary

| Endpoint | Can We Build It? | Effort | Complexity |
|----------|-----------------|--------|------------|
| Auth token | Yes | 3h | Medium (Zitadel password grant or separate auth) |
| Search by UUID | Yes | 2h | Low |
| Patch search | Yes | 4h | Medium (qualification parser) |
| Asset-patch relation | Yes | 4h | Medium (join inventory + patches + endpoints) |
| Execute scan | Yes (after Phase 0 fix) | 1h | Low |
| Create deployment | Yes | 4h | Medium (policy model bridging) |
| Deployment search | Yes | 3h | Low |
| ID mapping infrastructure | Yes | 2h | Low |
| Response transformation | Yes | 3h | Low |
| **Total** | **Yes** | **~26h** | |
