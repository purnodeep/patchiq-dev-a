# Shared UI Package Audit (`packages/ui/`)

Audited: 2026-04-09
Branch: `dev-a`

---

## 1. Unused Components (exported but never imported by any frontend)

### Severity: Important

The shared package exports 32 components + numerous sub-exports. Several are completely unused across all three frontends.

**Fully unused components (zero imports in web/, web-hub/, web-agent/):**

| Export | File | Notes |
|--------|------|-------|
| `DotMosaic` | `packages/ui/src/components/dot-mosaic.tsx` | Never imported. Only exists in shared package. |
| `ThemeConfigurator` | `packages/ui/src/components/theme-configurator.tsx` | Never imported. |
| `PageHeader` | `packages/ui/src/components/page-header.tsx` | Only used in `web/src/pages/preview/ComponentPreview.tsx` (a dev preview page, not production). |
| `StatCard` (shared) | `packages/ui/src/components/stat-card.tsx` | Never imported. All 3 frontends define local `StatCard` functions instead (see Duplication section). |
| `ACCENT_PRESETS` | `packages/ui/src/theme/index.ts` | Never imported. |
| `DataTable` (shared) | `packages/ui/src/components/ui/data-table.tsx` | Never imported. All 3 frontends have local `DataTable` implementations (see Duplication section). The shared version is a bare `<table>` wrapper, while local versions are full TanStack Table integrations. |
| `Toaster` | `packages/ui/src/components/ui/sonner.tsx` | Never imported from `@patchiq/ui`. All 3 apps import `Toaster` directly from `sonner` and `toast` from `sonner`. |
| `Progress` | `packages/ui/src/components/ui/progress.tsx` | Never imported. `web/` has a local `ProgressBar` component instead. |
| `Separator` | `packages/ui/src/components/ui/separator.tsx` | Never imported. |
| `Avatar`, `AvatarImage`, `AvatarFallback` | `packages/ui/src/components/ui/avatar.tsx` | Never imported. |
| `Tabs`, `TabsList`, `TabsTrigger`, `TabsContent` | `packages/ui/src/components/ui/tabs.tsx` | Never imported from `@patchiq/ui`. All tab UIs are built with custom markup. |
| `buttonVariants`, `badgeVariants` | Various | Variant helper exports never imported. |

**Sidebar exports mostly unused:** The shared package exports 23 Sidebar sub-components. Only `SidebarProvider` is imported (by `web-hub` test file). `web/` builds a fully custom 364-line sidebar. `web-hub/` builds a 274-line custom sidebar using only `DropdownMenu` from shared. `web-agent/` has no sidebar.

**Hook not exported:** `useIsMobile` in `packages/ui/src/hooks/use-mobile.ts` is used internally by the Sidebar component but is not exported from `index.ts`. This is fine since Sidebar is barely used, but if Sidebar adoption increases, the hook may need export.

---

## 2. Duplicated Components

### Severity: Critical

**StatCard -- 14+ local re-implementations:**

The shared package has a `StatCard` component that is never used. Instead, every page that needs stat cards defines its own inline `StatCard` function:

- `web/src/pages/endpoints/EndpointsPage.tsx:98`
- `web/src/pages/patches/PatchesPage.tsx:128`
- `web/src/pages/deployments/DeploymentsPage.tsx:198`
- `web/src/pages/policies/PoliciesPage.tsx:66`
- `web/src/pages/workflows/index.tsx:126`
- `web/src/pages/alerts/AlertsPage.tsx:217`
- `web/src/pages/audit/AuditPage.tsx:53`
- `web-hub/src/pages/catalog/CatalogPage.tsx:47`
- `web-hub/src/pages/clients/ClientsPage.tsx:109`
- `web-hub/src/pages/clients/ClientDetailPage.tsx:79`
- `web-hub/src/pages/feeds/FeedsPage.tsx:57`
- `web-hub/src/pages/licenses/LicensesPage.tsx:78`

The local versions have different interfaces (some have `active`, `onClick`, `valueColor`; the shared one has `icon`, `trend`). The shared component needs to be updated to cover all use cases, then all locals should be replaced.

**DataTable + DataTablePagination -- 3 local copies:**

Each frontend has its own `components/data-table/` directory with `DataTable.tsx` and `DataTablePagination.tsx`:

- `web/src/components/data-table/` (6 files, 139-line DataTable)
- `web-hub/src/components/data-table/` (2 files, 136-line DataTable)
- `web-agent/src/components/data-table/` (2 files, 126-line DataTable)

These are full TanStack Table wrappers with row click, expanded rows, and pagination. The shared `DataTable` in `packages/ui/` is just a bare `<table>` HTML wrapper (15 lines) -- completely different component. The local versions should be consolidated into the shared package as a proper TanStack-based DataTable.

**FilterBar -- 3 copies:**

- `web/src/components/FilterBar.tsx` (189 lines)
- `web-hub/src/components/FilterBar.tsx` (172 lines)
- `web-agent/src/components/FilterBar.tsx` (201 lines)

All export `FilterBar`, `FilterPill`, `FilterSeparator`, `FilterSearch`. These should be in `packages/ui/`.

**EmptyState -- shared + local copy coexist:**

- Shared: `packages/ui/src/components/empty-state.tsx` (used by many pages)
- Local duplicate: `web/src/components/EmptyState.tsx` (75 lines, inline styles)
- Two pages import the local one: `web/src/pages/cves/CVEsPage.tsx`, `web/src/pages/preview/ComponentPreview.tsx`

The local version should be deleted and all imports switched to `@patchiq/ui`.

---

## 3. Component Quality

### Severity: Important

**Accessibility gaps in custom components:**

| Component | Issue |
|-----------|-------|
| `empty-state.tsx` | No `role` attribute (should be `role="status"` or `role="alert"`) |
| `error-state.tsx` | No `role="alert"` for error messaging, no `aria-live="polite"` |
| `stat-card.tsx` | No semantic landmark. Screen readers get no context about the stat's meaning. |
| `page-header.tsx` | No `aria-*` attributes at all |
| `mono-tag.tsx` | No `role` or `aria-label` |
| `severity-text.tsx` | No `aria-label` to convey severity to screen readers |
| `skeleton-card.tsx` | No `aria-busy="true"` or `role="status"` for loading indication |

Components with good accessibility: `ring-gauge.tsx` (has `role="img"` + `aria-label`), `dot-mosaic.tsx` (has `role="img"` + `aria-label`), `theme-configurator.tsx` (has `role="radiogroup"` + `aria-checked`).

**Missing Textarea component:**

20+ raw `<textarea>` elements across `web/` and `web-hub/` with no shared styled component. The shared package has `Input` but no `Textarea`. Each usage has different styling approaches.

Files with raw textarea: `web/src/pages/endpoints/CreateTagDialog.tsx`, `web/src/pages/endpoints/EditTagDialog.tsx`, `web/src/pages/patches/PatchDeploymentDialog.tsx`, `web/src/pages/deployments/CreateDeploymentDialog.tsx`, `web/src/pages/policies/PolicyForm.tsx`, `web/src/pages/alerts/AlertRulesDialog.tsx`, `web-hub/src/pages/clients/ClientDetailPage.tsx`, `web-hub/src/pages/feeds/AddFeedForm.tsx`, `web-hub/src/pages/licenses/LicenseForm.tsx`, and more.

---

## 4. Consistency Issues

### Severity: Important

**Toaster/toast bypass:**

All three apps import `Toaster` from `sonner` directly instead of from `@patchiq/ui`, and `toast` from `sonner` directly (35+ imports across all apps). The shared package exports a configured `Toaster` that is ignored.

- `web/src/app/layout/AppLayout.tsx:4` -- `import { Toaster } from 'sonner'`
- `web-hub/src/app/layout/AppLayout.tsx:3` -- `import { Toaster } from 'sonner'`
- `web-agent/src/App.tsx:3` -- `import { Toaster } from 'sonner'`

**Sidebar inconsistency:**

- `web/` -- Fully custom sidebar (364 lines), zero shared Sidebar component usage
- `web-hub/` -- Fully custom sidebar (274 lines), only uses `DropdownMenu` from shared
- `web-agent/` -- No sidebar at all

The 23 exported Sidebar sub-components from `@patchiq/ui` are essentially dead code. Neither frontend adopted the shared Sidebar pattern.

**CSS animation definitions only in web:**

`web/src/index.css` has 19 extra lines of `@keyframes` (pulse, ping, pulse-dot, spin) that `web-hub/` and `web-agent/` do not have. If any shared component relies on these animations, it would break in hub/agent. These should either be in `packages/ui/src/globals.css` or removed.

---

## 5. Stale Exports

### Severity: Minor

These are exported from `packages/ui/src/index.ts` but have zero consumers:

**Value exports (not types):**
- `DotMosaic`
- `ThemeConfigurator`
- `ACCENT_PRESETS`
- `Toaster` (bypassed for direct `sonner` import)
- `StatCard` (bypassed for local implementations)
- `DataTable` (the bare table wrapper; local full implementations used instead)
- `Progress`
- `Separator`
- `Avatar`, `AvatarImage`, `AvatarFallback`
- `Tabs`, `TabsList`, `TabsTrigger`, `TabsContent`
- `PageHeader` (only in dev preview page)
- All 22 Sidebar sub-components except `SidebarProvider` (test only)
- `buttonVariants`, `badgeVariants`
- `CardFooter`, `CardAction`
- `SheetTrigger`, `SheetClose`, `SheetFooter`, `SheetDescription`
- `DialogPortal`, `DialogOverlay`, `DialogTrigger`, `DialogClose`
- `SelectGroup`, `SelectLabel`, `SelectSeparator`, `SelectScrollUpButton`, `SelectScrollDownButton`

---

## 6. Missing Shared Components

### Severity: Important

Components/patterns duplicated across apps that should be extracted to `packages/ui/`:

| Pattern | Current Location | Occurrences |
|---------|-----------------|-------------|
| **FilterBar** (FilterBar, FilterPill, FilterSeparator, FilterSearch) | `{web,web-hub,web-agent}/src/components/FilterBar.tsx` | 3 copies (172-201 lines each) |
| **DataTable** (TanStack Table wrapper with pagination) | `{web,web-hub,web-agent}/src/components/data-table/` | 3 copies (126-139 line DataTable + pagination) |
| **DataTablePagination** | `{web,web-hub,web-agent}/src/components/data-table/DataTablePagination.tsx` | 3 copies (41-55 lines each) |
| **Textarea** | Raw `<textarea>` elements with varying inline styles | 20+ raw `<textarea>` across web/ and web-hub/ |
| **ProgressBar** | `web/src/components/ProgressBar.tsx` + inline in `web-agent/src/pages/status/StatusPage.tsx` | 2 implementations |
| **Stat summary cards** (with active/onClick/valueColor) | Inline `StatCard` function in 12+ page files | 14+ copies |

---

## 7. Theme/Styling Issues

### Severity: Minor

**CSS import quote inconsistency:**

- `web/src/index.css:1` -- `@import "@patchiq/ui/src/globals.css";` (double quotes)
- `web-hub/src/index.css:1` -- `@import "@patchiq/ui/src/globals.css";` (double quotes)
- `web-agent/src/index.css:1` -- `@import '@patchiq/ui/src/globals.css';` (single quotes)

Functionally identical but inconsistent.

**Shared globals.css loads fonts from CDN:**

`packages/ui/src/theme/tokens.css` loads Geist fonts from `cdn.jsdelivr.net`. This is a runtime dependency on an external CDN. For an enterprise on-prem product, fonts should be self-hosted or bundled.

**No light mode tokens defined:**

`packages/ui/src/theme/tokens.css` only defines `:root` (dark mode) variables. The `.dark` variant in `globals.css` (`@custom-variant dark`) exists but there are no light-mode color definitions. The `ThemeProvider` supports `light`/`dark`/`system` modes, but switching to light mode would show dark-mode colors since no light overrides exist.

---

## Summary by Severity

### Critical (2)
1. **StatCard duplication** -- 14+ inline copies across pages, shared version unused. Interface mismatch between shared and local versions.
2. **DataTable/FilterBar duplication** -- 3 identical copies each across web/web-hub/web-agent.

### Important (5)
3. **Unused components clutter** -- DotMosaic, ThemeConfigurator, Avatar, Progress, Separator, Tabs, 22 Sidebar components exported but never used.
4. **Toaster bypass** -- All apps import from `sonner` directly, ignoring the shared `Toaster` export.
5. **Accessibility gaps** -- `EmptyState`, `ErrorState`, `StatCard`, `PageHeader`, `MonoTag`, `SeverityText`, `SkeletonCard` lack ARIA attributes.
6. **Missing Textarea component** -- 20+ raw `<textarea>` with no shared styled component.
7. **Duplicate EmptyState** -- `web/src/components/EmptyState.tsx` duplicates the shared component; 2 pages import the local copy.

### Minor (3)
8. **Stale exports** -- ~40 named exports have zero consumers.
9. **CDN font dependency** -- Geist fonts loaded from jsdelivr CDN; problematic for air-gapped enterprise deploys.
10. **No light mode tokens** -- Theme system supports light mode but no light-mode CSS variables are defined.
