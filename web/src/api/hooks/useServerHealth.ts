import { useQuery } from '@tanstack/react-query';

export interface ServerHealth {
  status: string;
  version: string;
  uptime: string;
}

export function useServerHealth() {
  return useQuery({
    queryKey: ['server', 'health'],
    queryFn: async (): Promise<ServerHealth> => {
      const res = await fetch('/health');
      if (!res.ok) throw new Error(`health check failed: ${res.status}`);
      return res.json() as Promise<ServerHealth>;
    },
    staleTime: 30_000,
    refetchInterval: 60_000,
  });
}
