import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

interface AuthUser {
  user_id: string;
  tenant_id: string;
  email?: string;
  name?: string;
  role?: string;
}

export function useCurrentUser() {
  return useQuery({
    queryKey: ['auth', 'me'],
    queryFn: async (): Promise<AuthUser> => {
      const res = await fetch('/api/v1/auth/me', {
        credentials: 'include',
      });
      if (!res.ok) throw new Error(`auth/me failed: ${res.status}`);
      return res.json() as Promise<AuthUser>;
    },
    retry: false,
    staleTime: 5 * 60 * 1000,
  });
}

export function useLogout() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      const res = await fetch('/api/v1/auth/logout', {
        method: 'POST',
        credentials: 'include',
      });
      if (!res.ok) {
        throw new Error(`Logout failed (status ${res.status})`);
      }
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['auth'] });
      window.location.href = '/login';
    },
  });
}
