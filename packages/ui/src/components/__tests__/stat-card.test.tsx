import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { StatCard } from '../stat-card';

describe('StatCard', () => {
  it('renders label and value', () => {
    render(<StatCard label="Endpoints" value={42} />);
    expect(screen.getByText('Endpoints')).toBeInTheDocument();
    expect(screen.getByText('42')).toBeInTheDocument();
  });

  it('renders string values', () => {
    render(<StatCard label="Status" value="Active" />);
    expect(screen.getByText('Active')).toBeInTheDocument();
  });

  it('renders icon when provided', () => {
    render(<StatCard label="Test" value={0} icon={<span data-testid="icon">I</span>} />);
    expect(screen.getByTestId('icon')).toBeInTheDocument();
  });

  it('renders upward trend with healthy color', () => {
    render(<StatCard label="Test" value={100} trend={{ value: 12, direction: 'up' }} />);
    const trend = screen.getByText('+12%');
    expect(trend).toBeInTheDocument();
    expect(trend).toHaveStyle({ color: 'var(--signal-healthy)' });
  });

  it('renders downward trend with critical color', () => {
    render(<StatCard label="Test" value={100} trend={{ value: 5, direction: 'down' }} />);
    const trend = screen.getByText('-5%');
    expect(trend).toHaveStyle({ color: 'var(--signal-critical)' });
  });

  it('renders flat trend with muted color', () => {
    render(<StatCard label="Test" value={100} trend={{ value: 0, direction: 'flat' }} />);
    const trend = screen.getByText('0%');
    expect(trend).toHaveStyle({ color: 'var(--text-muted)' });
  });

  it('applies card styling via inline styles', () => {
    const { container } = render(<StatCard label="Test" value={0} />);
    const card = container.firstChild as HTMLElement;
    expect(card.style.backgroundColor).toBe('var(--bg-card)');
    expect(card.style.borderColor).toBe('var(--border)');
  });
});
