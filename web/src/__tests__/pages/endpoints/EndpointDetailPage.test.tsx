import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { EndpointDetailPage } from '../../../pages/endpoints/EndpointDetailPage';

vi.mock('../../../api/hooks/useCompliance', () => ({
  useEndpointCompliance: () => ({
    data: [],
    isLoading: false,
  }),
}));

vi.mock('../../../api/hooks/useCommand', () => ({
  useCommand: () => ({ data: undefined }),
  isTerminalCommandStatus: (status: string) =>
    status === 'succeeded' || status === 'failed' || status === 'cancelled',
}));

vi.mock('../../../api/hooks/useEndpoints', () => ({
  useEndpointCVEs: () => ({
    data: { data: [], total_count: 0 },
    isLoading: false,
  }),
  useScanCVEs: () => ({
    mutate: vi.fn(),
    isPending: false,
  }),
  useEndpoint: () => ({
    data: {
      id: '1',
      tenant_id: 't1',
      hostname: 'web-prod-01',
      os_family: 'ubuntu',
      os_version: '22.04',
      agent_version: '1.2.0',
      status: 'active',
      last_seen: '2026-03-06T10:00:00Z',
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-03-06T10:00:00Z',
      package_count: 142,
      last_scan: '2026-03-06T09:00:00Z',
      vulnerable_cve_count: 3,
      disk_total_gb: 500,
      disk_used_gb: 200,
      uptime_seconds: 3888000,
      cpu_cores: 8,
      memory_total_mb: 32768,
      memory_used_mb: 16384,
    },
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  }),
  useEndpointPatches: () => ({
    data: {
      data: [
        {
          id: 'p1',
          name: 'openssl-patch',
          version: '3.0.2-0ubuntu1.15',
          severity: 'critical',
          os_family: 'ubuntu',
          status: 'available',
          cve_count: 2,
          created_at: '2026-03-01T00:00:00Z',
        },
      ],
      total_count: 1,
    },
    isLoading: false,
  }),
  useTriggerScan: () => ({
    mutate: vi.fn(),
    isPending: false,
  }),
  useEndpointDeployments: () => ({
    data: {
      data: [
        {
          id: 't1',
          deployment_id: 'd1d1d1d1-0000-0000-0000-000000000001',
          endpoint_id: '1',
          patch_id: 'p1',
          status: 'success',
          started_at: '2026-03-05T08:00:00Z',
          completed_at: '2026-03-05T08:30:00Z',
          duration_seconds: 1800,
          error_message: null,
        },
      ],
      total_count: 1,
    },
    isLoading: false,
  }),
  useEndpointPackages: () => ({
    data: { data: [], total_count: 0 },
    isLoading: false,
  }),
  useDecommissionEndpoint: () => ({
    mutate: vi.fn(),
    isPending: false,
  }),
  useDeployCritical: () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }),
  useEndpoints: () => ({
    data: { data: [], total_count: 0, next_cursor: null },
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  }),
  useCreateRegistration: () => ({ mutate: vi.fn(), isPending: false, reset: vi.fn() }),
}));

vi.mock('../../../api/hooks/useDeployments', () => ({
  useCreateDeployment: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useDeployments: () => ({ data: { data: [], total_count: 0 }, isLoading: false }),
}));

vi.mock('../../../api/hooks/usePatches', () => ({
  usePatches: () => ({ data: { data: [], total_count: 0 }, isLoading: false }),
  usePatch: vi.fn(() => ({ data: null, isLoading: false })),
  usePatchSeverityCounts: vi.fn(() => ({})),
}));

vi.mock('../../../api/hooks/usePolicies', () => ({
  usePolicies: () => ({ data: { data: [] }, isLoading: false }),
}));

vi.mock('../../../api/hooks/useTags', () => ({
  useTags: () => ({ data: [], isLoading: false }),
  useAssignTag: () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }),
  useUnassignTag: () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }),
  useCreateTag: () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }),
  useUpdateTag: () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }),
  useDeleteTag: () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }),
  useTag: () => ({ data: null, isLoading: false }),
}));

vi.mock('../../../api/hooks/useAudit', () => ({
  useAuditLog: () => ({
    data: { data: [], total_count: 0 },
    isLoading: false,
  }),
}));

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});

const renderPage = () =>
  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={['/endpoints/1']}>
        <Routes>
          <Route path="/endpoints/:id" element={<EndpointDetailPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );

describe('EndpointDetailPage', () => {
  it('renders hostname', () => {
    renderPage();
    expect(screen.getAllByText('web-prod-01').length).toBeGreaterThanOrEqual(1);
  });

  it('renders OS info', () => {
    renderPage();
    expect(screen.getAllByText(/22\.04/i).length).toBeGreaterThanOrEqual(1);
  });

  it('renders Overview tab by default', () => {
    renderPage();
    expect(screen.getByText('Overview')).toBeInTheDocument();
  });

  it('renders Software tab', () => {
    renderPage();
    expect(screen.getByText('Software')).toBeInTheDocument();
  });

  it('renders Patches tab', () => {
    renderPage();
    // "Patches" appears in the tab bar and potentially in "Deploy Patches" button
    expect(screen.getAllByText(/^Patches$/).length).toBeGreaterThanOrEqual(1);
  });

  it('renders Deployments tab', () => {
    renderPage();
    expect(screen.getByText('Deployments')).toBeInTheDocument();
  });

  it('renders endpoint hostname', () => {
    renderPage();
    // Endpoint hostname is displayed as page heading
    expect(screen.getAllByText('web-prod-01').length).toBeGreaterThanOrEqual(1);
  });

  it('renders all 8 tab triggers', () => {
    renderPage();
    // Tabs are <button> elements, not Radix UI tabs with role="tab"
    // Current tab labels per TABS definition in EndpointDetailPage.tsx
    const tabNames = [
      'Overview',
      'Hardware',
      'Software',
      'CVE Exposure',
      'Compliance',
      'Deployments',
      'Audit',
    ];
    for (const name of tabNames) {
      expect(screen.getByRole('button', { name })).toBeInTheDocument();
    }
    // "Patches" appears in both the tab button and "Deploy Patches" button,
    // so just verify at least one button with exact name "Patches" exists
    const patchBtns = screen.getAllByRole('button', { name: /^Patches$/ });
    expect(patchBtns.length).toBeGreaterThanOrEqual(1);
  });

  it('renders stat cards', () => {
    renderPage();
    // StatStrip renders "Risk Score", "Patch Coverage", "Compliance", "Last Scan"
    expect(screen.getByText('Risk Score')).toBeInTheDocument();
    expect(screen.getByText('Patch Coverage')).toBeInTheDocument();
    expect(screen.getByText('Last Scan')).toBeInTheDocument();
  });
});
