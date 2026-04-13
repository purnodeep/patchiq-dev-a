import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { RiskProjectionWidget } from '../RiskProjectionWidget';
import { DashboardDataProvider } from '../DashboardContext';
import type { DashboardSummary } from '../../../api/hooks/useDashboard';

const mockData: DashboardSummary = {
  total_endpoints: 100,
  active_endpoints: 80,
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
  compliance_rate: 87,
  active_deployments: [],
  overdue_sla_count: 3,
  failed_deployments_count: 2,
  failed_trend_7d: [3, 2, 1, 0, 1, 2, 2],
  workflows_running_count: 0,
  workflows_running: [],
  hub_sync_status: 'healthy',
  hub_last_sync_at: null,
  hub_url: '',
  framework_count: 6,
};

function renderWithProvider(ui: React.ReactElement) {
  return render(<DashboardDataProvider data={mockData}>{ui}</DashboardDataProvider>);
}

describe('RiskProjectionWidget', () => {
  it('renders chart title', () => {
    renderWithProvider(
      <RiskProjectionWidget complianceRate={87.3} failedTrend7d={[3, 2, 1, 0, 1, 2, 2]} />,
    );
    expect(screen.getByText('Risk Delta Projection')).toBeInTheDocument();
  });

  it('renders scenario badges', () => {
    renderWithProvider(
      <RiskProjectionWidget complianceRate={87.3} failedTrend7d={[3, 2, 1, 0, 1, 2, 2]} />,
    );
    expect(screen.getByText('Deploy All')).toBeInTheDocument();
    expect(screen.getByText('Trajectory')).toBeInTheDocument();
    expect(screen.getByText('Do Nothing')).toBeInTheDocument();
  });
});
