# Visual Design Standards

Visual consistency is the first signal of product quality. Enterprise buyers and their security teams evaluate software before committing — a visually inconsistent product signals immaturity. These standards define what "good" looks like at the pixel level.

---

## Spacing

**Rule: all spacing values must come from a 4px/8px grid.**

- Check: open DevTools, inspect margins, paddings, and gaps. Every value should be a multiple of 4 (4, 8, 12, 16, 20, 24, 32, 40, 48, 64...).
- Good: cards have `p-4` or `p-6`, list items have `gap-2` or `gap-3`.
- Bad: `padding: 7px`, `margin-top: 13px`, `gap: 11px` — these are rogue values that indicate copy-paste or ad hoc styling.
- Check consistent spacing between sibling elements: if two cards are 16px apart in one section, all cards should be 16px apart.

## Typography

**Rule: heading hierarchy must be consistent and semantic.**

- One `h1` per page — the page title. One level of `h2` per major section. `h3` for subsections within those.
- Check: no skipping levels (h1 → h3 is a violation). No heading levels used for styling purposes only.
- Font sizes must follow a defined scale (e.g., 12/14/16/18/20/24/30/36px). No arbitrary font sizes like 15px or 17px.
- Font weights: use defined weights only (400 regular, 500 medium, 600 semibold, 700 bold). No `font-weight: 550`.
- Line heights: body text should be 1.4–1.6x font size. Headings should be 1.1–1.3x. Check that long-form text is not cramped.
- Good: all table headers use the same 12px uppercase tracking. All card titles use the same 14px semibold.
- Bad: some stat cards use 18px titles, some use 20px, some use 22px with no logic behind the variation.

## Color

**Rule: all colors come from design tokens, never hardcoded hex values.**

- Semantic color mapping (non-negotiable):
  - Success / healthy / passing: green family
  - Danger / error / critical / failing: red family
  - Warning / at-risk / degraded: amber/orange family
  - Info / neutral emphasis: blue family
  - Neutral / secondary text: gray family
- Check: open the computed styles panel. Any color that is not a CSS variable (`--color-*` or `var(...)`) is a violation.
- Check: status badges, alert banners, and chart series all use the same semantic colors for the same meanings.
- Contrast: all text must meet 4.5:1 contrast ratio against its background. Use a browser accessibility tool to verify.
- Bad: using blue for a success state because it "looks nicer there," or red for an informational alert.

## Alignment

**Rule: elements must align to a consistent grid. No floating elements.**

- Check: open DevTools grid overlay. All major content blocks should align to column boundaries.
- Card padding must be consistent — if one card has 24px padding, all cards in the same context have 24px padding.
- Text within lists and tables must be vertically aligned (baseline or center — pick one, apply everywhere).
- Bad: one column of a dashboard has 16px left padding, the adjacent column has 24px.

## Icons

**Rule: one icon style throughout — either outlined or filled, never mixed.**

- Icon sizes must be consistent per context: 16px for inline/table, 20px for buttons, 24px for page-level or sidebar.
- Icons must be meaningful — every icon must pair with a label or tooltip explaining its function.
- Check: look for any icon that stands alone without accessible text. That is a violation.
- Bad: a gear icon for settings on one page, a wrench icon for settings on another.

## Shadows and Elevation

**Rule: use a defined elevation system, not arbitrary box-shadows.**

- Define 3–4 elevation levels: flat (no shadow), raised (card), elevated (dropdown/popover), overlay (modal).
- Check: look at all `box-shadow` values in computed styles. They should be a small, defined set.
- Bad: every component has a slightly different custom shadow value.

## Borders

**Rule: consistent border radius and border color tokens.**

- One border radius for interactive controls (buttons, inputs): typically 4–6px.
- One border radius for containers (cards, modals): typically 8–12px.
- Border colors come from tokens, not hardcoded values (e.g., `border-neutral-200` not `#e2e8f0`).
- Check: all table rows, all cards, all inputs in the same context use the same radius.

## Dark and Light Mode

**Rule: if both modes are supported, both must be complete — no white boxes in dark mode, no invisible text.**

- Check: toggle to dark mode and look for any element that still has a hardcoded light background or text color.
- Every custom component must respond to `prefers-color-scheme` or the app's theme class.
- Check: charts, custom badges, and status indicators are often hardcoded and break in dark mode.

## Visual Weight and Whitespace

**Rule: information hierarchy must be immediately legible without reading.**

- The most important element on each page should be visually dominant (larger, bolder, or more prominently placed).
- Secondary information should visually recede (smaller, lighter, more muted).
- Whitespace is not wasted space — every section needs breathing room. Check that content does not feel cramped.
- Bad: a page where every element is the same visual weight — everything competes, nothing is navigable at a glance.
