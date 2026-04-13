// prototype-ui/src/pages/pm/DashboardGrid.tsx
import { useEffect, useState } from 'react';
import ReactGridLayout, { WidthProvider } from 'react-grid-layout/legacy';
import type { LayoutItem } from 'react-grid-layout/legacy';
import 'react-grid-layout/css/styles.css';
import 'react-resizable/css/styles.css';

import { WidgetShell } from '@/components/shared/WidgetShell';
import { AlertBanner } from '@/components/shared/AlertBanner';
import { StatCard } from '@/components/shared/StatCard';
import { SectionHeader } from '@/components/shared/SectionHeader';
import { DASHBOARD_STATS, ALERT_MESSAGES } from '@/data/mock-data';
import { resolveIcon, buildMicroViz } from './DashboardHelpers';
import { RiskDeltaChart } from '@/components/charts/RiskDeltaChart';
import { BlastRadiusGraph } from '@/components/charts/BlastRadiusGraph';
import { SLABridgeWaterfall } from '@/components/charts/SLABridgeWaterfall';
import { DeploymentTimeline } from '@/components/charts/DeploymentTimeline';
import { VulnHeatmapGrid } from '@/components/charts/VulnHeatmapGrid';
import ComplianceRings from '@/components/charts/ComplianceRings';
import AgentRolloutSankey from '@/components/charts/AgentRolloutSankey';
import SLACountdownTimers from '@/components/charts/SLACountdownTimers';
import { WorkflowPipeline } from '@/components/charts/WorkflowPipeline';
import { PatchesHorizon } from '@/components/charts/PatchesHorizon';
import { TopCriticalCVEs } from '@/components/charts/TopCriticalCVEs';
import { MeanTimeToPatch } from '@/components/charts/MeanTimeToPatch';
import { PatchSuccessRate } from '@/components/charts/PatchSuccessRate';
import { CVEAgeDistribution } from '@/components/charts/CVEAgeDistribution';
import { EndpointCoverageMap } from '@/components/charts/EndpointCoverageMap';
import { PatchFailureReasons } from '@/components/charts/PatchFailureReasons';
import { UpcomingSLADeadlines } from '@/components/charts/UpcomingSLADeadlines';

const ResponsiveGridLayout = WidthProvider(ReactGridLayout);

interface DashboardGridProps {
  layout: LayoutItem[];
  isEditMode: boolean;
  alertDismissed: boolean;
  onLayoutChange: (layout: LayoutItem[]) => void;
  onAlertDismiss: () => void;
}

const STAT_TITLES = [
  'Endpoints Online',
  'Critical Patches',
  'Compliance Rate',
  'Active Deployments',
  'Overdue SLA',
  'Failed Deployments',
  'Workflows Running',
  'Hub Sync',
];

const WIDGET_TITLES: Record<string, string> = {
  'blast-radius': 'Blast Radius',
  'risk-delta': 'Risk Delta Projection',
  'sla-waterfall': 'SLA Bridge Waterfall',
  'deployment-timeline': 'Deployment Timeline',
  'vuln-heatmap': 'Vulnerability Heatmap',
  'compliance-rings': 'Compliance Gauges',
  'agent-rollout': 'Agent Rollout Pipeline',
  'sla-countdown': 'SLA Countdown',
  'workflow-pipeline': 'Workflow Pipeline',
  'patches-horizon': 'Critical Patches Horizon',
  'top-cves': 'Top Critical CVEs',
  mttp: 'Mean Time to Patch',
  'patch-success-rate': 'Patch Success Rate',
  'cve-age': 'CVE Age Distribution',
  'endpoint-coverage': 'Endpoint Coverage',
  'patch-failure-reasons': 'Patch Failure Reasons',
  'upcoming-sla': 'Upcoming SLA Deadlines',
};

function WidgetContent({ id }: { id: string }) {
  if (id.startsWith('stat-')) {
    const idx = parseInt(id.replace('stat-', ''), 10);
    const stat = DASHBOARD_STATS[idx];
    if (!stat) return null;
    return (
      <StatCard
        bare
        icon={resolveIcon(stat.icon, stat.iconColor)}
        iconColor={stat.iconColor}
        value={stat.value}
        valueColor={stat.valueColor}
        label={stat.label}
        trend={stat.trend}
        trendText={stat.trendText}
        microViz={buildMicroViz(stat.microViz, stat.microVizData)}
      />
    );
  }

  const W: React.CSSProperties = {
    padding: 16,
    height: '100%',
    display: 'flex',
    flexDirection: 'column',
    boxSizing: 'border-box',
  };
  const C: React.CSSProperties = { flex: 1, minHeight: 0, marginTop: 12, overflow: 'hidden' };

  switch (id) {
    case 'blast-radius':
      return (
        <div style={W}>
          <BlastRadiusGraph />
        </div>
      );
    case 'risk-delta':
      return (
        <div style={W}>
          <SectionHeader
            title="Risk Delta Projection"
            action={
              <span style={{ fontSize: 10, color: 'var(--color-muted)', fontWeight: 500 }}>
                30-day forecast
              </span>
            }
          />
          <div style={C}>
            <RiskDeltaChart />
          </div>
        </div>
      );
    case 'sla-waterfall':
      return (
        <div style={W}>
          <SectionHeader title="SLA Bridge Waterfall" />
          <div style={C}>
            <SLABridgeWaterfall />
          </div>
        </div>
      );
    case 'deployment-timeline':
      return (
        <div style={W}>
          <SectionHeader title="Deployment Timeline" />
          <div style={{ ...C, overflow: 'auto' }}>
            <DeploymentTimeline />
          </div>
        </div>
      );
    case 'vuln-heatmap':
      return (
        <div style={W}>
          <SectionHeader title="Vulnerability Heatmap" />
          <div style={C}>
            <VulnHeatmapGrid />
          </div>
        </div>
      );
    case 'compliance-rings':
      return (
        <div style={W}>
          <SectionHeader title="Compliance Gauges" />
          <div style={C}>
            <ComplianceRings />
          </div>
        </div>
      );
    case 'agent-rollout':
      return (
        <div style={W}>
          <SectionHeader title="Agent Rollout Pipeline" />
          <div style={C}>
            <AgentRolloutSankey />
          </div>
        </div>
      );
    case 'sla-countdown':
      return (
        <div style={W}>
          <SectionHeader title="SLA Countdown" />
          <div style={C}>
            <SLACountdownTimers />
          </div>
        </div>
      );
    case 'workflow-pipeline':
      return (
        <div style={W}>
          <SectionHeader title="Workflow Pipeline" />
          <div style={{ ...C, overflow: 'auto' }}>
            <WorkflowPipeline />
          </div>
        </div>
      );
    case 'patches-horizon':
      return (
        <div style={W}>
          <SectionHeader title="Critical Patches Horizon" />
          <div style={C}>
            <PatchesHorizon />
          </div>
        </div>
      );
    case 'top-cves':
      return (
        <div style={W}>
          <SectionHeader title="Top Critical CVEs" />
          <div style={C}>
            <TopCriticalCVEs />
          </div>
        </div>
      );
    case 'mttp':
      return (
        <div style={W}>
          <SectionHeader title="Mean Time to Patch" />
          <div style={C}>
            <MeanTimeToPatch />
          </div>
        </div>
      );
    case 'patch-success-rate':
      return (
        <div style={W}>
          <SectionHeader title="Patch Success Rate" />
          <div style={C}>
            <PatchSuccessRate />
          </div>
        </div>
      );
    case 'cve-age':
      return (
        <div style={W}>
          <SectionHeader title="CVE Age Distribution" />
          <div style={C}>
            <CVEAgeDistribution />
          </div>
        </div>
      );
    case 'endpoint-coverage':
      return (
        <div style={W}>
          <SectionHeader title="Endpoint Coverage" />
          <div style={C}>
            <EndpointCoverageMap />
          </div>
        </div>
      );
    case 'patch-failure-reasons':
      return (
        <div style={W}>
          <SectionHeader title="Patch Failure Reasons" />
          <div style={C}>
            <PatchFailureReasons />
          </div>
        </div>
      );
    case 'upcoming-sla':
      return (
        <div style={W}>
          <SectionHeader title="Upcoming SLA Deadlines" />
          <div style={{ ...C, overflow: 'auto' }}>
            <UpcomingSLADeadlines />
          </div>
        </div>
      );
    default:
      return null;
  }
}

export function DashboardGrid({
  layout,
  isEditMode,
  alertDismissed,
  onLayoutChange,
  onAlertDismiss,
}: DashboardGridProps) {
  // CSS opacity fade on mount — replaces framer-motion per-widget stagger
  const [mounted, setMounted] = useState(false);
  useEffect(() => {
    setMounted(true);
  }, []);

  // AlertBanner is rendered outside rgl to avoid blank-gap on dismiss.
  // The 'alert' layout item is retained in layout state for saved-layout
  // compatibility but filtered out of rgl's children below.
  const gridItems = layout.filter((item) => item.i !== 'alert');

  return (
    <div style={{ opacity: mounted ? 1 : 0, transition: 'opacity 0.4s ease' }}>
      {!alertDismissed && (
        <div style={{ marginBottom: 12 }}>
          <AlertBanner messages={ALERT_MESSAGES} onDismiss={onAlertDismiss} />
        </div>
      )}

      <ResponsiveGridLayout
        layout={gridItems}
        cols={12}
        rowHeight={80}
        margin={[12, 12]}
        containerPadding={[0, 0]}
        isDraggable={isEditMode}
        isResizable={isEditMode}
        draggableHandle=".drag-handle"
        onLayoutChange={(items) => onLayoutChange([...items])}
      >
        {gridItems.map((item) => {
          const title = item.i.startsWith('stat-')
            ? STAT_TITLES[parseInt(item.i.replace('stat-', ''), 10)]
            : (WIDGET_TITLES[item.i] ?? item.i);
          return (
            <WidgetShell key={item.i} title={title} isEditMode={isEditMode}>
              <WidgetContent id={item.i} />
            </WidgetShell>
          );
        })}
      </ResponsiveGridLayout>
    </div>
  );
}
