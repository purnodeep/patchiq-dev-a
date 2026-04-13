# Dashboard Enhancement Plan

**Date**: 2026-04-10
**Branch**: dev-a
**Track**: Standard (modified — design decisions made in-conversation)

---

## Overview

Comprehensive dashboard upgrade: 5 new Tier 1 widgets, sticky edit toolbar, drag-to-add widgets, per-widget configuration system, dashboard presets dropdown, and fix 3 stub/partial widgets.

---

## Workstream 1: Sticky Edit Toolbar

**Files**: `web/src/pages/dashboard/DashboardPage.tsx`

- Wrap the toolbar (Reset / Add Widget / Done buttons) in a sticky container when `isEditing === true`
- `position: sticky; top: 0; z-index: 50; backdrop-filter: blur(8px); background: var(--bg-page)/90%`
- Add bottom border separator
- Buttons remain accessible while scrolling the grid

---

## Workstream 2: Drag-to-Add from Widget Drawer

**Files**: `web/src/pages/dashboard/WidgetDrawer.tsx`, `web/src/pages/dashboard/DashboardPage.tsx`, `web/src/pages/dashboard/hooks/useDashboardLayout.ts`

- Make WidgetDrawer items draggable (`draggable="true"`, `onDragStart` sets widget ID in dataTransfer)
- Add `isDroppable={true}` and `onDrop` handler to ResponsiveGridLayout
- On drop: calculate grid position from drop coordinates, add widget at that position
- Keep click-to-add as fallback
- Visual feedback: drop zone highlight, ghost preview during drag

---

## Workstream 3: Per-Widget Configuration System

**Files**: `web/src/pages/dashboard/registry.ts`, `web/src/pages/dashboard/WidgetWrapper.tsx`, new `web/src/pages/dashboard/WidgetConfigPopover.tsx`, `web/src/pages/dashboard/hooks/useDashboardLayout.ts`, `web/src/pages/dashboard/types.ts`

### 3a. Config Schema in Registry

Each widget declares configurable options:

```ts
config?: {
  [key: string]: {
    type: 'select' | 'multi-select' | 'toggle' | 'number';
    label: string;
    category: 'data' | 'display' | 'behavior';
    options?: { label: string; value: string }[];
    default: any;
    min?: number;
    max?: number;
  };
};
```

### 3b. Config Categories

| Category | Options | Examples |
|----------|---------|---------|
| **Data** | timeRange, severityFilter, groupFilter, frameworkFilter, limit | 7d/30d/90d, critical/high/all, endpoint group, top 5/10/20 |
| **Display** | chartType, showLabels, compactMode, comparisonMode | line/bar/area, on/off, vs previous period |
| **Behavior** | refreshInterval, clickTarget, alertThreshold | 30s/1m/5m, target route, value threshold to highlight |

### 3c. UI

- Gear icon on WidgetWrapper header (visible on hover, always visible in edit mode)
- Click gear → Popover with tabbed form (Data / Display / Behavior)
- Auto-generated from config schema
- "Reset to defaults" button at bottom
- Live preview: changes apply immediately

### 3d. Persistence

- Config stored in localStorage alongside layout: `dashboard-config: { [widgetId]: { timeRange: '30d', ... } }`
- Widgets receive config via props from WidgetWrapper
- Each widget reads its config and adjusts API hook params + rendering

---

## Workstream 4: Dashboard Presets Dropdown

**Files**: `web/src/pages/dashboard/DashboardPage.tsx`, new `web/src/pages/dashboard/presets.ts`, `web/src/pages/dashboard/hooks/useDashboardLayout.ts`

### Presets

| Preset | Widgets Included |
|--------|-----------------|
| **Executive** | Primary KPIs, Compliance Rings, Risk Projection, MTTR Decay, Patch Velocity, SLA Bridge |
| **Operations** | Both stat card rows, Deployment Pipeline, SLA Status, SLA Countdown, SLA Breach Forecast, Activity Feed, Quick Actions |
| **Security** | Top Vulns, Blast Radius, Exposure Window, Attack Path Heatmap, Risk Landscape, OS Heatmap, Drift Detector |
| **Compliance** | Compliance Rings, Drift Detector, SLA Status, SLA Bridge, Risk Projection, Coverage (future) |
| **Custom** | User's saved widget selection + layout |

### UI

- Dropdown in header bar, left of Customize button
- Selecting a preset loads its widgets + default layout
- "Custom" is auto-selected when user modifies any preset
- Persist last-used preset in localStorage

---

## Workstream 5: Five New Tier 1 Widgets

### 5a. Exposure Window Timeline

**Backend**: `GET /api/v1/dashboard/exposure-windows` — returns critical CVEs with first_seen, patched_at (nullable), affected_endpoint_count
**Frontend**: `web/src/pages/dashboard/widgets/ExposureWindowTimeline.tsx`
**Visual**: Horizontal swim-lane chart. Each CVE = a row. Red bar = time window from first_seen to now (or patched_at). Sorted by duration descending.
**Hook**: `useExposureWindows()` in `web/src/api/hooks/useDashboard.ts`
**Config**: timeRange (30d/90d/1y), severityFilter (critical/high), limit (5/10/15)

### 5b. MTTR Decay Curve

**Backend**: `GET /api/v1/dashboard/mttr` — returns weekly MTTR values by severity for last N weeks
**Frontend**: `web/src/pages/dashboard/widgets/MTTRDecayCurve.tsx`
**Visual**: Multi-line area chart. X = weeks, Y = hours. One line per severity (critical, high, medium). Filled area under each.
**Hook**: `useMTTR()` in `web/src/api/hooks/useDashboard.ts`
**Config**: timeRange (13w/26w/52w), severities (multi-select)

### 5c. Attack Path Heatmap

**Backend**: `GET /api/v1/dashboard/attack-paths` — returns nodes (endpoints with unpatched CVEs) and edges (potential lateral movement based on shared network/CVE chains)
**Frontend**: `web/src/pages/dashboard/widgets/AttackPathHeatmap.tsx`
**Visual**: Force-directed graph (d3-force or custom canvas). Nodes = endpoints colored by risk. Edges = exploitation paths. Glowing edges for active chains (internet-facing → privesc → lateral).
**Hook**: `useAttackPaths()` in `web/src/api/hooks/useDashboard.ts`
**Config**: groupFilter, maxNodes (20/50/100)

### 5d. Drift Detector

**Backend**: `GET /api/v1/dashboard/drift` — returns endpoints with drift_score (0=compliant, higher=more drift), policy_name, last_compliant_at
**Frontend**: `web/src/pages/dashboard/widgets/DriftDetector.tsx`
**Visual**: Beeswarm plot. Center line = compliant. Dots migrate outward based on drift_score. Color by severity of drift. Tooltip shows endpoint + policy + days since compliant.
**Hook**: `useDrift()` in `web/src/api/hooks/useDashboard.ts`
**Config**: groupFilter, threshold (show only drift > N), limit (30/50/100)

### 5e. SLA Breach Forecast

**Backend**: `GET /api/v1/dashboard/sla-forecast` — returns endpoints approaching SLA breach with predicted_breach_at, current_velocity, probability_pct
**Frontend**: `web/src/pages/dashboard/widgets/SLABreachForecast.tsx`
**Visual**: Sorted list with countdown timers. Each row: endpoint name, SLA tier, time remaining, probability bar. Red pulse on items < 2h. Grouped by SLA tier.
**Hook**: `useSLAForecast()` in `web/src/api/hooks/useDashboard.ts`
**Config**: slaFilter (critical/high/all), showBreachedOnly (toggle)

---

## Workstream 6: Fix Stub/Partial Widgets

### 6a. SLA Countdown — replace mock data

**Backend**: `GET /api/v1/dashboard/sla-deadlines` — returns actual patch SLA deadlines with patch_id, patch_name, severity, deadline_at, remaining_seconds
**Frontend**: Update `SLACountdown.tsx` to use new hook instead of hardcoded data
**Hook**: `useSLADeadlines()` in `web/src/api/hooks/useDashboard.ts`

### 6b. SLA Status — replace fake progression

**Backend**: Extend `/api/v1/dashboard/summary` to include `sla_tiers: [{ tier, window_hours, total, compliant, overdue, avg_elapsed_pct }]`
**Frontend**: Update `SLAStatus.tsx` to use real tier data instead of simulated elapsed fractions

### 6c. Risk Projection — replace synthetic math

**Backend**: `GET /api/v1/dashboard/risk-projection` — server-calculated 30-day projection using actual patch velocity, CVE inflow rate, deployment success rate
**Frontend**: Update `RiskProjection.tsx` to consume backend projections instead of client-side linear math

---

## Implementation Order

1. **Workstream 1** (Sticky Toolbar) — 15min, no dependencies
2. **Workstream 3** (Config System) — foundation for everything else
3. **Workstream 4** (Presets Dropdown) — depends on config system
4. **Workstream 2** (Drag-to-Add) — independent
5. **Workstream 6** (Fix Stubs) — backend + frontend, independent
6. **Workstream 5** (New Widgets) — backend + frontend, use config system

## Parallel Subagent Dispatch

| Agent | Workstream | Scope |
|-------|-----------|-------|
| A | 1 + 2 | Sticky toolbar + drag-to-add (pure frontend) |
| B | 3 + 4 | Config system + presets (pure frontend) |
| C | 5a + 5b + 5c (backend) | New backend endpoints for Exposure Window, MTTR, Attack Path |
| D | 5d + 5e + 6 (backend) | New backend endpoints for Drift, SLA Forecast, fix stubs |
| E | 5a + 5b + 5c (frontend) | New widget components (after C finishes) |
| F | 5d + 5e + 6 (frontend) | New widget components + fix stubs (after D finishes) |

Agents A+B+C+D can run in parallel. E+F depend on C+D respectively.
