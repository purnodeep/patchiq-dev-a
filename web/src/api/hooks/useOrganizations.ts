import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../client';

export function useOrganizations() {
  return useQuery({
    queryKey: ['organizations'],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/organizations');
      if (error) throw error;
      return data;
    },
  });
}

export function useOrganization(id: string | undefined) {
  return useQuery({
    queryKey: ['organizations', id],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/organizations/{id}', {
        params: { path: { id: id! } },
      });
      if (error) throw error;
      return data;
    },
    enabled: Boolean(id),
  });
}

export function useOrgTenants(orgId: string | undefined) {
  return useQuery({
    queryKey: ['organizations', orgId, 'tenants'],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/organizations/{id}/tenants', {
        params: { path: { id: orgId! } },
      });
      if (error) throw error;
      return data;
    },
    enabled: Boolean(orgId),
  });
}

export interface ProvisionTenantInput {
  orgId: string;
  body: {
    name: string;
    slug: string;
    license_id?: string;
  };
}

export function useProvisionTenant() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({ orgId, body }: ProvisionTenantInput) => {
      const { data, error } = await api.POST('/api/v1/organizations/{id}/tenants', {
        params: { path: { id: orgId } },
        body,
      });
      if (error) throw error;
      return data;
    },
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: ['organizations'] });
      qc.invalidateQueries({ queryKey: ['organizations', variables.orgId, 'tenants'] });
    },
  });
}

export function useOrgDashboard(orgId: string | undefined) {
  return useQuery({
    queryKey: ['organizations', orgId, 'dashboard'],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/organizations/{id}/dashboard', {
        params: { path: { id: orgId! } },
      });
      if (error) throw error;
      return data;
    },
    enabled: Boolean(orgId),
    refetchInterval: 60_000,
  });
}
