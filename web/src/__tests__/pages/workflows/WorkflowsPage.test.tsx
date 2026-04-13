import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { describe, it, expect, vi } from 'vitest';
import { WorkflowsPage } from '../../../pages/workflows/index';

vi.mock('../../../flows/policy-workflow/hooks/use-workflows', () => ({
  useWorkflows: () => ({
    data: {
      data: [
        {
          id: 'w1',
          tenant_id: 't1',
          name: 'My Workflow',
          description: 'Test',
          created_at: '2026-01-01T00:00:00Z',
          updated_at: '2026-01-01T00:00:00Z',
        },
      ],
      next_cursor: null,
      total_count: 1,
    },
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  }),
}));

const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

describe('WorkflowsPage', () => {
  it('renders page title', () => {
    render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>
          <WorkflowsPage />
        </MemoryRouter>
      </QueryClientProvider>,
    );
    // Page header: "New Workflow" CTA is the stable title indicator
    expect(screen.getByRole('link', { name: /new workflow/i })).toBeInTheDocument();
  });

  it('renders workflow name', () => {
    render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>
          <WorkflowsPage />
        </MemoryRouter>
      </QueryClientProvider>,
    );
    expect(screen.getByText('My Workflow')).toBeInTheDocument();
  });

  it('renders create button', () => {
    render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>
          <WorkflowsPage />
        </MemoryRouter>
      </QueryClientProvider>,
    );
    expect(screen.getByRole('link', { name: /new workflow/i })).toBeInTheDocument();
  });
});
