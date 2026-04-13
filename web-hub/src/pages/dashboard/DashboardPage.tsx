import { SkeletonCard, ErrorState } from '@patchiq/ui';
import { useDashboardStats } from '../../api/hooks/useDashboard';
import { StatCards } from './StatCards';
import { FleetTopology } from './FleetTopology';
import { FeedPipeline } from './FeedPipeline';
import { CatalogGrowthChart } from './CatalogGrowthChart';
import { LicenseSunburst } from './LicenseSunburst';
import { RecentActivity } from './RecentActivity';

export const DashboardPage = () => {
  const { isLoading, error, refetch } = useDashboardStats();

  if (isLoading) {
    return (
      <div style={{ background: 'var(--bg-page)', padding: 24 }}>
        <div style={{ marginBottom: 20 }}>
          <SkeletonCard lines={1} />
        </div>
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(5, 1fr)',
            gap: 12,
            marginBottom: 12,
          }}
        >
          {Array.from({ length: 5 }).map((_, i) => (
            <SkeletonCard key={i} lines={3} />
          ))}
        </div>
        <SkeletonCard lines={5} className="min-h-[200px]" />
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: '1fr 1fr',
            gap: 12,
            marginTop: 12,
            marginBottom: 12,
          }}
        >
          <SkeletonCard lines={5} className="min-h-[200px]" />
          <SkeletonCard lines={5} className="min-h-[200px]" />
        </div>
        <SkeletonCard lines={5} className="min-h-[200px]" />
        <div style={{ marginTop: 12 }}>
          <SkeletonCard lines={5} className="min-h-[200px]" />
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div style={{ background: 'var(--bg-page)', padding: 24 }}>
        <ErrorState message="Failed to load dashboard data" onRetry={() => refetch()} />
      </div>
    );
  }

  return (
    <div style={{ background: 'var(--bg-page)', padding: 24 }}>
      {/* Page title — inline like Patch Manager */}
      <div
        style={{
          fontSize: 17,
          fontWeight: 600,
          color: 'var(--text-emphasis)',
          letterSpacing: '-0.02em',
          marginBottom: 20,
        }}
      >
        Dashboard
      </div>

      {/* Row 1: Stat Cards — primary KPIs */}
      <div style={{ marginBottom: 12 }}>
        <StatCards />
      </div>

      {/* Row 2: Fleet Topology + License Distribution */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: '55fr 45fr',
          gap: 12,
          marginBottom: 12,
          alignItems: 'stretch',
        }}
      >
        <FleetTopology />
        <LicenseSunburst />
      </div>

      {/* Row 3: Feed Pipeline + Catalog Growth */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: '55fr 45fr',
          gap: 12,
          marginBottom: 12,
          alignItems: 'stretch',
        }}
      >
        <FeedPipeline />
        <CatalogGrowthChart />
      </div>

      {/* Row 5: Recent Activity */}
      <RecentActivity />
    </div>
  );
};
