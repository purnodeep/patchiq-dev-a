import { describe, it, expect } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '../render';

describe('renderWithProviders', () => {
  it('wraps component in ThemeProvider + QueryClient + Router', () => {
    renderWithProviders(<div data-testid="test">Hello</div>);
    expect(screen.getByTestId('test')).toBeInTheDocument();
  });

  it('supports custom initial route', () => {
    renderWithProviders(<div>routed</div>, { initialRoute: '/endpoints' });
    // No error = success (MemoryRouter accepts the route)
  });
});
