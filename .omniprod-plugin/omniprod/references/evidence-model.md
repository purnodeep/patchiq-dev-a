# OmniProd — Evidence Model

## Layer 1: Programmatic Assertions

Before AI perspectives review a page, automated JavaScript assertions run directly in the browser via Chrome DevTools `evaluate_script`. These catch objective, reproducible issues instantly.

### Components

| Component | Purpose | Location |
|-----------|---------|----------|
| `assertions-runner.js` | Built-in checks: DOM errors, a11y basics, data integrity, performance | `scripts/assertions-runner.js` |
| `generate-assertions.py` | Generates custom assertions from source code analysis | `scripts/generate-assertions.py` |
| `assertion-defs.json` | Custom assertion definitions (generated per page) | `.omniprod/screenshots/current/assertion-defs.json` |
| `assertion-results.json` | Pass/fail results from running assertions | `.omniprod/screenshots/current/assertion-results.json` |

### Built-in Assertion Categories

- **Console/Error**: React error boundaries, DOM error indicators
- **Accessibility**: image alt text, form labels, heading structure, button names, main landmark
- **Data Integrity**: no raw undefined/null/NaN in visible text, tables have data or empty state
- **Performance**: DOM element count within reasonable bounds

### Custom Assertions (generated from source code)

`generate-assertions.py` analyzes a page's component file, finds TanStack Query hooks, traces to API endpoints, and generates JavaScript snippets that verify:
- API response counts match DOM table row counts
- Empty tables display proper empty state components
- No perpetual loading spinners remain visible

### How Assertion Failures Become Findings

Failed assertions are converted to findings with:
- `source: "assertion"` (vs `source: "perspective"` for AI findings)
- `assertion_id`: the assertion's unique ID (e.g., `BUILTIN-A11Y-002`)
- Severity from the assertion definition
- Element and observation from the assertion result's `actual` vs `expected` fields

## Principle

Reviews are evidence-based, not screenshot-based. Evidence is multi-modal — each captured state produces multiple evidence types. Perspectives receive evidence tailored to their concerns.

## Evidence Types

| Type | Source | What It Reveals | Chrome DevTools Tool |
|------|--------|-----------------|---------------------|
| Visual | Screenshots (.png) | What users see | `take_screenshot` |
| Structural | A11y snapshots (.txt) | DOM structure, ARIA roles, labels | `take_snapshot` |
| Data | Network requests/responses | API contracts, data shapes | `list_network_requests`, `get_network_request` |
| Behavioral | Console messages | Errors, warnings, runtime issues | `list_console_messages` |
| Performance | Lighthouse audit | Accessibility score, best practices, SEO | `lighthouse_audit` |

## Evidence Distribution

All perspectives receive ALL evidence types. There is no perspective-specific filtering — every reviewer sees the complete evidence package including screenshots, snapshots, console logs, network data, and assertion results.

## State Flow

Every screenshot exists in a flow:
```
page-load → scroll-1 → hover-evaluate → click-evaluate → ...
```

The annotated-captures.md documents this flow so reviewers understand the sequence of actions, not just isolated images.

## Business Rules

Business rules extracted from source code are testable assertions:
```
BR-001: "Only active frameworks appear in dashboard"
  → Check: all framework cards have active status
  → Verify against: API response data + visible UI
```

Failed business rules become findings. Passing rules confirm correctness.

## Entity Data

Entity values are recorded at each state for cross-page correlation:
```json
{
  "state": "SC-001",
  "entities": {
    "framework_count": 2,
    "framework_names": ["NIST CSF", "test-001"],
    "overall_score": "75%",
    "overdue_count": 4
  }
}
```

When the same entity appears on multiple pages (via flow reviews or product-level aggregation), values are compared programmatically.
