import { useParams, Link } from 'react-router';
import {
  useClient,
  useUpdateClient,
  useApproveClient,
  useSuspendClient,
  useClientSyncHistory,
  useClientEndpointTrend,
} from '../../api/hooks/useClients';
import { Button, Skeleton, Card, CardContent, CardHeader, CardTitle } from '@patchiq/ui';
import { ArrowLeft, Star, Calendar, CheckCircle2, XCircle, KeyRound } from 'lucide-react';
import { useState } from 'react';
import { toast } from 'sonner';
import type { Client } from '../../types/client';
import { useLicenses } from '../../api/hooks/useLicenses';
import type { License } from '../../types/license';
import { tierBadgeStyle } from '../../lib/tierUtils';

// ── Helpers ─────────────────────────────────────────────────────────────────

// Deterministic palette using CSS variable tokens — works in light and dark mode
const AVATAR_COLORS: string[] = [
  'var(--bg-elevated)',
  'var(--border-strong)',
  'var(--bg-card-hover)',
  'var(--bg-elevated)',
  'var(--border-strong)',
  'var(--bg-card-hover)',
  'var(--bg-elevated)',
  'var(--border-strong)',
];

function getAvatar(hostname: string): { letter: string; color: string } {
  const idx = hostname.charCodeAt(0) % AVATAR_COLORS.length;
  return { letter: hostname[0].toUpperCase(), color: AVATAR_COLORS[idx] };
}

function computeHealthScore(lastSyncAt: string | null): number {
  if (!lastSyncAt) return 10;
  const diffMin = (Date.now() - new Date(lastSyncAt).getTime()) / 60000;
  if (diffMin < 30) return 100;
  if (diffMin < 60) return 90;
  if (diffMin < 360) return 75;
  if (diffMin < 1440) return 60;
  if (diffMin < 10080) return 30;
  return 10;
}

function healthCssColor(score: number): string {
  if (score >= 80) return 'var(--signal-healthy)';
  if (score >= 60) return 'var(--signal-warning)';
  return 'var(--signal-critical)';
}

function formatRelativeTime(dateStr: string | null): string {
  if (!dateStr) return 'Never';
  const diffMs = Date.now() - new Date(dateStr).getTime();
  const diffMin = Math.floor(diffMs / 60000);
  if (diffMin < 1) return 'just now';
  if (diffMin < 60) return `${diffMin}m ago`;
  const diffH = Math.floor(diffMin / 60);
  if (diffH < 24) return `${diffH}h ago`;
  return `${Math.floor(diffH / 24)}d ago`;
}

function formatSyncInterval(seconds: number): string {
  if (seconds < 3600) return `${Math.floor(seconds / 60)} min`;
  const h = Math.floor(seconds / 3600);
  return `${h} hour${h !== 1 ? 's' : ''}`;
}

function isConnected(client: Client): boolean {
  const score = computeHealthScore(client.last_sync_at);
  return client.status === 'approved' && score >= 60;
}

// ── Stat Card ────────────────────────────────────────────────────────────────

function StatCard({
  label,
  value,
  sub,
  valueStyle,
  children,
}: {
  label: string;
  value?: string;
  sub?: string;
  valueStyle?: React.CSSProperties;
  children?: React.ReactNode;
}) {
  return (
    <div
      style={{
        padding: '12px 14px',
        borderRadius: 8,
        border: '1px solid var(--border)',
        background: 'var(--bg-card)',
      }}
    >
      <p
        style={{
          fontSize: 10,
          fontWeight: 500,
          fontFamily: 'var(--font-mono)',
          textTransform: 'uppercase',
          letterSpacing: '0.05em',
          color: 'var(--text-muted)',
          marginTop: 4,
        }}
      >
        {label}
      </p>
      {children ? (
        children
      ) : (
        <>
          <p
            style={{
              fontSize: 22,
              fontWeight: 700,
              fontFamily: 'var(--font-mono)',
              letterSpacing: '-0.02em',
              lineHeight: 1,
              ...valueStyle,
            }}
          >
            {value}
          </p>
          {sub && <p style={{ fontSize: 10, color: 'var(--text-muted)', marginTop: 4 }}>{sub}</p>}
        </>
      )}
    </div>
  );
}

// ── Tab content components ───────────────────────────────────────────────────

function DataPendingPlaceholder() {
  return (
    <div
      style={{
        height: '88px',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        border: '1px dashed var(--border)',
        borderRadius: '8px',
        background: 'var(--bg-card)',
      }}
    >
      <p style={{ fontSize: '12px', color: 'var(--text-muted)', fontFamily: 'var(--font-sans)' }}>
        Data available after next sync
      </p>
    </div>
  );
}

/** Bordered tile card matching Patch Manager Overview style. */
function Tile({
  title,
  rightLabel,
  children,
  style,
}: {
  title: string;
  rightLabel?: string;
  children: React.ReactNode;
  style?: React.CSSProperties;
}) {
  return (
    <div
      style={{
        background: 'var(--bg-card)',
        border: '1px solid var(--border)',
        borderRadius: 10,
        overflow: 'hidden',
        ...style,
      }}
    >
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '12px 16px 10px',
          borderBottom: '1px solid var(--border)',
        }}
      >
        <span
          style={{
            fontSize: 11,
            fontWeight: 600,
            letterSpacing: '0.06em',
            textTransform: 'uppercase',
            color: 'var(--text-secondary)',
            fontFamily: 'var(--font-mono)',
          }}
        >
          {title}
        </span>
        {rightLabel && (
          <span
            style={{ fontSize: 10, color: 'var(--text-muted)', fontFamily: 'var(--font-mono)' }}
          >
            {rightLabel}
          </span>
        )}
      </div>
      <div style={{ padding: '14px 16px' }}>{children}</div>
    </div>
  );
}

function EndpointTrendChart({ clientId }: { clientId: string }) {
  const { data, isLoading } = useClientEndpointTrend(clientId, 90);

  if (isLoading) {
    return <Skeleton className="h-[88px] w-full" />;
  }

  const points = data?.points ?? [];
  if (points.length === 0) {
    return <DataPendingPlaceholder />;
  }

  const maxV = Math.max(...points.map((p) => p.total), 1);
  const barW = 26;
  const gap = 8;
  const svgH = 72;
  const svgW = points.length * (barW + gap) - gap;

  // Show at most ~7 labels evenly distributed
  const labelStep = Math.max(1, Math.floor(points.length / 7));

  return (
    <div>
      <svg
        width="100%"
        viewBox={`0 0 ${svgW} ${svgH + 16}`}
        role="img"
        aria-label="Endpoint count trend"
      >
        {points.map((pt, i) => {
          const barH = Math.max(4, Math.round((pt.total / maxV) * svgH));
          const showLabel = i % labelStep === 0 || i === points.length - 1;
          const label = new Date(pt.date).toLocaleDateString('en-US', {
            month: 'short',
            day: 'numeric',
          });
          return (
            <g key={pt.date}>
              <rect
                x={i * (barW + gap)}
                y={svgH - barH}
                width={barW}
                height={barH}
                rx={3}
                style={{ fill: 'color-mix(in srgb, var(--signal-healthy) 70%, transparent)' }}
              />
              {showLabel && (
                <text
                  x={i * (barW + gap) + barW / 2}
                  y={svgH + 12}
                  textAnchor="middle"
                  fontSize="8"
                  className="fill-muted-foreground"
                >
                  {label}
                </text>
              )}
            </g>
          );
        })}
      </svg>
      <p className="text-xs text-muted-foreground mt-1">
        {points[points.length - 1]?.total.toLocaleString() ?? '—'} endpoints (latest)
      </p>
    </div>
  );
}

function SyncSuccessGrid({ clientId }: { clientId: string }) {
  const { data, isLoading } = useClientSyncHistory(clientId, 20);

  if (isLoading) {
    return <Skeleton className="h-8 w-full" />;
  }

  const items = data?.items ?? [];
  if (items.length === 0) {
    return <p className="text-xs text-muted-foreground">No sync history yet</p>;
  }

  const successCount = items.filter((s) => s.status === 'success').length;

  function squareColor(status: string): string {
    if (status === 'success') return 'var(--signal-healthy)';
    if (status === 'partial') return 'var(--signal-warning)';
    return 'var(--signal-critical)';
  }

  return (
    <div className="flex items-center gap-2 flex-wrap">
      <div className="flex flex-wrap gap-1">
        {items.map((s) => (
          <div
            key={s.id}
            className="w-4 h-4 rounded-sm"
            style={{ background: squareColor(s.status) }}
            title={`${new Date(s.synced_at).toLocaleString()} — ${s.status}`}
          />
        ))}
      </div>
      <span className="text-xs text-muted-foreground">
        {successCount}/{items.length} successful
      </span>
    </div>
  );
}

function OverviewTab({ client }: { client: Client }) {
  const rows: { label: string; value: React.ReactNode }[] = [
    {
      label: 'Status',
      value: (
        <span
          className="capitalize font-medium"
          style={{
            color:
              client.status === 'approved'
                ? 'var(--signal-healthy)'
                : client.status === 'suspended'
                  ? 'var(--signal-critical)'
                  : 'var(--signal-warning)',
          }}
        >
          {client.status}
        </span>
      ),
    },
    {
      label: 'PM Version',
      value: (
        <span style={{ fontFamily: 'var(--font-mono)', fontSize: 13 }}>
          {client.version ? `v${client.version}` : '—'}
        </span>
      ),
    },
    { label: 'OS', value: client.os ?? '—' },
    { label: 'Contact', value: client.contact_email ?? '—' },
    { label: 'First Connected', value: new Date(client.created_at).toLocaleDateString() },
    {
      label: 'Last Sync',
      value: client.last_sync_at ? new Date(client.last_sync_at).toLocaleString() : 'Never',
    },
  ];

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
      {/* Endpoint trend chart */}
      <Tile title="Endpoint Count" rightLabel="Last 90 days">
        <EndpointTrendChart clientId={client.id} />
      </Tile>

      {/* Sync frequency + success grid */}
      <Tile title="Sync Activity">
        <div className="flex flex-col sm:flex-row sm:items-start gap-6">
          <div className="flex-shrink-0">
            <p
              style={{
                fontSize: 10,
                fontWeight: 600,
                letterSpacing: '0.06em',
                textTransform: 'uppercase',
                color: 'var(--text-muted)',
                fontFamily: 'var(--font-mono)',
                marginBottom: 4,
              }}
            >
              Frequency
            </p>
            <p
              style={{
                fontSize: 20,
                fontWeight: 700,
                fontFamily: 'var(--font-mono)',
                color: 'var(--text-primary)',
              }}
            >
              {formatSyncInterval(client.sync_interval)}
            </p>
          </div>
          <div className="flex-1">
            <p
              style={{
                fontSize: 10,
                fontWeight: 600,
                letterSpacing: '0.06em',
                textTransform: 'uppercase',
                color: 'var(--text-muted)',
                fontFamily: 'var(--font-mono)',
                marginBottom: 8,
              }}
            >
              Recent syncs (last 20)
            </p>
            <SyncSuccessGrid clientId={client.id} />
          </div>
        </div>
      </Tile>

      {/* Connection info */}
      <Tile title="Connection Info">
        <table className="w-full" style={{ fontSize: 13 }}>
          <tbody>
            {rows.map(({ label, value }) => (
              <tr key={label} style={{ borderBottom: '1px solid var(--border)' }}>
                <td
                  style={{
                    padding: '7px 0',
                    color: 'var(--text-muted)',
                    fontFamily: 'var(--font-mono)',
                    fontSize: 11,
                    textTransform: 'uppercase',
                    letterSpacing: '0.05em',
                    width: '30%',
                    paddingRight: 16,
                  }}
                >
                  {label}
                </td>
                <td style={{ padding: '7px 0', color: 'var(--text-primary)' }}>{value}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </Tile>
    </div>
  );
}

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

function OsDonutChart({ client }: { client: Client }) {
  const raw = (client as unknown as { os_summary?: unknown }).os_summary;
  const decoded = decodeBase64Json<OsSummaryEntry[] | Record<string, unknown>>(raw);
  const osSummary: OsSummaryEntry[] | null = Array.isArray(decoded) ? decoded : null;
  if (!osSummary || osSummary.length === 0) {
    return <DataPendingPlaceholder />;
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
    <div className="flex items-center gap-3">
      <svg width="72" height="72" role="img" aria-label="OS breakdown">
        {paths.map((p) => (
          <path key={p.label} d={p.d} fill={p.color} />
        ))}
      </svg>
      <div className="space-y-1">
        {paths.map((p) => (
          <div key={p.label} className="flex items-center gap-1.5 text-xs">
            <span className="w-2 h-2 rounded-sm flex-shrink-0" style={{ background: p.color }} />
            <span style={{ color: 'var(--text-primary)' }}>{p.label}</span>
            <span className="text-muted-foreground">{Math.round(p.pct)}%</span>
          </div>
        ))}
      </div>
    </div>
  );
}

interface StatusSummaryEntry {
  status: string;
  count: number;
  pct: number;
}

function statusBarColor(status: string): string {
  if (status === 'online' || status === 'active') return 'var(--signal-healthy)';
  if (status === 'offline' || status === 'inactive') return 'var(--signal-critical)';
  if (status === 'pending') return 'var(--signal-warning)';
  return 'var(--text-muted)';
}

function EndpointStatusBars({ client }: { client: Client }) {
  const rawStatus = (client as unknown as { endpoint_status_summary?: unknown })
    .endpoint_status_summary;
  const decodedStatus = decodeBase64Json<StatusSummaryEntry[] | Record<string, unknown>>(rawStatus);
  const statusSummary: StatusSummaryEntry[] | null = Array.isArray(decodedStatus)
    ? decodedStatus
    : null;
  if (!statusSummary || statusSummary.length === 0) {
    return (
      <div role="img" aria-label="Endpoint status distribution">
        <DataPendingPlaceholder />
      </div>
    );
  }

  return (
    <div role="img" aria-label="Endpoint status distribution" className="space-y-2">
      {statusSummary.map((entry) => (
        <div key={entry.status}>
          <div className="flex justify-between text-xs mb-0.5">
            <span className="capitalize" style={{ color: 'var(--text-primary)' }}>
              {entry.status}
            </span>
            <span style={{ color: 'var(--text-muted)' }}>
              {entry.count} ({Math.round(entry.pct)}%)
            </span>
          </div>
          <div className="w-full rounded-full h-1.5" style={{ background: 'var(--bg-card-hover)' }}>
            <div
              className="h-1.5 rounded-full"
              style={{ width: `${entry.pct}%`, background: statusBarColor(entry.status) }}
            />
          </div>
        </div>
      ))}
    </div>
  );
}

interface ComplianceSummaryEntry {
  framework: string;
  score: number;
}

function ComplianceBars({ client }: { client: Client }) {
  const rawCompliance = (client as unknown as { compliance_summary?: unknown }).compliance_summary;
  const decodedCompliance = decodeBase64Json<ComplianceSummaryEntry[] | Record<string, unknown>>(
    rawCompliance,
  );
  const complianceSummary: ComplianceSummaryEntry[] | null = Array.isArray(decodedCompliance)
    ? decodedCompliance
    : null;
  if (!complianceSummary || complianceSummary.length === 0) {
    return (
      <div role="img" aria-label="Compliance by framework">
        <DataPendingPlaceholder />
      </div>
    );
  }

  function complianceColor(score: number): string {
    if (score >= 80) return 'var(--signal-healthy)';
    if (score >= 60) return 'var(--signal-warning)';
    return 'var(--signal-critical)';
  }

  return (
    <div role="img" aria-label="Compliance by framework" className="space-y-2">
      {complianceSummary.map((entry) => (
        <div key={entry.framework}>
          <div className="flex justify-between text-xs mb-0.5">
            <span style={{ color: 'var(--text-primary)' }}>{entry.framework}</span>
            <span style={{ color: complianceColor(entry.score) }}>{entry.score}%</span>
          </div>
          <div className="w-full rounded-full h-1.5" style={{ background: 'var(--bg-card-hover)' }}>
            <div
              className="h-1.5 rounded-full"
              style={{ width: `${entry.score}%`, background: complianceColor(entry.score) }}
            />
          </div>
        </div>
      ))}
    </div>
  );
}

function EndpointsTab({ client }: { client: Client }) {
  return (
    <div className="space-y-4">
      <p
        className="text-sm text-muted-foreground p-3 rounded-lg italic"
        style={{ background: 'var(--bg-canvas)', border: '1px solid var(--border)' }}
      >
        Hub Manager sees aggregate statistics only. Individual endpoint data is managed by each
        Patch Manager instance.
      </p>
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <Tile title="Endpoints by OS">
          <OsDonutChart client={client} />
        </Tile>
        <Tile title="Endpoints by Status">
          <EndpointStatusBars client={client} />
        </Tile>
        <Tile title="Compliance by Framework">
          <ComplianceBars client={client} />
        </Tile>
      </div>
    </div>
  );
}

function SyncHistoryTab({ clientId }: { clientId: string }) {
  const { data, isLoading } = useClientSyncHistory(clientId, 50);

  return (
    <div className="space-y-4">
      <p className="text-sm font-medium">Sync History</p>
      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-10 w-full" />
          ))}
        </div>
      ) : !data?.items.length ? (
        <div
          className="rounded-lg p-8 text-center text-sm"
          style={{
            background: 'var(--bg-canvas)',
            color: 'var(--text-muted)',
            border: '1px solid var(--border)',
          }}
        >
          No sync history recorded yet
        </div>
      ) : (
        <div className="rounded-lg overflow-hidden" style={{ border: '1px solid var(--border)' }}>
          <table className="w-full text-xs">
            <thead>
              <tr className="text-left" style={{ background: 'var(--bg-canvas)' }}>
                <th className="px-3 py-2 font-semibold text-muted-foreground">Time</th>
                <th className="px-3 py-2 font-semibold text-muted-foreground">Entries</th>
                <th className="px-3 py-2 font-semibold text-muted-foreground">Deletes</th>
                <th className="px-3 py-2 font-semibold text-muted-foreground">Duration</th>
                <th className="px-3 py-2 font-semibold text-muted-foreground">Endpoints</th>
                <th className="px-3 py-2 font-semibold text-muted-foreground">Status</th>
              </tr>
            </thead>
            <tbody>
              {data.items.map((item) => (
                <tr key={item.id} style={{ borderTop: '1px solid var(--border)' }}>
                  <td className="px-3 py-2 font-mono" style={{ color: 'var(--text-primary)' }}>
                    {new Date(item.synced_at).toLocaleString()}
                  </td>
                  <td className="px-3 py-2" style={{ color: 'var(--text-primary)' }}>
                    {item.patches_synced.toLocaleString()}
                  </td>
                  <td className="px-3 py-2" style={{ color: 'var(--text-primary)' }}>
                    {item.cves_synced.toLocaleString()}
                  </td>
                  <td className="px-3 py-2" style={{ color: 'var(--text-primary)' }}>
                    {item.duration_ms < 1000
                      ? `${item.duration_ms}ms`
                      : `${(item.duration_ms / 1000).toFixed(1)}s`}
                  </td>
                  <td className="px-3 py-2" style={{ color: 'var(--text-muted)' }}>
                    —
                  </td>
                  <td className="px-3 py-2">
                    {item.status === 'success' ? (
                      <span className="font-medium" style={{ color: 'var(--signal-healthy)' }}>
                        ✓ OK
                      </span>
                    ) : item.status === 'partial' ? (
                      <span className="font-medium" style={{ color: 'var(--signal-warning)' }}>
                        ⚠ Partial
                      </span>
                    ) : (
                      <span
                        className="font-medium"
                        style={{ color: 'var(--signal-critical)' }}
                        title={item.error ?? ''}
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
      )}
    </div>
  );
}

const TIER_COLORS: Record<string, string> = {
  community: 'var(--text-muted)',
  professional: 'var(--accent)',
  enterprise: 'var(--signal-warning)',
  msp: 'var(--signal-healthy)',
};

// Per-tier feature definitions (key → label, enabled per tier)
const ALL_FEATURES: { key: string; label: string; tiers: string[] }[] = [
  {
    key: 'basic_patching',
    label: 'Basic Patching',
    tiers: ['community', 'professional', 'enterprise', 'msp'],
  },
  {
    key: 'endpoint_management',
    label: 'Endpoint Management',
    tiers: ['community', 'professional', 'enterprise', 'msp'],
  },
  {
    key: 'workflow_builder',
    label: 'Workflow Builder',
    tiers: ['professional', 'enterprise', 'msp'],
  },
  {
    key: 'compliance_reports',
    label: 'Compliance Reports',
    tiers: ['professional', 'enterprise', 'msp'],
  },
  { key: 'custom_rbac', label: 'Custom RBAC', tiers: ['enterprise', 'msp'] },
  { key: 'sso_saml', label: 'SSO / SAML', tiers: ['enterprise', 'msp'] },
  { key: 'multi_site', label: 'Multi-Site', tiers: ['enterprise', 'msp'] },
  { key: 'ha_dr', label: 'HA / DR', tiers: ['enterprise', 'msp'] },
  { key: 'multi_tenant', label: 'Multi-Tenant', tiers: ['msp'] },
  { key: 'white_label', label: 'White-Label', tiers: ['msp'] },
];

function LicenseTab({ client, license }: { client: Client; license?: License }) {
  if (!license) {
    return (
      <Tile title="License">
        <div
          style={{ display: 'flex', flexDirection: 'column', gap: 12, alignItems: 'flex-start' }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            <div
              style={{
                width: 36,
                height: 36,
                borderRadius: 8,
                background: 'color-mix(in srgb, var(--text-muted) 10%, transparent)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                flexShrink: 0,
              }}
            >
              <KeyRound style={{ width: 16, height: 16, color: 'var(--text-muted)' }} />
            </div>
            <div>
              <p style={{ fontSize: 13, fontWeight: 600, color: 'var(--text-primary)', margin: 0 }}>
                No license assigned
              </p>
              <p style={{ fontSize: 12, color: 'var(--text-muted)', margin: '2px 0 0' }}>
                This client has no active license.
              </p>
            </div>
          </div>
          <Link
            to="/licenses"
            style={{
              fontSize: 12,
              fontWeight: 500,
              color: 'var(--accent)',
              textDecoration: 'none',
            }}
          >
            Manage Licenses →
          </Link>
        </div>
      </Tile>
    );
  }

  const usedEndpoints = license.client_endpoint_count ?? client.endpoint_count;
  const usagePct = Math.min(100, Math.round((usedEndpoints / license.max_endpoints) * 100));
  const tierColor = TIER_COLORS[license.tier] ?? 'var(--text-muted)';
  const maskedKey = license.license_key
    ? `${license.license_key.slice(0, 4)}-****-****-${license.license_key.slice(-4)}`
    : '****-****-****';

  const now = Date.now();
  const expiresMs = new Date(license.expires_at).getTime();
  const daysRemaining = Math.max(0, Math.ceil((expiresMs - now) / 86_400_000));
  const isExpired = expiresMs < now;
  const slotsRemaining = license.max_endpoints - usedEndpoints;

  const usageBarColor =
    usagePct > 85
      ? 'var(--signal-critical)'
      : usagePct > 60
        ? 'var(--signal-warning)'
        : 'var(--signal-healthy)';

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
      {/* Compound license card — tier badge + usage + expiry */}
      <div
        style={{
          background: 'var(--bg-card)',
          border: '1px solid var(--border)',
          borderRadius: 10,
          display: 'flex',
          overflow: 'hidden',
        }}
      >
        {/* Left: tier badge */}
        <div
          style={{
            minWidth: 96,
            padding: '16px 12px',
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            gap: 4,
            background: `linear-gradient(135deg, color-mix(in srgb, ${tierColor} 12%, transparent), color-mix(in srgb, ${tierColor} 4%, transparent))`,
            borderRight: `1px solid color-mix(in srgb, ${tierColor} 20%, transparent)`,
          }}
        >
          <Star style={{ width: 20, height: 20, color: tierColor }} />
          <span
            style={{
              fontSize: 12,
              fontWeight: 700,
              color: tierColor,
              textTransform: 'uppercase',
              letterSpacing: '0.04em',
              fontFamily: 'var(--font-sans)',
              marginTop: 2,
            }}
          >
            {license.tier}
          </span>
          <span
            style={{
              fontSize: 9,
              fontFamily: 'var(--font-mono)',
              color: 'var(--text-faint)',
              textTransform: 'uppercase',
              letterSpacing: '0.1em',
            }}
          >
            TIER
          </span>
        </div>

        {/* Right: customer info + usage + expiry */}
        <div
          style={{
            flex: 1,
            padding: '16px 20px',
            display: 'flex',
            flexDirection: 'column',
            gap: 12,
          }}
        >
          {/* Customer */}
          <div>
            <p style={{ fontSize: 14, fontWeight: 600, color: 'var(--text-emphasis)', margin: 0 }}>
              {license.customer_name}
            </p>
            {license.customer_email && (
              <p
                style={{
                  fontSize: 11,
                  color: 'var(--text-muted)',
                  margin: '2px 0 0',
                  fontFamily: 'var(--font-mono)',
                }}
              >
                {license.customer_email}
              </p>
            )}
          </div>

          {/* Usage bar */}
          <div style={{ display: 'flex', flexDirection: 'column', gap: 5 }}>
            <div
              style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline' }}
            >
              <span
                style={{
                  fontSize: 11,
                  color: 'var(--text-secondary)',
                  fontFamily: 'var(--font-sans)',
                }}
              >
                Endpoint Usage
              </span>
              <span
                style={{
                  fontSize: 11,
                  fontFamily: 'var(--font-mono)',
                  color: 'var(--text-primary)',
                  fontWeight: 600,
                }}
              >
                {usedEndpoints.toLocaleString()} / {license.max_endpoints.toLocaleString()}
              </span>
            </div>
            <div
              style={{
                height: 6,
                background: 'var(--bg-inset)',
                borderRadius: 3,
                overflow: 'hidden',
              }}
            >
              <div
                style={{
                  height: '100%',
                  width: `${usagePct}%`,
                  background: usageBarColor,
                  borderRadius: 3,
                  transition: 'width 0.3s',
                }}
              />
            </div>
            <p
              style={{
                fontSize: 10,
                color: 'var(--text-faint)',
                fontFamily: 'var(--font-sans)',
                margin: 0,
              }}
            >
              {slotsRemaining > 0
                ? `${slotsRemaining.toLocaleString()} slots remaining`
                : 'At capacity'}
            </p>
          </div>

          {/* Expiry */}
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <Calendar
              style={{ width: 12, height: 12, color: 'var(--text-faint)', flexShrink: 0 }}
            />
            <span
              style={{ fontSize: 11, fontFamily: 'var(--font-mono)', color: 'var(--text-primary)' }}
            >
              {new Date(license.expires_at).toLocaleDateString('en-US', {
                month: 'short',
                day: 'numeric',
                year: 'numeric',
              })}
            </span>
            <span
              style={{
                fontSize: 10,
                color: isExpired
                  ? 'var(--signal-critical)'
                  : daysRemaining < 30
                    ? 'var(--signal-warning)'
                    : 'var(--signal-healthy)',
                fontFamily: 'var(--font-sans)',
              }}
            >
              {isExpired ? 'Expired' : `${daysRemaining}d remaining`}
            </span>
          </div>
        </div>
      </div>

      {/* License metadata tile */}
      <Tile title="License Details">
        <table className="w-full" style={{ fontSize: 12 }}>
          <tbody>
            {[
              {
                label: 'License Key',
                value: (
                  <span style={{ fontFamily: 'var(--font-mono)', color: 'var(--text-muted)' }}>
                    {maskedKey}
                  </span>
                ),
              },
              {
                label: 'Issued',
                value: new Date(license.issued_at).toLocaleDateString('en-US', {
                  month: 'short',
                  day: 'numeric',
                  year: 'numeric',
                }),
              },
              {
                label: 'Expires',
                value: new Date(license.expires_at).toLocaleDateString('en-US', {
                  month: 'short',
                  day: 'numeric',
                  year: 'numeric',
                }),
              },
              { label: 'Max Endpoints', value: license.max_endpoints.toLocaleString() },
              ...(license.notes ? [{ label: 'Notes', value: license.notes }] : []),
            ].map(({ label, value }) => (
              <tr key={label} style={{ borderBottom: '1px solid var(--border)' }}>
                <td
                  style={{
                    padding: '7px 0',
                    color: 'var(--text-muted)',
                    fontFamily: 'var(--font-mono)',
                    fontSize: 10,
                    textTransform: 'uppercase' as const,
                    letterSpacing: '0.05em',
                    width: '28%',
                    paddingRight: 16,
                  }}
                >
                  {label}
                </td>
                <td style={{ padding: '7px 0', color: 'var(--text-primary)' }}>{value}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </Tile>

      {/* Feature entitlements grid */}
      <Tile title="Feature Entitlements">
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: '1fr 1fr',
            border: '1px solid var(--border)',
            borderRadius: 8,
            overflow: 'hidden',
          }}
        >
          {ALL_FEATURES.map(({ key, label, tiers }, idx) => {
            const enabled = tiers.includes(license.tier);
            const isLeftCol = idx % 2 === 0;
            const isLastRow = idx >= ALL_FEATURES.length - 2;
            return (
              <div
                key={key}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 8,
                  padding: '9px 12px',
                  borderBottom: isLastRow ? 'none' : '1px solid var(--border)',
                  borderRight: isLeftCol ? '1px solid var(--border)' : 'none',
                  background: 'var(--bg-card)',
                }}
              >
                {enabled ? (
                  <div
                    style={{
                      width: 16,
                      height: 16,
                      borderRadius: 4,
                      flexShrink: 0,
                      background: 'color-mix(in srgb, var(--signal-healthy) 10%, transparent)',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                    }}
                  >
                    <CheckCircle2
                      style={{ width: 11, height: 11, color: 'var(--signal-healthy)' }}
                    />
                  </div>
                ) : (
                  <div
                    style={{
                      width: 16,
                      height: 16,
                      borderRadius: 4,
                      flexShrink: 0,
                      background: 'color-mix(in srgb, var(--border) 40%, transparent)',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                    }}
                  >
                    <XCircle style={{ width: 11, height: 11, color: 'var(--text-faint)' }} />
                  </div>
                )}
                <span
                  style={{
                    fontSize: 12,
                    color: enabled ? 'var(--text-primary)' : 'var(--text-faint)',
                    fontFamily: 'var(--font-sans)',
                  }}
                >
                  {label}
                </span>
              </div>
            );
          })}
        </div>
      </Tile>

      <Link
        to={`/licenses/${license.id}`}
        style={{
          fontSize: 12,
          fontWeight: 500,
          color: 'var(--accent)',
          textDecoration: 'none',
          display: 'inline-block',
        }}
      >
        View full license detail →
      </Link>
    </div>
  );
}

interface ConfigTabProps {
  client: Client;
  syncInterval: number | undefined;
  setSyncInterval: (v: number | undefined) => void;
  notes: string;
  setNotes: (v: string) => void;
  onSave: () => void;
  onCancel: () => void;
  isPending: boolean;
  clientStatus: string;
  onStatusToggle: () => void;
  statusTogglePending: boolean;
}

function ConfigTab({
  client,
  syncInterval,
  setSyncInterval,
  notes,
  setNotes,
  onSave,
  onCancel,
  isPending,
  clientStatus,
  onStatusToggle,
  statusTogglePending,
}: ConfigTabProps) {
  const intervalOptions: { label: string; value: number }[] = [
    { label: '5 minutes', value: 300 },
    { label: '15 minutes', value: 900 },
    { label: '30 minutes', value: 1800 },
    { label: '1 hour', value: 3600 },
    { label: '6 hours', value: 21600 },
  ];

  const currentInterval = syncInterval ?? client.sync_interval;

  const labelStyle: React.CSSProperties = {
    fontSize: 11,
    fontWeight: 500,
    fontFamily: 'var(--font-mono)',
    textTransform: 'uppercase',
    letterSpacing: '0.05em',
    color: 'var(--text-muted)',
    marginBottom: 6,
    display: 'block',
  };

  const inputStyle: React.CSSProperties = {
    width: '100%',
    padding: '8px 12px',
    borderRadius: 7,
    border: '1px solid var(--border)',
    background: 'var(--bg-card)',
    color: 'var(--text-primary)',
    fontSize: 13,
    fontFamily: 'var(--font-sans)',
    outline: 'none',
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16, maxWidth: 680 }}>
      {/* Sync Settings */}
      <Card>
        <CardHeader style={{ paddingBottom: 8 }}>
          <CardTitle style={{ fontSize: 13, fontWeight: 600 }}>Sync Settings</CardTitle>
        </CardHeader>
        <CardContent style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
            <div>
              <label style={labelStyle}>Sync Interval</label>
              <select
                style={inputStyle}
                value={currentInterval}
                onChange={(e) => setSyncInterval(Number(e.target.value))}
              >
                {intervalOptions.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label style={labelStyle}>Client Status</label>
              <div style={{ display: 'flex', alignItems: 'center', gap: 10, height: 36 }}>
                <button
                  type="button"
                  onClick={onStatusToggle}
                  disabled={statusTogglePending}
                  aria-label="Toggle client status"
                  style={{
                    width: 44,
                    height: 24,
                    borderRadius: 999,
                    border: 'none',
                    position: 'relative',
                    cursor: statusTogglePending ? 'not-allowed' : 'pointer',
                    opacity: statusTogglePending ? 0.5 : 1,
                    transition: 'background 0.2s',
                    background:
                      clientStatus === 'approved'
                        ? 'var(--signal-healthy)'
                        : 'var(--bg-card-hover)',
                    flexShrink: 0,
                  }}
                >
                  <span
                    style={{
                      position: 'absolute',
                      top: 4,
                      width: 16,
                      height: 16,
                      borderRadius: '50%',
                      background: 'var(--bg-page, white)',
                      transition: 'left 0.2s',
                      left: clientStatus === 'approved' ? 24 : 4,
                    }}
                  />
                </button>
                <span
                  style={{
                    fontSize: 13,
                    color:
                      clientStatus === 'approved' ? 'var(--signal-healthy)' : 'var(--text-muted)',
                  }}
                >
                  {clientStatus === 'approved' ? 'Active' : 'Inactive'}
                </span>
              </div>
              <p style={{ fontSize: 11, color: 'var(--text-muted)', marginTop: 4 }}>
                {clientStatus === 'approved'
                  ? 'Client is approved and can sync with hub.'
                  : 'Client is not approved — syncing is disabled.'}
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Catalog Scope */}
      <Card>
        <CardHeader style={{ paddingBottom: 8 }}>
          <CardTitle style={{ fontSize: 13, fontWeight: 600 }}>Catalog Scope</CardTitle>
        </CardHeader>
        <CardContent>
          <label style={labelStyle}>Scope Filter</label>
          <textarea
            readOnly
            rows={3}
            placeholder="Catalog scope filtering available in a future release"
            style={{
              ...inputStyle,
              resize: 'none',
              fontFamily: 'var(--font-mono)',
              color: 'var(--text-muted)',
              cursor: 'not-allowed',
              fontSize: 12,
            }}
          />
          <p style={{ fontSize: 11, color: 'var(--text-muted)', marginTop: 6 }}>
            Restrict which catalog entries are pushed to this client. Leave empty to sync all
            entries.
          </p>
        </CardContent>
      </Card>

      {/* Notes */}
      <Card>
        <CardHeader style={{ paddingBottom: 8 }}>
          <CardTitle style={{ fontSize: 13, fontWeight: 600 }}>Notes</CardTitle>
        </CardHeader>
        <CardContent>
          <label style={labelStyle}>Internal Notes</label>
          <textarea
            rows={4}
            placeholder="Internal notes for this client..."
            style={{ ...inputStyle, resize: 'vertical', fontSize: 13 }}
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
          />
        </CardContent>
      </Card>

      {/* Save / Cancel */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        <button
          type="button"
          onClick={onSave}
          disabled={isPending}
          style={{
            padding: '7px 16px',
            borderRadius: 7,
            border: 'none',
            background: 'var(--accent)',
            color: 'var(--btn-accent-text, #000)',
            fontSize: 12,
            fontWeight: 600,
            cursor: isPending ? 'not-allowed' : 'pointer',
            opacity: isPending ? 0.7 : 1,
          }}
        >
          {isPending ? 'Saving…' : 'Save Changes'}
        </button>
        <Button variant="outline" size="sm" onClick={onCancel}>
          Cancel
        </Button>
      </div>
    </div>
  );
}

// ── Main Page ─────────────────────────────────────────────────────────────────

type TabId = 'overview' | 'endpoints' | 'sync' | 'license' | 'config';

const TABS: { id: TabId; label: string }[] = [
  { id: 'overview', label: 'Overview' },
  { id: 'endpoints', label: 'Endpoints' },
  { id: 'sync', label: 'Sync History' },
  { id: 'license', label: 'License' },
  { id: 'config', label: 'Configuration' },
];

export const ClientDetailPage = () => {
  const { id } = useParams<{ id: string }>();
  const { data: client, isLoading, isError } = useClient(id ?? '');
  const updateMutation = useUpdateClient();
  const approveMutation = useApproveClient();
  const suspendMutation = useSuspendClient();

  const [activeTab, setActiveTab] = useState<TabId>('overview');
  const [syncInterval, setSyncInterval] = useState<number | undefined>(undefined);
  const [notes, setNotes] = useState<string>('');
  // Track whether notes textarea has been touched
  const [notesTouched, setNotesTouched] = useState(false);

  const { data: licensesData } = useLicenses({ limit: 100, offset: 0 });

  if (isLoading) {
    return (
      <div className="p-6 space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-32 w-full" />
        <Skeleton className="h-48 w-full" />
      </div>
    );
  }

  if (isError || !client) {
    return (
      <div className="p-6">
        <Link
          to="/clients"
          className="text-sm hover:underline flex items-center gap-1 mb-4"
          style={{ color: 'var(--text-muted)' }}
        >
          <ArrowLeft className="h-4 w-4" /> Back to Clients
        </Link>
        <div className="text-sm" style={{ color: 'var(--signal-critical)' }}>
          Failed to load client.
        </div>
      </div>
    );
  }

  // narrowed — client is defined past the guard above
  const safeClient = client;
  const clientLicense = licensesData?.licenses.find((l) => l.client_id === safeClient.id);
  const usagePct = clientLicense
    ? Math.min(
        100,
        Math.round(
          ((clientLicense.client_endpoint_count ?? safeClient.endpoint_count) /
            clientLicense.max_endpoints) *
            100,
        ),
      )
    : 0;

  const avatar = getAvatar(safeClient.hostname);
  const healthScore = computeHealthScore(safeClient.last_sync_at);
  const connected = isConnected(safeClient);

  function handleSave() {
    updateMutation.mutate(
      {
        id: safeClient.id,
        data: {
          ...(syncInterval !== undefined ? { sync_interval: syncInterval } : {}),
          ...(notesTouched ? { notes: notes || undefined } : {}),
        },
      },
      {
        onSuccess: () => toast.success('Configuration saved'),
        onError: (err) =>
          toast.error(
            `Failed to save configuration: ${err instanceof Error ? err.message : 'Unknown error'}`,
          ),
      },
    );
  }

  function handleStatusToggle() {
    if (safeClient.status === 'approved') {
      suspendMutation.mutate(safeClient.id, {
        onSuccess: () => toast.success(`${safeClient.hostname} suspended`),
        onError: (err) =>
          toast.error(
            `Failed to suspend ${safeClient.hostname}: ${err instanceof Error ? err.message : 'Unknown error'}`,
          ),
      });
    } else {
      approveMutation.mutate(safeClient.id, {
        onSuccess: () => toast.success(`${safeClient.hostname} approved`),
        onError: (err) =>
          toast.error(
            `Failed to approve ${safeClient.hostname}: ${err instanceof Error ? err.message : 'Unknown error'}`,
          ),
      });
    }
  }

  return (
    <>
      <div style={{ padding: '20px 24px', display: 'flex', flexDirection: 'column', gap: 16 }}>
        {/* Identity Header — bare, no card, matches catalog detail style */}
        <div>
          <div
            style={{
              display: 'flex',
              alignItems: 'flex-start',
              justifyContent: 'space-between',
              gap: 16,
            }}
          >
            <div className="flex items-center gap-4">
              {/* Avatar */}
              <div
                className="w-16 h-16 rounded-2xl flex items-center justify-center text-foreground text-2xl font-bold flex-shrink-0"
                style={{ background: avatar.color }}
              >
                {avatar.letter}
              </div>

              <div>
                {/* Name + connection status + tier */}
                <div className="flex items-center gap-3 flex-wrap">
                  <h1 style={{ fontSize: 22, fontWeight: 600 }}>{safeClient.hostname}</h1>
                  <div className="flex items-center gap-1.5">
                    <span
                      className={`w-2.5 h-2.5 rounded-full inline-block ${connected ? 'animate-pulse' : ''}`}
                      style={{
                        background: connected ? 'var(--signal-healthy)' : 'var(--text-muted)',
                      }}
                    />
                    <span
                      className="text-sm font-medium"
                      style={{ color: connected ? 'var(--signal-healthy)' : 'var(--text-muted)' }}
                    >
                      {connected ? 'Connected' : 'Disconnected'}
                    </span>
                  </div>
                  {clientLicense ? (
                    <span
                      className="px-2 py-0.5 text-xs font-medium rounded border capitalize"
                      style={tierBadgeStyle(clientLicense.tier)}
                    >
                      {clientLicense.tier}
                    </span>
                  ) : (
                    <span
                      className="px-2 py-0.5 rounded-md text-xs font-semibold"
                      style={{ background: 'var(--bg-card-hover)', color: 'var(--text-muted)' }}
                    >
                      —
                    </span>
                  )}
                </div>

                {/* Meta row */}
                <div
                  className="flex flex-wrap items-center gap-4 mt-2 text-sm"
                  style={{ color: 'var(--text-muted)' }}
                >
                  <span>
                    PM{' '}
                    <span className="font-mono">
                      {safeClient.version ? `v${safeClient.version}` : '—'}
                    </span>
                  </span>
                  {safeClient.os && <span>{safeClient.os}</span>}
                  <span>
                    First connected:{' '}
                    <span style={{ color: 'var(--text-primary)' }}>
                      {new Date(safeClient.created_at).toLocaleDateString()}
                    </span>
                  </span>
                  <span>
                    Last sync:{' '}
                    <span style={{ color: 'var(--text-primary)' }}>
                      {formatRelativeTime(safeClient.last_sync_at)}
                    </span>
                  </span>
                </div>
              </div>
            </div>

            {/* Action buttons */}
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexShrink: 0 }}>
              <button
                type="button"
                disabled
                title="Hub-initiated sync not yet available"
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 6,
                  padding: '7px 14px',
                  borderRadius: 7,
                  border: '1px solid var(--border)',
                  background: 'none',
                  color: 'var(--text-secondary)',
                  fontSize: 12,
                  fontWeight: 500,
                  opacity: 0.5,
                  cursor: 'not-allowed',
                }}
              >
                Sync Now
              </button>
              <button
                type="button"
                onClick={() => setActiveTab('config')}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 6,
                  padding: '7px 14px',
                  borderRadius: 7,
                  border: '1px solid var(--border)',
                  background: 'none',
                  color: 'var(--text-secondary)',
                  fontSize: 12,
                  fontWeight: 500,
                  cursor: 'pointer',
                }}
              >
                Edit
              </button>
              <button
                type="button"
                onClick={() =>
                  suspendMutation.mutate(safeClient.id, {
                    onSuccess: () => toast.success('Client access revoked'),
                    onError: () => toast.error('Failed to revoke access'),
                  })
                }
                disabled={suspendMutation.isPending}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 6,
                  padding: '7px 14px',
                  borderRadius: 7,
                  border: '1px solid color-mix(in srgb, var(--signal-critical) 30%, transparent)',
                  background: 'none',
                  color: 'var(--signal-critical)',
                  fontSize: 12,
                  fontWeight: 500,
                  cursor: 'pointer',
                }}
              >
                Revoke Access
              </button>
            </div>
          </div>
        </div>

        {/* 6 Stat Cards */}
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fill, minmax(140px, 1fr))',
            gap: 12,
          }}
        >
          <StatCard label="Endpoints" value={safeClient.endpoint_count.toLocaleString()} />
          <StatCard label="Patch Coverage" value="—" sub="not tracked" />
          <StatCard label="Compliance" value="—" sub="not tracked" />
          <StatCard label="Catalog Synced" value="—" sub="not tracked" />
          <StatCard
            label="Sync Health"
            value={`${healthScore}%`}
            valueStyle={{ color: healthCssColor(healthScore) }}
            sub={healthScore >= 80 ? 'excellent' : healthScore >= 60 ? 'good' : 'degraded'}
          />
          <StatCard label="License Usage">
            {clientLicense ? (
              <>
                <div className="text-2xl font-bold">{usagePct}%</div>
                <div className="text-xs text-muted-foreground mt-1">
                  {(
                    clientLicense.client_endpoint_count ?? safeClient.endpoint_count
                  ).toLocaleString()}{' '}
                  / {clientLicense.max_endpoints.toLocaleString()}
                </div>
                <div
                  className="w-full rounded-full h-1 mt-2"
                  style={{ background: 'var(--bg-card-hover)' }}
                >
                  <div
                    className="h-1 rounded-full"
                    style={{ width: `${usagePct}%`, background: 'var(--accent)' }}
                  />
                </div>
              </>
            ) : (
              <>
                <div className="text-2xl font-bold">—</div>
                <div className="text-xs text-muted-foreground mt-1">no license</div>
              </>
            )}
          </StatCard>
        </div>

        {/* Tabs */}
        <div>
          {/* Tab bar */}
          <div
            style={{
              display: 'flex',
              gap: 0,
              borderBottom: '1px solid var(--border)',
            }}
          >
            {TABS.map((tab) => (
              <button
                key={tab.id}
                type="button"
                onClick={() => setActiveTab(tab.id)}
                style={{
                  padding: '8px 16px',
                  fontSize: 13,
                  fontWeight: activeTab === tab.id ? 600 : 400,
                  color: activeTab === tab.id ? 'var(--text-emphasis)' : 'var(--text-muted)',
                  border: 'none',
                  borderBottom:
                    activeTab === tab.id ? '2px solid var(--accent)' : '2px solid transparent',
                  background: 'transparent',
                  cursor: 'pointer',
                  transition: 'color 150ms ease',
                  marginBottom: -1,
                  whiteSpace: 'nowrap',
                }}
              >
                {tab.label}
              </button>
            ))}
          </div>

          {/* Tab panels */}
          <div style={{ paddingTop: 16 }}>
            {activeTab === 'overview' && <OverviewTab client={safeClient} />}
            {activeTab === 'endpoints' && <EndpointsTab client={safeClient} />}
            {activeTab === 'sync' && <SyncHistoryTab clientId={safeClient.id} />}
            {activeTab === 'license' && <LicenseTab client={safeClient} license={clientLicense} />}
            {activeTab === 'config' && (
              <ConfigTab
                client={safeClient}
                syncInterval={syncInterval}
                setSyncInterval={setSyncInterval}
                notes={notesTouched ? notes : (safeClient.notes ?? '')}
                setNotes={(v) => {
                  setNotes(v);
                  setNotesTouched(true);
                }}
                onSave={handleSave}
                onCancel={() => setActiveTab('overview')}
                isPending={updateMutation.isPending}
                clientStatus={safeClient.status}
                onStatusToggle={handleStatusToggle}
                statusTogglePending={approveMutation.isPending || suspendMutation.isPending}
              />
            )}
          </div>
        </div>
      </div>
    </>
  );
};
