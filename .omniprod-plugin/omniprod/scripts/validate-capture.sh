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
