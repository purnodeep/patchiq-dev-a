# Interaction Design Standards

Interactions define how responsive and trustworthy a product feels in real use. Enterprise users work in software for hours daily — every missing hover state, unresponsive button, or janky transition erodes confidence. These standards cover every interactive moment a user will encounter.

---

## Hover States

**Rule: every clickable or interactive element must have a visually distinct hover state.**

- Check: move the cursor over buttons, links, table rows, sidebar items, cards with actions, dropdown triggers, icon buttons.
- Good: row background lightens on hover, button background shifts one shade, link underlines on hover.
- Bad: an icon-only button with no visual change on hover — the user cannot tell if it responded to their cursor.
- Hover transitions should be immediate or very fast (50–100ms) — hover feedback must not feel delayed.

## Focus States

**Rule: every focusable element must have a visible focus ring. `outline: none` without a replacement is a violation.**

- Check: press Tab through the entire page. Every focused element must be clearly distinguishable.
- Good: a 2px solid ring in the primary brand color with a 2px offset, at minimum as visible as the browser default.
- Bad: `outline: none` on inputs, buttons, or links with no substitute focus indicator.
- Focus must be visible in both light and dark modes.

## Active and Pressed States

**Rule: buttons and controls must show a pressed/active state on click.**

- Good: button slightly depresses (scale or shadow change), or background shifts darker on mousedown.
- Check: click and hold on primary, secondary, and icon buttons. Is there visible feedback before release?
- This is especially important for actions that take time — the user needs to know the click registered.

## Loading States

**Rule: every async operation must show a loading indicator. Prefer skeletons over spinners for content areas.**

- Skeletons: use for pages, tables, cards, lists — any content that has a known shape before data arrives.
- Spinners: acceptable for button loading states, inline actions, or indeterminate-length operations.
- Check: slow down the network in DevTools to "Slow 3G". Does every data load show a skeleton or spinner?
- Bad: blank white space while data loads, or previous stale data shown without any loading indicator.
- Loading buttons must be disabled and show a spinner or label change ("Saving..." not "Save").

## Transitions and Animation

**Rule: transitions must be smooth (150–300ms), purposeful, and not distracting.**

- Page transitions: 150–200ms fade or slide. Not slower.
- Modal open/close: 200ms scale + fade.
- Dropdown open: 100–150ms. Feels instant but not jarring.
- Check: does any animation run longer than 400ms? That is too slow for a productivity app.
- Required: all motion must respect `prefers-reduced-motion`. Test by enabling the OS setting — animations should stop or reduce dramatically.
- Bad: a 600ms bounce animation on a delete confirmation modal in a patch management tool.

## Disabled States

**Rule: disabled elements must be visually distinct, non-interactive, and explain themselves.**

- Visual: reduced opacity (typically 40–50%) or desaturated/grayed styling.
- Cursor: `cursor: not-allowed` on disabled controls.
- Tooltip: hovering a disabled button must show a tooltip explaining why it is disabled ("Requires admin role", "Select an item first").
- Check: are there any disabled buttons with no tooltip? That is a gap.
- Bad: a button that appears to do nothing when clicked, with no feedback and no explanation.

## Click Targets

**Rule: minimum click target size is 32px on desktop, 44px on mobile.**

- Check: inspect icon-only buttons. Is the clickable area at least 32×32px even if the icon is 16px?
- Good: add padding around small icons to expand the hit area without changing visual size.
- Bad: a 16×16px icon that is also exactly the click target — nearly impossible to hit accurately.

## Cursor

**Rule: cursor must accurately reflect the nature of the element.**

- `pointer`: links, buttons, clickable cards, interactive icons.
- `default`: non-interactive text, labels, static content.
- `text`: editable inputs and text areas.
- `not-allowed`: disabled interactive elements.
- `grab`/`grabbing`: draggable elements.
- Check: hover over table column headers, sidebar items, status badges. Is the cursor correct in each case?

## Scroll Behavior

**Rule: scrolling must be smooth and context-aware.**

- Anchor links and "scroll to section" actions use smooth scrolling.
- If a table or list is taller than the viewport, the column headers (and ideally action toolbars) should be sticky.
- Check: on long tables, scroll down — do the column headers stay visible?
- Long pages benefit from a sticky top navigation or breadcrumb so context is never lost.

## Drag and Drop

**Rule: drag and drop must have clear affordance, feedback, and safety.**

- Drag affordance: a grip icon or visual cue on draggable items. Users should not have to discover drag by accident.
- Active dragging: the dragged item shows reduced opacity or a ghost/clone. The cursor is `grabbing`.
- Drop zones: valid drop targets highlight visually when an item is dragged over them.
- Position feedback: a gap or insertion line shows where the item will land.
- Escape key must cancel the drag operation.
- Good: a workflow builder where drag handles are clearly visible and dropping between steps shows an insertion indicator.

## Toasts and Notifications

**Rule: toasts must be consistent, non-intrusive, and not block content.**

- Position: one fixed location across the app (bottom-right or top-right — pick one).
- Auto-dismiss: 4–6 seconds for success/info. Error toasts should persist until dismissed.
- Progress indicator: a thin progress bar on the toast showing auto-dismiss countdown.
- Closable: every toast has an X button.
- Queued: multiple toasts stack vertically, not on top of each other. Limit to 3–4 visible at once.
- Check: trigger multiple actions in rapid succession. Do toasts pile up or queue gracefully?

## Modals and Dialogs

**Rule: modals must be fully controlled and escape-hatch friendly.**

- Focus trap: keyboard Tab must cycle only within the open modal. Focus must not reach elements behind the overlay.
- ESC key closes the modal (except for destructive confirmation dialogs where accidental close could lose form data).
- Click outside the modal closes it (except for destructive confirmations).
- Background scroll is locked while modal is open.
- On close: focus returns to the element that opened the modal.
- Check: open a modal, press Tab until focus escapes — if it reaches the background page, that is a violation.
