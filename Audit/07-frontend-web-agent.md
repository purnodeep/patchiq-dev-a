# Web-Agent Frontend Audit

Audited: `web-agent/src/` (41 source files)
Date: 2026-04-09

---

## 1. Dead / Unused Files

### 1.1 `TabNav` component never imported — Important

- **File**: `src/app/layout/TabNav.tsx` (all 89 lines)
- **Details**: `TabNav` is exported but never imported anywhere. `AppLayout.tsx` uses `TopBar` + `IconRail` only. The component references a top offset of `64px` that does not match the current `48px` TopBar height, confirming it is a leftover from a previous layout iteration.
- **Severity**: Important

### 1.2 Shared style exports never imported — Minor

- **File**: `src/lib/styles.ts`, lines 20-49
- **Details**: `CARD_HEADER_STYLE`, `CARD_BODY_STYLE`, `CARD_TITLE_STYLE`, and `SELECT_STYLE` are exported but never imported by any consumer. Only `CARD_STYLE` and `CARD_PAD_STYLE` are used (by `StatusPage.tsx` and `HardwarePage.tsx`). Meanwhile, `HistoryPage.tsx`, `LogsPage.tsx`, `HardwarePage.tsx`, and `SoftwarePage.tsx` define their own local `CARD_TITLE_STYLE` and/or `SELECT_STYLE` constants that duplicate these shared exports.
- **Severity**: Minor

---

## 2. Incomplete Pages / Stub Functionality

### 2.1 "Install" and "Skip" buttons are non-functional — Critical

- **File**: `src/pages/pending/PendingPatchesPage.tsx`, lines 474-506
- **Details**: Both the "Install" and "Skip" action buttons rendered per-row have no `onClick` handler. They render as styled buttons but perform no action when clicked. No mutation hook exists for installing or skipping a patch. For a beta client deployment, users will see actionable-looking buttons that do nothing.
- **Severity**: Critical

### 2.2 Compliance widget is a static placeholder — Important

- **File**: `src/pages/status/StatusPage.tsx`, lines 552-578
- **Details**: The "Compliance" card in the status dashboard is a dashed-border placeholder that only shows a static text message: "Compliance scores are managed by your Patch Manager." No data is fetched or displayed. Given that this is a visible dashboard widget, it should either show real compliance data or be omitted entirely.
- **Severity**: Important

---

## 3. Raw `fetch()` Instead of Typed `openapi-fetch` Client

### 3.1 Six hooks bypass the typed API client — Important

The project has a properly configured `openapi-fetch` client (`src/api/client.ts`) that provides type-safe API calls. However, 6 out of 9 hooks use raw `fetch()` instead, losing type safety and consistent error handling:

| Hook | File | Line | Endpoint |
|------|------|------|----------|
| `useAgentHardware` | `src/api/hooks/useHardware.ts` | 8 | `/api/v1/hardware` |
| `useAgentSoftware` | `src/api/hooks/useSoftware.ts` | 8 | `/api/v1/software` |
| `useAgentServices` | `src/api/hooks/useServices.ts` | 8 | `/api/v1/services` |
| `useMetrics` | `src/api/hooks/useMetrics.ts` | 8 | `/api/v1/metrics` |
| `useUpdateSettings` | `src/api/hooks/useSettings.ts` | 32 | `PUT /api/v1/settings` |
| `useTriggerScan` | `src/api/hooks/useSettings.ts` | 52 | `POST /api/v1/scan` |

- **Root cause**: The OpenAPI spec (`src/api/types.ts`) only defines 6 endpoints (`/health`, `/api/v1/status`, `/api/v1/patches/pending`, `/api/v1/history`, `/api/v1/settings` GET, `/api/v1/logs`). The remaining endpoints (`/api/v1/hardware`, `/api/v1/software`, `/api/v1/services`, `/api/v1/metrics`, `PUT /api/v1/settings`, `POST /api/v1/scan`) are not in the spec, forcing these hooks to use raw `fetch`.
- **Severity**: Important — the OpenAPI spec is incomplete and these 6 hooks lack type safety.

---

## 4. TypeScript Issues

### 4.1 No `any` types, `@ts-ignore`, or `@ts-expect-error` found

The codebase is clean of TypeScript escape hatches. No `any` types, no suppression comments.

### 4.2 Hand-written types for endpoints not in OpenAPI spec — Important

- **Files**: `src/types/hardware.ts`, `src/types/software.ts`, `src/types/metrics.ts`
- **Details**: These three files contain hand-written TypeScript interfaces that mirror Go structs (`HardwareInfo`, `ExtendedPackageInfo`, `ServiceInfo`, `LiveMetrics`). They are used by hooks that call raw `fetch()` (see section 3). If the Go struct changes, these will silently drift out of sync. Should be generated from the OpenAPI spec like the other types.
- **Severity**: Important

### 4.3 `MutableRefObject` cast in StatusPage — Minor

- **File**: `src/pages/status/StatusPage.tsx`, line 75
- **Details**: `(ref as React.MutableRefObject<HTMLDivElement | null>).current = el;` — manual cast to assign to a ref callback. This works but is a workaround for combining callback refs with `useRef`. A cleaner pattern would be `useCallback` ref.
- **Severity**: Minor

---

## 5. Missing Error / Loading States

### 5.1 All pages have loading and error states

Every page properly handles `isLoading` (showing `<Skeleton>` components) and `isError` (showing `<ErrorState>` or inline error messages). This is well implemented.

### 5.2 HistoryPage error state lacks retry — Minor

- **File**: `src/pages/history/HistoryPage.tsx`, line 527
- **Details**: `<ErrorState message="Failed to load history." />` does not pass an `onRetry` callback, unlike all other pages which provide retry functionality.
- **Severity**: Minor

### 5.3 LogsPage error state is inline text, not ErrorState component — Minor

- **File**: `src/pages/logs/LogsPage.tsx`, lines 477-486
- **Details**: The error state renders a plain `<span>` with "Failed to load logs." instead of using the shared `<ErrorState>` component (which is imported but only used in other pages). No retry button is provided.
- **Severity**: Minor

---

## 6. Routing Gaps

### 6.1 No 404 / catch-all route — Important

- **File**: `src/app/routes.tsx`
- **Details**: The router has no `*` wildcard route or `errorElement`. Navigating to any undefined path (e.g., `/foo`) renders a blank page inside the layout. Should show a "Page not found" message and link back to `/`.
- **Severity**: Important

### 6.2 No `errorElement` on the route tree — Important

- **File**: `src/app/routes.tsx`
- **Details**: No `errorElement` is defined on any route. If a page component throws during rendering, React Router will show its default error UI (or a blank screen). An error boundary with a retry/home link should be configured.
- **Severity**: Important

---

## 7. Form Validation Gaps

### 7.1 SettingsPage uses no schema validation — Important

- **File**: `src/pages/settings/SettingsPage.tsx`
- **Details**: The settings form uses manual state management with `useState` (11 separate state variables) instead of `react-hook-form` + Zod, which is the project-wide pattern per CLAUDE.md. Validation is limited to a single inline check on `bandwidthLimit` (line 382-389). Specifically missing:
  - No validation that `proxyUrl` is a valid URL format
  - No validation that `autoRebootWindow` matches the expected time-range format (e.g., `HH:MM-HH:MM`)
  - No validation on `scanInterval` or `heartbeatInterval` values
  - No Zod schema for the form
- **Severity**: Important

---

## 8. Style Duplication

### 8.1 `CARD_STYLE`, `CARD_TITLE_STYLE`, `SELECT_STYLE` redefined in multiple pages — Minor

- `CARD_TITLE_STYLE` is defined locally in: `HardwarePage.tsx` (line 22), `SoftwarePage.tsx` (line 13)
- `SELECT_STYLE` is defined locally in: `HistoryPage.tsx` (line 24), `LogsPage.tsx` (line 220)
- `CARD_STYLE` is defined locally in: `HistoryPage.tsx` (line 16)
- All are also exported from `src/lib/styles.ts` but the local versions differ slightly in some cases (e.g., HistoryPage's `CARD_STYLE` adds `flex: 1`).
- **Severity**: Minor

### 8.2 `StatFilterCard` component duplicated — Minor

- **Files**: `src/pages/pending/PendingPatchesPage.tsx` (lines 61-111), `src/pages/services/ServicesPage.tsx` (lines 36-86)
- **Details**: Identical `StatFilterCard` component is copy-pasted between two pages. Should be extracted to `src/components/`.
- **Severity**: Minor

---

## 9. Other Observations

### 9.1 `console.error` in QueryClient global handlers — Minor

- **File**: `src/App.tsx`, lines 9, 14
- **Details**: CLAUDE.md mandates `slog` for Go logging. On the frontend side, `console.error` in the global query/mutation error handlers is acceptable for development, but for a client-facing beta, consider integrating with the toast notification system or a structured error reporting mechanism.
- **Severity**: Minor

### 9.2 `hoverHandlers` uses imperative DOM event listeners — Minor

- **Files**: `src/pages/status/StatusPage.tsx` (line 54), `src/pages/hardware/HardwarePage.tsx` (line 29)
- **Details**: These functions directly attach `mouseenter`/`mouseleave` listeners via `addEventListener`, bypassing React's event system. This works but can leak listeners if the element unmounts. A React `onMouseEnter`/`onMouseLeave` approach or CSS `:hover` would be more idiomatic.
- **Severity**: Minor

### 9.3 No test coverage for HardwarePage, SoftwarePage, or ServicesPage — Important

- **Directory**: `src/__tests__/pages/`
- **Details**: Tests exist for StatusPage, PendingPatchesPage, HistoryPage, LogsPage, and SettingsPage. No tests exist for HardwarePage, SoftwarePage, or ServicesPage.
- **Severity**: Important

---

## Summary

| Severity | Count | Items |
|----------|-------|-------|
| Critical | 1 | Non-functional Install/Skip buttons |
| Important | 9 | Unused TabNav, raw fetch bypasses typed client, incomplete OpenAPI spec, hand-written types, no 404 route, no errorElement, no form validation schema, compliance placeholder, missing test coverage |
| Minor | 8 | Unused style exports, style duplication, StatFilterCard duplication, HistoryPage missing retry, LogsPage inline error, MutableRefObject cast, console.error in handlers, imperative DOM listeners |
