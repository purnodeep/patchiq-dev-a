---
description: "Show open product findings, review history, and quality trends"
argument-hint: "[page-slug] [--all]"
allowed-tools: ["Read", "Glob", "Grep", "Bash"]
---

# OmniProd — Status

Show the current state of product findings: what's open, what's fixed, and quality trends over time.

## Parse Arguments

Arguments: $ARGUMENTS

- `page-slug`: Optional. Show findings for a specific page only (e.g., `compliance`, `dashboard`)
- `--all`: Show all findings including fixed ones (default: open only)

## Execution

### 1. Read Findings

Glob for `.omniprod/findings/*.json`. Read each file.

### 2. Aggregate

For each page reviewed:
- Count open findings by severity
- Count fixed findings
- Note the date of last review
- Track trend (improving/degrading/stable based on last 2 reviews)

### 3. Report

```markdown
# OmniProd Status

**Last updated**: {most recent review date}
**Pages reviewed**: {count}

## Open Findings Summary

| Page | Critical | Major | Minor | Nitpick | Last Review | Trend |
|------|----------|-------|-------|---------|-------------|-------|
| /compliance | 2 | 3 | 5 | 1 | 2026-04-03 | — |
| /dashboard | 0 | 1 | 2 | 0 | 2026-04-02 | ↑ improving |

**Total open**: {N} critical, {N} major, {N} minor, {N} nitpick

## Open Critical Findings

| ID | Page | Element | Observation |
|----|------|---------|-------------|
| PO-001 | /compliance | ... | ... |

## Open Major Findings

| ID | Page | Element | Observation |
|----|------|---------|-------------|
| PO-003 | /dashboard | ... | ... |

## Recently Fixed ({count})
{list of findings marked as fixed in most recent reviews}
```

If no findings exist yet:
```
No reviews found. Run /product-review <url> to start.
```
