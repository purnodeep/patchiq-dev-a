import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../client';

export interface RoleMapping {
  external_role: string;
  patchiq_role_id: string;
  role_name: string;
}

export interface IAMSettings {
  sso_url: string;
  client_id: string;
  zitadel_org_id: string;
  user_sync_enabled: boolean;
  user_sync_interval_minutes: number;
  connection_status: string;
  last_tested_at?: string;
  role_mappings: RoleMapping[];
}

export interface IAMTestResult {
  success: boolean;
  latency_ms: number;
  error?: string;
}

export function useIAMSettings(reveal = false) {
  return useQuery({
    queryKey: ['settings', 'iam', reveal],
    queryFn: async () => {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any -- OpenAPI spec missing query.reveal param
      const { data, error } = await api.GET('/api/v1/settings/iam' as any, {
        params: {
          query: reveal ? { reveal: 'true' } : undefined,
        },
      });
      if (error) throw error;
      return data as IAMSettings;
    },
  });
}

export function useUpdateIAMSettings() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (body: {
      sso_url: string;
      client_id: string;
      zitadel_org_id: string;
      user_sync_enabled: boolean;
      user_sync_interval_minutes: number;
    }) => {
      const { data, error } = await api.PUT('/api/v1/settings/iam', {
        body,
      });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['settings', 'iam'] });
    },
  });
}

export function useTestIAMConnection() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any -- TODO(#332): remove cast after OpenAPI spec regeneration
      const { data, error } = await api.POST('/api/v1/settings/iam/test' as any, {});
      if (error) throw error;
      return data as IAMTestResult;
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['settings', 'iam'] });
    },
  });
}

export function useUpdateRoleMappings() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (mappings: Array<{ external_role: string; patchiq_role_id: string }>) => {
      const { data, error } = await api.PUT('/api/v1/settings/role-mapping', {
        body: { mappings },
      });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['settings', 'iam'] });
    },
  });
}
