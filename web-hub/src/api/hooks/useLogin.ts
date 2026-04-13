import { useMutation } from '@tanstack/react-query';

interface LoginRequest {
  email: string;
  password: string;
  remember_me: boolean;
}

interface LoginResponse {
  user_id: string;
  tenant_id: string;
  name: string;
  email: string;
  role: string;
}

export function useLogin() {
  return useMutation<LoginResponse, Error, LoginRequest>({
    mutationFn: async (data) => {
      const res = await fetch('/api/v1/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(data),
      });
      if (!res.ok) {
        const err = await res
          .json()
          .catch(() => ({ message: 'Something went wrong. Please try again.' }));
        throw new Error(err.message || 'Login failed');
      }
      return res.json() as Promise<LoginResponse>;
    },
  });
}
