import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

vi.mock('../../api/hooks/useFeeds', () => ({
  useFeeds: () => ({
    data: [
      {
        id: '1',
        name: 'nvd',
        display_name: 'National Vulnerability Database',
        enabled: true,
        sync_interval_seconds: 21600,
        last_sync_at: null,
        next_sync_at: null,
        status: 'idle',
        error_count: 0,
        last_error: null,
        entries_ingested: 4523,
        url: 'https://nvd.nist.gov/feeds/json/cve/1.1',
        auth_type: 'none',
        recent_history: [],
        new_this_week: 0,
        error_rate: 0,
        severity_filter: ['critical', 'high', 'medium', 'low'],
        os_filter: ['windows', 'ubuntu', 'rhel', 'debian'],
        severity_mapping: {},
      },
    ],
    isLoading: false,
    isError: false,
    error: null,
  }),
  useUpdateFeed: () => ({ mutate: vi.fn(), isPending: false }),
  useTriggerFeedSync: () => ({ mutate: vi.fn(), isPending: false }),
}));

import { FeedsPage } from '../../pages/feeds/FeedsPage';

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});

const renderPage = () =>
  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <FeedsPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );

describe('FeedsPage', () => {
  it('renders page via "Total Feeds" stat tile', () => {
    renderPage();
    // Redesigned page has no h1 — use stat tile button as the page identifier.
    expect(screen.getByRole('button', { name: /total feeds/i })).toBeInTheDocument();
  });

  it('renders feed display name', () => {
    renderPage();
    expect(screen.getByText('National Vulnerability Database')).toBeInTheDocument();
  });

  it('renders entry count', () => {
    renderPage();
    // toLocaleString() formats 4523 — check for either formatted or raw
    expect(screen.getAllByText(/4[,.]?523/).length).toBeGreaterThan(0);
  });

  it('renders toggle for enabled feed', () => {
    renderPage();
    // New UI: button text is "Disable" for enabled feeds (inline in actions column)
    // Multiple buttons may match /disable/i (stat card "Disabled" + action button "Disable")
    const buttons = screen.getAllByRole('button', { name: /disable/i });
    expect(buttons.length).toBeGreaterThanOrEqual(1);
  });

  it('renders "Sync" button for feed', () => {
    renderPage();
    // New UI: button text is "Sync" (not "Sync Now") in the actions column
    expect(screen.getByRole('button', { name: /^sync$/i })).toBeInTheDocument();
  });
});
