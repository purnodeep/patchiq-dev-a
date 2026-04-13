type EndpointStatus = 'online' | 'offline' | 'degraded' | 'stale' | 'decommissioned';
type DeploymentStatus = 'running' | 'completed' | 'failed' | 'pending' | 'cancelled';
type PolicyStatus = 'enforce' | 'audit' | 'disabled';

type Status =
  | EndpointStatus
  | DeploymentStatus
  | PolicyStatus
  | 'active'
  | 'inactive'
  | 'decommissioned';

const statusConfig: Record<Status, { color: string; pulse: boolean }> = {
  online: { color: 'var(--signal-healthy)', pulse: true },
  active: { color: 'var(--signal-healthy)', pulse: true },
  offline: { color: 'var(--text-muted)', pulse: false },
  inactive: { color: 'var(--text-muted)', pulse: false },
  degraded: { color: 'var(--signal-warning)', pulse: true },
  stale: { color: 'var(--signal-warning)', pulse: false },
  running: { color: 'var(--accent)', pulse: true },
  pending: { color: 'var(--signal-warning)', pulse: false },
  completed: { color: 'var(--signal-healthy)', pulse: false },
  failed: { color: 'var(--signal-critical)', pulse: false },
  cancelled: { color: 'var(--text-muted)', pulse: false },
  enforce: { color: 'var(--accent)', pulse: false },
  audit: { color: 'var(--signal-warning)', pulse: false },
  disabled: { color: 'var(--text-muted)', pulse: false },
  decommissioned: { color: 'var(--signal-critical)', pulse: false },
};

interface StatusBadgeProps {
  status: Status;
  className?: string;
}

export type { Status };

export const StatusBadge = ({ status }: StatusBadgeProps) => {
  const config = statusConfig[status] ?? { color: 'var(--text-muted)', pulse: false };
  return (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 6,
        fontSize: 13,
        fontFamily: 'var(--font-sans)',
        fontWeight: 500,
        color: config.color,
      }}
    >
      <span
        style={{
          width: 8,
          height: 8,
          borderRadius: '50%',
          backgroundColor: config.color,
          flexShrink: 0,
          animation: config.pulse ? 'pulse 2s cubic-bezier(0.4,0,0.6,1) infinite' : undefined,
        }}
      />
      {status === 'decommissioned' ? 'deleted' : status}
    </span>
  );
};
