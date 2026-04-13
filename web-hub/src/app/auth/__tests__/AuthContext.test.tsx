import { render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { vi, describe, it, expect, beforeEach } from 'vitest';
import { AuthProvider, useAuth } from '../AuthContext';

function TestConsumer() {
  const { user } = useAuth();
  return <div data-testid="user">{user.name ?? user.email ?? user.user_id}</div>;
}

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
};

describe('AuthProvider', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('shows loading state initially', () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation(() => new Promise(() => {}));
    const Wrapper = createWrapper();
    render(
      <Wrapper>
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      </Wrapper>,
    );
    expect(screen.getByText('Loading...')).toBeInTheDocument();
  });

  it('renders children with authenticated user', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          user_id: 'user-123',
          tenant_id: 'tenant-456',
          email: 'admin@test.com',
          name: 'Admin User',
          role: 'admin',
        }),
        { status: 200, headers: { 'Content-Type': 'application/json' } },
      ),
    );
    const Wrapper = createWrapper();
    render(
      <Wrapper>
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      </Wrapper>,
    );
    await waitFor(() => {
      expect(screen.getByTestId('user')).toHaveTextContent('Admin User');
    });
  });

  it('falls back to dev user on auth failure', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(new Response('', { status: 401 }));
    const Wrapper = createWrapper();
    render(
      <Wrapper>
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      </Wrapper>,
    );
    await waitFor(() => {
      expect(screen.getByTestId('user')).toHaveTextContent('Dev User');
    });
  });
});
