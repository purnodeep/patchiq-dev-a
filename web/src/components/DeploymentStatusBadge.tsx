type DeploymentStatus =
  | 'created'
  | 'scheduled'
  | 'running'
  | 'completed'
  | 'failed'
  | 'cancelled'
  | 'rolling_back'
  | 'rolled_back'
  | 'rollback_failed'
  // Target-level statuses (endpoint deployment history)
  | 'pending'
  | 'sent'
  | 'succeeded';

const config: Record<DeploymentStatus, { color: string; pulse: boolean }> = {
  created: { color: 'var(--signal-warning)', pulse: false },
  scheduled: { color: 'var(--accent)', pulse: false },
  running: { color: 'var(--accent)', pulse: true },
  completed: { color: 'var(--signal-healthy)', pulse: false },
  failed: { color: 'var(--signal-critical)', pulse: false },
  cancelled: { color: 'var(--text-muted)', pulse: false },
  rolling_back: { color: 'var(--signal-warning)', pulse: true },
  rolled_back: { color: 'var(--text-secondary)', pulse: false },
  rollback_failed: { color: 'var(--signal-critical)', pulse: false },
  // Target-level statuses
  pending: { color: 'var(--signal-warning)', pulse: false },
  sent: { color: 'var(--accent)', pulse: true },
  succeeded: { color: 'var(--signal-healthy)', pulse: false },
};

interface DeploymentStatusBadgeProps {
  status: DeploymentStatus;
  className?: string;
}

export const DeploymentStatusBadge = ({ status }: DeploymentStatusBadgeProps) => {
  const c = config[status] ?? { color: 'var(--text-muted)', pulse: false };
  return (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 6,
        fontSize: 13,
        fontFamily: 'var(--font-sans)',
        fontWeight: 500,
        color: c.color,
      }}
    >
      <span
        style={{
          width: 8,
          height: 8,
          borderRadius: '50%',
          background: c.color,
          flexShrink: 0,
          animation: c.pulse ? 'pulse 2s cubic-bezier(0.4,0,0.6,1) infinite' : undefined,
        }}
      />
      {status}
    </span>
  );
};
