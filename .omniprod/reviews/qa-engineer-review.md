# QA Engineer Review — Post-Overhaul Smoke Test

**Date**: 2026-04-03  
**Reviewer**: QA Engineer (OmniProd perspective)  
**Scope**: Alerts, Patches, CVEs, Deployments, Audit, Settings/Tags, Settings/Roles, Settings/User Roles, and redirect URLs  
**Branch**: dev-b  
**Build context**: Post-consistency overhaul — stat cards, filter bars, grid/card view toggles, sorting, settings sidebar consolidation

---

## Verdict: FAIL

**Reason**: Data count inconsistency on the Alerts page (sidebar badge vs stat cards), inaccessible view-toggle controls (div-based click handlers instead of buttons) across all list pages, and missing result count in card view on Alerts. These are functional and data integrity failures that block POC readiness.

---

## Findings Table

| ID | Severity | Page | Element | Observation | Suggestion |
|----|----------|------|---------|-------------|------------|
| QA-01 | **major** | Alerts | Stat cards vs sidebar badge | Sidebar shows "80 unread alerts", stat cards show 82 TOTAL (49 CRITICAL + 31 WARNING + 2 INFO = 82). These two counts disagree. The sidebar reads `total_unread` from the count API; the stat card total is recomputed client-side as `critical + warning + info`. Either the server count endpoint is wrong or the client-side summation double-counts a category. | Align both to the same API field. Don't compute totals client-side if the API already returns a total. |
| QA-02 | **major** | Alerts, Patches, Deployments | View toggle (`<div onClick>`) | The list/card toggle is implemented as a `<div onClick>` with no `role="button"`, no `tabIndex`, and no keyboard event handler. The element is invisible to screen readers and non-keyboard-accessible. The accessibility tree does not expose it as an interactive element. | Replace with `<button type="button">` elements (or at minimum add `role="button"` + `tabIndex={0}` + onKeyDown for Enter/Space). Verified in: `AlertsPage.tsx:1393–1421`, `PatchesPage.tsx:1374–1400`, `DeploymentsPage.tsx:1088–1098`. |
| QA-03 | **major** | Alerts (card view) | Missing result count | In list view, "33 alerts" is shown above the table. In card view, this count disappears entirely. A user switching to card view loses all visibility into how many results are shown and whether filtering is active. | Render the result count ("33 alerts") above the card grid in card view, matching the list view behavior. |
| QA-04 | **major** | CVEs | Missing TOTAL stat card | Every other list page (Alerts, Patches, Deployments, Audit, Tags) has a TOTAL stat card as the first and most prominent card. The CVEs page only shows CRITICAL (6), HIGH (14), MEDIUM (28), KEV LISTED (11) — no TOTAL. The total is visible only in the filter bar as "All 62". | Add a TOTAL stat card (value: 62) as the first card on the CVEs page, consistent with all other list pages. |
| QA-05 | **major** | CVEs | Attack vector display inconsistency: "Unknown" vs "—" | In the list/table view, CVEs with no attack vector data show "—" (em dash). In the card view, the same CVEs show the text "Unknown". Two representations of the same null/missing value in two views of the same page. | Pick one: either show "—" in both views (preferred — matches the data display standard) or "Unknown" in both. Do not mix. |
| QA-06 | **minor** | Patches | Unformatted numbers in filter buttons and pagination | The stat cards correctly format numbers: "207,222 TOTAL", "30,706 CRITICAL", etc. However, the filter buttons directly below show: "All 207222", "Critical 30706", "High 79657" — no thousand separators. The pagination footer also reads "Showing 1–15 of 207222 patches" without formatting. | Apply `toLocaleString()` or `Intl.NumberFormat` to all numeric counts rendered in filter pill buttons and pagination strings. |
| QA-07 | **minor** | Deployments | Missing stat cards for Created, Scheduled, Cancelled states | The stat card row shows 4 cards: TOTAL, RUNNING, COMPLETED, FAILED. The filter tabs below expose 3 additional states: Created, Scheduled, Cancelled. A "Cancelled" filter tab with no corresponding stat card creates an asymmetry — the user cannot quickly see how many deployments are in those states without clicking the tab. | Either add stat cards for Created, Scheduled, and Cancelled, or remove the states that don't have stat cards. Both options are valid — pick one and be consistent. |
| QA-08 | **minor** | Deployments | "Rolled Back" progress bar shows 100% | In the card view, the "Database Maintenance Window" deployment has status "Rolled Back" but Progress shows "100%". A rollback is not a completion event — displaying 100% progress for a rolled-back deployment is misleading. The TARGETS row correctly shows SUCCEEDED: 1 / FAILED: 2 which contradicts 100%. | For "Rolled Back" deployments, show the actual wave-level progress, not the total endpoint count. Or cap at the percentage of endpoints that reached any state before rollback was triggered. |
| QA-09 | **minor** | Roles | Missing stat cards and filter bar | Every other settings admin page (Tags) and every main list page has stat cards. The Roles page has only a search box and a table — no stat cards (e.g., "4 TOTAL", "4 SYSTEM", "0 CUSTOM") and no filter bar. The consistency overhaul spec says "all list pages now have stat cards". | Add stat cards: TOTAL, SYSTEM (built-in), CUSTOM (user-created). Add filter buttons for System/Custom. |
| QA-10 | **minor** | Audit | No sort indicator on column headers (visual) | Sorting IS implemented in the ActivityStream component (SortHeader components with up/down arrows). However, the DOM accessibility tree shows them as `StaticText` rather than interactive elements. The sort arrows are rendered in SVG within a `<th>` element, but the `<th>` has no `role`, `aria-sort`, or `tabIndex`. A keyboard-only user cannot sort the audit table. | Add `aria-sort="ascending|descending|none"` to each sortable `<th>` and ensure they are reachable via Tab and activatable via Enter. |
| QA-11 | **minor** | Audit | No "Showing X of Y" pagination count | Every other paginated table shows "Showing 1–25 of 62 CVEs" or "Showing 1–15 of 207222 patches". The audit log shows only "Previous page" / "Next page" buttons with no indication of total count or current position. | Add a pagination count string: "Showing 1–50 of 50 events" or "Showing page 1 of 3". |
| QA-12 | **minor** | Audit | Date format inconsistency (section headers) | Audit uses date section headers formatted as "APRIL 3, 2026" (all-caps month, full year). Other pages use "Mar 15, 2026" (abbreviated month, mixed case). These are the same data type (date) rendered with different formatting conventions in different pages. | Normalize to "April 3, 2026" or "Apr 3, 2026" — whichever matches the app-wide date format setting. Do not use ALL_CAPS for month names. |
| QA-13 | **nitpick** | Alerts | Card view is a vertical list, not a grid | The "card view" on Alerts renders cards in a single vertical column — the same layout as the list view, just with more card anatomy visible per row (description, action buttons). It is not a true card grid (2–3 columns). The CVEs and Deployments card views use a multi-column grid layout. | Either rename the toggle to "Expanded View" to set accurate expectations, or reflow the alerts card view into a 2-column grid to match the grid-style card views on other pages. |
| QA-14 | **nitpick** | Patches, CVEs | "—" in CATEGORY column for all patches | All 15 visible patches show "—" in the CATEGORY column. Given that 207,222 patches exist, at least some should have a category. This may be a seeding/data issue rather than a UI bug, but it makes the column appear broken during a demo. | Verify that patch category data is being populated from the catalog sync. If categories are intentionally empty for this data set, consider hiding the CATEGORY column until data exists. |
| QA-15 | **nitpick** | All pages | View toggle buttons have no accessible label | The list/card toggle buttons are icon-only (SVG lines icon for list, SVG grid icon for card) with no `aria-label`, no title, and no visible text. They are also missing a tooltip. A new user or screen reader user cannot determine what these buttons do. | Add `aria-label="List view"` / `aria-label="Card view"` (or a tooltip on hover) to both toggle buttons across all pages. |

---

## Redirect Test Results

| URL | Expected Destination | Actual Destination | Result |
|-----|---------------------|-------------------|--------|
| `/tags` | `/settings/tags` | `/settings/tags` | PASS |
| `/admin/roles` | `/settings/roles` | `/settings/roles` | PASS |
| `/admin/users/roles` | `/settings/user-roles` | `/settings/user-roles` | PASS |

All three redirects work correctly with client-side `<Navigate replace>`. After redirect, the final URL in the browser address bar is the canonical settings URL.

---

## Feature Checklist Results

| Feature | Page | Result | Notes |
|---------|------|--------|-------|
| Stat cards visible | Alerts | PASS | 4 cards: TOTAL, CRITICAL, WARNING, INFO |
| Stat cards visible | Patches | PASS | 5 cards: TOTAL, CRITICAL, HIGH, MEDIUM, LOW |
| Stat cards visible | CVEs | PARTIAL | Missing TOTAL card (see QA-04) |
| Stat cards visible | Deployments | PARTIAL | Missing Created/Scheduled/Cancelled (see QA-07) |
| Stat cards visible | Audit | PASS | 4 cards: TOTAL, SYSTEM, USER, TODAY |
| Stat cards visible | Roles | FAIL | No stat cards present (see QA-09) |
| Filter bar present | Alerts | PASS | Status tabs + category tabs + date range + refresh interval |
| Filter bar present | Patches | PASS | Severity filters + OS family + status dropdowns |
| Filter bar present | CVEs | PASS | Severity filters + attack vector + date range + remediation |
| Filter bar present | Deployments | PASS | Status tabs + date range filter |
| Table sorting implemented | Alerts | PASS | SortHeader on all 6 columns (code-verified) |
| Table sorting implemented | Patches | PASS | SortHeader on all columns (code-verified) |
| Table sorting implemented | CVEs | PASS | SortHeader on all columns (code-verified) |
| Table sorting implemented | Deployments | PASS | SortHeader on all columns (code-verified) |
| Table sorting implemented | Audit | PASS | SortHeader on all 5 columns (code-verified) |
| Grid/card view toggle | Alerts | PASS* | Toggle works but div-based (QA-02), card view is vertical list (QA-13) |
| Grid/card view toggle | Patches | PASS* | Toggle works but div-based (QA-02) |
| Grid/card view toggle | CVEs | PASS | Labeled buttons ("List view" / "Card view"), proper `<button>` elements |
| Grid/card view toggle | Deployments | PASS* | Toggle works but div-based (QA-02) |
| CVE card CVSS gauge | CVEs | PASS | SVG ring gauge rendered per card (code-verified in `CVEsPage.tsx:784`) |
| Expanded row detail | Patches | PASS | Expanded row shows CVE links, endpoint exposure, description, Deploy button |
| Progress bars in deployment cards | Deployments | PASS | Shows percentage + TARGETS/SUCCEEDED/FAILED breakdown |
| Settings sidebar has Tags | Settings | PASS | Tags link at `/settings/tags` under ADMINISTRATION section |
| Settings sidebar has Roles | Settings | PASS | Roles link at `/settings/roles` under ADMINISTRATION section |
| Settings sidebar has User Roles | Settings | PASS | User Roles link at `/settings/user-roles` under ADMINISTRATION section |
| No console errors | All pages | PASS | Zero errors or warnings in browser console on all visited pages |

---

## Summary by Priority

**Fix immediately (blockers for POC demo):**
- QA-01: Alert count inconsistency (sidebar 80 ≠ stat cards 82) — will confuse any demo reviewer
- QA-04: CVEs page missing TOTAL stat card — violates the consistency pattern the overhaul established
- QA-05: Attack vector shows "Unknown" in card view vs "—" in list view — same data, two representations

**Fix before next review:**
- QA-02: View toggle div-based click handlers — accessibility debt, also hides the buttons from the DOM accessibility tree
- QA-03: Card view missing result count on Alerts
- QA-06: Unformatted numbers in patch filter buttons
- QA-07: Deployments missing stat cards for Created/Scheduled/Cancelled
- QA-08: Rolled Back deployments showing 100% progress
- QA-09: Roles page missing stat cards
- QA-10: Audit column headers not keyboard-accessible
- QA-11: Audit missing pagination count

**Polish pass:**
- QA-12: Audit date format (APRIL 3 vs Apr 3)
- QA-13: Alerts card view is vertical list not grid
- QA-14: Patch CATEGORY column all "—" (may be data issue)
- QA-15: View toggle buttons missing accessible labels/tooltips
