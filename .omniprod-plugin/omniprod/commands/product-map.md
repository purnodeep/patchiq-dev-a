---
description: "Generate product map — pages, entities, flows, business rules, shared components"
argument-hint: "[--app=web|web-hub|web-agent] [--refresh]"
allowed-tools: ["Read", "Write", "Glob", "Grep", "Bash", "Agent"]
---

# OmniProd — Product Map

Generate `.omniprod/product-map.json` — a comprehensive graph of all pages, entities, user flows, business rules, and shared components. This is the foundation for product-scale reviews.

This command does NOT use the browser. It is pure source code analysis.

## Parse Arguments

Arguments: $ARGUMENTS

- `--app=<name>`: Which app to map. Options: `web`, `web-hub`, `web-agent`. Default: all three.
- `--refresh`: Regenerate even if `.omniprod/product-map.json` already exists.

If `.omniprod/product-map.json` exists and `--refresh` is not set, print the existing summary and exit early.

## Execution

### Step 1: Read Project Structure

1. Read `CLAUDE.md` at project root for product identity, architecture, team info, and platform structure.
2. Read `.omniprod/config.json` for detected context (tech stack, apps, ports).
3. For each target app (from `--app` or all three):
   - Read routes file: `{app}/src/app/routes.tsx`
   - Glob API hooks directory: `{app}/src/api/hooks/*.ts`
   - Glob pages directory: `{app}/src/pages/**/*.tsx`
   - Read layout files: `{app}/src/app/layout/*.tsx`

### Step 2: Build Page Registry

For each route discovered in routes files, create an entry:

```json
{
  "route": "/compliance",
  "name": "Compliance Dashboard",
  "app": "web",
  "component_file": "src/pages/compliance/CompliancePage.tsx",
  "priority": "critical|important|peripheral",
  "tier": "deep-review|flow-review|smoke-only",
  "entities_shown": ["framework", "control", "endpoint", "evaluation"],
  "entities_mutated": ["framework:create/edit/delete", "evaluation:trigger"],
  "api_hooks_used": ["useFrameworks", "useOverdueControls", "useComplianceTrend"],
  "shared_components": ["sidebar", "topbar"],
  "sub_routes": ["/compliance/:id"],
  "outbound_links": ["/compliance/:id", "/endpoints/:id"]
}
```

To populate each entry:
- Read the page component file to find imported hooks, components, and navigation links.
- Use Grep to find `use*` hook calls and `navigate`/`Link` references.
- Cross-reference with the hooks directory to identify API calls.

**Priority assignment:**
- `critical`: Pages with CRUD operations, complex data display, or primary user workflows (dashboard, endpoints, deployments, compliance, workflows)
- `important`: Pages with data display but limited mutation (patches, CVEs, policies, audit, settings)
- `peripheral`: Supporting pages (agent-downloads, roles, notifications settings)

**Tier assignment:**
- `deep-review`: critical pages — full state exploration
- `flow-review`: important pages — covered by user flows
- `smoke-only`: peripheral pages — screenshot + console check only

### Step 3: Build Entity Graph

Read API types files (`{app}/src/api/types.ts`) and hooks to map entities across pages:

```json
{
  "entity_graph": {
    "framework": {
      "created_on": "/compliance",
      "detail_page": "/compliance/:id",
      "shown_on": ["/compliance", "/compliance/:id", "/dashboard"],
      "api_endpoints": ["GET /api/v1/compliance/frameworks", "POST /api/v1/compliance/frameworks"],
      "consistency_checks": [
        "Framework count on /dashboard must match /compliance card count",
        "Framework score on /compliance card must match /compliance/:id header"
      ]
    },
    "endpoint": {
      "created_on": null,
      "detail_page": "/endpoints/:id",
      "shown_on": ["/endpoints", "/endpoints/:id", "/dashboard", "/compliance/:id?tab=endpoints", "/deployments/:id"],
      "api_endpoints": ["GET /api/v1/endpoints"],
      "consistency_checks": [
        "Endpoint count on /dashboard matches /endpoints total",
        "Endpoint compliance status on /endpoints/:id matches /compliance/:id?tab=endpoints"
      ]
    }
  }
}
```

For each entity:
- Identify which pages create, read, update, or delete it.
- Note where the same entity appears on multiple pages (cross-page consistency).
- Generate consistency checks: assertions that the same data must agree across pages.

### Step 4: Define User Flows

Based on the entity graph and page structure, generate 5-10 user flows covering the main user journeys:

```json
{
  "flows": [
    {
      "id": "flow-compliance-assess",
      "name": "Assess Compliance Posture",
      "role": "Compliance Officer",
      "priority": "critical",
      "pages": ["/compliance", "/compliance/:id", "/compliance/:id?tab=controls", "/compliance/:id?tab=endpoints"],
      "steps": [
        {"page": "/compliance", "action": "View dashboard, check scores and overdue controls"},
        {"page": "/compliance", "action": "Click Evaluate All, verify feedback"},
        {"page": "/compliance/:id", "action": "Click View Details on a framework"},
        {"page": "/compliance/:id?tab=controls", "action": "Review controls tab"},
        {"page": "/compliance/:id?tab=endpoints", "action": "Review endpoints tab"},
        {"page": "/compliance", "action": "Navigate back, verify consistency"}
      ],
      "cross_page_assertions": [
        "Framework score on card matches detail page header",
        "Control count on card matches Controls tab row count",
        "Evaluate All updates scores on both dashboard and detail"
      ]
    },
    {
      "id": "flow-deploy-patch",
      "name": "Deploy Patches to Endpoints",
      "role": "IT Admin",
      "priority": "critical",
      "pages": ["/patches", "/deployments/new", "/deployments/:id", "/endpoints/:id"],
      "steps": [
        {"page": "/patches", "action": "Find target patch"},
        {"page": "/deployments/new", "action": "Create deployment, select targets"},
        {"page": "/deployments/:id", "action": "Monitor deployment progress"},
        {"page": "/endpoints/:id", "action": "Verify patch applied"}
      ],
      "cross_page_assertions": [
        "New deployment appears in /deployments list",
        "Deployment status consistent between list and detail",
        "Endpoint patch status updates after deployment"
      ]
    }
  ]
}
```

Flow generation guidelines:
- Cover all `critical` pages in at least one flow.
- Each flow should cross at least 2 pages.
- Include cross-page assertions for every entity that appears on multiple pages in the flow.
- Assign priority: `critical` (core workflows), `important` (secondary workflows), `nice-to-have` (edge cases).
- Roles should reflect target users: IT Admin, Compliance Officer, Security Analyst, Hub Administrator.

### Step 5: Extract Business Rules

Use Grep to find relevant backend source files and extract testable business rules:

Search locations:
- `internal/server/api/v1/*.go` — handler-level validation and business logic
- `internal/server/store/queries/*.sql` — data filtering, scoping, ordering rules
- `internal/hub/api/v1/*.go` — hub-specific rules
- Page components — frontend validation rules (Zod schemas, conditional rendering)

```json
{
  "business_rules": [
    {
      "id": "BR-001",
      "rule": "Only active frameworks appear in compliance dashboard",
      "source": "internal/server/api/v1/compliance.go",
      "test_method": "data_check",
      "test": "Verify all framework cards on /compliance have active=true"
    },
    {
      "id": "BR-002",
      "rule": "Overdue controls scoped to active frameworks only",
      "source": "internal/server/store/queries/compliance.sql",
      "test_method": "data_check",
      "test": "Every framework in overdue table must appear in active framework cards"
    }
  ]
}
```

Focus on rules that are testable from the UI:
- Filtering/scoping rules (what data appears where)
- Validation rules (required fields, format constraints)
- State machine rules (allowed transitions)
- Authorization rules (what actions are available per role)
- Calculation rules (scores, counts, aggregations)

### Step 6: Identify Shared Components

Scan layout directories and shared component imports across pages:

```json
{
  "shared_components": [
    {
      "name": "sidebar",
      "component": "AppSidebar",
      "file": "src/app/layout/AppSidebar.tsx",
      "appears_on": "all pages",
      "review_once": true,
      "states": ["expanded", "collapsed", "mobile", "active-item-highlighted"]
    },
    {
      "name": "topbar",
      "component": "TopBar",
      "file": "src/app/layout/TopBar.tsx",
      "appears_on": "all pages",
      "review_once": true
    }
  ]
}
```

For each shared component:
- Note all visual states it can be in.
- Set `review_once: true` — these only need one thorough review, then spot-checks on other pages.
- Include any app-level providers or wrappers (auth context, theme, error boundaries).

### Step 7: Write Product Map

Assemble all sections and write to `.omniprod/product-map.json`:

```json
{
  "generated_at": "<ISO 8601 timestamp>",
  "product": "PatchIQ",
  "apps": ["web", "web-hub", "web-agent"],
  "pages": [ ... ],
  "entity_graph": { ... },
  "flows": [ ... ],
  "business_rules": [ ... ],
  "shared_components": [ ... ],
  "review_plan": {
    "deep_review_pages": ["/compliance", "/endpoints", "/deployments", "/workflows", "/dashboard"],
    "flow_reviews": ["flow-compliance-assess", "flow-deploy-patch"],
    "smoke_all": true,
    "estimated_total_time": "~4 hours",
    "estimated_captures": 500
  }
}
```

The `review_plan` section summarizes the recommended review strategy:
- `deep_review_pages`: All pages with tier `deep-review`
- `flow_reviews`: All flow IDs
- `smoke_all`: Whether to smoke-test all remaining pages
- `estimated_total_time`: Rough estimate based on page count and tiers
- `estimated_captures`: Estimated screenshot count for full review

Ensure the output is valid JSON. Use `Bash` to validate with `python3 -m json.tool` after writing.

### Step 8: Print Summary

```
=== Product Map Generated ===

Pages: {web_count} (web) + {hub_count} (web-hub) + {agent_count} (web-agent) = {total} total
  Deep review: {n} pages
  Flow review: {n} pages
  Smoke only: {n} pages

Entities: {n} types tracked
  With detail pages: {n}
  Cross-page: endpoint ({n} pages), framework ({n} pages), ...

User Flows: {n} defined
  Critical: {n}
  Important: {n}
  Nice-to-have: {n}

Business Rules: {n} extracted

Shared Components: {n} (review once, apply everywhere)

Product map saved to: .omniprod/product-map.json
```

## Performance Notes

- Dispatch sub-agents with `model: "sonnet"` for parallel per-app analysis if mapping all three apps.
- This is a one-time operation per product, or when routes change significantly. Use `--refresh` to regenerate.
- If route parsing fails or is ambiguous, mark entries with `"uncertain": true` and note the reason.
- Prefer Grep over reading entire files — target specific patterns (`createBrowserRouter`, `Route`, `path:`, hook names).
