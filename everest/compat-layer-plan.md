# Zirozen Compatibility Layer — Implementation Plan

**Goal**: Expose a `/api/compat/zirozen/` route group that speaks Zirozen's API dialect while using PatchIQ's internal services.
**Effort**: ~26 hours (core dev work, not intern-delegable)
**Depends on**: Phase 0 scan endpoint fix (Agent 2.1)

---

## Architecture

```
Client (Everest's existing automation)
  │
  │  POST /api/compat/zirozen/patch/patch/search
  │  Authorization: Bearer <compat-jwt>
  │
  ▼
┌─────────────────────────────────────────┐
│  Compat Router (/api/compat/zirozen/)   │
│                                         │
│  ┌─────────────┐  ┌──────────────────┐  │
│  │ Compat Auth  │  │ ID Mapper        │  │
│  │ Middleware   │  │ (numeric ↔ UUID) │  │
│  └─────────────┘  └──────────────────┘  │
│                                         │
│  ┌─────────────────────────────────────┐│
│  │ Compat Handlers                     ││
│  │  - TokenHandler                     ││
│  │  - AssetSearchHandler               ││
│  │  - PatchSearchHandler               ││
│  │  - AssetPatchRelationHandler        ││
│  │  - ScanHandler                      ││
│  │  - DeploymentHandler                ││
│  │  - DeploymentSearchHandler          ││
│  └─────────────────────────────────────┘│
│           │                              │
│           │ calls                         │
│           ▼                              │
│  ┌─────────────────────────────────────┐│
│  │ PatchIQ Store (same querier         ││
│  │ interfaces used by native handlers) ││
│  └─────────────────────────────────────┘│
│           │                              │
│           │ transforms                   │
│           ▼                              │
│  ┌─────────────────────────────────────┐│
│  │ Response Transformer                ││
│  │  - UUID → numeric ID               ││
│  │  - RFC3339 → unix millis            ││
│  │  - PatchIQ envelope → Zirozen       ││
│  │  - Field renaming                   ││
│  └─────────────────────────────────────┘│
└─────────────────────────────────────────┘
```

**Key principle**: Compat handlers use the **same store interfaces** as native handlers. No duplicate business logic. The compat layer is purely a request/response translator.

---

## Package Structure

```
internal/server/compat/
├── zirozen/
│   ├── router.go           # Chi subrouter mounted at /api/compat/zirozen
│   ├── middleware.go        # Compat auth middleware (validates compat JWTs)
│   ├── token.go             # POST /api/oauth/token handler
│   ├── asset_search.go      # POST /api/patch/agent/search/byUUID handler
│   ├── patch_search.go      # POST /api/patch/patch/search handler
│   ├── asset_patch.go       # POST /api/patch/asset-patch-relation/search handler
│   ├── scan.go              # POST /api/patch/asset/execute-scan-patch/{assetID} handler
│   ├── deployment.go        # POST /api/patch/deployment handler
│   ├── deployment_search.go # POST /api/patch/deployment/search handler
│   ├── id_mapper.go         # Numeric ID ↔ UUID mapping
│   ├── qualification.go     # Zirozen qualification[] filter parser
│   ├── transform.go         # Response transformation helpers
│   └── types.go             # Zirozen request/response type definitions
```

---

## Database: ID Mapping Table

Migration: `internal/server/store/migrations/054_compat_id_map.sql`

```sql
-- +goose Up
CREATE TABLE compat_id_map (
    numeric_id BIGSERIAL PRIMARY KEY,
    resource_type TEXT NOT NULL,
    uuid UUID NOT NULL,
    tenant_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(resource_type, uuid, tenant_id)
);

CREATE INDEX idx_compat_id_map_lookup
    ON compat_id_map(tenant_id, resource_type, uuid);

CREATE INDEX idx_compat_id_map_reverse
    ON compat_id_map(tenant_id, resource_type, numeric_id);

COMMENT ON TABLE compat_id_map IS
    'Maps PatchIQ UUIDs to stable numeric IDs for Zirozen API compatibility.';

-- +goose Down
DROP TABLE IF EXISTS compat_id_map;
```

sqlc queries: `internal/server/store/queries/compat_id_map.sql`

```sql
-- name: GetOrCreateCompatID :one
INSERT INTO compat_id_map (resource_type, uuid, tenant_id)
VALUES ($1, $2, $3)
ON CONFLICT (resource_type, uuid, tenant_id) DO UPDATE SET resource_type = EXCLUDED.resource_type
RETURNING numeric_id;

-- name: GetUUIDByCompatID :one
SELECT uuid FROM compat_id_map
WHERE tenant_id = $1 AND resource_type = $2 AND numeric_id = $3;

-- name: BatchGetOrCreateCompatIDs :many
INSERT INTO compat_id_map (resource_type, uuid, tenant_id)
SELECT $1, unnest($2::uuid[]), $3
ON CONFLICT (resource_type, uuid, tenant_id) DO UPDATE SET resource_type = EXCLUDED.resource_type
RETURNING numeric_id, uuid;
```

---

## Endpoint Implementations

### 1. Token Handler (`token.go`)

```
POST /api/compat/zirozen/api/oauth/token

Request:  { "username": "...", "password": "..." }
Response: { "access-token": "JWT...", "refresh-token": "JWT..." }
Error:    { "result": "Invalid username/password" }
```

**Implementation**:
- Accept username/password
- Authenticate against Zitadel using Resource Owner Password Grant (if enabled)
  OR against PatchIQ's direct login system
- Issue a compat JWT (HS256, short-lived) containing tenant_id and user_id
- Issue a refresh token (longer-lived)
- The compat auth middleware validates these tokens on subsequent requests

**Decision needed**: Do we create a separate "API user" concept for compat access, or reuse existing Zitadel users? Recommendation: Create API keys/service accounts in Zitadel, use password grant.

### 2. Asset Search by UUID (`asset_search.go`)

```
POST /api/compat/zirozen/api/patch/agent/search/byUUID

Request:  { "uuid": "6f720d3f-..." }
Response: { "result": { "assetId": 67 } }
```

**Implementation**:
- Query `endpoints` table by `system_uuid` (from agent enrollment metadata)
- Get-or-create a compat numeric ID for the endpoint UUID
- Return `{ "result": { "assetId": <numeric_id> } }`

**Note**: Need to verify where `system_uuid` is stored. Check enrollment flow — the agent sends hardware UUID during enrollment, stored in endpoint metadata or a dedicated column.

### 3. Patch Search (`patch_search.go`)

```
POST /api/compat/zirozen/api/patch/patch/search

Request:  { "offset": 0, "size": 20, "qualification": [...] }
Response: { "result": [...], "totalCount": N }
```

**Implementation**:
- Parse `qualification[]` filters → extract osPlatform, severity, etc.
- Map filter values: "mac" → "darwin", "ubuntu" → "debian"
- Call PatchIQ's `ListPatches` store query with mapped filters
- Transform each patch to Zirozen format:
  - UUID → numeric ID (via mapper)
  - Generate Zirozen-style name: `ZPH-{platform_letter}-{numeric_id:04d}`
  - `severity` → `patchSeverity`
  - `os_family` → `osPlatform`
  - Aggregate related CVEs into comma-separated `cveNumber` string
  - `released_at` → `releaseDate` (unix millis)
  - `affected_endpoint_count` → `missingEndpoints`
  - `endpoints_deployed_count` → `installedEndpoints`
  - Default values for fields we don't have (downloadStatus, kbId, etc.)

### 4. Asset-Patch Relation Search (`asset_patch.go`)

```
POST /api/compat/zirozen/api/patch/asset-patch-relation/search

Request:  { "offset": 0, "size": 20, "qualification": [
            { "column": "assetId", "value": "67", ... },
            { "column": "patchState", "value": "installed", ... }
          ]}
Response: { "result": [{ relation with nested endpoint + patch }], "totalCount": N }
```

**Implementation**:
- Parse qualification to extract `assetId` (numeric → UUID via mapper) and `patchState`
- For "installed": query `endpoint_packages` joined with patches
- For "missing": query `ListAvailablePatchesForEndpointByOS`
- Build nested response with both endpoint info and patch info per relation
- Transform all fields to Zirozen format

**This is the most complex handler** — it requires joining endpoint, inventory, and patch data into Zirozen's nested response format. May need a dedicated store query.

### 5. Scan Trigger (`scan.go`)

```
POST /api/compat/zirozen/api/patch/asset/execute-scan-patch/{assetID}

Request:  (empty body, assetID in URL)
Response: { "result": "success" }
```

**Implementation**:
- Map numeric `assetID` → UUID via mapper
- Call the same scan trigger logic as native `POST /endpoints/{id}/scan`
- Return `{ "result": "success" }`

**Depends on**: Phase 0 fix for agent scan endpoint (currently a no-op).

### 6. Create Deployment (`deployment.go`)

```
POST /api/compat/zirozen/api/patch/deployment

Request:  {
            "refIds": [1981],        // patch IDs (numeric)
            "assets": [67],          // endpoint IDs (numeric)
            "deploymentType": "install",
            "displayName": "Security Patches",
            "deploymentPolicyId": 1,
            ...
          }
Response: { "result": <deployment_numeric_id> }
```

**Implementation**:
- Map numeric patch IDs → UUIDs
- Map numeric endpoint IDs → UUIDs
- Map numeric `deploymentPolicyId` → UUID (or use a default policy)
- Create an ad-hoc deployment using PatchIQ's deployment creation logic
  - Option A: Use `QuickDeploy` for each patch (simpler but multiple deployments)
  - Option B: Create a temporary policy targeting the specified endpoints + patches, then create a single deployment (better match to Zirozen's model)
- Return the deployment's numeric ID

**Decision needed**: How to bridge Zirozen's "deploy these specific patches to these specific endpoints" model with PatchIQ's policy-based deployment model. Recommendation: Option B — create a transient policy.

### 7. Deployment Search (`deployment_search.go`)

```
POST /api/compat/zirozen/api/patch/deployment/search

Request:  { "offset": 0, "size": 20, "qualification": [...] }
Response: { "result": [...], "totalCount": N }
```

**Implementation**:
- Parse qualification filters (refModel, deploymentStage, etc.)
- Call PatchIQ's `ListDeployments` store query
- Transform each deployment to Zirozen format:
  - UUID → numeric ID
  - Generate Zirozen-style name: `ADR-{numeric_id:03d}`
  - Map status: "running" → "in_progress", "created" → "initiated", etc.
  - Include task counts (total, completed, success, failed, pending)
  - Map refIds back to numeric patch IDs
  - Map assets back to numeric endpoint IDs

---

## Qualification Filter Parser

The `qualification[]` system is Zirozen's generic filter mechanism. We need to parse it and map to PatchIQ's query parameters.

```go
// qualification.go

type Qualification struct {
    Operator  string `json:"operator"`  // "equals", "contains", "greaterThan", "lessThan"
    Column    string `json:"column"`    // field name in Zirozen's schema
    Value     string `json:"value"`     // filter value
    Condition string `json:"condition"` // "and", "or"
    Type      string `json:"type"`      // optional: "enum"
    Reference string `json:"reference"` // optional: enum name
}

type SearchRequest struct {
    Offset        int             `json:"offset"`
    Size          int             `json:"size"`
    Qualification []Qualification `json:"qualification"`
}
```

Column mapping table:

| Zirozen Column | PatchIQ Query Param | Value Mapping |
|---------------|-------------------|---------------|
| `osPlatform` | `os_family` | "mac" → "darwin", "ubuntu" → "debian" |
| `patchState` | Custom logic | "installed" / "missing" |
| `assetId` | `endpoint_id` | Numeric → UUID |
| `patchSeverity` | `severity` | Direct map |
| `refModel` | (always "Patch") | Ignored |
| `deploymentStage` | `status` | Value mapping |

---

## Response Transformer

```go
// transform.go

// TimeToUnixMillis converts time.Time to Zirozen's unix millisecond format.
// Zero time returns 0.
func TimeToUnixMillis(t time.Time) int64

// UnixMillisToTime converts Zirozen's unix milliseconds to time.Time.
func UnixMillisToTime(ms int64) time.Time

// WrapResult wraps data in Zirozen's { "result": data } envelope.
func WrapResult(w http.ResponseWriter, data any)

// WrapResultList wraps a list + count in Zirozen's { "result": [...], "totalCount": N } envelope.
func WrapResultList(w http.ResponseWriter, data any, totalCount int64)

// MapOSFamily converts PatchIQ os_family to Zirozen osPlatform.
func MapOSFamily(patchiqOS string) string

// MapSeverity converts PatchIQ severity to Zirozen patchSeverity.
func MapSeverity(severity string) string

// MapDeploymentStatus converts PatchIQ deployment status to Zirozen deploymentStage.
func MapDeploymentStatus(status string) string
```

---

## Middleware: Compat Auth

The compat auth middleware sits on the `/api/compat/zirozen/` route group and:

1. Skips the `/api/oauth/token` endpoint (it's the login endpoint)
2. Extracts the JWT from the `Authorization` header
3. Validates the JWT signature (using compat signing key)
4. Extracts tenant_id and user_id from claims
5. Injects them into the request context (same as native auth middleware)

This means compat handlers get the same tenant/user context as native handlers, and all RLS/RBAC works identically.

---

## Testing Strategy

Each compat handler needs:
1. **Unit test**: Mock the store, verify request parsing and response transformation
2. **Integration test**: Real database, verify the full flow from Zirozen-format request to Zirozen-format response

Critical test: **Round-trip test** — call the compat API with exact requests from the Zirozen PDF and verify responses match the expected shape.

---

## Implementation Order

1. `types.go` + `transform.go` + `qualification.go` — Foundation types and helpers
2. `id_mapper.go` + migration — ID mapping infrastructure
3. `token.go` + `middleware.go` — Auth (needed for everything else)
4. `asset_search.go` — Simplest handler, validates the pattern
5. `patch_search.go` — Most-used endpoint
6. `scan.go` — Simple, depends on Phase 0 fix
7. `deployment_search.go` — Read-only, moderate complexity
8. `deployment.go` — Most complex (policy bridging)
9. `asset_patch.go` — Most complex query (relation joins)
10. `router.go` — Wire everything together, mount on server

---

## Open Questions

1. **Auth model**: Password grant against Zitadel, or separate API key system for compat?
2. **Deployment bridging**: Transient policy per deployment, or extend QuickDeploy to accept multiple patches + endpoints?
3. **system_uuid storage**: Where exactly does PatchIQ store the agent's hardware UUID? Need to verify the enrollment flow stores this in a queryable column.
4. **KB article IDs**: Should we start storing these from hub catalog data? The WSUS/WUA feed likely has KB numbers.
5. **Risk score**: Should we compute one? Formula: weighted CVE count * severity. Or return 0 and tell the client it's not yet available.
