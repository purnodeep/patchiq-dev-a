import { useQuery } from '@tanstack/react-query';
import type { ExtendedPackageInfo } from '../../types/software';

export function useAgentSoftware() {
  return useQuery({
    queryKey: ['software'],
    queryFn: async (): Promise<ExtendedPackageInfo[]> => {
      const res = await fetch('/api/v1/software');
      if (!res.ok) {
        throw new Error(`Failed to fetch software info: ${res.status} ${res.statusText}`);
      }
      return res.json() as Promise<ExtendedPackageInfo[]>;
    },
  });
}
