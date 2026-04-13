import { useState, useMemo, useCallback } from 'react';
import { Plus, Download, Trash2, FileText } from 'lucide-react';
import { Skeleton, ErrorState, EmptyState } from '@patchiq/ui';
import {
  useReports,
  useReportCounts,
  useDeleteReport,
  useDownloadReport,
} from '../../api/hooks/useReports';
import type { ReportRecord } from '../../api/hooks/useReports';
import { DataTablePagination } from '../../components/data-table';
import { GenerateReportDialog } from './GenerateReportDialog';
import { useCan } from '../../app/auth/AuthContext';

// ── Helpers ──────────────────────────────────────────────────────────────────

function formatBytes(bytes: number | undefined): string {
  if (bytes == null) return '\u2014';
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function relativeTime(iso: string): string {
  const ms = Date.now() - new Date(iso).getTime();
  const mins = Math.floor(ms / 60000);
  if (mins < 1) return 'just now';
  if (mins < 60) return `${mins} min ago`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

// ── Stat Card ────────────────────────────────────────────────────────────────

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
        {value ?? '\u2014'}
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

// ── Status Badge ─────────────────────────────────────────────────────────────

const statusColorMap: Record<string, string> = {
  completed: 'var(--signal-healthy)',
  generating: 'var(--accent)',
  pending: 'var(--text-muted)',
  failed: 'var(--signal-critical)',
};

function StatusBadge({ status }: { status: string }) {
  const color = statusColorMap[status] ?? 'var(--text-muted)';
  const isLive = status === 'generating';
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
      <span
        style={{
          width: 6,
          height: 6,
          borderRadius: '50%',
          background: color,
          flexShrink: 0,
          animation: isLive ? 'pulse-dot 1.5s ease-in-out infinite' : undefined,
        }}
      />
      {status.charAt(0).toUpperCase() + status.slice(1)}
    </span>
  );
}

// ── Format Badge ─────────────────────────────────────────────────────────────

const formatColorMap: Record<string, string> = {
  pdf: 'var(--accent)',
  csv: 'var(--signal-healthy)',
  xlsx: 'var(--signal-warning)',
};

function FormatBadge({ format }: { format: string }) {
  const color = formatColorMap[format] ?? 'var(--text-muted)';
  return (
    <span
      style={{
        display: 'inline-flex',
        padding: '1px 6px',
        borderRadius: 100,
        fontSize: 10,
        fontWeight: 600,
        fontFamily: 'var(--font-mono)',
        textTransform: 'uppercase',
        background: `color-mix(in srgb, ${color} 12%, transparent)`,
        color,
        border: `1px solid color-mix(in srgb, ${color} 20%, transparent)`,
      }}
    >
      {format}
    </span>
  );
}

// ── Table styles ─────────────────────────────────────────────────────────────

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

// ── Main Component ───────────────────────────────────────────────────────────

export function ReportsPage() {
  const can = useCan();
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState<string | null>(null);
  const [typeFilter, setTypeFilter] = useState<string | null>(null);
  const [formatFilter, setFormatFilter] = useState<string | null>(null);
  const [cursors, setCursors] = useState<string[]>([]);
  const [dialogOpen, setDialogOpen] = useState(false);
  const currentCursor = cursors[cursors.length - 1];

  const { data, isLoading, isError, refetch } = useReports({
    cursor: currentCursor,
    limit: 25,
    status: statusFilter ?? undefined,
    report_type: typeFilter ?? undefined,
    format: formatFilter ?? undefined,
    search: search || undefined,
  });

  const { data: counts } = useReportCounts();
  const deleteMutation = useDeleteReport();
  const downloadReport = useDownloadReport();

  const reports = useMemo(() => data?.data ?? [], [data?.data]);

  const handleDownload = useCallback(
    async (report: ReportRecord) => {
      try {
        const ext = report.format;
        const filename = `${report.name}.${ext}`;
        await downloadReport(report.id, filename);
      } catch {
        // Download error is visible to the user via browser behavior
      }
    },
    [downloadReport],
  );

  const handleDelete = useCallback(
    (id: string) => {
      deleteMutation.mutate(id);
    },
    [deleteMutation],
  );

  // ── Error state ────────────────────────────────────────────────────────────
  if (isError) {
    return (
      <div style={{ padding: 24, background: 'var(--bg-page)' }}>
        <ErrorState
          title="Failed to load reports"
          message="Unable to fetch reports from the server."
          onRetry={refetch}
        />
      </div>
    );
  }

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
          label="All"
          value={counts?.total}
          active={statusFilter === null}
          onClick={() => {
            setStatusFilter(null);
            setCursors([]);
          }}
        />
        <StatCard
          label="Completed"
          value={counts?.completed}
          valueColor="var(--signal-healthy)"
          active={statusFilter === 'completed'}
          onClick={() => {
            setStatusFilter(statusFilter === 'completed' ? null : 'completed');
            setCursors([]);
          }}
        />
        <StatCard
          label="Generating"
          value={counts?.generating}
          valueColor="var(--accent)"
          active={statusFilter === 'generating'}
          onClick={() => {
            setStatusFilter(statusFilter === 'generating' ? null : 'generating');
            setCursors([]);
          }}
        />
        <StatCard
          label="Failed"
          value={counts?.failed}
          valueColor="var(--signal-critical)"
          active={statusFilter === 'failed'}
          onClick={() => {
            setStatusFilter(statusFilter === 'failed' ? null : 'failed');
            setCursors([]);
          }}
        />
        <StatCard
          label="Today"
          value={counts?.today}
          valueColor="var(--text-secondary)"
          active={false}
          onClick={() => {
            /* no-op, informational */
          }}
        />
      </div>

      {/* ── Filter Bar + Actions ──────────────────────────────────────────── */}
      <div style={{ display: 'flex', alignItems: 'stretch', gap: 8 }}>
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
              maxWidth: 300,
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
              aria-label="Search reports"
              placeholder="Search reports..."
              value={search}
              onChange={(e) => {
                setSearch(e.target.value);
                setCursors([]);
              }}
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
                onClick={() => {
                  setSearch('');
                  setCursors([]);
                }}
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

          {/* Type filter */}
          <select
            value={typeFilter ?? ''}
            onChange={(e) => {
              setTypeFilter(e.target.value || null);
              setCursors([]);
            }}
            aria-label="Filter by type"
            style={{
              background: 'var(--bg-inset)',
              border: '1px solid var(--border)',
              borderRadius: 6,
              padding: '5px 10px',
              fontSize: 12,
              color: 'var(--text-primary)',
              fontFamily: 'var(--font-sans)',
              outline: 'none',
            }}
          >
            <option value="">All Types</option>
            <option value="endpoints">Endpoints</option>
            <option value="patches">Patches</option>
            <option value="cves">CVEs</option>
            <option value="deployments">Deployments</option>
            <option value="compliance">Compliance</option>
            <option value="executive">Executive</option>
          </select>

          {/* Format filter */}
          <select
            value={formatFilter ?? ''}
            onChange={(e) => {
              setFormatFilter(e.target.value || null);
              setCursors([]);
            }}
            aria-label="Filter by format"
            style={{
              background: 'var(--bg-inset)',
              border: '1px solid var(--border)',
              borderRadius: 6,
              padding: '5px 10px',
              fontSize: 12,
              color: 'var(--text-primary)',
              fontFamily: 'var(--font-sans)',
              outline: 'none',
            }}
          >
            <option value="">All Formats</option>
            <option value="pdf">PDF</option>
            <option value="csv">CSV</option>
            <option value="xlsx">XLSX</option>
          </select>
        </div>

        {/* Generate Report button */}
        <button
          type="button"
          onClick={() => setDialogOpen(true)}
          disabled={!can('reports', 'create')}
          title={!can('reports', 'create') ? "You don't have permission" : undefined}
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: 6,
            padding: '5px 12px',
            borderRadius: 6,
            fontSize: 12,
            fontWeight: 600,
            cursor: !can('reports', 'create') ? 'not-allowed' : 'pointer',
            background: 'var(--accent)',
            color: 'var(--btn-accent-text, #000)',
            border: '1px solid var(--accent)',
            fontFamily: 'var(--font-sans)',
            whiteSpace: 'nowrap',
            opacity: !can('reports', 'create') ? 0.5 : 1,
          }}
        >
          <Plus style={{ width: 13, height: 13 }} />
          Generate Report
        </button>
      </div>

      {/* ── Content ───────────────────────────────────────────────────────── */}
      {isLoading ? (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
          {Array.from({ length: 7 }).map((_, i) => (
            <Skeleton key={i} className="h-11 rounded-lg" />
          ))}
        </div>
      ) : reports.length === 0 ? (
        <EmptyState
          icon={FileText}
          title="No reports yet"
          description="Generate a report to get started."
          action={
            can('reports', 'create')
              ? { label: 'Generate Report', onClick: () => setDialogOpen(true) }
              : undefined
          }
        />
      ) : (
        <>
          {/* Table */}
          <div
            style={{
              background: 'var(--bg-card)',
              border: '1px solid var(--border)',
              borderRadius: 8,
              overflow: 'hidden',
            }}
          >
            <table style={{ width: '100%', borderCollapse: 'collapse' }}>
              <thead>
                <tr>
                  <th style={TH_STYLE}>Report Name</th>
                  <th style={TH_STYLE}>Type</th>
                  <th style={TH_STYLE}>Format</th>
                  <th style={TH_STYLE}>Status</th>
                  <th style={TH_STYLE}>Size</th>
                  <th style={TH_STYLE}>Generated</th>
                  <th style={{ ...TH_STYLE, textAlign: 'right' }}>Actions</th>
                </tr>
              </thead>
              <tbody>
                {reports.map((report) => (
                  <tr
                    key={report.id}
                    style={{ transition: 'background 0.1s' }}
                    onMouseEnter={(e) =>
                      (e.currentTarget.style.background = 'var(--bg-card-hover, transparent)')
                    }
                    onMouseLeave={(e) => (e.currentTarget.style.background = 'transparent')}
                  >
                    <td style={TD_STYLE}>
                      <span
                        style={{
                          fontFamily: 'var(--font-mono)',
                          fontSize: 12,
                          color: 'var(--text-primary)',
                          fontWeight: 500,
                        }}
                      >
                        {report.name}
                      </span>
                    </td>
                    <td style={TD_STYLE}>
                      <span
                        style={{
                          display: 'inline-flex',
                          padding: '1px 6px',
                          borderRadius: 100,
                          fontSize: 10,
                          fontWeight: 600,
                          fontFamily: 'var(--font-mono)',
                          textTransform: 'uppercase',
                          background: 'color-mix(in srgb, var(--text-muted) 12%, transparent)',
                          color: 'var(--text-secondary)',
                          border:
                            '1px solid color-mix(in srgb, var(--text-muted) 20%, transparent)',
                        }}
                      >
                        {report.report_type}
                      </span>
                    </td>
                    <td style={TD_STYLE}>
                      <FormatBadge format={report.format} />
                    </td>
                    <td style={TD_STYLE}>
                      <StatusBadge status={report.status} />
                    </td>
                    <td style={TD_STYLE}>
                      <span
                        style={{
                          fontFamily: 'var(--font-mono)',
                          fontSize: 11,
                          color: 'var(--text-muted)',
                        }}
                      >
                        {report.status === 'completed'
                          ? formatBytes(report.file_size_bytes)
                          : '\u2014'}
                      </span>
                    </td>
                    <td style={TD_STYLE}>
                      <span
                        style={{
                          fontFamily: 'var(--font-mono)',
                          fontSize: 11,
                          color: 'var(--text-muted)',
                        }}
                      >
                        {relativeTime(report.created_at)}
                      </span>
                    </td>
                    <td style={{ ...TD_STYLE, textAlign: 'right' }}>
                      <div style={{ display: 'flex', gap: 4, justifyContent: 'flex-end' }}>
                        {report.status === 'completed' && (
                          <button
                            type="button"
                            onClick={() => handleDownload(report)}
                            title="Download"
                            style={{
                              padding: '4px 6px',
                              borderRadius: 4,
                              background: 'transparent',
                              border: '1px solid var(--border)',
                              cursor: 'pointer',
                              color: 'var(--text-muted)',
                              display: 'flex',
                              alignItems: 'center',
                              transition: 'all 0.15s',
                            }}
                            onMouseEnter={(e) => {
                              e.currentTarget.style.color = 'var(--accent)';
                              e.currentTarget.style.borderColor = 'var(--accent)';
                            }}
                            onMouseLeave={(e) => {
                              e.currentTarget.style.color = 'var(--text-muted)';
                              e.currentTarget.style.borderColor = 'var(--border)';
                            }}
                          >
                            <Download style={{ width: 13, height: 13 }} />
                          </button>
                        )}
                        {can('reports', 'delete') && (
                          <button
                            type="button"
                            onClick={() => handleDelete(report.id)}
                            title="Delete"
                            style={{
                              padding: '4px 6px',
                              borderRadius: 4,
                              background: 'transparent',
                              border: '1px solid var(--border)',
                              cursor: 'pointer',
                              color: 'var(--text-muted)',
                              display: 'flex',
                              alignItems: 'center',
                              transition: 'all 0.15s',
                            }}
                            onMouseEnter={(e) => {
                              e.currentTarget.style.color = 'var(--signal-critical)';
                              e.currentTarget.style.borderColor = 'var(--signal-critical)';
                            }}
                            onMouseLeave={(e) => {
                              e.currentTarget.style.color = 'var(--text-muted)';
                              e.currentTarget.style.borderColor = 'var(--border)';
                            }}
                          >
                            <Trash2 style={{ width: 13, height: 13 }} />
                          </button>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Pagination */}
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

      <GenerateReportDialog open={dialogOpen} onOpenChange={setDialogOpen} />
    </div>
  );
}
