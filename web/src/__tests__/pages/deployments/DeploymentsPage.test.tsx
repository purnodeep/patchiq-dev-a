import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { DeploymentsPage } from '../../../pages/deployments/DeploymentsPage';

vi.mock('../../../api/hooks/useDeployments', () => ({
  useDeployments: () => ({
    data: {
      data: [
        {
          id: 'd1',
          tenant_id: 't1',
          policy_id: 'p1',
          status: 'running',
          target_count: 10,
          completed_count: 6,
          success_count: 5,
          failed_count: 1,
          created_by: null,
          started_at: '2026-01-01T00:01:00Z',
          completed_at: null,
          created_at: '2026-01-01T00:00:00Z',
          updated_at: '2026-01-01T00:01:00Z',
        },
      ],
      next_cursor: null,
      total_count: 1,
      status_counts: {
        running: 1,
        completed: 0,
        failed: 0,
        pending: 0,
      },
    },
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  }),
  useCreateDeployment: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useCancelDeployment: () => ({ mutate: vi.fn(), isPending: false }),
  useRetryDeployment: () => ({ mutate: vi.fn(), isPending: false }),
  useRollbackDeployment: () => ({ mutate: vi.fn(), isPending: false }),
}));

vi.mock('../../../api/hooks/useEndpoints', () => ({
  useEndpoints: () => ({
    data: { data: [], next_cursor: null, total_count: 0 },
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  }),
}));

vi.mock('../../../api/hooks/usePatches', () => ({
  usePatches: () => ({ data: { data: [], total_count: 0 }, isLoading: false }),
  usePatchSeverityCounts: vi.fn(() => ({})),
}));

vi.mock('../../../api/hooks/usePolicies', () => ({
  usePolicies: () => ({
    data: {
      data: [
        {
          id: 'p1',
          name: 'My Policy',
          selection_mode: 'all_available',
          target_selector: { eq: { key: 'env', value: 'prod' } },
          enabled: true,
        },
      ],
    },
    isLoading: false,
  }),
}));

const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

const renderPage = () =>
  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DeploymentsPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );

describe('DeploymentsPage', () => {
  it('renders page title', () => {
    renderPage();
    // DeploymentsPage has no heading — it opens with stat cards; verify the page rendered
    expect(screen.getByPlaceholderText('Search deployments...')).toBeInTheDocument();
  });

  it('renders deployment status', () => {
    renderPage();
    // Status "Running" appears in the table row and in the filter pills
    expect(screen.getAllByText('Running').length).toBeGreaterThanOrEqual(1);
  });

  it('renders create deployment button', () => {
    renderPage();
    // Button text is "New Deployment" not "Create Deployment"
    expect(screen.getByRole('button', { name: /new deployment/i })).toBeInTheDocument();
  });

  it('renders filter pills', () => {
    renderPage();
    expect(screen.getByText('All')).toBeInTheDocument();
    expect(screen.getAllByText('Running').length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText('Failed').length).toBeGreaterThanOrEqual(1);
  });

  it('renders policy name from lookup', () => {
    renderPage();
    expect(screen.getAllByText('My Policy').length).toBeGreaterThanOrEqual(1);
  });
});
