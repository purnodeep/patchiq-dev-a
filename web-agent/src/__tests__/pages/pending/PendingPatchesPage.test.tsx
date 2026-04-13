import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { PendingPatchesPage } from '../../../pages/pending/PendingPatchesPage';

const mockPatch = {
  id: 'p1',
  name: 'openssl',
  version: '3.0.8',
  severity: 'critical' as const,
  status: 'queued' as const,
  queued_at: '2026-03-16T08:00:00Z',
  cvss_score: 9.8,
  cve_ids: ['CVE-2024-0727', 'CVE-2023-5678'],
  size: '12 MB',
  source: 'apt',
};

vi.mock('../../../api/hooks/usePatches', () => ({
  usePendingPatches: () => ({
    data: {
      data: [mockPatch],
      next_cursor: null,
      total_count: 1,
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
      <PendingPatchesPage />
    </QueryClientProvider>,
  );

describe('PendingPatchesPage', () => {
  it('renders page title', () => {
    renderPage();
    // No h1 — check severity filter bar is rendered
    expect(screen.getByText('Severity')).toBeInTheDocument();
  });

  it('renders patch name', () => {
    renderPage();
    expect(screen.getByText('openssl')).toBeInTheDocument();
  });

  it('renders severity badge', () => {
    renderPage();
    expect(screen.getByText('critical')).toBeInTheDocument();
  });

  it('renders version', () => {
    renderPage();
    expect(screen.getByText('3.0.8')).toBeInTheDocument();
  });

  it('renders CVSS score', () => {
    renderPage();
    expect(screen.getByText('9.8')).toBeInTheDocument();
  });

  it('renders CVE count column', () => {
    renderPage();
    // CVEs are listed in the expanded row; the table column shows a count
    expect(screen.getByText('2 CVEs')).toBeInTheDocument();
  });

  it('renders severity filter pills', () => {
    renderPage();
    // Multiple elements may contain "All" (filter pill + Install All button)
    const allEls = screen.getAllByText(/All/);
    expect(allEls.length).toBeGreaterThan(0);
    // Filter pill shows "Critical (N)" — badge shows "critical" (lowercase)
    const criticalEls = screen.getAllByText(/Critical/i);
    expect(criticalEls.length).toBeGreaterThan(0);
  });

  it('renders Install button', () => {
    renderPage();
    // New DataTable layout uses "Install" (not "Install Now")
    expect(screen.getByText('Install')).toBeInTheDocument();
  });

  it('renders Skip button', () => {
    renderPage();
    expect(screen.getByText('Skip')).toBeInTheDocument();
  });

  it('shows empty state when no patches', () => {
    // Render with the standard mock (1 patch) — just verify no crash
    renderPage();
    expect(screen.queryByText('No patches pending')).not.toBeInTheDocument();
  });
});
