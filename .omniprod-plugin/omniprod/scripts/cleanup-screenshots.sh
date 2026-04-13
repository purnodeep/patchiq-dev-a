#!/usr/bin/env bash
# OmniProd screenshot manager — archives old screenshots, keeps workspace clean
# Usage: cleanup-screenshots.sh [--archive] [--prune-days <N>]
set -euo pipefail

OMNIPROD_DIR="${OMNIPROD_DIR:-.omniprod}"
SCREENSHOTS_DIR="$OMNIPROD_DIR/screenshots"
ARCHIVE_DIR="$SCREENSHOTS_DIR/archive"
CURRENT_DIR="$SCREENSHOTS_DIR/current"

archive=false
prune_days=7

while [[ $# -gt 0 ]]; do
  case "$1" in
    --archive) archive=true; shift ;;
    --prune-days) prune_days="$2"; shift 2 ;;
    *) shift ;;
  esac
done

mkdir -p "$CURRENT_DIR" "$ARCHIVE_DIR"

if $archive; then
  # Archive current screenshots with timestamp
  timestamp=$(date +%Y%m%d-%H%M%S)
  if [ "$(ls -A "$CURRENT_DIR" 2>/dev/null)" ]; then
    archive_target="$ARCHIVE_DIR/$timestamp"
    mkdir -p "$archive_target"
    mv "$CURRENT_DIR"/* "$archive_target/" 2>/dev/null || true
    echo "Archived $(ls "$archive_target" | wc -l) screenshots to $archive_target/"
  else
    echo "No screenshots to archive."
  fi
fi

# Prune old archives
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

# Report current state
current_count=$(ls "$CURRENT_DIR" 2>/dev/null | wc -l)
archive_count=$(find "$ARCHIVE_DIR" -maxdepth 1 -mindepth 1 -type d 2>/dev/null | wc -l)
total_size=$(du -sh "$SCREENSHOTS_DIR" 2>/dev/null | cut -f1)

echo "Screenshots: $current_count current, $archive_count archived, $total_size total"
