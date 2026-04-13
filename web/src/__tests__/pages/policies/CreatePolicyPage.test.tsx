import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { CreatePolicyPage } from '../../../pages/policies/CreatePolicyPage';

const mockCreate = vi.fn().mockResolvedValue({ id: 'new-id' });

vi.mock('../../../api/hooks/usePolicies', () => ({
  useCreatePolicy: () => ({ mutateAsync: mockCreate, isPending: false }),
}));

vi.mock('../../../api/hooks/useTagKeys', () => ({
  useTagKeys: () => ({ data: [], isLoading: false }),
  useDistinctTagKeys: () => ({ data: ['env', 'role'], isLoading: false }),
  useUpsertTagKey: () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }),
  useDeleteTagKey: () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }),
}));

vi.mock('../../../api/hooks/useTagSelector', () => ({
  useValidateSelector: () => ({
    data: { valid: true, matched_count: 0 },
    isFetching: false,
  }),
}));

const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

const renderPage = () =>
  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <CreatePolicyPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );

describe('CreatePolicyPage', () => {
  beforeEach(() => mockCreate.mockClear());

  it('renders page title', () => {
    renderPage();
    expect(screen.getByRole('heading', { name: 'Create Policy' })).toBeInTheDocument();
  });

  it('renders name field', () => {
    renderPage();
    // Label text is "Name *" (with asterisk), so use the input id directly
    expect(screen.getByRole('textbox', { name: /name/i })).toBeInTheDocument();
  });

  it('renders selection mode options', () => {
    renderPage();
    // Radio options: "All Available Patches" and "By Severity"
    expect(screen.getByLabelText(/all available patches/i)).toBeInTheDocument();
    expect(screen.getByLabelText('By Severity')).toBeInTheDocument();
  });

  it('shows severity dropdown when by_severity selected', async () => {
    renderPage();
    fireEvent.click(screen.getByLabelText('By Severity'));
    await waitFor(() => {
      expect(screen.getByLabelText(/minimum severity/i)).toBeInTheDocument();
    });
  });

  it('shows validation error when name empty', async () => {
    renderPage();
    fireEvent.click(screen.getByRole('button', { name: /create policy/i }));
    await waitFor(() => {
      expect(screen.getByText(/name is required/i)).toBeInTheDocument();
    });
  });
});
