import { useState } from 'react';
import { Skeleton } from '@patchiq/ui';
import { FileText } from 'lucide-react';
import { useAuditLog } from '../../../api/hooks/useAudit';
import { EmptyState } from '../../../components/EmptyState';
import { getEventCategory } from '../../../lib/audit-utils';
import { ActivityStream } from '../../audit/ActivityStream';

interface AuditTabProps {
  endpointId: string;
}

const CATEGORY_OPTIONS = [
  { value: 'all', label: 'All Events' },
  { value: 'Endpoint', label: 'Endpoint' },
  { value: 'Deployment', label: 'Deployment' },
  { value: 'Policy', label: 'Policy' },
  { value: 'Compliance', label: 'Compliance' },
  { value: 'Patch', label: 'Patch / CVE' },
  { value: 'Auth', label: 'Auth' },
  { value: 'System', label: 'System' },
] as const;

const DAY_RANGES = [1, 3, 7, 14, 30] as const;

export function AuditTab({ endpointId }: AuditTabProps) {
  const [categoryFilter, setCategoryFilter] = useState<string>('all');
  const [actorSearch, setActorSearch] = useState('');
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [dayRange, setDayRange] = useState<number>(7);

  // Two queries: direct endpoint events + related events (deployments, commands)
  // that reference this endpoint in their payload.
  const {
    data: directData,
    isLoading: directLoading,
    error: directError,
  } = useAuditLog({
    resource_id: endpointId,
    exclude_type: 'heartbeat.received',
    limit: 50,
  });

  const {
    data: relatedData,
    isLoading: relatedLoading,
    error: relatedError,
  } = useAuditLog({
    search: endpointId,
    limit: 50,
  });

  const isLoading = directLoading || relatedLoading;
  const error = directError || relatedError;

  // Merge and deduplicate by event ID, sort by timestamp desc
  type AuditEvent = NonNullable<typeof directData>['data'][number];
  const mergedMap = new Map<string, AuditEvent>();
  for (const e of directData?.data ?? []) mergedMap.set(e.id, e);
  for (const e of relatedData?.data ?? []) mergedMap.set(e.id, e);
  const events = [...mergedMap.values()].sort(
    (a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime(),
  );

  const cutoff = new Date(Date.now() - dayRange * 86400000);
  const filtered = events.filter((event) => {
    const category = getEventCategory(event.type);
    if (categoryFilter !== 'all' && category !== categoryFilter) return false;
    if (actorSearch && !event.actor_id.toLowerCase().includes(actorSearch.toLowerCase()))
      return false;
    if (new Date(event.timestamp) < cutoff) return false;
    return true;
  });

  const handleToggleExpand = (id: string) => setExpandedId((prev) => (prev === id ? null : id));

  // ─── Filter Bar (matches main audit page visual language) ───────────────
  const filterBar = (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 8,
        flexWrap: 'wrap',
        background: 'var(--bg-card)',
        border: '1px solid var(--border)',
        borderRadius: 8,
        padding: '10px 14px',
        boxShadow: 'var(--shadow-sm)',
      }}
    >
      {/* Category pills */}
      <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap' }}>
        {CATEGORY_OPTIONS.map((opt) => {
          const active = categoryFilter === opt.value;
          return (
            <button
              key={opt.value}
              type="button"
              onClick={() => setCategoryFilter(opt.value)}
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 11,
                padding: '5px 10px',
                borderRadius: 6,
                border: `1px solid ${active ? 'var(--accent)' : 'var(--border)'}`,
                background: active ? 'var(--bg-inset)' : 'var(--bg-card)',
                color: active ? 'var(--accent)' : 'var(--text-secondary)',
                cursor: 'pointer',
                transition: 'all 0.1s',
              }}
            >
              {opt.label}
            </button>
          );
        })}
      </div>

      <select
        aria-label="Day range"
        value={dayRange}
        onChange={(e) => setDayRange(Number(e.target.value))}
        style={{
          background: 'var(--bg-card)',
          border: '1px solid var(--border)',
          borderRadius: 6,
          padding: '5px 10px',
          fontSize: 11.5,
          color: 'var(--text-secondary)',
          outline: 'none',
          cursor: 'pointer',
          fontFamily: 'var(--font-mono)',
        }}
      >
        {DAY_RANGES.map((d) => (
          <option key={d} value={d}>
            Last {d} day{d > 1 ? 's' : ''}
          </option>
        ))}
      </select>

      {/* Actor search */}
      <div
        style={{
          marginLeft: 'auto',
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          padding: '5px 10px',
          border: '1px solid var(--border)',
          borderRadius: 6,
          background: 'var(--bg-inset)',
        }}
      >
        <svg
          width="12"
          height="12"
          viewBox="0 0 24 24"
          fill="none"
          stroke="var(--text-muted)"
          strokeWidth="2.5"
          aria-hidden="true"
        >
          <circle cx="11" cy="11" r="8" />
          <path d="M21 21l-4.35-4.35" />
        </svg>
        <input
          type="text"
          value={actorSearch}
          onChange={(e) => setActorSearch(e.target.value)}
          placeholder="Search by actor..."
          aria-label="Search by actor"
          style={{
            background: 'transparent',
            border: 'none',
            outline: 'none',
            fontSize: 12,
            color: 'var(--text-primary)',
            width: 180,
            fontFamily: 'var(--font-mono)',
          }}
        />
      </div>
    </div>
  );

  if (isLoading) {
    return (
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        {filterBar}
        {Array.from({ length: 5 }).map((_, i) => (
          <Skeleton key={i} className="h-16 w-full rounded-lg" />
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        {filterBar}
        <div
          style={{
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 8,
            boxShadow: 'var(--shadow-sm)',
            padding: 16,
          }}
        >
          <span style={{ fontSize: 13, color: 'var(--signal-critical)' }}>
            Error loading audit events
          </span>
        </div>
      </div>
    );
  }

  if (events.length === 0) {
    return (
      <EmptyState
        icon={FileText}
        title="No audit events"
        description="No audit events have been recorded for this endpoint yet."
      />
    );
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      {filterBar}
      <ActivityStream
        events={filtered}
        expandedId={expandedId}
        onToggleExpand={handleToggleExpand}
      />
    </div>
  );
}
