import { SkeletonCard, ErrorState } from '@patchiq/ui';
import { useNavigate } from 'react-router';
import { useDashboardActivity } from '@/api/hooks/useDashboard';

const STATUS_STYLE: Record<string, { bg: string; border: string; color: string; symbol: string }> =
  {
    running: {
      bg: 'var(--accent-subtle)',
      border: 'var(--accent)',
      color: 'var(--accent)',
      symbol: '\u25b6',
    },
    completed: {
      bg: 'var(--accent-subtle)',
      border: 'var(--accent)',
      color: 'var(--accent)',
      symbol: '\u2713',
    },
    failed: {
      bg: 'var(--signal-critical-subtle)',
      border: 'var(--signal-critical)',
      color: 'var(--signal-critical)',
      symbol: '\u2717',
    },
    created: {
      bg: 'var(--accent-subtle)',
      border: 'var(--accent-border)',
      color: 'var(--accent)',
      symbol: '\u25c6',
    },
    paused: {
      bg: 'var(--signal-warning-subtle)',
      border: 'var(--signal-warning)',
      color: 'var(--signal-warning)',
      symbol: '\u23f8',
    },
    cancelled: {
      bg: 'var(--bg-card-hover)',
      border: 'var(--border-strong)',
      color: 'var(--text-muted)',
      symbol: '\u2717',
    },
  };

export function ActivityTimeline() {
  const { data, isLoading, error, refetch } = useDashboardActivity();
  const navigate = useNavigate();

  if (isLoading)
    return (
      <div
        className="rounded-lg border p-4"
        style={{ background: 'var(--bg-card)', borderColor: 'var(--border)' }}
      >
        <SkeletonCard lines={4} />
      </div>
    );
  if (error)
    return (
      <div
        className="rounded-lg border p-4"
        style={{ background: 'var(--bg-card)', borderColor: 'var(--border)' }}
      >
        <ErrorState message="Failed to load activity" onRetry={() => refetch()} />
      </div>
    );

  const items = data ?? [];

  return (
    <div
      className="rounded-lg border"
      style={{
        background: 'var(--bg-card)',
        borderColor: 'var(--border)',
        boxShadow: 'var(--shadow-sm)',
      }}
    >
      <div className="flex items-center justify-between p-4 pb-2">
        <h3 className="text-sm font-semibold" style={{ color: 'var(--text-emphasis)' }}>
          Deployment Activity
        </h3>
        <span
          className="rounded-full border px-2 py-0.5 text-[10px] font-medium"
          style={{
            color: 'var(--accent)',
            borderColor: 'var(--accent-border)',
          }}
        >
          LIVE
        </span>
      </div>
      <div className="max-h-[280px] overflow-y-auto p-4 pt-0">
        <div>
          {items.length === 0 && (
            <p className="text-sm" style={{ color: 'var(--text-muted)' }}>
              No recent activity
            </p>
          )}
          {items.map((item, i) => {
            const deploymentId = item.detail?.deployment_id;
            return (
              <div
                key={item.id}
                className={`flex gap-3 py-3 ${deploymentId ? 'cursor-pointer rounded px-1 -mx-1' : ''}`}
                style={{
                  borderBottomWidth: i < items.length - 1 ? '1px' : undefined,
                  borderBottomColor: 'var(--border-faint)',
                }}
                onClick={
                  deploymentId ? () => void navigate(`/deployments/${deploymentId}`) : undefined
                }
              >
                <div className="flex flex-col items-center">
                  <div className="relative">
                    {item.status === 'running' && (
                      <div
                        className="absolute inset-0 rounded-full animate-ping"
                        style={{ backgroundColor: 'var(--accent-subtle)' }}
                      />
                    )}
                    <div
                      className="relative flex items-center justify-center rounded-full text-[11px] font-bold"
                      style={{
                        width: 28,
                        height: 28,
                        backgroundColor: (STATUS_STYLE[item.status] ?? STATUS_STYLE['cancelled'])
                          .bg,
                        border: `2px solid ${(STATUS_STYLE[item.status] ?? STATUS_STYLE['cancelled']).border}`,
                        color: (STATUS_STYLE[item.status] ?? STATUS_STYLE['cancelled']).color,
                      }}
                    >
                      {(STATUS_STYLE[item.status] ?? STATUS_STYLE['cancelled']).symbol}
                    </div>
                  </div>
                  {i < items.length - 1 && (
                    <div
                      className="w-px flex-1 mt-1"
                      style={{ backgroundColor: 'var(--border)' }}
                    />
                  )}
                </div>
                <div className="flex-1 min-w-0">
                  <p
                    className="text-xs font-medium truncate"
                    style={{ color: 'var(--text-primary)' }}
                  >
                    {item.title}
                  </p>
                  <p className="text-[10px]" style={{ color: 'var(--text-muted)' }}>
                    {item.meta}
                  </p>
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
