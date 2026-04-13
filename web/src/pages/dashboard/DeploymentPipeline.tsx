import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router';
import { useDashboardSummary } from '@/api/hooks/useDashboard';
import type { ActiveDeployment } from '@/api/hooks/useDashboard';

const cardStyle: React.CSSProperties = {
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  borderRadius: 8,
  boxShadow: 'var(--shadow-sm)',
  transition: 'border-color 150ms ease',
  display: 'flex',
  flexDirection: 'column',
};

function statusColor(status: string): string {
  switch (status.toLowerCase()) {
    case 'complete':
    case 'completed':
      return 'var(--accent)';
    case 'running':
    case 'in_progress':
      return 'var(--text-primary)';
    case 'failed':
      return 'var(--signal-critical)';
    default:
      return 'var(--text-muted)';
  }
}

function statusLabel(status: string): string {
  switch (status.toLowerCase()) {
    case 'complete':
    case 'completed':
      return 'Complete';
    case 'running':
    case 'in_progress':
      return 'Running';
    case 'failed':
      return 'Failed';
    case 'pending':
    case 'created':
    case 'scheduled':
      return 'Pending';
    case 'rolling_back':
      return 'Rolling Back';
    case 'cancelled':
      return 'Cancelled';
    default:
      return status.charAt(0).toUpperCase() + status.slice(1);
  }
}

function isFailed(status: string): boolean {
  return status.toLowerCase() === 'failed';
}

function isRunning(status: string): boolean {
  return status.toLowerCase() === 'running' || status.toLowerCase() === 'in_progress';
}

/** Convert progress_pct to "completed/total" fraction string.
 *  We infer total from a round number near the percentage. */
function fractionLabel(progressPct: number): string {
  // Try common total values: 100, 60, 50, 40, 30, 20, 10
  const commonTotals = [100, 60, 50, 40, 30, 25, 20, 10];
  for (const total of commonTotals) {
    const completed = Math.round((progressPct / 100) * total);
    if (Math.abs((completed / total) * 100 - progressPct) < 2) {
      return `${completed}/${total}`;
    }
  }
  // Fallback: treat total as 100
  return `${Math.round(progressPct)}/100`;
}

function AnimatedEllipsis() {
  const [dots, setDots] = useState('.');

  useEffect(() => {
    const interval = setInterval(() => {
      setDots((d) => {
        if (d === '...') return '.';
        return d + '.';
      });
    }, 500);
    return () => clearInterval(interval);
  }, []);

  return (
    <span
      style={{
        color: 'var(--text-faint)',
        fontFamily: 'var(--font-mono)',
        fontSize: 10,
        display: 'inline-block',
        width: 20,
        textAlign: 'left',
      }}
    >
      {dots}
    </span>
  );
}

function DeploymentRow({ d, isLast }: { d: ActiveDeployment; isLast: boolean }) {
  const navigate = useNavigate();
  const running = isRunning(d.status);
  const failed = isFailed(d.status);

  return (
    <div
      style={{
        padding: '10px 0',
        borderBottom: isLast ? 'none' : '1px solid var(--border-faint)',
      }}
    >
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          marginBottom: 7,
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 7 }}>
          {running && (
            <span
              style={{
                width: 5,
                height: 5,
                borderRadius: '50%',
                background: 'var(--accent)',
                display: 'inline-block',
                flexShrink: 0,
                animation: 'pulse 1.4s ease-in-out infinite',
              }}
            />
          )}
          <span
            onClick={() => navigate(`/deployments/${d.id}`)}
            style={{ fontSize: 13, color: 'var(--text-primary)', cursor: 'pointer' }}
          >
            {d.name}
          </span>
          {running && (
            <span style={{ fontSize: 10, color: 'var(--text-faint)' }}>
              ongoing
              <AnimatedEllipsis />
            </span>
          )}
        </div>
        <span style={{ fontSize: 12, color: statusColor(d.status) }}>{statusLabel(d.status)}</span>
      </div>
      <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
        <div
          style={{
            flex: 1,
            height: 3,
            background: 'var(--progress-track, var(--border))',
            borderRadius: 2,
            overflow: 'hidden',
            position: 'relative',
          }}
        >
          <div
            style={{
              height: '100%',
              borderRadius: 2,
              background: failed ? 'var(--signal-critical)' : 'var(--accent)',
              width: `${d.progress_pct}%`,
            }}
          />
        </div>
        <span
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 11,
            color: 'var(--text-muted)',
            whiteSpace: 'nowrap',
            minWidth: 36,
            textAlign: 'right',
          }}
        >
          {fractionLabel(d.progress_pct)}
        </span>
      </div>
    </div>
  );
}

export function DeploymentPipeline() {
  const { data, isLoading } = useDashboardSummary();
  const deployments: ActiveDeployment[] = data?.active_deployments ?? [];
  const activeCount = deployments.filter((d) => isRunning(d.status)).length || deployments.length;

  return (
    <div
      style={cardStyle}
      onMouseEnter={(e) => {
        (e.currentTarget as HTMLDivElement).style.borderColor = 'var(--text-faint)';
      }}
      onMouseLeave={(e) => {
        (e.currentTarget as HTMLDivElement).style.borderColor = 'var(--border)';
      }}
    >
      <div
        style={{
          padding: '16px 20px 0',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}
      >
        <span style={{ fontSize: 13, color: 'var(--text-muted)', fontWeight: 400 }}>
          Deployment Pipeline
        </span>
        <span style={{ fontSize: 11, color: 'var(--text-faint)', fontFamily: 'var(--font-mono)' }}>
          {isLoading ? '\u2014' : `${activeCount} active`}
        </span>
      </div>
      <div style={{ padding: '12px 20px 16px', flex: 1 }}>
        {isLoading ? (
          <div style={{ color: 'var(--text-faint)', fontSize: 12, padding: '10px 0' }}>
            Loading...
          </div>
        ) : deployments.length === 0 ? (
          <div style={{ color: 'var(--text-faint)', fontSize: 12, padding: '10px 0' }}>
            No active deployments
          </div>
        ) : (
          deployments.map((d, i) => (
            <DeploymentRow key={d.id} d={d} isLast={i === deployments.length - 1} />
          ))
        )}
      </div>
    </div>
  );
}
