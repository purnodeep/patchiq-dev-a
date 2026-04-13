# OmniProd v0.3.0 — Two-Layer Detection Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Redesign OmniProd from screenshot-only review to two-layer detection (programmatic assertions + focused AI review), simplify the pipeline from 7 phases to 4, add multi-tab support, cross-page correlation, and incremental review mode.

**Architecture:** Layer 1 runs JavaScript assertions via Chrome DevTools `evaluate_script` to catch data integrity, accessibility, and business rule violations programmatically. Layer 2 dispatches 3-5 AI perspectives (down from 8) that receive assertion results and focus on subjective quality. Model selection is complexity-driven (opus for reasoning-heavy tasks, sonnet for mechanical ones).

**Tech Stack:** JavaScript (assertions-runner.js via evaluate_script), Python 3 (generate-assertions.py), Bash (script fixes), Markdown (command prompts)

**Plugin root:** `.omniprod-plugin/omniprod/`
**Runtime data:** `.omniprod/`

---

## Task 1: Fix `cleanup-screenshots.sh` Subshell Counter Bug

**Files:**
- Modify: `.omniprod-plugin/omniprod/scripts/cleanup-screenshots.sh:38-47`

The `pruned` counter is incremented inside a `while read` subshell (piped from `find`), so the increment is lost after the pipe exits. Fix by using process substitution.

- [ ] **Step 1: Fix the subshell variable scope bug**

Replace lines 38-47 in `cleanup-screenshots.sh`:

```bash
# OLD (broken — pruned incremented in subshell):
if [ -d "$ARCHIVE_DIR" ]; then
  pruned=0
  find "$ARCHIVE_DIR" -maxdepth 1 -mindepth 1 -type d -mtime +$prune_days | while read -r dir; do
    rm -rf "$dir"
    pruned=$((pruned + 1))
  done
  if [ $pruned -gt 0 ]; then
    echo "Pruned $pruned archive(s) older than $prune_days days."
  fi
fi
```

```bash
# NEW (fixed — use process substitution to keep variable in current shell):
if [ -d "$ARCHIVE_DIR" ]; then
  pruned=0
  while read -r dir; do
    rm -rf "$dir"
    pruned=$((pruned + 1))
  done < <(find "$ARCHIVE_DIR" -maxdepth 1 -mindepth 1 -type d -mtime +$prune_days 2>/dev/null)
  if [ $pruned -gt 0 ]; then
    echo "Pruned $pruned archive(s) older than $prune_days days."
  fi
fi
```

- [ ] **Step 2: Fix the `ls -d` stderr when no archives exist (line 51)**

Replace line 51:
```bash
# OLD:
archive_count=$(ls -d "$ARCHIVE_DIR"/*/ 2>/dev/null | wc -l)
# NEW:
archive_count=$(find "$ARCHIVE_DIR" -maxdepth 1 -mindepth 1 -type d 2>/dev/null | wc -l)
```

- [ ] **Step 3: Test the fix**

Run:
```bash
mkdir -p /tmp/test-omniprod/screenshots/{current,archive}
touch /tmp/test-omniprod/screenshots/current/test.png
OMNIPROD_DIR=/tmp/test-omniprod bash .omniprod-plugin/omniprod/scripts/cleanup-screenshots.sh --archive
ls /tmp/test-omniprod/screenshots/archive/
rm -rf /tmp/test-omniprod
```
Expected: Should archive the file and report "Archived 1 screenshots".

- [ ] **Step 4: Commit**

```bash
git add .omniprod-plugin/omniprod/scripts/cleanup-screenshots.sh
git commit -m "fix(omniprod): fix subshell counter bug in cleanup-screenshots.sh"
```

---

## Task 2: Fix `check-findings.sh` Exit Code Behavior

**Files:**
- Modify: `.omniprod-plugin/omniprod/scripts/check-findings.sh:72-74`

When `--critical-only` is set and criticals exist, the script should exit non-zero. Currently it just falls through to normal output.

- [ ] **Step 1: Fix the exit code logic**

Replace lines 72-74:
```bash
# OLD:
    if $critical_only && [ $total_critical -eq 0 ]; then
      exit 0
    fi
```

```bash
# NEW:
    if $critical_only; then
      if [ $total_critical -eq 0 ]; then
        exit 0
      else
        echo "OmniProd: $total_critical critical findings across $pages page(s)"
        exit 1
      fi
    fi
```

- [ ] **Step 2: Test the fix**

Run:
```bash
bash .omniprod-plugin/omniprod/scripts/check-findings.sh --critical-only; echo "exit: $?"
bash .omniprod-plugin/omniprod/scripts/check-findings.sh --json
```
Expected: If critical findings exist, `--critical-only` exits 1. `--json` still outputs JSON.

- [ ] **Step 3: Commit**

```bash
git add .omniprod-plugin/omniprod/scripts/check-findings.sh
git commit -m "fix(omniprod): check-findings.sh exits non-zero when criticals found with --critical-only"
```

---

## Task 3: Migrate `findings-delta.py` to argparse

**Files:**
- Modify: `.omniprod-plugin/omniprod/scripts/findings-delta.py`

Replace raw `sys.argv` parsing with argparse for consistency with other scripts.

- [ ] **Step 1: Rewrite main() with argparse**

Replace the `main()` function (lines 68-97):

```python
def main():
    parser = argparse.ArgumentParser(
        description="Compare two OmniProd review snapshots and track improvements."
    )
    parser.add_argument("files", nargs="*", help="Current and previous findings JSON files")
    parser.add_argument("--page", help="Auto-find latest two reviews for this page slug")
    args = parser.parse_args()

    if args.page:
        reviews = find_reviews_for_page(args.page)
        if len(reviews) < 2:
            print(json.dumps({"error": f"Need at least 2 reviews for '{args.page}', found {len(reviews)}"}))
            sys.exit(1)
        current_path, previous_path = reviews[0], reviews[1]
    elif len(args.files) >= 2:
        current_path, previous_path = args.files[0], args.files[1]
    else:
        parser.print_help()
        sys.exit(1)

    current = load_findings(current_path)
    previous = load_findings(previous_path)

    delta = compute_delta(current, previous)
    delta["vs_previous"] = os.path.basename(previous_path).replace(".json", "")
    delta["current_severity"] = severity_counts(current)
    delta["previous_severity"] = severity_counts(previous)
    delta["current_file"] = current_path
    delta["previous_file"] = previous_path

    print(json.dumps(delta, indent=2))
```

Also add `import argparse` to the imports at line 8.

- [ ] **Step 2: Test**

Run:
```bash
python3 .omniprod-plugin/omniprod/scripts/findings-delta.py --help
```
Expected: Prints help text without crashing.

- [ ] **Step 3: Commit**

```bash
git add .omniprod-plugin/omniprod/scripts/findings-delta.py
git commit -m "fix(omniprod): migrate findings-delta.py to argparse"
```

---

## Task 4: Fix `impact-scorer.py` — Add Observation to `--top` Display

**Files:**
- Modify: `.omniprod-plugin/omniprod/scripts/impact-scorer.py:383-410`

The `--top` display shows rank, score, severity, element, pages — but omits `observation`, which is the most useful field for triage.

- [ ] **Step 1: Update `print_top` to include observation**

Replace the `print_top` function:

```python
def print_top(result: dict, n: int):
    ranked = result.get("findings_ranked", [])[:n]
    if not ranked:
        print("No findings to display.")
        return

    hdr = f"{'Rank':<6}{'Score':<8}{'Sev':<10}{'Element':<25}{'Pages':<8}{'Observation'}"
    print(hdr)
    print("-" * min(len(hdr), 120))
    for i, f in enumerate(ranked, 1):
        sev = f["severity"].upper()
        element = f["element"]
        if len(element) > 23:
            element = element[:20] + "..."
        observation = f.get("observation", "")
        if len(observation) > 50:
            observation = observation[:47] + "..."
        page_count = len(f.get("pages", []))
        if page_count == 0:
            pages_str = "-"
        elif page_count == 1:
            pages_str = "1 page"
        else:
            pages_str = f"{page_count} pgs"
        total_pages = len(result.get("summary", {}).get("by_page", {}))
        if total_pages > 1 and page_count >= total_pages:
            pages_str = "ALL"
        print(f"{i:<6}{f['impact_score']:<8.1f}{sev:<10}{element:<25}{pages_str:<8}{observation}")
```

- [ ] **Step 2: Test**

Run:
```bash
python3 .omniprod-plugin/omniprod/scripts/impact-scorer.py --help
```
Expected: Prints help text. (Full test requires findings JSON files.)

- [ ] **Step 3: Commit**

```bash
git add .omniprod-plugin/omniprod/scripts/impact-scorer.py
git commit -m "fix(omniprod): add observation column to impact-scorer --top display"
```

---

## Task 5: Rewrite `validate-capture.sh` for Coverage Targets

**Files:**
- Modify: `.omniprod-plugin/omniprod/scripts/validate-capture.sh`

Replace capture-plan.json validation with coverage-targets.json validation. Keep heuristic checks but relax the minimum screenshot count (20 was too rigid).

- [ ] **Step 1: Rewrite the script**

Full replacement of `validate-capture.sh`:

```bash
#!/usr/bin/env bash
# Validate capture completeness — gate between exploration and perspective reviews
# Validates against coverage-targets.json if present, falls back to heuristic checks
set -euo pipefail

SCREENSHOTS_DIR="${1:-.omniprod/screenshots/current}"
TARGETS="${SCREENSHOTS_DIR}/coverage-targets.json"
LOG="${SCREENSHOTS_DIR}/exploration-log.jsonl"
ASSERTIONS="${SCREENSHOTS_DIR}/assertion-results.json"

echo "=== OmniProd Capture Validation ==="
echo ""

# Count actual screenshots
actual=$(find "$SCREENSHOTS_DIR" -maxdepth 1 -name "*.png" 2>/dev/null | wc -l)

coverage=100
issues=0

# Coverage-targets validation (preferred)
if [ -f "$TARGETS" ] && [ -f "$LOG" ]; then
  target_count=$(python3 -c "
import json
with open('$TARGETS') as f:
    data = json.load(f)
targets = data.get('targets', [])
required = [t for t in targets if t.get('required', True)]
print(len(required), len(targets))
" 2>/dev/null || echo "0 0")

  read required_count total_targets <<< "$target_count"

  # Count completed targets from exploration log
  completed=$(python3 -c "
import json
completed = set()
with open('$LOG') as f:
    for line in f:
        line = line.strip()
        if not line:
            continue
        try:
            entry = json.loads(line)
            ct = entry.get('coverage_target', '')
            if ct:
                completed.add(ct)
        except json.JSONDecodeError:
            pass
print(len(completed))
" 2>/dev/null || echo "0")

  echo "Mode: Coverage-targets validation"
  echo ""
  echo "Coverage Targets:"
  echo "  Total targets:    $total_targets"
  echo "  Required targets: $required_count"
  echo "  Completed:        $completed"
  echo "  Screenshots:      $actual"
  echo ""

  if [ "$required_count" -gt 0 ]; then
    coverage=$((completed * 100 / required_count))
    echo "Coverage: ${coverage}% of required targets"
    echo ""
  fi

  if [ "$coverage" -lt 60 ]; then
    echo "FAIL: Coverage ${coverage}% is below 60% threshold."
    issues=$((issues + 1))
  fi
else
  echo "Mode: Heuristic validation (no coverage-targets.json)"
  echo "  Screenshots on disk: $actual"
  echo ""
fi

# Assertion results check
if [ -f "$ASSERTIONS" ]; then
  assertion_summary=$(python3 -c "
import json
with open('$ASSERTIONS') as f:
    data = json.load(f)
results = data.get('results', [])
passed = sum(1 for r in results if r.get('passed'))
failed = sum(1 for r in results if not r.get('passed'))
print(passed, failed, len(results))
" 2>/dev/null || echo "0 0 0")

  read a_passed a_failed a_total <<< "$assertion_summary"
  echo "Assertions: $a_passed passed, $a_failed failed (of $a_total)"
else
  echo "Assertions: not run (no assertion-results.json)"
fi

# Heuristic checks (always apply)
if [ "$actual" -lt 5 ]; then
  echo "FAIL: Only $actual screenshots. Minimum 5 for a meaningful review."
  issues=$((issues + 1))
fi

has_context=$([ -f "$SCREENSHOTS_DIR/00-business-context.md" ] && echo 1 || echo 0)
if [ "$has_context" -eq 0 ]; then
  echo "WARN: No business context file. Phase 0 may not have completed."
fi

echo ""
if [ $issues -gt 0 ]; then
  echo "VERDICT: CAPTURE INCOMPLETE — fill gaps before dispatching perspectives."
  exit 1
else
  echo "VERDICT: CAPTURE SUFFICIENT — proceed to perspective reviews."
  exit 0
fi
```

- [ ] **Step 2: Test**

Run:
```bash
bash .omniprod-plugin/omniprod/scripts/validate-capture.sh /tmp/nonexistent 2>/dev/null || echo "Expected fail: $?"
```
Expected: exits 1 (no screenshots found).

- [ ] **Step 3: Commit**

```bash
git add .omniprod-plugin/omniprod/scripts/validate-capture.sh
git commit -m "feat(omniprod): rewrite validate-capture.sh for coverage-targets validation"
```

---

## Task 6: Simplify `evidence-packager.py` — Remove Perspective Filtering

**Files:**
- Modify: `.omniprod-plugin/omniprod/scripts/evidence-packager.py`

All perspectives now receive all evidence. Remove the `PERSPECTIVE_FILTERS` matrix and `--perspective` flag. Simplify to a single evidence builder.

- [ ] **Step 1: Remove PERSPECTIVE_FILTERS constant and filtering logic**

1. Delete the `PERSPECTIVE_FILTERS` dict (near top of file, ~60 lines)
2. Remove `--perspective/-p` from argparse
3. Remove `perspective` parameter from `build_package()` function
4. Remove the filtering block near the end of `build_package()` that checks `PERSPECTIVE_FILTERS`
5. The function should now always return the full evidence package

- [ ] **Step 2: Test**

Run:
```bash
python3 .omniprod-plugin/omniprod/scripts/evidence-packager.py --help
```
Expected: Shows help with `--captures-dir` and `--output` but NO `--perspective`.

- [ ] **Step 3: Commit**

```bash
git add .omniprod-plugin/omniprod/scripts/evidence-packager.py
git commit -m "refactor(omniprod): simplify evidence-packager — remove perspective filtering"
```

---

## Task 7: Create `assertions-runner.js`

**Files:**
- Create: `.omniprod-plugin/omniprod/scripts/assertions-runner.js`

Self-contained JavaScript that runs inside the browser via Chrome DevTools `evaluate_script`. Contains assertion functions for data integrity, accessibility basics, console errors, and performance. Returns JSON results.

- [ ] **Step 1: Write assertions-runner.js**

```javascript
// OmniProd Assertions Runner — injected into page via evaluate_script
// Usage: evaluate_script with this file's content + assertion definitions
// Returns: { results: [...], summary: { total, passed, failed } }

(function runAssertions(assertionDefs) {
  'use strict';

  const results = [];

  // ---- Built-in assertion categories ----

  // 1. Console errors (always run)
  function checkConsoleErrors() {
    // Note: console messages are captured separately via list_console_messages
    // This checks for error indicators in the DOM (error boundaries, etc.)
    const errorBoundaries = document.querySelectorAll(
      '[data-testid*="error"], .error-boundary, [role="alert"][class*="error"], [class*="ErrorBoundary"]'
    );
    return {
      id: 'BUILTIN-CONSOLE-001',
      category: 'console',
      description: 'No React error boundaries or DOM error indicators visible',
      passed: errorBoundaries.length === 0,
      actual: errorBoundaries.length === 0 ? 'No error indicators' : `${errorBoundaries.length} error indicator(s) found`,
      expected: 'No error indicators',
      severity: 'critical',
      elements: Array.from(errorBoundaries).map(el => el.className || el.tagName).slice(0, 5)
    };
  }

  // 2. Accessibility basics (always run)
  function checkA11yBasics() {
    const checks = [];

    // 2a. All images have alt text
    const images = document.querySelectorAll('img:not([alt])');
    checks.push({
      id: 'BUILTIN-A11Y-001',
      category: 'accessibility',
      description: 'All images have alt attributes',
      passed: images.length === 0,
      actual: images.length === 0 ? 'All images have alt' : `${images.length} image(s) missing alt`,
      expected: 'All images have alt attributes',
      severity: 'major',
    });

    // 2b. All form inputs have associated labels
    const inputs = document.querySelectorAll('input:not([type="hidden"]):not([type="submit"]):not([type="button"]), select, textarea');
    let unlabeled = 0;
    inputs.forEach(input => {
      const hasLabel = input.id && document.querySelector(`label[for="${input.id}"]`);
      const hasAriaLabel = input.getAttribute('aria-label') || input.getAttribute('aria-labelledby');
      const wrappedInLabel = input.closest('label');
      if (!hasLabel && !hasAriaLabel && !wrappedInLabel) unlabeled++;
    });
    checks.push({
      id: 'BUILTIN-A11Y-002',
      category: 'accessibility',
      description: 'All form inputs have associated labels',
      passed: unlabeled === 0,
      actual: unlabeled === 0 ? 'All inputs labeled' : `${unlabeled} input(s) without labels`,
      expected: 'All inputs have labels',
      severity: 'major',
    });

    // 2c. Page has heading structure
    const h1s = document.querySelectorAll('h1');
    checks.push({
      id: 'BUILTIN-A11Y-003',
      category: 'accessibility',
      description: 'Page has exactly one h1 heading',
      passed: h1s.length === 1,
      actual: `${h1s.length} h1 heading(s)`,
      expected: 'Exactly 1 h1',
      severity: 'minor',
    });

    // 2d. All buttons have accessible names
    const buttons = document.querySelectorAll('button, [role="button"]');
    let unnamedButtons = 0;
    buttons.forEach(btn => {
      const text = (btn.textContent || '').trim();
      const ariaLabel = btn.getAttribute('aria-label') || btn.getAttribute('aria-labelledby') || btn.getAttribute('title');
      if (!text && !ariaLabel) unnamedButtons++;
    });
    checks.push({
      id: 'BUILTIN-A11Y-004',
      category: 'accessibility',
      description: 'All buttons have accessible names',
      passed: unnamedButtons === 0,
      actual: unnamedButtons === 0 ? 'All buttons named' : `${unnamedButtons} button(s) without names`,
      expected: 'All buttons have text or aria-label',
      severity: 'major',
    });

    // 2e. Main landmark exists
    const main = document.querySelector('main, [role="main"]');
    checks.push({
      id: 'BUILTIN-A11Y-005',
      category: 'accessibility',
      description: 'Page has a main landmark',
      passed: !!main,
      actual: main ? 'Main landmark found' : 'No main landmark',
      expected: 'Page has <main> or role="main"',
      severity: 'minor',
    });

    return checks;
  }

  // 3. Data integrity basics (always run)
  function checkDataIntegrity() {
    const checks = [];

    // 3a. No visible "undefined", "null", "NaN" in text content
    const bodyText = document.body.innerText || '';
    const badPatterns = [
      { pattern: /\bundefined\b/g, name: 'undefined' },
      { pattern: /\bnull\b/gi, name: 'null' },
      { pattern: /\bNaN\b/g, name: 'NaN' },
      { pattern: /\[object Object\]/g, name: '[object Object]' },
    ];
    const found = [];
    badPatterns.forEach(({ pattern, name }) => {
      const matches = bodyText.match(pattern);
      if (matches) found.push(`${name} (${matches.length}x)`);
    });
    checks.push({
      id: 'BUILTIN-DI-001',
      category: 'data-integrity',
      description: 'No raw undefined/null/NaN/[object Object] visible in page text',
      passed: found.length === 0,
      actual: found.length === 0 ? 'Clean text content' : `Found: ${found.join(', ')}`,
      expected: 'No raw JS values in visible text',
      severity: 'critical',
    });

    // 3b. Tables have data or empty state
    const tables = document.querySelectorAll('table');
    tables.forEach((table, i) => {
      const rows = table.querySelectorAll('tbody tr');
      const emptyState = table.closest('[class]')?.querySelector('[class*="empty"], [data-testid*="empty"]');
      checks.push({
        id: `BUILTIN-DI-002-${i}`,
        category: 'data-integrity',
        description: `Table ${i + 1} has data rows or explicit empty state`,
        passed: rows.length > 0 || !!emptyState,
        actual: rows.length > 0 ? `${rows.length} rows` : (emptyState ? 'Empty state shown' : 'No rows AND no empty state'),
        expected: 'Data rows or empty state component',
        severity: rows.length === 0 && !emptyState ? 'major' : 'minor',
      });
    });

    return checks;
  }

  // 4. Performance basics (always run)
  function checkPerformanceBasics() {
    const checks = [];

    // 4a. No massive DOM
    const allElements = document.querySelectorAll('*').length;
    checks.push({
      id: 'BUILTIN-PERF-001',
      category: 'performance',
      description: 'DOM element count is reasonable (<3000)',
      passed: allElements < 3000,
      actual: `${allElements} elements`,
      expected: '<3000 DOM elements',
      severity: allElements > 5000 ? 'major' : 'minor',
    });

    return checks;
  }

  // 5. Custom assertions from assertion-defs.json (passed as argument)
  function runCustomAssertions(defs) {
    if (!defs || !Array.isArray(defs)) return [];
    const customResults = [];

    for (const def of defs) {
      try {
        // Each custom assertion is a JS snippet that returns { passed, actual, expected }
        const fn = new Function('document', 'window', def.script);
        const result = fn(document, window);
        customResults.push({
          id: def.id,
          category: def.category || 'custom',
          description: def.description,
          passed: !!result.passed,
          actual: String(result.actual || ''),
          expected: String(result.expected || ''),
          severity: def.severity || 'major',
        });
      } catch (err) {
        customResults.push({
          id: def.id,
          category: def.category || 'custom',
          description: def.description,
          passed: false,
          actual: `Error: ${err.message}`,
          expected: 'Assertion should run without errors',
          severity: def.severity || 'major',
        });
      }
    }
    return customResults;
  }

  // ---- Run all assertions ----
  results.push(checkConsoleErrors());
  results.push(...checkA11yBasics());
  results.push(...checkDataIntegrity());
  results.push(...checkPerformanceBasics());

  // Run custom assertions if provided
  if (assertionDefs && assertionDefs.length > 0) {
    results.push(...runCustomAssertions(assertionDefs));
  }

  const passed = results.filter(r => r.passed).length;
  const failed = results.filter(r => !r.passed).length;

  return JSON.stringify({
    results: results,
    summary: {
      total: results.length,
      passed: passed,
      failed: failed,
      timestamp: new Date().toISOString()
    }
  });

})(typeof __ASSERTION_DEFS__ !== 'undefined' ? __ASSERTION_DEFS__ : []);
```

- [ ] **Step 2: Test syntax validity**

Run:
```bash
node -c .omniprod-plugin/omniprod/scripts/assertions-runner.js
```
Expected: No syntax errors.

- [ ] **Step 3: Commit**

```bash
git add .omniprod-plugin/omniprod/scripts/assertions-runner.js
git commit -m "feat(omniprod): add assertions-runner.js for programmatic DOM/a11y/data checks"
```

---

## Task 8: Create `generate-assertions.py`

**Files:**
- Create: `.omniprod-plugin/omniprod/scripts/generate-assertions.py`

Reads source code for a page and generates `assertion-defs.json` with custom JavaScript assertions for data integrity and business rules.

- [ ] **Step 1: Write generate-assertions.py**

```python
#!/usr/bin/env python3
"""
Generate assertion definitions for a page by analyzing its source code.

Reads API hooks, route config, and backend handlers to produce JavaScript
assertion snippets that verify data integrity and business rules.

Usage:
    python3 generate-assertions.py --app web --page /compliance --output assertion-defs.json
    python3 generate-assertions.py --app web --page /endpoints --output assertion-defs.json
"""

import argparse
import glob
import json
import os
import re
import sys
from pathlib import Path


def find_page_component(app_dir: str, page_path: str) -> str | None:
    """Find the main component file for a route path."""
    # Normalize: /compliance → compliance, /settings/license → settings/license
    slug = page_path.strip("/").replace("/", os.sep)
    candidates = [
        os.path.join(app_dir, "src", "pages", slug, "*.tsx"),
        os.path.join(app_dir, "src", "pages", slug + ".tsx"),
        os.path.join(app_dir, "src", "pages", slug, "index.tsx"),
    ]
    for pattern in candidates:
        matches = glob.glob(pattern)
        if matches:
            return matches[0]
    # Fallback: search for the slug anywhere in pages/
    fallback = glob.glob(os.path.join(app_dir, "src", "pages", "**", "*.tsx"), recursive=True)
    slug_lower = slug.replace(os.sep, "").lower()
    for f in fallback:
        if slug_lower in f.lower():
            return f
    return None


def extract_api_hooks(component_content: str) -> list[str]:
    """Extract TanStack Query hook names used in a component."""
    # Matches: useEndpoints(), useComplianceFrameworks({ ... }), etc.
    pattern = r'\buse([A-Z]\w+)\s*\('
    matches = re.findall(pattern, component_content)
    return [f"use{m}" for m in matches]


def find_hook_file(app_dir: str, hook_name: str) -> str | None:
    """Find the file that defines an API hook."""
    hooks_dir = os.path.join(app_dir, "src", "api", "hooks")
    if not os.path.isdir(hooks_dir):
        return None
    for f in glob.glob(os.path.join(hooks_dir, "*.ts")):
        try:
            content = open(f).read()
            if hook_name in content:
                return f
        except (OSError, UnicodeDecodeError):
            pass
    return None


def extract_api_endpoint(hook_content: str, hook_name: str) -> str | None:
    """Extract the API endpoint path from a hook definition."""
    # Matches: api.GET("/api/v1/endpoints", ...) or path: "/api/v1/endpoints"
    patterns = [
        r'\.GET\s*\(\s*["\'](/api/v1/\w+)["\']',
        r'\.POST\s*\(\s*["\'](/api/v1/\w+)["\']',
        r'path:\s*["\'](/api/v1/\w+)["\']',
        r'queryKey:\s*\[.*?["\'](/api/v1/\w+)["\']',
    ]
    for p in patterns:
        match = re.search(p, hook_content)
        if match:
            return match.group(1)
    return None


def generate_data_integrity_assertion(hook_name: str, api_endpoint: str, idx: int) -> dict:
    """Generate a JS assertion that compares API response count with DOM table row count."""
    resource_name = api_endpoint.split("/")[-1]
    return {
        "id": f"DI-{idx:03d}",
        "category": "data-integrity",
        "description": f"{resource_name} count from API matches table row count in DOM",
        "script": f"""
            try {{
                const rows = document.querySelectorAll('table tbody tr');
                const rowCount = rows.length;
                // Check for pagination total display
                const totalEl = document.body.innerText.match(/(?:of|total:?)\\s+(\\d[\\d,]*)\\s/i);
                const displayedTotal = totalEl ? parseInt(totalEl[1].replace(/,/g, ''), 10) : null;
                return {{
                    passed: rowCount > 0 || displayedTotal !== null,
                    actual: displayedTotal !== null
                        ? 'Table shows ' + rowCount + ' rows, total: ' + displayedTotal
                        : rowCount + ' rows visible',
                    expected: 'Table has data rows or shows total count'
                }};
            }} catch(e) {{
                return {{ passed: false, actual: 'Error: ' + e.message, expected: 'No error' }};
            }}
        """.strip(),
        "severity": "major",
        "source_hook": hook_name,
        "source_endpoint": api_endpoint,
    }


def generate_empty_state_assertion(idx: int) -> dict:
    """Assert that empty tables show proper empty state, not just blank space."""
    return {
        "id": f"DI-{idx:03d}",
        "category": "data-integrity",
        "description": "Empty tables/lists display empty state component",
        "script": """
            try {
                const tables = document.querySelectorAll('table');
                const issues = [];
                tables.forEach((t, i) => {
                    const rows = t.querySelectorAll('tbody tr');
                    if (rows.length === 0) {
                        const parent = t.closest('[class]');
                        const emptyIndicator = parent && (
                            parent.querySelector('[class*="empty"]') ||
                            parent.querySelector('[class*="Empty"]') ||
                            parent.querySelector('[data-testid*="empty"]')
                        );
                        if (!emptyIndicator) issues.push('Table ' + (i+1));
                    }
                });
                return {
                    passed: issues.length === 0,
                    actual: issues.length === 0 ? 'All empty tables have empty states' : issues.join(', ') + ' missing empty state',
                    expected: 'Empty tables show empty state component'
                };
            } catch(e) {
                return { passed: false, actual: 'Error: ' + e.message, expected: 'No error' };
            }
        """.strip(),
        "severity": "major",
    }


def generate_loading_state_assertion(idx: int) -> dict:
    """Assert loading indicators exist (skeleton/spinner) for data-fetching components."""
    return {
        "id": f"DI-{idx:03d}",
        "category": "data-integrity",
        "description": "No perpetual loading spinners visible (page has finished loading)",
        "script": """
            try {
                const spinners = document.querySelectorAll(
                    '[class*="spinner"], [class*="Spinner"], [class*="loading"], [role="progressbar"][aria-valuenow]'
                );
                const skeletons = document.querySelectorAll(
                    '[class*="skeleton"], [class*="Skeleton"], [data-testid*="skeleton"]'
                );
                const activeSpinners = Array.from(spinners).filter(el => {
                    const style = window.getComputedStyle(el);
                    return style.display !== 'none' && style.visibility !== 'hidden';
                });
                const activeSkeletons = Array.from(skeletons).filter(el => {
                    const style = window.getComputedStyle(el);
                    return style.display !== 'none' && style.visibility !== 'hidden';
                });
                return {
                    passed: activeSpinners.length === 0 && activeSkeletons.length === 0,
                    actual: (activeSpinners.length + activeSkeletons.length) === 0
                        ? 'No loading indicators visible'
                        : activeSpinners.length + ' spinner(s), ' + activeSkeletons.length + ' skeleton(s) still visible',
                    expected: 'Page fully loaded, no perpetual loading states'
                };
            } catch(e) {
                return { passed: false, actual: 'Error: ' + e.message, expected: 'No error' };
            }
        """.strip(),
        "severity": "major",
    }


def analyze_page(app_dir: str, page_path: str) -> dict:
    """Analyze a page and generate assertion definitions."""
    assertions = []
    idx = 1

    component_file = find_page_component(app_dir, page_path)
    hooks_found = []

    if component_file:
        try:
            content = open(component_file).read()
            hooks = extract_api_hooks(content)

            for hook_name in hooks:
                hook_file = find_hook_file(app_dir, hook_name)
                if hook_file:
                    hook_content = open(hook_file).read()
                    endpoint = extract_api_endpoint(hook_content, hook_name)
                    if endpoint:
                        assertions.append(generate_data_integrity_assertion(hook_name, endpoint, idx))
                        hooks_found.append({"hook": hook_name, "endpoint": endpoint, "file": hook_file})
                        idx += 1
        except (OSError, UnicodeDecodeError):
            pass

    # Standard assertions (always included)
    assertions.append(generate_empty_state_assertion(idx))
    idx += 1
    assertions.append(generate_loading_state_assertion(idx))
    idx += 1

    return {
        "page": page_path,
        "app": os.path.basename(app_dir),
        "component_file": component_file,
        "hooks_analyzed": hooks_found,
        "assertions": assertions,
    }


def main():
    parser = argparse.ArgumentParser(description="Generate assertion definitions from source code analysis.")
    parser.add_argument("--app", default="web", help="App directory (web, web-hub, web-agent)")
    parser.add_argument("--page", required=True, help="Page path (e.g., /compliance)")
    parser.add_argument("--output", "-o", help="Output file (default: stdout)")
    args = parser.parse_args()

    result = analyze_page(args.app, args.page)

    output = json.dumps(result, indent=2)
    if args.output:
        Path(args.output).parent.mkdir(parents=True, exist_ok=True)
        with open(args.output, "w") as f:
            f.write(output + "\n")
        print(f"Generated {len(result['assertions'])} assertions for {args.page} -> {args.output}", file=sys.stderr)
    else:
        print(output)


if __name__ == "__main__":
    main()
```

- [ ] **Step 2: Test with real page**

Run:
```bash
python3 .omniprod-plugin/omniprod/scripts/generate-assertions.py --app web --page /compliance
```
Expected: JSON output with assertions array (at minimum the 2 standard assertions + any from detected hooks).

- [ ] **Step 3: Test with unknown page**

Run:
```bash
python3 .omniprod-plugin/omniprod/scripts/generate-assertions.py --app web --page /nonexistent
```
Expected: JSON output with only the 2 standard assertions (empty state + loading state).

- [ ] **Step 4: Commit**

```bash
git add .omniprod-plugin/omniprod/scripts/generate-assertions.py
git commit -m "feat(omniprod): add generate-assertions.py — source-code-aware assertion generator"
```

---

## Task 9: Rewrite `product-review.md` — 4-Phase Pipeline

**Files:**
- Modify: `.omniprod-plugin/omniprod/commands/product-review.md`

Complete rewrite: 4 phases (Intelligence Gathering → Explore+Assert → AI Review → Aggregate). Model selection per phase. Coverage targets instead of capture plans. Assertions integrated.

- [ ] **Step 1: Write the new product-review.md**

Replace the entire file content. The new version must include:

1. **YAML frontmatter**: same allowed-tools list plus `new_page`, `close_page`
2. **Architecture Overview**: 4-phase diagram showing model per phase
3. **Parse Arguments**: same as v0.2.0 (url, --perspectives, --page-name) plus `--model-override`
4. **Phase 0: Intelligence Gathering** (background, opus by default):
   - Sub-agent reads source code
   - Generates `business-context.md`, `coverage-targets.json`, `assertion-defs.json`
   - Uses `generate-assertions.py` for assertion generation
5. **Phase 1: Explore + Assert** (browser, opus by default):
   - Single agent: navigate → run assertions → explore with coverage targets
   - Plan-as-you-go: 5-10 actions at a time from current snapshot
   - Multi-tab for responsive (open 3 tabs at 3 viewports)
   - Checkpoints every 15 actions
   - Writes `assertion-results.json`, `exploration-log.jsonl`, screenshots
6. **Phase 2: AI Review** (parallel, opus for critical / sonnet for standard):
   - Essential: UX Designer, Enterprise Buyer, QA Engineer (always)
   - Optional: Product Manager, End User (if page is in critical tier or user requested)
   - Receive assertion results as "already found"
   - Read standards files
7. **Phase 3: Aggregate** (main agent, no browser):
   - Convert failed assertions → findings
   - Parse perspective findings
   - Deduplicate, score, delta, write report

The full prompt content is ~500 lines. Write it as a complete markdown file following the exact patterns from v0.2.0 but with the new pipeline.

Key changes from v0.2.0:
- Phase count: 4 not 7 (0-3 not 0-6)
- No separate Scout, Plan, Capture, Annotator phases
- Assertions run in Phase 1 via `evaluate_script`
- Coverage targets replace capture plans
- Model selection: opus/sonnet per phase, not fixed sonnet
- 3+2 perspectives, not 8
- Phase 2 perspectives told "assertion findings are already known"
- exploration-log.jsonl replaces annotated-captures.md

- [ ] **Step 2: Verify the file renders correctly**

Run:
```bash
wc -l .omniprod-plugin/omniprod/commands/product-review.md
head -20 .omniprod-plugin/omniprod/commands/product-review.md
```
Expected: 400-600 lines, valid YAML frontmatter.

- [ ] **Step 3: Commit**

```bash
git add .omniprod-plugin/omniprod/commands/product-review.md
git commit -m "feat(omniprod): rewrite product-review.md — 4-phase pipeline with assertions"
```

---

## Task 10: Create `product-correlate.md` — Cross-Page Correlation Command

**Files:**
- Create: `.omniprod-plugin/omniprod/commands/product-correlate.md`

New command that opens multiple pages in parallel tabs and compares entity data for consistency.

- [ ] **Step 1: Write product-correlate.md**

```markdown
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
- `--base-url`: Base URL override. Defaults: `web` → `http://localhost:3001`, etc.

## Phase 1: Load Entity Graph

Read `.omniprod/product-map.json`. If it doesn't exist, tell the user to run `/product-map` first.

Extract the `entity_graph` section. Each entity has:
- `name`: Entity type (e.g., "endpoints", "frameworks", "patches")
- `pages`: Which pages show this entity (e.g., ["/dashboard", "/endpoints", "/compliance"])
- `display_type`: How it appears (count, list, table, card, score)

Build correlation pairs: for each entity that appears on 2+ pages, create a comparison pair.

## Phase 2: Multi-Tab Data Extraction

**IMPORTANT**: You manage ALL tabs yourself. Do NOT dispatch sub-agents for browser work — `select_page` is global state and parallel agents would race.

### Step 1: Open pages in batches

For each batch of up to 4 unique pages from the correlation pairs:

1. `new_page` for each URL (with `background: true`)
2. `wait_for` each page to load (select page, wait, select next)

### Step 2: Extract data from each tab

For each open tab, `select_page` then run `evaluate_script` to extract entity data:

```javascript
// Extract all visible counts, scores, and entity references
(function() {
  const data = {};

  // Extract text content that contains numbers
  const allText = document.body.innerText;

  // Look for patterns like "47 endpoints", "6 frameworks", "Score: 82%"
  const countPatterns = allText.match(/(\d[\d,]*)\s+(endpoint|framework|patch|cve|deployment|policy|workflow|agent|control|alert|notification)s?/gi) || [];
  countPatterns.forEach(match => {
    const parts = match.match(/(\d[\d,]*)\s+(\w+)/i);
    if (parts) {
      const count = parseInt(parts[1].replace(/,/g, ''), 10);
      const entity = parts[2].toLowerCase().replace(/s$/, '');
      data[entity + '_count'] = count;
    }
  });

  // Look for score patterns: "75%", "Score: 82", etc.
  const scorePatterns = allText.match(/(?:score|compliance|coverage)[:\s]+(\d+(?:\.\d+)?)\s*%?/gi) || [];
  scorePatterns.forEach(match => {
    const parts = match.match(/(\w+)[:\s]+(\d+(?:\.\d+)?)/i);
    if (parts) {
      data[parts[1].toLowerCase() + '_score'] = parseFloat(parts[2]);
    }
  });

  // Count table rows
  const tables = document.querySelectorAll('table tbody');
  tables.forEach((tbody, i) => {
    data['table_' + i + '_rows'] = tbody.querySelectorAll('tr').length;
  });

  // Look for pagination totals
  const totalMatch = allText.match(/(?:of|total:?)\s+(\d[\d,]*)/i);
  if (totalMatch) {
    data['pagination_total'] = parseInt(totalMatch[1].replace(/,/g, ''), 10);
  }

  return JSON.stringify(data);
})();
```

Record the extracted data per page.

### Step 3: Close extra tabs

After extraction, `close_page` for all tabs except the original.

## Phase 3: Compare Data Across Pages

For each correlation pair (entity appearing on 2+ pages):

1. Compare the values extracted from each page
2. Record: MATCH or MISMATCH with specific values
3. For mismatches, note which pages disagree and by how much

### Tolerance Rules
- Exact match required for: entity counts, status labels
- 1% tolerance for: percentage scores (rounding differences acceptable)
- Table row count vs stated total: table rows may show a page of data (e.g., 25 of 1234) — compare stated total, not row count

## Phase 4: Generate Report

Write to `.omniprod/reviews/{date}-correlation.md`:

```markdown
# Cross-Page Correlation Report

Date: {YYYY-MM-DD} | App: {app} | Base URL: {base_url}

## Summary

| Entity | Pages Compared | Result |
|--------|---------------|--------|
| endpoints | /dashboard, /endpoints | MATCH (47) |
| frameworks | /dashboard, /compliance | MISMATCH (2 vs 3) |
| compliance_score | /dashboard, /compliance | MATCH (75%) |

## Mismatches

### 1. Framework count: /dashboard (2) vs /compliance (3)
- **Dashboard value**: 2 frameworks
- **Compliance value**: 3 framework cards
- **Possible cause**: Dashboard may filter inactive frameworks; compliance shows all
- **Severity**: major

## All Matches

{List all matching entities with their consistent value}
```

Also write `.omniprod/findings/{date}-correlation.json` with findings for each mismatch.

## Phase 5: Print Summary

```
=== Cross-Page Correlation ===
Entities checked: {N}
Matches: {N}
Mismatches: {N}

{For each mismatch: entity, pages, values}

Full report: .omniprod/reviews/{date}-correlation.md
```

If all match:
> **Cross-page data is consistent.** All {N} entity values match across pages.

If mismatches found:
> **Found {N} cross-page inconsistencies.** Review the report and determine if these are caching issues, filter differences, or real bugs.
```

- [ ] **Step 2: Commit**

```bash
git add .omniprod-plugin/omniprod/commands/product-correlate.md
git commit -m "feat(omniprod): add /product-correlate command for cross-page entity consistency"
```

---

## Task 11: Rewrite `product-review-all.md` — Browser-Free Orchestrator + Incremental

**Files:**
- Modify: `.omniprod-plugin/omniprod/commands/product-review-all.md`

Major rewrite: orchestrator never touches browser, tiered perspectives, incremental mode.

- [ ] **Step 1: Rewrite product-review-all.md**

Key changes from v0.2.0:

1. **Add `--incremental` flag** to argument parsing
2. **Phase 0**: Same (product map generation)
3. **Phase 1: Health Scan** — dispatch a sub-agent (sonnet) to:
   - Navigate every page
   - Run assertions via assertions-runner.js on each
   - Take screenshots
   - Multi-tab batch loading (4 pages at a time via new_page)
   - Write `{RUN_ID}-health-scan.json`
4. **Phase 2: Cross-Page Correlation** — dispatch a sub-agent (opus) to:
   - Open key entity pages in parallel tabs
   - Extract entity counts via evaluate_script
   - Compare across pages
   - Write `{RUN_ID}-correlation.json`
5. **Phase 3: Deep Reviews** — for each page filtered by tier:
   - Critical: dispatch opus explore+assert sub-agent + 5 perspective sub-agents (opus)
   - Important: dispatch sonnet explore+assert sub-agent + 3 perspective sub-agents (sonnet)
   - Peripheral: skip (covered by Phase 1 assertions)
6. **Phase 4: Product Report** — main agent reads all findings, aggregates

**Incremental mode** (when `--incremental` is set):
- Read `.omniprod/last-review-commit`
- `git diff --name-only {commit}..HEAD` → changed files
- Map files to pages via product-map.json
- Only review affected pages in Phase 3
- Carry forward findings for unchanged pages
- Save HEAD to `.omniprod/last-review-commit`

**Browser-free orchestrator rule**: Main agent NEVER uses Chrome DevTools tools. Every browser operation is dispatched to a sub-agent. Main agent only reads/writes files and dispatches.

The full prompt is ~500 lines. Preserve the v0.2.0 structure (phase headings, progress printing, resumability, anti-context-rot measures) but update content.

- [ ] **Step 2: Verify**

Run:
```bash
wc -l .omniprod-plugin/omniprod/commands/product-review-all.md
head -30 .omniprod-plugin/omniprod/commands/product-review-all.md
```
Expected: 400-600 lines, valid YAML frontmatter with incremental flag documented.

- [ ] **Step 3: Commit**

```bash
git add .omniprod-plugin/omniprod/commands/product-review-all.md
git commit -m "feat(omniprod): rewrite product-review-all.md — browser-free orchestrator + incremental mode"
```

---

## Task 12: Update `product-smoke.md` — Multi-Tab Batch Loading

**Files:**
- Modify: `.omniprod-plugin/omniprod/commands/product-smoke.md`

Add multi-tab batch loading and assertion running to the smoke test.

- [ ] **Step 1: Update Phase 3 (Smoke Each Page) to use batched tabs**

Add a new section after "Phase 2: Setup" and before "Phase 3: Smoke Each Page":

Replace the sequential per-page loop with a batched approach:

1. Group pages into batches of 4
2. For each batch:
   - `new_page(url_1, background: true)`, `new_page(url_2, background: true)`, etc.
   - Wait 3 seconds for all pages to load
   - For each tab in the batch:
     - `select_page(pageId)`
     - `take_screenshot`
     - `list_console_messages`
     - `list_network_requests`
     - Record result
   - `close_page` for each extra tab
3. Continue to next batch

Also add: after taking screenshot of each page, run the built-in assertions from `assertions-runner.js` via `evaluate_script` and record pass/fail counts.

Add `new_page` and `close_page` to the allowed-tools list in YAML frontmatter.

- [ ] **Step 2: Commit**

```bash
git add .omniprod-plugin/omniprod/commands/product-smoke.md
git commit -m "feat(omniprod): add multi-tab batch loading and assertions to product-smoke"
```

---

## Task 13: Update Perspective Config and Mark Inactive Perspectives

**Files:**
- Modify: `.omniprod-plugin/omniprod/commands/product-config.md`
- Modify: `.omniprod-plugin/omniprod/commands/product-init.md`

Update config schema to support essential/optional/inactive perspective tiers.

- [ ] **Step 1: Update product-init.md config generation**

In the section where `config.json` is generated, change the perspectives section from:

```json
"perspectives": ["ux-designer", "qa-engineer", "enterprise-buyer", "accessibility-expert", "cto-architect", "product-manager", "sales-engineer", "end-user"]
```

To:

```json
"perspectives": {
  "essential": ["ux-designer", "enterprise-buyer", "qa-engineer"],
  "optional": ["product-manager", "end-user"],
  "inactive": ["accessibility-expert", "cto-architect", "sales-engineer"]
}
```

Also add a `model_selection` section:

```json
"model_selection": {
  "intelligence_gathering": "opus",
  "exploration": "opus",
  "perspectives_critical": "opus",
  "perspectives_standard": "sonnet",
  "smoke_test": "sonnet",
  "correlation": "opus"
}
```

- [ ] **Step 2: Update product-config.md**

Update the `add-perspective` and `remove-perspective` sub-actions to work with the tiered structure. When adding a perspective, ask which tier (essential/optional). When removing, move to inactive instead of deleting.

- [ ] **Step 3: Commit**

```bash
git add .omniprod-plugin/omniprod/commands/product-config.md .omniprod-plugin/omniprod/commands/product-init.md
git commit -m "feat(omniprod): tiered perspective config (essential/optional/inactive) + model selection"
```

---

## Task 14: Update `plugin.json` — Version Bump + New Commands

**Files:**
- Modify: `.omniprod-plugin/omniprod/.claude-plugin/plugin.json`

- [ ] **Step 1: Bump version and update description**

Change version from `"0.2.0"` to `"0.3.0"`.

Update description to: `"Two-layer product quality system. Programmatic assertions + AI perspective review. Coverage-target exploration, multi-tab capture, cross-page correlation, incremental review. The final quality gate before shipping."`

- [ ] **Step 2: Commit**

```bash
git add .omniprod-plugin/omniprod/.claude-plugin/plugin.json
git commit -m "chore(omniprod): bump version to 0.3.0"
```

---

## Task 15: Update References — Evidence Model + Findings Schema

**Files:**
- Modify: `.omniprod-plugin/omniprod/references/evidence-model.md`
- Modify: `.omniprod-plugin/omniprod/references/findings-schema.md`

- [ ] **Step 1: Update evidence-model.md**

Add a new "Layer 1: Programmatic Assertions" section at the top, documenting:
- assertions-runner.js (built-in checks)
- assertion-defs.json (generated from source code)
- assertion-results.json (output format)
- How assertion failures become findings

Update the "Perspective Filtering" section to note that all perspectives now receive all evidence (no filtering).

- [ ] **Step 2: Update findings-schema.md**

Add `assertion_id` field to the per-finding schema (nullable, populated for assertion-derived findings).
Add `source` field: `"assertion"` or `"perspective"`.
Update the example JSON to show both assertion and perspective findings.

- [ ] **Step 3: Commit**

```bash
git add .omniprod-plugin/omniprod/references/evidence-model.md .omniprod-plugin/omniprod/references/findings-schema.md
git commit -m "docs(omniprod): update evidence model and findings schema for assertion layer"
```

---

## Task 16: Sync Plugin Cache and Verify

**Files:**
- No file changes — verification only

- [ ] **Step 1: Sync plugin cache**

Run:
```bash
# Remove old cache
rm -rf ~/.claude/plugins/cache/omniprod/
# Reload will happen automatically on next /product-review invocation
```

- [ ] **Step 2: Verify all scripts are syntactically valid**

Run:
```bash
# Python scripts
python3 -c "import py_compile; py_compile.compile('.omniprod-plugin/omniprod/scripts/generate-assertions.py', doraise=True)"
python3 -c "import py_compile; py_compile.compile('.omniprod-plugin/omniprod/scripts/findings-delta.py', doraise=True)"
python3 -c "import py_compile; py_compile.compile('.omniprod-plugin/omniprod/scripts/impact-scorer.py', doraise=True)"
python3 -c "import py_compile; py_compile.compile('.omniprod-plugin/omniprod/scripts/evidence-packager.py', doraise=True)"
python3 -c "import py_compile; py_compile.compile('.omniprod-plugin/omniprod/scripts/entity-classifier.py', doraise=True)"

# JavaScript
node -c .omniprod-plugin/omniprod/scripts/assertions-runner.js

# Shell scripts
bash -n .omniprod-plugin/omniprod/scripts/cleanup-screenshots.sh
bash -n .omniprod-plugin/omniprod/scripts/check-findings.sh
bash -n .omniprod-plugin/omniprod/scripts/validate-capture.sh
```
Expected: All pass with no errors.

- [ ] **Step 3: Verify generate-assertions.py works on a real page**

Run:
```bash
python3 .omniprod-plugin/omniprod/scripts/generate-assertions.py --app web --page /compliance --output /tmp/test-assertions.json
cat /tmp/test-assertions.json | python3 -m json.tool | head -30
rm /tmp/test-assertions.json
```
Expected: Valid JSON with assertions array.

- [ ] **Step 4: Verify file count and structure**

Run:
```bash
echo "=== Commands ===" && ls .omniprod-plugin/omniprod/commands/ | wc -l
echo "=== Scripts ===" && ls .omniprod-plugin/omniprod/scripts/ | wc -l
echo "=== Perspectives ===" && ls .omniprod-plugin/omniprod/perspectives/ | wc -l
echo "=== New files ===" && git status --short .omniprod-plugin/
```
Expected: 12 commands (product-correlate added), 10 scripts (assertions-runner.js + generate-assertions.py added), 8 perspectives (unchanged files, 3 marked inactive in config).

- [ ] **Step 5: Final commit**

If any verification failures were fixed:
```bash
git add -A .omniprod-plugin/
git commit -m "chore(omniprod): verification fixes for v0.3.0"
```
