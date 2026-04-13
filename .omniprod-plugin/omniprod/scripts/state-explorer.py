#!/usr/bin/env python3
"""State Explorer — deterministic exploration engine for product review screenshots.

Maintains exploration state on disk, returns batches of browser actions for the
orchestrator (an LLM agent) to execute. The orchestrator clicks, hovers, and
screenshots. This script is the brain.

Usage:
    state-explorer.py init --url "/compliance" --state-dir .omniprod/screenshots/current/exploration/
    state-explorer.py next --snapshot snapshot.txt --state-dir .omniprod/screenshots/current/exploration/ [--network network.txt]
    state-explorer.py status --state-dir .omniprod/screenshots/current/exploration/
    state-explorer.py narrative --state-dir .omniprod/screenshots/current/exploration/ --output narrative.md
"""
import argparse
import hashlib
import json
import os
import re
import sys
import textwrap
from collections import Counter, defaultdict
from pathlib import Path

# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------

MAX_BATCH_SIZE = 15
CRUD_KEYWORDS = re.compile(
    r"\b(create|new|add|edit|update|delete|remove)\b", re.IGNORECASE
)
ACTION_KEYWORDS = re.compile(
    r"\b(evaluate|export|deploy|import|sync|refresh|run|execute|download|upload|"
    r"approve|reject|assign|reset|enable|disable|activate|deactivate|archive|"
    r"restore|retry|cancel|generate|publish|revoke|renew|schedule|send|test|"
    r"install|uninstall|enroll|scan|validate)\b",
    re.IGNORECASE,
)
DETAIL_KEYWORDS = re.compile(
    r"\b(view\s+details?|view|open|go\s+to|see|details|inspect)\b", re.IGNORECASE
)
RESPONSIVE_WIDTHS = [(1440, "1440"), (1024, "1024"), (768, "768")]

# ---------------------------------------------------------------------------
# Snapshot hashing
# ---------------------------------------------------------------------------


def hash_snapshot(text: str) -> str:
    """Hash the STRUCTURE of a snapshot, not dynamic content.

    Two snapshots with the same structure (same elements, same roles, same
    nesting) but different timestamps/numbers produce the same hash.
    """
    normalized = text
    # Time-ago strings
    normalized = re.sub(r"\b\d{1,3}[hmd]\s*ago\b", "TIME_AGO", normalized)
    normalized = re.sub(
        r"\b(just now|a moment ago|seconds? ago|minutes? ago|hours? ago|days? ago)\b",
        "TIME_AGO",
        normalized,
        flags=re.IGNORECASE,
    )
    # UUIDs
    normalized = re.sub(
        r"\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b",
        "UUID",
        normalized,
    )
    # ISO datetimes
    normalized = re.sub(
        r"\b\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}(:\d{2})?\b", "DATETIME", normalized
    )
    # Percentages
    normalized = re.sub(r"\b\d+\.?\d*%\b", "PCT", normalized)
    # Large numbers (2+ digits) — keeps single-digit structural markers
    normalized = re.sub(r"\b\d{2,}\b", "N", normalized)
    # Collapse whitespace
    normalized = re.sub(r"[ \t]+", " ", normalized)
    return hashlib.sha256(normalized.encode("utf-8")).hexdigest()[:16]


# ---------------------------------------------------------------------------
# Element parsing
# ---------------------------------------------------------------------------


def _extract_name(line: str) -> str:
    """Pull a human-readable name from a snapshot line."""
    m = re.search(r"['\"]([^'\"]+)['\"]", line)
    if m:
        return m.group(1).strip()[:60]
    m = re.search(
        r"(?:button|link|tab|heading|combobox|textbox|checkbox|switch)\s+(.+?)(?:\[|$)",
        line,
        re.IGNORECASE,
    )
    if m:
        return m.group(1).strip()[:60]
    return line.strip()[:60]


def _extract_uid(line: str):
    """Extract the ref/uid from a snapshot line.  Returns None when absent."""
    # Chrome DevTools format: "uid=1_5 link ..." at start of (trimmed) line
    m = re.match(r"\s*uid=(\S+)", line)
    if m:
        return m.group(1)
    # Bracket formats: [ref12], [uid=xxx], [xxx]
    m = re.search(r"\[ref(\w+)\]", line)
    if m:
        return "ref" + m.group(1)
    m = re.search(r"\[uid[=:]([^\]]+)\]", line, re.IGNORECASE)
    if m:
        return m.group(1).strip()
    m = re.search(r"\[(\w+)\]\s*$", line)
    if m:
        return m.group(1)
    return None


def _indent_level(line: str) -> int:
    """Number of leading spaces (proxy for nesting depth)."""
    return len(line) - len(line.lstrip())


def _infer_zone(depth: int, ancestors: list) -> str:
    """Best-effort zone inference from nesting context."""
    joined = " ".join(ancestors).lower()
    if "navigation" in joined or "sidebar" in joined or "nav" in joined:
        return "navigation"
    if "dialog" in joined or "modal" in joined or "alertdialog" in joined:
        return "modal"
    if "table" in joined:
        return "table"
    if "toolbar" in joined or "menubar" in joined:
        return "toolbar"
    if "header" in joined or "banner" in joined:
        return "header"
    if "footer" in joined or "contentinfo" in joined:
        return "footer"
    if depth <= 4:
        return "page-header"
    return "main-content"


def parse_elements(text: str):
    """Parse interactive elements and structural context from a11y snapshot text.

    Returns (elements, tables, zones) where:
    - elements: list of dicts with role, name, uid, zone, depth
    - tables: list of dicts describing each table
    - zones: set of zone names found
    """
    elements = []
    tables = []
    zones = set()
    ancestor_stack = []  # (depth, label) pairs for zone inference
    current_table = None

    for raw_line in text.split("\n"):
        if not raw_line.strip():
            continue

        depth = _indent_level(raw_line)
        line = raw_line.strip()
        # Strip common prefixes: "- " for markdown-style or "uid=X_Y " for Chrome DevTools
        line_stripped = line.lstrip("- ").strip()
        # For Chrome DevTools format: "uid=1_5 link ..." → extract role after uid
        role_after_uid = re.match(r"uid=\S+\s+(.+)", line_stripped)
        if role_after_uid:
            line_stripped_for_role = role_after_uid.group(1)
        else:
            line_stripped_for_role = line_stripped

        # Maintain ancestor stack
        while ancestor_stack and ancestor_stack[-1][0] >= depth:
            ancestor_stack.pop()
        ancestor_stack.append((depth, line_stripped_for_role[:40]))

        # Detect table containers
        if re.match(r"table\b", line_stripped_for_role, re.IGNORECASE):
            tbl_name = _extract_name(line_stripped_for_role)
            current_table = {
                "name": tbl_name,
                "depth": depth,
                "rows": [],
            }
            tables.append(current_table)
            continue

        # Close table if we outdent past it
        if current_table and depth <= current_table["depth"]:
            current_table = None

        # Detect rows inside a table
        if current_table and re.match(r"row\b", line_stripped_for_role, re.IGNORECASE):
            uid = _extract_uid(line)
            current_table["rows"].append({
                "uid": uid,
                "depth": depth,
                "cells": [],
                "raw": line_stripped,
            })
            continue

        # Detect cells inside the most recent row
        if (
            current_table
            and current_table["rows"]
            and re.match(r"cell\b", line_stripped_for_role, re.IGNORECASE)
        ):
            cell_text = _extract_name(line_stripped_for_role)
            current_table["rows"][-1]["cells"].append(cell_text)
            continue

        # Interactive elements — match on role text, not raw line with uid prefix
        uid = _extract_uid(line)
        role = None
        name = _extract_name(line_stripped)

        rl = line_stripped_for_role.lower()
        if re.match(r"button\b", rl):
            role = "button"
        elif re.match(r"link\b", rl):
            role = "link"
        elif re.match(r"tab\b", rl) and "table" not in rl:
            role = "tab"
        elif re.match(r"(combobox|listbox|select|dropdown)\b", rl):
            role = "dropdown"
        elif re.match(r"(textbox|input|textarea|searchbox)\b", rl):
            role = "input"
        elif re.match(r"(checkbox|switch|toggle)\b", rl):
            role = "checkbox"
        elif re.match(r"(menuitem|option|treeitem)\b", rl):
            role = "menuitem"
        elif re.match(r"(article|card)\b", rl):
            role = "card"

        if role and uid:
            ancestor_labels = [a[1] for a in ancestor_stack[:-1]]
            zone = _infer_zone(depth, ancestor_labels)
            zones.add(zone)
            elements.append({
                "role": role,
                "name": name,
                "uid": uid,
                "zone": zone,
                "depth": depth,
                "raw": line_stripped,
            })

    return elements, tables, zones


# ---------------------------------------------------------------------------
# Entity classification (table row sampling)
# ---------------------------------------------------------------------------


def classify_table_rows(table: dict):
    """Classify rows by visible variant dimensions and pick one sample per class.

    Variant dimensions are columns with low cardinality relative to row count.
    """
    rows = table.get("rows", [])
    if not rows or not rows[0].get("cells"):
        return None

    n_rows = len(rows)
    n_cols = max(len(r["cells"]) for r in rows) if rows else 0

    if n_cols == 0 or n_rows <= 2:
        # Too few rows to bother classifying
        return None

    # Count distinct values per column
    col_values = defaultdict(list)
    for r in rows:
        for ci, cell in enumerate(r["cells"]):
            col_values[ci].append(cell)

    # Variant columns: cardinality <= min(5, n_rows/2)
    variant_cols = []
    for ci in range(n_cols):
        unique = len(set(col_values[ci]))
        if 1 < unique <= min(5, max(2, n_rows // 2)):
            variant_cols.append(ci)

    if not variant_cols:
        # Fall back: just pick first and last row
        variant_cols = [0]

    # Build class key per row
    classes = defaultdict(list)
    for r in rows:
        key_parts = []
        for ci in variant_cols:
            val = r["cells"][ci] if ci < len(r["cells"]) else ""
            key_parts.append(val)
        key = "|".join(key_parts)
        classes[key].append(r)

    # Pick one representative per class
    result_classes = []
    for label, members in classes.items():
        sample = members[0]
        sample_name = sample["cells"][0] if sample["cells"] else "row"
        result_classes.append({
            "label": label,
            "sample_uid": sample["uid"],
            "sample_name": sample_name,
            "count": len(members),
        })

    return {
        "table_name": table["name"],
        "total_rows": n_rows,
        "classes": result_classes,
        "samples_chosen": len(result_classes),
        "reduction": f"{n_rows} -> {len(result_classes)}",
    }


# ---------------------------------------------------------------------------
# Action generation
# ---------------------------------------------------------------------------


def _sanitize(name: str) -> str:
    return re.sub(r"[^a-z0-9]+", "-", name.lower().strip())[:30].strip("-")


def _is_icon_button(name: str) -> bool:
    return len(name) <= 2 or name in {"x", "X", "\u00d7", "\u22ee", "\u2026", "\u2022", "\u22ef", ""}


def _make_action(
    seq: list,
    action_type: str,
    target: str,
    uid: str | None,
    capture: str,
    annotation: str,
    zone: str,
    restore: str | None = None,
) -> dict:
    """Append an action dict and return it."""
    aid = f"SE-{len(seq) + 1:03d}"
    a = {
        "id": aid,
        "type": action_type,
        "target": target,
        "uid": uid,
        "capture": capture,
        "annotation": annotation,
        "zone": zone,
        "restore": restore,
    }
    seq.append(a)
    return a


def _build_journey(
    journey_id: str,
    name: str,
    trigger_target: str,
    trigger_uid: str | None,
) -> dict:
    """Build a CRUD journey — open form, screenshot, close without submitting."""
    slug = _sanitize(name)
    return {
        "id": journey_id,
        "name": name,
        "trigger": trigger_target,
        "max_steps": 8,
        "steps": [
            {
                "action": "click",
                "target": trigger_target,
                "uid": trigger_uid,
                "capture": f"journey-{slug}-0-trigger.png",
                "annotation": f"Opening {name} form/dialog",
            },
            {
                "action": "wait_and_screenshot",
                "capture": f"journey-{slug}-1-opened.png",
                "annotation": f"Dialog/form after opening ({name})",
            },
            {
                "action": "screenshot_only",
                "capture": f"journey-{slug}-2-fields.png",
                "annotation": "Form fields visible (do not submit)",
            },
            {
                "action": "press_key",
                "key": "Escape",
                "capture": None,
                "annotation": "Close form without submitting",
            },
        ],
    }


def generate_batch(
    elements: list,
    tables: list,
    state: dict,
    network_text: str | None = None,
):
    """Generate a prioritized batch of actions and journeys.

    Returns (actions, journeys, entity_classifications, frontier_additions).
    """
    actions = []
    journeys = []
    entity_classifications = {}
    frontier_add = []

    visited_uids = set()
    for entry in state.get("capture_log", []):
        if entry.get("uid"):
            visited_uids.add(entry["uid"])

    journey_names = {j["name"] for j in state.get("journeys_planned", [])}
    journey_names |= {j["name"] for j in state.get("journeys_completed", [])}
    next_journey_num = len(state.get("journeys_planned", [])) + len(
        state.get("journeys_completed", [])
    ) + 1

    def _budget_left():
        return len(actions) < MAX_BATCH_SIZE

    def _already_visited(uid):
        return uid in visited_uids

    # Categorize elements — check both buttons AND links for CRUD/action keywords
    tabs = [e for e in elements if e["role"] == "tab"]
    crud_buttons = [
        e for e in elements
        if e["role"] in ("button", "link") and CRUD_KEYWORDS.search(e["name"])
    ]
    action_buttons = [
        e
        for e in elements
        if e["role"] in ("button", "link")
        and ACTION_KEYWORDS.search(e["name"])
        and not CRUD_KEYWORDS.search(e["name"])
    ]
    detail_links = [
        e for e in elements
        if e["role"] == "link"
        and DETAIL_KEYWORDS.search(e["name"])
        and not CRUD_KEYWORDS.search(e["name"])
        and not ACTION_KEYWORDS.search(e["name"])
    ]
    dropdowns = [e for e in elements if e["role"] == "dropdown"]
    regular_buttons = [
        e
        for e in elements
        if e["role"] == "button"
        and not CRUD_KEYWORDS.search(e["name"])
        and not ACTION_KEYWORDS.search(e["name"])
        and not _is_icon_button(e["name"])
    ]
    icon_buttons = [
        e
        for e in elements
        if e["role"] == "button" and _is_icon_button(e["name"])
    ]

    # ----- Priority 1: Tabs -----
    for el in tabs:
        if not _budget_left():
            break
        if _already_visited(el["uid"]):
            continue
        _make_action(
            actions,
            "click",
            f"tab '{el['name']}'",
            el["uid"],
            f"tab-{_sanitize(el['name'])}.png",
            f"Switch to '{el['name']}' tab",
            el["zone"],
        )

    # ----- Priority 2: CRUD buttons/links -> journeys -----
    for el in crud_buttons:
        if not _budget_left():
            break
        if _already_visited(el["uid"]):
            continue
        jname = el["name"]
        target_label = f"{el['role']} '{el['name']}'"
        if jname not in journey_names:
            jid = f"JOURNEY-{next_journey_num:03d}"
            next_journey_num += 1
            journey = _build_journey(jid, jname, target_label, el["uid"])
            journeys.append(journey)
            journey_names.add(jname)
        _make_action(
            actions,
            "click",
            target_label,
            el["uid"],
            f"crud-{_sanitize(el['name'])}.png",
            f"CRUD action: {el['name']}",
            el["zone"],
            restore="press Escape to close",
        )

    # ----- Priority 3: Action buttons -----
    for el in action_buttons:
        if not _budget_left():
            break
        if _already_visited(el["uid"]):
            continue
        _make_action(
            actions,
            "click",
            f"button '{el['name']}'",
            el["uid"],
            f"action-{_sanitize(el['name'])}.png",
            f"Action button: {el['name']}",
            el["zone"],
        )

    # ----- Priority 4: Detail links -----
    for el in detail_links:
        if not _budget_left():
            break
        if _already_visited(el["uid"]):
            continue
        _make_action(
            actions,
            "click",
            f"link '{el['name']}'",
            el["uid"],
            f"detail-{_sanitize(el['name'])}.png",
            f"Navigate to detail: {el['name']}",
            el["zone"],
            restore="navigate back",
        )

    # ----- Priority 5: Dropdowns -----
    for el in dropdowns:
        if not _budget_left():
            break
        if _already_visited(el["uid"]):
            continue
        _make_action(
            actions,
            "click",
            f"dropdown '{el['name']}'",
            el["uid"],
            f"dropdown-{_sanitize(el['name'])}.png",
            f"Open dropdown: {el['name']}",
            el["zone"],
            restore="press Escape to close",
        )

    # ----- Priority 6: Hover states for regular buttons -----
    for el in regular_buttons:
        if not _budget_left():
            break
        if _already_visited(el["uid"]):
            continue
        _make_action(
            actions,
            "hover",
            f"button '{el['name']}'",
            el["uid"],
            f"hover-btn-{_sanitize(el['name'])}.png",
            f"Hover state for button '{el['name']}'",
            el["zone"],
            restore="move mouse away",
        )

    # ----- Priority 7: Hover states for icon buttons (tooltip check) -----
    for el in icon_buttons:
        if not _budget_left():
            break
        if _already_visited(el["uid"]):
            continue
        _make_action(
            actions,
            "hover",
            f"icon-button '{el['name']}'",
            el["uid"],
            f"tooltip-{_sanitize(el['name']) or el['uid']}.png",
            f"Hover icon button for tooltip: '{el['name']}'",
            el["zone"],
            restore="move mouse away",
        )

    # ----- Priority 8: Table row hovers (sampled) -----
    for tbl in tables:
        classification = classify_table_rows(tbl)
        if classification:
            entity_classifications[f"table '{tbl['name']}'"] = classification
            for cls in classification["classes"]:
                if not _budget_left():
                    break
                uid = cls["sample_uid"]
                if uid and not _already_visited(uid):
                    _make_action(
                        actions,
                        "hover",
                        f"table-row '{cls['sample_name']}' (class: {cls['label']})",
                        uid,
                        f"row-hover-{_sanitize(cls['sample_name'])}.png",
                        f"Sample row hover in '{tbl['name']}' — represents {cls['count']} rows",
                        "table",
                        restore="move mouse away",
                    )
        else:
            # Unclassified table — sample first 2 rows
            for i, row in enumerate(tbl.get("rows", [])[:2]):
                if not _budget_left():
                    break
                uid = row.get("uid")
                if uid and not _already_visited(uid):
                    rname = row["cells"][0] if row.get("cells") else f"row-{i}"
                    _make_action(
                        actions,
                        "hover",
                        f"table-row '{rname}'",
                        uid,
                        f"row-hover-{_sanitize(rname)}.png",
                        f"Row hover in '{tbl['name']}'",
                        "table",
                        restore="move mouse away",
                    )

    # ----- Priority 9: Focus sweep -----
    if _budget_left():
        focus_already = any(
            e.get("type") == "focus_sweep" for e in state.get("capture_log", [])
        )
        if not focus_already:
            _make_action(
                actions,
                "focus_sweep",
                "entire page",
                None,
                "focus-sweep.png",
                "Tab through interactive elements, screenshot every 3-4 stops",
                "page",
            )

    # ----- Priority 10: Scroll -----
    if _budget_left():
        scroll_already = any(
            e.get("type") == "scroll" for e in state.get("capture_log", [])
        )
        if not scroll_already:
            _make_action(
                actions,
                "scroll",
                "page content",
                None,
                "scroll-bottom.png",
                "Scroll to bottom, capture below-fold content",
                "page",
            )

    # ----- Priority 11: Responsive -----
    if _budget_left():
        captured_widths = {
            e.get("responsive_width")
            for e in state.get("capture_log", [])
            if e.get("type") == "responsive"
        }
        for width, label in RESPONSIVE_WIDTHS:
            if not _budget_left():
                break
            if width not in captured_widths:
                _make_action(
                    actions,
                    "responsive",
                    f"viewport {width}px",
                    None,
                    f"responsive-{label}.png",
                    f"Resize to {width}px, check layout integrity",
                    "page",
                    restore="resize back to original width",
                )

    return actions, journeys, entity_classifications, frontier_add


# ---------------------------------------------------------------------------
# State management
# ---------------------------------------------------------------------------


def load_state(state_dir: str) -> dict:
    path = os.path.join(state_dir, "exploration-state.json")
    if not os.path.exists(path):
        raise FileNotFoundError(f"State file not found: {path}")
    with open(path) as f:
        return json.load(f)


def save_state(state_dir: str, state: dict):
    path = os.path.join(state_dir, "exploration-state.json")
    with open(path, "w") as f:
        json.dump(state, f, indent=2)


# ---------------------------------------------------------------------------
# Commands
# ---------------------------------------------------------------------------


def cmd_init(args):
    state_dir = args.state_dir
    os.makedirs(state_dir, exist_ok=True)

    state = {
        "url": args.url,
        "visited_hashes": {},
        "frontier": [],
        "entity_samples": {},
        "capture_log": [],
        "batch_count": 0,
        "journeys_planned": [],
        "journeys_completed": [],
    }
    save_state(state_dir, state)
    print(json.dumps({"status": "initialized", "url": args.url, "state_dir": state_dir}))


def cmd_next(args):
    state_dir = args.state_dir
    try:
        state = load_state(state_dir)
    except FileNotFoundError as exc:
        print(json.dumps({"error": str(exc)}), file=sys.stderr)
        sys.exit(1)

    # Read snapshot
    snapshot_path = args.snapshot
    if not os.path.exists(snapshot_path):
        print(json.dumps({"error": f"Snapshot file not found: {snapshot_path}"}), file=sys.stderr)
        sys.exit(1)

    with open(snapshot_path) as f:
        snapshot_text = f.read()

    if not snapshot_text.strip():
        print(json.dumps({"error": "Snapshot file is empty"}), file=sys.stderr)
        sys.exit(1)

    # Optional network log
    network_text = None
    if args.network and os.path.exists(args.network):
        with open(args.network) as f:
            network_text = f.read()

    # Compute structural hash
    snap_hash = hash_snapshot(snapshot_text)

    # Duplicate check
    if snap_hash in state["visited_hashes"]:
        dup_id = state["visited_hashes"][snap_hash]
        state["batch_count"] += 1
        save_state(state_dir, state)
        result = {
            "batch_id": state["batch_count"],
            "status": "duplicate",
            "duplicate_of": dup_id,
            "total_explored": len(state["visited_hashes"]),
            "remaining_frontier": len(state["frontier"]),
            "actions": [],
            "journeys": [],
        }
        print(json.dumps(result, indent=2))
        return

    # Register new state
    state["batch_count"] += 1
    batch_id = state["batch_count"]
    state_label = f"SC-{batch_id:03d}"
    state["visited_hashes"][snap_hash] = state_label

    # Parse elements
    elements, tables, zones = parse_elements(snapshot_text)

    # Generate action batch
    actions, journeys, entity_classifications, frontier_add = generate_batch(
        elements, tables, state, network_text
    )

    # Record journeys
    for j in journeys:
        state["journeys_planned"].append(j)

    # Update capture log
    for a in actions:
        log_entry = {
            "batch_id": batch_id,
            "state_label": state_label,
            "action_id": a["id"],
            "type": a["type"],
            "target": a["target"],
            "uid": a.get("uid"),
            "capture": a.get("capture"),
            "annotation": a.get("annotation", ""),
            "zone": a.get("zone", ""),
        }
        if a["type"] == "responsive":
            # Extract width from target for tracking
            m = re.search(r"(\d+)px", a["target"])
            if m:
                log_entry["responsive_width"] = int(m.group(1))
        state["capture_log"].append(log_entry)

    # Update entity samples
    for tbl_key, classification in entity_classifications.items():
        state["entity_samples"][tbl_key] = classification

    # Update frontier
    state["frontier"].extend(frontier_add)

    # Determine overall status
    has_remaining = bool(state["frontier"])
    has_pending_journeys = len(state["journeys_planned"]) > len(
        state["journeys_completed"]
    )
    has_actions = bool(actions)
    if not has_actions and not has_remaining and not has_pending_journeys:
        status = "complete"
    else:
        status = "exploring"

    save_state(state_dir, state)

    # Count duplicates
    duplicate_count = batch_id - len(state["visited_hashes"])
    if duplicate_count < 0:
        duplicate_count = 0

    result = {
        "batch_id": batch_id,
        "status": status,
        "total_explored": len(state["visited_hashes"]),
        "remaining_frontier": len(state["frontier"]),
        "skipped_duplicates": duplicate_count,
        "actions": actions,
        "journeys": journeys,
        "entity_classifications": entity_classifications,
    }

    print(json.dumps(result, indent=2))


def cmd_status(args):
    try:
        state = load_state(args.state_dir)
    except FileNotFoundError as exc:
        print(json.dumps({"error": str(exc)}), file=sys.stderr)
        sys.exit(1)

    total_explored = len(state.get("visited_hashes", {}))
    remaining = len(state.get("frontier", []))
    captures = len(state.get("capture_log", []))
    journeys_done = len(state.get("journeys_completed", []))
    journeys_planned = len(state.get("journeys_planned", []))
    journeys_remaining = journeys_planned - journeys_done

    # Estimate coverage: explored / (explored + remaining + pending journeys)
    total_estimated = total_explored + remaining + max(0, journeys_remaining)
    coverage_pct = (
        round(100 * total_explored / total_estimated) if total_estimated > 0 else 100
    )

    # Duplicates = batch_count - unique hashes
    batch_count = state.get("batch_count", 0)
    duplicates = max(0, batch_count - total_explored)

    is_complete = remaining == 0 and journeys_remaining <= 0
    status_str = "complete" if is_complete else "exploring"

    result = {
        "status": status_str,
        "explored_states": total_explored,
        "remaining_frontier": remaining,
        "captures_taken": captures,
        "duplicates_skipped": duplicates,
        "journeys_completed": journeys_done,
        "journeys_remaining": max(0, journeys_remaining),
        "coverage_pct": coverage_pct,
    }
    print(json.dumps(result, indent=2))


def cmd_narrative(args):
    try:
        state = load_state(args.state_dir)
    except FileNotFoundError as exc:
        print(json.dumps({"error": str(exc)}), file=sys.stderr)
        sys.exit(1)

    url = state.get("url", "/unknown")
    page_name = url.strip("/").replace("/", " ").title() or "Page"

    lines = []
    lines.append(f"# Capture Narrative: {page_name}")
    lines.append("")

    # --- State Flow ---
    lines.append("## State Flow")
    lines.append("")

    # Group captures by batch
    batches = defaultdict(list)
    for entry in state.get("capture_log", []):
        batches[entry.get("batch_id", 0)].append(entry)

    # Categorize entries for narrative sections
    page_loads = []
    hover_entries = []
    click_entries = []
    detail_entries = []
    focus_entries = []
    scroll_entries = []
    responsive_entries = []
    journey_entries = []

    for entry in state.get("capture_log", []):
        t = entry.get("type", "")
        if t == "hover":
            hover_entries.append(entry)
        elif t == "click":
            target = entry.get("target", "").lower()
            if "tab" in target:
                click_entries.append(entry)
            elif "detail" in target or "view" in target:
                detail_entries.append(entry)
            else:
                click_entries.append(entry)
        elif t == "focus_sweep":
            focus_entries.append(entry)
        elif t == "scroll":
            scroll_entries.append(entry)
        elif t == "responsive":
            responsive_entries.append(entry)

    section_num = 0

    # Page load section
    section_num += 1
    lines.append(f"### {section_num}. Page Load (default state)")
    lines.append(f"- **URL**: {url}")
    first_batch = batches.get(1, [])
    if first_batch:
        lines.append(f"- **State label**: {first_batch[0].get('state_label', 'SC-001')}")
        lines.append(f"- **Elements discovered**: {len(first_batch)} interactive elements processed")
    lines.append("")

    # Hover states
    if hover_entries:
        section_num += 1
        lines.append(f"### {section_num}. Hover States")
        for entry in hover_entries:
            cap = entry.get("capture", "")
            ann = entry.get("annotation", "")
            lines.append(f"- **{cap}**: {ann}")
        lines.append("")

    # Tab/click exploration
    if click_entries:
        section_num += 1
        lines.append(f"### {section_num}. Interactive Elements")
        for entry in click_entries:
            cap = entry.get("capture", "")
            ann = entry.get("annotation", "")
            lines.append(f"- **{cap}**: {ann}")
        lines.append("")

    # Detail navigation
    if detail_entries:
        section_num += 1
        lines.append(f"### {section_num}. Detail Navigation")
        for entry in detail_entries:
            cap = entry.get("capture", "")
            ann = entry.get("annotation", "")
            lines.append(f"- **{cap}**: {ann}")
        lines.append("")

    # Entity sampling decisions
    entity_samples = state.get("entity_samples", {})
    if entity_samples:
        section_num += 1
        lines.append(f"### {section_num}. Entity Sampling Decisions")
        for tbl_key, classification in entity_samples.items():
            total = classification.get("total_rows", 0)
            sampled = classification.get("samples_chosen", 0)
            reduction = classification.get("reduction", "")
            lines.append(f"- **{tbl_key}**: {reduction}")
            for cls in classification.get("classes", []):
                lines.append(
                    f"  - Class '{cls['label']}': {cls['count']} rows, "
                    f"sample = {cls['sample_name']}"
                )
        lines.append("")

    # Journeys
    all_journeys = state.get("journeys_planned", []) + state.get(
        "journeys_completed", []
    )
    # Deduplicate by ID
    seen_jids = set()
    unique_journeys = []
    for j in all_journeys:
        if j["id"] not in seen_jids:
            seen_jids.add(j["id"])
            unique_journeys.append(j)

    if unique_journeys:
        section_num += 1
        lines.append(f"### {section_num}. Journeys")
        for j in unique_journeys:
            completed = j["id"] in {
                jj["id"] for jj in state.get("journeys_completed", [])
            }
            status_tag = "completed" if completed else "planned"
            lines.append(f"- **{j['id']}**: {j['name']} ({status_tag})")
            for step in j.get("steps", []):
                cap = step.get("capture", "(no capture)")
                ann = step.get("annotation", "")
                lines.append(f"  - {cap}: {ann}")
        lines.append("")

    # Focus sweep
    if focus_entries:
        section_num += 1
        lines.append(f"### {section_num}. Focus Sweep")
        for entry in focus_entries:
            lines.append(f"- {entry.get('annotation', 'Focus sweep')}")
        lines.append("")

    # Responsive
    if responsive_entries:
        section_num += 1
        lines.append(f"### {section_num}. Responsive Breakpoints")
        for entry in responsive_entries:
            cap = entry.get("capture", "")
            ann = entry.get("annotation", "")
            lines.append(f"- **{cap}**: {ann}")
        lines.append("")

    # --- Coverage Summary ---
    lines.append("## Coverage Summary")
    total_explored = len(state.get("visited_hashes", {}))
    duplicates = max(0, state.get("batch_count", 0) - total_explored)
    captures = len(state.get("capture_log", []))
    journeys_completed = len(state.get("journeys_completed", []))
    journeys_planned = len(state.get("journeys_planned", []))
    responsive_count = len(responsive_entries)

    lines.append(f"- {total_explored} unique states explored")
    lines.append(f"- {duplicates} skipped as duplicates")
    lines.append(f"- {captures} capture actions issued")
    if responsive_count:
        widths = [
            str(e.get("responsive_width", "?")) for e in responsive_entries
        ]
        lines.append(f"- {responsive_count} breakpoints captured ({', '.join(widths)})")
    lines.append(
        f"- {journeys_completed} of {journeys_planned} journeys executed"
    )
    lines.append("")

    narrative = "\n".join(lines)

    if args.output:
        out_path = args.output
        os.makedirs(os.path.dirname(out_path) or ".", exist_ok=True)
        with open(out_path, "w") as f:
            f.write(narrative)
        print(json.dumps({"status": "ok", "output": out_path, "lines": len(lines)}))
    else:
        print(narrative)


# ---------------------------------------------------------------------------
# CLI
# ---------------------------------------------------------------------------


def main():
    parser = argparse.ArgumentParser(
        description="State Explorer — deterministic exploration engine for product reviews",
    )
    subparsers = parser.add_subparsers(dest="command", required=True)

    # init
    p_init = subparsers.add_parser("init", help="Initialize exploration for a page")
    p_init.add_argument("--url", required=True, help="Page URL path (e.g. /compliance)")
    p_init.add_argument(
        "--state-dir",
        required=True,
        help="Directory for exploration state files",
    )

    # next
    p_next = subparsers.add_parser("next", help="Feed snapshot, get next action batch")
    p_next.add_argument(
        "--snapshot", required=True, help="Path to a11y snapshot text file"
    )
    p_next.add_argument(
        "--state-dir",
        required=True,
        help="Directory for exploration state files",
    )
    p_next.add_argument(
        "--network",
        default=None,
        help="Optional path to network log text file",
    )

    # status
    p_status = subparsers.add_parser("status", help="Check exploration status")
    p_status.add_argument(
        "--state-dir",
        required=True,
        help="Directory for exploration state files",
    )

    # narrative
    p_narrative = subparsers.add_parser(
        "narrative", help="Generate capture narrative markdown"
    )
    p_narrative.add_argument(
        "--state-dir",
        required=True,
        help="Directory for exploration state files",
    )
    p_narrative.add_argument(
        "--output",
        default=None,
        help="Output markdown file path (prints to stdout if omitted)",
    )

    args = parser.parse_args()

    dispatch = {
        "init": cmd_init,
        "next": cmd_next,
        "status": cmd_status,
        "narrative": cmd_narrative,
    }
    dispatch[args.command](args)


if __name__ == "__main__":
    main()
