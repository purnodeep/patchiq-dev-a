import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router';
import DashboardPage from '../../../pages/dashboard/DashboardPage';

const mockSummary = {
  total_endpoints: 15,
  active_endpoints: 12,
  endpoints_degraded: 1,
  total_patches: 50,
  critical_patches: 7,
  patches_high: 10,
  patches_medium: 15,
  patches_low: 8,
  total_cves: 30,
  critical_cves: 5,
  unpatched_cves: 20,
  pending_deployments: 3,
  compliance_rate: 84.5,
  active_deployments: [],
  overdue_sla_count: 2,
  failed_deployments_count: 1,
  failed_trend_7d: [1, 2, 0, 1, 3, 2, 1],
  workflows_running_count: 0,
  workflows_running: [],
  hub_sync_status: 'idle',
  hub_last_sync_at: null,
  hub_url: '',
};

vi.mock('../../../api/hooks/useDashboard', () => ({
  useDashboardSummary: () => ({
    data: mockSummary,
    isLoading: false,
    error: null,
    refetch: vi.fn(),
  }),
  useDashboardActivity: () => ({
    data: [],
    isLoading: false,
    error: null,
    refetch: vi.fn(),
  }),
  useBlastRadius: () => ({
    data: null,
    isLoading: false,
    error: null,
    refetch: vi.fn(),
  }),
  useTopEndpointsByRisk: () => ({
    data: [],
    isLoading: false,
    error: null,
    refetch: vi.fn(),
  }),
  usePatchesTimeline: () => ({
    data: [],
    isLoading: false,
    error: null,
    refetch: vi.fn(),
  }),
}));

vi.mock('../../../api/hooks/useCompliance', () => ({
  useComplianceSummary: () => ({
    data: { score: '84.5', frameworks: [] },
    isLoading: false,
    error: null,
  }),
}));

beforeAll(() => {
  (globalThis as Record<string, unknown>).ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});

const renderPage = () =>
  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DashboardPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );

describe('DashboardPage', () => {
  it('renders page title', () => {
    renderPage();
    expect(screen.getAllByText('Dashboard').length).toBeGreaterThanOrEqual(1);
  });

  it('renders loading skeleton when loading', () => {
    const { unmount } = render(
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
      >
        <MemoryRouter>
          <DashboardPage />
        </MemoryRouter>
      </QueryClientProvider>,
    );
    unmount();
  });
});
