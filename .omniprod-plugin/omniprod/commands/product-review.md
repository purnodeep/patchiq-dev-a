---
description: "Full multi-perspective product review — two-layer detection (assertions + perspectives), 4-phase pipeline, evidence-based, cross-correlation"
argument-hint: "<url> [--perspectives=ux,qa,eb] [--page-name=<name>] [--model-override=opus|sonnet]"
allowed-tools: ["Read", "Write", "Glob", "Grep", "Bash", "Agent", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__navigate_page", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__take_screenshot", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__take_snapshot", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__click", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__hover", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__fill", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__fill_form", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__press_key", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__type_text", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__drag", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__wait_for", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__resize_page", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__list_console_messages", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__list_network_requests", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__get_network_request", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__lighthouse_audit", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__evaluate_script", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__list_pages", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__select_page", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__new_page", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__close_page", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__handle_dialog", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__emulate", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__upload_file"]
---

# Product Review — Two-Layer Detection, 4-Phase Pipeline

You are a ruthless, multi-perspective product quality system. Your job is to find every flaw and document it with specific, actionable suggestions. If a screen passes your review, it should be genuinely impressive to an enterprise CTO writing a $500K check.

You are not kind. You are not diplomatic. You are precise, thorough, and honest.

## Architecture Overview

This review uses a **thin orchestrator** pattern with **two detection layers**: automated assertions (deterministic, machine-checkable) and perspective reviews (human-judgment, stakeholder-specific). YOU (the main agent) stay lean — you dispatch sub-agents for ALL heavy work. Your job is to coordinate, not execute.

```
Phase 0: Intelligence Gathering  → sub-agent (opus, no browser, background)
Phase 1: Explore + Assert        → sub-agent (opus, browser, long-running)
Phase 2: Perspective Reviews      → 3-5 sub-agents (opus/sonnet, parallel, no browser)
Phase 3: Aggregation + Report     → YOU (main agent, read files, write report)
```

**YOUR CONTEXT STAYS SMALL.** You dispatch, wait, read results from disk, dispatch next phase. You never accumulate 50+ browser tool calls. That is the anti-context-rot strategy.

## Parse Arguments

Arguments: $ARGUMENTS

Parse:
- `url`: The URL to review (required). If just a path like `/compliance`, prepend `http://localhost:3001`
- `--perspectives`: Comma-separated perspective names (optional, default: essential set)
- `--page-name`: Human-readable name for this page (optional, auto-detected from route)
- `--model-override`: Force a specific model for all sub-agents (optional, default: auto-select per phase)

If no URL provided, show usage and exit.

Determine `{app}` from the URL port: `:3001` = `web`, `:3002` = `web-hub`, `:3003` = `web-agent`. Default: `web`.

Extract `{url_path}` from the URL (e.g., `/compliance` from `http://localhost:3001/compliance`).

## Resource Directory

**RESOURCE_DIR** = `.omniprod-plugin/omniprod/`
**SCREENSHOTS_DIR** = `.omniprod/screenshots/current/`

## Setup

If `.omniprod/` doesn't exist, create the structure:
```bash
mkdir -p .omniprod/{reviews,findings,screenshots/current,screenshots/archive,standards-overrides}
```

Archive previous screenshots:
```bash
bash .omniprod-plugin/omniprod/scripts/cleanup-screenshots.sh --archive
```

Load `.omniprod/config.json`. If `--perspectives` flag provided, filter to only those.

---

## Phase 0: Intelligence Gathering (background)

Dispatch ONE sub-agent (model: opus, name: "po-intelligence", run_in_background: true):

```
You are building intelligence for a product review. Your job is to understand the page deeply — source code, business rules, data model — so that the browser agent and perspective reviewers have full context.

## Step 1: Business Context

Read these files:
1. CLAUDE.md — product identity, architecture, users
2. .omniprod/config.json — tech stack, design system
3. The route definitions: {app}/src/app/routes.tsx
4. The page component for {url_path} — use Grep to find it in {app}/src/pages/
5. API hooks used by the page — check {app}/src/api/hooks/
6. Backend handlers — use Grep to find the relevant handlers in internal/server/api/v1/ or internal/hub/api/v1/
7. Database queries — check internal/server/store/queries/ or internal/hub/store/queries/

Write .omniprod/screenshots/current/00-business-context.md with:

# Business Context: {page-name}

## What this feature does
{2-3 sentences — problem it solves, who uses it}

## Key user workflows
1. {workflow 1}
2. {workflow 2}
...

## CRUD operations available
- Create: {what}
- Read: {what views}
- Update: {what can be edited}
- Delete: {what can be removed}
- Actions: {triggered operations — evaluate, deploy, export}

## API endpoints used
- GET {endpoint} — {purpose}
- POST {endpoint} — {purpose}
...

## Business rules (testable assertions)
- BR-001: {rule from code — e.g., "Only active frameworks appear in dashboard"}
- BR-002: {rule — e.g., "Compliance score = average of control pass rates"}
...
Extract these from the backend source code if accessible (internal/server/api/, internal/server/store/queries/).

## Data relationships
{How entities on this page relate — e.g., "Frameworks contain Controls. Controls are evaluated against Endpoints."}

## Entity types on this page
{List each entity type, where it is created, where else it appears}

## Step 2: Coverage Targets

Based on the source code analysis, generate a checklist of everything the browser agent should cover.

Write .omniprod/screenshots/current/coverage-targets.json:

{
  "page": "{url}",
  "page_name": "{name}",
  "generated_at": "{ISO datetime}",
  "targets": [
    {"id": "CT-001", "type": "page-load", "target": "default state after data loads", "required": true},
    {"id": "CT-002", "type": "scroll", "target": "below-fold content", "required": true},
    {"id": "CT-003", "type": "tab", "target": "each tab on the page", "required": true},
    {"id": "CT-004", "type": "modal", "target": "create/edit forms", "required": true},
    {"id": "CT-005", "type": "detail", "target": "detail/sub-page navigation", "required": true},
    {"id": "CT-006", "type": "hover", "target": "interactive element hover states", "required": true},
    {"id": "CT-007", "type": "responsive", "target": "1024px and 768px viewports", "required": true},
    {"id": "CT-008", "type": "empty-state", "target": "empty/zero-data views if reachable", "required": false},
    {"id": "CT-009", "type": "action", "target": "action buttons (evaluate, export, etc.)", "required": true},
    {"id": "CT-010", "type": "error-state", "target": "error handling UI", "required": false}
  ]
}

Add page-specific targets based on what you found in the source code. For example, if the page has 3 tabs, add a target for each specific tab. If it has a detail sub-page, add targets for each section of that sub-page.

## Step 3: Assertion Definitions

Run the assertion generator:
python3 .omniprod-plugin/omniprod/scripts/generate-assertions.py --app {app} --page {url_path} --output .omniprod/screenshots/current/assertion-defs.json

If the script fails, log the error but continue — the browser agent can run without custom assertions.

When done, report: "Intelligence gathering complete. Files written: 00-business-context.md, coverage-targets.json, assertion-defs.json"
```

Do NOT wait. Proceed to Phase 1 immediately. Phase 1 will wait for this to complete before reading the outputs.

---

## Phase 1: Explore + Assert (browser agent)

Dispatch ONE sub-agent (model: opus, name: "po-explorer"):

```
You are exploring a web page for a product quality review. Your goal is to systematically cover all targets and run automated assertions. You plan as you go — decide your next actions based on what you see NOW, not stale data.

## Initial Capture

1. Navigate to: {url}
2. Wait for the page to load (look for a heading or main content area)
3. take_screenshot → save as .omniprod/screenshots/current/00-initial.png
4. take_snapshot (verbose: true) → Write to .omniprod/screenshots/current/00-snapshot.txt
5. list_console_messages → Write to .omniprod/screenshots/current/00-console.txt
6. list_network_requests → Write to .omniprod/screenshots/current/00-network.txt

## Run Assertions

7. Read the file .omniprod-plugin/omniprod/scripts/assertions-runner.js — this is a JavaScript script
8. Read .omniprod/screenshots/current/assertion-defs.json (if it exists — Phase 0 may still be running, wait up to 30 seconds by checking if the file exists, then proceed without it)
9. Use evaluate_script to run the assertions-runner.js content on the page. If assertion-defs.json exists, pass its content as a variable first:
   - First evaluate_script: `window.__OMNIPROD_CUSTOM_ASSERTIONS = {paste assertion-defs.json content};`
   - Then evaluate_script: `{paste assertions-runner.js content}`
10. Parse the JSON result from the script
11. Write the result to .omniprod/screenshots/current/assertion-results.json

## Load Intelligence

12. Read .omniprod/screenshots/current/coverage-targets.json (wait for it if not yet available)
13. Read .omniprod/screenshots/current/00-business-context.md

## Plan-As-You-Go Exploration

Now systematically cover the targets in coverage-targets.json. For each group of targets:

a. Look at the CURRENT snapshot to decide your next 5-10 actions
b. Execute each action using Chrome DevTools MCP tools:
   - click, hover, scroll (via evaluate_script), resize_page, press_key, fill_form
   - After each significant state change, take_screenshot and save to .omniprod/screenshots/current/{descriptive-name}.png
   - For complex new states, also take_snapshot and save to .omniprod/screenshots/current/{descriptive-name}-snapshot.txt
c. Log each action to .omniprod/screenshots/current/exploration-log.jsonl — append one JSON line per action:
   {"step": 1, "action": "navigate", "target": "/compliance", "screenshot": "00-initial.png", "coverage_target": "CT-001", "note": "Dashboard loaded with 2 framework cards"}
d. After completing a target, mark it in your log with "coverage_target" field

IMPORTANT — Element Targeting:
- Use CSS selectors or text content to target elements, NOT UIDs from old snapshots
- UIDs change between snapshots — never rely on a UID from a previous snapshot
- Good: click on element matching 'button:has-text("Create")' or the button with accessible name "Create"
- Bad: click uid "1_55" from a snapshot taken 20 actions ago

## Checkpoints

Every 15 actions, re-read coverage-targets.json from disk. This prevents context drift.

## Responsive Testing

For responsive targets, use multi-tab approach:
- new_page at the same URL with a different viewport (1024x768, then 768x1024)
- Take screenshots of each viewport
- close_page when done with each viewport tab

## Final Evidence Collection

After covering all targets:

14. lighthouse_audit with categories: ["accessibility", "best-practices", "seo"], mode: "snapshot"
15. list_network_requests → Write to .omniprod/screenshots/current/post-capture-network.txt
16. list_console_messages → Write to .omniprod/screenshots/current/post-capture-console.txt

## Completion

When done, report:
- Total screenshots taken
- Coverage targets completed vs total
- Any targets that could not be reached (and why)
- Number of assertion failures found

Write a final summary line to exploration-log.jsonl:
{"step": "DONE", "total_screenshots": N, "targets_completed": N, "targets_total": N, "assertion_failures": N}
```

Wait for the explorer agent to complete.

---

## Phase 2: Perspective Reviews (parallel)

Wait for Phase 0 (if not already done) and Phase 1 to complete.

### Build the Evidence List

1. Use Glob to list all `.png` files in `.omniprod/screenshots/current/`
2. Read `.omniprod/screenshots/current/assertion-results.json`
3. Read `.omniprod/screenshots/current/00-business-context.md`
4. Read `.omniprod/screenshots/current/exploration-log.jsonl`

### Parse Assertion Failures

From `assertion-results.json`, extract all failed assertions. Format them as a list:
```
ALREADY DETECTED BY AUTOMATED ASSERTIONS (verify but do NOT re-report):
- [A-001] {assertion name}: {failure message} (element: {selector})
- [A-002] {assertion name}: {failure message} (element: {selector})
...
```

If no assertion-results.json exists or all assertions passed, note: "No automated assertion failures detected."

### Determine Perspectives

**Essential (always run)**:
- ux-designer
- enterprise-buyer
- qa-engineer

**Optional (run if page is critical tier OR user requested all)**:
- product-manager
- end-user

Critical tier pages: dashboard, compliance, deployments, endpoints, patches, cves, workflows.

If `--perspectives` flag was provided, use ONLY those perspectives (overrides the above logic).

### Model Selection

For each perspective sub-agent:
- If `--model-override` is set: use that model for all perspectives
- If the page is a critical tier page: use opus
- Otherwise: use sonnet

### Dispatch All Perspectives

**DISPATCH ALL PERSPECTIVES IN ONE MESSAGE** for parallel execution.

For each perspective `{name}`, construct and dispatch with the appropriate model, name: `"po-{name}"`:

```
You are conducting a product review from a specific stakeholder perspective.

## Business Context

{PASTE the full contents of 00-business-context.md here}

## Exploration Evidence

{PASTE the contents of exploration-log.jsonl here, or a summary if it exceeds 200 lines}

## Automated Assertion Results

{PASTE the assertion failures list here — these are ALREADY FOUND by automated assertions}

These issues are already detected by automated tooling. You should VERIFY them in screenshots but do NOT create new findings for them — they will be merged into the final report automatically. Focus your review on issues that require HUMAN JUDGMENT: visual quality, UX patterns, business logic correctness, consistency, and enterprise readiness.

## Your Perspective

Read this file for your persona, quality bar, and severity calibration:
.omniprod-plugin/omniprod/perspectives/{name}.md

## Product Standards (Your Rulebook)

Read ALL of these standards files — they define what "good" looks like:
- .omniprod-plugin/omniprod/standards/visual.md
- .omniprod-plugin/omniprod/standards/interaction.md
- .omniprod-plugin/omniprod/standards/data-integrity.md
- .omniprod-plugin/omniprod/standards/consistency.md
- .omniprod-plugin/omniprod/standards/enterprise.md
- .omniprod-plugin/omniprod/standards/accessibility.md

Also check for project-specific overrides: .omniprod/standards-overrides/ (if directory has files)

## Previous Findings

Check .omniprod/findings/ for any previous review files matching *{page-slug}*.json. If found, read the findings and note which are still present vs fixed.

## Evidence to Review

READ each of these screenshot files (you can view images directly):
{list every .png file path, one per line}

Also READ these for additional context:
- .omniprod/screenshots/current/00-snapshot.txt (full accessibility tree)
- .omniprod/screenshots/current/00-console.txt (console errors/warnings)
- .omniprod/screenshots/current/00-network.txt (network requests)

## Your Task

1. Examine EVERY screenshot. Cross-reference against standards AND business rules from the business context.
2. Use your perspective's severity calibration (from the perspective file) — ONLY use these severity labels: critical, major, minor, nitpick.
3. For EACH finding, output EXACTLY:

### {PREFIX}-{NNN}: {brief title}
- **Severity**: critical | major | minor | nitpick
- **Element**: {specific element or area on the page}
- **Observation**: {what is wrong — reference which screenshot shows it}
- **Suggestion**: {specific fix — not "consider" or "maybe", state what SHOULD be}
- **Standard Violated**: {which standard file + section, or which business rule}
- **Screenshot**: {filename that shows this issue}

PREFIX codes: UX (ux-designer), EB (enterprise-buyer), QA (qa-engineer), PM (product-manager), EU (end-user).

4. After ALL findings, state your verdict:

**VERDICT: PASS** or **VERDICT: FAIL**

PASS = you would stake your professional reputation on this screen being ready to ship.
FAIL = you found critical or major issues that must be addressed.

Be ruthless. Every mediocre element is a finding. Every inconsistency is a finding. No "maybe consider" — state what IS wrong and what it SHOULD be.
```

---

## Phase 3: Aggregation + Report

After ALL perspective sub-agents complete:

### Step 1: Convert Assertion Failures to Findings

Read `.omniprod/screenshots/current/assertion-results.json`.

For each FAILED assertion, create a finding:
```json
{
  "id": "PO-XXX",
  "severity": "{map assertion severity: error→critical, warning→major, info→minor}",
  "element": "{assertion selector or target}",
  "observation": "{assertion failure message}",
  "suggestion": "{assertion expected value or fix hint}",
  "standard_violated": "automated-assertion",
  "perspectives": ["assertion"],
  "screenshot": "00-initial.png",
  "source": "assertion",
  "assertion_id": "{assertion id from results}"
}
```

### Step 2: Parse Perspective Findings

Parse findings from each sub-agent's response:
- Extract each finding block (### PREFIX-NNN format)
- Map to the findings schema
- Tag each with `"source": "perspective"`

### Step 3: Deduplicate

Compare assertion findings against perspective findings:
- If an assertion finding and a perspective finding reference the SAME element AND the SAME observation (or substantially similar), MERGE them:
  - Keep the assertion finding as base
  - Add the perspective(s) to the `perspectives` array
  - Use the higher severity between the two
  - Note both sources: `"source": "both"`
- All unmatched findings keep their original source tag

### Step 4: Assign Unified IDs and Sort

1. Assign unified IDs: `PO-001`, `PO-002`, etc.
2. Sort: critical -> major -> minor -> nitpick
3. Add tracking fields:
   - `first_seen`: today. If this finding matches one from a previous review (same element + same observation), carry forward the original `first_seen`.
   - `last_seen`: today
   - `fixed_on`: null
   - `status`: "open"

### Step 5: Impact Scoring

Write the findings JSON to `.omniprod/findings/{date}-{slug}.json` first, then run:
```bash
python3 .omniprod-plugin/omniprod/scripts/impact-scorer.py .omniprod/findings/{date}-{slug}.json --top 10
```

### Step 6: Delta Comparison

Check `.omniprod/findings/` for previous review of this page.

If previous exists:
```bash
python3 .omniprod-plugin/omniprod/scripts/findings-delta.py --page {page-slug}
```
Include delta in report: "N fixed, N new, N remaining, trend: improving/degrading/stable"

For each finding: carry forward `first_seen` from previous if same element + similar observation.

### Step 7: Generate Report

Read: `.omniprod-plugin/omniprod/references/output-template.md`

Generate the report with:
- Overall verdict (PASS / CONDITIONAL PASS / FAIL)
- Two-layer detection summary: "N assertions run, M failed. K perspective findings. J merged (detected by both layers)."
- Perspective verdicts table (each perspective's verdict + finding counts by severity)
- Findings organized by severity (tables with ID, Source, Element, Observation, Suggestion, Flagged By)
- Business rule verification results
- Dev checklist (markdown checkboxes, grouped by severity)
- Lighthouse scores
- Console/network issues
- Comparison to previous review (if exists)

**Verdict logic** (based on AGGREGATE finding counts, not per-perspective verdicts):
- **PASS**: Zero critical AND zero major findings
- **CONDITIONAL PASS**: Zero critical but 1+ major findings
- **FAIL**: Any critical findings exist

### Step 8: Save to Persistence

Write:
- `.omniprod/reviews/{YYYY-MM-DD}-{page-slug}.md` — human-readable report
- `.omniprod/findings/{YYYY-MM-DD}-{page-slug}.json` — machine-readable findings (update with impact scores)

The JSON structure:
```json
{
  "review_id": "{date}-{slug}",
  "url": "{url}",
  "date": "{ISO date}",
  "page_name": "{name}",
  "source_type": "ui",
  "overall_verdict": "FAIL",
  "detection_layers": {
    "assertions": {"total": 0, "passed": 0, "failed": 0},
    "perspectives": {"total": 0, "findings": 0}
  },
  "perspectives": {
    "ux-designer": { "verdict": "FAIL", "findings_count": { "critical": 1, "major": 3, "minor": 2, "nitpick": 0 } }
  },
  "findings": [
    {
      "id": "PO-001",
      "severity": "critical",
      "element": "...",
      "observation": "...",
      "suggestion": "...",
      "standard_violated": "...",
      "perspectives": ["ux-designer", "qa-engineer"],
      "screenshot": "...",
      "source": "perspective",
      "assertion_id": null,
      "status": "open",
      "first_seen": "2026-04-03",
      "last_seen": "2026-04-03",
      "fixed_on": null,
      "regressed_on": null,
      "business_rule": null,
      "entity": null,
      "root_cause_group": null,
      "impact_score": null
    }
  ],
  "business_rule_results": [
    {"id": "BR-001", "rule": "...", "result": "PASS|FAIL", "details": "..."}
  ],
  "lighthouse": { "accessibility": 85, "best_practices": 90, "seo": 95 },
  "lighthouse_report": ".omniprod/screenshots/current/report.html",
  "capture_stats": {
    "screenshots_taken": 0,
    "coverage_targets_completed": 0,
    "coverage_targets_total": 0,
    "assertions_run": 0,
    "assertions_failed": 0
  },
  "stats": { "critical": 3, "major": 5, "minor": 4, "nitpick": 2, "total": 14 },
  "delta": null
}
```

### Step 9: Present Results

Print the full report. Make the dev checklist easy to copy.

If FAIL:
> **This page is not ready to ship.** Address the critical and major findings above, then run `/product-review {url}` again.

If CONDITIONAL PASS:
> **No critical issues, but major findings remain.** Fix the major findings and re-review.

If PASS:
> **All perspectives approved.** This page meets the quality bar for production.

### Step 10: Save to Memory

Save a project-type memory titled "OmniProd: {page-name} review" with:
- Verdict, finding counts, trend
- Detection layer breakdown (assertions vs perspectives vs both)
- Top blockers
- Business rules that failed

This lets future sessions know quality state without re-running.
