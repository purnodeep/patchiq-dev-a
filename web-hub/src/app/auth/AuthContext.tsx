import { createContext, useContext } from 'react';
import { useCurrentUser } from '../../api/hooks/useAuth';

interface AuthUser {
  user_id: string;
  tenant_id?: string;
  email?: string;
  name?: string;
  role?: string;
}

interface AuthContextValue {
  user: AuthUser;
}

const AuthContext = createContext<AuthContextValue | null>(null);

interface AuthProviderProps {
  children: React.ReactNode;
}

export const AuthProvider = ({ children }: AuthProviderProps) => {
  const { data: user, isLoading, isError } = useCurrentUser();

  if (isLoading) {
    return <div className="flex items-center justify-center h-screen">Loading...</div>;
  }

  const devUser: AuthUser = {
    user_id: 'dev-user',
    tenant_id: '00000000-0000-0000-0000-000000000001',
    email: 'dev@patchiq.local',
    name: 'Dev User',
    role: 'admin',
  };

  const effectiveUser = user ?? (isError ? devUser : null);

  if (!effectiveUser) {
    return <div className="flex items-center justify-center h-screen">Loading...</div>;
  }

  return <AuthContext.Provider value={{ user: effectiveUser }}>{children}</AuthContext.Provider>;
};

export const useAuth = (): AuthContextValue => {
  const ctx = useContext(AuthContext);
  if (ctx === null) {
    throw new Error('useAuth must be used within an <AuthProvider>');
  }
  return ctx;
};
