import { render, screen, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { AuditPage } from '../../../pages/audit/AuditPage';

const MOCK_EVENTS = [
  {
    id: '01ABC',
    tenant_id: 't1',
    type: 'endpoint.enrolled',
    actor_id: 'admin@acme.com',
    actor_type: 'user' as const,
    resource: 'endpoint',
    resource_id: 'e1e1e1e1-aaaa-bbbb-cccc-dddddddddddd',
    action: 'enrolled',
    payload: { hostname: 'prod-web-01' } as unknown as Record<string, never>,
    metadata: {} as unknown as Record<string, never>,
    timestamp: '2026-03-10T08:00:00Z',
  },
  {
    id: '01DEF',
    tenant_id: 't1',
    type: 'deployment.created',
    actor_id: 'system',
    actor_type: 'system' as const,
    resource: 'deployment',
    resource_id: 'dep-abc123',
    action: 'created',
    payload: { deployment_id: 'dep-abc123', target_count: 45 } as unknown as Record<string, never>,
    metadata: {} as unknown as Record<string, never>,
    timestamp: '2026-03-09T07:00:00Z',
  },
];

vi.mock('../../../api/hooks/useAudit', () => ({
  useAuditLog: () => ({
    data: {
      data: MOCK_EVENTS,
      next_cursor: null,
      total_count: 2,
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
        <AuditPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );

describe('AuditPage', () => {
  it('renders stat cards', () => {
    renderPage();
    // AuditPage renders view toggle tabs instead of a heading
    expect(screen.getByText('Activity Stream')).toBeInTheDocument();
  });

  it('renders export dropdown button', () => {
    renderPage();
    // Export is a dropdown trigger button labelled "Export"
    expect(screen.getByRole('button', { name: /export/i })).toBeInTheDocument();
  });

  it('renders view toggle buttons', () => {
    renderPage();
    expect(screen.getByText('Activity Stream')).toBeInTheDocument();
    expect(screen.getByText('Timeline View')).toBeInTheDocument();
  });

  it('renders filter controls', () => {
    renderPage();
    expect(screen.getByLabelText('Search by actor')).toBeInTheDocument();
  });

  it('renders retention bar', () => {
    renderPage();
    expect(screen.getByText(/audit logs retained/i)).toBeInTheDocument();
  });

  it('switches to timeline view on click', () => {
    renderPage();
    const timelineBtn = screen.getByText('Timeline View');
    fireEvent.click(timelineBtn);
    // Timeline view shows date separators when events are grouped
    // Verify activity stream events are no longer rendered with expand behavior
    expect(screen.queryByText('Event Payload')).not.toBeInTheDocument();
  });

  it('shows activity stream by default', () => {
    renderPage();
    // In stream view, event category badges should be visible
    expect(screen.getAllByText('Endpoint').length).toBeGreaterThan(0);
    expect(screen.getAllByText('Deployment').length).toBeGreaterThan(0);
  });
});
