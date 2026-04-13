import { useQuery } from '@tanstack/react-query';
import { api } from '../client';

export function useLogs(params?: {
  cursor?: string;
  limit?: number;
  level?: 'debug' | 'info' | 'warn' | 'error';
  refetchInterval?: number;
}) {
  const { refetchInterval, ...queryParams } = params ?? {};
  return useQuery({
    queryKey: ['logs', queryParams],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/logs', {
        params: { query: queryParams },
      });
      if (error) throw error;
      return data;
    },
    refetchInterval: refetchInterval,
  });
}
