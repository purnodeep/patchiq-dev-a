import { render, screen, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { QueryClientProvider, QueryClient } from '@tanstack/react-query';
import { NotificationsPage } from '../../../pages/notifications/NotificationsPage';
import { vi } from 'vitest';

vi.mock('../../../pages/notifications/PreferencesTab', () => ({
  PreferencesTab: () => <div data-testid="preferences-tab">Preferences Content</div>,
}));
vi.mock('../../../pages/notifications/HistoryTab', () => ({
  HistoryTab: () => <div data-testid="history-tab">History Content</div>,
}));

const wrapper = ({ children }: { children: React.ReactNode }) => (
  <MemoryRouter>
    <QueryClientProvider client={new QueryClient()}>{children}</QueryClientProvider>
  </MemoryRouter>
);

test('renders Preferences tab by default', () => {
  render(<NotificationsPage />, { wrapper });
  expect(screen.getByTestId('preferences-tab')).toBeInTheDocument();
});

test('switches to History tab on click', () => {
  render(<NotificationsPage />, { wrapper });
  fireEvent.click(screen.getByRole('tab', { name: /history/i }));
  expect(screen.getByTestId('history-tab')).toBeInTheDocument();
});
