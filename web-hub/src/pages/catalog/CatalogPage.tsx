import { useState, useMemo, useRef } from 'react';
import { Link } from 'react-router';
import {
  createColumnHelper,
  flexRender,
  getCoreRowModel,
  useReactTable,
} from '@tanstack/react-table';
import {
  Button,
  Input,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Badge,
  ErrorState,
} from '@patchiq/ui';
import {
  useCatalogEntries,
  useCatalogStats,
  useCreateCatalogEntry,
  useDeleteCatalogEntry,
} from '../../api/hooks/useCatalog';
import { toast } from 'sonner';
import { useNavigate } from 'react-router';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@patchiq/ui';
import { useFeeds } from '../../api/hooks/useFeeds';
import type { CatalogEntry } from '../../types/catalog';
import { SyncDots } from '../../components/SyncDots';
import { SeverityBadge } from '../../components/SeverityBadge';
import { SourceBadge } from '../../components/SourceBadge';
import { fmtDate } from '../../lib/format';
import { CatalogForm } from './CatalogForm';

// ─── Stat Card (clickable, matches PM style) ──────────────────────────────────

interface StatCardProps {
  label: string;
  value: string | number;
  valueColor?: string;
  active?: boolean;
  onClick?: () => void;
}

function StatCard({ label, value, valueColor, active, onClick }: StatCardProps) {
  const [hovered, setHovered] = useState(false);
  const isClickable = !!onClick;
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
        border: `1px solid ${active ? (valueColor ?? 'var(--accent)') : hovered && isClickable ? 'var(--border-hover)' : 'var(--border)'}`,
        borderRadius: 8,
        cursor: isClickable ? 'pointer' : 'default',
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

// ─── Skeleton rows with pulse animation ──────────────────────────────────────

function SkeletonRows({ cols, rows = 8 }: { cols: number; rows?: number }) {
  return (
    <>
      {Array.from({ length: rows }).map((_, i) => (
        <tr key={i} style={{ borderBottom: '1px solid var(--border)' }}>
          {Array.from({ length: cols }).map((__, j) => (
            <td key={j} style={{ padding: '10px 12px' }}>
              <div
                style={{
                  height: 14,
                  borderRadius: 4,
                  background: 'var(--bg-inset)',
                  width: j === 0 ? '20px' : j === 1 ? '60%' : j === 2 ? '40%' : '50%',
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

const PAGE_SIZE = 20;

const OS_EMOJI: Record<string, string> = {
  windows: '🪟',
  ubuntu: '🐧',
  linux: '🐧',
  rhel: '🎩',
  debian: '🐧',
  macos: '🍎',
};

const osEmoji = (os: string) => OS_EMOJI[os.toLowerCase()] ?? '💻';

const columnHelper = createColumnHelper<CatalogEntry & { _expanded?: boolean }>();

export function CatalogPage() {
  const [page, setPage] = useState(0);
  const [search, setSearch] = useState('');
  const [osFamily, setOsFamily] = useState('');
  const [severity, setSeverity] = useState('all');
  const [feedSourceId, setFeedSourceId] = useState('');
  const [dateRange, setDateRange] = useState('all');
  const [entryType, setEntryType] = useState('all');
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set());
  const [formOpen, setFormOpen] = useState(false);
  const [importing, setImporting] = useState(false);
  const [deletingEntry, setDeletingEntry] = useState<CatalogEntry | null>(null);
  const importInputRef = useRef<HTMLInputElement>(null);
  const navigate = useNavigate();
  const createEntry = useCreateCatalogEntry();
  const deleteMutation = useDeleteCatalogEntry();

  const { data: feedsData } = useFeeds();

  const { data, isLoading, isError, error, refetch } = useCatalogEntries({
    limit: PAGE_SIZE,
    offset: page * PAGE_SIZE,
    search: search || undefined,
    os_family: osFamily || undefined,
    severity: severity === 'all' ? undefined : severity,
    feed_source_id: feedSourceId || undefined,
    date_range: dateRange === 'all' ? undefined : dateRange,
    entry_type: entryType === 'all' ? undefined : entryType,
  });

  const { data: stats } = useCatalogStats();

  const totalPages = data ? Math.ceil(data.total / PAGE_SIZE) : 0;
  const startItem = page * PAGE_SIZE + 1;
  const endItem = Math.min((page + 1) * PAGE_SIZE, data?.total ?? 0);

  const sevCounts = {
    all: stats?.total_entries ?? 0,
    critical: stats?.by_severity.critical ?? 0,
    high: stats?.by_severity.high ?? 0,
    medium: stats?.by_severity.medium ?? 0,
    low: stats?.by_severity.low ?? 0,
  };

  const toggleRow = (id: string) => {
    setExpandedRows((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const columns = useMemo(
    () => [
      columnHelper.display({
        id: 'checkbox',
        header: () => (
          <input
            type="checkbox"
            className="rounded"
            style={{ accentColor: 'var(--accent)', borderColor: 'var(--border)' }}
          />
        ),
        cell: () => (
          <input
            type="checkbox"
            className="rounded"
            style={{ accentColor: 'var(--accent)', borderColor: 'var(--border)' }}
          />
        ),
      }),
      columnHelper.accessor('name', {
        header: 'Name',
        cell: (info) => (
          <Link
            to={`/catalog/${info.row.original.id}`}
            className="font-mono text-sm hover:underline"
            style={{ color: 'var(--accent)' }}
          >
            {info.getValue()}
          </Link>
        ),
      }),
      columnHelper.accessor('vendor', {
        header: 'Vendor',
        cell: (info) => <span className="text-sm text-muted-foreground">{info.getValue()}</span>,
      }),
      columnHelper.accessor('os_family', {
        header: 'OS',
        cell: (info) => (
          <span className="text-sm">
            {osEmoji(info.getValue())} {info.getValue()}
          </span>
        ),
      }),
      columnHelper.accessor('version', {
        header: 'Version',
        cell: (info) => (
          <span className="font-mono text-xs text-muted-foreground">{info.getValue() || '—'}</span>
        ),
      }),
      columnHelper.accessor('severity', {
        header: 'Severity',
        cell: (info) => <SeverityBadge severity={info.getValue()} />,
      }),
      columnHelper.accessor('cve_count', {
        header: 'CVEs',
        cell: (info) => (
          <span
            className="text-sm font-bold"
            style={{ color: 'var(--accent)', fontFamily: 'var(--font-mono)' }}
          >
            {info.getValue()}
          </span>
        ),
      }),
      columnHelper.display({
        id: 'synced_pms',
        header: 'Synced PMs',
        cell: (info) => {
          const { synced_count, total_clients } = info.row.original;
          return (
            <div className="flex items-center gap-2">
              <SyncDots synced={synced_count} total={total_clients} />
              <span className="text-xs text-muted-foreground">
                {synced_count}/{total_clients}
              </span>
            </div>
          );
        },
      }),
      columnHelper.accessor('feed_source_name', {
        header: 'Source',
        cell: (info) => <SourceBadge source={info.getValue()} />,
      }),
      columnHelper.accessor('release_date', {
        header: 'Published',
        cell: (info) => (
          <span className="text-xs text-muted-foreground">{fmtDate(info.getValue())}</span>
        ),
      }),
      columnHelper.accessor('updated_at', {
        header: 'Updated',
        cell: (info) => (
          <span className="text-xs text-muted-foreground">{fmtDate(info.getValue())}</span>
        ),
      }),
      columnHelper.display({
        id: 'expand',
        header: '',
        cell: (info) => {
          const id = info.row.original.id;
          const isOpen = expandedRows.has(id);
          return (
            <button
              onClick={() => toggleRow(id)}
              className="text-xs text-muted-foreground hover:text-foreground"
            >
              {isOpen ? '▲' : '▼'}
            </button>
          );
        },
      }),
    ],
    [expandedRows],
  );

  const table = useReactTable({
    data: data?.entries ?? [],
    columns,
    getCoreRowModel: getCoreRowModel(),
  });

  const handleExport = () => {
    const entries = data?.entries ?? [];
    if (entries.length === 0) {
      toast.error('No entries to export');
      return;
    }
    const headers = [
      'name',
      'vendor',
      'os_family',
      'version',
      'severity',
      'release_date',
      'cve_count',
      'feed_source_name',
    ];
    const rows = entries.map((e) =>
      [
        e.name,
        e.vendor,
        e.os_family,
        e.version,
        e.severity,
        e.release_date ?? '',
        e.cve_count,
        e.feed_source_name ?? '',
      ]
        .map((v) => `"${String(v ?? '').replace(/"/g, '""')}"`)
        .join(','),
    );
    const csv = [headers.join(','), ...rows].join('\n');
    const blob = new Blob([csv], { type: 'text/csv' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `catalog-${new Date().toISOString().slice(0, 10)}.csv`;
    a.click();
    URL.revokeObjectURL(url);
    toast.success(`Exported ${entries.length} entries`);
  };

  const handleImportFile = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    e.target.value = '';
    setImporting(true);
    try {
      const text = await file.text();
      const lines = text
        .split('\n')
        .map((l) => l.trim())
        .filter(Boolean);
      if (lines.length < 2) {
        toast.error('CSV must have a header row and at least one data row');
        return;
      }
      const headers = lines[0].split(',').map((h) => h.replace(/^"|"$/g, '').trim());
      const nameIdx = headers.indexOf('name');
      const vendorIdx = headers.indexOf('vendor');
      const osIdx = headers.indexOf('os_family');
      const versionIdx = headers.indexOf('version');
      const severityIdx = headers.indexOf('severity');
      const dateIdx = headers.indexOf('release_date');
      if (nameIdx === -1 || severityIdx === -1) {
        toast.error('CSV must have at least "name" and "severity" columns');
        return;
      }
      const parseRow = (line: string) => line.split(',').map((v) => v.replace(/^"|"$/g, '').trim());

      let success = 0;
      let failed = 0;
      for (const line of lines.slice(1)) {
        const cols = parseRow(line);
        try {
          await createEntry.mutateAsync({
            name: cols[nameIdx] ?? '',
            vendor: cols[vendorIdx] ?? '',
            os_family: cols[osIdx] ?? '',
            version: cols[versionIdx] ?? '',
            severity: cols[severityIdx] ?? 'medium',
            release_date: dateIdx !== -1 ? cols[dateIdx] || undefined : undefined,
          });
          success++;
        } catch {
          failed++;
        }
      }
      if (failed === 0) toast.success(`Imported ${success} entries`);
      else toast.warning(`Imported ${success} entries, ${failed} failed`);
    } catch {
      toast.error('Failed to parse CSV file');
    } finally {
      setImporting(false);
    }
  };

  return (
    <div
      style={{
        padding: 24,
        display: 'flex',
        flexDirection: 'column',
        gap: 16,
        background: 'var(--bg-page)',
      }}
    >
      {/* Header — matches PM style: title + count inline, actions right */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}
      >
        <input
          ref={importInputRef}
          type="file"
          accept=".csv"
          style={{ display: 'none' }}
          onChange={handleImportFile}
        />
      </div>

      {/* Stat Cards — flush (zero gap), matches PM connected strip */}
      {stats && (
        <div style={{ display: 'flex', gap: 8 }}>
          <StatCard label="Total Entries" value={stats.total_entries.toLocaleString()} />
          <StatCard
            label="New This Week"
            value={stats.new_this_week.toLocaleString()}
            valueColor="var(--accent)"
          />
          <StatCard label="CVEs Tracked" value={stats.cves_tracked.toLocaleString()} />
          <StatCard
            label="Synced to PMs"
            value={
              stats.total_for_sync_pct > 0
                ? `${Math.round((stats.synced_entries / stats.total_for_sync_pct) * 100)}%`
                : '0%'
            }
            valueColor="var(--signal-warning)"
          />
        </div>
      )}

      {/* Filter Bar */}
      <div
        style={{
          display: 'flex',
          flexWrap: 'wrap',
          alignItems: 'center',
          gap: 8,
          borderRadius: 8,
          padding: '10px 14px',
          border: '1px solid var(--border)',
          background: 'var(--bg-card)',
        }}
      >
        <Select
          value={severity}
          onValueChange={(v: string) => {
            setSeverity(v);
            setPage(0);
          }}
        >
          <SelectTrigger
            className="h-7 w-32 text-sm"
            style={{ borderColor: 'var(--border)', background: 'var(--bg-card)' }}
          >
            <SelectValue placeholder="Severity" />
          </SelectTrigger>
          <SelectContent>
            {(
              [
                ['all', 'All', sevCounts.all],
                ['critical', 'Critical', sevCounts.critical],
                ['high', 'High', sevCounts.high],
                ['medium', 'Medium', sevCounts.medium],
                ['low', 'Low', sevCounts.low],
              ] as const
            ).map(([val, label, count]) => (
              <SelectItem key={val} value={val}>
                {label}{' '}
                <span
                  style={{
                    color: 'var(--text-muted)',
                    fontFamily: 'var(--font-mono)',
                    fontSize: 10,
                  }}
                >
                  {count}
                </span>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select
          value={osFamily || 'all'}
          onValueChange={(v: string) => {
            setOsFamily(v === 'all' ? '' : v);
            setPage(0);
          }}
        >
          <SelectTrigger
            className="h-7 w-32 text-sm"
            style={{ borderColor: 'var(--border)', background: 'var(--bg-card)' }}
          >
            <SelectValue placeholder="All OS" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All OS</SelectItem>
            <SelectItem value="windows">Windows</SelectItem>
            <SelectItem value="ubuntu">Ubuntu</SelectItem>
            <SelectItem value="rhel">RHEL</SelectItem>
            <SelectItem value="debian">Debian</SelectItem>
          </SelectContent>
        </Select>
        <Select
          value={feedSourceId || 'all'}
          onValueChange={(v: string) => {
            setFeedSourceId(v === 'all' ? '' : v);
            setPage(0);
          }}
        >
          <SelectTrigger
            className="h-7 w-36 text-sm"
            style={{ borderColor: 'var(--border)', background: 'var(--bg-card)' }}
          >
            <SelectValue placeholder="All Sources" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Sources</SelectItem>
            {(feedsData ?? []).map((feed) => (
              <SelectItem key={feed.id} value={feed.id}>
                {feed.display_name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <div style={{ width: 1, height: 20, background: 'var(--border)', margin: '0 4px' }} />
        <Input
          placeholder="Search catalog..."
          value={search}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
            setSearch(e.target.value);
            setPage(0);
          }}
          className="h-7 w-52 text-sm"
          style={{ borderColor: 'var(--border)', background: 'var(--bg-card)' }}
        />
        <Select
          value={dateRange}
          onValueChange={(v: string) => {
            setDateRange(v);
            setPage(0);
          }}
        >
          <SelectTrigger
            className="h-7 w-32 text-sm"
            style={{ borderColor: 'var(--border)', background: 'var(--bg-card)' }}
          >
            <SelectValue placeholder="All time" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="7d">Last 7 days</SelectItem>
            <SelectItem value="30d">Last 30 days</SelectItem>
            <SelectItem value="90d">Last 90 days</SelectItem>
            <SelectItem value="all">All time</SelectItem>
          </SelectContent>
        </Select>
        <Select
          value={entryType}
          onValueChange={(v: string) => {
            setEntryType(v);
            setPage(0);
          }}
        >
          <SelectTrigger
            className="h-7 w-28 text-sm"
            style={{ borderColor: 'var(--border)', background: 'var(--bg-card)' }}
          >
            <SelectValue placeholder="Type" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All</SelectItem>
            <SelectItem value="cve">CVEs</SelectItem>
            <SelectItem value="patch">Patches</SelectItem>
          </SelectContent>
        </Select>
        <div style={{ marginLeft: 'auto', display: 'flex', alignItems: 'center', gap: 8 }}>
          <Button variant="outline" size="sm" onClick={handleExport}>
            Export
          </Button>
          <Button
            variant="outline"
            size="sm"
            disabled={importing}
            onClick={() => importInputRef.current?.click()}
          >
            {importing ? 'Importing…' : 'Import CSV'}
          </Button>
          <Button size="sm" onClick={() => setFormOpen(true)}>
            + Add Entry
          </Button>
        </div>
      </div>

      {/* Table */}
      {isLoading ? (
        <div style={{ borderRadius: 8, border: '1px solid var(--border)', overflow: 'hidden' }}>
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13 }}>
            <thead>
              <tr>
                {[
                  '',
                  'Name',
                  'Vendor',
                  'OS',
                  'Version',
                  'Severity',
                  'CVEs',
                  'Synced PMs',
                  'Source',
                  'Published',
                  'Updated',
                  '',
                ].map((h, i) => (
                  <th
                    key={i}
                    style={{
                      height: 40,
                      padding: '0 12px',
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
                ))}
              </tr>
            </thead>
            <tbody>
              <SkeletonRows cols={12} rows={8} />
            </tbody>
          </table>
        </div>
      ) : isError ? (
        <ErrorState
          title="Failed to load catalog entries"
          message={error instanceof Error ? error.message : 'An unknown error occurred'}
          onRetry={() => void refetch()}
        />
      ) : (
        <>
          <div
            className="rounded-lg overflow-x-auto"
            style={{ border: '1px solid var(--border)', background: 'var(--bg-canvas)' }}
          >
            <table className="w-full caption-bottom text-sm">
              <thead>
                {table.getHeaderGroups().map((hg) => (
                  <tr key={hg.id} style={{ borderBottom: '1px solid var(--border)' }}>
                    {hg.headers.map((h) => (
                      <th
                        key={h.id}
                        className="h-10 px-3 text-left align-middle uppercase sticky top-0 z-10"
                        style={{
                          fontFamily: 'var(--font-mono)',
                          fontSize: 11,
                          fontWeight: 600,
                          letterSpacing: '0.05em',
                          color: 'var(--text-muted)',
                          background: 'var(--bg-inset)',
                        }}
                      >
                        {h.isPlaceholder
                          ? null
                          : flexRender(h.column.columnDef.header, h.getContext())}
                      </th>
                    ))}
                  </tr>
                ))}
              </thead>
              <tbody>
                {table.getRowModel().rows.length === 0 ? (
                  <tr>
                    <td colSpan={columns.length} className="h-24 text-center text-muted-foreground">
                      No catalog entries found.
                    </td>
                  </tr>
                ) : (
                  table.getRowModel().rows.map((row) => {
                    const isExpanded = expandedRows.has(row.original.id);
                    return [
                      <tr
                        key={row.id}
                        className="transition-colors"
                        style={{
                          borderBottom: '1px solid var(--border-faint)',
                          background: undefined,
                        }}
                        onMouseEnter={(e) =>
                          (e.currentTarget.style.background = 'var(--bg-card-hover)')
                        }
                        onMouseLeave={(e) => (e.currentTarget.style.background = 'transparent')}
                      >
                        {row.getVisibleCells().map((cell) => (
                          <td key={cell.id} className="px-3 py-2.5 align-middle">
                            {flexRender(cell.column.columnDef.cell, cell.getContext())}
                          </td>
                        ))}
                      </tr>,
                      isExpanded && (
                        <tr
                          key={`${row.id}-expand`}
                          style={{
                            borderBottom: '1px solid var(--border-faint)',
                            background: 'var(--bg-canvas)',
                          }}
                        >
                          <td colSpan={columns.length} className="px-4 py-3">
                            <div className="flex flex-col gap-2">
                              <p className="text-sm text-muted-foreground">
                                {row.original.description || 'No description available.'}
                              </p>
                              <div className="flex items-center gap-3">
                                <Badge variant="secondary" className="text-xs">
                                  {row.original.cve_count} CVE
                                  {row.original.cve_count !== 1 ? 's' : ''}
                                </Badge>
                                <div className="flex gap-2">
                                  <Button
                                    size="sm"
                                    variant="outline"
                                    className="h-7 text-xs"
                                    onClick={() =>
                                      toast.info(
                                        'Patch Managers will pull this entry on their next sync cycle.',
                                        { description: `Entry: ${row.original.name}` },
                                      )
                                    }
                                  >
                                    Push to PMs
                                  </Button>
                                  <Button
                                    size="sm"
                                    variant="outline"
                                    className="h-7 text-xs"
                                    onClick={() => void navigate(`/catalog/${row.original.id}`)}
                                  >
                                    Edit
                                  </Button>
                                  <Button
                                    size="sm"
                                    variant="destructive"
                                    className="h-7 text-xs"
                                    onClick={() => setDeletingEntry(row.original)}
                                  >
                                    Remove
                                  </Button>
                                </div>
                              </div>
                            </div>
                          </td>
                        </tr>
                      ),
                    ];
                  })
                )}
              </tbody>
            </table>
          </div>

          {/* Pagination */}
          <div className="flex items-center justify-between">
            <p className="text-sm text-muted-foreground">
              {data && data.total > 0
                ? `Showing ${startItem}–${endItem} of ${data.total.toLocaleString()} entries`
                : 'No entries'}
            </p>
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setPage((p) => Math.max(0, p - 1))}
                disabled={page === 0}
              >
                ← Prev
              </Button>
              {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
                const pageNum = Math.max(0, Math.min(page - 2, totalPages - 5)) + i;
                return (
                  <Button
                    key={pageNum}
                    variant={pageNum === page ? 'default' : 'outline'}
                    size="sm"
                    className="h-8 w-8 p-0"
                    onClick={() => setPage(pageNum)}
                  >
                    {pageNum + 1}
                  </Button>
                );
              })}
              <Button
                variant="outline"
                size="sm"
                onClick={() => setPage((p) => p + 1)}
                disabled={page + 1 >= totalPages}
              >
                Next →
              </Button>
            </div>
          </div>
        </>
      )}

      <CatalogForm
        open={formOpen}
        onSuccess={() => setFormOpen(false)}
        onCancel={() => setFormOpen(false)}
      />

      {/* Delete Confirmation Dialog */}
      <Dialog
        open={!!deletingEntry}
        onOpenChange={(open) => {
          if (!open) setDeletingEntry(null);
        }}
      >
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>Remove Catalog Entry</DialogTitle>
          </DialogHeader>
          <p style={{ fontSize: 13, color: 'var(--text-secondary)', lineHeight: 1.6 }}>
            Are you sure you want to remove{' '}
            <span style={{ fontWeight: 600, color: 'var(--text-emphasis)' }}>
              {deletingEntry?.name}
            </span>
            ? This will delete it from the hub catalog. Patch Managers that already synced this
            entry will not be affected.
          </p>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeletingEntry(null)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              disabled={deleteMutation.isPending}
              onClick={() => {
                if (!deletingEntry) return;
                deleteMutation.mutate(deletingEntry.id, {
                  onSuccess: () => {
                    toast.success('Catalog entry removed');
                    setDeletingEntry(null);
                  },
                  onError: (err) => {
                    toast.error(err instanceof Error ? err.message : 'Failed to remove entry');
                    setDeletingEntry(null);
                  },
                });
              }}
            >
              {deleteMutation.isPending ? 'Removing...' : 'Remove'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
