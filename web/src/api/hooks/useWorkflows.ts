import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../client';
import type { components } from '../types';

export type Workflow = components['schemas']['Workflow'];
export type WorkflowDetailResponse = components['schemas']['WorkflowDetailResponse'];
export type WorkflowResponse = components['schemas']['WorkflowResponse'];
export type WorkflowVersion = components['schemas']['WorkflowVersion'];
export type WorkflowTemplate = components['schemas']['WorkflowTemplate'];
export type WorkflowExecution = components['schemas']['WorkflowExecution'];
export type ExecutionDetailResponse = components['schemas']['ExecutionDetailResponse'];
export type CreateWorkflowRequest = components['schemas']['CreateWorkflowRequest'];

export function useWorkflows(params?: { cursor?: string; limit?: number; search?: string }) {
  return useQuery({
    queryKey: ['workflows', params],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/workflows', {
        params: { query: params },
      });
      if (error) throw error;
      return data;
    },
    refetchInterval: 30_000,
  });
}

export function useWorkflow(id: string) {
  return useQuery({
    queryKey: ['workflows', id],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/workflows/{id}', {
        params: { path: { id } },
      });
      if (error) throw error;
      return data;
    },
    enabled: !!id,
  });
}

export function useCreateWorkflow() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: CreateWorkflowRequest) => {
      const { data, error } = await api.POST('/api/v1/workflows', {
        body,
      });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['workflows'] });
    },
  });
}

export function useUpdateWorkflow() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, body }: { id: string; body: CreateWorkflowRequest }) => {
      const { data, error } = await api.PUT('/api/v1/workflows/{id}', {
        params: { path: { id } },
        body,
      });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['workflows'] });
    },
  });
}

export function useDeleteWorkflow() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const { error } = await api.DELETE('/api/v1/workflows/{id}', {
        params: { path: { id } },
      });
      if (error) throw error;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['workflows'] });
    },
  });
}

export function usePublishWorkflow() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const { data, error } = await api.PUT('/api/v1/workflows/{id}/publish', {
        params: { path: { id } },
      });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['workflows'] });
    },
  });
}

export function useWorkflowVersions(id: string) {
  return useQuery({
    queryKey: ['workflows', id, 'versions'],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/workflows/{id}/versions', {
        params: { path: { id } },
      });
      if (error) throw error;
      return data;
    },
    enabled: !!id,
  });
}

export function useWorkflowTemplates() {
  return useQuery({
    queryKey: ['workflow-templates'],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/workflow-templates', {});
      if (error) throw error;
      return data;
    },
  });
}

export function useExecuteWorkflow() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const { data, error } = await api.POST('/api/v1/workflows/{id}/execute', {
        params: { path: { id } },
      });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['workflows'] });
    },
  });
}

export function useWorkflowExecutions(id: string, params?: { limit?: number; status?: string }) {
  return useQuery({
    queryKey: ['workflows', id, 'executions', params],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/workflows/{id}/executions', {
        params: { path: { id }, query: params },
      });
      if (error) throw error;
      return data;
    },
    enabled: !!id,
    refetchInterval: 5_000,
  });
}

export function useWorkflowExecution(workflowId: string, execId: string) {
  return useQuery({
    queryKey: ['workflows', workflowId, 'executions', execId],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/workflows/{id}/executions/{execId}', {
        params: { path: { id: workflowId, execId } },
      });
      if (error) throw error;
      return data;
    },
    enabled: !!workflowId && !!execId,
  });
}

export function useCancelWorkflowExecution() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ workflowId, execId }: { workflowId: string; execId: string }) => {
      const { data, error } = await api.POST('/api/v1/workflows/{id}/executions/{execId}/cancel', {
        params: { path: { id: workflowId, execId } },
      });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['workflows'] });
    },
  });
}

export function useApproveWorkflowExecution() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({
      workflowId,
      execId,
      comment,
    }: {
      workflowId: string;
      execId: string;
      comment?: string;
    }) => {
      const { data, error } = await api.POST('/api/v1/workflows/{id}/executions/{execId}/approve', {
        params: { path: { id: workflowId, execId } },
        body: comment ? { comment } : undefined,
      });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['workflows'] });
    },
  });
}

export function useRejectWorkflowExecution() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({
      workflowId,
      execId,
      comment,
    }: {
      workflowId: string;
      execId: string;
      comment?: string;
    }) => {
      const { data, error } = await api.POST('/api/v1/workflows/{id}/executions/{execId}/reject', {
        params: { path: { id: workflowId, execId } },
        body: comment ? { comment } : undefined,
      });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['workflows'] });
    },
  });
}
