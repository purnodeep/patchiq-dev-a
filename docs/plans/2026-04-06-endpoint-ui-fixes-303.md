# Endpoint UI/UX Fixes — Issue #303

**Date:** 2026-04-06
**Author:** Rishab (Claude-assisted)
**Track:** Quick Track
**Scope:** 14 targeted fixes across Endpoints list, Endpoint detail tabs, Dashboard

---

## Objective

Fix 14 specific UI/UX issues in the Endpoints pages and one Dashboard widget. No redesigns — only precise, minimal changes to broken, misleading, or confusing elements.

---

## Fix Catalog

### A. Endpoints List Page

#### A1. Expanded row — memory gauge missing

**Current:** `ExpRow` in `EndpointsPage.tsx` (line ~311-347) renders CPU%, Memory%, Disk% in a loop over 3 metrics extracted from endpoint data. Memory data may be missing or not extracted from `hardware_details`.

**Fix:** Ensure Memory% is always extracted from `hardware_details.memory` fallback path. If data is null, show "N/A" instead of hiding the gauge.

#### A2. Expanded row — patch summary card layout broken

**Current:** `ExpRow` (line ~348-387) renders severity counts inline. Layout may break with long counts or missing data.

**Fix:** Ensure card has fixed minimum width, proper flex-wrap, and handles zero counts gracefully (show "0" not empty).

#### A3. Tags flow — create, assign, cancel all broken

**Current:**
- `CreateTagDialog.tsx` — mutation calls `createTag.mutate()` but may have incorrect payload or missing query invalidation.
- Cancel button (line ~150) may not reset form state properly.
- `AssignTagsDialog.tsx` — receives `selectedEndpointIds` as prop but no auto-selection from current context.

**Fix:**
- Debug and fix createTag mutation (check API payload matches backend schema).
- Ensure cancel resets all state (key, value, description, selectedIds).
- Fix cancel button sizing (match "Create" button dimensions).
- In AssignTagsDialog, accept optional `preSelectedEndpointIds` to auto-check endpoints.
- Enable existing tag assignment flow (ensure tags query returns available tags).

---

### B. Endpoint Detail Page

#### B1. Software tab — System/Third-party filter

**Current:** `SoftwareTab.tsx` has search, source, arch filters. No System vs Third-party classification.

**Fix:** Add a toggle/dropdown filter: "All / System / Third-party". Classification heuristic: packages from OS repos (apt, yum, apk, system) = System; everything else = Third-party.

#### B2. Blast Radius — cleanup and info tooltip

**Current:** `OverviewTab.tsx` BlastRadiusGraph (line ~287-857):
- 4 node types: CVE, Patch, Port/Interface, Tag
- Blast Score pill (top-right) with no explanation
- Legend shows all 4 types
- Filters: none explicit (all nodes always shown)

**Fix:**
- Remove Port/Interface and Tag node types from the graph
- Add filter toggles: "CVEs" / "Patches" (default both on)
- Add info (i) button next to Blast Score pill with tooltip:
  > "Blast Score (0-10) measures exposure risk. Calculation: each critical CVE adds 1.0, high adds 0.6, medium adds 0.3, low adds 0.1. Pending patches add 0.3 each. Capped at 10."
- Update legend to show only CVE + Patch
- Clean up visual clutter: reduce star count, simplify animations

#### B3. Risk Breakdown — visibility and drill-down

**Current:** `OverviewTab.tsx` RiskBreakdown (line ~1149-1226):
- Score displayed with donut gauge
- 5 breakdown bars with labels and scores
- Bar colors: red if >0.5, amber if >0.3, else muted
- Numbers use `(score * 10).toFixed(1)` format

**Fix:**
- Increase number font-weight to 700 (from current ~500)
- Use `var(--text-emphasis)` for numbers instead of muted color
- Lighten bar backgrounds (increase opacity from ~0.15 to ~0.25)
- Make each breakdown row clickable — navigate to relevant tab (CVEs tab for "Unpatched CVEs", Compliance tab for "Compliance Gaps", etc.)
- Add hover state (background highlight + cursor pointer)

#### B4. Tag line, Scan Now feedback, Decommission wording

**Current:**
- Tags section in detail page has an extra `<hr>` or border-bottom divider
- Scan Now button (line ~330-360) uses `triggerScan.mutate()` with toast — but visual feedback may not be prominent enough
- More menu says "Decommission Endpoint" (line ~393-399) but action is delete

**Fix:**
- Remove the extra line/divider near tags
- Scan Now: show a visible loading state on the button (spinner + "Scanning...") plus toast notification
- Change "Decommission Endpoint" to "Delete Endpoint" in the dropdown menu item

#### B5. Hardware tab — frequency, SoC, serial, battery

**Current:** `HardwareTab.tsx`:
- `formatMHz()` (line ~107-110) converts MHz to GHz display
- CPU model from `hw?.cpu?.model_name`
- Storage shows name/model but no serial
- No battery section exists

**Fix:**
- Fix `formatMHz()` — ensure it handles null/0/undefined values, displays "—" for missing data
- For Apple Silicon: show SoC chip name (e.g., "Apple M2") from `model_name` field, label as "SoC" instead of "Processor"
- Add disk serial number display from `hardware_details.storage[].serial` (if field exists in data; show "—" if not)
- Add battery section: show charge %, health %, charging status with a simple progress bar visual. Data from `hardware_details.battery`. If no battery data (desktop), hide section entirely.

#### B6. Patches tab — confusing summary cards

**Current:** `PatchesTab.tsx` (line ~105-119) computes counts (critical, high, medium, low, installed, failed, pending) and displays them.

**Fix:** Simplify top summary to 3 clear stat cards:
1. **Pending** — total count, with severity breakdown underneath (critical/high/medium/low)
2. **Installed** — total count
3. **Failed** — total count
Remove any deployment-related clutter from the summary area.

#### B7. CVE Exposure — confusing summary cards

**Current:** `VulnerabilitiesTab.tsx` (line ~108-149) shows total CVE count + critical count in a hero strip.

**Fix:** Expand to 4 clear stat cards in a row: Total, Critical, High, Medium+Low. Each with count and color-coded badge. Add average CVSS score as a secondary metric on the Total card.

#### B8. Compliance tab — evaluate button, last scan, drill-down, graph

**Current:** `ComplianceTab.tsx`:
- "Evaluate Now" button (line ~240-248) is outline/sm variant
- Framework ring gauge shows percentage
- Link to /compliance/{frameworkId} exists
- No last evaluation timestamp shown

**Fix:**
- Restyle "Evaluate Now" — use a more prominent button style (e.g., secondary variant, proper size)
- Show "Last evaluated: {timestamp}" next to or below the button
- Ensure policy drill-down works (expandable rows or click-through to framework detail)
- Verify graph shows percentage (already does via ring gauge — ensure label says "X%")

#### B9. Deployments tab — IDs, timeline dates, policy wording

**Current:** `HistoryTab.tsx`:
- `shortId()` (line ~91-93) returns first 8 chars of UUID/ULID
- `formatDateTime()` (line ~66-79) returns full timestamp
- Timeline shows both formatted date and relative time

**Fix:**
- Replace shortId with human-readable format: "D-{sequential_number}" or "D-{shortId}" padded (e.g., D-001, D-002). If backend doesn't provide sequence, use `D-${id.slice(0,6).toUpperCase()}`.
- Ensure timeline cards show both absolute date+time AND relative time (e.g., "Apr 5, 2026 14:30 (2h ago)")
- For Windows policy entries, show "Last run: {timestamp}" instead of vague wording

#### B10. Audit tab — system ID readability, day range selector

**Current:** `AuditTab.tsx`:
- Event IDs are full ULIDs displayed as-is
- Category filter exists (8 categories)
- No date range selector (fetches last 50 events)

**Fix:**
- Truncate system/event IDs to first 8 chars with monospace font + click-to-copy full ID (tooltip: "Click to copy")
- Add "Last N days" dropdown: 1, 3, 7, 14, 30 days. Default: 7 days. Pass as query parameter to audit API.

#### B11. Status logic — misleading online state

**Current:** Status determined by `endpoint.status` field from backend. If agent reports online but hasn't sent heartbeat recently, status may be stale.

**Fix:** Frontend-only — derive display status from `last_seen` timestamp:
- If `last_seen` < 5 minutes ago → online
- If `last_seen` < 30 minutes ago → stale  
- Else → offline
- Override backend status with this derived status for display purposes.
- Apply in both EndpointsPage.tsx (list) and EndpointDetailPage.tsx (detail).

---

### C. Dashboard

#### C1. Fleet card — percentage sign and decimal digits

**Current:** `StatCardsRow1.tsx` uses `Math.round()` for percentages. `GaugeChart.tsx` clamps and rounds to integer. Both show `{value}%`.

**Fix:** Audit all percentage displays in dashboard stat cards:
- Ensure `%` appears exactly once (not doubled)
- Cap decimals at 2 max: use `value.toFixed(value % 1 === 0 ? 0 : 2)` pattern
- Check GaugeChart, StatCardsRow1, and any fleet-related widget

---

## Out of Scope

- Full endpoint page redesign
- Table structure replacement
- Backend schema changes (except if needed for B11 status)
- Peripheral hardware section (roadmap)
- New components beyond minimal additions for these fixes

## Files to Touch

| File | Issues |
|------|--------|
| `web/src/pages/endpoints/EndpointsPage.tsx` | A1, A2, B11 |
| `web/src/pages/endpoints/EndpointDetailPage.tsx` | B4, B11 |
| `web/src/pages/endpoints/CreateTagDialog.tsx` | A3 |
| `web/src/pages/endpoints/AssignTagsDialog.tsx` | A3 |
| `web/src/pages/endpoints/tabs/OverviewTab.tsx` | B2, B3 |
| `web/src/pages/endpoints/tabs/SoftwareTab.tsx` | B1 |
| `web/src/pages/endpoints/tabs/HardwareTab.tsx` | B5 |
| `web/src/pages/endpoints/tabs/PatchesTab.tsx` | B6 |
| `web/src/pages/endpoints/tabs/VulnerabilitiesTab.tsx` | B7 |
| `web/src/pages/endpoints/tabs/ComplianceTab.tsx` | B8 |
| `web/src/pages/endpoints/tabs/HistoryTab.tsx` | B9 |
| `web/src/pages/endpoints/tabs/AuditTab.tsx` | B10 |
| `web/src/pages/dashboard/StatCardsRow1.tsx` | C1 |
| `web/src/pages/dashboard/GaugeChart.tsx` | C1 |
