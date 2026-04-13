import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { AddClientDialog } from '../../../pages/organizations/AddClientDialog';

const mutateAsync = vi.fn().mockResolvedValue({ id: 't-new' });

vi.mock('../../../api/hooks/useOrganizations', () => ({
  useProvisionTenant: () => ({
    mutateAsync,
    isPending: false,
    reset: vi.fn(),
  }),
}));

vi.mock('sonner', () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

const renderDialog = (open = true) =>
  render(
    <QueryClientProvider client={queryClient}>
      <AddClientDialog orgId="org-123" open={open} onOpenChange={vi.fn()} />
    </QueryClientProvider>,
  );

describe('AddClientDialog', () => {
  beforeEach(() => {
    mutateAsync.mockClear();
  });

  it('shows validation errors when submitting an empty form', async () => {
    renderDialog();
    fireEvent.click(screen.getByRole('button', { name: /create client/i }));
    await waitFor(() => {
      expect(screen.getByText(/name must be at least 2 characters/i)).toBeInTheDocument();
      expect(screen.getByText(/slug must be at least 2 characters/i)).toBeInTheDocument();
    });
    expect(mutateAsync).not.toHaveBeenCalled();
  });

  it('rejects a slug that is not kebab-case', async () => {
    renderDialog();
    fireEvent.change(screen.getByLabelText(/name/i), { target: { value: 'Acme Corp' } });
    fireEvent.change(screen.getByLabelText(/slug/i), { target: { value: 'Acme_Corp' } });
    fireEvent.click(screen.getByRole('button', { name: /create client/i }));
    await waitFor(() => {
      expect(screen.getByText(/slug must be lowercase kebab-case/i)).toBeInTheDocument();
    });
    expect(mutateAsync).not.toHaveBeenCalled();
  });

  it('calls the provision mutation with valid input', async () => {
    renderDialog();
    fireEvent.change(screen.getByLabelText(/name/i), { target: { value: 'Acme Corp' } });
    fireEvent.change(screen.getByLabelText(/slug/i), { target: { value: 'acme-corp' } });
    fireEvent.click(screen.getByRole('button', { name: /create client/i }));
    await waitFor(() => {
      expect(mutateAsync).toHaveBeenCalledWith({
        orgId: 'org-123',
        body: { name: 'Acme Corp', slug: 'acme-corp' },
      });
    });
  });
});
