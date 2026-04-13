import { useState, useMemo } from 'react';
import {
  getCoreRowModel,
  getExpandedRowModel,
  getPaginationRowModel,
  useReactTable,
  type ColumnDef,
  type ExpandedState,
} from '@tanstack/react-table';
import { Skeleton, ErrorState, EmptyState } from '@patchiq/ui';
import {
  Clock,
  Download,
  ChevronRight,
  ChevronDown,
  ShieldAlert,
  AlertTriangle,
  Minus,
  Package,
} from 'lucide-react';
import { usePendingPatches } from '../../api/hooks/usePatches';
import type { components } from '../../api/types';
import { FilterBar, FilterPill, FilterSeparator, FilterSearch } from '../../components/FilterBar';
import { DataTable } from '../../components/data-table/DataTable';
import { DataTablePagination } from '../../components/data-table/DataTablePagination';

type PendingPatch = components['schemas']['PendingPatch'] & {
  published_at?: string;
  source?: string;
  size?: number;
  cve_ids?: string[];
};

// ─── Helpers ──────────────────────────────────────────────────────────────────

function severityColor(severity: PendingPatch['severity']): string {
  switch (severity) {
    case 'critical':
      return 'var(--signal-critical)';
    case 'high':
      return 'var(--signal-warning)';
    case 'medium':
      return 'var(--text-secondary)';
    default:
      return 'var(--text-muted)';
  }
}

function cvssColor(score: number): string {
  if (score >= 9) return 'var(--signal-critical)';
  if (score >= 7) return 'var(--signal-warning)';
  if (score >= 4) return 'var(--text-secondary)';
  return 'var(--signal-healthy)';
}

// ─── Local stat card (clickable filter card) ──────────────────────────────────

interface StatFilterCardProps {
  label: string;
  value: number;
  valueColor?: string;
  active: boolean;
  onClick: () => void;
}

function StatFilterCard({ label, value, valueColor, active, onClick }: StatFilterCardProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      style={{
        flex: 1,
        minWidth: 0,
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'flex-start',
        padding: '12px 14px',
        background: active
          ? 'color-mix(in srgb, var(--text-primary) 3%, transparent)'
          : 'var(--bg-card)',
        border: `1px solid ${active ? (valueColor ?? 'var(--accent)') : 'var(--border)'}`,
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
        {value}
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

// ─── Status Badge ─────────────────────────────────────────────────────────────

function StatusBadge({ status }: { status: PendingPatch['status'] }) {
  const map: Record<
    PendingPatch['status'],
    { bg: string; color: string; border: string; label: string }
  > = {
    queued: {
      bg: 'transparent',
      color: 'var(--text-muted)',
      border: 'var(--border)',
      label: 'Queued',
    },
    downloading: {
      bg: 'var(--accent-subtle)',
      color: 'var(--accent)',
      border: 'var(--border)',
      label: 'Downloading',
    },
    installing: {
      bg: 'color-mix(in srgb, var(--signal-warning) 10%, transparent)',
      color: 'var(--signal-warning)',
      border: 'color-mix(in srgb, var(--signal-warning) 30%, transparent)',
      label: 'Installing',
    },
    pending_reboot: {
      bg: 'color-mix(in srgb, var(--signal-warning) 10%, transparent)',
      color: 'var(--signal-warning)',
      border: 'color-mix(in srgb, var(--signal-warning) 30%, transparent)',
      label: 'Pending Reboot',
    },
  };
  const c = map[status];
  return (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 4,
        fontSize: 11,
        padding: '2px 8px',
        borderRadius: 4,
        border: `1px solid ${c.border}`,
        background: c.bg,
        color: c.color,
        whiteSpace: 'nowrap',
      }}
    >
      {status === 'queued' && <Clock style={{ width: 10, height: 10 }} />}
      {status === 'downloading' && <Download style={{ width: 10, height: 10 }} />}
      {c.label}
    </span>
  );
}

// ─── Severity Badge ───────────────────────────────────────────────────────────

function SeverityBadge({ severity }: { severity: PendingPatch['severity'] }) {
  const color = severityColor(severity);
  const icons: Record<PendingPatch['severity'], React.ReactNode> = {
    critical: <ShieldAlert style={{ width: 10, height: 10 }} />,
    high: <AlertTriangle style={{ width: 10, height: 10 }} />,
    medium: <Minus style={{ width: 10, height: 10 }} />,
    low: <Minus style={{ width: 10, height: 10 }} />,
    none: <Minus style={{ width: 10, height: 10 }} />,
  };
  return (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 4,
        fontSize: 11,
        fontWeight: 600,
        textTransform: 'uppercase',
        letterSpacing: '0.03em',
        color,
      }}
    >
      {icons[severity]}
      {severity}
    </span>
  );
}

// ─── Expanded Row Content ─────────────────────────────────────────────────────

function ExpandedRowContent({ patch }: { patch: PendingPatch }) {
  return (
    <div
      style={{
        padding: '16px 24px',
        background: 'var(--bg-inset)',
        borderLeft: '2px solid var(--accent)',
        display: 'grid',
        gridTemplateColumns: '1fr 1fr',
        gap: 24,
      }}
    >
      {/* Details */}
      <div>
        <div
          style={{
            fontSize: 10,
            fontFamily: 'var(--font-mono)',
            fontWeight: 600,
            textTransform: 'uppercase',
            letterSpacing: '0.06em',
            color: 'var(--text-muted)',
            marginBottom: 8,
          }}
        >
          Patch Details
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 4, fontSize: 12 }}>
          <div>
            <span style={{ color: 'var(--text-muted)' }}>ID: </span>
            <span style={{ fontFamily: 'var(--font-mono)', color: 'var(--text-secondary)' }}>
              {patch.id}
            </span>
          </div>
          <div>
            <span style={{ color: 'var(--text-muted)' }}>Queued: </span>
            <span style={{ color: 'var(--text-primary)' }}>
              {new Date(patch.queued_at).toLocaleString()}
            </span>
          </div>
          {patch.published_at && (
            <div>
              <span style={{ color: 'var(--text-muted)' }}>Published: </span>
              <span style={{ color: 'var(--text-primary)' }}>
                {new Date(patch.published_at).toLocaleDateString()}
              </span>
            </div>
          )}
          {patch.source && (
            <div>
              <span style={{ color: 'var(--text-muted)' }}>Source: </span>
              <span style={{ fontFamily: 'var(--font-mono)', color: 'var(--text-primary)' }}>
                {patch.source}
              </span>
            </div>
          )}
          {patch.size && (
            <div>
              <span style={{ color: 'var(--text-muted)' }}>Size: </span>
              <span style={{ color: 'var(--text-primary)' }}>{patch.size}</span>
            </div>
          )}
        </div>
      </div>

      {/* CVE list */}
      <div>
        <div
          style={{
            fontSize: 10,
            fontFamily: 'var(--font-mono)',
            fontWeight: 600,
            textTransform: 'uppercase',
            letterSpacing: '0.06em',
            color: 'var(--text-muted)',
            marginBottom: 8,
          }}
        >
          CVEs ({patch.cve_ids?.length ?? 0})
        </div>
        {patch.cve_ids && patch.cve_ids.length > 0 ? (
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4 }}>
            {patch.cve_ids.map((cve) => (
              <span
                key={cve}
                style={{
                  background: 'var(--bg-card)',
                  border: '1px solid var(--border)',
                  borderRadius: 4,
                  padding: '2px 6px',
                  fontSize: 10,
                  fontFamily: 'var(--font-mono)',
                  color: 'var(--text-secondary)',
                }}
              >
                {cve}
              </span>
            ))}
          </div>
        ) : (
          <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>No CVEs associated</span>
        )}
      </div>
    </div>
  );
}

// ─── Column Definitions ───────────────────────────────────────────────────────

const SEVERITY_ORDER: PendingPatch['severity'][] = ['critical', 'high', 'medium', 'low', 'none'];

// ─── Page ─────────────────────────────────────────────────────────────────────

type SeverityFilter = PendingPatch['severity'] | 'all';
type StatusFilter = PendingPatch['status'] | 'all';

export const PendingPatchesPage = () => {
  const { data, isLoading, isError, refetch } = usePendingPatches();
  const allPatches = (data?.data ?? []) as PendingPatch[];

  const [severityFilter, setSeverityFilter] = useState<SeverityFilter>('all');
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');
  const [search, setSearch] = useState('');
  const [expanded, setExpanded] = useState<ExpandedState>({});

  const counts = useMemo(
    () => ({
      total: allPatches.length,
      critical: allPatches.filter((p) => p.severity === 'critical').length,
      high: allPatches.filter((p) => p.severity === 'high').length,
      medlow: allPatches.filter((p) => p.severity === 'medium' || p.severity === 'low').length,
    }),
    [allPatches],
  );

  const filtered = useMemo(
    () =>
      allPatches
        .filter((p) => severityFilter === 'all' || p.severity === severityFilter)
        .filter((p) => statusFilter === 'all' || p.status === statusFilter)
        .filter(
          (p) =>
            !search ||
            p.name.toLowerCase().includes(search.toLowerCase()) ||
            p.cve_ids?.some((c: string) => c.toLowerCase().includes(search.toLowerCase())),
        )
        .sort((a, b) => SEVERITY_ORDER.indexOf(a.severity) - SEVERITY_ORDER.indexOf(b.severity)),
    [allPatches, severityFilter, statusFilter, search],
  );

  const columns: ColumnDef<PendingPatch>[] = useMemo(
    () => [
      {
        id: 'expand',
        header: '',
        size: 32,
        cell: ({ row }) => (
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation();
              row.toggleExpanded();
            }}
            style={{
              background: 'none',
              border: 'none',
              cursor: 'pointer',
              color: 'var(--text-muted)',
              padding: 0,
              display: 'flex',
              alignItems: 'center',
            }}
          >
            {row.getIsExpanded() ? (
              <ChevronDown style={{ width: 14, height: 14 }} />
            ) : (
              <ChevronRight style={{ width: 14, height: 14 }} />
            )}
          </button>
        ),
      },
      {
        id: 'name',
        header: 'Patch Name',
        accessorKey: 'name',
        cell: ({ getValue }) => (
          <span
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 12,
              fontWeight: 500,
              color: 'var(--text-emphasis)',
            }}
          >
            {getValue() as string}
          </span>
        ),
      },
      {
        id: 'severity',
        header: 'Severity',
        accessorKey: 'severity',
        cell: ({ getValue }) => <SeverityBadge severity={getValue() as PendingPatch['severity']} />,
      },
      {
        id: 'version',
        header: 'Version',
        accessorKey: 'version',
        cell: ({ getValue }) => (
          <span
            style={{ fontFamily: 'var(--font-mono)', fontSize: 12, color: 'var(--text-secondary)' }}
          >
            {getValue() as string}
          </span>
        ),
      },
      {
        id: 'status',
        header: 'Status',
        accessorKey: 'status',
        cell: ({ getValue }) => <StatusBadge status={getValue() as PendingPatch['status']} />,
      },
      {
        id: 'cvss',
        header: 'CVSS',
        accessorKey: 'cvss_score',
        cell: ({ getValue }) => {
          const score = getValue() as number | null | undefined;
          if (score == null)
            return (
              <span
                style={{ color: 'var(--text-muted)', fontFamily: 'var(--font-mono)', fontSize: 12 }}
              >
                —
              </span>
            );
          return (
            <span
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 12,
                fontWeight: 700,
                color: cvssColor(score),
              }}
            >
              {score.toFixed(1)}
            </span>
          );
        },
      },
      {
        id: 'cves',
        header: 'CVEs',
        cell: ({ row }) => {
          const count = row.original.cve_ids?.length ?? 0;
          if (count === 0)
            return (
              <span
                style={{ color: 'var(--text-muted)', fontFamily: 'var(--font-mono)', fontSize: 12 }}
              >
                —
              </span>
            );
          return (
            <span style={{ fontFamily: 'var(--font-mono)', fontSize: 11, color: 'var(--accent)' }}>
              {count} CVE{count > 1 ? 's' : ''}
            </span>
          );
        },
      },
      {
        id: 'actions',
        header: '',
        cell: () => (
          <div style={{ display: 'flex', gap: 6 }}>
            <button
              type="button"
              style={{
                background: 'var(--accent)',
                color: 'white',
                border: 'none',
                borderRadius: 5,
                padding: '4px 10px',
                fontSize: 11,
                fontWeight: 600,
                cursor: 'pointer',
                whiteSpace: 'nowrap',
              }}
            >
              Install
            </button>
            <button
              type="button"
              style={{
                background: 'transparent',
                color: 'var(--signal-critical)',
                border: '1px solid var(--border)',
                borderRadius: 5,
                padding: '4px 10px',
                fontSize: 11,
                fontWeight: 500,
                cursor: 'pointer',
              }}
            >
              Skip
            </button>
          </div>
        ),
      },
    ],
    [],
  );

  const table = useReactTable({
    data: filtered,
    columns,
    state: { expanded },
    onExpandedChange: setExpanded,
    getCoreRowModel: getCoreRowModel(),
    getExpandedRowModel: getExpandedRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    initialState: { pagination: { pageSize: 20 } },
  });

  const { pageIndex, pageSize } = table.getState().pagination;
  const pageCount = table.getPageCount();

  if (isLoading) {
    return (
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 12 }}>
          {[1, 2, 3, 4].map((i) => (
            <Skeleton key={i} className="h-16 w-full rounded-lg" />
          ))}
        </div>
        <Skeleton className="h-10 w-full rounded-lg" />
        <Skeleton className="h-64 w-full rounded-lg" />
      </div>
    );
  }

  if (isError) {
    return <ErrorState message="Failed to load pending patches." onRetry={() => void refetch()} />;
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* Stat cards */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 12 }}>
        <StatFilterCard
          label="Total Pending"
          value={counts.total}
          active={severityFilter === 'all' && statusFilter === 'all'}
          onClick={() => {
            setSeverityFilter('all');
            setStatusFilter('all');
            table.resetPagination();
          }}
        />
        <StatFilterCard
          label="Critical"
          value={counts.critical}
          valueColor="var(--signal-critical)"
          active={severityFilter === 'critical'}
          onClick={() => {
            setSeverityFilter('critical');
            table.resetPagination();
          }}
        />
        <StatFilterCard
          label="High"
          value={counts.high}
          valueColor="var(--signal-warning)"
          active={severityFilter === 'high'}
          onClick={() => {
            setSeverityFilter('high');
            table.resetPagination();
          }}
        />
        <StatFilterCard
          label="Med + Low"
          value={counts.medlow}
          valueColor="var(--text-secondary)"
          active={severityFilter === 'medium' || severityFilter === 'low'}
          onClick={() => {
            setSeverityFilter('medium');
            table.resetPagination();
          }}
        />
      </div>

      {/* Filter bar */}
      <FilterBar>
        <FilterPill
          label="All"
          count={counts.total}
          active={severityFilter === 'all'}
          onClick={() => {
            setSeverityFilter('all');
            table.resetPagination();
          }}
        />
        <FilterPill
          label="Critical"
          count={counts.critical}
          active={severityFilter === 'critical'}
          variant="critical"
          onClick={() => {
            setSeverityFilter('critical');
            table.resetPagination();
          }}
        />
        <FilterPill
          label="High"
          count={counts.high}
          active={severityFilter === 'high'}
          variant="high"
          onClick={() => {
            setSeverityFilter('high');
            table.resetPagination();
          }}
        />
        <FilterPill
          label="Med+Low"
          count={counts.medlow}
          active={severityFilter === 'medium'}
          variant="medium"
          onClick={() => {
            setSeverityFilter('medium');
            table.resetPagination();
          }}
        />
        <FilterSeparator />
        <FilterPill
          label="All Status"
          active={statusFilter === 'all'}
          onClick={() => {
            setStatusFilter('all');
            table.resetPagination();
          }}
        />
        <FilterPill
          label="Queued"
          active={statusFilter === 'queued'}
          variant="muted"
          onClick={() => {
            setStatusFilter('queued');
            table.resetPagination();
          }}
        />
        <FilterPill
          label="Downloading"
          active={statusFilter === 'downloading'}
          onClick={() => {
            setStatusFilter('downloading');
            table.resetPagination();
          }}
        />
        <FilterPill
          label="Installing"
          active={statusFilter === 'installing'}
          variant="warning"
          onClick={() => {
            setStatusFilter('installing');
            table.resetPagination();
          }}
        />
        <FilterPill
          label="Pending Reboot"
          active={statusFilter === 'pending_reboot'}
          variant="warning"
          onClick={() => {
            setStatusFilter('pending_reboot');
            table.resetPagination();
          }}
        />
        <div style={{ marginLeft: 'auto' }}>
          <FilterSearch
            value={search}
            onChange={(v) => {
              setSearch(v);
              table.resetPagination();
            }}
            placeholder="Search patches or CVEs..."
          />
        </div>
      </FilterBar>

      {/* Table */}
      {filtered.length === 0 ? (
        <EmptyState
          icon={<Package style={{ width: 24, height: 24, color: 'var(--text-muted)' }} />}
          title={
            allPatches.length === 0 ? 'No patches pending' : 'No patches match the current filter'
          }
          description={
            allPatches.length === 0
              ? 'This endpoint is up to date.'
              : 'Try adjusting your severity or status filters.'
          }
        />
      ) : (
        <>
          <DataTable
            table={table}
            renderExpandedRow={(row) => <ExpandedRowContent patch={row} />}
          />
          {pageCount > 1 && (
            <DataTablePagination
              pageIndex={pageIndex}
              pageCount={pageCount}
              hasPrev={table.getCanPreviousPage()}
              hasNext={table.getCanNextPage()}
              onPrev={() => table.previousPage()}
              onNext={() => table.nextPage()}
            />
          )}
          <div style={{ fontSize: 11, color: 'var(--text-muted)', textAlign: 'right' }}>
            Showing {pageIndex * pageSize + 1}–
            {Math.min((pageIndex + 1) * pageSize, filtered.length)} of {filtered.length} patches
          </div>
        </>
      )}
    </div>
  );
};
