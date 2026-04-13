import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  listCatalog,
  getCatalogEntry,
  getCatalogStats,
  createCatalogEntry,
  updateCatalogEntry,
  deleteCatalogEntry,
} from '../catalog';
import type { CreateCatalogRequest } from '../../types/catalog';

export function useCatalogEntries(params?: {
  limit?: number;
  offset?: number;
  os_family?: string;
  severity?: string;
  search?: string;
  feed_source_id?: string;
  date_range?: string;
  entry_type?: string;
}) {
  return useQuery({
    queryKey: ['catalog', params],
    queryFn: () => listCatalog(params ?? {}),
  });
}

export function useCatalogEntry(id: string) {
  return useQuery({
    queryKey: ['catalog', id],
    queryFn: () => getCatalogEntry(id),
    enabled: !!id,
  });
}

export function useCatalogStats() {
  return useQuery({
    queryKey: ['catalog-stats'],
    queryFn: getCatalogStats,
  });
}

export function useCreateCatalogEntry() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateCatalogRequest) => createCatalogEntry(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['catalog'] });
      void queryClient.invalidateQueries({ queryKey: ['catalog-stats'] });
    },
  });
}

export function useUpdateCatalogEntry() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: CreateCatalogRequest }) =>
      updateCatalogEntry(id, data),
    onSuccess: (_data, variables) => {
      void queryClient.invalidateQueries({ queryKey: ['catalog', variables.id] });
      void queryClient.invalidateQueries({ queryKey: ['catalog'] });
      void queryClient.invalidateQueries({ queryKey: ['catalog-stats'] });
    },
  });
}

export function useDeleteCatalogEntry() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => deleteCatalogEntry(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['catalog'] });
      void queryClient.invalidateQueries({ queryKey: ['catalog-stats'] });
    },
  });
}
