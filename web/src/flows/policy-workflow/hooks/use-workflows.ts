import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import type {
  PaginatedList,
  WorkflowListItem,
  WorkflowDetail,
  WorkflowResponse,
  WorkflowRequest,
  WorkflowTemplate,
  WorkflowVersion,
} from '../types';
import { fetchJSON, fetchVoid } from './fetch-json';

export function useWorkflows(params?: {
  cursor?: string;
  limit?: number;
  status?: string;
  search?: string;
}) {
  const query = new URLSearchParams();
  if (params?.cursor) query.set('cursor', params.cursor);
  if (params?.limit) query.set('limit', String(params.limit));
  if (params?.status) query.set('status', params.status);
  if (params?.search) query.set('search', params.search);
  const qs = query.toString();

  return useQuery({
    queryKey: ['workflows', params],
    queryFn: () =>
      fetchJSON<PaginatedList<WorkflowListItem>>(`/api/v1/workflows${qs ? `?${qs}` : ''}`),
  });
}

export function useWorkflow(id: string) {
  return useQuery({
    queryKey: ['workflows', id],
    queryFn: () => fetchJSON<WorkflowDetail>(`/api/v1/workflows/${id}`),
    enabled: !!id,
  });
}

export function useCreateWorkflow() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: WorkflowRequest) =>
      fetchJSON<WorkflowResponse>('/api/v1/workflows', {
        method: 'POST',
        body: JSON.stringify(body),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['workflows'] });
    },
    onError: (err: Error) => {
      toast.error(`Failed to create workflow: ${err.message}`);
    },
  });
}

export function useUpdateWorkflow(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: WorkflowRequest) =>
      fetchJSON<WorkflowResponse>(`/api/v1/workflows/${id}`, {
        method: 'PUT',
        body: JSON.stringify(body),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['workflows'] });
    },
    onError: (err: Error) => {
      toast.error(`Failed to update workflow: ${err.message}`);
    },
  });
}

export function useDeleteWorkflow(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => fetchVoid(`/api/v1/workflows/${id}`, { method: 'DELETE' }),
    throwOnError: true,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['workflows'] });
    },
    onError: (err: Error) => {
      toast.error(`Failed to delete workflow: ${err.message}`);
    },
  });
}

export function usePublishWorkflow(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      fetchJSON<WorkflowVersion>(`/api/v1/workflows/${id}/publish`, { method: 'PUT' }),
    onSuccess: () => {
      toast.success('Workflow published');
      void queryClient.invalidateQueries({ queryKey: ['workflows'] });
    },
    onError: (err: Error) => {
      toast.error(`Publish failed: ${err.message}`);
    },
  });
}

export function useWorkflowVersions(id: string) {
  return useQuery({
    queryKey: ['workflows', id, 'versions'],
    queryFn: () => fetchJSON<WorkflowVersion[]>(`/api/v1/workflows/${id}/versions`),
    enabled: !!id,
  });
}

export function useWorkflowTemplates() {
  return useQuery({
    queryKey: ['workflow-templates'],
    queryFn: () => fetchJSON<WorkflowTemplate[]>('/api/v1/workflow-templates'),
  });
}
