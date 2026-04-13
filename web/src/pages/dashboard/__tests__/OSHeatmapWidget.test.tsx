import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { MemoryRouter } from 'react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { OSHeatmapWidget } from '../OSHeatmapWidget';

vi.mock('@/api/hooks/useDashboard', () => ({
  useTopEndpointsByRisk: () => ({
    data: [
      { hostname: 'prod-01', cve_count: 7, risk_score: 95 },
      { hostname: 'db-main', cve_count: 3, risk_score: 50 },
    ],
    isLoading: false,
    error: null,
    refetch: vi.fn(),
  }),
}));

const qc = new QueryClient();

describe('OSHeatmapWidget', () => {
  it('renders title', () => {
    render(
      <QueryClientProvider client={qc}>
        <MemoryRouter>
          <OSHeatmapWidget />
        </MemoryRouter>
      </QueryClientProvider>,
    );
    expect(screen.getByText('Risk Heatmap')).toBeInTheDocument();
  });

  it('renders endpoint cells', () => {
    render(
      <QueryClientProvider client={qc}>
        <MemoryRouter>
          <OSHeatmapWidget />
        </MemoryRouter>
      </QueryClientProvider>,
    );
    // Hostnames <= 8 chars are not truncated
    expect(screen.getByText('prod-01')).toBeInTheDocument();
    expect(screen.getByText('db-main')).toBeInTheDocument();
  });
});
