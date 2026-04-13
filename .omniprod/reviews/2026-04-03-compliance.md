# Review: Compliance

- **URL:** http://localhost:3001/compliance
- **Date:** 2026-04-03
- **Reviewer:** product-observer (automated)
- **Verdict:** FAIL

---

## Perspective Verdicts

| Perspective | Verdict | Critical | Major | Minor | Nitpick |
|-------------|---------|----------|-------|-------|---------|
| UX Designer | FAIL | 2 | 8 | 5 | 3 |
| Product Manager | FAIL | 2 | 5 | 4 | 4 |
| QA Engineer | FAIL | 2 | 7 | 5 | 2 |
| Enterprise Buyer | FAIL | 2 | 5 | 4 | 2 |
| End User | FAIL | 3 | 7 | 7 | 1 |
| Accessibility Expert | FAIL | 3 | 9 | 4 | 1 |
| Sales Engineer | FAIL | 2 | 6 | 5 | 0 |
| CTO/Architect | FAIL | 2 | 5 | 4 | 1 |
| **Total (deduplicated)** | **FAIL** | **5** | **13** | **9** | **4** |

---

## Findings

### Critical

| ID | Element | Observation | Suggestion | Flagged By |
|----|---------|-------------|------------|------------|
| PO-001 | Overdue Controls table, FRAMEWORK column, row 2 | Raw UUID `bbb41940-3cd2-4dc3-8805-f680f0f85422` rendered as the framework name instead of human-readable "test-001". Disqualifying for any client-facing demo. | Fix the data join in the overdue controls query to always resolve framework UUID to display name. Show "Unknown Framework" if lookup fails, never the UUID. | UX, PM, QA, EB, EU, A11Y, SE, CTO |
| PO-002 | Overdue Controls table, CONTROL NAME column, row 2 | CONTROL NAME cell shows "SW-001" — identical to the CONTROL ID cell. The name column is useless — it echoes the ID instead of providing a descriptive label. | Populate CONTROL NAME from the control definition (e.g., "Software Inventory Management"). Show em dash "—" if name is unavailable. | UX, PM, QA, EB, EU, SE, CTO |
| PO-003 | Overdue Controls table rows reference inactive frameworks | Table shows overdue controls from ISO 27001, HIPAA, and SOC 2 — but only 2 frameworks are active (Internal Security Standard and test-001). Inactive framework controls should not appear without explanation. | Scope overdue controls to active frameworks only, or add a filter control and label "Includes inactive frameworks". | EU |
| PO-004 | Compliance Trend chart — inaccessible to screen readers | Chart is exposed as a single opaque `application` role with 12 unlabeled groups. No data table alternative exists. Screen reader users cannot extract any trend data. | Add a visually-hidden data table below the chart, or use `role="img"` with a descriptive `aria-label` summarizing the trend data. | A11Y |
| PO-005 | Sidebar "Alerts 80" link — badge count concatenated into accessible name | Screen reader announces "Alerts 80 link" with no semantic separation — user cannot distinguish navigation label from notification count. | Use `aria-label="Alerts, 80 unread"` on the link. Mark the visible "80" badge with `aria-hidden="true"` and add a separate `<span>` with `aria-label="80 unread alerts"`. | A11Y |

### Major

| ID | Element | Observation | Suggestion | Flagged By |
|----|---------|-------------|------------|------------|
| PO-006 | Browser tab / favicon | 404 on `/favicon.ico`. Browser tab shows no product icon. Signals incomplete product in any demo context. | Add `favicon.ico` and `apple-touch-icon` to `web/public/`. 5-minute fix. | UX, PM, QA, EB, EU, SE, CTO |
| PO-007 | "Evaluate All", "Manage Frameworks", "Evaluate" buttons | All primary/secondary buttons show no perceptible hover state change. The delta between default and hovered is invisible at normal viewing distance. Systemic button component issue. | Add a 10-15% lightness shift on hover, or a subtle shadow lift. Apply to the button component globally, not per-instance. | UX, PM, QA, SE, CTO |
| PO-008 | "View Details" links on framework cards | No underline, no color shift on hover. Indistinguishable from static text except by cursor shape. | Add underline on hover and/or color shift. Links that navigate must look like links. | UX, SE |
| PO-009 | Chart title: "Compliance Trend — Last 90 Days" | X-axis spans only Mar 25 to Apr 3 (~10 days). Title claims 90 days. Misleading. Destroys metric credibility with any evaluator who reads the axis. | Dynamically set title to match actual data range, or render the full 90-day axis. | UX, PM, QA, EB, EU, SE, CTO |
| PO-010 | Framework name "test-001" | Test/seed artifact name appears throughout: framework card, chart legend, overdue table, manage frameworks panel. Unprofessional in a POC demo. | Rename seed data to realistic names (e.g., "Internal IT Policy"). | UX, PM, QA, EB, EU, SE, CTO |
| PO-011 | Overdue Controls table — no row actions | Table is read-only. No "View", "Remediate", or clickable row to navigate to affected controls or endpoints. Surfaces problems without any path to resolution. | Add row-level actions: at minimum "View Control" link to framework detail filtered to that control. | UX, PM, EU, SE, CTO |
| PO-012 | Overall Compliance ring gauge — green color with NON-COMPLIANT status | Ring gauge at 75% uses green/teal color, but status badge says "NON-COMPLIANT". Green signals passing — creates cognitive dissonance. Per-framework cards correctly use red for 50%, but summary doesn't follow same logic. | Apply semantic color: green only for COMPLIANT (95%+), amber for NEEDS WORK (80-94%), red for NON-COMPLIANT (<80%). | EU |
| PO-013 | Overall Compliance — no threshold context | Score shows 75% labeled "NON-COMPLIANT" with no explanation of what threshold defines compliance. Prospect will ask "why is 75% non-compliant?" | Add threshold indicator: "NON-COMPLIANT (threshold: 95%)" or a small marker on the ring gauge. | PM, EB, SE |
| PO-014 | "More actions" dropdown — single item | Dropdown contains only "Export Report". A dropdown for one item is an antipattern — adds a click and hides a useful action. | Promote "Export Report" to a standalone button. Remove dropdown unless 3+ items are imminent. | UX, PM, QA, EB, EU, SE, CTO |
| PO-015 | Overdue Controls count mismatch | Summary card shows "1 OVERDUE CONTROLS" but the table header shows "4" and lists 4 rows. The "1" counts frameworks with overdue controls; the "4" counts individual controls. Ambiguous without a label distinction. | Change summary card label to "1 FRAMEWORK WITH OVERDUE CONTROLS" or make the count match the table. | QA |
| PO-016 | "View Details" links — duplicate accessible names | Both framework cards have "View Details" with no `aria-label` differentiation. Screen reader link list shows two identical entries navigating to different pages. | Add `aria-label="View details for {framework name}"` to each link. | A11Y, PM |
| PO-017 | "Evaluate" buttons — duplicate accessible names | Both cards have "Evaluate" with no `aria-label` differentiation. Screen reader encounters two identical button labels. | Add `aria-label="Evaluate {framework name}"` to each button. | A11Y |
| PO-018 | Sidebar nav group labels — no semantic grouping | Labels "OVERVIEW", "ASSETS", "SECURITY" etc. are plain StaticText with no `role="group"` or `aria-labelledby`. Screen reader users hear a flat link list with no organizational structure. | Wrap each group in `<ul>` with `aria-label` matching the section name. | A11Y, UX |

### Minor

| ID | Element | Observation | Suggestion | Flagged By |
|----|---------|-------------|------------|------------|
| PO-019 | Sidebar hover state | Hover background change on non-active nav items is barely perceptible. Needs a more distinct intermediate state between resting and active. | Increase hover background contrast one step. | UX |
| PO-020 | Overall Compliance "NON-COMPLIANT" status | Rendered as plain text, not as a badge component. Individual framework cards use colored badge treatment for same status string. Inconsistent. | Apply same badge component to overall status as on per-framework cards. | UX |
| PO-021 | Responsive 768px | Chart and overdue table are below fold with no scroll affordance. Framework card CTAs may be cramped. | Verify chart renders at usable height; ensure table scrolls horizontally with visible affordance. | UX, PM, EB, SE, CTO |
| PO-022 | Overdue Controls section header "4" | Count badge lacks qualifying label. "4 what?" — controls, frameworks, endpoints? | Add text qualifier: "4 overdue controls" or match the pattern used elsewhere. | UX, QA |
| PO-023 | Framework Management panel — inconsistent section anatomy | Built-in frameworks use tag-chip layout; custom frameworks use status badges + edit/delete icons. Different row structures for conceptually the same entity type. | Unify card anatomy across both sections. Only difference should be available actions. | UX, EB, SE |
| PO-024 | Edit/delete icon buttons in drawer — no labels/tooltips | Icon-only action buttons with no visible tooltip on hover. Delete button has no red/danger hover state. | Add tooltips ("Edit framework", "Delete framework"). Delete button should show red on hover. | UX, EU, A11Y |
| PO-025 | "Evaluate" buttons — no loading/progress state | Clicking Evaluate triggers async compliance evaluation with no spinner, no disabled state, no "Evaluating..." feedback. Users will click repeatedly. | Button must enter disabled+spinner state on click until job response returns. | QA, EB, CTO |
| PO-026 | Chart Y-axis — no label | Y-axis shows percentage ticks but no axis label identifying what is measured. | Add rotated Y-axis label: "Compliance Score (%)". | UX, EB, SE, CTO |
| PO-027 | User avatar shows "U" / "User" | Placeholder display name "User" in sidebar footer. Auth endpoint is called but response not wired to display. | Resolve actual user name from auth session. Show email as fallback. | UX, EB, A11Y |

### Nitpick

| ID | Element | Observation | Suggestion | Flagged By |
|----|---------|-------------|------------|------------|
| PO-028 | Table column headers ALL-CAPS | Overdue Controls uses SCREAMING_CASE headers. Must be consistent with all other tables in the app. If CSS text-transform is not used, screen readers may spell out letters. | Use Title Case in HTML with `text-transform: uppercase` in CSS. | UX, PM, QA, EB, EU, A11Y |
| PO-029 | "AI Assistant (coming soon)" button in topbar | Coming-soon feature button visible in POC demo. Occupies prime real estate and may confuse buyers. | Remove for POC build or make visually distinct as unavailable (ghost + lock icon). | PM |
| PO-030 | Status badges ALL-CAPS consistency | COMPLIANT/NON-COMPLIANT/FAIL all use all-caps. Must verify this casing matches every other status badge in the app. | Audit all status badge text casing across the product. | PM, QA |
| PO-031 | "View Details" URLs use raw UUIDs | Framework detail URLs contain UUIDs (e.g., `/compliance/frameworks/a66565d6...`). Functional but unfriendly for sharing. | Consider slug-based routing for framework detail pages. | CTO |

---

## Dev Checklist

Ordered by severity. Check off items as they are resolved.

```
Critical (must fix before ship)
- [ ] [PO-001] Fix overdue controls query to resolve framework UUID to display name
- [ ] [PO-002] Fix overdue controls query to resolve control name (not echo control ID)
- [ ] [PO-003] Scope overdue controls to active frameworks or add filter/label for inactive
- [ ] [PO-004] Add accessible data table alternative for Compliance Trend chart
- [ ] [PO-005] Fix sidebar Alerts link aria-label to separate badge count from nav label

Major (must fix before merge)
- [ ] [PO-006] Add favicon.ico and apple-touch-icon to web/public/
- [ ] [PO-007] Fix button hover states globally — add 10-15% lightness shift
- [ ] [PO-008] Add hover underline/color-shift to "View Details" links
- [ ] [PO-009] Fix chart title to match actual data range (or populate 90 days of data)
- [ ] [PO-010] Rename seed framework "test-001" to a realistic name
- [ ] [PO-011] Add row-level actions to Overdue Controls table
- [ ] [PO-012] Fix overall compliance ring gauge color to match status semantic
- [ ] [PO-013] Add compliance threshold context to overall score display
- [ ] [PO-014] Promote "Export Report" to direct button, remove single-item dropdown
- [ ] [PO-015] Clarify overdue controls count label (1 framework vs 4 controls)
- [ ] [PO-016] Add aria-label with framework name to "View Details" links
- [ ] [PO-017] Add aria-label with framework name to "Evaluate" buttons
- [ ] [PO-018] Add semantic grouping (role="group" + aria-label) to sidebar nav sections

Minor (fix in this sprint)
- [ ] [PO-019] Increase sidebar nav item hover contrast
- [ ] [PO-020] Apply badge component to overall compliance status label
- [ ] [PO-021] Verify responsive layout at 768px — chart height, table scroll
- [ ] [PO-022] Add qualifying label to overdue controls count badge
- [ ] [PO-023] Unify framework management panel section anatomy
- [ ] [PO-024] Add tooltips to edit/delete icon buttons in framework drawer
- [ ] [PO-025] Add loading/spinner state to Evaluate buttons on click
- [ ] [PO-026] Add Y-axis label to compliance trend chart
- [ ] [PO-027] Wire auth user display name to sidebar avatar

Nitpick (fix when convenient)
- [ ] [PO-028] Use CSS text-transform for ALL-CAPS table headers (not HTML uppercase)
- [ ] [PO-029] Remove or style "AI Assistant (coming soon)" button for POC
- [ ] [PO-030] Audit status badge text casing consistency across all pages
- [ ] [PO-031] Consider slug-based routing for framework detail URLs
```

---

## Lighthouse Summary

| Category | Score | Status |
|----------|-------|--------|
| Accessibility | 94 | ✅ |
| Best Practices | 100 | ✅ |
| SEO | 60 | ⚠️ |

Key Lighthouse flags:
1. SEO score 60 — likely missing meta description, viewport issues, or crawlability signals
2. Accessibility at 94 — 6 points lost; manual review found significantly more issues than Lighthouse automated checks
3. Best Practices at 100 — no issues detected

---

## Console & Network Issues

### Console Errors / Warnings

| Level | Count | Top Messages |
|-------|-------|--------------|
| Error | 1 | `Failed to load resource: 404 (Not Found)` — favicon.ico |
| Warning | 0 | — |

### Network Issues

| Type | Count | Details |
|------|-------|---------|
| 4xx errors | 1 | `GET /favicon.ico` → 404 |
| 5xx errors | 0 | — |
| Slow requests (>2s) | 0 | — |

---

## Comparison to Previous Review

```
Previous review date: N/A — first review
Previous verdict:     N/A

First review — no prior baseline.
```

---

## Machine-Readable Output

A JSON version of all findings has been saved to:

```
.omniprod/findings/2026-04-03-compliance.json
```
