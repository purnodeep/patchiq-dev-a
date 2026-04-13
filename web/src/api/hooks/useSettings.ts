import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../client';

export interface GeneralSettings {
  org_name: string;
  timezone: string;
  date_format: string;
  scan_interval_hours: number;
}

export function useSettings() {
  return useQuery({
    queryKey: ['settings'],
    queryFn: async () => {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any -- TODO(#332): remove cast after OpenAPI spec regeneration
      const { data, error } = await api.GET('/api/v1/settings' as any, {});
      if (error) throw error;
      return data as GeneralSettings;
    },
  });
}

export function useUpdateSettings() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (body: GeneralSettings) => {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any -- TODO(#332): remove cast after OpenAPI spec regeneration
      const { data, error } = await api.PUT('/api/v1/settings' as any, {
        body,
      });
      if (error) throw error;
      return data as GeneralSettings;
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['settings'] });
    },
  });
}
