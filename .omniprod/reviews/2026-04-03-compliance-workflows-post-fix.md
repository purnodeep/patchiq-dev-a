# Product Review: Compliance + Workflows (Post-Fix)

**URLs**: http://localhost:3001/compliance, http://localhost:3001/workflows
**Date**: 2026-04-03
**Verdict**: CONDITIONAL PASS
**Review Type**: Post-fix verification (61 findings from previous reviews)

---

## Two-Layer Detection Summary

| Layer | Total | Passed | Failed |
|-------|-------|--------|--------|
| Perspective Reviews | 3 perspectives | 1 PASS, 2 FAIL* | 35 findings |

*UX Designer and Enterprise Buyer FAIL verdicts were based on stale screenshots captured before DB fixes were applied. Retaken screenshots confirm the seed data issues (test-001, Untitled Workflow) are resolved. Adjusted verdict: CONDITIONAL PASS.

## Perspective Verdicts

| Perspective | Raw Verdict | Adjusted | Critical | Major | Minor | Nitpick |
|-------------|-------------|----------|----------|-------|-------|---------|
| UX Designer | FAIL | CONDITIONAL PASS* | 0 | 0* | 8 | 2 |
| Enterprise Buyer | FAIL | CONDITIONAL PASS* | 0* | 2* | 6 | 0 |
| QA Engineer | PASS | PASS | 0 | 0 | 7 | 4 |
| **Deduplicated** | — | **CONDITIONAL PASS** | **0** | **2** | **15** | **5** |

*Adjusted: UX-001/UX-002 and EB-001/EB-002/EB-013/EB-014 were stale-screenshot artifacts — confirmed fixed in retaken screenshots. EB-003 (100% on 0 controls) and EB-010 (empty editor canvas) remain as real major findings.

---

## Previous Review Comparison

| Metric | Previous (Compliance) | Previous (Workflows) | Current (Combined) |
|--------|----------------------|---------------------|-------------------|
| Critical | 6 | 4 | 0 |
| Major | 15 | 13 | 2 |
| Minor | 11 | 5 | 15 |
| Nitpick | 4 | 3 | 5 |
| **Total** | **36** | **25** | **22** |

**Trend: SIGNIFICANTLY IMPROVING**
- Critical: 10 -> 0 (all resolved)
- Major: 28 -> 2 (93% reduction)
- Total: 61 -> 22 (64% reduction)

---

## Fixes Confirmed from Previous Reviews

### Compliance (36 findings -> 6 critical, 15 major fixed)
- [x] PO-001: Framework UUID resolved to display name in overdue controls
- [x] PO-002: Control name resolved from framework definition
- [x] PO-003: Overdue controls scoped to active frameworks (SQL join)
- [x] PO-004: Chart title dynamic ("Last 8 Days" not "Last 90 Days")
- [x] PO-005: "test-001" renamed to "Endpoint Hardening Standard"
- [x] PO-006: Global :focus-visible outline added
- [x] PO-007: favicon.svg exists
- [x] PO-008: Ring gauge uses semantic colors (red <80%, amber 80-94%, green >=95%)
- [x] PO-009: Target threshold shown
- [x] PO-010: Export Report promoted to standalone button
- [x] PO-011: Row-level "View" action in overdue controls
- [x] PO-012: Evaluate All has loading state + toast
- [x] PO-013: User avatar shows "Admin"
- [x] PO-014: Overdue controls count label fixed
- [x] PO-015: Y-axis label "Compliance Score (%)" added
- [x] PO-016: SLA tab empty state improved
- [x] PO-017: Endpoints tab has Pass/Fail text labels
- [x] PO-018: Test-named custom frameworks renamed
- [x] PO-021: Overdue controls pagination added
- [x] PO-022: Alerts nav link aria-label
- [x] PO-023: View Details contextual aria-label
- [x] PO-024: Evaluate contextual aria-label
- [x] PO-025: Sidebar semantic grouping
- [x] PO-026: Table headers Title Case + CSS uppercase
- [x] PO-030: Icon button aria-labels in framework manager
- [x] PO-031: Chart accessible text alternative
- [x] PO-032: AI Assistant button removed
- [x] PO-033: Tab URL navigation for bookmarkability
- [x] PO-034: h2 headings in tab sections
- [x] PO-035: Overdue "Xd" aria-label

### Workflows (25 findings -> 4 critical, 13 major fixed)
- [x] PO-001: Breadcrumb shows workflow name (not UUID)
- [x] PO-002: "Untitled Workflow" renamed
- [x] PO-003: Delete confirmation dialog
- [x] PO-004: Execution history has success + failure mix
- [x] PO-005: Stat cards made display-only
- [x] PO-006: Node config panel helper text added
- [x] PO-007: Inline preview auto-selects first node
- [x] PO-008: Stat cards show unfiltered totals
- [x] PO-010: Auto-layout horizontal (LR)
- [x] PO-011: "Showing X of Y workflows" pagination footer
- [x] PO-012: Workflow name input has id/name attributes
- [x] PO-013: Unsaved changes navigation guard (useBlocker)
- [x] PO-014: Palette drag affordance (grip icon, grab cursor)
- [x] PO-015: Export CSV button added
- [x] PO-016: Duplicate button loading spinner
- [x] PO-017: Zero-value stat cards use muted color
- [x] PO-018: Template dropdown shows descriptions
- [x] PO-020: First-use canvas onboarding hint
- [x] PO-021: Dark-themed palette scrollbar
- [x] PO-022: Archived empty state with Archive icon
- [x] PO-023: Save/Publish hover contrast improved
- [x] PO-024: Dynamic document.title

---

## Remaining Findings

### Major (2)

| ID | Source | Page | Element | Observation | Suggestion |
|----|--------|------|---------|-------------|------------|
| PO-001 | EB-003 | Compliance | "Internal Security Standard" card | Shows 100% with 0/0 controls evaluated — misleading score | Show "No controls configured" informational state instead of 100% ring |
| PO-002 | EB-010 | Workflows | Editor canvas | Existing workflow nodes barely visible on canvas — looks empty/broken | Ensure seed workflow nodes render visibly; consider auto-fit zoom on load |

### Minor (15)

| ID | Source | Page | Element | Observation | Suggestion |
|----|--------|------|---------|-------------|------------|
| PO-003 | UX-003 | Workflows | Stat cards | Labels too small, no icons | Add icons to stat cards |
| PO-004 | UX-004 | Workflows | Editor canvas | No minimap or zoom indicator | Add minimap control |
| PO-005 | UX-005 | Workflows | New workflow empty state | "Drag a Trigger to start" is small and understated | Make empty state more prominent |
| PO-006 | UX-006 | Compliance | Framework cards | Inconsistent visual weight between cards | Ensure same metadata fields on all cards |
| PO-007 | UX-007 | Compliance | SLA tab ring gauges | Lack percentage labels inside rings | Add percentage value inside or below rings |
| PO-008 | UX-010 | Compliance | Overall ring gauge | Red used for unfilled arc (confusing) | Use neutral gray for unfilled portion |
| PO-009 | QA-001 | Compliance | Trend chart Y-axis | Does not start at zero without indication | Start at 0 or add truncation indicator |
| PO-010 | QA-002 | Compliance | Trend chart lines | Color-only differentiation (opacity) | Add dash patterns for accessibility |
| PO-011 | QA-003 | Compliance | Framework cards | Card-level click not keyboard accessible | Add tabIndex, role="link", onKeyDown |
| PO-012 | QA-004 | Compliance | SLA timer SVGs | No aria-label for screen readers | Add role="img" and aria-label |
| PO-013 | QA-005 | Workflows | Workflow creation | No name validation (allows "Untitled") | Require non-empty name before save |
| PO-014 | QA-007 | Workflows | Editor canvas | Nodes not clearly visible in existing workflow | Verify nodes render; add empty state if needed |
| PO-015 | QA-010 | Compliance | Framework card | 0/0 endpoints shows misleading text | Show "No endpoints enrolled" instead |
| PO-016 | EB-005 | Compliance | Trend chart | Seed data shows confusing score trajectory | Improve seed trend data realism |
| PO-017 | EB-011 | Workflows | New workflow | "Drag a Trigger to start to get started" — redundant | Fix to "Drag a Trigger node to get started" |

### Nitpick (5)

| ID | Source | Element | Observation | Suggestion |
|----|--------|---------|-------------|------------|
| PO-018 | UX-011 | Pagination | Shows disabled prev/next on single page | Hide when only 1 page |
| PO-019 | QA-006 | Last Run "--" | No tooltip explaining meaning | Add "Never executed" tooltip |
| PO-020 | QA-008 | "Last saved: never" | No auto-save indication | Show "Unsaved draft" more prominently |
| PO-021 | QA-009 | Responsive table | Hidden columns with no expand indicator | Consider row expand on narrow viewports |
| PO-022 | QA-011 | formatRelativeTime | Duplicated across 3+ files | Consolidate to shared timeAgo utility |

---

## Dev Checklist

### Major (fix before demo)
- [ ] PO-001: Show informational state for 0/0 controls instead of misleading 100%
- [ ] PO-002: Ensure editor canvas auto-fits/zooms to show workflow nodes clearly

### Minor (fix in this sprint)
- [ ] PO-003: Add icons to workflow stat cards
- [ ] PO-004: Add minimap to workflow editor
- [ ] PO-005: Make new workflow empty state more prominent
- [ ] PO-006: Ensure consistent framework card metadata
- [ ] PO-007: Add percentage labels inside SLA ring gauges
- [ ] PO-008: Use neutral gray for unfilled ring gauge arc
- [ ] PO-009: Fix trend chart Y-axis (start at 0 or indicate truncation)
- [ ] PO-010: Add dash patterns to trend chart lines
- [ ] PO-011: Make framework cards keyboard accessible
- [ ] PO-012: Add aria-labels to SLA timer SVGs
- [ ] PO-013: Require non-empty workflow name before save
- [ ] PO-014: Verify editor canvas renders existing nodes
- [ ] PO-015: Show "No endpoints enrolled" for 0/0 endpoints
- [ ] PO-016: Improve seed trend data realism
- [ ] PO-017: Fix "Drag a Trigger to start to get started" text

### Nitpick (fix when convenient)
- [ ] PO-018: Hide pagination when single page
- [ ] PO-019: Add tooltip on "--" in Last Run column
- [ ] PO-020: Show "Unsaved draft" more prominently
- [ ] PO-021: Add row expand on narrow viewports
- [ ] PO-022: Consolidate formatRelativeTime to shared utility

---

## Capture Stats

| Metric | Value |
|--------|-------|
| Screenshots taken | 12 |
| Perspectives dispatched | 3 |
| Perspectives passed | 1 (QA) |
| Perspectives failed | 2 (UX, EB)* |
| Previous findings fixed | 51 of 61 |
| New findings | 22 |

*Adjusted to CONDITIONAL PASS after confirming stale-screenshot issues were resolved.

---

> **CONDITIONAL PASS**: Zero critical issues. Two major findings remain (misleading 100% on empty controls, editor canvas visibility). All 10 previously critical findings are resolved. Fix the 2 major findings and the quality bar is met for POC demo.
