import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router';
import { LogsPage } from '../../../pages/logs/LogsPage';

// jsdom doesn't support scrollIntoView
beforeAll(() => {
  Element.prototype.scrollIntoView = vi.fn();
});

vi.mock('../../../api/hooks/useLogs', () => ({
  useLogs: () => ({
    data: {
      data: [
        {
          id: 'l1',
          level: 'info',
          message: 'Agent started successfully',
          source: 'main',
          timestamp: '2026-03-16T08:00:00Z',
        },
        {
          id: 'l2',
          level: 'error',
          message: 'gRPC stream disconnected',
          source: 'comms',
          timestamp: '2026-03-16T07:55:00Z',
        },
        {
          id: 'l3',
          level: 'warn',
          message: 'Connection latency high: 450ms',
          source: 'comms',
          timestamp: '2026-03-16T08:03:00Z',
        },
      ],
      next_cursor: null,
      total_count: 3,
    },
    isLoading: false,
    isError: false,
  }),
}));

const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

const renderPage = () =>
  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <LogsPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );

describe('LogsPage', () => {
  it('renders log viewer container', () => {
    renderPage();
    const viewer = document.querySelector('[data-testid="log-viewer"]');
    expect(viewer).toBeInTheDocument();
  });

  it('renders log messages', () => {
    renderPage();
    expect(screen.getByText('Agent started successfully')).toBeInTheDocument();
    expect(screen.getByText('gRPC stream disconnected')).toBeInTheDocument();
  });

  it('renders log viewer with data-testid', () => {
    renderPage();
    const viewer = document.querySelector('[data-testid="log-viewer"]');
    expect(viewer).toBeInTheDocument();
  });

  it('renders level filter pills', () => {
    renderPage();
    // Level pills: All, Debug, Info, Warn, Error
    expect(screen.getByText('All')).toBeInTheDocument();
    expect(screen.getByText('Info')).toBeInTheDocument();
    expect(screen.getByText('Error')).toBeInTheDocument();
  });

  it('renders auto-refresh toggle', () => {
    renderPage();
    // When auto-refresh is on, shows "Pause" button
    expect(screen.getByText('Pause')).toBeInTheDocument();
  });

  it('renders export button', () => {
    renderPage();
    expect(screen.getByText('Export')).toBeInTheDocument();
  });

  it('renders search input', () => {
    renderPage();
    expect(screen.getByPlaceholderText(/Search/i)).toBeInTheDocument();
  });

  it('renders source filter dropdown', () => {
    renderPage();
    expect(screen.getByDisplayValue('All Sources')).toBeInTheDocument();
  });
});
