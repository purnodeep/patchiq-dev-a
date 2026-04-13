# Hub Authentication (PIQ-12) — Design Spec

> **Status**: Approved
> **Created**: 2026-03-31
> **Branch**: dev-b
> **Scope**: Replace mock auth in web-hub with real Zitadel OIDC authentication (slim variant)

---

## Overview

The Hub Manager (`web-hub/` + `internal/hub/`) currently uses a hardcoded mock user with no login, no session, and no auth middleware. This spec adds real authentication matching the Patch Manager's pattern but stripped to essentials: direct login + JWT middleware + session cookies. No SSO, no invite, no RBAC.

## Decision Record

- **Slim auth only** — Hub is an internal SaaS app for POC. All hub users are admins. SSO, invite, forgot-password, and RBAC are deferred.
- **Hub's own auth package** — `internal/hub/auth/` with its own code. No imports from `internal/server/` (violates import rules). No shared extraction (avoids server refactor risk during maturation).
- **Login page in React app** — `/login` route in web-hub SPA, matching PM's UI aesthetic.
- **Dev fallback** — If `/api/v1/auth/me` fails in dev mode, use a stub user so devs can work without Zitadel.

---

## Backend: `internal/hub/auth/`

### New Files

#### `zitadel.go` — Minimal Zitadel API Client

Subset of server's `auth.ZitadelClient`. Only two capabilities:

- `Authenticate(ctx, email, password) → (ZitadelUser, error)` — Calls Zitadel's session creation API to verify credentials. Returns user ID, email, display name.
- `GetUserInfo(ctx, userID) → (ZitadelUser, error)` — Fetches user details by ID (for `/auth/me` enrichment if needed).

Uses Zitadel v2 API (`/v2/sessions`, `/v2/users/{id}`). Configured with base URL + service account PAT from `configs/hub.yaml`.

#### `jwt.go` — JWT Middleware

chi middleware that:

1. Extracts token from `piq_hub_session` cookie (primary) or `Authorization: Bearer` header (fallback).
2. Validates HMAC-SHA256 signature using the local signing key.
3. Extracts claims: `sub` (user_id), `email`, `name`, `tenant_id`.
4. Injects user info into request context via `context.WithValue`.
5. Returns 401 if token is missing, expired, or invalid.

#### `login.go` — Auth Handlers

Three endpoints:

- **`POST /api/v1/auth/login`** — Accepts `{email, password, remember_me}`. Authenticates via Zitadel client. On success: mints HMAC-SHA256 JWT with user claims, sets `piq_hub_session` httpOnly cookie (24h TTL, or 168h if remember_me), returns `200` with user JSON.
- **`GET /api/v1/auth/me`** — Reads JWT from cookie/header (already validated by middleware), returns user claims as JSON: `{user_id, tenant_id, email, name, role: "admin"}`.
- **`POST /api/v1/auth/logout`** — Clears the `piq_hub_session` cookie, returns 200.

#### `session.go` — Session Configuration

```go
type SessionConfig struct {
    CookieName     string        // "piq_hub_session"
    CookieDomain   string        // from config
    CookieSecure   bool          // false for dev, true for prod
    AccessTokenTTL time.Duration // 24h default
    RememberMeTTL  time.Duration // 168h default
    SigningKey      []byte       // HMAC key, generated on startup
}
```

`InitSigningKey()` generates a random 32-byte HMAC key if not already set. Key lives in memory (not persisted — server restart = re-login, acceptable for POC).

### Router Changes (`internal/hub/api/router.go`)

1. **CORS**: Set `AllowCredentials: true`. Explicit origins only (already using `cfg.Hub.CORSOrigins`).
2. **Auth routes** (outside tenant middleware, outside JWT middleware):
   ```
   POST /api/v1/auth/login   → login handler
   ```
3. **Protected auth routes** (inside JWT middleware, outside tenant middleware):
   ```
   GET  /api/v1/auth/me      → me handler
   POST /api/v1/auth/logout   → logout handler
   ```
4. **JWT middleware** added to `/api/v1` route group, before tenant middleware. The middleware also sets `X-Tenant-ID` from JWT claims so tenant middleware passes.

### `cmd/hub/main.go` Changes

Add auth initialization between step 7 (MinIO) and step 8 (HTTP server):

1. Read `cfg.IAM.Zitadel` config (already defined in `configs/hub.yaml`).
2. If `ClientID != ""`: create Zitadel client, session config, init signing key, create login handler, create JWT middleware.
3. Else: log warning, no JWT middleware (header-based auth stubs for dev).
4. Pass `jwtMW` and `loginHandler` to `api.NewRouter()`.

### Config (`configs/hub.yaml`)

Already has the required fields:

```yaml
iam:
  zitadel:
    domain: "localhost:8085"
    secure: false
    client_id: "363618248001921030"
    client_secret: ""
    service_account_key: ""
    redirect_uri: "http://localhost:8082/api/v1/auth/callback"  # not used for slim auth
  session:
    cookie_name: "piq_hub_session"
    cookie_secure: false
    cookie_domain: "localhost"
    access_token_ttl: 24h       # used as default TTL (was 15m, changing to 24h)
    refresh_token_ttl: 168h     # used as remember_me TTL
    post_login_url: "http://localhost:3002/"
```

Only change: bump `access_token_ttl` from `15m` to `24h` for POC usability (no refresh token flow).

---

## Frontend: `web-hub/`

### New Files

#### `src/pages/login/LoginPage.tsx`

Login form matching PM's aesthetic:
- Card with "Welcome back" heading
- Email field + password field (with show/hide toggle)
- "Remember me" checkbox
- Sign in button with loading spinner
- Server error display
- Uses react-hook-form + Zod validation
- Styled with CSS variables from the design system (same as PM)

**Not included** (differs from PM):
- No SSO button
- No "Forgot password?" link
- No "Have an invite? Sign up" link

#### `src/pages/login/index.ts`

Barrel export for `LoginPage`.

#### `src/components/auth/AuthLayout.tsx`

Full-page centered layout for the login form. Dark background, centered card, PatchIQ logo + "Hub Manager" subtitle. Same pattern as PM's `AuthLayout`.

#### `src/api/hooks/useAuth.ts`

```typescript
useCurrentUser()  // GET /api/v1/auth/me, credentials: 'include'
useLogout()       // POST /api/v1/auth/logout, invalidates queries, redirects to /login
```

#### `src/api/hooks/useLogin.ts`

```typescript
useLogin()  // POST /api/v1/auth/login {email, password, remember_me}
```

### Modified Files

#### `src/app/auth/AuthContext.tsx`

Rewrite to match PM's pattern:

```typescript
export const AuthProvider = ({ children }) => {
  const { data: user, isLoading, isError } = useCurrentUser();

  if (isLoading) return <LoadingScreen />;

  // Dev fallback: if auth fails, use stub user
  const devUser = { user_id: 'dev-user', tenant_id: '...', email: 'dev@patchiq.local', name: 'Dev User', role: 'admin' };
  const effectiveUser = user ?? (isError ? devUser : null);

  if (!effectiveUser) return <LoadingScreen />;

  return <AuthContext.Provider value={{ user: effectiveUser }}>{children}</AuthContext.Provider>;
};
```

AuthUser interface updated to match PM's: `user_id`, `tenant_id`, `email`, `name`, `role`, `roles`.

#### `src/app/routes.tsx`

Add `/login` route pointing to `LoginPage`.

#### `src/app/layout/` (sidebar/topbar)

Wire logout action to `useLogout()` hook wherever user avatar/menu exists.

---

## Data Flow

```
Browser opens web-hub (any route)
  → App renders → AuthProvider mounts
  → useCurrentUser() fires GET /api/v1/auth/me (credentials: include)
  → Case 1: No cookie → 401 → isError → dev fallback OR redirect to /login
  → Case 2: Valid cookie → 200 {user_id, email, name, role} → app renders

Login flow:
  → User at /login → fills email + password → submits
  → useLogin() fires POST /api/v1/auth/login {email, password, remember_me}
  → Hub backend → Zitadel API: create session (verify credentials)
  → Success → mint HMAC JWT → Set-Cookie: piq_hub_session (httpOnly, 24h or 168h)
  → 200 {user_id, email, name} → navigate to /
  → AuthProvider re-fetches /auth/me → cookie valid → user loaded

Logout:
  → User clicks logout → useLogout() fires POST /api/v1/auth/logout
  → Backend clears cookie → 200
  → Frontend invalidates auth queries → redirect to /login
```

---

## What's Excluded

| Feature | Reason |
|---------|--------|
| SSO/PKCE flow | Hub is internal-only for POC |
| Invite/register | Hub users are provisioned in Zitadel directly |
| Forgot password | Use Zitadel console for password resets |
| RBAC/permissions | All hub users are admins for POC |
| Refresh tokens | Single access token with 24h TTL; re-login after expiry |
| Shared auth package | Avoids server refactor risk; consolidate post-POC |

---

## Testing Strategy

### Backend
- `internal/hub/auth/login_test.go` — Table-driven tests: successful login, invalid credentials, missing fields, cookie setting
- `internal/hub/auth/jwt_test.go` — Valid token, expired token, malformed token, missing cookie
- `internal/hub/auth/zitadel_test.go` — Mock Zitadel API responses: success, auth failure, server error

### Frontend
- `web-hub/src/pages/login/__tests__/LoginPage.test.tsx` — Form validation, submit flow, error display, loading state
- `web-hub/src/app/auth/__tests__/AuthContext.test.tsx` — Auth provider with mock API: authenticated user, dev fallback, loading state
