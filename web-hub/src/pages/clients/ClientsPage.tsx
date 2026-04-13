import React, { useState, useMemo } from 'react';
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
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
  TooltipProvider,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Skeleton,
} from '@patchiq/ui';
import { ChevronDown, ChevronRight, Download, Plus } from 'lucide-react';
import { toast } from 'sonner';
import {
  useClients,
  useApproveClient,
  useDeclineClient,
  useSuspendClient,
  useDeleteClient,
  useClientSyncHistory,
  useClientEndpointTrend,
} from '../../api/hooks/useClients';
import { useLicenses } from '../../api/hooks/useLicenses';
import type { Client } from '../../types/client';
import type { License } from '../../types/license';
import { tierBadgeStyle } from '../../lib/tierUtils';
import { FilterBar, FilterSearch } from '../../components/FilterBar';
import { DataTable } from '../../components/data-table/DataTable';
import { DataTablePagination } from '../../components/data-table/DataTablePagination';

const PAGE_SIZE = 20;

// ─── Helpers ──────────────────────────────────────────────────────────────────

function computeHealthScore(lastSyncAt: string | null, status?: string): number {
  if (!lastSyncAt) {
    return status === 'approved' ? 65 : 10;
  }
  const diffMin = (Date.now() - new Date(lastSyncAt).getTime()) / 60000;
  if (diffMin < 30) return 100;
  if (diffMin < 60) return 90;
  if (diffMin < 360) return 75;
  if (diffMin < 1440) return 60;
  if (diffMin < 10080) return 30;
  return 10;
}

function formatRelativeTime(dateStr: string | null): string {
  if (!dateStr) return 'Never';
  const diffMs = Date.now() - new Date(dateStr).getTime();
  const diffSeconds = Math.floor(diffMs / 1000);
  if (diffSeconds < 60) return `${diffSeconds}s ago`;
  const diffMinutes = Math.floor(diffSeconds / 60);
  if (diffMinutes < 60) return `${diffMinutes}m ago`;
  const diffHours = Math.floor(diffMinutes / 60);
  if (diffHours < 24) return `${diffHours}h ago`;
  return `${Math.floor(diffHours / 24)}d ago`;
}

function formatNextSync(lastSyncAt: string | null, syncIntervalSeconds: number): string {
  if (!lastSyncAt) return '—';
  const nextMs = new Date(lastSyncAt).getTime() + syncIntervalSeconds * 1000;
  const diffMin = Math.round((nextMs - Date.now()) / 60000);
  if (diffMin <= 0) return 'Due now';
  if (diffMin < 60) return `${diffMin}m`;
  return `${Math.floor(diffMin / 60)}h`;
}

function formatSyncInterval(seconds: number): string {
  if (seconds < 3600) return `${Math.floor(seconds / 60)} min`;
  const h = Math.floor(seconds / 3600);
  return `${h} hour${h !== 1 ? 's' : ''}`;
}

function getClientLicense(clientId: string, licenses: License[]): License | undefined {
  return licenses.find((l) => l.client_id === clientId);
}

function getAvatar(hostname: string): { letter: string } {
  return { letter: hostname[0].toUpperCase() };
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

// ─── Status Cell ──────────────────────────────────────────────────────────────

function StatusCell({ client }: { client: Client }) {
  const health = computeHealthScore(client.last_sync_at, client.status);
  const connected = client.status === 'approved' && health >= 60;

  if (client.status === 'approved') {
    if (connected) {
      return (
        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
          <span style={{ position: 'relative', display: 'flex', height: 8, width: 8 }}>
            <span
              style={{
                position: 'absolute',
                display: 'inline-flex',
                height: '100%',
                width: '100%',
                borderRadius: '50%',
                opacity: 0.75,
                background: 'var(--signal-healthy)',
                animation: 'ping 1s cubic-bezier(0,0,0.2,1) infinite',
              }}
            />
            <span
              style={{
                position: 'relative',
                display: 'inline-flex',
                borderRadius: '50%',
                height: 8,
                width: 8,
                background: 'var(--signal-healthy)',
              }}
            />
          </span>
          <span style={{ fontSize: 12, fontWeight: 500, color: 'var(--signal-healthy)' }}>
            Connected
          </span>
        </div>
      );
    }
    return (
      <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
        <span
          style={{
            height: 8,
            width: 8,
            borderRadius: '50%',
            background: 'var(--signal-critical)',
            flexShrink: 0,
          }}
        />
        <span style={{ fontSize: 12, fontWeight: 500, color: 'var(--signal-critical)' }}>
          Disconnected
        </span>
      </div>
    );
  }
  if (client.status === 'pending') {
    return (
      <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
        <span
          style={{
            height: 8,
            width: 8,
            borderRadius: '50%',
            background: 'var(--signal-warning)',
            flexShrink: 0,
          }}
        />
        <span style={{ fontSize: 12, fontWeight: 500, color: 'var(--signal-warning)' }}>
          Pending
        </span>
      </div>
    );
  }
  if (client.status === 'suspended') {
    return (
      <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
        <span
          style={{
            height: 8,
            width: 8,
            borderRadius: '50%',
            background: 'var(--text-muted)',
            flexShrink: 0,
          }}
        />
        <span style={{ fontSize: 12, fontWeight: 500, color: 'var(--text-muted)' }}>Suspended</span>
      </div>
    );
  }
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
      <span
        style={{
          height: 8,
          width: 8,
          borderRadius: '50%',
          background: 'var(--text-muted)',
          flexShrink: 0,
        }}
      />
      <span
        style={{
          fontSize: 12,
          fontWeight: 500,
          color: 'var(--text-muted)',
          textTransform: 'capitalize',
        }}
      >
        {client.status}
      </span>
    </div>
  );
}

// ─── Health Cell ──────────────────────────────────────────────────────────────

function HealthCell({ score }: { score: number }) {
  const barColor =
    score >= 80
      ? 'var(--signal-healthy)'
      : score >= 60
        ? 'var(--signal-warning)'
        : 'var(--signal-critical)';
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 8, minWidth: 90 }}>
      <div
        style={{
          flex: 1,
          borderRadius: 9999,
          height: 6,
          background: 'var(--bg-card-hover)',
          overflow: 'hidden',
        }}
      >
        <div style={{ height: 6, borderRadius: 9999, width: `${score}%`, background: barColor }} />
      </div>
      <span
        style={{
          fontSize: 11,
          fontWeight: 600,
          color: barColor,
          fontFamily: 'var(--font-mono)',
          minWidth: 32,
        }}
      >
        {score}%
      </span>
    </div>
  );
}

// ─── Endpoint Sparkline ───────────────────────────────────────────────────────

function EndpointSparkline({ clientId, count }: { clientId: string; count: number }) {
  const { data } = useClientEndpointTrend(clientId, 30);
  const points = data?.points ?? [];

  const svgH = 20;
  const barW = 4;
  const gap = 2;

  if (points.length === 0) {
    const svgW = 7 * (barW + gap) - gap;
    return (
      <svg width={svgW} height={svgH} role="img" aria-label="Endpoint trend">
        {Array.from({ length: 7 }).map((_, i) => (
          <rect
            key={i}
            x={i * (barW + gap)}
            y={svgH - 4}
            width={barW}
            height={4}
            rx={1}
            style={{ fill: 'var(--border)' }}
          />
        ))}
      </svg>
    );
  }

  const totals = points.map((p) => p.total);
  const max = Math.max(...totals, count, 1);
  const svgW = points.length * (barW + gap) - gap;

  return (
    <svg width={svgW} height={svgH} role="img" aria-label="Endpoint trend">
      {points.map((pt, i) => {
        const h = Math.max(2, Math.round((pt.total / max) * svgH));
        return (
          <rect
            key={pt.date}
            x={i * (barW + gap)}
            y={svgH - h}
            width={barW}
            height={h}
            rx={1}
            style={{ fill: 'var(--accent)' }}
          />
        );
      })}
    </svg>
  );
}

// ─── OS Breakdown ─────────────────────────────────────────────────────────────

interface OsSummaryEntry {
  os: string;
  count: number;
  pct: number;
}

const OS_COLORS: Record<string, string> = {
  windows: 'var(--accent)',
  ubuntu: 'var(--signal-warning)',
  rhel: 'var(--signal-critical)',
  debian: 'var(--chart-purple, var(--accent))',
  centos: 'var(--chart-cyan, var(--accent))',
  macos: 'var(--text-muted)',
};

function osColor(os: string): string {
  const key = os.toLowerCase();
  for (const [k, v] of Object.entries(OS_COLORS)) {
    if (key.includes(k)) return v;
  }
  return 'var(--text-muted)';
}

function decodeBase64Json<T>(value: unknown): T | null {
  if (!value || typeof value !== 'string') return null;
  try {
    const json = atob(value);
    const parsed = JSON.parse(json) as unknown;
    return parsed as T;
  } catch {
    return null;
  }
}

function OsBreakdownPie({ client }: { client: Client }) {
  const raw = (client as unknown as { os_summary?: unknown }).os_summary;
  const decoded = decodeBase64Json<OsSummaryEntry[] | Record<string, unknown>>(raw);
  const osSummary: OsSummaryEntry[] | null = Array.isArray(decoded) ? decoded : null;

  if (!osSummary || osSummary.length === 0) {
    return (
      <div>
        <div
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 10,
            fontWeight: 600,
            textTransform: 'uppercase',
            letterSpacing: '0.06em',
            color: 'var(--text-muted)',
            marginBottom: 8,
          }}
        >
          Endpoint OS Breakdown
        </div>
        <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>
          Data available after next sync
        </div>
      </div>
    );
  }

  const r = 28;
  const cx = 36;
  const cy = 36;
  let cumAngle = -Math.PI / 2;
  const paths = osSummary.map((entry) => {
    const startAngle = cumAngle;
    const sweepAngle = (entry.pct / 100) * 2 * Math.PI;
    cumAngle += sweepAngle;
    const x1 = cx + r * Math.cos(startAngle);
    const y1 = cy + r * Math.sin(startAngle);
    const x2 = cx + r * Math.cos(cumAngle);
    const y2 = cy + r * Math.sin(cumAngle);
    const lg = sweepAngle > Math.PI ? 1 : 0;
    return {
      d: `M ${cx} ${cy} L ${x1.toFixed(1)} ${y1.toFixed(1)} A ${r} ${r} 0 ${lg} 1 ${x2.toFixed(1)} ${y2.toFixed(1)} Z`,
      color: osColor(entry.os),
      label: entry.os,
      pct: entry.pct,
    };
  });

  return (
    <div>
      <div
        style={{
          fontFamily: 'var(--font-mono)',
          fontSize: 10,
          fontWeight: 600,
          textTransform: 'uppercase',
          letterSpacing: '0.06em',
          color: 'var(--text-muted)',
          marginBottom: 8,
        }}
      >
        Endpoint OS Breakdown
      </div>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
        <svg width="72" height="72" role="img" aria-label="OS breakdown">
          {paths.map((p) => (
            <path key={p.label} d={p.d} fill={p.color} />
          ))}
        </svg>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
          {paths.map((p) => (
            <div
              key={p.label}
              style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 12 }}
            >
              <span
                style={{ width: 8, height: 8, borderRadius: 2, background: p.color, flexShrink: 0 }}
              />
              <span style={{ color: 'var(--text-primary)' }}>{p.label}</span>
              <span style={{ color: 'var(--text-muted)' }}>{Math.round(p.pct)}%</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

// ─── Confirm Dialog ───────────────────────────────────────────────────────────

interface ConfirmDialogProps {
  open: boolean;
  title: string;
  description: string;
  confirmLabel: string;
  isPending: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

function ConfirmDialog({
  open,
  title,
  description,
  confirmLabel,
  isPending,
  onConfirm,
  onCancel,
}: ConfirmDialogProps) {
  return (
    <Dialog
      open={open}
      onOpenChange={(v) => {
        if (!v) onCancel();
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" size="sm" onClick={onCancel} disabled={isPending}>
            Cancel
          </Button>
          <Button size="sm" variant="destructive" onClick={onConfirm} disabled={isPending}>
            {isPending ? 'Processing…' : confirmLabel}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ─── Expanded Panel ───────────────────────────────────────────────────────────

interface ExpandedPanelProps {
  client: Client;
  license?: License;
  onApprove: () => void;
  onDeclineRequest: () => void;
  onSuspendRequest: () => void;
  onDeleteRequest: () => void;
  onDeleteConfirm: () => void;
  onDeleteCancel: () => void;
  confirmingDelete: boolean;
  approvePending: boolean;
  declinePending: boolean;
  suspendPending: boolean;
  deletePending: boolean;
}

function ExpandedPanel({
  client,
  license,
  onApprove,
  onDeclineRequest,
  onSuspendRequest,
  onDeleteRequest,
  onDeleteConfirm,
  onDeleteCancel,
  confirmingDelete,
  approvePending,
  declinePending,
  suspendPending,
  deletePending,
}: ExpandedPanelProps) {
  const { data: syncData, isLoading: syncLoading } = useClientSyncHistory(client.id, 5);
  const syncItems = syncData?.items ?? [];

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
        gridTemplateColumns: '1fr 1fr 200px',
        gap: 24,
        borderLeft: '2px solid var(--accent)',
      }}
    >
      {/* Sync history */}
      <div>
        <div style={sectionLabel}>Last 5 Sync Events</div>
        {syncLoading ? (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} style={{ height: 28, width: '100%' }} />
            ))}
          </div>
        ) : syncItems.length > 0 ? (
          <div style={{ borderRadius: 6, overflow: 'hidden', border: '1px solid var(--border)' }}>
            <table style={{ width: '100%', fontSize: 12, borderCollapse: 'collapse' }}>
              <thead>
                <tr style={{ background: 'var(--bg-canvas)' }}>
                  {['Timestamp', 'Entries', 'Duration', 'Status'].map((h) => (
                    <th
                      key={h}
                      style={{
                        padding: '6px 12px',
                        textAlign: 'left',
                        fontFamily: 'var(--font-mono)',
                        fontSize: 10,
                        fontWeight: 600,
                        textTransform: 'uppercase',
                        letterSpacing: '0.05em',
                        color: 'var(--text-muted)',
                      }}
                    >
                      {h}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {syncItems.map((item) => (
                  <tr key={item.id} style={{ borderTop: '1px solid var(--border)' }}>
                    <td
                      style={{
                        padding: '6px 12px',
                        fontFamily: 'var(--font-mono)',
                        fontSize: 11,
                        color: 'var(--text-primary)',
                      }}
                    >
                      {new Date(item.synced_at).toLocaleString()}
                    </td>
                    <td style={{ padding: '6px 12px', color: 'var(--text-primary)', fontSize: 12 }}>
                      {item.patches_synced.toLocaleString()}
                    </td>
                    <td style={{ padding: '6px 12px', color: 'var(--text-primary)', fontSize: 12 }}>
                      {item.duration_ms < 1000
                        ? `${item.duration_ms}ms`
                        : `${(item.duration_ms / 1000).toFixed(1)}s`}
                    </td>
                    <td style={{ padding: '6px 12px' }}>
                      {item.status === 'success' ? (
                        <span
                          style={{ fontWeight: 600, color: 'var(--signal-healthy)', fontSize: 12 }}
                        >
                          ✓ OK
                        </span>
                      ) : item.status === 'partial' ? (
                        <span
                          style={{ fontWeight: 600, color: 'var(--signal-warning)', fontSize: 12 }}
                        >
                          ⚠ Partial
                        </span>
                      ) : (
                        <span
                          style={{ fontWeight: 600, color: 'var(--signal-critical)', fontSize: 12 }}
                        >
                          ✕ Error
                        </span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <div
            style={{
              fontSize: 12,
              color: 'var(--text-muted)',
              padding: '12px 16px',
              background: 'var(--bg-canvas)',
              borderRadius: 6,
              textAlign: 'center',
            }}
          >
            No sync history available
          </div>
        )}
      </div>

      {/* OS breakdown */}
      <div>
        <OsBreakdownPie client={client} />

        {/* Assigned license */}
        <div style={{ marginTop: 16 }}>
          <div style={sectionLabel}>Assigned License</div>
          {license ? (
            <div style={{ borderRadius: 8, padding: 12, background: 'var(--bg-canvas)' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6 }}>
                <span
                  style={{
                    padding: '2px 8px',
                    fontSize: 11,
                    fontWeight: 500,
                    borderRadius: 4,
                    border: '1px solid',
                    textTransform: 'capitalize',
                    ...tierBadgeStyle(license.tier),
                  }}
                >
                  {license.tier}
                </span>
                <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>
                  {license.client_endpoint_count ?? client.endpoint_count} / {license.max_endpoints}
                </span>
              </div>
              <div
                style={{
                  width: '100%',
                  borderRadius: 9999,
                  height: 6,
                  background: 'var(--bg-card-hover)',
                  overflow: 'hidden',
                }}
              >
                <div
                  style={{
                    height: 6,
                    borderRadius: 9999,
                    background: 'var(--accent)',
                    width: `${Math.min(100, Math.round(((license.client_endpoint_count ?? client.endpoint_count) / license.max_endpoints) * 100))}%`,
                  }}
                />
              </div>
            </div>
          ) : (
            <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>No license assigned</div>
          )}
        </div>
      </div>

      {/* Actions */}
      <div>
        <div style={sectionLabel}>Actions</div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
          <button
            type="button"
            style={{
              padding: '6px 12px',
              fontSize: 12,
              fontWeight: 500,
              borderRadius: 6,
              border: 'none',
              cursor: 'pointer',
              background: 'var(--accent)',
              color: 'var(--text-emphasis)',
            }}
            onClick={(e) => {
              e.stopPropagation();
              toast.info('Sync initiated — real-time sync will be available in a future release.');
            }}
          >
            Sync Now
          </button>
          <Link
            to={`/clients/${client.id}`}
            style={{
              display: 'inline-block',
              padding: '6px 12px',
              fontSize: 12,
              fontWeight: 500,
              borderRadius: 6,
              border: '1px solid var(--border)',
              color: 'var(--text-primary)',
              textDecoration: 'none',
              textAlign: 'center',
            }}
            onClick={(e) => e.stopPropagation()}
          >
            View Detail
          </Link>
          {client.status === 'pending' && (
            <>
              <button
                type="button"
                style={{
                  padding: '6px 12px',
                  fontSize: 12,
                  fontWeight: 500,
                  borderRadius: 6,
                  border: 'none',
                  cursor: approvePending ? 'not-allowed' : 'pointer',
                  opacity: approvePending ? 0.5 : 1,
                  background: 'var(--signal-healthy)',
                  color: 'var(--text-emphasis)',
                }}
                onClick={(e) => {
                  e.stopPropagation();
                  onApprove();
                }}
                disabled={approvePending}
              >
                Approve
              </button>
              <button
                type="button"
                style={{
                  padding: '6px 12px',
                  fontSize: 12,
                  fontWeight: 500,
                  borderRadius: 6,
                  border: '1px solid color-mix(in srgb, var(--signal-critical) 40%, transparent)',
                  background: 'transparent',
                  cursor: declinePending ? 'not-allowed' : 'pointer',
                  opacity: declinePending ? 0.5 : 1,
                  color: 'var(--signal-critical)',
                }}
                onClick={(e) => {
                  e.stopPropagation();
                  onDeclineRequest();
                }}
                disabled={declinePending}
              >
                Decline
              </button>
            </>
          )}
          {client.status === 'approved' && (
            <button
              type="button"
              style={{
                padding: '6px 12px',
                fontSize: 12,
                fontWeight: 500,
                borderRadius: 6,
                border: '1px solid color-mix(in srgb, var(--signal-critical) 30%, transparent)',
                background: 'transparent',
                cursor: suspendPending ? 'not-allowed' : 'pointer',
                opacity: suspendPending ? 0.5 : 1,
                color: 'var(--signal-critical)',
              }}
              onClick={(e) => {
                e.stopPropagation();
                onSuspendRequest();
              }}
              disabled={suspendPending}
            >
              Revoke Access
            </button>
          )}
          {confirmingDelete ? (
            <div style={{ display: 'flex', gap: 4 }} onClick={(e) => e.stopPropagation()}>
              <button
                type="button"
                style={{
                  flex: 1,
                  padding: '6px 8px',
                  fontSize: 11,
                  fontWeight: 500,
                  borderRadius: 6,
                  border: 'none',
                  cursor: deletePending ? 'not-allowed' : 'pointer',
                  opacity: deletePending ? 0.5 : 1,
                  background: 'var(--signal-critical)',
                  color: 'var(--text-emphasis)',
                }}
                onClick={onDeleteConfirm}
                disabled={deletePending}
              >
                Confirm Delete
              </button>
              <button
                type="button"
                style={{
                  padding: '6px 8px',
                  fontSize: 11,
                  fontWeight: 500,
                  borderRadius: 6,
                  border: '1px solid var(--border)',
                  background: 'transparent',
                  cursor: 'pointer',
                  color: 'var(--text-primary)',
                }}
                onClick={(e) => {
                  e.stopPropagation();
                  onDeleteCancel();
                }}
              >
                Cancel
              </button>
            </div>
          ) : (
            <button
              type="button"
              style={{
                padding: '6px 12px',
                fontSize: 12,
                fontWeight: 500,
                borderRadius: 6,
                border: '1px solid color-mix(in srgb, var(--signal-critical) 40%, transparent)',
                background: 'transparent',
                cursor: 'pointer',
                color: 'var(--signal-critical)',
              }}
              onClick={(e) => {
                e.stopPropagation();
                onDeleteRequest();
              }}
            >
              Delete
            </button>
          )}
        </div>
      </div>
    </div>
  );
}

// ─── Column helper ────────────────────────────────────────────────────────────

const columnHelper = createColumnHelper<Client>();

// ─── Main Page ────────────────────────────────────────────────────────────────

type StatusFilter = '' | 'connected' | 'pending' | 'disconnected';

export const ClientsPage = () => {
  const [page, setPage] = useState(0);
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('');
  const [search, setSearch] = useState('');
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);
  const [declineConfirm, setDeclineConfirm] = useState<string | null>(null);
  const [suspendConfirm, setSuspendConfirm] = useState<string | null>(null);
  const [showAddClientInfo, setShowAddClientInfo] = useState(false);
  const [expanded, setExpanded] = useState<ExpandedState>({});

  // Map UI status filter to API filter
  const apiStatusFilter = useMemo(() => {
    if (statusFilter === 'pending') return 'pending';
    if (statusFilter === 'connected' || statusFilter === 'disconnected') return 'approved';
    return undefined;
  }, [statusFilter]);

  const { data, isLoading, isError, error } = useClients({
    limit: PAGE_SIZE,
    offset: page * PAGE_SIZE,
    status: apiStatusFilter,
  });

  const { data: licensesData } = useLicenses({ limit: 100, offset: 0 });

  const approveMutation = useApproveClient();
  const declineMutation = useDeclineClient();
  const suspendMutation = useSuspendClient();
  const deleteMutation = useDeleteClient();

  function handleExport() {
    const clients = data?.clients ?? [];
    if (clients.length === 0) {
      toast.error('No clients to export');
      return;
    }
    const headers = [
      'hostname',
      'status',
      'os',
      'version',
      'endpoint_count',
      'sync_interval',
      'last_sync_at',
      'created_at',
    ];
    const rows = clients.map((c) => [
      c.hostname,
      c.status,
      c.os ?? '',
      c.version ?? '',
      String(c.endpoint_count),
      String(c.sync_interval),
      c.last_sync_at ?? '',
      c.created_at,
    ]);
    const csv = [headers, ...rows]
      .map((r) => r.map((v) => `"${v.replace(/"/g, '""')}"`).join(','))
      .join('\n');
    const blob = new Blob([csv], { type: 'text/csv' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `clients-${new Date().toISOString().slice(0, 10)}.csv`;
    document.body.appendChild(a);
    a.click();
    URL.revokeObjectURL(url);
    a.remove();
    toast.success(`Exported ${clients.length} clients`);
  }

  // Compute stat counts from current page data
  const stats = useMemo(() => {
    const clients = data?.clients ?? [];
    const total = data?.total ?? 0;
    const connected = clients.filter(
      (c) => c.status === 'approved' && computeHealthScore(c.last_sync_at, c.status) >= 60,
    ).length;
    const pending = clients.filter((c) => c.status === 'pending').length;
    const disconnected = clients.filter(
      (c) => c.status === 'approved' && computeHealthScore(c.last_sync_at, c.status) < 60,
    ).length;
    return { total, connected, pending, disconnected };
  }, [data]);

  const filteredClients = useMemo(() => {
    let clients = data?.clients ?? [];

    // Further filter 'connected' / 'disconnected' client-side since API only has 'approved'
    if (statusFilter === 'connected') {
      clients = clients.filter(
        (c) => c.status === 'approved' && computeHealthScore(c.last_sync_at, c.status) >= 60,
      );
    } else if (statusFilter === 'disconnected') {
      clients = clients.filter(
        (c) => c.status === 'approved' && computeHealthScore(c.last_sync_at, c.status) < 60,
      );
    }

    if (search) {
      const q = search.toLowerCase();
      clients = clients.filter(
        (c) =>
          c.hostname.toLowerCase().includes(q) ||
          (c.os?.toLowerCase().includes(q) ?? false) ||
          (c.version?.toLowerCase().includes(q) ?? false),
      );
    }

    return clients;
  }, [data, search, statusFilter]);

  const totalPages = data ? Math.ceil(data.total / PAGE_SIZE) : 0;

  function handleApprove(clientId: string, hostname: string) {
    approveMutation.mutate(clientId, {
      onSuccess: () => toast.success(`${hostname} approved`),
      onError: (err) =>
        toast.error(
          `Failed to approve ${hostname}: ${err instanceof Error ? err.message : 'Unknown error'}`,
        ),
    });
  }

  function handleDecline(clientId: string, hostname: string) {
    declineMutation.mutate(clientId, {
      onSuccess: () => {
        toast.success(`${hostname} declined`);
        setDeclineConfirm(null);
      },
      onError: (err) => {
        toast.error(
          `Failed to decline ${hostname}: ${err instanceof Error ? err.message : 'Unknown error'}`,
        );
        setDeclineConfirm(null);
      },
    });
  }

  function handleSuspend(clientId: string, hostname: string) {
    suspendMutation.mutate(clientId, {
      onSuccess: () => {
        toast.success(`${hostname} suspended`);
        setSuspendConfirm(null);
      },
      onError: (err) => {
        toast.error(
          `Failed to suspend ${hostname}: ${err instanceof Error ? err.message : 'Unknown error'}`,
        );
        setSuspendConfirm(null);
      },
    });
  }

  function handleDelete(clientId: string, hostname: string) {
    deleteMutation.mutate(clientId, {
      onSuccess: () => {
        toast.success(`${hostname} deleted`);
        setDeleteConfirm(null);
        setExpanded({});
      },
      onError: (err) => {
        toast.error(
          `Failed to delete ${hostname}: ${err instanceof Error ? err.message : 'Unknown error'}`,
        );
        setDeleteConfirm(null);
      },
    });
  }

  const declineTarget = data?.clients.find((c) => c.id === declineConfirm);
  const suspendTarget = data?.clients.find((c) => c.id === suspendConfirm);

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
                <ChevronDown style={{ width: 14, height: 14 }} />
              ) : (
                <ChevronRight style={{ width: 14, height: 14 }} />
              )}
            </button>
          );
        },
      }),
      columnHelper.accessor('hostname', {
        header: 'Client Name',
        cell: (info) => {
          const client = info.row.original;
          const avatar = getAvatar(client.hostname);
          return (
            <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
              <div
                style={{
                  width: 32,
                  height: 32,
                  borderRadius: 8,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: 13,
                  fontWeight: 700,
                  flexShrink: 0,
                  background: 'var(--bg-card-hover)',
                  border: '1px solid var(--border)',
                  color: 'var(--text-secondary)',
                }}
              >
                {avatar.letter}
              </div>
              <div>
                <Link
                  to={`/clients/${client.id}`}
                  style={{
                    fontWeight: 500,
                    fontSize: 13,
                    color: 'var(--accent)',
                    textDecoration: 'none',
                  }}
                  onClick={(e) => e.stopPropagation()}
                >
                  {client.hostname}
                </Link>
                {client.os && (
                  <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>{client.os}</div>
                )}
              </div>
            </div>
          );
        },
      }),
      columnHelper.display({
        id: 'status',
        header: 'Status',
        cell: (info) => <StatusCell client={info.row.original} />,
      }),
      columnHelper.accessor('endpoint_count', {
        header: 'Endpoints',
        cell: (info) => {
          const client = info.row.original;
          return (
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <span style={{ fontWeight: 500, fontFamily: 'var(--font-mono)', fontSize: 13 }}>
                {info.getValue()}
              </span>
              <EndpointSparkline clientId={client.id} count={info.getValue()} />
            </div>
          );
        },
      }),
      columnHelper.accessor('last_sync_at', {
        header: 'Last Sync',
        cell: (info) => {
          const client = info.row.original;
          return (
            <div>
              <div style={{ fontSize: 12, color: 'var(--text-secondary)' }}>
                {formatRelativeTime(info.getValue())}
              </div>
              <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>
                Next: {formatNextSync(info.getValue(), client.sync_interval)}
              </div>
            </div>
          );
        },
      }),
      columnHelper.display({
        id: 'health',
        header: 'Health Score',
        cell: (info) => (
          <HealthCell
            score={computeHealthScore(info.row.original.last_sync_at, info.row.original.status)}
          />
        ),
      }),
      columnHelper.accessor('version', {
        header: 'Version',
        cell: (info) => (
          <span
            style={{ fontFamily: 'var(--font-mono)', fontSize: 12, color: 'var(--text-secondary)' }}
          >
            {info.getValue() ?? '—'}
          </span>
        ),
      }),
      columnHelper.display({
        id: 'tier',
        header: 'Tier',
        cell: (info) => {
          const clientLicense = getClientLicense(
            info.row.original.id,
            licensesData?.licenses ?? [],
          );
          if (!clientLicense) {
            return <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>—</span>;
          }
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
                ...tierBadgeStyle(clientLicense.tier),
              }}
            >
              {clientLicense.tier}
            </span>
          );
        },
      }),
      columnHelper.display({
        id: 'interval',
        header: 'Interval',
        cell: (info) => (
          <span style={{ fontSize: 12, color: 'var(--text-secondary)' }}>
            {formatSyncInterval(info.row.original.sync_interval)}
          </span>
        ),
      }),
    ],
    [licensesData],
  );

  const table = useReactTable({
    data: filteredClients,
    columns,
    state: { expanded },
    onExpandedChange: setExpanded,
    getCoreRowModel: getCoreRowModel(),
    getExpandedRowModel: getExpandedRowModel(),
  });

  return (
    <TooltipProvider>
      <div style={{ padding: 24 }}>
        {/* Confirm dialogs */}
        <ConfirmDialog
          open={!!declineConfirm}
          title="Decline client?"
          description={`This will set ${declineTarget?.hostname ?? 'this client'} to declined. It will no longer be able to sync.`}
          confirmLabel="Decline"
          isPending={declineMutation.isPending}
          onConfirm={() =>
            declineConfirm && declineTarget && handleDecline(declineConfirm, declineTarget.hostname)
          }
          onCancel={() => setDeclineConfirm(null)}
        />
        <ConfirmDialog
          open={!!suspendConfirm}
          title="Suspend client?"
          description={`This will suspend ${suspendTarget?.hostname ?? 'this client'}. Sync will be paused.`}
          confirmLabel="Suspend"
          isPending={suspendMutation.isPending}
          onConfirm={() =>
            suspendConfirm && suspendTarget && handleSuspend(suspendConfirm, suspendTarget.hostname)
          }
          onCancel={() => setSuspendConfirm(null)}
        />

        <Dialog open={showAddClientInfo} onOpenChange={setShowAddClientInfo}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Add a Client</DialogTitle>
              <DialogDescription>
                Clients are added by installing the PatchIQ Agent on a Patch Manager instance. Once
                installed, the agent will automatically enroll and appear here as a pending client
                for approval.
              </DialogDescription>
            </DialogHeader>
            <div
              style={{
                padding: '12px 16px',
                background: 'var(--bg-inset)',
                borderRadius: 8,
                border: '1px solid var(--border)',
                fontFamily: 'var(--font-mono)',
                fontSize: 12,
                color: 'var(--text-primary)',
              }}
            >
              <div
                style={{
                  fontSize: 10,
                  textTransform: 'uppercase',
                  color: 'var(--text-muted)',
                  marginBottom: 6,
                  fontWeight: 600,
                  letterSpacing: '0.05em',
                }}
              >
                Steps
              </div>
              <ol
                style={{
                  margin: 0,
                  paddingLeft: 16,
                  display: 'flex',
                  flexDirection: 'column',
                  gap: 4,
                }}
              >
                <li>Download the agent from the Agent Downloads page</li>
                <li>Install it on the target Patch Manager instance</li>
                <li>The agent will register and appear here as &quot;Pending&quot;</li>
                <li>Approve the client to start syncing</li>
              </ol>
            </div>
            <DialogFooter>
              <Button size="sm" onClick={() => setShowAddClientInfo(false)}>
                Got it
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        {/* Stat Cards */}
        <div style={{ display: 'flex', gap: 12, marginBottom: 16 }}>
          <StatCard
            label="Total Clients"
            value={stats.total}
            active={statusFilter === ''}
            onClick={() => {
              setStatusFilter('');
              setPage(0);
            }}
          />
          <StatCard
            label="Connected"
            value={stats.connected}
            valueColor="var(--signal-healthy)"
            active={statusFilter === 'connected'}
            onClick={() => {
              setStatusFilter('connected');
              setPage(0);
            }}
          />
          <StatCard
            label="Pending"
            value={stats.pending}
            valueColor="var(--signal-warning)"
            active={statusFilter === 'pending'}
            onClick={() => {
              setStatusFilter('pending');
              setPage(0);
            }}
          />
          <StatCard
            label="Disconnected"
            value={stats.disconnected}
            valueColor="var(--signal-critical)"
            active={statusFilter === 'disconnected'}
            onClick={() => {
              setStatusFilter('disconnected');
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
              className="h-7 text-sm"
              style={{
                borderColor: 'var(--border)',
                background: 'var(--bg-card)',
                width: 'auto',
                minWidth: 90,
              }}
            >
              <SelectValue placeholder="All" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">
                All{' '}
                <span
                  style={{
                    color: 'var(--text-muted)',
                    fontFamily: 'var(--font-mono)',
                    fontSize: 10,
                  }}
                >
                  {stats.total}
                </span>
              </SelectItem>
              <SelectItem value="connected">
                Connected{' '}
                <span
                  style={{
                    color: 'var(--text-muted)',
                    fontFamily: 'var(--font-mono)',
                    fontSize: 10,
                  }}
                >
                  {stats.connected}
                </span>
              </SelectItem>
              <SelectItem value="pending">
                Pending{' '}
                <span
                  style={{
                    color: 'var(--text-muted)',
                    fontFamily: 'var(--font-mono)',
                    fontSize: 10,
                  }}
                >
                  {stats.pending}
                </span>
              </SelectItem>
              <SelectItem value="disconnected">
                Disconnected{' '}
                <span
                  style={{
                    color: 'var(--text-muted)',
                    fontFamily: 'var(--font-mono)',
                    fontSize: 10,
                  }}
                >
                  {stats.disconnected}
                </span>
              </SelectItem>
            </SelectContent>
          </Select>
          <FilterSearch
            value={search}
            onChange={(v) => {
              setSearch(v);
              setPage(0);
            }}
            placeholder="Search clients..."
          />
          <div style={{ marginLeft: 'auto', display: 'flex', alignItems: 'center', gap: 8 }}>
            <Button
              variant="outline"
              size="sm"
              onClick={handleExport}
              style={{ display: 'flex', alignItems: 'center', gap: 6 }}
            >
              <Download style={{ width: 14, height: 14 }} />
              Export
            </Button>
            <Button
              size="sm"
              onClick={() => setShowAddClientInfo(true)}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 6,
                background: 'var(--accent)',
                color: 'var(--text-emphasis)',
              }}
            >
              <Plus style={{ width: 14, height: 14 }} />
              Add Client
            </Button>
          </div>
        </FilterBar>

        {/* Table */}
        {isLoading ? (
          <div
            style={{
              borderRadius: 8,
              border: '1px solid var(--border)',
              overflow: 'hidden',
            }}
          >
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13 }}>
              <thead>
                <tr>
                  {[
                    '',
                    'Client Name',
                    'Status',
                    'Endpoints',
                    'Last Sync',
                    'Health Score',
                    'Version',
                    'Tier',
                    'Interval',
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
            title="Failed to load clients"
            message={error instanceof Error ? error.message : 'An unknown error occurred'}
          />
        ) : filteredClients.length === 0 ? (
          <EmptyState
            title="No clients found"
            description={
              statusFilter || search
                ? 'Try adjusting your filters'
                : 'No Patch Manager instances have registered yet'
            }
          />
        ) : (
          <>
            <DataTable
              table={table}
              isRowFailed={(client) =>
                client.status === 'declined' || client.status === 'suspended'
              }
              onRowClick={(client) => {
                const row = table.getRowModel().rows.find((r) => r.original === client);
                row?.toggleExpanded();
              }}
              renderExpandedRow={(client) => {
                const clientLicense = getClientLicense(client.id, licensesData?.licenses ?? []);
                return (
                  <ExpandedPanel
                    client={client}
                    license={clientLicense}
                    onApprove={() => handleApprove(client.id, client.hostname)}
                    onDeclineRequest={() => setDeclineConfirm(client.id)}
                    onSuspendRequest={() => setSuspendConfirm(client.id)}
                    onDeleteRequest={() => setDeleteConfirm(client.id)}
                    onDeleteConfirm={() => handleDelete(client.id, client.hostname)}
                    onDeleteCancel={() => setDeleteConfirm(null)}
                    confirmingDelete={deleteConfirm === client.id}
                    approvePending={approveMutation.isPending}
                    declinePending={declineMutation.isPending}
                    suspendPending={suspendMutation.isPending}
                    deletePending={deleteMutation.isPending}
                  />
                );
              }}
            />
            <DataTablePagination
              hasPrev={page > 0}
              hasNext={page + 1 < totalPages}
              onPrev={() => setPage((p) => Math.max(0, p - 1))}
              onNext={() => setPage((p) => p + 1)}
            />
          </>
        )}
      </div>
    </TooltipProvider>
  );
};
