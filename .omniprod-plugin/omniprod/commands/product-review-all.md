---
description: "Full product review — browser-free orchestrator with incremental mode, tiered deep reviews, and unified product report"
argument-hint: "[--app=web] [--base-url=<url>] [--skip-smoke] [--tier=1|2|3] [--incremental]"
allowed-tools: ["Read", "Write", "Glob", "Grep", "Bash", "Agent", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__navigate_page", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__take_screenshot", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__take_snapshot", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__click", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__hover", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__fill", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__fill_form", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__press_key", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__type_text", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__drag", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__wait_for", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__resize_page", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__list_console_messages", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__list_network_requests", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__get_network_request", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__lighthouse_audit", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__evaluate_script", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__list_pages", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__select_page", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__new_page", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__close_page", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__handle_dialog", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__emulate", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__upload_file"]
---

# Product Review All — Browser-Free Orchestrator

You are a LONG-RUNNING product quality orchestrator. Your job is to coordinate a complete, multi-phase product review: map the product, health-scan every page, correlate cross-page data, deep-review priority pages, and produce a unified product report.

**CRITICAL ARCHITECTURAL RULE: You (the orchestrator) NEVER use Chrome DevTools tools directly.** Every browser operation is dispatched to a sub-agent. Your job is to coordinate, read results from disk, and dispatch next phases. This is the anti-context-rot strategy — your context stays small.

This command may take 2-6 hours depending on product size. You are methodical, resumable, and anti-context-rot by design.

## Parse Arguments

Arguments: $ARGUMENTS

Parse:
- `--app`: Which app to review. Options: `web`, `web-hub`, `web-agent`. Default: `web`
- `--base-url`: Base URL for the app. Default: inferred from app (`web` = `http://localhost:3001`, `web-hub` = `http://localhost:3002`, `web-agent` = `http://localhost:3003`)
- `--skip-smoke`: Skip Phase 1 (health scan). Useful when re-running after fixes.
- `--tier`: Only deep-review pages up to this tier. `1` = critical only, `2` = critical + important, `3` = all. Default: `2`
- `--incremental`: Only deep-review pages affected by code changes since the last review. See Incremental Mode section.

If the user just typed `/product-review-all` with no arguments, proceed with defaults (app=web, tier=2, all phases, non-incremental).

## Resource Directory

**RESOURCE_DIR** = `.omniprod-plugin/omniprod/`

All paths below reference files relative to this directory:
- Perspectives: `RESOURCE_DIR/perspectives/{name}.md`
- Standards: `RESOURCE_DIR/standards/{name}.md`
- References: `RESOURCE_DIR/references/{name}.md`
- Scripts: `RESOURCE_DIR/scripts/`

---

## Setup & Context (~2 min)

### Auto-Init

If `.omniprod/` does not exist, create the full structure:
```bash
mkdir -p .omniprod/{reviews,findings,screenshots/current,screenshots/archive,standards-overrides}
```

If `.omniprod/config.json` does not exist, auto-detect the project (read CLAUDE.md, package.json) and write the default config.

### Load Configuration

1. Read `.omniprod/config.json` for active perspectives and settings
2. Read `CLAUDE.md` for product identity, architecture, milestone status
3. Read `docs/roadmap.md` (if it exists) for current priorities

### Determine Today's Run ID

```
RUN_ID = {YYYY-MM-DD}  (today's date)
```

Check `.omniprod/reviews/` for existing files with today's date. If any phase was already completed today, note which phases can be skipped (resumability).

### Incremental Mode — Affected Page Detection

**Only if `--incremental` is set:**

1. Read `.omniprod/last-review-commit` (if it exists). If missing, warn: "No previous review commit found. Running full review." and disable incremental mode.
2. Run `git diff --name-only {commit}..HEAD` to get all changed files since the last review.
3. Map changed files to affected pages using `.omniprod/product-map.json`:
   - Files matching `{app}/src/pages/{slug}/` — that page is affected
   - Files matching `packages/ui/` — ALL pages are affected (shared component change; disable incremental, run full)
   - Files matching `internal/server/api/` or `internal/hub/api/` — map to pages that use the affected endpoint via `api_hooks_used` in the product map
   - Files matching `{app}/src/api/hooks/` — map to pages that import those hooks
   - Files matching `{app}/src/components/` — map to pages that use those components
4. Store the affected page slugs in a variable `AFFECTED_PAGES` for use in Phase 3.
5. If no pages are affected, print: "No UI-affecting changes detected since last review. Nothing to do." and exit.

Print:
```
OmniProd Full Product Review
App: {app}
Base URL: {base-url}
Tier: {tier}
Mode: {incremental | full}
{if incremental: Affected pages: {N} (changes since {commit_short})}
Phases: {list active phases, noting any skipped}
Starting at: {timestamp}
```

---

## Phase 0: Product Map (~5 min)

Check if `.omniprod/product-map.json` exists and is recent (less than 7 days old based on file modification time).

**If missing or stale:** Dispatch ONE sub-agent to build the product map:

```
Agent:
  model: "sonnet"
  name: "po-product-map"
  description: "Build product map for {app}"
  prompt: |
    You are building a product map for the {app} application.

    1. Read route definitions: {app}/src/app/routes.tsx
    2. Read API hooks: glob {app}/src/api/hooks/*.ts
    3. Read page components: glob {app}/src/pages/**/*.tsx
    4. Read layout files: glob {app}/src/app/layout/*.tsx
    5. Read shared UI components: glob packages/ui/src/**/*.tsx

    Generate .omniprod/product-map.json with:
    - **Page registry**: every route with name, route, component_file, priority (critical/important/peripheral), tier (1/2/3), entities_shown, entities_mutated, api_hooks_used, shared_components
    - **Entity graph**: relationships between entities (endpoint has patches, patch has CVEs, etc.)
    - **Business rules**: invariants the product must satisfy
    - **Shared components**: sidebar, topbar, theme, notifications — with component file paths

    Priority classification:
    - critical (tier 1): dashboard, endpoints, patches, cves, compliance, deployments, workflows
    - important (tier 2): policies, schedules, notifications, settings, audit, tags
    - peripheral (tier 3): agent-downloads, admin sub-pages, help pages
```

Wait for the sub-agent to complete.

**If exists and recent:** read it and proceed.

Print:
```
Product map: {N} pages, {N} business rules
  Tier 1 (critical): {N} pages
  Tier 2 (important): {N} pages
  Tier 3 (peripheral): {N} pages
```

---

## Phase 1: Health Scan — All Pages (~10 min)

**Skip if `--skip-smoke` is set or if `.omniprod/reviews/{RUN_ID}-health-scan.json` already exists.**

Dispatch ONE sub-agent for the entire health scan:

```
Agent:
  model: "sonnet"
  name: "po-health-scan"
  description: "Health scan all pages for {app}"
  prompt: |
    You are performing a health scan of every page in the product.

    Read .omniprod/product-map.json to get the full page list.

    ## Multi-Tab Batch Strategy

    Process pages in batches of 4:
    1. Open 4 pages using new_page(url, background: true) for each
    2. For each page (select_page to switch):
       a. wait_for key content (heading or data table, 5s timeout)
       b. Run assertions: Read .omniprod-plugin/omniprod/scripts/assertions-runner.js, then evaluate_script to execute it
       c. take_screenshot → save as .omniprod/screenshots/current/smoke-{page-slug}.png
       d. list_console_messages → record errors (filter: ignore React dev warnings, favicon 404s)
       e. list_network_requests → record failed requests (4xx, 5xx)
    3. Close extra tabs (close_page), keeping only the first tab
    4. Repeat for next batch

    ## Per-Page Recording

    For each page, record:
    - path: the route
    - status: ok | console-errors | network-failures | load-failure | blank-page
    - assertions: { total, passed, failed }
    - console_errors: count of real errors (not warnings)
    - network_errors: count of failed requests
    - screenshot: filename

    ## Output

    Write .omniprod/reviews/{RUN_ID}-health-scan.json:
    {
      "run_id": "{RUN_ID}",
      "app": "{app}",
      "scanned_at": "{ISO timestamp}",
      "pages": [
        {
          "path": "/dashboard",
          "status": "ok",
          "assertions": {"total": 10, "passed": 9, "failed": 1},
          "console_errors": 0,
          "network_errors": 0,
          "screenshot": "smoke-dashboard.png",
          "load_time_ms": null
        }
      ],
      "summary": {
        "total": N,
        "ok": N,
        "console_errors": N,
        "network_failures": N,
        "load_failures": N
      }
    }

    Replace {RUN_ID} with: {RUN_ID}

    When complete, print: "Health scan complete: {N} pages scanned, {N} issues found."
```

Wait for the sub-agent to complete. Read `.omniprod/reviews/{RUN_ID}-health-scan.json`.

Print summary:
```
Health Scan: {N} pages scanned
  OK: {N}
  Console errors: {N}
  Network failures: {N}
  Load failures: {N}
  Pages needing attention:
    /{route} — {status} ({detail})
    ...
```

---

## Phase 2: Cross-Page Correlation (~5 min)

**Skip if `.omniprod/reviews/{RUN_ID}-correlation.json` already exists.**

Dispatch ONE sub-agent for cross-page correlation:

```
Agent:
  model: "opus"
  name: "po-correlation"
  description: "Cross-page entity correlation for {app}"
  prompt: |
    You are checking cross-page data consistency. Open multiple pages simultaneously and compare entity counts, scores, and statuses across them.

    Read .omniprod/product-map.json to understand the entity graph — which entities appear on which pages.

    ## Strategy

    For each entity that appears on 2+ pages:
    1. Open the relevant pages in parallel tabs using new_page(url, background: true)
    2. On each page, use evaluate_script to extract:
       - Entity counts (e.g., total endpoints, total patches)
       - Key metrics (e.g., compliance score, deployment success rate)
       - Status distributions (e.g., how many endpoints online vs offline)
    3. Compare values across pages — flag mismatches
    4. Take screenshots of mismatched pages as evidence
    5. Close extra tabs when done with each batch

    ## Key Correlation Pairs

    Check at minimum:
    - Dashboard counts vs list page counts (endpoints, patches, CVEs)
    - Compliance scores on dashboard vs compliance page
    - Deployment counts on dashboard vs deployments page
    - Entity totals in sidebar badges (if any) vs page content

    ## Output

    Write .omniprod/reviews/{RUN_ID}-correlation.json:
    {
      "run_id": "{RUN_ID}",
      "correlations": [
        {
          "entity": "endpoints",
          "pages_compared": ["/dashboard", "/endpoints"],
          "values": {"/dashboard": 42, "/endpoints": 42},
          "match": true,
          "screenshot": null
        },
        {
          "entity": "compliance_score",
          "pages_compared": ["/dashboard", "/compliance"],
          "values": {"/dashboard": "85%", "/compliance": "82%"},
          "match": false,
          "screenshot": "correlation-compliance-mismatch.png",
          "finding": "Compliance score on dashboard (85%) differs from compliance page (82%)"
        }
      ],
      "summary": {
        "total_checks": N,
        "matches": N,
        "mismatches": N
      }
    }

    Replace {RUN_ID} with: {RUN_ID}

    When complete, print: "Correlation check complete: {N} checks, {N} mismatches found."
```

Wait for the sub-agent to complete. Read `.omniprod/reviews/{RUN_ID}-correlation.json`.

Print:
```
Cross-Page Correlation: {N} entity checks
  Matches: {N}
  Mismatches: {N}
  {for each mismatch: {entity}: {detail}}
```

---

## Phase 3: Deep Reviews — Tiered (~12 min each)

### Determine Review Set

Filter pages from the product map by `--tier`:
- Tier 1: pages where `priority == "critical"`
- Tier 2: pages where `priority == "critical"` OR `priority == "important"`
- Tier 3: all pages

**If `--incremental`**: further filter to only pages in `AFFECTED_PAGES`. Carry forward findings for unchanged pages by copying previous findings files:
- For each unchanged page, check if `.omniprod/findings/*-{page-slug}.json` exists from a previous run
- If yes, copy it to `.omniprod/findings/{RUN_ID}-{page-slug}.json` (carried forward, not re-reviewed)
- Print: "Carrying forward {N} pages from previous review (unchanged)"

### Tiered Perspective Selection

| Page Priority | Explore Model | Perspectives | Perspective Model |
|---|---|---|---|
| Critical | opus | UX + Enterprise + QA + PM + End User (5) | opus |
| Important | sonnet | UX + Enterprise + QA (3) | sonnet |
| Peripheral | — | assertions only (from Phase 1 health scan) | — |

For peripheral pages: extract their assertion results from the health scan and save as their findings file. No deep review needed.

### ANTI-CONTEXT-ROT MEASURES (NON-NEGOTIABLE)

1. **After every 2 page reviews**: re-read `.omniprod/product-map.json` to refresh the full task list
2. **Write findings to disk IMMEDIATELY** after each page review — never accumulate
3. **Archive screenshots** between pages: `bash .omniprod-plugin/omniprod/scripts/cleanup-screenshots.sh --archive`
4. **Keep a running progress counter** and print it after each page
5. **Delegate aggregation**: for critical pages with 5 perspectives, dispatch aggregation to a sub-agent to avoid accumulating perspective output in orchestrator context

### Per-Page Deep Review Pipeline

For each page in the review set (sequential):

**Skip if `.omniprod/findings/{RUN_ID}-{page-slug}.json` already exists (resumability).**

Archive screenshots from previous page:
```bash
bash .omniprod-plugin/omniprod/scripts/cleanup-screenshots.sh --archive
```

#### Step 1: Intelligence Gathering (sub-agent, background)

Dispatch ONE sub-agent (model: {explore_model}, name: "po-intel-{page-slug}", run_in_background: true):

```
You are building intelligence for a product review of the "{page-name}" page at {base-url}{route}.

Read these files:
1. CLAUDE.md — product identity, architecture
2. .omniprod/config.json — tech stack, design system
3. {app}/src/app/routes.tsx — route definitions
4. The page component — use Grep to find it in {app}/src/pages/
5. API hooks used by the page — check {app}/src/api/hooks/
6. Backend handlers — grep internal/server/api/v1/ or internal/hub/api/v1/

Write .omniprod/screenshots/current/00-business-context.md with:
- What this feature does (2-3 sentences)
- Key user workflows
- CRUD operations available
- API endpoints used
- Business rules (testable assertions from source code)
- Data relationships
- Entity types on this page

Write .omniprod/screenshots/current/coverage-targets.json with a checklist of everything the browser agent should cover (page-load, scroll, tabs, modals, detail views, hover states, responsive, empty states, actions, error states — plus page-specific targets).

Run the assertion generator if available:
python3 .omniprod-plugin/omniprod/scripts/generate-assertions.py --app {app} --page {route} --output .omniprod/screenshots/current/assertion-defs.json

Report when done: "Intelligence gathering complete for {page-name}."
```

#### Step 2: Explore + Assert (sub-agent, browser)

Dispatch ONE sub-agent (model: {explore_model}, name: "po-explorer-{page-slug}"):

```
You are exploring the "{page-name}" page at {base-url}{route} for a product quality review. Systematically cover all targets and run automated assertions.

## Initial Capture

1. Navigate to: {base-url}{route}
2. Wait for page to load (heading or main content, 5s)
3. take_screenshot → .omniprod/screenshots/current/00-initial.png
4. take_snapshot (verbose: true) → .omniprod/screenshots/current/00-snapshot.txt
5. list_console_messages → .omniprod/screenshots/current/00-console.txt
6. list_network_requests → .omniprod/screenshots/current/00-network.txt

## Run Assertions

7. Read .omniprod-plugin/omniprod/scripts/assertions-runner.js
8. Read .omniprod/screenshots/current/assertion-defs.json (wait up to 30s if not yet available)
9. evaluate_script: window.__OMNIPROD_CUSTOM_ASSERTIONS = {assertion-defs content};
10. evaluate_script: {assertions-runner.js content}
11. Write result to .omniprod/screenshots/current/assertion-results.json

## Load Intelligence

12. Read .omniprod/screenshots/current/coverage-targets.json (wait if needed)
13. Read .omniprod/screenshots/current/00-business-context.md

## Plan-As-You-Go Exploration

Systematically cover targets from coverage-targets.json:
- For each target: perform the action, take_screenshot with descriptive name
- For complex states: also take_snapshot
- Log each action to .omniprod/screenshots/current/exploration-log.jsonl (one JSON line per action)
- SKIP shared components (sidebar, topbar, theme) — those are covered separately

Element targeting: use CSS selectors or text content, NEVER UIDs from old snapshots.

## Checkpoints

Every 15 actions, re-read coverage-targets.json from disk.

## Responsive Testing

Use multi-tab approach:
- new_page at same URL with 1024x768 viewport, screenshot, close_page
- new_page at same URL with 768x1024 viewport, screenshot, close_page

## Final Evidence

- lighthouse_audit categories: ["accessibility", "best-practices", "seo"], mode: "snapshot"
- list_network_requests → post-capture-network.txt
- list_console_messages → post-capture-console.txt

Report: total screenshots, coverage targets completed, assertion failures.

Write final summary to exploration-log.jsonl:
{"step": "DONE", "total_screenshots": N, "targets_completed": N, "targets_total": N, "assertion_failures": N}
```

Wait for the explorer sub-agent to complete.

#### Step 3: Perspective Reviews (parallel sub-agents, no browser)

Build the evidence list (orchestrator reads from disk):
1. Glob all `.png` files in `.omniprod/screenshots/current/`
2. Read `.omniprod/screenshots/current/assertion-results.json`
3. Read `.omniprod/screenshots/current/00-business-context.md`
4. Read `.omniprod/screenshots/current/exploration-log.jsonl`

Parse assertion failures into a list:
```
ALREADY DETECTED BY AUTOMATED ASSERTIONS (verify but do NOT re-report):
- [A-001] {name}: {failure message} (element: {selector})
...
```

Dispatch ALL perspectives for this page in ONE message (parallel execution).

For each perspective `{name}`, use the model from the tiered selection table, name: `"po-{name}-{page-slug}"`:

```
You are conducting a product review of the "{page-name}" page from a specific stakeholder perspective.

## Business Context

{PASTE contents of 00-business-context.md}

## Exploration Evidence

{PASTE contents of exploration-log.jsonl, or summary if >200 lines}

## Automated Assertion Results

{PASTE assertion failures list}

These are already detected by automation. VERIFY in screenshots but do NOT create new findings for them. Focus on issues requiring HUMAN JUDGMENT: visual quality, UX patterns, business logic, consistency, enterprise readiness.

## Your Perspective

Read: .omniprod-plugin/omniprod/perspectives/{name}.md

## Product Standards

Read ALL:
- .omniprod-plugin/omniprod/standards/visual.md
- .omniprod-plugin/omniprod/standards/interaction.md
- .omniprod-plugin/omniprod/standards/data-integrity.md
- .omniprod-plugin/omniprod/standards/consistency.md
- .omniprod-plugin/omniprod/standards/enterprise.md
- .omniprod-plugin/omniprod/standards/accessibility.md

Check for overrides: .omniprod/standards-overrides/

## Previous Findings

Check .omniprod/findings/ for previous *{page-slug}*.json. If found, note which are still present vs fixed.

## SKIP Shared Components

Do NOT report findings about sidebar, topbar, or theme — those are reviewed separately. Only report if a shared component behaves INCORRECTLY on THIS specific page (e.g., wrong active state).

## Evidence to Review

READ each screenshot:
{list every .png file path}

Also READ:
- .omniprod/screenshots/current/00-snapshot.txt
- .omniprod/screenshots/current/00-console.txt
- .omniprod/screenshots/current/00-network.txt

## Output Format

For EACH finding:
### {PREFIX}-{NNN}: {title}
- **Severity**: critical | major | minor | nitpick
- **Element**: {specific element or area}
- **Observation**: {what is wrong — reference which screenshot}
- **Suggestion**: {specific fix}
- **Standard Violated**: {standard file + section}
- **Screenshot**: {filename}

PREFIX codes: UX (ux-designer), EB (enterprise-buyer), QA (qa-engineer), PM (product-manager), EU (end-user).

**VERDICT: PASS** or **VERDICT: FAIL**
```

Wait for all perspective sub-agents to complete.

#### Step 4: Aggregate Page Findings

**For critical pages (5 perspectives):** Dispatch aggregation to a sub-agent to avoid context growth in the orchestrator:

```
Agent:
  model: "sonnet"
  name: "po-agg-{page-slug}"
  description: "Aggregate findings for {page-name}"
  prompt: |
    You are aggregating product review findings for the "{page-name}" page.

    ## Step 1: Convert Assertion Failures

    Read .omniprod/screenshots/current/assertion-results.json.
    For each FAILED assertion, create a finding with severity mapping: error→critical, warning→major, info→minor.

    ## Step 2: Parse Perspective Findings

    The following perspective reviews were conducted. Parse their findings:
    {list perspective names and their output — paste or reference files}

    ## Step 3: Deduplicate

    - Same element + same observation from multiple perspectives = merge, list all perspectives
    - Assertion finding matching a perspective finding = merge, source: "both"

    ## Step 4: Assign IDs and Sort

    Assign unified IDs: PO-{page-slug}-001, PO-{page-slug}-002, etc.
    Sort: critical → major → minor → nitpick.
    Add tracking: first_seen (today or carried forward), last_seen (today), status: "open".

    ## Step 5: Write Output

    Write .omniprod/findings/{RUN_ID}-{page-slug}.json with the standard findings schema:
    {
      "review_id": "{RUN_ID}-{page-slug}",
      "url": "{base-url}{route}",
      "date": "{RUN_ID}",
      "page_name": "{page-name}",
      "overall_verdict": "PASS|FAIL",
      "perspectives": { ... },
      "findings": [ ... ],
      "stats": { "critical": N, "major": N, "minor": N, "nitpick": N, "total": N }
    }

    Replace {RUN_ID} with: {RUN_ID}

    Print: "{page-name}: {VERDICT} — {N} critical, {N} major, {N} minor, {N} nitpick"
```

**For important pages (3 perspectives):** The orchestrator can aggregate directly since the output is smaller. Follow the same steps: convert assertions, parse perspectives, deduplicate, assign IDs, write to disk.

### Progress Printing

After each page completes, print:
```
Page "/{route}": {PASS|FAIL} ({N} critical, {N} major) | Progress: {done}/{total}
```

### Anti-Context-Rot Checkpoint

After every 2 pages:
1. Re-read `.omniprod/product-map.json`
2. Verify the remaining page list
3. Confirm progress count is accurate

---

## Phase 4: Product Report (~3 min)

### 4a. Impact Scoring

Run the impact scorer across ALL findings files from today:
```bash
python3 .omniprod-plugin/omniprod/scripts/impact-scorer.py .omniprod/findings/{RUN_ID}-*.json --output .omniprod/findings/{RUN_ID}-product-scored.json
```

If the script does not exist or fails, proceed with manual aggregation (read all findings JSON files, sort by severity).

### 4b. Root Cause Grouping

Read all findings from `.omniprod/findings/{RUN_ID}-*.json` and group by root cause:
- Same underlying issue on multiple pages = one root cause
- Examples: "missing focus indicators" (accessibility), "seed data artifacts" (data quality), "inconsistent empty states" (UX)

### 4c. Correlation Findings

Read `.omniprod/reviews/{RUN_ID}-correlation.json` and convert mismatches into findings:
- Each mismatch becomes a finding with severity "major" and scope "cross-page"
- Assign IDs: `COR-001`, `COR-002`, etc.

### 4d. Generate Product Report

Write `.omniprod/reviews/{RUN_ID}-product-report.md`:

```markdown
# OmniProd Product Report — {product_name} ({app})

**Date**: {RUN_ID}
**Duration**: {elapsed time}
**Reviewer**: OmniProd Automated Review
**Mode**: {full | incremental (since {commit_short})}

---

## Executive Summary

{2-3 sentences: overall product quality state, biggest risks, readiness verdict}

## Verdict: {READY TO SHIP | NOT READY TO SHIP | CONDITIONALLY READY}

Criteria:
- READY TO SHIP: zero critical findings, fewer than 5 major findings, all correlations match
- CONDITIONALLY READY: zero critical, some major findings, most correlations match
- NOT READY TO SHIP: any critical findings OR multiple correlation mismatches

---

## Top 10 Issues by Impact

| # | ID | Severity | Issue | Pages Affected | Root Cause |
|---|-----|----------|-------|----------------|------------|
| 1 | ... | critical | ...   | 5              | ...        |

---

## Root Cause Analysis

{For each root cause group:}
### {Root Cause Title} ({N} findings, {severity})
- **Affected pages**: {list}
- **Description**: {underlying issue}
- **Fix approach**: {address root cause, not symptoms}

---

## Cross-Page Consistency Issues

{From correlation phase — entity mismatches with evidence}

---

{if incremental:}
## Changes Since Last Review

- **Previous review commit**: {commit_short}
- **Current HEAD**: {head_short}
- **Pages re-reviewed**: {N} ({list slugs})
- **Pages carried forward**: {N} (unchanged)
- **Reason for re-review**: {list changed files mapped to pages}
{end if}

---

## Coverage Summary

| Phase | Status | Details |
|-------|--------|---------|
| Health scan | {N}/{N} pages | {N} issues found |
| Correlation | {N} checks | {N} mismatches |
| Deep page reviews | {N}/{N} pages (tier {tier}) | {N} total findings |

---

## Per-Page Summary

| Page | Verdict | Critical | Major | Minor | Nitpick |
|------|---------|----------|-------|-------|---------|
| /dashboard | PASS | 0 | 1 | 2 | 0 |
| /compliance | FAIL | 3 | 5 | 2 | 1 |
| ... | ... | ... | ... | ... | ... |

---

## Trend

{If a previous product report exists in .omniprod/reviews/:}
- Previous total findings vs current
- Fixed findings
- New findings
- Regressed findings
{If no previous report: "First full product review — no trend data available."}

---

## Prioritized Dev Checklist

Critical (fix before any demo):
- [ ] {ID}: {description} — {affected pages}

Major (fix before ship):
- [ ] {ID}: {description} — {affected pages}

Minor (fix when convenient):
- [ ] ...
```

### 4e. Save Incremental State

Save current HEAD for future incremental runs:
```bash
git rev-parse HEAD > .omniprod/last-review-commit
```

### 4f. Save Findings Summary

Write the aggregated scored findings to `.omniprod/findings/{RUN_ID}-product-scored.json` if the impact scorer was not available.

### 4g. Save to Memory

Save a brief review summary to Claude's auto-memory:
- Title: "OmniProd: Product review — {app}"
- Content: "{verdict} — {N} root causes, {N} critical, {N} major, {N} minor findings. Top issues: {top 3 root causes}."

---

## Print Final Summary

Print this exact format (fill in actual values):

```
=============================================
     OmniProd Full Product Review
=============================================

Product: {product_name} ({app})
Date: {RUN_ID}
Duration: {elapsed time estimate}
Mode: {full | incremental}

Coverage:
  Health scan: {N}/{N} pages
  Correlation checks: {N} entities across {N} page pairs
  Deep page reviews: {N}/{N} (tier {tier})
  {if incremental: Re-reviewed: {N} pages | Carried forward: {N} pages}

Findings:
  Total: {N} findings -> {N} unique root causes
  Critical: {N}
  Major: {N}
  Minor: {N}
  Nitpick: {N}

Cross-Page Mismatches: {N}

Top 3 Root Causes:
  1. {root cause} ({N} pages, {severity})
  2. {root cause} ({N} pages, {severity})
  3. {root cause} ({N} pages, {severity})

Verdict: {READY TO SHIP | NOT READY TO SHIP | CONDITIONALLY READY}

Full report: .omniprod/reviews/{RUN_ID}-product-report.md
=============================================
```

---

## Important Operational Notes

### Browser-Free Orchestrator Rule

The orchestrator (main agent) NEVER calls Chrome DevTools tools directly. Every browser interaction is dispatched to a sub-agent. This is not optional — it is the architectural foundation that prevents context rot.

If you find yourself about to call `navigate_page`, `take_screenshot`, `evaluate_script`, or any other Chrome DevTools tool: STOP. Dispatch a sub-agent instead.

### Resumability

This command is designed to be resumable. If context becomes too long or the session is interrupted:

1. Every phase saves its output to disk immediately
2. On re-run with the same date, completed phases are detected and skipped
3. The command prints which phases are being skipped and why
4. If context is getting dangerously long (many pages reviewed), save progress and print:
   ```
   Context limit approaching. Progress saved.
   Completed: {N}/{N} pages reviewed
   Remaining: {list remaining pages}
   Resume by running: /product-review-all --app={app} --tier={tier}
   The command will automatically skip completed phases.
   ```

### Anti-Context-Rot Protocol

Context rot is the primary failure mode. These measures are NON-NEGOTIABLE:

1. **Re-read product-map.json** after every 2 deep page reviews
2. **Write findings to disk** immediately after each page review — never accumulate
3. **Archive screenshots** between page reviews so current/ only has the active page
4. **Delegate aggregation** to sub-agents for critical pages (5 perspectives = too much output for orchestrator)
5. **Print progress** after each completed page so the user can track
6. **Never accumulate browser output** — the orchestrator never touches Chrome DevTools

### Sub-Agent Dispatch Rules

| Phase | Agent Name | Model | Browser? | Parallelism |
|-------|-----------|-------|----------|-------------|
| Phase 0 | po-product-map | sonnet | No | Single |
| Phase 1 | po-health-scan | sonnet | Yes | Single (batches internally) |
| Phase 2 | po-correlation | opus | Yes | Single (multi-tab internally) |
| Phase 3 intel | po-intel-{slug} | varies | No | Background |
| Phase 3 explore | po-explorer-{slug} | varies | Yes | Single per page |
| Phase 3 perspectives | po-{perspective}-{slug} | varies | No | ALL parallel |
| Phase 3 aggregate | po-agg-{slug} | sonnet | No | Single per page |

- **Evidence**: always list every `.png` file in `.omniprod/screenshots/current/` — sub-agents must READ the images
- **Naming**: consistent prefix `po-` for all product-review-all sub-agents

### Shared Component Deduplication

Phase 1 health scan covers shared components (sidebar, topbar, theme) across all pages via assertions. During deep page reviews in Phase 3:
- Perspective sub-agent prompts explicitly instruct: "Do NOT report findings about sidebar, topbar, or theme"
- If a shared component behaves incorrectly on a SPECIFIC page (e.g., wrong active state), that IS a page finding
- The product report includes health scan results alongside deep review findings

### Tier Selection Guide

| Tier | Pages Reviewed | Use When |
|------|---------------|----------|
| 1 | Critical only (~8-10 pages) | Quick pre-demo check, 1-2 hours |
| 2 | Critical + Important (~15-20 pages) | Standard review, 2-4 hours |
| 3 | All pages (~30+ pages) | Full audit, 4-6+ hours |

### Incremental Mode Guide

Use `--incremental` when:
- You've already done a full review and want to verify fixes
- Code changes are scoped to specific pages
- You want a quick re-check after a PR merge

Do NOT use `--incremental` when:
- No `.omniprod/last-review-commit` exists (first review)
- Shared UI components changed (will auto-disable and run full)
- More than 2 weeks since last full review (stale baselines)
