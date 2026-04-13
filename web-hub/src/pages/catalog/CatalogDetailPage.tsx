import React, { useState } from 'react';
import { useParams, useNavigate } from 'react-router';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { toast } from 'sonner';
import {
  Skeleton,
  Input,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Button,
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@patchiq/ui';
import {
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  Legend,
  Area,
  Line,
  ComposedChart,
  CartesianGrid,
} from 'recharts';
import {
  useCatalogEntry,
  useUpdateCatalogEntry,
  useDeleteCatalogEntry,
} from '../../api/hooks/useCatalog';
import { SourceBadge } from '../../components/SourceBadge';
import { fmtDate, relativeTime } from '../../lib/format';
import type { CatalogSync, CVEFeed } from '../../types/catalog';

// ── Helpers ────────────────────────────────────────────────────────────────────

function cvssColor(score: number): string {
  if (score >= 9) return 'var(--signal-critical)';
  if (score >= 7) return 'var(--signal-warning)';
  if (score >= 4) return 'var(--signal-warning)';
  return 'var(--accent)';
}

function severityColor(severity: string): string {
  switch (severity?.toLowerCase()) {
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

function isoDate(ts: string | null | undefined): string {
  if (!ts) return '—';
  const d = new Date(ts);
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

const buildTimeline = (syncs: CatalogSync[], totalClients: number) => {
  const synced = syncs
    .filter((s) => s.status === 'synced' && s.synced_at)
    .sort((a, b) => new Date(a.synced_at!).getTime() - new Date(b.synced_at!).getTime());
  if (synced.length === 0) return [];

  const points: { date: string; pmsSynced: number; pending: number }[] = [];
  let cumulative = 0;

  const startDate = new Date(new Date(synced[0].synced_at!).getTime() - 86400000);
  points.push({
    date: startDate.toLocaleDateString('en-US', { month: 'short', day: 'numeric' }),
    pmsSynced: 0,
    pending: totalClients,
  });

  const byDate = new Map<string, number>();
  for (const s of synced) {
    const d = new Date(s.synced_at!).toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
    });
    byDate.set(d, (byDate.get(d) ?? 0) + 1);
  }
  for (const [date, count] of byDate) {
    cumulative += count;
    points.push({ date, pmsSynced: cumulative, pending: Math.max(0, totalClients - cumulative) });
  }

  const today = new Date().toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
  if (points[points.length - 1].date !== today) {
    points.push({
      date: today,
      pmsSynced: cumulative,
      pending: Math.max(0, totalClients - cumulative),
    });
  }
  return points;
};

async function downloadBinary(id: string, binaryRef: string) {
  const response = await fetch(`/api/v1/catalog/${id}/download`, { credentials: 'include' });
  if (!response.ok) throw new Error(`Download failed: ${response.status}`);
  const blob = await response.blob();
  const url = window.URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = binaryRef.split('/').pop() || `patch-${id}.bin`;
  document.body.appendChild(a);
  a.click();
  window.URL.revokeObjectURL(url);
  a.remove();
}

// ── Shared micro-components (identical to PatchDetailPage) ─────────────────────

function Chip({ children, color }: { children: React.ReactNode; color?: string }) {
  return (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        padding: '2px 8px',
        border: `1px solid ${color ? `${color}35` : 'var(--border)'}`,
        borderRadius: 4,
        fontSize: 11,
        fontFamily: 'var(--font-mono)',
        color: color ?? 'var(--text-secondary)',
        background: color ? `${color}0d` : 'transparent',
        whiteSpace: 'nowrap',
      }}
    >
      {children}
    </span>
  );
}

function HealthCell({
  label,
  value,
  valueColor,
  last,
}: {
  label: string;
  value: React.ReactNode;
  valueColor?: string;
  last?: boolean;
}) {
  return (
    <div
      style={{
        flex: 1,
        display: 'flex',
        alignItems: 'center',
        gap: 10,
        padding: '0 16px',
        borderRight: last ? 'none' : '1px solid var(--border)',
      }}
    >
      <div>
        <div
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 10,
            fontWeight: 600,
            textTransform: 'uppercase',
            letterSpacing: '0.06em',
            color: 'var(--text-muted)',
            marginBottom: 1,
          }}
        >
          {label}
        </div>
        <div
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 16,
            fontWeight: 700,
            color: valueColor ?? 'var(--text-emphasis)',
            lineHeight: 1.1,
          }}
        >
          {value}
        </div>
      </div>
    </div>
  );
}

function TabButton({
  label,
  active,
  onClick,
}: {
  label: string;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      style={{
        padding: '8px 16px',
        fontSize: 13,
        fontWeight: active ? 600 : 400,
        color: active ? 'var(--text-emphasis)' : 'var(--text-muted)',
        background: 'transparent',
        border: 'none',
        borderBottom: `2px solid ${active ? 'var(--accent)' : 'transparent'}`,
        marginBottom: -1,
        cursor: 'pointer',
        transition: 'color 150ms ease, border-color 150ms ease',
        whiteSpace: 'nowrap',
        outline: 'none',
      }}
      onMouseEnter={(e) => {
        if (!active) e.currentTarget.style.color = 'var(--text-primary)';
      }}
      onMouseLeave={(e) => {
        if (!active) e.currentTarget.style.color = 'var(--text-muted)';
      }}
    >
      {label}
    </button>
  );
}

// Dot grid showing synced/pending/failed PMs (mirrors EndpointDotGrid)
function SyncDotGrid({
  total,
  synced,
  pending,
}: {
  total: number;
  synced: number;
  pending: number;
}) {
  const count = Math.min(total, 80);
  const syncedDots = Math.round((synced / Math.max(1, total)) * count);
  const pendingDots = Math.round((pending / Math.max(1, total)) * count);
  const notPushedDots = count - syncedDots - pendingDots;
  const dots: Array<'synced' | 'pending' | 'not_pushed'> = [];
  for (let i = 0; i < syncedDots; i++) dots.push('synced');
  for (let i = 0; i < pendingDots; i++) dots.push('pending');
  for (let i = 0; i < Math.max(0, notPushedDots); i++) dots.push('not_pushed');
  const colorMap = {
    synced: 'var(--signal-healthy)',
    pending: 'var(--signal-warning)',
    not_pushed: 'var(--text-muted)',
  };
  return (
    <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4, maxWidth: 280 }}>
      {dots.map((state, i) => (
        <div
          key={i}
          title={state}
          style={{
            width: 8,
            height: 8,
            borderRadius: '50%',
            background: colorMap[state],
            opacity: state === 'not_pushed' ? 0.35 : 1,
            flexShrink: 0,
          }}
        />
      ))}
      {total > 80 && (
        <span
          style={{
            fontSize: 9,
            color: 'var(--text-muted)',
            fontFamily: 'var(--font-mono)',
            alignSelf: 'center',
          }}
        >
          +{total - 80}
        </span>
      )}
    </div>
  );
}

// CVSS breakdown panel (identical structure to PatchDetailPage's CvssBreakdown)
function CvssPanel({ cves }: { cves: CVEFeed[] }) {
  const topCve = cves.find((c) => c.cvss_v3_score != null) ?? cves[0];
  const score = topCve?.cvss_v3_score ?? 0;
  const color = cvssColor(score);

  return (
    <div
      style={{
        background: 'var(--bg-inset)',
        border: '1px solid var(--border)',
        borderRadius: 8,
        padding: '16px 20px',
        height: '100%',
      }}
    >
      <div
        style={{
          fontFamily: 'var(--font-mono)',
          fontSize: 10,
          fontWeight: 600,
          textTransform: 'uppercase',
          letterSpacing: '0.06em',
          color: 'var(--text-muted)',
          marginBottom: 16,
        }}
      >
        Highest CVSS Score
      </div>
      <div style={{ display: 'flex', alignItems: 'center', gap: 16, marginBottom: 20 }}>
        <div>
          <div
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 40,
              fontWeight: 800,
              color,
              lineHeight: 1,
              letterSpacing: '-0.04em',
            }}
          >
            {score.toFixed(1)}
          </div>
          <div style={{ fontSize: 11, color: 'var(--text-muted)', marginTop: 2 }}>/ 10</div>
        </div>
        <div style={{ flex: 1 }}>
          <div
            style={{
              height: 6,
              background: 'color-mix(in srgb, white 8%, transparent)',
              borderRadius: 3,
              overflow: 'hidden',
              marginBottom: 6,
            }}
          >
            <div
              style={{
                width: `${Math.min(100, score * 10)}%`,
                height: '100%',
                background: color,
                borderRadius: 3,
                transition: 'width 0.6s ease',
              }}
            />
          </div>
          <div style={{ fontSize: 11, color, fontWeight: 600 }}>
            {score >= 9 ? 'Critical' : score >= 7 ? 'High' : score >= 4 ? 'Medium' : 'Low'}
          </div>
          <div style={{ fontSize: 10, color: 'var(--text-muted)', marginTop: 4 }}>
            {topCve?.cve_id ?? '—'}
          </div>
        </div>
      </div>
      {/* CVE severity breakdown */}
      {cves.length > 0 && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          {(['critical', 'high', 'medium', 'low'] as const).map((sev) => {
            const cnt = cves.filter((c) => c.severity?.toLowerCase() === sev).length;
            const pct = cves.length > 0 ? Math.round((cnt / cves.length) * 100) : 0;
            const sevColor = severityColor(sev);
            return (
              <div key={sev} style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <div
                  style={{
                    fontSize: 10,
                    color: 'var(--text-secondary)',
                    width: 60,
                    flexShrink: 0,
                    textTransform: 'capitalize',
                  }}
                >
                  {sev}
                </div>
                <div
                  style={{
                    flex: 1,
                    height: 4,
                    background: 'color-mix(in srgb, white 8%, transparent)',
                    borderRadius: 2,
                    overflow: 'hidden',
                  }}
                >
                  <div
                    style={{
                      width: `${pct}%`,
                      height: '100%',
                      background: sevColor,
                      borderRadius: 2,
                    }}
                  />
                </div>
                <div
                  style={{
                    fontSize: 10,
                    color: sevColor,
                    fontWeight: 600,
                    width: 30,
                    textAlign: 'right',
                    flexShrink: 0,
                  }}
                >
                  {cnt}
                </div>
              </div>
            );
          })}
        </div>
      )}
      {cves.length === 0 && (
        <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>No CVEs linked.</div>
      )}
    </div>
  );
}

// Metadata panel (identical structure to PatchDetailPage's MetadataPanel)
function CatalogMetadataPanel({
  entry,
}: {
  entry: ReturnType<typeof useCatalogEntry>['data'] & object;
}) {
  const rows: Array<{ label: string; value: React.ReactNode }> = [
    { label: 'Vendor', value: entry.vendor || '—' },
    {
      label: 'OS Family',
      value: entry.os_family ? (
        <span style={{ textTransform: 'capitalize' }}>{entry.os_family}</span>
      ) : (
        '—'
      ),
    },
    { label: 'Version', value: entry.version || '—' },
    {
      label: 'Severity',
      value: entry.severity ? (
        <span style={{ textTransform: 'capitalize', color: severityColor(entry.severity) }}>
          {entry.severity}
        </span>
      ) : (
        '—'
      ),
    },
    { label: 'Released', value: isoDate(entry.release_date) },
    {
      label: 'SHA-256',
      value: entry.checksum_sha256 ? (
        <span style={{ fontSize: 10, wordBreak: 'break-all' }} title={entry.checksum_sha256}>
          {entry.checksum_sha256}
        </span>
      ) : (
        '—'
      ),
    },
    {
      label: 'Feed',
      value: <SourceBadge source={entry.feed_source_display_name ?? entry.feed_source_name} />,
    },
    { label: 'Last Updated', value: relativeTime(entry.updated_at) },
  ];

  return (
    <div
      style={{
        background: 'var(--bg-inset)',
        border: '1px solid var(--border)',
        borderRadius: 8,
        overflow: 'hidden',
      }}
    >
      <div
        style={{
          fontFamily: 'var(--font-mono)',
          fontSize: 10,
          fontWeight: 600,
          textTransform: 'uppercase',
          letterSpacing: '0.06em',
          color: 'var(--text-muted)',
          padding: '12px 16px',
          borderBottom: '1px solid var(--border)',
        }}
      >
        Catalog Metadata
      </div>
      <div>
        {rows.map(({ label, value }, i) => (
          <div
            key={label}
            style={{
              display: 'flex',
              alignItems: 'flex-start',
              padding: '9px 16px',
              borderBottom:
                i < rows.length - 1
                  ? '1px solid color-mix(in srgb, white 4%, transparent)'
                  : 'none',
              gap: 8,
            }}
          >
            <span style={{ fontSize: 11, color: 'var(--text-muted)', minWidth: 90, flexShrink: 0 }}>
              {label}
            </span>
            <span
              style={{
                fontSize: 12,
                color: 'var(--text-primary)',
                fontFamily: 'var(--font-mono)',
                minWidth: 0,
                wordBreak: 'break-word',
              }}
            >
              {value}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

// CVE table tab (mirrors CVEsTab from PatchDetailPage)
function CVEsTab({ cves }: { cves: CVEFeed[] }) {
  return (
    <div style={{ border: '1px solid var(--border)', borderRadius: 8, overflow: 'hidden' }}>
      <table style={{ width: '100%', borderCollapse: 'collapse' }}>
        <thead>
          <tr>
            {['CVE ID', 'CVSS', 'Severity', 'Exploit', 'KEV', 'Published', 'Description'].map(
              (h) => (
                <th
                  key={h}
                  style={{
                    padding: '8px 12px',
                    textAlign: 'left',
                    fontFamily: 'var(--font-mono)',
                    fontSize: 10,
                    fontWeight: 600,
                    textTransform: 'uppercase',
                    letterSpacing: '0.06em',
                    color: 'var(--text-muted)',
                    background: 'var(--bg-inset)',
                    borderBottom: '1px solid var(--border)',
                    whiteSpace: 'nowrap',
                  }}
                >
                  {h}
                </th>
              ),
            )}
          </tr>
        </thead>
        <tbody>
          {cves.map((cve) => {
            const score = cve.cvss_v3_score ?? 0;
            const color = cvssColor(score);
            return (
              <tr
                key={cve.id}
                style={{
                  borderBottom: '1px solid color-mix(in srgb, white 4%, transparent)',
                  transition: 'background 0.1s',
                }}
                onMouseEnter={(e) =>
                  (e.currentTarget.style.background = 'color-mix(in srgb, white 2%, transparent)')
                }
                onMouseLeave={(e) => (e.currentTarget.style.background = 'transparent')}
              >
                <td style={{ padding: '10px 12px' }}>
                  <span
                    style={{
                      fontFamily: 'var(--font-mono)',
                      fontSize: 12,
                      fontWeight: 600,
                      color: 'var(--accent)',
                    }}
                  >
                    {cve.cve_id}
                  </span>
                </td>
                <td style={{ padding: '10px 12px' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                    <div
                      style={{
                        width: 48,
                        height: 4,
                        background: 'color-mix(in srgb, white 8%, transparent)',
                        borderRadius: 2,
                        overflow: 'hidden',
                      }}
                    >
                      <div
                        style={{
                          width: `${Math.min(100, score * 10)}%`,
                          height: '100%',
                          background: color,
                          borderRadius: 2,
                        }}
                      />
                    </div>
                    <span
                      style={{
                        fontFamily: 'var(--font-mono)',
                        fontSize: 12,
                        fontWeight: 700,
                        color,
                      }}
                    >
                      {score.toFixed(1)}
                    </span>
                  </div>
                </td>
                <td style={{ padding: '10px 12px' }}>
                  <span
                    style={{
                      fontSize: 12,
                      fontWeight: 500,
                      color: severityColor(cve.severity ?? ''),
                      textTransform: 'capitalize',
                    }}
                  >
                    {cve.severity}
                  </span>
                </td>
                <td style={{ padding: '10px 12px' }}>
                  {cve.exploit_known ? (
                    <span
                      style={{
                        fontSize: 10,
                        fontWeight: 700,
                        color: 'var(--signal-critical)',
                        fontFamily: 'var(--font-mono)',
                      }}
                    >
                      Yes
                    </span>
                  ) : (
                    <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>No</span>
                  )}
                </td>
                <td style={{ padding: '10px 12px' }}>
                  {cve.in_kev ? (
                    <span
                      style={{
                        fontSize: 10,
                        fontWeight: 700,
                        color: 'var(--signal-warning)',
                        fontFamily: 'var(--font-mono)',
                      }}
                    >
                      Yes
                    </span>
                  ) : (
                    <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>No</span>
                  )}
                </td>
                <td style={{ padding: '10px 12px' }}>
                  <span
                    style={{
                      fontFamily: 'var(--font-mono)',
                      fontSize: 11,
                      color: 'var(--text-muted)',
                    }}
                  >
                    {isoDate(cve.published_at)}
                  </span>
                </td>
                <td style={{ padding: '10px 12px', maxWidth: 260 }}>
                  <span
                    style={{
                      fontSize: 11,
                      color: 'var(--text-muted)',
                      display: '-webkit-box',
                      WebkitLineClamp: 1,
                      WebkitBoxOrient: 'vertical',
                      overflow: 'hidden',
                    }}
                  >
                    {cve.description || '—'}
                  </span>
                </td>
              </tr>
            );
          })}
          {cves.length === 0 && (
            <tr>
              <td
                colSpan={7}
                style={{
                  padding: '48px 24px',
                  textAlign: 'center',
                  color: 'var(--text-muted)',
                  fontSize: 13,
                }}
              >
                No CVEs linked.
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}

// PM sync status table (mirrors EndpointsTab from PatchDetailPage)
function DistributionTab({ entry }: { entry: { syncs: CatalogSync[]; total_clients: number } }) {
  const timeline = buildTimeline(entry.syncs, entry.total_clients);
  const syncedCount = entry.syncs.filter((s) => s.status === 'synced').length;
  const pendingCount = entry.syncs.filter((s) => s.status === 'pending').length;
  const notPushedCount = entry.syncs.filter((s) => s.status === 'not_pushed').length;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* PM Sync Table */}
      <div style={{ border: '1px solid var(--border)', borderRadius: 8, overflow: 'hidden' }}>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            padding: '10px 16px',
            background: 'var(--bg-inset)',
            borderBottom: '1px solid var(--border)',
          }}
        >
          <span
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 10,
              fontWeight: 600,
              textTransform: 'uppercase',
              letterSpacing: '0.06em',
              color: 'var(--text-muted)',
            }}
          >
            PM Sync Status ({syncedCount}/{entry.total_clients} synced)
          </span>
          {notPushedCount > 0 && (
            <button
              type="button"
              onClick={() =>
                toast.info(
                  `${notPushedCount} Patch Manager(s) will sync this entry on their next scheduled poll.`,
                )
              }
              style={{
                padding: '4px 10px',
                fontSize: 11,
                fontWeight: 500,
                borderRadius: 5,
                border: '1px solid var(--border)',
                background: 'transparent',
                color: 'var(--text-secondary)',
                cursor: 'pointer',
                transition: 'all 0.15s',
              }}
              onMouseEnter={(e) => (e.currentTarget.style.color = 'var(--text-primary)')}
              onMouseLeave={(e) => (e.currentTarget.style.color = 'var(--text-secondary)')}
            >
              {notPushedCount} PM{notPushedCount !== 1 ? 's' : ''} pending sync
            </button>
          )}
        </div>
        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
          <thead>
            <tr>
              {['Client', 'Status', 'Synced At', 'Endpoints'].map((h) => (
                <th
                  key={h}
                  style={{
                    padding: '8px 12px',
                    textAlign: 'left',
                    fontFamily: 'var(--font-mono)',
                    fontSize: 10,
                    fontWeight: 600,
                    textTransform: 'uppercase',
                    letterSpacing: '0.06em',
                    color: 'var(--text-muted)',
                    background: 'var(--bg-inset)',
                    borderBottom: '1px solid var(--border)',
                    whiteSpace: 'nowrap',
                  }}
                >
                  {h}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {entry.syncs.map((sync) => {
              const statusColor =
                sync.status === 'synced'
                  ? 'var(--signal-healthy)'
                  : sync.status === 'pending'
                    ? 'var(--signal-warning)'
                    : 'var(--text-muted)';
              return (
                <tr
                  key={sync.id}
                  style={{
                    borderBottom: '1px solid color-mix(in srgb, white 4%, transparent)',
                    transition: 'background 0.1s',
                  }}
                  onMouseEnter={(e) =>
                    (e.currentTarget.style.background = 'color-mix(in srgb, white 2%, transparent)')
                  }
                  onMouseLeave={(e) => (e.currentTarget.style.background = 'transparent')}
                >
                  <td style={{ padding: '10px 12px' }}>
                    <span
                      style={{
                        fontFamily: 'var(--font-mono)',
                        fontSize: 12,
                        fontWeight: 600,
                        color: 'var(--text-primary)',
                      }}
                    >
                      {sync.client_name}
                    </span>
                  </td>
                  <td style={{ padding: '10px 12px' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 5 }}>
                      <div
                        style={{
                          width: 6,
                          height: 6,
                          borderRadius: '50%',
                          background: statusColor,
                          flexShrink: 0,
                        }}
                      />
                      <span
                        style={{
                          fontSize: 12,
                          color: statusColor,
                          fontWeight: 500,
                          textTransform: 'capitalize',
                        }}
                      >
                        {sync.status === 'synced'
                          ? 'Synced'
                          : sync.status === 'pending'
                            ? 'Pending'
                            : 'Not Pushed'}
                      </span>
                    </div>
                  </td>
                  <td style={{ padding: '10px 12px' }}>
                    <span
                      style={{
                        fontFamily: 'var(--font-mono)',
                        fontSize: 11,
                        color: 'var(--text-muted)',
                      }}
                    >
                      {sync.synced_at ? isoDate(sync.synced_at) : '—'}
                    </span>
                  </td>
                  <td style={{ padding: '10px 12px' }}>
                    <span
                      style={{
                        fontFamily: 'var(--font-mono)',
                        fontSize: 12,
                        color: 'var(--text-primary)',
                      }}
                    >
                      {sync.endpoint_count ?? 0}
                    </span>
                  </td>
                </tr>
              );
            })}
            {entry.syncs.length === 0 && (
              <tr>
                <td
                  colSpan={4}
                  style={{
                    padding: '48px 24px',
                    textAlign: 'center',
                    color: 'var(--text-muted)',
                    fontSize: 13,
                  }}
                >
                  No Patch Managers have synced this entry yet.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {/* Distribution Timeline */}
      <div
        style={{
          background: 'var(--bg-inset)',
          border: '1px solid var(--border)',
          borderRadius: 8,
          padding: '16px',
        }}
      >
        <div
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 10,
            fontWeight: 600,
            textTransform: 'uppercase',
            letterSpacing: '0.06em',
            color: 'var(--text-muted)',
            marginBottom: 16,
          }}
        >
          Distribution Timeline — PMs Synced Over Time
        </div>
        {timeline.length < 2 ? (
          <p style={{ fontSize: 12, color: 'var(--text-muted)', margin: 0 }}>
            Not enough sync data to show a timeline.
          </p>
        ) : (
          <div style={{ width: '100%', height: 260 }}>
            <ResponsiveContainer>
              <ComposedChart data={timeline}>
                <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
                <XAxis dataKey="date" tick={{ fill: 'var(--text-muted)', fontSize: 11 }} />
                <YAxis
                  tick={{ fill: 'var(--text-muted)', fontSize: 11 }}
                  domain={[0, entry.total_clients || 'auto']}
                  allowDecimals={false}
                />
                <Tooltip
                  contentStyle={{
                    background: 'var(--bg-card)',
                    border: '1px solid var(--border)',
                    borderRadius: 6,
                    fontSize: 12,
                    color: 'var(--text-primary)',
                  }}
                />
                <Legend />
                <Area
                  type="monotone"
                  dataKey="pmsSynced"
                  name="PMs Synced"
                  stroke="var(--accent)"
                  fill="color-mix(in srgb, var(--accent) 15%, transparent)"
                  strokeWidth={2}
                />
                <Line
                  type="monotone"
                  dataKey="pending"
                  name="Pending"
                  stroke="var(--signal-warning)"
                  strokeWidth={2}
                  strokeDasharray="8 4"
                  dot={{ fill: 'var(--signal-warning)', r: 3 }}
                />
              </ComposedChart>
            </ResponsiveContainer>
          </div>
        )}
        {entry.total_clients > 0 && (
          <div style={{ display: 'flex', gap: 16, marginTop: 12 }}>
            {[
              { label: 'Synced', value: syncedCount, color: 'var(--signal-healthy)' },
              { label: 'Pending', value: pendingCount, color: 'var(--signal-warning)' },
              { label: 'Not Pushed', value: notPushedCount, color: 'var(--text-muted)' },
            ].map(({ label, value, color }) => (
              <div key={label} style={{ display: 'flex', alignItems: 'center', gap: 5 }}>
                <div
                  style={{
                    width: 6,
                    height: 6,
                    borderRadius: '50%',
                    background: color,
                    flexShrink: 0,
                  }}
                />
                <span style={{ fontSize: 11, color: 'var(--text-secondary)' }}>{label}</span>
                <span
                  style={{
                    fontFamily: 'var(--font-mono)',
                    fontSize: 12,
                    fontWeight: 600,
                    color,
                    marginLeft: 2,
                  }}
                >
                  {value}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

// ── Main Page ─────────────────────────────────────────────────────────────────

type CatalogTab = 'overview' | 'cves' | 'distribution' | 'source';

// ── Edit Form Schema ──────────────────────────────────────────────────────────

const editSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  vendor: z.string().min(1, 'Vendor is required'),
  os_family: z.string().min(1, 'OS family is required'),
  version: z.string().min(1, 'Version is required'),
  severity: z.enum(['critical', 'high', 'medium', 'low', 'none']),
  description: z.string().optional(),
  installer_type: z.string().optional(),
  cve_ids: z.string().optional(),
});

type EditFormData = z.infer<typeof editSchema>;

export function CatalogDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { data: entry, isLoading, isError } = useCatalogEntry(id ?? '');
  const updateMutation = useUpdateCatalogEntry();
  const deleteMutation = useDeleteCatalogEntry();
  const [activeTab, setActiveTab] = useState<CatalogTab>('overview');
  const [editOpen, setEditOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);

  if (isLoading) {
    return (
      <div style={{ padding: 24, display: 'flex', flexDirection: 'column', gap: 12 }}>
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-14 rounded-xl" />
        <Skeleton className="h-10 rounded-xl" />
        <Skeleton className="h-[300px] rounded-xl" />
      </div>
    );
  }

  if (isError || !entry) {
    return (
      <div style={{ padding: 24 }}>
        <div
          style={{
            borderRadius: 8,
            border: '1px solid color-mix(in srgb, var(--signal-critical) 30%, transparent)',
            background: 'color-mix(in srgb, var(--signal-critical) 10%, transparent)',
            padding: 16,
            fontSize: 13,
            color: 'var(--signal-critical)',
          }}
        >
          Failed to load catalog entry.
        </div>
      </div>
    );
  }

  const highestCVSS =
    entry.cves.length > 0 ? Math.max(...entry.cves.map((c) => c.cvss_v3_score ?? 0)) : null;
  const cvssCol = highestCVSS !== null ? cvssColor(highestCVSS) : 'var(--text-muted)';
  const totalEndpoints = entry.syncs.reduce((sum, s) => sum + (s.endpoint_count ?? 0), 0);
  const syncedCount = entry.syncs.filter((s) => s.status === 'synced').length;
  const pendingCount = entry.syncs.filter((s) => s.status === 'pending').length;
  const syncedPct =
    entry.total_clients > 0 ? Math.round((syncedCount / entry.total_clients) * 100) : 0;

  const tabs: { id: CatalogTab; label: string }[] = [
    { id: 'overview', label: 'Overview' },
    { id: 'cves', label: `CVEs${entry.cves.length ? ` (${entry.cves.length})` : ''}` },
    {
      id: 'distribution',
      label: `Distribution${entry.total_clients ? ` (${entry.total_clients})` : ''}`,
    },
    { id: 'source', label: 'Source' },
  ];

  return (
    <div
      style={{
        padding: '24px',
        display: 'flex',
        flexDirection: 'column',
        gap: 0,
        minHeight: '100%',
        background: 'var(--bg-page)',
      }}
    >
      {/* Page Header */}
      <div style={{ marginBottom: 12 }}>
        {/* Row 1: title + actions */}
        <div
          style={{
            display: 'flex',
            alignItems: 'flex-start',
            justifyContent: 'space-between',
            gap: 16,
            marginBottom: 8,
          }}
        >
          <div style={{ flex: 1, minWidth: 0 }}>
            <h1
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 26,
                fontWeight: 800,
                color: 'var(--text-emphasis)',
                margin: 0,
                lineHeight: 1.15,
                letterSpacing: '-0.02em',
              }}
            >
              {entry.name}
            </h1>
          </div>
          <div style={{ display: 'flex', gap: 8, flexShrink: 0 }}>
            <button
              type="button"
              onClick={() => {
                toast.info('Patch Managers will pull this entry on their next sync cycle.', {
                  description: `Entry: ${entry.name}`,
                });
              }}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 6,
                padding: '7px 14px',
                fontSize: 12,
                fontWeight: 600,
                borderRadius: 6,
                border: 'none',
                background: 'var(--accent)',
                color: 'var(--btn-accent-text, #000)',
                cursor: 'pointer',
                transition: 'opacity 0.15s',
              }}
              onMouseEnter={(e) => (e.currentTarget.style.opacity = '0.85')}
              onMouseLeave={(e) => (e.currentTarget.style.opacity = '1')}
            >
              Push to PMs
            </button>
            <button
              type="button"
              onClick={() => setEditOpen(true)}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 6,
                padding: '7px 14px',
                fontSize: 12,
                fontWeight: 500,
                borderRadius: 6,
                border: '1px solid var(--border)',
                background: 'transparent',
                color: 'var(--text-secondary)',
                cursor: 'pointer',
                transition: 'all 0.15s',
              }}
              onMouseEnter={(e) => {
                e.currentTarget.style.borderColor = 'var(--border-hover)';
                e.currentTarget.style.color = 'var(--text-primary)';
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.borderColor = 'var(--border)';
                e.currentTarget.style.color = 'var(--text-secondary)';
              }}
            >
              Edit
            </button>
            <button
              type="button"
              onClick={() => setDeleteOpen(true)}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 6,
                padding: '7px 14px',
                fontSize: 12,
                fontWeight: 500,
                borderRadius: 6,
                border: '1px solid color-mix(in srgb, var(--signal-critical) 35%, transparent)',
                background: 'transparent',
                color: 'var(--signal-critical)',
                cursor: 'pointer',
                transition: 'all 0.15s',
              }}
            >
              Remove
            </button>
          </div>
        </div>

        {/* Row 2: severity + chips */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 5 }}>
            <div
              style={{
                width: 7,
                height: 7,
                borderRadius: '50%',
                background: severityColor(entry.severity),
                flexShrink: 0,
              }}
            />
            <span
              style={{
                fontSize: 13,
                fontWeight: 600,
                color: severityColor(entry.severity),
                textTransform: 'capitalize',
              }}
            >
              {entry.severity}
            </span>
          </div>
          <span style={{ color: 'var(--border)', fontSize: 10 }}>·</span>
          <Chip>{entry.vendor}</Chip>
          <Chip>{entry.os_family}</Chip>
          {entry.version && <Chip>v{entry.version}</Chip>}
          <Chip>Published {fmtDate(entry.release_date)}</Chip>
          <SourceBadge source={entry.feed_source_name} />
        </div>
      </div>

      {/* Health Strip */}
      <div
        style={{
          height: 56,
          background: 'var(--bg-card)',
          border: '1px solid var(--border)',
          borderRadius: 8,
          display: 'flex',
          alignItems: 'center',
          marginBottom: 16,
          overflow: 'hidden',
          boxShadow: 'var(--shadow-sm)',
        }}
      >
        <HealthCell
          label="CVEs Linked"
          value={entry.cves?.length ?? entry.cve_count ?? 0}
          valueColor={
            (entry.cves?.length ?? 0) > 0 ? 'var(--signal-critical)' : 'var(--text-emphasis)'
          }
        />
        <HealthCell
          label="Highest CVSS"
          value={
            highestCVSS !== null ? (
              <span>
                <span style={{ color: cvssCol }}>{highestCVSS.toFixed(1)}</span>
                <span style={{ fontSize: 12, color: 'var(--text-muted)', fontWeight: 400 }}>
                  /10
                </span>
              </span>
            ) : (
              '—'
            )
          }
          valueColor={cvssCol}
        />
        <HealthCell
          label="PMs Synced"
          value={`${syncedCount}/${entry.total_clients}`}
          valueColor={syncedCount > 0 ? 'var(--signal-healthy)' : 'var(--text-muted)'}
        />
        <HealthCell
          label="Endpoints Affected"
          value={totalEndpoints.toLocaleString()}
          valueColor={totalEndpoints > 0 ? 'var(--signal-warning)' : 'var(--text-emphasis)'}
        />
        <HealthCell label="Last Updated" value={relativeTime(entry.updated_at)} last />
      </div>

      {/* Tabs */}
      <div
        style={{
          display: 'flex',
          borderBottom: '1px solid var(--border)',
          marginBottom: 16,
        }}
      >
        {tabs.map((tab) => (
          <TabButton
            key={tab.id}
            label={tab.label}
            active={activeTab === tab.id}
            onClick={() => setActiveTab(tab.id)}
          />
        ))}
      </div>

      {/* Tab: Overview — 3fr / 2fr grid matching PatchDetailPage */}
      {activeTab === 'overview' && (
        <div style={{ display: 'grid', gridTemplateColumns: '3fr 2fr', gap: 16 }}>
          {/* LEFT: PM Sync Overview + CVSS Panel */}
          <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
            {/* PM Sync Overview — mirrors Endpoint Exposure card */}
            <div
              style={{
                background: 'var(--bg-card)',
                border: '1px solid var(--border)',
                borderRadius: 8,
                padding: '16px 20px',
                boxShadow: 'var(--shadow-sm)',
              }}
            >
              <div
                style={{
                  fontFamily: 'var(--font-mono)',
                  fontSize: 10,
                  fontWeight: 600,
                  textTransform: 'uppercase',
                  letterSpacing: '0.06em',
                  color: 'var(--text-muted)',
                  marginBottom: 12,
                }}
              >
                PM Sync Overview
              </div>
              <div style={{ display: 'flex', alignItems: 'flex-start', gap: 24 }}>
                <div>
                  <div
                    style={{
                      fontFamily: 'var(--font-mono)',
                      fontSize: 36,
                      fontWeight: 800,
                      color: syncedCount > 0 ? 'var(--signal-healthy)' : 'var(--text-muted)',
                      lineHeight: 1,
                      letterSpacing: '-0.03em',
                      marginBottom: 4,
                    }}
                  >
                    {syncedCount}
                  </div>
                  <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>
                    of {entry.total_clients} PMs synced
                  </div>
                  {entry.total_clients > 0 && (
                    <div
                      style={{ marginTop: 10, display: 'flex', flexDirection: 'column', gap: 4 }}
                    >
                      {[
                        { label: 'Synced', val: syncedCount, color: 'var(--signal-healthy)' },
                        { label: 'Pending', val: pendingCount, color: 'var(--signal-warning)' },
                        {
                          label: 'Not Pushed',
                          val: Math.max(0, entry.total_clients - entry.syncs.length),
                          color: 'var(--text-muted)',
                        },
                      ].map(({ label, val, color }) => (
                        <div key={label} style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                          <div
                            style={{
                              width: 6,
                              height: 6,
                              borderRadius: '50%',
                              background: color,
                              flexShrink: 0,
                            }}
                          />
                          <span style={{ fontSize: 11, color: 'var(--text-secondary)' }}>
                            {label}
                          </span>
                          <span
                            style={{
                              fontFamily: 'var(--font-mono)',
                              fontSize: 12,
                              fontWeight: 600,
                              color,
                              marginLeft: 'auto',
                            }}
                          >
                            {val}
                          </span>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
                {entry.total_clients > 0 && (
                  <div style={{ flex: 1 }}>
                    <SyncDotGrid
                      total={entry.total_clients}
                      synced={syncedCount}
                      pending={pendingCount}
                    />
                    <div style={{ marginTop: 10 }}>
                      <div
                        style={{
                          height: 5,
                          background: 'color-mix(in srgb, white 8%, transparent)',
                          borderRadius: 3,
                          overflow: 'hidden',
                          marginBottom: 4,
                        }}
                      >
                        <div
                          style={{
                            width: `${syncedPct}%`,
                            height: '100%',
                            background: 'var(--signal-healthy)',
                            borderRadius: 3,
                            transition: 'width 0.6s ease',
                          }}
                        />
                      </div>
                      <div
                        style={{
                          fontSize: 10,
                          color: 'var(--signal-healthy)',
                          fontFamily: 'var(--font-mono)',
                        }}
                      >
                        {syncedPct}% synced
                      </div>
                    </div>
                  </div>
                )}
              </div>
            </div>

            {/* CVSS Breakdown */}
            <CvssPanel cves={entry.cves} />
          </div>

          {/* RIGHT: Catalog Metadata */}
          <div>
            <CatalogMetadataPanel entry={entry} />
            {entry.description && (
              <div
                style={{
                  background: 'var(--bg-card)',
                  border: '1px solid var(--border)',
                  borderRadius: 8,
                  padding: '14px 16px',
                  marginTop: 12,
                  boxShadow: 'var(--shadow-sm)',
                }}
              >
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
                  Description
                </div>
                <p
                  style={{
                    fontSize: 12,
                    color: 'var(--text-secondary)',
                    lineHeight: 1.65,
                    margin: 0,
                  }}
                >
                  {entry.description}
                </p>
              </div>
            )}
            {entry.binary_ref && (
              <div
                style={{
                  background: 'var(--bg-card)',
                  border: '1px solid var(--border)',
                  borderRadius: 8,
                  padding: '14px 16px',
                  marginTop: 12,
                  boxShadow: 'var(--shadow-sm)',
                }}
              >
                <div
                  style={{
                    fontFamily: 'var(--font-mono)',
                    fontSize: 10,
                    fontWeight: 600,
                    textTransform: 'uppercase',
                    letterSpacing: '0.06em',
                    color: 'var(--text-muted)',
                    marginBottom: 10,
                  }}
                >
                  Binary
                </div>
                <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                  <span
                    style={{
                      display: 'inline-flex',
                      padding: '2px 8px',
                      borderRadius: 4,
                      fontSize: 11,
                      fontFamily: 'var(--font-mono)',
                      background: 'color-mix(in srgb, var(--signal-healthy) 12%, transparent)',
                      color: 'var(--signal-healthy)',
                      border:
                        '1px solid color-mix(in srgb, var(--signal-healthy) 30%, transparent)',
                      width: 'fit-content',
                    }}
                  >
                    Binary Available
                  </span>
                  <button
                    type="button"
                    onClick={() => downloadBinary(entry.id, entry.binary_ref)}
                    style={{
                      padding: '5px 12px',
                      borderRadius: 5,
                      border: '1px solid var(--border)',
                      background: 'transparent',
                      color: 'var(--text-secondary)',
                      fontSize: 11,
                      fontWeight: 500,
                      cursor: 'pointer',
                      width: 'fit-content',
                      transition: 'all 0.15s',
                    }}
                    onMouseEnter={(e) => (e.currentTarget.style.color = 'var(--text-primary)')}
                    onMouseLeave={(e) => (e.currentTarget.style.color = 'var(--text-secondary)')}
                  >
                    Download Binary
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Tab: CVEs */}
      {activeTab === 'cves' && <CVEsTab cves={entry.cves} />}

      {/* Tab: Distribution */}
      {activeTab === 'distribution' && <DistributionTab entry={entry} />}

      {/* Tab: Source */}
      {activeTab === 'source' && (
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
          {/* Feed Origin */}
          <div
            style={{
              background: 'var(--bg-inset)',
              border: '1px solid var(--border)',
              borderRadius: 8,
              overflow: 'hidden',
            }}
          >
            <div
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 10,
                fontWeight: 600,
                textTransform: 'uppercase',
                letterSpacing: '0.06em',
                color: 'var(--text-muted)',
                padding: '12px 16px',
                borderBottom: '1px solid var(--border)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
              }}
            >
              Feed Origin
              <button
                type="button"
                style={{
                  padding: '3px 9px',
                  fontSize: 11,
                  fontWeight: 500,
                  borderRadius: 4,
                  border: '1px solid var(--border)',
                  background: 'transparent',
                  color: 'var(--text-secondary)',
                  cursor: 'pointer',
                  transition: 'all 0.15s',
                  textTransform: 'none',
                  letterSpacing: 'normal',
                  fontFamily: 'var(--font-sans)',
                }}
                onMouseEnter={(e) => (e.currentTarget.style.color = 'var(--text-primary)')}
                onMouseLeave={(e) => (e.currentTarget.style.color = 'var(--text-secondary)')}
              >
                Re-fetch from Source
              </button>
            </div>
            <div>
              {[
                {
                  label: 'Feed Source',
                  value: (
                    <SourceBadge
                      source={entry.feed_source_display_name ?? entry.feed_source_name}
                    />
                  ),
                },
                {
                  label: 'Feed URL',
                  value: (
                    <span
                      style={{
                        fontFamily: 'var(--font-mono)',
                        fontSize: 10,
                        color: 'var(--accent)',
                        wordBreak: 'break-all',
                      }}
                    >
                      {entry.source_url || '—'}
                    </span>
                  ),
                },
                { label: 'Last Refreshed', value: fmtDate(entry.updated_at) },
                {
                  label: 'Entry ID',
                  value: (
                    <span
                      style={{
                        fontFamily: 'var(--font-mono)',
                        fontSize: 10,
                        color: 'var(--text-muted)',
                        wordBreak: 'break-all',
                      }}
                    >
                      {entry.id}
                    </span>
                  ),
                },
              ].map(({ label, value }, i, arr) => (
                <div
                  key={label}
                  style={{
                    display: 'flex',
                    alignItems: 'flex-start',
                    padding: '9px 16px',
                    borderBottom:
                      i < arr.length - 1
                        ? '1px solid color-mix(in srgb, white 4%, transparent)'
                        : 'none',
                    gap: 8,
                  }}
                >
                  <span
                    style={{
                      fontSize: 11,
                      color: 'var(--text-muted)',
                      minWidth: 90,
                      flexShrink: 0,
                    }}
                  >
                    {label}
                  </span>
                  <span
                    style={{
                      fontSize: 12,
                      color: 'var(--text-primary)',
                      fontFamily: 'var(--font-mono)',
                    }}
                  >
                    {value}
                  </span>
                </div>
              ))}
            </div>
          </div>

          {/* Raw Feed Entry */}
          <div
            style={{
              background: 'var(--bg-inset)',
              border: '1px solid var(--border)',
              borderRadius: 8,
              overflow: 'hidden',
            }}
          >
            <div
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 10,
                fontWeight: 600,
                textTransform: 'uppercase',
                letterSpacing: '0.06em',
                color: 'var(--text-muted)',
                padding: '12px 16px',
                borderBottom: '1px solid var(--border)',
              }}
            >
              Raw Feed Entry
            </div>
            <div style={{ padding: '14px 16px' }}>
              <pre
                style={{
                  overflowX: 'auto',
                  borderRadius: 6,
                  background: 'color-mix(in srgb, white 3%, transparent)',
                  border: '1px solid var(--border)',
                  padding: '12px',
                  fontFamily: 'var(--font-mono)',
                  fontSize: 11,
                  color: 'var(--text-primary)',
                  maxHeight: 320,
                  overflowY: 'auto',
                  lineHeight: 1.6,
                  margin: 0,
                }}
              >
                {JSON.stringify(
                  {
                    name: entry.name,
                    vendor: entry.vendor,
                    os_family: entry.os_family,
                    version: entry.version,
                    severity: entry.severity,
                    release_date: entry.release_date,
                    description: entry.description,
                    source_url: entry.source_url,
                    cves: entry.cves.map((c) => c.cve_id),
                  },
                  null,
                  2,
                )}
              </pre>
            </div>
          </div>
        </div>
      )}

      {/* Edit Dialog */}
      {editOpen && (
        <EditCatalogDialog
          entry={entry}
          open={editOpen}
          onOpenChange={setEditOpen}
          onSave={(data) => {
            updateMutation.mutate(
              { id: entry.id, data },
              {
                onSuccess: () => {
                  toast.success('Catalog entry updated');
                  setEditOpen(false);
                },
                onError: (err) => {
                  toast.error(err instanceof Error ? err.message : 'Failed to update entry');
                },
              },
            );
          }}
          isPending={updateMutation.isPending}
        />
      )}

      {/* Delete Confirmation Dialog */}
      <Dialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>Remove Catalog Entry</DialogTitle>
          </DialogHeader>
          <p style={{ fontSize: 13, color: 'var(--text-secondary)', lineHeight: 1.6 }}>
            Are you sure you want to remove{' '}
            <span style={{ fontWeight: 600, color: 'var(--text-emphasis)' }}>{entry.name}</span>?
            This will delete it from the hub catalog. Patch Managers that already synced this entry
            will not be affected.
          </p>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteOpen(false)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              disabled={deleteMutation.isPending}
              onClick={() => {
                deleteMutation.mutate(entry.id, {
                  onSuccess: () => {
                    toast.success('Catalog entry removed');
                    setDeleteOpen(false);
                    void navigate('/catalog');
                  },
                  onError: (err) => {
                    toast.error(err instanceof Error ? err.message : 'Failed to remove entry');
                    setDeleteOpen(false);
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

// ── Edit Dialog ──────────────────────────────────────────────────────────────

interface EditCatalogDialogProps {
  entry: {
    name: string;
    vendor: string;
    os_family: string;
    version: string;
    severity: string;
    description: string | null;
    installer_type: string;
    cves: { cve_id: string }[];
  };
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSave: (data: {
    name: string;
    vendor: string;
    os_family: string;
    version: string;
    severity: string;
    description?: string;
    installer_type?: string;
    cve_ids?: string[];
  }) => void;
  isPending: boolean;
}

function EditCatalogDialog({
  entry,
  open,
  onOpenChange,
  onSave,
  isPending,
}: EditCatalogDialogProps) {
  const {
    register,
    handleSubmit,
    setValue,
    watch,
    formState: { errors },
  } = useForm<EditFormData>({
    resolver: zodResolver(editSchema),
    defaultValues: {
      name: entry.name,
      vendor: entry.vendor,
      os_family: entry.os_family,
      version: entry.version,
      severity: entry.severity as EditFormData['severity'],
      description: entry.description ?? '',
      installer_type: entry.installer_type ?? '',
      cve_ids: entry.cves.map((c) => c.cve_id).join(', '),
    },
  });

  const osFamily = watch('os_family');
  const severityValue = watch('severity');

  const onSubmit = (data: EditFormData) => {
    const cveIdList = data.cve_ids
      ? data.cve_ids
          .split(',')
          .map((s) => s.trim())
          .filter(Boolean)
      : [];

    onSave({
      name: data.name,
      vendor: data.vendor,
      os_family: data.os_family,
      version: data.version,
      severity: data.severity,
      description: data.description || undefined,
      installer_type: data.installer_type || undefined,
      cve_ids: cveIdList.length > 0 ? cveIdList : undefined,
    });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Edit Catalog Entry</DialogTitle>
        </DialogHeader>
        <form onSubmit={(e) => void handleSubmit(onSubmit)(e)} className="space-y-3">
          <div>
            <label className="text-sm font-medium">Name</label>
            <Input {...register('name')} placeholder="e.g. KB5001234" />
            {errors.name && <p className="text-sm text-destructive mt-1">{errors.name.message}</p>}
          </div>

          <div>
            <label className="text-sm font-medium">Vendor</label>
            <Input {...register('vendor')} placeholder="e.g. Microsoft, Canonical" />
            {errors.vendor && (
              <p className="text-sm text-destructive mt-1">{errors.vendor.message}</p>
            )}
          </div>

          <div>
            <label className="text-sm font-medium">OS Family</label>
            <Select value={osFamily ?? ''} onValueChange={(v: string) => setValue('os_family', v)}>
              <SelectTrigger>
                <SelectValue placeholder="Select OS family" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="windows">Windows</SelectItem>
                <SelectItem value="linux">Linux</SelectItem>
                <SelectItem value="macos">macOS</SelectItem>
                <SelectItem value="ubuntu">Ubuntu</SelectItem>
                <SelectItem value="debian">Debian</SelectItem>
                <SelectItem value="rhel">RHEL</SelectItem>
              </SelectContent>
            </Select>
            {errors.os_family && (
              <p className="text-sm text-destructive mt-1">{errors.os_family.message}</p>
            )}
          </div>

          <div>
            <label className="text-sm font-medium">Version</label>
            <Input {...register('version')} placeholder="e.g. 1.0.0" />
            {errors.version && (
              <p className="text-sm text-destructive mt-1">{errors.version.message}</p>
            )}
          </div>

          <div>
            <label className="text-sm font-medium">Severity</label>
            <Select
              value={severityValue ?? ''}
              onValueChange={(v: string) => setValue('severity', v as EditFormData['severity'])}
            >
              <SelectTrigger>
                <SelectValue placeholder="Select severity" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="critical">Critical</SelectItem>
                <SelectItem value="high">High</SelectItem>
                <SelectItem value="medium">Medium</SelectItem>
                <SelectItem value="low">Low</SelectItem>
                <SelectItem value="none">None</SelectItem>
              </SelectContent>
            </Select>
            {errors.severity && (
              <p className="text-sm text-destructive mt-1">{errors.severity.message}</p>
            )}
          </div>

          <div>
            <label className="text-sm font-medium">Installer Type</label>
            <Input {...register('installer_type')} placeholder="e.g. msi, deb, rpm" />
          </div>

          <div>
            <label className="text-sm font-medium">CVE IDs</label>
            <Input {...register('cve_ids')} placeholder="e.g. CVE-2024-1234, CVE-2024-5678" />
            <p className="text-xs text-muted-foreground mt-1">Comma-separated list of CVE IDs</p>
          </div>

          <div>
            <label className="text-sm font-medium">Description</label>
            <textarea
              {...register('description')}
              className="flex min-h-[60px] w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              placeholder="Optional description"
            />
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={isPending}>
              {isPending ? 'Saving...' : 'Save Changes'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
