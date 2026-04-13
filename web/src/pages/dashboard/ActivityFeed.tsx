import { useNavigate } from 'react-router';
import { useDashboardActivity } from '@/api/hooks/useDashboard';
import type { ActivityItem } from '@/api/hooks/useDashboard';

const cardStyle: React.CSSProperties = {
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  borderRadius: 8,
  boxShadow: 'var(--shadow-sm)',
  transition: 'border-color 150ms ease',
  display: 'flex',
  flexDirection: 'column',
};

/** Map event type to a dot color. */
function dotColorByType(type: string): string {
  const t = (type ?? '').toLowerCase();
  if (t.includes('deploy') || t.includes('deployment')) return 'var(--signal-healthy)';
  if (t.includes('cve') || t.includes('security') || t.includes('vulnerab'))
    return 'var(--signal-critical)';
  if (t.includes('compliance') || t.includes('audit')) return 'var(--accent)';
  return 'var(--text-faint)';
}

/** Fallback based on status when type is ambiguous. */
function dotColorByStatus(status: string): string {
  const s = (status ?? '').toLowerCase();
  if (s === 'success' || s === 'completed' || s === 'complete') return 'var(--signal-healthy)';
  if (s === 'failed' || s === 'error' || s === 'failure') return 'var(--signal-critical)';
  if (s === 'running') return 'var(--signal-healthy)';
  return 'var(--text-faint)';
}

function resolveDotColor(item: ActivityItem): string {
  const t = (item.type ?? '').toLowerCase();
  // If type gives us a clear category, use it
  if (t && !['system', 'event', 'unknown', ''].includes(t)) {
    return dotColorByType(t);
  }
  return dotColorByStatus(item.status);
}

function formatRelativeTime(timestamp: string): string {
  const now = Date.now();
  const then = new Date(timestamp).getTime();
  const diffMs = now - then;
  const diffMin = Math.floor(diffMs / 60_000);
  if (diffMin < 1) return 'just now';
  if (diffMin < 60) return `${diffMin}m ago`;
  const diffHr = Math.floor(diffMin / 60);
  if (diffHr < 24) return `${diffHr}h ago`;
  const diffDay = Math.floor(diffHr / 24);
  return `${diffDay}d ago`;
}

function resolveNavTarget(item: ActivityItem): string | undefined {
  const t = (item.type ?? '').toLowerCase();
  if ((t.includes('deploy') || t.includes('deployment')) && item.detail?.deployment_id) {
    return `/deployments/${item.detail.deployment_id}`;
  }
  return undefined;
}

function ActivityRow({
  item,
  isLast,
  isFirst,
  onNavigate,
}: {
  item: ActivityItem;
  isLast: boolean;
  isFirst: boolean;
  onNavigate: (path: string) => void;
}) {
  const dotColor = resolveDotColor(item);
  const navTarget = resolveNavTarget(item);

  return (
    <div
      onClick={navTarget ? () => onNavigate(navTarget) : undefined}
      style={{
        display: 'flex',
        alignItems: 'flex-start',
        gap: 10,
        padding: '8px 0',
        borderBottom: isLast ? 'none' : '1px solid var(--border-faint)',
        cursor: navTarget ? 'pointer' : 'default',
      }}
    >
      <div style={{ position: 'relative', marginTop: 4, flexShrink: 0, width: 8, height: 8 }}>
        {isFirst && (
          <span
            style={{
              position: 'absolute',
              inset: -3,
              borderRadius: '50%',
              background: dotColor,
              opacity: 0.25,
              animation: 'pulse 1.8s ease-in-out infinite',
              display: 'block',
            }}
          />
        )}
        <div
          style={{
            width: 8,
            height: 8,
            borderRadius: '50%',
            background: dotColor,
            position: 'relative',
            zIndex: 1,
          }}
        />
      </div>
      <div style={{ flex: 1 }}>
        <div style={{ fontSize: 12, color: 'var(--text-secondary)', lineHeight: 1.45 }}>
          {item.type && (
            <span style={{ color: 'var(--text-muted)', textTransform: 'capitalize' }}>
              {item.type}{' '}
            </span>
          )}
          <strong style={{ color: 'var(--text-primary)', fontWeight: 600 }}>{item.title}</strong>
          {item.status && (
            <span style={{ color: 'var(--text-muted)' }}>
              {' '}
              {item.status === 'completed'
                ? 'completed'
                : item.status === 'failed'
                  ? 'failed'
                  : item.status === 'running'
                    ? 'in progress'
                    : item.status}
            </span>
          )}
          {item.meta && <span style={{ color: 'var(--text-faint)' }}> — {item.meta}</span>}
        </div>
      </div>
      <div
        style={{
          fontFamily: 'var(--font-mono)',
          fontSize: 10,
          color: 'var(--text-faint)',
          whiteSpace: 'nowrap',
          marginTop: 2,
        }}
      >
        {formatRelativeTime(item.timestamp)}
      </div>
    </div>
  );
}

export function ActivityFeed() {
  const navigate = useNavigate();
  const { data: apiItems, isLoading } = useDashboardActivity();
  const visibleItems = apiItems?.slice(0, 6) ?? [];

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
        <span style={{ fontSize: 13, color: 'var(--text-muted)', fontWeight: 400 }}>Activity</span>
      </div>
      <div style={{ padding: '12px 20px 16px', flex: 1 }}>
        {isLoading ? (
          <div style={{ color: 'var(--text-faint)', fontSize: 12, padding: '8px 0' }}>
            Loading...
          </div>
        ) : visibleItems.length === 0 ? (
          <div style={{ color: 'var(--text-faint)', fontSize: 12, padding: '8px 0' }}>
            No recent activity
          </div>
        ) : (
          visibleItems.map((item, i) => (
            <ActivityRow
              key={item.id ?? i}
              item={item}
              isFirst={i === 0}
              isLast={i === visibleItems.length - 1}
              onNavigate={navigate}
            />
          ))
        )}
      </div>
    </div>
  );
}
