# UI Design Overhaul

**Status**: Next Up
**Wave**: 1 — Foundation Polish
**Scope**: All 3 UIs (web, web-hub, web-agent)
**Priority**: First M3 feature — sets visual foundation for everything after

---

## Problem

The current UI was built feature-by-feature during M1-M2. Each page works but there's no unified design language:

- Inconsistent spacing, card styles, and typography across pages
- Dark mode only on agent UI, light-only on Patch Manager — no user choice
- Navigation sidebar varies between apps (different widths, icon styles, grouping logic)
- No consistent empty states, loading states, or error states
- Table designs vary (some use DataTable, some custom)
- StatCard, Badge, and Button variants are inconsistent
- No established color system for severity (critical/high/medium/low colors differ between pages)
- Charts use different libraries/styles (Recharts vs custom SVG)
- Mobile/responsive behavior is untested

## Goals

1. **Consistent design system** — unified theme tokens, component patterns, spacing scale
2. **Dark/light mode** — user-selectable across all 3 apps, persistent preference
3. **Enterprise-grade navigation** — collapsible sidebar, breadcrumbs, command palette (Cmd+K)
4. **Polished empty/loading/error states** — skeleton loaders, informative empty states with CTAs
5. **Severity color system** — one canonical palette used everywhere (badges, charts, tables, pills)
6. **Responsive layout** — sidebar collapses on mobile, tables scroll horizontally
7. **Accessibility** — WCAG 2.1 AA compliance (contrast ratios, focus rings, screen reader labels)

## Scope

### Theme System
- Extend Tailwind v4 `@theme` with design tokens: `--color-severity-critical`, `--color-severity-high`, etc.
- Dark/light mode toggle in TopBar, persisted to localStorage
- Consistent card elevation (shadow levels), border radius scale, spacing scale
- Typography: display font for headings, mono for data, proportional for body

### Component Audit
- Audit every use of Button, Badge, Card, Input, Select, Dialog, Sheet, Tooltip across all 3 apps
- Ensure all use `@patchiq/ui` variants — no one-off styled components
- Standardize table patterns: all use DataTable with consistent column widths, sort indicators, pagination
- Standardize form patterns: all use react-hook-form + Zod with consistent label/error/help text styling

### Navigation
- Sidebar: consistent width (240px expanded, 64px collapsed), icon+label format, section grouping with dividers
- Breadcrumbs: automatic from route structure
- Command palette (Cmd+K): search endpoints, patches, CVEs, navigate to any page
- TopBar: org name, user avatar, dark/light toggle, notification bell, help menu

### Page Templates
- Dashboard page template: stat cards row → hero widgets → operational widgets → bottom widgets
- List page template: header (title + actions) → filters → table → pagination
- Detail page template: header (title + status + actions) → tabs → tab content
- Form page template: header → form sections → footer (save/cancel)
- Empty state template: icon + title + description + CTA button

### Severity Palette (canonical)
```
Critical: red-500 (bg: red-500/10, border: red-500/30, text: red-500)
High:     orange-500
Medium:   amber-500
Low:      blue-400
None:     gray-400
```

## Out of Scope
- New features or functionality — purely visual/structural
- Backend changes
- New pages or routes

## Approach

Use the `frontend-design` skill for high-quality, distinctive UI work. Apply changes app by app:
1. `packages/ui/` — theme tokens, component variants, shared patterns
2. `web/` — Patch Manager (richest UI, most pages)
3. `web-hub/` — Hub Manager (6 pages)
4. `web-agent/` — Agent (5 pages, read-only)

## Dependencies
- None — this is the first M3 feature

## License Gating
- None — UI improvements apply to all tiers
