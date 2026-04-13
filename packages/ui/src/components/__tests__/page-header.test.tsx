import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { PageHeader } from '../page-header';

describe('PageHeader', () => {
  it('renders title', () => {
    render(<PageHeader title="Endpoints" />);
    expect(screen.getByText('Endpoints')).toBeInTheDocument();
  });

  it('renders subtitle when provided', () => {
    render(<PageHeader title="Endpoints" subtitle="Manage your fleet" />);
    expect(screen.getByText('Manage your fleet')).toBeInTheDocument();
  });

  it('renders actions slot', () => {
    render(<PageHeader title="Test" actions={<button data-testid="action-btn">Add</button>} />);
    expect(screen.getByTestId('action-btn')).toBeInTheDocument();
  });

  it('renders breadcrumbs when provided', () => {
    render(
      <PageHeader title="Test" breadcrumbs={<nav data-testid="breadcrumbs">Home / Test</nav>} />,
    );
    expect(screen.getByTestId('breadcrumbs')).toBeInTheDocument();
  });

  it('applies text-emphasis color to title', () => {
    render(<PageHeader title="Endpoints" />);
    expect(screen.getByText('Endpoints')).toHaveStyle({ color: 'var(--text-emphasis)' });
  });

  it('does not render subtitle when not provided', () => {
    const { container } = render(<PageHeader title="Test" />);
    expect(container.querySelector('p')).toBeNull();
  });
});
