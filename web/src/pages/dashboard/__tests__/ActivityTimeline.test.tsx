import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router';
import { ActivityTimeline } from '../ActivityTimeline';

vi.mock('@/api/hooks/useDashboard', () => ({
  useDashboardActivity: () => ({
    data: [
      {
        id: '1',
        type: 'deployment',
        title: 'Windows Critical Batch',
        status: 'running',
        meta: '142/180 endpoints',
        timestamp: '2026-03-13T10:22:00Z',
      },
      {
        id: '2',
        type: 'deployment',
        title: 'Ubuntu Security Patch',
        status: 'completed',
        meta: '68/68 endpoints',
        timestamp: '2026-03-13T08:00:00Z',
      },
    ],
    isLoading: false,
    error: null,
  }),
}));

const qc = new QueryClient();

describe('ActivityTimeline', () => {
  it('renders activity items', () => {
    render(
      <QueryClientProvider client={qc}>
        <MemoryRouter>
          <ActivityTimeline />
        </MemoryRouter>
      </QueryClientProvider>,
    );
    expect(screen.getByText('Windows Critical Batch')).toBeInTheDocument();
    expect(screen.getByText('Ubuntu Security Patch')).toBeInTheDocument();
  });

  it('shows status indicators', () => {
    render(
      <QueryClientProvider client={qc}>
        <MemoryRouter>
          <ActivityTimeline />
        </MemoryRouter>
      </QueryClientProvider>,
    );
    expect(screen.getByText('142/180 endpoints')).toBeInTheDocument();
  });
});
