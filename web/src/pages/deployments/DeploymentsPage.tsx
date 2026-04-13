import { useState, useMemo, useCallback, Fragment } from 'react';
import { formatDeploymentId } from '../../lib/format';
import { Link, useNavigate, useSearchParams } from 'react-router';
import {
  getCoreRowModel,
  useReactTable,
  createColumnHelper,
  flexRender,
} from '@tanstack/react-table';
import { Plus, ChevronDown, ChevronRight } from 'lucide-react';
import { Skeleton, ErrorState } from '@patchiq/ui';
import {
  useDeployments,
  useCancelDeployment,
  useRetryDeployment,
  useRollbackDeployment,
} from '../../api/hooks/useDeployments';
import { usePolicies } from '../../api/hooks/usePolicies';
import { DataTablePagination } from '../../components/data-table';
import { DeploymentExpandedRow } from './components/DeploymentExpandedRow';
import { DeploymentWizard } from '../../components/DeploymentWizard';
import { useCan } from '../../app/auth/AuthContext';
import type { components } from '../../api/types';

type Deployment = components['schemas']['Deployment'];

// ── Helpers ──────────────────────────────────────────────────────────────────

function displayUser(id: string | null | undefined): string {
  if (!id) return 'System';
  if (id.startsWith('cc000000')) return 'Admin';
  return id.slice(0, 8) + '\u2026';
}

function deploymentDisplayName(d: Deployment, policyName?: string): string {
  if (d.name) return d.name;
  if (d.policy_name) return d.policy_name;
  if (policyName) return policyName;
  return formatDeploymentId(d.id);
}

function elapsed(created: string): string {
  const ms = Date.now() - new Date(created).getTime();
  const h = Math.floor(ms / 3600000);
  const m = Math.floor((ms % 3600000) / 60000);
  if (h > 24) return `${Math.floor(h / 24)}d ${h % 24}h`;
  return h > 0 ? `${h}h ${m}m` : `${m}m`;
}

function computeDuration(d: Deployment): string {
  if (d.status === 'running' || d.status === 'rolling_back') return elapsed(d.created_at);
  if (d.status === 'created' || d.status === 'scheduled') return '\u2014';
  if (!d.started_at || !d.completed_at) return '\u2014';
  const ms = new Date(d.completed_at).getTime() - new Date(d.started_at).getTime();
  const mins = Math.floor(ms / 60000);
  const secs = Math.floor((ms % 60000) / 1000);
  if (mins >= 60) return `${Math.floor(mins / 60)}h ${mins % 60}m`;
  return `${mins}m ${secs.toString().padStart(2, '0')}s`;
}

function computeSegments(d: Deployment) {
  const succeeded = d.success_count;
  const failed = d.failed_count;
  const active = Math.max(
    0,
    d.status === 'running' ? d.completed_count - d.success_count - d.failed_count : 0,
  );
  const pending = Math.max(0, d.target_count - d.completed_count);
  return { succeeded, failed, active, pending };
}

// ── Segmented bar (inline, pure CSS tokens) ──────────────────────────────────

function SegBar({ d }: { d: Deployment }) {
  if (d.target_count === 0)
    return (
      <span style={{ color: 'var(--text-muted)', fontFamily: 'var(--font-mono)', fontSize: 11 }}>
        —
      </span>
    );
  const { succeeded, active, failed, pending } = computeSegments(d);
  const total = d.target_count;

  return (
    <div style={{ minWidth: 130 }}>
      <div
        style={{
          display: 'flex',
          height: 5,
          borderRadius: 3,
          overflow: 'hidden',
          background: 'var(--bg-inset)',
          gap: 1,
        }}
      >
        {succeeded > 0 && (
          <div
            style={{
              flex: succeeded / total,
              background: 'var(--signal-healthy)',
              borderRadius: 2,
            }}
          />
        )}
        {active > 0 && (
          <div style={{ flex: active / total, background: 'var(--accent)', borderRadius: 2 }} />
        )}
        {failed > 0 && (
          <div
            style={{ flex: failed / total, background: 'var(--signal-critical)', borderRadius: 2 }}
          />
        )}
        {pending > 0 && (
          <div style={{ flex: pending / total, background: 'var(--border)', borderRadius: 2 }} />
        )}
      </div>
      <div
        style={{
          marginTop: 4,
          fontFamily: 'var(--font-mono)',
          fontSize: 10,
          color: 'var(--text-muted)',
        }}
      >
        {d.completed_count}/{total}
      </div>
    </div>
  );
}

// ── Status cell ───────────────────────────────────────────────────────────────

const statusColorMap: Record<string, string> = {
  running: 'var(--signal-healthy)',
  completed: 'var(--signal-healthy)',
  failed: 'var(--signal-critical)',
  rollback_failed: 'var(--signal-critical)',
  rolling_back: 'var(--signal-warning)',
  rolled_back: 'var(--text-muted)',
  scheduled: 'var(--text-muted)',
  created: 'var(--text-muted)',
  cancelled: 'var(--text-muted)',
};

const statusLabelMap: Record<string, string> = {
  running: 'Running',
  completed: 'Completed',
  failed: 'Failed',
  rollback_failed: 'Rollback Failed',
  rolling_back: 'Rolling Back',
  rolled_back: 'Rolled Back',
  scheduled: 'Scheduled',
  created: 'Created',
  cancelled: 'Cancelled',
};

function StatusCell({ status }: { status: string }) {
  const color = statusColorMap[status] ?? 'var(--text-muted)';
  const isLive = status === 'running' || status === 'rolling_back';
  return (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 6,
        fontFamily: 'var(--font-mono)',
        fontSize: 11,
        fontWeight: 500,
        color,
      }}
    >
      {isLive && (
        <span
          style={{
            width: 6,
            height: 6,
            borderRadius: '50%',
            background: color,
            flexShrink: 0,
            animation: 'pulse-dot 1.5s ease-in-out infinite',
          }}
        />
      )}
      {statusLabelMap[status] ?? status}
    </span>
  );
}

// ── Stat Card ─────────────────────────────────────────────────────────────────

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

// ── Filter config ─────────────────────────────────────────────────────────────

type StatusFilter = string | null;

const statusFilters: {
  label: string;
  value: StatusFilter;
  variant: 'default' | 'critical' | 'medium';
}[] = [
  { label: 'All', value: null, variant: 'default' },
  { label: 'Running', value: 'running', variant: 'default' },
  { label: 'Completed', value: 'completed', variant: 'default' },
  { label: 'Failed', value: 'failed', variant: 'critical' },
  { label: 'Created', value: 'created', variant: 'medium' },
  { label: 'Scheduled', value: 'scheduled', variant: 'default' },
  { label: 'Cancelled', value: 'cancelled', variant: 'default' },
];

const col = createColumnHelper<Deployment>();

// ── Table header / cell shared styles ────────────────────────────────────────

const TH_STYLE: React.CSSProperties = {
  padding: '9px 12px',
  textAlign: 'left',
  fontFamily: 'var(--font-mono)',
  fontSize: 11,
  fontWeight: 600,
  textTransform: 'uppercase',
  letterSpacing: '0.05em',
  color: 'var(--text-muted)',
  whiteSpace: 'nowrap',
  background: 'var(--bg-inset)',
  borderBottom: '1px solid var(--border)',
};

const TD_STYLE: React.CSSProperties = {
  padding: '10px 12px',
  verticalAlign: 'middle',
  borderBottom: '1px solid var(--border)',
};

// ── SortHeader ───────────────────────────────────────────────────────────────

function SortHeader({
  label,
  colKey,
  sortCol,
  sortDir,
  onSort,
}: {
  label: string;
  colKey: string;
  sortCol: string | null;
  sortDir: 'asc' | 'desc';
  onSort: (col: string) => void;
}) {
  const active = sortCol === colKey;
  const [hovered, setHovered] = useState(false);
  return (
    <th
      style={{
        ...TH_STYLE,
        cursor: 'pointer',
        userSelect: 'none',
      }}
      onClick={() => onSort(colKey)}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
        <span style={{ color: active ? 'var(--text-emphasis)' : undefined }}>{label}</span>
        <svg
          width="10"
          height="10"
          viewBox="0 0 10 10"
          fill="none"
          style={{
            opacity: active ? 1 : hovered ? 0.5 : 0,
            transition: 'opacity 0.15s',
            flexShrink: 0,
          }}
        >
          {(!active || sortDir === 'asc') && (
            <path
              d="M5 2L8 5.5H2L5 2Z"
              fill={active ? 'var(--text-emphasis)' : 'var(--text-muted)'}
            />
          )}
          {(!active || sortDir === 'desc') && (
            <path
              d="M5 8L2 4.5H8L5 8Z"
              fill={active ? 'var(--text-emphasis)' : 'var(--text-muted)'}
            />
          )}
        </svg>
      </div>
    </th>
  );
}

// ── Deployment Card (grid view) ──────────────────────────────────────────────

const cardStatusColorMap: Record<string, string> = {
  running: 'var(--accent)',
  completed: 'var(--signal-healthy)',
  failed: 'var(--signal-critical)',
  rollback_failed: 'var(--signal-critical)',
  rolling_back: 'var(--signal-warning)',
  rolled_back: 'var(--signal-warning)',
  scheduled: 'var(--text-muted)',
  created: 'var(--text-secondary)',
  cancelled: 'var(--text-faint)',
};

function DeploymentCard({
  deployment: d,
  policyName,
  onNavigate,
  onRollback,
}: {
  deployment: Deployment;
  policyName?: string;
  onNavigate: () => void;
  onRollback: () => void;
}) {
  const can = useCan();
  const [hovered, setHovered] = useState(false);
  const label = deploymentDisplayName(d, policyName);
  const statusColor = cardStatusColorMap[d.status] ?? 'var(--text-muted)';
  const statusLabel = statusLabelMap[d.status] ?? d.status;
  const dur = computeDuration(d);
  const pct = d.target_count > 0 ? Math.round((d.completed_count / d.target_count) * 100) : 0;
  const canRollback = d.status === 'completed' || d.status === 'failed';

  return (
    <div
      onClick={onNavigate}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        background: hovered ? 'var(--bg-card-hover)' : 'var(--bg-card)',
        border: '1px solid var(--border)',
        borderRadius: 10,
        padding: 16,
        cursor: 'pointer',
        transition: 'background 0.15s, border-color 0.15s',
        borderColor: hovered ? 'var(--border-hover)' : 'var(--border)',
        display: 'flex',
        flexDirection: 'column',
        gap: 12,
      }}
    >
      {/* Header: name + status */}
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'flex-start',
          gap: 8,
        }}
      >
        <div style={{ minWidth: 0 }}>
          <div
            style={{
              fontSize: 13,
              fontWeight: 600,
              color: 'var(--text-emphasis)',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
            }}
          >
            {label}
          </div>
          {policyName && (
            <div
              style={{
                fontSize: 11,
                color: 'var(--text-muted)',
                marginTop: 2,
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
              }}
            >
              {policyName}
            </div>
          )}
        </div>
        <span
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: 5,
            padding: '2px 8px',
            borderRadius: 100,
            fontSize: 10,
            fontWeight: 600,
            fontFamily: 'var(--font-mono)',
            background: `color-mix(in srgb, ${statusColor} 12%, transparent)`,
            color: statusColor,
            border: `1px solid color-mix(in srgb, ${statusColor} 20%, transparent)`,
            whiteSpace: 'nowrap',
            flexShrink: 0,
          }}
        >
          {(d.status === 'running' || d.status === 'rolling_back') && (
            <span
              style={{
                width: 5,
                height: 5,
                borderRadius: '50%',
                background: statusColor,
                animation: 'pulse-dot 1.5s ease-in-out infinite',
              }}
            />
          )}
          {statusLabel}
        </span>
      </div>

      {/* Progress bar */}
      <div>
        <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
          <span
            style={{ fontSize: 10, fontFamily: 'var(--font-mono)', color: 'var(--text-muted)' }}
          >
            Progress
          </span>
          <span
            style={{
              fontSize: 10,
              fontFamily: 'var(--font-mono)',
              color: 'var(--text-secondary)',
              fontWeight: 600,
            }}
          >
            {pct}%
          </span>
        </div>
        <div
          style={{
            height: 5,
            borderRadius: 3,
            overflow: 'hidden',
            background: 'var(--bg-inset)',
            display: 'flex',
            gap: 1,
          }}
        >
          {d.success_count > 0 && (
            <div
              style={{
                flex: d.success_count / Math.max(d.target_count, 1),
                background: 'var(--signal-healthy)',
                borderRadius: 2,
              }}
            />
          )}
          {d.failed_count > 0 && (
            <div
              style={{
                flex: d.failed_count / Math.max(d.target_count, 1),
                background: 'var(--signal-critical)',
                borderRadius: 2,
              }}
            />
          )}
          {d.target_count - d.completed_count > 0 && (
            <div
              style={{
                flex: (d.target_count - d.completed_count) / Math.max(d.target_count, 1),
                background: 'var(--border)',
                borderRadius: 2,
              }}
            />
          )}
        </div>
      </div>

      {/* Stats row */}
      <div style={{ display: 'flex', gap: 12 }}>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
          <span
            style={{
              fontSize: 10,
              fontFamily: 'var(--font-mono)',
              color: 'var(--text-muted)',
              textTransform: 'uppercase',
              letterSpacing: '0.04em',
            }}
          >
            Targets
          </span>
          <span
            style={{
              fontSize: 13,
              fontFamily: 'var(--font-mono)',
              fontWeight: 700,
              color: 'var(--text-emphasis)',
            }}
          >
            {d.target_count}
          </span>
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
          <span
            style={{
              fontSize: 10,
              fontFamily: 'var(--font-mono)',
              color: 'var(--text-muted)',
              textTransform: 'uppercase',
              letterSpacing: '0.04em',
            }}
          >
            Succeeded
          </span>
          <span
            style={{
              fontSize: 13,
              fontFamily: 'var(--font-mono)',
              fontWeight: 700,
              color: d.success_count > 0 ? 'var(--signal-healthy)' : 'var(--text-muted)',
            }}
          >
            {d.success_count}
          </span>
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
          <span
            style={{
              fontSize: 10,
              fontFamily: 'var(--font-mono)',
              color: 'var(--text-muted)',
              textTransform: 'uppercase',
              letterSpacing: '0.04em',
            }}
          >
            Failed
          </span>
          <span
            style={{
              fontSize: 13,
              fontFamily: 'var(--font-mono)',
              fontWeight: 700,
              color: d.failed_count > 0 ? 'var(--signal-critical)' : 'var(--text-muted)',
            }}
          >
            {d.failed_count}
          </span>
        </div>
      </div>

      {/* Meta row */}
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          borderTop: '1px solid var(--border)',
          paddingTop: 10,
        }}
      >
        <div style={{ display: 'flex', gap: 12 }}>
          <span
            style={{ fontSize: 10, fontFamily: 'var(--font-mono)', color: 'var(--text-muted)' }}
          >
            {new Date(d.created_at).toLocaleDateString('en-US', {
              month: 'short',
              day: 'numeric',
              year: 'numeric',
            })}
          </span>
          <span
            style={{ fontSize: 10, fontFamily: 'var(--font-mono)', color: 'var(--text-muted)' }}
          >
            {dur}
          </span>
          <span
            style={{ fontSize: 10, fontFamily: 'var(--font-mono)', color: 'var(--text-muted)' }}
          >
            {displayUser(d.created_by)}
          </span>
        </div>
        <div style={{ display: 'flex', gap: 6 }}>
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation();
              onNavigate();
            }}
            style={{
              padding: '3px 8px',
              borderRadius: 4,
              fontSize: 10,
              fontWeight: 500,
              background: 'var(--bg-inset)',
              border: '1px solid var(--border)',
              color: 'var(--text-secondary)',
              cursor: 'pointer',
              fontFamily: 'var(--font-sans)',
            }}
          >
            View Details
          </button>
          {canRollback && (
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation();
                onRollback();
              }}
              disabled={!can('deployments', 'update')}
              title={!can('deployments', 'update') ? "You don't have permission" : undefined}
              style={{
                padding: '3px 8px',
                borderRadius: 4,
                fontSize: 10,
                fontWeight: 500,
                background: 'color-mix(in srgb, var(--signal-warning) 10%, transparent)',
                border: '1px solid color-mix(in srgb, var(--signal-warning) 25%, transparent)',
                color: 'var(--signal-warning)',
                cursor: !can('deployments', 'update') ? 'not-allowed' : 'pointer',
                fontFamily: 'var(--font-sans)',
                opacity: !can('deployments', 'update') ? 0.5 : 1,
              }}
            >
              Rollback
            </button>
          )}
        </div>
      </div>
    </div>
  );
}

// ── Main component ────────────────────────────────────────────────────────────

export function DeploymentsPage() {
  const can = useCan();
  const resolvedMode = (document.documentElement.classList.contains('dark') ? 'dark' : 'light') as
    | 'dark'
    | 'light';
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState<StatusFilter>(null);
  const [sortCol, setSortCol] = useState<string | null>(null);
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('asc');
  const [cursors, setCursors] = useState<string[]>([]);
  const [createOpen, setCreateOpen] = useState(false);
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set());
  const [dateFrom, setDateFrom] = useState('');
  const [dateTo, setDateTo] = useState('');
  const currentCursor = cursors[cursors.length - 1];
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const viewMode = (searchParams.get('view') === 'card' ? 'card' : 'list') as 'list' | 'card';
  const setViewMode = useCallback(
    (mode: 'list' | 'card') => {
      setSearchParams(
        (prev) => {
          const next = new URLSearchParams(prev);
          if (mode === 'list') next.delete('view');
          else next.set('view', mode);
          return next;
        },
        { replace: true },
      );
    },
    [setSearchParams],
  );

  const cancelMutation = useCancelDeployment();
  const retryMutation = useRetryDeployment();
  const rollbackMutation = useRollbackDeployment();

  const { data, isLoading, isError, refetch } = useDeployments({
    cursor: currentCursor,
    limit: 25,
    status: statusFilter ?? undefined,
    created_after: dateFrom ? `${dateFrom}T00:00:00Z` : undefined,
    created_before: dateTo ? `${dateTo}T23:59:59Z` : undefined,
  });

  const { data: policiesData } = usePolicies({ limit: 200 });

  const policyMap = useMemo(() => {
    const m = new Map<string, { name: string; id: string }>();
    if (policiesData?.data) {
      for (const p of policiesData.data) {
        m.set(p.id, { name: p.name, id: p.id });
      }
    }
    return m;
  }, [policiesData?.data]);

  const statusCounts = (data as { status_counts?: Record<string, number> })?.status_counts ?? {};

  const statCounts = useMemo(() => {
    const allDeployments = data?.data ?? [];
    let running = 0;
    let completed = 0;
    let failed = 0;
    for (const d of allDeployments) {
      if (d.status === 'running') running++;
      else if (d.status === 'completed') completed++;
      else if (d.status === 'failed' || d.status === 'rollback_failed') failed++;
    }
    return { running, completed, failed };
  }, [data?.data]);

  const filtered = useMemo(() => {
    if (!data?.data) return [];
    if (!search) return data.data;
    const lower = search.toLowerCase();
    return data.data.filter(
      (d) =>
        d.id.toLowerCase().includes(lower) ||
        d.status.toLowerCase().includes(lower) ||
        (d.name ?? '').toLowerCase().includes(lower) ||
        (d.policy_name ?? '').toLowerCase().includes(lower) ||
        (policyMap.get(d.policy_id)?.name ?? '').toLowerCase().includes(lower),
    );
  }, [data?.data, search, policyMap]);

  const toggleSort = useCallback(
    (col: string) => {
      if (sortCol === col) {
        if (sortDir === 'asc') setSortDir('desc');
        else if (sortDir === 'desc') {
          setSortCol(null);
          setSortDir('asc');
        }
      } else {
        setSortCol(col);
        setSortDir('asc');
      }
    },
    [sortCol, sortDir],
  );

  const sortedData = useMemo(() => {
    if (!sortCol) return filtered;
    const sorted = [...filtered].sort((a, b) => {
      switch (sortCol) {
        case 'id':
          return a.id.localeCompare(b.id);
        case 'status':
          return a.status.localeCompare(b.status);
        case 'created_at':
          return new Date(a.created_at).getTime() - new Date(b.created_at).getTime();
        case 'targets':
          return (a.target_count ?? 0) - (b.target_count ?? 0);
        case 'progress':
          return (
            a.completed_count / Math.max(a.target_count, 1) -
            b.completed_count / Math.max(b.target_count, 1)
          );
        case 'name': {
          const aName = deploymentDisplayName(a, policyMap.get(a.policy_id ?? '')?.name);
          const bName = deploymentDisplayName(b, policyMap.get(b.policy_id ?? '')?.name);
          return aName.localeCompare(bName);
        }
        case 'policy_id': {
          const aPolicy = policyMap.get(a.policy_id ?? '')?.name ?? '';
          const bPolicy = policyMap.get(b.policy_id ?? '')?.name ?? '';
          return aPolicy.localeCompare(bPolicy);
        }
        case 'duration': {
          const aDur = computeDuration(a);
          const bDur = computeDuration(b);
          return aDur.localeCompare(bDur);
        }
        case 'triggered_by': {
          const aUser = displayUser(a.created_by);
          const bUser = displayUser(b.created_by);
          return aUser.localeCompare(bUser);
        }
        default:
          return 0;
      }
    });
    return sortDir === 'desc' ? sorted.reverse() : sorted;
  }, [filtered, sortCol, sortDir, policyMap]);

  const toggleExpand = (id: string) => {
    setExpandedRows((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const columns = useMemo(
    () => [
      col.display({
        id: 'expand',
        header: '',
        cell: (info) => {
          const isExpanded = expandedRows.has(info.row.original.id);
          return (
            <button
              type="button"
              style={{
                background: 'none',
                border: 'none',
                cursor: 'pointer',
                color: 'var(--text-muted)',
                padding: '2px 4px',
                borderRadius: 4,
                display: 'flex',
                alignItems: 'center',
              }}
              onClick={(e) => {
                e.stopPropagation();
                toggleExpand(info.row.original.id);
              }}
            >
              {isExpanded ? (
                <ChevronDown style={{ width: 13, height: 13 }} />
              ) : (
                <ChevronRight style={{ width: 13, height: 13 }} />
              )}
            </button>
          );
        },
        size: 32,
      }),
      col.accessor('id', {
        header: 'ID',
        cell: (info) => (
          <Link
            to={`/deployments/${info.getValue()}`}
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 11,
              color: 'var(--accent)',
              textDecoration: 'none',
              letterSpacing: '0.02em',
            }}
            onClick={(e) => e.stopPropagation()}
          >
            {formatDeploymentId(info.getValue())}
          </Link>
        ),
      }),
      col.display({
        id: 'name',
        header: 'Name',
        cell: (info) => {
          const d = info.row.original;
          const pol = policyMap.get(d.policy_id ?? '');
          const label = deploymentDisplayName(d, pol?.name);
          return (
            <span style={{ fontSize: 12, color: 'var(--text-primary)', fontWeight: 500 }}>
              {label}
            </span>
          );
        },
      }),
      col.accessor('policy_id', {
        header: 'Policy',
        cell: (info) => {
          const pol = policyMap.get(info.getValue() ?? '');
          if (!pol) return <span style={{ color: 'var(--text-muted)', fontSize: 12 }}>—</span>;
          return (
            <Link
              to={`/policies/${pol.id}`}
              style={{
                fontSize: 12,
                color: 'var(--text-primary)',
                textDecoration: 'none',
              }}
              onClick={(e) => e.stopPropagation()}
              onMouseEnter={(e) => (e.currentTarget.style.color = 'var(--accent)')}
              onMouseLeave={(e) => (e.currentTarget.style.color = 'var(--text-primary)')}
            >
              {pol.name}
            </Link>
          );
        },
      }),
      col.accessor('status', {
        header: 'Status',
        cell: (info) => <StatusCell status={info.getValue()} />,
      }),
      col.display({
        id: 'progress',
        header: 'Progress',
        cell: (info) => <SegBar d={info.row.original} />,
      }),
      col.display({
        id: 'targets',
        header: 'Targets',
        cell: (info) => {
          const d = info.row.original;
          return (
            <span
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 11,
                color: 'var(--text-primary)',
                fontWeight: 600,
              }}
            >
              {d.success_count}
              <span style={{ color: 'var(--text-muted)', fontWeight: 400 }}>/{d.target_count}</span>
            </span>
          );
        },
      }),
      col.accessor('created_at', {
        header: 'Created',
        cell: (info) => (
          <span
            style={{ fontFamily: 'var(--font-mono)', fontSize: 11, color: 'var(--text-muted)' }}
          >
            {new Date(info.getValue()).toLocaleDateString('en-US', {
              month: 'short',
              day: 'numeric',
              year: 'numeric',
            })}
          </span>
        ),
      }),
      col.display({
        id: 'duration',
        header: 'Duration',
        cell: (info) => {
          const d = info.row.original;
          const dur = computeDuration(d);
          const isOngoing = d.status === 'running' || d.status === 'rolling_back';
          return (
            <span
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 11,
                color: isOngoing ? 'var(--accent)' : 'var(--text-muted)',
                fontStyle: isOngoing ? 'italic' : 'normal',
              }}
            >
              {dur}
            </span>
          );
        },
      }),
      col.display({
        id: 'triggered_by',
        header: 'By',
        cell: (info) => (
          <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>
            {displayUser(info.row.original.created_by)}
          </span>
        ),
      }),
    ],
    [expandedRows, policyMap],
  );

  const table = useReactTable({
    data: sortedData,
    columns,
    getCoreRowModel: getCoreRowModel(),
  });

  // ── Error state ────────────────────────────────────────────────────────────

  if (isError) {
    return (
      <div style={{ padding: 24, background: 'var(--bg-page)' }}>
        <ErrorState
          title="Failed to load deployments"
          message="Unable to fetch deployment data from the server."
          onRetry={refetch}
        />
      </div>
    );
  }

  // ── Loading state ─────────────────────────────────────────────────────────

  if (isLoading) {
    return (
      <div
        style={{
          padding: 24,
          background: 'var(--bg-page)',
          display: 'flex',
          flexDirection: 'column',
          gap: 16,
        }}
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          {Array.from({ length: 7 }).map((_, i) => (
            <Skeleton key={i} className="h-11 rounded-lg" />
          ))}
        </div>
        <DeploymentWizard open={createOpen} onOpenChange={setCreateOpen} />
      </div>
    );
  }

  // ── Main render ───────────────────────────────────────────────────────────

  return (
    <div
      style={{
        padding: 24,
        background: 'var(--bg-page)',
        minHeight: '100%',
        display: 'flex',
        flexDirection: 'column',
        gap: 16,
      }}
    >
      {/* ── Stat Cards ────────────────────────────────────────────────────── */}
      <div style={{ display: 'flex', gap: 8 }}>
        <StatCard
          label="Total"
          value={data?.total_count}
          active={statusFilter === null}
          onClick={() => {
            setStatusFilter(null);
            setCursors([]);
          }}
        />
        <StatCard
          label="Running"
          value={(statusCounts['running'] as number | undefined) ?? statCounts.running}
          valueColor="var(--accent)"
          active={statusFilter === 'running'}
          onClick={() => {
            setStatusFilter(statusFilter === 'running' ? null : 'running');
            setCursors([]);
          }}
        />
        <StatCard
          label="Completed"
          value={(statusCounts['completed'] as number | undefined) ?? statCounts.completed}
          valueColor="var(--signal-healthy)"
          active={statusFilter === 'completed'}
          onClick={() => {
            setStatusFilter(statusFilter === 'completed' ? null : 'completed');
            setCursors([]);
          }}
        />
        <StatCard
          label="Failed"
          value={(statusCounts['failed'] as number | undefined) ?? statCounts.failed}
          valueColor="var(--signal-critical)"
          active={statusFilter === 'failed'}
          onClick={() => {
            setStatusFilter(statusFilter === 'failed' ? null : 'failed');
            setCursors([]);
          }}
        />
      </div>

      {/* ── Filter Bar + Actions ──────────────────────────────────────────── */}
      <div style={{ display: 'flex', alignItems: 'stretch', gap: 8 }}>
        {/* Filter Bar */}
        <div
          style={{
            flex: 1,
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 8,
            padding: '10px 14px',
            boxShadow: 'var(--shadow-sm)',
            display: 'flex',
            alignItems: 'center',
            gap: 8,
            flexWrap: 'wrap',
          }}
        >
          {/* Search */}
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 8,
              padding: '5px 10px',
              background: 'var(--bg-inset)',
              border: '1px solid var(--border)',
              borderRadius: 6,
              flex: 1,
              maxWidth: 360,
              transition: 'border-color 0.15s',
            }}
            onFocusCapture={(e) => (e.currentTarget.style.borderColor = 'var(--accent)')}
            onBlurCapture={(e) => (e.currentTarget.style.borderColor = 'var(--border)')}
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
              aria-label="Search deployments"
              placeholder="Search deployments..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              style={{
                background: 'transparent',
                border: 'none',
                outline: 'none',
                fontSize: 12,
                color: 'var(--text-primary)',
                fontFamily: 'var(--font-sans)',
                width: '100%',
              }}
            />
            {search && (
              <button
                type="button"
                aria-label="Clear search"
                onClick={() => setSearch('')}
                style={{
                  width: 16,
                  height: 16,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  background: 'transparent',
                  border: 'none',
                  cursor: 'pointer',
                  color: 'var(--text-muted)',
                  padding: 0,
                }}
              >
                <svg
                  width="10"
                  height="10"
                  viewBox="0 0 10 10"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="1.5"
                >
                  <path d="M2 2l6 6M8 2l-6 6" />
                </svg>
              </button>
            )}
          </div>

          {/* Status filter pills */}
          <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
            {statusFilters.map((f) => {
              const active = statusFilter === f.value;
              const color =
                f.variant === 'critical'
                  ? 'var(--signal-critical)'
                  : f.variant === 'medium'
                    ? 'var(--signal-warning)'
                    : 'var(--accent)';
              return (
                <button
                  key={f.label}
                  type="button"
                  onClick={() => {
                    setStatusFilter(f.value);
                    setCursors([]);
                  }}
                  style={{
                    display: 'inline-flex',
                    alignItems: 'center',
                    padding: '2px 7px',
                    borderRadius: 100,
                    fontSize: 11,
                    fontWeight: 500,
                    cursor: 'pointer',
                    fontFamily: 'var(--font-sans)',
                    border: `1px solid ${active ? `color-mix(in srgb, ${color} 30%, transparent)` : 'transparent'}`,
                    background: active
                      ? `color-mix(in srgb, ${color} 10%, transparent)`
                      : 'transparent',
                    color: active ? color : 'var(--text-muted)',
                    transition: 'all 0.15s',
                  }}
                >
                  {f.label}
                </button>
              );
            })}
          </div>

          {/* Date range */}
          <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
            <input
              type="date"
              value={dateFrom}
              onChange={(e) => {
                setDateFrom(e.target.value);
                setCursors([]);
              }}
              style={{
                background: 'var(--bg-page)',
                border: '1px solid var(--border)',
                borderRadius: 6,
                padding: '4px 8px',
                fontSize: 11,
                color: 'var(--text-primary)',
                fontFamily: 'var(--font-mono)',
                colorScheme: resolvedMode,
                outline: 'none',
              }}
            />
            <span style={{ color: 'var(--text-muted)', fontSize: 11 }}>—</span>
            <input
              type="date"
              value={dateTo}
              onChange={(e) => {
                setDateTo(e.target.value);
                setCursors([]);
              }}
              style={{
                background: 'var(--bg-page)',
                border: '1px solid var(--border)',
                borderRadius: 6,
                padding: '4px 8px',
                fontSize: 11,
                color: 'var(--text-primary)',
                fontFamily: 'var(--font-mono)',
                colorScheme: resolvedMode,
                outline: 'none',
              }}
            />
          </div>

          {/* Spacer */}
          <div style={{ flex: 1 }} />

          {/* View toggle */}
          <div
            style={{
              display: 'flex',
              border: '1px solid var(--border)',
              borderRadius: 6,
              overflow: 'hidden',
            }}
          >
            <button
              type="button"
              onClick={() => setViewMode('list')}
              style={{
                padding: '5px 8px',
                background: viewMode === 'list' ? 'var(--bg-card)' : 'transparent',
                border: 'none',
                cursor: 'pointer',
                color: viewMode === 'list' ? 'var(--text-emphasis)' : 'var(--text-muted)',
                display: 'flex',
                alignItems: 'center',
              }}
            >
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                <path d="M2 3.5h10M2 7h10M2 10.5h10" stroke="currentColor" strokeWidth="1.2" />
              </svg>
            </button>
            <button
              type="button"
              onClick={() => setViewMode('card')}
              style={{
                padding: '5px 8px',
                background: viewMode === 'card' ? 'var(--bg-card)' : 'transparent',
                border: 'none',
                borderLeft: '1px solid var(--border)',
                cursor: 'pointer',
                color: viewMode === 'card' ? 'var(--text-emphasis)' : 'var(--text-muted)',
                display: 'flex',
                alignItems: 'center',
              }}
            >
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                <rect
                  x="2"
                  y="2"
                  width="4"
                  height="4"
                  rx="1"
                  stroke="currentColor"
                  strokeWidth="1.2"
                />
                <rect
                  x="8"
                  y="2"
                  width="4"
                  height="4"
                  rx="1"
                  stroke="currentColor"
                  strokeWidth="1.2"
                />
                <rect
                  x="2"
                  y="8"
                  width="4"
                  height="4"
                  rx="1"
                  stroke="currentColor"
                  strokeWidth="1.2"
                />
                <rect
                  x="8"
                  y="8"
                  width="4"
                  height="4"
                  rx="1"
                  stroke="currentColor"
                  strokeWidth="1.2"
                />
              </svg>
            </button>
          </div>
        </div>

        {/* New Deployment button */}
        <button
          type="button"
          onClick={() => setCreateOpen(true)}
          disabled={!can('deployments', 'create')}
          title={!can('deployments', 'create') ? "You don't have permission" : undefined}
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: 6,
            padding: '5px 12px',
            borderRadius: 6,
            fontSize: 12,
            fontWeight: 600,
            cursor: !can('deployments', 'create') ? 'not-allowed' : 'pointer',
            background: 'var(--accent)',
            color: 'var(--btn-accent-text, #000)',
            border: '1px solid var(--accent)',
            fontFamily: 'var(--font-sans)',
            whiteSpace: 'nowrap',
            opacity: !can('deployments', 'create') ? 0.5 : 1,
          }}
        >
          <Plus style={{ width: 13, height: 13 }} />
          New Deployment
        </button>
      </div>

      {/* Table / Card / Empty State */}
      {filtered.length === 0 && !search ? (
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            height: 280,
            borderRadius: 8,
            border: '1px solid var(--border)',
            background: 'var(--bg-card)',
          }}
        >
          <div style={{ textAlign: 'center' }}>
            <div style={{ fontSize: 13, color: 'var(--text-secondary)', marginBottom: 8 }}>
              No deployments yet
            </div>
            <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>
              Create a deployment from a policy to get started.
            </div>
          </div>
        </div>
      ) : viewMode === 'card' ? (
        <>
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: 'repeat(auto-fill, minmax(320px, 1fr))',
              gap: 12,
            }}
          >
            {sortedData.length === 0 ? (
              <div
                style={{
                  gridColumn: '1 / -1',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  height: 200,
                  borderRadius: 8,
                  border: '1px solid var(--border)',
                  background: 'var(--bg-card)',
                  color: 'var(--text-muted)',
                  fontSize: 13,
                }}
              >
                No results match your filters.
              </div>
            ) : (
              sortedData.map((d) => (
                <DeploymentCard
                  key={d.id}
                  deployment={d}
                  policyName={policyMap.get(d.policy_id ?? '')?.name}
                  onNavigate={() => navigate(`/deployments/${d.id}`)}
                  onRollback={() => rollbackMutation.mutate(d.id)}
                />
              ))
            )}
          </div>
          <DataTablePagination
            hasNext={!!data?.next_cursor}
            hasPrev={cursors.length > 0}
            onNext={() => {
              if (data?.next_cursor) setCursors((prev) => [...prev, data.next_cursor!]);
            }}
            onPrev={() => setCursors((prev) => prev.slice(0, -1))}
          />
        </>
      ) : (
        <>
          <div
            style={{
              background: 'var(--bg-card)',
              border: '1px solid var(--border)',
              borderRadius: 8,
              boxShadow: 'var(--shadow-sm)',
              overflow: 'auto',
            }}
          >
            <table
              style={{
                width: '100%',
                borderCollapse: 'collapse',
                minWidth: 1050,
                tableLayout: 'auto',
              }}
            >
              <thead>
                {table.getHeaderGroups().map((hg) => {
                  const sortableSet = new Set([
                    'id',
                    'name',
                    'policy_id',
                    'status',
                    'progress',
                    'targets',
                    'created_at',
                    'duration',
                    'triggered_by',
                  ]);
                  const labelMap: Record<string, string> = {
                    id: 'ID',
                    name: 'Name',
                    policy_id: 'Policy',
                    status: 'Status',
                    progress: 'Progress',
                    targets: 'Targets',
                    created_at: 'Created',
                    duration: 'Duration',
                    triggered_by: 'By',
                  };
                  return (
                    <tr key={hg.id}>
                      {hg.headers.map((header) => {
                        if (sortableSet.has(header.id)) {
                          const label =
                            labelMap[header.id] ??
                            header.id.charAt(0).toUpperCase() +
                              header.id.slice(1).replace('_', ' ');
                          return (
                            <SortHeader
                              key={header.id}
                              label={label}
                              colKey={header.id}
                              sortCol={sortCol}
                              sortDir={sortDir}
                              onSort={toggleSort}
                            />
                          );
                        }
                        return (
                          <th
                            key={header.id}
                            style={{
                              ...TH_STYLE,
                              width: header.id === 'expand' ? 32 : undefined,
                            }}
                          >
                            {header.isPlaceholder
                              ? null
                              : flexRender(header.column.columnDef.header, header.getContext())}
                          </th>
                        );
                      })}
                    </tr>
                  );
                })}
              </thead>
              <tbody>
                {table.getRowModel().rows.length === 0 ? (
                  <tr>
                    <td
                      colSpan={columns.length}
                      style={{
                        ...TD_STYLE,
                        textAlign: 'center',
                        color: 'var(--text-muted)',
                        fontSize: 13,
                        padding: 40,
                      }}
                    >
                      No results match your filters.
                    </td>
                  </tr>
                ) : (
                  table.getRowModel().rows.map((row) => {
                    const isFailed = row.original.status === 'failed';
                    const isExpanded = expandedRows.has(row.original.id);

                    return (
                      <Fragment key={row.id}>
                        <tr
                          onClick={() => navigate(`/deployments/${row.original.id}`)}
                          style={{
                            cursor: 'pointer',
                            background: isFailed
                              ? 'color-mix(in srgb, var(--signal-critical) 3%, transparent)'
                              : 'transparent',
                            borderLeft: 'none',
                            transition: 'background 0.1s',
                          }}
                          onMouseEnter={(e) => {
                            e.currentTarget.style.background = 'var(--bg-card-hover)';
                          }}
                          onMouseLeave={(e) => {
                            e.currentTarget.style.background = isFailed
                              ? 'color-mix(in srgb, var(--signal-critical) 3%, transparent)'
                              : 'transparent';
                          }}
                        >
                          {row.getVisibleCells().map((cell) => (
                            <td key={cell.id} style={TD_STYLE}>
                              {flexRender(cell.column.columnDef.cell, cell.getContext())}
                            </td>
                          ))}
                        </tr>
                        {isExpanded && (
                          <tr>
                            <td
                              colSpan={columns.length}
                              style={{ padding: 0, borderBottom: '1px solid var(--border)' }}
                            >
                              <DeploymentExpandedRow
                                deployment={row.original}
                                onCancel={() => cancelMutation.mutate(row.original.id)}
                                onRetry={() => retryMutation.mutate(row.original.id)}
                                onRollback={() => rollbackMutation.mutate(row.original.id)}
                              />
                            </td>
                          </tr>
                        )}
                      </Fragment>
                    );
                  })
                )}
              </tbody>
            </table>
          </div>

          <DataTablePagination
            hasNext={!!data?.next_cursor}
            hasPrev={cursors.length > 0}
            onNext={() => {
              if (data?.next_cursor) setCursors((prev) => [...prev, data.next_cursor!]);
            }}
            onPrev={() => setCursors((prev) => prev.slice(0, -1))}
          />
        </>
      )}

      <DeploymentWizard open={createOpen} onOpenChange={setCreateOpen} />
    </div>
  );
}
