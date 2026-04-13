import { render, screen, fireEvent } from '@testing-library/react';
import { QueryClientProvider, QueryClient } from '@tanstack/react-query';
import { PreferencesTab } from '../../../pages/notifications/PreferencesTab';
import { vi } from 'vitest';

vi.mock('../../../api/hooks/useNotifications', () => ({
  useNotificationPreferences: () => ({
    data: {
      categories: [
        {
          id: 'deployments',
          label: 'Deployments',
          description: 'Patch deployment lifecycle events',
          events: [
            {
              trigger_type: 'deployment.started',
              label: 'Deployment Started',
              email_enabled: false,
              slack_enabled: true,
              webhook_enabled: false,
              urgency: 'digest',
            },
          ],
        },
      ],
      channels: [
        { type: 'email', configured: true, channel_id: 'ch-1' },
        { type: 'slack', configured: true, channel_id: 'ch-2' },
        { type: 'webhook', configured: false },
      ],
    },
    isLoading: false,
  }),
  useDigestConfig: () => ({
    data: { frequency: 'daily', delivery_time: '09:00', format: 'html' },
    isLoading: false,
  }),
  useUpdatePreferences: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateDigestConfig: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useTestDigest: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useNotificationChannels: () => ({
    data: [],
    isLoading: false,
  }),
  useTestChannel: () => ({
    mutateAsync: vi.fn(),
    isPending: false,
  }),
}));

const wrapper = ({ children }: { children: React.ReactNode }) => (
  <QueryClientProvider client={new QueryClient()}>{children}</QueryClientProvider>
);

test('renders Deployments category card', () => {
  render(<PreferencesTab />, { wrapper });
  expect(screen.getByText('Deployments')).toBeInTheDocument();
});

test('expands category card on click', () => {
  render(<PreferencesTab />, { wrapper });
  const trigger = screen.getByText('Deployments');
  fireEvent.click(trigger);
  expect(screen.getByText('Deployment Started')).toBeInTheDocument();
});

test('renders Email/Slack/Webhook toggle columns', () => {
  render(<PreferencesTab />, { wrapper });
  expect(screen.getAllByText('Email').length).toBeGreaterThanOrEqual(1);
  expect(screen.getAllByText('Slack').length).toBeGreaterThanOrEqual(1);
  expect(screen.getAllByText('Webhook').length).toBeGreaterThanOrEqual(1);
});

test('renders digest configuration section', () => {
  render(<PreferencesTab />, { wrapper });
  expect(screen.getByText('Digest Configuration')).toBeInTheDocument();
});
