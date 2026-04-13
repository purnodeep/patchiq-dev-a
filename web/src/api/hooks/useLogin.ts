import { useMutation, useQuery } from '@tanstack/react-query';

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
}

interface ForgotPasswordRequest {
  email: string;
}

interface ForgotPasswordResponse {
  message: string;
}

interface InviteInfo {
  email: string;
  tenant_name: string;
  role_name: string;
  expires_at: string;
}

interface RegisterRequest {
  code: string;
  name: string;
  password: string;
}

interface RegisterResponse {
  user_id: string;
  email: string;
  name: string;
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

export function useForgotPassword() {
  return useMutation<ForgotPasswordResponse, Error, ForgotPasswordRequest>({
    mutationFn: async (data) => {
      const res = await fetch('/api/v1/auth/forgot-password', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      });
      if (!res.ok) {
        throw new Error('Something went wrong. Please try again.');
      }
      return res.json() as Promise<ForgotPasswordResponse>;
    },
  });
}

export function useValidateInvite(code: string | null) {
  return useQuery<InviteInfo, Error>({
    queryKey: ['invite', code],
    queryFn: async () => {
      const res = await fetch(`/api/v1/auth/invite/${code}`);
      if (!res.ok) {
        const err = await res
          .json()
          .catch(() => ({ message: 'This invite link is invalid or has expired.' }));
        throw new Error(err.message || 'Invalid invite');
      }
      return res.json() as Promise<InviteInfo>;
    },
    enabled: !!code,
    retry: false,
  });
}

export function useRegister() {
  return useMutation<RegisterResponse, Error, RegisterRequest>({
    mutationFn: async (data) => {
      const res = await fetch('/api/v1/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(data),
      });
      if (!res.ok) {
        const err = await res
          .json()
          .catch(() => ({ message: 'Something went wrong. Please try again.' }));
        throw new Error(err.message || 'Registration failed');
      }
      return res.json() as Promise<RegisterResponse>;
    },
  });
}
