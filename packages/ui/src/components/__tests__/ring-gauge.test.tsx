import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { RingGauge } from '../ring-gauge';

describe('RingGauge', () => {
  it('renders the percentage value', () => {
    render(<RingGauge value={75} />);
    const svg = screen.getByRole('img');
    expect(svg).toHaveAttribute('aria-label', '75%');
  });

  it('renders a label when provided', () => {
    render(<RingGauge value={50} label="Compliance" />);
    expect(screen.getByText('Compliance')).toBeInTheDocument();
  });

  it('clamps value to 0-100', () => {
    render(<RingGauge value={150} />);
    expect(screen.getByRole('img')).toHaveAttribute('aria-label', '100%');
  });

  it('clamps negative values to 0', () => {
    render(<RingGauge value={-10} />);
    expect(screen.getByRole('img')).toHaveAttribute('aria-label', '0%');
  });

  it('uses accent color by default for fill stroke', () => {
    const { container } = render(<RingGauge value={60} />);
    const circles = container.querySelectorAll('circle');
    // Second circle is the fill
    expect(circles[1]).toHaveAttribute('stroke', 'var(--accent)');
  });

  it('uses signal-healthy color when colorByValue and value >= 80', () => {
    const { container } = render(<RingGauge value={85} colorByValue />);
    const circles = container.querySelectorAll('circle');
    expect(circles[1]).toHaveAttribute('stroke', 'var(--signal-healthy)');
  });

  it('uses signal-warning color when colorByValue and value >= 50', () => {
    const { container } = render(<RingGauge value={60} colorByValue />);
    const circles = container.querySelectorAll('circle');
    expect(circles[1]).toHaveAttribute('stroke', 'var(--signal-warning)');
  });

  it('uses signal-critical color when colorByValue and value < 50', () => {
    const { container } = render(<RingGauge value={30} colorByValue />);
    const circles = container.querySelectorAll('circle');
    expect(circles[1]).toHaveAttribute('stroke', 'var(--signal-critical)');
  });

  it('uses track stroke of var(--ring-track)', () => {
    const { container } = render(<RingGauge value={50} />);
    const circles = container.querySelectorAll('circle');
    expect(circles[0]).toHaveAttribute('stroke', 'var(--ring-track)');
  });

  it('respects custom size prop', () => {
    const { container } = render(<RingGauge value={50} size={120} />);
    const svg = container.querySelector('svg');
    expect(svg).toHaveAttribute('width', '120');
    expect(svg).toHaveAttribute('height', '120');
  });
});
