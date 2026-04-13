import { useQuery } from '@tanstack/react-query';
import type { HardwareInfo } from '../../types/hardware';

export function useAgentHardware() {
  return useQuery({
    queryKey: ['hardware'],
    queryFn: async (): Promise<HardwareInfo> => {
      const res = await fetch('/api/v1/hardware');
      if (!res.ok) {
        throw new Error(`Failed to fetch hardware info: ${res.status} ${res.statusText}`);
      }
      return res.json() as Promise<HardwareInfo>;
    },
  });
}
