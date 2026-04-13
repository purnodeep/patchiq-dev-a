import { useState } from 'react';
import {
  Skeleton,
  Tooltip,
  TooltipTrigger,
  TooltipContent,
  TooltipProvider,
  ErrorState,
} from '@patchiq/ui';
import { ChevronDown, ChevronUp, AlertTriangle } from 'lucide-react';
import { useHistory } from '../../api/hooks/useHistory';
import type { components } from '../../api/types';

type HistoryEntry = components['schemas']['HistoryEntry'] & {
  stdout?: string;
  stderr?: string;
  reboot_required?: boolean;
  duration_seconds?: number | null;
  attempt?: number;
  exit_code?: number | null;
};

const CARD_STYLE: React.CSSProperties = {
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  borderRadius: 'var(--radius-lg)',
  padding: '12px 14px',
  flex: 1,
};

const SELECT_STYLE: React.CSSProperties = {
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  borderRadius: '6px',
  color: 'var(--text-emphasis)',
  padding: '6px 10px',
  fontSize: '13px',
  cursor: 'pointer',
};

const resultColors: Record<string, { color: string; border: string; bg: string; label: string }> = {
  success: {
    color: 'var(--signal-healthy)',
    border: 'color-mix(in srgb, var(--signal-healthy) 30%, transparent)',
    bg: 'color-mix(in srgb, var(--signal-healthy) 10%, transparent)',
    label: 'Success',
  },
  failed: {
    color: 'var(--signal-critical)',
    border: 'color-mix(in srgb, var(--signal-critical) 30%, transparent)',
    bg: 'color-mix(in srgb, var(--signal-critical) 10%, transparent)',
    label: 'Failed',
  },
  pending: {
    color: 'var(--signal-warning)',
    border: 'color-mix(in srgb, var(--signal-warning) 30%, transparent)',
    bg: 'color-mix(in srgb, var(--signal-warning) 10%, transparent)',
    label: 'Pending',
  },
  rolled_back: {
    color: 'var(--signal-warning)',
    border: 'color-mix(in srgb, var(--signal-warning) 30%, transparent)',
    bg: 'color-mix(in srgb, var(--signal-warning) 10%, transparent)',
    label: 'Rolled Back',
  },
  cancelled: {
    color: 'var(--text-muted)',
    border: 'var(--border)',
    bg: 'transparent',
    label: 'Cancelled',
  },
  skipped: {
    color: 'var(--signal-warning)',
    border: 'color-mix(in srgb, var(--signal-warning) 30%, transparent)',
    bg: 'color-mix(in srgb, var(--signal-warning) 10%, transparent)',
    label: 'Skipped',
  },
};

const resultNodeColor: Record<string, string> = {
  success: 'var(--signal-healthy)',
  failed: 'var(--signal-critical)',
  pending: 'var(--signal-warning)',
  rolled_back: 'var(--signal-warning)',
  cancelled: 'var(--text-muted)',
  skipped: 'var(--signal-warning)',
};

const resultBadgeStyle = (result: string): React.CSSProperties => {
  const c = resultColors[result] ?? resultColors.cancelled;
  return {
    fontSize: '11px',
    padding: '2px 8px',
    borderRadius: '4px',
    border: `1px solid ${c.border}`,
    background: c.bg,
    color: c.color,
  };
};

const actionBadgeStyle = (action: string): React.CSSProperties => {
  const colors: Record<string, { color: string; border: string }> = {
    install: { color: 'var(--text-secondary)', border: 'var(--border)' },
    rollback: {
      color: 'var(--signal-warning)',
      border: 'color-mix(in srgb, var(--signal-warning) 30%, transparent)',
    },
    scan: { color: 'var(--text-secondary)', border: 'var(--border)' },
    skip: { color: 'var(--text-muted)', border: 'var(--border)' },
  };
  const c = colors[action] ?? colors.skip;
  return {
    fontSize: '11px',
    padding: '2px 8px',
    borderRadius: '4px',
    border: `1px solid ${c.border}`,
    color: c.color,
  };
};

function formatDuration(seconds: number | null | undefined): string {
  if (seconds == null) return '\u2014';
  if (seconds < 60) return `${seconds}s`;
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  return `${m}m ${s}s`;
}

function formatRelativeTime(iso: string): string {
  const diff = Math.floor((Date.now() - new Date(iso).getTime()) / 1000);
  if (diff < 60) return `${diff}s ago`;
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  const days = Math.floor(diff / 86400);
  if (days === 1) return 'yesterday';
  if (days < 30) return `${days}d ago`;
  return new Date(iso).toLocaleDateString();
}

function groupByDate(entries: HistoryEntry[]): [string, HistoryEntry[]][] {
  const map = new Map<string, HistoryEntry[]>();
  for (const e of entries) {
    const day = new Date(e.completed_at)
      .toLocaleDateString('en-US', {
        month: 'long',
        day: 'numeric',
        year: 'numeric',
      })
      .toUpperCase();
    if (!map.has(day)) map.set(day, []);
    map.get(day)!.push(e);
  }
  return Array.from(map.entries());
}

function SummaryStats({ entries }: { entries: HistoryEntry[] }) {
  const total = entries.length;
  const successCount = entries.filter((e) => e.result === 'success').length;
  const failedCount = entries.filter((e) => e.result === 'failed').length;
  const successRate = total > 0 ? Math.round((successCount / total) * 100) : 0;

  const stats = [
    { label: 'Total Deployments', value: total, color: 'var(--text-emphasis)' },
    { label: 'Success Rate', value: `${successRate}%`, color: 'var(--signal-healthy)' },
    {
      label: 'Failed',
      value: failedCount,
      color: failedCount > 0 ? 'var(--signal-critical)' : 'var(--text-muted)',
    },
  ];

  return (
    <div style={{ display: 'flex', gap: '12px' }}>
      {stats.map((s) => (
        <div
          key={s.label}
          style={{
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 'var(--radius-lg)',
            padding: '12px 16px',
            flex: 1,
            textAlign: 'center',
          }}
        >
          <div
            style={{
              fontSize: '24px',
              fontWeight: 700,
              color: s.color,
              fontFamily: 'var(--font-mono)',
            }}
          >
            {s.value}
          </div>
          <div
            style={{
              fontSize: '11px',
              color: 'var(--text-muted)',
              marginTop: '4px',
              textTransform: 'uppercase',
              letterSpacing: '0.05em',
            }}
          >
            {s.label}
          </div>
        </div>
      ))}
    </div>
  );
}

function EventCard({ entry }: { entry: HistoryEntry }) {
  const [expanded, setExpanded] = useState(false);
  const nodeColor = resultNodeColor[entry.result] ?? 'var(--text-muted)';
  const hasDetails = !!(entry.stdout || entry.stderr);

  return (
    <div style={{ display: 'flex', gap: '12px', marginBottom: '12px' }}>
      {/* Relative time */}
      <TooltipProvider delayDuration={200}>
        <Tooltip>
          <TooltipTrigger asChild>
            <span
              style={{
                fontSize: '11px',
                fontFamily: 'var(--font-mono)',
                color: 'var(--text-muted)',
                minWidth: '72px',
                paddingTop: '10px',
                textAlign: 'right',
                flexShrink: 0,
                cursor: 'default',
              }}
            >
              {formatRelativeTime(entry.completed_at)}
            </span>
          </TooltipTrigger>
          <TooltipContent side="left">
            <span style={{ fontSize: '11px' }}>
              {new Date(entry.completed_at).toLocaleString()}
            </span>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>

      {/* Timeline node + line */}
      <div
        style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', flexShrink: 0 }}
      >
        <div
          style={{
            width: '14px',
            height: '14px',
            borderRadius: '50%',
            background: nodeColor,
            marginTop: '8px',
            border: '2px solid var(--bg-canvas)',
            flexShrink: 0,
          }}
        />
        <div style={{ width: '2px', flex: 1, background: 'var(--border)', marginTop: '4px' }} />
      </div>

      {/* Event card */}
      <div
        style={{
          ...CARD_STYLE,
          cursor: hasDetails ? 'pointer' : 'default',
        }}
        onClick={hasDetails ? () => setExpanded((e) => !e) : undefined}
      >
        <div
          style={{
            display: 'flex',
            alignItems: 'flex-start',
            justifyContent: 'space-between',
            gap: '8px',
            flexWrap: 'wrap',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px', flexWrap: 'wrap' }}>
            <span style={actionBadgeStyle(entry.action)}>{entry.action}</span>
            <span
              style={{
                fontSize: '13px',
                fontWeight: 600,
                fontFamily: 'var(--font-mono)',
                color: 'var(--text-emphasis)',
              }}
            >
              {entry.patch_name}
            </span>
            <span
              style={{
                fontSize: '12px',
                fontFamily: 'var(--font-mono)',
                color: 'var(--text-muted)',
              }}
            >
              {entry.patch_version}
            </span>
            <span style={resultBadgeStyle(entry.result)}>
              {resultColors[entry.result]?.label ?? entry.result}
            </span>
            {entry.reboot_required && (
              <span
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '4px',
                  fontSize: '10px',
                  padding: '2px 6px',
                  borderRadius: '4px',
                  border: 'color-mix(in srgb, var(--signal-warning) 30%, transparent)',
                  background: 'color-mix(in srgb, var(--signal-warning) 10%, transparent)',
                  color: 'var(--signal-warning)',
                }}
              >
                <AlertTriangle style={{ width: '10px', height: '10px' }} />
                Reboot
              </span>
            )}
          </div>
          {hasDetails && (
            <button
              onClick={(e) => {
                e.stopPropagation();
                setExpanded((x) => !x);
              }}
              style={{
                fontSize: '11px',
                color: 'var(--text-muted)',
                background: 'none',
                border: 'none',
                cursor: 'pointer',
                flexShrink: 0,
                display: 'flex',
                alignItems: 'center',
                gap: '4px',
              }}
            >
              {expanded ? (
                <ChevronUp style={{ width: '14px', height: '14px' }} />
              ) : (
                <ChevronDown style={{ width: '14px', height: '14px' }} />
              )}
              Details
            </button>
          )}
        </div>

        <div
          style={{
            display: 'flex',
            flexWrap: 'wrap',
            gap: '12px',
            marginTop: '6px',
            fontSize: '11px',
            color: 'var(--text-muted)',
          }}
        >
          <span>{formatDuration(entry.duration_seconds)}</span>
          {entry.attempt != null && entry.attempt > 1 && <span>attempt {entry.attempt}</span>}
          {entry.exit_code != null && entry.exit_code !== 0 && (
            <span>exit code {entry.exit_code}</span>
          )}
        </div>

        {entry.error_message && (
          <p style={{ marginTop: '8px', fontSize: '12px', color: 'var(--signal-critical)' }}>
            {entry.error_message}
          </p>
        )}

        {expanded && (
          <div style={{ marginTop: '10px', display: 'flex', flexDirection: 'column', gap: '8px' }}>
            {entry.stdout && (
              <div>
                <div
                  style={{
                    fontSize: '10px',
                    color: 'var(--text-muted)',
                    marginBottom: '4px',
                    textTransform: 'uppercase',
                  }}
                >
                  stdout
                </div>
                <pre
                  style={{
                    background: 'var(--bg-canvas)',
                    border: '1px solid var(--border)',
                    borderRadius: '6px',
                    padding: '10px',
                    fontSize: '11px',
                    fontFamily: 'var(--font-mono)',
                    color: 'var(--text-secondary)',
                    overflowX: 'auto',
                    whiteSpace: 'pre-wrap',
                    margin: 0,
                  }}
                >
                  {entry.stdout}
                </pre>
              </div>
            )}
            {entry.stderr && (
              <div>
                <div
                  style={{
                    fontSize: '10px',
                    color: 'var(--text-muted)',
                    marginBottom: '4px',
                    textTransform: 'uppercase',
                  }}
                >
                  stderr
                </div>
                <pre
                  style={{
                    background: 'var(--bg-canvas)',
                    border: '1px solid var(--signal-critical)',
                    borderRadius: '6px',
                    padding: '10px',
                    fontSize: '11px',
                    fontFamily: 'var(--font-mono)',
                    color: 'var(--signal-critical)',
                    overflowX: 'auto',
                    whiteSpace: 'pre-wrap',
                    margin: 0,
                  }}
                >
                  {entry.stderr}
                </pre>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

const DATE_RANGES = [
  { label: 'All Time', value: '' as const },
  { label: 'Last 24h', value: '24h' as const },
  { label: 'Last 7d', value: '7d' as const },
  { label: 'Last 30d', value: '30d' as const },
];

export const HistoryPage = () => {
  const [dateRange, setDateRange] = useState<'' | '24h' | '7d' | '30d'>('');
  const [actionFilter, setActionFilter] = useState('all');

  const { data, isLoading, isError } = useHistory({
    date_range: dateRange || undefined,
    limit: 100,
  });

  const all = data?.data ?? [];
  const entries = actionFilter === 'all' ? all : all.filter((e) => e.action === actionFilter);
  const grouped = groupByDate(entries);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      {/* Subtitle */}
      <p style={{ fontSize: '13px', color: 'var(--text-muted)', margin: 0 }}>
        Patch deployment history &mdash; records of patches installed, updated, or rolled back on
        this endpoint
      </p>

      {/* Summary stats */}
      {!isLoading && !isError && all.length > 0 && <SummaryStats entries={all} />}

      {/* Filter bar */}
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: '12px', alignItems: 'center' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <span
            style={{
              fontSize: '10px',
              textTransform: 'uppercase',
              color: 'var(--text-muted)',
              letterSpacing: '0.1em',
            }}
          >
            Action
          </span>
          <select
            value={actionFilter}
            onChange={(e) => setActionFilter(e.target.value)}
            style={SELECT_STYLE}
          >
            <option value="all">All</option>
            <option value="install">Install</option>
            <option value="rollback">Rollback</option>
            <option value="scan">Scan</option>
            <option value="skip">Skip</option>
          </select>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <span
            style={{
              fontSize: '10px',
              textTransform: 'uppercase',
              color: 'var(--text-muted)',
              letterSpacing: '0.1em',
            }}
          >
            Date
          </span>
          <select
            value={dateRange}
            onChange={(e) => setDateRange(e.target.value as typeof dateRange)}
            style={SELECT_STYLE}
          >
            {DATE_RANGES.map((r) => (
              <option key={r.value} value={r.value}>
                {r.label}
              </option>
            ))}
          </select>
        </div>
      </div>

      {isLoading && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-20 w-full" />
          ))}
        </div>
      )}

      {isError && <ErrorState message="Failed to load history." />}

      {!isLoading && !isError && entries.length === 0 && (
        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            gap: '8px',
            padding: '48px 20px',
            color: 'var(--text-muted)',
          }}
        >
          <p style={{ fontWeight: 500, margin: 0 }}>No deployment history yet</p>
          <p style={{ fontSize: '12px', color: 'var(--text-faint)', margin: 0 }}>
            Records will appear after patches are deployed to this endpoint.
          </p>
        </div>
      )}

      {!isLoading && grouped.length > 0 && (
        <div>
          {grouped.map(([day, dayEntries]) => (
            <div key={day} style={{ marginBottom: '24px' }}>
              {/* Date separator */}
              <div
                style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '16px' }}
              >
                <div style={{ flex: 1, height: '1px', background: 'var(--border)' }} />
                <span
                  style={{
                    fontSize: '11px',
                    color: 'var(--text-muted)',
                    letterSpacing: '0.05em',
                    fontVariant: 'small-caps',
                    whiteSpace: 'nowrap',
                  }}
                >
                  {day}
                </span>
                <div style={{ flex: 1, height: '1px', background: 'var(--border)' }} />
              </div>
              <div>
                {dayEntries.map((entry) => (
                  <EventCard key={entry.id} entry={entry} />
                ))}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};
