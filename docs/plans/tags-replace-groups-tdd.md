# TDD Plan: Tags Replace Groups

Companion to `tags-replace-groups.md`. Granular, test-first task list. Each task is one commit-sized unit. File paths are relative to repo root.

Legend: 🧪 = write test first, ⚙️ = implementation, 🔗 = wiring only (no new logic), 🗑️ = deletion.

---

## Phase 1 — Backend Foundation (this worktree, PR #1)

### 1.1 Migration 048

- [ ] 🧪 `internal/server/store/migrations_test.go` — add case asserting 048 up+down on fresh DB, post-up schema has `tag_keys`, `policy_tag_selectors`, `tags.key`, `tags.value`; `endpoint_groups`, `policy_groups`, `endpoint_group_members` do not exist.
- [ ] ⚙️ `internal/server/store/migrations/048_tags_key_value_and_drop_groups.sql` — the single destructive migration (schema §4).
- [ ] ⚙️ Verify `down` is reversible enough to not wedge dev machines (recreate empty group tables, drop new columns). Down is best-effort — data loss is acknowledged.
- [ ] 🔗 Run `make migrate` on local dev DB. Confirm.

### 1.2 sqlc queries

- [ ] 🗑️ Delete `internal/server/store/queries/groups.sql`.
- [ ] 🗑️ Delete the group-related queries from `internal/server/store/queries/policies.sql` (lines 67-95, 113-121, 187-193 per audit). Hand-edit carefully.
- [ ] ⚙️ `internal/server/store/queries/tags.sql` — rewrite for key=value:
  - `CreateTag(tenant_id, key, value, description)`
  - `GetTagByID`, `GetTagByKeyValue(tenant_id, key, value)`
  - `ListTags(tenant_id, key_filter)` with `endpoint_count` subquery
  - `ListTagKeys(tenant_id)` — distinct keys with counts
  - `UpdateTag` (description only — key/value are immutable once created)
  - `DeleteTag`
  - `AssignTagToEndpoint`, `RemoveTagFromEndpoint`
  - `RemoveTagsByKeyFromEndpoint(tenant_id, endpoint_id, key)` — for exclusive-key semantics
  - `ListTagsForEndpoint`, `ListEndpointsByTag`
- [ ] ⚙️ `internal/server/store/queries/tag_keys.sql` — new file:
  - `UpsertTagKey`, `GetTagKey`, `ListTagKeys`, `UpdateTagKey`, `DeleteTagKey`, `IsKeyExclusive`
- [ ] ⚙️ `internal/server/store/queries/policy_tag_selectors.sql` — new file:
  - `UpsertPolicyTagSelector(policy_id, tenant_id, expression jsonb)`
  - `GetPolicyTagSelector(policy_id)`
  - `DeletePolicyTagSelector(policy_id)`
  - `ListPolicyTagSelectorsForTenant` (used by admin tooling / tests)
- [ ] 🔗 `make sqlc` — regenerate. Commit `sqlcgen/` changes.
- [ ] 🧪 `internal/server/store/tags_test.go` — integration test (testcontainers) for: create key+value tag, assign to endpoint, list by key, exclusive-key replacement, RLS cross-tenant isolation.

### 1.3 Targeting package — AST & validation

- [ ] ⚙️ `internal/server/targeting/ast.go` — `Op` consts, `Selector` struct, custom `UnmarshalJSON` that rejects unknown ops.
- [ ] 🧪 `internal/server/targeting/ast_test.go` — round-trip every op through JSON; reject unknown op; reject wrong field combinations (e.g., `eq` with `values`).
- [ ] ⚙️ `internal/server/targeting/validate.go` — `Validate(sel Selector) error`:
  - `eq` requires non-empty `key`, `value`; forbids `values`, `args`, `arg`.
  - `in` requires non-empty `key`, at least one non-empty `values`; forbids `value`, `args`, `arg`.
  - `exists` requires non-empty `key`; forbids everything else.
  - `and`/`or` requires `len(args) >= 1`; recurses.
  - `not` requires non-nil `arg`; recurses.
  - Depth limit 8; returns `ErrDepthExceeded`.
  - Key/value charset: printable ASCII, length ≤128.
- [ ] 🧪 `internal/server/targeting/validate_test.go` — table-driven: one row per failure mode, plus happy-path compositions.

### 1.4 Targeting package — optimizer

- [ ] ⚙️ `internal/server/targeting/optimize.go` — `Optimize(sel Selector) Selector`:
  - Flatten nested same-op (`and(and(a,b), c)` → `and(a,b,c)`).
  - Unwrap single-arg `and`/`or` to their child.
  - Double-negation `not(not(x))` → `x`.
  - Dedup identical leaves under `and`/`or`.
- [ ] 🧪 `internal/server/targeting/optimize_test.go` — each rewrite rule has its own test case.

### 1.5 Targeting package — compiler

- [ ] ⚙️ `internal/server/targeting/compile.go` — `Compile(sel Selector) (sql string, args []any, err error)`:
  - Assumes caller has an alias `e` for the `endpoints` table (documented).
  - Walks the AST, appends `$N` placeholders, builds args slice.
  - Returns a SQL fragment suitable for dropping into a `WHERE` clause.
- [ ] 🧪 `internal/server/targeting/compile_test.go` — table-driven:
  - `eq` → `EXISTS (...)` with 2 args.
  - `in` → `EXISTS (...)` with `ANY($2)` and `[]string` arg.
  - `exists` → `EXISTS (...)` with 1 arg.
  - `and(eq, eq)` → `(frag1 AND frag2)` with 4 args, correct numbering.
  - `or` → `(...)` with `OR`.
  - `not` → `(NOT frag)`.
  - Nested 3-deep composite — verifies arg numbering across recursion.
  - Empty selector → error.

### 1.6 Targeting package — resolver

- [ ] ⚙️ `internal/server/targeting/resolver.go` — `Resolver` type with `pgxpool.Pool` dep:
  - `Resolve(ctx, tenantID, sel) ([]uuid.UUID, error)` — runs `SELECT id FROM endpoints e WHERE <compiled>`.
  - `Count(ctx, tenantID, sel) (int, error)` — `SELECT count(*)`.
  - `ResolveForPolicy(ctx, tenantID, policyID) ([]uuid.UUID, error)` — loads selector from `policy_tag_selectors`, then `Resolve`. Null selector → all endpoints in tenant.
  - All methods set `SET LOCAL app.current_tenant_id` inside their txn.
- [ ] 🧪 `internal/server/targeting/resolver_test.go` (testcontainers) — seeds real endpoints + tags, runs representative selectors, asserts resolved IDs. Covers: single eq, `or` of two eqs, `and` with `not`, exclusive-key replacement behavior, null-selector-returns-all, cross-tenant isolation.
- [ ] 🧪 Fuzz test (`compile_fuzz_test.go`) — generates random ASTs (bounded depth), compares resolver output against an in-memory reference evaluator that walks the AST against a known tag map. Any divergence = compiler bug.

### 1.7 Package exports

- [ ] ⚙️ `internal/server/targeting/doc.go` — package doc comment stating: "Targeting is the single abstraction for endpoint selection. Engines must depend only on Resolver. Direct SQL construction is forbidden."

### Phase 1 acceptance gate

- `go test ./internal/server/targeting/... -race` green
- `go test ./internal/server/store/...` green
- `make lint` green
- `grep -rn "endpoint_groups\|policy_groups\|endpoint_group_members" internal/server/store` empty
- `/review-pr all parallel` — Critical/Important issues fixed
- `/commit-push-pr`

---

## Phase 2 — API Layer (PR #2)

### 2.1 Delete groups handler

- [ ] 🗑️ `internal/server/api/v1/groups.go` — delete file.
- [ ] 🗑️ `internal/server/api/v1/groups_test.go` — delete file.
- [ ] 🗑️ `internal/server/api/router.go:106` — remove `NewGroupHandler` instantiation.
- [ ] 🗑️ `internal/server/api/router.go:183-189` — remove `/groups` route block.

### 2.2 Tags handler — key=value rewrite

- [ ] 🧪 `internal/server/api/v1/tags_test.go` — tests for new behavior first:
  - POST `{key, value, description}` → 201; missing key → 400.
  - POST duplicate `(key,value)` → 409.
  - GET `?key=env` → filtered list.
  - GET `/tags/keys` → distinct keys with counts.
  - POST `/{id}/assign` with exclusive key removes prior value on same endpoints.
- [ ] ⚙️ `internal/server/api/v1/tags.go` — rewrite handler methods. `TagQuerier` interface gains `IsKeyExclusive`, `RemoveTagsByKeyFromEndpoint`, `ListTagKeys`.

### 2.3 Tag-keys handler

- [ ] 🧪 `internal/server/api/v1/tag_keys_test.go` — CRUD tests.
- [ ] ⚙️ `internal/server/api/v1/tag_keys.go` — new handler.
- [ ] 🔗 Wire into router under `/api/v1/tag-keys`.
- [ ] 🔗 Permissions: add `tag_keys:read|create|update|delete` to RBAC seed; update `internal/server/auth/*` permission tests.

### 2.4 Selector validation endpoint

- [ ] 🧪 Test: valid selector returns `{valid:true, matched_count:N}`; invalid returns `{valid:false, error:"..."}`.
- [ ] ⚙️ `internal/server/api/v1/tag_selectors.go` — `POST /api/v1/tags/selectors/validate` handler. Depends on `targeting.Validate` and `Resolver.Count`.

### 2.5 Policies handler — remove groups, add selector

- [ ] 🧪 `internal/server/api/v1/policies_test.go` — update existing tests:
  - Create policy with `target_selector` → persisted to `policy_tag_selectors`.
  - Update policy with new selector → upserts.
  - Response echoes `target_selector`.
  - Create with `group_ids` field → 400 (field no longer recognized).
- [ ] ⚙️ `internal/server/api/v1/policies.go`:
  - Delete `GroupIDs`/`GroupNames` from `createPolicyRequest`, `policyResponse`.
  - Delete `ListGroupsForPolicy`, `AddGroupToPolicy`, `RemoveAllGroupsFromPolicy`, `ListGroupsForPolicies`, `ListEndpointsForPolicyGroups` from `PolicyQuerier` interface (lines 22-44 per audit).
  - Add `GetPolicyTagSelector`, `UpsertPolicyTagSelector`, `DeletePolicyTagSelector`.
  - `Create` (line 470-574): validate & persist selector.
  - `Update` (line 820-930): upsert selector.
  - `List` (line 650-689): batch-load selectors instead of group names. Show a short text summary (`resolver.Describe(sel)` — new method in Phase 1.7 if needed, or inline here).
  - `toPolicyResponse` / `toPolicyResponseWithStats`: new signature.

### 2.6 Other handlers touched by groups

- [ ] ⚙️ `internal/server/api/v1/cves.go:189, 308-321` — replace `GroupNames` field with `TopTags []TagSummary`. New test: CVE response includes top-3 tags by endpoint count.
- [ ] ⚙️ `internal/server/api/v1/dashboard_test.go`, `deployments_test.go`, `notifications_test.go`, `workflow_executions_test.go`, `workflows_test.go`, `endpoints_test.go` — remove any `group_*` fixtures; replace with tag fixtures. Compilation-only change for many.

### 2.7 OpenAPI

- [ ] ⚙️ `api/server.yaml`:
  - Delete `Groups` tag (line 20-21).
  - Delete Group schemas (210-275).
  - Delete 6 group paths (3185-3365).
  - Delete `group_ids`/`group_names` from Policy, Deployment, FilterConfig, Workflow, CVEFilter schemas (261, 287, 348, 390, 405, 477, 566-569).
  - Add `Tag` (with key, value), `TagKey`, `TagSelector`, `PolicySelectorValidateRequest`, `PolicySelectorValidateResponse` schemas.
  - Add paths: `/tags/keys` (list), `/tag-keys` CRUD, `/tags/selectors/validate`.
  - Update `/policies` request/response to use `target_selector`.
- [ ] 🔗 `make api-client` — validate spec parses.

### 2.8 Events

- [ ] ⚙️ `internal/server/events/topics.go` — delete lines 21-24 and 190-193. Add `TagKeyCreated|Updated|Deleted`, `PolicyTargetSelectorUpdated`.
- [ ] 🔗 Emit new events from tag-keys handler and policies handler (on selector upsert).

### Phase 2 acceptance gate

- `go test ./internal/server/...` green
- `grep -rn "GroupID\|GroupIDs\|GroupNames\|AddGroupToPolicy\|ListEndpointsForPolicyGroups" internal/server` empty
- `/review-pr all parallel`
- `/commit-push-pr`

---

## Phase 3 — Engine Wiring (PR #3, can bundle with Phase 2)

### 3.1 Policy evaluator

- [ ] 🧪 `internal/server/policy/datasource_test.go` — update: `ListEndpointsForPolicy` uses resolver; integration test seeds policy_tag_selectors.
- [ ] ⚙️ `internal/server/policy/datasource.go`:
  - Line 14-20: `EvaluatorQuerier` loses `ListEndpointsForPolicyGroups`, gains nothing (datasource now holds a `*targeting.Resolver`).
  - Line 57-85: `ListEndpointsForPolicy` body becomes `return ds.resolver.ResolveForPolicy(ctx, tenantID, policyID)`.
- [ ] ⚙️ Constructor wiring in `cmd/server/main.go` — inject resolver.

### 3.2 Deployment evaluator

- [ ] 🧪 `internal/server/deployment/evaluator_test.go` — update.
- [ ] ⚙️ `internal/server/deployment/evaluator.go` lines 19-24, 47-90 — same swap.
- [ ] ⚙️ `internal/server/deployment/integration_test.go` — update fixtures.

### 3.3 Workflow filters

- [ ] 🧪 `internal/server/workflow/model_test.go:111` — test now uses `Selector` field.
- [ ] ⚙️ `internal/server/workflow/model.go:111` — `FilterConfig.GroupIDs` → `FilterConfig.Selector *targeting.Selector`.
- [ ] ⚙️ `internal/server/workflow/model.go:119` — validation calls `targeting.Validate`.
- [ ] ⚙️ Workflow executor — wherever it applied the filter, swap to `resolver.Resolve(ctx, tenantID, *filter.Selector)`.

### 3.4 Compliance scope

- [ ] ⚙️ `internal/server/compliance/scorer.go` (and siblings) — audit references to `scope_type = 'group'`. Replace with `'tag'` where a compliance framework scopes to a tag selector. `compliance_scores.scope_ref` now stores a tag id or a selector id (decision: inline selector JSONB in `scope_config` column — new column if needed, but probably existing `scope_config` JSONB works).
- [ ] 🧪 `internal/server/compliance/scorer_test.go` — update fixtures.

### Phase 3 acceptance gate

- `go test ./internal/server/... -race` green
- Integration: create policy with selector, trigger deployment, verify wave dispatcher picks correct endpoints.
- `/review-pr`
- `/commit-push-pr`

---

## Phase 4 — Frontend (PR #4)

### 4.1 Delete group pages

- [ ] 🗑️ `web/src/pages/groups/` (GroupsPage, CreateGroupDialog, EditGroupDialog)
- [ ] 🗑️ `web/src/pages/policies/tabs/GroupsEndpointsTab.tsx`
- [ ] 🗑️ `web/src/pages/patches/AddToGroupDialog.tsx`
- [ ] 🗑️ `web/src/__tests__/pages/groups/*`
- [ ] 🗑️ `web/src/api/hooks/useGroups.ts`
- [ ] 🗑️ Routes in `web/src/app/routes.tsx` — remove `/groups`, `/groups/*`.
- [ ] 🗑️ Sidebar nav — remove Groups link.

### 4.2 Regenerate API types

- [ ] 🔗 `cd web && pnpm api:generate` (or project equivalent) — regenerates `web/src/api/types.ts` from the new OpenAPI. Commit changes.

### 4.3 TagSelectorBuilder

- [ ] 🧪 `web/src/__tests__/components/TagSelectorBuilder.test.tsx`:
  - Renders empty state with "Add condition" button.
  - Adding a leaf shows key autocomplete → value autocomplete.
  - Adding an AND group nests children.
  - Serialization emits valid AST JSON (match expected shape).
  - Invalid state disables submit.
  - Live match count calls `/api/v1/tags/selectors/validate` (mocked).
- [ ] ⚙️ `web/src/components/TagSelectorBuilder.tsx` — component.
- [ ] ⚙️ `web/src/api/hooks/useTagKeys.ts`, `useValidateSelector.ts` — new hooks.
- [ ] ⚙️ `web/src/api/hooks/useTags.ts` — update signature (key, value fields).

### 4.4 Policy form

- [ ] 🧪 `web/src/__tests__/pages/policies/PolicyForm.test.tsx` — update:
  - Has a "Targeting" tab with `TagSelectorBuilder`.
  - Submits `target_selector` in request body.
  - Loads existing selector on edit.
- [ ] ⚙️ `web/src/pages/policies/PolicyForm.tsx`:
  - Remove `useGroups` import.
  - Remove `group_ids` from form schema (zod).
  - Add `target_selector` field.
  - Replace `GroupsEndpointsTab` with `<TagSelectorBuilder />`.

### 4.5 Tag key management page

- [ ] 🧪 Tests for TagKeyManager.
- [ ] ⚙️ `web/src/pages/settings/tags/TagKeyManager.tsx` — add/edit/delete keys, toggle exclusive.
- [ ] 🔗 Add route `/settings/tags/keys`.

### 4.6 Tags page update

- [ ] ⚙️ `web/src/pages/tags/TagsPage.tsx` — display tags grouped by key; create form accepts key+value.
- [ ] ⚙️ Update `AssignTagsDialog` — filter by key.

### 4.7 CVE & workflow filter UIs

- [ ] ⚙️ Any CVE list filter that used `group_ids` → uses a mini `TagSelectorBuilder`.
- [ ] ⚙️ Workflow filter node config — same.

### 4.8 Dev mocks

- [ ] ⚙️ `web/src/dev-mocks.ts` — remove group fixtures, add tag-key + tag-with-key-value fixtures.

### Phase 4 acceptance gate

- `pnpm typecheck` green
- `pnpm test` green
- `pnpm lint` green
- Manual browser walk: create tag keys, tag endpoints, build selector, save policy, confirm match count matches reality.
- `/review-pr`
- `/commit-push-pr`

---

## Phase 5 — Cleanup (PR #5, small)

- [ ] 🗑️ `internal/server/auth/seed*.go` — remove `groups:read|create|update|delete` permission rows. Add `tags:*`, `tag_keys:*` if not already present.
- [ ] 🧪 `internal/server/auth/permission_test.go`, `evaluator_test.go`, `store_test.go` — update to new permission set.
- [ ] ⚙️ `docs/adr/` — add `026-replace-groups-with-tag-selectors.md` documenting the decision. Update `004-custom-rbac-model.md` and `022-postgresql-rls-multi-tenancy.md` to note groups are removed.
- [ ] ⚙️ `docs/blueprint/` — search for group mentions, update or delete as appropriate.
- [ ] ⚙️ `CLAUDE.md` — any tables/sections that list "groups" as a feature area.
- [ ] 🔗 Final grep sweep: `grep -rn "endpoint_groups\|policy_groups\|endpoint_group_members\|GroupID\|group_ids\|groups:read" internal web api docs/plans` returns nothing meaningful.
- [ ] 🔗 `make ci-full` green.
- [ ] 🔗 Re-seed client demo DB, walk the golden path end-to-end.
- [ ] `/review-pr`, `/commit-push-pr`.

---

## Cross-phase notes

- **Parallel-dev safety:** Phase 1 touches only `internal/server/targeting/` (new), `internal/server/store/migrations/` (append-only), `internal/server/store/queries/` (edits). Collision risk with other devs is low-to-medium. Phase 2 hits `policies.go` and `router.go` — land fast.
- **sqlc regeneration discipline:** Never hand-edit `sqlcgen/`. Always edit `queries/*.sql` then run `make sqlc`.
- **Test data:** `db_test.go` has group seed rows. Those need to go in Phase 1 sub-task 1.2 (before the compilation breaks).
- **Commit cadence:** Each task marker in this doc is a commit candidate. Bundle only when the intermediate state wouldn't compile.
