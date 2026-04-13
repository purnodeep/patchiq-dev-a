import { useNavigate } from 'react-router';
import { RingGauge } from '@patchiq/ui';
import type { DashboardSummary } from '../../api/hooks/useDashboard';
import { useDashboardData } from './DashboardContext';

interface StatCardsRow1Props {
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

export function StatCardsRow1({ data: dataProp }: StatCardsRow1Props) {
  const contextData = useDashboardData();
  const data = dataProp ?? contextData;
  const navigate = useNavigate();

  const activeEndpoints = data.active_endpoints ?? 0;
  const totalEndpoints = data.total_endpoints ?? 0;
  const activeDeployments = data.active_deployments ?? [];

  const onlineRatio = totalEndpoints > 0 ? Math.round((activeEndpoints / totalEndpoints) * 100) : 0;

  const offlineCount = totalEndpoints - activeEndpoints;
  const endpointSublabel = offlineCount > 0 ? `↓ ${offlineCount} offline` : `↑ All online`;

  const criticalPatches = data.critical_patches ?? 0;
  const criticalSublabel =
    criticalPatches > 0 ? `↑ ${criticalPatches} unresolved` : '→ None pending';

  const complianceRate = data.compliance_rate ?? 0;
  const complianceSublabel =
    complianceRate >= 90
      ? `↑ ${data.framework_count ?? 0} frameworks`
      : complianceRate >= 70
        ? `→ ${data.framework_count ?? 0} frameworks`
        : '↓ Below threshold';

  const activeCount = activeDeployments.length;
  const runningCount = activeDeployments.filter((d) => d.status === 'running').length;
  const deploymentSublabel = activeCount > 0 ? `↑ ${runningCount} running` : '→ None active';

  return (
    <div className="grid h-full grid-cols-4 gap-2">
      {/* Endpoints Online */}
      <div style={cardStyle} onClick={() => void navigate('/endpoints')}>
        <div style={labelStyle}>Endpoints Online</div>
        <div
          style={{
            flex: 1,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            marginTop: 4,
            marginBottom: 4,
          }}
        >
          <div style={valueStyle}>{activeEndpoints}</div>
          <RingGauge value={onlineRatio} size={32} strokeWidth={4} />
        </div>
        <div style={sublabelStyle}>{endpointSublabel}</div>
      </div>

      {/* Critical Patches */}
      <div style={cardStyle} onClick={() => void navigate('/patches?severity=critical')}>
        <div style={labelStyle}>Critical Patches</div>
        <div
          style={{
            flex: 1,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            marginTop: 4,
            marginBottom: 4,
          }}
        >
          <div
            style={{
              ...valueStyle,
              color: criticalPatches > 0 ? 'var(--signal-critical)' : 'var(--text-emphasis)',
            }}
          >
            {criticalPatches}
          </div>
          <div className="flex flex-col gap-0.5">
            {[
              { label: 'C', count: data.critical_patches },
              { label: 'H', count: data.patches_high },
              { label: 'M', count: data.patches_medium },
              { label: 'L', count: data.patches_low },
            ].map(({ label, count }) => (
              <div key={label} className="flex items-center gap-1">
                <span className="w-2.5 text-[9px]" style={{ color: 'var(--text-muted)' }}>
                  {label}
                </span>
                <div
                  className="h-1.5 rounded-full"
                  style={{
                    width: `${Math.min(40, count / 2 + 4)}px`,
                    backgroundColor: 'var(--accent)',
                    opacity: label === 'C' ? 1 : label === 'H' ? 0.7 : label === 'M' ? 0.5 : 0.3,
                  }}
                />
                <span
                  className="text-[9px]"
                  style={{ color: 'var(--text-muted)', fontFamily: 'var(--font-mono)' }}
                >
                  {count}
                </span>
              </div>
            ))}
          </div>
        </div>
        <div style={sublabelStyle}>{criticalSublabel}</div>
      </div>

      {/* Compliance Rate */}
      <div style={cardStyle} onClick={() => void navigate('/compliance')}>
        <div style={labelStyle}>Compliance Rate</div>
        <div
          style={{
            flex: 1,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            marginTop: 4,
            marginBottom: 4,
          }}
        >
          <div style={{ ...valueStyle, color: 'var(--accent)' }}>
            {complianceRate >= 0 ? `${Math.round(complianceRate)}%` : 'N/A'}
          </div>
          {complianceRate >= 0 ? (
            <RingGauge value={complianceRate} size={32} strokeWidth={4} colorByValue />
          ) : undefined}
        </div>
        <div style={sublabelStyle}>{complianceSublabel}</div>
      </div>

      {/* Active Deployments */}
      <div style={cardStyle} onClick={() => void navigate('/deployments?status=running')}>
        <div style={labelStyle}>Active Deployments</div>
        <div
          style={{
            flex: 1,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            marginTop: 4,
            marginBottom: 4,
          }}
        >
          <div style={valueStyle}>{activeCount}</div>
          {activeCount > 0 && (
            <div className="flex flex-col gap-1">
              {activeDeployments.slice(0, 4).map((d) => (
                <div key={d.id} className="flex items-center gap-1">
                  <span
                    className={`h-1.5 w-1.5 rounded-full ${d.status === 'running' ? 'animate-pulse' : ''}`}
                    style={{ backgroundColor: 'var(--accent)' }}
                  />
                  <span
                    className="max-w-[56px] truncate text-[9px]"
                    style={{ color: 'var(--text-muted)' }}
                  >
                    {d.name}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
        <div style={sublabelStyle}>{deploymentSublabel}</div>
      </div>
    </div>
  );
}
