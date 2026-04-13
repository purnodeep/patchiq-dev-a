import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, beforeEach } from 'vitest';
import { ThemeConfigurator } from '../theme-configurator';
import { ThemeProvider } from '../../theme';

function renderWithTheme(ui: React.ReactElement) {
  return render(<ThemeProvider>{ui}</ThemeProvider>);
}

describe('ThemeConfigurator', () => {
  beforeEach(() => {
    localStorage.clear();
    // Reset document classes
    document.documentElement.classList.remove('light', 'dark');
  });

  it('renders mode picker with 3 options', () => {
    renderWithTheme(<ThemeConfigurator />);
    expect(screen.getByText('Dark')).toBeInTheDocument();
    expect(screen.getByText('Light')).toBeInTheDocument();
    expect(screen.getByText('System')).toBeInTheDocument();
  });

  it('renders accent color presets', () => {
    renderWithTheme(<ThemeConfigurator />);
    const presetButtons = screen
      .getAllByRole('button')
      .filter((btn) => btn.getAttribute('aria-label')?.includes('accent color'));
    expect(presetButtons.length).toBe(8);
  });

  it('renders density toggle', () => {
    renderWithTheme(<ThemeConfigurator />);
    expect(screen.getByText('comfortable')).toBeInTheDocument();
    expect(screen.getByText('compact')).toBeInTheDocument();
  });

  it('switches theme mode on click', async () => {
    const user = userEvent.setup();
    renderWithTheme(<ThemeConfigurator />);
    await user.click(screen.getByText('Light'));
    expect(localStorage.getItem('patchiq-theme-mode')).toBe('light');
  });

  it('renders custom hex input', () => {
    renderWithTheme(<ThemeConfigurator />);
    const input = screen.getByPlaceholderText('#10b981');
    expect(input).toBeInTheDocument();
  });

  it('stores density selection in localStorage', async () => {
    const user = userEvent.setup();
    renderWithTheme(<ThemeConfigurator />);
    await user.click(screen.getByText('compact'));
    expect(localStorage.getItem('patchiq-density')).toBe('compact');
  });
});
