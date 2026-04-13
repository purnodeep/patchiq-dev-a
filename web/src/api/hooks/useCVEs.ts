import { useQuery } from '@tanstack/react-query';
import { api } from '../client';
import type { CVEListResponse, CVEDetail, CVESummary } from '../../types/cves';

export function useCVEs(params?: {
  cursor?: string;
  limit?: number;
  search?: string;
  severity?: string;
  cisa_kev?: string;
  exploit_available?: string;
  attack_vector?: string;
  published_after?: string;
  has_patch?: string;
}) {
  return useQuery({
    queryKey: ['cves', params],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/cves', {
        params: { query: params },
      });
      if (error) throw error;
      // Cast needed: openapi-fetch returns union type for response envelope; our domain types are narrower
      return data as unknown as CVEListResponse;
    },
    refetchInterval: 30_000,
  });
}

export function useCVE(id: string | undefined) {
  return useQuery({
    queryKey: ['cves', id],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/cves/{id}', {
        params: { path: { id: id! } },
      });
      if (error) throw error;
      // Cast needed: openapi-fetch returns union type for response envelope; our domain types are narrower
      return data as unknown as CVEDetail;
    },
    enabled: !!id,
  });
}

export function useCVESummary() {
  return useQuery({
    queryKey: ['cves', 'summary'],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/cves/summary');
      if (error) throw error;
      // Cast needed: openapi-fetch returns union type for response envelope; our domain types are narrower
      return data as unknown as CVESummary;
    },
    refetchInterval: 30_000,
  });
}
