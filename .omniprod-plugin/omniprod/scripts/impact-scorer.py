#!/usr/bin/env python3
"""
Impact Scorer — scores, ranks, and groups findings from OmniProd product reviews.

Usage:
  python3 impact-scorer.py .omniprod/findings/2026-04-03-compliance.json --output scored.json
  python3 impact-scorer.py .omniprod/findings/*.json --output product-scored.json
  python3 impact-scorer.py .omniprod/findings/*.json --top 10
"""

import argparse
import json
import math
import sys
from datetime import datetime, timezone
from pathlib import Path


# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------

SEVERITY_WEIGHT = {
    "critical": 4.0,
    "major": 3.0,
    "minor": 2.0,
    "nitpick": 1.0,
}

SEVERITY_ORDER = {"critical": 0, "major": 1, "minor": 2, "nitpick": 3}


# ---------------------------------------------------------------------------
# Scoring helpers
# ---------------------------------------------------------------------------

def severity_weight(severity: str) -> float:
    return SEVERITY_WEIGHT.get(severity.lower(), 1.0)


def scope_multiplier(affected_pages: int) -> float:
    return max(1.0, math.log2(affected_pages + 1))


def perspective_weight(perspectives: list) -> float:
    return max(1.0, len(perspectives) / 3)


def age_weight(first_seen: str, now: datetime) -> float:
    if not first_seen:
        return 1.0
    try:
        seen_date = datetime.strptime(first_seen, "%Y-%m-%d").replace(tzinfo=timezone.utc)
        days_open = max(0, (now - seen_date).days)
        return max(1.0, 1.0 + (days_open / 30) * 0.5)
    except (ValueError, TypeError):
        return 1.0


def compute_impact(finding: dict, affected_pages: int, now: datetime) -> float:
    sev = severity_weight(finding.get("severity", "nitpick"))
    scope = scope_multiplier(affected_pages)
    persp = perspective_weight(finding.get("perspectives", []))
    age = age_weight(finding.get("first_seen", ""), now)
    return round(sev * scope * persp * age, 2)


# ---------------------------------------------------------------------------
# Fuzzy matching for root cause grouping
# ---------------------------------------------------------------------------

def word_set(text: str) -> set:
    """Lowercase words from a string, stripping basic punctuation."""
    return set(
        w.strip(".,;:!?\"'()[]{}") for w in text.lower().split() if w.strip(".,;:!?\"'()[]{}")
    )


def word_overlap(a: str, b: str) -> float:
    """Fraction of shared words relative to the union of both word sets."""
    wa, wb = word_set(a), word_set(b)
    if not wa or not wb:
        return 0.0
    return len(wa & wb) / len(wa | wb)


def elements_match(e1: str, e2: str) -> bool:
    """Exact or substring match on element strings."""
    if not e1 or not e2:
        return False
    e1l, e2l = e1.lower(), e2.lower()
    return e1l == e2l or e1l in e2l or e2l in e1l


def same_root_cause(f1: dict, f2: dict) -> bool:
    """Determine whether two findings share a root cause."""
    el1 = f1.get("element", "")
    el2 = f2.get("element", "")
    obs1 = f1.get("observation", "")
    obs2 = f2.get("observation", "")

    # Rule 1: same element AND similar observation
    if elements_match(el1, el2) and word_overlap(obs1, obs2) > 0.6:
        return True

    # Rule 2: same observation pattern across pages
    if obs1 and obs2 and word_overlap(obs1, obs2) > 0.6:
        page1 = f1.get("_page", "")
        page2 = f2.get("_page", "")
        if page1 and page2 and page1 != page2:
            return True

    return False


# ---------------------------------------------------------------------------
# Root cause clustering (simple union-find)
# ---------------------------------------------------------------------------

class UnionFind:
    def __init__(self, n: int):
        self.parent = list(range(n))
        self.rank = [0] * n

    def find(self, x: int) -> int:
        while self.parent[x] != x:
            self.parent[x] = self.parent[self.parent[x]]
            x = self.parent[x]
        return x

    def union(self, a: int, b: int):
        ra, rb = self.find(a), self.find(b)
        if ra == rb:
            return
        if self.rank[ra] < self.rank[rb]:
            ra, rb = rb, ra
        self.parent[rb] = ra
        if self.rank[ra] == self.rank[rb]:
            self.rank[ra] += 1


def cluster_findings(findings: list[dict]) -> dict[int, list[int]]:
    """Return clusters as {root_index: [member_indices]}."""
    n = len(findings)
    uf = UnionFind(n)
    for i in range(n):
        for j in range(i + 1, n):
            if same_root_cause(findings[i], findings[j]):
                uf.union(i, j)
    clusters: dict[int, list[int]] = {}
    for i in range(n):
        root = uf.find(i)
        clusters.setdefault(root, []).append(i)
    return clusters


# ---------------------------------------------------------------------------
# Loading
# ---------------------------------------------------------------------------

def extract_page(url: str) -> str:
    """Extract path portion from a URL, falling back to the raw value."""
    if not url:
        return ""
    # Strip scheme + host
    if "://" in url:
        url = url.split("://", 1)[1]
    slash = url.find("/")
    if slash == -1:
        return "/"
    return url[slash:]


def load_findings(paths: list[str]) -> tuple[list[dict], list[str]]:
    """Load and merge findings from one or more JSON files.

    Returns (all_findings, source_files).  Each finding gets a ``_page``
    and ``_source`` annotation.
    """
    all_findings: list[dict] = []
    source_files: list[str] = []

    for p in paths:
        fp = Path(p)
        if not fp.exists():
            print(f"warning: {p} not found, skipping", file=sys.stderr)
            continue
        try:
            data = json.loads(fp.read_text())
        except json.JSONDecodeError as exc:
            print(f"warning: {p} invalid JSON ({exc}), skipping", file=sys.stderr)
            continue

        source_files.append(fp.name)
        page = extract_page(data.get("url", ""))
        for f in data.get("findings", []):
            f["_page"] = page
            f["_source"] = fp.name
            all_findings.append(f)

    return all_findings, source_files


# ---------------------------------------------------------------------------
# Main logic
# ---------------------------------------------------------------------------

def highest_severity(severities: list[str]) -> str:
    best = "nitpick"
    for s in severities:
        sl = s.lower()
        if SEVERITY_ORDER.get(sl, 99) < SEVERITY_ORDER.get(best, 99):
            best = sl
    return best


def build_label(findings: list[dict], indices: list[int]) -> str:
    """Create a human-readable label for a root-cause cluster."""
    # Use the observation of the highest-severity finding
    best_idx = min(indices, key=lambda i: SEVERITY_ORDER.get(findings[i].get("severity", "nitpick").lower(), 99))
    obs = findings[best_idx].get("observation", "")
    if len(obs) > 80:
        obs = obs[:77] + "..."
    return obs or findings[best_idx].get("element", "unknown")


def infer_fix_scope(findings: list[dict], indices: list[int]) -> str:
    """Best-effort fix scope from the cluster's suggestions."""
    suggestions = [findings[i].get("suggestion", "") for i in indices if findings[i].get("suggestion")]
    if suggestions:
        # Return the longest (most detailed) suggestion, capped
        best = max(suggestions, key=len)
        return best[:200] if len(best) > 200 else best
    return "Investigate and fix"


def score_and_rank(paths: list[str], now: datetime) -> dict:
    findings, source_files = load_findings(paths)
    if not findings:
        return {
            "scored_at": now.isoformat(),
            "source_files": source_files,
            "total_findings": 0,
            "unique_root_causes": 0,
            "findings_ranked": [],
            "root_causes": [],
            "summary": {
                "by_severity": {},
                "by_page": {},
                "top_root_causes": [],
            },
        }

    # -- Cluster root causes --------------------------------------------------
    clusters = cluster_findings(findings)

    # Build root-cause records and assign RC IDs
    rc_records: list[dict] = []
    finding_to_rc: dict[int, str] = {}

    # Sort clusters by highest severity, then size (largest first)
    sorted_cluster_keys = sorted(
        clusters.keys(),
        key=lambda k: (
            SEVERITY_ORDER.get(
                highest_severity([findings[i].get("severity", "nitpick") for i in clusters[k]]),
                99,
            ),
            -len(clusters[k]),
        ),
    )

    for rc_num, root_idx in enumerate(sorted_cluster_keys, start=1):
        indices = clusters[root_idx]
        rc_id = f"RC-{rc_num:03d}"

        for idx in indices:
            finding_to_rc[idx] = rc_id

        pages = sorted(set(findings[i].get("_page", "") for i in indices))
        severities = [findings[i].get("severity", "nitpick") for i in indices]
        finding_ids = [findings[i].get("id", f"?-{i}") for i in indices]

        # Compute the max impact score across the cluster's findings
        affected_page_count = max(1, len(pages))
        max_score = max(
            compute_impact(findings[i], affected_page_count, now) for i in indices
        )

        rc_records.append({
            "id": rc_id,
            "label": build_label(findings, indices),
            "max_severity": highest_severity(severities),
            "impact_score": max_score,
            "findings": finding_ids,
            "pages": pages,
            "fix_scope": infer_fix_scope(findings, indices),
        })

    # -- Score individual findings --------------------------------------------
    # For each finding, affected_pages = number of distinct pages in its RC cluster
    scored: list[dict] = []
    for idx, f in enumerate(findings):
        rc_id = finding_to_rc.get(idx, "RC-???")
        # Find the cluster this finding belongs to
        rc_cluster_indices = []
        for root_idx, members in clusters.items():
            if idx in members:
                rc_cluster_indices = members
                break
        affected_pages = max(1, len(set(findings[i].get("_page", "") for i in rc_cluster_indices)))

        impact = compute_impact(f, affected_pages, now)
        days_open = 0
        if f.get("first_seen"):
            try:
                seen = datetime.strptime(f["first_seen"], "%Y-%m-%d").replace(tzinfo=timezone.utc)
                days_open = max(0, (now - seen).days)
            except (ValueError, TypeError):
                pass

        perspectives_raw = f.get("perspectives", [])
        if len(perspectives_raw) >= 8:
            persp_display = "all 8"
        else:
            persp_display = ", ".join(perspectives_raw) if perspectives_raw else "none"

        pages_list = sorted(set(findings[i].get("_page", "") for i in rc_cluster_indices))

        scored.append({
            "rank": 0,  # filled after sort
            "id": f.get("id", f"?-{idx}"),
            "root_cause": rc_id,
            "severity": f.get("severity", "nitpick"),
            "impact_score": impact,
            "element": f.get("element", ""),
            "observation": f.get("observation", ""),
            "pages": pages_list,
            "perspectives": persp_display,
            "suggestion": f.get("suggestion", ""),
            "status": f.get("status", "open"),
            "first_seen": f.get("first_seen", ""),
            "days_open": days_open,
        })

    # Sort by impact_score desc, then severity asc
    scored.sort(key=lambda s: (-s["impact_score"], SEVERITY_ORDER.get(s["severity"].lower(), 99)))
    for i, s in enumerate(scored, start=1):
        s["rank"] = i

    # -- Summary ---------------------------------------------------------------
    by_severity: dict[str, int] = {}
    by_page: dict[str, int] = {}
    for f in findings:
        sev = f.get("severity", "nitpick").lower()
        by_severity[sev] = by_severity.get(sev, 0) + 1
        page = f.get("_page", "unknown")
        by_page[page] = by_page.get(page, 0) + 1

    # Sort rc_records by impact score desc
    rc_records.sort(key=lambda r: -r["impact_score"])
    top_rcs = [r["id"] for r in rc_records[:5]]

    return {
        "scored_at": now.isoformat(),
        "source_files": source_files,
        "total_findings": len(findings),
        "unique_root_causes": len(rc_records),
        "findings_ranked": scored,
        "root_causes": rc_records,
        "summary": {
            "by_severity": by_severity,
            "by_page": by_page,
            "top_root_causes": top_rcs,
        },
    }


# ---------------------------------------------------------------------------
# Display
# ---------------------------------------------------------------------------

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


# ---------------------------------------------------------------------------
# CLI
# ---------------------------------------------------------------------------

def main():
    parser = argparse.ArgumentParser(
        description="Score and rank OmniProd product review findings.",
    )
    parser.add_argument(
        "files",
        nargs="+",
        help="One or more findings JSON files to process.",
    )
    parser.add_argument(
        "--output", "-o",
        help="Write scored JSON to this file.",
    )
    parser.add_argument(
        "--top",
        type=int,
        default=0,
        help="Print top N findings to stdout in table format.",
    )

    args = parser.parse_args()
    now = datetime.now(timezone.utc)

    result = score_and_rank(args.files, now)

    if args.output:
        out_path = Path(args.output)
        out_path.parent.mkdir(parents=True, exist_ok=True)
        # Strip internal annotation keys before writing
        for f in result.get("findings_ranked", []):
            f.pop("_page", None)
            f.pop("_source", None)
        out_path.write_text(json.dumps(result, indent=2) + "\n")
        print(f"Scored {result['total_findings']} findings "
              f"({result['unique_root_causes']} root causes) -> {args.output}")

    if args.top:
        print_top(result, args.top)

    # If neither --output nor --top, dump to stdout
    if not args.output and not args.top:
        print(json.dumps(result, indent=2))


if __name__ == "__main__":
    main()
