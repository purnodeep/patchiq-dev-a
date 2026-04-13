---
description: "Quick product quality check — screenshot + 3 critical perspectives (UX, QA, Enterprise)"
argument-hint: "<url> [--page-name=<name>]"
allowed-tools: ["Read", "Write", "Glob", "Grep", "Bash", "Agent", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__navigate_page", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__take_screenshot", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__take_snapshot", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__click", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__hover", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__wait_for", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__resize_page", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__list_console_messages", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__list_network_requests", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__list_pages", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__select_page", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__evaluate_script"]
---

# Product Check — Quick Quality Review

A lightweight version of `/product-review`. Takes a screenshot and dispatches 3 critical perspectives: UX Designer, QA Engineer, and Enterprise Buyer.

No interaction audit (no hover/click/focus testing). Just initial state + these 3 critical lenses.

## Parse Arguments

Arguments: $ARGUMENTS

- `url`: Required. If just a path like `/dashboard`, prepend `http://localhost:3001`
- `--page-name`: Optional human-readable name

## Execution

### 0. Auto-Init

If `.omniprod/` doesn't exist, create the structure:
```bash
mkdir -p .omniprod/{reviews,findings,screenshots/current,screenshots/archive,standards-overrides}
```
If `.omniprod/config.json` doesn't exist, auto-detect the project (read CLAUDE.md) and write a basic config.

Archive previous screenshots:
```bash
bash .omniprod-plugin/omniprod/scripts/cleanup-screenshots.sh --archive
```

### 1. Capture (no interaction audit)

1. `navigate_page` → target URL
2. `wait_for` → key content visible
3. `take_screenshot` → save as `.omniprod/screenshots/current/00-initial.png`
4. `take_snapshot` (verbose: true) → save text to `.omniprod/screenshots/current/00-snapshot.txt`
5. `list_console_messages` → note errors
6. `list_network_requests` → note failures

### 2. Dispatch 3 Perspectives (Parallel)

Dispatch sub-agents for: `ux-designer`, `qa-engineer`, `enterprise-buyer`

Use the same sub-agent prompt format as `/product-review` but with only the initial screenshot and snapshot (no interaction screenshots to review).

The resource directory is `.omniprod-plugin/omniprod/`.

Each sub-agent reads:
- Their perspective file from `.omniprod-plugin/omniprod/perspectives/`
- All standards from `.omniprod-plugin/omniprod/standards/`
- The initial screenshot and snapshot
- Console and network logs

Use `model: "sonnet"` and launch all 3 in parallel.

### 3. Quick Report

Generate a condensed report:

```markdown
# Quick Check: {Page Name}

**URL**: {url} | **Date**: {date} | **Verdict**: {PASS/FAIL}

| Perspective | Verdict | Critical | Major |
|-------------|---------|----------|-------|
| UX Designer | ❌/✅ | N | N |
| QA Engineer | ❌/✅ | N | N |
| Enterprise Buyer | ❌/✅ | N | N |

## Key Findings (critical + major only)

| ID | Severity | Element | Observation | Suggestion |
|----|----------|---------|-------------|------------|
| ... | ... | ... | ... | ... |

## Quick Checklist
- [ ] **PO-001** [critical] ...
- [ ] **PO-002** [major] ...
```

Save to `.omniprod/reviews/` and `.omniprod/findings/` same as full review.

Verdict: same logic as full review but with 3 perspectives instead of 8.

### 4. Save to Memory

Save a brief summary to Claude's auto-memory:
- Project memory: "OmniProd quick check: {page-name} — {verdict}, {N} critical, {N} major findings"
- This ensures future sessions know the last quality state.
