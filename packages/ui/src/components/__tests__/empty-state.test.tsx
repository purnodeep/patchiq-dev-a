import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi } from 'vitest';
import { EmptyState } from '../empty-state';

describe('EmptyState', () => {
  it('renders title', () => {
    render(<EmptyState title="No endpoints" />);
    expect(screen.getByText('No endpoints')).toBeInTheDocument();
  });

  it('renders description when provided', () => {
    render(<EmptyState title="Empty" description="Add your first endpoint" />);
    expect(screen.getByText('Add your first endpoint')).toBeInTheDocument();
  });

  it('renders icon when provided', () => {
    render(<EmptyState title="Empty" icon={<span data-testid="icon">X</span>} />);
    expect(screen.getByTestId('icon')).toBeInTheDocument();
  });

  it('renders action button and handles click', async () => {
    const user = userEvent.setup();
    const onClick = vi.fn();
    render(<EmptyState title="Empty" action={{ label: 'Create', onClick }} />);
    const btn = screen.getByText('Create');
    expect(btn).toBeInTheDocument();
    await user.click(btn);
    expect(onClick).toHaveBeenCalledOnce();
  });

  it('does not render action button when not provided', () => {
    const { container } = render(<EmptyState title="Empty" />);
    expect(container.querySelector('button')).toBeNull();
  });

  it('renders action as a Button component', () => {
    const { container } = render(
      <EmptyState title="Empty" action={{ label: 'Go', onClick: () => {} }} />,
    );
    expect(container.querySelector('button')).toBeInTheDocument();
    expect(screen.getByText('Go')).toBeInTheDocument();
  });
});
