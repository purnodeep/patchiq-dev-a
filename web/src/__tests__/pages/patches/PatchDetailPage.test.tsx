import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { PatchDetailPage } from '../../../pages/patches/PatchDetailPage';

vi.mock('../../../api/hooks/useEndpoints', () => ({
  useEndpoints: () => ({
    data: { data: [], next_cursor: null, total_count: 0 },
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  }),
}));

vi.mock('../../../api/hooks/usePolicies', () => ({
  usePolicies: () => ({ data: { data: [] }, isLoading: false }),
}));

vi.mock('../../../api/hooks/useDeployments', () => ({
  useCreateDeployment: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useDeployments: () => ({ data: { data: [], total_count: 0 }, isLoading: false }),
}));

vi.mock('../../../api/hooks/usePatches', () => ({
  usePatches: vi.fn(() => ({ data: null, isLoading: false, isError: false, refetch: vi.fn() })),
  usePatchSeverityCounts: vi.fn(() => ({})),
  usePatch: () => ({
    data: {
      id: '1',
      tenant_id: 't1',
      name: 'openssl',
      version: '3.0.14-1',
      severity: 'critical',
      os_family: 'ubuntu',
      status: 'available',
      os_distribution: 'jammy',
      package_url: 'pkg:deb/ubuntu/openssl@3.0.14-1',
      checksum_sha256: 'abc123',
      source_repo: 'https://packages.ubuntu.com',
      description: 'OpenSSL security update',
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-03-06T10:00:00Z',
      file_size: 2048576,
      cves: [
        {
          id: 'cve-1',
          cve_id: 'CVE-2026-0001',
          cvss_v3_score: 9.8,
          severity: 'critical',
          published_at: '2026-01-15T00:00:00Z',
        },
        {
          id: 'cve-2',
          cve_id: 'CVE-2026-0002',
          cvss_v3_score: 7.5,
          severity: 'high',
          published_at: null,
        },
      ],
      remediation: {
        endpoints_affected: 10,
        endpoints_patched: 6,
        endpoints_pending: 3,
        endpoints_failed: 1,
      },
      affected_endpoints: {
        count: 10,
        items: [
          {
            id: 'ep-1',
            hostname: 'web-server-01',
            os_family: 'ubuntu',
            current_version: '3.0.13-1',
            group: 'production',
          },
          {
            id: 'ep-2',
            hostname: 'db-server-01',
            os_family: 'debian',
            current_version: '3.0.12-1',
            group: null,
          },
        ],
        has_more: true,
      },
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
      <MemoryRouter initialEntries={['/patches/1']}>
        <Routes>
          <Route path="/patches/:id" element={<PatchDetailPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );

describe('PatchDetailPage', () => {
  it('renders patch name', () => {
    renderPage();
    expect(screen.getAllByText('openssl').length).toBeGreaterThanOrEqual(1);
  });

  it('renders version', () => {
    renderPage();
    expect(screen.getAllByText(/3.0.14-1/).length).toBeGreaterThanOrEqual(1);
  });

  it('renders severity badge', () => {
    renderPage();
    expect(screen.getAllByText('critical').length).toBeGreaterThanOrEqual(1);
  });

  it('renders CVE count context card', () => {
    renderPage();
    // HealthStrip renders "CVEs Linked" label
    expect(screen.getByText('CVEs Linked')).toBeInTheDocument();
  });

  it('renders description from CVE', () => {
    renderPage();
    expect(screen.getByText('OpenSSL security update')).toBeInTheDocument();
  });

  it('renders deployment summary section', () => {
    renderPage();
    // Overview tab shows "Endpoint Exposure" section with deployment stats
    expect(screen.getByText('Endpoint Exposure')).toBeInTheDocument();
  });

  it('renders deployed count in overview', () => {
    renderPage();
    // "Deployed", "Pending", "Failed" appear in the Endpoint Exposure section and HealthStrip
    expect(screen.getAllByText('Deployed').length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText('Pending').length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText('Failed').length).toBeGreaterThanOrEqual(1);
  });

  it('renders tab navigation', () => {
    renderPage();
    // CVEs tab appears with count, e.g. "CVEs (2)"
    expect(screen.getAllByText(/CVEs/).length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText(/Affected Endpoints/).length).toBeGreaterThanOrEqual(1);
  });

  it('renders deployed context card', () => {
    renderPage();
    // CVSS Score card was removed; verify Deployed health cell is present instead
    expect(screen.getAllByText('Deployed').length).toBeGreaterThanOrEqual(1);
  });

  it('renders blast radius section', () => {
    renderPage();
    // Overview shows "Endpoint Exposure" (replaces "Blast Radius" in new design)
    expect(screen.getByText('Endpoint Exposure')).toBeInTheDocument();
  });
});
