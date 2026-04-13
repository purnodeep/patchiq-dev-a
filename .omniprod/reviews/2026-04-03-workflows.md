# Review: Workflows Page

- **URL:** http://localhost:3001/workflows
- **Date:** 2026-04-03
- **Reviewer:** product-observer (automated)
- **Verdict:** ❌ FAIL

---

## Perspective Verdicts

| Perspective | Verdict | Critical | Major | Minor | Nitpick |
|-------------|---------|----------|-------|-------|---------|
| UX Designer | ❌ FAIL | 0 | 13 | 8 | 4 |
| QA Engineer | ❌ FAIL | 5 | 9 | 7 | 3 |
| Enterprise Buyer | ❌ FAIL | 4 | 11 | 3 | 3 |
| Accessibility Expert | ❌ FAIL | 2 | 14 | 5 | 0 |
| Product Manager | ❌ FAIL | 5 | 10 | 3 | 1 |
| CTO Architect | ❌ FAIL | 3 | 14 | 3 | 0 |
| Sales Engineer | ❌ FAIL | 4 | 8 | 6 | 0 |
| End User | ❌ FAIL | 4 | 11 | 4 | 2 |
| **Total (deduplicated)** | **❌ FAIL** | **10** | **23** | **14** | **6** |

---

## Findings

### Critical

| ID | Element | Observation | Suggestion | Flagged By |
|----|---------|-------------|------------|------------|
| PO-001 | More actions button (row action column) | Button renders with hover styles but has no click handler — opens nothing, executes nothing. A broken action in production demos signals incomplete engineering. | Implement dropdown with Archive, Delete (with confirmation) or remove the button until ready. | QA, EB, PM, SE, EU |
| PO-002 | Duplicate button (row action column) | Button silently no-ops on click — no handler, no feedback, no error. Broken affordance with no recovery path. | Wire to clone-workflow API with optimistic UI update, or remove until implemented. | QA, EB, PM, SE, EU |
| PO-003 | React key prop — WorkflowsPage list render | React.Fragment used in map without a key prop — console error fires on every page load. Indicates an untested render path. | Add `key={workflow.id}` to the Fragment wrapper in the list map. | QA, EB, UX, PM, CTO, SE, EU |
| PO-004 | Workflow name form field | Field is missing `id` and `name` attributes — browser logs an accessibility warning on every load. Breaks label association and autofill. | Add `id="workflow-name"` and `name="workflow-name"` to the input element. | QA, EB, UX, A11Y, CTO, SE, EU |
| PO-005 | LAST RUN column + execution history drawer | `last_run_status` is null despite 10 execution records existing. LAST RUN shows "—" for affected rows. All history records show "paused" status regardless of actual outcome. Backend bug: status is not written back to the workflow record after execution completes. | Fix execution completion handler to update `last_run_at` and `last_run_status`. Fix execution status enum mapping so completed/failed/running states render correctly. | QA, EB, UX, PM, CTO, SE, EU |
| PO-006 | Stat card counts and filter tabs | Client-side filtering operates on the current page of 25 records only. With 100+ workflows, the DRAFT count will show e.g. 18 instead of the actual 94. Data integrity failure at any real customer scale. | Move filter state to URL query params, pass to server. Add server-side aggregate count endpoints for stat cards. | QA, PM, CTO, EB |
| PO-007 | Row-expand toggle buttons (chevron icons) | Expand/collapse buttons have no accessible name — screen readers announce "button" with no context about which row. Flagged by axe in Lighthouse audit. | Add `aria-label="Expand {workflow name}"` toggling to `aria-label="Collapse {workflow name}"`. | A11Y |
| PO-008 | Page-level `<main>` landmark missing | Page has no `<main>` element — assistive technology users cannot skip to primary content. Lighthouse a11y flags this. | Wrap page content area in `<main>` or add `role="main"` to the top-level content container. | A11Y |
| PO-009 | All workflows DRAFT — zero published in seed data | Demo dataset contains zero published workflows. "0 Published" stat card renders in green/brand color. Published tab shows empty state with no CTA. Tells an enterprise buyer the platform has never shipped a live automation. | Add at least 2 seed workflows in PUBLISHED status. Fix stat card highlight logic so a zero-value published card does not render in success/brand color. | EB, PM, SE, UX |
| PO-010 | No Run Now / manual trigger from list or editor | There is no way to manually trigger execution of a published workflow from the list page or the editor. RUNS column is display-only. No execution monitoring view exists. | Add a "Run" button to PUBLISHED workflow rows (and in more actions). Surface an execution monitoring panel or link to per-workflow execution history. | PM, CTO, EU, SE |

### Major

| ID | Element | Observation | Suggestion | Flagged By |
|----|---------|-------------|------------|------------|
| PO-011 | Stat card row + filter pill row | Both UI components perform identical status filtering on click — redundant controls with no visual distinction between informational and interactive roles. | Make stat cards display-only (remove click handler) or remove filter pills. One filter surface is sufficient. | UX, QA, PM, CTO, EB |
| PO-012 | Row click vs. expand chevron — competing interactions | Clicking any part of a table row opens the inline editor panel. Clicking the chevron expands row details. The two interaction targets overlap; users cannot predict which action will fire. | Separate intents: row click expands details, explicit "Edit" button opens inline editor. | UX, QA, PM, EB |
| PO-013 | Inline editor — right config panel shows placeholder text | Selecting a node in the inline editor highlights it with a green border, but the right panel renders "Enter Configuration" placeholder instead of the node's properties. State binding between node selection and config panel is broken. | Fix the selected-node to config panel data flow. Panel should render appropriate fields for the node type on selection. | UX, QA, EB, SE, CTO |
| PO-014 | Full canvas editor — node config panels empty | In the full /workflows/{id}/edit editor, node config panels render with no labels, no field controls, no explanatory text. No node type has a working configuration form. | Implement config form components for each node type: Trigger (schedule/event), Action (patch targets, wave %), Condition (rule builder). | UX, EB, A11Y, CTO, SE |
| PO-015 | Breadcrumb shows raw UUID in editor | In /workflows/{uuid}/edit, the breadcrumb renders the raw UUID instead of the workflow name. Exposed UUIDs break professional appearance and reveal internal identifiers to clients. | Resolve workflow name from the loaded workflow record and render it in the breadcrumb. Also set `document.title` to match. | UX, QA, EB, A11Y, PM, SE, EU |
| PO-016 | "Untitled Workflow" in seed / demo data | Default seed contains a workflow with no name, no description, 0 runs, and no last-run date. Placeholder data visible in a demo environment damages credibility. | Rename seed workflow with a meaningful name and description. Add name validation: require non-empty name before a workflow can be saved. | UX, QA, EB, PM, CTO |
| PO-017 | Search does not reset pagination cursor | Typing in the search field filters the visible list but does not reset to page 1. Users searching for a workflow on page 3 may get zero results even though the match is on page 1 of the search results. | On search input change, reset cursor/page to 0. Move search to server-side via `?q=` query param. | QA, PM, CTO |
| PO-018 | No "Clear filters" / "Clear search" in empty filtered state | When filtering or searching returns no results, there is no recovery action in the empty state message. User is stuck. | Add "Clear filters" / "Clear search" button below the empty state copy. | UX, QA, PM, EU |
| PO-019 | Node palette — keyboard inaccessible, no drag affordance | Nodes in the palette are drag-only. No keyboard alternative exists to add a node to the canvas. Palette items show no visual affordance (grab cursor, drag handle) indicating they are draggable. | Add keyboard alternative: select node with Enter/Space then click canvas to place. Add `cursor: grab` and a drag-handle icon to palette items. | UX, A11Y |
| PO-020 | Sidebar nav and section label contrast failures | Sidebar navigation item labels fail WCAG AA contrast: nav labels 4.42:1 (threshold 4.5:1), section group labels 2.68:1, alerts badge 3.76:1. | Increase sidebar text color values to achieve ≥4.5:1 for normal text. | A11Y |
| PO-021 | Stat card label, table header, and row text contrast failures | Stat card labels 3.97:1, table column headers 4.17:1, row description/timestamp text 3.97:1 — all below WCAG AA 4.5:1 threshold. | Darken these text tokens in the design system CSS variables; verify in both light and dark modes. | A11Y |
| PO-022 | "Edit", "Duplicate", "More actions" buttons lack row context | "Edit" link announces as "Edit" with no workflow name. "Duplicate" and "More actions" buttons have no row identification. Screen reader users cannot distinguish which row an action targets. | Add `aria-label="Edit {name}"`, `aria-label="Duplicate {name}"`, `aria-label="More actions for {name}"` to each button. | A11Y |
| PO-023 | Filter tabs missing ARIA role and state | Status filter tabs (All, Draft, Published, Archived) have no `role="tab"`, no `aria-selected`, no `role="tablist"` container. Keyboard and screen reader navigation is broken. | Add `role="tablist"` to container, `role="tab"` and `aria-selected={isActive}` to each tab. | A11Y |
| PO-024 | Search input missing programmatic label | Search field has no `<label>`, no `aria-label`, and no `aria-labelledby`. Screen readers cannot announce what the field is for. | Add `aria-label="Search workflows"` to the input or a visually-hidden `<label>`. | A11Y |
| PO-025 | Publish button — no pre-flight guard, no loading state, no error handler | Clicking Publish issues the API call with no validation check, no loading/spinner, and no `onError` callback. A failed publish silently disappears. | Add pre-flight validation before calling publish. Show loading state on button during request. Show toast on success and on error. | QA, PM |
| PO-026 | Node config Save — no commit feedback | After editing a node config and clicking Save, there is no loading indicator, no success toast, and no visual confirmation the save occurred. | Add loading state to Save button. Show success toast on completion. Show error toast on failure. | UX, QA, SE |
| PO-027 | No unsaved changes warning on navigate away from editor | Navigating away from the editor after making changes gives no confirmation prompt. Changes are silently discarded. | Add `beforeunload` handler and router-level navigation guard showing "You have unsaved changes — discard?" dialog. | PM |
| PO-028 | No execution monitoring view | There is no page or panel showing live execution progress, per-node status, logs, or retry controls for a running workflow. The RUNS column is non-interactive. | Add execution detail view with per-step status, duration, and log output. Make RUNS column a link. | CTO, PM, EU |
| PO-029 | Version history invisible in UI | Workflow version history exists in the data model but is not surfaced in any UI. No version compare, no rollback, no audit trail for workflow changes. | Add version history panel in the editor with past saves, diff view, and one-click rollback. | CTO |
| PO-030 | Inline canvas parallel-branch layout drops or overlaps nodes | Parallel branches in the inline editor DAG canvas render with node overlap or are dropped entirely due to layout engine constraints. | Upgrade inline canvas to elk.js horizontal-rank layout or configure dagre with proper parallel-branch spacing. | CTO |
| PO-031 | Workflow name input placeholder copy — typo and incomplete grammar | New workflow name input shows "Work Flow Name..." (two words, mixed case). Template selector shows "Build from name..." (grammatically incomplete). | Fix to "Workflow name" and "Search templates..." respectively. | UX, SE, EB |
| PO-032 | Topbar search — label contrast failure and label-content-name mismatch | Topbar search has contrast failure on placeholder text and the visible label text does not match the aria-label (WCAG 2.5.3 violation). | Align visible label with aria-label. Fix placeholder contrast to ≥4.5:1. | A11Y |
| PO-033 | Inline editor "Enter Configuration" dead placeholder state | When the inline editor panel opens with no node selected, or after selecting a node with no config, the panel shows static placeholder text with no action available. | Show useful empty state: "Select a node to configure it" with icon, or auto-select the first node. | UX, EB |

### Minor

| ID | Element | Observation | Suggestion | Flagged By |
|----|---------|-------------|------------|------------|
| PO-034 | Table column headers — ALL-CAPS style | Column headers (NAME, STATUS, NODES, RUNS, LAST RUN, UPDATED) use ALL-CAPS, inconsistent with other pages in the product that use Title Case. | Standardize to Title Case across all table headers product-wide. | UX |
| PO-035 | Node palette — no drag affordance visual cues | Palette items show no cursor change, no drag handle icon, and no tooltip indicating they are draggable onto the canvas. | Add `cursor: grab`, a drag-handle glyph, and a tooltip "Drag to canvas" on hover. | UX |
| PO-036 | No "filtered from X total" count indicator | When a filter is active the list shows results but gives no indication of total record count being filtered. | Add "Showing 12 of 47 workflows" label below the filter row or in the pagination footer. | UX |
| PO-037 | Pagination no total count | Pagination shows Previous/Next buttons only. No "Page 2 of 5" or "Showing 26–50 of 47" indicator. | Add total count to pagination footer. Requires server-side count endpoint. | UX, QA, EB |
| PO-038 | Relative timestamps no absolute tooltip | LAST RUN and UPDATED columns show relative times ("3 days ago") with no tooltip showing the ISO absolute date. Time context is lost for older records. | Add `<Tooltip content={isoDate}><span>{relativeTime}</span></Tooltip>` to all time cells. | UX, QA, EB, EU |
| PO-039 | Table at 768px clips action column | At 768px viewport the action column is cut off — only NAME, STATUS, and NODES are fully visible. | Implement responsive column priority: hide NODES, RUNS, UPDATED at ≤900px; keep action column always reachable or move to row expand. | UX, QA, EB |
| PO-040 | Editor at 768px — palette overlaps canvas, non-functional | At 768px the node palette overlaps the canvas making both unusable. | Collapse palette to a drawer/bottom-sheet at ≤1024px. | UX, QA, EB, A11Y |
| PO-041 | outline:none on stat cards and editor name input — focus ring invisible | Explicit `outline: none` on stat card interactive elements and editor name input removes focus ring with no replacement, making keyboard focus invisible. | Replace with `focus-visible:ring-2` Tailwind utility or equivalent `:focus-visible` CSS rule. | QA, EB, CTO |
| PO-042 | New workflow blank canvas — no onboarding guidance | When creating a new workflow, the canvas shows a single Trigger node on blank space with no instructions, no template shortcuts, no tooltip hints. | Add first-use overlay: "Add your first action — drag from the palette or click +" with a pointer to the palette. | UX, CTO, PM |
| PO-043 | NODES column raw integer no unit context | NODES column shows a plain integer (e.g. "4") with no unit label or icon indicating what it represents. | Change to show "4 nodes" or use a node-count icon badge. Add info tooltip to column header. | EB |
| PO-044 | RUNS column no time scope and not clickable | RUNS shows a cumulative integer with no scope (all-time vs. 30-day) and clicking it does nothing. | Add "(all time)" sub-label or tooltip. Make RUNS count a link to the workflow's execution history. | EB, EU |
| PO-045 | Canvas aria-live="assertive" interrupts screen readers | The DAG canvas uses `aria-live="assertive"` which interrupts any ongoing screen reader announcement when the canvas updates. | Change to `aria-live="polite"` for non-critical canvas status updates. | A11Y |
| PO-046 | Palette search uses placeholder as its only label | The palette filter input has no `<label>` — relies solely on placeholder text which disappears on focus and is not read by all screen readers. | Add a visually hidden `<label>` or `aria-label="Filter node palette"`. | A11Y |
| PO-047 | Icon-only topbar buttons no tooltip on keyboard focus | Icon-only action buttons in the topbar show tooltips on mouse hover but not on keyboard focus. | Trigger tooltip on `:focus-visible` in addition to `:hover`. | A11Y |

### Nitpick

| ID | Element | Observation | Suggestion | Flagged By |
|----|---------|-------------|------------|------------|
| PO-048 | Palette search — no clear button | Palette filter input shows no X / clear button once text is entered. User must manually select and delete. | Add a clear icon button that appears when the input is non-empty. | UX |
| PO-049 | "Open Editor" vs. "Edit" label inconsistency | Expanded row CTA says "Open Editor"; row action button says "Edit"; both navigate to the same URL. | Standardize to "Edit" everywhere (shorter, consistent with product-wide conventions). | UX, EB |
| PO-050 | Archived tab shows "0" count badge | Archived tab renders a "0" badge, which is visual noise when there are no archived workflows. | Suppress the badge when count is 0. | EB |
| PO-051 | Zap icon used for node count column | NODES column uses a zap/lightning bolt icon which implies execution speed, not node count. | Use a graph/node icon (e.g. GitFork, Network) for the node count indicator. | PM |
| PO-052 | Topbar breadcrumb contrast 4.42:1 — marginally below threshold | Topbar breadcrumb text renders at 4.42:1 contrast, just below the WCAG AA 4.5:1 threshold. | Increase breadcrumb text color by 1–2 shades to reach ≥4.5:1. | A11Y |
| PO-053 | Expanded row relative time truncated — "46 ago" | Expanded row timestamp renders as "46 ago" with the unit word missing or truncated. | Fix timeAgo formatter to always include the unit: "46 minutes ago". Handle future timestamps gracefully. | EB, QA |

---

## Dev Checklist

Ordered by severity. Check off items as they are resolved.

```
Critical (must fix before ship)
- [ ] [PO-001] Implement More actions dropdown (Archive, Delete with confirm) or remove stub button
- [ ] [PO-002] Wire Duplicate button to clone-workflow API or remove from UI
- [ ] [PO-003] Add key={workflow.id} to React.Fragment in workflow list map
- [ ] [PO-004] Add id and name attributes to workflow name form field
- [ ] [PO-005] Fix execution completion handler to write back last_run_at/last_run_status; fix execution status enum display
- [ ] [PO-006] Move filtering to server-side query params; add aggregate count endpoints for stat cards
- [ ] [PO-007] Add aria-label="Expand/Collapse {workflow name}" to row chevron buttons
- [ ] [PO-008] Add <main> landmark wrapping the page content area
- [ ] [PO-009] Add 2+ PUBLISHED seed workflows; fix green highlight logic on zero-value stat card
- [ ] [PO-010] Add Run Now button for published workflows; add execution monitoring panel

Major (must fix before merge)
- [ ] [PO-011] Remove click handler from stat cards OR remove filter pills — one filter surface only
- [ ] [PO-012] Separate row click (expand) from inline editor trigger — use explicit button
- [ ] [PO-013] Fix node selection to config panel state binding in inline editor
- [ ] [PO-014] Implement config form components for each node type in full editor
- [ ] [PO-015] Replace UUID with workflow name in breadcrumb and document title
- [ ] [PO-016] Rename seed "Untitled Workflow"; add non-empty name validation on save
- [ ] [PO-017] Reset pagination cursor on search; move search to server-side ?q= param
- [ ] [PO-018] Add "Clear filters" / "Clear search" button to empty filtered/search state
- [ ] [PO-019] Add keyboard alternative for node palette (select + place); add drag affordance styles
- [ ] [PO-020] Fix sidebar nav label contrast to ≥4.5:1; fix section group label contrast
- [ ] [PO-021] Fix stat card label, table header, and row text contrast to ≥4.5:1
- [ ] [PO-022] Add contextual aria-label to Edit, Duplicate, More actions buttons per row
- [ ] [PO-023] Add role="tablist"/role="tab"/aria-selected to filter tab row
- [ ] [PO-024] Add aria-label="Search workflows" to search input
- [ ] [PO-025] Add pre-flight validation + loading state + onError to Publish button
- [ ] [PO-026] Add loading state + success/error toast to node config Save button
- [ ] [PO-027] Add unsaved changes warning on navigate-away from editor
- [ ] [PO-028] Add execution monitoring view with per-step status and logs; make RUNS column a link
- [ ] [PO-029] Surface version history panel in editor with diff view and rollback
- [ ] [PO-030] Fix inline canvas parallel-branch layout so nodes do not overlap or drop
- [ ] [PO-031] Fix "Work Flow Name..." to "Workflow name" and "Build from name..." to "Search templates..."
- [ ] [PO-032] Fix topbar search aria-label to match visible label; fix placeholder contrast
- [ ] [PO-033] Fix inline editor no-selection state to show "Select a node to configure it"

Minor (fix in this sprint)
- [ ] [PO-034] Standardize table column headers to Title Case
- [ ] [PO-035] Add cursor:grab + drag-handle icon + tooltip to palette items
- [ ] [PO-036] Add "Showing X of Y" filter count indicator below filter row
- [ ] [PO-037] Add total count to pagination footer
- [ ] [PO-038] Add absolute-date tooltip to all relative timestamps
- [ ] [PO-039] Hide lower-priority columns at ≤900px; keep action column always accessible
- [ ] [PO-040] Collapse node palette to drawer/bottom-sheet at ≤1024px
- [ ] [PO-041] Replace outline:none with focus-visible ring on stat cards and editor name input
- [ ] [PO-042] Add first-use onboarding overlay or guidance to new workflow blank canvas
- [ ] [PO-043] Add unit context to NODES column (icon or sub-label)
- [ ] [PO-044] Add time scope sub-label to RUNS; make count a link to execution history
- [ ] [PO-045] Change canvas aria-live from "assertive" to "polite"
- [ ] [PO-046] Add aria-label or visually-hidden label to palette filter input
- [ ] [PO-047] Show topbar icon-button tooltips on :focus-visible not just :hover

Nitpick (fix when convenient)
- [ ] [PO-048] Add clear button to palette search input
- [ ] [PO-049] Standardize "Open Editor" to "Edit" across expanded row and action column
- [ ] [PO-050] Suppress "0" badge on Archived tab when count is 0
- [ ] [PO-051] Replace zap icon in NODES column with graph/node icon
- [ ] [PO-052] Increase topbar breadcrumb text contrast to ≥4.5:1
- [ ] [PO-053] Fix timeAgo formatter to always include unit; handle future timestamps
```

---

## Lighthouse Summary

| Category | Score | Status |
|----------|-------|--------|
| Performance | N/A | — |
| Accessibility | 89 | ⚠️ 50–89 |
| Best Practices | 100 | ✅ ≥90 |
| SEO | 60 | ⚠️ 50–89 |

Key Lighthouse flags:
1. **Missing `<main>` landmark** — screen reader users cannot skip to primary content (a11y impact: high)
2. **Buttons without accessible names** — row-expand chevrons and icon-only buttons fail accessible name check (a11y impact: high)
3. **Low contrast text** — multiple text elements below WCAG AA 4.5:1 across sidebar, table headers, and stat cards (a11y impact: medium)

---

## Console & Network Issues

### Console Errors / Warnings

| Level | Count | Top Messages |
|-------|-------|--------------|
| Error | 2 | `Warning: Each child in a list should have a unique "key" prop. Check the render method of WorkflowsPage`; `404 GET /favicon.ico` |
| Warning | 1 | `Warning: A form field element should have an id or name attribute` |

### Network Issues

| Type | Count | Details |
|------|-------|---------|
| 4xx errors | 1 | GET /favicon.ico → 404 |
| 5xx errors | 0 | None |
| Slow requests (>2s) | 0 | None observed |

---

## Comparison to Previous Review

```
Previous review date: 2026-04-03 (enterprise-buyer perspective only)
Previous verdict:     FAIL (3 critical, 9 major, 6 minor, 3 nitpick = 21 findings)

Fixed since last review:   0 findings
New since last review:    32 findings (PO-007 through PO-053, net after deduplication)
Remaining from last:      21 findings (PO-001 through PO-006, PO-009–PO-021 carried forward and re-mapped)

Trend: STABLE — no fixes applied between same-day reviews.
       Expanding from 1 perspective to 8 revealed 32 additional findings.
       Accessibility and architectural concerns are now fully surfaced.
```

**Root cause grouping** — findings cluster into 5 systemic failure areas:

1. **Broken affordances** (PO-001, PO-002, PO-010, PO-028, PO-029) — stub buttons and non-interactive columns shipped to production UI with no backing implementation.
2. **Data integrity failures** (PO-005, PO-006, PO-009, PO-016) — null statuses despite existing execution records, wrong counts, placeholder seed content.
3. **Accessibility baseline not met** (PO-007, PO-008, PO-020–PO-024, PO-032, PO-045, PO-046) — no `<main>`, contrast failures throughout, missing ARIA roles, keyboard dead-ends.
4. **Editor UX incomplete** (PO-013, PO-014, PO-019, PO-027, PO-030, PO-033, PO-042) — config panels empty, no unsaved changes guard, parallel-branch layout bugs, no onboarding.
5. **Interaction model confusion** (PO-011, PO-012, PO-015, PO-031) — redundant filter surfaces, competing click targets, UUID exposed in breadcrumb, incorrect placeholder copy.

---

## Machine-Readable Output

A JSON version of all findings has been saved to:

```
.omniprod/findings/2026-04-03-workflows.json
```

Schema: `{ review_id, url, date, page_name, overall_verdict, perspectives, findings, lighthouse, capture_stats, stats, delta }`
