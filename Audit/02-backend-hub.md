# Hub Manager Backend Audit

**Scope**: `internal/hub/` + `cmd/hub/main.go`
**Date**: 2026-04-09
**Auditor**: Claude Opus 4.6

---

## Critical

### H-C1: JWT `mintJWT` vulnerable to JSON injection

**File**: `internal/hub/auth/session.go:39`
**Lines**: 39-40

The `mintJWT` function builds the JWT claims payload via `fmt.Sprintf` with raw string interpolation:

```go
claims := fmt.Sprintf(`{"sub":"%s","tenant_id":"%s","email":"%s","name":"%s","iat":%d,"exp":%d,"iss":"patchiq-hub"}`,
    sub, tenantID, email, name, now.Unix(), now.Add(ttl).Unix())
```

If `email` or `name` contain a double quote (`"`), the JSON is malformed or an attacker could inject arbitrary claims. For example, a Zitadel display name containing `"` would break JWT parsing or allow claim injection. Must use `json.Marshal` to build the claims map properly.

### H-C2: Route path mismatch for RegistrationStatus endpoint

**File**: `internal/hub/api/router.go:93` vs `internal/hub/api/v1/clients.go:135`

The router registers:
```go
r.Get("/api/v1/clients/register/status", clients.RegistrationStatus)
```

But the handler comment says `GET /api/v1/clients/registration-status` and all tests use `/api/v1/clients/registration-status`:
- `internal/hub/api/v1/clients_test.go:212`
- `internal/hub/api/v1/clients_test.go:237`
- `internal/hub/api/v1/clients_test.go:260`
- `internal/hub/api/v1/clients_test.go:273`

This means: (a) tests are not actually testing the real route (they call the handler directly, not through the router), and (b) any client calling the documented path gets 404.

### H-C3: UUID formatting inconsistency between workers and API handlers

**File**: `internal/hub/workers/binary_fetch.go:167-171` vs `internal/hub/api/v1/catalog.go:607-610` vs `internal/hub/catalog/pipeline.go:286-293`

Three separate `uuidToString`/`uuidToStr` implementations exist:

1. **API handlers** (`catalog.go:607`): `%08x-%04x-%04x-%04x-%012x` (zero-padded, correct)
2. **Pipeline** (`pipeline.go:286`): Same as API handlers (correct)
3. **Binary fetch worker** (`binary_fetch.go:167`): `%x-%x-%x-%x-%x` (NO zero-padding)

The worker's `uuidToStr` produces UUIDs like `a1b2c3d-e4f5-6789-abcd-ef1234567890` instead of `0a1b2c3d-e4f5-6789-abcd-0ef1234567890`. This causes:
- Audit events with malformed resource IDs
- Inability to join audit events back to catalog entries
- Inconsistent event payloads between pipeline and binary fetch events

---

## Important

### H-I1: License key generation is a JSON placeholder, not cryptographically signed

**File**: `internal/hub/api/v1/licenses.go:131-146`

The `Create` handler generates license keys as plain JSON:
```go
// Placeholder license key (M2 will use RSA-signed keys).
licenseKeyData, err := json.Marshal(map[string]any{
    "tier": req.Tier, "max_endpoints": req.MaxEndpoints, ...
})
```

Meanwhile, `internal/hub/license/generate.go` has a fully functional RSA-signed license generator (`Generator.Generate()`). The API handler does NOT use the generator. The license key stored in the DB is trivially forgeable by any client.

### H-I2: License generator (`internal/hub/license/generate.go`) is dead code in production

**File**: `internal/hub/license/generate.go`

`NewGenerator` is only called from `internal/hub/license/generate_test.go`. Functions `SaveToFile` and `DecodeSignature` are also only referenced in tests. The entire `license` package is built but never wired into `cmd/hub/main.go` or any API handler.

### H-I3: All tenant CRUD queries are unused

**File**: `internal/hub/store/sqlcgen/tenants.sql.go`

The following generated queries are never called anywhere in the hub codebase (only exist in the generated file):
- `CreateTenant`
- `GetTenantByID`
- `GetTenantBySlug`
- `ListTenants`
- `UpdateTenant`

No tenant management API handler exists. The hub hardcodes `defaultTenantID = "00000000-0000-0000-0000-000000000001"` everywhere.

### H-I4: All agent binary queries are unused

**File**: `internal/hub/store/sqlcgen/agent_binaries.sql.go`

The following generated queries are never called:
- `CreateAgentBinary`
- `GetAgentBinaryByID`
- `GetLatestBinary`
- `ListAgentBinaries`

No agent binary management API handler exists in the hub.

### H-I5: Defined event types never emitted

**File**: `internal/hub/events/topics.go`

The following event types are defined in `AllTopics()` and registered for wildcard subscribers, but never emitted anywhere:
- `SyncStarted` (`sync.started`) - line 43
- `SyncFailed` (`sync.failed`) - line 45
- `CatalogSynced` (`catalog.synced`) - line 14
- `TenantCreated` (`tenant.created`) - line 8
- `TenantUpdated` (`tenant.updated`) - line 9

This wastes Watermill subscriber resources (each topic gets a consumer group) and indicates features that were planned but not implemented.

### H-I6: `PackageAliasHandler.Update` ignores the route `{id}` parameter

**File**: `internal/hub/api/v1/package_aliases.go:180-228`

The `Update` handler parses `{id}` from the URL (line 181) but then calls `UpsertPackageAlias` which is an upsert by natural key (feed_product + os_family + os_distribution + os_package_name), completely ignoring the parsed `id`. The `id` variable is only used in an error log (line 212). This means:
- PUT `/package-aliases/{id}` does NOT update the record with that ID
- It creates/updates by the composite natural key from the request body
- The route semantics are misleading

### H-I7: Catalog fetchers (APT, YUM, MSU, Apple) are not wired to the pipeline

**File**: `internal/hub/catalog/fetcher_apt.go`, `fetcher_yum.go`, `fetcher_msu.go`, `fetcher_apple.go`

These four OS-specific fetchers (`NewAPTFetcher`, `NewYUMFetcher`, `NewMSUFetcher`, `NewAppleFetcher`) are never instantiated in `cmd/hub/main.go`. Only the generic `BinaryFetcher` is wired:
```go
fetcher := catalog.NewBinaryFetcher(ms, minIOCfg.Bucket, nil)  // line 252
```

The specialized fetchers have OS-specific key path logic (e.g., `apt/{osVersion}/{filename}`) that the generic fetcher does not replicate. They are dead code.

### H-I8: Settings `UpdatedBy` uses tenant UUID instead of user ID

**File**: `internal/hub/api/v1/settings.go:144`

```go
UpdatedBy: tid, // TODO(PIQ-245): use actual user ID from auth context
```

The `updated_by` field is set to the tenant UUID, which is meaningless for auditing who changed a setting. The user ID is available from the auth context but is not used.

---

## Minor

### H-M1: `SessionConfig.PostLoginURL` is set but never read

**File**: `internal/hub/auth/session.go:22`

`PostLoginURL` is defined in `SessionConfig` and set in `cmd/hub/main.go:278` but never used anywhere in the login flow. Dead field.

### H-M2: Duplicate `uuidToString` functions across packages

**Files**:
- `internal/hub/api/v1/catalog.go:606-610`
- `internal/hub/catalog/pipeline.go:285-293`
- `internal/hub/workers/binary_fetch.go:167-172`

Three separate implementations of UUID-to-string conversion. Should be a single shared function. The pipeline version also checks `id.Valid` while the API version does not (API version would produce `00000000-0000-0000-0000-000000000000` for invalid UUIDs instead of empty string).

### H-M3: Ubuntu feed hardcodes severity to "medium" for all entries

**File**: `internal/hub/feeds/ubuntu.go:179`

```go
Severity: "medium",
```

Every Ubuntu USN entry gets severity "medium" regardless of actual severity. The Ubuntu API does not provide severity directly, but CVE cross-referencing or parsing USN titles could improve this.

### H-M4: Apple feed hardcodes severity to "high" for all entries

**File**: `internal/hub/feeds/apple.go:162`

```go
Severity: "high",
```

All Apple security releases get severity "high" regardless of content.

### H-M5: MSRC feed cursor comparison uses lexicographic ordering on "YYYY-Mon" strings

**File**: `internal/hub/feeds/msrc.go:68-69`

```go
if cursor != "" && updateID <= cursor {
    continue
}
```

Update IDs like "2024-Feb", "2024-Mar" are compared lexicographically. This works because the year prefix dominates, but "2024-Aug" < "2024-Feb" alphabetically, which would skip August updates if February was the cursor. The filtering logic is incorrect for within-year comparison.

### H-M6: Red Hat OVAL feed only fetches RHEL 9

**File**: `internal/hub/feeds/redhat.go:15`

```go
const redhatOVALURL = "https://www.redhat.com/security/data/oval/v2/RHEL9/rhel-9.oval.xml.bz2"
```

Only RHEL 9 is covered. RHEL 7 and RHEL 8 (still very common in enterprise) are not fetched. This should be configurable or multi-URL.

### H-M7: CISA KEV feed downloads full catalog every sync

**File**: `internal/hub/feeds/cisa_kev.go:37-89`

The full CISA KEV JSON catalog is downloaded on every sync, then filtered client-side by `dateAdded`. Unlike the NVD feed which uses `lastModStartDate` for incremental fetch, KEV has no server-side filtering. This works but wastes bandwidth as the catalog grows.

### H-M8: In-memory idempotency store used in production

**File**: `cmd/hub/main.go:301-302`

```go
idempotencyStore := idempotency.NewMemoryStore()
slog.Warn("using in-memory idempotency store, cached responses will not survive restarts")
```

The hub uses an in-memory idempotency store. On restart, all idempotency keys are lost. The TODO references PIQ-14 for Valkey integration.

### H-M9: `mintJWT` claims use `fmt.Sprintf` without escaping (beyond injection risk)

**File**: `internal/hub/auth/session.go:39`

Even beyond the injection risk (H-C1), using `fmt.Sprintf` for JWT construction means IAT/EXP timestamps are embedded as plain integers in a manually built JSON string, without any structural validation. If the time values somehow overflow or produce unexpected output, the JWT would be silently malformed.

### H-M10: Sync handler records history AFTER writing the HTTP response

**File**: `internal/hub/api/v1/sync.go:207-222`

The sync history insert happens after `json.NewEncoder(w).Encode(resp)` (line 202). If the history insert fails, the client already received a success response but the sync history is incomplete. This is a minor data consistency issue.

---

## Summary

| Severity | Count |
|----------|-------|
| Critical | 3 |
| Important | 8 |
| Minor | 10 |

### Key Themes

1. **Security**: JWT claims built via string interpolation (H-C1) is a real vulnerability if user-controlled strings reach `mintJWT`.
2. **Dead code**: License generator, tenant queries, agent binary queries, specialized fetchers are all built but never used (H-I2, H-I3, H-I4, H-I7).
3. **Consistency**: Three different UUID formatting functions with different behavior (H-C3, H-M2).
4. **Route mismatch**: RegistrationStatus route path doesn't match tests or documentation (H-C2).
5. **Feed limitations**: Hardcoded severities (Ubuntu, Apple), single RHEL version, broken MSRC cursor comparison (H-M3, H-M4, H-M5, H-M6).
6. **Incomplete integration**: RSA license signing exists but API uses JSON placeholder (H-I1).
