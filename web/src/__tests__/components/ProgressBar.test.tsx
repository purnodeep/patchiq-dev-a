import { render } from '@testing-library/react';
import { ProgressBar } from '../../components/ProgressBar';

describe('ProgressBar', () => {
  it('renders with correct width percentage', () => {
    const { container } = render(<ProgressBar value={75} max={100} />);
    const bar = container.querySelector('[role="progressbar"]');
    expect(bar).toBeInTheDocument();
    expect(bar).toHaveAttribute('aria-valuenow', '75');
  });

  it('renders 0% without error', () => {
    const { container } = render(<ProgressBar value={0} max={100} />);
    const bar = container.querySelector('[role="progressbar"]');
    expect(bar).toHaveAttribute('aria-valuenow', '0');
  });

  it('clamps to 100%', () => {
    const { container } = render(<ProgressBar value={150} max={100} />);
    const bar = container.querySelector('[role="progressbar"]');
    expect(bar).toHaveAttribute('aria-valuenow', '100');
  });
});
