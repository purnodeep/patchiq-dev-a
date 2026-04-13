# Accessibility Standards (WCAG 2.1 AA)

Accessibility is not optional for enterprise software. Government contracts require it. Large enterprise procurement checklists ask for VPAT documentation. More importantly, a significant percentage of enterprise users have visual, motor, or cognitive differences that affect how they use software daily. WCAG 2.1 AA is the baseline. These standards define what to check and what good looks like.

---

## Color Contrast

**Rule: text must meet minimum contrast ratios. Failing contrast is a WCAG Level AA failure.**

- Normal text (under 18px regular or 14px bold): 4.5:1 contrast ratio against background.
- Large text (18px+ regular, or 14px+ bold): 3:1 minimum.
- UI components (buttons, inputs, focus indicators, icons): 3:1 against adjacent colors.
- Check: use a browser extension (Colour Contrast Analyser, Axe DevTools) to scan every page.
- Bad: light gray text on white background, muted status labels, disabled input text that is unreadable.
- Check both light and dark mode if both are supported — contrast often breaks when themes are added.

## Keyboard Navigation

**Rule: every feature must be fully operable using only the keyboard. No exceptions.**

- All interactive elements (links, buttons, inputs, select menus, modals, dropdowns, tabs, toggles) must be reachable via Tab.
- Tab order must be logical — follow the visual reading order (left to right, top to bottom). No jumping around.
- No keyboard traps: pressing Tab must always allow the user to move forward. The only exception is modals (Tab should cycle within the modal until it is closed).
- All actions available via mouse click must also have a keyboard equivalent.
- Check: navigate an entire user workflow using only Tab, Shift+Tab, Enter, Space, and arrow keys. Can you complete the task?
- Bad: a dropdown that opens on click but cannot be triggered with Enter or Space. A table row action that is only reachable by hovering.

## Focus Indicators

**Rule: every focusable element must have a visible focus ring. `outline: none` without a custom replacement is a WCAG failure.**

- Focus ring must be at minimum as visible as the browser default (2px solid blue).
- Better: 2–3px solid ring in the brand primary color with 2px offset from the element edge.
- Check: Tab through the full application. Can you see exactly which element is focused at all times?
- Check both light and dark mode — a blue ring may be invisible on a dark blue button in dark mode.
- Bad: any focusable element that visually disappears when focused. Any use of `outline: 0` or `outline: none` without a visible alternative.

## Heading Hierarchy

**Rule: headings must follow a strict hierarchy with no skipped levels. One h1 per page.**

- Every page has exactly one `h1` — the main page title.
- Section headings are `h2`. Subsections within those are `h3`. Go deeper only if the content genuinely warrants it.
- Do not skip levels (h1 → h3 with no h2 between them is a violation).
- Do not use heading elements for styling purposes — if you want large bold text that is not a structural heading, use `p` or `span` with CSS.
- Check: use a browser extension (HeadingsMap, Accessibility Insights) to visualize the heading tree on any page.

## Landmark Regions

**Rule: pages must use HTML landmark elements to define regions for screen reader navigation.**

- `<main>`: wraps the primary page content. One per page.
- `<nav>`: wraps navigation menus. Each `nav` should have a unique `aria-label` if there are multiple.
- `<header>`: page or section header.
- `<footer>`: page footer.
- `<aside>`: supplementary content (sidebars, related links).
- `<section>` / `<article>`: meaningful content regions, each with an accessible name via `aria-labelledby`.
- Check: use Accessibility Insights or VoiceOver/NVDA to view the landmarks list. Are all major regions represented?

## Images and Icons

**Rule: every image must have appropriate alt text. Decorative images must be hidden from assistive technology.**

- Meaningful images: `alt` attribute describes the content or function ("Screenshot of the compliance dashboard showing 73% score").
- Decorative images (used for visual styling only): `alt=""` (empty string) or rendered as CSS `background-image`.
- Icons that convey meaning: use `aria-label` on the containing button, or `title` on the SVG, or a visually-hidden sibling span.
- Icon-only buttons: must have `aria-label` describing the action ("Delete endpoint", not "Delete").
- Check: disable CSS and look at every image on the page. Is the alt text meaningful and correct?

## Forms

**Rule: every form field must have a visible, programmatically associated label.**

- Placeholder text is not a label. It disappears on focus and is not read by all screen readers.
- Every `<input>`, `<select>`, and `<textarea>` must have a `<label>` element with matching `for`/`id`, or `aria-labelledby`.
- Required fields: mark with `aria-required="true"` and a visible indicator (asterisk with legend).
- Error messages: each error must be associated with its field via `aria-describedby`. The error must be in the DOM and readable by screen readers.
- Check: inspect every form input. Does each have a programmatically associated label? Does each error message have `aria-describedby`?

## Color as the Sole Differentiator

**Rule: never convey information using color alone. Always pair with text, icon, or pattern.**

- Status indicators: a green dot alone is not accessible. Use "green dot + Healthy text" or "green dot + checkmark icon + Healthy text".
- Charts with multiple series: differentiate by color AND by pattern (dashed vs solid) or by direct label.
- Form validation: do not use only a red border to indicate an error. Include a visible error message.
- Check: view the product in a color-blindness simulator (Chrome DevTools > Rendering > Emulate vision deficiencies). Is all information still interpretable?

## Motion and Animation

**Rule: respect prefers-reduced-motion. No content flashes more than 3 times per second.**

- All animations and transitions must have a `prefers-reduced-motion: reduce` CSS media query that disables or significantly reduces motion.
- No flashing content (applies even to videos or animated charts) — flashing more than 3Hz can trigger seizures (WCAG 2.3.1, Level A).
- Check: enable "Reduce motion" in OS accessibility settings. Re-test the app. Do animations stop or simplify significantly?

## Touch and Pointer Targets

**Rule: all interactive elements must meet minimum touch target size.**

- Minimum: 44×44 CSS pixels for touch targets (WCAG 2.5.5 AAA recommends this; WCAG 2.5.8 at AA requires 24×24px with adequate spacing).
- Best practice for enterprise mobile/tablet: 44×44px.
- Spacing: touch targets must have at least 8px of spacing from adjacent targets.
- Check: use browser DevTools mobile emulation. Are small icon buttons tappable without accidentally hitting adjacent controls?

## Live Regions

**Rule: dynamic content changes must be announced to screen readers.**

- Use `aria-live="polite"` for non-urgent updates (toast notifications, table data reloading, status changes).
- Use `aria-live="assertive"` for urgent updates only (errors that block the user, critical alerts).
- Status messages that appear without focus movement must use `role="status"` or `role="alert"`.
- Check: trigger a toast notification. Use a screen reader — is the message announced without requiring the user to move focus to it?

## Tables

**Rule: data tables must be properly marked up for screen reader navigation.**

- Column headers: `<th scope="col">` on every column header.
- Row headers: `<th scope="row">` if the first column is an identifier/name.
- Complex tables (merged cells, nested structure): add `<caption>` describing the table's purpose.
- Never use a `<table>` for layout purposes — use CSS Grid or Flexbox instead.
- Check: navigate a data table using a screen reader. Are column headers announced when moving between cells?

## Links and Buttons

**Rule: links and buttons must have descriptive, context-independent labels.**

- Links navigate to a new location or resource. Buttons perform an action.
- Link text: "View endpoint details" not "click here" or "learn more".
- Multiple links with "Read more": if they all say "Read more", screen reader users cannot distinguish them. Add `aria-label` with context.
- Check: use accessibility tools to list all links on the page. Do any say "here", "click here", "read more", or "learn more" without context?

## Language

**Rule: the page language must be declared. Language changes within the page must be marked.**

- `<html lang="en">` (or the appropriate BCP 47 language tag) on every page.
- If a section of the page is in a different language: `lang="fr"` on that element.
- Check: view page source or inspect the `<html>` tag. Is the `lang` attribute present and correct?

## Error Identification

**Rule: errors must be clearly identified, associated with their source, and include a correction suggestion.**

- When a form submission fails, focus must move to the first error or to a summary of errors at the top.
- Each error message must state: what the problem is + how to fix it. ("Email address is required. Enter a valid email address in the format user@example.com.")
- Error messages must be in the DOM (not just CSS-visible) so screen readers can find them.
- Check: submit an invalid form. Does focus move to the error? Is the error message descriptive and actionable?
