import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  listClients,
  getClient,
  approveClient,
  declineClient,
  suspendClient,
  updateClient,
  deleteClient,
  getPendingClientCount,
  getClientSyncHistory,
  getClientEndpointTrend,
} from '../clients';
export type { SyncHistoryItem, EndpointTrendPoint } from '../clients';

export function useClients(params?: { limit?: number; offset?: number; status?: string }) {
  return useQuery({
    queryKey: ['clients', params],
    queryFn: () => listClients(params ?? {}),
  });
}

export function useClient(id: string | undefined) {
  return useQuery({
    queryKey: ['clients', id],
    queryFn: () => getClient(id!),
    enabled: !!id,
  });
}

export function usePendingClientCount() {
  return useQuery({
    queryKey: ['clients', 'pending-count'],
    queryFn: getPendingClientCount,
    refetchInterval: 30_000,
  });
}

export function useApproveClient() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: approveClient,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['clients'] });
      void queryClient.invalidateQueries({ queryKey: ['clients', 'pending-count'] });
    },
  });
}

export function useDeclineClient() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: declineClient,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['clients'] });
      void queryClient.invalidateQueries({ queryKey: ['clients', 'pending-count'] });
    },
  });
}

export function useSuspendClient() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: suspendClient,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['clients'] });
    },
  });
}

export function useUpdateClient() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: { sync_interval?: number; notes?: string } }) =>
      updateClient(id, data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['clients'] });
    },
  });
}

export function useDeleteClient() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: deleteClient,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['clients'] });
    },
  });
}

export function useClientSyncHistory(
  clientId: string | undefined,
  limit?: number,
  offset?: number,
) {
  return useQuery({
    queryKey: ['clients', clientId, 'sync-history', { limit, offset }],
    queryFn: () => getClientSyncHistory(clientId!, { limit, offset }),
    enabled: !!clientId,
  });
}

export function useClientEndpointTrend(clientId: string | undefined, days: number) {
  return useQuery({
    queryKey: ['clients', clientId, 'endpoint-trend', days],
    queryFn: () => getClientEndpointTrend(clientId!, days),
    enabled: !!clientId,
  });
}
