# Hub UI Consistency Revamp

> **Goal**: Make web-hub visually consistent with the Patch Manager's design system.
>
> **Scope**: Component swaps, settings restructure, polish. No new features.
>
> **Created**: 2026-03-31 | **Branch**: dev-b

---

## Audit Summary

The hub already uses ThemeProvider, @patchiq/ui, and CSS variable tokens in 97% of files. The gaps are surgical:

| Gap | Impact | Fix |
|-----|--------|-----|
| Dashboard stat cards are custom | Visual inconsistency | Swap to shared `StatCard` |
| Settings page is single scroll | UX inconsistency | Add sidebar + sub-pages (match PM) |
| Some hardcoded colors in stat cards | Token violation | Replace with var() tokens |
| Native HTML tables (catalog, feeds, licenses) | Style inconsistency | Wrap with DataTable patterns |
| License create is a dialog | Pattern inconsistency | Keep dialog but standardize styling |
| Emoji in feed buttons | Unprofessional | Replace with Lucide icons |
| Avatar color palettes hardcoded | Token violation | Use theme-aware grayscale palette |

## Implementation Tasks

### Task 1: Dashboard StatCards → shared StatCard component
- File: `web-hub/src/pages/dashboard/StatCards.tsx`
- Replace custom inline-styled cards with `<StatCard>` from `@patchiq/ui`
- Remove hardcoded `#f59e0b` and `#10b981` colors

### Task 2: Settings page → sidebar + sub-pages
- Create `web-hub/src/pages/settings/SettingsSidebar.tsx` (matching PM pattern)
- Split SettingsPage into sub-routes: `/settings/general`, `/settings/iam`, `/settings/feeds`, `/settings/api`
- Update `web-hub/src/app/routes.tsx` with nested settings routes
- Update sidebar nav to link to `/settings` (renders sub-sidebar)

### Task 3: Catalog page → consistent table + filter patterns
- File: `web-hub/src/pages/catalog/CatalogPage.tsx`
- Replace hardcoded stat colors with var() tokens
- Clean up native `<input type="checkbox">` → proper styled checkbox

### Task 4: Feed page → remove emoji, polish cards
- File: `web-hub/src/pages/feeds/FeedsPage.tsx`
- Replace "⚡ Sync Now" → Lucide `RefreshCw` icon + "Sync Now" text
- Ensure feed cards use consistent border/shadow tokens

### Task 5: Client pages → clean up avatar colors + use DataTable patterns
- Files: `ClientsPage.tsx`, `ClientDetailPage.tsx`
- Replace hardcoded avatar colors with theme-aware grayscale from tokens
- Use shared `StatCard` for client detail stat row

### Task 6: Final polish pass
- Ensure all pages use `PageHeader` consistently
- Remove any remaining hardcoded hex colors
- Verify all forms use @patchiq/ui Select/Input (no native elements)
- Verify empty states use `EmptyState` component
