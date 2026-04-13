#!/usr/bin/env python3
"""Parse an a11y snapshot into a capture manifest — a mechanical checklist of every
element that needs to be tested during the interaction audit.

Usage:
    parse-snapshot.py <snapshot.txt> [--output <manifest.json>]

The manifest removes judgment from capture. The orchestrator works through it
item by item instead of deciding what to test.
"""
import re
import json
import sys
from collections import defaultdict


def parse_snapshot(text):
    """Extract interactive elements from a11y snapshot text."""
    elements = {
        "buttons": [],
        "links": [],
        "table_rows": [],
        "dropdowns": [],
        "tabs": [],
        "inputs": [],
        "checkboxes": [],
        "icon_buttons": [],
        "nav_items": [],
        "cards": [],
        "other_interactive": [],
    }

    for line in text.split("\n"):
        line = line.strip()
        if not line:
            continue

        # Extract UID — patterns like [uid=abc123] or uid: abc123
        uid_match = re.search(r'\[uid[=:]([^\]]+)\]|uid[=:]\s*(\S+)', line, re.IGNORECASE)
        if not uid_match:
            # Also try patterns like "- button 'text' [ref12]"
            uid_match = re.search(r'\[(\w+)\]\s*$', line)

        uid = None
        if uid_match:
            uid = uid_match.group(1) or uid_match.group(2) if uid_match.lastindex and uid_match.lastindex > 1 else uid_match.group(1)

        if not uid:
            continue

        line_lower = line.lower()

        # Categorize by role/type
        if re.search(r'\bbutton\b', line_lower):
            # Distinguish icon buttons (no visible text or very short)
            name = extract_name(line)
            if len(name) <= 2 or name in ['×', '⋮', '…', '•']:
                elements["icon_buttons"].append({"uid": uid, "name": name, "raw": line.strip()})
            else:
                elements["buttons"].append({"uid": uid, "name": name, "raw": line.strip()})
        elif re.search(r'\blink\b', line_lower):
            name = extract_name(line)
            if re.search(r'nav|sidebar|menu', line_lower):
                elements["nav_items"].append({"uid": uid, "name": name, "raw": line.strip()})
            else:
                elements["links"].append({"uid": uid, "name": name, "raw": line.strip()})
        elif re.search(r'\b(combobox|listbox|select|dropdown)\b', line_lower):
            elements["dropdowns"].append({"uid": uid, "name": extract_name(line), "raw": line.strip()})
        elif re.search(r'\btab\b', line_lower) and not re.search(r'\btable\b', line_lower):
            elements["tabs"].append({"uid": uid, "name": extract_name(line), "raw": line.strip()})
        elif re.search(r'\b(textbox|input|textarea|searchbox)\b', line_lower):
            elements["inputs"].append({"uid": uid, "name": extract_name(line), "raw": line.strip()})
        elif re.search(r'\b(checkbox|switch|toggle)\b', line_lower):
            elements["checkboxes"].append({"uid": uid, "name": extract_name(line), "raw": line.strip()})
        elif re.search(r'\brow\b', line_lower) and re.search(r'\bcell\b', line_lower):
            elements["table_rows"].append({"uid": uid, "name": extract_name(line), "raw": line.strip()})
        elif re.search(r'\b(card|article)\b', line_lower):
            elements["cards"].append({"uid": uid, "name": extract_name(line), "raw": line.strip()})
        elif re.search(r'\b(menuitem|option|treeitem)\b', line_lower):
            elements["other_interactive"].append({"uid": uid, "name": extract_name(line), "raw": line.strip()})

    return elements


def extract_name(line):
    """Extract a human-readable name from a snapshot line."""
    # Try to find quoted text
    match = re.search(r"['\"]([^'\"]+)['\"]", line)
    if match:
        return match.group(1)[:50]
    # Try to find text after role keyword
    match = re.search(r'(?:button|link|tab)\s+(.+?)(?:\[|$)', line, re.IGNORECASE)
    if match:
        return match.group(1).strip()[:50]
    return line.strip()[:50]


def generate_manifest(elements):
    """Generate the capture manifest with specific instructions per element."""
    manifest = {
        "total_interactive_elements": sum(len(v) for v in elements.values()),
        "expected_min_screenshots": 0,
        "audit_tasks": []
    }

    task_id = 0

    # Hover tasks for buttons
    for btn in elements["buttons"]:
        task_id += 1
        manifest["audit_tasks"].append({
            "id": task_id,
            "type": "hover",
            "uid": btn["uid"],
            "element": f"button: {btn['name']}",
            "action": "hover",
            "screenshot": f"hover-btn-{sanitize(btn['name'])}.png",
            "check": "visual feedback on hover (background change, shadow, or scale)"
        })

    # Hover tasks for links
    for link in elements["links"]:
        task_id += 1
        manifest["audit_tasks"].append({
            "id": task_id,
            "type": "hover",
            "uid": link["uid"],
            "element": f"link: {link['name']}",
            "action": "hover",
            "screenshot": f"hover-link-{sanitize(link['name'])}.png",
            "check": "underline or color change on hover"
        })

    # Hover tasks for nav items
    for nav in elements["nav_items"]:
        task_id += 1
        manifest["audit_tasks"].append({
            "id": task_id,
            "type": "hover",
            "uid": nav["uid"],
            "element": f"nav: {nav['name']}",
            "action": "hover",
            "screenshot": f"hover-nav-{sanitize(nav['name'])}.png",
            "check": "hover highlight distinct from active state"
        })

    # Hover tasks for icon buttons (also check for tooltip)
    for icon in elements["icon_buttons"]:
        task_id += 1
        manifest["audit_tasks"].append({
            "id": task_id,
            "type": "hover",
            "uid": icon["uid"],
            "element": f"icon-button: {icon['name']}",
            "action": "hover",
            "screenshot": f"tooltip-{sanitize(icon['name'])}.png",
            "check": "tooltip appears explaining the icon's purpose"
        })

    # Table row hovers (first 5 rows)
    for i, row in enumerate(elements["table_rows"][:5]):
        task_id += 1
        manifest["audit_tasks"].append({
            "id": task_id,
            "type": "hover",
            "uid": row["uid"],
            "element": f"table-row-{i+1}",
            "action": "hover",
            "screenshot": f"table-row-hover-{i+1}.png",
            "check": "row highlight on hover"
        })

    # Click tasks for dropdowns
    for dd in elements["dropdowns"]:
        task_id += 1
        manifest["audit_tasks"].append({
            "id": task_id,
            "type": "click",
            "uid": dd["uid"],
            "element": f"dropdown: {dd['name']}",
            "action": "click",
            "screenshot": f"expanded-{sanitize(dd['name'])}.png",
            "check": "dropdown opens, options visible, no overflow",
            "restore": "click again to close"
        })

    # Click tasks for tabs
    for tab in elements["tabs"]:
        task_id += 1
        manifest["audit_tasks"].append({
            "id": task_id,
            "type": "click",
            "uid": tab["uid"],
            "element": f"tab: {tab['name']}",
            "action": "click",
            "screenshot": f"tab-{sanitize(tab['name'])}.png",
            "check": "tab content switches, active state visible"
        })

    # Focus audit (not per-element, just a sweep)
    task_id += 1
    manifest["audit_tasks"].append({
        "id": task_id,
        "type": "focus-sweep",
        "element": "entire page",
        "action": "press Tab repeatedly, screenshot every 3-4 stops",
        "screenshot": "focus-{N}.png",
        "check": "visible focus ring, logical tab order, no traps",
        "expected_screenshots": max(3, manifest["total_interactive_elements"] // 5)
    })

    # Sub-page navigation for links
    nav_links = [l for l in elements["links"] if any(kw in l["name"].lower() for kw in ["view", "detail", "open", "go to", "see"])]
    for link in nav_links[:3]:  # Follow up to 3 sub-pages
        task_id += 1
        manifest["audit_tasks"].append({
            "id": task_id,
            "type": "navigate",
            "uid": link["uid"],
            "element": f"sub-page: {link['name']}",
            "action": "click to navigate, wait for load, screenshot, take snapshot",
            "screenshot": f"subpage-{sanitize(link['name'])}.png",
            "check": "page loads, data displays correctly, navigation works",
            "restore": "navigate back"
        })

    # Responsive checks
    for width, name in [(1440, "1440"), (1024, "1024"), (768, "768")]:
        task_id += 1
        manifest["audit_tasks"].append({
            "id": task_id,
            "type": "responsive",
            "element": f"viewport {width}px",
            "action": f"resize_page to {width}",
            "screenshot": f"responsive-{name}.png",
            "check": "layout holds, no horizontal scroll, no overlapping"
        })

    # Scroll check
    task_id += 1
    manifest["audit_tasks"].append({
        "id": task_id,
        "type": "scroll",
        "element": "page content",
        "action": "scroll to bottom, screenshot at each screenful",
        "screenshot": "scroll-{N}.png",
        "check": "below-fold content captured"
    })

    # Count expected screenshots
    hover_count = len([t for t in manifest["audit_tasks"] if t["type"] == "hover"])
    click_count = len([t for t in manifest["audit_tasks"] if t["type"] == "click"])
    nav_count = len([t for t in manifest["audit_tasks"] if t["type"] == "navigate"])
    focus_tasks = [t for t in manifest["audit_tasks"] if t["type"] == "focus-sweep"]
    focus_count = focus_tasks[0].get("expected_screenshots", 3) if focus_tasks else 3

    manifest["expected_min_screenshots"] = (
        1 +  # initial
        hover_count +
        click_count +
        nav_count * 2 +  # screenshot + snapshot per sub-page
        focus_count +
        3 +  # responsive
        2    # scroll
    )

    manifest["summary"] = {
        "buttons": len(elements["buttons"]),
        "links": len(elements["links"]),
        "nav_items": len(elements["nav_items"]),
        "icon_buttons": len(elements["icon_buttons"]),
        "table_rows": len(elements["table_rows"]),
        "dropdowns": len(elements["dropdowns"]),
        "tabs": len(elements["tabs"]),
        "inputs": len(elements["inputs"]),
        "checkboxes": len(elements["checkboxes"]),
        "cards": len(elements["cards"]),
    }

    return manifest


def sanitize(name):
    """Sanitize a name for use in filenames."""
    return re.sub(r'[^a-z0-9]+', '-', name.lower().strip())[:30].strip('-')


def main():
    if len(sys.argv) < 2:
        print("Usage: parse-snapshot.py <snapshot.txt> [--output <manifest.json>]")
        sys.exit(1)

    snapshot_path = sys.argv[1]
    output_path = None
    if "--output" in sys.argv:
        idx = sys.argv.index("--output")
        if idx + 1 < len(sys.argv):
            output_path = sys.argv[idx + 1]

    with open(snapshot_path) as f:
        text = f.read()

    elements = parse_snapshot(text)
    manifest = generate_manifest(elements)

    output = json.dumps(manifest, indent=2)

    if output_path:
        with open(output_path, "w") as f:
            f.write(output)
        print(f"Manifest saved to {output_path}")
        print(f"Total elements: {manifest['total_interactive_elements']}")
        print(f"Audit tasks: {len(manifest['audit_tasks'])}")
        print(f"Expected min screenshots: {manifest['expected_min_screenshots']}")
    else:
        print(output)


if __name__ == "__main__":
    main()
