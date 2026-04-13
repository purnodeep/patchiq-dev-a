import { useState } from 'react';
import { useCan } from '../../../app/auth/AuthContext';
import { useNavigate } from 'react-router';
import {
  getCoreRowModel,
  getSortedRowModel,
  useReactTable,
  createColumnHelper,
  type SortingState,
} from '@tanstack/react-table';
import { Plus, Shield, Trash2 } from 'lucide-react';
import { Skeleton } from '@patchiq/ui';
import { toast } from 'sonner';
import { useRoles, useDeleteRole, type Role } from '../../../api/hooks/useRoles';
import { DataTable, DataTablePagination, DataTableSearch } from '../../../components/data-table';
import { timeAgo } from '../../../lib/time';

const col = createColumnHelper<Role>();

function makeColumns(
  onDelete: (role: Role) => void,
  onRowClick: (role: Role) => void,
  canDeleteRole: boolean,
) {
  return [
    col.accessor('name', {
      header: 'Name',
      cell: (info) => {
        const role = info.row.original;
        return (
          <button
            type="button"
            onClick={() => onRowClick(role)}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: '8px',
              background: 'none',
              border: 'none',
              padding: 0,
              cursor: 'pointer',
              textAlign: 'left',
            }}
          >
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                width: '28px',
                height: '28px',
                borderRadius: '6px',
                background: 'var(--bg-inset)',
                border: '1px solid var(--border)',
                flexShrink: 0,
              }}
            >
              <Shield style={{ width: '13px', height: '13px', color: 'var(--text-muted)' }} />
            </div>
            <span
              style={{
                fontSize: '13px',
                fontWeight: 500,
                color: 'var(--accent)',
                fontFamily: 'var(--font-sans)',
                textDecoration: 'underline',
                textDecorationColor: 'color-mix(in srgb, var(--accent) 30%, transparent)',
                textUnderlineOffset: '2px',
              }}
            >
              {info.getValue()}
            </span>
          </button>
        );
      },
    }),
    col.accessor('description', {
      header: 'Description',
      cell: (info) => (
        <span
          style={{ fontSize: '12px', color: 'var(--text-muted)', fontFamily: 'var(--font-sans)' }}
        >
          {info.getValue() || '—'}
        </span>
      ),
    }),
    col.accessor('is_system', {
      header: 'Type',
      cell: (info) =>
        info.getValue() ? (
          <span
            style={{ fontSize: '12px', color: 'var(--text-faint)', fontFamily: 'var(--font-sans)' }}
          >
            System
          </span>
        ) : (
          <span
            style={{ fontSize: '12px', color: 'var(--text-muted)', fontFamily: 'var(--font-sans)' }}
          >
            Custom
          </span>
        ),
    }),
    col.accessor('permission_count', {
      header: 'Permissions',
      cell: (info) => (
        <span
          style={{
            fontSize: '13px',
            fontFamily: 'var(--font-mono)',
            color: 'var(--text-secondary)',
          }}
        >
          {info.getValue() ?? 0}
        </span>
      ),
    }),
    col.accessor('created_at', {
      header: 'Created',
      cell: (info) => (
        <span
          style={{
            fontSize: '11px',
            color: 'var(--text-muted)',
            fontFamily: 'var(--font-mono)',
          }}
        >
          {timeAgo(info.getValue())}
        </span>
      ),
    }),
    col.display({
      id: 'actions',
      header: '',
      cell: (info) => {
        const role = info.row.original;
        if (role.is_system) return null;
        return (
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation();
              onDelete(role);
            }}
            disabled={!canDeleteRole}
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              width: '30px',
              height: '30px',
              background: 'transparent',
              border: '1px solid transparent',
              borderRadius: '6px',
              cursor: !canDeleteRole ? 'not-allowed' : 'pointer',
              color: 'var(--signal-critical)',
              opacity: !canDeleteRole ? 0.5 : 1,
              transition: 'border-color 0.1s',
            }}
            title={!canDeleteRole ? "You don't have permission" : 'Delete role'}
          >
            <Trash2 style={{ width: '13px', height: '13px' }} />
          </button>
        );
      },
    }),
  ];
}

export function RolesPage() {
  const can = useCan();
  const [search, setSearch] = useState('');
  const [sorting, setSorting] = useState<SortingState>([]);
  const [cursors, setCursors] = useState<string[]>([]);
  const currentCursor = cursors[cursors.length - 1];
  const navigate = useNavigate();
  const deleteRole = useDeleteRole();

  const { data, isLoading, isError, refetch } = useRoles({
    cursor: currentCursor,
    limit: 25,
    search: search || undefined,
  });

  const items = data?.data ?? [];

  const handleDelete = (role: Role) => {
    if (!confirm(`Delete role "${role.name}"? This action cannot be undone.`)) return;
    deleteRole.mutate(role.id, {
      onSuccess: () => toast.success(`Role "${role.name}" deleted`),
      onError: (err) => toast.error(`Failed to delete role: ${err.message}`),
    });
  };

  const handleRowClick = (role: Role) => {
    navigate(`/settings/roles/${role.id}/edit`);
  };

  const columns = makeColumns(handleDelete, handleRowClick, can('roles', 'delete'));

  const table = useReactTable({
    data: items,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    onSortingChange: setSorting,
    state: { sorting },
  });

  if (isError) {
    return (
      <div style={{ padding: '24px' }}>
        <div
          style={{
            borderRadius: '8px',
            border: '1px solid color-mix(in srgb, var(--signal-critical) 10%, transparent)',
            background: 'color-mix(in srgb, var(--signal-critical) 10%, transparent)',
            padding: '14px 16px',
            fontSize: '13px',
            color: 'var(--signal-critical)',
            fontFamily: 'var(--font-sans)',
          }}
        >
          Failed to load roles.{' '}
          <button
            onClick={() => refetch()}
            style={{
              background: 'none',
              border: 'none',
              cursor: 'pointer',
              color: 'var(--signal-critical)',
              textDecoration: 'underline',
              fontFamily: 'var(--font-sans)',
              fontSize: '13px',
            }}
          >
            Retry
          </button>
        </div>
      </div>
    );
  }

  return (
    <div
      style={{
        padding: '24px',
        background: 'var(--bg-page)',
        minHeight: '100%',
        display: 'flex',
        flexDirection: 'column',
        gap: '20px',
      }}
    >
      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <div style={{ display: 'flex', alignItems: 'baseline', gap: '10px' }}>
          <h1
            style={{
              fontSize: '22px',
              fontWeight: 600,
              color: 'var(--text-emphasis)',
              letterSpacing: '-0.02em',
              fontFamily: 'var(--font-sans)',
            }}
          >
            Roles
          </h1>
          {!isLoading && (
            <span
              style={{
                fontSize: '13px',
                color: 'var(--text-muted)',
                fontFamily: 'var(--font-mono)',
              }}
            >
              {items.length}
            </span>
          )}
        </div>
        <button
          type="button"
          onClick={() => navigate('/settings/roles/new')}
          disabled={!can('roles', 'create')}
          title={!can('roles', 'create') ? "You don't have permission" : undefined}
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: '6px',
            height: '34px',
            padding: '0 14px',
            background: 'var(--accent)',
            border: 'none',
            borderRadius: '6px',
            fontSize: '13px',
            fontWeight: 600,
            color: 'var(--text-on-color, #fff)',
            cursor: !can('roles', 'create') ? 'not-allowed' : 'pointer',
            opacity: !can('roles', 'create') ? 0.5 : 1,
            fontFamily: 'var(--font-sans)',
          }}
        >
          <Plus style={{ width: '14px', height: '14px' }} />
          Create Role
        </button>
      </div>

      {/* Search */}
      <div>
        <DataTableSearch value={search} onChange={setSearch} placeholder="Search roles..." />
      </div>

      {/* Table */}
      {isLoading ? (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} className="h-12 rounded-md" />
          ))}
        </div>
      ) : items.length === 0 && !search ? (
        <div
          style={{
            display: 'flex',
            height: '300px',
            alignItems: 'center',
            justifyContent: 'center',
            fontSize: '13px',
            color: 'var(--text-muted)',
            fontFamily: 'var(--font-sans)',
          }}
        >
          No roles yet. Create a role to get started.
        </div>
      ) : (
        <>
          <DataTable table={table} onRowClick={(r) => navigate(`/settings/roles/${r.id}/edit`)} />
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
    </div>
  );
}
