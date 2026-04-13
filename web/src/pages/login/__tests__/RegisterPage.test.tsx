import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router';
import { RegisterPage } from '../RegisterPage';

function createWrapper(initialEntries: string[] = ['/register?code=test-code']) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={initialEntries}>{children}</MemoryRouter>
      </QueryClientProvider>
    );
  };
}

const mockInviteResponse = {
  email: 'invited@acme.com',
  tenant_name: 'Acme Corp',
  role_name: 'Operator',
  expires_at: '2026-04-01T00:00:00Z',
};

describe('RegisterPage', () => {
  beforeEach(() => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify(mockInviteResponse), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    );
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('shows error state when no invite code is present', () => {
    render(<RegisterPage />, { wrapper: createWrapper(['/register']) });
    expect(screen.getByText(/invalid invite link/i)).toBeInTheDocument();
  });

  it('renders form fields after invite validation', async () => {
    render(<RegisterPage />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText('Create your account')).toBeInTheDocument();
    });

    expect(screen.getByLabelText(/full name/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/^email$/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/^password$/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/^confirm password$/i)).toBeInTheDocument();
  });

  it('shows tenant name and role from invite', async () => {
    render(<RegisterPage />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText('Acme Corp')).toBeInTheDocument();
      expect(screen.getByText('Operator')).toBeInTheDocument();
    });
  });

  it('prefills email from invite as disabled input', async () => {
    render(<RegisterPage />, { wrapper: createWrapper() });

    await waitFor(() => {
      const emailInput = screen.getByLabelText(/^email$/i);
      expect(emailInput).toHaveValue('invited@acme.com');
      expect(emailInput).toBeDisabled();
    });
  });

  it('shows validation error for empty name', async () => {
    const user = userEvent.setup();
    render(<RegisterPage />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText('Create your account')).toBeInTheDocument();
    });

    await user.type(screen.getByLabelText(/^password$/i), 'Password123');
    await user.type(screen.getByLabelText(/^confirm password$/i), 'Password123');
    await user.click(screen.getByRole('button', { name: /create account/i }));

    await waitFor(() => {
      expect(screen.getByText('Full name is required')).toBeInTheDocument();
    });
  });

  it('shows validation error for short password', async () => {
    const user = userEvent.setup();
    render(<RegisterPage />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText('Create your account')).toBeInTheDocument();
    });

    await user.type(screen.getByLabelText(/full name/i), 'Jane Doe');
    await user.type(screen.getByLabelText(/^password$/i), 'short');
    await user.type(screen.getByLabelText(/^confirm password$/i), 'short');
    await user.click(screen.getByRole('button', { name: /create account/i }));

    await waitFor(() => {
      expect(screen.getByText('Password must be at least 8 characters')).toBeInTheDocument();
    });
  });

  it('shows validation error when passwords do not match', async () => {
    const user = userEvent.setup();
    render(<RegisterPage />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText('Create your account')).toBeInTheDocument();
    });

    await user.type(screen.getByLabelText(/full name/i), 'Jane Doe');
    await user.type(screen.getByLabelText(/^password$/i), 'Password123');
    await user.type(screen.getByLabelText(/^confirm password$/i), 'Password456');
    await user.click(screen.getByRole('button', { name: /create account/i }));

    await waitFor(() => {
      expect(screen.getByText("Passwords don't match")).toBeInTheDocument();
    });
  });

  it('shows error state for invalid invite', async () => {
    vi.restoreAllMocks();
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({ message: 'This invite link is invalid or has expired.' }), {
        status: 404,
        headers: { 'Content-Type': 'application/json' },
      }),
    );

    render(<RegisterPage />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText(/invalid invite/i)).toBeInTheDocument();
      expect(screen.getByText('This invite link is invalid or has expired.')).toBeInTheDocument();
    });
  });

  it('renders sign in link', async () => {
    render(<RegisterPage />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText('Create your account')).toBeInTheDocument();
    });

    expect(screen.getByText('Sign in')).toBeInTheDocument();
  });

  it('toggles password visibility', async () => {
    const user = userEvent.setup();
    render(<RegisterPage />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText('Create your account')).toBeInTheDocument();
    });

    const passwordInput = screen.getByLabelText(/^password$/i);
    expect(passwordInput).toHaveAttribute('type', 'password');

    const toggleButton = screen.getByRole('button', { name: /^show password$/i });
    await user.click(toggleButton);
    expect(passwordInput).toHaveAttribute('type', 'text');
  });
});
