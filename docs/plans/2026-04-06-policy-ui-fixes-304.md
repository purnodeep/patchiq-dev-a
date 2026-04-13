# Policy UI/UX Fixes — Design Spec (#304)

> **Issue**: herambskanda/patchiq#304
> **Branch**: `fix/policy-ui-fixes`
> **Author**: Rishab
> **Date**: 2026-04-06

---

## Problem

The policy creation/edit flow has 6 targeted issues: no policy type concept, inconsistent group/tag targeting, terse mode descriptions, no timezone/calendar support in scheduling, weak error handling, and a fragile edit page.

## Scope

Fix only these 6 issues. No redesign of list page, detail page tabs, stat cards, or expanded rows.

---

## 1. Database Changes

New migration `053_policy_type_and_timezone.sql`:

```sql
-- +goose Up
ALTER TABLE policies ADD COLUMN policy_type TEXT NOT NULL DEFAULT 'patch';
ALTER TABLE policies ADD CONSTRAINT chk_policy_type
    CHECK (policy_type IN ('patch', 'deploy', 'compliance'));

ALTER TABLE policies ADD COLUMN timezone TEXT NOT NULL DEFAULT 'UTC';

ALTER TABLE policies ADD COLUMN mw_enabled BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE policies DROP CONSTRAINT IF EXISTS chk_policy_type;
ALTER TABLE policies DROP COLUMN IF EXISTS mw_enabled;
ALTER TABLE policies DROP COLUMN IF EXISTS timezone;
ALTER TABLE policies DROP COLUMN IF EXISTS policy_type;
```

Existing rows: `policy_type = 'patch'`, `timezone = 'UTC'`, `mw_enabled = false`.

### sqlc Query Changes (`policies.sql`)

Add `policy_type`, `timezone`, `mw_enabled` to:
- `CreatePolicy` INSERT + params
- `UpdatePolicy` SET + params
- All SELECT queries (they use `SELECT *` so no change needed for reads)

Add `@type_filter` to `ListPolicies`, `ListPoliciesWithStats`, `CountPolicies`, `CountPoliciesFiltered`:
```sql
AND (@type_filter::text = '' OR p.policy_type = @type_filter)
```

---

## 2. OpenAPI + Backend Handler

### OpenAPI (`api/server.yaml`)

Add to `CreatePolicyRequest`:
```yaml
policy_type:
  type: string
  enum: [patch, deploy, compliance]
  default: patch
timezone:
  type: string
  default: UTC
mw_enabled:
  type: boolean
  default: false
```

Add same fields to `UpdatePolicyRequest`.

### Backend Handler (`policies.go`)

- Parse `policy_type`, `timezone`, `mw_enabled` in create/update handlers
- Validate `timezone` via `time.LoadLocation(tz)` — reject invalid IANA zone names
- Validate `schedule_cron` when `schedule_type = 'recurring'` — parse with cron library, return field-level error
- If `policy_type = 'compliance'`, force `mode = 'advisory'` server-side regardless of what client sends
- If `policy_type = 'deploy'`, reject `mode = 'advisory'`
- Return structured validation errors:
  ```json
  {"message": "Invalid cron expression", "field": "schedule_cron"}
  ```
- Add `policy_type` to `policyResponse` struct

---

## 3. Frontend Form Changes

### 3.1 Policy Type Selector

Both `PolicyForm.tsx` and `CreatePolicyDialog.tsx` get a new **Policy Type** section as the first card/section in the form.

Three selectable cards (same visual pattern as existing mode cards):
- **Patch Policy** — "Select patches by severity, CVE, or regex. Evaluate compliance and optionally auto-deploy."
- **Deploy Policy** — "Target specific updates for direct deployment to endpoints."
- **Compliance Policy** — "Evaluate patch compliance on a schedule. Report only, no deployments."

Default: `patch`.

### 3.2 Conditional Form Sections by Type

| Section | Patch | Deploy | Compliance |
|---------|-------|--------|------------|
| Basics (name, desc) | Yes | Yes | Yes |
| Mode | All 3 options | Automatic + Manual only | Locked to Advisory |
| Target Tags | Yes | Yes | Yes |
| Patch Selection | Yes | Hidden | Yes |
| Schedule | All presets | All presets | Defaults to Weekly/Monthly |
| Maintenance Window | Yes | Yes | Hidden |

### 3.3 Mode Descriptions (Expanded)

Replace terse descriptions with 2-line explanations:

- **Automatic**: "Evaluates on schedule. Matching patches deploy automatically within the maintenance window."
- **Manual**: "Evaluates on schedule. Patches are queued but NOT deployed until you click Deploy."
- **Advisory**: "Evaluates on schedule. Reports compliance status only. No patches are ever deployed."

Apply in both `PolicyForm.tsx` (card-style radios) and `CreatePolicyDialog.tsx` (dropdown → convert to card-style radios for consistency).

### 3.4 Schedule Section Overhaul

Replace raw cron text input with:

1. **Schedule type**: Manual | One-time | Recurring (replace current manual/recurring radio)
2. **One-time**: Date picker + time picker → stored as cron with specific date
3. **Recurring presets**: Daily, Weekly, Monthly, Custom buttons
   - **Daily**: Time picker → `0 {H} * * *`
   - **Weekly**: Day-of-week checkboxes + time picker → `0 {H} * * {D}`
   - **Monthly**: Day-of-month dropdown + time picker → `0 {H} {D} * *`
   - **Custom**: Raw cron input (collapsed, for advanced users)
4. **"Next 3 runs" preview**: Computed client-side using `cron-parser` npm package
5. **Timezone dropdown**: Searchable select with IANA timezone names (e.g., "Asia/Kolkata (IST)"). Stored as `timezone` field.

### 3.5 Maintenance Window Toggle

Add enable/disable toggle (`mw_enabled`) at the top of the Maintenance Window card:
- **Off**: Start/end time inputs hidden. Backend ignores `mw_start`/`mw_end`.
- **On**: Show start/end time pickers. Display timezone (inherited from schedule timezone).

### 3.6 CreatePolicyDialog — Groups → Tags

Switch from `useGroups()` + `group_ids` to `useTags()` + `tag_ids`:
- Change section title: "Target Groups" → "Target Tags"
- Replace group checkboxes with tag checkboxes (same pattern as `PolicyForm`)
- Remove `group_ids` from Zod schema, add `tag_ids`
- On submit, map `tag_ids` → `group_ids` for the API (same as `PolicyForm` does today, until backend supports tag-based targeting natively)

### 3.7 Zod Schema Updates

Both forms add:
```typescript
policy_type: z.enum(['patch', 'deploy', 'compliance']).default('patch'),
timezone: z.string().default('UTC'),
mw_enabled: z.boolean().default(false),
```

---

## 4. Error Handling

### Backend
- Validation errors return HTTP 422 with `{"message": "...", "field": "field_name"}`
- Duplicate name: `{"message": "A policy with this name already exists", "field": "name"}`
- Invalid cron: `{"message": "Invalid cron expression", "field": "schedule_cron"}`
- Invalid timezone: `{"message": "Unknown timezone", "field": "timezone"}`

### Frontend
- Parse `error.field` from mutation error response
- If `field` matches a form field, display error inline next to that field (red text below input)
- If no `field` or unknown field, show toast notification with `error.message`
- `PolicyForm.tsx`: Add `onError` handling (currently missing — only `CreatePolicyDialog` shows errors)
- `CreatePolicyDialog.tsx`: Improve existing error display to show field-level errors inline

---

## 5. Edit Policy Cleanup

**EditPolicyPage.tsx**:
- Remove `group_ids ↔ tag_ids` mapping comment and shim
- Add new fields to `defaultValues`: `policy_type`, `timezone`, `mw_enabled`
- Pass `policy.policy_type ?? 'patch'` (backward compat for existing policies)
- Pass `policy.timezone ?? 'UTC'`
- Pass `policy.mw_enabled ?? false`

No other changes. `PolicyForm` handles all rendering.

---

## 6. Files to Touch

### Backend
| File | Change |
|------|--------|
| `internal/server/store/migrations/053_policy_type_and_timezone.sql` | New migration |
| `internal/server/store/queries/policies.sql` | Add new columns to INSERT/UPDATE, add type_filter |
| `internal/server/store/sqlcgen/` | Regenerate (`make sqlc`) |
| `internal/server/api/v1/policies.go` | Parse new fields, validation, error format |
| `api/server.yaml` | Add fields to Create/Update request schemas |

### Frontend
| File | Change |
|------|--------|
| `web/src/pages/policies/PolicyForm.tsx` | Type selector, mode descriptions, schedule overhaul, MW toggle, error handling |
| `web/src/pages/policies/CreatePolicyDialog.tsx` | Type selector, groups→tags, mode descriptions, schedule, MW toggle, errors |
| `web/src/pages/policies/EditPolicyPage.tsx` | Remove shim, add new default values |
| `web/src/pages/policies/PolicyPreview.tsx` | Show policy type in preview |
| `web/src/pages/policies/tabs/ScheduleTab.tsx` | Display timezone |
| `web/src/api/hooks/usePolicies.ts` | No structural changes (hooks already handle errors) |
| `web/src/api/types.ts` | Regenerate (`make api-client`) |

### New Dependencies
- `cron-parser` npm package for "Next 3 runs" preview (client-side only)

---

## 7. What's NOT Changing

- `PoliciesPage.tsx` — no column, stat card, expanded row, or filter changes
- `PolicyDetailPage.tsx` — no tab structure or header changes
- Tab components (Overview, PatchScope, GroupsEndpoints, EvalHistory, DeploymentHistory) — untouched
- `PolicyPreview.tsx` — minor addition only (show type)
- Backend policy engine / evaluator — no evaluation logic changes
- Compliance engine — untouched
- No new API endpoints — only modifying existing create/update schemas
