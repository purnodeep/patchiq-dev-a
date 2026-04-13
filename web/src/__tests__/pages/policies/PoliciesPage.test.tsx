import { render, screen, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { PoliciesPage } from '../../../pages/policies/PoliciesPage';

const mockPolicies = [
  {
    id: 'p1',
    tenant_id: 't1',
    name: 'Critical Patches Only',
    description: 'Deploy critical patches',
    enabled: true,
    selection_mode: 'by_severity' as const,
    target_selector: { eq: { key: 'env', value: 'prod' } },
    min_severity: 'critical' as const,
    cve_ids: [],
    package_regex: null,
    exclude_packages: [],
    schedule_type: 'manual' as const,
    schedule_cron: null,
    mw_start: null,
    mw_end: null,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    mode: 'automatic' as const,
    target_endpoints_count: 10,
    last_evaluated_at: '2026-01-01T00:00:00Z',
    last_eval_pass: true,
    last_eval_endpoint_count: 10,
    last_eval_compliant_count: 9,
  },
  {
    id: 'p2',
    tenant_id: 't1',
    name: 'Manual Review',
    description: null,
    enabled: false,
    selection_mode: 'all_available' as const,
    target_selector: null,
    cve_ids: [],
    exclude_packages: [],
    schedule_type: 'manual' as const,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    mode: 'manual' as const,
    target_endpoints_count: 0,
  },
];

vi.mock('../../../api/hooks/usePolicies', () => ({
  usePolicies: () => ({
    data: { data: mockPolicies, next_cursor: null, total_count: 2 },
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  }),
  useCreatePolicy: () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }),
  useTogglePolicy: () => ({ mutate: vi.fn() }),
  useBulkPolicyAction: () => ({ mutate: vi.fn() }),
  useEvaluatePolicy: () => ({ mutate: vi.fn(), isPending: false }),
  useDeletePolicy: () => ({ mutate: vi.fn(), isPending: false }),
  useUpdatePolicy: () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }),
  usePolicy: () => ({ data: null, isLoading: false }),
}));


const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

const renderPage = () =>
  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <PoliciesPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );

describe('PoliciesPage', () => {
  it('renders header with count', () => {
    renderPage();
    // PoliciesPage uses StatCards instead of a heading; verify total count is rendered
    expect(screen.getByText('Total')).toBeInTheDocument();
    expect(screen.getByText('2')).toBeInTheDocument();
  });

  it('renders filter pills', () => {
    renderPage();
    // Filter pills are buttons; mode badges in the table are not buttons
    const buttons = screen.getAllByRole('button');
    const buttonTexts = buttons.map((b) => b.textContent);
    expect(buttonTexts).toContain('Automatic');
    expect(buttonTexts).toContain('Advisory');
    expect(buttonTexts).toContain('Enabled');
    expect(buttonTexts).toContain('Disabled');
  });

  it('renders policy names', () => {
    renderPage();
    expect(screen.getByText('Critical Patches Only')).toBeInTheDocument();
    expect(screen.getByText('Manual Review')).toBeInTheDocument();
  });

  it('renders create policy button', () => {
    renderPage();
    // Create Policy is a button (opens dialog), not a link
    expect(screen.getByRole('button', { name: /create policy/i })).toBeInTheDocument();
  });

  it('shows bulk action bar on selection', () => {
    renderPage();
    // CB is a custom div (not a checkbox role); click the select-all CB in the table header
    const headerCBs = document.querySelectorAll('thead div[style*="cursor: pointer"]');
    if (headerCBs.length > 0) {
      fireEvent.click(headerCBs[0]);
    }
    expect(screen.getByText('selected')).toBeInTheDocument();
  });

  it('hides bulk action bar when cleared', () => {
    renderPage();
    // CB is a custom div; click select-all
    const headerCBs = document.querySelectorAll('thead div[style*="cursor: pointer"]');
    if (headerCBs.length > 0) {
      fireEvent.click(headerCBs[0]);
    }
    expect(screen.getByText('selected')).toBeInTheDocument();
    const clearBtn = screen.getByRole('button', { name: 'Clear' });
    fireEvent.click(clearBtn);
    expect(screen.queryByText('selected')).not.toBeInTheDocument();
  });
});
