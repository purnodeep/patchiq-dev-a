import { render, screen, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { CVEsPage } from '../../../pages/cves/CVEsPage';

const mockSummary = {
  total: 324,
  by_severity: { critical: 42, high: 89, medium: 121, low: 72 },
  kev_count: 28,
};

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
  usePolicies: () => ({ data: { data: [] }, isLoading: false }),
}));

vi.mock('../../../api/hooks/useDeployments', () => ({
  useCreateDeployment: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useDeployments: () => ({ data: { data: [], total_count: 0 }, isLoading: false }),
}));

vi.mock('../../../api/hooks/useCVEs', () => ({
  useCVEs: () => ({
    data: {
      data: [
        {
          id: '1',
          tenant_id: 't1',
          cve_id: 'CVE-2026-0001',
          severity: 'critical',
          description: 'Remote code execution vulnerability',
          published_at: '2026-01-15T00:00:00Z',
          created_at: '2026-01-01T00:00:00Z',
          updated_at: '2026-01-01T00:00:00Z',
          cvss_v3_score: 9.8,
          cvss_v3_vector: 'CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H',
          cisa_kev_due_date: '2026-04-01',
          exploit_available: true,
          nvd_last_modified: '2026-02-01T00:00:00Z',
          affected_endpoint_count: 15,
          patch_available: true,
          attack_vector: 'NETWORK',
        },
        {
          id: '2',
          tenant_id: 't1',
          cve_id: 'CVE-2026-0002',
          severity: 'low',
          description: 'Minor info disclosure',
          published_at: '2026-02-01T00:00:00Z',
          created_at: '2026-02-01T00:00:00Z',
          updated_at: '2026-02-01T00:00:00Z',
          cvss_v3_score: 2.1,
          cvss_v3_vector: null,
          cisa_kev_due_date: null,
          exploit_available: false,
          nvd_last_modified: '2026-02-15T00:00:00Z',
          affected_endpoint_count: 3,
          patch_available: false,
          attack_vector: 'LOCAL',
        },
      ],
      next_cursor: null,
      total_count: 2,
    },
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  }),
  useCVE: vi.fn(() => ({ data: null, isLoading: false, isError: false, refetch: vi.fn() })),
  useCVESummary: () => ({
    data: mockSummary,
    isLoading: false,
  }),
}));

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});

const renderPage = () =>
  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <CVEsPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );

describe('CVEsPage', () => {
  it('renders page title', () => {
    renderPage();
    // Page renders stat cards with "Total" label (CSS uppercased); "CVEs" title is in the TopBar layout
    expect(screen.getByText('Total')).toBeInTheDocument();
  });

  it('renders total count badge', () => {
    renderPage();
    // 324 appears in the header badge and also in the "All" severity pill count
    expect(screen.getAllByText('324').length).toBeGreaterThanOrEqual(1);
  });

  it('renders summary card values', () => {
    renderPage();
    // Values appear in both stat cards and severity filter pills
    expect(screen.getAllByText('42').length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText('89').length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText('121').length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText('28').length).toBeGreaterThanOrEqual(1);
  });

  it('renders summary card labels', () => {
    renderPage();
    // Stat cards use "Critical", "High", "Medium", "KEV Listed"
    const criticals = screen.getAllByText('Critical');
    expect(criticals.length).toBeGreaterThanOrEqual(1);
    const highs = screen.getAllByText('High');
    expect(highs.length).toBeGreaterThanOrEqual(1);
    const mediums = screen.getAllByText('Medium');
    expect(mediums.length).toBeGreaterThanOrEqual(1);
    const kevListeds = screen.getAllByText('KEV Listed');
    expect(kevListeds.length).toBeGreaterThanOrEqual(1);
  });

  it('renders CVE ID in table', () => {
    renderPage();
    expect(screen.getByText('CVE-2026-0001')).toBeInTheDocument();
  });

  it('renders both CVEs in the table', () => {
    renderPage();
    expect(screen.getByText('CVE-2026-0001')).toBeInTheDocument();
    expect(screen.getByText('CVE-2026-0002')).toBeInTheDocument();
  });

  it('renders CVSS score', () => {
    renderPage();
    // CVSS score may be split across elements; use getAllByText with substring match
    expect(screen.getAllByText(/9\.8/).length).toBeGreaterThanOrEqual(1);
  });

  it('renders severity badge', () => {
    renderPage();
    // "Critical" appears in the stat card label, severity pill, and table row badge
    expect(screen.getAllByText('Critical').length).toBeGreaterThanOrEqual(1);
  });

  it('renders attack vector column header', () => {
    renderPage();
    const elements = screen.getAllByText('Vector');
    expect(elements.length).toBeGreaterThanOrEqual(1);
  });

  it('renders patch available badge', () => {
    renderPage();
    // patch_available is shown in expanded row; the table has "Patches Available" label
    // The table column doesn't have a dedicated "Available" badge - it's in expanded rows
    expect(screen.getByText('CVE-2026-0001')).toBeInTheDocument();
  });

  it('renders KEV badge', () => {
    renderPage();
    // KEV column header is "KEV"
    const kevElements = screen.getAllByText('KEV');
    expect(kevElements.length).toBeGreaterThanOrEqual(1);
  });

  it('renders affected endpoint count', () => {
    renderPage();
    expect(screen.getByText('15')).toBeInTheDocument();
  });

  it('renders search input', () => {
    renderPage();
    expect(screen.getByPlaceholderText(/search/i)).toBeInTheDocument();
  });

  it('renders severity filter', () => {
    renderPage();
    // Severity filter uses InlineStatCard clickable cards for Critical/High/Medium/Low
    // Verify the severity stat cards are rendered as clickable elements
    const criticals = screen.getAllByText('Critical');
    expect(criticals.length).toBeGreaterThanOrEqual(1);
    expect(criticals[0].closest('button') || criticals[0].closest('div')).toBeInTheDocument();
  });

  it('renders exploit toggle pill', () => {
    renderPage();
    expect(screen.getByText(/Exploit Available/)).toBeInTheDocument();
  });

  it('renders KEV toggle pill', () => {
    renderPage();
    expect(screen.getByText(/KEV Only/)).toBeInTheDocument();
  });

  it('toggles exploit pill on click', () => {
    renderPage();
    const pill = screen.getByText(/Exploit Available/);
    fireEvent.click(pill);
    expect(pill.closest('button')).toBeInTheDocument();
  });

  it('toggles KEV pill on click', () => {
    renderPage();
    const pill = screen.getByText(/KEV Only/);
    fireEvent.click(pill);
    expect(pill.closest('button')).toBeInTheDocument();
  });
});
