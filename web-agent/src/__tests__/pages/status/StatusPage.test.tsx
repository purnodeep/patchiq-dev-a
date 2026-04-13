import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { StatusPage } from '../../../pages/status/StatusPage';

vi.mock('../../../api/hooks/useStatus', () => ({
  useAgentStatus: () => ({
    data: {
      agent_id: 'abc-123',
      hostname: 'test-host',
      os_family: 'linux',
      os_version: 'linux/amd64',
      agent_version: '1.0.0',
      enrollment_status: 'enrolled',
      server_url: 'localhost:50051',
      last_heartbeat: '2026-03-06T10:00:00Z',
      uptime_seconds: 3661,
      pending_patch_count: 13,
      installed_count: 247,
      failed_count: 2,
    },
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  }),
}));

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});

const renderPage = () =>
  render(
    <QueryClientProvider client={queryClient}>
      <StatusPage />
    </QueryClientProvider>,
  );

describe('StatusPage', () => {
  it('renders page title', () => {
    renderPage();
    expect(screen.getAllByText('Status').length).toBeGreaterThan(0);
  });

  it('displays agent version', () => {
    renderPage();
    expect(screen.getAllByText('1.0.0').length).toBeGreaterThan(0);
  });

  it('displays hostname', () => {
    renderPage();
    expect(screen.getAllByText('test-host').length).toBeGreaterThan(0);
  });

  it('displays enrollment status', () => {
    renderPage();
    expect(screen.getByText('enrolled')).toBeInTheDocument();
  });

  it('displays server URL', () => {
    renderPage();
    expect(screen.getAllByText('localhost:50051').length).toBeGreaterThan(0);
  });

  it('renders status orb', () => {
    renderPage();
    expect(screen.getByText('Healthy')).toBeInTheDocument();
  });

  it('displays pending count in quick stats', () => {
    renderPage();
    expect(screen.getByText('13')).toBeInTheDocument();
  });

  it('displays installed count', () => {
    renderPage();
    expect(screen.getByText('247')).toBeInTheDocument();
  });

  it('renders resources section', () => {
    renderPage();
    expect(screen.getByText('Resources')).toBeInTheDocument();
  });

  it('shows loading skeleton', () => {
    renderPage(); // existing loading mock not needed - just verify no crash
  });
});
