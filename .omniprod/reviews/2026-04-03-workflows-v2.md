# Product Review: Workflows (Re-Review)

**URL**: http://localhost:3001/workflows
**Date**: 2026-04-03
**Verdict**: FAIL
**Review Version**: v2 (OmniProd v0.3.0 — two-layer detection)

---

## Two-Layer Detection Summary

| Layer | Total | Passed | Failed |
|-------|-------|--------|--------|
| Automated Assertions | 9 | 9 | 0 |
| Perspective Reviews | 3 perspectives | — | 25 findings |

All 9 automated assertions passed (DOM errors, a11y basics, data integrity, performance). All findings come from perspective review (human judgment layer).

## Perspective Verdicts

| Perspective | Verdict | Critical | Major | Minor | Nitpick |
|-------------|---------|----------|-------|-------|---------|
| UX Designer | FAIL | 0 | 10 | 7 | 4 |
| Enterprise Buyer | FAIL | 3 | 8 | 3 | 3 |
| QA Engineer | FAIL | 1 | 7 | 5 | 2 |

## Lighthouse

| Category | Score |
|----------|-------|
| Accessibility | 92 |
| Best Practices | 100 |
| SEO | 60 |

---

## Comparison to Previous Review

| Metric | Previous | Current | Change |
|--------|----------|---------|--------|
| Total findings | 53 | 25 | -28 (-53%) |
| Critical | 10 | 4 | -6 |
| Major | 23 | 13 | -10 |
| Minor | 14 | 5 | -9 |
| Nitpick | 6 | 3 | -3 |

**Trend: IMPROVING**

### Fixes Confirmed (8 from previous review)
- PO-007: Expand buttons now have contextual accessible names
- PO-008: Page now has `<main>` landmark
- PO-009: Published workflows now exist in seed data (3 published)
- PO-010: Run button now visible on published workflow rows
- PO-018: Empty states now show "Clear filters" recovery action
- PO-022: Row action buttons now have contextual aria-labels
- PO-023: Filter tabs now have proper ARIA tablist/tab/aria-selected roles
- PO-038: Relative timestamps now show absolute date in tooltip

---

## Findings — Critical (4)

| ID | Element | Observation | Suggestion | Flagged By |
|----|---------|-------------|------------|------------|
| PO-001 | Breadcrumb in editor | Displays raw UUID instead of workflow name | Resolve name from loaded record, set document.title | EB, UX, QA |
| PO-002 | Seed data — "Untitled Workflow" | Placeholder name, no description, 0 runs visible in demo | Replace with meaningful name, add validation | EB, UX, QA |
| PO-003 | More Actions menu | Dropdown renders but functional wiring unverified; no confirmation dialog for Delete | Verify handlers work, add confirmation dialog | EB, UX, QA |
| PO-004 | Execution history status | All records show identical status; Last Run shows failures for some, dash for others, nothing for rest — data inconsistency | Fix status mapping, seed data should have success + failure mix | EB, UX, QA |

## Findings — Major (13)

| ID | Element | Observation | Suggestion | Flagged By |
|----|---------|-------------|------------|------------|
| PO-005 | Stat cards + tab pills | Duplicate filtering controls for same dimension | Make stat cards display-only or remove tab pills | EB, UX, QA |
| PO-006 | Editor node config panel | Fields lack descriptions, units, validation hints | Add labels, units, info tooltips, validation constraints | EB, UX |
| PO-007 | Inline preview config panel | Placeholder text on empty state; ambiguous if display-only or editable | Auto-select first node; label "Node Details" with "Edit in editor" CTA | EB, UX, QA |
| PO-008 | Stat cards during search | Show 0 TOTAL when search returns no results — misleading | Decouple stat cards from search filter; show unfiltered totals | EB, UX, QA |
| PO-009 | Last Run column | 3 different display patterns in one column; no text labels for status | Standardize: Badge component with text + relative time for all rows | EB, UX, QA |
| PO-010 | Auto-layout direction | Produces vertical stack; existing workflows display horizontally | Default to horizontal (left-to-right) via elkjs | EB, UX, QA |
| PO-011 | Pagination footer | No "Showing X of Y" or page position indicator | Add "Showing 1-6 of 6 workflows" text | EB, UX, QA |
| PO-012 | Console form field | Missing id/name attribute — affects autofill, a11y, testing | Add id and name attributes to input | EB, QA |
| PO-013 | Editor unsaved changes | Navigate away silently discards changes | Add beforeunload handler + router navigation guard | UX |
| PO-014 | Node palette | No drag affordance — no grab cursor, no handle icon, no tooltip | Add cursor:grab, grip icon, tooltip "Drag to canvas" | UX, QA |
| PO-015 | No export capability | No CSV/XLSX export for workflow list | Add export button in page header | EB |
| PO-016 | Duplicate button | No loading/success feedback when clicked | Wire to API, show spinner + toast | QA |
| PO-017 | Draft stat card color | "0 PUBLISHED" rendered in green — contradictory signal | Neutral color for zero values | EB |

## Findings — Minor (5)

| ID | Element | Observation | Suggestion | Flagged By |
|----|---------|-------------|------------|------------|
| PO-018 | Template dropdown | Names only, no descriptions | Add 1-line description per template | EB, UX |
| PO-019 | Responsive 768px | No layout adaptation — sidebar full, table cramped | Collapse sidebar; hide secondary columns | EB, UX, QA |
| PO-020 | New workflow canvas | No onboarding guidance for first-time users | Add "Drag nodes from palette" overlay | UX |
| PO-021 | Palette scrollbar | White browser-default on dark theme | Apply dark theme scrollbar styling | UX |
| PO-022 | Archived empty state | Generic icon and message | Use Archive icon, contextual copy | UX |

## Findings — Nitpick (3)

| ID | Element | Observation | Suggestion | Flagged By |
|----|---------|-------------|------------|------------|
| PO-023 | Save/Run hover states | Indistinguishable from default in dark theme | Increase hover contrast | UX, QA |
| PO-024 | SEO score 60 | Missing meta description, generic page title | Set document.title to "Workflows — PatchIQ" | EB, QA |
| PO-025 | "Open Editor" vs "Edit" | Same destination, different labels | Standardize to "Edit" | UX |

---

## Dev Checklist

### Critical (fix before any demo)
- [ ] **PO-001**: Resolve workflow name in breadcrumb (editor) — `web/src/pages/workflows/editor.tsx`
- [ ] **PO-002**: Replace "Untitled Workflow" in seed data + add name validation — `internal/server/store/seed/`
- [ ] **PO-003**: Verify More Actions menu handlers work; add Delete confirmation dialog — `web/src/pages/workflows/workflow-card.tsx`
- [ ] **PO-004**: Fix execution status mapping; add success runs to seed data — `internal/server/deployment/` + seed data

### Major (fix before ship)
- [ ] **PO-005**: Remove click handler from stat cards OR remove tab pills
- [ ] **PO-006**: Add labels, units, tooltips to node config panel fields
- [ ] **PO-007**: Auto-select first node in inline preview; add "Edit in editor" CTA
- [ ] **PO-008**: Decouple stat card values from search/filter — always show unfiltered totals
- [ ] **PO-009**: Standardize Last Run column: Badge + text + relative time
- [ ] **PO-010**: Change auto-layout default to horizontal (LR) in elkjs config
- [ ] **PO-011**: Add "Showing X of Y workflows" to pagination footer
- [ ] **PO-012**: Add id/name attributes to workflow name input
- [ ] **PO-013**: Add beforeunload + router guard for unsaved changes
- [ ] **PO-014**: Add drag affordance to palette items (cursor:grab, grip icon)
- [ ] **PO-015**: Add CSV export button to page header
- [ ] **PO-016**: Wire Duplicate button to API with loading spinner + success toast
- [ ] **PO-017**: Neutral color for zero-value stat cards

### Minor (fix when convenient)
- [ ] PO-018: Add descriptions to template dropdown items
- [ ] PO-019: Responsive sidebar collapse at < 1024px
- [ ] PO-020: First-use canvas onboarding overlay
- [ ] PO-021: Dark-themed scrollbar for palette
- [ ] PO-022: Contextual archived empty state

---

## Capture Stats

| Metric | Value |
|--------|-------|
| Screenshots taken | 24 |
| Coverage targets completed | 27 / 36 (75%) |
| Assertions run | 9 |
| Assertions failed | 0 |
| Console issues | 1 |
| Perspectives dispatched | 3 |
