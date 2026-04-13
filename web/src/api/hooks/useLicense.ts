import { useQuery } from '@tanstack/react-query';
import { api } from '../client';
import type { components } from '../types';

export type LicenseStatus = components['schemas']['LicenseStatus'];
export type EndpointUsage = components['schemas']['EndpointUsage'];

export function useLicenseStatus() {
  return useQuery({
    queryKey: ['license', 'status'],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/license/status', {});
      if (error) throw error;
      return data;
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
  });
}
