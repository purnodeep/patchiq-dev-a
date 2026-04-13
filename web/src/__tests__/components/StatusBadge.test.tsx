import { render, screen } from '@testing-library/react';
import { StatusBadge } from '../../components/StatusBadge';

describe('StatusBadge', () => {
  it('renders the status text', () => {
    render(<StatusBadge status="active" />);
    expect(screen.getByText('active')).toBeInTheDocument();
  });

  it('applies correct color style for active status', () => {
    const { container } = render(<StatusBadge status="active" />);
    const badge = container.firstChild as HTMLElement;
    expect(badge.style.color).toBe('var(--signal-healthy)');
  });

  it('applies correct color style for pending status', () => {
    const { container } = render(<StatusBadge status="pending" />);
    const badge = container.firstChild as HTMLElement;
    expect(badge.style.color).toBe('var(--signal-warning)');
  });

  it('applies correct color style for inactive status', () => {
    const { container } = render(<StatusBadge status="inactive" />);
    const badge = container.firstChild as HTMLElement;
    expect(badge.style.color).toBe('var(--text-muted)');
  });

  it('applies correct color style for decommissioned status', () => {
    const { container } = render(<StatusBadge status="decommissioned" />);
    const badge = container.firstChild as HTMLElement;
    expect(badge.style.color).toBe('var(--signal-critical)');
  });
});
