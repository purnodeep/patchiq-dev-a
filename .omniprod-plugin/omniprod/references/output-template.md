# Product Quality Review — Output Template

Fill every `{placeholder}` before saving. Do not leave placeholder text in the final report.

---

## Header

```
# Review: {page_name}

- **URL:** {url}
- **Date:** {date}
- **Reviewer:** product-observer (automated)
- **Verdict:** {✅ PASS | ⚠️ CONDITIONAL PASS | ❌ FAIL}
```

Verdict rules:
- **PASS** — zero Critical, zero Major findings
- **CONDITIONAL PASS** — zero Critical, 1+ Major (must be fixed before merge)
- **FAIL** — 1+ Critical findings

---

## Perspective Verdicts

| Perspective | Verdict | Critical | Major | Minor | Nitpick |
|-------------|---------|----------|-------|-------|---------|
| Visual Design | {emoji} | {n} | {n} | {n} | {n} |
| Interaction & UX | {emoji} | {n} | {n} | {n} | {n} |
| Accessibility | {emoji} | {n} | {n} | {n} | {n} |
| Copy & Content | {emoji} | {n} | {n} | {n} | {n} |
| Performance | {emoji} | {n} | {n} | {n} | {n} |
| Consistency | {emoji} | {n} | {n} | {n} | {n} |
| **Total** | **{overall}** | **{n}** | **{n}** | **{n}** | **{n}** |

---

## Findings

### Critical

| ID | Element | Observation | Suggestion | Flagged By |
|----|---------|-------------|------------|------------|
| C-01 | {element selector or label} | {what is broken and why it matters} | {specific fix} | {perspective(s)} |

### Major

| ID | Element | Observation | Suggestion | Flagged By |
|----|---------|-------------|------------|------------|
| M-01 | {element} | {observation} | {suggestion} | {perspective(s)} |

### Minor

| ID | Element | Observation | Suggestion | Flagged By |
|----|---------|-------------|------------|------------|
| N-01 | {element} | {observation} | {suggestion} | {perspective(s)} |

### Nitpick

| ID | Element | Observation | Suggestion | Flagged By |
|----|---------|-------------|------------|------------|
| P-01 | {element} | {observation} | {suggestion} | {perspective(s)} |

_If a severity bucket has no findings, write "None." instead of an empty table._

---

## Dev Checklist

Ordered by severity. Check off items as they are resolved.

```
Critical (must fix before ship)
- [ ] [C-01] {one-line description of the fix required}

Major (must fix before merge)
- [ ] [M-01] {one-line description}

Minor (fix in this sprint)
- [ ] [N-01] {one-line description}

Nitpick (fix when convenient)
- [ ] [P-01] {one-line description}
```

---

## Lighthouse Summary

Run via `lighthouse_audit` tool with categories: performance, accessibility, best-practices, seo.

| Category | Score | Status |
|----------|-------|--------|
| Performance | {0–100} | {✅ ≥90 / ⚠️ 50–89 / ❌ <50} |
| Accessibility | {0–100} | {✅ ≥90 / ⚠️ 50–89 / ❌ <50} |
| Best Practices | {0–100} | {✅ ≥90 / ⚠️ 50–89 / ❌ <50} |
| SEO | {0–100} | {✅ ≥90 / ⚠️ 50–89 / ❌ <50} |

Key Lighthouse flags: {list top 3 Lighthouse audit items by impact, or "None above threshold."}

---

## Console & Network Issues

### Console Errors / Warnings

| Level | Count | Top Messages |
|-------|-------|--------------|
| Error | {n} | {first 2 unique error messages, truncated to 80 chars} |
| Warning | {n} | {first 2 unique warning messages} |

_Source: `list_console_messages` output. Suppress known browser extension noise._

### Network Issues

| Type | Count | Details |
|------|-------|---------|
| 4xx errors | {n} | {endpoints that returned 4xx} |
| 5xx errors | {n} | {endpoints that returned 5xx} |
| Slow requests (>2s) | {n} | {slowest endpoint and duration} |

_Source: `list_network_requests` output filtered for status ≥400 and duration >2000ms._

---

## Comparison to Previous Review

```
Previous review date: {prior_date | "N/A — first review"}
Previous verdict:     {prior_verdict | "N/A"}

Fixed since last review:  {n} findings ({list IDs, e.g. C-01, M-02})
New since last review:    {n} findings ({list IDs})
Remaining from last:      {n} findings ({list IDs})
```

_If this is the first review for this page, write "First review — no prior baseline." and omit the table._

---

## Machine-Readable Output

A JSON version of all findings has been saved to:

```
findings/{page_slug}_{date}.json
```

Schema: `{ url, date, verdict, perspectives: [...], findings: [{id, severity, element, observation, suggestion, perspectives}], lighthouse: {...}, console_errors: n, network_errors: n }`
