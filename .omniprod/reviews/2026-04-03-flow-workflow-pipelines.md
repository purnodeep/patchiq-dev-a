# Flow Review: Workflow Create/Edit/Publish Pipelines

- **Date**: 2026-04-03
- **Reviewer**: product-observer (manual flow test)
- **Flows Tested**: 2
- **Pages Visited**: /workflows, /workflows/new, /workflows/{id}/edit

---

## Flow 1: Create New Workflow → Save → Publish → Verify

| Step | Action | Result |
|------|--------|--------|
| 1 | Navigate to /workflows | List page loads, 3 workflows, 0 published |
| 2 | Click "New Workflow" | Editor opens with breadcrumb "Workflows / New Workflow", 1 default trigger node |
| 3 | Set name "Emergency Patch Deploy" | Name field accepts input, breadcrumb updates |
| 4 | Load template "Critical Patch Fast-Track" | 4 nodes + 3 edges loaded, status bar shows "Valid" |
| 5 | Click Save | Workflow created, URL changes to /workflows/{uuid}/edit, Publish button appears |
| 6 | Click Publish | Toast: "Workflow published", Publish button disappears |
| 7 | Navigate to /workflows | "Emergency Patch Deploy" shows PUBLISHED, counts: 4 Total / 1 Published / 3 Draft |

**Verdict: PASS** — Create → save → publish works end-to-end.

---

## Flow 2: Edit Existing Workflow → Modify → Save → Publish

| Step | Action | Result |
|------|--------|--------|
| 1 | Click Edit on "Critical Patch Deployment" | Editor opens with 7 nodes, 6 connections, breadcrumb shows name |
| 2 | Rename to "Critical Patch Deployment v2" | Name field updates, breadcrumb updates |
| 3 | Click Save | "Saved" indicator appears in toolbar and status bar |
| 4 | Click Publish | **FAIL** — Toast shows detailed validation errors (5 node config violations) |
| 5 | Navigate to /workflows | Workflow remains DRAFT, name updated to "v2" |

**Verdict: CONDITIONAL PASS** — Edit and save work. Publish fails due to missing node config forms (PO-014, deferred). Error messages are now detailed and actionable.

---

## Cross-Page Assertions

| # | Assertion | Result | Details |
|---|-----------|--------|---------|
| 1 | Workflow name persists from editor to list | PASS | "Emergency Patch Deploy" shows correctly in both |
| 2 | Published status reflects in list after editor publish | PASS | Status changes from DRAFT to PUBLISHED |
| 3 | Stat card counts match filter tab counts | PASS | Both show 4 Total, 1 Published, 3 Draft |
| 4 | Run button appears only for published workflows | PASS | Only "Emergency Patch Deploy" has Run button |
| 5 | Breadcrumb shows workflow name, not UUID | PASS (editor) / FAIL (topbar) | Editor breadcrumb correct; TopBar still shows raw UUID |

---

## Findings

### FL-001: TopBar breadcrumb shows raw UUID instead of workflow name
- **Severity**: major
- **Element**: TopBar breadcrumb (banner area), editor page
- **Observation**: The TopBar renders the URL segment as-is (e.g., `bb000000-0000-0000-0000-000000000001`). The editor's own breadcrumb nav shows the correct workflow name. Two competing breadcrumbs exist.
- **Suggestion**: Hide the TopBar breadcrumb when on the editor page (editor has its own), or resolve the workflow name in TopBar.

### FL-002: Publish fails for UI-created workflows with no node config forms
- **Severity**: major (known, deferred as PO-014)
- **Element**: Publish button, editor page
- **Observation**: Workflows created/edited through the UI have empty node configs because the editor has no config forms for any node type. The backend validator correctly rejects these. Only template-based workflows (which have pre-configured nodes) can be published.
- **Suggestion**: Implement node config forms (PO-014) or skip config validation for empty configs to allow basic publishing.

### FL-003: React duplicate key console error on template load
- **Severity**: minor
- **Element**: Console, editor page after loading template
- **Observation**: Two "Encountered two children with the same key" errors fire when loading a template. Likely ReactFlow node IDs colliding during template swap.
- **Suggestion**: Clear existing nodes before setting template nodes, or generate fresh UUIDs for template nodes.

### FL-004: Stat cards still show as clickable buttons despite being display-only
- **Severity**: minor
- **Element**: Stat card row, list page
- **Observation**: StatCard components render as `<button>` elements even though click handlers were removed. Screen readers announce them as buttons, but clicking does nothing.
- **Suggestion**: Change StatCard to render as `<div>` when no onClick is provided.

---

## Screenshots

| Step | Flow | Screenshot |
|------|------|------------|
| 1 | Flow 1 | flow-1-list.png |
| 2 | Flow 1 | flow-2-new-editor.png |
| 3 | Flow 1 | flow-3-saved.png |
| 4 | Flow 1 | flow-4-published.png |
| 5 | Flow 1 | flow-5-list-published.png |
| 1 | Flow 2 | flow-1-editor.png |
| 2 | Flow 2 | flow-2-saved.png |
| 3 | Flow 2 | flow-3-publish-failed-detailed.png |
| 4 | Flow 2 | flow-4-list-final.png |

---

## Summary

| Metric | Value |
|--------|-------|
| Flows tested | 2 |
| Flow 1 (Create → Publish) | PASS |
| Flow 2 (Edit → Publish) | CONDITIONAL PASS |
| Cross-page assertions | 4 PASS, 1 PARTIAL |
| Findings | 0 critical, 2 major, 2 minor |
| Console errors | 2 (duplicate React keys) |
