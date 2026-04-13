import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

vi.mock('../../api/hooks/useLicenses', () => ({
  useLicenses: () => ({
    data: {
      licenses: [
        {
          id: '1',
          tenant_id: 't1',
          client_id: null,
          client_hostname: null,
          license_key: 'key-1',
          tier: 'professional' as const,
          max_endpoints: 100,
          issued_at: '2026-01-01T00:00:00Z',
          expires_at: '2027-01-01T00:00:00Z',
          revoked_at: null,
          customer_name: 'Test Corp',
          customer_email: 'test@corp.com',
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

vi.mock('../../api/hooks/useClients', () => ({
  useClients: () => ({
    data: { clients: [], total: 0 },
    isLoading: false,
    isError: false,
    error: null,
  }),
}));

// Mock LicenseForm to avoid Dialog portal issues
vi.mock('../../pages/licenses/LicenseForm', () => ({
  LicenseForm: () => null,
}));

import { LicensesPage } from '../../pages/licenses/LicensesPage';

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});

const renderPage = () =>
  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <LicensesPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );

describe('LicensesPage', () => {
  it('renders page via "Total Licenses" stat tile', () => {
    renderPage();
    // Redesigned page has no h1 — use stat tile button as the page identifier.
    expect(screen.getByRole('button', { name: /total licenses/i })).toBeInTheDocument();
  });

  it('renders "+ Generate License" button', () => {
    renderPage();
    expect(screen.getByRole('button', { name: /generate license/i })).toBeInTheDocument();
  });

  it('renders customer name', () => {
    renderPage();
    expect(screen.getByText('Test Corp')).toBeInTheDocument();
  });

  it('renders tier badge', () => {
    renderPage();
    expect(screen.getByText('professional')).toBeInTheDocument();
  });

  it('renders total count in stat tile', () => {
    renderPage();
    const totalTile = screen.getByRole('button', { name: /total licenses/i });
    expect(totalTile).toHaveTextContent('1');
  });

  it('renders tier filter pills', () => {
    renderPage();
    // New UI uses FilterPill buttons for tier filtering: "Standard" and "Enterprise"
    expect(screen.getByRole('button', { name: /standard/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /enterprise/i })).toBeInTheDocument();
  });

  it('renders status stat tiles', () => {
    renderPage();
    // Redesigned page filters status via stat tile buttons.
    expect(screen.getByRole('button', { name: /total licenses/i })).toBeInTheDocument();
    expect(screen.getAllByRole('button', { name: /active/i }).length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByRole('button', { name: /expiring/i }).length).toBeGreaterThanOrEqual(1);
  });
});
