# PatchIQ Consistency Standards — Project Override

> Overrides the base `consistency.md` with PatchIQ-specific page patterns, component anatomy, and behavioral rules. Every page in every app (web, web-hub, web-agent) must follow these patterns.

---

## Page Type 1: Listing Page (Main Navigation Pages)

Every listing page (endpoints, patches, CVEs, policies, deployments, workflows, compliance, audit, catalog, feeds, etc.) MUST follow this exact anatomy from top to bottom:

### 1.1 Page Header

```
[ Page Title (h1, 22px, font-600, --text-emphasis) ]     [ Primary Action Button (accent bg) ]
[ Optional subtitle (13px, --text-secondary) ]
```

- Title is left-aligned, actions are right-aligned, same row.
- Primary action button: `bg: var(--accent)`, white text, `--radius-md`, padding `7px 14px`.
- Secondary actions (if any): ghost or outline variant, to the left of primary.

### 1.2 Filter/Search Bar

Immediately below the header, separated by `16px`:

```
[ Search Input (left, 240-320px) ] [ Filter Dropdowns... ] [ ... ] [ View Toggle: Table | Grid (right) ]
```

- Search: text input with search icon, `height: 36px`, `--radius-md`, `border: var(--border)`.
- Filters: dropdown selects for key dimensions (status, severity, type, date range).
- View toggle: icon buttons for table view and grid view. Active view uses `--accent` color, inactive uses `--text-muted`. **Every listing page MUST support both table and grid view.**
- The filter bar is always visible (not collapsible behind a "Filters" button).

### 1.3 Optional Stat Cards (Filter Chips)

For pages with countable statuses (endpoints, deployments, compliance):

```
[ All (count) ] [ Online (count) ] [ Offline (count) ] [ Stale (count) ]
```

- Styled as clickable card-chips in a `flex row, gap: 8px`.
- Inactive: `bg: var(--bg-card)`, `border: var(--border)`.
- Active/selected: `border: 1px solid [contextual color]`, `background: color-mix(in srgb, white 3%, transparent)`.
- Hover (inactive): `border-color: var(--border-hover)`.
- Count uses monospace font. Label uses sans font.
- Clicking filters the table below.

### 1.4 Data Table (Table View)

See "Table Standards" section below for full specification.

### 1.5 Grid View

When grid view is active:
- Cards in a responsive grid: `grid-template-columns: repeat(auto-fill, minmax(300px, 1fr))`, `gap: 12px`.
- Each card: `bg: var(--bg-card)`, `border: var(--border)`, `--radius-lg`, `padding: 16px 20px`.
- Card anatomy: entity name (top, font-600) + key attributes (body, grid or flex) + status badge + footer with actions.
- Card hover: `border-color: var(--border-hover)`, `transform: translateY(-1px)`.
- Card click navigates to detail page.

---

## Page Type 2: Detail Page

Every detail page (endpoint detail, deployment detail, policy detail, patch detail, etc.) MUST follow this anatomy:

### 2.1 Header Section (No Card — Flat on Page Background)

```
[ Back arrow / Breadcrumb ]
[ Entity Name (h1, 22px, font-600, --text-emphasis) ]     [ Action Buttons ]
[ Meta chips: status badge, key metadata inline ]
```

- Back navigation: arrow icon + "Back to [list page]" or breadcrumb trail.
- Entity name: the primary identifier (hostname, policy name, deployment name).
- Meta chips: inline-flex pills below the title. `padding: 2px 8px`, `border: var(--border)`, `--radius-sm`, monospace 11px, `--text-muted`.
- Action buttons right-aligned: primary action (accent), secondary actions (outline/ghost).

### 2.2 Summary Bar Card (The "Health Strip")

A single card immediately below the header showing the 3-5 most important metrics for this entity:

```
+--[ Metric 1 ]--|--[ Metric 2 ]--|--[ Metric 3 ]--|--[ Metric 4 ]--+
```

- Card: `bg: var(--bg-card)`, `border: var(--border)`, `--radius-lg`, `height: ~52-64px`.
- Metrics separated by vertical dividers: `1px solid var(--border)`, `height: 28px`.
- Each metric: label (10px monospace uppercase muted) above value (16-20px monospace emphasis).
- Values may include mini visualizations: tiny bar charts (64x3px), RingGauges, colored dots.
- Signal colors on values where semantically appropriate (risk score, compliance %).
- **Must respond to accent color changes** — any accent-colored element here updates with theme.

### 2.3 Tab Navigation

Flat underline tabs below the summary bar:

```
[ Tab 1 ]  [ Tab 2 ]  [ Tab 3 ]  [ Tab 4 ]
─────────────────────────────────────────────
```

- Container: `display: flex`, `gap: 0`, `border-bottom: 1px solid var(--border)`.
- Each tab: `padding: 8px 16px`.
- Active tab: `border-bottom: 2px solid var(--accent)`, `font-weight: 600`, `color: var(--text-emphasis)`.
- Inactive tab: `border-bottom: 2px solid transparent`, `font-weight: 400`, `color: var(--text-muted)`.
- Hover (inactive): `color: var(--text-secondary)`.
- Transition: `color 150ms, border-color 150ms`.
- **Active tab border MUST use `var(--accent)`** — changes with accent preset.

### 2.4 Tab Content

Below the tabs, each tab panel contains:
- Intricate visual components specific to the data domain.
- Sub-sections with section labels (10px monospace uppercase `--text-muted`).
- Data displayed in cards, mini-tables, or custom visualizations.
- All visual components must be accent-color-aware.

---

## Page Type 3: Dashboard

### 3.1 Grid Structure

```
Row 1: [ Stat Card ] [ Stat Card ] [ Stat Card ] [ Stat Card ]    (4-column grid)
Row 2: [ Stat Card ] [ Stat Card ] [ Stat Card ] [ Stat Card ]    (4-column grid)
Row 3: [ Large Widget (55%) ] [ Medium Widget (45%) ]              (fractional grid)
Row 4: [ Widget ] [ Widget ] [ Widget ]                            (3-column grid)
Row 5: [ Widget (50%) ] [ Widget (50%) ]                           (2-column grid)
```

- All grids: `display: grid`, `gap: 12px`.
- Stat card rows: `grid-template-columns: repeat(4, 1fr)`.
- Mixed rows: fractional units (e.g., `55fr 45fr`).
- Each platform dashboard (web, web-hub, web-agent) uses the same grid vocabulary but different widget content appropriate to its domain.

### 3.2 Stat Card Anatomy

```
+-----------------------------------+
| LABEL              (trend icon)   |  <- 10px mono uppercase muted
| 1,234                             |  <- 28px mono bold emphasis
| sublabel                          |  <- 11px mono secondary
+-----------------------------------+
```

- Card: `bg: var(--bg-card)`, `border: var(--border)`, `--radius-lg`, `min-height: 120px`.
- Hover: `border-color: var(--border-hover)`, `transform: translateY(-1px)`, `transition: 150ms`.
- Label: 10px, mono, uppercase, `letter-spacing: 0.06em`, `color: var(--text-muted)`.
- Value: 28px, mono, `font-weight: 700`, `letter-spacing: -0.03em`, `color: var(--text-emphasis)`.
- Sublabel: 11px, mono, `color: var(--text-secondary)`.
- Trend: up arrow = `--signal-healthy`, down arrow = `--signal-critical`, flat = `--text-muted`.
- Numbers: always formatted with thousands separators (`1,234` not `1234`).

### 3.3 Widget Card Anatomy

```
+---------------------------------------------+
| SECTION TITLE                    (controls)  |  <- 10px mono uppercase, border-bottom
|                                              |
|   [ Visual content: chart, gauge, list ]     |
|                                              |
+---------------------------------------------+
```

- Card: `bg: var(--bg-card)`, `border: var(--border)`, `--radius-lg`, `padding: 20px 24px`.
- Section title: 10px, mono, `font-weight: 600`, uppercase, `letter-spacing: 0.06em`, `color: var(--text-emphasis)`, `margin-bottom: 16px`, `padding-bottom: 12px`, `border-bottom: 1px solid var(--border)`.
- Content fills remaining space.
- Charts, gauges, and visual components inside MUST use signal colors semantically and accent color for interactive/selected elements.

---

## Page Type 4: Creation/Edit Forms (Right-Side Slide Panel)

ALL creation and edit forms across the entire platform use right-side slide panels. Never full-page forms. Never modals for complex forms.

### 4.1 Slide Panel Structure

```
+--[ Main Page Content ]--+--[ Slide Panel (560px) ]--+
|                         | [ Panel Title ]    [ X ]  |
|  (dims/stays visible)   | [ Form Content ]          |
|                         |                           |
|  (live preview of       | [ Field Group 1 ]         |
|   changes shows here)   | [ Field Group 2 ]         |
|                         | [ Field Group 3 ]         |
|                         |                           |
|                         | [ Cancel ]    [ Submit ]   |
+-------------------------+---------------------------+
```

- Panel width: `560px` default.
- Background: `var(--bg-elevated)`.
- Border-left: `1px solid var(--border)`.
- Shadow: `var(--shadow-lg)`.
- The main page content remains visible and dims slightly — this enables **live preview** of changes on the actual page behind.
- Panel slides in from right with `200ms ease-out` animation.

### 4.2 Live Preview Behavior

- As the user fills the form, the main page behind SHOULD update in real-time where possible (e.g., deployment target preview, policy scope preview, workflow DAG preview).
- The preview is a key differentiator — maximize visual feedback in forms.
- Preview elements use subtle accent highlighting to show what's being configured.

### 4.3 Form Field Standards

**Labels**:
```css
font-size: 10px;
font-weight: 600;
text-transform: uppercase;
letter-spacing: 0.06em;
font-family: var(--font-mono);
color: var(--text-muted);
margin-bottom: 6px;
```

**Text Inputs**:
```css
width: 100%;
height: 36px;
border-radius: var(--radius-md);  /* 6px */
border: 1px solid var(--border);
background: var(--bg-card);
padding: 0 10px;
font-size: 13px;
color: var(--text-primary);
```
- Focus: `border-color: var(--accent)`, `box-shadow: 0 0 0 2px var(--accent-subtle)`.
- Error: `border-color: var(--signal-critical)`, error text below in 12px `--signal-critical`.

**Select Dropdowns**: Same dimensions as text inputs. Chevron icon on right.

**Toggle Switches**:
```css
height: 18px;
border-radius: 100px;  /* pill */
background: var(--bg-inset);  /* off state */
```
- On state: `background: var(--accent)`. Knob slides right.

**Radio Groups**: Flex column, `gap: 8px`. Each option: label + optional description.

**Textareas**: Same styling as text inputs but `min-height: 80px`, `padding: 8px 10px`.

### 4.4 Form Section Cards

Complex forms group fields into section cards within the panel:

```css
background: var(--bg-card);
border: 1px solid var(--border);
border-radius: var(--radius-lg);  /* 8px */
box-shadow: var(--shadow-sm);
padding: 20px 24px;
```

Section title: same 10px mono uppercase pattern as dashboard widget titles, with `border-bottom: 1px solid var(--border)`.

### 4.5 Form Footer

```
[ Cancel (ghost button) ]                    [ Submit (accent button) ]
```

- Sticky to bottom of panel.
- Cancel: ghost/outline variant, left side.
- Submit: `bg: var(--accent)`, white text, right side.
- When submitting: button shows spinner, text changes to "Saving...", button is disabled.

---

## Table Standards (Applies to Every Table in the Platform)

### Column Order (Left to Right)

```
[ > Expand ] [ Name/ID ] [ Key Attrs... ] [ Status ] [ ... ] [ Kebab Menu ]
```

1. **Expand chevron** (leftmost): `>` icon that rotates 90deg on expand. Always present on every table row.
2. **Primary identifier**: Entity name or hostname. `font-weight: 600`, monospace for technical names.
3. **Key attributes**: 3-6 data columns appropriate to the entity.
4. **Status column** (near-right): badge with dot + text.
5. **Kebab menu** (rightmost): 3-dot vertical icon. Opens dropdown with row actions.

### Selection Mode

- Checkboxes are **NOT visible by default**. The table looks clean without them.
- Selection mode activates when:
  - User clicks a "Select" button in the filter bar, OR
  - User long-presses/right-clicks a row, OR
  - A keyboard shortcut triggers selection mode.
- When active: checkbox column appears (leftmost, before expand), header checkbox for select-all, bulk action toolbar appears above the table.
- Bulk action toolbar: `bg: var(--accent-subtle)`, `border: 1px solid var(--accent-border)`, shows selected count + bulk actions.
- Exiting selection mode: "Cancel" button or Escape key.

### Row Expand

- Clicking the `>` chevron expands the row to show additional detail below.
- Expanded area: `colspan: full`, `padding: 16px 20px`.
- Animation: `expandRow 200ms ease-out` (height from 0).
- Content layout: `display: grid`, `grid-template-columns: 1fr 1fr 1fr`, `gap: 20px`.
- Each section within the expanded row has a section label (10px monospace uppercase muted).

### Table Header Row

```css
padding: 9px 12px;
font-family: var(--font-mono);
font-size: 11px;
font-weight: 500;
text-transform: uppercase;
letter-spacing: 0.05em;
color: var(--text-muted);
border-bottom: 1px solid var(--border);
```

- Every column with data MUST be sortable. Sort icon (chevron up/down) shows on hover, active direction highlighted.
- Sorting must be semantically appropriate:
  - Text columns: alphabetical A-Z / Z-A.
  - Numeric columns (scores, counts): highest-first / lowest-first.
  - Date columns: newest-first / oldest-first.
  - Severity columns: critical > high > medium > low > info ordering.
  - Status columns: custom order (e.g., offline before online, failing before passing).

### Table Body Row

```css
padding: 12px 12px;
border-bottom: 1px solid var(--border-faint);
vertical-align: middle;
font-size: 13px;
color: var(--text-primary);
```

- Row hover: `background: var(--bg-card-hover)`.
- Row height: `48px` comfortable default, `36px` compact option.
- No colored left-border stripes on rows. Use status badges and icon indicators instead.

### Kebab Menu (Row Actions)

- Trigger: 3-dot vertical icon (`...`), `color: var(--text-muted)`, hover: `color: var(--text-primary)`.
- Dropdown: `bg: var(--bg-elevated)`, `border: var(--border)`, `--radius-md`, `--shadow-lg`.
- Menu items: `padding: 8px 12px`, `font-size: 13px`. Hover: `bg: var(--bg-card-hover)`.
- Destructive items (Delete, Remove): `color: var(--signal-critical)`.

### Pagination Footer

```
Showing 1-25 of 1,234 endpoints     [ < ] [ 1 ] [ 2 ] [ 3 ] ... [ > ]     [ 25 | 50 | 100 per page ]
```

- Always show: current range, total count, page numbers, per-page selector.
- Total count uses thousands separator.
- Active page: `bg: var(--accent)`, white text.
- "Showing X of Y" when filtered: "Showing 47 results (filtered from 1,234 total)".

### Empty Table State

- Uses the `EmptyState` component.
- Icon relevant to the entity type.
- Title: "No [entities] found" or "No [entities] yet".
- Description: context + next step.
- CTA button if the user can create one.

---

## Status Badge Pattern (Universal)

Every status indicator across the platform uses the same component:

```
[ colored dot (6px) ] [ status text (13px) ]
```

- Dot: `6px` circle, `border-radius: 50%`.
- Text: 13px, `font-weight: 500`, same color as dot or `--text-primary`.
- Color mapping:
  - Online/Healthy/Passing/Compliant/Active: `--signal-healthy`
  - Offline/Critical/Failing/Non-compliant: `--signal-critical`
  - Stale/Warning/Degraded/At-risk: `--signal-warning`
  - Pending/Scheduled/In-progress: `--accent`
  - Disabled/Inactive/Unknown: `--text-muted`

**Check**: find all status indicators across all pages. Do they all use the same dot+text pattern? Do the same semantic states use the same colors?

---

## Toast Notifications

- Position: bottom-right, consistent across all pages (Sonner component).
- Success: includes `--signal-healthy` accent.
- Error: includes `--signal-critical` accent, persists until dismissed.
- Auto-dismiss: 4-5 seconds for success/info.
- Stack: max 3 visible, newest on top.

---

## Confirmation Dialogs (Destructive Actions)

All delete/remove/revoke actions trigger a dialog:

```
+-------------------------------------------+
| Delete [Entity Type]?                      |
|                                            |
| Are you sure you want to delete "[name]"?  |
| This action cannot be undone.              |
|                                            |
|          [ Cancel ]  [ Delete (red) ]      |
+-------------------------------------------+
```

- Dialog: centered, `bg: var(--bg-elevated)`, `--shadow-lg`.
- Cancel: ghost/outline variant.
- Confirm: `bg: var(--signal-critical)`, white text. Text matches the action verb.
- Entity name in quotes so the user knows exactly what they're deleting.

---

## Navigation

- Sidebar navigation: consistent across all apps.
- Active nav item: `bg: var(--accent-subtle)`, `border-left: 2px solid var(--accent)` or similar accent indication.
- Active nav item MUST update when accent color changes.
- Hover: `bg: var(--bg-card-hover)`.
- Nav icons: `16px`, monoline style, consistent set.
