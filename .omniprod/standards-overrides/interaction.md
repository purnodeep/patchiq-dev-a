# PatchIQ Interaction Standards — Project Override

> Overrides the base `interaction.md` with PatchIQ-specific interactive behaviors, theme switching, and component interactions.

---

## Theme Switching (Accent + Mode)

### Mode Toggle (Dark/Light)

- Toggle location: settings or topbar (consistent placement).
- Switching must be instant — no full-page reload, no flash of unstyled content.
- Implementation: `html.dark` / `html.light` class on `<html>` element.
- Stored in `localStorage` key `patchiq-theme-mode`.
- Respects `prefers-color-scheme` on first visit (system preference).
- **Every element must respond**: no hardcoded dark-mode-only colors that break in light mode.

### Accent Preset Switcher

- Location: ThemeConfigurator component (settings or sidebar).
- 8 preset options shown as colored swatches.
- Clicking a swatch immediately applies the accent across the entire UI.
- Stored in `localStorage` key `patchiq-theme-accent`.
- **Full coverage test**: after switching accent, verify these elements update:
  1. Primary buttons (all pages)
  2. Active sidebar nav item
  3. Active tab underlines (all detail pages)
  4. Toggle switches (on state)
  5. Focus rings on inputs
  6. Selected filter chips / stat cards
  7. Links and clickable text
  8. Progress bars and ring gauges
  9. Onboarding banners / accent-subtle backgrounds
  10. Pagination active page
  11. Bulk selection toolbar
  12. Slide panel submit buttons
- If ANY of these do not update, it is a **critical violation** — means a hardcoded color.

### Combined Test Matrix

Reviewers MUST test: `8 accent presets x 2 modes = 16 combinations`. At minimum, test 4 extremes:
1. Dark + forest (default emerald) — baseline
2. Dark + ruby (red accent) — tests that accent red doesn't conflict with signal-critical red
3. Light + amethyst (purple) — tests light mode contrast
4. Light + arctic (cyan) — tests light mode with cool tone

For each combination, check: buttons, tabs, nav, toggles, focus rings, status badges (must stay signal colors, not accent).

---

## Table Interactions

### Row Hover

- Background transitions to `var(--bg-card-hover)`.
- Transition: `background 150ms ease`.
- Cursor: `pointer` if row is clickable (navigates to detail), `default` if not.
- The expand chevron and kebab menu icons become more visible on hover (from `--text-faint` to `--text-muted`).

### Row Expand/Collapse

- Click the `>` chevron to expand.
- Chevron rotates 90deg clockwise on expand, back on collapse.
- Expanded content slides down: `animation: expandRow 200ms ease-out`.
- Only one row expanded at a time (optional per page — some may allow multiple).
- Expanded area has a subtle top border or different background to distinguish from regular rows.

### Column Sorting

- Click column header to sort ascending. Click again for descending. Third click clears sort.
- Visual: active sort column header gets `color: var(--text-emphasis)` + directional arrow icon.
- Inactive sortable headers show a subtle bi-directional chevron on hover.
- Multi-column sort: not required for POC, but if implemented, show sort priority numbers.

### Filtering

- Filter dropdowns in the filter bar update the table immediately (no "Apply" button needed).
- Active filters: show a visual indicator (count badge on filter dropdown, or filter pills below the bar).
- "Clear all filters" action visible when any filter is active.
- Search input: debounced (300ms), searches as you type.

### View Toggle (Table / Grid)

- Two icon buttons (list icon + grid icon) in the filter bar.
- Active view: icon color `var(--accent)`, or `bg: var(--accent-subtle)`.
- Inactive: icon color `var(--text-muted)`.
- Switching preserves current filters and sort state.
- Transition between views: smooth crossfade or instant swap (no janky re-render).

### Selection Mode

- Trigger: "Select" button in filter bar, or keyboard shortcut.
- On activate:
  - Checkbox column fades in at the leftmost position.
  - Header checkbox appears (select all on page).
  - Bulk action toolbar slides down between filter bar and table.
- Selecting rows:
  - Click checkbox to toggle.
  - Shift+click for range selection.
  - Header checkbox: selects/deselects all visible rows.
- Bulk action toolbar shows: `"X selected"` count + action buttons (e.g., "Deploy", "Delete", "Tag").
- Toolbar: `bg: var(--accent-subtle)`, `border: 1px solid var(--accent-border)`, `--radius-md`.
- Exit: "Cancel" button or Escape key. Checkboxes fade out.

---

## Slide Panel Interactions

### Opening

- Panel slides in from right edge: `transform: translateX(100%) -> translateX(0)`, `200ms ease-out`.
- Main content area may optionally dim (overlay with 20% black opacity).
- Focus moves to the first focusable element in the panel.

### Closing

- Close button (X) in top-right corner of panel.
- ESC key closes the panel.
- Click on the dimmed main content area closes the panel (if no unsaved changes).
- If unsaved changes: show confirmation dialog ("Discard changes?") before closing.
- Panel slides out: `translateX(0) -> translateX(100%)`, `200ms ease-in`.
- Focus returns to the element that triggered the panel open.

### Form Within Panel

- Scroll: panel content scrolls independently of the main page.
- Sticky footer: Cancel/Submit buttons stay visible at bottom regardless of scroll position.
- Validation: on blur (leaving a field), not on keystroke.
- Submit: button shows spinner + "Saving..." text, disabled state.
- Success: panel closes, toast appears, table/page refreshes data.
- Error: error banner at top of form or inline field errors. Panel stays open.

---

## Detail Page Interactions

### Tab Switching

- Instant content swap — no loading indicator if data is cached.
- If tab content requires new data: show skeleton in the tab content area.
- Tab switch does NOT change the URL hash (or if it does, it must be consistent across all detail pages).
- Keyboard: arrow keys move between tabs when tab bar is focused.

### Summary Bar

- Values in the summary bar may animate on data refresh (number count-up animation, optional).
- Mini visualizations (bars, gauges) should transition smoothly when values change.
- Clicking a metric in the summary bar may navigate to the relevant tab (optional but nice).

### Action Buttons

- Primary action (e.g., "Deploy", "Scan", "Run"): `bg: var(--accent)`.
- Dropdown actions: click reveals a menu with additional options.
- Actions should show loading state when triggered (spinner on button, disabled).
- After action completes: toast notification + data refresh.

---

## Dashboard Interactions

### Stat Card Hover

- Border: transitions to `var(--border-hover)`.
- Transform: `translateY(-1px)`.
- Transition: `all 150ms ease`.
- Cursor: `pointer` if card is clickable (navigates to detail page).

### Widget Interactions

- Charts: tooltips on hover showing exact values.
- Lists within widgets: scrollable if content exceeds widget height, with subtle scrollbar.
- Refresh: widgets may have individual refresh controls (icon in header).
- Click-through: clicking a data point in a chart or an item in a list navigates to the relevant entity.

### Loading States

- On initial load: all stat cards show `SkeletonCard` simultaneously.
- Widgets show skeletons matching their content shape.
- Skeletons use the shimmer animation: gradient sweep at `--shimmer-duration` (1.5s).
- Data loads in parallel — cards and widgets appear as their data arrives, not all at once.

---

## Global Interactions

### Loading Button Pattern

When any button triggers an async operation:
1. Button becomes disabled immediately.
2. Text changes to progressive verb: "Save" -> "Saving...", "Deploy" -> "Deploying...", "Delete" -> "Deleting...".
3. A small spinner appears inline (left of text or replacing icon).
4. On success: button returns to normal, toast appears.
5. On error: button returns to normal, error is shown (toast or inline).

### Empty State Interactions

- CTA button in empty state is the primary action button (accent-styled).
- Clicking it opens the creation slide panel.
- Empty state must be centered in the available space, not stuck to the top.

### Keyboard Shortcuts

- `Escape`: close panel, exit selection mode, close dropdown.
- `Enter`/`Space`: activate focused button, toggle, checkbox.
- `Tab`: move focus forward. `Shift+Tab`: move focus backward.
- Arrow keys: navigate within dropdowns, tab bars, menu items.
- Focus rings must be visible in both dark and light modes using `var(--accent)` or a high-contrast ring.

### Scroll Behavior

- Table headers: sticky when table scrolls vertically.
- Sidebar: independent scroll from main content.
- Slide panel: independent scroll from main content.
- "Scroll to top" button: appears after scrolling down significantly on long pages.
- Smooth scroll on anchor navigation.

### Responsive Behavior

- Minimum supported width: 1024px (enterprise desktop product).
- Below 1024px: show a "best viewed on desktop" message, but don't break.
- Dashboard grid: 4 columns above 1280px, 2 columns at 1024-1280px.
- Tables: horizontal scroll when columns exceed viewport width (not column collapsing).
- Slide panels: full-width on smaller viewports (< 1280px).
