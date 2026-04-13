# PatchIQ Data Integrity Standards — Project Override

> Overrides the base `data-integrity.md` with PatchIQ-specific data display and formatting rules.

---

## Monospace for Data, Sans for Prose

This is a hard rule:

| Data Type | Font | Examples |
|-----------|------|----------|
| Hostnames | mono | `DESKTOP-ABC123`, `srv-prod-01` |
| IP addresses | mono | `192.168.1.42` |
| Version numbers | mono | `v2.3.1`, `KB5034441` |
| Patch IDs / KB numbers | mono | `KB5034441`, `CVE-2024-1234` |
| Counts and scores | mono | `1,234`, `7.8/10`, `94%` |
| Timestamps | mono | `2m ago`, `Mar 15, 2026` |
| ULIDs / UUIDs (when shown) | mono | `01HX...` (always truncated) |
| Table headers | mono | `HOSTNAME`, `RISK SCORE` |
| Section labels | mono | `SYSTEM INFORMATION`, `PATCH HISTORY` |
| Descriptions | sans | "This policy targets all Windows endpoints..." |
| Page titles | sans | "Endpoints", "Compliance Overview" |
| Button labels | sans | "Deploy", "Save Changes" |
| Toast messages | sans | "Deployment created successfully" |
| Form labels | mono | `POLICY NAME`, `TARGET SCOPE` |
| Form field values (user input) | sans | User-typed text in inputs |

**Check**: inspect any table. Are numeric values in monospace? Are hostnames in monospace? Are descriptions in sans? Mix-ups are violations.

---

## Risk Score Display

Risk scores (0-10 scale) MUST follow this pattern everywhere:

```
[ 7.8/10 ]  — monospace, signal-critical color
[ 4.2/10 ]  — monospace, signal-warning color
[ 1.5/10 ]  — monospace, signal-healthy color
```

- Always show as `X.X/10` format (one decimal, denominator shown).
- Color thresholds: `>= 7.0` critical, `>= 4.0` warning, `< 4.0` healthy.
- **Consistent everywhere**: endpoint list, endpoint detail, dashboard widgets, expanded rows.

## Compliance Score Display

Compliance percentages MUST follow this pattern:

```
[ 94% ]  — signal-healthy (>= 80%)
[ 62% ]  — signal-warning (>= 50%)
[ 23% ]  — signal-critical (< 50%)
```

- RingGauge component with `colorByValue: true` for visual displays.
- Always show the framework name alongside the score: "CIS: 94%", "PCI-DSS: 78%".
- Never show a percentage without the framework context.

## Severity Display

Severity levels use consistent colors and ordering:

| Severity | Color | Sort Order |
|----------|-------|------------|
| Critical | `--signal-critical` | 1 (highest) |
| High | `--signal-warning` | 2 |
| Medium | `--text-secondary` | 3 |
| Low | `--text-muted` | 4 |
| Info | `--text-faint` | 5 (lowest) |

- Use `SeverityText` component or consistent inline styling.
- Severity is always capitalized: "Critical", not "CRITICAL" or "critical".
- In tables, severity column should sort by this order, NOT alphabetically.

## Date/Time Formatting

| Context | Format | Example |
|---------|--------|---------|
| Recent (< 24h) | Relative | "2m ago", "3h ago" |
| Recent relative (tooltip) | Absolute | "Mar 15, 2026 at 2:34 PM" |
| Older (> 24h) | Absolute short | "Mar 15, 2026" |
| Timestamp columns | Absolute | "Mar 15, 2026 2:34 PM" |
| Duration | Human readable | "2h 15m", "3d 12h" |

- Always show absolute time in tooltip when displaying relative time.
- Monospace font for all date/time values.
- UTC indicator when relevant: "Mar 15, 2026 2:34 PM UTC".

## Count Formatting

- Always use thousands separators: `12,437` not `12437`.
- Large numbers in constrained space: `1.2K`, `45.3K`, `1.2M` — with full value in tooltip.
- Zero is `0`, not blank or `—`.
- Unknown/null is `—` (em dash), never `0` or blank.
- Monospace font for all numeric values.

## Status Text Mapping

Canonical status names per entity type (one word, one meaning, everywhere):

**Endpoints**: online, offline, stale, pending, decommissioned
**Deployments**: draft, scheduled, in-progress, completed, failed, cancelled, rolling-back
**Patches**: available, approved, installed, failed, superseded
**Policies**: active, inactive, draft
**Compliance**: compliant, non-compliant, partial, not-assessed
**Scans**: running, completed, failed, queued

**Check**: search for any synonym usage (e.g., "succeeded" instead of "completed", "disabled" instead of "inactive"). Synonyms are violations.

## Table Cell Empty Values

- Missing string values: `—` (em dash)
- Missing numeric values: `—` (em dash), NOT `0`
- Missing dates: `—` or "Never"
- Missing status: `—` with muted color
- No cell should ever show: `null`, `undefined`, `NaN`, `N/A`, or be completely blank.
