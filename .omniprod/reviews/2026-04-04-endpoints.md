# Review: Endpoints

- **URL:** http://localhost:3001/endpoints
- **Date:** 2026-04-04
- **Reviewer:** product-observer (automated)
- **Verdict:** FAIL

## Detection Summary

- **Assertions:** 16 run, 3 failed (a11y: search input label, h1 heading, 17 buttons without names)
- **Perspectives:** 5 dispatched (UX Designer, QA Engineer, Enterprise Buyer, Product Manager, End User)
- **Total raw findings:** 131 across all perspectives
- **After deduplication:** 47 unique findings
- **Cross-perspective hits:** 15 findings flagged by 3+ perspectives

---

## Perspective Verdicts

| Perspective | Verdict | Critical | Major | Minor | Nitpick |
|-------------|---------|----------|-------|-------|---------|
| UX Designer | FAIL | 2 | 9 | 12 | 4 |
| QA Engineer | FAIL | 2 | 8 | 16 | 2 |
| Enterprise Buyer | FAIL | 2 | 8 | 13 | 2 |
| Product Manager | FAIL | 2 | 6 | 10 | 3 |
| End User | FAIL | 5 | 10 | 15 | 0 |
| **Aggregate** | **FAIL** | **4** | **18** | **20** | **5** |

---

## Findings

### Critical

| ID | Element | Observation | Suggestion | Flagged By |
|----|---------|-------------|------------|------------|
| PO-001 | Endpoint detail > Compliance tab | Compliance tab renders CVE Exposure content (severity ring gauges, CVE filters) instead of compliance framework data. An admin checking per-endpoint compliance posture sees vulnerability data. | Fix tab content mapping so Compliance tab renders framework evaluation results (CIS, PCI-DSS, HIPAA scores, control pass/fail). Check if ComplianceTab component incorrectly imports CVEExposureTab content. | UX, QA, EB, PM, EU (5/5) |
| PO-002 | Breadcrumb on detail page | Breadcrumb shows `Endpoints / 4bb20cfa-ad3e-4936-a551-d...` -- a raw truncated UUID instead of the endpoint hostname. Visible on every detail page load. | Replace UUID in breadcrumb with endpoint hostname. The data is already fetched -- pass hostname to `useBreadcrumb()` via route state or lookup. | UX, QA, EB, PM, EU (5/5) |
| PO-003 | Detail tabs (Patches, Deployments, Audit) | Multiple perspectives observed that clicking Patches, Deployments, and Audit tabs on the detail page navigated to the global listing pages instead of showing endpoint-scoped data. The `13-` series screenshots show global pages; `50-` series show correct inline content. This may be intermittent or route-dependent. | Verify tab click handlers do not trigger Link navigation. Ensure tab content stays within the detail page context. Test with multiple endpoints to confirm. | PM, EU, QA (3/5) |
| PO-004 | OS column in endpoints table | OS column appears blank/empty for most endpoints. DOM snapshot confirms empty `<span>` elements. Some endpoints show OS icons but no text labels. Icons alone are not accessible or searchable. | Display OS name as text alongside icon (e.g., "Ubuntu 22.04", "Windows Server 2022"). Show em-dash when OS data is unavailable. | EU, UX, EB, PM, QA (5/5) |

### Major

| ID | Element | Observation | Suggestion | Flagged By |
|----|---------|-------------|------------|------------|
| PO-005 | Stat cards (Total/Online/Offline/Patching) | Stat card counts are computed client-side from the current page of 15 endpoints, not the full fleet. Total=32 but Online(11)+Offline(4)+Patching(--)=15. Clicking "Online" filter shows Total=27, Online=15 -- numbers don't add up. | Fetch aggregate status counts from a dedicated `/endpoints/stats` API endpoint, or label cards to indicate page-level counts. | QA, EB, PM, EU (4/5) |
| PO-006 | Risk Score column in table | Risk scores display as bare numbers (0.0, 2.6, 0.5) without the "/10" denominator. New users can't tell the scale. Detail page correctly shows "0.0 / 10" but table omits it. | Update table cell renderer to show "X.X/10" format consistently per Data Integrity Override. | UX, QA, EB, EU (4/5) |
| PO-007 | Filter bar -- missing Stale status | No "Stale" stat card or filter chip despite stale being a canonical endpoint status (BR-026). Endpoints not seen for days cannot be filtered or counted, breaking daily fleet triage workflow. | Add "Stale" stat card and filter chip. Define threshold (e.g., not seen in 7+ days). | UX, QA, PM, EU (4/5) |
| PO-008 | List/Grid view toggle | View toggle uses `<div onClick>` instead of `<button>`. Not keyboard-focusable, no focus ring, no ARIA role, cannot be activated via Enter/Space. | Change to `<button type="button">` with `aria-label` and `aria-pressed` state. | UX, QA, EB (3/5) |
| PO-009 | Pagination footer | Shows "1-15 of 32" with only prev/next arrows. No per-page selector (25/50/100), no page numbers, no entity name. Signals small-dataset design. | Add page numbers, per-page selector, and format as "Showing 1-15 of 32 endpoints". | UX, QA, EB, PM, EU (5/5) |
| PO-010 | Empty state (search no results) | "No endpoints found." as plain text. No icon, no helpful description, no CTA button to clear search. | Use EmptyState component: search icon, headline, description ("Try adjusting your search or filters"), "Clear Search" button. | UX, QA, EB, PM, EU (5/5) |
| PO-011 | Detail page health strip | PATCH COVERAGE, COMPLIANCE, and LAST SCAN all show em-dashes for an enrolled, online endpoint. Three out of four metrics blank gives "product is broken" impression. | Show "Not assessed" with "Run scan" CTA, or "Pending scan" with appropriate styling. Show "Never" for Last Scan if no scan has been run. | UX, EB, PM, EU (4/5) |
| PO-012 | Risk score color thresholds | Detail page HealthStrip uses `riskScore >= 3` for warning (amber), but listing page and BR-003 specify `>= 4`. An endpoint with score 3.5 shows green in list but amber in detail. | Change detail page threshold from `>= 3` to `>= 4` to match BR-003 and listing page. | QA (code-verified) |
| PO-013 | Status label: Pending vs Patching | Listing page correctly maps `pending` to "Patching" (BR-010), but detail page shows "Pending". Same status, different labels across pages. | Change detail page `STATUS_COLORS.pending.label` from 'Pending' to 'Patching'. | QA (code-verified) |
| PO-014 | Status dot colors inconsistency | Pending: listing uses `var(--accent)`, detail uses `var(--signal-warning)`. Stale: listing uses `var(--signal-warning)`, detail uses `var(--text-faint)`. Same statuses, different colors. | Align both pages: pending=`var(--accent)`, stale=`var(--text-muted)` per BR-026. | QA (code-verified) |
| PO-015 | Patching stat card value | Patching stat card shows "--" (em-dash) instead of "0" when no endpoints are patching. Em-dash means "unknown" per standards; zero is a known quantity. | Show `0` when count is zero. Reserve em-dash for genuinely unknown/unmeasured values. | UX, EB, PM, EU (4/5) |
| PO-016 | Card/grid view data density | Card view shows basic info but lacks last-seen time, tags are hard to read, risk score bar is very small, and there's no kebab menu for quick actions. | Add kebab menu to cards, make risk score numeric value larger, ensure last-seen and tags are visible. | EB, PM, EU (3/5) |
| PO-017 | Export Endpoints dialog | Minimal dialog: CSV only, no column selection, no format choice (CSV/XLSX), no option to export filtered vs all, no filename preview. | Add column selection, CSV/XLSX format toggle, filtered/all scope option, filename preview with date. | UX, EB, PM, EU (4/5) |
| PO-018 | Expanded row layout | Content is sparse and does not follow the specified 3-column grid (System Health, Pending Patches, Actions). Section labels not visibly following 10px monospace uppercase pattern. | Structure as 3-column grid with labeled sections per consistency override. Add severity breakdown for pending patches. | UX, EB, PM (3/5) |
| PO-019 | Risk score color coding in table | Risk scores in the table column all appear in neutral text color regardless of value. No red/amber/green color coding per BR-003 thresholds. | Apply signal colors: >=7 red, >=4 amber, <4 green to risk score text in the table. | EU (1/5, but standards violation) |
| PO-020 | Audit tab raw UUIDs | Audit events show "System heartbeat from eb08cfcc-ac06-..." exposing raw UUIDs. Should reference endpoint by hostname. | Replace raw UUIDs in audit messages with human-readable names. | UX, EB (2/5) |
| PO-021 | Bulk actions not visible | No visible bulk action workflow in any screenshot. Checkbox column exists but no bulk action toolbar appears on selection. Critical for enterprise fleet management. | Implement full bulk action flow: Select mode, bulk toolbar (Deploy, Scan, Tag, Export), selected count. | PM, EU (2/5) |
| PO-022 | Search input accessibility | Search input has no `<label>`, no `aria-label`, no `aria-labelledby`. Screen readers cannot identify its purpose. Confirmed by assertion BUILTIN-A11Y-002. | Add `aria-label="Search endpoints"` to the search input. | Assertion + QA |

### Minor

| ID | Element | Observation | Suggestion | Flagged By |
|----|---------|-------------|------------|------------|
| PO-023 | Checkbox column visibility | Checkbox column header visible by default. Per consistency override, checkboxes should be hidden until selection mode is activated. | Hide checkbox column by default. Add "Select" button to filter bar. | UX, QA, EB, PM, EU (5/5) |
| PO-024 | Tags column empty | Tags column blank for most endpoints. Empty cells should show em-dash per standards, not blank space. | Show em-dash or subtle "Add tag" affordance for endpoints with no tags. | UX, EB, PM, EU (4/5) |
| PO-025 | Page title not h1 | Page title "Endpoints" rendered as `<span>` not `<h1>`. Screen readers rely on heading hierarchy. | Change to `<h1>` with appropriate styling. | Assertion + UX, EB |
| PO-026 | 17 buttons without accessible names | Kebab triggers, stat card buttons, view toggle all lack aria-labels. | Add `aria-label` to all icon-only buttons with contextual descriptions. | Assertion + QA, EB |
| PO-027 | Kebab menu onBlur 150ms | Row kebab closes too fast for motor-impaired users. Custom implementation vs detail page uses `DropdownMenu`. | Use `DropdownMenu` from @patchiq/ui for all kebab menus, or increase timeout to 300ms. | UX, QA (2/5) |
| PO-028 | Tags column not sortable | Tags column header has no sort icon. All other data columns are sortable per consistency override. | Add sort functionality (by tag count or first tag alphabetically). | QA, PM, EU (3/5) |
| PO-029 | Register dialog step labels | Wizard steps show icons/numbers but no text labels. Users can't anticipate workflow. | Add text labels under each step: "Platform", "Configuration", "Download", "Verify". | UX, EB (2/5) |
| PO-030 | Agent version "vDEV" | Detail page meta chips show "vDEV" -- looks like debug artifact in a POC demo. | Display proper semver version (e.g., "v1.0.0"). Fix agent to report proper version. | EB (1/5) |
| PO-031 | Last Seen format inconsistency | "1 hr ago" vs "17 days ago" -- "hr" abbreviated but "days" not. | Standardize: either full words or consistent abbreviations. | EB (1/5) |
| PO-032 | Overview tab all zeros no guidance | All metrics zero with no scan-prompt banner. No indication of what action to take. | Add prominent "Run a scan to populate data" banner with CTA button. | PM, EU (2/5) |
| PO-033 | Hardware tab "OK" gauges | CPU/Memory gauges show "OK" text instead of actual utilization percentages. Not useful for IT admin. | Show actual percentages with "OK" as secondary label. Add disk usage, uptime. | UX, EB, EU (3/5) |
| PO-034 | CVE Exposure tab zero-state | Four ring gauges all at zero are visually heavy for "nothing here". | Hide gauges when all zero, show positive confirmation: "No known vulnerabilities" with green checkmark. | PM, EU (2/5) |
| PO-035 | Deploy wizard step labels small | Step indicator labels at top of wizard are barely visible. | Increase font size of wizard step labels. | EU (1/5) |
| PO-036 | Register dialog is centered modal | Multi-step registration form uses centered modal instead of right-side slide panel per consistency override. | Convert to right-side slide panel. | EU (1/5) |
| PO-037 | No data freshness indicator | No "Last updated" timestamp or manual refresh button on listing page. | Add "Last updated: X seconds ago" indicator near stat cards with refresh button. | EU (1/5) |
| PO-038 | Responsive columns drop not scroll | At 768px, table columns are hidden instead of enabling horizontal scroll per consistency override. | Enable horizontal scroll with sticky hostname column. | UX, EU (2/5) |
| PO-039 | Custom kebab vs DropdownMenu | Listing uses custom `KebabMenu` (manual state, onBlur), detail uses `@patchiq/ui DropdownMenu`. Different behavior. | Standardize to DropdownMenu from @patchiq/ui everywhere. | QA (1/5) |
| PO-040 | Pending column header ambiguous | Column says "PENDING" -- pending what? Patches? Deployments? | Rename to "PENDING PATCHES" or add tooltip. | EU (1/5) |
| PO-041 | Software tab empty state no CTA | Shows "No packages found. Run a scan..." but no "Scan Now" button in the empty state. | Add inline "Scan Now" CTA button below the empty state message. | PM (1/5) |
| PO-042 | Detail card boundaries low contrast | Overview tab dark cards on dark background with very subtle borders. Hard to distinguish sections. | Ensure card borders use `var(--border)` with sufficient contrast. | UX (1/5) |

### Nitpick

| ID | Element | Observation | Suggestion | Flagged By |
|----|---------|-------------|------------|------------|
| PO-043 | Stat card hover | No visible `translateY(-1px)` elevation shift on hover. | Verify hover includes border change + elevation with 150ms transition. | UX (1/5) |
| PO-044 | Kebab menu items | Text-only menu items. Adding 16px leading icons would improve scanability. | Add icons: eye (View), refresh (Scan), rocket (Deploy), tag (Tag), trash (Delete). | UX (1/5) |
| PO-045 | Client-side sorting only | Sorting reorders only current 15-item page, not full 32-item dataset. | Pass sort params to API for server-side sorting, or indicate "sorting current page only". | QA (1/5) |
| PO-046 | Stat card Total green border | "Total" stat card selected state uses green border (signal-healthy). Generic selection should use accent color. | Use `var(--accent)` for Total stat card selected border. | EB (1/5) |
| PO-047 | Deploy wizard CVE-based selection | Wizard shows CVE IDs for patch selection. Some admins think in patches, not CVEs. | Allow selection by both patches and CVEs. Show patch name/KB alongside CVE ID. | PM (1/5) |

---

## Business Rule Verification

| Rule | Status | Notes |
|------|--------|-------|
| BR-001 | PASS | Decommissioned endpoints excluded from default view |
| BR-002 | PASS | Risk score computed client-side from CVE severity counts |
| BR-003 | PARTIAL FAIL | Listing correct (>=7/>=4/<4), detail uses >=3 threshold (PO-012) |
| BR-008 | PASS | Cursor pagination with page size 15 |
| BR-010 | PARTIAL FAIL | Listing says "Patching", detail says "Pending" (PO-013) |
| BR-011 | NOT VERIFIED | Insufficient tags in test data to verify +N overflow |
| BR-019 | PASS | 6 columns sortable with three-state toggle |
| BR-020 | PASS | View mode persisted via `?view=card` URL param |
| BR-021 | PASS (by spec) | Counts from current page as designed, but UX is misleading (PO-005) |
| BR-022 | NOT VERIFIED | Bulk bar not observed in screenshots |
| BR-024 | PARTIAL FAIL | Health strip present but 3/4 metrics show dashes (PO-011) |
| BR-025 | PASS | All 8 tabs present and named correctly |
| BR-026 | PARTIAL FAIL | Status dot colors inconsistent between pages (PO-014) |
| BR-027 | PASS | Expanded row shows System Health, Pending Patches, Actions |

---

## Dev Checklist

```
Critical (must fix before ship)
- [ ] [PO-001] Fix Compliance tab to render framework data, not CVE Exposure content
- [ ] [PO-002] Replace UUID in breadcrumb with endpoint hostname
- [ ] [PO-003] Verify and fix detail tab routing (Patches/Deployments/Audit may navigate to global pages)
- [ ] [PO-004] Display OS name as text in table column (not icon-only)

Major (must fix before merge)
- [ ] [PO-005] Fix stat card counts to show fleet-wide totals, not page-level counts
- [ ] [PO-006] Display risk scores as "X.X/10" format in table
- [ ] [PO-007] Add "Stale" stat card and filter chip
- [ ] [PO-008] Change view toggle from div to button with aria-label
- [ ] [PO-009] Add per-page selector and page numbers to pagination
- [ ] [PO-010] Implement proper EmptyState component for no-results
- [ ] [PO-011] Show contextual empty states for health strip metrics (not bare dashes)
- [ ] [PO-012] Fix detail page risk threshold: change >= 3 to >= 4 (BR-003)
- [ ] [PO-013] Fix detail page status label: 'Pending' -> 'Patching' (BR-010)
- [ ] [PO-014] Align status dot colors between listing and detail pages (BR-026)
- [ ] [PO-015] Show "0" instead of "--" for Patching stat card when count is zero
- [ ] [PO-016] Add kebab menu and improve data density in card view
- [ ] [PO-017] Add format/column selection to Export dialog
- [ ] [PO-018] Restructure expanded row as 3-column grid with section labels
- [ ] [PO-019] Add color coding to risk scores in table (red/amber/green)
- [ ] [PO-020] Replace raw UUIDs in audit event descriptions with hostnames
- [ ] [PO-021] Implement bulk action toolbar (Deploy, Scan, Tag, Export)
- [ ] [PO-022] Add aria-label="Search endpoints" to search input

Minor (fix in this sprint)
- [ ] [PO-023] Hide checkbox column by default, add Select mode toggle
- [ ] [PO-024] Show em-dash for empty Tags cells
- [ ] [PO-025] Change page title to h1 element
- [ ] [PO-026] Add aria-label to all 17 icon-only buttons
- [ ] [PO-027] Replace custom KebabMenu with DropdownMenu from @patchiq/ui
- [ ] [PO-028] Add sort functionality to Tags column
- [ ] [PO-029] Add text labels to Register Endpoint wizard steps
- [ ] [PO-030] Replace "vDEV" agent version with proper semver
- [ ] [PO-031] Standardize relative time format (consistent abbreviations)
- [ ] [PO-032] Add "Run scan" banner on Overview tab when data is empty
- [ ] [PO-033] Show actual CPU/Memory percentages in Hardware tab gauges
- [ ] [PO-034] Simplify CVE Exposure zero-state (hide empty gauges)
- [ ] [PO-035] Increase deploy wizard step label font size
- [ ] [PO-036] Convert Register Endpoint dialog to slide panel
- [ ] [PO-037] Add "Last updated" freshness indicator to listing page
- [ ] [PO-038] Enable horizontal table scroll at narrow viewports (not column hiding)
- [ ] [PO-039] Standardize kebab implementation (use DropdownMenu everywhere)
- [ ] [PO-040] Rename "PENDING" column to "PENDING PATCHES"
- [ ] [PO-041] Add "Scan Now" CTA button to Software tab empty state
- [ ] [PO-042] Improve card border contrast on Overview tab

Nitpick (fix when convenient)
- [ ] [PO-043] Verify stat card hover elevation effect
- [ ] [PO-044] Add leading icons to kebab menu items
- [ ] [PO-045] Pass sort params to API for server-side sorting
- [ ] [PO-046] Use accent color (not green) for Total stat card selected border
- [ ] [PO-047] Allow patch-based selection in deploy wizard alongside CVEs
```

---

## Lighthouse Summary

| Category | Score | Status |
|----------|-------|--------|
| Accessibility | 86 | Warning (50-89) |
| Best Practices | 100 | Pass (>=90) |
| SEO | 60 | Warning (50-89) |

Key Lighthouse flags:
1. Accessibility 86: Missing form labels, buttons without accessible names, heading hierarchy
2. SEO 60: Missing meta description, no canonical URL, viewport not optimized
3. Best Practices 100: No issues detected

---

## Console & Network Issues

### Console Errors / Warnings

| Level | Count | Top Messages |
|-------|-------|--------------|
| Error | 0 | None |
| Warning | 0 | None (only Vite dev messages and React DevTools suggestion) |

### Network Issues

| Type | Count | Details |
|------|-------|---------|
| 4xx errors | 0 | None |
| 5xx errors | 0 | None |
| Slow requests (>2s) | 0 | All API responses within normal range |

---

## Comparison to Previous Review

```
Previous review date: N/A -- first review
Previous verdict:     N/A

First review -- no prior baseline.
```

---

## Capture Stats

- Screenshots taken: 55
- Coverage targets completed: 13 / 13 (100%)
- Assertions run: 16, passed: 13, failed: 3
- Perspectives dispatched: 5 (all opus model)
- Total perspective findings (pre-dedup): 131
- Unique findings (post-dedup): 47

---

## Machine-Readable Output

A JSON version of all findings has been saved to:

```
.omniprod/findings/2026-04-04-endpoints.json
```
