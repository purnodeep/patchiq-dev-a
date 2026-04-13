import { describe, it, expect } from 'vitest';
import fs from 'fs';
import path from 'path';

describe('Design Tokens CSS', () => {
  const css = fs.readFileSync(path.resolve(__dirname, '../../theme/tokens.css'), 'utf-8');

  it('defines all surface tokens in :root (dark mode default)', () => {
    expect(css).toContain('--bg-page');
    expect(css).toContain('--bg-card');
    expect(css).toContain('--bg-card-hover');
    expect(css).toContain('--bg-elevated');
    expect(css).toContain('--bg-canvas');
    expect(css).toContain('--bg-inset');
  });

  it('defines all border tokens', () => {
    expect(css).toContain('--border:');
    expect(css).toContain('--border-hover');
    expect(css).toContain('--border-strong');
    expect(css).toContain('--border-faint');
  });

  it('defines all text tokens', () => {
    expect(css).toContain('--text-emphasis');
    expect(css).toContain('--text-primary');
    expect(css).toContain('--text-secondary');
    expect(css).toContain('--text-muted');
    expect(css).toContain('--text-faint');
  });

  it('defines signal colors that are theme-independent', () => {
    expect(css).toContain('--signal-healthy');
    expect(css).toContain('--signal-critical');
    expect(css).toContain('--signal-warning');
  });

  it('defines accent as CSS variable (user-configurable)', () => {
    expect(css).toContain('--accent');
    expect(css).toContain('--accent-subtle');
    expect(css).toContain('--accent-border');
  });

  it('defines light mode overrides under html.light', () => {
    expect(css).toContain('html.light');
    expect(css).toMatch(/html\.light[\s\S]*--bg-page:\s*#f5f5f5/);
  });

  it('defines font family tokens', () => {
    expect(css).toContain('--font-sans');
    expect(css).toContain('--font-mono');
    expect(css).toContain('Geist');
    expect(css).toContain('GeistMono');
  });

  it('defines spacing tokens', () => {
    expect(css).toContain('--space-xs');
    expect(css).toContain('--space-sm');
    expect(css).toContain('--space-md');
    expect(css).toContain('--space-lg');
    expect(css).toContain('--space-xl');
  });

  it('defines layout dimension tokens', () => {
    expect(css).toContain('--sidebar-width');
    expect(css).toContain('--topbar-height');
    expect(css).toContain('--sheet-width');
  });
});
