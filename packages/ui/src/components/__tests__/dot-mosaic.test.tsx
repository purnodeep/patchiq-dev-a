import { render } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { DotMosaic } from '../dot-mosaic';

describe('DotMosaic', () => {
  const sampleData = [
    { id: '1', risk: 'critical' as const },
    { id: '2', risk: 'high' as const },
    { id: '3', risk: 'medium' as const },
    { id: '4', risk: 'healthy' as const },
  ];

  it('renders correct number of dots', () => {
    const { container } = render(<DotMosaic data={sampleData} />);
    const dots = container.querySelectorAll('[data-slot="dot"]');
    expect(dots).toHaveLength(4);
  });

  it('applies critical color to critical dots', () => {
    const { container } = render(<DotMosaic data={[{ id: '1', risk: 'critical' }]} />);
    const dot = container.querySelector('[data-slot="dot"]') as HTMLElement;
    expect(dot.style.backgroundColor).toBe('var(--signal-critical)');
  });

  it('applies warning color to high dots', () => {
    const { container } = render(<DotMosaic data={[{ id: '1', risk: 'high' }]} />);
    const dot = container.querySelector('[data-slot="dot"]') as HTMLElement;
    expect(dot.style.backgroundColor).toBe('var(--signal-warning)');
  });

  it('applies muted color to medium dots', () => {
    const { container } = render(<DotMosaic data={[{ id: '1', risk: 'medium' }]} />);
    const dot = container.querySelector('[data-slot="dot"]') as HTMLElement;
    expect(dot.style.backgroundColor).toBe('var(--text-muted)');
  });

  it('applies healthy color to healthy dots', () => {
    const { container } = render(<DotMosaic data={[{ id: '1', risk: 'healthy' }]} />);
    const dot = container.querySelector('[data-slot="dot"]') as HTMLElement;
    expect(dot.style.backgroundColor).toBe('var(--signal-healthy)');
  });

  it('renders empty when data is empty', () => {
    const { container } = render(<DotMosaic data={[]} />);
    const dots = container.querySelectorAll('[data-slot="dot"]');
    expect(dots).toHaveLength(0);
  });
});
