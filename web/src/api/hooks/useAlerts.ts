import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../client';

export interface AlertFilters {
  cursor?: string;
  limit?: number;
  severity?: string;
  category?: string;
  status?: string;
  search?: string;
  from_date?: string;
  to_date?: string;
}

export function useAlerts(params?: AlertFilters, refetchInterval?: number) {
  return useQuery({
    queryKey: ['alerts', params],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/alerts', {
        params: { query: params as Record<string, unknown> },
      });
      if (error) throw error;
      return data;
    },
    refetchInterval: refetchInterval ?? undefined,
  });
}

export function useAlertCount(
  params?: { from_date?: string; to_date?: string },
  refetchInterval = 30000,
) {
  return useQuery({
    queryKey: ['alerts', 'count', params],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/alerts/count', {
        params: { query: params as never },
      });
      if (error) throw error;
      return data;
    },
    refetchInterval,
  });
}

export function useUpdateAlertStatus() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, status }: { id: string; status: string }) => {
      const { data, error } = await api.PATCH('/api/v1/alerts/{id}/status', {
        params: { path: { id } },
        body: { status } as never,
      });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['alerts'] });
      qc.invalidateQueries({ queryKey: ['alerts', 'count'] });
    },
  });
}

export function useBulkUpdateAlertStatus() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({ ids, status }: { ids: string[]; status: string }) => {
      const { data, error } = await api.PATCH('/api/v1/alerts/bulk-status', {
        body: { ids, status } as never,
      });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['alerts'] });
      qc.invalidateQueries({ queryKey: ['alerts', 'count'] });
    },
  });
}

export function useAlertRules() {
  return useQuery({
    queryKey: ['alert-rules'],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/alert-rules');
      if (error) throw error;
      return data;
    },
  });
}

export function useCreateAlertRule() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (body: {
      event_type: string;
      severity: string;
      category: string;
      title_template: string;
      description_template: string;
      enabled: boolean;
    }) => {
      const { data, error } = await api.POST('/api/v1/alert-rules', {
        body: body as never,
      });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['alert-rules'] });
    },
  });
}

export function useUpdateAlertRule() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({
      id,
      ...body
    }: {
      id: string;
      event_type: string;
      severity: string;
      category: string;
      title_template: string;
      description_template: string;
      enabled: boolean;
    }) => {
      const { data, error } = await api.PUT('/api/v1/alert-rules/{id}', {
        params: { path: { id } },
        body: body as never,
      });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['alert-rules'] });
    },
  });
}

export function useDeleteAlertRule() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const { error } = await api.DELETE('/api/v1/alert-rules/{id}', {
        params: { path: { id } },
      });
      if (error) throw error;
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['alert-rules'] });
    },
  });
}
