import { useQuery } from '@tanstack/react-query';
import type { ServiceInfo } from '../../types/software';

export function useAgentServices() {
  return useQuery({
    queryKey: ['services'],
    queryFn: async (): Promise<ServiceInfo[]> => {
      const res = await fetch('/api/v1/services');
      if (!res.ok) {
        throw new Error(`Failed to fetch services info: ${res.status} ${res.statusText}`);
      }
      return res.json() as Promise<ServiceInfo[]>;
    },
  });
}
