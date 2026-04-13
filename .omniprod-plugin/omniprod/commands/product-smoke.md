---
description: "Quick smoke test — screenshot + console check every page in the product"
argument-hint: "[--app=web|web-hub|web-agent] [--base-url=<url>]"
allowed-tools: ["Read", "Write", "Bash", "Glob", "Grep", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__navigate_page", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__take_screenshot", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__take_snapshot", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__wait_for", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__list_console_messages", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__list_network_requests", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__list_pages", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__select_page", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__evaluate_script", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__new_page", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__close_page"]
---

# Product Smoke Test — Every Page, One Pass

You are a fast, mechanical smoke tester. Your job is to visit every page in the product, take a screenshot, check for console errors and network failures, and produce a status report. No interactions, no perspectives, no sub-agents. Speed is the priority.

## Parse Arguments

Arguments: $ARGUMENTS

Parse:
- `--app`: Which app to smoke test (default: `web`). Options: `web`, `web-hub`, `web-agent`
- `--base-url`: Base URL override (optional). Defaults based on app:
  - `web` → `http://localhost:3001`
  - `web-hub` → `http://localhost:3002`
  - `web-agent` → `http://localhost:3003`

If invalid arguments, show usage:
```
Usage: /product-smoke [options]

Examples:
  /product-smoke
  /product-smoke --app=web-hub
  /product-smoke --app=web --base-url=http://localhost:3001
```

## Phase 1: Build Page List

### Step 1: Read Product Map (preferred)

Check if `.omniprod/product-map.json` exists. If it does, read it and extract all pages for the target app.

Expected format:
```json
{
  "apps": {
    "web": {
      "pages": [
        { "path": "/dashboard", "name": "Dashboard", "priority": 1 },
        { "path": "/endpoints", "name": "Endpoints", "priority": 1 }
      ]
    }
  }
}
```

### Step 2: Fall Back to Routes (if no product map)

If `.omniprod/product-map.json` doesn't exist, read the routes file directly:
- `web` → `web/src/app/routes.tsx`
- `web-hub` → `web-hub/src/app/routes.tsx`
- `web-agent` → `web-agent/src/app/routes.tsx`

Parse every route path from the routes file. Build a page list sorted by navigation order (dashboard first, settings last). Exclude:
- Wildcard/catch-all routes (`*`)
- Dynamic segments that require IDs (e.g., `/endpoints/:id`) — unless you can construct a valid URL from seed data
- Auth callback routes

### Step 3: Sort by Priority

Sort pages: dashboard and overview pages first, then core feature pages, then settings/admin pages last.

## Phase 2: Setup

### Create Output Directory

```bash
DATE=$(date +%Y-%m-%d)
mkdir -p .omniprod/screenshots/smoke/${DATE}
```

### Initialize Tracking

Create an in-memory results array to track each page's status:
```
page_path, page_name, status (OK|ERROR|TIMEOUT), console_errors (count), network_errors (count), screenshot_file, notes
```

## Phase 3: Smoke Each Page

Group all pages into batches of 4. Process each batch using multi-tab loading to improve throughput. Do NOT skip pages.

### Read Assertions Runner

Before starting the batches, read the assertions script:
- Read `.omniprod-plugin/omniprod/scripts/assertions-runner.js`
- Store its contents in memory — you will inject it via `evaluate_script` on every page

### Per-Batch Sequence

For each batch of up to 4 pages:

1. **Open tabs**:
   - Navigate the current (existing) tab to the first page URL via `navigate_page`
   - For pages 2–4 in the batch: call `new_page(url, background: true)` to open each in a background tab

2. **Wait for all tabs to load**:
   - `select_page(pageId)` for each tab in the batch
   - `wait_for` main content to appear (page title, heading, or main content area)
   - Use a 2-second timeout — if a page doesn't load, mark it TIMEOUT

3. **Per-tab capture** — for each tab in the batch, in order:
   a. `select_page(pageId)`
   b. **Screenshot**: `take_screenshot` → save to `.omniprod/screenshots/smoke/{date}/smoke-{page-slug}.png`
      - Page slug: lowercase path with `/` replaced by `-`, leading `-` removed
      - Example: `/compliance` → `smoke-compliance.png`
      - Example: `/settings/license` → `smoke-settings-license.png`
      - Example: `/admin/roles` → `smoke-admin-roles.png`
   c. **Assertions**: `evaluate_script` with the contents of `assertions-runner.js`
      - Parse the returned JSON result
      - Record `assertions_passed` and `assertions_failed` counts
   d. **Console Check**: `list_console_messages` — count errors only (ignore warnings and info). Record the count and any error messages.
   e. **Network Check**: `list_network_requests` — count any responses with status 4xx or 5xx. Ignore:
      - `favicon.ico` 404s (common, not a real issue — still count them but flag as "known")
      - Requests to external domains (CDN, analytics, etc.)
   f. **Record Result**: Add to the results array:
      - Status: `OK` if no console errors (excluding favicon), no network errors (excluding favicon), and no failed assertions
      - Status: `ERROR` if any non-favicon console errors, network failures, or assertion failures
      - Status: `TIMEOUT` if the page didn't load

4. **Close extra tabs**: Call `close_page` for each background tab opened in this batch (pages 2–4). Keep only the original tab open for the next batch.

### Pacing

Aim for ~20-30 seconds per batch (not per page). Do not over-analyze. The goal is breadth, not depth. If a page is broken, note it and move on.

### Error Recovery

If navigation fails entirely (browser crash, connection refused):
1. Try `list_pages` to see if the browser is still alive
2. If yes, `select_page` back to a working page and continue
3. If no, note the failure and skip remaining pages

## Phase 4: Generate Report

After all pages are tested, generate two outputs.

### Markdown Report

Write to `.omniprod/reviews/{date}-smoke.md`:

```markdown
# Smoke Test Report

Date: {YYYY-MM-DD} | App: {app} | Base URL: {base_url} | Pages: {total_count}

## Results

| Page | Status | Console Errors | Network Errors | Assertions | Screenshot |
|------|--------|----------------|----------------|------------|------------|
| /dashboard | OK | 0 | 0 | 12/12 | smoke-dashboard.png |
| /endpoints | OK | 0 | 0 | 10/10 | smoke-endpoints.png |
| /compliance | OK | 1 (favicon 404) | 1 (favicon) | 8/8 | smoke-compliance.png |
| /settings/broken | ERROR | 3 | 2 (500 on /api/v1/settings) | 4/9 | smoke-settings-broken.png |

## Summary

- **Pages tested**: {total}
- **All OK**: {ok_count}
- **With errors**: {error_count}
- **Timed out**: {timeout_count}
- **Console errors total**: {console_total} ({console_excluding_favicon} excluding favicon)
- **Network failures total**: {network_total} ({network_excluding_favicon} excluding favicon)

## Pages Needing Attention

{For each page with ERROR or TIMEOUT status, list:}

### {N}. {page_path} — {brief issue description}
- **Status**: ERROR | TIMEOUT
- **Console errors**: {list each error message}
- **Network failures**: {list each failed request with status code and URL}
- **Screenshot**: {filename}

## Clean Pages

{List of all OK pages — just the names, no details needed}
```

### JSON Output

Write to `.omniprod/findings/{date}-smoke.json`:

```json
{
  "type": "smoke",
  "date": "{YYYY-MM-DD}",
  "app": "{app}",
  "base_url": "{base_url}",
  "pages_tested": {total},
  "summary": {
    "ok": {ok_count},
    "error": {error_count},
    "timeout": {timeout_count},
    "console_errors_total": {count},
    "network_errors_total": {count}
  },
  "pages": [
    {
      "path": "/dashboard",
      "name": "Dashboard",
      "status": "ok",
      "screenshot": "smoke-dashboard.png",
      "console_errors": [],
      "network_errors": [],
      "notes": ""
    },
    {
      "path": "/settings/broken",
      "name": "Settings (Broken)",
      "status": "error",
      "screenshot": "smoke-settings-broken.png",
      "console_errors": [
        "TypeError: Cannot read properties of undefined (reading 'map')"
      ],
      "network_errors": [
        { "url": "/api/v1/settings", "status": 500, "method": "GET" }
      ],
      "notes": "API returning 500, component crashes"
    }
  ]
}
```

## Phase 5: Present Results

Print a concise summary to the conversation:

1. The summary table (pages tested, OK, errors, timeouts)
2. The "Pages Needing Attention" section with details
3. Total time elapsed (approximate)

If all pages are OK:
> **Smoke test passed.** All {N} pages loaded without errors.

If any pages have errors:
> **Smoke test found issues on {N} pages.** See `.omniprod/reviews/{date}-smoke.md` for the full report. Run `/product-review {worst_page_url}` for a deep review of the most critical failures.

## Important Constraints

- **No interactions**: Do not click, hover, fill, or type anything. Just navigate and screenshot.
- **No sub-agents**: This runs entirely in the main thread.
- **No perspectives**: No UX/QA/enterprise analysis. Just load status + errors.
- **No Lighthouse**: Skip performance audits — this is about page-level health only.
- **Speed over depth**: If something is slow, note it and move on. Target 15-20 minutes for 30 pages.
- **Idempotent**: Running this twice should produce comparable results (no side effects).
