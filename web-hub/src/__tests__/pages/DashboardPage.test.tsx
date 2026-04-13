import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

vi.mock('../../api/hooks/useDashboard', () => ({
  useDashboardStats: () => ({
    data: {
      total_catalog_entries: 4500,
      active_feeds: 5,
      connected_clients: 12,
      pending_clients: 3,
      active_licenses: 8,
    },
    isLoading: false,
    isError: false,
    error: null,
  }),
  useLicenseBreakdown: () => ({
    data: [],
    isLoading: false,
    isError: false,
    error: null,
  }),
  useCatalogGrowth: () => ({
    data: [],
    isLoading: false,
    isError: false,
    error: null,
  }),
  useClientSummary: () => ({
    data: [],
    isLoading: false,
    isError: false,
    error: null,
  }),
  useRecentActivity: () => ({
    data: [],
    isLoading: false,
    isError: false,
    error: null,
  }),
}));

vi.mock('../../api/hooks/useFeeds', () => ({
  useFeeds: () => ({
    data: [],
    isLoading: false,
    isError: false,
    error: null,
  }),
}));

vi.mock('recharts', () => ({
  ResponsiveContainer: ({ children }: { children: React.ReactNode }) => children,
  AreaChart: () => null,
  Area: () => null,
  XAxis: () => null,
  YAxis: () => null,
  CartesianGrid: () => null,
  Tooltip: () => null,
}));

import { DashboardPage } from '../../pages/dashboard/DashboardPage';

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
  it('renders "Dashboard" title', () => {
    renderPage();
    expect(screen.getByText('Dashboard')).toBeInTheDocument();
  });

  it('renders Fleet Topology section', () => {
    renderPage();
    expect(screen.getByText('Fleet Topology')).toBeInTheDocument();
  });

  it('renders Feed Pipeline section', () => {
    renderPage();
    expect(screen.getByText('Feed Pipeline')).toBeInTheDocument();
  });

  it('renders Recent Activity section', () => {
    renderPage();
    expect(screen.getByText('Recent Activity')).toBeInTheDocument();
  });

  it('renders License Distribution section', () => {
    renderPage();
    expect(screen.getByText('No license data available')).toBeInTheDocument();
  });
});
