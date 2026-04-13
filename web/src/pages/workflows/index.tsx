import { useState, useEffect, useRef, Fragment } from 'react';
import { Link } from 'react-router';
import {
  getCoreRowModel,
  getExpandedRowModel,
  useReactTable,
  flexRender,
  type ExpandedState,
  type ColumnDef,
} from '@tanstack/react-table';
import {
  Plus,
  GitBranch,
  Layers,
  Clock,
  Play,
  Pencil,
  Copy,
  MoreHorizontal,
  Trash2,
  Loader2,
  CircleCheckBig,
  Download,
  Archive,
} from 'lucide-react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { EmptyState, ErrorState } from '@patchiq/ui';
import { useCan } from '../../app/auth/AuthContext';
import { useWorkflows } from '../../flows/policy-workflow/hooks/use-workflows';
import { fetchJSON, fetchVoid } from '../../flows/policy-workflow/hooks/fetch-json';
import { DataTablePagination } from '../../components/data-table';
import { WorkflowInlineEditor } from './workflow-inline-editor';
import { timeAgo } from '../../lib/time';
import { uuid } from '../../lib/uuid';
import type { WorkflowListItem, WorkflowDetail } from '../../flows/policy-workflow/types';

// ─── Helpers ──────────────────────────────────────────────────────────────────

type StatusFilter = 'all' | 'published' | 'draft' | 'archived';

const statusConfig: Record<string, { label: string; color: string }> = {
  published: { label: 'Published', color: 'var(--accent)' },
  draft: { label: 'Draft', color: 'var(--signal-warning)' },
  archived: { label: 'Archived', color: 'var(--text-muted)' },
};

function StatusBadge({ status }: { status: string }) {
  const cfg = statusConfig[status] ?? statusConfig.draft;
  return (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        padding: '2px 8px',
        borderRadius: 9999,
        fontSize: 10,
        fontFamily: 'var(--font-mono)',
        fontWeight: 600,
        textTransform: 'uppercase',
        letterSpacing: '0.04em',
        color: cfg.color,
        background:
          status === 'published'
            ? 'color-mix(in srgb, var(--accent) 8%, transparent)'
            : status === 'draft'
              ? 'color-mix(in srgb, var(--signal-warning) 8%, transparent)'
              : 'var(--bg-inset)',
        border: `1px solid ${
          status === 'published'
            ? 'color-mix(in srgb, var(--accent) 20%, transparent)'
            : status === 'draft'
              ? 'color-mix(in srgb, var(--signal-warning) 20%, transparent)'
              : 'var(--border)'
        }`,
      }}
    >
      {cfg.label}
    </span>
  );
}

function LastRunStatus({ status }: { status: string | null }) {
  if (!status)
    return (
      <span
        title="This workflow has not been executed yet"
        style={{ color: 'var(--text-muted)', fontSize: 11 }}
      >
        —
      </span>
    );
  switch (status) {
    case 'completed':
      return <CircleCheckBig style={{ width: 12, height: 12, color: 'var(--signal-healthy)' }} />;
    case 'failed':
      return (
        <span
          style={{ fontSize: 10, color: 'var(--signal-critical)', fontWeight: 700, lineHeight: 1 }}
        >
          ✗
        </span>
      );
    case 'running':
      return (
        <Loader2
          style={{ width: 12, height: 12, color: 'var(--signal-warning)' }}
          className="animate-spin"
        />
      );
    default:
      return <span style={{ color: 'var(--text-muted)', fontSize: 11 }}>—</span>;
  }
}

// ─── Stat Card Button ─────────────────────────────────────────────────────────

interface StatCardProps {
  label: string;
  value: number | undefined;
  valueColor?: string;
  icon?: React.ComponentType<React.SVGProps<SVGSVGElement> & { style?: React.CSSProperties }>;
  active?: boolean;
  onClick?: () => void;
}

function StatCard({ label, value, valueColor, icon: Icon }: StatCardProps) {
  return (
    <div
      style={{
        flex: 1,
        minWidth: 0,
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'flex-start',
        padding: '12px 14px',
        background: 'var(--bg-card)',
        border: '1px solid var(--border)',
        borderRadius: 8,
        textAlign: 'left',
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        {Icon && (
          <Icon
            style={{
              width: 16,
              height: 16,
              color: valueColor ?? 'var(--text-muted)',
              flexShrink: 0,
            }}
          />
        )}
        <span
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 22,
            fontWeight: 700,
            lineHeight: 1,
            color:
              value != null && value > 0
                ? (valueColor ?? 'var(--text-emphasis)')
                : 'var(--text-muted)',
            letterSpacing: '-0.02em',
          }}
        >
          {value ?? '—'}
        </span>
      </div>
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

// ─── Skeleton for stat cards ──────────────────────────────────────────────────

function StatCardSkeleton() {
  return (
    <div
      style={{
        flex: 1,
        minWidth: 0,
        padding: '12px 14px',
        background: 'var(--bg-card)',
        border: '1px solid var(--border)',
        borderRadius: 8,
        display: 'flex',
        flexDirection: 'column',
        gap: 6,
      }}
    >
      <div
        style={{
          height: 22,
          width: 40,
          borderRadius: 4,
          background: 'var(--bg-inset)',
          animation: 'pulse 1.5s ease-in-out infinite',
        }}
      />
      <div
        style={{
          height: 10,
          width: 48,
          borderRadius: 3,
          background: 'var(--bg-inset)',
          animation: 'pulse 1.5s ease-in-out infinite',
        }}
      />
    </div>
  );
}

// ─── Skeleton rows ────────────────────────────────────────────────────────────

function SkeletonRows({ cols, rows = 8 }: { cols: number; rows?: number }) {
  return (
    <>
      {Array.from({ length: rows }).map((_, i) => (
        <tr key={i}>
          {Array.from({ length: cols }).map((__, j) => (
            <td key={j} style={{ padding: '10px 16px' }}>
              <div
                style={{
                  height: 14,
                  borderRadius: 4,
                  background: 'var(--bg-inset)',
                  width: j === 0 ? '30%' : j === 1 ? '55%' : '40%',
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

// ─── Expanded Row ─────────────────────────────────────────────────────────────

function ExpandedWorkflowRow({
  workflow,
  onPreview,
}: {
  workflow: WorkflowListItem;
  onPreview: () => void;
}) {
  const CARD: React.CSSProperties = {
    background: 'var(--bg-inset)',
    border: '1px solid var(--border)',
    borderRadius: 6,
    padding: '12px 14px',
  };
  const LBL: React.CSSProperties = {
    fontFamily: 'var(--font-mono)',
    fontSize: 9,
    fontWeight: 600,
    textTransform: 'uppercase',
    letterSpacing: '0.07em',
    color: 'var(--text-muted)',
    marginBottom: 10,
  };
  const BTN: React.CSSProperties = {
    display: 'inline-flex',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 5,
    padding: '7px 14px',
    borderRadius: 5,
    fontSize: 13,
    fontWeight: 500,
    cursor: 'pointer',
    border: '1px solid var(--border)',
    background: 'transparent',
    color: 'var(--text-secondary)',
    letterSpacing: '0.01em',
    width: '100%',
    textDecoration: 'none',
    fontFamily: 'var(--font-sans)',
  };
  const TH: React.CSSProperties = {
    padding: '6px 10px',
    textAlign: 'left',
    fontFamily: 'var(--font-mono)',
    fontSize: 10,
    fontWeight: 600,
    letterSpacing: '0.05em',
    color: 'var(--text-muted)',
    textTransform: 'uppercase',
    borderBottom: '1px solid var(--border)',
  };
  const TD: React.CSSProperties = {
    padding: '6px 10px',
    fontSize: 12,
    color: 'var(--text-primary)',
    borderBottom: '1px solid var(--border)',
  };

  const lastStatusColor =
    workflow.last_run_status === 'completed'
      ? 'var(--signal-healthy)'
      : workflow.last_run_status === 'failed'
        ? 'var(--signal-critical)'
        : 'var(--text-muted)';

  return (
    <div
      style={{
        padding: '8px 10px',
        background: 'var(--bg-page)',
        borderTop: '1px solid var(--border)',
        display: 'flex',
        gap: 8,
        alignItems: 'stretch',
      }}
    >
      {/* Workflow Info table */}
      <div style={{ ...CARD, flex: '0 0 500px' }}>
        <div style={LBL}>Workflow Info</div>
        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
          <thead>
            <tr>
              <th style={TH}>Field</th>
              <th style={TH}>Value</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td style={{ ...TD, color: 'var(--text-secondary)', width: 120 }}>Description</td>
              <td style={TD}>
                {workflow.description || (
                  <span style={{ color: 'var(--text-muted)', fontStyle: 'italic' }}>
                    No description provided
                  </span>
                )}
              </td>
            </tr>
            <tr>
              <td style={{ ...TD, color: 'var(--text-secondary)' }}>Version</td>
              <td
                style={{
                  ...TD,
                  fontFamily: 'var(--font-mono)',
                  color: 'var(--text-secondary)',
                }}
              >
                v{workflow.current_version}
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      {/* Execution Summary table */}
      <div style={{ ...CARD, flex: '0 0 480px', marginLeft: 24 }}>
        <div style={LBL}>Execution Summary</div>
        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
          <thead>
            <tr>
              <th style={TH}>Metric</th>
              <th style={{ ...TH, textAlign: 'right' }}>Value</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td style={{ ...TD, color: 'var(--text-secondary)' }}>Total Runs</td>
              <td
                style={{
                  ...TD,
                  textAlign: 'right',
                  fontFamily: 'var(--font-mono)',
                  color: 'var(--text-primary)',
                }}
              >
                {workflow.total_runs}
              </td>
            </tr>
            <tr>
              <td style={{ ...TD, color: 'var(--text-secondary)' }}>Last Run</td>
              <td
                style={{
                  ...TD,
                  textAlign: 'right',
                  fontFamily: 'var(--font-mono)',
                  color: 'var(--text-secondary)',
                }}
              >
                {workflow.last_run_at ? timeAgo(workflow.last_run_at) : 'Never'}
              </td>
            </tr>
            <tr>
              <td style={{ ...TD, color: 'var(--text-secondary)' }}>Last Status</td>
              <td
                style={{
                  ...TD,
                  textAlign: 'right',
                  fontFamily: 'var(--font-mono)',
                  color: lastStatusColor,
                }}
              >
                {workflow.last_run_status ?? '—'}
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      {/* Action buttons — outside cards */}
      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          gap: 6,
          flexShrink: 0,
          width: 160,
          alignSelf: 'start',
          marginLeft: 40,
        }}
      >
        <Link
          to={`/workflows/${workflow.id}/edit`}
          onClick={(e) => e.stopPropagation()}
          style={{
            ...BTN,
            color: 'var(--btn-accent-text, #000)',
            borderColor: 'var(--accent)',
            background: 'var(--accent)',
          }}
        >
          <Pencil style={{ width: 12, height: 12 }} />
          Edit
        </Link>
        <button
          type="button"
          style={BTN}
          onClick={(e) => {
            e.stopPropagation();
            onPreview();
          }}
        >
          <Play style={{ width: 12, height: 12 }} />
          Preview
        </button>
      </div>
    </div>
  );
}

// ─── More Actions Menu ───────────────────────────────────────────────────────

function MoreActionsMenu({
  workflow,
  onDelete,
}: {
  workflow: WorkflowListItem;
  onDelete: (id: string) => void;
}) {
  const can = useCan();
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [open]);

  const itemStyle: React.CSSProperties = {
    display: 'flex',
    alignItems: 'center',
    gap: 8,
    width: '100%',
    padding: '7px 12px',
    border: 'none',
    background: 'none',
    fontSize: 12,
    cursor: 'pointer',
    borderRadius: 4,
    color: 'var(--text-secondary)',
    transition: 'background 0.1s',
  };

  return (
    <div ref={ref} style={{ position: 'relative' }}>
      <button
        type="button"
        aria-label={`More actions for ${workflow.name}`}
        onClick={() => setOpen(!open)}
        style={{
          width: 26,
          height: 26,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          borderRadius: 5,
          border: '1px solid var(--border)',
          background: 'transparent',
          color: 'var(--text-muted)',
          cursor: 'pointer',
          padding: 0,
          transition: 'all 0.15s',
        }}
        onMouseEnter={(e) => {
          e.currentTarget.style.borderColor = 'var(--border-hover)';
          e.currentTarget.style.color = 'var(--text-primary)';
        }}
        onMouseLeave={(e) => {
          e.currentTarget.style.borderColor = 'var(--border)';
          e.currentTarget.style.color = 'var(--text-muted)';
        }}
      >
        <MoreHorizontal style={{ width: 11, height: 11 }} />
      </button>
      {open && (
        <div
          style={{
            position: 'absolute',
            right: 0,
            top: '100%',
            marginTop: 4,
            zIndex: 50,
            background: 'var(--bg-elevated)',
            border: '1px solid var(--border)',
            borderRadius: 8,
            boxShadow: 'var(--shadow-lg)',
            minWidth: 160,
            padding: 4,
          }}
        >
          <button
            style={{
              ...itemStyle,
              color: 'var(--signal-critical)',
              opacity: !can('workflows', 'execute') ? 0.5 : undefined,
            }}
            disabled={!can('workflows', 'execute')}
            title={!can('workflows', 'execute') ? "You don't have permission" : undefined}
            onMouseEnter={(e) => {
              e.currentTarget.style.background = 'var(--bg-card-hover)';
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.background = 'none';
            }}
            onClick={() => {
              if (confirm(`Delete workflow "${workflow.name}"? This cannot be undone.`)) {
                onDelete(workflow.id);
              }
              setOpen(false);
            }}
          >
            <Trash2 style={{ width: 12, height: 12 }} />
            Delete
          </button>
        </div>
      )}
    </div>
  );
}

// ─── Main Page ────────────────────────────────────────────────────────────────

// Inject responsive styles for hiding secondary columns at narrow viewports
const RESPONSIVE_STYLE_ID = 'workflows-responsive';
function ensureResponsiveStyles() {
  if (document.getElementById(RESPONSIVE_STYLE_ID)) return;
  const style = document.createElement('style');
  style.id = RESPONSIVE_STYLE_ID;
  style.textContent = `
    @media (max-width: 768px) {
      .wf-col-nodes, .wf-col-runs { display: none !important; }
    }
  `;
  document.head.appendChild(style);
}

export function WorkflowsPage() {
  const can = useCan();
  useEffect(() => {
    ensureResponsiveStyles();
    document.title = 'Workflows — PatchIQ';
    return () => {
      document.title = 'PatchIQ';
    };
  }, []);

  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');
  const [expanded, setExpanded] = useState<ExpandedState>({});
  const [cursors, setCursors] = useState<string[]>([]);
  const [selectedWorkflow, setSelectedWorkflow] = useState<WorkflowListItem | null>(null);
  const currentCursor = cursors[cursors.length - 1];

  const { data, isLoading, isError, refetch } = useWorkflows({
    cursor: currentCursor,
    limit: 25,
    status: statusFilter === 'all' ? undefined : statusFilter,
    search: search || undefined,
  });

  // Unfiltered query for stat card totals — always shows full counts regardless of search/filter
  const { data: unfilteredData } = useWorkflows({ limit: 25 });

  // Reset pagination when filters change
  useEffect(() => {
    setCursors([]);
  }, [statusFilter, search]);

  const queryClient = useQueryClient();

  const deleteWorkflow = useMutation({
    mutationFn: (id: string) => fetchVoid(`/api/v1/workflows/${id}`, { method: 'DELETE' }),
    onSuccess: () => {
      toast.success('Workflow deleted');
      void queryClient.invalidateQueries({ queryKey: ['workflows'] });
    },
    onError: () => {
      toast.error('Failed to delete workflow');
    },
  });

  const duplicateWorkflow = useMutation({
    mutationFn: async (id: string) => {
      const detail = await fetchJSON<WorkflowDetail>(`/api/v1/workflows/${id}`);
      const nodeIdMap = new Map<string, string>();
      const newNodes = (detail.nodes ?? []).map((n) => {
        const newId = uuid();
        nodeIdMap.set(n.id, newId);
        return {
          id: newId,
          node_type: n.node_type,
          label: n.label,
          position_x: n.position_x,
          position_y: n.position_y,
          config: n.config,
        };
      });
      const newEdges = (detail.edges ?? []).map((e) => ({
        source_node_id: nodeIdMap.get(e.source_node_id) ?? e.source_node_id,
        target_node_id: nodeIdMap.get(e.target_node_id) ?? e.target_node_id,
        label: e.label,
      }));
      return fetchJSON('/api/v1/workflows', {
        method: 'POST',
        body: JSON.stringify({
          name: `${detail.name} (Copy)`,
          description: detail.description,
          nodes: newNodes,
          edges: newEdges,
        }),
      });
    },
    onSuccess: () => {
      toast.success('Workflow duplicated');
      void queryClient.invalidateQueries({ queryKey: ['workflows'] });
    },
    onError: () => {
      toast.error('Failed to duplicate workflow');
    },
  });

  const runWorkflow = useMutation({
    mutationFn: (id: string) => fetchVoid(`/api/v1/workflows/${id}/execute`, { method: 'POST' }),
    onSuccess: () => {
      toast.success('Workflow execution started');
      void queryClient.invalidateQueries({ queryKey: ['workflows'] });
    },
    onError: () => {
      toast.error('Failed to start workflow');
    },
  });

  // Keep selectedWorkflow in sync with refetched data.
  useEffect(() => {
    if (!selectedWorkflow || !data?.data) return;
    const updated = data.data.find((w) => w.id === selectedWorkflow.id);
    if (updated && updated.current_status !== selectedWorkflow.current_status) {
      setSelectedWorkflow(updated);
    }
  }, [data?.data, selectedWorkflow]);

  // Derived counts from current page data (simple client-side counts)
  const allWorkflows = data?.data ?? [];
  const totalCount = data?.total_count ?? 0;

  // Stat card counts — always use unfiltered data so cards show true totals
  const unfilteredWorkflows = unfilteredData?.data ?? [];
  const unfilteredTotal = unfilteredData?.total_count ?? 0;
  const publishedCount = unfilteredWorkflows.filter((w) => w.current_status === 'published').length;
  const draftCount = unfilteredWorkflows.filter((w) => w.current_status === 'draft').length;
  const archivedCount = unfilteredWorkflows.filter((w) => w.current_status === 'archived').length;

  // ─── Column definitions ─────────────────────────────────────────────────────

  const expandCol: ColumnDef<WorkflowListItem> = {
    id: 'expand',
    size: 36,
    header: () => null,
    cell: ({ row }) => (
      <button
        type="button"
        aria-label={
          row.getIsExpanded() ? `Collapse ${row.original.name}` : `Expand ${row.original.name}`
        }
        onClick={(e) => {
          e.stopPropagation();
          row.toggleExpanded();
        }}
        style={{
          width: 24,
          height: 24,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          background: 'transparent',
          border: 'none',
          borderRadius: 4,
          cursor: 'pointer',
          color: 'var(--text-muted)',
          padding: 0,
          transition: 'color 0.15s, background 0.15s',
        }}
        onMouseEnter={(e) => {
          e.currentTarget.style.color = 'var(--text-primary)';
          e.currentTarget.style.background = 'var(--bg-card-hover)';
        }}
        onMouseLeave={(e) => {
          e.currentTarget.style.color = 'var(--text-muted)';
          e.currentTarget.style.background = 'transparent';
        }}
      >
        <svg
          width="12"
          height="12"
          viewBox="0 0 12 12"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          style={{
            transform: row.getIsExpanded() ? 'rotate(90deg)' : 'rotate(0deg)',
            transition: 'transform 0.2s',
          }}
        >
          <path d="M4.5 2.5L8.5 6L4.5 9.5" />
        </svg>
      </button>
    ),
  };

  const columns: ColumnDef<WorkflowListItem>[] = [
    expandCol,
    {
      id: 'name',
      accessorKey: 'name',
      header: 'Name',
      cell: ({ row }) => (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          <span style={{ fontWeight: 600, color: 'var(--text-emphasis)', fontSize: 13 }}>
            {row.original.name}
          </span>
          {row.original.description && (
            <span
              style={{
                fontSize: 11,
                color: 'var(--text-muted)',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
                maxWidth: 360,
              }}
            >
              {row.original.description}
            </span>
          )}
        </div>
      ),
    },
    {
      id: 'status',
      accessorKey: 'current_status',
      header: 'Status',
      size: 100,
      cell: ({ getValue }) => <StatusBadge status={getValue() as string} />,
    },
    {
      id: 'nodes',
      accessorKey: 'node_count',
      header: () => (
        <span title="Number of nodes in workflow" className="wf-col-nodes">
          Nodes
        </span>
      ),
      size: 80,
      meta: { className: 'wf-col-nodes' },
      cell: ({ getValue }) => {
        const count = getValue() as number;
        return (
          <div style={{ display: 'flex', alignItems: 'center', gap: 5 }}>
            <GitBranch
              style={{ width: 11, height: 11, color: 'var(--text-muted)', flexShrink: 0 }}
            />
            <span
              style={{ fontFamily: 'var(--font-mono)', fontSize: 12, color: 'var(--text-primary)' }}
            >
              {count}
            </span>
          </div>
        );
      },
    },
    {
      id: 'runs',
      accessorKey: 'total_runs',
      header: () => (
        <span title="Total executions (all time)" className="wf-col-runs">
          Runs
        </span>
      ),
      size: 80,
      meta: { className: 'wf-col-runs' },
      cell: ({ getValue }) => {
        const count = getValue() as number;
        return (
          <div style={{ display: 'flex', alignItems: 'center', gap: 5 }}>
            <Play style={{ width: 10, height: 10, color: 'var(--text-muted)', flexShrink: 0 }} />
            <span
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 12,
                color: 'var(--text-secondary)',
              }}
            >
              {count}
            </span>
          </div>
        );
      },
    },
    {
      id: 'last_run',
      accessorKey: 'last_run_at',
      header: 'Last Run',
      cell: ({ row }) => {
        const at = row.original.last_run_at;
        const status = row.original.last_run_status;
        return (
          <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
            <LastRunStatus status={status} />
            <span
              title={at ? new Date(at).toLocaleString() : undefined}
              style={{ fontFamily: 'var(--font-mono)', fontSize: 11, color: 'var(--text-muted)' }}
            >
              {at ? timeAgo(at) : 'Never'}
            </span>
          </div>
        );
      },
    },
    {
      id: 'updated',
      accessorKey: 'updated_at',
      header: 'Updated',
      cell: ({ getValue }) => {
        const val = getValue() as string | undefined;
        return (
          <div style={{ display: 'flex', alignItems: 'center', gap: 5 }}>
            <Clock style={{ width: 10, height: 10, color: 'var(--text-muted)', flexShrink: 0 }} />
            <span
              title={val ? new Date(val).toLocaleString() : undefined}
              style={{ fontFamily: 'var(--font-mono)', fontSize: 11, color: 'var(--text-muted)' }}
            >
              {val ? timeAgo(val) : '—'}
            </span>
          </div>
        );
      },
    },
    {
      id: 'actions',
      header: '',
      size: 36,
      cell: ({ row }) => (
        <div
          style={{ display: 'flex', alignItems: 'center', gap: 4 }}
          onClick={(e) => e.stopPropagation()}
        >
          {row.original.current_status === 'published' && (
            <button
              type="button"
              aria-label={`Run ${row.original.name}`}
              onClick={(e) => {
                e.stopPropagation();
                runWorkflow.mutate(row.original.id);
              }}
              disabled={runWorkflow.isPending}
              style={{
                width: 26,
                height: 26,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                borderRadius: 5,
                border: '1px solid var(--border)',
                background: 'transparent',
                color: 'var(--text-muted)',
                cursor: 'pointer',
                padding: 0,
                transition: 'all 0.15s',
              }}
              onMouseEnter={(e) => {
                e.currentTarget.style.borderColor = 'var(--accent)';
                e.currentTarget.style.color = 'var(--accent)';
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.borderColor = 'var(--border)';
                e.currentTarget.style.color = 'var(--text-muted)';
              }}
            >
              <Play style={{ width: 11, height: 11 }} />
            </button>
          )}
          <Link
            to={`/workflows/${row.original.id}/edit`}
            aria-label={`Edit ${row.original.name}`}
            style={{
              width: 26,
              height: 26,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              borderRadius: 5,
              border: '1px solid var(--border)',
              background: 'transparent',
              color: 'var(--text-muted)',
              textDecoration: 'none',
              transition: 'all 0.15s',
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.borderColor = 'var(--accent)';
              e.currentTarget.style.color = 'var(--accent)';
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.borderColor = 'var(--border)';
              e.currentTarget.style.color = 'var(--text-muted)';
            }}
          >
            <Pencil style={{ width: 11, height: 11 }} />
          </Link>
          <button
            type="button"
            aria-label={`Duplicate ${row.original.name}`}
            onClick={(e) => {
              e.stopPropagation();
              duplicateWorkflow.mutate(row.original.id);
            }}
            disabled={duplicateWorkflow.isPending}
            style={{
              width: 26,
              height: 26,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              borderRadius: 5,
              border: '1px solid var(--border)',
              background: 'transparent',
              color: 'var(--text-muted)',
              cursor: duplicateWorkflow.isPending ? 'not-allowed' : 'pointer',
              padding: 0,
              transition: 'all 0.15s',
              opacity: duplicateWorkflow.isPending ? 0.6 : 1,
            }}
            onMouseEnter={(e) => {
              if (!duplicateWorkflow.isPending) {
                e.currentTarget.style.borderColor = 'var(--border-hover)';
                e.currentTarget.style.color = 'var(--text-primary)';
              }
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.borderColor = 'var(--border)';
              e.currentTarget.style.color = 'var(--text-muted)';
            }}
          >
            {duplicateWorkflow.isPending ? (
              <Loader2 style={{ width: 11, height: 11 }} className="animate-spin" />
            ) : (
              <Copy style={{ width: 11, height: 11 }} />
            )}
          </button>
          <MoreActionsMenu workflow={row.original} onDelete={(id) => deleteWorkflow.mutate(id)} />
        </div>
      ),
    },
  ];

  const table = useReactTable({
    data: allWorkflows,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getExpandedRowModel: getExpandedRowModel(),
    onExpandedChange: setExpanded,
    state: { expanded },
    getRowId: (row) => row.id,
  });

  // ─── Error state ────────────────────────────────────────────────────────────

  if (isError) {
    return (
      <div style={{ padding: '24px' }}>
        <ErrorState
          title="Failed to load workflows"
          message="Unable to fetch workflow data from the server."
          onRetry={refetch}
        />
      </div>
    );
  }

  // ─── Render ─────────────────────────────────────────────────────────────────

  return (
    <div
      style={{
        padding: '24px',
        display: 'flex',
        flexDirection: 'column',
        gap: 16,
        minHeight: '100%',
        background: 'var(--bg-page)',
      }}
    >
      {/* Stat Cards */}
      <div style={{ display: 'flex', gap: 8 }}>
        {isLoading ? (
          <>
            <StatCardSkeleton />
            <StatCardSkeleton />
            <StatCardSkeleton />
            <StatCardSkeleton />
          </>
        ) : (
          <>
            <StatCard label="Total" value={unfilteredTotal} icon={Layers} />
            <StatCard
              label="Published"
              value={publishedCount}
              valueColor="var(--accent)"
              icon={CircleCheckBig}
            />
            <StatCard
              label="Draft"
              value={draftCount}
              valueColor="var(--signal-warning)"
              icon={Pencil}
            />
            <StatCard
              label="Archived"
              value={archivedCount}
              valueColor="var(--text-muted)"
              icon={Archive}
            />
          </>
        )}
      </div>

      {/* Filter Bar + Actions */}
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
          role="tablist"
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
              aria-label="Search workflows"
              placeholder="Search workflows..."
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
            {(
              [
                ['All', 'all', totalCount, 'var(--accent)'],
                ['Published', 'published', publishedCount, 'var(--accent)'],
                ['Draft', 'draft', draftCount, 'var(--signal-warning)'],
                ['Archived', 'archived', archivedCount, 'var(--text-muted)'],
              ] as const
            ).map(([label, value, count, color]) => {
              const active = statusFilter === value;
              return (
                <button
                  key={label}
                  type="button"
                  role="tab"
                  aria-selected={active}
                  onClick={() => setStatusFilter(value === 'all' || !active ? value : 'all')}
                  style={{
                    display: 'inline-flex',
                    alignItems: 'center',
                    gap: 4,
                    padding: '3px 9px',
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
                  {label}
                  {count != null && count > 0 && (
                    <span
                      style={{
                        fontFamily: 'var(--font-mono)',
                        fontSize: 10,
                        color: active ? color : 'var(--text-faint)',
                      }}
                    >
                      {count}
                    </span>
                  )}
                </button>
              );
            })}
          </div>
        </div>

        {/* Export CSV button */}
        <button
          type="button"
          onClick={() => {
            if (!allWorkflows.length) {
              toast.error('No workflows to export');
              return;
            }
            const headers = ['Name', 'Status', 'Nodes', 'Runs', 'Last Run', 'Updated'];
            const rows = allWorkflows.map((w) => [
              `"${w.name.replace(/"/g, '""')}"`,
              w.current_status,
              String(w.node_count),
              String(w.total_runs),
              w.last_run_at ? new Date(w.last_run_at).toISOString() : '',
              w.updated_at ? new Date(w.updated_at).toISOString() : '',
            ]);
            const csv = [headers.join(','), ...rows.map((r) => r.join(','))].join('\n');
            const blob = new Blob([csv], { type: 'text/csv' });
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = `workflows-${new Date().toISOString().slice(0, 10)}.csv`;
            a.click();
            URL.revokeObjectURL(url);
          }}
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: 6,
            padding: '5px 12px',
            borderRadius: 6,
            border: '1px solid var(--border)',
            background: 'var(--bg-card)',
            color: 'var(--text-secondary)',
            fontSize: 12,
            fontWeight: 500,
            cursor: 'pointer',
            fontFamily: 'var(--font-sans)',
            whiteSpace: 'nowrap',
            flexShrink: 0,
            transition: 'border-color 0.15s',
          }}
          onMouseEnter={(e) => {
            e.currentTarget.style.borderColor = 'var(--border-hover)';
          }}
          onMouseLeave={(e) => {
            e.currentTarget.style.borderColor = 'var(--border)';
          }}
        >
          <Download style={{ width: 13, height: 13 }} />
          Export CSV
        </button>

        {/* New Workflow button */}
        <Link
          to="/workflows/new"
          onClick={(e) => {
            if (!can('workflows', 'execute')) e.preventDefault();
          }}
          aria-disabled={!can('workflows', 'execute')}
          title={!can('workflows', 'execute') ? "You don't have permission" : undefined}
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: 6,
            padding: '5px 12px',
            borderRadius: 6,
            background: 'var(--accent)',
            color: 'var(--btn-accent-text, #000)',
            border: '1px solid var(--accent)',
            fontSize: 12,
            fontWeight: 600,
            textDecoration: 'none',
            fontFamily: 'var(--font-sans)',
            whiteSpace: 'nowrap',
            flexShrink: 0,
            opacity: !can('workflows', 'execute') ? 0.5 : undefined,
            pointerEvents: !can('workflows', 'execute') ? 'none' : undefined,
          }}
        >
          <Plus style={{ width: 13, height: 13 }} />
          New Workflow
        </Link>
      </div>

      {/* Table */}
      <div
        style={{
          position: 'relative',
          width: '100%',
          overflowX: 'auto',
          borderRadius: 8,
          border: '1px solid var(--border)',
          background: 'var(--bg-card)',
          boxShadow: 'var(--shadow-sm)',
        }}
      >
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13 }}>
          <thead>
            {table.getHeaderGroups().map((headerGroup) => (
              <tr key={headerGroup.id}>
                {headerGroup.headers.map((header) => (
                  <th
                    key={header.id}
                    className={
                      (header.column.columnDef.meta as Record<string, string> | undefined)
                        ?.className
                    }
                    style={{
                      height: 40,
                      padding: '0 16px',
                      textAlign: 'left',
                      verticalAlign: 'middle',
                      fontFamily: 'var(--font-mono)',
                      fontSize: 11,
                      fontWeight: 600,
                      letterSpacing: '0.05em',
                      color: 'var(--text-muted)',
                      background: 'var(--bg-inset)',
                      borderBottom: '1px solid var(--border)',
                      whiteSpace: 'nowrap',
                    }}
                  >
                    {header.isPlaceholder
                      ? null
                      : flexRender(header.column.columnDef.header, header.getContext())}
                  </th>
                ))}
              </tr>
            ))}
          </thead>
          <tbody>
            {isLoading ? (
              <SkeletonRows cols={columns.length} rows={8} />
            ) : allWorkflows.length === 0 ? (
              <tr>
                <td colSpan={columns.length} style={{ padding: '48px 24px', textAlign: 'center' }}>
                  {search || statusFilter !== 'all' ? (
                    <div
                      style={{
                        display: 'flex',
                        flexDirection: 'column',
                        alignItems: 'center',
                        gap: 8,
                      }}
                    >
                      {statusFilter === 'archived' ? (
                        <Archive style={{ width: 24, height: 24, color: 'var(--text-faint)' }} />
                      ) : (
                        <GitBranch style={{ width: 24, height: 24, color: 'var(--text-faint)' }} />
                      )}
                      <span style={{ fontSize: 13, color: 'var(--text-muted)' }}>
                        {statusFilter === 'archived'
                          ? 'No archived workflows. Archived workflows will appear here.'
                          : 'No workflows match your filters'}
                      </span>
                      <button
                        onClick={() => {
                          setStatusFilter('all');
                          setSearch('');
                        }}
                        style={{
                          marginTop: 8,
                          padding: '5px 12px',
                          borderRadius: 6,
                          border: '1px solid var(--border)',
                          background: 'none',
                          color: 'var(--text-secondary)',
                          fontSize: 12,
                          cursor: 'pointer',
                        }}
                      >
                        Clear filters
                      </button>
                    </div>
                  ) : (
                    <EmptyState
                      icon={GitBranch}
                      title="No workflows defined"
                      description="Create your first workflow to automate patch deployments across your endpoints."
                      action={{
                        label: 'Create Workflow',
                        onClick: () => {
                          window.location.href = '/workflows/new';
                        },
                      }}
                    />
                  )}
                </td>
              </tr>
            ) : (
              table.getRowModel().rows.map((row) => (
                <Fragment key={row.id}>
                  <tr
                    style={{
                      borderBottom: '1px solid var(--border)',
                      cursor: 'pointer',
                      transition: 'background 0.1s ease',
                    }}
                    onClick={() => row.toggleExpanded()}
                    onMouseEnter={(e) => {
                      (e.currentTarget as HTMLTableRowElement).style.background =
                        'var(--bg-card-hover)';
                    }}
                    onMouseLeave={(e) => {
                      (e.currentTarget as HTMLTableRowElement).style.background = '';
                    }}
                  >
                    {row.getVisibleCells().map((cell) => (
                      <td
                        key={cell.id}
                        className={
                          (cell.column.columnDef.meta as Record<string, string> | undefined)
                            ?.className
                        }
                        style={{
                          padding: '10px 16px',
                          verticalAlign: 'middle',
                          fontFamily: 'var(--font-sans)',
                          color: 'var(--text-primary)',
                        }}
                      >
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </td>
                    ))}
                  </tr>
                  {row.getIsExpanded() && (
                    <tr
                      style={{
                        borderBottom: '1px solid var(--border)',
                        background: 'var(--bg-inset)',
                      }}
                    >
                      <td colSpan={columns.length} style={{ padding: 0 }}>
                        <ExpandedWorkflowRow
                          workflow={row.original}
                          onPreview={() => setSelectedWorkflow(row.original)}
                        />
                      </td>
                    </tr>
                  )}
                </Fragment>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      {!isLoading && (
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <span
            style={{ fontSize: 12, color: 'var(--text-muted)', fontFamily: 'var(--font-mono)' }}
          >
            Showing {allWorkflows.length} of {totalCount} workflow{totalCount !== 1 ? 's' : ''}
          </span>
          {(!!data?.next_cursor || cursors.length > 0) && (
            <DataTablePagination
              hasNext={!!data?.next_cursor}
              hasPrev={cursors.length > 0}
              onNext={() => {
                if (data?.next_cursor) setCursors((prev) => [...prev, data.next_cursor!]);
              }}
              onPrev={() => setCursors((prev) => prev.slice(0, -1))}
            />
          )}
        </div>
      )}

      {/* Inline editor panel */}
      {selectedWorkflow && (
        <WorkflowInlineEditor
          key={selectedWorkflow.id}
          workflowId={selectedWorkflow.id}
          workflowName={selectedWorkflow.name}
          workflowStatus={selectedWorkflow.current_status}
          onClose={() => setSelectedWorkflow(null)}
        />
      )}
    </div>
  );
}
