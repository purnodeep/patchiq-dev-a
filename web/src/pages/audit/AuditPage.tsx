import { useState, useCallback, useMemo } from 'react';
import { Download, Info, FileText, ChevronDown } from 'lucide-react';
import {
  Button,
  Skeleton,
  ErrorState,
  EmptyState,
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from '@patchiq/ui';
import { useAuditLog } from '../../api/hooks/useAudit';
import { DataTablePagination } from '../../components/data-table';
import { AuditFilters } from './AuditFilters';
import { ActivityStream } from './ActivityStream';
import { TimelineView } from './TimelineView';

type ViewMode = 'stream' | 'timeline';

function dateRangeToISO(range: string): { from?: string; to?: string } {
  const now = new Date();
  if (range === '24h') {
    const from = new Date(now.getTime() - 24 * 60 * 60 * 1000);
    return { from: from.toISOString() };
  }
  if (range === '7d') {
    const from = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000);
    return { from: from.toISOString() };
  }
  if (range === '30d') {
    const from = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000);
    return { from: from.toISOString() };
  }
  return {};
}

function categoryToSearchPrefix(eventType: string): string | undefined {
  if (!eventType || eventType === '__all__') return undefined;
  // "system" maps to events not covered by other categories (catalog, notification, workflow, license, etc.)
  if (eventType === 'system') return 'catalog';
  return eventType;
}

// ─── Stat Card ────────────────────────────────────────────────────────────────

interface StatCardProps {
  label: string;
  value: number;
  valueColor?: string;
}

function StatCard({ label, value, valueColor }: StatCardProps) {
  const [hovered, setHovered] = useState(false);
  return (
    <div
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        flex: 1,
        minWidth: 0,
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'flex-start',
        padding: '12px 14px',
        background: 'var(--bg-card)',
        border: `1px solid ${hovered ? 'var(--border-hover)' : 'var(--border)'}`,
        borderRadius: 8,
        transition: 'all 0.15s',
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
        {value}
      </span>
      <span
        style={{
          fontSize: 10,
          fontFamily: 'var(--font-mono)',
          fontWeight: 500,
          textTransform: 'uppercase',
          letterSpacing: '0.05em',
          color: 'var(--text-muted)',
          marginTop: 4,
        }}
      >
        {label}
      </span>
    </div>
  );
}

// ─── Main Page ────────────────────────────────────────────────────────────────

export const AuditPage = () => {
  const [viewMode, setViewMode] = useState<ViewMode>('stream');
  const [eventType, setEventType] = useState('__all__');
  const [actorSearch, setActorSearch] = useState('');
  const [resource, setResource] = useState('__all__');
  const [dateRange, setDateRange] = useState('30d');
  const [fromDate, setFromDate] = useState('');
  const [toDate, setToDate] = useState('');
  const [cursors, setCursors] = useState<string[]>([]);
  const [expandedId, setExpandedId] = useState<string | null>(null);

  const currentCursor = cursors[cursors.length - 1];
  const resetCursors = useCallback(() => setCursors([]), []);

  const { from: computedFrom, to: computedTo } = useMemo(() => {
    if (dateRange === 'custom') {
      return {
        from: fromDate ? new Date(fromDate).toISOString() : undefined,
        to: toDate ? new Date(toDate + 'T23:59:59').toISOString() : undefined,
      };
    }
    const { from, to } = dateRangeToISO(dateRange);
    return { from, to };
  }, [dateRange, fromDate, toDate]);

  const searchPrefix = categoryToSearchPrefix(eventType);
  const excludeType = !searchPrefix && eventType === '__all__' ? 'heartbeat.received' : undefined;

  const { data, isLoading, isError, refetch } = useAuditLog({
    cursor: currentCursor,
    limit: 50,
    actor_id: actorSearch || undefined,
    resource: resource !== '__all__' ? resource : undefined,
    search: searchPrefix,
    exclude_type: excludeType,
    from_date: computedFrom,
    to_date: computedTo,
  });

  const events = data?.data ?? [];

  // ─── Stat Counts ──────────────────────────────────────────────────────────
  const stats = useMemo(() => {
    const todayStart = new Date();
    todayStart.setHours(0, 0, 0, 0);

    let system = 0;
    let user = 0;
    let today = 0;

    for (const ev of events) {
      const actorId = ev.actor_id ?? '';
      // System events have no actor_id or have "system" actor
      if (!actorId || actorId === 'system') {
        system++;
      } else {
        user++;
      }
      const ts = ev.timestamp ?? '';
      if (ts && new Date(ts) >= todayStart) {
        today++;
      }
    }

    return { total: events.length, system, user, today };
  }, [events]);

  const handleExport = useCallback(
    async (format: 'csv' | 'json') => {
      const params = new URLSearchParams();
      params.set('format', format);
      if (actorSearch) params.set('actor_id', actorSearch);
      if (resource !== '__all__') params.set('resource', resource);
      if (searchPrefix) params.set('search', searchPrefix);
      if (computedFrom) params.set('from_date', computedFrom);
      if (computedTo) params.set('to_date', computedTo);
      if (excludeType) params.set('exclude_type', excludeType);

      const response = await fetch(`/api/v1/audit/export?${params.toString()}`, {
        credentials: 'include',
      });
      if (!response.ok) return;

      const blob = await response.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `audit-export-${new Date().toISOString().slice(0, 10)}.${format === 'csv' ? 'csv' : 'json'}`;
      a.click();
      URL.revokeObjectURL(url);
    },
    [actorSearch, resource, searchPrefix, excludeType, computedFrom, computedTo],
  );

  const handleToggleExpand = useCallback((id: string) => {
    setExpandedId((prev) => (prev === id ? null : id));
  }, []);

  const handleEventTypeChange = (v: string) => {
    setEventType(v);
    resetCursors();
  };
  const handleActorSearchChange = (v: string) => {
    setActorSearch(v);
    resetCursors();
  };
  const handleResourceChange = (v: string) => {
    setResource(v);
    resetCursors();
  };
  const handleDateRangeChange = (v: string) => {
    setDateRange(v);
    resetCursors();
  };
  const handleFromDateChange = (v: string) => {
    setFromDate(v);
    resetCursors();
  };
  const handleToDateChange = (v: string) => {
    setToDate(v);
    resetCursors();
  };

  if (isError) {
    return (
      <div style={{ padding: 24 }}>
        <ErrorState
          title="Failed to load audit events"
          message="An unexpected error occurred. Please try again."
          onRetry={refetch}
        />
      </div>
    );
  }

  return (
    <div
      style={{
        background: 'var(--bg-page)',
        minHeight: '100%',
        padding: '24px',
        display: 'flex',
        flexDirection: 'column',
        gap: 16,
      }}
    >
      {/* ─── Stat Cards ───────────────────────────────────────────────────────── */}
      <div style={{ display: 'flex', gap: 8 }}>
        <StatCard label="Total" value={stats.total} />
        <StatCard label="System" value={stats.system} valueColor="var(--text-secondary)" />
        <StatCard label="User" value={stats.user} valueColor="var(--accent)" />
        <StatCard label="Today" value={stats.today} valueColor="var(--signal-healthy)" />
      </div>

      {/* ─── View Toggle Tabs ─────────────────────────────────────────────────── */}
      <div
        style={{
          display: 'flex',
          borderBottom: '1px solid var(--border)',
          width: 'fit-content',
          gap: 0,
        }}
      >
        {(['stream', 'timeline'] as ViewMode[]).map((mode) => {
          const isActive = viewMode === mode;
          return (
            <button
              key={mode}
              onClick={() => setViewMode(mode)}
              style={{
                padding: '6px 16px',
                fontFamily: 'var(--font-mono)',
                fontSize: 11,
                fontWeight: isActive ? 600 : 400,
                color: isActive ? 'var(--text-emphasis)' : 'var(--text-muted)',
                background: 'transparent',
                border: 'none',
                borderBottom: `2px solid ${isActive ? 'var(--accent)' : 'transparent'}`,
                cursor: 'pointer',
                transition: 'color 0.1s ease, border-color 0.1s ease',
                marginBottom: -1,
              }}
            >
              {mode === 'stream' ? 'Activity Stream' : 'Timeline View'}
            </button>
          );
        })}
      </div>

      {/* ─── Filter Bar + Actions ─────────────────────────────────────────────── */}
      <div style={{ display: 'flex', alignItems: 'stretch', gap: 8 }}>
        <AuditFilters
          eventType={eventType}
          onEventTypeChange={handleEventTypeChange}
          actorSearch={actorSearch}
          onActorSearchChange={handleActorSearchChange}
          resource={resource}
          onResourceChange={handleResourceChange}
          dateRange={dateRange}
          onDateRangeChange={handleDateRangeChange}
          fromDate={fromDate}
          onFromDateChange={handleFromDateChange}
          toDate={toDate}
          onToDateChange={handleToDateChange}
        />

        {/* Actions Card */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 8,
            padding: '10px 12px',
            boxShadow: 'var(--shadow-sm)',
          }}
        >
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                variant="outline"
                size="sm"
                style={{ fontFamily: 'var(--font-mono)', fontSize: 11 }}
              >
                <Download style={{ width: 12, height: 12, marginRight: 6 }} />
                Export
                <ChevronDown style={{ width: 10, height: 10, marginLeft: 4 }} />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => handleExport('csv')}>Export as CSV</DropdownMenuItem>
              <DropdownMenuItem onClick={() => handleExport('json')}>
                Export as JSON
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      {/* ─── Content ──────────────────────────────────────────────────────────── */}
      {isLoading ? (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
          {Array.from({ length: 8 }).map((_, i) => (
            <Skeleton key={i} className="h-12 rounded-lg" />
          ))}
        </div>
      ) : events.length === 0 ? (
        <EmptyState
          icon={FileText}
          title="No audit events"
          description="Events will appear as actions are performed in the system."
        />
      ) : (
        <>
          {viewMode === 'stream' ? (
            <ActivityStream
              events={events}
              expandedId={expandedId}
              onToggleExpand={handleToggleExpand}
            />
          ) : (
            <TimelineView events={events} />
          )}

          {(data?.next_cursor || cursors.length > 0) && (
            <DataTablePagination
              hasNext={!!data?.next_cursor}
              hasPrev={cursors.length > 0}
              onNext={() => {
                if (data?.next_cursor) setCursors((prev) => [...prev, data.next_cursor!]);
              }}
              onPrev={() => setCursors((prev) => prev.slice(0, -1))}
            />
          )}
        </>
      )}

      {/* ─── Retention Bar ────────────────────────────────────────────────────── */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 10,
          background: 'var(--bg-card)',
          border: '1px solid var(--border)',
          borderRadius: 8,
          padding: '10px 16px',
        }}
      >
        <Info style={{ width: 13, height: 13, color: 'var(--text-muted)', flexShrink: 0 }} />
        <span
          style={{
            fontFamily: 'var(--font-sans)',
            fontSize: 11,
            color: 'var(--text-muted)',
          }}
        >
          Audit logs retained for 365 days. Oldest entry: March 12, 2025.{' '}
          <button
            style={{
              background: 'none',
              border: 'none',
              cursor: 'pointer',
              padding: 0,
              fontFamily: 'var(--font-sans)',
              fontSize: 11,
              color: 'var(--text-secondary)',
              textDecoration: 'underline',
            }}
          >
            Manage Retention Policy
          </button>
        </span>
      </div>
    </div>
  );
};
