#!/usr/bin/env bash
# OmniProd findings checker — used by hooks and commands
# Usage: check-findings.sh [--critical-only] [--json] [--page <slug>]
set -euo pipefail

OMNIPROD_DIR="${OMNIPROD_DIR:-.omniprod}"
FINDINGS_DIR="$OMNIPROD_DIR/findings"

critical_only=false
json_output=false
page_filter=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --critical-only) critical_only=true; shift ;;
    --json) json_output=true; shift ;;
    --page) page_filter="$2"; shift 2 ;;
    *) shift ;;
  esac
done

if [ ! -d "$FINDINGS_DIR" ]; then
  if $json_output; then
    echo '{"total":0,"critical":0,"major":0,"minor":0,"nitpick":0,"pages":0}'
  fi
  exit 0
fi

total_critical=0
total_major=0
total_minor=0
total_nitpick=0
pages=0

for f in "$FINDINGS_DIR"/*.json; do
  [ -f "$f" ] || continue

  if [ -n "$page_filter" ]; then
    basename "$f" | grep -q "$page_filter" || continue
  fi

  pages=$((pages + 1))

  # Count open findings by severity
  c=$(python3 -c "
import json, sys
with open('$f') as fh:
    data = json.load(fh)
findings = [f for f in data.get('findings', []) if f.get('status') == 'open']
counts = {'critical': 0, 'major': 0, 'minor': 0, 'nitpick': 0}
for f in findings:
    s = f.get('severity', 'minor')
    counts[s] = counts.get(s, 0) + 1
print(counts['critical'], counts['major'], counts['minor'], counts['nitpick'])
" 2>/dev/null || echo "0 0 0 0")

  read cr ma mi ni <<< "$c"
  total_critical=$((total_critical + cr))
  total_major=$((total_major + ma))
  total_minor=$((total_minor + mi))
  total_nitpick=$((total_nitpick + ni))
done

total=$((total_critical + total_major + total_minor + total_nitpick))

if $json_output; then
  echo "{\"total\":$total,\"critical\":$total_critical,\"major\":$total_major,\"minor\":$total_minor,\"nitpick\":$total_nitpick,\"pages\":$pages}"
else
  if [ $total -eq 0 ]; then
    echo "No open findings."
  else
    if $critical_only; then
      if [ $total_critical -eq 0 ]; then
        exit 0
      else
        echo "OmniProd: $total_critical critical findings across $pages page(s)"
        exit 1
      fi
    fi
    echo "OmniProd: $total open findings across $pages page(s)"
    [ $total_critical -gt 0 ] && echo "  ❌ $total_critical critical"
    [ $total_major -gt 0 ] && echo "  ⚠️  $total_major major"
    [ $total_minor -gt 0 ] && echo "  📝 $total_minor minor"
    [ $total_nitpick -gt 0 ] && echo "  💭 $total_nitpick nitpick"
  fi
fi
