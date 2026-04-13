import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../client';
import type { components } from '../types';

// --- Type Aliases (re-exported for consumers) ---

export type Role = components['schemas']['Role'];
export type RolePermission = components['schemas']['RolePermission'];
export type Permission = components['schemas']['PermissionInput'];
export type RoleRequest = components['schemas']['RoleRequest'];

// Inferred from the listRoleUsers response schema
export type RoleUser = { user_id: string; assigned_at: string };

// --- Query Hooks ---

export function useRoles(params?: { cursor?: string; limit?: number; search?: string }) {
  return useQuery({
    queryKey: ['roles', params],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/roles', {
        params: { query: params },
      });
      if (error) throw error;
      return data;
    },
    refetchInterval: 30_000,
  });
}

export function useRole(id: string) {
  return useQuery({
    queryKey: ['roles', id],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/roles/{id}', {
        params: { path: { id } },
      });
      if (error) throw error;
      return data;
    },
    enabled: !!id,
  });
}

export function useRolePermissions(id: string) {
  return useQuery({
    queryKey: ['roles', id, 'permissions'],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/roles/{id}/permissions', {
        params: { path: { id } },
      });
      if (error) throw error;
      return data;
    },
    enabled: !!id,
  });
}

export function useRoleUsers(roleId: string) {
  return useQuery({
    queryKey: ['roles', roleId, 'users'],
    queryFn: async () => {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any -- TODO(#332): remove cast after OpenAPI spec regeneration
      const { data, error } = await (api as any).GET('/api/v1/roles/{id}/users', {
        params: { path: { id: roleId } },
      });
      if (error) throw error;
      return data as { users: RoleUser[] };
    },
    enabled: !!roleId,
  });
}

export function useUserRoles(userId: string) {
  return useQuery({
    queryKey: ['user-roles', userId],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/users/{id}/roles', {
        params: { path: { id: userId } },
      });
      if (error) throw error;
      return data;
    },
    enabled: !!userId,
  });
}

// --- Mutation Hooks ---

export function useCreateRole() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: RoleRequest) => {
      const { data, error } = await api.POST('/api/v1/roles', {
        body,
      });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['roles'] });
    },
  });
}

export function useUpdateRole() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, body }: { id: string; body: RoleRequest }) => {
      const { data, error } = await api.PUT('/api/v1/roles/{id}', {
        params: { path: { id } },
        body,
      });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['roles'] });
    },
  });
}

export function useDeleteRole() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const { data, error } = await api.DELETE('/api/v1/roles/{id}', {
        params: { path: { id } },
      });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['roles'] });
    },
  });
}

export function useUpdateRolePermissions() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, permissions }: { id: string; permissions: Permission[] }) => {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any -- OpenAPI spec missing PUT method for this endpoint
      const { data, error } = await (api as any).PUT('/api/v1/roles/{id}/permissions', {
        params: { path: { id } },
        body: { permissions },
      });
      if (error) throw error;
      return data;
    },
    onSuccess: (_data: unknown, variables: { id: string; permissions: Permission[] }) => {
      void queryClient.invalidateQueries({ queryKey: ['roles'] });
      void queryClient.invalidateQueries({ queryKey: ['roles', variables.id, 'permissions'] });
    },
  });
}

export function useAssignUserRole() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ userId, roleId }: { userId: string; roleId: string }) => {
      const { data, error } = await api.POST('/api/v1/users/{id}/roles', {
        params: { path: { id: userId } },
        body: { role_id: roleId },
      });
      if (error) throw error;
      return data;
    },
    onSuccess: (_data, variables) => {
      void queryClient.invalidateQueries({ queryKey: ['user-roles', variables.userId] });
    },
  });
}

export function useRevokeUserRole() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ userId, roleId }: { userId: string; roleId: string }) => {
      const { data, error } = await api.DELETE('/api/v1/users/{id}/roles/{roleId}', {
        params: { path: { id: userId, roleId } },
      });
      if (error) throw error;
      return data;
    },
    onSuccess: (_data, variables) => {
      void queryClient.invalidateQueries({ queryKey: ['user-roles', variables.userId] });
    },
  });
}
