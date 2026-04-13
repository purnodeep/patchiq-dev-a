---
description: "Dry-run capture phase only — verifies business context, screenshots, CRUD flows, and validation gate WITHOUT dispatching perspective sub-agents"
argument-hint: "<url> [--page-name=<name>]"
allowed-tools: ["Read", "Write", "Glob", "Grep", "Bash", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__navigate_page", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__take_screenshot", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__take_snapshot", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__click", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__hover", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__fill", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__fill_form", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__press_key", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__type_text", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__drag", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__wait_for", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__resize_page", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__list_console_messages", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__list_network_requests", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__lighthouse_audit", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__evaluate_script", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__list_pages", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__select_page", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__new_page", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__handle_dialog", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__emulate", "mcp__plugin_chrome-devtools-mcp_chrome-devtools__upload_file"]
---

# OmniProd Dry Run — Capture Verification

This runs ONLY Phase 0 (business context) + Phase 2 (capture) + validation. No sub-agents, no report, no findings. Purpose: verify the capture pipeline works before spending on a full 8-perspective review.

## Parse Arguments

Arguments: $ARGUMENTS

- `url`: Required. If just a path, prepend `http://localhost:3001`
- `--page-name`: Optional human-readable name

## Step 1: Business Context (Phase 0)

### 1a. Read Project Documentation

1. Read `CLAUDE.md` — understand the product identity, users, architecture
2. Read `.omniprod/config.json` — tech stack, design system, apps
3. Read the route definitions for the app being reviewed (e.g., `web/src/app/routes.tsx`)

### 1b. Read Page Source Code

Use Grep to find the page component for the URL path. Read it. Understand:
- What API endpoints does this page call?
- What data does it display?
- What user actions are available?

### 1c. Write Business Context

Write a `.omniprod/screenshots/current/00-business-context.md` file containing:

```markdown
# Business Context: {page-name}

## What this feature does
{2-3 sentences — what problem it solves, who uses it}

## Key user workflows
1. {workflow 1 — e.g., "Create a compliance framework"}
2. {workflow 2}
3. ...

## CRUD operations available
- Create: {what can be created?}
- Read: {what views/details are available?}
- Update: {what can be edited?}
- Delete: {what can be removed?}
- Actions: {what operations can be triggered? e.g., "Evaluate", "Export"}

## API endpoints used
- GET {endpoint} — {what it returns}
- POST {endpoint} — {what it does}
- ...

## Data relationships
{e.g., "Frameworks contain Controls. Controls are evaluated against Endpoints. Scores roll up to framework-level compliance."}
```

This file proves business context was loaded. If this file doesn't exist after the dry run, Phase 0 failed.

## Step 2: Initial Capture

1. Archive previous screenshots: `bash .omniprod-plugin/omniprod/scripts/cleanup-screenshots.sh --archive`
2. `navigate_page` → target URL
3. `wait_for` → page content loaded
4. `take_screenshot` → `.omniprod/screenshots/current/00-initial.png`
5. `take_snapshot` (verbose: true) → `.omniprod/screenshots/current/00-snapshot.txt`
6. `list_console_messages` → `.omniprod/screenshots/current/00-console.txt`
7. `list_network_requests` → `.omniprod/screenshots/current/00-network.txt`

## Step 3: Generate & Execute Capture Manifest

```bash
python3 .omniprod-plugin/omniprod/scripts/parse-snapshot.py .omniprod/screenshots/current/00-snapshot.txt --output .omniprod/screenshots/current/capture-manifest.json
```

Read the manifest. Execute every task mechanically — hover, click, focus, scroll.

**CRITICAL: Also execute CRUD flow testing:**
- Click every "Create"/"New"/"Add" button → screenshot the form → `create-{entity}-form.png`
- Click every "Edit" button → screenshot → `edit-{entity}-form.png`
- Click every "Delete" button → screenshot confirmation → `delete-{entity}-confirm.png` → cancel
- Click every action button (Evaluate, Export, etc.) → screenshot result → `action-{name}-result.png`

**CRITICAL: Sub-page tab exploration:**
- Follow "View Details" links → click EVERY tab → screenshot each → `subpage-{name}-{tab}.png`
- Interact within tabs (hover rows, click expandable items)

## Step 4: Validation

Run:
```bash
bash .omniprod-plugin/omniprod/scripts/validate-capture.sh
```

## Step 5: Dry Run Report

Print a summary:

```
=== OmniProd Dry Run Report ===

Business Context:
- CLAUDE.md read: {yes/no}
- Page source read: {yes/no — which file}
- Business context file written: {yes/no}
- CRUD flows identified: {count}
- API endpoints identified: {count}

Capture:
- Total screenshots: {N}
- Hover states: {N}
- Focus states: {N}
- Scroll states: {N}
- Responsive: {N}
- Sub-page tabs: {N}
- CRUD forms: {N}
- Action results: {N}

Validation: {PASS/INCOMPLETE — details if incomplete}

Ready for full review: {YES/NO}
```

Do NOT dispatch any sub-agents. Do NOT generate findings. This is capture-only.
