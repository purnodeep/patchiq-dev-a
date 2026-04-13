import { useQuery } from '@tanstack/react-query';
import { api } from '../client';
import type { PatchListResponse, PatchDetail } from '../../types/patches';

export interface PatchesParams {
  cursor?: string;
  limit?: number;
  severity?: string;
  os_family?: string;
  status?: string;
  search?: string;
  sort_by?: string;
  sort_dir?: string;
}

export function usePatches(params?: PatchesParams) {
  return useQuery({
    queryKey: ['patches', params],
    queryFn: async () => {
      const query: Record<string, string | number | undefined> = {};
      if (params?.cursor) query.cursor = params.cursor;
      if (params?.limit) query.limit = params.limit;
      if (params?.severity) query.severity = params.severity;
      if (params?.os_family) query.os_family = params.os_family;
      if (params?.status) query.status = params.status;
      if (params?.search) query.search = params.search;
      if (params?.sort_by) query.sort_by = params.sort_by;
      if (params?.sort_dir) query.sort_dir = params.sort_dir;

      const { data, error } = await api.GET('/api/v1/patches', {
        params: { query },
      });
      if (error) throw error;
      return data as unknown as PatchListResponse;
    },
    staleTime: 60_000,
    refetchInterval: 60_000,
  });
}

export function usePatchSeverityCounts(params?: {
  os_family?: string;
  status?: string;
  search?: string;
}) {
  return useQuery({
    queryKey: ['patches', 'severity-counts', params],
    queryFn: async () => {
      const res = await fetch(
        `/api/v1/patches/severity-counts?` +
          new URLSearchParams(
            Object.fromEntries(Object.entries(params ?? {}).filter(([, v]) => v)),
          ),
        { credentials: 'include' },
      );
      if (!res.ok) throw new Error('failed to fetch severity counts');
      return res.json() as Promise<Record<string, number>>;
    },
    refetchInterval: 30_000,
  });
}

export function usePatch(id: string) {
  return useQuery({
    queryKey: ['patches', id],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/patches/{id}', {
        params: { path: { id } },
      });
      if (error) throw error;
      return data as unknown as PatchDetail;
    },
    enabled: !!id,
  });
}
