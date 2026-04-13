# OmniProd — Universal Findings Schema

## Review File: `.omniprod/findings/{YYYY-MM-DD}-{page-slug}.json`

```json
{
  "review_id": "2026-04-03-compliance",
  "url": "http://localhost:3001/compliance",
  "date": "2026-04-03T14:30:00Z",
  "page_name": "Compliance Dashboard",
  "source_type": "ui",
  "overall_verdict": "FAIL",

  "perspectives": {
    "ux-designer": {
      "verdict": "FAIL",
      "findings_count": {
        "critical": 1,
        "major": 3,
        "minor": 2,
        "nitpick": 0
      }
    }
  },

  "findings": [
    {
      "id": "PO-001",
      "severity": "critical",
      "element": "Overdue Controls table, FRAMEWORK column",
      "observation": "Raw UUID rendered as framework name instead of display name",
      "suggestion": "Fix SQL join in overdue controls query to resolve framework UUID to name",
      "standard_violated": "data-integrity.md#data-display",
      "perspectives": ["ux-designer", "qa-engineer", "enterprise-buyer"],
      "screenshot": "page-load.png",
      "status": "open",
      "first_seen": "2026-04-03",
      "last_seen": "2026-04-03",
      "fixed_on": null,
      "regressed_on": null,
      "business_rule": "BR-001",
      "entity": "framework",
      "root_cause_group": null,
      "impact_score": null,
      "source": "assertion",
      "assertion_id": "BUILTIN-DI-001"
    }
  ],

  "business_rule_results": [
    {
      "id": "BR-001",
      "rule": "Overdue controls show framework display name, not UUID",
      "source": "internal/server/store/queries/compliance.sql",
      "result": "FAIL",
      "details": "Row 2 shows raw UUID bbb41940-3cd2-... instead of framework name",
      "related_finding": "PO-001"
    }
  ],

  "lighthouse": {
    "accessibility": 94,
    "best_practices": 100,
    "seo": 60
  },
  "lighthouse_report": ".omniprod/screenshots/current/report.html",

  "capture_stats": {
    "planned": 52,
    "captured": 48,
    "skipped": 4,
    "entity_samples": {
      "controls": "43 rows -> 2 samples",
      "frameworks": "10 -> 3 samples"
    }
  },

  "console_errors": 1,
  "network_failures": 1,

  "stats": {
    "critical": 1,
    "major": 3,
    "minor": 2,
    "nitpick": 0,
    "total": 6
  },

  "delta": {
    "vs_previous": "2026-04-02-compliance",
    "fixed": 2,
    "new": 1,
    "remaining": 3,
    "trend": "improving"
  }
}
```

## Field Definitions

### source_type

Extensible source identifier for cross-domain reviews:

| Value | Description |
|-------|-------------|
| `ui` | Browser-based UI review (current) |
| `api` | Direct API contract review (future) |
| `code` | Static code analysis review (future) |
| `flow` | Cross-page flow review |
| `smoke` | Smoke test (page-level status only) |
| `shared` | Shared component review |

### Finding Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unified ID: PO-001, PO-002, etc. |
| `severity` | enum | critical, major, minor, nitpick |
| `element` | string | Specific UI element or area |
| `observation` | string | What's wrong — precise, references screenshot |
| `suggestion` | string | Specific fix — not "consider", state what SHOULD be |
| `standard_violated` | string | Which standard file + section |
| `perspectives` | string[] | Which perspectives flagged this |
| `screenshot` | string | Screenshot filename showing the issue |
| `status` | enum | open, fixed, regressed, wontfix |
| `first_seen` | date | When first discovered (carried forward across reviews) |
| `last_seen` | date | Most recent review where this was found |
| `fixed_on` | date/null | When it was no longer found |
| `regressed_on` | date/null | When a fixed finding reappeared |
| `business_rule` | string/null | BR-XXX if this violates a business rule |
| `entity` | string/null | Entity type affected (framework, endpoint, etc.) |
| `root_cause_group` | string/null | RC-XXX for product-level grouping |
| `impact_score` | number/null | Composite score from impact-scorer.py |
| `assertion_id` | string/null | The assertion ID if this finding came from an automated assertion (e.g., `BUILTIN-A11Y-002`). Null for perspective-sourced findings. |
| `source` | enum | Whether this finding came from Layer 1 (`"assertion"`) or Layer 2 (`"perspective"`). |

### Finding Status Lifecycle

```
open → fixed (not found in next review)
open → open (still found in next review)
fixed → regressed (was fixed, found again)
open → wontfix (manually marked as accepted)
```

### Severity Levels

| Level | Meaning | Blocks Ship? |
|-------|---------|-------------|
| `critical` | Loses enterprise deals, broken flows, wrong data, WCAG A failures | Yes |
| `major` | Hurts credibility, inconsistent patterns, missing states, WCAG AA failures | Conditional |
| `minor` | Polish issues, slight misalignment, could-be-better copy | No |
| `nitpick` | Perfectionist details, only matters when everything else is clean | No |

### Verdict Logic

| Condition | Verdict |
|-----------|---------|
| ALL perspectives PASS + zero critical + zero major | `PASS` |
| All perspectives PASS but has major findings | `CONDITIONAL_PASS` |
| ANY perspective FAIL or any critical finding | `FAIL` |

### Business Rule Results

Business rules are testable assertions extracted from source code during Phase 0. Each rule is verified against what's visible on the page.

| Field | Description |
|-------|-------------|
| `id` | BR-001, BR-002, etc. |
| `rule` | Human-readable rule statement |
| `source` | Source code file where rule was found |
| `result` | PASS or FAIL |
| `details` | What was checked, what was found |
| `related_finding` | PO-XXX if a UI finding was created for this failure |

### Capture Stats

Tracks the quality of the capture phase:

| Field | Description |
|-------|-------------|
| `planned` | Steps in capture plan |
| `captured` | Screenshots actually taken |
| `skipped` | Steps skipped (element not found, duplicate state) |
| `entity_samples` | Table/list sampling decisions |

### Delta Tracking

The `delta` section compares to the most recent previous review of the same page:
- `fixed`: findings present before but not now
- `new`: findings not present before but found now
- `remaining`: findings present in both reviews
- `trend`: `"improving"` if fixed > new, `"degrading"` if new > fixed, `"stable"` if equal

### Impact Score (Product-Level)

When running `impact-scorer.py` across multiple reviews:

```
impact_score = severity_weight x scope_multiplier x perspective_weight x age_weight

severity_weight:   critical=4, major=3, minor=2, nitpick=1
scope_multiplier:  max(1.0, log2(affected_pages + 1))
perspective_weight: max(1.0, len(perspectives) / 3)
age_weight:        max(1.0, 1.0 + (days_open / 30) * 0.5)
```

### Root Cause Groups (Product-Level)

When aggregating across pages, findings are grouped by root cause:

```json
{
  "id": "RC-001",
  "label": "Seed data artifact 'test-001' visible across product",
  "max_severity": "critical",
  "impact_score": 12.4,
  "findings": ["PO-005", "WF-018"],
  "pages": ["/compliance", "/workflows", "/dashboard"],
  "fix_scope": "Rename seed framework in database"
}
```

## File Organization

```
.omniprod/
├── product-map.json              # Product structure (pages, entities, flows)
├── config.json                   # Review configuration
├── reviews/
│   ├── {date}-{page-slug}.md     # Page review report
│   ├── {date}-flow-{id}.md       # Flow review report
│   ├── {date}-smoke.md           # Smoke test report
│   ├── {date}-shared.md          # Shared component report
│   └── {date}-product-report.md  # Product-level aggregate
├── findings/
│   ├── {date}-{page-slug}.json   # Page findings
│   ├── {date}-flow-{id}.json     # Flow findings
│   ├── {date}-smoke.json         # Smoke findings
│   ├── {date}-shared.json        # Shared component findings
│   └── {date}-product-scored.json # Impact-scored product findings
├── screenshots/
│   ├── current/                  # Active review screenshots
│   │   ├── 00-initial.png
│   │   ├── 00-snapshot.txt
│   │   ├── 00-console.txt
│   │   ├── 00-network.txt
│   │   ├── 00-business-context.md
│   │   ├── capture-plan.json
│   │   ├── capture-log.jsonl
│   │   ├── annotated-captures.md
│   │   ├── evidence-package.json
│   │   ├── entity-classes.json
│   │   ├── exploration/          # State explorer data
│   │   ├── report.html           # Lighthouse
│   │   └── *.png                 # All captured screenshots
│   ├── flows/                    # Flow review screenshots
│   │   └── {flow-id}/
│   ├── smoke/                    # Smoke test screenshots
│   │   └── {date}/
│   └── archive/                  # Timestamped archives
│       └── {YYYYMMDD-HHMMSS}/
└── standards-overrides/          # Project-specific standard overrides
```
