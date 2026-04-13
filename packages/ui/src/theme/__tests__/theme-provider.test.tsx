import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ThemeProvider, useTheme } from '../theme-provider';

function TestConsumer() {
  const { mode, accent, setMode, setAccent } = useTheme();
  return (
    <div>
      <span data-testid="mode">{mode}</span>
      <span data-testid="accent">{accent}</span>
      <button onClick={() => setMode('light')}>Light</button>
      <button onClick={() => setAccent('#3b82f6')}>Ocean</button>
    </div>
  );
}

describe('ThemeProvider', () => {
  beforeEach(() => {
    document.documentElement.classList.remove('light', 'dark');
    localStorage.clear();
  });

  it('defaults to dark mode and adds dark class', () => {
    render(
      <ThemeProvider>
        <TestConsumer />
      </ThemeProvider>,
    );
    expect(screen.getByTestId('mode')).toHaveTextContent('dark');
    expect(document.documentElement.classList.contains('dark')).toBe(true);
    expect(document.documentElement.classList.contains('light')).toBe(false);
  });

  it('applies light class and removes dark class when switched', async () => {
    const user = userEvent.setup();
    render(
      <ThemeProvider>
        <TestConsumer />
      </ThemeProvider>,
    );
    await user.click(screen.getByText('Light'));
    expect(document.documentElement.classList.contains('light')).toBe(true);
    expect(document.documentElement.classList.contains('dark')).toBe(false);
  });

  it('updates accent CSS variable', async () => {
    const user = userEvent.setup();
    render(
      <ThemeProvider>
        <TestConsumer />
      </ThemeProvider>,
    );
    await user.click(screen.getByText('Ocean'));
    expect(screen.getByTestId('accent')).toHaveTextContent('#3b82f6');
    expect(document.documentElement.style.getPropertyValue('--accent')).toBe('#3b82f6');
  });

  it('persists mode to localStorage', async () => {
    const user = userEvent.setup();
    render(
      <ThemeProvider>
        <TestConsumer />
      </ThemeProvider>,
    );
    await user.click(screen.getByText('Light'));
    expect(localStorage.getItem('patchiq-theme-mode')).toBe('light');
  });

  it('persists accent to localStorage', async () => {
    const user = userEvent.setup();
    render(
      <ThemeProvider>
        <TestConsumer />
      </ThemeProvider>,
    );
    await user.click(screen.getByText('Ocean'));
    expect(localStorage.getItem('patchiq-theme-accent')).toBe('#3b82f6');
  });
});
