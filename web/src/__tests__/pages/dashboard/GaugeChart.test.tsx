import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { GaugeChart } from '../../../pages/dashboard/GaugeChart';

describe('GaugeChart', () => {
  it('renders the value as percentage text', () => {
    render(<GaugeChart value={75} />);
    expect(screen.getByText('75%')).toBeDefined();
  });

  it('clamps value above 100 to 100', () => {
    render(<GaugeChart value={150} />);
    expect(screen.getByText('100%')).toBeDefined();
  });

  it('clamps value below 0 to 0', () => {
    render(<GaugeChart value={-10} />);
    expect(screen.getByText('0%')).toBeDefined();
  });

  it('renders an SVG element', () => {
    const { container } = render(<GaugeChart value={50} />);
    expect(container.querySelector('svg')).not.toBeNull();
  });

  it('renders the optional label', () => {
    render(<GaugeChart value={42} label="Compliance" />);
    expect(screen.getByText('Compliance')).toBeDefined();
  });
});
