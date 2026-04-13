# Role-Based Dashboard Templates — Implementation Plan
**Design doc:** `2026-04-08-role-based-dashboards.md`
**Track:** Standard | **TDD required for all tasks**

---

## Phase 1 — Backend Foundation

### Task 1.1 — Migration 054: dashboard template tables
**File:** `internal/server/store/migrations/054_dashboard_templates.sql`
- Write goose Up/Down migration
- Tables: `dashboard_templates`, `dashboard_template_assignments`, `dashboard_user_overrides`
- RLS policies on all three tables (same pattern as migration 014)
- JSONB column `widget_positions` with `DEFAULT '{}'::jsonb` (same pattern as migration 030)
- `UNIQUE (tenant_id, name)` on `dashboard_templates`
- **Failing test first:** `TestMigration054_Up` — verify tables exist, RLS enforced
- Run `make migrate` and `make migrate-status` to verify

### Task 1.2 — sqlc queries for dashboard templates
**File:** `internal/server/store/queries/dashboard_templates.sql`
- Write all queries from design doc (CreateDashboardTemplate, GetDashboardTemplate, ListDashboardTemplates, UpdateDashboardTemplate, DeleteDashboardTemplate, CountDashboardTemplates, CreateDashboardAssignment, ListTemplateAssignments, DeleteDashboardAssignment, GetMyTemplates, UpsertLayoutOverride, GetLayoutOverride)
- Run `make sqlc` — verify `internal/server/store/sqlcgen/dashboard_templates.sql.go` is generated with no errors
- No test here — sqlc generation is the verification

### Task 1.3 — System template definitions (Go constants)
**File:** `internal/server/api/v1/dashboard_templates.go` (new file)
- Define `systemTemplates` slice with the 4 system templates (IT Admin, Security Analyst, Executive, Auditor)
- Each entry: name, description, widget_ids, widget_positions (use registry defaults), is_system=true
- Widget positions must use the same breakpoint shape (`lg`, `md`, `sm`) as the frontend `ResponsiveLayouts`
- **Failing test first:** `TestSystemTemplates_AllHaveValidWidgetIDs` — validate every widget_id in system templates is one of the 16 known IDs (defined as a const list matching frontend `WidgetId` type)
- **Failing test first:** `TestSystemTemplates_PositionsMatchWidgetIDs` — widget_positions keys match widget_ids list
- These tests are pure Go unit tests, no DB needed

### Task 1.4 — DashboardTemplateHandler: seedSystemTemplates
**File:** `internal/server/api/v1/dashboard_templates.go`
- `seedSystemTemplates(ctx, tenantID)` — wraps `BeginTx`, loops `CreateDashboardTemplate` for each system template, commits
- Idempotent: uses `ON CONFLICT DO NOTHING` (or check `CountDashboardTemplates` first)
- **Failing test first:** `TestSeedSystemTemplates` (integration, testcontainers) — call seed twice, assert exactly 4 templates exist with correct names, `is_system=true`
- **Failing test first:** `TestSeedSystemTemplates_IdempotentOnRepeat` — seeding a tenant with existing templates does not add duplicates

### Task 1.5 — DashboardTemplateHandler: template CRUD
**File:** `internal/server/api/v1/dashboard_templates.go`
- `ListTemplates` — `GET /api/v1/dashboard/templates` — returns all templates for tenant
- `CreateTemplate` — `POST /api/v1/dashboard/templates` — validates name/widget_ids, inserts, emits `dashboard_template.created` event
- `GetTemplate` — `GET /api/v1/dashboard/templates/:id`
- `UpdateTemplate` — `PUT /api/v1/dashboard/templates/:id` — blocks if `is_system=true`, emits `dashboard_template.updated`
- `DeleteTemplate` — `DELETE /api/v1/dashboard/templates/:id` — blocks if `is_system=true`, emits `dashboard_template.deleted`
- Validation: `widget_ids` must be non-empty; each ID must be in the allowed set
- **Failing test first (per endpoint):**
  - `TestListTemplates_ReturnsTenantTemplatesOnly`
  - `TestCreateTemplate_Success`
  - `TestCreateTemplate_DuplicateNameReturns409`
  - `TestUpdateTemplate_BlockedForSystemTemplate`
  - `TestDeleteTemplate_BlockedForSystemTemplate`
  - `TestGetTemplate_NotFoundReturns404`

### Task 1.6 — DashboardTemplateHandler: assignments
**File:** `internal/server/api/v1/dashboard_templates.go`
- `ListAssignments` — `GET /api/v1/dashboard/templates/:id/assignments`
- `CreateAssignment` — `POST /api/v1/dashboard/templates/:id/assignments` — body: `{ assignee_type, assignee_id, display_order, is_default }`
- `DeleteAssignment` — `DELETE /api/v1/dashboard/templates/:id/assignments/:aid`
- **Failing test first:**
  - `TestCreateAssignment_RoleType`
  - `TestCreateAssignment_UserType`
  - `TestCreateAssignment_UpsertOnConflict`
  - `TestDeleteAssignment_NotOwnedByTenantReturns404`

### Task 1.7 — DashboardTemplateHandler: GetMyTemplates
**File:** `internal/server/api/v1/dashboard_templates.go`
- `GetMyTemplates` — `GET /api/v1/dashboard/my-templates`
- Resolves current user ID from `X-User-ID` header, fetches their role IDs from `user_roles` table (server-side resolution — no frontend change needed)
- Calls `CountDashboardTemplates`; if 0 seeds system templates first
- Calls `GetMyTemplates` sqlc query with `(tenant_id, user_id, role_ids)`
- Fetches `GetLayoutOverride` for each returned template and merges into response
- Returns `[]MyTemplateEntry` with template + assignment metadata + user_override (nullable)
- **Failing test first:**
  - `TestGetMyTemplates_SeedsOnFirstCall`
  - `TestGetMyTemplates_ReturnsRoleAssignedTemplates`
  - `TestGetMyTemplates_UserAssignmentOverridesRole`
  - `TestGetMyTemplates_DeduplicatesWhenRoleAndUserBothAssigned`
  - `TestGetMyTemplates_EmptyWhenNoAssignments`

### Task 1.8 — DashboardTemplateHandler: SaveLayoutOverride
**File:** `internal/server/api/v1/dashboard_templates.go`
- `SaveLayoutOverride` — `PUT /api/v1/dashboard/my-templates/:id/layout`
- Validates template exists for tenant and belongs to current user's assignments
- Checks `template.allow_customization = true`; returns 403 if false
- Upserts `dashboard_user_overrides`
- **No event emitted** (personal preference, not auditable business action)
- **Failing test first:**
  - `TestSaveLayoutOverride_Success`
  - `TestSaveLayoutOverride_BlockedWhenCustomizationDisabled`
  - `TestSaveLayoutOverride_BlockedForUnassignedTemplate`

### Task 1.9 — Router registration
**File:** `internal/server/api/router.go`
- Instantiate `DashboardTemplateHandler` (same pattern as other handlers)
- Add routes inside `r.Route("/dashboard", ...)` block (additive only — existing routes untouched)
- Add `GET /my-templates` and `PUT /my-templates/{id}/layout` (auth-only, no rp() wrapper)
- Add `r.Route("/templates", ...)` with rp() per endpoint
- **Failing test first:** `TestRouter_DashboardTemplateRoutesExist` — verify 200/401/403 on each new route

---

## Phase 2 — Frontend Template Loader

### Task 2.1 — API hook: useMyTemplates
**File:** `web/src/api/hooks/useDashboardTemplates.ts` (new file)
- `useMyTemplates()` — `GET /api/v1/dashboard/my-templates`, queryKey `['dashboard', 'my-templates']`, staleTime 2min
- `useTemplates()` — `GET /api/v1/dashboard/templates`, queryKey `['dashboard', 'templates']`
- `useCreateTemplate()` — mutation, invalidates `['dashboard', 'templates']` on success
- `useUpdateTemplate()` — mutation, invalidates `['dashboard', 'templates']` and specific template
- `useDeleteTemplate()` — mutation, invalidates `['dashboard', 'templates']`
- `useTemplateAssignments(templateId)` — queryKey `['dashboard', 'templates', templateId, 'assignments']`
- `useAssignTemplate()` — mutation, invalidates assignments
- `useRemoveAssignment()` — mutation, invalidates assignments
- `useSaveLayoutOverride()` — mutation `PUT /my-templates/:id/layout`, invalidates `['dashboard', 'my-templates']`
- **Failing test first:** `web/src/__tests__/api/hooks/useDashboardTemplates.test.ts`
  - Mock API, test `useMyTemplates` returns data
  - Test `useCreateTemplate` invalidates query cache
  - Test `useSaveLayoutOverride` sends correct body

### Task 2.2 — TemplateDashboardContent component
**File:** `web/src/pages/dashboard/TemplateDashboardContent.tsx`
- Accepts `templates: MyTemplateEntry[]` prop
- Manages `activeTemplateId` in `sessionStorage` (persists across refresh, not across tabs)
- For the active template, resolves effective layout: `user_override ?? template.widget_positions`
- Renders same widget grid as existing `DashboardPage` (`ReactGridLayout`, `WidgetWrapper`, `AnimatePresence`)
- When `allow_customization=true` and `isEditing`: calls `useSaveLayoutOverride` on layout change (debounced 400ms — same as `useDashboardLayout`)
- When `allow_customization=false`: hides Customize button
- Does NOT import or call `useDashboardLayout` — completely separate code path
- **Failing test first:** `web/src/__tests__/pages/dashboard/TemplateDashboardContent.test.tsx`
  - Renders widgets from template widget_ids
  - Customize button hidden when allow_customization=false
  - Customize button visible when allow_customization=true
  - Active template persists to sessionStorage

### Task 2.3 — TemplateSwitcher component
**File:** `web/src/pages/dashboard/TemplateSwitcher.tsx`
- Accepts `templates: MyTemplateEntry[]`, `activeId: string`, `onSwitch: (id: string) => void`
- Renders nothing if `templates.length <= 1`
- Renders tab strip (use existing `Tabs` from `@patchiq/ui`) otherwise
- **Failing test first:** `web/src/__tests__/pages/dashboard/TemplateSwitcher.test.tsx`
  - Returns null with 1 template
  - Renders N tabs with N templates
  - Calls onSwitch with correct id on tab click

### Task 2.4 — DashboardPage conditional branch
**File:** `web/src/pages/dashboard/DashboardPage.tsx`
- Add `useMyTemplates()` call at top of component
- If `isLoading`: render existing skeleton (no change)
- If `myTemplates && myTemplates.length > 0`: render `<TemplateDashboardContent templates={myTemplates} />`
- Otherwise (no templates, error, or empty): render existing content unchanged
- Add `<TemplateSwitcher>` above the grid when templates are active
- **Failing test first:** `web/src/__tests__/pages/dashboard/DashboardPage.test.tsx`
  - When `useMyTemplates` returns empty, renders existing localStorage-based grid
  - When `useMyTemplates` returns 1 template, renders TemplateDashboardContent
  - When `useMyTemplates` returns 2 templates, renders TemplateSwitcher

---

## Phase 3 — Admin Settings UI

### Task 3.1 — DashboardsSettingsPage (template list)
**File:** `web/src/pages/settings/DashboardsSettingsPage.tsx`
- Lists all templates in a table: Name, Type (System/Custom), Widgets (count), Customizable, Actions
- "New Template" button (opens create flow)
- Row click → expand inline or navigate to edit
- Delete button (disabled for system templates, tooltip: "System templates cannot be deleted")
- Follows same layout pattern as `AppearanceSettingsPage`: `padding: '28px 40px 80px'`, `maxWidth: 860px`
- **Failing test first:** `web/src/__tests__/pages/settings/DashboardsSettingsPage.test.tsx`
  - Renders template list from `useTemplates`
  - Delete disabled for system templates
  - "New Template" button visible

### Task 3.2 — Template create/edit form
**File:** `web/src/pages/settings/DashboardsSettingsPage.tsx` (same file, inline panel or dialog)
- Fields: Name (required), Description, Allow Customization toggle
- Widget selector: checklist of the 16 available widgets grouped by category (kpi, security, operations, activity) — uses `WIDGET_REGISTRY` categories from frontend registry
- On save: calls `useCreateTemplate` or `useUpdateTemplate`, shows toast, refreshes list
- **Failing test first:**
  - Name required validation
  - At least one widget required validation
  - System template fields are read-only

### Task 3.3 — Assignment panel
**File:** `web/src/pages/settings/DashboardsSettingsPage.tsx` (expanded section per template)
- Shows current assignments (role/user chips with remove button)
- "Add Assignment" button → popover with: Type (Role/User) selector + searchable assignee dropdown
- Display order field (number input) for switcher ordering
- Is Default checkbox
- Uses `useRoles` hook (already exists) to populate role dropdown
- **Failing test first:**
  - Renders existing assignments
  - Add assignment calls useAssignTemplate
  - Remove assignment calls useRemoveAssignment

### Task 3.4 — "Save as Template" in Customize mode
**File:** `web/src/pages/dashboard/DashboardPage.tsx`
- Add "Save as Template" button to the Customize toolbar (only visible when `isEditing` and user has admin role)
- `hasPermission` check using the new `lib/permissions.ts` helper
- Clicking opens `SaveTemplateDialog`
- **File:** `web/src/pages/dashboard/SaveTemplateDialog.tsx`
  - Name + description fields
  - Allow customization toggle
  - On submit: calls `useCreateTemplate` with current `activeWidgets` + `layouts`
  - Shows success toast with link to `/settings/dashboards`
- **Failing test first:**
  - Button hidden for non-admin
  - Button visible for admin
  - Dialog submits with current layout data

### Task 3.5 — Settings sidebar + route registration
**File:** `web/src/pages/settings/SettingsSidebar.tsx`
- Add `{ to: '/settings/dashboards', label: 'Dashboards', icon: LayoutDashboard }` to `adminItems`
- Import `LayoutDashboard` from `lucide-react`

**File:** `web/src/app/routes.tsx`
- Add `{ path: '/settings/dashboards', element: <DashboardsSettingsPage /> }` to AppLayout children
- Add import for `DashboardsSettingsPage`
- **Failing test first:** `web/src/__tests__/app/routes.test.tsx` (or snapshot test) — `/settings/dashboards` route resolves without 404

---

## Phase 4 — Template Switcher (active template UX)

### Task 4.1 — sessionStorage active template persistence
**File:** `web/src/pages/dashboard/TemplateDashboardContent.tsx`
- On mount, read `sessionStorage.getItem('patchiq-active-template-id')`
- If stored ID is in `templates` list, set as active
- Otherwise, set `is_default=true` template as active (or first)
- On switch: write to `sessionStorage`
- **Failing test first:**
  - Reads sessionStorage on mount
  - Falls back to default template if stored ID not in list
  - Writes to sessionStorage on switch

### Task 4.2 — E2E: template switcher full flow
**File:** `web/src/__tests__/pages/dashboard/DashboardPage.test.tsx`
- Test: user with 2 templates assigned sees TemplateSwitcher
- Test: switching tab changes rendered widgets
- Test: switching tab updates sessionStorage
- Test: after switch, refresh renders the previously active tab

---

## Regression Safety Checklist

Run after each phase before merging:

- [ ] `make test` passes (Go unit + race detector)
- [ ] `make test-integration` passes (DB tests)
- [ ] `make sqlc` generates without errors
- [ ] `make migrate-status` shows migration 054 applied
- [ ] Existing dashboard route `/` still renders for a user with **no** templates assigned
- [ ] `useDashboardLayout` localStorage read/write still works (no modifications to that hook)
- [ ] `make lint` passes (no orphaned imports)
- [ ] `make lint-frontend` passes
- [ ] All existing `/api/v1/dashboard/*` endpoints return same responses as before
- [ ] `web/src/api/types.ts` regenerated after OpenAPI spec updated (`make api-client`)

---

## Files Created (new — no existing files modified except where noted)

**New:**
- `internal/server/store/migrations/054_dashboard_templates.sql`
- `internal/server/store/queries/dashboard_templates.sql`
- `internal/server/api/v1/dashboard_templates.go`
- `internal/server/api/v1/dashboard_templates_test.go`
- `web/src/api/hooks/useDashboardTemplates.ts`
- `web/src/pages/dashboard/TemplateDashboardContent.tsx`
- `web/src/pages/dashboard/TemplateSwitcher.tsx`
- `web/src/pages/dashboard/SaveTemplateDialog.tsx`
- `web/src/pages/settings/DashboardsSettingsPage.tsx`
- `web/src/lib/permissions.ts`
- `web/src/__tests__/api/hooks/useDashboardTemplates.test.ts`
- `web/src/__tests__/pages/dashboard/TemplateDashboardContent.test.tsx`
- `web/src/__tests__/pages/dashboard/TemplateSwitcher.test.tsx`
- `web/src/__tests__/pages/settings/DashboardsSettingsPage.test.tsx`

**Modified (additive changes only):**
- `internal/server/api/router.go` — add handler instantiation + routes inside existing `/dashboard` block
- `internal/server/events/topics.go` — add 3 new event type constants
- `web/src/pages/dashboard/DashboardPage.tsx` — add conditional branch + TemplateSwitcher (existing branch untouched)
- `web/src/app/routes.tsx` — add 1 route to AppLayout children
- `web/src/pages/settings/SettingsSidebar.tsx` — add 1 item to adminItems
- `web/src/api/types.ts` — regenerated (do not hand-edit)
- `internal/server/store/sqlcgen/` — regenerated (do not hand-edit)

**Not modified:**
- `web/src/pages/dashboard/hooks/useDashboardLayout.ts`
- `web/src/pages/dashboard/registry.ts`
- `web/src/pages/dashboard/types.ts`
- All existing widget components
- All existing dashboard API handlers
- All existing migrations 001–053
