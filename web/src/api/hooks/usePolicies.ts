import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../client';
import type { components } from '../types';

type CreatePolicyRequest = components['schemas']['CreatePolicyRequest'];
type UpdatePolicyRequest = components['schemas']['UpdatePolicyRequest'];
type BulkPolicyRequest = components['schemas']['BulkPolicyRequest'];

export function usePolicies(params?: {
  cursor?: string;
  limit?: number;
  mode?: 'automatic' | 'manual' | 'advisory';
  enabled?: string;
  search?: string;
}) {
  return useQuery({
    queryKey: ['policies', params],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/policies', {
        params: { query: params },
      });
      if (error) throw error;
      return data;
    },
  });
}

export function usePolicy(id: string) {
  return useQuery({
    queryKey: ['policies', id],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/policies/{id}', {
        params: { path: { id } },
      });
      if (error) throw error;
      return data;
    },
    enabled: !!id,
  });
}

export function useCreatePolicy() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: CreatePolicyRequest) => {
      const { data, error } = await api.POST('/api/v1/policies', {
        body,
      });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['policies'] });
      void queryClient.invalidateQueries({ queryKey: ['endpoints'] });
    },
  });
}

export function useUpdatePolicy(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: UpdatePolicyRequest) => {
      const { data, error } = await api.PUT('/api/v1/policies/{id}', {
        params: { path: { id } },
        body,
      });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['policies'] });
      void queryClient.invalidateQueries({ queryKey: ['endpoints'] });
    },
  });
}

export function useTogglePolicy(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (enabled: boolean) => {
      // PATCH not in generated openapi-fetch client; use PUT with partial body
      const { data, error } = await api.PUT('/api/v1/policies/{id}', {
        params: { path: { id } },
        body: { enabled } as UpdatePolicyRequest,
      });
      if (error) throw error;
      return data;
    },
    onMutate: async (enabled) => {
      await queryClient.cancelQueries({ queryKey: ['policies'] });
      const previous = queryClient.getQueriesData({ queryKey: ['policies'] });
      queryClient.setQueriesData(
        { queryKey: ['policies'] },
        (old: { data: { id: string; enabled: boolean }[] } | undefined) => {
          if (!old) return old;
          return {
            ...old,
            data: old.data.map((p) => (p.id === id ? { ...p, enabled } : p)),
          };
        },
      );
      return { previous };
    },
    onError: (_err, _vars, context) => {
      if (context?.previous) {
        for (const [key, value] of context.previous) {
          queryClient.setQueryData(key, value);
        }
      }
    },
    onSettled: () => {
      void queryClient.invalidateQueries({ queryKey: ['policies'] });
      void queryClient.invalidateQueries({ queryKey: ['endpoints'] });
    },
  });
}

export function useEvaluatePolicy(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      const { data, error } = await api.POST('/api/v1/policies/{id}/evaluate', {
        params: { path: { id } },
      });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['policies', id] });
      void queryClient.invalidateQueries({ queryKey: ['endpoints'] });
    },
  });
}

export function useDeletePolicy(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      const { error } = await api.DELETE('/api/v1/policies/{id}', {
        params: { path: { id } },
      });
      if (error) throw error;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['policies'] });
      void queryClient.invalidateQueries({ queryKey: ['endpoints'] });
    },
  });
}

export function useBulkPolicyAction() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: BulkPolicyRequest) => {
      const { data, error } = await api.POST('/api/v1/policies/bulk', {
        body,
      });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['policies'] });
      void queryClient.invalidateQueries({ queryKey: ['endpoints'] });
    },
  });
}
