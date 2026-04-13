#!/usr/bin/env python3
"""OmniProd findings delta — compare two review snapshots, track improvements.

Usage:
    findings-delta.py <current.json> <previous.json>
    findings-delta.py --page <slug>     # auto-find latest two reviews for page
"""
import argparse
import json
import sys
import glob
import os
from pathlib import Path


def load_findings(path):
    with open(path) as f:
        data = json.load(f)
    return {f["id"]: f for f in data.get("findings", []) if f.get("status") == "open"}


def find_reviews_for_page(slug, findings_dir=".omniprod/findings"):
    pattern = os.path.join(findings_dir, f"*-{slug}.json")
    files = sorted(glob.glob(pattern), reverse=True)
    return files


def compute_delta(current_findings, previous_findings):
    current_elements = {
        (f.get("element", ""), f.get("observation", "")[:80])
        for f in current_findings.values()
    }
    previous_elements = {
        (f.get("element", ""), f.get("observation", "")[:80])
        for f in previous_findings.values()
    }

    fixed = previous_elements - current_elements
    new = current_elements - previous_elements
    remaining = current_elements & previous_elements

    if len(fixed) > len(new):
        trend = "improving"
    elif len(new) > len(fixed):
        trend = "degrading"
    else:
        trend = "stable"

    return {
        "fixed": len(fixed),
        "new": len(new),
        "remaining": len(remaining),
        "trend": trend,
        "fixed_items": [list(x) for x in fixed],
        "new_items": [list(x) for x in new],
        "current_total": len(current_findings),
        "previous_total": len(previous_findings),
    }


def severity_counts(findings):
    counts = {"critical": 0, "major": 0, "minor": 0, "nitpick": 0}
    for f in findings.values():
        s = f.get("severity", "minor")
        counts[s] = counts.get(s, 0) + 1
    return counts


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


if __name__ == "__main__":
    main()
