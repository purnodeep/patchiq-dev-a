import { useState } from 'react';
import {
  createColumnHelper,
  getCoreRowModel,
  getSortedRowModel,
  useReactTable,
  type SortingState,
} from '@tanstack/react-table';
import { Plus } from 'lucide-react';
import { Badge, Button, PageHeader, Skeleton } from '@patchiq/ui';
import { useAuth } from '../../app/auth/AuthContext';
import { useOrgTenants } from '../../api/hooks/useOrganizations';
import type { components } from '../../api/types';
import { DataTable } from '../../components/data-table';
import { AddClientDialog } from './AddClientDialog';

type ClientRow = components['schemas']['OrgTenantSummary'];

const col = createColumnHelper<ClientRow>();

const columns = [
  col.accessor('name', {
    header: 'Name',
    cell: (info) => <span className="text-sm font-medium">{info.getValue()}</span>,
  }),
  col.accessor('slug', {
    header: 'Slug',
    cell: (info) => (
      <span className="font-mono text-xs text-muted-foreground">{info.getValue()}</span>
    ),
  }),
  col.display({
    id: 'status',
    header: 'Status',
    cell: () => <Badge variant="secondary">Active</Badge>,
  }),
  col.accessor('created_at', {
    header: 'Created',
    cell: (info) => {
      const v = info.getValue();
      const formatted = v ? new Date(v).toLocaleDateString() : '—';
      return <span className="text-xs text-muted-foreground">{formatted}</span>;
    },
  }),
];

// ClientsPage lists every child tenant belonging to the operator's current
// organization and provides an "Add Client" affordance for provisioning new
// tenants via POST /api/v1/organizations/{id}/tenants.
export function ClientsPage() {
  const { user } = useAuth();
  const orgId = user.organization?.id;
  const [sorting, setSorting] = useState<SortingState>([]);
  const [dialogOpen, setDialogOpen] = useState(false);

  const { data, isLoading, isError, error } = useOrgTenants(orgId);
  const rows: ClientRow[] = data?.data ?? [];

  const table = useReactTable({
    data: rows,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    onSortingChange: setSorting,
    state: { sorting },
  });

  if (!orgId) {
    return (
      <div className="p-6">
        <PageHeader title="Clients" subtitle="Tenants in your organization" />
        <p className="mt-6 text-sm text-muted-foreground">
          No organization context available for this session.
        </p>
      </div>
    );
  }

  return (
    <div className="p-6 space-y-6">
      <PageHeader
        title="Clients"
        subtitle="Tenants in your organization"
        actions={
          <Button onClick={() => setDialogOpen(true)}>
            <Plus className="w-4 h-4 mr-1" />
            Add Client
          </Button>
        }
      />

      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-12 rounded-md" />
          ))}
        </div>
      ) : isError ? (
        <div className="rounded-md border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
          Failed to load clients
          {error instanceof Error && error.message ? `: ${error.message}` : '.'}
        </div>
      ) : rows.length === 0 ? (
        <div className="rounded-md border border-border bg-card px-4 py-12 text-center text-sm text-muted-foreground">
          No clients yet. Click <span className="font-medium">Add Client</span> to provision your
          first tenant.
        </div>
      ) : (
        <DataTable table={table} />
      )}

      <AddClientDialog orgId={orgId} open={dialogOpen} onOpenChange={setDialogOpen} />
    </div>
  );
}
