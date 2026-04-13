import { useState, useMemo, useRef } from 'react';
import { Link } from 'react-router';
import {
  getCoreRowModel,
  getExpandedRowModel,
  useReactTable,
  createColumnHelper,
  type ExpandedState,
} from '@tanstack/react-table';
import {
  Button,
  EmptyState,
  ErrorState,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@patchiq/ui';
import { Copy, CheckCircle } from 'lucide-react';
import { toast } from 'sonner';
import {
  useLicenses,
  useRevokeLicense,
  useAssignLicense,
  useCreateLicense,
} from '../../api/hooks/useLicenses';
import { useClients } from '../../api/hooks/useClients';
import type { License } from '../../types/license';
import { LicenseForm } from './LicenseForm';
import { computeStatus } from '../../lib/licenseUtils';
import type { ComputedStatus } from '../../lib/licenseUtils';
import { tierBadgeStyle, getTierFeatures } from '../../lib/tierUtils';
import { FilterBar, FilterPill, FilterSeparator, FilterSearch } from '../../components/FilterBar';
import { DataTable } from '../../components/data-table/DataTable';
import { DataTablePagination } from '../../components/data-table/DataTablePagination';

const PAGE_SIZE = 20;

// ─── Helpers ──────────────────────────────────────────────────────────────────

function statusBadgeStyle(status: ComputedStatus): React.CSSProperties {
  const base: React.CSSProperties = {
    background: 'var(--bg-card-hover)',
    borderColor: 'var(--border-strong, var(--border))',
  };
  switch (status) {
    case 'active':
      return { ...base, color: 'var(--signal-healthy)' };
    case 'expiring':
      return { ...base, color: 'var(--signal-warning)' };
    case 'expired':
    case 'revoked':
      return { ...base, color: 'var(--signal-critical)' };
  }
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString();
}

function maskKey(key: string): string {
  if (!key) return '—';
  if (key.length <= 4) return '****';
  return `****-****-${key.slice(-4)}`;
}

// ─── Stat Card ────────────────────────────────────────────────────────────────

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
        background: active
          ? 'color-mix(in srgb, var(--text-emphasis) 3%, transparent)'
          : 'var(--bg-card)',
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

// ─── Skeleton Rows ────────────────────────────────────────────────────────────

function SkeletonRows({ cols, rows = 8 }: { cols: number; rows?: number }) {
  return (
    <>
      {Array.from({ length: rows }).map((_, i) => (
        <tr key={i}>
          {Array.from({ length: cols }).map((__, j) => (
            <td key={j} style={{ padding: '10px 12px' }}>
              <div
                style={{
                  height: 14,
                  borderRadius: 4,
                  background: 'var(--bg-inset)',
                  width: j === 0 ? '60%' : j === 1 ? '80%' : '50%',
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

interface ExpandedRowContentProps {
  license: License;
  revokeMutation: ReturnType<typeof useRevokeLicense>;
  assigningId: string | null;
  setAssigningId: (id: string | null) => void;
  selectedClientId: string;
  setSelectedClientId: (id: string) => void;
  assignMutation: ReturnType<typeof useAssignLicense>;
  clientsData: { clients: { id: string; hostname: string; status: string }[] } | undefined;
}

function ExpandedRowContent({
  license,
  revokeMutation,
  assigningId,
  setAssigningId,
  selectedClientId,
  setSelectedClientId,
  assignMutation,
  clientsData,
}: ExpandedRowContentProps) {
  const [copied, setCopied] = useState(false);
  const features = getTierFeatures(license.tier);
  const status = computeStatus(license);

  const handleCopy = () => {
    void navigator.clipboard.writeText(license.license_key);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const sectionLabel: React.CSSProperties = {
    fontFamily: 'var(--font-mono)',
    fontSize: 10,
    fontWeight: 600,
    textTransform: 'uppercase',
    letterSpacing: '0.06em',
    color: 'var(--text-muted)',
    marginBottom: 8,
  };

  return (
    <div
      style={{
        padding: '16px 48px 16px 20px',
        display: 'grid',
        gridTemplateColumns: '1fr 1fr',
        gap: 24,
        borderLeft: '2px solid var(--accent)',
      }}
    >
      {/* Left: key + features */}
      <div>
        <div style={sectionLabel}>License Key</div>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 8,
            background: 'var(--bg-canvas)',
            border: '1px solid var(--border)',
            borderRadius: 6,
            padding: '8px 12px',
            marginBottom: 16,
          }}
        >
          <span
            style={{
              flex: 1,
              fontFamily: 'var(--font-mono)',
              fontSize: 12,
              color: 'var(--text-primary)',
            }}
          >
            {maskKey(license.license_key)}
          </span>
          <button
            type="button"
            onClick={handleCopy}
            style={{
              background: 'none',
              border: 'none',
              cursor: 'pointer',
              padding: 2,
              color: 'var(--text-muted)',
            }}
            title="Copy license key"
          >
            {copied ? (
              <CheckCircle style={{ width: 14, height: 14, color: 'var(--signal-healthy)' }} />
            ) : (
              <Copy style={{ width: 14, height: 14 }} />
            )}
          </button>
        </div>

        <div style={sectionLabel}>Features Enabled</div>
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
          {features.map((f) => (
            <span
              key={f}
              style={{
                display: 'inline-flex',
                alignItems: 'center',
                gap: 4,
                padding: '2px 8px',
                borderRadius: 4,
                fontSize: 11,
                fontWeight: 500,
                border: '1px solid',
                background: 'color-mix(in srgb, var(--accent) 15%, transparent)',
                color: 'var(--accent)',
                borderColor: 'color-mix(in srgb, var(--accent) 30%, transparent)',
              }}
            >
              <CheckCircle style={{ width: 10, height: 10 }} />
              {f}
            </span>
          ))}
        </div>

        <div style={{ marginTop: 16 }}>
          <div style={sectionLabel}>Usage Trend</div>
          <div
            style={{
              border: '1px dashed var(--border)',
              borderRadius: 8,
              padding: '12px 16px',
              background: 'var(--bg-card)',
            }}
          >
            <span
              style={{ fontSize: 12, color: 'var(--text-muted)', fontFamily: 'var(--font-sans)' }}
            >
              No usage history available
            </span>
          </div>
        </div>
      </div>

      {/* Right: actions */}
      <div>
        <div style={sectionLabel}>Actions</div>
        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
          {status !== 'revoked' && (
            <>
              <Button
                size="sm"
                disabled
                title="License renewal will be available in a future release"
                style={{ background: 'var(--signal-healthy)', color: 'var(--text-emphasis)' }}
              >
                Renew
              </Button>
              {assigningId === license.id ? (
                <div style={{ display: 'flex', gap: 4, alignItems: 'center' }}>
                  <Select
                    value={selectedClientId}
                    onValueChange={(v: string) => setSelectedClientId(v)}
                  >
                    <SelectTrigger style={{ width: 144, height: 32 }}>
                      <SelectValue placeholder="Select client..." />
                    </SelectTrigger>
                    <SelectContent>
                      {(clientsData?.clients ?? [])
                        .filter((c) => c.status === 'approved')
                        .map((c) => (
                          <SelectItem key={c.id} value={c.id}>
                            {c.hostname}
                          </SelectItem>
                        ))}
                    </SelectContent>
                  </Select>
                  <Button
                    size="sm"
                    onClick={() => {
                      if (selectedClientId) {
                        assignMutation.mutate(
                          { id: license.id, clientId: selectedClientId },
                          {
                            onSuccess: () => toast.success('License assigned'),
                            onError: (err) =>
                              toast.error(
                                err instanceof Error ? err.message : 'Failed to assign license',
                              ),
                          },
                        );
                        setAssigningId(null);
                        setSelectedClientId('');
                      }
                    }}
                    disabled={!selectedClientId || assignMutation.isPending}
                  >
                    OK
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => {
                      setAssigningId(null);
                      setSelectedClientId('');
                    }}
                  >
                    ✕
                  </Button>
                </div>
              ) : (
                <Button variant="outline" size="sm" onClick={() => setAssigningId(license.id)}>
                  Assign
                </Button>
              )}
              <Button
                variant="destructive"
                size="sm"
                onClick={() =>
                  revokeMutation.mutate(license.id, {
                    onSuccess: () => toast.success('License revoked'),
                    onError: (err) =>
                      toast.error(err instanceof Error ? err.message : 'Failed to revoke license'),
                  })
                }
                disabled={revokeMutation.isPending}
              >
                Revoke
              </Button>
            </>
          )}
        </div>
      </div>
    </div>
  );
}

// ─── Column helper ────────────────────────────────────────────────────────────

const columnHelper = createColumnHelper<License>();

// ─── Main Page ────────────────────────────────────────────────────────────────

type StatusFilter = '' | 'active' | 'expiring' | 'expired';
type TierFilter = '' | 'community' | 'professional' | 'enterprise' | 'msp';

export const LicensesPage = () => {
  const [page, setPage] = useState(0);
  const [tierFilter, setTierFilter] = useState<TierFilter>('');
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('');
  const [search, setSearch] = useState('');
  const [formOpen, setFormOpen] = useState(false);
  const [expanded, setExpanded] = useState<ExpandedState>({});
  const [assigningId, setAssigningId] = useState<string | null>(null);
  const [selectedClientId, setSelectedClientId] = useState('');
  const [importing, setImporting] = useState(false);
  const importInputRef = useRef<HTMLInputElement>(null);

  const { data, isLoading, isError, error, refetch } = useLicenses({
    limit: PAGE_SIZE,
    offset: page * PAGE_SIZE,
    tier: tierFilter || undefined,
    status: statusFilter || undefined,
  });

  const revokeMutation = useRevokeLicense();
  const assignMutation = useAssignLicense();
  const createMutation = useCreateLicense();
  const { data: clientsData } = useClients({ limit: 100 });

  const handleImportFile = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
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
      const headers = lines[0].split(',').map((h) => h.trim().toLowerCase().replace(/"/g, ''));
      const nameIdx = headers.indexOf('customer_name');
      const tierIdx = headers.indexOf('tier');
      const maxIdx = headers.indexOf('max_endpoints');
      const expiresIdx = headers.indexOf('expires_at');
      if (nameIdx === -1 || tierIdx === -1 || maxIdx === -1 || expiresIdx === -1) {
        toast.error('CSV must have columns: customer_name, tier, max_endpoints, expires_at');
        return;
      }
      const emailIdx = headers.indexOf('customer_email');
      const notesIdx = headers.indexOf('notes');
      let success = 0,
        failed = 0;
      for (let i = 1; i < lines.length; i++) {
        const cols = lines[i].split(',').map((c) => c.trim().replace(/^"|"$/g, ''));
        try {
          await createMutation.mutateAsync({
            customer_name: cols[nameIdx],
            tier: cols[tierIdx],
            max_endpoints: parseInt(cols[maxIdx], 10) || 100,
            expires_at: cols[expiresIdx],
            customer_email: emailIdx !== -1 ? cols[emailIdx] || undefined : undefined,
            notes: notesIdx !== -1 ? cols[notesIdx] || undefined : undefined,
          });
          success++;
        } catch {
          failed++;
        }
      }
      if (failed === 0) toast.success(`Imported ${success} licenses`);
      else toast.warning(`Imported ${success} licenses, ${failed} failed`);
      void refetch();
    } catch (err) {
      toast.error(`Import failed: ${err instanceof Error ? err.message : 'unknown error'}`);
    } finally {
      setImporting(false);
      e.target.value = '';
    }
  };

  const stats = useMemo(() => {
    const licenses = data?.licenses ?? [];
    const total = data?.total ?? 0;
    const now = Date.now();
    const thirtyDays = 30 * 24 * 60 * 60 * 1000;

    const active = licenses.filter(
      (l) => !l.revoked_at && new Date(l.expires_at).getTime() > now + thirtyDays,
    ).length;
    const expiring = licenses.filter((l) => {
      if (l.revoked_at) return false;
      const exp = new Date(l.expires_at).getTime();
      return exp > now && exp < now + thirtyDays;
    }).length;
    const expired = licenses.filter(
      (l) => l.revoked_at || new Date(l.expires_at).getTime() <= now,
    ).length;

    return { total, active, expiring, expired };
  }, [data]);

  const filteredLicenses = useMemo(() => {
    let licenses = data?.licenses ?? [];
    if (search) {
      const q = search.toLowerCase();
      licenses = licenses.filter(
        (l) =>
          l.customer_name?.toLowerCase().includes(q) ||
          l.license_key?.toLowerCase().includes(q) ||
          l.client_hostname?.toLowerCase().includes(q) ||
          l.tier.toLowerCase().includes(q),
      );
    }
    return licenses;
  }, [data, search]);

  const totalPages = data ? Math.ceil(data.total / PAGE_SIZE) : 0;

  const columns = useMemo(
    () => [
      columnHelper.display({
        id: 'expand',
        header: '',
        cell: (info) => {
          const isExp = info.row.getIsExpanded();
          return (
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation();
                info.row.toggleExpanded();
              }}
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                background: 'none',
                border: 'none',
                cursor: 'pointer',
                padding: 2,
                color: 'var(--text-muted)',
              }}
            >
              {isExp ? (
                <CheckCircle style={{ width: 14, height: 14 }} />
              ) : (
                <Copy style={{ width: 14, height: 14 }} />
              )}
            </button>
          );
        },
      }),
      columnHelper.accessor('license_key', {
        header: 'License Key',
        cell: (info) => (
          <Link
            to={`/licenses/${info.row.original.id}`}
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 12,
              color: 'var(--accent)',
              textDecoration: 'none',
            }}
            onClick={(e) => e.stopPropagation()}
          >
            {maskKey(info.getValue())}
          </Link>
        ),
      }),
      columnHelper.accessor('customer_name', {
        header: 'Customer',
        cell: (info) => (
          <span style={{ fontSize: 13, color: 'var(--text-primary)', fontWeight: 500 }}>
            {info.getValue() ?? '—'}
          </span>
        ),
      }),
      columnHelper.accessor('tier', {
        header: 'Tier',
        cell: (info) => {
          const tier = info.getValue();
          return (
            <span
              style={{
                display: 'inline-flex',
                alignItems: 'center',
                padding: '2px 8px',
                borderRadius: 4,
                border: '1px solid',
                fontSize: 11,
                fontWeight: 500,
                textTransform: 'capitalize',
                ...tierBadgeStyle(tier),
              }}
            >
              {tier}
            </span>
          );
        },
      }),
      columnHelper.display({
        id: 'status',
        header: 'Status',
        cell: (info) => {
          const status = computeStatus(info.row.original);
          return (
            <span
              style={{
                display: 'inline-flex',
                alignItems: 'center',
                padding: '2px 8px',
                borderRadius: 4,
                border: '1px solid',
                fontSize: 11,
                fontWeight: 500,
                textTransform: 'capitalize',
                ...statusBadgeStyle(status),
              }}
            >
              {status}
            </span>
          );
        },
      }),
      columnHelper.display({
        id: 'client',
        header: 'Client',
        cell: (info) => {
          const license = info.row.original;
          if (!license.client_hostname) {
            return <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>—</span>;
          }
          if (license.client_id) {
            return (
              <Link
                to={`/clients/${license.client_id}`}
                style={{ fontSize: 13, color: 'var(--accent)', textDecoration: 'none' }}
                onClick={(e) => e.stopPropagation()}
              >
                {license.client_hostname}
              </Link>
            );
          }
          return <span style={{ fontSize: 13 }}>{license.client_hostname}</span>;
        },
      }),
      columnHelper.display({
        id: 'endpoints',
        header: 'Endpoints',
        cell: (info) => {
          const license = info.row.original;
          if (license.client_endpoint_count == null) {
            return <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>—</span>;
          }
          const pct = Math.round((license.client_endpoint_count / license.max_endpoints) * 100);
          const barColor =
            pct > 90
              ? 'var(--signal-critical)'
              : pct > 70
                ? 'var(--signal-warning)'
                : 'var(--signal-healthy)';
          return (
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <div
                style={{
                  width: 60,
                  height: 6,
                  borderRadius: 9999,
                  overflow: 'hidden',
                  background: 'var(--bg-card-hover)',
                }}
              >
                <div
                  style={{
                    height: 6,
                    borderRadius: 9999,
                    width: `${Math.min(pct, 100)}%`,
                    background: barColor,
                  }}
                />
              </div>
              <span
                style={{
                  fontFamily: 'var(--font-mono)',
                  fontSize: 11,
                  color: 'var(--text-secondary)',
                }}
              >
                {license.client_endpoint_count}/{license.max_endpoints}
              </span>
            </div>
          );
        },
      }),
      columnHelper.accessor('expires_at', {
        header: 'Expires',
        cell: (info) => {
          const dateStr = info.getValue();
          const days = Math.ceil(
            (new Date(dateStr).getTime() - Date.now()) / (1000 * 60 * 60 * 24),
          );
          return (
            <div>
              <div style={{ fontSize: 12, color: 'var(--text-secondary)' }}>
                {formatDate(dateStr)}
              </div>
              {days < 0 ? (
                <div style={{ fontSize: 11, fontWeight: 600, color: 'var(--signal-critical)' }}>
                  EXPIRED
                </div>
              ) : days <= 30 ? (
                <div style={{ fontSize: 11, color: 'var(--signal-warning)' }}>{days} days</div>
              ) : (
                <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>{days} days</div>
              )}
            </div>
          );
        },
      }),
      columnHelper.display({
        id: 'actions',
        header: 'Actions',
        cell: (info) => {
          const license = info.row.original;
          const status = computeStatus(license);
          return (
            <div style={{ display: 'flex', gap: 6 }}>
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation();
                  void navigator.clipboard.writeText(license.license_key);
                  toast.success('Key copied');
                }}
                style={{
                  padding: '4px 8px',
                  fontSize: 11,
                  borderRadius: 4,
                  border: '1px solid var(--border)',
                  background: 'transparent',
                  cursor: 'pointer',
                  color: 'var(--text-muted)',
                  display: 'flex',
                  alignItems: 'center',
                  gap: 4,
                }}
                title="Copy key"
              >
                <Copy style={{ width: 11, height: 11 }} />
              </button>
              {status !== 'revoked' && (
                <button
                  type="button"
                  onClick={(e) => {
                    e.stopPropagation();
                    revokeMutation.mutate(license.id, {
                      onSuccess: () => toast.success('License revoked'),
                      onError: (err) =>
                        toast.error(err instanceof Error ? err.message : 'Failed to revoke'),
                    });
                  }}
                  disabled={revokeMutation.isPending}
                  style={{
                    padding: '4px 8px',
                    fontSize: 11,
                    borderRadius: 4,
                    border: '1px solid color-mix(in srgb, var(--signal-critical) 40%, transparent)',
                    background: 'transparent',
                    cursor: revokeMutation.isPending ? 'not-allowed' : 'pointer',
                    opacity: revokeMutation.isPending ? 0.5 : 1,
                    color: 'var(--signal-critical)',
                  }}
                >
                  Revoke
                </button>
              )}
            </div>
          );
        },
      }),
    ],
    [revokeMutation],
  );

  const table = useReactTable({
    data: filteredLicenses,
    columns,
    state: { expanded },
    onExpandedChange: setExpanded,
    getCoreRowModel: getCoreRowModel(),
    getExpandedRowModel: getExpandedRowModel(),
  });

  return (
    <div style={{ padding: 24 }}>
      {/* Page Header */}

      {/* Stat Cards */}
      <div style={{ display: 'flex', gap: 12, marginBottom: 16 }}>
        <StatCard
          label="Total Licenses"
          value={stats.total}
          active={statusFilter === ''}
          onClick={() => {
            setStatusFilter('');
            setPage(0);
          }}
        />
        <StatCard
          label="Active"
          value={stats.active}
          valueColor="var(--signal-healthy)"
          active={statusFilter === 'active'}
          onClick={() => {
            setStatusFilter('active');
            setPage(0);
          }}
        />
        <StatCard
          label="Expiring Soon"
          value={stats.expiring}
          valueColor="var(--signal-warning)"
          active={statusFilter === 'expiring'}
          onClick={() => {
            setStatusFilter('expiring');
            setPage(0);
          }}
        />
        <StatCard
          label="Expired / Revoked"
          value={stats.expired}
          valueColor="var(--signal-critical)"
          active={statusFilter === 'expired'}
          onClick={() => {
            setStatusFilter('expired');
            setPage(0);
          }}
        />
      </div>

      {/* Filter Bar */}
      <FilterBar>
        <Select
          value={statusFilter || 'all'}
          onValueChange={(v: string) => {
            setStatusFilter((v === 'all' ? '' : v) as StatusFilter);
            setPage(0);
          }}
        >
          <SelectTrigger
            className="h-7 w-32 text-sm"
            style={{ borderColor: 'var(--border)', background: 'var(--bg-card)' }}
          >
            <SelectValue placeholder="All" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">
              All{' '}
              <span
                style={{ color: 'var(--text-muted)', fontFamily: 'var(--font-mono)', fontSize: 10 }}
              >
                {stats.total}
              </span>
            </SelectItem>
            <SelectItem value="active">
              Active{' '}
              <span
                style={{ color: 'var(--text-muted)', fontFamily: 'var(--font-mono)', fontSize: 10 }}
              >
                {stats.active}
              </span>
            </SelectItem>
            <SelectItem value="expiring">
              Expiring{' '}
              <span
                style={{ color: 'var(--text-muted)', fontFamily: 'var(--font-mono)', fontSize: 10 }}
              >
                {stats.expiring}
              </span>
            </SelectItem>
            <SelectItem value="expired">
              Expired{' '}
              <span
                style={{ color: 'var(--text-muted)', fontFamily: 'var(--font-mono)', fontSize: 10 }}
              >
                {stats.expired}
              </span>
            </SelectItem>
          </SelectContent>
        </Select>
        <FilterPill
          label="Standard"
          active={tierFilter === 'professional'}
          onClick={() => {
            setTierFilter(tierFilter === 'professional' ? '' : 'professional');
            setPage(0);
          }}
        />
        <FilterPill
          label="Enterprise"
          active={tierFilter === 'enterprise'}
          onClick={() => {
            setTierFilter(tierFilter === 'enterprise' ? '' : 'enterprise');
            setPage(0);
          }}
        />
        <FilterSeparator />
        <FilterSearch
          value={search}
          onChange={(v) => {
            setSearch(v);
            setPage(0);
          }}
          placeholder="Search licenses..."
        />
        <div style={{ marginLeft: 'auto', display: 'flex', alignItems: 'center', gap: 8 }}>
          <input
            ref={importInputRef}
            type="file"
            accept=".csv"
            style={{ display: 'none' }}
            onChange={handleImportFile}
          />
          <Button
            variant="outline"
            size="sm"
            disabled={importing}
            onClick={() => importInputRef.current?.click()}
          >
            {importing ? 'Importing…' : 'Import CSV'}
          </Button>
          <Button size="sm" onClick={() => setFormOpen(true)}>
            + Generate License
          </Button>
        </div>
      </FilterBar>

      {/* Table */}
      {isLoading ? (
        <div style={{ borderRadius: 8, border: '1px solid var(--border)', overflow: 'hidden' }}>
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13 }}>
            <thead>
              <tr>
                {[
                  '',
                  'License Key',
                  'Customer',
                  'Tier',
                  'Status',
                  'Client',
                  'Endpoints',
                  'Expires',
                  'Actions',
                ].map((h) => (
                  <th
                    key={h}
                    style={{
                      height: 40,
                      padding: '0 16px',
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
              <SkeletonRows cols={9} rows={6} />
            </tbody>
          </table>
        </div>
      ) : isError ? (
        <ErrorState
          title="Failed to load licenses"
          message={error instanceof Error ? error.message : 'An unknown error occurred'}
        />
      ) : filteredLicenses.length === 0 ? (
        <EmptyState
          title="No licenses found"
          description={
            statusFilter || tierFilter || search
              ? 'Try adjusting your filters'
              : 'No licenses have been generated yet'
          }
          action={{ label: '+ Generate License', onClick: () => setFormOpen(true) }}
        />
      ) : (
        <>
          <DataTable
            table={table}
            isRowFailed={(license) =>
              license.revoked_at !== null || new Date(license.expires_at) < new Date()
            }
            onRowClick={(license) => {
              const row = table.getRowModel().rows.find((r) => r.original === license);
              row?.toggleExpanded();
            }}
            renderExpandedRow={(license) => (
              <ExpandedRowContent
                license={license}
                revokeMutation={revokeMutation}
                assigningId={assigningId}
                setAssigningId={setAssigningId}
                selectedClientId={selectedClientId}
                setSelectedClientId={setSelectedClientId}
                assignMutation={assignMutation}
                clientsData={clientsData}
              />
            )}
          />
          <DataTablePagination
            hasPrev={page > 0}
            hasNext={page + 1 < totalPages}
            onPrev={() => setPage((p) => Math.max(0, p - 1))}
            onNext={() => setPage((p) => p + 1)}
          />
        </>
      )}

      <LicenseForm
        open={formOpen}
        onSuccess={() => setFormOpen(false)}
        onCancel={() => setFormOpen(false)}
      />
    </div>
  );
};
