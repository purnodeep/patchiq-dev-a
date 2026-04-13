import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { StatCardsRow2 } from '../StatCardsRow2';
import { DashboardDataProvider } from '../DashboardContext';
import type { DashboardSummary } from '../../../api/hooks/useDashboard';

const mockSummary: DashboardSummary = {
  total_endpoints: 100,
  active_endpoints: 82,
  endpoints_degraded: 2,
  total_patches: 400,
  critical_patches: 9,
  patches_high: 28,
  patches_medium: 60,
  patches_low: 180,
  total_cves: 40,
  critical_cves: 5,
  unpatched_cves: 15,
  pending_deployments: 3,
  compliance_rate: 91,
  active_deployments: [{ id: 'dep1', name: 'Prod Deploy', status: 'running', progress_pct: 70 }],
  overdue_sla_count: 5,
  failed_deployments_count: 3,
  failed_trend_7d: [0, 1, 2, 1, 3, 2, 3],
  workflows_running_count: 2,
  workflows_running: [{ id: 'wf1', name: 'Security Workflow', current_stage: 'scan' }],
  hub_sync_status: 'healthy',
  hub_last_sync_at: '2026-03-13T09:00:00Z',
  hub_url: 'https://hub.patchiq.io',
  framework_count: 6,
};

function renderRow(overrides?: Partial<DashboardSummary>) {
  const data = { ...mockSummary, ...overrides };
  return render(
    <MemoryRouter>
      <DashboardDataProvider data={data}>
        <StatCardsRow2 data={data} />
      </DashboardDataProvider>
    </MemoryRouter>,
  );
}

describe('StatCardsRow2', () => {
  it('renders Overdue SLA label', () => {
    renderRow();
    expect(screen.getByText('Overdue SLA')).toBeInTheDocument();
  });

  it('renders Failed Deployments label', () => {
    renderRow();
    expect(screen.getByText('Failed Deployments')).toBeInTheDocument();
  });

  it('renders Workflows Running label', () => {
    renderRow();
    expect(screen.getByText('Workflows Running')).toBeInTheDocument();
  });

  it('renders Hub Sync Status label', () => {
    renderRow();
    expect(screen.getByText('Hub Sync Status')).toBeInTheDocument();
  });

  it('renders "Sync\'d" for healthy hub status', () => {
    renderRow({ hub_sync_status: 'healthy' });
    expect(screen.getByText("Sync'd")).toBeInTheDocument();
  });

  it('renders "Sync\'d" for idle hub status', () => {
    renderRow({ hub_sync_status: 'idle' });
    expect(screen.getByText("Sync'd")).toBeInTheDocument();
  });

  it('renders raw status when hub status is not healthy/idle', () => {
    renderRow({ hub_sync_status: 'error' });
    expect(screen.getByText('error')).toBeInTheDocument();
  });
});
