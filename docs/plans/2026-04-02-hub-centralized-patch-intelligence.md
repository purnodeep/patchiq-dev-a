# Hub Centralized Patch Intelligence

> **Goal**: Make Hub the single source of truth for CVEs, patches, and binaries. Eliminate duplicate NVD calls. Fix package name mapping.
>
> **Created**: 2026-04-02 | **Branch**: TBD

---

## 1. Problem

Four issues exist in the current Hub ↔ PM ↔ Agent pipeline:

### 1.1 Hub CVEs Are Minimal

Hub's `cve_feeds` table has 11 columns. PM's `cves` table has 16 columns with rich data (CVSS vector, attack vector, CWE, KEV due date, external references). Hub collects CVEs from 6 feeds but stores almost nothing useful about them — just `cve_id`, `severity`, and `source`.

### 1.2 Duplicate NVD/CISA API Calls

Hub fetches from NVD, CISA KEV, MSRC, RedHat, Ubuntu, Apple. PM **also** fetches from NVD and CISA KEV independently (`internal/server/cve/client.go`). This means:
- Double API calls to NVD (Hub + every PM instance)
- Double API calls to CISA KEV
- Hub's CVE data is wasted — PM never consumes it

### 1.3 No Patch Binaries

Hub's `patch_catalog` has `binary_ref` and `checksum_sha256` columns, but feeds never populate them. `BinaryFetcher` exists (`internal/hub/catalog/fetcher.go`) but is never wired in. MinIO is configured but unused for patch storage. The system is advisory-metadata-only — no actual installable files flow through.

### 1.4 Package Name Mismatch

Feeds report product names (e.g., `"curl"`, `"python"`). OS package managers use different names (e.g., `libcurl4` on Ubuntu, `curl-libs` on RHEL). No mapping exists. Agent runs `apt install curl` — may fail if the real package name is `libcurl4`. CVE matching also breaks: inventory reports `libcurl4` but CVE database has `curl`.

---

## 2. Design — Four Phases

### Phase 1: Enrich Hub CVE Records

**What changes:**

Hub's `cve_feeds` table gets 6 new columns to match PM's richness:

```sql
ALTER TABLE cve_feeds ADD COLUMN cvss_v3_vector TEXT NOT NULL DEFAULT '';
ALTER TABLE cve_feeds ADD COLUMN attack_vector TEXT NOT NULL DEFAULT '';
ALTER TABLE cve_feeds ADD COLUMN cwe_id TEXT NOT NULL DEFAULT '';
ALTER TABLE cve_feeds ADD COLUMN cisa_kev_due_date DATE;
ALTER TABLE cve_feeds ADD COLUMN external_references JSONB NOT NULL DEFAULT '[]';
ALTER TABLE cve_feeds ADD COLUMN nvd_last_modified TIMESTAMPTZ;
```

**Feed parser changes:**

| Feed | New data extracted |
|------|--------------------|
| NVD | CVSS vector, attack vector, CWE, references, last_modified (all available in API response) |
| CISA KEV | KEV due date, ransomware flag (already parsed, now stored) |
| MSRC | References (KB article URLs) |
| RedHat OVAL | References (advisory URLs), CWE if available |
| Ubuntu USN | References (USN URLs) |
| Apple | References (support article URLs) |

**RawEntry struct changes:**

```go
type RawEntry struct {
    // ... existing fields ...
    CVSSv3Vector     string            // NEW
    AttackVector     string            // NEW
    CweID            string            // NEW
    CISAKEVDueDate   string            // NEW (YYYY-MM-DD)
    References       []CVEReference    // NEW
    NVDLastModified  time.Time         // NEW
}

type CVEReference struct {
    URL    string
    Source string
}
```

**Pipeline changes:**

`ensureCVEFeed()` currently creates minimal CVE records. Updated to pass all enrichment fields. Add `UpsertCVEFeed` query that does `ON CONFLICT (cve_id) DO UPDATE` with all new fields.

### Phase 2: PM Fetches CVEs from Hub Only

**What changes:**

New Hub endpoint:

```
GET /api/v1/sync/cves?since=<RFC3339>
Authorization: Bearer <api_key>

Response:
{
  "cves": [
    {
      "cve_id": "CVE-2024-1234",
      "severity": "high",
      "description": "...",
      "cvss_v3_score": 8.8,
      "cvss_v3_vector": "CVSS:3.1/AV:N/AC:L/...",
      "attack_vector": "Network",
      "cwe_id": "CWE-79",
      "cisa_kev_due_date": "2024-06-15",
      "exploit_known": true,
      "external_references": [...],
      "published_at": "2024-01-15T...",
      "nvd_last_modified": "2024-02-10T...",
      "source": "nvd"
    }
  ],
  "server_time": "2026-04-02T..."
}
```

**PM-side changes:**

- New `CVEHubSyncWorker` in `internal/server/workers/` — calls Hub's `/api/v1/sync/cves` endpoint
- `hub_sync_state` table gets a `last_cve_sync_at` column (separate cursor for CVE sync)
- Remove direct NVD/CISA calls from PM — delete or disable `internal/server/cve/client.go` usage
- Keep `Correlator` intact — it receives CVE records from Hub instead of NVD, same correlation logic
- Keep `NVDSyncService` interface but swap the `CVEFetcher` implementation from `NVDClient` to `HubCVEClient`

**What PM still does independently:**

- CVE ↔ Patch correlation (Correlator)
- CVE ↔ Endpoint matching (Matcher)
- Policy evaluation
- Deployment decisions

### Phase 3: Package Name Mapping

**New Hub table:**

```sql
CREATE TABLE package_aliases (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    feed_product    TEXT NOT NULL,        -- "curl" (from feed)
    os_family       TEXT NOT NULL,        -- "linux"
    os_distribution TEXT NOT NULL,        -- "ubuntu-22.04", "rhel-9"
    os_package_name TEXT NOT NULL,        -- "libcurl4" (what apt/yum expects)
    confidence      TEXT NOT NULL DEFAULT 'manual',  -- manual|discovery|verified
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (feed_product, os_family, os_distribution)
);
```

**Three ways to populate:**

1. **Manual seeding** — ship with known mappings for top 200 packages (curl→libcurl4, python→python3, etc.)

2. **Discovery-driven auto-populate** — PM's discovery engine already scans APT/YUM repos and gets real OS package names. When discovery finds a package, match it against feed product names using fuzzy logic:
   - Exact match: `curl` = `curl`
   - Contains match: `libcurl4` contains `curl`
   - Known prefix/suffix patterns: `lib*`, `*-libs`, `*-dev`, `*-devel`
   - Store with `confidence: "discovery"`

3. **Agent inventory feedback** — Agent reports installed packages via `dpkg-query`/`rpm -qa`. When a patch deployment succeeds, record the mapping `(feed_product, os_distribution, os_package_name)` with `confidence: "verified"`.

**How it flows:**

```
Hub: patch_catalog entry has product="curl"
  ↓ lookup package_aliases for (curl, ubuntu-22.04)
  ↓ found: os_package_name = "libcurl4"
Hub sync response includes both: product="curl", os_package_name="libcurl4"
  ↓
PM: patches table stores package_name="libcurl4"
  ↓
Agent: receives InstallPatchPayload with Name="libcurl4"
  ↓
Agent: runs `apt install libcurl4` — works
```

**CVE matching also fixed:**

Agent inventory reports `libcurl4` installed. PM's CVE matcher looks up `libcurl4` in patches. Finds the patch with `package_name=libcurl4` which is linked to CVE-2024-1234. Match found.

### Phase 4: Binary Storage in Hub

**Architecture:**

```
Feed sync → patch_catalog entry created
  ↓ (async enrichment job)
BinaryFetchWorker → download binary from vendor repo
  ↓
Upload to MinIO → get object key
  ↓
Update patch_catalog.binary_ref = MinIO key
Update patch_catalog.checksum_sha256 = SHA256 of file
```

**Per-OS binary sources:**

| OS | Source | Method |
|----|--------|--------|
| Ubuntu/Debian | APT repos (archive.ubuntu.com, security.ubuntu.com) | Download `.deb` via `Packages.gz` metadata → direct URL |
| RHEL/CentOS | YUM repos (mirrorlist.centos.org, cdn.redhat.com) | Download `.rpm` via `repomd.xml` → primary.xml → direct URL |
| Windows | Microsoft Update Catalog (catalog.update.microsoft.com) | Parse KB article → download `.msu`/`.cab` |
| macOS | Apple CDN (swcdn.apple.com) | Parse release page → download `.pkg`/`.dmg` |

**New River job:** `BinaryFetchJobArgs` — triggered after each feed sync completes. For each catalog entry without a `binary_ref`:
1. Look up `package_aliases` to get OS package name
2. Query the appropriate OS repo for the download URL
3. Download the binary
4. Compute SHA256
5. Upload to MinIO
6. Update `patch_catalog` entry

**MinIO bucket structure:**

```
patchiq-binaries/
  ├── apt/
  │   ├── ubuntu-22.04/
  │   │   ├── libcurl4_7.81.0-1ubuntu1.16_amd64.deb
  │   │   └── ...
  │   └── ubuntu-24.04/
  ├── yum/
  │   └── rhel-9/
  ├── msu/
  │   └── windows-server-2022/
  └── pkg/
      └── macos-14/
```

**PM/Agent consumption:**

- During Hub→PM sync, `binary_ref` and `checksum_sha256` are included
- PM stores them in `patches.package_url` and `patches.checksum_sha256`
- Agent can download binary from Hub (via PM proxy or direct) instead of relying on OS repos
- Fallback: if binary not available in Hub, agent still uses OS package manager

---

## 3. Dependency Graph

```
Phase 1 (Enrich Hub CVEs)
    ↓ required by
Phase 2 (PM fetches CVEs from Hub)

Phase 3 (Package name mapping)
    ↓ required by  
Phase 4 (Binary storage)

Phase 1 and Phase 3 are independent — can run in parallel.
```

---

## 4. Database Changes Summary

### Hub Migrations

| Migration | Phase | What |
|-----------|-------|------|
| `00013_cve_enrichment.sql` | 1 | Add 6 columns to `cve_feeds`, rename `cvss_score` → `cvss_v3_score` for consistency with PM |
| `00014_cve_sync_endpoint.sql` | 2 | Add index for CVE sync queries |
| `00015_package_aliases.sql` | 3 | New `package_aliases` table (global, no tenant_id — Hub is single-tenant SaaS) |
| `00016_binary_fetch_state.sql` | 4 | Track binary fetch jobs per catalog entry |

### Server Migrations

| Migration | Phase | What |
|-----------|-------|------|
| `047_hub_cve_sync.sql` | 2 | Add `last_cve_sync_at` to `hub_sync_state` |

---

## 5. Files Changed Per Phase

### Phase 1
- `internal/hub/store/migrations/00013_cve_enrichment.sql` (new)
- `internal/hub/store/queries/cve_feeds.sql` (update upsert query)
- `internal/hub/feeds/feed.go` (enrich RawEntry struct)
- `internal/hub/feeds/nvd.go` (extract CVSS vector, CWE, attack vector, refs)
- `internal/hub/feeds/cisa_kev.go` (extract KEV due date)
- `internal/hub/feeds/msrc.go` (extract references)
- `internal/hub/feeds/redhat.go` (extract references)
- `internal/hub/feeds/ubuntu.go` (extract references)
- `internal/hub/feeds/apple.go` (extract references)
- `internal/hub/catalog/pipeline.go` (pass enrichment fields to ensureCVEFeed)
- `internal/hub/store/sqlcgen/` (regenerate)

### Phase 2
- `internal/hub/api/v1/sync.go` (add CVE sync endpoint)
- `internal/hub/store/queries/cve_feeds.sql` (add ListCVEsUpdatedSince query)
- `internal/hub/store/migrations/00014_cve_sync_endpoint.sql` (new)
- `internal/server/workers/cve_hub_sync.go` (new — replaces direct NVD sync)
- `internal/server/store/migrations/047_hub_cve_sync.sql` (new)
- `internal/server/store/queries/hub_sync.sql` (add CVE cursor queries)
- `internal/server/cve/sync.go` (swap fetcher from NVDClient to HubCVEClient)
- `internal/server/cve/hub_client.go` (new — Hub CVE fetch client)
- `cmd/server/main.go` (wire new CVE sync worker)
- `internal/server/store/sqlcgen/` (regenerate)
- `internal/hub/store/sqlcgen/` (regenerate)

### Phase 3
- `internal/hub/store/migrations/00015_package_aliases.sql` (new)
- `internal/hub/store/queries/package_aliases.sql` (new)
- `internal/hub/catalog/pipeline.go` (lookup alias during sync)
- `internal/hub/api/v1/sync.go` (include os_package_name in response)
- `internal/server/workers/catalog_sync.go` (store os_package_name)
- `internal/server/discovery/store_adapter.go` (report discovered mappings back)
- `internal/hub/store/sqlcgen/` (regenerate)
- Seed file: `internal/hub/store/seeds/package_aliases.sql` (top 200 mappings)

### Phase 4
- `internal/hub/store/migrations/00016_binary_fetch_state.sql` (new)
- `internal/hub/catalog/fetcher.go` (wire into pipeline, implement per-OS fetchers)
- `internal/hub/catalog/fetcher_apt.go` (new — download .deb from APT repos)
- `internal/hub/catalog/fetcher_yum.go` (new — download .rpm from YUM repos)
- `internal/hub/catalog/fetcher_msu.go` (new — download .msu from Microsoft)
- `internal/hub/catalog/fetcher_apple.go` (new — download .pkg from Apple)
- `internal/hub/workers/binary_fetch.go` (new River job)
- `internal/hub/store/queries/patch_catalog.sql` (update binary_ref query)
- `cmd/hub/main.go` (wire BinaryFetchWorker)
- `internal/hub/store/sqlcgen/` (regenerate)

---

## 6. Risk Assessment

| Risk | Severity | Mitigation |
|------|----------|------------|
| NVD rate limiting during enriched parsing | Medium | Already handled — API key support exists, no extra calls needed (same response has all fields) |
| PM downtime during CVE source cutover | High | Feature flag via `hub_sync_state`: if row exists with `hub_url` set, PM uses Hub for CVEs; if Hub unreachable, falls back to direct NVD. No code flag needed — presence of config is the flag. |
| Package alias table incomplete | Medium | Start with top 200 packages, auto-populate via discovery, agent feedback loop fills gaps |
| Binary download failures (vendor repos down) | Medium | Async retry with backoff, mark as "pending", agent falls back to OS repos |
| MinIO storage growth | Low | Retention policy — keep only latest version per package per distro |
| Migration on existing data | Low | All new columns have defaults, backward compatible |

---

## 7. New Domain Events

Per project rules, every write operation must emit a domain event.

| Event | Phase | Emitted When |
|-------|-------|-------------|
| `cve_feed.enriched` | 1 | CVE record updated with enrichment fields |
| `cve_sync.completed` | 2 | Hub CVE sync endpoint returns data to PM |
| `cve_sync.failed` | 2 | Hub CVE sync endpoint fails |
| `package_alias.created` | 3 | New alias mapping created |
| `package_alias.updated` | 3 | Alias mapping updated (e.g., confidence upgraded) |
| `binary.fetched` | 4 | Binary successfully downloaded and stored in MinIO |
| `binary.fetch_failed` | 4 | Binary download failed |

## 8. What Doesn't Change

- Agent patcher implementations (APT, YUM, WUA, Homebrew) — same interface
- gRPC protocol between PM and Agent — same protobuf messages
- PM's Correlator logic — same algorithm, different data source
- PM's deployment/wave dispatcher — same code, just gets better package_name data
- Hub's feed schedule and cursor mechanism — same periodic River jobs
- Tenant isolation — same RLS everywhere
