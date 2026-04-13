---
description: "Live cross-page entity correlation — compare data across simultaneously-open pages"
argument-hint: "[--app=web] [--base-url=<url>]"
allowed-tools: ["Read", "Write", "Glob", "Grep", "Bash", "Agent", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__navigate_page", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__take_screenshot", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__take_snapshot", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__wait_for", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__list_console_messages", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__list_network_requests", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__evaluate_script", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__list_pages", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__select_page", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__new_page", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__close_page"]
---

# Product Correlate — Live Cross-Page Entity Consistency Check

You are a cross-page data consistency checker. Your job is to open multiple pages simultaneously in browser tabs, extract entity counts and values from each via `evaluate_script`, and compare them for consistency. This catches integration bugs that single-page reviews miss: stale caches, different API filters, count mismatches.

## Parse Arguments

Arguments: $ARGUMENTS

Parse:
- `--app`: Which app (default: `web`). Options: `web`, `web-hub`, `web-agent`
- `--base-url`: Base URL override. Defaults: `web` → `http://localhost:3001`, `web-hub` → `http://localhost:3002`, `web-agent` → `http://localhost:3003`

## Phase 1: Load Entity Graph

Read `.omniprod/product-map.json`. If it doesn't exist, tell the user:
> No product map found. Run `/product-map` first to define pages and flows.

Extract the `entity_graph` section. Each entity has:
- `name`: Entity type (e.g., "endpoints", "frameworks", "patches")
- `pages`: Which pages show this entity (e.g., ["/dashboard", "/endpoints", "/compliance"])
- `display_type`: How it appears (count, list, table, card, score)

Build correlation pairs: for each entity that appears on 2+ pages, create a comparison pair.

If no `entity_graph` exists in the product map, fall back to a default set of correlation checks:
- Dashboard → each list page (compare displayed counts)
- Any page with a "total" indicator → the corresponding detail page

## Phase 2: Multi-Tab Data Extraction

**IMPORTANT**: You manage ALL tabs yourself. Do NOT dispatch sub-agents for browser work — `select_page` is global state and parallel agents would race.

### Step 1: Identify unique pages

Collect all unique page URLs from the correlation pairs. Sort by importance (dashboard first).

### Step 2: Open pages in batches

For each batch of up to 4 unique pages:

1. Navigate the current tab to the first URL and wait for it to load
2. For pages 2-4: `new_page` with `background: true` for each URL
3. Wait for each page to load:
   - `select_page` to each tab
   - `wait_for` main content (heading, table, or data element)

### Step 3: Extract data from each tab

For each open tab, `select_page` then run this via `evaluate_script`:

```javascript
(function() {
  var data = {};
  var allText = document.body.innerText || '';

  // Extract count patterns: "47 endpoints", "6 frameworks", etc.
  var countPatterns = allText.match(/(\d[\d,]*)\s+(endpoint|framework|patch|cve|deployment|policy|workflow|agent|control|alert|notification)s?/gi) || [];
  countPatterns.forEach(function(match) {
    var parts = match.match(/(\d[\d,]*)\s+(\w+)/i);
    if (parts) {
      var count = parseInt(parts[1].replace(/,/g, ''), 10);
      var entity = parts[2].toLowerCase().replace(/s$/, '');
      data[entity + '_count'] = count;
    }
  });

  // Extract score patterns: "75%", "Score: 82", etc.
  var scorePatterns = allText.match(/(?:score|compliance|coverage)[:\s]+(\d+(?:\.\d+)?)\s*%?/gi) || [];
  scorePatterns.forEach(function(match) {
    var parts = match.match(/(\w+)[:\s]+(\d+(?:\.\d+)?)/i);
    if (parts) {
      data[parts[1].toLowerCase() + '_score'] = parseFloat(parts[2]);
    }
  });

  // Count table rows
  var tables = document.querySelectorAll('table tbody');
  tables.forEach(function(tbody, i) {
    data['table_' + i + '_rows'] = tbody.querySelectorAll('tr').length;
  });

  // Pagination totals
  var totalMatch = allText.match(/(?:of|total:?)\s+(\d[\d,]*)/i);
  if (totalMatch) {
    data['pagination_total'] = parseInt(totalMatch[1].replace(/,/g, ''), 10);
  }

  return JSON.stringify(data);
})();
```

Record the extracted data per page in memory.

### Step 4: Take evidence screenshots

For each tab, take a screenshot: `correlation-{page-slug}.png`

### Step 5: Close extra tabs

After extraction, `close_page` for all tabs except the original.

## Phase 3: Compare Data Across Pages

For each correlation pair (entity appearing on 2+ pages):

1. Look up the entity data extracted from each page
2. Compare the values
3. Record: MATCH or MISMATCH with specific values

### Tolerance Rules
- **Exact match** required for: entity counts, status labels
- **1% tolerance** for: percentage scores (rounding differences acceptable)
- **Table rows vs stated total**: table rows may show a page of data (e.g., 25 of 1234) — compare the stated total (pagination indicator), not the visible row count
- **Case-insensitive** for entity names and status labels

## Phase 4: Generate Report

Create output directories:
```bash
mkdir -p .omniprod/reviews .omniprod/findings
```

Write to `.omniprod/reviews/{YYYY-MM-DD}-correlation.md`:

```markdown
# Cross-Page Correlation Report

Date: {YYYY-MM-DD} | App: {app} | Base URL: {base_url}

## Summary

| Entity | Pages Compared | Result | Values |
|--------|---------------|--------|--------|
| endpoints | /dashboard, /endpoints | MATCH | 47 |
| frameworks | /dashboard, /compliance | MISMATCH | 2 vs 3 |
| compliance_score | /dashboard, /compliance | MATCH | 75% |

## Mismatches

### 1. {Entity}: {page_a} ({value_a}) vs {page_b} ({value_b})
- **Page A value**: {value_a} (from {page_a})
- **Page B value**: {value_b} (from {page_b})
- **Possible cause**: {analysis — caching, different filters, real bug}
- **Severity**: {major if counts differ, minor if within tolerance}
- **Evidence**: correlation-{page-a-slug}.png, correlation-{page-b-slug}.png

## All Matches

{List each matching entity with consistent value}

## Raw Data

{For each page, show the full extracted data object}
```

Write to `.omniprod/findings/{YYYY-MM-DD}-correlation.json`:
```json
{
  "review_id": "{date}-correlation",
  "date": "{ISO date}",
  "app": "{app}",
  "type": "correlation",
  "overall_verdict": "PASS or FAIL",
  "pairs_checked": N,
  "matches": N,
  "mismatches": N,
  "findings": [
    {
      "id": "COR-001",
      "severity": "major",
      "element": "{entity} count",
      "observation": "{page_a} shows {value_a}, {page_b} shows {value_b}",
      "suggestion": "Investigate whether {page_a} uses cached data or a different API filter",
      "source": "correlation",
      "pages": ["{page_a}", "{page_b}"],
      "status": "open",
      "first_seen": "{today}",
      "last_seen": "{today}"
    }
  ]
}
```

## Phase 5: Print Summary

```
=== Cross-Page Correlation ===
Entities checked: {N}
Matches: {N}
Mismatches: {N}

{For each mismatch:}
  ❌ {entity}: {page_a} ({value_a}) ≠ {page_b} ({value_b})

Full report: .omniprod/reviews/{date}-correlation.md
```

If all match:
> **Cross-page data is consistent.** All {N} entity values match across pages.

If mismatches found:
> **Found {N} cross-page inconsistencies.** Review the report and determine if these are caching issues, filter differences, or real bugs.

## Important Constraints

- **Single agent**: Do NOT dispatch sub-agents for browser work. `select_page` is global state — parallel agents would race.
- **Max 4 tabs**: Never open more than 4 tabs simultaneously. Process in batches.
- **Close tabs after each batch**: Always close extra tabs before opening new ones.
- **No interactions**: Only navigate and extract data. Don't click, hover, or fill anything.
- **Speed over depth**: Target ~5 minutes for 10 correlation checks. Don't over-analyze individual pages.
- **Idempotent**: Running twice should produce comparable results.
