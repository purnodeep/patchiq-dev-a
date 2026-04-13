import { useQuery } from '@tanstack/react-query';
import type { LiveMetrics } from '../../types/metrics';

export function useMetrics() {
  return useQuery<LiveMetrics>({
    queryKey: ['metrics'],
    queryFn: async () => {
      const res = await fetch('/api/v1/metrics');
      if (!res.ok) throw new Error('Failed to fetch metrics');
      return res.json() as Promise<LiveMetrics>;
    },
    refetchInterval: 3000,
  });
}
