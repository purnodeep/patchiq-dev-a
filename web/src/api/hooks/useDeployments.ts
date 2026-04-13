import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../client';
import type { components } from '../types';

type CreateDeploymentRequest = components['schemas']['CreateDeploymentRequest'];
type DeploymentWave = components['schemas']['DeploymentWave'];
type WaveConfig = components['schemas']['WaveConfig'];
type DeploymentSchedule = components['schemas']['DeploymentSchedule'];

export type { DeploymentWave, WaveConfig, DeploymentSchedule };

const TERMINAL_STATUSES: string[] = [
  'completed',
  'failed',
  'cancelled',
  'rolled_back',
  'rollback_failed',
];

export function useDeployments(params?: {
  cursor?: string;
  limit?: number;
  status?: string;
  created_after?: string;
  created_before?: string;
}) {
  return useQuery({
    queryKey: ['deployments', params],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/deployments', {
        params: { query: params },
      });
      if (error) throw error;
      return data;
    },
    refetchInterval: 30_000,
  });
}

export function useDeployment(id: string) {
  return useQuery({
    queryKey: ['deployments', id],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/deployments/{id}', {
        params: { path: { id } },
      });
      if (error) throw error;
      return data;
    },
    enabled: !!id,
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      if (status && TERMINAL_STATUSES.includes(status)) return false;
      return 5_000;
    },
  });
}

export function useCreateDeployment() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: CreateDeploymentRequest) => {
      const { data, error } = await api.POST('/api/v1/deployments', {
        body,
      });
      if (error) {
        // error is the parsed JSON body: { code, message, details }
        const apiErr = error as { code?: string; message?: string };
        const msg = apiErr.message || 'Failed to create deployment';
        const err = new Error(msg);
        (err as Error & { code?: string }).code = apiErr.code;
        throw err;
      }
      return data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['deployments'] });
      void queryClient.invalidateQueries({ queryKey: ['endpoints'] });
    },
  });
}

export function useCancelDeployment() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const { data, error } = await api.POST('/api/v1/deployments/{id}/cancel', {
        params: { path: { id } },
      });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['deployments'] });
      void queryClient.invalidateQueries({ queryKey: ['endpoints'] });
    },
  });
}

export function useDeploymentWaves(deploymentId: string) {
  return useQuery({
    queryKey: ['deployments', deploymentId, 'waves'],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/deployments/{id}/waves', {
        params: { path: { id: deploymentId } },
      });
      if (error) throw error;
      return data;
    },
    enabled: !!deploymentId,
    refetchInterval: 5_000,
  });
}

export function useDeploymentSchedules() {
  return useQuery({
    queryKey: ['deployment-schedules'],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/deployment-schedules', {});
      if (error) throw error;
      return data;
    },
  });
}

export function useCreateSchedule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: components['schemas']['CreateScheduleRequest']) => {
      const { data, error } = await api.POST('/api/v1/deployment-schedules', {
        body,
      });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['deployment-schedules'] });
    },
  });
}

export function useRetryDeployment() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const { data, error } = await api.POST('/api/v1/deployments/{id}/retry', {
        params: { path: { id } },
      });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['deployments'] });
      void queryClient.invalidateQueries({ queryKey: ['endpoints'] });
    },
  });
}

export function useRollbackDeployment() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const { data, error } = await api.POST('/api/v1/deployments/{id}/rollback', {
        params: { path: { id } },
      });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['deployments'] });
      void queryClient.invalidateQueries({ queryKey: ['endpoints'] });
    },
  });
}

export function useDeploymentPatches(id: string) {
  return useQuery({
    queryKey: ['deployments', id, 'patches'],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/deployments/{id}/patches', {
        params: { path: { id } },
      });
      if (error) throw error;
      return data;
    },
    enabled: !!id,
  });
}

export function useDeleteSchedule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const { data, error } = await api.DELETE('/api/v1/deployment-schedules/{id}', {
        params: { path: { id } },
      });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['deployment-schedules'] });
    },
  });
}
