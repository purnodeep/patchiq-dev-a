# Workflows Page Product Fixes

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix all 53 product review findings (10 critical, 23 major, 14 minor, 6 nitpick) from the workflows page OmniProd review.

**Architecture:** Frontend-heavy fixes across the workflows list page, editor, inline editor, and shared layout. One backend change (add `status` filter to ListWorkflows SQL). Seed data updates for demo readiness. CSS token adjustments for WCAG AA contrast compliance.

**Tech Stack:** React 19, TypeScript, TanStack Table/Query, Lucide icons, sqlc/PostgreSQL, CSS custom properties

---

## File Map

| File | Changes |
|------|---------|
| `web/src/pages/workflows/index.tsx` | PO-001/002/003/006/007/010/011/012/017/018/022/023/024/034/036/037/038/043/044/049/050/053 |
| `web/src/pages/workflows/editor.tsx` | PO-004/015/025/026/027/031 |
| `web/src/pages/workflows/workflow-inline-editor.tsx` | PO-013/033 |
| `web/src/app/layout/AppLayout.tsx` | PO-008 |
| `web/src/lib/time.ts` | PO-053 |
| `web/src/components/FilterBar.tsx` | PO-024 |
| `web/src/flows/policy-workflow/hooks/use-workflows.ts` | PO-006 (add status param) |
| `packages/ui/src/theme/tokens.css` | PO-020/021/032/052 |
| `internal/server/store/queries/workflows.sql` | PO-006 (add status filter) |
| `internal/server/store/sqlcgen/workflows.sql.go` | Regenerated via `make sqlc` |
| `internal/server/api/v1/workflows.go` | PO-006 (pass status param) |
| `scripts/seed-dev.sql` | PO-009/016 |

---

## Chunk 1: Parallel Frontend Tasks

These 7 tasks are independent and can be dispatched to parallel subagents.

### Task 1: Quick A11Y & HTML Fixes (index.tsx)

Fixes: PO-003, PO-007, PO-022, PO-023, PO-024, PO-034, PO-050, PO-053

**Files:**
- Modify: `web/src/pages/workflows/index.tsx`
- Modify: `web/src/lib/time.ts`
- Modify: `web/src/components/FilterBar.tsx`

- [ ] **Step 1: Fix PO-003 — Add key prop to Fragment in row map**

In `web/src/pages/workflows/index.tsx:934-935`, the `table.getRowModel().rows.map()` wraps rows in a bare `<>` Fragment with no key. Change:

```tsx
// BEFORE (line 935):
<>

// AFTER:
<React.Fragment key={row.id}>
```

And change the closing tag on line 983 from `</>` to `</React.Fragment>`.

Add `React` to the import from 'react' on line 1 (it's not currently imported as a namespace).

Alternatively, since `React.Fragment` needs the namespace import, just use a `<div>` wrapper or use `import React from 'react'`. The simplest fix: add `Fragment` to the destructured import on line 1, then use `<Fragment key={row.id}>` and `</Fragment>`.

- [ ] **Step 2: Fix PO-007 — Add aria-label to expand/collapse chevron buttons**

In `web/src/pages/workflows/index.tsx`, the `expandCol` cell (lines 422-468): add `aria-label` to the button that toggles based on expanded state. The `row.original.name` is accessible from the cell render function.

```tsx
// Add to the <button> in expandCol cell:
aria-label={row.getIsExpanded() ? `Collapse ${row.original.name}` : `Expand ${row.original.name}`}
```

- [ ] **Step 3: Fix PO-022 — Add contextual aria-labels to action buttons**

In the `actions` column cell (lines 588-678), update the three action elements:

```tsx
// Edit link (line 593): change title to aria-label
aria-label={`Edit ${row.original.name}`}

// Duplicate button (line 620): add aria-label
aria-label={`Duplicate ${row.original.name}`}

// More actions button (line 648): add aria-label
aria-label={`More actions for ${row.original.name}`}
```

- [ ] **Step 4: Fix PO-023 — Add ARIA roles to filter tabs**

In `web/src/pages/workflows/index.tsx`, the `<FilterBar>` section (lines 824-853). Add `role="tablist"` to the FilterBar wrapper, and pass `role="tab"` + `aria-selected` to each FilterPill.

In `web/src/components/FilterBar.tsx`:
- Add `role` and `aria-selected` as optional props to `FilterPillProps` interface
- Pass them through to the `<button>` element

Then in `index.tsx`:
```tsx
<FilterBar role="tablist">
  <FilterPill role="tab" aria-selected={statusFilter === 'all'} ... />
  ...
```

For FilterBar: add `role?: string` to `FilterBarProps`, pass to the inner `<div>`.

- [ ] **Step 5: Fix PO-024 — Add aria-label to search input**

In `web/src/components/FilterBar.tsx`, the `FilterSearch` component (line 123): add `aria-label` prop support and default it.

```tsx
// Add to FilterSearchProps:
'aria-label'?: string;

// In the <input>:
aria-label={props['aria-label'] ?? placeholder}
```

- [ ] **Step 6: Fix PO-034 — Change table headers from UPPERCASE to Title Case**

In `web/src/pages/workflows/index.tsx`, the `<th>` style (line 883): remove `textTransform: 'uppercase'`.

The column `header` strings are already Title Case ('Name', 'Status', 'Nodes', etc.), so removing the CSS uppercase is sufficient.

- [ ] **Step 7: Fix PO-050 — Suppress "0" badge on Archived tab**

In the Archived FilterPill (line 845-850), change `count={archivedCount}` to:
```tsx
count={archivedCount || undefined}
```

- [ ] **Step 8: Fix PO-053 — Fix timeAgo to always include unit**

In `web/src/lib/time.ts`, the timeAgo function returns `${mins}m ago` but should say `${mins} min ago` for clarity. More importantly, the finding says "46 ago" with missing unit — this happens because `hours / 24` can produce 0 when hours < 24 but the hours branch wasn't hit. Actually, looking at the code, the issue is fine for mins/hours/days. The real bug: if `hours >= 24`, it shows `${days}d ago` — but if days is very large, it truncates. The specific bug "46 ago" suggests a missing unit. Let me trace: `hours=46`, `hours < 24` is false, so `Math.floor(46/24) = 1` → shows `1d ago`. Actually the "46 ago" might be `46` minutes showing as `46m ago` with the "m" being cut off in the UI due to container truncation.

But to be safe, make units more explicit:
```ts
if (mins < 60) return `${mins} min ago`;
const hours = Math.floor(mins / 60);
if (hours < 24) return `${hours}h ago`;
const days = Math.floor(hours / 24);
return `${days}d ago`;
```

Also handle future timestamps gracefully:
```ts
if (diff < 0) return 'just now';
```

---

### Task 2: AppLayout + Main Landmark (PO-008)

**Files:**
- Modify: `web/src/app/layout/AppLayout.tsx`

- [ ] **Step 1: Replace content div with `<main>`**

In `web/src/app/layout/AppLayout.tsx:39`, change:
```tsx
// BEFORE:
<div style={{ flex: 1, overflowY: 'auto', background: 'var(--bg-page)' }}>

// AFTER:
<main style={{ flex: 1, overflowY: 'auto', background: 'var(--bg-page)' }}>
```

And on line 41, change `</div>` to `</main>`.

---

### Task 3: Editor Fixes (PO-004, PO-015, PO-025, PO-026, PO-027, PO-031)

**Files:**
- Modify: `web/src/pages/workflows/editor.tsx`

- [ ] **Step 1: Fix PO-004 — Add id and name to workflow name input**

In `editor.tsx:283-299`, add `id` and `name` attributes to the input:
```tsx
<input
  id="workflow-name"
  name="workflow-name"
  value={name}
  ...
```

- [ ] **Step 2: Fix PO-031 — Fix placeholder text**

In `editor.tsx:286`, change placeholder from `"Workflow name…"` to `"Workflow name"` (remove trailing ellipsis since it's a text input, not a search).

In `editor.tsx:319`, change `"Load template…"` to `"Load template"` for consistency.

- [ ] **Step 3: Fix PO-015 — Replace UUID with workflow name in breadcrumb and document title**

Add a `useEffect` to set `document.title` when workflow name changes:
```tsx
useEffect(() => {
  document.title = name ? `${name} — Edit Workflow` : 'New Workflow';
  return () => { document.title = 'PatchIQ'; };
}, [name]);
```

For breadcrumb: if the app uses a TopBar breadcrumb that reads from the URL, the editor page should communicate the workflow name. Check if TopBar renders breadcrumbs from route — if it does, this needs the workflow name passed via context or route state. If TopBar just shows the current path, the fix is to render a custom breadcrumb inside the editor toolbar.

Add a breadcrumb above the toolbar in `editor.tsx`:
```tsx
// After the opening div of EditorInner's return (line 261):
<nav aria-label="Breadcrumb" style={{ padding: '8px 16px 0', fontSize: 12, color: 'var(--text-muted)' }}>
  <Link to="/workflows" style={{ color: 'var(--text-muted)', textDecoration: 'none' }}>Workflows</Link>
  <span style={{ margin: '0 6px', color: 'var(--text-faint)' }}>/</span>
  <span style={{ color: 'var(--text-secondary)' }}>{name || 'New Workflow'}</span>
</nav>
```

- [ ] **Step 4: Fix PO-025 — Add loading state and error handler to Publish button**

In `editor.tsx:408-430`, the publish button already has `isPending` state and disabled styling. It needs:
1. Pre-flight validation (check nodes.length >= 2)
2. Success/error toasts

```tsx
// Replace the onClick handler:
onClick={async () => {
  if (nodes.length < 2) {
    toast.error('Add at least 2 nodes before publishing');
    return;
  }
  try {
    await publishWorkflow.mutateAsync();
    toast.success('Workflow published');
  } catch (err) {
    toast.error(`Publish failed: ${err instanceof Error ? err.message : 'Unknown error'}`);
  }
}}
```

Add `import { toast } from 'sonner';` to the imports.

Note: `usePublishWorkflow` already has `onSuccess` and `onError` with toasts in `use-workflows.ts:80-86`. So just adding the pre-flight validation is enough. Change `onClick={() => publishWorkflow.mutate()}` to add the node count check.

- [ ] **Step 5: Fix PO-027 — Add unsaved changes warning on navigate away**

Track dirty state and add beforeunload handler:
```tsx
const [isDirty, setIsDirty] = useState(false);

// Mark dirty on any node/edge/name/description change:
useEffect(() => {
  if (!isEditMode) return;
  setIsDirty(true);
}, [nodes, edges, name, description]);

// Reset dirty on save:
// In handleSave success path, add: setIsDirty(false);

// beforeunload handler:
useEffect(() => {
  if (!isDirty) return;
  const handler = (e: BeforeUnloadEvent) => {
    e.preventDefault();
  };
  window.addEventListener('beforeunload', handler);
  return () => window.removeEventListener('beforeunload', handler);
}, [isDirty]);
```

- [ ] **Step 6: Fix PO-041 — Replace outline:none with focus-visible ring on editor name input**

In `editor.tsx:296`, change `outline: 'none'` to remove it and add a focus style. Since this uses inline styles, we can't use `:focus-visible` directly. Replace `outline: 'none'` with `outline: 'none'` but add an `onFocus`/`onBlur` handler pair, or better: use a CSS class.

Simplest approach — just remove `outline: 'none'` so the browser default focus ring shows, or replace with:
```tsx
outline: '2px solid transparent',
```
And add `onFocus`/`onBlur` to toggle outline to accent:
```tsx
onFocus={(e) => { e.currentTarget.style.outline = '2px solid var(--accent)'; e.currentTarget.style.outlineOffset = '-1px'; }}
onBlur={(e) => { e.currentTarget.style.outline = '2px solid transparent'; }}
```

---

### Task 4: Action Buttons — More Actions, Duplicate, Run Now (PO-001, PO-002, PO-010)

**Files:**
- Modify: `web/src/pages/workflows/index.tsx`
- Modify: `web/src/flows/policy-workflow/hooks/use-workflows.ts` (for duplicate)

- [ ] **Step 1: Implement PO-001 — More Actions dropdown**

Replace the More Actions button stub with a functioning dropdown. Use a simple state-driven dropdown (no external library needed):

```tsx
// Add state at top of actions cell or use a small MoreActionsDropdown component:
function MoreActionsDropdown({ workflow, onDelete }: { workflow: WorkflowListItem; onDelete: (id: string) => void }) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [open]);

  return (
    <div ref={ref} style={{ position: 'relative' }}>
      <button ... onClick={() => setOpen(!open)} aria-label={`More actions for ${workflow.name}`}>
        <MoreHorizontal ... />
      </button>
      {open && (
        <div style={{
          position: 'absolute', right: 0, top: '100%', marginTop: 4, zIndex: 50,
          background: 'var(--bg-elevated)', border: '1px solid var(--border)',
          borderRadius: 8, boxShadow: 'var(--shadow-lg)', minWidth: 160, padding: 4,
        }}>
          {workflow.current_status !== 'archived' && (
            <button onClick={() => { /* archive logic */ setOpen(false); }} style={menuItemStyle}>
              <Archive style={{ width: 12, height: 12 }} /> Archive
            </button>
          )}
          <button onClick={() => {
            if (confirm(`Delete workflow "${workflow.name}"?`)) onDelete(workflow.id);
            setOpen(false);
          }} style={{ ...menuItemStyle, color: 'var(--signal-critical)' }}>
            <Trash2 style={{ width: 12, height: 12 }} /> Delete
          </button>
        </div>
      )}
    </div>
  );
}
```

Wire `onDelete` to `useDeleteWorkflow` from the hooks.

- [ ] **Step 2: Implement PO-002 — Wire Duplicate button**

The backend has no clone endpoint. Implement client-side duplication: fetch the full workflow detail, then create a new workflow with the same nodes/edges but a new name.

In the actions column, add an `onClick` handler for the Duplicate button:
```tsx
onClick={async () => {
  try {
    // Fetch full workflow detail
    const detail = await fetchJSON(`/api/v1/workflows/${row.original.id}`);
    // Create duplicate with modified name
    await createWorkflow.mutateAsync({
      name: `${row.original.name} (Copy)`,
      description: detail.description,
      nodes: detail.nodes.map(n => ({
        id: crypto.randomUUID(),
        node_type: n.node_type,
        label: n.label,
        position_x: n.position_x,
        position_y: n.position_y,
        config: n.config,
      })),
      edges: /* remap source/target IDs using new node IDs */,
    });
    toast.success('Workflow duplicated');
  } catch {
    toast.error('Failed to duplicate workflow');
  }
}}
```

This is complex due to edge ID remapping. Simpler approach: navigate to `/workflows/new` with the workflow ID as a query param, and let the editor page handle loading it as a template. But that's a different UX.

Better approach: use `useCreateWorkflow` + `useWorkflow` hooks. Add a `handleDuplicate` function at the page level.

- [ ] **Step 3: Implement PO-010 — Add Run Now button for published workflows**

In the actions column, add a "Run" button that appears only for published workflows:

```tsx
{row.original.current_status === 'published' && (
  <button
    type="button"
    aria-label={`Run ${row.original.name}`}
    onClick={(e) => {
      e.stopPropagation();
      executeWorkflow.mutate(row.original.id);
    }}
    style={{ /* same 26x26 action button style */ }}
  >
    <Play style={{ width: 11, height: 11 }} />
  </button>
)}
```

Need to add `useExecuteWorkflow` import and usage. The hook from `use-workflow-executions.ts` takes a workflowId. Use it at page level with dynamic ID:

```tsx
import { useExecuteWorkflow } from '../../flows/policy-workflow/hooks/use-workflow-executions';
// In component:
const executeWorkflow = useExecuteWorkflow(''); // We'll need to handle dynamic ID
```

Actually, `useExecuteWorkflow` takes a fixed ID. For the list page, create a mutation directly:
```tsx
const executeWorkflowMutation = useMutation({
  mutationFn: (id: string) => fetchVoid(`/api/v1/workflows/${id}/execute`, { method: 'POST' }),
  onSuccess: () => { toast.success('Workflow execution started'); queryClient.invalidateQueries({ queryKey: ['workflows'] }); },
  onError: () => { toast.error('Failed to start workflow execution'); },
});
```

---

### Task 5: Server-Side Filtering & Counts (PO-006, PO-011, PO-017)

**Files:**
- Modify: `internal/server/store/queries/workflows.sql`
- Run: `make sqlc` to regenerate
- Modify: `internal/server/api/v1/workflows.go`
- Modify: `web/src/flows/policy-workflow/hooks/use-workflows.ts`
- Modify: `web/src/pages/workflows/index.tsx`

- [ ] **Step 1: Add status filter to ListWorkflows SQL query**

In `internal/server/store/queries/workflows.sql`, add a `status_filter` param to `ListWorkflows`:

```sql
-- name: ListWorkflows :many
SELECT w.*,
       COALESCE(v.version, 0)::int AS current_version,
       COALESCE(v.status, 'draft') AS current_status,
       (SELECT count(*) FROM workflow_nodes wn WHERE wn.version_id = v.id AND wn.tenant_id = @tenant_id)::int AS node_count,
       (SELECT count(*) FROM workflow_executions we WHERE we.workflow_id = w.id AND we.tenant_id = @tenant_id)::int AS total_runs,
       COALESCE((SELECT we2.status FROM workflow_executions we2 WHERE we2.workflow_id = w.id AND we2.tenant_id = @tenant_id ORDER BY we2.created_at DESC LIMIT 1), '')::text AS last_run_status,
       (SELECT we3.created_at FROM workflow_executions we3 WHERE we3.workflow_id = w.id AND we3.tenant_id = @tenant_id ORDER BY we3.created_at DESC LIMIT 1) AS last_run_at
FROM workflows w
LEFT JOIN workflow_versions v ON v.workflow_id = w.id
    AND v.version = (
        SELECT max(v2.version) FROM workflow_versions v2
        WHERE v2.workflow_id = w.id AND v2.tenant_id = @tenant_id
    )
WHERE w.tenant_id = @tenant_id
  AND w.deleted_at IS NULL
  AND (@search::text = '' OR w.name ILIKE '%' || @search || '%' ESCAPE '\')
  AND (@status_filter::text = '' OR COALESCE(v.status, 'draft') = @status_filter)
  AND (
    @cursor_created_at::timestamptz IS NULL
    OR (w.created_at, w.id) > (@cursor_created_at, @cursor_id::uuid)
  )
ORDER BY w.created_at, w.id
LIMIT @page_limit;
```

Also add status filter to CountWorkflows:
```sql
-- name: CountWorkflows :one
SELECT count(*) FROM workflows w
LEFT JOIN workflow_versions v ON v.workflow_id = w.id
    AND v.version = (
        SELECT max(v2.version) FROM workflow_versions v2
        WHERE v2.workflow_id = w.id AND v2.tenant_id = @tenant_id
    )
WHERE w.tenant_id = @tenant_id
  AND w.deleted_at IS NULL
  AND (@search::text = '' OR w.name ILIKE '%' || @search || '%' ESCAPE '\')
  AND (@status_filter::text = '' OR COALESCE(v.status, 'draft') = @status_filter);
```

Add a new query for status counts:
```sql
-- name: CountWorkflowsByStatus :many
SELECT COALESCE(v.status, 'draft') AS status, count(*)::int AS count
FROM workflows w
LEFT JOIN workflow_versions v ON v.workflow_id = w.id
    AND v.version = (
        SELECT max(v2.version) FROM workflow_versions v2
        WHERE v2.workflow_id = w.id AND v2.tenant_id = @tenant_id
    )
WHERE w.tenant_id = @tenant_id
  AND w.deleted_at IS NULL
GROUP BY COALESCE(v.status, 'draft');
```

- [ ] **Step 2: Run `make sqlc` to regenerate Go code**

- [ ] **Step 3: Update workflows.go to pass status filter**

In `internal/server/api/v1/workflows.go`, the `List` handler (line 83):
```go
statusFilter := r.URL.Query().Get("status")
```

Pass it to `ListWorkflowsParams`:
```go
StatusFilter: statusFilter,
```

And to `CountWorkflowsParams`.

Add a new endpoint or extend List response to include per-status counts using the new `CountWorkflowsByStatus` query.

- [ ] **Step 4: Update frontend hooks to pass status filter**

In `web/src/flows/policy-workflow/hooks/use-workflows.ts`, add `status` and `search` params:
```tsx
export function useWorkflows(params?: { cursor?: string; limit?: number; status?: string; search?: string }) {
  const query = new URLSearchParams();
  if (params?.cursor) query.set('cursor', params.cursor);
  if (params?.limit) query.set('limit', String(params.limit));
  if (params?.status) query.set('status', params.status);
  if (params?.search) query.set('search', params.search);
  ...
```

- [ ] **Step 5: Update index.tsx to use server-side filtering**

Pass `statusFilter` and `search` to the `useWorkflows` hook instead of filtering client-side. Remove the client-side `filtered` useMemo. Remove stat card click handlers (PO-011 — single filter surface).

```tsx
const { data, isLoading, isError, refetch } = useWorkflows({
  cursor: currentCursor,
  limit: 25,
  status: statusFilter === 'all' ? undefined : statusFilter,
  search: search || undefined,
});
```

- [ ] **Step 6: Fix PO-011 — Remove duplicate filter surface**

Remove onClick handlers from StatCard components. Make them display-only. Remove the `cursor: 'pointer'` and interactive styling. Keep filter pills as the sole filter mechanism.

- [ ] **Step 7: Fix PO-017 — Reset pagination on search/filter change**

```tsx
// Reset cursors when search or statusFilter changes:
useEffect(() => {
  setCursors([]);
}, [statusFilter, search]);
```

---

### Task 6: Inline Editor Fixes (PO-012, PO-013, PO-033)

**Files:**
- Modify: `web/src/pages/workflows/index.tsx`
- Modify: `web/src/pages/workflows/workflow-inline-editor.tsx`

- [ ] **Step 1: Fix PO-012 — Separate row click (expand) from inline editor trigger**

Currently, clicking a row sets `selectedWorkflow` which opens the inline editor. The expand chevron toggles row expansion. These compete.

Change row click to only toggle expansion (like the chevron does), and use an explicit "View" or "Inspect" button to open the inline editor:

In `index.tsx:943-947`, change the `onClick` handler:
```tsx
onClick={() => row.toggleExpanded()}
```

Add an "Inspect" or "Preview" button in the expanded row's Quick Actions section (or in the actions column) that sets `selectedWorkflow`.

- [ ] **Step 2: Fix PO-033 — Inline editor no-selection empty state**

In `workflow-inline-editor.tsx:587-590`, the no-selection state currently says "Click a node to inspect its configuration." This is fine but could be more helpful. Update:

```tsx
<div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 8, padding: '24px 12px', textAlign: 'center' }}>
  <CircleCheckBig style={{ width: 20, height: 20, color: 'var(--text-faint)' }} />
  <p style={{ fontSize: 11, color: 'var(--text-muted)', margin: 0 }}>
    Select a node to view its configuration
  </p>
</div>
```

- [ ] **Step 3: Fix PO-013 — Config panel data binding check**

The inline editor's config panel reads `selectedNode.config` directly. If config is base64-encoded (common from the API), the panel correctly decodes it (lines 561-573). The finding says "Enter Configuration placeholder instead of properties" — this happens when the node has no config or empty config. The fix is to show a meaningful message:

After the config field rendering (line 583), add a fallback for empty config:
```tsx
{/* After the config entries, if no fields rendered: */}
{(!selectedNode.config || Object.keys(parsed ?? {}).length === 0) && (
  <p style={{ fontSize: 11, color: 'var(--text-muted)', fontStyle: 'italic' }}>
    No configuration set for this node
  </p>
)}
```

---

### Task 7: Contrast & Visual Polish (PO-020, PO-021, PO-032, PO-038, PO-041, PO-043, PO-044, PO-049, PO-051, PO-052)

**Files:**
- Modify: `packages/ui/src/theme/tokens.css`
- Modify: `web/src/pages/workflows/index.tsx`
- Modify: `web/src/pages/workflows/workflow-node-styles.ts` (PO-051)

- [ ] **Step 1: Fix PO-020 — Sidebar nav label contrast**

In `packages/ui/src/theme/tokens.css:103`:
```css
/* BEFORE: */
--nav-label: #525252;
/* AFTER (4.5:1+ against #000000 bg): */
--nav-label: #8a8a8a;
```

And `--nav-item-color` on line 101:
```css
/* BEFORE: */
--nav-item-color: #737373;
/* AFTER: */
--nav-item-color: #8a8a8a;
```

- [ ] **Step 2: Fix PO-021 — Stat card label, table header, row text contrast**

In `packages/ui/src/theme/tokens.css`:
```css
/* --text-muted is used for stat card labels, table headers, row descriptions */
/* BEFORE: */
--text-muted: #737373;
/* AFTER (4.58:1 against #000000): */
--text-muted: #787878;
```

This is a global change. `#787878` on `#000000` = 4.56:1 which passes AA. `#737373` on `#000000` = 4.19:1 which fails. Going to `#7a7a7a` gives 4.67:1.

```css
--text-muted: #7a7a7a;
```

- [ ] **Step 3: Fix PO-032/PO-052 — Topbar breadcrumb contrast**

The topbar uses `--text-muted` for breadcrumb text, which is fixed by Step 2.

- [ ] **Step 4: Fix PO-038 — Add absolute date tooltips to relative timestamps**

In `index.tsx`, for the `last_run` and `updated` columns, wrap the time text in a `<span title={isoDate}>`:

```tsx
// last_run column (line 558):
<span title={at ? new Date(at).toLocaleString() : undefined}
  style={{ fontFamily: 'var(--font-mono)', fontSize: 11, color: 'var(--text-muted)' }}>
  {at ? timeAgo(at) : 'Never'}
</span>

// updated column (line 576):
<span title={val ? new Date(val).toLocaleString() : undefined}
  style={{ ... }}>
  {val ? timeAgo(val) : '—'}
</span>
```

- [ ] **Step 5: Fix PO-043 — Add unit context to NODES column**

In `index.tsx`, the nodes column cell (line 517):
```tsx
// BEFORE:
{count}
// AFTER:
{count} <span style={{ fontSize: 10, color: 'var(--text-faint)' }}>nodes</span>
```

Or simpler: just show as `{count}` with a column header tooltip.

- [ ] **Step 6: Fix PO-044 — RUNS column sub-label**

In the runs column cell, add "(all time)" as a tooltip on the column header or a small sub-label:
```tsx
// Column header:
header: () => <span title="Total executions (all time)">Runs</span>,
```

- [ ] **Step 7: Fix PO-049 — Standardize "Open Editor" to "Edit"**

In `index.tsx:362`, the expanded row CTA says "Open Editor". Change to "Edit":
```tsx
// BEFORE:
Open Editor
// AFTER:
Edit
```

- [ ] **Step 8: Fix PO-051 — Replace Zap icon in NODES column with graph/node icon**

In `index.tsx:514`, the Zap icon is used for node count. Import `GitFork` or `Network` from lucide-react and replace:
```tsx
// BEFORE:
<Zap style={{ ... }} />
// AFTER:
<GitBranch style={{ ... }} />
```

`GitBranch` is already imported on line 14. Use it instead of `Zap` for the nodes column.

---

## Chunk 2: Backend, Seed Data, and Complex Features

### Task 8: Seed Data Fixes (PO-009, PO-016)

**Files:**
- Modify: `scripts/seed-dev.sql`

- [ ] **Step 1: Fix PO-016 — Rename "Untitled Workflow"**

Find and replace any workflow with name "Untitled Workflow" in `scripts/seed-dev.sql` with a meaningful name like "Automated Security Patching" or similar.

- [ ] **Step 2: Fix PO-009 — Verify published workflows exist in seed data**

The seed data already has one PUBLISHED workflow ("Critical Patch Deployment") with 5 executions. If the review shows "0 Published", this may be because:
1. Seed data wasn't loaded after recent changes
2. The stat card counts are from client-side filtering of the current page only (fixed by Task 5)

Verify the seed data is correct. Add a second published workflow for better demo coverage:
```sql
-- Add a "Compliance Scan Automation" workflow in PUBLISHED status with nodes and edges
```

- [ ] **Step 3: Fix stat card green highlight on zero value**

In `index.tsx`, the Published stat card always uses `valueColor="var(--accent)"` even when count is 0. Fix:
```tsx
<StatCard
  label="Published"
  value={publishedCount}
  valueColor={publishedCount > 0 ? 'var(--accent)' : undefined}
  ...
/>
```

---

### Task 9: Empty State & UX Polish (PO-018, PO-036, PO-037, PO-042, PO-045, PO-046, PO-047, PO-048)

**Files:**
- Modify: `web/src/pages/workflows/index.tsx`
- Modify: `web/src/pages/workflows/editor.tsx`

- [ ] **Step 1: Fix PO-018 — Add "Clear filters" to empty filtered state**

In `index.tsx:904-917`, the empty state when filters are active shows "No workflows match your filters" but no action button. Add:
```tsx
<button
  onClick={() => { setStatusFilter('all'); setSearch(''); }}
  style={{
    marginTop: 8, padding: '5px 12px', borderRadius: 6,
    border: '1px solid var(--border)', background: 'none',
    color: 'var(--text-secondary)', fontSize: 12, cursor: 'pointer',
  }}
>
  Clear filters
</button>
```

- [ ] **Step 2: Fix PO-036 — Add "Showing X of Y" indicator**

Below the filter bar, add:
```tsx
{!isLoading && statusFilter !== 'all' && (
  <span style={{ fontSize: 11, color: 'var(--text-muted)', fontFamily: 'var(--font-mono)' }}>
    Showing {filtered.length} of {totalCount} workflows
  </span>
)}
```

- [ ] **Step 3: Fix PO-037 — Add total count to pagination**

In `DataTablePagination`, if it supports a `totalLabel` or similar, add it. Otherwise, add a span next to the pagination component:
```tsx
{!isLoading && (
  <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
    <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>
      {totalCount} total
    </span>
    <DataTablePagination ... />
  </div>
)}
```

- [ ] **Step 4: Fix PO-045 — Change canvas aria-live to "polite"**

Search for `aria-live="assertive"` in workflow canvas files and change to `"polite"`.

- [ ] **Step 5: Fix PO-046 — Add label to palette search input**

In the Palette component, find the search/filter input and add `aria-label="Filter node palette"`.

- [ ] **Step 6: Fix PO-047 — Show topbar icon-button tooltips on focus-visible**

This is a CSS fix. In the TopBar component, ensure tooltip trigger also fires on `:focus-visible`. If using a custom Tooltip, add the onFocus handler.

---

### Task 10: Complex Feature Work (PO-014, PO-019, PO-028, PO-029, PO-030, PO-026)

These items require significant implementation effort and should be scoped as separate issues rather than fixed inline.

**Recommendation:** Create GitHub issues for these and defer to a dedicated sprint:

- **PO-014** (Major): Node config forms for each node type — requires designing 8+ form schemas, validation, persistence
- **PO-019** (Major): Keyboard accessible node palette — requires implementing keyboard-based node placement
- **PO-028** (Major): Execution monitoring view — requires new page/panel with live status updates, per-node progress, log streaming
- **PO-029** (Major): Version history UI — requires new panel with diff view, rollback capability
- **PO-030** (Major): Inline canvas parallel-branch layout — requires elk.js integration or dagre configuration changes

**PO-026** (Major): Node config save feedback — the inline editor config panel is currently read-only. Adding save feedback requires first making it editable (which is part of PO-014).

---

## Execution Strategy

**Parallel Task Groups:**

| Group | Tasks | Can Run In Parallel |
|-------|-------|-------------------|
| A | Task 1, Task 2, Task 3, Task 7 | Yes (independent files) |
| B | Task 4, Task 6, Task 8, Task 9 | Yes (independent, but Task 4+6 both touch index.tsx — assign to same agent) |
| C | Task 5 | Sequential (backend → sqlc → frontend) |
| D | Task 10 | Deferred to separate issues |

**Recommended dispatch order:**
1. **Wave 1 (parallel):** Task 1, Task 2, Task 3, Task 7, Task 5 (backend portion)
2. **Wave 2 (parallel, after Wave 1):** Task 4+6 (combined, both touch index.tsx), Task 8, Task 9
3. **Wave 3:** Task 10 — create GitHub issues only
