import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { DashboardAlertBanner } from '../DashboardAlertBanner';

describe('DashboardAlertBanner', () => {
  it('renders message when overdueCount > 0', () => {
    render(<DashboardAlertBanner overdueCount={3} onDismiss={vi.fn()} />);
    expect(screen.getByRole('alert')).toBeInTheDocument();
    expect(screen.getByText(/3 deployments overdue SLA/i)).toBeInTheDocument();
  });

  it('renders singular form when overdueCount is 1', () => {
    render(<DashboardAlertBanner overdueCount={1} onDismiss={vi.fn()} />);
    expect(screen.getByText(/1 deployment overdue SLA/i)).toBeInTheDocument();
  });

  it('renders nothing when overdueCount is 0', () => {
    const { container } = render(<DashboardAlertBanner overdueCount={0} onDismiss={vi.fn()} />);
    expect(container.firstChild).toBeNull();
  });
});
