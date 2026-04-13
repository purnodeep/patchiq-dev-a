import { render, screen, fireEvent } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { PolicyDetailPage } from '../../../pages/policies/PolicyDetailPage';

vi.mock('../../../api/hooks/usePolicies', () => ({
  usePolicy: () => ({
    data: {
      id: 'p1',
      tenant_id: 't1',
      name: 'Critical Only',
      description: 'Deploy critical patches only',
      enabled: true,
      selection_mode: 'by_severity',
      target_selector: { eq: { key: 'env', value: 'prod' } },
      min_severity: 'critical',
      cve_ids: [],
      package_regex: null,
      exclude_packages: [],
      schedule_type: 'recurring',
      schedule_cron: '0 2 * * *',
      mw_start: '02:00',
      mw_end: '06:00',
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
      mode: 'automatic',
      target_endpoints_count: 42,
      last_evaluated_at: '2026-01-01T00:00:00Z',
      last_eval_pass: true,
      last_eval_endpoint_count: 42,
      last_eval_compliant_count: 40,
      matched_endpoints: [],
      recent_evaluations: [
        {
          id: 'e1',
          evaluated_at: '2026-01-01T00:00:00Z',
          matched_patches: 5,
          in_scope_endpoints: 42,
          compliant_count: 40,
          non_compliant_count: 2,
          duration_ms: 120,
          pass: true,
        },
      ],
      recent_deployments: [
        {
          id: 'd1',
          tenant_id: 't1',
          policy_id: 'p1',
          status: 'completed',
          created_at: '2026-01-01T00:00:00Z',
          updated_at: '2026-01-01T00:00:00Z',
          target_count: 42,
          success_count: 42,
          started_at: '2026-01-01T00:00:00Z',
          completed_at: '2026-01-01T00:01:00Z',
        },
      ],
      deployment_count: 5,
    },
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  }),
  useEvaluatePolicy: () => ({
    mutate: vi.fn(),
    mutateAsync: vi.fn(),
    isPending: false,
    data: null,
  }),
  useDeletePolicy: () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }),
  useTogglePolicy: () => ({ mutate: vi.fn() }),
}));

const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

const renderPage = () =>
  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={['/policies/p1']}>
        <Routes>
          <Route path="/policies/:id" element={<PolicyDetailPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );

describe('PolicyDetailPage', () => {
  it('renders name and mode badge', () => {
    renderPage();
    expect(screen.getByRole('heading', { name: 'Critical Only' })).toBeInTheDocument();
    expect(screen.getByText('Automatic')).toBeInTheDocument();
  });

  it('renders all 6 tabs', () => {
    renderPage();
    expect(screen.getByText('Overview')).toBeInTheDocument();
    expect(screen.getByText('Patch Scope')).toBeInTheDocument();
    expect(screen.getByText(/Groups.*Endpoints/)).toBeInTheDocument();
    expect(screen.getByText('Evaluation History')).toBeInTheDocument();
    expect(screen.getByText('Deployment History')).toBeInTheDocument();
    expect(screen.getByText('Schedule')).toBeInTheDocument();
  });

  it('renders context ring stats', () => {
    renderPage();
    // "Pass Rate" appears in the health strip and possibly in overview tab
    expect(screen.getAllByText('Pass Rate').length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText('Endpoints').length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText('Patches').length).toBeGreaterThanOrEqual(1);
  });

  it('renders action buttons including Deploy Now and Clone', () => {
    renderPage();
    expect(screen.getByText('Evaluate Now')).toBeInTheDocument();
    expect(screen.getByText('Deploy Now')).toBeInTheDocument();
    // Clone Policy and Delete Policy live in a "..." overflow dropdown — click it to open
    const allButtons = screen.getAllByRole('button');
    // The overflow button has no text, only an SVG icon
    const moreButton = allButtons.find(
      (btn) => btn.textContent?.trim() === '' && btn.querySelector('svg'),
    );
    expect(moreButton).toBeDefined();
    fireEvent.click(moreButton!);
    expect(screen.getByText('Clone Policy')).toBeInTheDocument();
    expect(screen.getByText('Delete Policy')).toBeInTheDocument();
  });
});
