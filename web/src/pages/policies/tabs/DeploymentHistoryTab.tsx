import { Link } from 'react-router';
import { timeAgo } from '../../../lib/time';
import type { components } from '../../../api/types';

type PolicyDetail = components['schemas']['PolicyDetail'];
type Deployment = components['schemas']['Deployment'];

interface DeploymentHistoryTabProps {
  policy: PolicyDetail;
}

const TH: React.CSSProperties = {
  padding: '9px 12px',
  textAlign: 'left',
  fontFamily: 'var(--font-mono)',
  fontSize: 9,
  fontWeight: 600,
  textTransform: 'uppercase',
  letterSpacing: '0.06em',
  color: 'var(--text-muted)',
  whiteSpace: 'nowrap',
  background: 'var(--bg-inset)',
  borderBottom: '1px solid var(--border)',
};

const TD: React.CSSProperties = {
  padding: '9px 12px',
  verticalAlign: 'middle',
  borderBottom: '1px solid var(--border)',
  fontSize: 12,
};

const statusColorMap: Record<string, string> = {
  completed: 'var(--signal-healthy)',
  failed: 'var(--signal-critical)',
  rollback_failed: 'var(--signal-critical)',
  running: 'var(--accent)',
  cancelled: 'var(--text-muted)',
};

function getStatusColor(status: Deployment['status']): string {
  return statusColorMap[status] ?? 'var(--text-muted)';
}

// Summary stat strip
function DeploymentSummaryStrip({ deployments }: { deployments: Deployment[] }) {
  if (deployments.length === 0) return null;

  const completed = deployments.filter((d) => d.status === 'completed').length;
  const failed = deployments.filter(
    (d) => d.status === 'failed' || d.status === 'rollback_failed',
  ).length;
  const running = deployments.filter((d) => d.status === 'running').length;

  const avgSuccessPct =
    deployments.reduce((sum: number, dep) => {
      const total = dep.target_count ?? 0;
      if (total === 0) return sum;
      return sum + Math.round((dep.success_count / total) * 100);
    }, 0) / Math.max(deployments.length, 1);

  return (
    <div
      style={{
        background: 'var(--bg-card)',
        border: '1px solid var(--border)',
        borderRadius: 8,
        padding: '14px 20px',
        marginBottom: 14,
        display: 'flex',
        alignItems: 'center',
        gap: 28,
      }}
    >
      {[
        { label: 'Total', value: deployments.length, color: 'var(--text-emphasis)' },
        { label: 'Completed', value: completed, color: 'var(--signal-healthy)' },
        {
          label: 'Failed',
          value: failed,
          color: failed > 0 ? 'var(--signal-critical)' : 'var(--text-muted)',
        },
        {
          label: 'Running',
          value: running,
          color: running > 0 ? 'var(--accent)' : 'var(--text-muted)',
        },
        {
          label: 'Avg Success',
          value: `${Math.round(avgSuccessPct)}%`,
          color: avgSuccessPct >= 80 ? 'var(--signal-healthy)' : 'var(--signal-warning)',
        },
      ].map(({ label, value, color }) => (
        <div key={label}>
          <div
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 18,
              fontWeight: 700,
              color,
              lineHeight: 1,
              marginBottom: 3,
            }}
          >
            {value}
          </div>
          <div
            style={{
              fontSize: 9,
              color: 'var(--text-muted)',
              textTransform: 'uppercase',
              letterSpacing: '0.06em',
              fontFamily: 'var(--font-mono)',
            }}
          >
            {label}
          </div>
        </div>
      ))}

      <div style={{ flex: 1 }} />

      {/* Mini status bar */}
      {deployments.length > 0 && (
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <div style={{ fontSize: 9, color: 'var(--text-muted)' }}>outcomes</div>
          <div
            style={{
              width: 80,
              height: 4,
              borderRadius: 2,
              background: 'var(--bg-inset)',
              display: 'flex',
              overflow: 'hidden',
              gap: 1,
            }}
          >
            {completed > 0 && (
              <div style={{ flex: completed, background: 'var(--signal-healthy)' }} />
            )}
            {running > 0 && <div style={{ flex: running, background: 'var(--accent)' }} />}
            {failed > 0 && <div style={{ flex: failed, background: 'var(--signal-critical)' }} />}
          </div>
        </div>
      )}
    </div>
  );
}

export const DeploymentHistoryTab = ({ policy }: DeploymentHistoryTabProps) => {
  const deployments = policy.recent_deployments ?? [];

  if (deployments.length === 0) {
    return (
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          height: 140,
          borderRadius: 8,
          border: '1px dashed var(--border)',
          background: 'var(--bg-card)',
          fontSize: 12,
          color: 'var(--text-muted)',
        }}
      >
        No deployments recorded yet.
      </div>
    );
  }

  return (
    <div>
      <DeploymentSummaryStrip deployments={deployments} />

      <div
        style={{
          background: 'var(--bg-card)',
          border: '1px solid var(--border)',
          borderRadius: 8,
          overflow: 'hidden',
        }}
      >
        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
          <thead>
            <tr>
              <th style={TH}>ID</th>
              <th style={TH}>Date</th>
              <th style={TH}>Status</th>
              <th style={TH}>Targets</th>
              <th style={TH}>Success Rate</th>
              <th style={TH}>Duration</th>
            </tr>
          </thead>
          <tbody>
            {deployments.map((dep: Deployment) => {
              const targetCount = dep.target_count ?? 0;
              const successPct =
                targetCount > 0 ? Math.round((dep.success_count / targetCount) * 100) : null;
              const startedAt = dep.started_at ?? null;
              const completedAt = dep.completed_at ?? null;
              const duration =
                startedAt && completedAt
                  ? `${Math.round(
                      (new Date(completedAt).getTime() - new Date(startedAt).getTime()) / 1000,
                    )}s`
                  : '—';
              const statusColor = getStatusColor(dep.status);
              const isFailed = dep.status === 'failed' || dep.status === 'rollback_failed';

              return (
                <tr
                  key={dep.id}
                  style={{
                    background: isFailed
                      ? 'color-mix(in srgb, var(--signal-critical) 1%, transparent)'
                      : 'transparent',
                    borderLeft: isFailed
                      ? '2px solid color-mix(in srgb, var(--signal-critical) 1%, transparent)'
                      : '2px solid transparent',
                    transition: 'background 0.1s',
                    cursor: 'pointer',
                  }}
                  onMouseEnter={(e) => (e.currentTarget.style.background = 'var(--bg-card-hover)')}
                  onMouseLeave={(e) =>
                    (e.currentTarget.style.background = isFailed
                      ? 'color-mix(in srgb, var(--signal-critical) 1%, transparent)'
                      : 'transparent')
                  }
                >
                  <td style={TD}>
                    <Link
                      to={`/deployments/${dep.id}`}
                      style={{
                        fontFamily: 'var(--font-mono)',
                        fontSize: 11,
                        color: 'var(--accent)',
                        textDecoration: 'none',
                        letterSpacing: '0.01em',
                      }}
                    >
                      {dep.id.slice(0, 8)}
                    </Link>
                  </td>
                  <td
                    style={{
                      ...TD,
                      fontFamily: 'var(--font-mono)',
                      fontSize: 10,
                      color: 'var(--text-muted)',
                    }}
                  >
                    {timeAgo(dep.created_at)}
                  </td>
                  <td style={TD}>
                    <span
                      style={{
                        display: 'inline-flex',
                        alignItems: 'center',
                        gap: 5,
                        fontFamily: 'var(--font-mono)',
                        fontSize: 10,
                        fontWeight: 600,
                        color: statusColor,
                      }}
                    >
                      <span
                        style={{
                          width: 5,
                          height: 5,
                          borderRadius: '50%',
                          background: statusColor,
                          flexShrink: 0,
                        }}
                      />
                      {dep.status}
                    </span>
                  </td>
                  <td
                    style={{
                      ...TD,
                      fontFamily: 'var(--font-mono)',
                      fontSize: 12,
                      color: 'var(--text-primary)',
                    }}
                  >
                    {targetCount}
                  </td>
                  <td style={TD}>
                    {successPct != null ? (
                      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                        <div
                          style={{
                            width: 56,
                            height: 4,
                            borderRadius: 2,
                            background: 'var(--bg-inset)',
                            overflow: 'hidden',
                          }}
                        >
                          <div
                            style={{
                              width: `${successPct}%`,
                              height: '100%',
                              background:
                                successPct >= 80
                                  ? 'var(--signal-healthy)'
                                  : successPct >= 50
                                    ? 'var(--signal-warning)'
                                    : 'var(--signal-critical)',
                              borderRadius: 2,
                              transition: 'width 0.3s',
                            }}
                          />
                        </div>
                        <span
                          style={{
                            fontFamily: 'var(--font-mono)',
                            fontSize: 10,
                            color: 'var(--text-muted)',
                            flexShrink: 0,
                          }}
                        >
                          {successPct}%
                        </span>
                      </div>
                    ) : (
                      <span style={{ color: 'var(--text-muted)', fontSize: 11 }}>—</span>
                    )}
                  </td>
                  <td
                    style={{
                      ...TD,
                      fontFamily: 'var(--font-mono)',
                      fontSize: 10,
                      color: 'var(--text-muted)',
                    }}
                  >
                    {duration}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
};
