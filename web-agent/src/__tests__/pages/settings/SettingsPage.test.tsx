import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router';
import { SettingsPage } from '../../../pages/settings/SettingsPage';

vi.mock('../../../api/hooks/useSettings', () => ({
  useSettings: () => ({
    data: {
      agent_id: 'abc-123',
      agent_version: '1.0.0',
      config_file: '/etc/patchiq/agent.yaml',
      data_dir: '/var/lib/patchiq',
      log_file: '/var/log/patchiq/agent.log',
      db_path: '/var/lib/patchiq/agent.db',
      server_url: 'grpc.example.com:50051',
      http_addr: '127.0.0.1:8090',
      scan_interval: '6h',
      scan_timeout: '300s',
      auto_deploy: false,
      log_level: 'info',
      heartbeat_interval: '30s',
      bandwidth_limit_kbps: 0,
      proxy_url: '',
      offline_mode: false,
      max_concurrent_installs: 1,
      auto_reboot_window: '',
      log_retention_days: 30,
    },
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  }),
  useUpdateSettings: () => ({
    mutate: vi.fn(),
    isPending: false,
    reset: vi.fn(),
  }),
  useTriggerScan: () => ({
    mutate: vi.fn(),
    isPending: false,
  }),
}));

vi.mock('../../../api/hooks/useStatus', () => ({
  useAgentStatus: () => ({
    data: {
      agent_id: 'abc-123',
      hostname: 'test-host',
      os_family: 'linux',
      os_version: 'linux/amd64',
      agent_version: '1.0.0',
      enrollment_status: 'enrolled',
      server_url: 'grpc.example.com:50051',
      last_heartbeat: '2026-03-16T08:00:00Z',
      uptime_seconds: 3661,
      pending_patch_count: 0,
      installed_count: 10,
      failed_count: 0,
    },
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  }),
}));

const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

const renderPage = () =>
  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <SettingsPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );

describe('SettingsPage', () => {
  it('renders Communication section', () => {
    renderPage();
    expect(screen.getByText('Communication')).toBeInTheDocument();
  });

  it('renders Patch Management section', () => {
    renderPage();
    expect(screen.getByText('Patch Management')).toBeInTheDocument();
  });

  it('renders config file path', () => {
    renderPage();
    expect(screen.getByText('/etc/patchiq/agent.yaml')).toBeInTheDocument();
  });

  it('renders server URL', () => {
    renderPage();
    expect(screen.getByText('grpc.example.com:50051')).toBeInTheDocument();
  });

  it('renders Storage & Logging section', () => {
    renderPage();
    expect(screen.getByText('Storage & Logging')).toBeInTheDocument();
  });

  it('renders scan interval setting', () => {
    renderPage();
    expect(screen.getByText('Scan Interval')).toBeInTheDocument();
  });

  it('renders Agent Information section', () => {
    renderPage();
    expect(screen.getByText('Agent Information')).toBeInTheDocument();
  });

  it('renders Trigger Scan Now action', () => {
    renderPage();
    expect(screen.getByText('Trigger Scan Now')).toBeInTheDocument();
  });
});
