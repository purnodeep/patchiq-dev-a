# Hub Centralized Patch Intelligence — Implementation Plan

> **Design doc**: `docs/plans/2026-04-02-hub-centralized-patch-intelligence.md`
> **Created**: 2026-04-02

---

## Phase 1: Enrich Hub CVE Records

### Task 1.1 — Hub migration: add enrichment columns to cve_feeds
- **File**: `internal/hub/store/migrations/00013_cve_enrichment.sql`
- **What**: Rename `cvss_score` → `cvss_v3_score` for PM consistency. Add `cvss_v3_vector`, `attack_vector`, `cwe_id`, `cisa_kev_due_date`, `external_references` (JSONB), `nvd_last_modified` to `cve_feeds`.
- **Test**: Run `make migrate-hub && make migrate-status` — verify columns exist
- **Depends on**: Nothing

### Task 1.2 — Update Hub CVE sqlc queries
- **File**: `internal/hub/store/queries/cve_feeds.sql`
- **What**: Update `CreateCVEFeed` to accept all new fields. Add `UpsertCVEFeed` query with `ON CONFLICT (cve_id) DO UPDATE` for all enrichment fields. Update `GetCVEFeedByCVEID` to return new fields.
- **Test**: `make sqlc` succeeds, generated code compiles
- **Depends on**: 1.1

### Task 1.3 — Enrich RawEntry struct and add CVEReference type
- **File**: `internal/hub/feeds/feed.go`
- **What**: Add `CVSSv3Vector`, `AttackVector`, `CweID`, `CISAKEVDueDate`, `References []CVEReference`, `NVDLastModified` fields to `RawEntry`. Add `CVEReference` struct with `URL` and `Source` fields.
- **Test**: Write unit test in `internal/hub/feeds/feed_test.go` — create RawEntry with new fields, validate passes
- **Depends on**: Nothing

### Task 1.4 — Update NVD feed parser to extract rich CVE data
- **File**: `internal/hub/feeds/nvd.go`
- **What**:
  1. Write failing test in `internal/hub/feeds/nvd_test.go` with real NVD API response JSON fixture. Assert CVSSv3Vector, AttackVector, CweID, References, NVDLastModified are populated.
  2. Update `parsePage()` to extract: CVSS v3 vector string from `metrics.cvssMetricV31[].cvssData.vectorString`, attack vector from vector string, CWE ID from `weaknesses[].description[]`, references from `references[]`, lastModified timestamp.
- **Test**: Failing test passes after implementation.
- **Depends on**: 1.3

### Task 1.5 — Update CISA KEV feed parser to extract due date
- **File**: `internal/hub/feeds/cisa_kev.go`
- **What**:
  1. Write failing test in `internal/hub/feeds/cisa_kev_test.go` with KEV response fixture. Assert `CISAKEVDueDate` is set on RawEntry.
  2. Move `dueDate` from Metadata map to `RawEntry.CISAKEVDueDate`. Add ransomware campaign info to References.
- **Test**: Failing test passes after implementation.
- **Depends on**: 1.3

### Task 1.6 — Update MSRC feed parser to extract references
- **File**: `internal/hub/feeds/msrc.go`
- **What**:
  1. Write failing test in `internal/hub/feeds/msrc_test.go` with MSRC response fixture. Assert References contains KB article URLs.
  2. Convert KB article URLs from Metadata to `RawEntry.References` as `CVEReference` entries.
- **Test**: Failing test passes after implementation.
- **Depends on**: 1.3

### Task 1.7 — Update RedHat OVAL feed parser to extract references
- **File**: `internal/hub/feeds/redhat.go`
- **What**:
  1. Write failing test in `internal/hub/feeds/redhat_test.go` with OVAL XML fixture. Assert References populated, CweID extracted if present.
  2. Add advisory URL to References array. Extract CWE from OVAL XML if present.
- **Test**: Failing test passes after implementation.
- **Depends on**: 1.3

### Task 1.8 — Update Ubuntu USN feed parser to extract references
- **File**: `internal/hub/feeds/ubuntu.go`
- **What**:
  1. Write failing test in `internal/hub/feeds/ubuntu_test.go` with USN response fixture. Assert References populated.
  2. Add USN URL and CVE links to References.
- **Test**: Failing test passes after implementation.
- **Depends on**: 1.3

### Task 1.9 — Update Apple feed parser to extract references
- **File**: `internal/hub/feeds/apple.go`
- **What**:
  1. Write failing test in `internal/hub/feeds/apple_test.go` with Apple response fixture. Assert References populated.
  2. Add support article URL to References.
- **Test**: Failing test passes after implementation.
- **Depends on**: 1.3

### Task 1.10 — Update catalog pipeline to store enriched CVEs
- **File**: `internal/hub/catalog/pipeline.go`
- **Test file**: `internal/hub/catalog/pipeline_test.go`
- **What**:
  1. Write failing test: sync two feeds (NVD + CISA KEV) that report the same CVE. Assert the upserted `cve_feeds` row has both CVSS vector (from NVD) and KEV due date (from CISA).
  2. Update `ensureCVEFeed()` to pass all enrichment fields from RawEntry to the new `UpsertCVEFeed` query. On conflict, merge: keep non-empty fields from whichever feed provides them.
- **Test**: Failing test passes. Multi-feed CVE enrichment verified.
- **Depends on**: 1.2, 1.3, 1.4-1.9

### Task 1.11 — Define and emit domain events for CVE enrichment
- **File**: `internal/hub/events/topics.go`
- **What**: Add `CVEFeedEnriched` event constant. Emit from `ensureCVEFeed()` when a CVE record is created or updated with enrichment data.
- **Test**: Verify event emitted in pipeline_test.go
- **Depends on**: 1.10

### Task 1.12 — Regenerate sqlc and verify build
- **Command**: `make sqlc && make build && make test`
- **Depends on**: 1.11

---

## Phase 2: PM Fetches CVEs from Hub Only

### Task 2.1 — Hub: Add CVE sync query
- **File**: `internal/hub/store/queries/cve_feeds.sql`
- **What**: Add `ListCVEFeedsUpdatedSince` query — returns all CVE records where `updated_at > $1`, ordered by `updated_at ASC`.
- **Test**: `make sqlc` succeeds
- **Depends on**: Phase 1 complete

### Task 2.2 — Hub: Add CVE sync REST endpoint and register route
- **Files**: `internal/hub/api/v1/sync.go`, Hub router file (e.g., `internal/hub/api/v1/router.go` or `cmd/hub/main.go` route registration)
- **What**: Add handler for `GET /api/v1/sync/cves?since=<RFC3339>`. Same auth as existing sync endpoint (Bearer token). Returns JSON with `cves` array and `server_time`. Register route in Hub router.
- **Test**: Write unit test — call endpoint with since param, verify response shape and auth. Test 401 on bad token.
- **Depends on**: 2.1

### Task 2.3 — Hub: Emit domain events for CVE sync
- **File**: `internal/hub/events/topics.go`, `internal/hub/api/v1/sync.go`
- **What**: Add `CVESyncCompleted` and `CVESyncFailed` event constants. Emit from the CVE sync handler.
- **Test**: Verify events emitted in sync handler test.
- **Depends on**: 2.2

### Task 2.4 — Server migration: add CVE sync cursor to hub_sync_state
- **File**: `internal/server/store/migrations/047_hub_cve_sync.sql`
- **What**: `ALTER TABLE hub_sync_state ADD COLUMN last_cve_sync_at TIMESTAMPTZ;`
- **Test**: `make migrate && make migrate-status`
- **Depends on**: Nothing

### Task 2.5 — Server: Add Hub CVE sync queries
- **File**: `internal/server/store/queries/hub_sync.sql`
- **What**: Add `UpdateHubCVESyncCompleted` and `UpdateHubCVESyncFailed` queries that update `last_cve_sync_at`.
- **Test**: `make sqlc` succeeds
- **Depends on**: 2.4

### Task 2.6 — Server: Create HubCVEClient
- **File**: `internal/server/cve/hub_client.go` (new)
- **What**:
  1. Write failing test in `internal/server/cve/hub_client_test.go` — mock HTTP server returns Hub CVE response, assert `[]CVERecord` parsed correctly with all enrichment fields.
  2. Implement `CVEFetcher` interface. `FetchCVEs()` calls Hub's `/api/v1/sync/cves?since=...`. `FetchKEV()` returns empty map (Hub already enriches with KEV). Parse Hub response into `[]CVERecord`.
- **Test**: Failing test passes after implementation.
- **Depends on**: 2.2

### Task 2.7 — Server: Wire HubCVEClient into NVDSyncService
- **File**: `cmd/server/main.go`
- **What**: When `hub_sync_state` row exists with `hub_url` set, use `HubCVEClient` as the `CVEFetcher` instead of `NVDClient`. If Hub is not configured (no hub_sync_state), fall back to direct NVD (backward compatible). No code feature flag needed — presence of hub_sync_state config is the flag.
- **Test**: Write test — with hub_sync_state configured, verify HubCVEClient is selected. Without config, verify NVDClient is selected.
- **Depends on**: 2.6

### Task 2.8 — Server: Add CVE sync to CatalogSyncWorker or create separate worker
- **File**: `internal/server/workers/cve_hub_sync.go` (new) or extend `catalog_sync.go`
- **What**: After patch catalog sync completes, also sync CVEs from Hub. Use `last_cve_sync_at` as cursor. Update cursor on success. Emit `CatalogSyncStarted`/`CatalogSynced` events.
- **Test**: Write unit test — mock Hub CVE endpoint, verify worker fetches and upserts CVEs, updates cursor.
- **Depends on**: 2.5, 2.6

### Task 2.9 — Regenerate sqlc and verify build
- **Command**: `make sqlc && make build && make test`
- **Depends on**: 2.8

---

## Phase 3: Package Name Mapping

### Task 3.1 — Hub migration: create package_aliases table
- **File**: `internal/hub/store/migrations/00015_package_aliases.sql`
- **What**: Create `package_aliases` table with `feed_product`, `os_family`, `os_distribution`, `os_package_name`, `confidence`, timestamps. Unique on `(feed_product, os_family, os_distribution)`. Global table (no tenant_id — Hub is single-tenant SaaS).
- **Test**: `make migrate-hub && make migrate-status`
- **Depends on**: Nothing

### Task 3.2 — Hub: Add package_aliases sqlc queries
- **File**: `internal/hub/store/queries/package_aliases.sql` (new)
- **What**: Queries: `GetPackageAlias(feed_product, os_family, os_distribution)`, `UpsertPackageAlias(...)`, `ListPackageAliases(limit, offset)`, `ListPackageAliasesByProduct(feed_product)`, `DeletePackageAlias(id)`.
- **Test**: `make sqlc` succeeds
- **Depends on**: 3.1

### Task 3.3 — Seed top 200 package aliases
- **File**: `internal/hub/store/seeds/package_aliases.sql` (new)
- **What**: Insert known mappings for common packages across Ubuntu (20.04, 22.04, 24.04) and RHEL (8, 9). Examples:
  - curl → curl (ubuntu-*), curl (rhel-*) [direct match]
  - curl → libcurl4 (ubuntu-*, for the library)
  - python → python3 (ubuntu-22.04), python3 (rhel-9)
  - openssl → libssl3 (ubuntu-22.04), openssl-libs (rhel-9)
  - nginx → nginx (direct match on all)
  - kernel → linux-image-generic (ubuntu-*), kernel (rhel-*)
- **Test**: Load seed, query aliases, verify mappings exist
- **Depends on**: 3.2

### Task 3.4 — Hub: Lookup alias in catalog pipeline during enrichment
- **File**: `internal/hub/catalog/pipeline.go`
- **What**: After upserting a catalog entry, lookup `package_aliases` by `(product, os_family)`. If found, store the `os_package_name` on the catalog entry (may need a new column or use existing `product` field as canonical and add `os_package_name`).
- **Test**: Write unit test in `internal/hub/catalog/pipeline_test.go` — catalog entry for "curl" on "linux"/"ubuntu-22.04" resolves to "libcurl4".
- **Depends on**: 3.2

### Task 3.5 — Hub: Include os_package_name in sync response
- **File**: `internal/hub/api/v1/sync.go`
- **What**: When building sync response, for each catalog entry, include `os_package_name` field. If alias was resolved in pipeline, use it. Otherwise, fall back to `product` field.
- **Test**: Write unit test — catalog entry with known alias returns correct os_package_name in sync JSON.
- **Depends on**: 3.4

### Task 3.6 — Server: Store os_package_name from Hub sync
- **File**: `internal/server/workers/catalog_sync.go`
- **What**: Parse `os_package_name` from Hub response. Use it as `PackageName` in `UpsertDiscoveredPatch`. This means `patches.package_name` will have the correct OS-specific name.
- **Test**: Write unit test — mock Hub response with os_package_name="libcurl4", verify patches table has package_name="libcurl4".
- **Depends on**: 3.5

### Task 3.7 — Hub API: CRUD endpoints for package_aliases and register routes
- **File**: `internal/hub/api/v1/package_aliases.go` (new), Hub router file
- **What**: REST endpoints: `GET /api/v1/package-aliases`, `POST /api/v1/package-aliases`, `PUT /api/v1/package-aliases/{id}`, `DELETE /api/v1/package-aliases/{id}`. Register all routes in Hub router.
- **Test**: Write unit tests for each endpoint.
- **Depends on**: 3.2

### Task 3.8 — Hub API: Discovery endpoint for auto-populating aliases
- **File**: `internal/hub/api/v1/package_aliases.go`
- **What**: Add `POST /api/v1/package-aliases/discover` endpoint. Accepts batch of `[(os_package_name, os_family, os_distribution)]`. For each, fuzzy-match against existing feed products (exact match, contains match, known prefix/suffix patterns like `lib*`, `*-libs`). Create aliases with `confidence: "discovery"`. Register route.
- **Test**: Write unit test — POST discover with "libcurl4" on "ubuntu-22.04", assert alias created linking to "curl" feed product.
- **Depends on**: 3.7

### Task 3.9 — Hub: Emit domain events for alias operations
- **File**: `internal/hub/events/topics.go`, `internal/hub/api/v1/package_aliases.go`
- **What**: Add `PackageAliasCreated` and `PackageAliasUpdated` event constants. Emit from CRUD and discover endpoints.
- **Test**: Verify events emitted in handler tests.
- **Depends on**: 3.7, 3.8

### Task 3.10 — Server: Discovery-driven alias reporting
- **File**: `internal/server/discovery/store_adapter.go`
- **What**: When discovery finds packages in APT/YUM repos, report discovered mappings back to Hub via `POST /api/v1/package-aliases/discover`. Batch report.
- **Test**: Write unit test — discovery finds "libcurl4" on ubuntu-22.04, calls Hub discover endpoint.
- **Depends on**: 3.8

### Task 3.11 — Regenerate sqlc and verify build
- **Command**: `make sqlc && make build && make test`
- **Depends on**: 3.10

---

## Phase 4: Binary Storage in Hub

### Task 4.1 — Hub migration: binary fetch tracking
- **File**: `internal/hub/store/migrations/00016_binary_fetch_state.sql`
- **What**: Create `binary_fetch_state` table: `catalog_id` (FK), `os_distribution`, `status` (pending|fetching|complete|failed), `binary_ref`, `checksum_sha256`, `file_size_bytes`, `fetch_url`, `error_message`, `retry_count`, `last_attempt_at`. Unique on `(catalog_id, os_distribution)`.
- **Test**: `make migrate-hub`
- **Depends on**: Nothing

### Task 4.2 — Hub: Add binary fetch sqlc queries
- **File**: `internal/hub/store/queries/binary_fetch.sql` (new)
- **What**: `ListPendingBinaryFetches(limit)`, `CreateBinaryFetchState(...)`, `UpdateBinaryFetchSuccess(...)`, `UpdateBinaryFetchFailed(...)`, `GetBinaryFetchState(catalog_id, os_distribution)`.
- **Test**: `make sqlc` succeeds
- **Depends on**: 4.1

### Task 4.3 — Hub: APT binary fetcher
- **File**: `internal/hub/catalog/fetcher_apt.go` (new)
- **What**:
  1. Write failing test in `internal/hub/catalog/fetcher_apt_test.go` — mock HTTP server serves fake Packages.gz and .deb. Assert correct download, checksum computation, and MinIO upload.
  2. Implement: fetch `Packages.gz` from `archive.ubuntu.com`, parse for package, extract `Filename`, download .deb, compute SHA256, upload to MinIO under `apt/{codename}/{filename}`.
- **Test**: Failing test passes after implementation.
- **Depends on**: Nothing

### Task 4.4 — Hub: YUM binary fetcher
- **File**: `internal/hub/catalog/fetcher_yum.go` (new)
- **What**:
  1. Write failing test in `internal/hub/catalog/fetcher_yum_test.go` — mock YUM repo.
  2. Implement: fetch `repomd.xml`, parse for `primary.xml.gz`, decompress, find package, download .rpm, compute SHA256, upload to MinIO.
- **Test**: Failing test passes after implementation.
- **Depends on**: Nothing

### Task 4.5 — Hub: MSU binary fetcher (Windows)
- **File**: `internal/hub/catalog/fetcher_msu.go` (new)
- **What**:
  1. Write failing test in `internal/hub/catalog/fetcher_msu_test.go` — mock Microsoft catalog.
  2. Implement: parse KB article → find download link → download .msu/.cab → compute SHA256 → upload to MinIO.
- **Test**: Failing test passes after implementation.
- **Depends on**: Nothing

### Task 4.6 — Hub: Apple binary fetcher (macOS)
- **File**: `internal/hub/catalog/fetcher_apple.go` (new)
- **What**:
  1. Write failing test in `internal/hub/catalog/fetcher_apple_test.go` — mock Apple CDN.
  2. Implement: parse Apple release page → find download URL → download .pkg/.dmg → compute SHA256 → upload to MinIO.
- **Test**: Failing test passes after implementation.
- **Depends on**: Nothing

### Task 4.7 — Hub: BinaryFetchWorker (River job)
- **File**: `internal/hub/workers/binary_fetch.go` (new)
- **What**:
  1. Write failing test — mock fetcher returns binary, verify catalog entry updated with binary_ref and checksum.
  2. Implement River periodic job. For each catalog entry without binary_ref: determine OS, lookup package_aliases for correct name, select fetcher (APT/YUM/MSU/Apple), download and upload, update catalog entry and binary_fetch_state. Max 3 retries with exponential backoff.
- **Test**: Failing test passes. Verify retry logic on failure.
- **Depends on**: 4.2, 4.3, 4.4, 4.5, 4.6, Phase 3 (package aliases)

### Task 4.8 — Hub: Emit domain events for binary operations
- **File**: `internal/hub/events/topics.go`, `internal/hub/workers/binary_fetch.go`
- **What**: Add `BinaryFetched` and `BinaryFetchFailed` event constants. Emit from BinaryFetchWorker on success/failure.
- **Test**: Verify events emitted in worker test.
- **Depends on**: 4.7

### Task 4.9 — Hub: Wire BinaryFetchWorker into startup and register route
- **File**: `cmd/hub/main.go`
- **What**: Register `BinaryFetchWorker` with River. Configure as periodic job (run 30 minutes after each feed sync). Pass MinIO client to worker.
- **Test**: Start Hub, verify worker registers.
- **Depends on**: 4.7

### Task 4.10 — Hub: Serve binaries via API
- **File**: `internal/hub/api/v1/binaries.go` (new or existing), Hub router file
- **What**: `GET /api/v1/binaries/{binary_ref}` — streams binary from MinIO. Auth required (Bearer token). Include `Content-Disposition` and `X-Checksum-SHA256` headers. Register route in Hub router.
- **Test**: Write unit test — upload binary to MinIO, request via API, verify content and headers match.
- **Depends on**: 4.7

### Task 4.11 — Hub: Include binary info in sync response
- **File**: `internal/hub/api/v1/sync.go`
- **What**: Include `binary_ref` and `checksum_sha256` in sync response for catalog entries that have binaries available.
- **Test**: Write unit test — catalog entry with binary_ref appears in sync response with both fields.
- **Depends on**: 4.7

### Task 4.12 — Regenerate sqlc and verify build
- **Command**: `make sqlc && make build && make test`
- **Depends on**: 4.11

---

## Execution Order

```
Week 1-2: Phase 1 (Tasks 1.1-1.12) + Phase 3 Tasks 3.1-3.3 (in parallel)
Week 2-3: Phase 2 (Tasks 2.1-2.9) + Phase 3 Tasks 3.4-3.11 (in parallel)
Week 3-5: Phase 4 (Tasks 4.1-4.12)
```

Each phase gets its own branch, PR, and squash merge.
