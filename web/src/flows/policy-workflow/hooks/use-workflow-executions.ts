import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import type { PaginatedList, WorkflowExecution, ExecutionDetail } from '../types';
import { fetchJSON } from './fetch-json';

export function useWorkflowExecutions(
  workflowId: string,
  params?: { status?: string; limit?: number },
) {
  const query = new URLSearchParams();
  if (params?.status) query.set('status', params.status);
  if (params?.limit) query.set('limit', String(params.limit));
  const qs = query.toString();

  return useQuery({
    queryKey: ['workflows', workflowId, 'executions', params],
    queryFn: () =>
      fetchJSON<PaginatedList<WorkflowExecution>>(
        `/api/v1/workflows/${workflowId}/executions${qs ? `?${qs}` : ''}`,
      ),
    enabled: !!workflowId,
  });
}

export function useWorkflowExecution(
  workflowId: string,
  execId: string,
  opts?: { refetchInterval?: number },
) {
  return useQuery({
    queryKey: ['workflows', workflowId, 'executions', execId],
    queryFn: () =>
      fetchJSON<ExecutionDetail>(`/api/v1/workflows/${workflowId}/executions/${execId}`),
    enabled: !!workflowId && !!execId,
    refetchInterval: opts?.refetchInterval,
  });
}

export function useExecuteWorkflow(workflowId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      fetchJSON<{ id: string; status: string }>(`/api/v1/workflows/${workflowId}/execute`, {
        method: 'POST',
      }),
    onSuccess: () => {
      toast.success('Workflow execution started');
      void queryClient.invalidateQueries({ queryKey: ['workflows', workflowId, 'executions'] });
    },
    onError: (err: Error) => {
      toast.error(`Execution failed: ${err.message}`);
    },
  });
}

export function useCancelExecution(workflowId: string, execId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      fetchJSON<{ id: string; status: string }>(
        `/api/v1/workflows/${workflowId}/executions/${execId}/cancel`,
        { method: 'POST' },
      ),
    throwOnError: true,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['workflows', workflowId, 'executions'] });
    },
  });
}
