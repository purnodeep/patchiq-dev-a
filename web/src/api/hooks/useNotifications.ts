import { useQuery, useMutation, useInfiniteQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../client';
import type { components } from '../types';

// Re-export type aliases so consumers can still import them
export type NotificationChannel = components['schemas']['NotificationChannel'];
export type NotificationPreference = components['schemas']['NotificationPreference'];
export type NotificationHistoryEntry = components['schemas']['NotificationHistoryEntry'];
export type NotificationPreferencesResponse =
  components['schemas']['NotificationPreferencesResponse'];
export type DigestConfigResponse = components['schemas']['DigestConfigResponse'];
export type CreateChannelRequest = components['schemas']['CreateChannelRequest'];
export type UpdateChannelRequest = components['schemas']['UpdateChannelRequest'];

// Channel hooks
export function useNotificationChannels() {
  return useQuery({
    queryKey: ['notification-channels'],
    queryFn: async () => {
      const { data, error, response } = await api.GET('/api/v1/notifications/channels', {});
      // 404 means the endpoint isn't configured yet — treat as empty list
      if (response?.status === 404) return [] as NotificationChannel[];
      if (error) throw error;
      return (data ?? []) as NotificationChannel[];
    },
  });
}

export function useCreateChannel() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (body: CreateChannelRequest) => {
      const { data, error } = await api.POST('/api/v1/notifications/channels', {
        body,
      });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['notification-channels'] });
    },
  });
}

export function useUpdateChannel() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, data: body }: { id: string; data: UpdateChannelRequest }) => {
      const { data, error } = await api.PUT('/api/v1/notifications/channels/{id}', {
        params: { path: { id } },
        body,
      });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['notification-channels'] });
    },
  });
}

export function useDeleteChannel() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const { error } = await api.DELETE('/api/v1/notifications/channels/{id}', {
        params: { path: { id } },
      });
      if (error) throw error;
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['notification-channels'] });
    },
  });
}

export function useTestChannel() {
  return useMutation({
    mutationFn: async (id: string) => {
      const { data, error } = await api.POST('/api/v1/notifications/channels/{id}/test', {
        params: { path: { id } },
      });
      if (error) throw error;
      return data;
    },
  });
}

// Preference hooks
export function useNotificationPreferences() {
  return useQuery({
    queryKey: ['notification-preferences'],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/notifications/preferences', {});
      if (error) throw error;
      return data as NotificationPreferencesResponse;
    },
  });
}

export function useUpdatePreferences() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (preferences: components['schemas']['PreferenceInput'][]) => {
      const { data, error } = await api.PUT('/api/v1/notifications/preferences', {
        body: { preferences },
      });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['notification-preferences'] });
    },
  });
}

// History hooks
export interface HistoryFilters {
  trigger_type?: string;
  status?: string;
  channel_type?: string;
  category?: string;
  from?: string;
  to?: string;
}

export function useNotificationHistory(filters?: HistoryFilters, limit = 50) {
  return useInfiniteQuery({
    queryKey: ['notification-history', filters],
    queryFn: async ({ pageParam }: { pageParam?: string }) => {
      const { data, error } = await api.GET('/api/v1/notifications/history', {
        params: {
          query: {
            limit,
            cursor: pageParam,
            ...filters,
          },
        },
      });
      if (error) throw error;
      return data;
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.next_cursor ?? undefined,
  });
}

export function useRetryNotification() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const { data, error } = await api.POST('/api/v1/notifications/history/{id}/retry', {
        params: { path: { id } },
      });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['notification-history'] });
    },
  });
}

// Digest config hooks
export function useDigestConfig() {
  return useQuery({
    queryKey: ['notification-digest-config'],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/notifications/digest-config', {});
      if (error) throw error;
      return data as DigestConfigResponse;
    },
  });
}

export function useUpdateDigestConfig() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (body: components['schemas']['UpdateDigestConfigRequest']) => {
      const { data, error } = await api.PUT('/api/v1/notifications/digest-config', {
        body,
      });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['notification-digest-config'] });
    },
  });
}

export function useTestDigest() {
  return useMutation({
    mutationFn: async () => {
      const { data, error } = await api.POST('/api/v1/notifications/digest/test', {});
      if (error) throw error;
      return data;
    },
  });
}
