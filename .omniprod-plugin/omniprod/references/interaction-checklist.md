# Interaction Audit Checklist

For each element type found in the accessibility snapshot (`take_snapshot`), perform the listed checks. After every action that changes visual state, call `take_screenshot` and save it with the specified filename pattern. All filenames are relative to the session's `findings/screenshots/` directory.

---

## Buttons

- [ ] **Hover (primary):** `hover` each primary button → `take_screenshot` → verify background darkens or lightens, shadow appears, or scale changes. File: `btn-primary-hover.png`
- [ ] **Hover (secondary/ghost):** `hover` secondary/outline buttons → verify border or background shift. File: `btn-secondary-hover.png`
- [ ] **Hover (icon button):** `hover` icon-only buttons → verify tooltip appears or background changes. File: `btn-icon-hover.png`
- [ ] **Focus ring:** `press_key Tab` through buttons → `take_screenshot` → verify visible focus ring (2px+ outline, not just browser default). File: `btn-focus-ring.png`
- [ ] **Disabled state:** Identify disabled buttons → `take_screenshot` → verify reduced opacity, `cursor: not-allowed`, and no hover effect fires. File: `btn-disabled.png`
- [ ] **Destructive button:** `hover` delete/remove buttons → verify red/danger color treatment. File: `btn-destructive-hover.png`
- [ ] **Click (non-destructive):** `click` a non-destructive action button → `take_screenshot` → verify expected state change (loading spinner, success state, or navigation). File: `btn-click-result.png`

---

## Links

- [ ] **Nav link hover:** `hover` sidebar/topbar nav links → verify color shift or underline. File: `link-nav-hover.png`
- [ ] **Active nav link:** `take_screenshot` current page's nav item → verify active/selected visual distinction. File: `link-nav-active.png`
- [ ] **Inline link hover:** `hover` in-body text links → verify underline and color change. File: `link-inline-hover.png`
- [ ] **External link:** identify external links → verify they open in new tab (check `target="_blank"`) and show external icon if present. File: `link-external.png`

---

## Tables

- [ ] **Sortable headers:** `hover` column headers → verify sort cursor/icon appears. `click` a sortable header → `take_screenshot` → verify sort direction indicator. File: `table-sort-header.png`
- [ ] **Row hover:** `hover` a table row → `take_screenshot` → verify row highlight (background shift). File: `table-row-hover.png`
- [ ] **Row selection:** if checkboxes present, `click` row checkbox → `take_screenshot` → verify row selected state and bulk-action bar appears. File: `table-row-select.png`
- [ ] **Pagination:** `click` next-page control → `take_screenshot` → verify page changes and current-page indicator updates. File: `table-pagination.png`
- [ ] **Empty state:** navigate to an empty data set if possible → `take_screenshot` → verify empty state illustration/message is shown, not a blank area. File: `table-empty-state.png`
- [ ] **Row actions menu:** `hover` over a row → `click` the row action button/kebab → `take_screenshot` → verify action menu opens. File: `table-row-actions.png`

---

## Forms

- [ ] **Text input focus:** `click` a text input → `take_screenshot` → verify focus ring and label remains visible. File: `form-input-focus.png`
- [ ] **Text input typing:** `fill` a text field with sample value → `take_screenshot` → verify value renders, no layout shift. File: `form-input-filled.png`
- [ ] **Validation error:** submit a required field empty → `take_screenshot` → verify inline error message with red border appears. File: `form-input-error.png`
- [ ] **Select/dropdown:** `click` a select element → `take_screenshot` → verify options list opens, items are legible. File: `form-select-open.png`
- [ ] **Checkbox:** `click` an unchecked checkbox → `take_screenshot` → verify checked state with visible checkmark. File: `form-checkbox-checked.png`
- [ ] **Toggle/switch:** `click` a toggle → `take_screenshot` → verify on/off visual state changes (color + position of thumb). File: `form-toggle.png`
- [ ] **Radio buttons:** `click` each radio option in a group → `take_screenshot` → verify selected indicator and only one selected at a time. File: `form-radio.png`
- [ ] **Date picker:** `click` date picker input → `take_screenshot` → verify calendar opens and is positioned correctly (not clipped). File: `form-datepicker.png`

---

## Dropdowns / Menus

- [ ] **Trigger:** `click` dropdown trigger → `take_screenshot` → verify menu opens below (or above if near viewport bottom). File: `dropdown-open.png`
- [ ] **Item hover:** `hover` each menu item → verify highlight state. File: `dropdown-item-hover.png`
- [ ] **Keyboard nav:** `press_key ArrowDown` / `ArrowUp` → verify focus moves through items. `press_key Enter` on an item → verify selection. File: `dropdown-keyboard-nav.png`
- [ ] **Close on outside click:** `click` outside an open dropdown → `take_screenshot` → verify it closes. File: `dropdown-close-outside.png`
- [ ] **Close on ESC:** open dropdown → `press_key Escape` → verify it closes. File: `dropdown-close-esc.png`

---

## Tabs

- [ ] **Default active tab:** `take_screenshot` → verify active tab has distinct background/underline vs inactive. File: `tabs-active-state.png`
- [ ] **Tab hover:** `hover` inactive tabs → verify hover highlight. File: `tabs-hover.png`
- [ ] **Tab click:** `click` each tab → `take_screenshot` → verify panel content switches without page reload. File: `tabs-switch.png`
- [ ] **Keyboard switching:** `press_key Tab` to tab list → `press_key ArrowRight` → verify focus moves to next tab and content switches. File: `tabs-keyboard.png`

---

## Accordions / Collapsibles

- [ ] **Expand:** `click` a collapsed accordion item → `take_screenshot` → verify content expands with animation and chevron rotates. File: `accordion-expand.png`
- [ ] **Collapse:** `click` an expanded item → `take_screenshot` → verify content collapses. File: `accordion-collapse.png`
- [ ] **Multiple open:** if multi-expand is supported, open two → `take_screenshot` → verify both are visible simultaneously. File: `accordion-multi.png`

---

## Cards

- [ ] **Card hover:** `hover` a clickable card → `take_screenshot` → verify shadow lift or border highlight. File: `card-hover.png`
- [ ] **Card click:** `click` a clickable card → verify navigation or panel opens. File: `card-click.png`
- [ ] **Card action buttons:** `hover` action buttons within a card (edit, delete icons) → verify they appear or highlight. File: `card-actions.png`

---

## Modals / Dialogs

- [ ] **Open:** `click` the trigger → `take_screenshot` → verify modal renders centered, backdrop dims the page. File: `modal-open.png`
- [ ] **Close via X:** `click` the close (X) button → `take_screenshot` → verify modal closes, backdrop removed. File: `modal-close-x.png`
- [ ] **Close via ESC:** open modal → `press_key Escape` → verify modal closes. File: `modal-close-esc.png`
- [ ] **Close via backdrop:** `click` on the backdrop area → verify modal closes (or stays open if intentional — note behavior). File: `modal-close-backdrop.png`
- [ ] **Focus trap:** with modal open, `press_key Tab` repeatedly → verify focus cycles only within modal, never reaching page behind. File: `modal-focus-trap.png`

---

## Tooltips

- [ ] **Trigger:** `hover` an element with a tooltip → `take_screenshot` (allow 500ms for tooltip to appear via `wait_for`) → verify tooltip text is readable. File: `tooltip-visible.png`
- [ ] **Positioning:** verify tooltip does not overflow viewport edges. File: `tooltip-position.png`
- [ ] **Dismiss:** `hover` away → verify tooltip disappears. File: `tooltip-dismissed.png`

---

## Toast Notifications

- [ ] **Trigger:** perform an action that fires a toast → `take_screenshot` → verify toast appears in correct corner with appropriate color (success=green, error=red, etc.). File: `toast-appear.png`
- [ ] **Auto-dismiss:** wait for auto-dismiss timeout (`wait_for` selector gone) → `take_screenshot` → verify toast is removed. File: `toast-dismissed.png`
- [ ] **Manual close:** if toast has close button, `click` it → verify immediate dismissal. File: `toast-manual-close.png`

---

## Navigation

- [ ] **Sidebar active state:** `take_screenshot` → verify current page item is visually distinct (background, bold, indicator bar). File: `nav-sidebar-active.png`
- [ ] **Sidebar item hover:** `hover` a non-active item → verify hover state. File: `nav-sidebar-hover.png`
- [ ] **Sidebar collapse (if applicable):** `click` collapse trigger → `take_screenshot` → verify sidebar collapses to icon-only mode and content area expands. File: `nav-sidebar-collapsed.png`
- [ ] **Breadcrumbs:** if present, `take_screenshot` → verify hierarchy is accurate and each segment is a working link. File: `nav-breadcrumbs.png`

---

## Charts / Graphs

- [ ] **Data point hover:** `hover` over a chart data point or bar → `take_screenshot` → verify tooltip with value appears. File: `chart-hover-tooltip.png`
- [ ] **Legend interaction:** if legend items are clickable, `click` one → verify corresponding series shows/hides. File: `chart-legend-toggle.png`
- [ ] **Responsive sizing:** `resize_page` to 1024x768 → `take_screenshot` → verify chart reflows and does not overflow its container. File: `chart-responsive.png`

---

## Search

- [ ] **Input focus:** `click` search input → verify focus ring and placeholder clears. File: `search-focus.png`
- [ ] **Typing suggestions:** `fill` search input with partial term → `take_screenshot` → verify suggestion dropdown appears. File: `search-suggestions.png`
- [ ] **Clear:** `click` clear (X) button in search → verify input empties and results reset. File: `search-clear.png`
- [ ] **Results:** submit a search → `take_screenshot` → verify results list or filtered table renders correctly. File: `search-results.png`

---

## Responsive Breakpoints

Run these three passes after all interaction checks are complete.

- [ ] **1440px:** `resize_page` width=1440, height=900 → `take_screenshot` → verify layout uses available width, no unnecessary wrapping. File: `responsive-1440.png`
- [ ] **1024px:** `resize_page` width=1024, height=768 → `take_screenshot` → verify sidebar/content still usable, no overflow. File: `responsive-1024.png`
- [ ] **768px:** `resize_page` width=768, height=1024 → `take_screenshot` → verify sidebar collapses or stacks, tables scroll horizontally, no content clipped. File: `responsive-768.png`
