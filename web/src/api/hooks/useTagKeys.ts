import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

export interface TagKey {
  key: string;
  description: string | null;
  exclusive: boolean;
  value_type: 'string' | 'enum';
  allowed_values: string[] | null;
  created_at: string;
  updated_at: string;
}

export interface UpsertTagKeyRequest {
  key: string;
  description?: string;
  exclusive?: boolean;
  value_type?: 'string' | 'enum';
  allowed_values?: string[];
}

export function useTagKeys() {
  return useQuery({
    queryKey: ['tag-keys'],
    queryFn: async () => {
      const res = await fetch('/api/v1/tag-keys', { credentials: 'include' });
      if (!res.ok) throw new Error(`Failed to fetch tag keys: ${res.status}`);
      return res.json() as Promise<TagKey[]>;
    },
    refetchInterval: 60_000,
  });
}

/**
 * Lightweight: returns just the distinct keys observed on existing tags.
 * Used by the selector builder to populate a key typeahead even for keys
 * that were never explicitly registered in tag_keys.
 */
export function useDistinctTagKeys() {
  return useQuery({
    queryKey: ['tag-keys', 'distinct'],
    queryFn: async () => {
      const res = await fetch('/api/v1/tags/keys', { credentials: 'include' });
      if (!res.ok) throw new Error(`Failed to fetch distinct tag keys: ${res.status}`);
      const body = (await res.json()) as { key: string; value_count: number }[];
      return body.map((entry) => entry.key);
    },
    refetchInterval: 60_000,
  });
}

export function useUpsertTagKey() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: UpsertTagKeyRequest) => {
      const res = await fetch('/api/v1/tag-keys', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        const err = await res.json().catch(() => ({}));
        throw new Error(
          (err as { message?: string }).message ?? `Failed to upsert tag key: ${res.status}`,
        );
      }
      return res.json() as Promise<TagKey>;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['tag-keys'] });
    },
  });
}

export function useDeleteTagKey() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (key: string) => {
      const res = await fetch(`/api/v1/tag-keys/${encodeURIComponent(key)}`, {
        method: 'DELETE',
        credentials: 'include',
      });
      if (!res.ok && res.status !== 204) {
        const err = await res.json().catch(() => ({}));
        throw new Error(
          (err as { message?: string }).message ?? `Failed to delete tag key: ${res.status}`,
        );
      }
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['tag-keys'] });
    },
  });
}
