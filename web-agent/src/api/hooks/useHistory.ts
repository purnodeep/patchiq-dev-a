import { useQuery } from '@tanstack/react-query';
import { api } from '../client';

export function useHistory(params?: {
  cursor?: string;
  limit?: number;
  date_range?: '24h' | '7d' | '30d';
}) {
  return useQuery({
    queryKey: ['history', params],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/history', {
        params: { query: params },
      });
      if (error) throw error;
      return data;
    },
  });
}
