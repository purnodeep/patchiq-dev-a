# Review: Compliance (Re-review #2)

- **URL:** http://localhost:3001/compliance
- **Date:** 2026-04-03
- **Reviewer:** product-observer (automated, 8 perspectives)
- **Verdict:** FAIL
- **Screenshots:** 43 captured (42 minimum)

---

## Perspective Verdicts

| Perspective | Verdict | Critical | Major | Minor | Nitpick |
|-------------|---------|----------|-------|-------|---------|
| UX Designer | FAIL | 1 | 9 | 6 | 0 |
| Product Manager | FAIL | 5 | 7 | 5 | 3 |
| QA Engineer | FAIL | 2 | 8 | 6 | 3 |
| Enterprise Buyer | FAIL | 3 | 6 | 7 | 2 |
| End User | FAIL | 3 | 8 | 9 | 0 |
| Accessibility Expert | FAIL | 5 | 9 | 5 | 1 |
| Sales Engineer | FAIL | 3 | 7 | 8 | 1 |
| CTO Architect | FAIL | 4 | 5 | 7 | 2 |
| **Total (deduplicated)** | **FAIL** | **6** | **15** | **11** | **4** |

---

## Findings

### Critical

| ID | Element | Observation | Suggestion | Flagged By |
|----|---------|-------------|------------|------------|
| PO-001 | Overdue Controls table, FRAMEWORK column, row 2 | Raw UUID `bbb41940-3cd2-...` rendered as framework name instead of display name | Fix SQL join in overdue controls query to resolve framework UUID to name | All 8 perspectives |
| PO-002 | Overdue Controls table, CONTROL NAME column, row 2 | CONTROL NAME shows "SW-001" — identical to CONTROL ID. Name not resolved. | Populate control name from framework definition; show em dash if unavailable | 7 of 8 perspectives |
| PO-003 | Overdue Controls table — inactive framework rows | Table shows controls from ISO 27001, HIPAA, SOC 2 which are not among the 2 active frameworks | Scope overdue query to active frameworks only, or label inactive data clearly | UX, QA, EB, EU, CTO |
| PO-004 | Compliance Trend chart title | Chart title says "Last 90 Days" but X-axis spans only Mar 25–Apr 3 (~10 days) — factually wrong | Dynamically set title to match actual data range, or backfill 90 days of seed data | 7 of 8 perspectives |
| PO-005 | "test-001" framework name across all surfaces | Seed data artifact "test-001" visible on framework cards, chart legend, detail page H1, breadcrumb, overdue table | Rename seed framework to a realistic name (e.g., "Endpoint Hardening Standard") | All 8 perspectives |
| PO-006 | Focus indicators — entire page | Focus screenshots (focus-1/2/3.png) show zero visible focus rings on any element. Keyboard navigation is blind. WCAG 2.4.7 failure. | Apply `outline: 2px solid var(--accent)` on all `:focus-visible` states globally | UX, QA, A11Y |

### Major

| ID | Element | Observation | Suggestion | Flagged By |
|----|---------|-------------|------------|------------|
| PO-007 | Missing favicon | favicon.ico returns 404 (console + network confirm). Blank browser tab icon. | Add favicon.ico to `web/public/` | 6 perspectives |
| PO-008 | Overall Compliance ring gauge | Ring renders green/teal at 75% while status is "NON-COMPLIANT". Green signals "good" — contradicts the label. | Apply semantic color: green >=95%, amber 80-94%, red <80% | 5 perspectives |
| PO-009 | NON-COMPLIANT status label | No threshold indicator showing what score defines compliance. Users cannot interpret their score. | Add "Target: 95%" next to status badge or threshold marker on ring gauge | 5 perspectives |
| PO-010 | "More actions" dropdown | Contains exactly one item: "Export Report". Single-item dropdown is an antipattern. | Promote Export to standalone button; remove dropdown until 2+ actions exist | 5 perspectives |
| PO-011 | Overdue Controls table — no row actions | 4 overdue controls with zero remediation path. Users see the problem but cannot act on it. | Add "View" or "Remediate" action column linking to framework detail controls tab | 5 perspectives |
| PO-012 | "Evaluate All" button feedback | Post-click screenshot identical to pre-click. No toast, no loading state, no timestamp update. | Show loading state, success toast, update "Last evaluated" timestamp | 5 perspectives |
| PO-013 | User avatar "U" / "User" | Sidebar user area shows placeholder "U" / "User" instead of authenticated user name | Wire /api/v1/auth/me response to sidebar avatar component | 3 perspectives |
| PO-014 | Overdue Controls count mismatch | Summary shows "1 OVERDUE CONTROLS" but table has 4 rows. "1" = frameworks, "4" = controls — ambiguous. | Change label to "FRAMEWORKS WITH OVERDUE CONTROLS" or show consistent count | 4 perspectives |
| PO-015 | Chart Y-axis no label | Y-axis shows percentage ticks but no label ("Compliance Score (%)"). Ambiguous unit. | Add rotated Y-axis label; also starts at 40% not 0% without indication | 5 perspectives |
| PO-016 | SLA Tracking tab empty state | Shows "No controls with SLA deadlines" but Controls tab shows SLA data. Contradictory across tabs. | Fix query or clarify empty state with CTA to configure SLA deadlines | 5 perspectives |
| PO-017 | Endpoints tab — color-only status | Green circle indicators on endpoint rows with no text label. Color alone conveys status. | Add PASS/FAIL text label and compliance score column per endpoint | 5 perspectives |
| PO-018 | Framework Management panel — test names | Panel shows "test-001", "test", "Test QA Framework" as custom framework names alongside real one | Remove/rename all test-prefixed seed data frameworks | 4 perspectives |
| PO-019 | Remediate button inconsistency | "Remediate" appears on framework 2 failing controls but not framework 1. No visible outcome on click. | Standardize action availability; wire to deployment wizard or confirmation dialog | 4 perspectives |
| PO-020 | Evaluation history data contradiction | Framework 2 detail shows current score 50% but history cards show transitions from 100% — inconsistent | Fix evaluation history query to show actual recorded scores per framework | 3 perspectives |
| PO-021 | Overdue Controls table — no pagination | No pagination controls, no "Showing X of Y", no total count. Silently truncates at scale. | Add server-side pagination with visible controls and total count | 2 perspectives |

### Minor

| ID | Element | Observation | Suggestion | Flagged By |
|----|---------|-------------|------------|------------|
| PO-022 | Alerts nav link "Alerts 80" | Badge count concatenated into link accessible name. Screen readers announce "Alerts 80 link". | Add `aria-label="Alerts"` on link; `aria-hidden="true"` on badge span | A11Y, UX |
| PO-023 | Duplicate "View Details" accessible names | Two links both named "View Details" — indistinguishable by screen reader link list | Add `aria-label="View details for {framework name}"` per link | A11Y, UX |
| PO-024 | Duplicate "Evaluate" accessible names | Two buttons both named "Evaluate" — indistinguishable by screen reader | Add `aria-label="Evaluate {framework name}"` per button | A11Y, UX |
| PO-025 | Sidebar nav — no semantic grouping | Section labels (OVERVIEW, ASSETS, etc.) are plain StaticText with no list/group semantics | Wrap each section in `<ul aria-label="{Section}">` with `<li>` elements | A11Y, UX |
| PO-026 | ALL-CAPS column headers in HTML | Headers rendered as "FRAMEWORK", "CONTROL ID" etc. in DOM — some screen readers spell letter-by-letter | Use Title Case in HTML, apply `text-transform: uppercase` via CSS | A11Y, QA, EU |
| PO-027 | Responsive 768px layout | Action buttons not visible/clipped at 768px. Sidebar consumes ~40% of viewport. | Collapse sidebar to icon-only at 768px; stack action buttons below title | UX, PM, QA, SE |
| PO-028 | Relative timestamps — no absolute tooltip | "4h ago" labels have no tooltip showing absolute date/time | Add `title` attribute with absolute ISO timestamp on all relative time labels | UX, QA, EB |
| PO-029 | Built-in vs custom framework row anatomy | Framework Management drawer uses different row design for built-in vs custom frameworks | Unify row anatomy; only difference should be built-in lacking delete | UX, EB, SE |
| PO-030 | Icon-only edit/delete buttons — no tooltips or aria-labels | Edit (pencil) and delete (trash) icon buttons in framework management have no accessible names | Add `aria-label="Edit {name}"` / `aria-label="Delete {name}"` per button | A11Y, UX, QA |
| PO-031 | Chart inaccessible to screen readers | Chart uses `role="application"` with no text alternative, data table, or aria description | Add `aria-label` with summary; add visually-hidden data table alternative | A11Y |
| PO-032 | "AI Assistant (coming soon)" in topbar | Active-looking button for unimplemented feature visible during POC demo | Remove button until functional, or style as clearly unavailable | PM, SE |

### Nitpick

| ID | Element | Observation | Suggestion | Flagged By |
|----|---------|-------------|------------|------------|
| PO-033 | Framework detail tab URLs not bookmarkable | Switching tabs does not update URL — cannot bookmark or share direct link to Controls/Endpoints tab | Append tab identifier to URL via search params or nested routes | QA |
| PO-034 | Framework detail heading structure | Tab content sections have no `<h2>` headings — screen reader heading navigation finds nothing below h1 | Add `<h2>` for each content section (Score Breakdown, Failing Controls, etc.) | A11Y |
| PO-035 | Overdue Controls "OVERDUE BY" column abbreviation | "22d", "7d" split into two text nodes; no tooltip for full duration | Add `aria-label="22 days"` or use `<abbr title="days">d</abbr>` | A11Y, QA |
| PO-036 | Notifications live region label includes keyboard shortcut | Region labeled "Notifications alt+T" — shortcut hint should not be in landmark label | Change `aria-label` to simply "Notifications" | A11Y |

---

## Dev Checklist

Critical (must fix before ship)
- [ ] [PO-001] Fix overdue controls SQL join to resolve framework UUID to display name
- [ ] [PO-002] Fix overdue controls query to pull control name from definition, not duplicate control_id
- [ ] [PO-003] Scope overdue controls query to active frameworks only (or clearly label inactive data)
- [ ] [PO-004] Dynamically set chart title to actual date range, or backfill 90 days of seed data
- [ ] [PO-005] Rename "test-001" seed framework to realistic name (also "test", "Test QA Framework")
- [ ] [PO-006] Add global `:focus-visible` outline styles — `outline: 2px solid var(--accent); outline-offset: 2px`

Major (must fix before merge)
- [ ] [PO-007] Add favicon.ico and favicon.svg to `web/public/`
- [ ] [PO-008] Apply semantic color to ring gauge: green >=95%, amber 80-94%, red <80%
- [ ] [PO-009] Show compliance threshold next to NON-COMPLIANT badge ("Target: 95%")
- [ ] [PO-010] Promote "Export Report" to standalone button; remove single-item dropdown
- [ ] [PO-011] Add row-level "View" action to Overdue Controls table linking to framework detail
- [ ] [PO-012] Add loading state + success toast + timestamp refresh to "Evaluate All" action
- [ ] [PO-013] Wire auth/me response to sidebar avatar display name
- [ ] [PO-014] Fix overdue controls summary stat label to "FRAMEWORKS WITH OVERDUE CONTROLS" or show count=4
- [ ] [PO-015] Add Y-axis label ("Compliance Score (%)") to trend charts
- [ ] [PO-016] Fix SLA Tracking tab query or improve empty state with CTA
- [ ] [PO-017] Add PASS/FAIL text labels and compliance score column to endpoints tab
- [ ] [PO-018] Remove/rename all test-prefixed custom frameworks from seed data
- [ ] [PO-019] Standardize Remediate action availability; wire to deployment wizard
- [ ] [PO-020] Fix evaluation history query to show actual recorded scores
- [ ] [PO-021] Add pagination to Overdue Controls table with "Showing X of Y" footer

Minor (fix in this sprint)
- [ ] [PO-022] Add `aria-label="Alerts"` to alerts nav link; hide badge from accessible name
- [ ] [PO-023] Add contextual `aria-label` to "View Details" links with framework name
- [ ] [PO-024] Add contextual `aria-label` to "Evaluate" buttons with framework name
- [ ] [PO-025] Add semantic list grouping (`<ul aria-label>`) to sidebar nav sections
- [ ] [PO-026] Use Title Case in HTML for table headers; apply uppercase via CSS
- [ ] [PO-027] Fix responsive layout at 768px — collapse sidebar, stack action buttons
- [ ] [PO-028] Add absolute timestamp tooltips on all relative time labels
- [ ] [PO-029] Unify framework management row anatomy between built-in and custom
- [ ] [PO-030] Add aria-labels to icon-only edit/delete buttons with target name
- [ ] [PO-031] Add accessible text alternative for compliance trend charts
- [ ] [PO-032] Remove "AI Assistant" button from topbar until functional

Nitpick (fix when convenient)
- [ ] [PO-033] Update URL on tab navigation for bookmarkability
- [ ] [PO-034] Add `<h2>` headings within framework detail tab content sections
- [ ] [PO-035] Add full duration tooltip on "Xd" abbreviations in overdue table
- [ ] [PO-036] Remove keyboard shortcut from notifications region aria-label

---

## Lighthouse Summary

| Category | Score | Status |
|----------|-------|--------|
| Accessibility | 94 | >=90 |
| Best Practices | 100 | >=90 |
| SEO | 60 | 50-89 |

Key Lighthouse flags:
1. SEO score 60 — likely missing meta description, canonical URL, or viewport issues
2. Accessibility 94 — 6 points lost (likely color contrast or missing labels not caught by automated scan)
3. Best Practices 100 — clean

---

## Console & Network Issues

### Console Errors / Warnings

| Level | Count | Top Messages |
|-------|-------|--------------|
| Error | 1 | Failed to load resource: 404 (favicon.ico) |
| Warning | 0 | — |

### Network Issues

| Type | Count | Details |
|------|-------|---------|
| 4xx errors | 1 | GET /favicon.ico [404] |
| 5xx errors | 0 | — |
| Slow requests (>2s) | 0 | — |

All 6 compliance API calls returned 200 successfully.

---

## Comparison to Previous Review

```
Previous review date: 2026-04-03 (earlier session)
Previous verdict:     FAIL (5 critical, 13 major, 9 minor, 4 nitpick = 31 total)

Fixed since last review:  0 findings
New since last review:    5 findings (PO-032 through PO-036)
Remaining from last:      31 findings (all prior findings still open)
Trend:                    DEGRADING (0 fixed, 5 new)
```

All 31 findings from the previous review remain open. This re-review additionally identified 5 new findings from expanded sub-page coverage and more rigorous accessibility testing. Zero regressions were fixed between reviews.

---

## Machine-Readable Output

JSON findings saved to: `.omniprod/findings/2026-04-03-compliance-r2.json`
Lighthouse report: `.omniprod/screenshots/current/report.html`

---

> **This page is not ready to ship.** Address the 6 critical and 15 major findings above, then run `/product-review http://localhost:3001/compliance` again.
