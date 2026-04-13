# Audit: web-hub Frontend

**Scope**: `/home/patchiq/patchiq-dev-a/web-hub/src/` (all files)
**Date**: 2026-04-09
**Auditor**: Claude Opus 4.6

---

## 1. Dead / Unused Files

### 1.1 PlaceholderPage never imported (Minor)
- **File**: `src/pages/PlaceholderPage.tsx`
- **Issue**: Exported but never imported by any file. The component was likely used during scaffolding and is now dead code.
- **Action**: Delete the file.

### 1.2 StatsStrip component never imported (Minor)
- **File**: `src/components/StatsStrip.tsx`
- **Issue**: Exported but never imported anywhere. Pages use their own inline StatCard patterns instead.
- **Action**: Delete the file.

### 1.3 SeverityPills component never imported (Minor)
- **File**: `src/components/SeverityPills.tsx`
- **Issue**: Exported but never imported. CatalogPage uses its own severity filter strip with Select dropdowns.
- **Action**: Delete the file.

### 1.4 `api/client.ts` (openapi-fetch) never imported (Important)
- **File**: `src/api/client.ts`
- **Issue**: Creates a typed `api` client using `openapi-fetch`, but no hook or page imports it. All API calls go through `apiFetch()` (raw `fetch` wrapper) or local `apiFetch` copies in `dashboard.ts`, `feeds.ts`, `settings.ts`. The generated `api/types.ts` (308 lines of OpenAPI types) is also unused -- no file imports from it.
- **Action**: Either migrate all API calls to use the typed `openapi-fetch` client (as `web/` does), or remove `client.ts` and `types.ts` to avoid confusion. The typed client is the correct pattern per CLAUDE.md conventions.

### 1.5 `types/settings.ts` never imported (Minor)
- **File**: `src/types/settings.ts`
- **Issue**: Defines `HubSettings`, `IAMSettings`, `RoleMapping`, `WebhookSettings` but no file imports any of these types. Settings pages use `Record<string, unknown>` instead.
- **Action**: Either use these types in the settings hooks/pages or delete.

### 1.6 `api/settings.ts` -- `getSetting()` exported but never called (Minor)
- **File**: `src/api/settings.ts`, line 37
- **Issue**: `getSetting(key)` function is exported but never used. Only `getSettings()` and `upsertSetting()` are consumed via hooks.
- **Action**: Remove the unused function.

---

## 2. Duplicate / Inconsistent API Layer

### 2.1 Three duplicate `apiFetch` implementations (Important)
- **Files**:
  - `src/api/fetch.ts` (canonical, used by `catalog.ts`, `clients.ts`, `licenses.ts`)
  - `src/api/dashboard.ts`, lines 9-26 (local copy)
  - `src/api/feeds.ts`, lines 4-20 (local copy)
  - `src/api/settings.ts`, lines 7-23 (local copy)
- **Issue**: Three API modules define their own private `apiFetch` function instead of importing from `./fetch`. All three are identical. This is a maintenance hazard -- changes to error handling or headers must be applied in four places.
- **Action**: Refactor `dashboard.ts`, `feeds.ts`, and `settings.ts` to import `apiFetch` from `./fetch`.

### 2.2 Auth hooks use raw `fetch` instead of `apiFetch` or typed client (Minor)
- **Files**:
  - `src/api/hooks/useAuth.ts`, lines 14-18 (raw `fetch('/api/v1/auth/me')`)
  - `src/api/hooks/useAuth.ts`, lines 30-33 (raw `fetch('/api/v1/auth/logout')`)
  - `src/api/hooks/useLogin.ts`, lines 20-26 (raw `fetch('/api/v1/auth/login')`)
- **Issue**: These hooks call `fetch()` directly with manual header/error handling, bypassing both the `apiFetch` wrapper and the openapi-fetch typed client. Missing `X-Tenant-ID` header on auth requests.
- **Action**: Either create proper auth API functions in an `api/auth.ts` file using `apiFetch`, or route through the openapi-fetch client.

### 2.3 `CatalogDetailPage` uses raw `fetch` for download (Minor)
- **File**: `src/pages/catalog/CatalogDetailPage.tsx`, line 111
- **Issue**: `downloadBinary()` calls `fetch()` directly without `X-Tenant-ID` header.
- **Action**: Route through `apiFetch` or the typed client.

---

## 3. TypeScript Issues

### 3.1 `as any` cast to bypass OpenAPI types (Important)
- **File**: `src/pages/catalog/CatalogPage.tsx`, lines 170-171
- **Code**: `} as any);` with comment `// eslint-disable-next-line @typescript-eslint/no-explicit-any -- entry_type not yet in generated OpenAPI types`
- **Issue**: The `entry_type` parameter is not in the generated types, so the entire params object is cast to `any`. This silences all type checking for the hook call.
- **Action**: Update the OpenAPI spec to include `entry_type`, regenerate types, or add it to the local `CatalogEntry` type definition. If the backend supports the parameter, the spec should reflect it.

### 3.2 Type assertion in CatalogForm (Minor)
- **File**: `src/pages/catalog/CatalogForm.tsx`, line 99
- **Code**: `createMutation.mutate(payload as Parameters<typeof createMutation.mutate>[0], {`
- **Issue**: The local `CreateCatalogEntryRequest` interface and the mutation's expected type diverge, requiring a type assertion. This happens because `CatalogForm` redefines its own interface (line 23) instead of importing from `types/catalog.ts`.
- **Action**: Import `CreateCatalogRequest` from `types/catalog.ts` instead of redefining it locally.

### 3.3 Hardcoded tenant ID in four places (Important)
- **Files**:
  - `src/api/client.ts`, line 4: `const TENANT_ID = '00000000-0000-0000-0000-000000000001'`
  - `src/api/fetch.ts`, line 1: same
  - `src/api/dashboard.ts`, line 9: same
  - `src/api/feeds.ts`, line 3: same
  - `src/api/settings.ts`, line 6: same
- **Issue**: Tenant ID is duplicated in 5 files. Should be a single constant.
- **Action**: Define once in a shared location (e.g., `lib/constants.ts`) and import.

---

## 4. Incomplete Pages / Stubs

### 4.1 DeploymentsPage is a stub (Important)
- **File**: `src/pages/deployments/DeploymentsPage.tsx`
- **Issue**: Entire page is a "Coming Soon" placeholder with no data, no hooks, and no real content. Shows: "Cross-client deployment management is coming in a future release." This is a routed page in the sidebar nav that does nothing.
- **Action**: Either implement the page or remove the route and sidebar link until ready. A deployed client POC will see a dead page.

### 4.2 AddFeedForm does not actually create feeds (Important)
- **File**: `src/pages/feeds/AddFeedForm.tsx`, lines 43-47
- **Code**: `toast('Feed creation is not yet supported.', { description: 'Backend endpoint pending.' });`
- **Issue**: The "Add Feed" button on FeedsPage opens a form that collects data but does not submit it. It just shows a toast saying "not yet supported." No API call is made.
- **Action**: Either wire up to a backend endpoint or hide the "Add Feed" button until the feature is ready.

### 4.3 IAMSettings "Test Connection" is fake (Important)
- **File**: `src/pages/settings/IAMSettings.tsx`, lines 65-72
- **Issue**: `handleTestConnection()` uses `setTimeout` to simulate a success response. It does not make any API call. The UI shows "Connection OK" after 1.5 seconds regardless of actual connectivity.
- **Action**: Wire to a real backend health-check endpoint or remove the button.

### 4.4 IAMSettings "Add Role Mapping" button is non-functional (Minor)
- **File**: `src/pages/settings/IAMSettings.tsx`, line 318
- **Code**: `<button className="mt-3 text-sm ..."><Plus .../> Add Role Mapping</button>`
- **Issue**: Button has no `onClick` handler. Clicking does nothing.
- **Action**: Implement role mapping creation or hide the button.

### 4.5 FeedConfigSettings "Save All Feeds" button is non-functional (Minor)
- **File**: `src/pages/settings/FeedConfigSettings.tsx`, line 337
- **Code**: `<Button>Save All Feeds</Button>`
- **Issue**: No `onClick` handler. Individual feed toggles work via `handleToggleFeed`, but the global save button does nothing.
- **Action**: Wire up or remove.

### 4.6 FeedConfigSettings "Custom Feed / Configure" button is non-functional (Minor)
- **File**: `src/pages/settings/FeedConfigSettings.tsx`, lines 298-332
- **Issue**: The "Custom Feed" card with "Configure" button has no `onClick` handler.
- **Action**: Wire up or remove.

### 4.7 API Webhooks "Rotate" API key button is non-functional (Minor)
- **File**: `src/pages/settings/APIWebhookSettings.tsx`, line 228
- **Issue**: "Rotate" button for API key has no `onClick` handler.
- **Action**: Wire up or remove.

### 4.8 TopBar Search button is non-functional (Minor)
- **File**: `src/app/layout/TopBar.tsx`, lines 117-154
- **Issue**: Search button shows "Cmd+K" shortcut hint but has no click handler and no keyboard shortcut binding. Clicking does nothing.
- **Action**: Implement search or remove the button.

### 4.9 TopBar Notification bell is non-functional (Minor)
- **File**: `src/app/layout/TopBar.tsx`, lines 189-213
- **Issue**: Bell icon button has no click handler or notification system wired up.
- **Action**: Implement or remove.

---

## 5. API Hook Coverage

### 5.1 Overall assessment: Good
All pages use hooks from `api/hooks/` for data fetching. No page calls `apiFetch` directly. The hooks layer is complete for catalog, clients, dashboard, feeds, licenses, and settings.

### 5.2 No auth API module (Minor)
- **Issue**: Auth calls (`/auth/me`, `/auth/login`, `/auth/logout`) use raw `fetch` directly inside hooks rather than having an `api/auth.ts` module. This is inconsistent with all other resources which have `api/<resource>.ts` + `api/hooks/use<Resource>.ts`.
- **Action**: Create `api/auth.ts` and route through `apiFetch`.

---

## 6. Missing Error / Loading States

### 6.1 FeedDetailPage config form lacks error handling (Minor)
- **File**: `src/pages/feeds/FeedDetailPage.tsx`, line 220
- **Issue**: `onSaveConfig` calls `updateFeed.mutate()` but the form doesn't display errors if the mutation fails. No toast or inline error for config save failures.
- **Action**: Add error handling to the config save flow.

### 6.2 FeedDetailPage config form uses `useForm` without `zodResolver` (Minor)
- **File**: `src/pages/feeds/FeedDetailPage.tsx`, lines 201-217
- **Issue**: The config form uses `useForm<ConfigFormData>` but has no Zod schema or validation resolver. Fields are not validated before submission. This is inconsistent with CatalogForm and LicenseForm which both use `zodResolver`.
- **Action**: Add Zod validation schema.

### 6.3 LicenseDetailPage renew dialog lacks form validation (Minor)
- **File**: `src/pages/licenses/LicenseDetailPage.tsx`
- **Issue**: The renew dialog uses raw state (`renewForm`) with no validation. If the user submits with empty fields, it will send invalid data to the backend.
- **Action**: Add Zod/react-hook-form validation.

---

## 7. Routing Gaps

### 7.1 No 404 / catch-all route (Minor)
- **File**: `src/app/routes.tsx`
- **Issue**: No wildcard `*` route for unmatched paths. Navigating to a non-existent route shows a blank page.
- **Action**: Add `{ path: '*', element: <NotFoundPage /> }`.

### 7.2 No auth guard on protected routes (Important)
- **File**: `src/app/routes.tsx`
- **Issue**: All routes (dashboard, catalog, feeds, etc.) are accessible without authentication. The `AuthProvider` in `AppLayout` does check for the current user, but if the auth API fails, it silently falls back to a hardcoded `devUser` (line 29-35 in `AuthContext.tsx`). In production, this means:
  - Auth failure = auto-login as "Dev User" with admin role
  - No redirect to `/login` on 401
- **Contrast with web/**: The `web/` app client has a `401 -> redirect to /login` middleware on the openapi-fetch client.
- **Action**: Remove the dev fallback user in production builds, and add 401 handling middleware.

---

## 8. Form Validation Gaps

### 8.1 AddFeedForm has no validation (Minor)
- **File**: `src/pages/feeds/AddFeedForm.tsx`
- **Issue**: Uses raw `useState` instead of `react-hook-form` + `zod`. No field validation. Submitting with empty fields is possible (though the form doesn't actually submit to API anyway -- see 4.2).
- **Action**: When implementing the real form, use `react-hook-form` + `zodResolver` per project conventions.

### 8.2 IAMSettings form has no validation (Minor)
- **File**: `src/pages/settings/IAMSettings.tsx`
- **Issue**: SSO URL, Client ID, Client Secret, Redirect URI fields use raw `useState` with no validation.
- **Action**: Add Zod schema validation, especially for URL fields.

### 8.3 GeneralSettings form has no validation (Minor)
- **File**: `src/pages/settings/GeneralSettings.tsx`
- **Issue**: Uses raw `useState` for all fields. No react-hook-form, no Zod. Hub name could be empty string.
- **Action**: Add validation.

---

## 9. Consistency with web/

### 9.1 API client pattern divergence (Critical)
- **web/** uses `openapi-fetch` typed client (`api.GET`, `api.POST`) with generated types from OpenAPI spec. Every API call is type-safe.
- **web-hub/** has the typed client in `api/client.ts` but never uses it. All API calls use a hand-written `apiFetch` wrapper that returns untyped `res.json() as Promise<T>`. The generated `api/types.ts` is dead code.
- **Impact**: No compile-time validation of API request/response shapes. Type mismatches between frontend types and actual API responses will only be caught at runtime.
- **Action**: Migrate all API modules to use the openapi-fetch typed client.

### 9.2 No auth redirect middleware (Important)
- **web/** has middleware on the openapi-fetch client that redirects to `/login` on 401 responses.
- **web-hub/** has no equivalent. 401 responses are either silently swallowed (in `AuthProvider` fallback) or thrown as generic errors.
- **Action**: Add auth redirect middleware.

### 9.3 Duplicate helper functions across pages (Minor)
- `formatRelativeTime()` is defined in 3 places:
  - `src/lib/format.ts` (canonical)
  - `src/pages/clients/ClientsPage.tsx`, lines 64-74
  - `src/pages/clients/ClientDetailPage.tsx`, lines 55-63
- `formatSyncInterval()` is defined in 3 places:
  - `src/pages/clients/ClientsPage.tsx`, lines 85-89
  - `src/pages/clients/ClientDetailPage.tsx`, lines 66-70
  - `src/pages/settings/FeedConfigSettings.tsx`, lines 7-11
- `computeHealthScore()` is defined in 2 places:
  - `src/pages/clients/ClientsPage.tsx`, lines 51-61
  - `src/pages/clients/ClientDetailPage.tsx`, lines 38-47
- `getAvatar()` is defined in 2 places:
  - `src/pages/clients/ClientsPage.tsx`, line 95
  - `src/pages/clients/ClientDetailPage.tsx`, lines 33-36
- `StatCard` component is defined independently in CatalogPage, FeedsPage, LicensesPage, ClientsPage, and ClientDetailPage -- 5 separate implementations with slight variations.
- **Action**: Extract shared helpers to `lib/` and create a single `StatCard` component in `components/`.

### 9.4 `web/` uses `@patchiq/ui` DataTable; `web-hub/` has its own (Minor)
- **web-hub/** has a custom `components/data-table/DataTable.tsx` and `DataTablePagination.tsx`. These partially overlap with `@patchiq/ui`'s `DataTable` component.
- **Action**: Evaluate if `@patchiq/ui` DataTable is sufficient, or if the custom one is needed.

---

## Summary by Severity

| Severity | Count | Description |
|----------|-------|-------------|
| Critical | 1 | API client pattern divergence (openapi-fetch unused, all API calls untyped) |
| Important | 8 | Dead openapi-fetch client; duplicate apiFetch; hardcoded tenant IDs; DeploymentsPage stub; AddFeedForm stub; fake IAMSettings test; no auth guard; no auth redirect middleware |
| Minor | 22 | Dead files (3); unused exports (2); raw fetch in auth hooks; duplicate helpers; missing validation (4); non-functional buttons (5); no 404 route; no Zod on config forms; etc. |

**Total findings: 31**
