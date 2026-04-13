import { Fragment, useState } from 'react';
import { Skeleton } from '@patchiq/ui';
import { History } from 'lucide-react';
import { Link } from 'react-router';
import { useEndpointDeployments } from '../../../api/hooks/useEndpoints';
import { EmptyState } from '../../../components/EmptyState';

interface HistoryTabProps {
  endpointId: string;
}

// ── design tokens ──────────────────────────────────────────────
const S = {
  card: {
    background: 'var(--bg-card)',
    border: '1px solid var(--border)',
    borderRadius: 8,
    boxShadow: 'var(--shadow-sm)',
    overflow: 'hidden' as const,
  },
  cardTitle: {
    fontFamily: 'var(--font-mono)',
    fontSize: 10,
    fontWeight: 500,
    textTransform: 'uppercase' as const,
    letterSpacing: '0.05em',
    color: 'var(--text-muted)',
    padding: '12px 16px',
    borderBottom: '1px solid var(--border)',
    background: 'var(--bg-inset)',
  },
  th: {
    fontFamily: 'var(--font-mono)',
    fontSize: 10,
    fontWeight: 500,
    textTransform: 'uppercase' as const,
    letterSpacing: '0.05em',
    color: 'var(--text-muted)',
    padding: '9px 12px',
    background: 'var(--bg-inset)',
    borderBottom: '1px solid var(--border)',
    textAlign: 'left' as const,
    whiteSpace: 'nowrap' as const,
  },
  td: {
    padding: '10px 12px',
    borderBottom: '1px solid var(--border)',
    color: 'var(--text-primary)',
    fontSize: 13,
  },
};

const STATUS_COLOR: Record<string, string> = {
  completed: 'var(--signal-healthy)',
  succeeded: 'var(--signal-healthy)',
  running: 'var(--accent)',
  failed: 'var(--signal-critical)',
  cancelled: 'var(--text-faint)',
  pending: 'var(--signal-warning)',
  scheduled: 'var(--signal-warning)',
  created: 'var(--text-secondary)',
  rolling_back: 'var(--signal-warning)',
  rolled_back: 'var(--text-secondary)',
  rollback_failed: 'var(--signal-critical)',
};

function formatDateTime(dateStr: string | null | undefined): string {
  if (!dateStr) return '—';
  const d = new Date(dateStr);
  if (isNaN(d.getTime())) return '—';
  return d.toLocaleString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  });
}

function formatDuration(startedAt: string, completedAt: string | null | undefined): string {
  if (!completedAt) return '—';
  const diffMs = new Date(completedAt).getTime() - new Date(startedAt).getTime();
  if (diffMs < 0) return '—';
  const totalSec = Math.floor(diffMs / 1000);
  const minutes = Math.floor(totalSec / 60);
  const seconds = totalSec % 60;
  return `${minutes}m ${seconds}s`;
}

function shortId(id: string): string {
  return `D-${id.slice(0, 6).toUpperCase()}`;
}

function timeAgo(dateStr: string): string {
  const diffMs = Date.now() - new Date(dateStr).getTime();
  const mins = Math.floor(diffMs / 60000);
  if (mins < 1) return 'just now';
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  const days = Math.floor(hrs / 24);
  return `${days}d ago`;
}

export function HistoryTab({ endpointId }: HistoryTabProps) {
  const { data: deploymentsData, isLoading, error } = useEndpointDeployments(endpointId);
  const targets = deploymentsData?.data ?? [];
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set());

  const toggleExpand = (key: string) => {
    setExpandedRows((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  };

  // Summary stats
  const completedCount = targets.filter(
    (t) => t.status === 'completed' || t.status === 'succeeded',
  ).length;
  const failedCount = targets.filter((t) => t.status === 'failed').length;
  const runningCount = targets.filter((t) => t.status === 'running').length;

  if (isLoading) {
    return (
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        <Skeleton className="h-20 w-full rounded-lg" />
        {Array.from({ length: 5 }).map((_, i) => (
          <Skeleton key={i} className="h-10 w-full rounded" />
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <div style={{ ...S.card, padding: 16 }}>
        <span style={{ fontSize: 13, color: 'var(--signal-critical)' }}>
          Error loading deployment history
        </span>
      </div>
    );
  }

  if (!targets || targets.length === 0) {
    return (
      <EmptyState
        icon={History}
        title="No deployment history"
        description="This endpoint has not been part of any deployments yet."
      />
    );
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* Hero stat strip */}
      <div
        style={{
          ...S.card,
          padding: '16px 20px',
          display: 'flex',
          gap: 32,
          alignItems: 'center',
          flexWrap: 'wrap' as const,
        }}
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          <span
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 26,
              fontWeight: 700,
              color: 'var(--text-emphasis)',
              lineHeight: 1,
            }}
          >
            {targets.length}
          </span>
          <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>total deployments</span>
        </div>

        <div style={{ width: 1, height: 28, background: 'var(--border)', flexShrink: 0 }} />
        <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          <span
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 22,
              fontWeight: 700,
              color: 'var(--signal-healthy)',
              lineHeight: 1,
            }}
          >
            {completedCount}
          </span>
          <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>completed</span>
        </div>

        <div style={{ width: 1, height: 28, background: 'var(--border)', flexShrink: 0 }} />
        <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          <span
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 22,
              fontWeight: 700,
              color: failedCount > 0 ? 'var(--signal-critical)' : 'var(--text-muted)',
              lineHeight: 1,
            }}
          >
            {failedCount}
          </span>
          <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>failed</span>
        </div>

        {runningCount > 0 && (
          <>
            <div style={{ width: 1, height: 28, background: 'var(--border)', flexShrink: 0 }} />
            <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
              <span
                style={{
                  fontFamily: 'var(--font-mono)',
                  fontSize: 22,
                  fontWeight: 700,
                  color: 'var(--accent)',
                  lineHeight: 1,
                }}
              >
                {runningCount}
              </span>
              <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>running</span>
            </div>
          </>
        )}
      </div>

      {/* Deployment table */}
      <div style={S.card}>
        <div
          style={{
            ...S.cardTitle,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
          }}
        >
          <span>Deployment History</span>
          <span style={{ color: 'var(--text-faint)' }}>{targets.length} records</span>
        </div>
        <div style={{ overflowX: 'auto' }}>
          <table style={{ width: '100%', borderCollapse: 'collapse' }}>
            <thead>
              <tr>
                <th style={S.th}>Deployment</th>
                <th style={S.th}>Patch</th>
                <th style={S.th}>Status</th>
                <th style={S.th}>Started</th>
                <th style={S.th}>Completed</th>
                <th style={S.th}>Duration</th>
                <th style={{ ...S.th, width: 28 }} />
              </tr>
            </thead>
            <tbody>
              {targets.map((target) => {
                const key = target.id;
                const expanded = expandedRows.has(key);
                const statusColor = STATUS_COLOR[target.status] ?? 'var(--text-muted)';

                return (
                  <Fragment key={key}>
                    <tr
                      style={{ cursor: 'pointer' }}
                      onMouseEnter={(e) => {
                        (e.currentTarget as HTMLTableRowElement).style.background =
                          'var(--bg-card-hover)';
                      }}
                      onMouseLeave={(e) => {
                        (e.currentTarget as HTMLTableRowElement).style.background = '';
                      }}
                    >
                      <td style={S.td}>
                        <Link
                          to={`/deployments/${target.deployment_id}`}
                          style={{
                            fontFamily: 'var(--font-mono)',
                            fontSize: 12,
                            color: 'var(--accent)',
                            textDecoration: 'none',
                          }}
                          onMouseEnter={(e) => {
                            (e.currentTarget as HTMLAnchorElement).style.textDecoration =
                              'underline';
                          }}
                          onMouseLeave={(e) => {
                            (e.currentTarget as HTMLAnchorElement).style.textDecoration = 'none';
                          }}
                        >
                          {shortId(target.deployment_id)}
                        </Link>
                      </td>
                      <td
                        style={{
                          ...S.td,
                          fontFamily: 'var(--font-mono)',
                          fontSize: 11,
                          color: 'var(--text-secondary)',
                        }}
                      >
                        {target.patch_id.slice(0, 8)}…
                      </td>
                      <td style={S.td}>
                        <span
                          style={{
                            display: 'inline-flex',
                            alignItems: 'center',
                            gap: 5,
                            fontFamily: 'var(--font-mono)',
                            fontSize: 12,
                            color: statusColor,
                          }}
                        >
                          <span
                            style={{
                              width: 6,
                              height: 6,
                              borderRadius: '50%',
                              background: statusColor,
                              flexShrink: 0,
                            }}
                          />
                          {target.status}
                        </span>
                      </td>
                      <td
                        style={{
                          ...S.td,
                          fontFamily: 'var(--font-mono)',
                          fontSize: 11,
                          color: 'var(--text-secondary)',
                          whiteSpace: 'nowrap',
                        }}
                      >
                        <span>{formatDateTime(target.started_at)}</span>
                        {target.started_at && (
                          <span style={{ color: 'var(--text-muted)', fontSize: 10, marginLeft: 4 }}>
                            ({timeAgo(target.started_at)})
                          </span>
                        )}
                      </td>
                      <td
                        style={{
                          ...S.td,
                          fontFamily: 'var(--font-mono)',
                          fontSize: 11,
                          color: 'var(--text-secondary)',
                          whiteSpace: 'nowrap',
                        }}
                      >
                        {formatDateTime(target.completed_at)}
                      </td>
                      <td
                        style={{
                          ...S.td,
                          fontFamily: 'var(--font-mono)',
                          fontSize: 11,
                          color: 'var(--text-muted)',
                        }}
                      >
                        {target.started_at && target.completed_at
                          ? formatDuration(target.started_at, target.completed_at)
                          : target.duration_seconds
                            ? `${target.duration_seconds}s`
                            : '—'}
                      </td>
                      <td style={S.td}>
                        <button
                          onClick={() => toggleExpand(key)}
                          style={{
                            background: 'none',
                            border: 'none',
                            cursor: 'pointer',
                            color: 'var(--text-muted)',
                            fontSize: 10,
                            padding: 0,
                            transition: 'transform 0.15s',
                            display: 'inline-block',
                            transform: expanded ? 'rotate(90deg)' : 'none',
                          }}
                        >
                          ▶
                        </button>
                      </td>
                    </tr>
                    {expanded && (
                      <tr key={`${key}-expanded`}>
                        <td
                          colSpan={7}
                          style={{
                            padding: '12px 16px',
                            borderBottom: '1px solid var(--border)',
                            background: 'var(--bg-inset)',
                          }}
                        >
                          <pre
                            style={{
                              background: 'var(--bg-page)',
                              border: '1px solid var(--border)',
                              borderRadius: 6,
                              padding: '12px 14px',
                              fontSize: 11,
                              color: 'var(--text-secondary)',
                              overflowX: 'auto',
                              fontFamily: 'var(--font-mono)',
                              margin: 0,
                            }}
                          >
                            {target.error_message || 'No output available'}
                          </pre>
                        </td>
                      </tr>
                    )}
                  </Fragment>
                );
              })}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
