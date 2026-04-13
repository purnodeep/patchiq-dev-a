import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router';
import { AuthProvider } from '../app/auth/AuthContext';
import { AppSidebar } from '../app/layout/AppSidebar';

vi.mock('../api/hooks/useAuth', () => ({
  useCurrentUser: () => ({
    data: {
      user_id: 'test-user',
      tenant_id: 'test-tenant',
      email: 'test@example.com',
      name: 'Test User',
      role: 'admin',
    },
    isLoading: false,
    isError: false,
  }),
  useLogout: () => ({ mutate: vi.fn(), isPending: false }),
}));

vi.mock('../api/hooks/useAlerts', () => ({
  useAlertCount: () => ({ data: { count: 0 } }),
}));

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});

const navLabels = [
  'Dashboard',
  'Endpoints',
  'Workflows',
  'Patches',
  'CVEs',
  'Policies',
  'Deployments',
  'Audit',
  'Settings',
];

describe('AppSidebar', () => {
  const renderSidebar = () =>
    render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>
          <AuthProvider>
            <AppSidebar />
          </AuthProvider>
        </MemoryRouter>
      </QueryClientProvider>,
    );

  it('renders without crashing', () => {
    const { container } = renderSidebar();
    expect(container).toBeTruthy();
  });

  it.each(navLabels)('renders navigation link: %s', (label) => {
    renderSidebar();
    expect(screen.getByText(label)).toBeInTheDocument();
  });
});
