# Enterprise Readiness Standards

Enterprise software faces a different bar than consumer apps. Procurement teams, security reviewers, and IT administrators evaluate software against a mental checklist of "does this look production-ready?" before recommending a purchase. These standards define the signals that separate professional-grade software from a demo prototype.

---

## Professional Appearance

**Rule: no placeholder content, debug artifacts, or internal identifiers visible to end users.**

- Check every page for: "Lorem ipsum", "Test data", "TODO", "FIXME", placeholder images, and unformatted UUID strings visible to users.
- UUIDs/internal IDs must never appear in the primary display of an entity. Show a human-readable name or a formatted reference code instead.
- Check: look at any entity detail page. Is a raw UUID the only identifier shown? That is a violation.
- Bad: a notification showing "Deployment d7f3e21a-4b1c-11ef-bc90-0a580af41234 completed" instead of "Deployment 'Q1 Patch Wave' completed".

## Branding

**Rule: branding must be present, consistent, and correct across all surfaces.**

- Logo: appears in the same position on all pages (typically sidebar header or topbar). Not missing on any route.
- Favicon: set and matches the product logo.
- Page titles: browser tab title should include the product name and the current page (e.g., "Endpoints — PatchIQ").
- Check: open 5 different routes. Is the browser tab title descriptive and consistent?
- Check: look for any page without a logo — particularly settings subpages, empty states, and error pages.

## Error Messages

**Rule: every error message shown to users must be human-readable, actionable, and include a reference code.**

- Structure: what happened + why it matters + what to do next + error code for support.
- Good: "We could not save your policy changes. The server returned a validation error. Check that all required fields are filled and try again. (Error: POL-4022)"
- Bad: "Error 500", "Something went wrong", a raw stack trace, or a Go/Python error string.
- Check: intentionally trigger errors (submit invalid forms, disconnect network, try forbidden actions). Read every error message.
- Error codes: include a short code that support staff can use to look up the error context quickly.

## Security Signals

**Rule: the product must demonstrate security awareness in its UI.**

- HTTPS: if the product is web-based, all pages load over HTTPS. Any mixed-content warning is a violation.
- Session timeout: inform the user before session expiry ("Your session will expire in 5 minutes"). After timeout, redirect to login gracefully — never show a broken state.
- Sensitive data masking: API keys, tokens, passwords, and secrets shown in the UI must be masked by default (show as `••••••••`) with a show/hide toggle.
- Check: look for any place where a token, credential, or secret is displayed in plaintext without masking.
- Check: leave the app idle and observe what happens at session expiry.

## Performance Perception

**Rule: the product must feel fast to the user, even when it is not technically instant.**

- Time to first meaningful content: under 2 seconds on a standard connection. Use skeletons to show structure immediately.
- Interaction response time: every button click, toggle, or form submit must respond within 200ms — even if only to show a loading state.
- No layout shift: content must not jump after it loads. Reserve space for loading content using skeletons.
- Check: hard refresh a page and watch for layout shift. Any visible jump is a Cumulative Layout Shift (CLS) violation.
- Check: slow the network in DevTools. Does the app show skeletons or just blank space?

## Localization Readiness

**Rule: even if only one language is supported today, the product must be structured to support localization.**

- No hardcoded strings in UI components — all user-visible text should come from a centralized source.
- Text containers must have room for 30–40% text expansion (German and French are longer than English).
- Layouts must not break if text doubles in length.
- Avoid icon-only buttons in areas that will need localization — the icon meaning may not translate.
- Check: identify 3 UI labels and search the codebase for them. Are they hardcoded in components or in a string source?

## Print and Export

**Rule: tables and reports must be printable or exportable.**

- Tables: export to CSV at minimum. Export to XLSX if the data is financial or compliance-related.
- Reports and dashboards: print-friendly layout (page breaks in logical places, no dark backgrounds in print).
- Exported file naming: descriptive with date — "compliance-report-2026-03-15.csv", not "export.csv".
- Check: trigger a CSV export. Open the file. Does it include column headers? Is the data formatted correctly?

## Contextual Help

**Rule: complex UI must provide in-context help without requiring external documentation.**

- Tooltip on complex fields: every form field that is not self-explanatory has an info icon with a tooltip.
- Empty state CTAs: lead users to the right action without requiring them to read docs.
- Onboarding hints: first-time users see contextual prompts pointing to key actions (dismissible).
- Check: look at your most complex form. How many fields have no explanation? Each unexplained field is a potential support ticket.

## Audit Trail

**Rule: all significant actions must be auditable and the audit log must be accessible in the UI.**

- The audit log must show: who performed the action, what the action was, what it affected, and when.
- Audit log must be filterable by user, action type, entity, and time range.
- Non-destructive read operations do not need to be logged (avoid log noise).
- Check: perform 3 different actions in the app and verify they appear in the audit log with correct attribution and timestamp.

## Multi-Tenant Isolation

**Rule: users must always know which tenant they are in, and must never see another tenant's data.**

- Tenant name or identifier must be visible in the app (topbar or sidebar header).
- Check: look for any data that could belong to another tenant leaking into the current view. Any cross-tenant data is a critical violation.
- When switching tenants (if supported), the UI must fully reset — no stale data from the previous tenant.

## Compliance Data Display

**Rule: compliance-related data must cite the source framework and explain the methodology.**

- Every compliance score or status must reference the framework it is evaluated against (CIS, PCI-DSS, HIPAA, etc.).
- Show the last assessment date and the number of controls evaluated.
- Do not show a compliance percentage without context — 73% compliant means nothing without knowing the framework and scope.
- Check: look at compliance views. Is the framework name, control count, and assessment date visible?

## Scale Behavior

**Rule: the product must handle large data gracefully without looking broken.**

- All lists must be paginated — never load everything at once. Show total count.
- Search and filter must work on the server, not only the client.
- Empty states must look intentional, not like loading failures.
- Check: look at the pagination controls. Are they consistent across all tables? Do they show total count?
- Check: set page size to maximum. Does the page layout hold up with a full table?

## Version and Changelog

**Rule: the current version must be visible, and users must be able to see what changed.**

- Product version: visible in the settings or about section (not buried).
- Changelog or release notes: accessible from within the app (link to changelog or inline what's new).
- Check: can a user find the current version without leaving the app?
