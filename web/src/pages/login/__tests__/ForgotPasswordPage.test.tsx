import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router';
import { ForgotPasswordPage } from '../ForgotPasswordPage';

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>{children}</MemoryRouter>
      </QueryClientProvider>
    );
  };
}

describe('ForgotPasswordPage', () => {
  it('renders email field and submit button', () => {
    render(<ForgotPasswordPage />, { wrapper: createWrapper() });
    expect(screen.getByLabelText(/^email$/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /send reset link/i })).toBeInTheDocument();
  });

  it('renders heading and description', () => {
    render(<ForgotPasswordPage />, { wrapper: createWrapper() });
    expect(screen.getByText('Reset your password')).toBeInTheDocument();
  });

  it('renders back to sign in link', () => {
    render(<ForgotPasswordPage />, { wrapper: createWrapper() });
    expect(screen.getByText(/back to sign in/i)).toBeInTheDocument();
  });

  it('shows validation error for empty email', async () => {
    const user = userEvent.setup();
    render(<ForgotPasswordPage />, { wrapper: createWrapper() });

    await user.click(screen.getByRole('button', { name: /send reset link/i }));

    await waitFor(() => {
      expect(screen.getByText('Email is required')).toBeInTheDocument();
    });
  });

  it('shows success state after submission', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(
      new Response(JSON.stringify({ message: "If that email exists, we've sent a reset link." }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    );

    const user = userEvent.setup();
    render(<ForgotPasswordPage />, { wrapper: createWrapper() });

    await user.type(screen.getByLabelText(/^email$/i), 'user@example.com');
    await user.click(screen.getByRole('button', { name: /send reset link/i }));

    await waitFor(() => {
      expect(screen.getByText(/check your email/i)).toBeInTheDocument();
      expect(
        screen.getByText("If that email exists, we've sent a reset link."),
      ).toBeInTheDocument();
    });

    vi.restoreAllMocks();
  });
});
