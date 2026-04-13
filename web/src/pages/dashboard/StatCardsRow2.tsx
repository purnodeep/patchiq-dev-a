import { useNavigate } from 'react-router';
import { SparklineChart } from '@patchiq/ui';
import type { DashboardSummary } from '../../api/hooks/useDashboard';
import { useDashboardData } from './DashboardContext';

function formatRelativeTime(iso: string | null): string {
  if (!iso) return '';
  const diff = Math.floor((Date.now() - new Date(iso).getTime()) / 1000);
  if (diff < 60) return 'just now';
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  return `${Math.floor(diff / 86400)}d ago`;
}

interface StatCardsRow2Props {
  data?: DashboardSummary;
}

const cardStyle: React.CSSProperties = {
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  borderRadius: 8,
  padding: '8px 14px',
  boxShadow: 'var(--shadow-sm)',
  display: 'flex',
  flexDirection: 'column',
  cursor: 'pointer',
  transition: 'border-color 150ms ease',
  minHeight: 0,
  overflow: 'hidden',
};

const labelStyle: React.CSSProperties = {
  fontSize: 10,
  fontWeight: 500,
  letterSpacing: '0.06em',
  color: 'var(--text-muted)',
  textTransform: 'uppercase',
  fontFamily: 'var(--font-mono)',
};

const valueStyle: React.CSSProperties = {
  fontSize: 24,
  fontWeight: 700,
  lineHeight: 1,
  color: 'var(--text-emphasis)',
  fontFamily: 'var(--font-mono)',
  letterSpacing: '-0.03em',
};

const sublabelStyle: React.CSSProperties = {
  fontSize: 11,
  color: 'var(--text-secondary)',
  fontFamily: 'var(--font-mono)',
};

export function StatCardsRow2({ data: dataProp }: StatCardsRow2Props) {
  const contextData = useDashboardData();
  const data = dataProp ?? contextData;
  const navigate = useNavigate();

  const isSynced = data.hub_sync_status === 'healthy' || data.hub_sync_status === 'idle';
  const syncValue = isSynced ? "Sync'd" : data.hub_sync_status;
  const syncTimeAgo = formatRelativeTime(data.hub_last_sync_at);

  const overdueCount = data.overdue_sla_count ?? 0;
  const failedCount = data.failed_deployments_count ?? 0;
  const workflowsCount = data.workflows_running_count ?? 0;

  return (
    <div className="grid h-full grid-cols-4 gap-2">
      {/* Overdue SLA */}
      <div style={cardStyle} onClick={() => void navigate('/deployments?overdue=true')}>
        <div style={labelStyle}>Overdue SLA</div>
        <div
          style={{
            flex: 1,
            display: 'flex',
            alignItems: 'center',
            marginTop: 8,
            marginBottom: 8,
          }}
        >
          <div
            style={{
              ...valueStyle,
              color: overdueCount > 0 ? 'var(--signal-critical)' : 'var(--text-emphasis)',
            }}
          >
            {overdueCount}
          </div>
        </div>
        <div style={sublabelStyle}>
          {overdueCount > 0 ? `↑ ${overdueCount} past deadline` : '→ On track'}
        </div>
      </div>

      {/* Failed Deployments */}
      <div style={cardStyle} onClick={() => void navigate('/deployments?status=failed')}>
        <div style={labelStyle}>Failed Deployments</div>
        <div
          style={{
            flex: 1,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            marginTop: 8,
            marginBottom: 8,
          }}
        >
          <div
            style={{
              ...valueStyle,
              color: failedCount > 0 ? 'var(--signal-critical)' : 'var(--text-emphasis)',
            }}
          >
            {failedCount}
          </div>
          {(data.failed_trend_7d ?? []).length >= 2 ? (
            <SparklineChart
              data={data.failed_trend_7d}
              color="var(--signal-critical)"
              width={60}
              height={32}
            />
          ) : undefined}
        </div>
        <div style={sublabelStyle}>
          {failedCount > 0 ? `↑ ${failedCount} this week` : '→ None failed'}
        </div>
      </div>

      {/* Workflows Running */}
      <div style={cardStyle} onClick={() => void navigate('/workflows')}>
        <div style={labelStyle}>Workflows Running</div>
        <div
          style={{
            flex: 1,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            marginTop: 8,
            marginBottom: 8,
          }}
        >
          <div style={valueStyle}>{workflowsCount}</div>
          {(data.workflows_running ?? []).length > 0 ? (
            <div className="flex items-center gap-0.5 pt-2">
              <div
                className="h-2.5 w-2.5 rounded-full"
                style={{ backgroundColor: 'var(--accent)' }}
              />
              <div className="h-px w-4" style={{ backgroundColor: 'var(--border)' }} />
              <div
                className="h-2.5 w-2.5 rounded-full"
                style={{ backgroundColor: 'var(--accent)' }}
              />
              <div className="h-px w-4" style={{ backgroundColor: 'var(--border)' }} />
              <div
                className="relative h-2.5 w-2.5 rounded-full"
                style={{ backgroundColor: 'var(--accent)' }}
              >
                <div
                  className="absolute inset-[-3px] animate-ping rounded-full border"
                  style={{ borderColor: 'var(--accent)' }}
                />
              </div>
            </div>
          ) : undefined}
        </div>
        <div style={sublabelStyle}>
          {workflowsCount > 0
            ? `↑ ${(data.workflows_running ?? []).length > 0 ? data.workflows_running[0].name : `${workflowsCount} active`}`
            : '→ None running'}
        </div>
      </div>

      {/* Hub Sync Status */}
      <div style={cardStyle} onClick={() => void navigate('/settings')}>
        <div style={labelStyle}>Hub Sync Status</div>
        <div
          style={{
            flex: 1,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            marginTop: 8,
            marginBottom: 8,
          }}
        >
          <div
            style={{
              ...valueStyle,
              fontSize: syncValue.length > 6 ? 20 : 28,
              color: isSynced ? 'var(--signal-healthy)' : 'var(--signal-critical)',
            }}
          >
            {syncValue}
          </div>
          <div className="flex items-center gap-1">
            <span
              className="h-2 w-2 animate-pulse rounded-full"
              style={{
                backgroundColor: isSynced ? 'var(--signal-healthy)' : 'var(--signal-critical)',
              }}
            />
          </div>
        </div>
        <div style={sublabelStyle}>{syncTimeAgo ? `↑ ${syncTimeAgo}` : '→ Never synced'}</div>
      </div>
    </div>
  );
}
