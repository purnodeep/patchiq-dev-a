---
description: "Continuous product quality monitoring via Ralph Loop — iterative review until all perspectives pass"
argument-hint: "<url> [--max-iterations=<N>] [--page-name=<name>]"
allowed-tools: ["Read", "Write", "Glob", "Grep", "Bash", "Agent", "Skill", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__navigate_page", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__take_screenshot", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__take_snapshot", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__click", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__hover", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__fill", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__fill_form", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__press_key", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__type_text", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__drag", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__wait_for", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__resize_page", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__list_console_messages", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__list_network_requests", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__get_network_request", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__lighthouse_audit", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__evaluate_script", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__list_pages", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__select_page", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__new_page", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__handle_dialog", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__emulate", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__upload_file"]
---

# OmniProd Loop — Continuous Quality Monitoring

Runs an iterative product review cycle using Ralph Loop. Each iteration:
1. Archives previous screenshots
2. Runs a full `/product-review` cycle
3. Compares findings to the previous iteration
4. Reports delta (fixed / new / remaining)
5. Continues until ALL perspectives pass — then outputs the completion promise

This is designed to run while a developer fixes issues. The loop keeps re-checking until the page is clean.

## Parse Arguments

Arguments: $ARGUMENTS

Parse:
- `url`: The URL to monitor (required). If just a path, prepend `http://localhost:3001`
- `--max-iterations`: Maximum iterations before auto-stop (optional, default: 20)
- `--page-name`: Human-readable name (optional, auto-detected)

## Auto-Init

If `.omniprod/` doesn't exist, create the full directory structure:
```bash
mkdir -p .omniprod/{reviews,findings,screenshots/current,screenshots/archive,standards-overrides}
```

If `.omniprod/config.json` doesn't exist, run the initialization logic from `/product-init` (auto-detect project, write config).

## Iteration Logic

### On Each Iteration

**Step 1: Archive previous screenshots**
```bash
bash .omniprod-plugin/omniprod/scripts/cleanup-screenshots.sh --archive
```

**Step 2: Run the full review pipeline**

Follow the EXACT same process as `/product-review`:
1. Navigate to URL with Chrome DevTools MCP
2. Take initial screenshot + snapshot + console + network
3. Systematic interaction audit (hover, click, focus, expand, resize)
4. Lighthouse audit
5. Dispatch all perspective sub-agents in parallel (Sonnet)
6. Aggregate findings

**Step 3: Compare to previous**

Check if a previous findings file exists for this page in `.omniprod/findings/`.
If it does, compute the delta:
```bash
python3 .omniprod-plugin/omniprod/scripts/findings-delta.py --page {page-slug}
```

**Step 4: Report iteration results**

```markdown
## Iteration {N} — {page-name}

**Verdict**: {PASS/FAIL}
**Delta**: {N} fixed, {N} new, {N} remaining
**Trend**: {improving/degrading/stable}

### Current Open Findings
- {N} critical, {N} major, {N} minor, {N} nitpick

### What Changed Since Last Iteration
**Fixed:**
- PO-003: [element] — was [severity]
- PO-007: [element] — was [severity]

**New:**
- PO-012: [element] — [severity] — [observation]

**Still Open:**
- PO-001: [element] — [severity]
- PO-005: [element] — [severity]
```

**Step 5: Save findings**

Save the review report and findings JSON as usual:
- `.omniprod/reviews/{YYYY-MM-DD}-{page-slug}.md`
- `.omniprod/findings/{YYYY-MM-DD}-{page-slug}.json`

Include the `delta` section in the findings JSON.

**Step 6: Save review summary to memory**

After each iteration, save a brief summary to Claude's auto-memory system:
```
Write to memory: project type
Title: "OmniProd review: {page-name} — iteration {N}"
Content: "{verdict} — {N} critical, {N} major findings. {delta summary}. {trend}."
```

This lets future sessions know the current quality state without re-running reviews.

**Step 7: Check completion**

If ALL perspectives returned PASS and there are ZERO critical and ZERO major findings:

```
<promise>ALL PERSPECTIVES PASSED</promise>
```

This signals Ralph Loop to stop. Output a celebration:

```markdown
# ✅ ALL PERSPECTIVES PASSED

**Page**: {page-name}
**URL**: {url}
**Iterations**: {N}
**Final State**: Zero critical, zero major findings

This page meets the quality bar for production deployment.
Every stakeholder perspective approved unanimously.
```

If NOT all perspectives passed, end the iteration with:
```
Iteration {N} complete. {N} findings remain. Waiting for fixes before re-checking.
```

Ralph Loop will feed this prompt again, and you'll start the next iteration.

## How to Start a Loop

Use with Ralph Loop:
```
/ralph-loop "/product-loop http://localhost:3001/compliance" --completion-promise "ALL PERSPECTIVES PASSED" --max-iterations 20
```

Or invoke directly for a single check-and-compare iteration:
```
/product-loop http://localhost:3001/compliance
```

## Resource Paths

- Perspectives: `.omniprod-plugin/omniprod/perspectives/`
- Standards: `.omniprod-plugin/omniprod/standards/`
- References: `.omniprod-plugin/omniprod/references/`
- Scripts: `.omniprod-plugin/omniprod/scripts/`
- Runtime data: `.omniprod/`
