import { useQuery } from '@tanstack/react-query';
import { api } from '../client';

export function usePendingPatches(params?: { cursor?: string; limit?: number }) {
  return useQuery({
    queryKey: ['patches', 'pending', params],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/patches/pending', {
        params: { query: params },
      });
      if (error) throw error;
      return data;
    },
  });
}
