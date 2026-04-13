import { render, screen, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

vi.mock('../../api/hooks/useClients', () => ({
  useClients: () => ({
    data: {
      clients: [
        {
          id: '1',
          hostname: 'pm-server-01',
          version: '1.0.0',
          os: 'linux',
          endpoint_count: 50,
          contact_email: 'admin@test.com',
          status: 'pending' as const,
          sync_interval: 21600,
          last_sync_at: null,
          notes: null,
          created_at: '2026-01-01T00:00:00Z',
          updated_at: '2026-01-01T00:00:00Z',
        },
      ],
      total: 1,
    },
    isLoading: false,
    isError: false,
    error: null,
  }),
  useApproveClient: () => ({ mutate: vi.fn(), isPending: false }),
  useDeclineClient: () => ({ mutate: vi.fn(), isPending: false }),
  useSuspendClient: () => ({ mutate: vi.fn(), isPending: false }),
  useDeleteClient: () => ({ mutate: vi.fn(), isPending: false }),
  useClientEndpointTrend: () => ({ data: null, isLoading: false, isError: false }),
  useClientSyncHistory: () => ({ data: null, isLoading: false, isError: false }),
}));

vi.mock('../../api/hooks/useLicenses', () => ({
  useLicenses: () => ({
    data: { licenses: [], total: 0 },
    isLoading: false,
    isError: false,
    error: null,
  }),
  useRevokeLicense: () => ({ mutate: vi.fn(), isPending: false }),
  useAssignLicense: () => ({ mutate: vi.fn(), isPending: false }),
  useCreateLicense: () => ({
    mutate: vi.fn(),
    mutateAsync: vi.fn(),
    isPending: false,
    isError: false,
    error: null,
  }),
}));

import { ClientsPage } from '../../pages/clients/ClientsPage';

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});

const renderPage = () =>
  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <ClientsPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );

describe('ClientsPage', () => {
  it('renders page via "Total Clients" stat tile', () => {
    renderPage();
    // Redesigned page has no h1 — use stat tile button as the page identifier.
    expect(screen.getByRole('button', { name: /total clients/i })).toBeInTheDocument();
  });

  it('renders hostname', () => {
    renderPage();
    expect(screen.getByText('pm-server-01')).toBeInTheDocument();
  });

  it.each(['Total Clients', 'Connected', 'Pending', 'Disconnected'])(
    'renders status stat tile "%s"',
    (label) => {
      renderPage();
      const buttons = screen.getAllByRole('button', { name: new RegExp(label, 'i') });
      expect(buttons.length).toBeGreaterThanOrEqual(1);
    },
  );

  it('renders "Approve" and "Decline" buttons for pending client after expanding row', async () => {
    renderPage();
    // Approve/Decline are in the expandable row — click the row to expand it
    const row = screen.getByText('pm-server-01').closest('tr');
    if (row) fireEvent.click(row);
    expect(await screen.findByRole('button', { name: 'Approve' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Decline' })).toBeInTheDocument();
  });

  it('renders total count in stat tile', () => {
    renderPage();
    // Redesigned page shows total inside the "Total Clients" stat tile button.
    const totalTile = screen.getByRole('button', { name: /total clients/i });
    expect(totalTile).toHaveTextContent('1');
  });
});
