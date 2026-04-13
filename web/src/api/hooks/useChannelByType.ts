import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../client';

export interface ChannelConfig {
  id: string;
  type: string;
  name: string;
  enabled: boolean;
  config: Record<string, unknown>;
  last_tested_at?: string;
  last_test_status?: string;
}

export function useChannelByType(type: string) {
  return useQuery({
    queryKey: ['notification-channel', type],
    queryFn: async () => {
      const { data, error } = await api.GET(
        // eslint-disable-next-line @typescript-eslint/no-explicit-any -- TODO(#332): remove cast after OpenAPI spec regeneration
        `/api/v1/notifications/channels/by-type/${type}` as any,
        {},
      );
      if (error) throw error;
      return data as ChannelConfig;
    },
    retry: false,
  });
}

export function useUpdateChannelByType(type: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (body: { name: string; config: Record<string, unknown> }) => {
      const { data, error } = await api.PUT(
        // eslint-disable-next-line @typescript-eslint/no-explicit-any -- TODO(#332): remove cast after OpenAPI spec regeneration
        `/api/v1/notifications/channels/by-type/${type}` as any,
        {
          body,
        },
      );
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['notification-channel', type] });
    },
  });
}

export function useTestChannelByType(type: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      const { data, error } = await api.POST(
        // eslint-disable-next-line @typescript-eslint/no-explicit-any -- TODO(#332): remove cast after OpenAPI spec regeneration
        `/api/v1/notifications/channels/by-type/${type}/test` as any,
        {},
      );
      if (error) throw error;
      return data as { success: boolean; error?: string };
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['notification-channel', type] });
    },
  });
}
