import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi } from 'vitest';
import { ErrorState } from '../error-state';

describe('ErrorState', () => {
  it('renders default title and message', () => {
    render(<ErrorState message="Network error" />);
    expect(screen.getByText('Something went wrong')).toBeInTheDocument();
    expect(screen.getByText('Network error')).toBeInTheDocument();
  });

  it('renders custom title', () => {
    render(<ErrorState title="Connection failed" message="Timeout" />);
    expect(screen.getByText('Connection failed')).toBeInTheDocument();
  });

  it('renders retry button when onRetry provided', async () => {
    const user = userEvent.setup();
    const onRetry = vi.fn();
    render(<ErrorState message="Error" onRetry={onRetry} />);
    const btn = screen.getByText('Retry');
    await user.click(btn);
    expect(onRetry).toHaveBeenCalledOnce();
  });

  it('does not render retry button when onRetry not provided', () => {
    const { container } = render(<ErrorState message="Error" />);
    expect(container.querySelector('button')).toBeNull();
  });
});
