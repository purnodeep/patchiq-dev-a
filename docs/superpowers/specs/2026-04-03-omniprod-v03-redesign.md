# OmniProd v0.3.0 — Two-Layer Detection Redesign

**Date**: 2026-04-03
**Status**: Approved
**Author**: Heramb + Claude

---

## Problem Statement

OmniProd v0.2.0 relies on screenshot-based AI review with 8 overlapping personas. This produces high finding volume (88 findings across 3 pages) but suffers from:

1. **Low signal-to-noise**: 8 perspectives generate duplicate findings; deduplication is imperfect
2. **Missed business logic bugs**: Screenshots can't detect wrong-but-plausible numbers (e.g., compliance score of 75% that should be 72%)
3. **Stale UIDs**: Capture plan generated from scout snapshot; by execution time, UIDs point to different elements
4. **7 sequential phases**: Scout → Plan → Capture → Annotate → Perspectives → Aggregate adds ~6 minutes of handoff overhead per page
5. **No programmatic verification**: All detection depends on LLM visual analysis
6. **No incremental mode**: Every review starts from scratch regardless of what changed
7. **Orchestrator touches browser**: `product-review-all` main agent does browser work, causing context rot

## Architecture: Two-Layer Detection

```
Layer 1: Automated Assertions (fast, objective, reproducible)
  ├── Data integrity: API response counts vs DOM element counts
  ├── Accessibility: axe-core subset via evaluate_script
  ├── Console: error/warning detection
  ├── Performance: LCP, network waterfall, bundle indicators
  └── Business rules: custom assertions from source code analysis
  → Output: assertion-results.json (machine-verifiable pass/fail)

Layer 2: AI Perspective Review (slower, subjective, judgment-based)
  ├── UX Designer — visual quality, interaction design
  ├── Enterprise Buyer — "would I pay $500K for this"
  ├── QA Engineer — edge cases, error handling, state management
  ├── Product Manager — feature completeness (critical pages only)
  └── End User — discoverability (critical pages only)
  → Receives Layer 1 findings as "already known — don't re-report"
  → Output: subjective quality findings only
```

### Model Selection Strategy

Model choice is complexity-driven, not fixed:

| Task | Model | Rationale |
|------|-------|-----------|
| Business context + assertion generation | opus | Reads source code, extracts business rules — needs deep reasoning |
| Exploration + capture | opus | Real-time planning decisions, adaptive exploration |
| Perspective reviews (complex pages) | opus | Deep analysis of critical pages with many states |
| Perspective reviews (simple pages) | sonnet | Mechanical review against standards |
| Smoke test execution | sonnet | Sequential navigate + screenshot — no judgment needed |
| Cross-page correlation | opus | Comparing data across pages, reasoning about consistency |
| Report aggregation | main agent | File I/O only |
| Batch assertions execution | sonnet | Running pre-defined JS, recording results |

The command prompts specify model recommendations but allow override via `--model` flag.

## Single Page Review — 4 Phases (down from 7)

### Phase 0: Intelligence Gathering (background)

**Agent**: opus (reads source code, generates assertions — needs reasoning)
**Browser**: no

Reads source code for the target page and produces three artifacts:

#### Output 1: `business-context.md`
Same as v0.2.0 — what the feature does, CRUD operations, API endpoints, business rules, entity relationships.

#### Output 2: `coverage-targets.json`
Replaces the rigid capture plan. A checklist the explorer checks off freely:

```json
{
  "page": "/compliance",
  "page_name": "Compliance Dashboard",
  "targets": [
    {"id": "CT-001", "type": "page-load", "target": "default state", "required": true},
    {"id": "CT-002", "type": "scroll", "target": "below-fold content", "required": true},
    {"id": "CT-003", "type": "tabs", "targets": ["Overview", "Controls"], "required": true},
    {"id": "CT-004", "type": "crud", "targets": ["Create Framework"], "required": true},
    {"id": "CT-005", "type": "subpage", "target": "first framework detail", "required": true},
    {"id": "CT-006", "type": "responsive", "targets": ["1440", "1024", "768"], "required": true},
    {"id": "CT-007", "type": "hover", "target": "primary action buttons", "required": false},
    {"id": "CT-008", "type": "focus", "target": "tab key sweep x3", "required": false},
    {"id": "CT-009", "type": "empty-state", "target": "filter to zero results", "required": false},
    {"id": "CT-010", "type": "error-state", "target": "trigger validation error", "required": false}
  ]
}
```

#### Output 3: `assertion-defs.json`
JavaScript assertion definitions generated from source code analysis:

```json
{
  "assertions": [
    {
      "id": "DI-001",
      "category": "data-integrity",
      "description": "Framework count matches API response",
      "script": "...",
      "severity": "critical",
      "business_rule": "BR-001"
    },
    {
      "id": "A11Y-001",
      "category": "accessibility",
      "description": "No critical axe-core violations",
      "script": "...",
      "severity": "major"
    },
    {
      "id": "PERF-001",
      "category": "performance",
      "description": "No console errors on page load",
      "script": "...",
      "severity": "major"
    }
  ]
}
```

### Phase 1: Explore + Assert (browser agent)

**Agent**: opus (adaptive exploration, real-time decision-making)
**Browser**: yes

Single agent that:
1. Navigates to page, takes initial screenshot + snapshot
2. Runs ALL assertions via `evaluate_script` → writes `assertion-results.json`
3. Reads `coverage-targets.json` and `business-context.md` (Phase 0 should be done by now)
4. Explores interactively, checking off coverage targets
5. Plans 5-10 actions at a time from CURRENT snapshot (no stale UIDs)
6. Uses multi-tab for responsive (open 3 tabs at 3 viewports, capture all)
7. Checkpoints to disk every 15 actions (anti-context-rot)
8. Final: lighthouse audit, network summary, console summary

#### Multi-Tab Responsive Pattern
```
1. new_page(url, background: true)  → tab B (1024px)
2. new_page(url, background: true)  → tab C (768px)
3. select_page(tab_B) → resize_page(1024, 768) → screenshot
4. select_page(tab_C) → resize_page(768, 1024) → screenshot
5. select_page(original) → continue at 1920x1080
6. close_page(tab_B), close_page(tab_C)
```

#### Exploration Log
Writes `exploration-log.jsonl` with inline annotations (replaces separate annotator phase):
```json
{"step": 1, "action": "navigate", "target": "/compliance", "screenshot": "page-load.png", "coverage_target": "CT-001", "note": "Dashboard loaded with 2 framework cards"}
{"step": 2, "action": "assert", "results": 12, "passed": 10, "failed": 2, "note": "2 assertion failures: DI-001, A11Y-003"}
{"step": 3, "action": "click", "target": "tab 'Controls'", "screenshot": "tab-controls.png", "coverage_target": "CT-003", "note": "Controls tab shows 24 controls in table"}
```

### Phase 2: AI Review (parallel perspectives)

**Agent**: opus for critical pages, sonnet for important pages
**Browser**: no

3 essential perspectives always run: UX Designer, Enterprise Buyer, QA Engineer.
2 optional perspectives for critical pages: Product Manager, End User.

Each perspective receives:
- All screenshots from Phase 1
- `assertion-results.json` with instruction: "These issues are ALREADY FOUND — verify but don't re-report"
- `business-context.md`
- `exploration-log.jsonl` (replaces annotated-captures.md)
- Standards files (same as v0.2.0)

### Phase 3: Aggregate (main agent)

**Agent**: main agent (no sub-agent)
**Browser**: no

1. Read assertion-results.json → convert failed assertions to findings
2. Read perspective findings from each sub-agent
3. Deduplicate (assertion finding + same issue from perspective = merge)
4. Score, delta comparison, write report
5. Save to `.omniprod/findings/` and `.omniprod/reviews/`

## Full Product Review — Browser-Free Orchestrator

The main orchestrator NEVER touches Chrome DevTools. Every browser operation is a sub-agent.

### Phase 0: Product Map
Same as v0.2.0. Sub-agent reads source code, generates `product-map.json`.

### Phase 1: Health Scan (all pages)
**Agent**: sonnet (mechanical)
**Browser**: yes

For each page: navigate → run standard assertions → screenshot → record results.
Uses multi-tab batch loading: open 4 pages, wait, capture sequentially, close, next 4.

Output: `{RUN_ID}-health-scan.json` with per-page assertion pass/fail counts.

### Phase 2: Cross-Page Correlation (NEW)
**Agent**: opus (reasoning about data consistency)
**Browser**: yes, multi-tab

Opens key entity pages simultaneously and compares:
```
Tab 1: /dashboard    → extract "47 endpoints" via evaluate_script
Tab 2: /endpoints    → extract row count via evaluate_script
Tab 3: /compliance   → extract "3 frameworks" via evaluate_script
Tab 4: /compliance   → extract framework card count

Compare: dashboard endpoint count == endpoint list count
Compare: dashboard framework count == compliance card count
```

Output: `{RUN_ID}-correlation.json` with pass/fail per assertion.

### Phase 3: Deep Reviews (tiered)
For each page filtered by tier:

| Page Priority | Perspectives | Model |
|---|---|---|
| Critical (dashboard, compliance, deployments, endpoints) | UX + Enterprise + QA + PM + End User (5) | opus |
| Important (patches, CVEs, workflows, policies) | UX + Enterprise + QA (3) | sonnet |
| Peripheral (settings, admin, audit) | Assertions only (Phase 1) | — |

Each deep review runs the 4-phase single-page pipeline.

### Phase 4: Product Report
Main agent reads all findings files, runs impact scorer, groups root causes, writes product report.

### Incremental Mode

`/product-review-all --incremental`:

1. Read `.omniprod/last-review-commit` (saved from previous run)
2. `git diff --name-only {commit}..HEAD` → changed files
3. Map changed files to affected pages via `product-map.json`:
   - `web/src/pages/compliance/*` → `/compliance`, `/compliance/:id`
   - `packages/ui/src/DataTable.tsx` → all pages using DataTable
   - `internal/server/api/v1/endpoints.go` → `/endpoints`, `/dashboard`
4. Only run deep reviews for affected pages
5. Re-run correlation if entity graph edges affected
6. Carry forward findings for unchanged pages
7. Save `HEAD` to `.omniprod/last-review-commit`
8. Diff report: "4 pages re-reviewed. 3 findings fixed, 2 new, 45 unchanged."

## New Files

### `scripts/assertions-runner.js`

A self-contained JavaScript file injected into pages via `evaluate_script`. Contains assertion functions:

```javascript
// Categories of assertions:
// 1. data-integrity: Compare API response totals with DOM element counts
// 2. accessibility: Subset of axe-core checks (labels, contrast, roles)
// 3. console: Count errors/warnings in console
// 4. performance: Check LCP, CLS, network metrics
// 5. business-rules: Custom assertions from assertion-defs.json

// Returns: { results: [{ id, category, description, passed, actual, expected, severity }] }
```

### `scripts/generate-assertions.py`

Reads source code and generates `assertion-defs.json`:

Input: page URL, app directory, API hooks directory
Process:
1. Find the page component file
2. Extract API hooks used (e.g., `useEndpoints`, `useComplianceFrameworks`)
3. Find corresponding backend handler → query → expected data shape
4. Generate data-integrity assertions (API count vs DOM count)
5. Generate standard assertions (console errors, a11y basics, performance)
6. Generate business rule assertions from backend logic

Output: `assertion-defs.json`

### `commands/product-correlate.md`

Dedicated cross-page correlation command:

```
/product-correlate [--base-url=<url>]
```

Opens entity pages in parallel tabs, compares counts and statuses.
Uses entity graph from `product-map.json` to know what to compare.

## Modified Files

### `commands/product-review.md` — Major Rewrite
4 phases instead of 7. Coverage targets instead of capture plan. Assertions integrated. Model selection per phase.

### `commands/product-review-all.md` — Major Rewrite
Browser-free orchestrator. Tiered perspectives. Multi-tab health scan. Incremental mode.

### `commands/product-smoke.md` — Multi-Tab Enhancement
Batch loading with `new_page` for faster smoke testing.

### Script Fixes
- `cleanup-screenshots.sh`: Fix subshell counter bug (use temp file or process substitution)
- `findings-delta.py`: Migrate to argparse
- `impact-scorer.py`: Add observation to `--top` display, note O(n^2) for future
- `validate-capture.sh`: Validate against coverage-targets.json instead of capture-plan.json
- `check-findings.sh`: Fix exit code for `--critical-only`
- `evidence-packager.py`: Simplify — remove perspective filtering (all get all evidence)

### Retired Scripts
- `state-explorer.py` → replaced by coverage-targets checklist
- `parse-snapshot.py` → replaced by coverage-targets

### Perspective Changes
- Active (default): `ux-designer`, `enterprise-buyer`, `qa-engineer`
- Optional (critical pages): `product-manager`, `end-user`
- Inactive (kept for reference): `accessibility-expert`, `cto-architect`, `sales-engineer`

Config change in `.omniprod/config.json`:
```json
{
  "perspectives": {
    "essential": ["ux-designer", "enterprise-buyer", "qa-engineer"],
    "optional": ["product-manager", "end-user"],
    "inactive": ["accessibility-expert", "cto-architect", "sales-engineer"]
  }
}
```

## Plugin Version

Bump from 0.2.0 → 0.3.0 in `plugin.json`.

## Success Criteria

1. Single-page review completes in ~12 minutes (down from ~21)
2. Assertion failures catch data integrity bugs that v0.2.0 missed
3. Cross-page correlation finds entity count mismatches
4. Incremental mode reviews only changed pages
5. Finding noise reduced by ~40% (fewer duplicate findings from fewer perspectives)
6. All known script bugs fixed
7. Main orchestrator in product-review-all never touches browser
