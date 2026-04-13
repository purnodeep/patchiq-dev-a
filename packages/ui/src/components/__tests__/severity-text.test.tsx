import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { SeverityText } from '../severity-text';

describe('SeverityText', () => {
  it('renders capitalized severity text', () => {
    render(<SeverityText severity="critical" />);
    expect(screen.getByText('Critical')).toBeInTheDocument();
  });

  it('uses signal-critical color for critical', () => {
    render(<SeverityText severity="critical" />);
    expect(screen.getByText('Critical')).toHaveStyle({ color: 'var(--signal-critical)' });
  });

  it('uses signal-warning color for high', () => {
    render(<SeverityText severity="high" />);
    expect(screen.getByText('High')).toHaveStyle({ color: 'var(--signal-warning)' });
  });

  it('uses text-secondary color for medium', () => {
    render(<SeverityText severity="medium" />);
    expect(screen.getByText('Medium')).toHaveStyle({ color: 'var(--text-secondary)' });
  });

  it('uses text-secondary color for low', () => {
    render(<SeverityText severity="low" />);
    expect(screen.getByText('Low')).toHaveStyle({ color: 'var(--text-secondary)' });
  });

  it('handles unknown severity as text-secondary', () => {
    render(<SeverityText severity="unknown" />);
    expect(screen.getByText('Unknown')).toHaveStyle({ color: 'var(--text-secondary)' });
  });
});
