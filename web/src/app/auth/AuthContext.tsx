import { createContext, useCallback, useContext, useEffect } from 'react';
import { useCurrentUser } from '../../api/hooks/useAuth';
import { getActiveTenantId, setActiveTenantId } from '../../api/activeTenantStore';

export interface AccessibleTenant {
  id: string;
  name: string;
  slug: string;
}

export interface OrganizationInfo {
  id: string;
  name: string;
  slug: string;
  type: 'direct' | 'msp' | 'reseller';
}

interface AuthUser {
  user_id: string;
  tenant_id?: string;
  email?: string;
  name?: string;
  preferred_username?: string;
  role?: string;
  roles?: string[];
  permissions?: Array<{ resource: string; action: string; scope: string }>;
  // Organization / MSP fields. Optional because older backend versions and
  // single-tenant deployments do not populate them.
  organization?: OrganizationInfo;
  active_tenant_id?: string;
  accessible_tenants?: AccessibleTenant[];
  org_permissions?: Array<{ resource: string; action: string; scope: string }>;
}

interface AuthContextValue {
  user: AuthUser;
  can: (resource: string, action: string) => boolean;
  isMspOperator: boolean;
}

const AuthContext = createContext<AuthContextValue | null>(null);

interface AuthProviderProps {
  children: React.ReactNode;
}

const devUser: AuthUser = {
  user_id: 'dev-user',
  tenant_id: '00000000-0000-0000-0000-000000000001',
  email: 'dev@patchiq.local',
  name: 'Dev User',
  preferred_username: 'dev-user',
  role: 'admin',
  roles: ['admin'],
  permissions: [{ resource: '*', action: '*', scope: '*' }],
};
const isDev = import.meta.env.DEV;

export function checkPermission(
  permissions: AuthUser['permissions'],
  resource: string,
  action: string,
): boolean {
  if (permissions === undefined) return true;
  if (permissions.length === 0) return false;
  return permissions.some((p) => {
    const resourceMatch = p.resource === '*' || p.resource === resource;
    const actionMatch = p.action === '*' || p.action === action;
    return resourceMatch && actionMatch;
  });
}

export const AuthProvider = ({ children }: AuthProviderProps) => {
  const { data: user, isLoading, isError } = useCurrentUser();

  const effectiveUser = user ?? (isError && isDev ? devUser : null);

  // Keep the active tenant store aligned with the authenticated session:
  // prefer an already-stored tenant (from a previous session/tab) if it's
  // in the accessible set, otherwise fall back to the user's primary
  // tenant_id. Re-run when the effective user changes.
  useEffect(() => {
    if (!effectiveUser) return;
    const accessible = effectiveUser.accessible_tenants ?? [];
    const stored = getActiveTenantId();
    const storedStillValid =
      stored && accessible.length > 0 ? accessible.some((t) => t.id === stored) : Boolean(stored);
    if (storedStillValid) return;
    const fallback = effectiveUser.active_tenant_id ?? effectiveUser.tenant_id ?? null;
    if (fallback) {
      setActiveTenantId(fallback);
    }
  }, [effectiveUser]);

  const can = useCallback(
    (resource: string, action: string): boolean => {
      if (!effectiveUser) return false;
      // An org-scoped grant (MSP Admin etc.) satisfies regardless of which
      // tenant is active.
      if (checkPermission(effectiveUser.org_permissions, resource, action)) return true;
      return checkPermission(effectiveUser.permissions, resource, action);
    },
    [effectiveUser],
  );

  const isMspOperator = Boolean(
    effectiveUser?.organization?.type === 'msp' ||
      (effectiveUser?.accessible_tenants && effectiveUser.accessible_tenants.length > 1),
  );

  if (isLoading) {
    return <div className="flex items-center justify-center h-screen">Loading...</div>;
  }

  if (!effectiveUser) {
    return <div className="flex items-center justify-center h-screen">Loading...</div>;
  }

  return (
    <AuthContext.Provider value={{ user: effectiveUser, can, isMspOperator }}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = (): AuthContextValue => {
  const ctx = useContext(AuthContext);
  if (ctx === null) {
    throw new Error('useAuth must be used within an <AuthProvider>');
  }
  return ctx;
};

export const useCan = (): ((resource: string, action: string) => boolean) => {
  const { can } = useAuth();
  return can;
};
