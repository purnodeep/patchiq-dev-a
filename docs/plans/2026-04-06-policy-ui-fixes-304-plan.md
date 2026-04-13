# Policy UI/UX Fixes (#304) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix 6 targeted issues in the Policies UI — add policy type selector, fix group/tag consistency, improve mode clarity, overhaul schedule/timezone/maintenance window, add proper error handling, and fix edit policy.

**Architecture:** Backend-first approach. New DB migration adds `policy_type`, `timezone`, `mw_enabled` columns. OpenAPI spec updated. Handler validation enhanced with field-level errors. Frontend forms updated to use new fields with conditional sections per policy type. Schedule section gets preset buttons + timezone dropdown.

**Tech Stack:** Go (backend handler, migration), PostgreSQL (migration), sqlc (query codegen), OpenAPI (spec), React 19 + TypeScript + react-hook-form + Zod (frontend forms), cron-parser (npm, client-side next-run preview)

**Worktree:** `/home/rishabh/patchiq/.worktrees/fix/policy-ui-fixes`

---

## File Structure

### Backend (create/modify)
| File | Responsibility |
|------|---------------|
| `internal/server/store/migrations/053_policy_type_and_timezone.sql` | **Create** — new migration adding 3 columns |
| `internal/server/store/queries/policies.sql` | **Modify** — add new columns to INSERT/UPDATE, add type_filter to list queries |
| `internal/server/store/sqlcgen/` | **Regenerate** — `make sqlc` |
| `internal/server/api/v1/policies.go` | **Modify** — parse new fields, validation, field-level errors, policy type constraints |
| `internal/server/api/v1/response.go` | **Modify** — add `WriteFieldError` helper |
| `api/server.yaml` | **Modify** — add new fields to Create/Update request & response schemas |

### Frontend (create/modify)
| File | Responsibility |
|------|---------------|
| `web/src/pages/policies/PolicyForm.tsx` | **Modify** — type selector, mode descriptions, schedule overhaul, MW toggle, error handling |
| `web/src/pages/policies/CreatePolicyDialog.tsx` | **Modify** — type selector, groups→tags, mode descriptions, schedule, MW toggle, errors |
| `web/src/pages/policies/EditPolicyPage.tsx` | **Modify** — remove shim, add new default values |
| `web/src/pages/policies/PolicyPreview.tsx` | **Modify** — show policy type |
| `web/src/pages/policies/tabs/ScheduleTab.tsx` | **Modify** — display timezone |
| `web/src/api/types.ts` | **Regenerate** — `make api-client` |

---

## Task 1: Database Migration

**Files:**
- Create: `internal/server/store/migrations/053_policy_type_and_timezone.sql`

- [ ] **Step 1: Create the migration file**

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

- [ ] **Step 2: Run the migration**

Run: `cd /home/rishabh/patchiq/.worktrees/fix/policy-ui-fixes && make migrate`
Expected: Migration 053 applied successfully.

- [ ] **Step 3: Verify columns exist**

Run: `cd /home/rishabh/patchiq/.worktrees/fix/policy-ui-fixes && PGPASSWORD=$(grep PATCHIQ_DB_PASSWORD .env | cut -d= -f2) psql -h localhost -p 5732 -U patchiq -d patchiq_dev_rishab -c "\d policies" | grep -E "policy_type|timezone|mw_enabled"`
Expected: Three new columns shown.

- [ ] **Step 4: Commit**

```bash
cd /home/rishabh/patchiq/.worktrees/fix/policy-ui-fixes
git add internal/server/store/migrations/053_policy_type_and_timezone.sql
git commit -m "feat(server): add policy_type, timezone, mw_enabled columns (#304)"
```

---

## Task 2: Update sqlc Queries

**Files:**
- Modify: `internal/server/store/queries/policies.sql`

- [ ] **Step 1: Add new columns to CreatePolicy query**

In `policies.sql`, update the `CreatePolicy` INSERT to include the 3 new columns:

```sql
-- name: CreatePolicy :one
INSERT INTO policies (
    tenant_id, name, description, enabled, mode,
    selection_mode, min_severity, cve_ids, package_regex, exclude_packages,
    schedule_type, schedule_cron, mw_start, mw_end, deployment_strategy,
    policy_type, timezone, mw_enabled
) VALUES (
    @tenant_id, @name, @description, @enabled, @mode,
    @selection_mode, @min_severity, @cve_ids, @package_regex, @exclude_packages,
    @schedule_type, @schedule_cron, @mw_start, @mw_end, @deployment_strategy,
    @policy_type, @timezone, @mw_enabled
) RETURNING *;
```

- [ ] **Step 2: Add new columns to UpdatePolicy query**

```sql
-- name: UpdatePolicy :one
UPDATE policies SET
    name = @name,
    description = @description,
    enabled = @enabled,
    mode = @mode,
    selection_mode = @selection_mode,
    min_severity = @min_severity,
    cve_ids = @cve_ids,
    package_regex = @package_regex,
    exclude_packages = @exclude_packages,
    schedule_type = @schedule_type,
    schedule_cron = @schedule_cron,
    mw_start = @mw_start,
    mw_end = @mw_end,
    deployment_strategy = @deployment_strategy,
    policy_type = @policy_type,
    timezone = @timezone,
    mw_enabled = @mw_enabled,
    updated_at = now()
WHERE id = @id AND tenant_id = @tenant_id AND deleted_at IS NULL
RETURNING *;
```

- [ ] **Step 3: Add type_filter to ListPoliciesWithStats**

Add `AND (@type_filter::text = '' OR p.policy_type = @type_filter)` to the WHERE clause, after the `mode_filter` line. Do the same for `CountPoliciesFiltered`.

- [ ] **Step 4: Regenerate sqlc**

Run: `cd /home/rishabh/patchiq/.worktrees/fix/policy-ui-fixes && make sqlc`
Expected: Generated files updated with new fields in `CreatePolicyParams`, `UpdatePolicyParams`, and `Policy` struct.

- [ ] **Step 5: Verify generated code has new fields**

Run: `grep -n "PolicyType\|Timezone\|MwEnabled" internal/server/store/sqlcgen/policies.sql.go | head -10`
Expected: `PolicyType`, `Timezone`, `MwEnabled` fields in `Policy`, `CreatePolicyParams`, `UpdatePolicyParams` structs.

- [ ] **Step 6: Commit**

```bash
cd /home/rishabh/patchiq/.worktrees/fix/policy-ui-fixes
git add internal/server/store/queries/policies.sql internal/server/store/sqlcgen/
git commit -m "feat(server): add policy_type, timezone, mw_enabled to sqlc queries (#304)"
```

---

## Task 3: Backend Handler — New Fields + Validation

**Files:**
- Modify: `internal/server/api/v1/policies.go`
- Modify: `internal/server/api/v1/response.go`

- [ ] **Step 1: Add WriteFieldError to response.go**

Add after the existing `WriteError` function in `response.go`:

```go
// WriteFieldError writes a JSON validation error response with a field indicator.
func WriteFieldError(w http.ResponseWriter, status int, code, message, field string) {
	WriteJSON(w, status, map[string]any{
		"code":    code,
		"message": message,
		"field":   field,
		"details": []any{},
	})
}
```

- [ ] **Step 2: Add validPolicyTypes map and update createPolicyRequest struct**

In `policies.go`, add after `validPolicyModes`:

```go
var validPolicyTypes = map[string]bool{
	"patch":      true,
	"deploy":     true,
	"compliance": true,
}
```

Add to `createPolicyRequest` struct:
```go
PolicyType string `json:"policy_type,omitempty"`
Timezone   string `json:"timezone,omitempty"`
MwEnabled  *bool  `json:"mw_enabled,omitempty"`
```

- [ ] **Step 3: Update validatePolicyRequest with new fields + field-level errors**

Update `validatePolicyRequest` signature to return a third value (field name):

```go
func validatePolicyRequest(body *createPolicyRequest) (string, string, string) {
```

Add validations:
- `policy_type`: if non-empty, must be in `validPolicyTypes`
- `timezone`: if non-empty, validate with `time.LoadLocation(body.Timezone)` — return field `"timezone"` on error
- `compliance` type forces `mode = advisory` — if body has `policy_type = "compliance"` and `mode != "" && mode != "advisory"`, return error on `"mode"` field
- `deploy` type rejects `advisory` mode — return error on `"mode"` field
- Update existing validation returns to include field names (e.g., `"name"`, `"selection_mode"`, `"schedule_cron"`)

Update all callers of `validatePolicyRequest` to handle the third return value and use `WriteFieldError`.

- [ ] **Step 4: Update resolvePolicyDefaults for new fields**

Add to `policyDefaults` struct:
```go
PolicyType string
Timezone   string
MwEnabled  bool
```

In `resolvePolicyDefaults`:
```go
policyType := body.PolicyType
if policyType == "" {
    policyType = "patch"
}
tz := body.Timezone
if tz == "" {
    tz = "UTC"
}
mwEnabled := false
if body.MwEnabled != nil {
    mwEnabled = *body.MwEnabled
}
// Compliance type forces advisory mode.
if policyType == "compliance" {
    mode = "advisory"
}
```

- [ ] **Step 5: Update buildCreateParams and buildUpdateParams**

Add the three new fields to `buildCreateParams`:
```go
PolicyType: d.PolicyType,
Timezone:   d.Timezone,
MwEnabled:  d.MwEnabled,
```

Do the same for `buildUpdateParams` (or wherever the update params are constructed — check the Update handler).

- [ ] **Step 6: Update policyResponse and toPolicyResponse**

Add to `policyResponse`:
```go
PolicyType string `json:"policy_type"`
Timezone   string `json:"timezone"`
MwEnabled  bool   `json:"mw_enabled"`
```

In `toPolicyResponse`:
```go
PolicyType: p.PolicyType,
Timezone:   p.Timezone,
MwEnabled:  p.MwEnabled,
```

In `toPolicyResponseWithStats`:
```go
PolicyType: p.PolicyType,
Timezone:   p.Timezone,
MwEnabled:  p.MwEnabled,
```

- [ ] **Step 7: Build and verify**

Run: `cd /home/rishabh/patchiq/.worktrees/fix/policy-ui-fixes && go build ./cmd/server/`
Expected: Compiles without errors.

- [ ] **Step 8: Commit**

```bash
cd /home/rishabh/patchiq/.worktrees/fix/policy-ui-fixes
git add internal/server/api/v1/policies.go internal/server/api/v1/response.go
git commit -m "feat(server): add policy_type/timezone/mw_enabled to handler with field-level errors (#304)"
```

---

## Task 4: OpenAPI Spec + Regenerate Types

**Files:**
- Modify: `api/server.yaml`
- Regenerate: `web/src/api/types.ts`

- [ ] **Step 1: Add new fields to CreatePolicyRequest in server.yaml**

After the `mw_end` field in `CreatePolicyRequest` (around line 432), add:

```yaml
        policy_type:
          type: string
          enum:
            - patch
            - deploy
            - compliance
          default: patch
        timezone:
          type: string
          default: UTC
        mw_enabled:
          type: boolean
          default: false
```

- [ ] **Step 2: Add same fields to UpdatePolicyRequest**

After the `mw_end` field in `UpdatePolicyRequest` (around line 491), add the same three fields.

- [ ] **Step 3: Add fields to Policy response schema**

Find the Policy response schema in `server.yaml` and add `policy_type`, `timezone`, `mw_enabled` fields.

- [ ] **Step 4: Regenerate frontend types**

Run: `cd /home/rishabh/patchiq/.worktrees/fix/policy-ui-fixes && make api-client`
Expected: `web/src/api/types.ts` updated with new fields.

- [ ] **Step 5: Verify generated types**

Run: `grep -n "policy_type\|timezone\|mw_enabled" web/src/api/types.ts | head -10`
Expected: Fields present in `CreatePolicyRequest`, `UpdatePolicyRequest`, and `Policy` schemas.

- [ ] **Step 6: Commit**

```bash
cd /home/rishabh/patchiq/.worktrees/fix/policy-ui-fixes
git add api/server.yaml web/src/api/types.ts
git commit -m "feat(api): add policy_type, timezone, mw_enabled to OpenAPI spec (#304)"
```

---

## Task 5: Install cron-parser npm Package

**Files:**
- Modify: `web/package.json`

- [ ] **Step 1: Install cron-parser**

Run: `cd /home/rishabh/patchiq/.worktrees/fix/policy-ui-fixes/web && pnpm add cron-parser`
Expected: Package added to `web/package.json` dependencies.

- [ ] **Step 2: Commit**

```bash
cd /home/rishabh/patchiq/.worktrees/fix/policy-ui-fixes
git add web/package.json web/pnpm-lock.yaml pnpm-lock.yaml
git commit -m "chore(web): add cron-parser for schedule preview (#304)"
```

---

## Task 6: PolicyForm.tsx — Policy Type Selector + Mode Descriptions

**Files:**
- Modify: `web/src/pages/policies/PolicyForm.tsx`

- [ ] **Step 1: Update Zod schema**

Add to the schema in `PolicyForm.tsx`:

```typescript
policy_type: z.enum(['patch', 'deploy', 'compliance']).default('patch'),
timezone: z.string().default('UTC'),
mw_enabled: z.boolean().default(false),
```

Update the `defaultValues` in `useForm` to include:
```typescript
policy_type: 'patch',
timezone: 'UTC',
mw_enabled: false,
```

- [ ] **Step 2: Add policy type selector as the first card**

Add a new card before the "Basics" card. Three selectable cards (same visual style as existing mode cards):

```typescript
const typeOptions = [
  {
    value: 'patch',
    label: 'Patch Policy',
    desc: 'Select patches by severity, CVE, or regex. Evaluate and optionally auto-deploy.',
    color: 'var(--accent)',
  },
  {
    value: 'deploy',
    label: 'Deploy Policy',
    desc: 'Target specific updates for direct deployment to endpoints.',
    color: 'var(--signal-healthy)',
  },
  {
    value: 'compliance',
    label: 'Compliance Policy',
    desc: 'Evaluate patch compliance on a schedule. Report only, no deployments.',
    color: 'var(--text-muted)',
  },
];
```

Render using same card pattern as the existing mode cards (lines 230-283 in current file). Register as `policy_type`.

- [ ] **Step 3: Add `watch('policy_type')` and conditional rendering**

```typescript
const policyType = watch('policy_type');
```

Conditionally render sections:
- Patch Selection card: `{policyType !== 'deploy' && ( ... )}`
- Maintenance Window card: `{policyType !== 'compliance' && ( ... )}`
- Mode section: if `policyType === 'compliance'`, lock to advisory (set value via `useEffect` when type changes, visually show advisory-only with a note)
- Mode section: if `policyType === 'deploy'`, filter out `advisory` from `modeOptions`

- [ ] **Step 4: Update mode descriptions**

Replace the `desc` strings in `modeOptions`:
```typescript
const modeOptions = [
  {
    value: 'automatic',
    label: 'Automatic',
    desc: 'Evaluates on schedule. Matching patches deploy automatically within the maintenance window.',
    color: 'var(--signal-healthy)',
  },
  {
    value: 'manual',
    label: 'Manual',
    desc: 'Evaluates on schedule. Patches are queued but NOT deployed until you click Deploy.',
    color: 'var(--accent)',
  },
  {
    value: 'advisory',
    label: 'Advisory',
    desc: 'Evaluates on schedule. Reports compliance status only. No patches are ever deployed.',
    color: 'var(--text-muted)',
  },
];
```

- [ ] **Step 5: Verify the form renders**

Run: `cd /home/rishabh/patchiq/.worktrees/fix/policy-ui-fixes/web && pnpm tsc --noEmit`
Expected: No TypeScript errors.

- [ ] **Step 6: Commit**

```bash
cd /home/rishabh/patchiq/.worktrees/fix/policy-ui-fixes
git add web/src/pages/policies/PolicyForm.tsx
git commit -m "feat(web): add policy type selector and mode descriptions to PolicyForm (#304)"
```

---

## Task 7: PolicyForm.tsx — Schedule Overhaul + Timezone

**Files:**
- Modify: `web/src/pages/policies/PolicyForm.tsx`

- [ ] **Step 1: Replace raw cron input with schedule presets**

Replace the Schedule card contents (currently lines 478-526) with:

1. Schedule type radio: Manual | Recurring (keep existing)
2. When `recurring`, show preset buttons: Daily, Weekly, Monthly, Custom
3. Each preset shows relevant picker:
   - **Daily**: time input → generates `0 {H} * * *`
   - **Weekly**: day-of-week checkboxes (Mon-Sun) + time input → generates `0 {H} * * {D}`
   - **Monthly**: day-of-month select (1-28) + time input → generates `0 {H} {D} * *`
   - **Custom**: raw cron input (for advanced users)
4. All presets write back to `schedule_cron` field via `setValue`

Add local state for preset tracking:
```typescript
const [schedulePreset, setSchedulePreset] = useState<'daily' | 'weekly' | 'monthly' | 'custom'>('weekly');
```

When compliance type is selected, default preset to `'weekly'`.

- [ ] **Step 2: Add "Next 3 runs" preview**

Below the schedule inputs, add a preview section:

```typescript
import { parseExpression } from 'cron-parser';

// Inside the component:
const cronValue = watch('schedule_cron');
const timezoneValue = watch('timezone');

const nextRuns = useMemo(() => {
  if (!cronValue) return [];
  try {
    const interval = parseExpression(cronValue, { tz: timezoneValue || 'UTC' });
    return [interval.next(), interval.next(), interval.next()].map(d =>
      d.toDate().toLocaleString('en-US', { timeZone: timezoneValue || 'UTC', dateStyle: 'medium', timeStyle: 'short' })
    );
  } catch {
    return [];
  }
}, [cronValue, timezoneValue]);
```

Render as a small list:
```tsx
{nextRuns.length > 0 && (
  <div style={{ marginTop: 10 }}>
    <span style={{ ...LABEL, marginBottom: 4 }}>Next runs</span>
    {nextRuns.map((r, i) => (
      <div key={i} style={{ fontSize: 12, color: 'var(--text-muted)', fontFamily: 'var(--font-mono)' }}>{r}</div>
    ))}
  </div>
)}
```

- [ ] **Step 3: Add timezone dropdown**

Add a searchable timezone select below the schedule presets. Use a plain `<select>` with common IANA timezones:

```typescript
const COMMON_TIMEZONES = [
  'UTC', 'America/New_York', 'America/Chicago', 'America/Denver', 'America/Los_Angeles',
  'Europe/London', 'Europe/Paris', 'Europe/Berlin', 'Asia/Kolkata', 'Asia/Tokyo',
  'Asia/Shanghai', 'Australia/Sydney', 'Pacific/Auckland',
];
```

Register as `timezone` field. Show below the cron/preset inputs when `scheduleType === 'recurring'`.

- [ ] **Step 4: Add maintenance window toggle**

In the Maintenance Window card, add an enable/disable toggle at the top:

```tsx
<div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 16 }}>
  <div style={SECTION_TITLE}>Maintenance Window</div>
  <label style={{ display: 'flex', alignItems: 'center', gap: 8, cursor: 'pointer' }}>
    <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>
      {watch('mw_enabled') ? 'Enabled' : 'Disabled'}
    </span>
    <input
      type="checkbox"
      {...register('mw_enabled')}
      style={{ accentColor: 'var(--accent)', width: 16, height: 16, cursor: 'pointer' }}
    />
  </label>
</div>
```

Only show start/end time inputs when `mw_enabled` is true:
```tsx
{watch('mw_enabled') && (
  <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
    {/* existing start/end inputs */}
  </div>
)}
```

- [ ] **Step 5: TypeScript check**

Run: `cd /home/rishabh/patchiq/.worktrees/fix/policy-ui-fixes/web && pnpm tsc --noEmit`
Expected: No errors.

- [ ] **Step 6: Commit**

```bash
cd /home/rishabh/patchiq/.worktrees/fix/policy-ui-fixes
git add web/src/pages/policies/PolicyForm.tsx
git commit -m "feat(web): schedule presets, timezone, MW toggle in PolicyForm (#304)"
```

---

## Task 8: CreatePolicyDialog.tsx — Full Update

**Files:**
- Modify: `web/src/pages/policies/CreatePolicyDialog.tsx`

- [ ] **Step 1: Switch from groups to tags**

Replace:
- `import { useGroups } from '../../api/hooks/useGroups'` → `import { useTags } from '../../api/hooks/useTags'`
- `const groups = useGroups()` → `const { data: tagsData, isLoading: tagsLoading } = useTags({ limit: 100 })`
- In schema: `group_ids: z.array(z.string()).min(1, 'Select at least one group')` → `tag_ids: z.array(z.string()).min(1, 'Select at least one tag')`
- Update default values: `group_ids: []` → `tag_ids: []`
- Section title: "Target Groups" → "Target Tags"
- Replace group checkbox list with tag checkbox list (same pattern as `PolicyForm.tsx` lines 306-367)
- On submit: map `tag_ids` → `group_ids` for the API: `await createPolicy.mutateAsync({ ...values, group_ids: values.tag_ids })`

- [ ] **Step 2: Add policy type, timezone, mw_enabled to schema**

Add to the Zod schema:
```typescript
policy_type: z.enum(['patch', 'deploy', 'compliance']).default('patch'),
timezone: z.string().default('UTC'),
mw_enabled: z.boolean().default(false),
```

Add to default values:
```typescript
policy_type: 'patch',
timezone: 'UTC',
mw_enabled: false,
```

- [ ] **Step 3: Add Policy Type selector section**

Add as first section in the form (before Basics). Three options as radio cards (same visual pattern as the mode selector but using the policy type options from Task 6).

- [ ] **Step 4: Replace mode dropdown with card-style radios**

Replace the `<select>` for mode (line 134) with card-style radio buttons matching `PolicyForm.tsx` pattern. Use the same expanded descriptions. Apply conditional filtering based on `policy_type` (same logic as Task 6 Step 3).

- [ ] **Step 5: Add schedule presets + timezone + MW toggle**

Apply the same schedule preset pattern from Task 7 to the Schedule section. Add timezone dropdown. Add MW toggle to Maintenance Window section.

- [ ] **Step 6: Improve error handling**

The dialog already shows `createPolicy.error?.message` in the footer. Enhance:
- Parse `error.field` from the API response
- If field matches a form field, show inline error via `setError(field, { message })` from react-hook-form
- Keep the footer error as fallback for non-field errors

```typescript
const onSubmit = handleSubmit(async (values) => {
  try {
    await createPolicy.mutateAsync({ ...values, group_ids: values.tag_ids });
    reset();
    onOpenChange(false);
    navigate('/policies');
  } catch (err: unknown) {
    const apiErr = err as { message?: string; field?: string };
    if (apiErr?.field && apiErr.field in values) {
      setError(apiErr.field as keyof FormValues, { message: apiErr.message ?? 'Validation error' });
    }
  }
});
```

- [ ] **Step 7: TypeScript check**

Run: `cd /home/rishabh/patchiq/.worktrees/fix/policy-ui-fixes/web && pnpm tsc --noEmit`
Expected: No errors.

- [ ] **Step 8: Commit**

```bash
cd /home/rishabh/patchiq/.worktrees/fix/policy-ui-fixes
git add web/src/pages/policies/CreatePolicyDialog.tsx
git commit -m "feat(web): update CreatePolicyDialog with type, tags, schedule, errors (#304)"
```

---

## Task 9: EditPolicyPage.tsx — Cleanup + New Fields

**Files:**
- Modify: `web/src/pages/policies/EditPolicyPage.tsx`

- [ ] **Step 1: Remove group_ids → tag_ids mapping shim**

Replace the submit handler:
```typescript
const handleSubmit = async (values: PolicyFormValues) => {
  await updatePolicy.mutateAsync({
    ...values,
    group_ids: values.tag_ids,
  } as UpdatePolicyRequest);
  navigate(`/policies/${id}`);
};
```

- [ ] **Step 2: Add new fields to defaultValues**

Add to the `defaultValues` object passed to `PolicyForm`:
```typescript
policy_type: policy.policy_type ?? 'patch',
timezone: policy.timezone ?? 'UTC',
mw_enabled: policy.mw_enabled ?? false,
```

- [ ] **Step 3: TypeScript check**

Run: `cd /home/rishabh/patchiq/.worktrees/fix/policy-ui-fixes/web && pnpm tsc --noEmit`
Expected: No errors.

- [ ] **Step 4: Commit**

```bash
cd /home/rishabh/patchiq/.worktrees/fix/policy-ui-fixes
git add web/src/pages/policies/EditPolicyPage.tsx
git commit -m "fix(web): clean up EditPolicyPage, add new policy fields (#304)"
```

---

## Task 10: PolicyPreview.tsx + ScheduleTab.tsx — Minor Updates

**Files:**
- Modify: `web/src/pages/policies/PolicyPreview.tsx`
- Modify: `web/src/pages/policies/tabs/ScheduleTab.tsx`

- [ ] **Step 1: Show policy type in PolicyPreview**

Add a small badge/label at the top of the preview sidebar showing the selected policy type:

```tsx
<div style={{ ...LABEL }}>Policy Type</div>
<div style={{ fontSize: 13, fontWeight: 600, color: 'var(--text-primary)', marginBottom: 12 }}>
  {values.policy_type === 'patch' ? 'Patch Policy' : values.policy_type === 'deploy' ? 'Deploy Policy' : 'Compliance Policy'}
</div>
```

Update the `PolicyFormValues` import — it already includes the new fields since Task 6 updated the schema.

- [ ] **Step 2: Display timezone in ScheduleTab**

In `ScheduleTab.tsx`, find where the schedule/cron is displayed and add a timezone line below it:

```tsx
{policy.timezone && policy.timezone !== 'UTC' && (
  <div style={{ fontSize: 12, color: 'var(--text-muted)', marginTop: 4 }}>
    Timezone: {policy.timezone}
  </div>
)}
```

- [ ] **Step 3: TypeScript check**

Run: `cd /home/rishabh/patchiq/.worktrees/fix/policy-ui-fixes/web && pnpm tsc --noEmit`
Expected: No errors.

- [ ] **Step 4: Commit**

```bash
cd /home/rishabh/patchiq/.worktrees/fix/policy-ui-fixes
git add web/src/pages/policies/PolicyPreview.tsx web/src/pages/policies/tabs/ScheduleTab.tsx
git commit -m "feat(web): show policy type in preview, timezone in ScheduleTab (#304)"
```

---

## Task 11: PolicyForm Error Handling

**Files:**
- Modify: `web/src/pages/policies/PolicyForm.tsx`

- [ ] **Step 1: Add server error state and field-level error handling**

The `PolicyForm` receives `onSubmit` from parent. The parent (CreatePolicyPage/EditPolicyPage) calls the mutation. To surface field errors, the parent needs to catch and pass them back.

Update `PolicyFormProps`:
```typescript
interface PolicyFormProps {
  defaultValues?: Partial<PolicyFormValues>;
  onSubmit: (values: PolicyFormValues) => Promise<void>;
  submitLabel: string;
  isPending: boolean;
  serverError?: { message?: string; field?: string } | null;
}
```

In the component, use `useEffect` to set field errors when `serverError` changes:
```typescript
useEffect(() => {
  if (serverError?.field && serverError.field in schema.shape) {
    setError(serverError.field as keyof PolicyFormValues, {
      message: serverError.message ?? 'Validation error',
    });
  }
}, [serverError, setError]);
```

Add `setError` to the destructured form methods.

Show a general error toast/banner if no field:
```tsx
{serverError && !serverError.field && (
  <div style={{
    padding: '10px 14px',
    background: 'color-mix(in srgb, var(--signal-critical) 10%, transparent)',
    border: '1px solid var(--signal-critical)',
    borderRadius: 6,
    fontSize: 13,
    color: 'var(--signal-critical)',
    marginBottom: 8,
  }}>
    {serverError.message ?? 'Failed to save policy'}
  </div>
)}
```

- [ ] **Step 2: Update CreatePolicyPage to pass serverError**

In `CreatePolicyPage.tsx` (the full-page create route), catch the mutation error and pass to `PolicyForm`:

```typescript
const [serverError, setServerError] = useState<{ message?: string; field?: string } | null>(null);

const handleSubmit = async (values: PolicyFormValues) => {
  try {
    setServerError(null);
    await createPolicy.mutateAsync({ ...values, group_ids: values.tag_ids });
    navigate('/policies');
  } catch (err: unknown) {
    const apiErr = err as { message?: string; field?: string };
    setServerError(apiErr ?? { message: 'Failed to create policy' });
  }
};

<PolicyForm ... serverError={serverError} />
```

Do the same in `EditPolicyPage.tsx`.

- [ ] **Step 3: TypeScript check**

Run: `cd /home/rishabh/patchiq/.worktrees/fix/policy-ui-fixes/web && pnpm tsc --noEmit`
Expected: No errors.

- [ ] **Step 4: Commit**

```bash
cd /home/rishabh/patchiq/.worktrees/fix/policy-ui-fixes
git add web/src/pages/policies/PolicyForm.tsx web/src/pages/policies/CreatePolicyPage.tsx web/src/pages/policies/EditPolicyPage.tsx
git commit -m "feat(web): add field-level error handling to policy forms (#304)"
```

---

## Task 12: Final Build + Lint Verification

- [ ] **Step 1: Backend build**

Run: `cd /home/rishabh/patchiq/.worktrees/fix/policy-ui-fixes && go build ./cmd/server/`
Expected: Compiles cleanly.

- [ ] **Step 2: Frontend TypeScript check**

Run: `cd /home/rishabh/patchiq/.worktrees/fix/policy-ui-fixes/web && pnpm tsc --noEmit`
Expected: No errors.

- [ ] **Step 3: Frontend lint**

Run: `cd /home/rishabh/patchiq/.worktrees/fix/policy-ui-fixes && make lint-frontend`
Expected: No lint errors (or fix any that appear).

- [ ] **Step 4: Backend lint**

Run: `cd /home/rishabh/patchiq/.worktrees/fix/policy-ui-fixes && make lint`
Expected: No lint errors (or fix any that appear).

- [ ] **Step 5: Run Go tests**

Run: `cd /home/rishabh/patchiq/.worktrees/fix/policy-ui-fixes && go test ./internal/server/api/v1/... -race -count=1`
Expected: All tests pass.

- [ ] **Step 6: Fix any issues found, commit**

```bash
cd /home/rishabh/patchiq/.worktrees/fix/policy-ui-fixes
git add -A
git commit -m "fix: resolve lint and build issues (#304)"
```
