import { render, screen } from '@testing-library/react';
import { DeploymentStatusBadge } from '../../components/DeploymentStatusBadge';

describe('DeploymentStatusBadge', () => {
  it('renders running status with pulse', () => {
    const { container } = render(<DeploymentStatusBadge status="running" />);
    expect(screen.getByText('running')).toBeInTheDocument();
    // pulse is implemented via inline animation style on the dot span
    const dot = container.querySelector('span > span') as HTMLElement;
    expect(dot).toBeInTheDocument();
    expect(dot.style.animation).toMatch(/pulse/);
  });

  it('renders completed status without pulse', () => {
    const { container } = render(<DeploymentStatusBadge status="completed" />);
    expect(screen.getByText('completed')).toBeInTheDocument();
    const dot = container.querySelector('span > span') as HTMLElement;
    expect(dot).toBeInTheDocument();
    expect(dot.style.animation).toBe('');
  });

  it('renders all statuses', () => {
    const statuses = [
      'created',
      'scheduled',
      'running',
      'completed',
      'failed',
      'cancelled',
      'rolling_back',
      'rolled_back',
      'rollback_failed',
    ] as const;
    for (const status of statuses) {
      const { unmount } = render(<DeploymentStatusBadge status={status} />);
      expect(screen.getByText(status)).toBeInTheDocument();
      unmount();
    }
  });
});
