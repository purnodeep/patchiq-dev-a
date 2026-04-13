# Hub Manager E2E

**Status**: Planned
**Wave**: 1 — Foundation Polish
**Moved from**: Partially M3 (Structured Patch Pipeline) + M4 (MSP infra)
**Priority**: Critical — the platform's data source must work reliably

---

## Problem

The Hub Manager has feed infrastructure but doesn't work end-to-end for production:

- NVD feed perpetually 429'd without API key — 0 entries from the largest CVE database
- Apple feed returns HTML, not JSON — parser fails silently
- MSRC feed returns 0 entries — cursor/pagination not bootstrapped
- Hub→PM sync exists but "Synced to PMs: 0%" — per-entry tracking not updating
- No PM client registration flow — Fleet Topology shows "0/0 Connected PMs"
- No catalog publish pipeline — no review/approve workflow for new entries
- Binary distribution: fetcher exists but no automated download triggers
- Feed error handling: failed syncs retry immediately instead of backing off

## Goals

1. **All 6 feeds syncing reliably** — NVD (with API key), CISA KEV, MSRC, RedHat OVAL, Ubuntu USN, Apple
2. **Hub→PM sync working for real** — catalog entries flow to PM, per-entry sync tracking updates, PM registers as client
3. **Catalog publish pipeline** — new entries go through review → approve → publish flow
4. **Binary distribution** — automated binary fetch for high-confidence installer types
5. **Feed monitoring** — accurate health dashboard, error alerting, retry backoff
6. **Feed configuration UI** — API keys, intervals, filters, severity mappings all configurable from Hub UI

## Deliverables

### Feed Reliability
- [ ] NVD: API key configuration (env var + Hub settings UI), proper rate limit handling (600ms with key, 6s without), exponential backoff on 429
- [ ] Apple: fix parser to handle HTML response (scrape security releases page or use alternative data source)
- [ ] MSRC: bootstrap cursor with current month's update ID, handle CVRF XML response pagination
- [ ] Ubuntu USN: verified working (12 entries), increase initial sync window
- [ ] All feeds: configurable retry backoff (1min, 5min, 15min, 1hr progression)
- [ ] Feed health alerting: emit domain events on repeated failures, surface in dashboard

### Hub→PM Sync
- [ ] PM client registration: PM registers with Hub on first catalog sync (auto-create client record)
- [ ] Per-entry sync tracking: when PM pulls catalog entries, mark them as synced for that client
- [ ] Delta sync optimization: only send entries modified since client's last sync timestamp
- [ ] Sync health dashboard: per-client sync status, lag, error rate

### Catalog Publish Pipeline
- [ ] New entry states: `pending_review` → `approved` → `published` → `withdrawn`
- [ ] Review queue UI: list pending entries, approve/reject with notes
- [ ] Auto-approve rules: entries from trusted sources (CISA KEV) auto-publish
- [ ] Manual entry creation: admin can add custom catalog entries

### Binary Distribution
- [ ] Automated binary fetch for DEB/RPM packages (resolve download URL from feed metadata)
- [ ] MinIO upload with SHA256 verification
- [ ] Server-side cache population from Hub binary store
- [ ] Binary availability indicator in catalog UI

### Feed Configuration UI
- [ ] Settings page per feed: API key, sync interval, severity filter, OS filter, severity mapping
- [ ] Test connection button (dry-run fetch with 1 entry)
- [ ] Sync history with detailed error messages

## Dependencies
- MinIO (already running)
- NVD API key (free registration at https://nvd.nist.gov/developers/request-an-api-key)

## License Gating
- Basic feeds (NVD, CISA KEV): all tiers
- Extended feeds (MSRC, RedHat, Ubuntu, Apple): PROFESSIONAL+
- Catalog publish pipeline: PROFESSIONAL+
- Binary distribution: ENTERPRISE
