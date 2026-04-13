# Cross-Page and Cross-Component Consistency Standards

Inconsistency is the most common quality failure in complex software. When the same action behaves differently in two places, users lose trust and support tickets accumulate. Consistency is not about uniformity — it is about predictability. These standards define what must be the same everywhere.

---

## Action Patterns

**Rule: the same action must follow the same pattern everywhere in the product.**

- Delete/remove: always triggers a confirmation dialog with the item name, a warning, a Cancel button, and a destructive-styled Confirm button. Never inline delete without confirmation.
- Edit: always opens a form in the same presentation (inline edit, drawer, or modal — pick one per action type).
- Bulk actions: always shown in a toolbar that appears when rows are selected. Same toolbar pattern across all tables.
- Check: find two different places that support delete. Do they both use the same confirmation pattern? If not, that is a violation.

## Data Formatting

**Rule: the same data type must be formatted the same way everywhere.**

- Dates: if the dashboard shows "Mar 15, 2026", the audit log must not show "2026-03-15".
- Status badges: if endpoint status uses a dot + label badge on the endpoint list, it must use the same component in the endpoint detail page and in any modal that references it.
- Names: if user display names are shown as "First Last", that format applies everywhere users appear.
- Check: identify 3–4 data types (date, status, name, count) and search for every place they appear. Are they identical?

## Terminology

**Rule: one concept, one word, everywhere.**

- Define the canonical name for each entity in the product and enforce it throughout: UI labels, page titles, tooltips, error messages, button labels, empty states, and documentation.
- Check: look for synonym drift — if the product calls them "endpoints" in the navigation but "devices" in a modal and "machines" in an email template, that is a violation.
- Build a short terminology glossary per project and use it as a checklist.
- Bad examples: "endpoints" vs "devices" vs "machines", "patches" vs "updates" vs "fixes", "policies" vs "rules" vs "configurations".

## Navigation Patterns

**Rule: choose breadcrumbs or back buttons — not a random mix of both.**

- Breadcrumbs: for deep hierarchical navigation (more than 2 levels deep). Show the full path.
- Back button: for linear flows (wizard steps, detail → list).
- If breadcrumbs are used, they appear in the same position on every page that uses them.
- Check: navigate to three different detail pages. Does the navigation pattern match?
- Bad: the endpoint detail page has a back button, the deployment detail page has breadcrumbs, the policy detail page has neither.

## Page Layout

**Rule: all pages in the same category must share the same layout structure.**

- List pages: page header (title + primary action) → optional filter bar → table/card grid.
- Detail pages: page header (title + breadcrumb + actions) → summary section → tabbed content or section grid.
- Settings pages: sidebar sub-navigation → form section with save button.
- Check: look at all list pages. Do they all have the same header-to-content structure?

## Table Patterns

**Rule: all tables in the product follow the same anatomy.**

- Column order convention: identifier/name first, key attributes middle, status penultimate, actions (or row menu) last.
- Action column: always rightmost, always a consistent width.
- Status columns: always use the same badge component with the same sizing.
- Sortable indication: same icon style across all tables (not a chevron on one table and an arrow on another).
- Checkboxes: if bulk selection is supported, the checkbox column is always leftmost.
- Check: compare the column structure of three different tables. Does the action column always appear in the same position?

## Form Patterns

**Rule: all forms follow the same layout and behavior conventions.**

- Label position: top-aligned above the field (not inline/floating for enterprise forms — cleaner at scale).
- Field width: consistent with the expected input — short fields for codes, full-width for descriptions.
- Required indicators: `*` after the label, with a legend "* Required" at the top or bottom of the form.
- Button placement: primary action button on the right, Cancel/secondary on the left, aligned to the bottom of the form.
- Validation timing: validate on blur (leaving the field), not on every keystroke — less aggressive.
- Check: look at three different forms. Do they all match this pattern? Are buttons in the same position?

## Error Display Patterns

**Rule: choose one primary error display strategy and use it consistently.**

- Field validation errors: inline, below the field, in red, always visible (not just on submit).
- Form-level errors: a banner at the top of the form summarizing what failed.
- API/server errors: a toast or an error banner, not an alert box.
- Check: cause a validation error in two different forms. Is the error displayed in the same position and style?
- Bad: one form shows errors in a red border with a tooltip, another shows a red text below the field, another shows a toast.

## Icon Usage

**Rule: the same icon must mean the same thing everywhere. Different icons must not mean the same thing.**

- Maintain a one-to-one mapping of icon to concept: one gear = settings everywhere, one trash = delete everywhere, one pencil = edit everywhere.
- Check: look for the same icon used for different actions in different contexts — that is a violation.
- Check: look for different icons used for the same action — also a violation.
- Bad: a gear icon for settings on the sidebar, a sliders icon for settings on a card header, a wrench icon for settings in a modal.

## Card Patterns

**Rule: if cards are used, all cards of the same type share the same anatomy.**

- Stat cards: icon (top-left) → metric value → label → optional trend indicator.
- Entity cards: header (title + status badge) → body (key attributes) → footer (actions or metadata).
- Do not mix card anatomies within the same grid section.
- Check: look at a dashboard with multiple stat cards. Is the layout of each card identical in structure even if the content differs?

## Color Meaning

**Rule: a color in one context must carry the same semantic meaning in all other contexts.**

- If green means "compliant" in the compliance dashboard, green must not mean "active" in the deployment list and "enabled" in the policy table — unless those concepts are intentionally equivalent.
- Codify the meaning of each semantic color and audit all usages.
- Check: find all uses of red, amber, green, and blue. Do they all map to the same semantic meanings as defined in the visual standards?

## Button Hierarchy

**Rule: button variants are used for the correct hierarchy, consistently.**

- Primary (filled, brand color): the single most important action on the page or in the modal.
- Secondary (outlined or ghost): alternative actions, cancel actions.
- Destructive (red/danger styled): delete, remove, revoke — actions that cannot be undone.
- Never two primary buttons in the same context. Never a primary button for a destructive action (use destructive style).
- Check: look for pages or modals with two filled primary-colored buttons side by side — that is a hierarchy violation.

## Naming Conventions

**Rule: page titles, table headers, and labels follow consistent casing and style.**

- Page titles: Title Case ("Endpoint Management", "Compliance Overview").
- Table column headers: Title Case or Sentence case — pick one, apply everywhere.
- Form labels: Sentence case ("First name", not "First Name" or "FIRST NAME").
- Button labels: Title Case verbs ("Save Changes", "Delete Endpoint", "Add Policy").
- Check: scan table headers across three tables. Is the casing consistent?
