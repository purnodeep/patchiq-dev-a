import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { SidebarProvider } from '@patchiq/ui';
import { AuthProvider } from '../app/auth/AuthContext';
import { AppSidebar } from '../app/layout/AppSidebar';

// Mock useCurrentUser so AuthProvider resolves immediately instead of loading
vi.mock('../api/hooks/useAuth', () => ({
  useCurrentUser: () => ({
    data: {
      user_id: 'test-user',
      tenant_id: '00000000-0000-0000-0000-000000000001',
      email: 'test@patchiq.local',
      name: 'Test User',
      role: 'admin',
    },
    isLoading: false,
    isError: false,
  }),
  useLogout: () => ({ mutate: vi.fn() }),
}));

// Mock useClients hooks used by AppSidebar
vi.mock('../api/hooks/useClients', () => ({
  usePendingClientCount: () => ({ data: { count: 0 }, isLoading: false }),
}));

const navLabels = ['Dashboard', 'Catalog', 'Feeds', 'Licenses', 'Clients', 'Settings'];

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});

describe('AppSidebar', () => {
  const renderSidebar = () =>
    render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>
          <AuthProvider>
            <SidebarProvider>
              <AppSidebar />
            </SidebarProvider>
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
    const elements = screen.getAllByText(label);
    expect(elements.length).toBeGreaterThanOrEqual(1);
  });
});
