import { render, screen, fireEvent } from '@testing-library/react';
import { QueryClientProvider, QueryClient } from '@tanstack/react-query';
import { HistoryTab } from '../../../pages/notifications/HistoryTab';
import { vi } from 'vitest';

vi.mock('../../../api/hooks/useNotifications', () => ({
  useNotificationHistory: () => ({
    data: {
      pages: [
        {
          data: [
            {
              id: 'h-1',
              trigger_type: 'deployment.failed',
              category: 'deployments',
              channel_type: 'email',
              recipient: 'ops@example.com',
              subject: '[PatchIQ] Deployment failed',
              status: 'failed',
              retry_count: 0,
              created_at: new Date().toISOString(),
            },
            {
              id: 'h-2',
              trigger_type: 'cve.critical_discovered',
              category: 'security',
              channel_type: 'slack',
              recipient: '#security',
              subject: '[PatchIQ] Critical CVE',
              status: 'delivered',
              retry_count: 0,
              created_at: new Date().toISOString(),
            },
          ],
          next_cursor: null,
        },
      ],
    },
    isLoading: false,
    fetchNextPage: vi.fn(),
    hasNextPage: false,
  }),
  useRetryNotification: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

const wrapper = ({ children }: { children: React.ReactNode }) => (
  <QueryClientProvider client={new QueryClient()}>{children}</QueryClientProvider>
);

test('renders filter bar with channel and status dropdowns', () => {
  render(<HistoryTab />, { wrapper });
  expect(screen.getByText('All Channels')).toBeInTheDocument();
  expect(screen.getByText('All Statuses')).toBeInTheDocument();
});

test('renders history table with entries', () => {
  render(<HistoryTab />, { wrapper });
  expect(screen.getByText('ops@example.com')).toBeInTheDocument();
  expect(screen.getByText('#security')).toBeInTheDocument();
});

test('shows Retry button only on failed entries', () => {
  render(<HistoryTab />, { wrapper });
  const retryButtons = screen.getAllByText('Retry');
  expect(retryButtons).toHaveLength(1);
});

test('expands row on click', () => {
  render(<HistoryTab />, { wrapper });
  const row = screen.getByText('ops@example.com').closest('tr')!;
  fireEvent.click(row);
  expect(screen.getByText(/payload/i)).toBeInTheDocument();
});
