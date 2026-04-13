import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { MonoTag } from '../mono-tag';

describe('MonoTag', () => {
  it('renders children text', () => {
    render(<MonoTag>env:production</MonoTag>);
    expect(screen.getByText('env:production')).toBeInTheDocument();
  });

  it('applies monochrome styling', () => {
    render(<MonoTag>os:linux</MonoTag>);
    const el = screen.getByText('os:linux');
    expect(el).toHaveStyle({ backgroundColor: 'var(--bg-card-hover)' });
    expect(el).toHaveStyle({ color: 'var(--text-secondary)' });
    // borderColor is set via individual style properties; verify via style object
    expect(el.style.borderColor).toBe('var(--border-strong)');
  });
});
