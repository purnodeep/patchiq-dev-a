# RBAC Error Handling Audit - Critical Issues

**Date**: 2026-04-09
**Scope**: RBAC implementation across `/internal/server/auth/sso.go`, `/web/src/api/client.ts`, `/web/src/app/auth/AuthContext.tsx`, and 35+ page files
**Severity**: CRITICAL + HIGH issues found

---

## Executive Summary

This RBAC implementation contains **5 CRITICAL error handling defects** and **3 HIGH severity issues** that will silently fail in production, leaving users confused and creating hard-to-debug permission issues. The code follows a dangerous pattern: catch errors, log warnings, and continue as if nothing happened.

**Core Problem**: Permission load failures silently degrade to implicit "deny all" without user awareness, contradicting the CLAUDE.md requirement: "**No silent failures in production code**."

---

## CRITICAL Issues

### 1. CRITICAL: Me() Handler Silent Permission Load Failure

**Location**: `/Users/shandesh/src/VS-code/patchiq-final/internal/server/auth/sso.go:318-329`

**Severity**: CRITICAL - Silent failure, deceptive UX, difficult debugging

**Issue**: When the permission store query fails (DB connection lost, query timeout, RLS error), the handler:
- Logs a **WARNING** (insufficient level for security failures)
- **Silently returns 200 OK with empty permissions array**
- User sees themselves as having NO permissions despite being authenticated
- No indication to user that permission loading failed vs. user actually having no permissions
- Frontend has no way to distinguish "permissions unknown" from "permissions loaded but user has none"

**Code**:
```go
if h.PermStore != nil && resp.TenantID != "" {
    perms, err := h.PermStore.GetUserPermissions(r.Context(), resp.TenantID, userID)
    if err != nil {
        slog.WarnContext(r.Context(), "sso me: failed to load user permissions", "error", err)
        // ERROR: NO INDICATION THIS IS A FAILURE. Returns 200 with empty perms.
    } else {
        entries := make([]permissionEntry, len(perms))
        for i, p := range perms {
            entries[i] = permissionEntry{Resource: p.Resource, Action: p.Action, Scope: p.Scope}
        }
        resp.Permissions = entries
    }
}
```

**Hidden Errors This Catch Block Could Be Hiding**:
- Database connection timeouts (connection pool exhausted)
- RLS policy evaluation errors (invalid tenant_id context)
- Row decoding errors (corrupt permission data)
- Transaction rollback failures
- Network errors (if DB is remote)
- Auth policy violations

**User Impact**:
- User appears to have no permissions in UI, all action buttons disabled
- User cannot perform any actions despite being authenticated
- User sees "Access denied" toasts on every action attempt
- Support gets confused: "User is authenticated but everything is denied"
- Security incident: User unknowingly working without permissions for 6+ hours

**Why This Violates CLAUDE.md**:
- Direct violation: "No silent failures in production code"
- Direct violation: "Catch blocks must be specific" - catches all `GetUserPermissions` errors without distinguishing recoverable vs. permanent failures
- Direct violation: "Fallback behavior must be explicit and justified" - silently falling back to "no permissions" is not justified

**Recommendation**:

Return **500 Internal Server Error** on permission load failure, not 200 OK. Users and operators must know this is a system problem, not a permission issue.

```go
if h.PermStore != nil && resp.TenantID != "" {
    perms, err := h.PermStore.GetUserPermissions(r.Context(), resp.TenantID, userID)
    if err != nil {
        // CRITICAL: Permission load failure is a security/availability issue, not a degradation
        slog.ErrorContext(r.Context(),
            "sso me: CRITICAL - failed to load user permissions (user will be locked out)",
            "error", err,
            "user_id", userID,
            "tenant_id", resp.TenantID,
        )
        // Return 500, not 200. User and operator must know this is a system failure.
        writeAuthError(r.Context(), w, http.StatusInternalServerError,
            "unable to load user permissions; please try again")
        return
    }
    entries := make([]permissionEntry, len(perms))
    for i, p := range perms {
        entries[i] = permissionEntry{Resource: p.Resource, Action: p.Action, Scope: p.Scope}
    }
    resp.Permissions = entries
}
```

---

### 2. CRITICAL: useCurrentUser Hook Silent Failure with No User Feedback

**Location**: `/Users/shandesh/src/VS-code/patchiq-final/web/src/api/hooks/useAuth.ts:13-26`

**Severity**: CRITICAL - Silent failure, no retry logic, no error logging

**Issue**: The `useCurrentUser()` hook:
- Throws generic error `auth/me failed: {status}` with no details
- **No logging of error status code** - user/developer cannot see if it's 401 (auth failure) vs. 500 (server error) vs. 503 (temp outage)
- `retry: false` means transient failures (network blip, temp server hiccup) are **permanent failures**
- Empty response on error - `useQuery` will have `data: undefined` and `isError: true`
- No indication of what the error is (timeout? server error? network failure?)
- Frontend has **no way to distinguish temporary from permanent failures**

**Code**:
```typescript
export function useCurrentUser() {
  return useQuery({
    queryKey: ['auth', 'me'],
    queryFn: async (): Promise<AuthUser> => {
      const res = await fetch('/api/v1/auth/me', {
        credentials: 'include',
      });
      if (!res.ok) throw new Error(`auth/me failed: ${res.status}`); // CRITICAL: No details
      return res.json() as Promise<AuthUser>;
    },
    retry: false, // CRITICAL: Transient failures become permanent
    staleTime: 5 * 60 * 1000,
  });
}
```

**Hidden Errors This Could Be Hiding**:
- Network timeout (fetch.abort)
- DNS resolution failure
- TLS certificate error
- Network unreachable
- Server 500 error (bug in /api/v1/auth/me)
- Server 503 error (temporary outage)
- 401 Unauthorized (token expired, browser cookies cleared)
- 403 Forbidden (user revoked)
- Body parse error (invalid JSON response)
- Content-Type mismatch

**User Impact**:
- Network hiccup causes permanent "Loading..." screen
- User refreshes page → same error (caught by hooks)
- User doesn't know if it's their network, server, or browser issue
- No actionable feedback: "Try again", "Check your internet", "Server is down", etc.

**Recommendation**:

Add retry logic, detailed error logging, and error status code capture:

```typescript
export function useCurrentUser() {
  return useQuery({
    queryKey: ['auth', 'me'],
    queryFn: async (): Promise<AuthUser> => {
      const res = await fetch('/api/v1/auth/me', {
        credentials: 'include',
      });
      if (!res.ok) {
        // Log full context for debugging
        const body = await res.text().catch(() => '(could not read body)');
        const error = new Error(
          `auth/me failed: HTTP ${res.status} ${res.statusText}`
        );
        console.error('[useCurrentUser] Auth fetch failed', {
          status: res.status,
          statusText: res.statusText,
          body: body.slice(0, 200),
        });
        throw error;
      }
      return res.json() as Promise<AuthUser>;
    },
    retry: (failureCount, error) => {
      // Retry on network/transient errors, not on 401/403
      const message = error instanceof Error ? error.message : String(error);
      const isPermanent = message.includes('401') || message.includes('403');
      return !isPermanent && failureCount < 3; // Retry up to 3 times for transient errors
    },
    staleTime: 5 * 60 * 1000,
  });
}
```

---

### 3. CRITICAL: Frontend Fallback to Dev User Masks Permission Load Failures

**Location**: `/Users/shandesh/src/VS-code/patchiq-final/web/src/app/auth/AuthContext.tsx:54-76`

**Severity**: CRITICAL - Production code using mock/stub on error

**Issue**: AuthProvider silently falls back to **devUser** when `useCurrentUser()` fails:
```typescript
const effectiveUser = user ?? (isError ? devUser : null);
```

This violates CLAUDE.md: **"Mock/fake implementations belong only in tests"**

**Why This Is Critical**:
- **ANY authentication failure** (network, server error, permission DB down) → user gets **admin devUser with wildcard permissions**
- User sees full admin UI, thinks they have admin access
- User attempts sensitive operations → all succeed in UI, but fail silently on backend (403 responses swallowed by middleware)
- User doesn't know their actual permission level
- **Security incident**: Unprivileged user thinks they have admin access
- **Debugging nightmare**: "It worked in my browser but failed on the server" (because backend rejected 403, frontend didn't see it)

**Code**:
```typescript
const devUser: AuthUser = {
  user_id: 'dev-user',
  tenant_id: '00000000-0000-0000-0000-000000000001',
  email: 'dev@patchiq.local',
  name: 'Dev User',
  preferred_username: 'dev-user',
  role: 'admin',
  roles: ['admin'],
  permissions: [{ resource: '*', action: '*', scope: '*' }], // WILDCARD ADMIN PERMS
};

// ...
const effectiveUser = user ?? (isError ? devUser : null); // CRITICAL: Fallback to admin on error
```

**Example Attack Scenario**:
1. Permission DB goes down (network partition)
2. useCurrentUser() fails
3. User gets devUser (full admin)
4. User clicks "Delete all endpoints" button
5. UI says "Success" (no 403 from middleware)
6. But actual endpoint remains (backend rejected it)
7. User is confused, support is confused

**Recommendation**:

Remove devUser fallback entirely from production code. Show error UI, not admin mock:

```typescript
export const AuthProvider = ({ children }: AuthProviderProps) => {
  const { data: user, isLoading, isError, error } = useCurrentUser();

  if (isLoading) {
    return <div className="flex items-center justify-center h-screen">Loading...</div>;
  }

  if (isError) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    return (
      <div className="flex items-center justify-center h-screen flex-col gap-4">
        <div className="text-lg font-semibold">Unable to load user profile</div>
        <div className="text-sm text-gray-600">{errorMessage}</div>
        <button
          onClick={() => window.location.reload()}
          className="px-4 py-2 bg-blue-600 text-white rounded"
        >
          Retry
        </button>
      </div>
    );
  }

  if (!user) {
    // Should not happen with proper error handling above
    return <div className="flex items-center justify-center h-screen">Invalid user state</div>;
  }

  return (
    <AuthContext.Provider value={{ user, can }}>{children}</AuthContext.Provider>
  );
};
```

---

### 4. CRITICAL: Forbid Middleware Swallows 403 Errors Without Logging Context

**Location**: `/Users/shandesh/src/VS-code/patchiq-final/web/src/api/client.ts:14-23`

**Severity**: CRITICAL - Error suppression, no context, no retry guidance

**Issue**: The `forbiddenInterceptor` middleware:
- Catches all 403 responses (permission denied)
- Shows **generic toast message**: "You don't have permission to perform this action"
- **No context about what action failed** - which API endpoint? What resource?
- **No error ID or trace ID** for support debugging
- **Toast disappears after 5 seconds** - user doesn't see it, doesn't know action failed
- **No indication if it's a permission issue vs. temporary policy evaluation failure**
- **No logging** - backend has the error, frontend has no record

**Code**:
```typescript
const forbiddenInterceptor: Middleware = {
  async onResponse({ response }) {
    if (response.status === 403) {
      toast.error('Access denied', {
        description: 'You don\'t have permission to perform this action.',
      });
    }
    return response; // Return 403 response, letting caller handle it too
  },
};
```

**Problems**:
1. **No context**: User sees "access denied" but doesn't know which operation
   - User was creating a deployment? Editing settings? Viewing dashboard?
   - Toast message alone is useless
2. **Double handling**: Middleware shows toast, then caller might handle it again
3. **No API context**: Which endpoint? `PUT /endpoints/123/patches`? `POST /deployments`?
4. **Silent failure on chained operations**: If operation A fails with 403, subsequent code might run anyway
5. **No operator debugging**: Support can't correlate frontend toast with backend error
6. **Transient failures not distinguished**: Is it permanent ("you never have permission") or temporary ("role assignment is being updated")?

**Hidden Errors This Could Be Hiding**:
- Backend permission evaluator crash (returns 403 by default)
- RLS policy evaluation error (mishandled, returns 403)
- Permission DB inconsistency (user thought they had permission but didn't)
- Race condition in permission cache invalidation (user just got permission, but cache still says no)
- Tenant isolation violation (permission for wrong tenant returned as 403)

**Recommendation**:

Log operation details and provide actionable feedback:

```typescript
const forbiddenInterceptor: Middleware = {
  async onResponse({ response }) {
    if (response.status === 403) {
      // Extract context from request
      const method = response.request?.method || 'UNKNOWN';
      const url = response.url || 'unknown endpoint';
      const operation = `${method} ${url}`;

      // Log with full context for operator debugging
      console.error('[API 403] Permission denied', {
        operation,
        status: response.status,
        timestamp: new Date().toISOString(),
      });

      // Show actionable toast with operation context
      toast.error('Action not allowed', {
        description: `You don't have permission to perform this action. Contact your administrator if you believe this is incorrect.`,
        duration: 10000, // Keep visible longer for user to read
      });
    }
    return response;
  },
};
```

Better yet: **don't use middleware for this**. Let callers handle 403 with full context:

```typescript
// Remove forbiddenInterceptor entirely. Each API call hook should handle 403:
export function useCreateDeployment() {
  return useMutation({
    mutationFn: async (deployment: DeploymentInput) => {
      const res = await api.POST('/deployments', { body: deployment });
      if (res.error && res.response.status === 403) {
        // CALLER has full context about what failed
        throw new Error('Permission denied: You cannot create deployments. Contact your admin.');
      }
      return res.data;
    },
    onError: (err) => {
      // Caller decides how to show the error
      toast.error('Failed to create deployment', {
        description: err.message,
      });
    },
  });
}
```

---

### 5. CRITICAL: checkPermission Function Lacks Error Handling for Undefined

**Location**: `/Users/shandesh/src/VS-code/patchiq-final/web/src/app/auth/AuthContext.tsx:38-52`

**Severity**: CRITICAL - Implicit allow on permission mismatch, silently permits unprivileged actions

**Issue**: The `checkPermission()` function treats `undefined` permissions as "allow all":

```typescript
if (permissions === undefined) return true;  // CRITICAL: Implicit allow
```

**Why This Is Critical**:
- If permission array is malformed or absent, user gets full access
- If backend returns invalid response (missing `permissions` field), user assumed to be admin
- Different from what CLAUDE.md expects: "Fallback must be explicit and justified"
- Not justified: why should undefined mean "allow all"?

**Better approach**: Undefined should mean "permission unknown, deny all":

```typescript
export function checkPermission(
  permissions: AuthUser['permissions'],
  resource: string,
  action: string,
): boolean {
  // undefined = server doesn't support permissions yet, allow all (backwards compat)
  if (permissions === undefined) return true;  // ← This is the problem

  // Should be:
  // if (permissions === undefined) {
  //   console.warn('[checkPermission] permissions is undefined - denying by default');
  //   return false;
  // }
```

---

## HIGH Severity Issues

### 1. HIGH: JWT Parsing Failure in Me() Handler Not Propagated

**Location**: `/Users/shandesh/src/VS-code/patchiq-final/internal/server/auth/sso.go:298-316`

**Severity**: HIGH - Silently incomplete user profile, confusing UX

**Issue**: When parsing the session JWT for profile claims fails:
```go
if cookie, err := r.Cookie(cookieName); err == nil {
    tok, err := jwtParseInsecure(cookie.Value)
    if err != nil {
        slog.WarnContext(r.Context(), "sso me: failed to parse session JWT for profile claims",
            "error", err,
        )
        // Continues, returns user without Name/Email/PreferredUsername
    } else {
        var profile oidcProfileClaims
        if err := tok.UnsafeClaimsWithoutVerification(&profile); err != nil {
            slog.WarnContext(r.Context(), "sso me: failed to extract profile claims from JWT",
                "error", err,
            )
            // Continues, returns user without profile
        }
    }
}
```

- Logs WARNING but continues
- User sees profile with no name/email (just user_id)
- Frontend shows "null" or blank fields
- User is confused: "Why is my profile incomplete?"
- Support can't tell if it's a profile load failure or user really has no name

**Recommendation**:

Don't swallow profile load failures. Return them as warnings but let the 200 response include a signal:

```go
// Option 1: Include success/failure flags in response
type meResponse struct {
    UserID            string            `json:"user_id"`
    TenantID          string            `json:"tenant_id,omitempty"`
    Name              string            `json:"name,omitempty"`
    Email             string            `json:"email,omitempty"`
    PreferredUsername string            `json:"preferred_username,omitempty"`
    Permissions       []permissionEntry `json:"permissions,omitempty"`
    // NEW: Indicate what failed to load
    ProfileFailure    *string           `json:"profile_failure,omitempty"` // "failed to parse JWT" if error
}

// In handler:
if cookie, err := r.Cookie(cookieName); err == nil {
    tok, err := jwtParseInsecure(cookie.Value)
    if err != nil {
        slog.WarnContext(r.Context(), "sso me: failed to parse session JWT", "error", err)
        resp.ProfileFailure = to.Ptr("failed to parse JWT: " + err.Error())
    } else {
        var profile oidcProfileClaims
        if err := tok.UnsafeClaimsWithoutVerification(&profile); err != nil {
            slog.WarnContext(r.Context(), "sso me: failed to extract profile claims", "error", err)
            resp.ProfileFailure = to.Ptr("failed to extract claims: " + err.Error())
        } else {
            resp.Name = profile.Name
            resp.Email = profile.Email
            resp.PreferredUsername = profile.PreferredUsername
        }
    }
}
```

---

### 2. HIGH: No Error ID / Trace ID in Sentry for RBAC Failures

**Location**: All three files - sso.go, client.ts, AuthContext.tsx

**Severity**: HIGH - Production debugging nightmare

**Issue**: All errors are logged but **no error ID from constants/errorIds.ts** for Sentry correlation:

```go
slog.WarnContext(r.Context(), "sso me: failed to load user permissions", "error", err)
// Missing: "error_id", errorIds.RBACPermissionLoadFailed,
```

- Support gets a user report: "I can't do anything"
- Operator looks at server logs, finds 100s of "failed to load user permissions" entries
- No way to correlate: which exact failures are from this user?
- Sentry shows error count but can't group by user/tenant for targeted fixes

**Recommendation**:

Use error IDs:

```go
slog.ErrorContext(r.Context(),
    "sso me: failed to load user permissions",
    "error", err,
    "error_id", errorIds.RBACPermissionLoadFailed, // ← Add this
    "user_id", userID,
    "tenant_id", tenantID,
)
```

---

### 3. HIGH: can() Hook in 35+ Pages Has No Null/Undefined Safety

**Location**: All page files using `useCan()` (SettingsSidebar, AgentFleetSettingsPage, IdentitySettingsPage, etc.)

**Severity**: HIGH - Potential runtime errors if AuthContext is not set up properly

**Issue**: Pages call `useCan()` without null checks:

```typescript
const can = useCan();
disabled={!can('endpoints', 'create')}
```

If `AuthContext` is not initialized (network error causes AuthProvider to unmount), `useAuth()` throws:
```typescript
export const useAuth = (): AuthContextValue => {
  const ctx = useContext(AuthContext);
  if (ctx === null) {
    throw new Error('useAuth must be used within an <AuthProvider>');
  }
  return ctx;
};
```

**Scenario**:
1. Auth provider encounters network error
2. Auth provider renders error screen
3. But user already navigated to a page that renders before error is shown
4. Page tries to call `useCan()` → throws uncaught error
5. White screen of death

**Recommendation**:

Use error boundary:

```typescript
export const AuthBoundary = ({ children }: { children: React.ReactNode }) => {
  const [error, setError] = useState<Error | null>(null);

  if (error) {
    return (
      <div className="flex items-center justify-center h-screen flex-col gap-4">
        <div className="text-lg font-semibold">Something went wrong</div>
        <div className="text-sm text-gray-600">{error.message}</div>
        <button onClick={() => window.location.reload()}>Reload</button>
      </div>
    );
  }

  return (
    <ErrorBoundary fallback={(err) => setError(err)}>
      {children}
    </ErrorBoundary>
  );
};
```

---

## Summary Table

| File | Line | Issue | Severity | Category |
|------|------|-------|----------|----------|
| `sso.go` | 318-329 | Silent permission load failure, returns 200 with empty perms | CRITICAL | Silent failure |
| `useAuth.ts` | 13-26 | No retry, no error details, retry: false on transient failures | CRITICAL | Error suppression |
| `AuthContext.tsx` | 54-76 | Fallback to admin devUser on auth error | CRITICAL | Mock in production |
| `client.ts` | 14-23 | 403 middleware swallows context, no logging | CRITICAL | Silent failure |
| `AuthContext.tsx` | 38-52 | checkPermission undefined → implicit allow | CRITICAL | Insecure default |
| `sso.go` | 298-316 | JWT parse failures logged but response incomplete | HIGH | Error handling gap |
| All files | Various | No error IDs for Sentry tracking | HIGH | Debugging difficulty |
| Pages (35+) | Various | useCan() called without null safety | HIGH | Runtime error risk |

---

## Actionable Summary for Fix

**Do NOT merge this PR until these are fixed**:

1. Make permission load failures return 500, not 200
2. Add retry logic and error details to useCurrentUser()
3. Remove devUser fallback from production AuthProvider
4. Move 403 handling from global middleware to individual API hooks with context
5. Add error IDs from constants/errorIds.ts to all RBAC logs
6. Add error boundaries around auth-dependent pages
7. Test all three platforms' auth flows with simulated permission DB failures

---

## Files to Review More Closely

- `/Users/shandesh/src/VS-code/patchiq-final/internal/server/auth/sso.go`
- `/Users/shandesh/src/VS-code/patchiq-final/web/src/api/hooks/useAuth.ts`
- `/Users/shandesh/src/VS-code/patchiq-final/web/src/app/auth/AuthContext.tsx`
- `/Users/shandesh/src/VS-code/patchiq-final/web/src/api/client.ts`
- `/Users/shandesh/src/VS-code/patchiq-final/internal/server/api/router.go` (lines 86-90, where PermStore is initialized)

---

**Generated by error handling auditor - 2026-04-09**
