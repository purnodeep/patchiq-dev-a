import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../client';
import type { components } from '../types';

export type HubSyncStatus = components['schemas']['HubSyncStatus'];
export type SyncConfigRequest = components['schemas']['SyncConfigRequest'];

export class NotConfiguredError extends Error {
  constructor() {
    super('Hub sync not configured');
    this.name = 'NotConfiguredError';
  }
}

export function isNotConfiguredError(error: unknown): error is NotConfiguredError {
  return error instanceof Error && error.name === 'NotConfiguredError';
}

export function useHubSyncStatus() {
  return useQuery({
    queryKey: ['hubSync', 'status'],
    queryFn: async () => {
      const res = await api.GET('/api/v1/sync/status', {});
      if (res.error) {
        if (res.response.status === 404) throw new NotConfiguredError();
        throw res.error;
      }
      return res.data;
    },
    retry: (failureCount, error) => {
      if (isNotConfiguredError(error)) return false;
      return failureCount < 3;
    },
    refetchInterval: 30_000,
  });
}

export function useTriggerHubSync() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      const { data, error } = await api.POST('/api/v1/sync/trigger', {});
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['hubSync'] });
      void queryClient.invalidateQueries({ queryKey: ['catalog'] });
    },
  });
}

export function useUpdateSyncConfig() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (config: SyncConfigRequest) => {
      const { data, error } = await api.PUT('/api/v1/sync/config', {
        body: config,
      });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['hubSync'] });
    },
  });
}
