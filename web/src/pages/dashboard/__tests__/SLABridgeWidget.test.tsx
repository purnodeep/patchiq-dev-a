import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { SLABridgeWidget } from '../SLABridgeWidget';
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
  overdue_sla_count: 30,
  failed_deployments_count: 2,
  failed_trend_7d: [1, 2, 1, 3, 2, 2, 2],
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

describe('SLABridgeWidget', () => {
  it('renders title', () => {
    renderWithProvider(<SLABridgeWidget startingGap={30} deployed={22} newCves={8} />);
    expect(screen.getByText('SLA Bridge')).toBeInTheDocument();
  });

  it('renders waterfall stages', () => {
    renderWithProvider(<SLABridgeWidget startingGap={30} deployed={22} newCves={8} />);
    expect(screen.getByText('Start')).toBeInTheDocument();
    expect(screen.getByText('Deployed')).toBeInTheDocument();
    expect(screen.getByText('New CVEs')).toBeInTheDocument();
    expect(screen.getByText('Projected')).toBeInTheDocument();
  });
});
