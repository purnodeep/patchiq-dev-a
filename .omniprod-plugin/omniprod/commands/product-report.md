---
description: "Aggregate all findings into a product-level report with root cause grouping and impact ranking"
argument-hint: "[--since=<date>] [--format=md|json]"
allowed-tools: ["Read", "Write", "Bash", "Glob", "Grep", "Agent"]
---

# Product Report — Aggregate Analysis

You are a product quality analyst. Your job is to aggregate ALL review findings (page reviews, flow reviews, smoke tests) into a single product-level report with root cause grouping, impact scoring, and cross-page correlation. You do NOT use the browser — you analyze existing review data.

## Parse Arguments

Arguments: $ARGUMENTS

Parse:
- `--since`: Only include reviews from this date forward (YYYY-MM-DD format, optional, default: all reviews)
- `--format`: Output format (optional, default: `md`). Options: `md`, `json`, `both`

If invalid arguments, show usage:
```
Usage: /product-report [options]

Examples:
  /product-report
  /product-report --since=2026-04-01
  /product-report --since=2026-04-01 --format=both
```

## Phase 1: Collect Review Data

### Step 1: Find All Findings Files

```bash
ls -la .omniprod/findings/*.json 2>/dev/null
```

If no findings files exist, stop immediately:
> **No review data found.** Run `/product-review <url>` or `/product-smoke` first to generate findings, then run `/product-report` to aggregate.

### Step 2: Filter by Date

If `--since` is provided, filter findings files to only those with dates >= the since date. Findings files are named `{YYYY-MM-DD}-{slug}.json` — parse the date from the filename.

### Step 3: Read All Findings

Read every matching findings JSON file. Categorize each by type:
- Files with `"type": "smoke"` → smoke test results
- Files with `"type": "flow"` → flow review results
- Files with `"findings"` array and `"perspectives"` → page review results
- Files with `"type": "product-report"` → previous product reports (skip these, don't aggregate reports into reports)

Build a master list of:
- All individual findings (from page and flow reviews)
- All page statuses (from smoke tests)
- All verdicts (from page and flow reviews)

## Phase 2: Impact Scoring

### Step 1: Try Automated Scorer

Check if the impact scorer script exists:
```bash
test -f .omniprod-plugin/omniprod/scripts/impact-scorer.py && echo "EXISTS" || echo "MISSING"
```

If it exists, run it:
```bash
python3 .omniprod-plugin/omniprod/scripts/impact-scorer.py .omniprod/findings/*.json --output .omniprod/findings/product-scored.json
```

Read the scored output and use those impact scores.

### Step 2: Manual Scoring (if script missing or fails)

If the script doesn't exist or fails, score each finding manually using this formula:

**Impact Score = Severity Weight x Breadth x Persistence**

| Factor | Values |
|--------|--------|
| Severity Weight | critical=4, major=3, minor=2, nitpick=1 |
| Breadth | Number of pages where this finding (or its root cause) appears. Min 1. |
| Persistence | 1.0 for new findings, 1.5 for findings open > 7 days (check `first_seen`) |

Example: A critical finding appearing on 3 pages, open for 2 weeks = 4 x 3 x 1.5 = 18.0

## Phase 3: Root Cause Analysis

### Step 1: Cluster Findings

Group findings by root cause. Two findings share a root cause if:
- They reference the same element type AND the same observation pattern (e.g., "missing focus indicator" on button vs. link = same root cause: no global focus styles)
- They reference the same data artifact (e.g., "test-001" seed data visible on /compliance and /deployments = same root cause: seed data cleanup)
- They reference the same API endpoint or data source issue
- They reference the same CSS/styling gap applied globally

### Step 2: Assign Root Cause IDs

For each cluster, assign an ID: `RC-001`, `RC-002`, etc. Sorted by total impact score (sum of all findings in the cluster).

### Step 3: Write Root Cause Descriptions

For each root cause:
- **ID**: RC-{NNN}
- **Title**: Brief description of the underlying problem
- **Severity**: Highest severity of any finding in the cluster
- **Pages affected**: List of pages
- **Findings**: List of finding IDs in this cluster
- **Suggested fix**: One actionable fix that resolves ALL findings in the cluster
- **Total impact score**: Sum of individual finding scores

## Phase 4: Cross-Page Consistency Analysis

### Step 1: Dispatch Consistency Checker (sub-agent)

Use the `Agent` tool to dispatch a sub-agent for cross-page entity consistency analysis:

```
Agent:
  model: "sonnet"
  name: "consistency-checker"
  description: "Cross-page entity consistency analysis"
```

Sub-agent prompt:

```
You are analyzing cross-page data consistency for a product review.

Read ALL of these findings files:
{list every findings JSON file path}

Also read the product map if it exists:
.omniprod/product-map.json

For each entity type mentioned across reviews (endpoints, patches, deployments, compliance scores, etc.), check:

1. **Count consistency**: Does the same entity count appear on all pages that show it?
   - Example: Dashboard says "47 endpoints" but /endpoints page lists 49 rows
   - Check sidebar counts, dashboard stats, page headers, table row counts

2. **Status consistency**: Does the same entity show the same status everywhere?
   - Example: Deployment shows "Completed" on /deployments but "In Progress" on /endpoints detail

3. **Data consistency**: Does the same entity show the same data everywhere?
   - Example: Endpoint "web-server-01" shows different OS on /endpoints vs /compliance

4. **Terminology consistency**: Are the same concepts named the same way?
   - Example: "Patches" on one page, "Updates" on another, "Hotfixes" on a third
   - Example: "Critical" severity vs "High" severity for the same level

5. **Navigation consistency**: Do links between pages work bidirectionally?
   - Example: Dashboard links to /endpoints but /endpoints has no way back to dashboard context

For each inconsistency found, output:

### CONSISTENCY-{NNN}: {brief title}
- **Entity**: {what entity or concept}
- **Page A**: {page path} shows {value/state}
- **Page B**: {page path} shows {value/state}
- **Likely cause**: {caching, different queries, stale data, terminology mismatch}
- **Impact**: {user confusion, incorrect decisions, broken navigation}

If no inconsistencies found, state: "No cross-page consistency issues detected."
```

### Step 2: Incorporate Consistency Findings

Add any consistency issues found to the report as a dedicated section. These are cross-cutting issues that no single page review would catch.

## Phase 5: Generate Report

### Markdown Report

Write to `.omniprod/reviews/{date}-product-report.md`:

```markdown
# Product Review Report: {project_name}

Generated: {YYYY-MM-DD}
Reviews included: {N} page reviews, {N} flow reviews, {N} smoke tests
Period: {earliest_date} to {latest_date}

---

## Executive Summary

| Metric | Value |
|--------|-------|
| Total findings | {count} |
| Unique root causes | {count} |
| Critical | {count} |
| Major | {count} |
| Minor | {count} |
| Nitpick | {count} |
| Pages reviewed | {smoke_count} (smoke) + {deep_count} (deep) |
| Flows tested | {flow_count} |
| Cross-page issues | {consistency_count} |

## Top 10 Issues by Impact

| Rank | Score | Severity | Issue | Pages | Fix |
|------|-------|----------|-------|-------|-----|
{For each of the top 10 findings by impact score:}
| {rank} | {score} | {severity} | {observation} | {page_count} | {brief fix} |

## Root Causes (grouped)

{For each root cause, sorted by total impact score descending:}

### {RC-ID}: {title} ({severity}, {N} pages)

{Description of the underlying problem.}

**Affected findings:**
{List each finding ID and which page it appears on}

**Suggested fix:** {One actionable fix that resolves all findings in this cluster}

**Total impact score:** {score}

---

## Cross-Page Consistency Issues

| Issue | Page A | Page B | Details |
|-------|--------|--------|---------|
{For each consistency issue found by the sub-agent}

{If no consistency issues: "No cross-page consistency issues detected."}

## Per-Page Summary

| Page | Verdict | Critical | Major | Minor | Nitpick | Last Reviewed |
|------|---------|----------|-------|-------|---------|---------------|
{For each page that has a deep review, sorted by severity}

## Flow Summary

| Flow | Verdict | Assertions | Passed | Failed | Last Reviewed |
|------|---------|------------|--------|--------|---------------|
{For each flow review, if any}

{If no flow reviews: "No flow reviews found. Run `/product-flow <flow-name>` to test user flows."}

## Smoke Test Summary

| Metric | Value |
|--------|-------|
| Pages tested | {count} |
| All OK | {count} |
| With errors | {count} |
| Timed out | {count} |

{If no smoke tests: "No smoke test data found. Run `/product-smoke` for a quick health check of all pages."}

## Trend

{If multiple reviews of the same page exist across different dates, show the trend:}

```
{date_1}: {finding_count} findings ({details})
{date_2}: {finding_count} findings ({delta}: +N new, -N fixed)
```

{Classify trend as: IMPROVING, STABLE, or DEGRADING based on finding count direction.}

{If only one review per page exists: "Insufficient data for trend analysis. Run reviews on multiple dates to track progress."}

## Dev Checklist (Priority Order)

### Fix First (impact score >= 10)
{For each root cause with total impact >= 10:}
- [ ] {RC-ID}: {title} — {brief fix description}

### Fix Next (impact score 5-9.9)
{For each root cause with total impact 5-9.9:}
- [ ] {RC-ID}: {title} — {brief fix description}

### Fix Later (impact score < 5)
{For each root cause with total impact < 5:}
- [ ] {RC-ID}: {title} — {brief fix description}

---

*Generated by OmniProd Product Report. Re-run with `/product-report` after fixing issues to track progress.*
```

### JSON Report

Write to `.omniprod/findings/{date}-product-report.json`:

```json
{
  "type": "product-report",
  "date": "{YYYY-MM-DD}",
  "project_name": "{project_name}",
  "period": {
    "from": "{earliest_date}",
    "to": "{latest_date}"
  },
  "sources": {
    "page_reviews": {count},
    "flow_reviews": {count},
    "smoke_tests": {count},
    "files_included": ["{list of all findings file paths}"]
  },
  "summary": {
    "total_findings": {count},
    "unique_root_causes": {count},
    "critical": {count},
    "major": {count},
    "minor": {count},
    "nitpick": {count},
    "cross_page_issues": {count}
  },
  "top_issues": [
    {
      "rank": 1,
      "impact_score": 12.4,
      "severity": "critical",
      "observation": "...",
      "pages": ["..."],
      "fix": "...",
      "root_cause_id": "RC-001"
    }
  ],
  "root_causes": [
    {
      "id": "RC-001",
      "title": "...",
      "severity": "critical",
      "pages_affected": ["..."],
      "finding_ids": ["PO-001", "WF-003"],
      "suggested_fix": "...",
      "total_impact_score": 12.4
    }
  ],
  "consistency_issues": [
    {
      "id": "CONSISTENCY-001",
      "title": "...",
      "entity": "...",
      "page_a": { "path": "...", "value": "..." },
      "page_b": { "path": "...", "value": "..." },
      "likely_cause": "...",
      "impact": "..."
    }
  ],
  "page_summaries": [
    {
      "path": "/compliance",
      "verdict": "FAIL",
      "critical": 6,
      "major": 15,
      "minor": 11,
      "nitpick": 4,
      "last_reviewed": "2026-04-03"
    }
  ],
  "flow_summaries": [
    {
      "name": "Assess Compliance",
      "verdict": "FAIL",
      "assertions": 5,
      "passed": 3,
      "failed": 2,
      "last_reviewed": "2026-04-03"
    }
  ],
  "smoke_summary": {
    "pages_tested": 31,
    "ok": 28,
    "error": 3,
    "timeout": 0
  },
  "trend": {
    "direction": "DEGRADING",
    "data_points": [
      { "date": "2026-04-01", "findings": 31, "delta": null },
      { "date": "2026-04-03", "findings": 36, "delta": "+5 new, 0 fixed" }
    ]
  },
  "checklist": {
    "fix_first": [
      { "root_cause_id": "RC-001", "title": "...", "fix": "...", "impact_score": 12.4 }
    ],
    "fix_next": [],
    "fix_later": []
  }
}
```

## Phase 6: Present Results

Print to the conversation:

1. **Executive Summary table** — total findings, severities, review coverage
2. **Top 5 Issues** — the highest-impact findings with brief fix descriptions
3. **Root cause count** — how many unique root causes vs total findings (indicates fix leverage)
4. **Cross-page issues** — any consistency problems found
5. **Trend** — improving, stable, or degrading
6. **Dev checklist** — the "Fix First" items only (keep it actionable)

End with a recommendation:

If mostly clean:
> **Product is in good shape.** {N} root causes to address, all minor. See the full report at `.omniprod/reviews/{date}-product-report.md`.

If moderate issues:
> **Product needs attention.** {N} root causes, {critical_count} critical. Start with the "Fix First" checklist above. Re-run `/product-report` after fixes to verify progress.

If severe issues:
> **Product is not ready for client review.** {N} critical root causes affecting {page_count} pages. The dev checklist above is ordered by impact — work top to bottom. Re-run individual page reviews with `/product-review` after fixing each root cause cluster.

## Important Constraints

- **No browser usage**: This command aggregates existing data only. It does not navigate, screenshot, or interact with the product.
- **No new reviews**: If data is stale or missing, tell the user to run `/product-review` or `/product-smoke` — do not attempt to generate new findings.
- **Reports don't aggregate reports**: Skip any `product-report` type findings files to avoid recursive aggregation.
- **Root cause grouping is the key value**: The main insight this report provides is clustering 67 individual findings into ~28 actionable root causes. Emphasize this leverage.
- **Impact scoring enables prioritization**: Without scores, dev teams fix whatever is easiest. With scores, they fix what matters most.
- **Cross-page analysis requires a sub-agent**: Entity consistency checks involve reading and cross-referencing multiple large JSON files. Dispatch this to a sonnet sub-agent for thoroughness.
- **Trend requires multiple data points**: If only one review exists per page, state that trend data is insufficient rather than fabricating a trend.
- **Format flag**: If `--format=json`, skip the markdown report. If `--format=md` (default), generate both markdown and JSON (JSON is always needed for future aggregation). If `--format=both`, generate both (same as default behavior).
