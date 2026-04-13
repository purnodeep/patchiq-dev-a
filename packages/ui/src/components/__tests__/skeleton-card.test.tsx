import { render } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { SkeletonCard } from '../skeleton-card';

describe('SkeletonCard', () => {
  it('renders default 3 shimmer lines', () => {
    const { container } = render(<SkeletonCard />);
    const lines = container.querySelectorAll('[data-slot="shimmer-line"]');
    expect(lines).toHaveLength(3);
  });

  it('renders custom number of lines', () => {
    const { container } = render(<SkeletonCard lines={5} />);
    const lines = container.querySelectorAll('[data-slot="shimmer-line"]');
    expect(lines).toHaveLength(5);
  });

  it('applies skeleton base color', () => {
    const { container } = render(<SkeletonCard />);
    const card = container.firstChild as HTMLElement;
    expect(card.style.backgroundColor).toBe('var(--skel-base)');
  });

  it('applies border-faint border', () => {
    const { container } = render(<SkeletonCard />);
    const card = container.firstChild as HTMLElement;
    expect(card.style.borderColor).toBe('var(--border-faint)');
  });
});
