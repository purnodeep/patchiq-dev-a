#!/usr/bin/env python3
"""Classify a11y snapshot entities into equivalence classes by visible properties.

Parses tables, card grids, and lists from a11y snapshot text, clusters rows by
variant columns (status, score bucket, action set), and picks one representative
per cluster. This dramatically reduces the number of rows a reviewer agent needs
to inspect.

Usage:
    entity-classifier.py <snapshot.txt> [--output <classes.json>]
"""
import argparse
import json
import re
import sys
from collections import defaultdict


# ---------------------------------------------------------------------------
# Numeric bucket helpers
# ---------------------------------------------------------------------------

_PERCENT_RE = re.compile(r"^(\d+(?:\.\d+)?)\s*%$")  # Require % suffix — plain integers are counts, not percentages


def _bucket(value_str):
    """Bucket a numeric string into high/medium/low/critical."""
    m = _PERCENT_RE.match(value_str.strip())
    if not m:
        return None
    v = float(m.group(1))
    if v >= 95:
        return "high"
    if v >= 80:
        return "medium"
    if v >= 50:
        return "low"
    return "critical"


# ---------------------------------------------------------------------------
# Snapshot line parser
# ---------------------------------------------------------------------------

_INDENT_RE = re.compile(r"^(\s*)")

# Matches both formats:
#   "  - table "Controls" [ref9]"  (markdown-style)
#   "  uid=1_5 table "Controls""   (Chrome DevTools)
_ELEMENT_RE = re.compile(
    r"^(\s*)"                           # leading whitespace (indent)
    r"(?:-\s+)?"                        # optional "- " prefix (markdown style)
    r"(?:uid=\S+\s+)?"                  # optional "uid=X_Y " prefix (Chrome DevTools)
    r"(\w+)"                            # element type (table, row, cell, button, ...)
    r'(?:\s+"([^"]*)")?'                # optional quoted label
    r"(?:\s+\[([^\]]+)\])?"             # optional ref like [ref9]
    r".*$"                              # remaining attrs (url=, etc.)
)

# Dedicated Chrome DevTools uid extractor
_UID_RE = re.compile(r"\buid=(\S+)")


class _Line:
    """Parsed representation of a single snapshot line."""
    __slots__ = ("depth", "element", "label", "ref", "raw")

    def __init__(self, depth, element, label, ref, raw):
        self.depth = depth
        self.element = element
        self.label = label or ""
        self.ref = ref or ""
        self.raw = raw


def _parse_lines(text):
    """Parse snapshot text into a list of _Line objects."""
    lines = []
    for raw in text.splitlines():
        if not raw.strip():
            continue
        m = _ELEMENT_RE.match(raw)
        if m:
            indent = len(m.group(1))
            # Normalise indent to a depth level (2 spaces per level is common)
            depth = indent // 2
            # Get ref from bracket format OR Chrome DevTools uid= format
            ref = m.group(4)
            if not ref:
                uid_m = _UID_RE.search(raw)
                if uid_m:
                    ref = uid_m.group(1)
            lines.append(_Line(depth, m.group(2), m.group(3), ref, raw))
        else:
            # Non-matching lines — keep as raw context but don't parse
            indent_m = _INDENT_RE.match(raw)
            depth = len(indent_m.group(1)) // 2 if indent_m else 0
            lines.append(_Line(depth, "__text__", raw.strip().lstrip("- "), None, raw))
    return lines


# ---------------------------------------------------------------------------
# Table extraction
# ---------------------------------------------------------------------------

def _children_of(lines, parent_idx):
    """Yield (index, line) for direct children of lines[parent_idx]."""
    parent_depth = lines[parent_idx].depth
    i = parent_idx + 1
    while i < len(lines):
        if lines[i].depth <= parent_depth:
            break
        if lines[i].depth == parent_depth + 1:
            yield i, lines[i]
        i += 1


def _descendants_of(lines, parent_idx):
    """Yield (index, line) for all descendants of lines[parent_idx]."""
    parent_depth = lines[parent_idx].depth
    i = parent_idx + 1
    while i < len(lines):
        if lines[i].depth <= parent_depth:
            break
        yield i, lines[i]
        i += 1


def _extract_cell_content(lines, cell_idx):
    """Return (text_value, list_of_buttons) for a cell.

    Chrome DevTools nests content inside ignored/generic wrappers,
    so we walk ALL descendants, not just direct children.
    """
    cell_line = lines[cell_idx]
    text_parts = []
    buttons = []
    if cell_line.label:
        text_parts.append(cell_line.label)
    for _, desc in _descendants_of(lines, cell_idx):
        if desc.element == "button":
            buttons.append(desc.label)
        elif desc.element == "StaticText" and desc.label:
            text_parts.append(desc.label)
        elif desc.element not in ("ignored", "generic", "InlineTextBox",
                                   "image", "__text__") and desc.label:
            text_parts.append(desc.label)
    # Deduplicate — StaticText often repeats InlineTextBox content
    seen = set()
    unique_parts = []
    for p in text_parts:
        if p not in seen:
            seen.add(p)
            unique_parts.append(p)
    return " ".join(unique_parts), buttons


def _extract_tables(lines):
    """Find all tables and return structured data."""
    tables = []
    for ti, tl in enumerate(lines):
        if tl.element != "table":
            continue

        table_name = tl.label or f"Table@line{ti}"
        rows_data = []

        # Collect header row if present
        header_row = None
        data_rows_start = []

        def _collect_rows_recursive(parent_idx, parent_line):
            """Collect row elements from a container, descending through
            rowgroup, ignored, and generic wrappers."""
            for ci, cl in _children_of(lines, parent_idx):
                if cl.element == "row":
                    # Check if this is a header row (contains columnheader children)
                    is_header = False
                    if parent_line and parent_line.label and "header" in parent_line.label.lower():
                        is_header = True
                    if is_header:
                        nonlocal header_row
                        if not header_row:
                            header_row = (ci, cl)
                    else:
                        data_rows_start.append((ci, cl))
                elif cl.element in ("rowgroup", "ignored", "generic", "__text__"):
                    # Chrome DevTools wraps rows in ignored/generic containers
                    _collect_rows_recursive(ci, cl)

        _collect_rows_recursive(ti, tl)

        # If no explicit header, check if first row looks like a header
        # (columnheader descendants — may be nested in ignored/generic)
        if not header_row and data_rows_start:
            first_ri, first_rl = data_rows_start[0]
            descendants = list(_descendants_of(lines, first_ri))
            if descendants and any(c.element == "columnheader" for _, c in descendants):
                header_row = (first_ri, first_rl)
                data_rows_start = data_rows_start[1:]

        # Extract column names from header (use descendants to handle wrappers)
        col_names = []
        if header_row:
            hri, _ = header_row
            for _, hc in _descendants_of(lines, hri):
                if hc.element in ("columnheader", "cell"):
                    col_names.append(hc.label or "")

        # Extract data rows (use descendants to handle ignored/generic wrappers)
        for ri, rl in data_rows_start:
            cells = []
            row_buttons = []
            row_ref = rl.ref or ""
            for ci, cl in _descendants_of(lines, ri):
                if cl.element in ("cell", "gridcell", "columnheader"):
                    text, btns = _extract_cell_content(lines, ci)
                    cells.append(text)
                    row_buttons.extend(btns)
                elif cl.element == "button":
                    row_buttons.append(cl.label)
            rows_data.append({
                "ref": row_ref,
                "cells": cells,
                "buttons": sorted(set(row_buttons)),
            })

        # Infer column names if no header found
        if not col_names and rows_data:
            col_names = [f"Col{i+1}" for i in range(len(rows_data[0]["cells"]))]

        tables.append({
            "name": table_name,
            "columns": col_names,
            "rows": rows_data,
        })

    return tables


# ---------------------------------------------------------------------------
# Card / article extraction
# ---------------------------------------------------------------------------

def _extract_cards(lines):
    """Find groups of article elements (card grids)."""
    # Find all article elements and group consecutive ones
    groups = []
    current_group = []
    current_parent = None

    for i, ln in enumerate(lines):
        if ln.element == "article":
            # Check parent context
            parent_depth = ln.depth - 1
            parent_name = None
            for j in range(i - 1, -1, -1):
                if lines[j].depth == parent_depth:
                    parent_name = lines[j].label or lines[j].element
                    break

            if parent_name != current_parent and current_group:
                groups.append((current_parent or "Cards", current_group))
                current_group = []

            current_parent = parent_name

            # Extract card content
            card_texts = []
            card_buttons = []
            card_ref = ln.ref or ""
            if ln.label:
                card_texts.append(ln.label)

            for _, desc in _descendants_of(lines, i):
                if desc.element == "button":
                    card_buttons.append(desc.label)
                elif desc.label and desc.element not in ("img", "image"):
                    card_texts.append(desc.label)

            current_group.append({
                "ref": card_ref,
                "texts": card_texts,
                "buttons": sorted(set(card_buttons)),
            })

    if current_group:
        groups.append((current_parent or "Cards", current_group))

    return groups


# ---------------------------------------------------------------------------
# List extraction
# ---------------------------------------------------------------------------

def _extract_lists(lines):
    """Find unordered/ordered lists and extract items."""
    results = []
    for i, ln in enumerate(lines):
        if ln.element not in ("list", "ul", "ol"):
            continue

        list_name = ln.label or f"List@line{i}"
        items = []
        for ci, cl in _children_of(lines, i):
            if cl.element in ("listitem", "li", "item"):
                item_texts = []
                item_buttons = []
                item_ref = cl.ref or ""
                if cl.label:
                    item_texts.append(cl.label)
                for _, desc in _descendants_of(lines, ci):
                    if desc.element == "button":
                        item_buttons.append(desc.label)
                    elif desc.label and desc.element not in ("img", "image"):
                        item_texts.append(desc.label)
                items.append({
                    "ref": item_ref,
                    "texts": item_texts,
                    "buttons": sorted(set(item_buttons)),
                })

        if items:
            results.append({"name": list_name, "items": items})

    return results


# ---------------------------------------------------------------------------
# Equivalence class computation
# ---------------------------------------------------------------------------

def _cardinality_ratio(values):
    """Return unique-count / total-count. Low ratio = variant column."""
    if not values:
        return 1.0
    return len(set(values)) / len(values)


def _classify_table(table):
    """Classify table rows into equivalence classes."""
    columns = table["columns"]
    rows = table["rows"]

    if not rows:
        return {
            "name": table["name"],
            "total_rows": 0,
            "columns": columns,
            "variant_columns": [],
            "identity_columns": columns,
            "equivalence_classes": [],
            "total_samples": 0,
            "reduction": "0 rows -> 0 samples",
        }

    num_cols = len(columns)

    # Analyse cardinality per column
    col_values = defaultdict(list)
    for row in rows:
        for ci in range(min(num_cols, len(row["cells"]))):
            col_values[ci].append(row["cells"][ci])

    variant_indices = []
    identity_indices = []
    # Threshold: if cardinality ratio <= 0.5, it's a variant column.
    # Also: if all values are numeric-ish percentages, treat as variant.
    # Also: if unique count <= 10 regardless of ratio, it's likely a variant
    # (handles small tables where ratio might be misleading).
    for ci in range(num_cols):
        vals = col_values[ci]
        ratio = _cardinality_ratio(vals)
        unique_count = len(set(vals))
        non_empty = [v.strip() for v in vals if v.strip()]
        all_numeric = bool(non_empty) and all(_PERCENT_RE.match(v) for v in non_empty)

        is_variant = (
            (ratio <= 0.5 and unique_count <= 10)
            or (all_numeric and len(vals) > 1)
        )
        if is_variant:
            variant_indices.append(ci)
        else:
            identity_indices.append(ci)

    # Actions (buttons) are always a variant dimension
    variant_col_names = [columns[ci] for ci in variant_indices if ci < len(columns)]
    identity_col_names = [columns[ci] for ci in identity_indices if ci < len(columns)]

    # Always include Actions as variant
    has_any_buttons = any(row["buttons"] for row in rows)
    if has_any_buttons:
        variant_col_names.append("Actions")

    # Build variant key per row
    clusters = defaultdict(list)
    for ri, row in enumerate(rows):
        key_parts = {}
        for ci in variant_indices:
            if ci >= len(row["cells"]):
                continue
            col_name = columns[ci] if ci < len(columns) else f"Col{ci+1}"
            val = row["cells"][ci].strip()
            # Bucket numeric values
            b = _bucket(val)
            if b is not None:
                key_parts[col_name] = b
            else:
                key_parts[col_name] = val if val else "(empty)"

        if has_any_buttons:
            key_parts["Actions"] = ",".join(row["buttons"]) if row["buttons"] else "(none)"

        key_tuple = tuple(sorted(key_parts.items()))
        clusters[key_tuple].append(ri)

    # Build equivalence classes
    eq_classes = []
    for key_tuple, row_indices in clusters.items():
        key_dict = dict(key_tuple)
        label = "-".join(str(v) for v in key_dict.values())
        sample_idx = row_indices[0]
        sample_row = rows[sample_idx]

        # Build sample name from identity columns
        name_parts = []
        for ci in identity_indices:
            if ci < len(sample_row["cells"]) and sample_row["cells"][ci].strip():
                name_parts.append(sample_row["cells"][ci].strip())
        sample_name = " ".join(name_parts) if name_parts else f"Row {sample_idx}"

        eq_classes.append({
            "label": label,
            "variant_key": key_dict,
            "sample_row_index": sample_idx,
            "sample_uid": sample_row["ref"],
            "sample_name": sample_name,
            "count": len(row_indices),
        })

    # Sort by count descending for readability
    eq_classes.sort(key=lambda c: -c["count"])

    return {
        "name": table["name"],
        "total_rows": len(rows),
        "columns": columns,
        "variant_columns": variant_col_names,
        "identity_columns": identity_col_names,
        "equivalence_classes": eq_classes,
        "total_samples": len(eq_classes),
        "reduction": f"{len(rows)} rows -> {len(eq_classes)} samples",
    }


def _classify_card_group(name, cards):
    """Classify a group of cards into equivalence classes."""
    if not cards:
        return {
            "name": name,
            "total_items": 0,
            "equivalence_classes": [],
            "total_samples": 0,
            "reduction": "0 items -> 0 samples",
        }

    # For cards, variant key is the button set + any bucketed numeric values
    clusters = defaultdict(list)
    for ci, card in enumerate(cards):
        key_parts = {}
        # Check for numeric values in texts
        for t in card["texts"]:
            b = _bucket(t)
            if b is not None:
                key_parts["score"] = b
                break

        # Check for status-like words
        status_words = {"PASS", "FAIL", "WARNING", "ERROR", "OK", "ACTIVE",
                        "INACTIVE", "PENDING", "COMPLIANT", "NON-COMPLIANT",
                        "CRITICAL", "HIGH", "MEDIUM", "LOW", "INFO"}
        for t in card["texts"]:
            if t.upper() in status_words:
                key_parts["status"] = t.upper()
                break

        key_parts["actions"] = ",".join(card["buttons"]) if card["buttons"] else "(none)"
        key_tuple = tuple(sorted(key_parts.items()))
        clusters[key_tuple].append(ci)

    eq_classes = []
    for key_tuple, indices in clusters.items():
        key_dict = dict(key_tuple)
        label = "-".join(str(v) for v in key_dict.values())
        sample_idx = indices[0]
        sample_card = cards[sample_idx]
        sample_name = " ".join(sample_card["texts"][:2]) if sample_card["texts"] else f"Card {sample_idx}"

        eq_classes.append({
            "label": label,
            "variant_key": key_dict,
            "sample_row_index": sample_idx,
            "sample_uid": sample_card["ref"],
            "sample_name": sample_name,
            "count": len(indices),
        })

    eq_classes.sort(key=lambda c: -c["count"])

    return {
        "name": name,
        "total_items": len(cards),
        "equivalence_classes": eq_classes,
        "total_samples": len(eq_classes),
        "reduction": f"{len(cards)} items -> {len(eq_classes)} samples",
    }


def _classify_list_group(list_data):
    """Classify a list's items into equivalence classes."""
    items = list_data["items"]
    return _classify_card_group(list_data["name"], items)


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def classify(snapshot_text):
    """Main classification entry point. Returns the full result dict."""
    lines = _parse_lines(snapshot_text)

    tables = _extract_tables(lines)
    card_groups = _extract_cards(lines)
    list_groups = _extract_lists(lines)

    result = {
        "tables": [_classify_table(t) for t in tables],
        "cards": [_classify_card_group(name, cards) for name, cards in card_groups],
        "lists": [_classify_list_group(lg) for lg in list_groups],
    }

    return result


def main():
    parser = argparse.ArgumentParser(
        description="Classify a11y snapshot entities into equivalence classes"
    )
    parser.add_argument("snapshot", help="Path to a11y snapshot text file")
    parser.add_argument(
        "--output", "-o",
        help="Output JSON file (default: stdout)",
        default=None,
    )
    args = parser.parse_args()

    try:
        with open(args.snapshot, "r", encoding="utf-8") as f:
            text = f.read()
    except FileNotFoundError:
        print(f"error: snapshot file not found: {args.snapshot}", file=sys.stderr)
        sys.exit(1)
    except OSError as e:
        print(f"error: could not read snapshot: {e}", file=sys.stderr)
        sys.exit(1)

    result = classify(text)

    output_json = json.dumps(result, indent=2)

    if args.output:
        with open(args.output, "w", encoding="utf-8") as f:
            f.write(output_json)
            f.write("\n")
        # Print summary to stderr
        total_tables = len(result["tables"])
        total_cards = len(result["cards"])
        total_lists = len(result["lists"])
        total_samples = sum(t["total_samples"] for t in result["tables"])
        total_samples += sum(c["total_samples"] for c in result["cards"])
        total_samples += sum(l["total_samples"] for l in result["lists"])
        total_items = sum(t["total_rows"] for t in result["tables"])
        total_items += sum(c["total_items"] for c in result["cards"])
        total_items += sum(l["total_items"] for l in result["lists"])
        print(
            f"Classified {total_items} items -> {total_samples} samples "
            f"({total_tables} tables, {total_cards} card groups, {total_lists} lists) "
            f"-> {args.output}",
            file=sys.stderr,
        )
    else:
        print(output_json)


if __name__ == "__main__":
    main()
