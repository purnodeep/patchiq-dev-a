# OmniProd — Development Expectations

> **For development agents and implementation sessions.** Read this before implementing any UI change.
> These are the quality standards that `/product-review` checks against. Following them means fewer review cycles.

## Visual

- Use the project's spacing scale (4/8/16/24/32/48px). No arbitrary values.
- All text must meet 4.5:1 contrast ratio (3:1 for large text and UI components).
- Use semantic color tokens — `signal-healthy`, `signal-critical`, `signal-warning`. Never hardcode hex colors.
- Maintain heading hierarchy: one `h1` per page, no skipped levels.
- Icons must be consistent style (outlined or filled — not mixed) and sized uniformly.
- Cards, containers, and panels use consistent border-radius and padding.

## Interaction

- Every clickable element needs a visible hover state.
- Every focusable element needs a visible focus ring. Never use `outline: none` without replacement.
- Async operations show loading indicators — skeletons for content areas, spinners for actions.
- Disabled elements show `cursor: not-allowed` and have a tooltip explaining why.
- Transitions: 150–300ms, respect `prefers-reduced-motion`.
- Modals trap focus, close on ESC and backdrop click.
- Toast notifications queue (don't stack), auto-dismiss with progress, and are closable.

## Data Display

- Format numbers with locale separators (1,234 not 1234).
- Dates: relative for recent ("2 hours ago"), absolute for older, consistent format everywhere.
- Never show `null`, `undefined`, `NaN`, or blank space. Use the `EmptyState` component.
- Distinguish "no data" from "value is zero" — they mean different things.
- Tables: right-align numbers, left-align text. Show "Showing N of M" for paginated data.
- Charts: label axes, include legends for multi-series, ensure readability at small sizes.
- Status indicators always pair color with text/icon — never color alone.

## Consistency

- Same action = same pattern everywhere. Delete always confirms. Create always uses a form.
- Same data = same format everywhere. Don't mix date formats across pages.
- Same terminology everywhere. Pick one word per concept and use it consistently.
- Button hierarchy: primary (main action), secondary (alternative), destructive (danger). Never ambiguous.
- Error display: pick inline, toast, or banner — use that pattern consistently.

## Enterprise Readiness

- No placeholder text, lorem ipsum, TODO comments, or debug info visible to users.
- Error messages: human-readable, include what happened + what to do + error code for support.
- Page loads in <2s, interactions respond in <200ms, no visible layout shift.
- Sensitive data masked. No raw UUIDs, internal IDs, or stack traces in the UI.
- Audit trail: every write action should be traceable (who, what, when).

## Accessibility (WCAG 2.1 AA)

- All interactive elements reachable and operable via keyboard.
- Every form input has a visible label (not just placeholder text).
- Every image has meaningful alt text (or empty alt if decorative).
- Proper landmark elements: `main`, `nav`, `header`, `footer`.
- Dynamic content changes announced via `aria-live` regions.
- Minimum 44x44px touch targets with adequate spacing.

## Before Submitting Your Work

1. Tab through the entire page — is everything reachable?
2. Resize to 768px — does the layout hold?
3. Check the browser console — any errors or warnings?
4. Check empty states — what happens with zero items?
5. Check loading states — is there a skeleton or spinner?
6. Hover every button — is there visual feedback?

These checks prevent 80% of findings from `/product-review`.
