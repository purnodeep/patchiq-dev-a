import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

interface AccessibleTenant {
  id: string;
  name: string;
  slug: string;
}

interface OrganizationInfo {
  id: string;
  name: string;
  slug: string;
  type: 'direct' | 'msp' | 'reseller';
}

interface AuthUser {
  user_id: string;
  tenant_id: string;
  email?: string;
  name?: string;
  roles?: string[];
  role?: string;
  permissions?: Array<{ resource: string; action: string; scope: string }>;
  // Organization / MSP fields. Optional because older backend versions and
  // single-tenant deployments do not populate them.
  organization?: OrganizationInfo;
  active_tenant_id?: string;
  accessible_tenants?: AccessibleTenant[];
  org_permissions?: Array<{ resource: string; action: string; scope: string }>;
}

export function useCurrentUser() {
  return useQuery({
    queryKey: ['auth', 'me'],
    queryFn: async (): Promise<AuthUser> => {
      const res = await fetch('/api/v1/auth/me', {
        credentials: 'include',
      });
      if (!res.ok) throw new Error(`auth/me failed: ${res.status}`);
      return res.json() as Promise<AuthUser>;
    },
    retry: 2,
    staleTime: 5 * 60 * 1000, // Permissions refresh every 5 min; role changes have delayed UI effect
  });
}

export function useLogout() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      // Logout returns a 302 redirect with no body, which doesn't fit openapi-fetch's
      // typed response model. Use plain fetch with credentials for this endpoint.
      const res = await fetch('/api/v1/auth/logout', {
        method: 'POST',
        credentials: 'include',
        redirect: 'manual',
      });
      // 302 redirect (opaqueredirect with manual) or 2xx are both success
      if (res.type !== 'opaqueredirect' && !res.ok) {
        throw new Error(`Logout failed (status ${res.status})`);
      }
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['auth'] });
      window.location.href = '/login';
    },
  });
}
