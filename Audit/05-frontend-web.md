# Frontend Audit: web/ (Patch Manager)

**Audited**: 2026-04-09
**Scope**: `web/src/` -- 340 TypeScript/TSX files
**Branch**: `dev-a`

---

## 1. Dead/Unused Files

### Critical

**F1.1 -- PlaceholderPage.tsx (stale M0 artifact)**
- File: `src/pages/PlaceholderPage.tsx`
- Never imported anywhere. Contains "Coming in M1" text. M1 is complete.
- **Action**: Delete.

**F1.2 -- NotificationSettingsPage.tsx (superseded)**
- File: `src/pages/settings/NotificationSettingsPage.tsx` (549 lines)
- Never imported. The route `/settings/notifications` points to `NotificationsPage` instead.
- **Action**: Delete. All notification config now lives in `NotificationsPage` + `PreferencesTab`.

**F1.3 -- GroupsPage and related files (orphaned page, no route)**
- Files: `src/pages/groups/GroupsPage.tsx`, `src/pages/groups/CreateGroupDialog.tsx`, `src/pages/groups/EditGroupDialog.tsx`
- No route in `routes.tsx` points to GroupsPage. Groups feature appears to have been replaced by tags.
- **Action**: Confirm removal or restore route.

### Important

**F1.4 -- Duplicate DeploymentModal implementations**
- `src/components/DeploymentModal.tsx` (461 lines) -- never imported outside its own file
- `src/pages/patches/DeploymentModal.tsx` (264 lines) -- only imported by its test file
- Both are dead code; the DeploymentWizard component replaced them.
- **Action**: Delete both. Delete the test at `src/__tests__/pages/patches/DeploymentModal.test.tsx`.

**F1.5 -- PatchDeploymentDialog.tsx (dead)**
- File: `src/pages/patches/PatchDeploymentDialog.tsx` (278 lines)
- Never imported anywhere.
- **Action**: Delete.

**F1.6 -- PatchExpandedRow.tsx (dead)**
- File: `src/pages/patches/PatchExpandedRow.tsx`
- Never imported anywhere.
- **Action**: Delete.

**F1.7 -- CreateDeploymentDialog.tsx (dead)**
- File: `src/pages/deployments/CreateDeploymentDialog.tsx`
- Never imported outside its own file.
- **Action**: Delete.

**F1.8 -- Compliance components -- dead stubs and unused files**
- `src/pages/compliance/components/framework-header.tsx` -- returns `null`, comment says "superseded"
- `src/pages/compliance/components/stats-bar.tsx` -- returns `null`, comment says "superseded"
- `src/pages/compliance/components/evidence-tab.tsx` -- placeholder "available in a future release"
- `src/pages/compliance/components/category-card.tsx` -- never imported
- `src/pages/compliance/components/custom-framework-card.tsx` -- never imported
- `src/pages/compliance/components/sla-tracker.tsx` -- never imported
- **Action**: Delete all six files.

**F1.9 -- AlertRow.tsx (dead)**
- File: `src/pages/alerts/AlertRow.tsx`
- Never imported; AlertsPage renders alerts inline.
- **Action**: Delete.

**F1.10 -- workflow-card.tsx (dead)**
- File: `src/pages/workflows/workflow-card.tsx`
- Never imported; WorkflowsPage renders cards inline.
- **Action**: Delete.

**F1.11 -- PermissionMatrix.tsx (dead)**
- File: `src/pages/admin/roles/components/PermissionMatrix.tsx` (245 lines)
- Never imported. RoleEditPage has its own inline permission UI.
- **Action**: Delete.

**F1.12 -- ContextRing.tsx (dead)**
- File: `src/pages/policies/components/ContextRing.tsx`
- Never imported.
- **Action**: Delete.

**F1.13 -- wave-progress.tsx (dead)**
- File: `src/pages/deployments/components/wave-progress.tsx`
- Never imported.
- **Action**: Delete.

**F1.14 -- TagsTab.tsx (dead)**
- File: `src/pages/endpoints/TagsTab.tsx`
- Never imported. Endpoint detail uses tabs in `src/pages/endpoints/tabs/`.
- **Action**: Delete.

### Minor

**F1.15 -- Unused shared components**
- `src/components/CVSSVectorBreakdown.tsx` (101 lines) -- never imported
- `src/components/SegmentedProgressBar.tsx` (86 lines) -- never imported
- `src/components/SlidePanel.tsx` (195 lines) -- never imported
- `src/components/StatCard.tsx` -- only imported in `ComponentPreview.tsx` (dev-only preview page)
- **Action**: Delete CVSSVectorBreakdown, SegmentedProgressBar, SlidePanel. Move StatCard deletion decision to after review (used in dev preview).

**F1.16 -- data-table subcomponents never used outside barrel**
- `src/components/data-table/DataTableFilters.tsx` -- exported from index but never imported by any page
- `src/components/data-table/BulkActionBar.tsx` -- exported from index but never imported by any page
- `src/components/data-table/selection-column.tsx` -- exported from index but never imported by any page
- **Action**: Delete or mark as planned-for-use.

**F1.17 -- dev-mocks.ts (dead)**
- File: `src/dev-mocks.ts`
- Never imported anywhere.
- **Action**: Delete.

---

## 2. Incomplete Pages / Placeholder Content

### Important

**F2.1 -- EvidenceTab placeholder**
- File: `src/pages/compliance/components/evidence-tab.tsx`
- Renders static text: "Compliance report generation will be available in a future release."
- No actual functionality. Also dead (not imported). See F1.8.

**F2.2 -- AddToGroupDialog -- backend not implemented**
- File: `src/pages/patches/AddToGroupDialog.tsx`, lines 58-59
- `// TODO(#306): call API to associate patchIds with groupId when backend supports patch-group membership`
- The confirm handler stores groupId in localStorage but does not actually add patches to the group.
- **Severity**: Important -- functional gap, user sees success toast but nothing happens.

**F2.3 -- Patch recall not implemented**
- File: `src/pages/patches/PatchDetailPage.tsx`, line 1985
- `// TODO(#306): call PATCH /api/v1/patches/{id} with status: 'recalled' when backend supports it`
- Recall button exists in UI but handler is incomplete.
- **Severity**: Important -- button visible to users, no effect.

---

## 3. TypeScript Issues (`any` types)

### Critical

**F3.1 -- Compliance module is riddled with `any` casts (30+ occurrences)**
- `src/pages/compliance/framework.tsx` -- 8 `any` casts (lines 471-481, 807-810)
- `src/pages/compliance/components/overview-tab.tsx` -- 8 `any` casts (lines 98, 218, 220-221, 369, 414, 478, 510)
- `src/pages/compliance/components/overall-score-card.tsx` -- 3 `any` casts (lines 30, 50-51)
- `src/pages/compliance/components/sla-tab.tsx` -- 3 `any` casts (lines 116, 136, 167)
- `src/pages/compliance/components/compliance-trend.tsx` -- line 39: `(api as any).GET`
- `src/pages/compliance/components/framework-card.tsx` -- line 51: `const fw = framework as any`
- Root cause: OpenAPI types for compliance endpoints are incomplete or mismatched. These casts mask real type errors.
- **Action**: Regenerate OpenAPI types to include compliance response shapes, then remove all `as any`.

**F3.2 -- API hooks with `as any` route casts (25+ occurrences)**
- `src/api/hooks/useCompliance.ts` -- 8 casts, lines 221-407: `(api as any).POST/GET/PUT/DELETE`
- `src/api/hooks/useDashboard.ts` -- 6 casts, lines 89-190: `(api as any).GET` with TODO(PIQ-233)
- `src/api/hooks/useIAMSettings.ts` -- 4 casts, lines 32-86: `api.GET('/api/v1/settings/iam' as any)`
- `src/api/hooks/useChannelByType.ts` -- 3 casts, lines 20-56: template path `as any`
- `src/api/hooks/useRoles.ts` -- 2 casts, lines 64, 144
- `src/api/hooks/useSettings.ts` -- 2 casts, lines 16, 28
- Root cause: These API routes are not in the generated OpenAPI spec (`src/api/types.ts`).
- **Action**: Update the OpenAPI spec to include all routes, regenerate types, remove casts.

### Important

**F3.3 -- Notification PreferencesTab `any` types**
- File: `src/pages/notifications/PreferencesTab.tsx`, line 238: `channel: any | undefined`
- Line 391: `(c: any) => c.channel_type === type`
- Line 557: cast to `as any`
- **Action**: Define proper channel types.

**F3.4 -- Policy creation `as any` casts**
- `src/pages/policies/CreatePolicyPage.tsx`, line 16: `{ ...values } as any`
- `src/components/PolicyWizard/index.tsx`, line 105: `body as any`
- **Action**: Align PolicyWizardValues type with CreatePolicyRequest.

**F3.5 -- DeploymentWizard zodResolver `as any`**
- `src/components/DeploymentWizard/index.tsx`, line 62: `zodResolver(deploymentWizardSchema) as any`
- Comment explains recursive TagExpression type mismatch. Legitimate but should be resolved.

**F3.6 -- Workflow editor `any` in node data**
- `src/pages/workflows/editor.tsx`, line 55: `function nodeData(node: { data: any })`
- **Action**: Type as `WorkflowNodeData`.

**F3.7 -- Deployment detail `any` casts**
- `src/pages/deployments/DeploymentDetailPage.tsx`, line 172: `const targetAny = target as any`
- `src/pages/deployments/components/DeploymentTimeline.tsx`, line 75: `(t as any).wave_id`
- **Action**: Extend deployment target type with `wave_id`, `output`, etc.

**F3.8 -- Audit `any` casts**
- `src/pages/audit/AuditPage.tsx`, line 156: `const e = ev as any`
- `src/pages/audit/ActivityStream.tsx`, lines 460-462: `event.payload as any`, `event.metadata as any`
- **Action**: Type audit event payload/metadata.

**F3.9 -- flows/policy-workflow config-panel**
- `src/flows/policy-workflow/panels/config-panel.tsx`, line 46: `type AnySave = (c: any) => void`
- **Action**: Define proper config type.

---

## 4. API Hook Coverage -- Direct API Calls

### Important

**F4.1 -- CommandPalette uses raw `fetch()` for search**
- File: `src/pages/dashboard/CommandPalette.tsx`, lines 299-305
- Three raw `fetch()` calls to `/api/v1/endpoints`, `/api/v1/patches`, `/api/v1/cves` bypassing openapi-fetch.
- **Action**: Create a search hook or use existing hooks with search params.

**F4.2 -- Endpoint CSV export uses raw `fetch()`**
- File: `src/pages/endpoints/export-csv.ts`, line 47
- `fetch('/api/v1/endpoints/export?...')` -- bypasses typed API client.
- **Action**: Add export endpoint to OpenAPI spec and use typed client.

**F4.3 -- Audit export uses raw `fetch()`**
- File: `src/pages/audit/AuditPage.tsx`, line 184
- `fetch('/api/v1/audit/export?...')` -- same pattern.
- **Action**: Add to OpenAPI spec.

**F4.4 -- flows/policy-workflow uses raw `fetch()` via fetchJSON helper**
- File: `src/flows/policy-workflow/hooks/fetch-json.ts`
- Custom `fetchJSON` and `fetchVoid` functions using raw `fetch()`, completely bypassing the typed API client.
- Used by `use-workflows.ts` and `use-workflow-executions.ts`.
- **Action**: Migrate to openapi-fetch client.

**F4.5 -- DeploymentWizard calls `api.GET` directly**
- File: `src/components/DeploymentWizard/index.tsx`, line 140
- `api.GET('/api/v1/endpoints', ...)` called directly inside component instead of through a hook.
- **Action**: Use `useEndpoints` hook.

---

## 5. Missing Error/Loading States

### Important

**F5.1 -- AccountSettingsPage -- no loading/error handling**
- File: `src/pages/settings/AccountSettingsPage.tsx`
- Reads auth context synchronously but has no fallback for missing user data.
- No loading state, no error state.

**F5.2 -- AppearanceSettingsPage -- no error handling**
- File: `src/pages/settings/AppearanceSettingsPage.tsx`
- Pure client-side (localStorage), so loading state is not needed, but has no error handling for `useTheme()` failures.

### Minor

**F5.3 -- NotificationsPage -- tab container has no error boundary**
- File: `src/pages/notifications/NotificationsPage.tsx`
- Parent tab container does not handle errors from child tabs (PreferencesTab/HistoryTab). If a child tab throws, the whole page crashes.
- Sub-tabs do have their own error handling, so this is low risk.

---

## 6. Routing Gaps

### Minor

**F6.1 -- Groups page has no route**
- `src/pages/groups/GroupsPage.tsx` exists with full implementation but no route in `routes.tsx`.
- Either the page should be deleted (see F1.3) or a route should be added.

**F6.2 -- Stale redirect routes**
- `/tags` redirects to `/settings/tags` -- fine
- `/notifications` redirects to `/settings/notifications` -- fine
- `/admin/roles` redirects to `/settings/roles` -- fine
- `/admin/users/roles` redirects to `/settings/user-roles` -- fine
- These are backward-compat redirects. Not a bug, but consider removing after a release cycle.

---

## 7. Stale Imports

### Minor

**F7.1 -- No broken imports detected**
- All imports resolve to existing files. The codebase compiles without import errors.
- The `as any` casts in hooks (F3.2) mask missing OpenAPI route types but are not broken imports per se.

---

## 8. Form Validation Gaps

### Important

**F8.1 -- PolicyWizard sub-steps lack local Zod validation**
- Files: `src/components/PolicyWizard/BasicsStep.tsx`, `TargetsStep.tsx`, `PatchesStep.tsx`, `ReviewStep.tsx`, `ImpactPreview.tsx`
- These use `useFormContext()` from the parent form (which has Zod) but do not validate per-step.
- The parent `PolicyWizard/index.tsx` has a Zod schema via `policyWizardSchema`.
- Per-step validation would improve UX (errors shown when navigating between steps).

**F8.2 -- DeploymentWizard sub-steps lack local Zod validation**
- Files: `src/components/DeploymentWizard/SourceStep.tsx`, `TargetsStep.tsx`, `StrategyStep.tsx`, `ReviewStep.tsx`, `ImpactPreview.tsx`
- Same pattern as PolicyWizard. Parent has Zod schema, but steps don't validate individually.

### Minor

**F8.3 -- UserRolesPage form uses inline validation only**
- File: `src/pages/admin/users/UserRolesPage.tsx`, line 163
- User ID input has no Zod schema; validated via `useState` + manual checks.
- Low risk (admin-only page).

---

## 9. Component Quality / Duplication

### Important

**F9.1 -- Duplicate EmptyState: local vs @patchiq/ui**
- `src/components/EmptyState.tsx` -- local implementation (icon + title + description + action button)
- `@patchiq/ui` exports its own `EmptyState` component
- 5 files import the local version: `CVEsPage.tsx`, `HistoryTab.tsx`, `ComplianceTab.tsx`, `AuditTab.tsx`, `ComponentPreview.tsx`
- 6 files import from `@patchiq/ui`: `CompliancePage`, `TagsPage`, `WorkflowsPage`, `PatchesPage`, `PatchDetailPage`, `AlertsPage`
- **Action**: Migrate all to `@patchiq/ui` EmptyState, delete local copy.

**F9.2 -- Duplicate ErrorAlert vs @patchiq/ui ErrorState**
- `src/components/ErrorAlert.tsx` -- local error banner (1 consumer: `PatchDetailPage.tsx`)
- `@patchiq/ui` exports `ErrorState` (used by 5+ pages)
- **Action**: Replace ErrorAlert usage with ErrorState, delete local component.

**F9.3 -- PageHeader component exists but almost never used**
- `src/components/PageHeader.tsx` exists
- Only used in `ComponentPreview.tsx` (dev preview page)
- 20+ pages implement their own `<h1>` headers with inline styles
- **Action**: Adopt PageHeader (or `@patchiq/ui` PageHeader) across all pages for consistency.

**F9.4 -- Massive inline style usage**
- 20+ page files have >50 `style={{...}}` occurrences each.
- Top offenders: `PatchDetailPage.tsx` (186), `CVEDetailPage.tsx` (174), `EndpointsPage.tsx` (150), `HardwareTab.tsx` (146)
- The project uses Tailwind CSS but many pages use raw inline styles (CSS variables via `var(--...)` pattern).
- **Action**: Gradual migration to Tailwind utility classes. Not blocking but hurts maintainability.

### Minor

**F9.5 -- Two ProgressBar implementations**
- `src/components/ProgressBar.tsx` -- used by `HardwareTab.tsx`
- `src/pages/compliance/components/progress-bar.tsx` -- used by compliance components
- `@patchiq/ui` has a `Progress` component
- **Action**: Consolidate to one implementation.

---

## Summary

| Severity | Count | Category |
|----------|-------|----------|
| Critical | 2 | TypeScript `any` epidemic in compliance + API hooks |
| Important | 18 | Dead files, raw fetch, duplicate components, form gaps, incomplete features |
| Minor | 10 | Unused components, inline styles, redirects |

### Top 5 Actions by Impact

1. **Regenerate OpenAPI types** to cover compliance, dashboard, IAM, settings, roles, channels, and custom-framework endpoints. This eliminates 50+ `as any` casts across hooks and pages (F3.1, F3.2).
2. **Delete ~20 dead files** totaling ~3,000+ lines of dead code (F1.1-F1.17).
3. **Consolidate EmptyState/ErrorAlert** to `@patchiq/ui` versions (F9.1, F9.2).
4. **Migrate raw `fetch()` calls** in CommandPalette, export functions, and policy-workflow to the typed API client (F4.1-F4.5).
5. **Fix incomplete features** -- AddToGroupDialog and Patch Recall button either need backend support or should be hidden (F2.2, F2.3).
