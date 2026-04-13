import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { StatCardsRow1 } from '../StatCardsRow1';
import { DashboardDataProvider } from '../DashboardContext';
import type { DashboardSummary } from '../../../api/hooks/useDashboard';

const mockData: DashboardSummary = {
  total_endpoints: 120,
  active_endpoints: 98,
  endpoints_degraded: 3,
  total_patches: 500,
  critical_patches: 12,
  patches_high: 34,
  patches_medium: 87,
  patches_low: 210,
  total_cves: 55,
  critical_cves: 8,
  unpatched_cves: 20,
  pending_deployments: 4,
  compliance_rate: 87,
  active_deployments: [
    { id: 'd1', name: 'Deploy Alpha', status: 'running', progress_pct: 45 },
    { id: 'd2', name: 'Deploy Beta', status: 'created', progress_pct: 0 },
  ],
  overdue_sla_count: 3,
  failed_deployments_count: 2,
  failed_trend_7d: [1, 2, 1, 3, 2, 2, 2],
  workflows_running_count: 1,
  workflows_running: [{ id: 'w1', name: 'Patch Workflow', current_stage: 'verify' }],
  hub_sync_status: 'healthy',
  hub_last_sync_at: '2026-03-13T10:00:00Z',
  hub_url: 'https://hub.example.com',
  framework_count: 6,
};

function renderRow() {
  return render(
    <MemoryRouter>
      <DashboardDataProvider data={mockData}>
        <StatCardsRow1 data={mockData} />
      </DashboardDataProvider>
    </MemoryRouter>,
  );
}

describe('StatCardsRow1', () => {
  it('renders Endpoints Online card with correct value', () => {
    renderRow();
    // getAllByText because RingChart also renders the value in SVG text
    expect(screen.getAllByText('98').length).toBeGreaterThan(0);
    expect(screen.getByText('Endpoints Online')).toBeInTheDocument();
  });

  it('renders Critical Patches card with correct value', () => {
    renderRow();
    expect(screen.getAllByText('12').length).toBeGreaterThan(0);
    expect(screen.getByText('Critical Patches')).toBeInTheDocument();
  });

  it('renders Compliance Rate card with correct value', () => {
    renderRow();
    // getAllByText because GaugeChart also renders the value in SVG text
    expect(screen.getAllByText('87%').length).toBeGreaterThan(0);
    expect(screen.getByText('Compliance Rate')).toBeInTheDocument();
  });

  it('renders Active Deployments card with correct count', () => {
    renderRow();
    expect(screen.getAllByText('2').length).toBeGreaterThan(0);
    expect(screen.getByText('Active Deployments')).toBeInTheDocument();
  });
});
