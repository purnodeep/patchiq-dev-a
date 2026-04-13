# Role-Based Dashboard Templates
**Date:** 2026-04-08
**Status:** Design — pending plan
**Track:** Standard

---

## Problem

The current dashboard (`/`) gives every user the same 15-widget layout, persisted in localStorage. There is no way for an admin to control what different users see, and the layout is device-local (not shared across machines). As the platform is deployed to enterprise clients with distinct personas (IT Admin, Security Analyst, Executive, Auditor), a one-size-fits-all dashboard fails: it shows noisy operational widgets to executives and buries CVE data for security analysts.

---

## Goals

- Admin can create named **dashboard templates** (curated widget sets + layouts).
- Admin can assign templates to **roles** or individual **users** via the existing RBAC system.
- A user with **one template assigned** sees a single dashboard — identical UX to today.
- A user with **multiple templates assigned** sees a template **switcher** in the dashboard header.
- Admin can toggle whether a template allows **personal customization** by the user.
- Four **system templates** are seeded per tenant on first use (IT Admin, Security Analyst, Executive, Auditor).
- **Zero breaking changes** to existing behavior: users with no template assigned fall back to the current localStorage-based layout.

## Non-Goals

- Widget-level permission gating (showing/hiding individual widgets per role) — future iteration.
- Dashboard sharing between users — future iteration.
- Mobile-specific dashboard layouts — future iteration.
- Changing the widget components themselves — untouched.
- Changing any existing dashboard API endpoints (`/summary`, `/activity`, `/blast-radius`, `/endpoints-risk`) — untouched.

---

## Design

### Mental Model

```
Template   = named widget set + grid layout (the "what")
Assignment = role or user → template(s) (the "who")
```

The admin creates templates using an extended version of the existing Customize mode, then assigns them to roles from the Role management page. Users see whichever templates are assigned to their roles (union, deduplicated). User-level assignments override role-level assignments when set.

### User Experience

**Single template (most users):**
- Dashboard loads their assigned template's widget set and positions.
- If `allow_customization = true`, the Customize button works and saves a personal override layer on top of the template.
- If `allow_customization = false`, Customize button is hidden.

**Multiple templates (power users/admins):**
- A template switcher tab strip appears in the dashboard header between the title and toolbar.
- Active tab is the `is_default` template for their role.
- Switching tabs is instant (client-side — all templates are pre-fetched on mount).

**No template assigned (fallback):**
- Existing localStorage behavior unchanged — `useDashboardLayout` reads/writes `patchiq-dashboard-layout-v13` exactly as it does today.
- This ensures zero regression for existing users.

**Admin (template builder):**
- In Customize mode, a new "Save as Template" button appears (only for users with `dashboard_templates:manage` permission).
- Clicking opens a dialog: enter template name + description + toggle allow_customization.
- After saving, template appears in `/settings/dashboards`.

---

## Data Model

### Migration 054 — `dashboard_templates`

```sql
-- +goose Up

CREATE TABLE dashboard_templates (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name                 TEXT NOT NULL,
    description          TEXT NOT NULL DEFAULT '',
    widget_ids           TEXT[] NOT NULL DEFAULT '{}',
    widget_positions     JSONB NOT NULL DEFAULT '{}'::jsonb,
    is_system            BOOLEAN NOT NULL DEFAULT false,
    allow_customization  BOOLEAN NOT NULL DEFAULT true,
    created_by           UUID,                           -- NULL for system templates
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, name)
);

CREATE TABLE dashboard_template_assignments (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    template_id    UUID NOT NULL REFERENCES dashboard_templates(id) ON DELETE CASCADE,
    assignee_type  TEXT NOT NULL CHECK (assignee_type IN ('role', 'user')),
    assignee_id    UUID NOT NULL,
    display_order  INT NOT NULL DEFAULT 0,
    is_default     BOOLEAN NOT NULL DEFAULT false,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, template_id, assignee_type, assignee_id)
);

CREATE TABLE dashboard_user_overrides (
    tenant_id        UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id          UUID NOT NULL,
    template_id      UUID NOT NULL REFERENCES dashboard_templates(id) ON DELETE CASCADE,
    widget_ids       TEXT[] NOT NULL DEFAULT '{}',
    widget_positions JSONB NOT NULL DEFAULT '{}'::jsonb,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, user_id, template_id)
);

-- RLS: tenant isolation
ALTER TABLE dashboard_templates ENABLE ROW LEVEL SECURITY;
CREATE POLICY dashboard_templates_tenant_isolation ON dashboard_templates
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

ALTER TABLE dashboard_template_assignments ENABLE ROW LEVEL SECURITY;
CREATE POLICY dashboard_template_assignments_tenant_isolation ON dashboard_template_assignments
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

ALTER TABLE dashboard_user_overrides ENABLE ROW LEVEL SECURITY;
CREATE POLICY dashboard_user_overrides_tenant_isolation ON dashboard_user_overrides
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- +goose Down
DROP TABLE IF EXISTS dashboard_user_overrides;
DROP TABLE IF EXISTS dashboard_template_assignments;
DROP TABLE IF EXISTS dashboard_templates;
```

### `widget_positions` JSONB shape

Matches the existing `ResponsiveLayouts` type from `useDashboardLayout`:

```json
{
  "lg": [{ "i": "stat-cards-row-1", "x": 0, "y": 0, "w": 12, "h": 2 }],
  "md": [...],
  "sm": [...]
}
```

---

## System Templates (seeded via Go on first API call per tenant)

Seeded lazily on first `GET /api/v1/dashboard/my-templates` when the tenant has zero templates. This avoids a migration data dependency on tenant IDs.

| Template | `is_system` | `allow_customization` | Widget IDs |
|----------|-------------|----------------------|------------|
| IT Admin | true | true | `stat-cards-row-1`, `stat-cards-row-2`, `deployment-pipeline`, `activity-feed`, `patch-velocity`, `blast-radius`, `quick-actions`, `os-heatmap` |
| Security Analyst | true | true | `stat-cards-row-1`, `top-vulnerabilities`, `blast-radius`, `compliance-rings`, `risk-projection`, `risk-landscape` |
| Executive | true | false | `stat-cards-row-1`, `compliance-rings`, `sla-status`, `patch-velocity` |
| Auditor | true | false | `compliance-rings`, `activity-feed`, `sla-status`, `sla-countdown` |

---

## API Surface

All new endpoints live under `/api/v1/dashboard/templates` and `/api/v1/dashboard/my-templates`. Existing endpoints are **not modified**.

### RBAC permissions (new)

| Resource | Action | Who |
|----------|--------|-----|
| `dashboard_templates` | `manage` | Admin — create/edit/delete templates |
| `dashboard_templates` | `assign` | Admin — assign templates to roles/users |
| `dashboard_templates` | `read` | All authenticated users (view their own templates) |

### Endpoints

```
# Template CRUD (requires dashboard_templates:manage)
GET    /api/v1/dashboard/templates
POST   /api/v1/dashboard/templates
GET    /api/v1/dashboard/templates/:id
PUT    /api/v1/dashboard/templates/:id
DELETE /api/v1/dashboard/templates/:id        -- blocked if is_system=true

# Assignments (requires dashboard_templates:assign)
GET    /api/v1/dashboard/templates/:id/assignments
POST   /api/v1/dashboard/templates/:id/assignments
DELETE /api/v1/dashboard/templates/:id/assignments/:assignmentId

# Current user's templates (requires dashboard_templates:read — all users)
GET    /api/v1/dashboard/my-templates
       → returns []Template resolved from user's roles + user-level overrides
       → seeds system templates if tenant has none

# Personal layout override (no extra permission — own data only)
PUT    /api/v1/dashboard/my-templates/:id/layout
       → upserts dashboard_user_overrides for (tenant_id, user_id, template_id)
       → only allowed when template.allow_customization = true
```

### Response shapes

**Template:**
```json
{
  "id": "uuid",
  "name": "IT Admin",
  "description": "Operational view for IT administrators",
  "widget_ids": ["stat-cards-row-1", "deployment-pipeline"],
  "widget_positions": { "lg": [...], "md": [...], "sm": [...] },
  "is_system": true,
  "allow_customization": true,
  "created_by": null,
  "created_at": "2026-04-08T00:00:00Z"
}
```

**My-templates entry:**
```json
{
  "template": { ...Template },
  "assignment": {
    "display_order": 0,
    "is_default": true,
    "assignee_type": "role"
  },
  "user_override": {          // null if no personal customization saved
    "widget_ids": [...],
    "widget_positions": { ... },
    "updated_at": "..."
  }
}
```

---

## Frontend Architecture

### No breaking changes strategy

`useDashboardLayout` is **not modified**. A new hook `useTemplateDashboardLayout` wraps the same interface but sources data from the server. `DashboardPage` gains a conditional at the top:

```tsx
// DashboardPage.tsx — new logic, old path untouched
const { data: myTemplates, isLoading } = useMyTemplates();
const hasTemplates = myTemplates && myTemplates.length > 0;

// If no templates → render existing DashboardContent (localStorage path)
// If templates    → render TemplateDashboardContent
```

This means every existing test passes unchanged. The new code path is only entered when the server returns templates.

### New components

| Component | Location | Purpose |
|-----------|----------|---------|
| `TemplateDashboardContent` | `pages/dashboard/TemplateDashboardContent.tsx` | Template-aware wrapper, replaces localStorage hooks with server state |
| `TemplateSwitcher` | `pages/dashboard/TemplateSwitcher.tsx` | Tab strip shown when user has 2+ templates |
| `SaveTemplateDialog` | `pages/dashboard/SaveTemplateDialog.tsx` | Dialog triggered by "Save as Template" in Customize mode |
| `DashboardsSettingsPage` | `pages/settings/DashboardsSettingsPage.tsx` | Admin template management |
| `TemplateAssignmentPanel` | `pages/settings/DashboardsSettingsPage.tsx` | Role/user assignment UI within settings page |

### New hooks

| Hook | File | Purpose |
|------|------|---------|
| `useMyTemplates()` | `api/hooks/useDashboardTemplates.ts` | `GET /my-templates` — current user's templates |
| `useTemplates()` | same | `GET /templates` — admin list |
| `useCreateTemplate()` | same | `POST /templates` |
| `useUpdateTemplate()` | same | `PUT /templates/:id` |
| `useDeleteTemplate()` | same | `DELETE /templates/:id` |
| `useTemplateAssignments()` | same | `GET /templates/:id/assignments` |
| `useAssignTemplate()` | same | `POST /templates/:id/assignments` |
| `useRemoveAssignment()` | same | `DELETE /templates/:id/assignments/:assignmentId` |
| `useSaveLayoutOverride()` | same | `PUT /my-templates/:id/layout` |

### Routes added (non-breaking)

```tsx
// routes.tsx — add inside AppLayout children
{ path: '/settings/dashboards', element: <DashboardsSettingsPage /> },
```

### Settings sidebar (non-breaking)

```tsx
// SettingsSidebar.tsx — add to adminItems
{ to: '/settings/dashboards', label: 'Dashboards', icon: LayoutDashboard },
```

### Permission helper (new utility)

Since `AuthContext` has no `hasPermission` helper:

```tsx
// lib/permissions.ts
export function hasPermission(user: AuthUser, resource: string, action: string): boolean {
  // roles checked server-side; this is UI-only gating for showing/hiding controls
  // For now: 'admin' role or roles containing the resource grant access
  // Real enforcement happens via RBAC middleware on the backend
  return (user.roles ?? []).some(r => r === 'admin' || r === 'superadmin');
}
```

Admin-only UI elements (Save as Template button, /settings/dashboards nav item) are gated by this check. Backend enforces correctly regardless.

---

## Backend Architecture

### Handler: `DashboardTemplateHandler`

New file: `internal/server/api/v1/dashboard_templates.go`

```go
type DashboardTemplateHandler struct {
    q        *sqlcgen.Queries
    eventBus domain.EventBus
}

// Methods:
// ListTemplates, CreateTemplate, GetTemplate, UpdateTemplate, DeleteTemplate
// ListAssignments, CreateAssignment, DeleteAssignment
// GetMyTemplates, SaveLayoutOverride
```

### Router registration (additive only)

```go
// router.go — inside r.Route("/dashboard", ...) block
r.Route("/templates", func(r chi.Router) {
    r.With(rp("dashboard_templates", "read")).Get("/", th.ListTemplates)
    r.With(rp("dashboard_templates", "manage")).Post("/", th.CreateTemplate)
    r.With(rp("dashboard_templates", "read")).Get("/{id}", th.GetTemplate)
    r.With(rp("dashboard_templates", "manage")).Put("/{id}", th.UpdateTemplate)
    r.With(rp("dashboard_templates", "manage")).Delete("/{id}", th.DeleteTemplate)
    r.With(rp("dashboard_templates", "assign")).Get("/{id}/assignments", th.ListAssignments)
    r.With(rp("dashboard_templates", "assign")).Post("/{id}/assignments", th.CreateAssignment)
    r.With(rp("dashboard_templates", "assign")).Delete("/{id}/assignments/{aid}", th.DeleteAssignment)
})
r.Get("/my-templates", th.GetMyTemplates)          // auth only, no extra permission
r.Put("/my-templates/{id}/layout", th.SaveLayout)  // auth only, owns own data
```

### sqlc queries: `internal/server/store/queries/dashboard_templates.sql`

```sql
-- name: CreateDashboardTemplate :one
INSERT INTO dashboard_templates (tenant_id, name, description, widget_ids, widget_positions, is_system, allow_customization, created_by)
VALUES (@tenant_id, @name, @description, @widget_ids, @widget_positions, @is_system, @allow_customization, @created_by)
RETURNING *;

-- name: GetDashboardTemplate :one
SELECT * FROM dashboard_templates WHERE id = @id AND tenant_id = @tenant_id;

-- name: ListDashboardTemplates :many
SELECT * FROM dashboard_templates WHERE tenant_id = @tenant_id ORDER BY is_system DESC, name ASC;

-- name: UpdateDashboardTemplate :one
UPDATE dashboard_templates
SET name = @name, description = @description, widget_ids = @widget_ids,
    widget_positions = @widget_positions, allow_customization = @allow_customization,
    updated_at = now()
WHERE id = @id AND tenant_id = @tenant_id AND is_system = false
RETURNING *;

-- name: DeleteDashboardTemplate :exec
DELETE FROM dashboard_templates WHERE id = @id AND tenant_id = @tenant_id AND is_system = false;

-- name: CountDashboardTemplates :one
SELECT count(*) FROM dashboard_templates WHERE tenant_id = @tenant_id;

-- name: CreateDashboardAssignment :one
INSERT INTO dashboard_template_assignments (tenant_id, template_id, assignee_type, assignee_id, display_order, is_default)
VALUES (@tenant_id, @template_id, @assignee_type, @assignee_id, @display_order, @is_default)
ON CONFLICT (tenant_id, template_id, assignee_type, assignee_id) DO UPDATE
    SET display_order = EXCLUDED.display_order, is_default = EXCLUDED.is_default
RETURNING *;

-- name: ListTemplateAssignments :many
SELECT * FROM dashboard_template_assignments WHERE template_id = @template_id AND tenant_id = @tenant_id;

-- name: DeleteDashboardAssignment :exec
DELETE FROM dashboard_template_assignments WHERE id = @id AND tenant_id = @tenant_id;

-- name: GetMyTemplates :many
-- Returns all templates assigned to the current user (via role or direct user assignment),
-- unioned and deduplicated, ordered by display_order.
SELECT DISTINCT ON (dt.id)
    dt.*,
    dta.display_order,
    dta.is_default,
    dta.assignee_type
FROM dashboard_templates dt
JOIN dashboard_template_assignments dta ON dta.template_id = dt.id AND dta.tenant_id = dt.tenant_id
WHERE dt.tenant_id = @tenant_id
  AND (
      (dta.assignee_type = 'user' AND dta.assignee_id = @user_id)
      OR
      (dta.assignee_type = 'role' AND dta.assignee_id = ANY(@role_ids::uuid[]))
  )
ORDER BY dt.id, dta.assignee_type DESC, dta.display_order ASC;

-- name: UpsertLayoutOverride :one
INSERT INTO dashboard_user_overrides (tenant_id, user_id, template_id, widget_ids, widget_positions, updated_at)
VALUES (@tenant_id, @user_id, @template_id, @widget_ids, @widget_positions, now())
ON CONFLICT (tenant_id, user_id, template_id) DO UPDATE
    SET widget_ids = EXCLUDED.widget_ids,
        widget_positions = EXCLUDED.widget_positions,
        updated_at = now()
RETURNING *;

-- name: GetLayoutOverride :one
SELECT * FROM dashboard_user_overrides
WHERE tenant_id = @tenant_id AND user_id = @user_id AND template_id = @template_id;

-- name: InsertSystemTemplates :exec
-- Called once per tenant when count = 0. Params are passed as a batch from Go.
-- Handled in Go via loop over CreateDashboardTemplate.
```

### System template seeding

```go
// In GetMyTemplates handler, before resolving assignments:
count, _ := h.q.CountDashboardTemplates(ctx, tenantID)
if count == 0 {
    h.seedSystemTemplates(ctx, tenantID)
}
```

`seedSystemTemplates` inserts the 4 system templates in a transaction. Idempotent due to `UNIQUE (tenant_id, name)`.

### Domain events

```go
// Template created
emitEventWithActor(ctx, h.eventBus, "dashboard_template.created", "dashboard_template", template.ID.String(), tenantID, nil)

// Assignment created
emitEventWithActor(ctx, h.eventBus, "dashboard_template.assigned", "dashboard_template", templateID.String(), tenantID, map[string]string{
    "assignee_type": assigneeType,
    "assignee_id":   assigneeID.String(),
})
```

---

## Non-Breaking Change Verification

| Existing behaviour | Impact |
|-------------------|--------|
| `useDashboardLayout` localStorage logic | **Untouched** — new hook is separate |
| `DashboardPage` component | Gains one conditional branch; existing branch is the default |
| `DEFAULT_WIDGET_IDS` | Untouched — still used for fallback + new template seeding |
| Dashboard API endpoints (`/summary`, `/activity`, etc.) | **Not modified** |
| `AppSidebar` navigation | Not modified |
| RBAC role/permission tables | New permission rows only — no schema changes to existing tables |
| Existing migrations 001–053 | No modifications |
| Existing sqlc generated code | New file only — `dashboard_templates.sql` generates new functions, no changes to existing |
| `routes.tsx` | One new child route appended |
| `SettingsSidebar.tsx` | One new item in `adminItems` array |

---

## Implementation Phases

### Phase 1 — Backend foundation (no UI yet)
- Migration 054 (`dashboard_templates`, `dashboard_template_assignments`, `dashboard_user_overrides`)
- sqlc queries + `make sqlc`
- `DashboardTemplateHandler` with all endpoints
- System template seeding on first `GET /my-templates`
- Router registration
- Tests: handler unit tests, system template seeding

### Phase 2 — Frontend template loader
- `useMyTemplates` hook
- `TemplateDashboardContent` component (server-sourced layout, same widget rendering)
- Conditional branch in `DashboardPage`
- `useSaveLayoutOverride` hook (replaces localStorage save when template active)
- Tests: hook tests, component renders

### Phase 3 — Admin settings UI
- `DashboardsSettingsPage` at `/settings/dashboards`
- Template list + create/edit/delete
- Role/user assignment panel
- "Save as Template" button in Customize mode (admin-only)
- `SaveTemplateDialog`
- Settings sidebar entry
- Tests: page renders, CRUD flows

### Phase 4 — Template switcher
- `TemplateSwitcher` tab strip (only renders when `myTemplates.length > 1`)
- Active template state (stored in `sessionStorage` so refresh stays on same tab)
- Tests: switcher hidden with 1 template, visible with 2+

---

## Open Questions (resolve before implementation starts)

1. **Role ID resolution**: `GET /my-templates` needs the user's role IDs. The current `AuthUser` only has role names (`roles: string[]`), not UUIDs. Options: (a) add role IDs to `GET /api/v1/auth/me` response, or (b) resolve role UUIDs server-side from the user ID. **Recommendation: (b) — server resolves from `user_roles` table using the `X-User-ID` header, no frontend change needed.**

2. **Template default for new roles**: When an admin creates a new custom role, should it get a template auto-assigned? **Recommendation: No auto-assign. Admin explicitly assigns. Unassigned role = localStorage fallback.**

3. **OpenAPI spec**: New endpoints need to be added to the OpenAPI spec so `api/types.ts` can be regenerated (`make api-client`). The spec file format needs to be confirmed (auto-generated from Go handlers or hand-maintained YAML?).
