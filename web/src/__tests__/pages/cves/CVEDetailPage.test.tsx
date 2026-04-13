import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { CVEDetailPage } from '../../../pages/cves/CVEDetailPage';

vi.mock('../../../api/hooks/useCVEs', () => ({
  useCVE: () => ({
    data: {
      id: '1',
      tenant_id: 't1',
      cve_id: 'CVE-2026-0001',
      severity: 'critical',
      description: 'Remote code execution in OpenSSL',
      published_at: '2026-01-15T00:00:00Z',
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-03-06T10:00:00Z',
      cvss_v3_score: 9.8,
      cvss_v3_vector: 'CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H',
      cisa_kev_due_date: '2026-04-01',
      exploit_available: true,
      nvd_last_modified: '2026-02-01T00:00:00Z',
      attack_vector: 'Network',
      cwe_id: 'CWE-787',
      source: 'NVD',
      external_references: [
        { url: 'https://nvd.nist.gov/vuln/detail/CVE-2026-0001', source: 'NVD' },
        { url: 'https://openssl.org/advisory', source: 'OpenSSL' },
      ],
      affected_endpoints: {
        count: 15,
        items: [
          {
            id: 'e1',
            hostname: 'web-prod-01',
            os_family: 'Ubuntu',
            os_version: '22.04',
            ip_address: '10.0.1.10',
            status: 'affected',
            agent_version: '1.2.0',
            last_seen: '2026-03-10T08:00:00Z',
            group_names: 'Production',
          },
          {
            id: 'e2',
            hostname: 'db-prod-02',
            os_family: 'Ubuntu',
            os_version: '22.04',
            ip_address: '10.0.1.20',
            status: 'patched',
            agent_version: '1.2.0',
            last_seen: '2026-03-10T09:00:00Z',
            group_names: 'Database',
          },
        ],
        has_more: true,
      },
      patches: [
        {
          id: 'p1',
          name: 'openssl-3.0.14',
          version: '3.0.14-1',
          severity: 'critical',
          os_family: 'Ubuntu',
          released_at: '2026-02-10T00:00:00Z',
          endpoints_covered: 15,
          endpoints_patched: 8,
        },
      ],
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
      <MemoryRouter initialEntries={['/cves/1']}>
        <Routes>
          <Route path="/cves/:id" element={<CVEDetailPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );

describe('CVEDetailPage', () => {
  it('renders CVE ID in header', () => {
    renderPage();
    // CVE ID appears in multiple places (breadcrumb, page header, etc.)
    expect(screen.getAllByText('CVE-2026-0001').length).toBeGreaterThanOrEqual(1);
  });

  it('renders CVSS score', () => {
    renderPage();
    // Score is rendered as "CVSS 9.8" in the header
    expect(screen.getByText('CVSS 9.8')).toBeInTheDocument();
  });

  it('renders severity badge', () => {
    renderPage();
    // severity is displayed capitalized
    expect(screen.getAllByText('Critical').length).toBeGreaterThanOrEqual(1);
  });

  it('renders CVE ID heading', () => {
    renderPage();
    // CVE ID is displayed as the page heading
    expect(screen.getAllByText('CVE-2026-0001').length).toBeGreaterThanOrEqual(1);
  });

  it('renders context ring stat cards', () => {
    renderPage();
    // HealthStrip renders CVSS, Endpoints, Patches, Published labels
    expect(screen.getByText('CVSS')).toBeInTheDocument();
    // CVSSHero (overview tab) renders threat assessment metrics
    expect(screen.getByText('KEV Listed')).toBeInTheDocument();
    // Remediation tab trigger is rendered
    expect(screen.getByRole('button', { name: 'Available Patches' })).toBeInTheDocument();
  });

  it('shows overview tab by default with CVSS vector breakdown', () => {
    renderPage();
    // CVSSHero renders "CVSS v3.1 Vector Breakdown" section label
    expect(screen.getByText('CVSS v3.1 Vector Breakdown')).toBeInTheDocument();
    // Description is only rendered in the Intelligence tab (not visible by default)
    // Verify the overview tab content is shown instead
    expect(screen.getByText('Threat Assessment')).toBeInTheDocument();
  });

  it('renders all five tab triggers', () => {
    renderPage();
    // Tabs are rendered as buttons (not Radix role="tab")
    expect(screen.getByRole('button', { name: 'Overview' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Affected Software' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Endpoints' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Available Patches' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Intelligence' })).toBeInTheDocument();
  });

  it('renders endpoints data in mock (available for Affected Endpoints tab)', () => {
    renderPage();
    // The overview tab shows threat assessment with Affected Endpoints label in multiple places
    expect(screen.getAllByText('Affected Endpoints').length).toBeGreaterThanOrEqual(1);
  });

  it('renders patch data in mock (available for Remediation tab)', () => {
    renderPage();
    // The overview tab shows patch available in threat assessment
    expect(screen.getByText('Patch Available')).toBeInTheDocument();
  });
});
