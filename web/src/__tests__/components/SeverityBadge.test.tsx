import { render, screen } from '@testing-library/react';
import { SeverityBadge } from '../../components/SeverityBadge';

describe('SeverityBadge', () => {
  it('renders severity text', () => {
    render(<SeverityBadge severity="critical" />);
    expect(screen.getByText('critical')).toBeInTheDocument();
  });

  it('applies red color for critical', () => {
    const { container } = render(<SeverityBadge severity="critical" />);
    const badge = container.firstChild as HTMLElement;
    expect(badge.style.color).toBe('var(--signal-critical)');
  });

  it('applies warning color for high', () => {
    const { container } = render(<SeverityBadge severity="high" />);
    const badge = container.firstChild as HTMLElement;
    expect(badge.style.color).toBe('var(--signal-warning)');
  });

  it('applies muted color for none', () => {
    const { container } = render(<SeverityBadge severity="none" />);
    const badge = container.firstChild as HTMLElement;
    expect(badge.style.color).toBe('var(--text-muted)');
  });
});
