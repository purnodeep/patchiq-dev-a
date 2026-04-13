import { render, screen, fireEvent, act } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { vi, describe, it, expect } from 'vitest';
import { DeploymentModal } from '../../../pages/patches/DeploymentModal';

vi.mock('../../../api/hooks/usePatchDeploy', () => ({
  usePatchDeploy: () => ({ mutate: vi.fn(), isPending: false }),
}));

const wrapper = ({ children }: { children: React.ReactNode }) => (
  <QueryClientProvider client={new QueryClient()}>{children}</QueryClientProvider>
);

describe('DeploymentModal', () => {
  it('renders all required fields', () => {
    render(<DeploymentModal open={true} patchId="p1" patchName="KB5034441" onClose={vi.fn()} />, {
      wrapper,
    });
    expect(screen.getByLabelText(/deployment name/i)).toBeInTheDocument();
    expect(screen.getByText(/configuration type/i)).toBeInTheDocument();
    expect(screen.getByText(/target endpoints/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /publish/i })).toBeInTheDocument();
  });

  it('shows validation error when name is empty on submit', async () => {
    render(<DeploymentModal open={true} patchId="p1" patchName="KB5034441" onClose={vi.fn()} />, {
      wrapper,
    });
    const nameInput = screen.getByLabelText(/deployment name/i);
    fireEvent.change(nameInput, { target: { value: '' } });
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: /publish/i }));
    });
    expect(screen.getByText(/name is required/i)).toBeInTheDocument();
  });
});
