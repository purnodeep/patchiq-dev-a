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
import { ChevronRight, ChevronDown, Server } from 'lucide-react';
import { useAgentServices } from '../../api/hooks/useServices';
import type { ServiceInfo } from '../../types/software';
import { FilterBar, FilterPill, FilterSeparator, FilterSearch } from '../../components/FilterBar';
import { DataTable } from '../../components/data-table/DataTable';
import { DataTablePagination } from '../../components/data-table/DataTablePagination';

// ─── Types ────────────────────────────────────────────────────────────────────

type AgentService = ServiceInfo;

// ─── Helpers ──────────────────────────────────────────────────────────────────

type StatusFilter = 'all' | 'running' | 'stopped' | 'failed';

// ─── Local stat filter card ────────────────────────────────────────────────────

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

// ─── Sub-components ───────────────────────────────────────────────────────────

function ActiveStateDot({ state }: { state: string }) {
  const color =
    state === 'active'
      ? 'var(--signal-healthy)'
      : state === 'failed'
        ? 'var(--signal-critical)'
        : state === 'inactive'
          ? 'var(--text-muted)'
          : 'var(--signal-warning)';
  return (
    <span
      style={{
        width: 8,
        height: 8,
        borderRadius: '50%',
        background: color,
        display: 'inline-block',
        flexShrink: 0,
      }}
    />
  );
}

function CategoryBadge({ category }: { category: string | undefined }) {
  const cat = category || 'Other';
  return (
    <span
      style={{
        fontSize: 11,
        padding: '2px 8px',
        borderRadius: 4,
        border: '1px solid var(--border)',
        background: 'var(--bg-card-hover)',
        color: 'var(--text-secondary)',
        fontWeight: 500,
        whiteSpace: 'nowrap',
      }}
    >
      {cat}
    </span>
  );
}

function EnabledBadge({ enabled }: { enabled: boolean }) {
  return (
    <span
      style={{
        fontSize: 11,
        padding: '2px 8px',
        borderRadius: 4,
        border: `1px solid ${enabled ? 'color-mix(in srgb, var(--signal-healthy) 30%, transparent)' : 'var(--border)'}`,
        background: enabled
          ? 'color-mix(in srgb, var(--signal-healthy) 10%, transparent)'
          : 'transparent',
        color: enabled ? 'var(--signal-healthy)' : 'var(--text-muted)',
      }}
    >
      {enabled ? 'enabled' : 'disabled'}
    </span>
  );
}

function ExpandedRowContent({ svc }: { svc: AgentService }) {
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
          Description
        </div>
        <p style={{ fontSize: 12, color: 'var(--text-secondary)', margin: 0, lineHeight: 1.5 }}>
          {svc.description || 'No description available.'}
        </p>
      </div>
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
          State
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 4, fontSize: 12 }}>
          <div>
            <span style={{ color: 'var(--text-muted)' }}>Active state: </span>
            <span style={{ color: 'var(--text-primary)', fontFamily: 'var(--font-mono)' }}>
              {svc.active_state}
            </span>
          </div>
          <div>
            <span style={{ color: 'var(--text-muted)' }}>Sub state: </span>
            <span style={{ color: 'var(--text-primary)', fontFamily: 'var(--font-mono)' }}>
              {svc.sub_state}
            </span>
          </div>
          <div>
            <span style={{ color: 'var(--text-muted)' }}>Startup: </span>
            <EnabledBadge enabled={svc.enabled} />
          </div>
        </div>
      </div>
    </div>
  );
}

// ─── Page ─────────────────────────────────────────────────────────────────────

export function ServicesPage() {
  const { data, isLoading, isError, refetch } = useAgentServices();
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('running');
  const [categoryFilter, setCategoryFilter] = useState<string>('all');
  const [expanded, setExpanded] = useState<ExpandedState>({});

  const counts = useMemo(() => {
    if (!data) return { total: 0, running: 0, stopped: 0, failed: 0 };
    return {
      total: data.length,
      running: data.filter((s) => s.sub_state === 'running').length,
      stopped: data.filter((s) => s.active_state === 'inactive').length,
      failed: data.filter((s) => s.active_state === 'failed').length,
    };
  }, [data]);

  const categories = useMemo(() => {
    if (!data) return [];
    const result: Record<string, number> = {};
    for (const s of data) {
      const cat = s.category || 'Other';
      result[cat] = (result[cat] || 0) + 1;
    }
    return Object.entries(result)
      .sort((a, b) => b[1] - a[1])
      .map(([cat]) => cat);
  }, [data]);

  const filtered = useMemo(() => {
    if (!data) return [];
    return data
      .filter((s) => {
        if (search) {
          const q = search.toLowerCase();
          return s.name.toLowerCase().includes(q) || s.description.toLowerCase().includes(q);
        }
        return true;
      })
      .filter((s) => {
        switch (statusFilter) {
          case 'running':
            return s.sub_state === 'running';
          case 'stopped':
            return s.active_state === 'inactive';
          case 'failed':
            return s.active_state === 'failed';
          default:
            return true;
        }
      })
      .filter((s) => categoryFilter === 'all' || (s.category || 'Other') === categoryFilter)
      .sort((a, b) => a.name.localeCompare(b.name));
  }, [data, search, statusFilter, categoryFilter]);

  const columns: ColumnDef<AgentService>[] = useMemo(
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
        id: 'status',
        header: 'Status',
        size: 80,
        cell: ({ row }) => {
          const svc = row.original;
          const label =
            svc.active_state === 'failed'
              ? 'Failed'
              : svc.sub_state === 'running'
                ? 'Running'
                : 'Stopped';
          return (
            <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
              <ActiveStateDot state={svc.active_state} />
              <span style={{ fontSize: 12, color: 'var(--text-secondary)' }}>{label}</span>
            </div>
          );
        },
      },
      {
        id: 'name',
        header: 'Service Name',
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
        id: 'category',
        header: 'Category',
        accessorKey: 'category',
        cell: ({ getValue }) => <CategoryBadge category={getValue() as string | undefined} />,
      },
      {
        id: 'description',
        header: 'Description',
        accessorKey: 'description',
        cell: ({ getValue }) => (
          <span
            style={{
              fontSize: 12,
              color: 'var(--text-muted)',
              display: 'block',
              maxWidth: 400,
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
            }}
          >
            {(getValue() as string) || '\u2014'}
          </span>
        ),
      },
      {
        id: 'enabled',
        header: 'Startup',
        accessorKey: 'enabled',
        cell: ({ getValue }) => <EnabledBadge enabled={getValue() as boolean} />,
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
    initialState: { pagination: { pageSize: 50 } },
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
    return <ErrorState message="Failed to load services." onRetry={() => void refetch()} />;
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* Stat cards */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 12 }}>
        <StatFilterCard
          label="Total Services"
          value={counts.total}
          active={statusFilter === 'all' && categoryFilter === 'all'}
          onClick={() => {
            setStatusFilter('all');
            setCategoryFilter('all');
            table.resetPagination();
          }}
        />
        <StatFilterCard
          label="Running"
          value={counts.running}
          valueColor="var(--signal-healthy)"
          active={statusFilter === 'running'}
          onClick={() => {
            setStatusFilter('running');
            table.resetPagination();
          }}
        />
        <StatFilterCard
          label="Stopped"
          value={counts.stopped}
          valueColor="var(--text-secondary)"
          active={statusFilter === 'stopped'}
          onClick={() => {
            setStatusFilter('stopped');
            table.resetPagination();
          }}
        />
        <StatFilterCard
          label="Failed"
          value={counts.failed}
          valueColor={counts.failed > 0 ? 'var(--signal-critical)' : 'var(--text-muted)'}
          active={statusFilter === 'failed'}
          onClick={() => {
            setStatusFilter('failed');
            table.resetPagination();
          }}
        />
      </div>

      {/* Filter bar: status + category + search */}
      <FilterBar>
        <FilterPill
          label="All"
          count={counts.total}
          active={statusFilter === 'all'}
          onClick={() => {
            setStatusFilter('all');
            table.resetPagination();
          }}
        />
        <FilterPill
          label="Running"
          count={counts.running}
          active={statusFilter === 'running'}
          variant="success"
          onClick={() => {
            setStatusFilter('running');
            table.resetPagination();
          }}
        />
        <FilterPill
          label="Stopped"
          count={counts.stopped}
          active={statusFilter === 'stopped'}
          variant="muted"
          onClick={() => {
            setStatusFilter('stopped');
            table.resetPagination();
          }}
        />
        <FilterPill
          label="Failed"
          count={counts.failed}
          active={statusFilter === 'failed'}
          variant="danger"
          onClick={() => {
            setStatusFilter('failed');
            table.resetPagination();
          }}
        />
        {categories.length > 0 && <FilterSeparator />}
        {categories.length > 0 && (
          <FilterPill
            label="All Categories"
            active={categoryFilter === 'all'}
            onClick={() => {
              setCategoryFilter('all');
              table.resetPagination();
            }}
          />
        )}
        {categories.map((cat) => (
          <FilterPill
            key={cat}
            label={cat}
            active={categoryFilter === cat}
            onClick={() => {
              setCategoryFilter(cat);
              table.resetPagination();
            }}
          />
        ))}
        <div style={{ marginLeft: 'auto' }}>
          <FilterSearch
            value={search}
            onChange={(v) => {
              setSearch(v);
              table.resetPagination();
            }}
            placeholder="Search services..."
          />
        </div>
      </FilterBar>

      {/* Table */}
      {filtered.length === 0 ? (
        <EmptyState
          icon={Server}
          title={
            !data || data.length === 0
              ? 'No services found'
              : 'No services match the current filter'
          }
          description="Try adjusting your status or category filters."
        />
      ) : (
        <>
          <DataTable table={table} renderExpandedRow={(row) => <ExpandedRowContent svc={row} />} />
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
            {Math.min((pageIndex + 1) * pageSize, filtered.length)} of {filtered.length} services
          </div>
        </>
      )}
    </div>
  );
}
