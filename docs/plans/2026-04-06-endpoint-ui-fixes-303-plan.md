# Endpoint UI/UX Fixes (#303) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix 14 targeted UI/UX issues across Endpoints list, detail tabs, and Dashboard.

**Architecture:** All changes are frontend-only (web/ app). No backend or API changes. Each fix is isolated to 1-2 files max.

**Tech Stack:** React 19, TypeScript, inline styles (project pattern), TanStack Query hooks, @patchiq/ui components.

---

## File Map

| File | Changes |
|------|---------|
| `web/src/pages/endpoints/EndpointsPage.tsx` | A1 (memory fallback debug), A2 (patch card layout), B11 (derived status) |
| `web/src/pages/endpoints/EndpointDetailPage.tsx` | B4 (tag line, scan feedback, decommission wording), B11 (derived status) |
| `web/src/pages/endpoints/CreateTagDialog.tsx` | A3 (assign endpoints after creation) |
| `web/src/pages/endpoints/AssignTagsDialog.tsx` | A3 (minor — already works, verify) |
| `web/src/pages/endpoints/tabs/OverviewTab.tsx` | B2 (blast radius cleanup), B3 (risk breakdown visibility) |
| `web/src/pages/endpoints/tabs/SoftwareTab.tsx` | B1 (system/third-party filter) |
| `web/src/pages/endpoints/tabs/HardwareTab.tsx` | B5 (frequency fix, SoC label for Mac) |
| `web/src/pages/endpoints/tabs/PatchesTab.tsx` | B6 (add stat cards) |
| `web/src/pages/endpoints/tabs/VulnerabilitiesTab.tsx` | B7 (expand hero strip) |
| `web/src/pages/endpoints/tabs/ComplianceTab.tsx` | B8 (restyle evaluate, last scan, drill-down) |
| `web/src/pages/endpoints/tabs/HistoryTab.tsx` | B9 (human-readable IDs, dual timestamps) |
| `web/src/pages/endpoints/tabs/AuditTab.tsx` | B10 (truncated IDs, day range selector) |
| `web/src/pages/dashboard/StatCardsRow1.tsx` | C1 (% formatting audit) |

---

## Task 1: B11 — Derived status logic (shared helper)

This is a cross-cutting concern used by Tasks 2 and 3, so it goes first.

**Files:**
- Create: `web/src/pages/endpoints/deriveStatus.ts`

- [ ] **Step 1: Create the deriveStatus helper**

```typescript
// web/src/pages/endpoints/deriveStatus.ts
/**
 * Derives display status from last_seen timestamp.
 * Overrides backend status to prevent misleading "online" when agent is unreachable.
 */
export function deriveStatus(
  backendStatus: string,
  lastSeen: string | null | undefined,
): string {
  if (!lastSeen) return backendStatus;
  const diffMs = Date.now() - new Date(lastSeen).getTime();
  const diffMin = diffMs / 60_000;
  if (diffMin < 5) return 'online';
  if (diffMin < 30) return 'stale';
  return 'offline';
}
```

- [ ] **Step 2: Commit**

```
feat(web): add deriveStatus helper for endpoint status logic (#303 B11)
```

---

## Task 2: A1+A2+B11 — EndpointsPage.tsx fixes

**Files:**
- Modify: `web/src/pages/endpoints/EndpointsPage.tsx`

- [ ] **Step 1: Import deriveStatus and apply to ExpRow**

Add import at top:
```typescript
import { deriveStatus } from './deriveStatus';
```

In ExpRow, after the `const disk = ...` line, add:
```typescript
const displayStatus = deriveStatus(ep.status, ep.last_seen);
```

- [ ] **Step 2: Apply deriveStatus to all status rendering in the list**

Find all uses of `ep.status` in status badge/dot rendering and replace with `deriveStatus(ep.status, ep.last_seen)` where appropriate. The `dotColor`, `sLabel`, `sColor` functions already accept a string — just pass the derived status.

- [ ] **Step 3: Fix A1 — Ensure memory gauge shows "N/A" when data missing**

In ExpRow, the memory computation already exists with fallback. The fix: when `effectiveMt` is 0 (no data at all), show "N/A" instead of "0%":

Replace the `{p}%` display in the gauge loop with:
```typescript
{l === 'Memory' && effectiveMt === 0 ? 'N/A' : `${p}%`}
```

Actually, better approach — make all three metrics handle zero total:

```typescript
const metrics = [
  { l: 'CPU', p: cpu, hasData: true },
  { l: 'Memory', p: mem, hasData: effectiveMt > 0 },
  { l: 'Disk', p: disk, hasData: effectiveDt > 0 },
];
```

Then in the render loop, show "N/A" when `!hasData`.

- [ ] **Step 4: Fix A2 — Patch summary card layout**

The current code looks functional. Ensure `minWidth: 180` on the pending patches card and that zero counts render as "0" not blank. Wrap severity items with `flexWrap: 'wrap'`.

- [ ] **Step 5: Commit**

```
fix(web): endpoint list — derived status, memory gauge fallback, patch card layout (#303 A1,A2,B11)
```

---

## Task 3: B4+B11 — EndpointDetailPage.tsx fixes

**Files:**
- Modify: `web/src/pages/endpoints/EndpointDetailPage.tsx`

- [ ] **Step 1: Import and apply deriveStatus**

```typescript
import { deriveStatus } from './deriveStatus';
```

Replace `endpoint.status` in the STATUS_COLORS lookup with:
```typescript
const displayStatus = deriveStatus(endpoint.status, endpoint.last_seen);
const statusColors = STATUS_COLORS[displayStatus] ?? STATUS_COLORS.offline;
```

- [ ] **Step 2: Fix decommission wording**

Change "Decommission Endpoint" to "Delete Endpoint":
```typescript
<Trash2 style={{ width: 13, height: 13, marginRight: 8 }} />
Delete Endpoint
```

- [ ] **Step 3: Scan Now button — enhance feedback**

The button already has spinner + toast. Enhance by changing text to "Scanning..." when pending:
```typescript
{triggerScan.isPending ? 'Scanning...' : 'Scan Now'}
```
(Check if this is already the case — if so, skip.)

- [ ] **Step 4: Remove extra tag line/divider**

Search for any `<hr>`, `border-bottom`, or extra `<span>` dividers near the tags section in the header. Remove the unnecessary one. (The vertical divider between status and meta chips is intentional — look for an extra one.)

- [ ] **Step 5: Commit**

```
fix(web): endpoint detail — status logic, delete wording, scan feedback (#303 B4,B11)
```

---

## Task 4: A3 — Fix tags flow

**Files:**
- Modify: `web/src/pages/endpoints/CreateTagDialog.tsx`

- [ ] **Step 1: Fix create tag to assign endpoints after creation**

The current `handleSubmit` calls `createTag.mutate({ key, value, description })` but never uses `selectedIds`. After tag creation succeeds, assign selected endpoints:

```typescript
const handleSubmit = () => {
  createTag.mutate(
    { key: tagKey, value: tagValue, description: description || undefined },
    {
      onSuccess: (newTag) => {
        // Assign selected endpoints to the newly created tag
        if (selectedIds.size > 0 && newTag?.id) {
          assignTag.mutate({
            tagId: newTag.id,
            endpointIds: Array.from(selectedIds),
          });
        }
        setTagKey('');
        setTagValue('');
        setDescription('');
        setSelectedIds(new Set());
        setEpSearch('');
        onOpenChange(false);
      },
    },
  );
};
```

Add the `useAssignTag` import and hook:
```typescript
import { useCreateTag, useAssignTag } from '../../api/hooks/useTags';
// ...
const assignTag = useAssignTag();
```

- [ ] **Step 2: Ensure preSelectedEndpointId works**

The prop already initializes `selectedIds` — verify it works when opened from detail page. The current code does:
```typescript
preSelectedEndpointId ? new Set([preSelectedEndpointId]) : new Set()
```
This should work. If the dialog is reused without unmounting, the state may not reset. Add a useEffect:

```typescript
useEffect(() => {
  if (open && preSelectedEndpointId) {
    setSelectedIds(new Set([preSelectedEndpointId]));
  }
}, [open, preSelectedEndpointId]);
```

- [ ] **Step 3: Commit**

```
fix(web): tag creation assigns endpoints, pre-selection works (#303 A3)
```

---

## Task 5: B1 — Software tab System/Third-party filter

**Files:**
- Modify: `web/src/pages/endpoints/tabs/SoftwareTab.tsx`

- [ ] **Step 1: Add typeFilter state**

After the existing filter state declarations:
```typescript
const [typeFilter, setTypeFilter] = useState<'all' | 'system' | 'third-party'>('all');
```

- [ ] **Step 2: Add classification logic**

```typescript
const SYSTEM_SOURCES = new Set(['apt', 'yum', 'dnf', 'apk', 'dpkg', 'rpm', 'pacman', 'system', 'wua', 'hotfix', 'softwareupdate']);

function classifyPackage(source: string | null | undefined): 'system' | 'third-party' {
  if (!source) return 'third-party';
  return SYSTEM_SOURCES.has(source.toLowerCase()) ? 'system' : 'third-party';
}
```

- [ ] **Step 3: Apply filter in the filtered memo**

Add after the existing source/arch filters:
```typescript
.filter((p) => typeFilter === 'all' || classifyPackage(p.source) === typeFilter)
```

- [ ] **Step 4: Add filter toggle UI**

Add a segmented toggle before/after the existing source dropdown. Use inline styles matching the existing filter pattern:

```tsx
<div style={{ display: 'flex', gap: 2, background: 'var(--bg-inset)', borderRadius: 6, padding: 2 }}>
  {(['all', 'system', 'third-party'] as const).map((t) => (
    <button
      key={t}
      type="button"
      onClick={() => { setTypeFilter(t); setPage(0); }}
      style={{
        padding: '4px 10px',
        borderRadius: 4,
        border: 'none',
        fontSize: 11,
        fontFamily: 'var(--font-mono)',
        fontWeight: 500,
        cursor: 'pointer',
        background: typeFilter === t ? 'var(--bg-card)' : 'transparent',
        color: typeFilter === t ? 'var(--text-emphasis)' : 'var(--text-muted)',
        boxShadow: typeFilter === t ? 'var(--shadow-sm)' : 'none',
      }}
    >
      {t === 'all' ? 'All' : t === 'system' ? 'System' : 'Third-party'}
    </button>
  ))}
</div>
```

- [ ] **Step 5: Commit**

```
feat(web): software tab — add System/Third-party filter (#303 B1)
```

---

## Task 6: B2+B3 — Blast Radius + Risk Breakdown

**Files:**
- Modify: `web/src/pages/endpoints/tabs/OverviewTab.tsx`

- [ ] **Step 1: B2 — Add filter state and remove port/tag nodes**

In BlastRadiusGraph, add filter state:
```typescript
const [showCves, setShowCves] = useState(true);
const [showPatches, setShowPatches] = useState(true);
```

Remove the port node generation block (lines that push nodeType: 'port') and tag node generation block (lines that push nodeType: 'tag').

Filter remaining nodes:
```typescript
const visibleNodes = nodes.filter((n) => {
  if (n.nodeType === 'cve') return showCves;
  if (n.nodeType === 'patch') return showPatches;
  return false;
});
```

Use `visibleNodes` instead of `nodes` in all rendering.

- [ ] **Step 2: B2 — Remove ports from blast score calculation**

Change:
```typescript
const blastScore = Math.min(10, cveWeight + patchCount * 0.3 + ports.length * 0.15).toFixed(1);
```
to:
```typescript
const blastScore = Math.min(10, cveWeight + patchCount * 0.3).toFixed(1);
```

- [ ] **Step 3: B2 — Add info tooltip next to Blast Score**

After the blast score pill, add an info icon with tooltip:
```tsx
<div
  style={{ position: 'relative', display: 'inline-flex', cursor: 'help' }}
  title="Blast Score (0-10) measures exposure risk. Critical CVEs add 1.0, high add 0.6, medium add 0.3, low add 0.1. Pending patches add 0.3 each. Capped at 10."
>
  <Info style={{ width: 12, height: 12, color: 'var(--text-muted)' }} />
</div>
```

Import `Info` from `lucide-react`.

- [ ] **Step 4: B2 — Update legend to CVE + Patch only**

Replace the 4-item legend with:
```typescript
{[
  { color: 'var(--signal-critical)', label: `CVE (${cveNodes.length})` },
  { color: 'var(--signal-warning)', label: `Pending Patch (${patchCount})` },
].map((item) => (
```

- [ ] **Step 5: B2 — Add filter toggles above/below graph**

Add toggle buttons for CVEs / Patches:
```tsx
<div style={{ display: 'flex', gap: 6, padding: '8px 16px', background: '#000' }}>
  {[
    { key: 'cves' as const, label: 'CVEs', active: showCves, toggle: () => setShowCves(!showCves) },
    { key: 'patches' as const, label: 'Patches', active: showPatches, toggle: () => setShowPatches(!showPatches) },
  ].map(({ key, label, active, toggle }) => (
    <button
      key={key}
      type="button"
      onClick={toggle}
      style={{
        padding: '3px 10px',
        borderRadius: 4,
        border: '1px solid',
        borderColor: active ? 'var(--accent)' : 'var(--border)',
        background: active ? 'color-mix(in srgb, var(--accent) 15%, transparent)' : 'transparent',
        color: active ? 'var(--accent)' : 'var(--text-muted)',
        fontSize: 10,
        fontFamily: 'var(--font-mono)',
        cursor: 'pointer',
      }}
    >
      {label}
    </button>
  ))}
</div>
```

- [ ] **Step 6: B2 — Reduce star count for cleaner visual**

Change the star generation from 40 to 20 stars. Find the line like:
```typescript
Array.from({ length: 40 })
```
Change to:
```typescript
Array.from({ length: 20 })
```

- [ ] **Step 7: B3 — Fix Risk Breakdown visibility**

In the RiskBreakdown section, update the score number styling:
- Change score color from `barColor` to `var(--text-emphasis)` for the number
- Add `fontWeight: 700` (already set — verify)
- Lighten bar track from `var(--bg-inset)` to `color-mix(in srgb, var(--border) 50%, var(--bg-inset))`

```typescript
<span
  style={{
    fontSize: 12,
    fontFamily: 'var(--font-mono)',
    fontWeight: 700,
    color: 'var(--text-emphasis)',  // was barColor
  }}
>
  {(item.score * 10).toFixed(1)}
</span>
```

- [ ] **Step 8: B3 — Add drill-down navigation to breakdown rows**

Make each row clickable. Map labels to tab indices:
```typescript
const BREAKDOWN_NAV: Record<string, string> = {
  'Unpatched CVEs': 'cves',
  'Critical Exposure': 'cves',
  'Compliance Gaps': 'compliance',
  'Config Drift': 'policies',
  'Network Exposure': 'overview',
};
```

Wrap each breakdown row in a clickable div:
```typescript
<div
  key={item.label}
  onClick={() => {
    const tab = BREAKDOWN_NAV[item.label];
    if (tab && onTabChange) onTabChange(tab);
  }}
  style={{ cursor: 'pointer', borderRadius: 4, padding: '4px 0' }}
  onMouseEnter={(e) => { e.currentTarget.style.background = 'var(--bg-inset)'; }}
  onMouseLeave={(e) => { e.currentTarget.style.background = 'transparent'; }}
>
```

Note: This requires passing an `onTabChange` callback from the parent. Check if the OverviewTab receives tab switching capability.

- [ ] **Step 9: Commit**

```
fix(web): blast radius cleanup, risk breakdown visibility + drill-down (#303 B2,B3)
```

---

## Task 7: B5 — Hardware tab fixes

**Files:**
- Modify: `web/src/pages/endpoints/tabs/HardwareTab.tsx`

- [ ] **Step 1: Fix formatMHz for null/zero/undefined**

```typescript
function formatMHz(mhz: number | null | undefined): string {
  if (mhz == null || mhz <= 0) return '—';
  if (mhz >= 1000) return `${(mhz / 1000).toFixed(2)} GHz`;
  return `${mhz} MHz`;
}
```

- [ ] **Step 2: Show SoC label for Apple Silicon Macs**

In the CPU card, change the title conditionally:
```typescript
<div style={S.cardTitle}>
  {endpoint.os_family === 'darwin' ? 'SoC' : 'CPU'}
</div>
```

- [ ] **Step 3: Verify disk serial and battery exist**

From code review: disk serial (`disk.serial`) and battery section (`hw.battery?.present`) already exist in the code. Verify they render correctly. If battery section has any visual issues with the ProgressBar, fix the color mapping (currently uses Tailwind class names like `bg-red-500` but the file uses inline styles — check if ProgressBar supports these).

- [ ] **Step 4: Commit**

```
fix(web): hardware tab — formatMHz null safety, SoC label for Mac (#303 B5)
```

---

## Task 8: B6+B7 — Patches + CVE summary cards

**Files:**
- Modify: `web/src/pages/endpoints/tabs/PatchesTab.tsx`
- Modify: `web/src/pages/endpoints/tabs/VulnerabilitiesTab.tsx`

- [ ] **Step 1: B6 — Add stat cards to PatchesTab**

After the loading/error guards, before the table, add a stat strip:
```tsx
{/* Summary cards */}
<div style={{ display: 'flex', gap: 12, flexWrap: 'wrap' }}>
  {[
    { label: 'Pending', count: pendingCount, color: 'var(--signal-warning)', sub: `${criticalCount}C · ${highCount}H · ${mediumCount}M · ${lowCount}L` },
    { label: 'Installed', count: installedCount, color: 'var(--signal-healthy)', sub: null },
    { label: 'Failed', count: failedCount, color: 'var(--signal-critical)', sub: null },
  ].map(({ label, count, color, sub }) => (
    <div
      key={label}
      style={{
        flex: '1 1 140px',
        background: 'var(--bg-card)',
        border: '1px solid var(--border)',
        borderRadius: 8,
        padding: '14px 16px',
      }}
    >
      <div style={{ fontSize: 10, fontFamily: 'var(--font-mono)', color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.04em' }}>
        {label}
      </div>
      <div style={{ fontSize: 28, fontWeight: 700, fontFamily: 'var(--font-mono)', color, lineHeight: 1, marginTop: 4 }}>
        {count}
      </div>
      {sub && (
        <div style={{ fontSize: 10, fontFamily: 'var(--font-mono)', color: 'var(--text-muted)', marginTop: 4 }}>
          {sub}
        </div>
      )}
    </div>
  ))}
</div>
```

- [ ] **Step 2: B7 — Expand CVE hero strip to 4 cards**

In VulnerabilitiesTab, replace the hero strip with 4 stat cards matching the pattern above:
```tsx
<div style={{ display: 'flex', gap: 12, flexWrap: 'wrap' }}>
  {[
    { label: 'Total CVEs', count: cves.length, color: 'var(--text-emphasis)' },
    { label: 'Critical', count: counts.critical, color: 'var(--signal-critical)' },
    { label: 'High', count: counts.high, color: 'var(--signal-warning)' },
    { label: 'Medium + Low', count: counts.medium + counts.low, color: 'var(--text-secondary)' },
  ].map(({ label, count, color }) => (
    <div
      key={label}
      style={{
        flex: '1 1 120px',
        background: 'var(--bg-card)',
        border: '1px solid var(--border)',
        borderRadius: 8,
        padding: '14px 16px',
      }}
    >
      <div style={{ fontSize: 10, fontFamily: 'var(--font-mono)', color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.04em' }}>
        {label}
      </div>
      <div style={{ fontSize: 26, fontWeight: 700, fontFamily: 'var(--font-mono)', color, lineHeight: 1, marginTop: 4 }}>
        {count}
      </div>
    </div>
  ))}
</div>
```

- [ ] **Step 3: Commit**

```
fix(web): patches + CVE tabs — clear summary stat cards (#303 B6,B7)
```

---

## Task 9: B8 — Compliance tab fixes

**Files:**
- Modify: `web/src/pages/endpoints/tabs/ComplianceTab.tsx`

- [ ] **Step 1: Restyle Evaluate Now button**

Change from `variant="outline" size="sm"` to a more prominent style:
```tsx
<Button
  variant="secondary"
  size="default"
  className="w-full text-xs"
  disabled={isPending}
  onClick={onEvaluate}
>
  {isPending ? (
    <>
      <Loader2 className="mr-2 h-3 w-3 animate-spin" />
      Evaluating...
    </>
  ) : (
    'Evaluate Now'
  )}
</Button>
```

Import `Loader2` from `lucide-react`.

- [ ] **Step 2: Show last evaluation timestamp**

Below the Evaluate button, add:
```tsx
{summary.lastEvaluatedAt && (
  <div style={{ fontSize: 10, fontFamily: 'var(--font-mono)', color: 'var(--text-muted)', textAlign: 'center', marginTop: 4 }}>
    Last: {new Date(summary.lastEvaluatedAt).toLocaleDateString('en-US', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })}
  </div>
)}
```

Note: Check if `summary` has a `lastEvaluatedAt` field. If not, derive from the compliance data or the evaluation response.

- [ ] **Step 3: Commit**

```
fix(web): compliance tab — restyle evaluate button, show last scan (#303 B8)
```

---

## Task 10: B9+B10 — Deployments + Audit tab fixes

**Files:**
- Modify: `web/src/pages/endpoints/tabs/HistoryTab.tsx`
- Modify: `web/src/pages/endpoints/tabs/AuditTab.tsx`

- [ ] **Step 1: B9 — Human-readable deployment IDs**

Replace `shortId`:
```typescript
function shortId(id: string): string {
  return `D-${id.slice(0, 6).toUpperCase()}`;
}
```

- [ ] **Step 2: B9 — Show both absolute and relative time**

Add a `timeAgo` helper:
```typescript
function timeAgo(dateStr: string): string {
  const diffMs = Date.now() - new Date(dateStr).getTime();
  const mins = Math.floor(diffMs / 60000);
  if (mins < 1) return 'just now';
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  const days = Math.floor(hrs / 24);
  return `${days}d ago`;
}
```

In timeline cards, show both:
```tsx
<span>{formatDateTime(target.started_at)}</span>
<span style={{ color: 'var(--text-muted)', fontSize: 10 }}>
  ({timeAgo(target.started_at)})
</span>
```

- [ ] **Step 3: B10 — Truncate audit IDs with click-to-copy**

In AuditTab, create a helper component:
```tsx
function CopyId({ id }: { id: string }) {
  const [copied, setCopied] = useState(false);
  return (
    <span
      title={copied ? 'Copied!' : 'Click to copy'}
      onClick={() => {
        navigator.clipboard.writeText(id);
        setCopied(true);
        setTimeout(() => setCopied(false), 1500);
      }}
      style={{
        fontFamily: 'var(--font-mono)',
        fontSize: 11,
        color: 'var(--text-muted)',
        cursor: 'pointer',
        borderBottom: '1px dashed var(--border)',
      }}
    >
      {id.slice(0, 8)}
    </span>
  );
}
```

Replace raw ID displays with `<CopyId id={event.id} />`.

- [ ] **Step 4: B10 — Add day range selector**

Add state:
```typescript
const [dayRange, setDayRange] = useState(7);
```

Add dropdown UI near the category filter:
```tsx
<select
  value={dayRange}
  onChange={(e) => setDayRange(Number(e.target.value))}
  style={{
    padding: '5px 8px',
    borderRadius: 6,
    border: '1px solid var(--border)',
    background: 'var(--bg-card)',
    color: 'var(--text-secondary)',
    fontSize: 11,
    fontFamily: 'var(--font-mono)',
  }}
>
  {[1, 3, 7, 14, 30].map((d) => (
    <option key={d} value={d}>Last {d} day{d > 1 ? 's' : ''}</option>
  ))}
</select>
```

Filter events by date:
```typescript
const cutoff = new Date(Date.now() - dayRange * 86400000);
// Add to filtered:
.filter((event) => new Date(event.timestamp) >= cutoff)
```

- [ ] **Step 5: Commit**

```
fix(web): deployments + audit tabs — readable IDs, timestamps, day range (#303 B9,B10)
```

---

## Task 11: C1 — Dashboard percentage fix

**Files:**
- Modify: `web/src/pages/dashboard/StatCardsRow1.tsx`

- [ ] **Step 1: Audit percentage displays**

The current code uses `Math.round()` which returns integers — no decimal issue. The fleet card `%` issue may be in a different widget. Search for fleet/percentage display across all dashboard files.

Check `RiskLandscape.tsx` (shows `{ep.compliance_pct}% compliant`) and `OSHeatmapWidget.tsx` (shows `Risk: ${ep.risk_score}%`) for the actual % sign issue.

Fix any found issues:
- Double `%` — remove one
- Excess decimals — use `Math.round()` or `.toFixed(2)` pattern
- Missing `%` — add it

- [ ] **Step 2: Commit**

```
fix(web): dashboard — fix percentage sign and decimal display (#303 C1)
```

---

## Execution Order

Tasks 1 → 2 → 3 (status logic dependency), then Tasks 4-11 in any order (independent).

Recommended: 1 → 2 → 3 → 4 → 5 → 6 → 7 → 8 → 9 → 10 → 11, commit after each task.
