import { useState, useMemo } from 'react';
import { Link } from 'react-router';
import {
  getCoreRowModel,
  getExpandedRowModel,
  useReactTable,
  createColumnHelper,
  type ExpandedState,
} from '@tanstack/react-table';
import {
  Button,
  EmptyState,
  ErrorState,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@patchiq/ui';
import { RefreshCw, Loader2, ChevronDown, ChevronRight } from 'lucide-react';
import { useFeeds, useUpdateFeed, useTriggerFeedSync } from '../../api/hooks/useFeeds';
import { Sparkline } from '../../components/Sparkline';
import { formatRelativeTime, formatFutureTime } from '../../lib/format';
import { getFeedIconConfig } from '../../lib/feed-icons';
import type { Feed } from '../../types/feed';
import { AddFeedForm } from './AddFeedForm';
import { FilterBar, FilterSearch } from '../../components/FilterBar';
import { DataTable } from '../../components/data-table/DataTable';

// ─── Helpers ──────────────────────────────────────────────────────────────────

function getEffectiveStatus(feed: Feed): 'active' | 'failing' | 'disabled' | 'never_synced' {
  if (!feed.enabled) return 'disabled';
  const recentHistory = feed.recent_history ?? [];
  const recentNonEmpty = recentHistory.filter(
    (h) => h.status === 'success' || h.status === 'failed',
  );
  const effectivelyFailing =
    feed.status === 'idle' &&
    recentNonEmpty.length >= 5 &&
    recentNonEmpty.slice(0, 5).every((h) => h.status === 'failed');
  if (feed.status === 'error' || effectivelyFailing) return 'failing';
  if (feed.status === 'never_synced') return 'never_synced';
  return 'active';
}

// ─── Stat Card ────────────────────────────────────────────────────────────────

interface StatCardProps {
  label: string;
  value: number | undefined;
  valueColor?: string;
  active?: boolean;
  onClick: () => void;
}

function StatCard({ label, value, valueColor, active, onClick }: StatCardProps) {
  const [hovered, setHovered] = useState(false);
  return (
    <button
      type="button"
      onClick={onClick}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        flex: 1,
        minWidth: 0,
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'flex-start',
        padding: '12px 14px',
        background: active ? 'color-mix(in srgb, white 3%, transparent)' : 'var(--bg-card)',
        border: `1px solid ${active ? (valueColor ?? 'var(--accent)') : hovered ? 'var(--border-hover)' : 'var(--border)'}`,
        borderRadius: 8,
        cursor: 'pointer',
        transition: 'all 0.15s',
        outline: 'none',
        textAlign: 'left',
      }}
    >
      <span
        style={{
          fontFamily: 'var(--font-mono)',
          fontSize: 22,
          fontWeight: 700,
          lineHeight: 1,
          color: valueColor ?? 'var(--text-emphasis)',
          letterSpacing: '-0.02em',
        }}
      >
        {value ?? '—'}
      </span>
      <span
        style={{
          fontSize: 10,
          fontFamily: 'var(--font-mono)',
          fontWeight: 500,
          textTransform: 'uppercase',
          letterSpacing: '0.05em',
          color: active ? (valueColor ?? 'var(--accent)') : 'var(--text-muted)',
          marginTop: 4,
        }}
      >
        {label}
      </span>
    </button>
  );
}

// ─── Skeleton Rows ────────────────────────────────────────────────────────────

function SkeletonRows({ cols, rows = 8 }: { cols: number; rows?: number }) {
  return (
    <>
      {Array.from({ length: rows }).map((_, i) => (
        <tr key={i}>
          {Array.from({ length: cols }).map((__, j) => (
            <td key={j} style={{ padding: '10px 12px' }}>
              <div
                style={{
                  height: 14,
                  borderRadius: 4,
                  background: 'var(--bg-inset)',
                  width: j === 0 ? '60%' : j === 1 ? '80%' : '50%',
                  animation: 'pulse 1.5s ease-in-out infinite',
                }}
              />
            </td>
          ))}
        </tr>
      ))}
    </>
  );
}

// ─── Status Cell ──────────────────────────────────────────────────────────────

function FeedStatusCell({ feed }: { feed: Feed }) {
  const effectiveStatus = getEffectiveStatus(feed);

  if (effectiveStatus === 'disabled') {
    return (
      <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
        <span
          style={{
            height: 7,
            width: 7,
            borderRadius: '50%',
            background: 'var(--text-muted)',
            flexShrink: 0,
          }}
        />
        <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>Disabled</span>
      </div>
    );
  }
  if (effectiveStatus === 'failing') {
    return (
      <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
        <span
          style={{
            height: 7,
            width: 7,
            borderRadius: '50%',
            background: 'var(--signal-critical)',
            flexShrink: 0,
          }}
        />
        <span style={{ fontSize: 12, color: 'var(--signal-critical)' }}>Failing</span>
      </div>
    );
  }
  if (effectiveStatus === 'never_synced') {
    return (
      <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
        <span
          style={{
            height: 7,
            width: 7,
            borderRadius: '50%',
            background: 'var(--text-muted)',
            flexShrink: 0,
          }}
        />
        <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>Never Synced</span>
      </div>
    );
  }
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
      <span style={{ position: 'relative', display: 'flex', height: 7, width: 7 }}>
        <span
          style={{
            position: 'absolute',
            display: 'inline-flex',
            height: '100%',
            width: '100%',
            borderRadius: '50%',
            opacity: 0.75,
            background: 'var(--accent)',
            animation: 'ping 1s cubic-bezier(0,0,0.2,1) infinite',
          }}
        />
        <span
          style={{
            position: 'relative',
            display: 'inline-flex',
            borderRadius: '50%',
            height: 7,
            width: 7,
            background: 'var(--accent)',
          }}
        />
      </span>
      <span style={{ fontSize: 12, color: 'var(--accent)' }}>
        {feed.status === 'syncing' ? 'Syncing' : 'Active'}
      </span>
    </div>
  );
}

// ─── Expanded Row ─────────────────────────────────────────────────────────────

function ExpandedFeedRow({ feed }: { feed: Feed }) {
  const effectiveStatus = getEffectiveStatus(feed);
  const sectionLabel: React.CSSProperties = {
    fontFamily: 'var(--font-mono)',
    fontSize: 10,
    fontWeight: 600,
    textTransform: 'uppercase',
    letterSpacing: '0.06em',
    color: 'var(--text-muted)',
    marginBottom: 8,
  };

  return (
    <div
      style={{
        padding: '16px 48px 16px 20px',
        display: 'grid',
        gridTemplateColumns: '1fr 1fr 1fr',
        gap: 24,
        borderLeft: '2px solid var(--accent)',
      }}
    >
      {/* Sync history sparkline */}
      <div>
        <div style={sectionLabel}>Sync History</div>
        <Sparkline data={feed.recent_history ?? []} />
        {feed.last_error && effectiveStatus === 'failing' && (
          <div
            style={{
              marginTop: 10,
              padding: '8px 12px',
              borderRadius: 6,
              border: '1px solid color-mix(in srgb, var(--signal-critical) 30%, transparent)',
              background: 'color-mix(in srgb, var(--signal-critical) 8%, transparent)',
              fontSize: 11,
              color: 'var(--signal-critical)',
            }}
          >
            <div style={{ fontWeight: 600, marginBottom: 2 }}>Last Error</div>
            <div style={{ color: 'var(--text-muted)', lineHeight: 1.4 }}>{feed.last_error}</div>
          </div>
        )}
      </div>

      {/* Configuration summary */}
      <div>
        <div style={sectionLabel}>Configuration</div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
          {[
            {
              label: 'Total Entries',
              value: feed.entries_ingested.toLocaleString(),
              color: 'var(--text-primary)',
            },
            {
              label: 'New This Week',
              value: feed.new_this_week > 0 ? `+${feed.new_this_week}` : '0',
              color: feed.new_this_week > 0 ? 'var(--accent)' : 'var(--text-muted)',
            },
            {
              label: 'Error Rate',
              value: `${feed.error_rate?.toFixed(1) ?? '0.0'}%`,
              color: (feed.error_rate ?? 0) > 5 ? 'var(--signal-critical)' : 'var(--accent)',
            },
            {
              label: 'Next Sync',
              value: formatFutureTime(feed.next_sync_at),
              color: 'var(--text-muted)',
            },
          ].map(({ label, value, color }) => (
            <div
              key={label}
              style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12 }}
            >
              <span style={{ color: 'var(--text-secondary)' }}>{label}</span>
              <span
                style={{ fontWeight: 600, color, fontFamily: 'var(--font-mono)', fontSize: 12 }}
              >
                {value}
              </span>
            </div>
          ))}
          {feed.url && (
            <div style={{ marginTop: 4 }}>
              <span
                style={{
                  fontFamily: 'var(--font-mono)',
                  fontSize: 10,
                  color: 'var(--text-muted)',
                  wordBreak: 'break-all',
                }}
                title={feed.url}
              >
                {feed.url}
              </span>
            </div>
          )}
        </div>
      </div>

      {/* Actions */}
      <div>
        <div style={sectionLabel}>Actions</div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
          <Link
            to={`/feeds/${feed.id}`}
            style={{
              display: 'inline-block',
              padding: '6px 12px',
              fontSize: 12,
              fontWeight: 500,
              borderRadius: 6,
              border: '1px solid var(--border)',
              color: 'var(--text-primary)',
              textDecoration: 'none',
              textAlign: 'center',
            }}
            onClick={(e) => e.stopPropagation()}
          >
            View Details
          </Link>
        </div>
      </div>
    </div>
  );
}

// ─── Column helper ────────────────────────────────────────────────────────────

const columnHelper = createColumnHelper<Feed>();

// ─── Main Page ────────────────────────────────────────────────────────────────

type StatusFilter = '' | 'active' | 'failing' | 'disabled';

export function FeedsPage() {
  const { data, isLoading, isError, error } = useFeeds();
  const updateFeed = useUpdateFeed();
  const triggerSync = useTriggerFeedSync();
  const [addFeedOpen, setAddFeedOpen] = useState(false);
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('');
  const [search, setSearch] = useState('');
  const [expanded, setExpanded] = useState<ExpandedState>({});

  const feeds = data ?? [];

  const stats = useMemo(() => {
    const total = feeds.length;
    const active = feeds.filter((f) => getEffectiveStatus(f) === 'active').length;
    const failing = feeds.filter((f) => getEffectiveStatus(f) === 'failing').length;
    const disabled = feeds.filter((f) => getEffectiveStatus(f) === 'disabled').length;
    return { total, active, failing, disabled };
  }, [feeds]);

  const filteredFeeds = useMemo(() => {
    let result = feeds;
    if (statusFilter) {
      result = result.filter((f) => getEffectiveStatus(f) === statusFilter);
    }
    if (search) {
      const q = search.toLowerCase();
      result = result.filter(
        (f) =>
          f.name.toLowerCase().includes(q) ||
          f.display_name.toLowerCase().includes(q) ||
          (f.url?.toLowerCase().includes(q) ?? false),
      );
    }
    return result;
  }, [feeds, statusFilter, search]);

  const columns = useMemo(
    () => [
      columnHelper.display({
        id: 'expand',
        header: '',
        cell: (info) => {
          const isExp = info.row.getIsExpanded();
          return (
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation();
                info.row.toggleExpanded();
              }}
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                background: 'none',
                border: 'none',
                cursor: 'pointer',
                padding: 2,
                color: 'var(--text-muted)',
              }}
            >
              {isExp ? (
                <ChevronDown style={{ width: 14, height: 14 }} />
              ) : (
                <ChevronRight style={{ width: 14, height: 14 }} />
              )}
            </button>
          );
        },
      }),
      columnHelper.accessor('name', {
        header: 'Feed Name',
        cell: (info) => {
          const feed = info.row.original;
          const { emoji, bgClass } = getFeedIconConfig(feed.name);
          return (
            <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
              <div
                className={bgClass}
                style={{
                  width: 36,
                  height: 36,
                  borderRadius: 8,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: 18,
                  flexShrink: 0,
                }}
              >
                {emoji}
              </div>
              <div>
                <Link
                  to={`/feeds/${feed.id}`}
                  style={{
                    fontWeight: 600,
                    fontSize: 13,
                    color: 'var(--text-emphasis)',
                    textDecoration: 'none',
                  }}
                  onClick={(e) => e.stopPropagation()}
                >
                  {feed.name}
                </Link>
                <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>{feed.display_name}</div>
              </div>
            </div>
          );
        },
      }),
      columnHelper.display({
        id: 'status',
        header: 'Status',
        cell: (info) => <FeedStatusCell feed={info.row.original} />,
      }),
      columnHelper.accessor('last_sync_at', {
        header: 'Last Sync',
        cell: (info) => {
          const val = info.getValue();
          return (
            <span style={{ fontSize: 12, color: 'var(--text-secondary)' }}>
              {val ? formatRelativeTime(val) : 'Never'}
            </span>
          );
        },
      }),
      columnHelper.accessor('entries_ingested', {
        header: 'Entries',
        cell: (info) => (
          <span
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 13,
              fontWeight: 600,
              color: 'var(--text-primary)',
            }}
          >
            {info.row.original.enabled ? info.getValue().toLocaleString() : '—'}
          </span>
        ),
      }),
      columnHelper.accessor('error_rate', {
        header: 'Error Rate',
        cell: (info) => {
          const rate = info.getValue() ?? 0;
          const color =
            rate > 5
              ? 'var(--signal-critical)'
              : rate > 0
                ? 'var(--signal-warning)'
                : 'var(--accent)';
          return (
            <span style={{ fontFamily: 'var(--font-mono)', fontSize: 12, color }}>
              {info.row.original.enabled ? `${rate.toFixed(1)}%` : '—'}
            </span>
          );
        },
      }),
      columnHelper.display({
        id: 'actions',
        header: 'Actions',
        cell: (info) => {
          const feed = info.row.original;
          const effectiveStatus = getEffectiveStatus(feed);
          return (
            <div style={{ display: 'flex', gap: 6 }}>
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation();
                  triggerSync.mutate(feed.id);
                }}
                disabled={triggerSync.isPending || !feed.enabled}
                style={{
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: 4,
                  padding: '4px 10px',
                  fontSize: 11,
                  borderRadius: 4,
                  border:
                    effectiveStatus === 'failing'
                      ? '1px solid color-mix(in srgb, var(--signal-critical) 30%, transparent)'
                      : 'none',
                  cursor: triggerSync.isPending || !feed.enabled ? 'not-allowed' : 'pointer',
                  opacity: !feed.enabled ? 0.5 : 1,
                  background:
                    effectiveStatus === 'failing'
                      ? 'color-mix(in srgb, var(--signal-critical) 10%, transparent)'
                      : 'var(--accent)',
                  color:
                    effectiveStatus === 'failing'
                      ? 'var(--signal-critical)'
                      : 'var(--text-emphasis)',
                }}
              >
                {triggerSync.isPending ? (
                  <Loader2
                    style={{ width: 11, height: 11, animation: 'spin 1s linear infinite' }}
                  />
                ) : (
                  <RefreshCw style={{ width: 11, height: 11 }} />
                )}
                {effectiveStatus === 'failing' ? 'Retry' : 'Sync'}
              </button>
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation();
                  updateFeed.mutate({ id: feed.id, data: { enabled: !feed.enabled } });
                }}
                disabled={updateFeed.isPending}
                style={{
                  padding: '4px 10px',
                  fontSize: 11,
                  borderRadius: 4,
                  border: '1px solid var(--border)',
                  background: 'transparent',
                  cursor: updateFeed.isPending ? 'not-allowed' : 'pointer',
                  color: 'var(--text-secondary)',
                }}
              >
                {feed.enabled ? 'Disable' : 'Enable'}
              </button>
            </div>
          );
        },
      }),
    ],
    [triggerSync, updateFeed],
  );

  const table = useReactTable({
    data: filteredFeeds,
    columns,
    state: { expanded },
    onExpandedChange: setExpanded,
    getCoreRowModel: getCoreRowModel(),
    getExpandedRowModel: getExpandedRowModel(),
  });

  return (
    <div style={{ padding: 24 }}>
      {/* Page Header */}

      <AddFeedForm open={addFeedOpen} onOpenChange={setAddFeedOpen} />

      {/* Stat Cards */}
      {!isLoading && !isError && (
        <div style={{ display: 'flex', gap: 8, marginBottom: 16 }}>
          <StatCard
            label="Total Feeds"
            value={stats.total}
            active={statusFilter === ''}
            onClick={() => setStatusFilter('')}
          />
          <StatCard
            label="Active / Healthy"
            value={stats.active}
            valueColor="var(--accent)"
            active={statusFilter === 'active'}
            onClick={() => setStatusFilter(statusFilter === 'active' ? '' : 'active')}
          />
          <StatCard
            label="Failing"
            value={stats.failing}
            valueColor="var(--signal-critical)"
            active={statusFilter === 'failing'}
            onClick={() => setStatusFilter(statusFilter === 'failing' ? '' : 'failing')}
          />
          <StatCard
            label="Disabled"
            value={stats.disabled}
            valueColor="var(--text-muted)"
            active={statusFilter === 'disabled'}
            onClick={() => setStatusFilter(statusFilter === 'disabled' ? '' : 'disabled')}
          />
        </div>
      )}

      {/* Filter Bar */}
      <FilterBar>
        <Select
          value={statusFilter || 'all'}
          onValueChange={(v: string) => {
            setStatusFilter((v === 'all' ? '' : v) as StatusFilter);
          }}
        >
          <SelectTrigger
            className="h-7 w-32 text-sm"
            style={{ borderColor: 'var(--border)', background: 'var(--bg-card)' }}
          >
            <SelectValue placeholder="All" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">
              All{' '}
              <span
                style={{ color: 'var(--text-muted)', fontFamily: 'var(--font-mono)', fontSize: 10 }}
              >
                {stats.total}
              </span>
            </SelectItem>
            <SelectItem value="active">
              Active{' '}
              <span
                style={{ color: 'var(--text-muted)', fontFamily: 'var(--font-mono)', fontSize: 10 }}
              >
                {stats.active}
              </span>
            </SelectItem>
            <SelectItem value="failing">
              Failing{' '}
              <span
                style={{ color: 'var(--text-muted)', fontFamily: 'var(--font-mono)', fontSize: 10 }}
              >
                {stats.failing}
              </span>
            </SelectItem>
            <SelectItem value="disabled">
              Disabled{' '}
              <span
                style={{ color: 'var(--text-muted)', fontFamily: 'var(--font-mono)', fontSize: 10 }}
              >
                {stats.disabled}
              </span>
            </SelectItem>
          </SelectContent>
        </Select>
        <FilterSearch value={search} onChange={setSearch} placeholder="Search feeds..." />
        <div style={{ marginLeft: 'auto' }}>
          <Button size="sm" onClick={() => setAddFeedOpen(true)}>
            + Add Feed
          </Button>
        </div>
      </FilterBar>

      {/* Table */}
      {isLoading ? (
        <div style={{ borderRadius: 8, border: '1px solid var(--border)', overflow: 'hidden' }}>
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13 }}>
            <thead>
              <tr>
                {['', 'Feed Name', 'Status', 'Last Sync', 'Entries', 'Error Rate', 'Actions'].map(
                  (h) => (
                    <th
                      key={h}
                      style={{
                        height: 40,
                        padding: '0 16px',
                        textAlign: 'left',
                        fontFamily: 'var(--font-mono)',
                        fontSize: 11,
                        fontWeight: 600,
                        textTransform: 'uppercase',
                        letterSpacing: '0.05em',
                        color: 'var(--text-muted)',
                        background: 'var(--bg-inset)',
                        borderBottom: '1px solid var(--border)',
                      }}
                    >
                      {h}
                    </th>
                  ),
                )}
              </tr>
            </thead>
            <tbody>
              <SkeletonRows cols={7} rows={6} />
            </tbody>
          </table>
        </div>
      ) : isError ? (
        <ErrorState
          title="Failed to load feeds"
          message={error instanceof Error ? error.message : 'An unknown error occurred'}
        />
      ) : filteredFeeds.length === 0 ? (
        <EmptyState
          title="No feeds found"
          description={
            statusFilter || search ? 'Try adjusting your filters' : 'No feeds configured yet'
          }
          action={{ label: '+ Add Feed', onClick: () => setAddFeedOpen(true) }}
        />
      ) : (
        <DataTable
          table={table}
          isRowFailed={(feed) => getEffectiveStatus(feed) === 'failing'}
          onRowClick={(feed) => {
            const row = table.getRowModel().rows.find((r) => r.original === feed);
            row?.toggleExpanded();
          }}
          renderExpandedRow={(feed) => <ExpandedFeedRow feed={feed} />}
        />
      )}
    </div>
  );
}
