# Design: Replace Groups with Key=Value Tags + Selector DSL

**Status:** Done (2026-04-11). Shipped across PRs #367 (phase 1 scaffold),
#368 (phase 2+3 destructive migration + engine rewire + hardening), and the
phase 4+5 follow-up (frontend migration, RBAC cleanup, config.ScopeGroup
retirement).
**Author:** Heramb (with Claude)
**Tracking issue:** TBD
**Worktree:** `feat/tags-replace-groups`

---

## 1. Problem

Groups and tags coexist as two competing endpoint-targeting primitives in Patch Manager:

- **Groups** (`endpoint_groups`, `policy_groups`, `endpoint_group_members`) are static, curated collections. They are the *only* way policies, deployments, workflow filters, CVE filters, and compliance scopes target endpoints today. Engines call `ListEndpointsForPolicyGroups` directly.
- **Tags** (`tags`, `endpoint_tags`, `tag_rules`) are flat labels, currently name-only after migration 047 rolled back key=value. They have a handler, rules-based auto-assignment, a UI, and events — but no engine reads them. They are effectively decorative.

The platform has two nouns for the same concept, and only one of them is load-bearing. This has already produced confusion (the 037 → 039 → 047 migration churn) and blocks the richer targeting stories clients need (cloud-native key=value attributes, compliance scoping, wave-based deployments, auto-assignment driven by selectors rather than names).

**Decision:** Delete groups entirely. Promote tags to key=value. Add a JSONB selector DSL that all engines consume through a single abstraction. No backward compatibility — we wipe the `endpoint_groups*` and `policy_groups` tables and users re-tag.

## 2. Goals

1. **One targeting primitive.** `(key, value)` tags + JSONB selector expressions. Every policy, deployment, workflow, and compliance scope references a selector, not a group.
2. **Cloud-native shape.** Keys are first-class (`env`, `os`, `region`, `criticality`). UI and auto-rules are built around keys, not free-form strings.
3. **Unified evaluator.** One `internal/server/targeting` package owns the AST, parser, validator, and Postgres compiler. Engines call `Resolve(ctx, selector) → []EndpointID`.
4. **Full DSL from day one.** `eq`, `in`, `and`, `or`, `not`, `exists` (key exists regardless of value), `key_eq_any`. Shipping half the operators means shipping it twice.
5. **No dead code.** At the end of the migration: zero references to `endpoint_groups`, `policy_groups`, `endpoint_group_members`, `GroupIDs`, or the `groups:*` RBAC resource anywhere in the tree.

## 3. Non-Goals

- **Data migration.** Existing groups in dev/staging are wiped. Production has no real user data yet (pre-client-deploy). Users re-tag using the new UI.
- **Tag expression *names*.** Selectors are inline JSONB on the parent resource (`policies.target_selector`). We do *not* introduce a separate `tag_expressions` table with named reusable selectors in v1 — YAGNI until someone asks.
- **Agent-side changes.** The agent already exposes a generic `tags map<string,string>` in `common.proto`. No protobuf changes needed.
- **RBAC redesign.** We remove the `groups` resource and add `tags` + `tag_rules`. Role seeding updates only.

## 4. Schema

### 4.1 `tags` — promote to key=value

Migration adds `key` and `value` columns, replaces the `name`-based unique constraint, and enforces `(tenant_id, key, value)` uniqueness. An endpoint may not carry two values for a single-valued key (enforced at the application layer via `tag_keys.exclusive`, not the DB — too rigid to enforce in a CHECK).

```sql
ALTER TABLE tags
  ADD COLUMN key   TEXT NOT NULL DEFAULT '',
  ADD COLUMN value TEXT NOT NULL DEFAULT '';

-- Backfill is a no-op: we're wiping all existing tag data in the same migration
-- (see migration 048 below — DELETE FROM endpoint_tags; DELETE FROM tags;).

ALTER TABLE tags ALTER COLUMN key   DROP DEFAULT;
ALTER TABLE tags ALTER COLUMN value DROP DEFAULT;
ALTER TABLE tags DROP COLUMN name;

CREATE UNIQUE INDEX tags_tenant_key_value_uq
  ON tags (tenant_id, lower(key), lower(value));
CREATE INDEX tags_tenant_key_idx
  ON tags (tenant_id, lower(key));
```

**`tag_keys` (new)** — metadata/catalog for known keys:

```sql
CREATE TABLE tag_keys (
  tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  key         TEXT NOT NULL,
  description TEXT,
  exclusive   BOOLEAN NOT NULL DEFAULT false,  -- enforce single value per endpoint
  value_type  TEXT NOT NULL DEFAULT 'string',  -- string|enum — enum reserved for future
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, lower(key))
);
ALTER TABLE tag_keys ENABLE ROW LEVEL SECURITY;
ALTER TABLE tag_keys FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON tag_keys
  USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
```

The `exclusive` flag is what makes `env=prod` work the way users expect: assigning `env=staging` to the same endpoint *replaces* `env=prod` rather than adding a second value. This is enforced in the `AssignTag` handler, not the DB.

### 4.2 `endpoint_tags` — add compound index

```sql
CREATE INDEX endpoint_tags_tenant_tag_idx
  ON endpoint_tags (tenant_id, tag_id);
-- existing idx_endpoint_tags_tag stays; this one covers the selector-resolution path
```

### 4.3 `policy_tag_selectors` (new)

Replaces `policy_groups`. One selector per policy, stored as JSONB (AST shape in §5).

```sql
CREATE TABLE policy_tag_selectors (
  policy_id  UUID PRIMARY KEY REFERENCES policies(id) ON DELETE CASCADE,
  tenant_id  UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  expression JSONB NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX policy_tag_selectors_tenant_idx ON policy_tag_selectors (tenant_id);
CREATE INDEX policy_tag_selectors_expr_gin   ON policy_tag_selectors USING gin (expression);
ALTER TABLE policy_tag_selectors ENABLE ROW LEVEL SECURITY;
ALTER TABLE policy_tag_selectors FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON policy_tag_selectors
  USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
```

**Why a separate table and not a JSONB column on `policies`?** Keeps `policies` untouched, makes selector updates independent of policy metadata updates (different event topics), and lets us soft-delete policy changes without touching the selector history.

### 4.4 Drop everything group-shaped

Single migration, destructive:

```sql
DROP TABLE IF EXISTS policy_groups           CASCADE;
DROP TABLE IF EXISTS endpoint_group_members  CASCADE;
DROP TABLE IF EXISTS endpoint_groups         CASCADE;

-- Wipe stale tag data before the column rename
DELETE FROM endpoint_tags;
DELETE FROM tags;

-- Tighten config_overrides and compliance_scores
ALTER TABLE config_overrides DROP CONSTRAINT IF EXISTS config_overrides_scope_type_check;
ALTER TABLE config_overrides ADD CONSTRAINT config_overrides_scope_type_check
  CHECK (scope_type IN ('tenant', 'tag', 'endpoint'));
ALTER TABLE compliance_scores DROP CONSTRAINT IF EXISTS compliance_scores_scope_type_check;
ALTER TABLE compliance_scores ADD CONSTRAINT compliance_scores_scope_type_check
  CHECK (scope_type IN ('tenant', 'tag', 'endpoint'));
```

### 4.5 Migration file layout

| File | Purpose |
|---|---|
| `048_tags_key_value_and_drop_groups.sql` | All of §4.1–§4.4 in one atomic migration (up + down) |

We use one migration, not several, because the steps are not independently valid — dropping groups while tag schema is still flat would leave the engines with no targeting primitive at all.

## 5. Selector DSL

### 5.1 AST (JSONB shape)

```jsonc
// Leaf: equality on a single key
{ "op": "eq", "key": "env", "value": "prod" }

// Leaf: any of several values
{ "op": "in", "key": "os", "values": ["ubuntu", "debian"] }

// Leaf: key exists (any value)
{ "op": "exists", "key": "owner" }

// Composite
{ "op": "and", "args": [ <selector>, <selector>, ... ] }
{ "op": "or",  "args": [ <selector>, <selector>, ... ] }
{ "op": "not", "arg":  <selector> }
```

**Design rules:**
- No shorthand. Always `{op, ...}`. Easier to parse, easier to lint, easier to extend.
- `args` is always an array (even for `and`/`or` with a single arg — which is valid and normalized to just that arg by the optimizer).
- `value` / `values` are always strings. Numeric comparisons and ranges are explicitly out of scope for v1.
- Empty `and`/`or` are rejected by the validator. An empty selector (match everything) is expressed by omitting the selector entirely on the parent resource (= "all endpoints in tenant").
- Depth limit: 8. Prevents pathological expressions from DoS'ing the compiler.

### 5.2 Go AST

```go
// internal/server/targeting/ast.go
package targeting

type Op string
const (
    OpEq     Op = "eq"
    OpIn     Op = "in"
    OpExists Op = "exists"
    OpAnd    Op = "and"
    OpOr     Op = "or"
    OpNot    Op = "not"
)

type Selector struct {
    Op     Op          `json:"op"`
    Key    string      `json:"key,omitempty"`
    Value  string      `json:"value,omitempty"`
    Values []string    `json:"values,omitempty"`
    Args   []Selector  `json:"args,omitempty"`
    Arg    *Selector   `json:"arg,omitempty"`
}
```

Exactly one field set per op. Validator enforces. Unmarshalling is straightforward because each op has a disjoint set of required fields.

### 5.3 Postgres compilation

Engines need `SELECT endpoint_id FROM endpoints WHERE <selector>`. The compiler translates the AST into a single SQL fragment using `EXISTS` subqueries over `endpoint_tags JOIN tags`. Every leaf becomes one `EXISTS`; composites wrap them in `AND`/`OR`/`NOT`.

```sql
-- eq
EXISTS (SELECT 1 FROM endpoint_tags et JOIN tags t ON t.id = et.tag_id
        WHERE et.endpoint_id = e.id
          AND lower(t.key) = lower($1) AND lower(t.value) = lower($2))

-- in
EXISTS (SELECT 1 FROM endpoint_tags et JOIN tags t ON t.id = et.tag_id
        WHERE et.endpoint_id = e.id
          AND lower(t.key) = lower($1) AND lower(t.value) = ANY($2))

-- exists
EXISTS (SELECT 1 FROM endpoint_tags et JOIN tags t ON t.id = et.tag_id
        WHERE et.endpoint_id = e.id AND lower(t.key) = lower($1))
```

Placeholders are numbered by the compiler and bound via pgx. **No string interpolation of user data ever.** The compiler appends to a `[]any` args slice as it walks the AST.

### 5.4 Package layout

```
internal/server/targeting/
├── ast.go          # Selector struct, Op constants
├── validate.go     # Validate(sel) — shape rules, depth limit, key/value charset
├── compile.go      # Compile(sel) → (sqlFrag string, args []any, err error)
├── resolve.go      # Resolver — Resolve(ctx, tenantID, sel) → []EndpointID
├── optimize.go     # Fold single-arg and/or, dedup leaves, flatten nested same-op
├── ast_test.go
├── validate_test.go
├── compile_test.go   # table-driven: AST → expected SQL fragment + args
├── resolve_test.go   # integration, hits real Postgres via testcontainers
└── optimize_test.go
```

Engines depend only on `Resolver`, never on `compile.go` directly. This is the abstraction boundary.

## 6. Engine Integration

Three current call sites for `ListEndpointsForPolicyGroups`:

| File | Current call | Replacement |
|---|---|---|
| `internal/server/policy/datasource.go:68` | `q.ListEndpointsForPolicyGroups` | `resolver.ResolveForPolicy(ctx, policyID)` |
| `internal/server/deployment/evaluator.go:57` | `q.ListEndpointsForPolicyGroups` | `resolver.ResolveForPolicy(ctx, policyID)` |
| `internal/server/compliance/*` | indirect via policy scope | same |

`ResolveForPolicy` loads the selector from `policy_tag_selectors`, runs it through the compiler, and executes. One round-trip. The existing query interfaces (`EvaluatorQuerier`, `EvalQuerier`) gain a `GetPolicyTagSelector` method and lose the `ListEndpointsForPolicyGroups` method.

**Workflow filters** (`internal/server/workflow/model.go:111`) currently have a `GroupIDs []string` field on `FilterConfig`. Replaced with `Selector *targeting.Selector` — same file, same struct, different field. Workflow executor calls `resolver.Resolve` directly when the filter is evaluated.

**CVE response's `GroupNames`** (`internal/server/api/v1/cves.go:189,308-321`) — this was leaking group names into CVE listings for "which groups own endpoints affected by this CVE". Replaced with the top-K tags (by count) on affected endpoints. Net-different feature, same UI slot. Spec'd in Phase 4.

## 7. API Surface

### 7.1 Deleted

- `DELETE` the entire `/api/v1/groups` route tree (6 endpoints).
- Delete `GroupIDs` / `GroupNames` from every request/response struct.
- Delete `groups:*` from RBAC seed.

### 7.2 New/updated

**`/api/v1/tags`** — breaking changes to existing handler:
- `POST /api/v1/tags` — body now `{ key, value, description }`. 409 if `(tenant, key, value)` exists.
- `GET /api/v1/tags?key=env` — filter by key.
- `GET /api/v1/tags/keys` — list distinct keys with counts (drives UI key autocomplete).
- Assignment: `POST /api/v1/tags/{id}/assign` with body `{ endpoint_ids: [...] }`. If the tag's key is marked `exclusive` in `tag_keys`, the handler first removes any other tag with the same key from each target endpoint.

**`/api/v1/tag-keys`** — new CRUD:
- `GET /api/v1/tag-keys` — list all known keys.
- `POST /api/v1/tag-keys` — register a key with `{ key, description, exclusive }`. Implicitly created when a tag is created with a new key, with `exclusive=false`.
- `PATCH /api/v1/tag-keys/{key}` — toggle `exclusive`, update description.

**`/api/v1/policies`** — selector field on create/update:
- Request: `{ ..., target_selector: <AST> | null }`. Null means "all endpoints in tenant".
- Response: `target_selector` echoed back.
- Handler validates the selector via `targeting.Validate` before writing.
- Old `group_ids` field is **removed**, not deprecated.

**Selector validation endpoint:**
- `POST /api/v1/tags/selectors/validate` — takes a selector AST, returns `{ valid: bool, error: string?, matched_count: int? }`. Used by the UI builder to live-preview match counts as the user builds a selector. Matched count uses `resolver.Count(ctx, sel)`.

### 7.3 OpenAPI

`api/server.yaml` changes are significant: delete the Groups tag, delete 6 paths, delete 4 schemas (Group, CreateGroupRequest, UpdateGroupRequest, GroupMembers), add 6 paths (tag-keys CRUD, selector validate), add 4 schemas (Tag — now with key/value, TagKey, TagSelectorExpression, TagSelectorLeaf). Generated `web/src/api/types.ts` is regenerated (not hand-edited).

## 8. Events

Delete:
- `GroupCreated`, `GroupUpdated`, `GroupDeleted`, `GroupMembersUpdated` (`events/topics.go:21-24, 190-193`)

Add:
- `TagKeyCreated`, `TagKeyUpdated`, `TagKeyDeleted`
- `PolicyTargetSelectorUpdated` (separate from `PolicyUpdated` because the engines care about this specifically)

Existing tag events (`TagCreated`, `TagUpdated`, `TagDeleted`, `EndpointTagged`) stay.

## 9. Frontend

### 9.1 Deleted files

```
web/src/pages/groups/GroupsPage.tsx
web/src/pages/groups/CreateGroupDialog.tsx
web/src/pages/groups/EditGroupDialog.tsx
web/src/pages/policies/tabs/GroupsEndpointsTab.tsx
web/src/pages/patches/AddToGroupDialog.tsx
web/src/__tests__/pages/groups/*
web/src/api/hooks/useGroups.ts
```

Plus every import of the above, and every `group`/`groups` field from request/response types (regenerated from OpenAPI).

### 9.2 New components

- **`TagSelectorBuilder`** (`web/src/components/TagSelectorBuilder.tsx`) — visual AST editor. Tree view with `+ AND`, `+ OR`, `+ NOT`, `+ leaf` buttons. Leaf row: key autocomplete (from `GET /api/v1/tag-keys`) → op select (`eq`, `in`, `exists`) → value input with autocomplete (from `GET /api/v1/tags?key=X`). Live match count via `POST /api/v1/tags/selectors/validate`. Serializes to the AST JSON.
- **`TagKeyManager`** (`web/src/pages/settings/tags/TagKeyManager.tsx`) — manage keys (add, toggle exclusive, delete).
- **`useTagKeys`**, **`useValidateSelector`** — new hooks.

### 9.3 Updated components

- `PolicyForm.tsx` — drops `GroupsEndpointsTab`, adds `TagSelectorBuilder` on the "Targeting" tab.
- `TagsPage.tsx` — now displays tags grouped by key.
- `PoliciesPage.tsx` list view — shows a compact summary of the selector instead of group names.
- `CVE filters`, `Workflow filters` — same swap.

## 10. Rollout Phases

Each phase is an independently-mergeable PR to `dev-a`. Parallel devs are unaffected until their code hits the deleted files — at which point they get a clear failure and can rebase.

**Phase 1 — Backend foundation (schema + evaluator).** No API/UI changes. Lands migration 048, new `targeting` package, new `policy_tag_selectors` queries. Policies temporarily have no usable targeting (groups are gone, selector wiring not yet in handler). Integration tests for the package use testcontainers. *Branch: this worktree. First PR.*

**Phase 2 — API layer.** Deletes groups handler, adds selector fields on policy handlers, adds tag-key handlers, regenerates OpenAPI. Backend tests pass end-to-end. Frontend is broken (intentionally — OpenAPI types diverge). *Second PR off `dev-a`.*

**Phase 3 — Engine wiring.** `policy/datasource.go`, `deployment/evaluator.go`, `compliance`, `workflow/model.go` all swap to the resolver. Full backend integration tests green. *Can be bundled with Phase 2 if small enough.*

**Phase 4 — Frontend.** Deletes group pages, adds `TagSelectorBuilder`, updates `PolicyForm`, regenerates types. Vitest tests updated. *Third PR.*

**Phase 5 — Cleanup.** RBAC seed, ADRs, `dev-mocks.ts`, any lingering references. `grep -r "group" internal/server web` comes back clean (modulo unrelated matches like "group by" SQL and "grouping" in unrelated contexts, which the reviewer verifies). *Fourth PR, small.*

## 11. Risks

1. **Parallel-dev conflict on `policies.go`.** High-touch file. Mitigation: land Phase 2 fast once Phase 1 is approved. Rebase pain is proportional to delay.
2. **RLS on `policy_tag_selectors`.** Every new table is a chance to forget `FORCE ROW LEVEL SECURITY`. Covered by integration test that attempts cross-tenant read and asserts zero rows.
3. **Selector compiler correctness.** Wrong SQL = wrong endpoints deployed to = production incident. Mitigation: table-driven compiler tests cover every op + 2-level composites; fuzz test with randomly generated ASTs that compares compiler output against a naive in-memory evaluator.
4. **Compliance scoring migration.** `compliance_scores.scope_type` currently allows `'group'`. Wiping the CHECK constraint is fine if no rows reference it — migration first `DELETE`s any `scope_type = 'group'` rows, then alters the constraint. Documented in migration.
5. **Event consumers that subscribed to `Group*`.** If any exist outside what the audit found, they break silently. Mitigation: grep `events.Group` before merging Phase 2.
6. **"All endpoints" default.** Null selector = all endpoints in tenant. A bug that nulls the selector silently would deploy to every endpoint. Mitigation: handler logs a warning on null-selector writes, and the policy list view renders a red badge for policies with no selector.

## 12. Open Questions

- **Do we need selector versioning / audit?** `policy_tag_selectors` has `updated_at` but no history. If a deployment goes wrong and we want to know "what selector was in effect at wave-dispatch time," we can't answer. Probably fine for v1 — deployment events already capture the resolved endpoint set at the time they run.
- **Tag value length.** Currently unbounded `TEXT`. Client-side UI should cap at 128 chars to keep the builder sane. Enforced in validator, not DB.

## 13. Acceptance

Phase 1 complete when:
- Migration 048 applies cleanly up and down on a fresh DB.
- `go test ./internal/server/targeting/... -race` green with ≥90% coverage.
- `go test ./internal/server/store/...` green (sqlc regenerated).
- `make lint` green.
- `grep -rn "endpoint_groups\|policy_groups\|endpoint_group_members" internal/server/store` returns nothing.

Phase 5 complete when:
- `grep -rn "GroupID\|GroupIDs\|endpoint_groups\|policy_groups\|groups:read\|groups:create\|groups:update\|groups:delete" internal/server web api` returns nothing (modulo comments in ADR history).
- Full CI green.
- Client demo environment re-seeded and walked end-to-end: create tag keys, tag endpoints, build a selector, create a policy, run a deployment, verify endpoint set.
