import { useQuery } from '@tanstack/react-query';
import { api } from '../client';

export function useAgentStatus() {
  return useQuery({
    queryKey: ['status'],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/status');
      if (error) throw error;
      return data;
    },
    refetchInterval: 10_000,
  });
}
