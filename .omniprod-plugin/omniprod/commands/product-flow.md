---
description: "Cross-page flow review — walk a user journey, test cross-page assertions and data consistency"
argument-hint: "<flow-name> [--base-url=<url>]"
allowed-tools: ["Read", "Write", "Glob", "Grep", "Bash", "Agent", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__navigate_page", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__take_screenshot", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__take_snapshot", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__click", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__hover", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__fill", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__fill_form", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__press_key", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__type_text", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__wait_for", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__resize_page", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__list_console_messages", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__list_network_requests", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__get_network_request", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__lighthouse_audit", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__evaluate_script", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__take_snapshot", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__list_pages", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__select_page"]
---

# Product Flow Review — Cross-Page User Journey Audit

You are a cross-page quality auditor. Your job is to walk a complete user journey across multiple pages, record entity data at each step, and verify that data is consistent across the entire flow. Single-page reviews catch UI bugs; flow reviews catch integration bugs, stale caches, and broken workflows.

## Parse Arguments

Arguments: $ARGUMENTS

Parse:
- `flow-name`: Name/ID of the flow to execute (required). Use `"all"` to run all flows sequentially.
- `--base-url`: Base URL for the app (default: `http://localhost:3001`)

If no flow-name is provided, list available flows from `.omniprod/product-map.json` and exit:
```
Usage: /product-flow <flow-name> [--base-url=<url>]

Available flows:
  compliance-assess    Assess Compliance Posture (6 steps, 4 pages)
  patch-deploy         Deploy a Patch End-to-End (8 steps, 5 pages)
  ...

Run /product-flow all to execute all flows.
```

If `.omniprod/product-map.json` does not exist, tell the user:
```
No product map found. Run /product-map first to define pages and flows.
```

## Step 1: Load Flow Definition

1. Read `.omniprod/product-map.json`
2. Find the flow matching `flow-name` in the `flows` array (match on `id` or `name`)
3. If not found, show available flows with their step counts and exit
4. Extract:
   - `flow.id` — unique identifier
   - `flow.name` — human-readable name
   - `flow.description` — what this journey tests
   - `flow.role` — the persona walking this journey (e.g., "IT Admin", "CISO")
   - `flow.steps[]` — ordered list of steps, each with `page`, `url`, `action`, `expected`, and optional `assertions`
   - `flow.cross_page_assertions[]` — assertions that compare data across steps

## Step 2: Setup

1. Create the flow-specific screenshot directory:
   ```bash
   mkdir -p .omniprod/screenshots/flows/{flow-id}/
   ```
2. Initialize `flow-log.jsonl` in that directory (overwrite if exists):
   ```bash
   echo "" > .omniprod/screenshots/flows/{flow-id}/flow-log.jsonl
   ```

## Step 3: Execute Flow Steps

For each step in the flow (in order):

### 3a. Navigate
1. `navigate_page` to `{base-url}{step.url}`
2. `wait_for` the page's main content to load (heading, table, or key element)

### 3b. Capture
3. `take_screenshot` — save to `.omniprod/screenshots/flows/{flow-id}/flow-{step-number}-{page-slug}.png`
4. `take_snapshot` — save to `.omniprod/screenshots/flows/{flow-id}/flow-{step-number}-{page-slug}-snapshot.txt`

### 3c. Record Data
5. `list_network_requests` — note API calls made during page load
6. **Record entity data**: From the a11y snapshot (NOT from API responses), extract key data values visible on the page:
   - Counts (e.g., "47 endpoints", "6 frameworks", "12 critical patches")
   - Scores (e.g., "75% compliant", "Score: 82")
   - Statuses (e.g., "3 overdue", "2 failed", "Active")
   - Names/labels of key entities displayed

### 3d. Execute Action
7. If the step specifies an `action` (click a button, fill a form, select a tab):
   - Perform the action using the appropriate MCP tool (`click`, `fill`, `press_key`, etc.)
   - `wait_for` the result to appear
   - `take_screenshot` — save as `flow-{step-number}-{page-slug}-after.png`
   - Record any new entity data visible after the action

### 3e. Log Step
8. Append a JSON line to `flow-log.jsonl`:
```json
{
  "step": 1,
  "page": "/compliance",
  "action": "View dashboard",
  "screenshot": "flow-1-compliance.png",
  "entity_data": {
    "framework_count": 2,
    "overall_score": "75%",
    "overdue_count": 4
  },
  "api_calls": ["GET /api/v1/compliance/frameworks [200]"],
  "console_errors": 0,
  "timestamp": "2026-04-03T14:30:00Z"
}
```

### 3f. Check Console
9. `list_console_messages` — note any errors. If console errors appear, record them in the log entry.

## Step 4: Cross-Page Assertions

After executing ALL steps, evaluate each `cross_page_assertion` from the flow definition.

For each assertion:
1. Read the `entity_data` from the relevant steps (referenced by step number)
2. Compare values across pages
3. Record the result as PASS or FAIL with details

Build the assertions result:
```json
{
  "assertions": [
    {
      "assertion": "Framework count on /dashboard matches /compliance card count",
      "step_a": 1,
      "step_b": 3,
      "value_a": "2 frameworks",
      "value_b": "2 framework cards",
      "result": "PASS"
    },
    {
      "assertion": "Endpoint count on /dashboard matches /endpoints total",
      "step_a": 1,
      "step_b": 5,
      "value_a": "47 endpoints",
      "value_b": "49 endpoints",
      "result": "FAIL",
      "details": "Dashboard shows 47, endpoint list shows 49 — possible caching issue"
    }
  ]
}
```

## Step 5: Dispatch Perspective Reviews

Dispatch 3-4 key perspectives as parallel sub-agents. Flow reviews are lighter than full page reviews — use a focused subset, not all 8.

**Perspectives to dispatch:**
- Enterprise Buyer (overall impression of the journey)
- QA Engineer (workflow completeness and data consistency)
- End User (usability and discoverability)
- Product Manager (feature coherence and flow logic)

For each perspective, dispatch a sub-agent using the `Agent` tool with `model: "sonnet"`. **Dispatch ALL sub-agents in a SINGLE message** for parallel execution.

### Sub-Agent Prompt

```
You are reviewing a cross-page user flow: "{flow.name}"

This flow tests: {flow.description}
Role: {flow.role}

## Flow Steps
{For each step, list: step number, page URL, action, expected behavior}

## Screenshots
{List ALL .png files in .omniprod/screenshots/flows/{flow-id}/ — instruct the agent to READ each image}

## Cross-Page Assertions
{List each assertion with its PASS/FAIL result and details}

## Entity Data Recorded
{For each step, show the entity_data captured}

## Your Perspective
Read: .omniprod-plugin/omniprod/perspectives/{name}.md

Review this flow for:
1. Can the user complete this journey successfully?
2. Is data consistent across pages?
3. Is navigation logical and recoverable?
4. Are there any breaks in the workflow?
5. Is the flow discoverable (would a new user find it)?

For EACH finding, output EXACTLY this format:

### {PREFIX}-{NNN}: {brief title}
- **Severity**: critical | major | minor | nitpick
- **Element**: {specific element — include which page/step}
- **Observation**: {what's wrong — reference which screenshot shows it}
- **Suggestion**: {how to fix}

After ALL findings, state your verdict:

**VERDICT: PASS** or **VERDICT: FAIL**

PASS = the flow works end-to-end, data is consistent, navigation is logical
FAIL = you found critical or major issues that break the journey
```

### Dispatch

Use the `Agent` tool with:
- `model: "sonnet"`
- `name: "flow-{perspective-name}"` (e.g., `flow-enterprise-buyer`)
- `description: "Flow review: {perspective} perspective"` (e.g., `"Flow review: Enterprise Buyer perspective"`)

Launch ALL 4 perspectives in one message (parallel).

## Step 6: Generate Flow Report

### 6a. Collect & Merge Findings

1. Parse findings from each sub-agent's response
2. **Deduplicate**: Same element + same issue from multiple perspectives — merge, list all perspectives
3. **Re-ID**: Assign unified IDs: `FL-001`, `FL-002`, etc.
4. **Sort**: critical -> major -> minor -> nitpick

### 6b. Determine Verdict

- **PASS**: ALL perspectives pass AND all cross-page assertions pass AND zero critical findings
- **CONDITIONAL PASS**: All perspectives pass but has FAIL assertions or major findings
- **FAIL**: ANY perspective fails OR has critical findings

### 6c. Write Report

Write to `.omniprod/reviews/{date}-flow-{flow-id}.md`:

```markdown
# Flow Review: {flow.name}

- **Flow ID**: {flow-id}
- **Date**: {date}
- **Role**: {flow.role}
- **Steps**: {N}
- **Pages visited**: {list of unique pages}
- **Verdict**: PASS / CONDITIONAL PASS / FAIL

## Cross-Page Assertions

| # | Assertion | Result | Details |
|---|-----------|--------|---------|
| 1 | Framework count matches | PASS | 2 = 2 |
| 2 | Endpoint count matches | FAIL | 47 != 49 |

## Perspective Verdicts

| Perspective | Verdict | Critical | Major | Minor |
|-------------|---------|----------|-------|-------|
| Enterprise Buyer | FAIL | 1 | 2 | 0 |
| QA Engineer | FAIL | 0 | 3 | 1 |
| End User | PASS | 0 | 0 | 2 |
| Product Manager | PASS | 0 | 1 | 0 |

## Findings

### Critical

#### FL-001: {title}
- **Severity**: critical
- **Element**: {element} (Step {N}, {page})
- **Observation**: {what's wrong}
- **Suggestion**: {how to fix}
- **Perspectives**: enterprise-buyer, qa-engineer

### Major
...

### Minor
...

## Dev Checklist

- [ ] Fix endpoint count mismatch (dashboard vs list) [FL-002]
- [ ] ...

## Flow Screenshots

| Step | Page | Screenshot |
|------|------|------------|
| 1 | /compliance | flow-1-compliance.png |
| 2 | /compliance/CIS | flow-2-compliance-cis.png |
| ... | ... | ... |
```

### 6d. Save Findings JSON

Write to `.omniprod/findings/{date}-flow-{flow-id}.json`:

```json
{
  "review_id": "{date}-flow-{flow-id}",
  "flow_id": "{flow-id}",
  "flow_name": "{flow.name}",
  "date": "{ISO date}",
  "overall_verdict": "FAIL",
  "steps_executed": 6,
  "pages_visited": ["/dashboard", "/compliance", "/compliance/CIS", "/endpoints"],
  "cross_page_assertions": [
    {
      "assertion": "Framework count matches",
      "result": "PASS",
      "step_a": 1,
      "step_b": 3,
      "value_a": "2",
      "value_b": "2"
    }
  ],
  "perspectives": {
    "enterprise-buyer": { "verdict": "FAIL", "findings_count": { "critical": 1, "major": 2, "minor": 0 } },
    "qa-engineer": { "verdict": "FAIL", "findings_count": { "critical": 0, "major": 3, "minor": 1 } },
    "end-user": { "verdict": "PASS", "findings_count": { "critical": 0, "major": 0, "minor": 2 } },
    "product-manager": { "verdict": "PASS", "findings_count": { "critical": 0, "major": 1, "minor": 0 } }
  },
  "findings": [
    {
      "id": "FL-001",
      "severity": "critical",
      "element": "...",
      "observation": "...",
      "suggestion": "...",
      "perspectives": ["enterprise-buyer", "qa-engineer"],
      "step": 3,
      "page": "/compliance/CIS",
      "status": "open",
      "first_seen": "{today}",
      "last_seen": "{today}",
      "fixed_on": null,
      "regressed_on": null
    }
  ],
  "stats": { "critical": 1, "major": 6, "minor": 3, "nitpick": 0, "total": 10 },
  "assertion_stats": { "pass": 3, "fail": 1, "total": 4 }
}
```

## Step 7: Print Summary

Print a concise summary to the conversation:

```
=== Flow Review: {flow.name} ===

Steps: {N} | Pages: {N unique} | Verdict: {PASS/CONDITIONAL PASS/FAIL}

Cross-Page Assertions:
  [checkmark] Framework count matches across pages
  [X] Endpoint count: dashboard (47) != list (49)
  [checkmark] Scores consistent after Evaluate All

Findings: {N} critical, {N} major, {N} minor

Full report: .omniprod/reviews/{date}-flow-{flow-id}.md
```

If the verdict is FAIL, end with:
> **This flow is broken.** Address the critical findings and failing assertions above, then run `/product-flow {flow-id}` again.

If PASS:
> **Flow verified.** This user journey works end-to-end with consistent data across all pages.

## Important Notes

- Each flow is a SEQUENCE of page visits with data recording at each step
- The key differentiator from `/product-review`: cross-page ASSERTIONS that compare data values
- Entity data must be recorded from visible text in the a11y snapshot, NOT from raw API responses
- Screenshots use the flow-specific directory (`.omniprod/screenshots/flows/{flow-id}/`), not `screenshots/current/`
- Flow reviews are lighter than page reviews: 3-4 perspectives, not all 8
- If `.omniprod/product-map.json` does not exist, tell the user to run `/product-map` first
- If running `all` flows, execute them sequentially and produce a combined summary at the end
