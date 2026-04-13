import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { DeploymentDetailPage } from '../../../pages/deployments/DeploymentDetailPage';

const mockDeployment = {
  id: 'abc-123-def-456',
  tenant_id: 't1',
  policy_id: 'p1',
  status: 'running' as const,
  target_count: 10,
  completed_count: 6,
  success_count: 5,
  failed_count: 1,
  created_by: null,
  started_at: '2026-01-01T00:01:00Z',
  completed_at: null,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:01:00Z',
  targets: [
    {
      id: 'tgt-1',
      deployment_id: 'abc-123-def-456',
      endpoint_id: 'ep-1',
      hostname: 'web-server-01',
      status: 'succeeded' as const,
      exit_code: 0,
      output: 'installed OK',
      error: null,
      wave_id: null,
      started_at: '2026-01-01T00:01:00Z',
      created_at: '2026-01-01T00:01:00Z',
      completed_at: '2026-01-01T00:02:00Z',
    },
  ],
};

let mockStatus = 'running';
let mockData: typeof mockDeployment | null = mockDeployment;

vi.mock('react-router', async () => {
  const actual = await vi.importActual('react-router');
  return {
    ...actual,
    useParams: () => ({ id: 'abc-123-def-456' }),
  };
});

vi.mock('../../../api/hooks/useDeployments', () => ({
  useDeployment: () => ({
    data: mockData ? { ...mockData, status: mockStatus } : null,
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  }),
  useCancelDeployment: () => ({
    mutateAsync: vi.fn(),
    isPending: false,
  }),
  useRetryDeployment: () => ({
    mutateAsync: vi.fn(),
    isPending: false,
  }),
  useRollbackDeployment: () => ({
    mutateAsync: vi.fn(),
    isPending: false,
  }),
  useDeploymentWaves: () => ({
    data: undefined,
    isLoading: false,
  }),
  useDeploymentPatches: () => ({
    data: undefined,
    isLoading: false,
  }),
}));

vi.mock('../../../api/hooks/usePolicies', () => ({
  usePolicy: () => ({
    data: { id: 'p1', name: 'Test Policy' },
    isLoading: false,
  }),
}));

const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

const renderPage = () =>
  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DeploymentDetailPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );

describe('DeploymentDetailPage', () => {
  beforeEach(() => {
    mockStatus = 'running';
    mockData = mockDeployment;
  });

  it('renders deployment ID in monospace', () => {
    renderPage();
    // formatDeploymentId('abc-123-def-456') = 'D-ABC-12' (first 6 chars uppercased, prefixed with D-)
    expect(screen.getByText('D-ABC-12')).toBeInTheDocument();
  });

  it('renders tabs', () => {
    renderPage();
    expect(screen.getByText('Overview')).toBeInTheDocument();
    expect(screen.getByText('Waves')).toBeInTheDocument();
    // Tab label is "Endpoints" (with optional count), not "Endpoint Results"
    const endpointsTab = screen.getByRole('button', { name: /^Endpoints/ });
    expect(endpointsTab).toBeInTheDocument();
    expect(screen.getByText('Patches Deployed')).toBeInTheDocument();
    // "Timeline" appears in tab nav and also as a section label inside the overview tab
    expect(screen.getAllByText('Timeline').length).toBeGreaterThanOrEqual(1);
  });

  it('renders stat cards', () => {
    renderPage();
    // Progress strip has "Progress", "Succeeded", "Failed", "Duration"
    expect(screen.getByText('Progress')).toBeInTheDocument();
    // "Succeeded" appears in both the stat strip and the endpoint dot map legend
    expect(screen.getAllByText('Succeeded').length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText('Failed').length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText('Duration').length).toBeGreaterThanOrEqual(1);
  });

  it('shows cancel button when status is running', () => {
    mockStatus = 'running';
    renderPage();
    expect(screen.getByRole('button', { name: /cancel/i })).toBeInTheDocument();
  });

  it('does NOT show cancel button when status is completed', () => {
    mockStatus = 'completed';
    renderPage();
    expect(screen.queryByRole('button', { name: /cancel/i })).not.toBeInTheDocument();
  });
});
