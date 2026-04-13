#!/usr/bin/env python3
"""Package captured evidence into structured JSON for perspective reviewer agents.

Scans a captures directory for screenshots, a11y snapshots, console logs, network
requests, performance reports, and exploration state, then assembles a single
evidence package that reviewer agents can consume.

Usage:
    evidence-packager.py --captures-dir <dir> [--output <file.json>]
"""
import argparse
import glob
import json
import os
import re
import sys
from datetime import date
from pathlib import Path


# ---------------------------------------------------------------------------
# File discovery
# ---------------------------------------------------------------------------

def _discover_files(captures_dir):
    """Discover all evidence files in the captures directory."""
    base = Path(captures_dir)
    files = {
        "screenshots": sorted(base.glob("*.png")),
        "snapshots": sorted(base.glob("*-snapshot.txt")),
        "console": base / "00-console.txt",
        "network": base / "00-network.txt",
        "capture_log_jsonl": base / "capture-log.jsonl",
        "capture_manifest": base / "capture-manifest.json",
        "lighthouse_report": base / "report.html",
        "exploration_state": base / "exploration" / "exploration-state.json",
        "narrative": base / "exploration" / "narrative.md",
    }
    return files


# ---------------------------------------------------------------------------
# Parsers
# ---------------------------------------------------------------------------

def _parse_capture_log(files):
    """Parse capture log (JSONL or JSON manifest) for screenshot metadata."""
    metadata = {}

    # Try JSONL first
    jsonl_path = files["capture_log_jsonl"]
    if jsonl_path.exists():
        try:
            with open(jsonl_path, "r", encoding="utf-8") as f:
                for line in f:
                    line = line.strip()
                    if not line:
                        continue
                    try:
                        entry = json.loads(line)
                        filename = entry.get("filename") or entry.get("file", "")
                        if filename:
                            metadata[os.path.basename(filename)] = entry
                    except json.JSONDecodeError:
                        continue
        except OSError:
            pass

    # Try JSON manifest
    manifest_path = files["capture_manifest"]
    if manifest_path.exists() and not metadata:
        try:
            with open(manifest_path, "r", encoding="utf-8") as f:
                data = json.load(f)
            captures = data if isinstance(data, list) else data.get("captures", [])
            for entry in captures:
                filename = entry.get("filename") or entry.get("file", "")
                if filename:
                    metadata[os.path.basename(filename)] = entry
        except (json.JSONDecodeError, OSError):
            pass

    return metadata


_NETWORK_LINE_RE = re.compile(
    r"^(?:\d+\.\s*)?"                           # optional leading "1. " numbering
    r"(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)\s+"
    r"(\S+)"
    r"\s+\[(\d+)\]"
    r"(?:\s+(\d+(?:\.\d+)?)ms)?"
)


def _parse_network(files):
    """Parse network request log."""
    path = files["network"]
    if not path.exists():
        return None

    endpoints = []
    errors = []

    try:
        with open(path, "r", encoding="utf-8") as f:
            for line in f:
                line = line.strip()
                if not line:
                    continue
                m = _NETWORK_LINE_RE.match(line)
                if m:
                    method = m.group(1)
                    url = m.group(2)
                    status = int(m.group(3))
                    latency = m.group(4)

                    entry = {
                        "endpoint": f"{method} {url}",
                        "status": status,
                    }

                    if latency:
                        entry["latency_ms"] = float(latency)

                    # Try to extract item count from response body hints
                    # Some network logs include item counts after the status
                    count_match = re.search(r"items?[=:]\s*(\d+)", line, re.IGNORECASE)
                    if count_match:
                        entry["item_count"] = int(count_match.group(1))

                    endpoints.append(entry)

                    if status >= 400:
                        errors.append({
                            "endpoint": f"{method} {url}",
                            "status": status,
                            "line": line,
                        })
    except OSError:
        return None

    return {"endpoints_called": endpoints, "errors": errors}


_CONSOLE_LINE_RE = re.compile(
    r"^(?:\d+\.\s*)?\[(\w+)\]\s*(.*)"
)


def _parse_console(files):
    """Parse console message log."""
    path = files["console"]
    if not path.exists():
        return None

    errors = []
    warnings = []

    try:
        with open(path, "r", encoding="utf-8") as f:
            for line in f:
                line = line.strip()
                if not line:
                    continue
                m = _CONSOLE_LINE_RE.match(line)
                if m:
                    level = m.group(1).lower()
                    message = m.group(2)
                    if level == "error":
                        errors.append(message)
                    elif level in ("warning", "warn", "issue"):
                        warnings.append(message)
                else:
                    # Lines without a level prefix — check for error/warning keywords
                    lower = line.lower()
                    if "error" in lower or "failed" in lower:
                        errors.append(line)
                    elif "warning" in lower or "warn" in lower:
                        warnings.append(line)
    except OSError:
        return None

    return {
        "errors": errors,
        "warnings": warnings,
        "error_count": len(errors),
        "warning_count": len(warnings),
    }


def _parse_narrative(files):
    """Read the narrative markdown if it exists."""
    path = files["narrative"]
    if not path.exists():
        return None
    try:
        with open(path, "r", encoding="utf-8") as f:
            return f.read().strip()
    except OSError:
        return None


def _parse_exploration_state(files):
    """Parse exploration state for state flow reconstruction."""
    path = files["exploration_state"]
    if not path.exists():
        return None
    try:
        with open(path, "r", encoding="utf-8") as f:
            data = json.load(f)
        # Extract state transitions if available
        transitions = data.get("transitions", data.get("state_flow", []))
        if isinstance(transitions, list):
            return transitions
        return None
    except (json.JSONDecodeError, OSError):
        return None


def _build_state_flow(capture_metadata):
    """Build state flow from capture log entries that have action annotations."""
    flow = []
    sorted_entries = sorted(
        capture_metadata.items(),
        key=lambda x: x[1].get("order", x[1].get("timestamp", x[0]))
    )

    prev_file = None
    for filename, entry in sorted_entries:
        action = entry.get("action") or entry.get("annotation", "")
        if prev_file and action:
            flow.append({
                "from": prev_file,
                "action": action,
                "to": filename,
            })
        prev_file = filename

    return flow


def _extract_entity_data(api_data):
    """Extract entity names and counts from API response data."""
    if not api_data:
        return None

    entity_data = {"counts": {}}
    names_by_type = {}

    for ep in api_data.get("endpoints_called", []):
        endpoint = ep.get("endpoint", "")
        # Try to infer entity type from URL path
        # e.g., GET /api/v1/compliance/frameworks -> frameworks
        path_match = re.search(r"/api/v\d+/(?:\w+/)*(\w+)(?:\?|$)", endpoint)
        if path_match:
            entity_type = path_match.group(1)
            if "item_count" in ep:
                entity_data["counts"][entity_type] = ep["item_count"]

        # Extract entity names from response data if present
        items = ep.get("items", ep.get("data", []))
        if isinstance(items, list):
            entity_names = []
            for item in items:
                if isinstance(item, dict):
                    name = (
                        item.get("name")
                        or item.get("title")
                        or item.get("label")
                        or item.get("id", "")
                    )
                    if name:
                        entity_names.append(str(name))
            if entity_names and path_match:
                names_by_type[path_match.group(1)] = entity_names

    entity_data.update(names_by_type)
    return entity_data if (entity_data["counts"] or names_by_type) else None


def _infer_page(captures_dir):
    """Try to infer the page slug from the captures directory name or contents."""
    dir_name = os.path.basename(os.path.normpath(captures_dir))
    if dir_name and dir_name != "current":
        return f"/{dir_name}"

    # Try parent directory
    parent = os.path.basename(os.path.dirname(os.path.normpath(captures_dir)))
    if parent and parent not in ("screenshots", ".omniprod"):
        return f"/{parent}"

    return "/unknown"


# ---------------------------------------------------------------------------
# Package builder
# ---------------------------------------------------------------------------

def build_package(captures_dir):
    """Build the evidence package."""
    files = _discover_files(captures_dir)
    capture_metadata = _parse_capture_log(files)

    # Screenshots
    screenshots = []
    for i, png in enumerate(files["screenshots"]):
        filename = png.name
        meta = capture_metadata.get(filename, {})
        screenshots.append({
            "filename": filename,
            "category": meta.get("category", meta.get("zone", _infer_category(filename))),
            "annotation": meta.get("annotation", meta.get("description", "")),
            "zone": meta.get("zone", "full-page"),
            "order": meta.get("order", i + 1),
        })

    # Snapshots
    snapshot_files = [p.name for p in files["snapshots"]]
    snapshots_info = {
        "files": snapshot_files,
        "count": len(snapshot_files),
    }

    # API / network data
    api_data = _parse_network(files)

    # Console
    console_data = _parse_console(files)

    # Performance
    perf = {}
    if files["lighthouse_report"].exists():
        perf["lighthouse_report"] = "report.html"
        # Try to extract scores from a companion JSON
        lh_json = files["lighthouse_report"].with_suffix(".json")
        if lh_json.exists():
            try:
                with open(lh_json, "r", encoding="utf-8") as f:
                    lh_data = json.load(f)
                cats = lh_data.get("categories", {})
                perf["lighthouse_scores"] = {
                    k: round(v.get("score", 0) * 100)
                    for k, v in cats.items()
                    if isinstance(v, dict) and "score" in v
                }
            except (json.JSONDecodeError, OSError):
                perf["lighthouse_scores"] = None
        else:
            perf["lighthouse_scores"] = None
    else:
        perf = None

    # Narrative
    narrative = _parse_narrative(files)

    # Entity data
    entity_data = _extract_entity_data(api_data)

    # State flow
    state_flow = _parse_exploration_state(files) or _build_state_flow(capture_metadata)

    # Build the package
    package = {
        "page": _infer_page(captures_dir),
        "capture_date": str(date.today()),
        "total_screenshots": len(screenshots),
        "screenshots": screenshots,
        "snapshots": snapshots_info,
        "api_data": api_data,
        "console": console_data,
        "performance": perf,
        "narrative": narrative,
        "entity_data": entity_data,
        "state_flow": state_flow or [],
    }

    return package


def _infer_category(filename):
    """Infer a screenshot category from its filename."""
    name = filename.lower().replace(".png", "").replace("-", " ").replace("_", " ")
    categories = [
        "page-load", "overview", "detail", "modal", "dialog", "dropdown",
        "hover", "focus", "error", "empty", "loading", "sidebar", "menu",
        "form", "table", "chart", "tab", "tooltip", "notification",
    ]
    for cat in categories:
        if cat.replace("-", " ") in name:
            return cat
    return "interaction"


# ---------------------------------------------------------------------------
# CLI
# ---------------------------------------------------------------------------

def main():
    parser = argparse.ArgumentParser(
        description="Package captured evidence for perspective reviewer agents"
    )
    parser.add_argument(
        "--captures-dir",
        required=True,
        help="Directory containing captured evidence (screenshots, snapshots, etc.)",
    )
    parser.add_argument(
        "--output", "-o",
        help="Output JSON file (default: stdout)",
        default=None,
    )
    args = parser.parse_args()

    if not os.path.isdir(args.captures_dir):
        print(f"error: captures directory not found: {args.captures_dir}", file=sys.stderr)
        sys.exit(1)

    package = build_package(args.captures_dir)

    output_json = json.dumps(package, indent=2, default=str)

    if args.output:
        with open(args.output, "w", encoding="utf-8") as f:
            f.write(output_json)
            f.write("\n")
        total = package["total_screenshots"]
        print(
            f"Packaged {total} screenshots + evidence -> {args.output}",
            file=sys.stderr,
        )
    else:
        print(output_json)


if __name__ == "__main__":
    main()
