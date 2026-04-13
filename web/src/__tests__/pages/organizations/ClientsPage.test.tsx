import { render, screen, fireEvent } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ClientsPage } from '../../../pages/organizations/ClientsPage';

vi.mock('../../../app/auth/AuthContext', () => ({
  useAuth: () => ({
    user: {
      user_id: 'u1',
      organization: {
        id: 'org-123',
        name: 'Acme Holdings',
        slug: 'acme',
        type: 'msp' as const,
      },
    },
  }),
}));

vi.mock('../../../api/hooks/useOrganizations', () => ({
  useOrgTenants: () => ({
    data: {
      data: [
        {
          id: 't1',
          name: 'Tenant One',
          slug: 'tenant-one',
          created_at: '2026-01-01T00:00:00Z',
        },
        {
          id: 't2',
          name: 'Tenant Two',
          slug: 'tenant-two',
          created_at: '2026-02-01T00:00:00Z',
        },
      ],
    },
    isLoading: false,
    isError: false,
  }),
  useProvisionTenant: () => ({
    mutateAsync: vi.fn(),
    isPending: false,
    reset: vi.fn(),
  }),
}));

const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

const renderPage = () =>
  render(
    <QueryClientProvider client={queryClient}>
      <ClientsPage />
    </QueryClientProvider>,
  );

describe('ClientsPage', () => {
  it('renders the list of client tenants', () => {
    renderPage();
    expect(screen.getByText('Tenant One')).toBeInTheDocument();
    expect(screen.getByText('Tenant Two')).toBeInTheDocument();
  });

  it('shows the Add Client button', () => {
    renderPage();
    expect(screen.getByRole('button', { name: /add client/i })).toBeInTheDocument();
  });

  it('opens the AddClientDialog when Add Client is clicked', () => {
    renderPage();
    fireEvent.click(screen.getByRole('button', { name: /add client/i }));
    expect(screen.getByRole('dialog')).toBeInTheDocument();
    expect(screen.getByText(/provision a new client tenant/i)).toBeInTheDocument();
  });
});
