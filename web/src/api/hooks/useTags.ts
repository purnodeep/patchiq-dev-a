import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

export interface Tag {
  id: string;
  tenant_id: string;
  key: string;
  value: string;
  color: string | null;
  description: string | null;
  created_at: string;
  updated_at: string;
  endpoint_count?: number;
}

export interface TagDetail extends Tag {
  members?: { id: string; hostname: string; status: string }[];
}

export interface CreateTagRequest {
  key: string;
  value: string;
  color?: string;
  description?: string;
}

export interface UpdateTagRequest {
  color?: string;
  description?: string;
}

export function useTags(params?: { cursor?: string; limit?: number; search?: string }) {
  return useQuery({
    queryKey: ['tags', params],
    queryFn: async () => {
      const searchParams = new URLSearchParams();
      if (params?.cursor) searchParams.set('cursor', params.cursor);
      if (params?.limit) searchParams.set('limit', String(params.limit));
      if (params?.search) searchParams.set('search', params.search);
      const qs = searchParams.toString();
      const res = await fetch(`/api/v1/tags${qs ? `?${qs}` : ''}`, {
        credentials: 'include',
      });
      if (!res.ok) throw new Error(`Failed to fetch tags: ${res.status}`);
      return res.json() as Promise<Tag[]>;
    },
    refetchInterval: 30_000,
  });
}

export function useTag(id: string) {
  return useQuery({
    queryKey: ['tags', id],
    queryFn: async () => {
      const res = await fetch(`/api/v1/tags/${id}`, {
        credentials: 'include',
      });
      if (!res.ok) throw new Error(`Failed to fetch tag: ${res.status}`);
      return res.json() as Promise<TagDetail>;
    },
    enabled: !!id,
  });
}

export function useCreateTag() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: CreateTagRequest) => {
      const res = await fetch('/api/v1/tags', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        const err = await res.json().catch(() => ({}));
        throw new Error(
          (err as { message?: string }).message ?? `Failed to create tag: ${res.status}`,
        );
      }
      return res.json() as Promise<Tag>;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['tags'] });
      void queryClient.invalidateQueries({ queryKey: ['endpoints'] });
    },
  });
}

export function useUpdateTag() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, body }: { id: string; body: UpdateTagRequest }) => {
      const res = await fetch(`/api/v1/tags/${id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        const err = await res.json().catch(() => ({}));
        throw new Error(
          (err as { message?: string }).message ?? `Failed to update tag: ${res.status}`,
        );
      }
      return res.json() as Promise<Tag>;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['tags'] });
      void queryClient.invalidateQueries({ queryKey: ['endpoints'] });
    },
  });
}

export function useDeleteTag() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const res = await fetch(`/api/v1/tags/${id}`, {
        method: 'DELETE',
        credentials: 'include',
      });
      if (!res.ok) throw new Error(`Failed to delete tag: ${res.status}`);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['tags'] });
      void queryClient.invalidateQueries({ queryKey: ['endpoints'] });
    },
  });
}

export interface AssignTagRequest {
  endpoint_ids: string[];
}

export function useAssignTag() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ tagId, endpointIds }: { tagId: string; endpointIds: string[] }) => {
      const res = await fetch(`/api/v1/tags/${tagId}/assign`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ endpoint_ids: endpointIds } satisfies AssignTagRequest),
      });
      if (!res.ok) {
        const err = await res.json().catch(() => ({}));
        throw new Error(
          (err as { message?: string }).message ?? `Failed to assign tag: ${res.status}`,
        );
      }
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['tags'] });
      void queryClient.invalidateQueries({ queryKey: ['endpoints'] });
      void queryClient.invalidateQueries({ queryKey: ['endpoint'] });
    },
  });
}

export function useUnassignTag() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ tagId, endpointIds }: { tagId: string; endpointIds: string[] }) => {
      const res = await fetch(`/api/v1/tags/${tagId}/unassign`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ endpoint_ids: endpointIds }),
      });
      if (!res.ok) throw new Error(`Failed to unassign tag: ${res.status}`);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['tags'] });
      void queryClient.invalidateQueries({ queryKey: ['endpoints'] });
      void queryClient.invalidateQueries({ queryKey: ['endpoint'] });
    },
  });
}
