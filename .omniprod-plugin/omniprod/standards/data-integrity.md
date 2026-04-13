# Data Display and Integrity Standards

Enterprise software handles real data that drives real decisions. A number formatted incorrectly, a date shown ambiguously, or an empty state that shows "null" destroys trust immediately. These standards ensure data is always displayed accurately, legibly, and with appropriate context.

---

## Numbers

**Rule: numbers must be formatted with locale-appropriate separators and appropriate precision.**

- Use thousands separators: `1,234,567` not `1234567`.
- Decimal places must match the domain: percentages to 1 decimal (45.3%), counts are whole numbers, storage sizes to 1–2 decimals.
- Large numbers may be abbreviated with units when space is constrained: `1.2M`, `45K`, `2.3B` — but show the full value in a tooltip.
- Check: look for raw unformatted numbers in tables and stat cards. Any number above 999 without a separator is a violation.
- Bad: a CVE count showing `12437` instead of `12,437`.

## Percentages

**Rule: percentages use the % symbol, display on a 0–100 scale, and handle edge cases.**

- Always append the `%` symbol — never display a bare decimal like `0.45`.
- Handle 0%: show `0%` not blank, not `0.0%` (unless precision is important in context).
- Handle 100%: show `100%` not `99.9%` due to floating point. Clamp if needed.
- Handle undefined/null: show `—` or "N/A", not `NaN%` or `undefined%`.
- Progress bars should visually reflect 0% and 100% accurately (a 100% bar is fully filled).

## Dates and Times

**Rule: date/time format must be consistent throughout the app, unambiguous, and timezone-aware.**

- Choose one date format and use it everywhere: `Mar 15, 2026` (recommended over `03/15/2026` — avoids US/EU ambiguity).
- Recent events: use relative time — "2 minutes ago", "3 hours ago", "yesterday". Threshold: within 24 hours.
- Older events: use absolute dates — "Mar 15, 2026" or "Mar 15, 2026 at 2:34 PM".
- Always show absolute time in a tooltip when relative time is displayed.
- Timezone: if users in multiple timezones use the product, show the timezone indicator (UTC, or local). Never silently assume a timezone.
- Check: look for inconsistent formats like `2026-03-15` on one table and `Mar 15` on another. That is a violation.
- Bad: showing `01/02/2026` — this is ambiguous between January 2nd and February 1st depending on locale.

## Currency

**Rule: currency values must use the proper symbol, correct decimal places, and consistent alignment.**

- Always prefix with the currency symbol: `$1,234.56`, not `1234.56`.
- Right-align currency values in tables so decimal points align vertically.
- Two decimal places for most currencies. Zero for whole-number currencies (JPY).
- Negative values: use `($1,234.56)` accounting notation or `−$1,234.56` — not just a red color.

## Empty States

**Rule: never show "null", "undefined", "NaN", or blank space where data is expected.**

- Every empty list, table, or data area must have:
  1. An icon or illustration relevant to the empty state
  2. A headline explaining what is empty ("No endpoints enrolled yet")
  3. A subtext with context or next step
  4. A call-to-action button when applicable ("Enroll your first endpoint")
- Check: empty every filter, clear all data, look for any raw JS values or blank space in data areas.
- Per-cell empty values in tables: show `—` (em dash) for missing/null values. Never blank.

## Zero vs Null

**Rule: distinguish between "the value is zero" and "the value is unknown/missing".**

- Zero: show `0` with appropriate formatting and units. A compliance score of 0% is meaningful.
- Null/unknown: show `—` or `N/A`. Do not show `0` when the value has not been measured.
- Check: look at newly created records with no history. Are null fields showing `0`? That is misleading.

## Long Text

**Rule: long text must truncate with ellipsis and reveal in full via tooltip.**

- Set `overflow: hidden; text-overflow: ellipsis; white-space: nowrap` on constrained containers.
- Truncated text must show the full value in a tooltip on hover.
- Check: paste a very long name into a form field or look for long hostnames in endpoint tables. Does it truncate or overflow?
- Bad: long text that breaks table layout or overflows its container onto adjacent elements.

## Tables

**Rule: table column alignment and indicator conventions must be consistent.**

- Right-align: numbers, currency, percentages, dates (if all same year — debatable, be consistent).
- Left-align: text fields, names, descriptions, status badges.
- Sortable columns: indicated by a sort icon (both directions visible on hover, active direction highlighted).
- Column widths: constrained and consistent. Name columns should not collapse to unreadable widths.
- Check: look at all tables. Do number columns align consistently? Are sortable headers visually distinct?

## Charts

**Rule: charts must be labeled, legible, and honest.**

- Both axes must be labeled with units unless completely obvious from context.
- Multiple series require a legend.
- Color-coded series must also be distinguished by pattern or label (not color alone).
- Y-axis should start at zero unless there is a justified reason to zoom (and then indicate this clearly).
- Charts must be readable at the size they are rendered — no invisible labels, no overlapping tick marks.
- Check: reduce the browser zoom level. Do charts remain readable? Do labels overlap?

## Status Indicators

**Rule: every status must combine color with a text label — never color alone.**

- Define the full state machine for each entity and ensure all states have an indicator.
- Good: a badge showing a green dot + "Healthy" text, an amber dot + "Degraded" text.
- Bad: a green dot with no label — screen readers cannot interpret it, and color-blind users may not distinguish it.
- Status colors must align with semantic color rules: green = good, red = bad, amber = warning, blue = info/neutral.

## Counts and Totals

**Rule: always show the total when displaying a subset.**

- Pagination footer: "Showing 25 of 1,234 endpoints" — both the page count and the total.
- Filtered results: "Showing 47 results (filtered from 1,234 total)" — make filtering visible.
- Check: apply a filter to any table. Does the UI show that filtering is active and what the unfiltered total was?
- Bad: a table that just shows 47 rows with no indication that 1,187 are hidden by a filter.

## Loading and Stale Data

**Rule: skeleton placeholders must match the shape of the content they replace.**

- A table skeleton should show table-shaped gray bars matching the column layout.
- A stat card skeleton should show a number-sized bar and a label-sized bar.
- Check: skeleton shapes that bear no resemblance to the final content are confusing, not helpful.
- Stale data: show a "Last updated X minutes ago" indicator on dashboards and live-data sections.
- Offer a manual refresh control where users may need fresh data on demand.
- If data is more than a configurable threshold old, show a visible staleness warning.
