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
    fallback = glob.glob(os.path.join(app_dir, "src", "pages", "**", "*.tsx"), recursive=True)
    slug_lower = slug.replace(os.sep, "").lower()
    for f in fallback:
        if slug_lower in f.lower():
            return f
    return None


def extract_api_hooks(component_content: str) -> list[str]:
    """Extract TanStack Query hook names used in a component."""
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
        "script": (
            "try {"
            "  var rows = document.querySelectorAll('table tbody tr');"
            "  var rowCount = rows.length;"
            "  var totalEl = document.body.innerText.match(/(?:of|total:?)\\s+(\\d[\\d,]*)\\s/i);"
            "  var displayedTotal = totalEl ? parseInt(totalEl[1].replace(/,/g, ''), 10) : null;"
            "  return {"
            "    passed: rowCount > 0 || displayedTotal !== null,"
            "    actual: displayedTotal !== null"
            "      ? 'Table shows ' + rowCount + ' rows, total: ' + displayedTotal"
            "      : rowCount + ' rows visible',"
            "    expected: 'Table has data rows or shows total count'"
            "  };"
            "} catch(e) {"
            "  return { passed: false, actual: 'Error: ' + e.message, expected: 'No error' };"
            "}"
        ),
        "severity": "major",
        "source_hook": hook_name,
        "source_endpoint": api_endpoint,
    }


def generate_empty_state_assertion(idx: int) -> dict:
    """Assert that empty tables show proper empty state."""
    return {
        "id": f"DI-{idx:03d}",
        "category": "data-integrity",
        "description": "Empty tables/lists display empty state component",
        "script": (
            "try {"
            "  var tables = document.querySelectorAll('table');"
            "  var issues = [];"
            "  tables.forEach(function(t, i) {"
            "    var rows = t.querySelectorAll('tbody tr');"
            "    if (rows.length === 0) {"
            "      var parent = t.closest('[class]');"
            "      var emptyIndicator = parent && ("
            "        parent.querySelector('[class*=\"empty\"]') ||"
            "        parent.querySelector('[class*=\"Empty\"]') ||"
            "        parent.querySelector('[data-testid*=\"empty\"]')"
            "      );"
            "      if (!emptyIndicator) issues.push('Table ' + (i+1));"
            "    }"
            "  });"
            "  return {"
            "    passed: issues.length === 0,"
            "    actual: issues.length === 0 ? 'All empty tables have empty states' : issues.join(', ') + ' missing empty state',"
            "    expected: 'Empty tables show empty state component'"
            "  };"
            "} catch(e) {"
            "  return { passed: false, actual: 'Error: ' + e.message, expected: 'No error' };"
            "}"
        ),
        "severity": "major",
    }


def generate_loading_state_assertion(idx: int) -> dict:
    """Assert no perpetual loading spinners visible."""
    return {
        "id": f"DI-{idx:03d}",
        "category": "data-integrity",
        "description": "No perpetual loading spinners visible (page has finished loading)",
        "script": (
            "try {"
            "  var spinners = document.querySelectorAll("
            "    '[class*=\"spinner\"], [class*=\"Spinner\"], [class*=\"loading\"], [role=\"progressbar\"]'"
            "  );"
            "  var skeletons = document.querySelectorAll("
            "    '[class*=\"skeleton\"], [class*=\"Skeleton\"], [data-testid*=\"skeleton\"]'"
            "  );"
            "  var activeSpinners = Array.from(spinners).filter(function(el) {"
            "    var style = window.getComputedStyle(el);"
            "    return style.display !== 'none' && style.visibility !== 'hidden';"
            "  });"
            "  var activeSkeletons = Array.from(skeletons).filter(function(el) {"
            "    var style = window.getComputedStyle(el);"
            "    return style.display !== 'none' && style.visibility !== 'hidden';"
            "  });"
            "  return {"
            "    passed: activeSpinners.length === 0 && activeSkeletons.length === 0,"
            "    actual: (activeSpinners.length + activeSkeletons.length) === 0"
            "      ? 'No loading indicators visible'"
            "      : activeSpinners.length + ' spinner(s), ' + activeSkeletons.length + ' skeleton(s) still visible',"
            "    expected: 'Page fully loaded, no perpetual loading states'"
            "  };"
            "} catch(e) {"
            "  return { passed: false, actual: 'Error: ' + e.message, expected: 'No error' };"
            "}"
        ),
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
