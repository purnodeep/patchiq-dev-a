import React, { useState } from 'react';
import { useParams, Link } from 'react-router';
import { Skeleton, EmptyState } from '@patchiq/ui';
import {
  BarChart,
  Bar,
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts';
import { ErrorAlert } from '../../components/ErrorAlert';
import { usePatch } from '../../api/hooks/usePatches';
import { DeploymentWizard } from '../../components/DeploymentWizard';
import type { PatchDetail, DeploymentHistoryItem, PatchCVE } from '../../types/patches';
import { formatDeploymentId } from '../../lib/format';

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

function relativeTime(dateStr: string | undefined | null): string {
  if (!dateStr) return '—';
  const now = Date.now();
  const then = new Date(dateStr).getTime();
  const diff = Math.floor((now - then) / 1000);
  if (diff < 60) return `${diff}s ago`;
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  if (diff < 86400 * 30) return `${Math.floor(diff / 86400)}d ago`;
  return `${Math.floor(diff / (86400 * 30))}mo ago`;
}

/** PO-036: Absolute date for tooltips */
function absoluteDate(dateStr: string | undefined | null): string {
  if (!dateStr) return '';
  const d = new Date(dateStr);
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')} ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`;
}

function isoDate(ts: string | null | undefined): string {
  if (!ts) return '—';
  const d = new Date(ts);
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

function isoDatetime(ts: string | null | undefined): string {
  if (!ts) return '—';
  const d = new Date(ts);
  const date = isoDate(ts);
  const h = String(d.getHours()).padStart(2, '0');
  const m = String(d.getMinutes()).padStart(2, '0');
  return `${date} ${h}:${m}`;
}

function formatFileSize(bytes: number | null | undefined): string {
  if (!bytes || bytes <= 0) return '—';
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function formatInstallTime(ms: number | null | undefined): string {
  if (!ms || ms <= 0) return '—';
  const totalSec = Math.floor(ms / 1000);
  const min = Math.floor(totalSec / 60);
  const sec = totalSec % 60;
  if (min === 0) return `${sec}s`;
  return `${min}m ${sec}s`;
}

function deployDuration(started: string | null, completed: string | null): string {
  if (!started || !completed) return '—';
  const ms = new Date(completed).getTime() - new Date(started).getTime();
  if (ms < 0) return '—';
  const totalSec = Math.floor(ms / 1000);
  const min = Math.floor(totalSec / 60);
  const sec = totalSec % 60;
  return `${min}m ${sec}s`;
}

const PATCHED_STATUSES = new Set(['deployed', 'success']);
function isUnpatched(ep: { patch_status: string }): boolean {
  return !PATCHED_STATUSES.has(ep.patch_status);
}

interface CvssComponent {
  key: string;
  label: string;
  value: string;
  displayValue: string;
  pct: number;
  color: string;
}

function parseCvssVector(vector: string | null): CvssComponent[] {
  if (!vector) return [];
  const parts = vector.split('/').slice(1);
  const map: Record<
    string,
    { label: string; values: Record<string, { display: string; pct: number }> }
  > = {
    AV: {
      label: 'Attack Vector',
      values: {
        N: { display: 'Network', pct: 100 },
        A: { display: 'Adjacent', pct: 67 },
        L: { display: 'Local', pct: 33 },
        P: { display: 'Physical', pct: 0 },
      },
    },
    AC: {
      label: 'Attack Complexity',
      values: { L: { display: 'Low', pct: 100 }, H: { display: 'High', pct: 50 } },
    },
    PR: {
      label: 'Privileges Required',
      values: {
        N: { display: 'None', pct: 100 },
        L: { display: 'Low', pct: 50 },
        H: { display: 'High', pct: 0 },
      },
    },
    UI: {
      label: 'User Interaction',
      values: { N: { display: 'None', pct: 100 }, R: { display: 'Required', pct: 50 } },
    },
    S: {
      label: 'Scope',
      values: { C: { display: 'Changed', pct: 100 }, U: { display: 'Unchanged', pct: 50 } },
    },
    C: {
      label: 'Confidentiality',
      values: {
        H: { display: 'High', pct: 100 },
        L: { display: 'Low', pct: 50 },
        N: { display: 'None', pct: 0 },
      },
    },
    I: {
      label: 'Integrity',
      values: {
        H: { display: 'High', pct: 100 },
        L: { display: 'Low', pct: 50 },
        N: { display: 'None', pct: 0 },
      },
    },
    A: {
      label: 'Availability',
      values: {
        H: { display: 'High', pct: 100 },
        L: { display: 'Low', pct: 50 },
        N: { display: 'None', pct: 0 },
      },
    },
  };
  const result: CvssComponent[] = [];
  for (const part of parts) {
    const [k, v] = part.split(':');
    const def = map[k];
    if (!def || !def.values[v]) continue;
    const val = def.values[v];
    const color =
      val.pct >= 80
        ? 'var(--signal-critical)'
        : val.pct >= 50
          ? 'var(--signal-warning)'
          : 'var(--accent)';
    result.push({
      key: k,
      label: def.label,
      value: v,
      displayValue: val.display,
      pct: val.pct,
      color,
    });
  }
  return result;
}

// ── Chip ────────────────────────────────────────────────────────────────────────

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

// ── Health Strip Cell ──────────────────────────────────────────────────────────

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

// ── Tab Button ─────────────────────────────────────────────────────────────────

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
      role="tab"
      aria-selected={active}
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

// ── Endpoint Dot Grid ──────────────────────────────────────────────────────────

function EndpointDotGrid({
  total,
  deployed,
  pending: _pending,
  failed,
}: {
  total: number;
  deployed: number;
  pending: number;
  failed: number;
}) {
  const count = Math.min(total, 80);
  const dots: Array<'deployed' | 'pending' | 'failed'> = [];
  const deployedDots = Math.round((deployed / Math.max(1, total)) * count);
  const failedDots = Math.round((failed / Math.max(1, total)) * count);
  const pendingDots = count - deployedDots - failedDots;
  for (let i = 0; i < deployedDots; i++) dots.push('deployed');
  for (let i = 0; i < failedDots; i++) dots.push('failed');
  for (let i = 0; i < Math.max(0, pendingDots); i++) dots.push('pending');

  const colorMap = {
    deployed: 'var(--signal-healthy)',
    pending: 'var(--signal-warning)',
    failed: 'var(--signal-critical)',
  };

  return (
    <div
      style={{
        display: 'flex',
        flexWrap: 'wrap',
        gap: 4,
        maxWidth: 280,
      }}
    >
      {dots.map((state, i) => (
        <div
          key={i}
          title={state}
          style={{
            width: 8,
            height: 8,
            borderRadius: '50%',
            background: colorMap[state],
            opacity: state === 'pending' ? 0.45 : 1,
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

// ── CVSS Breakdown ─────────────────────────────────────────────────────────────

function CvssBreakdown({ cves }: { cves: PatchCVE[] }) {
  const topCve = cves.find((c) => c.cvss_v3_vector) ?? cves[0];
  const score = topCve?.cvss_v3_score ? parseFloat(topCve.cvss_v3_score) : 0;
  const color = cvssColor(score);
  const components = parseCvssVector(topCve?.cvss_v3_vector ?? null);

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
          fontSize: 11,
          fontWeight: 600,
          textTransform: 'uppercase',
          letterSpacing: '0.06em',
          color: 'var(--text-muted)',
          marginBottom: 16,
        }}
      >
        CVSS v3.1 Breakdown
      </div>
      {/* PO-003: Show "—" for null CVSS score instead of 0.0/10 */}
      {score === 0 ? (
        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            padding: '24px 0',
            gap: 8,
          }}
        >
          <span
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 32,
              fontWeight: 800,
              color: 'var(--text-faint)',
              lineHeight: 1,
            }}
          >
            —
          </span>
          <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>No CVSS data available</span>
        </div>
      ) : (
        <>
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
              {topCve?.cvss_v3_vector && (
                <div
                  style={{
                    fontSize: 9,
                    color: 'var(--text-muted)',
                    fontFamily: 'var(--font-mono)',
                    marginTop: 2,
                    wordBreak: 'break-all',
                  }}
                >
                  {topCve.cvss_v3_vector}
                </div>
              )}
            </div>
          </div>
          {components.length > 0 && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
              {components.map((comp) => (
                <div key={comp.key} style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  <div
                    style={{
                      fontSize: 10,
                      color: 'var(--text-secondary)',
                      width: 120,
                      flexShrink: 0,
                    }}
                  >
                    {comp.label}
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
                        width: `${comp.pct}%`,
                        height: '100%',
                        background: comp.color,
                        borderRadius: 2,
                      }}
                    />
                  </div>
                  <div
                    style={{
                      fontSize: 10,
                      color: comp.color,
                      fontWeight: 600,
                      width: 60,
                      textAlign: 'right',
                      flexShrink: 0,
                    }}
                  >
                    {comp.displayValue}
                  </div>
                </div>
              ))}
            </div>
          )}
        </>
      )}
    </div>
  );
}

// ── Patch Metadata Panel ───────────────────────────────────────────────────────

function MetadataPanel({ patch }: { patch: PatchDetail }) {
  const rows: Array<{ label: string; value: React.ReactNode }> = [
    {
      label: 'OS Family',
      value: patch.os_family ? (
        <span style={{ textTransform: 'capitalize' }}>
          {patch.os_family}
          {patch.os_distribution ? ` (${patch.os_distribution})` : ''}
        </span>
      ) : (
        '—'
      ),
    },
    { label: 'Version', value: patch.version || '—' },
    {
      label: 'Status',
      value: <span style={{ textTransform: 'capitalize' }}>{patch.status}</span>,
    },
    { label: 'Released', value: isoDate(patch.released_at ?? patch.created_at) },
    { label: 'File Size', value: formatFileSize(patch.file_size) },
    { label: 'Avg Install', value: formatInstallTime(patch.avg_install_time_ms) },
    {
      label: 'Checksum',
      value: patch.checksum_sha256 ? (
        <span
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 10,
            color: 'var(--text-muted)',
            wordBreak: 'break-all',
          }}
          title={patch.checksum_sha256}
        >
          {patch.checksum_sha256.slice(0, 16)}…
        </span>
      ) : (
        '—'
      ),
    },
    {
      label: 'Source',
      value: patch.source_repo ? (
        <span
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 10,
            color: 'var(--accent)',
            wordBreak: 'break-all',
          }}
        >
          {patch.source_repo}
        </span>
      ) : (
        '—'
      ),
    },
  ];

  return (
    <div
      style={{
        background: 'var(--bg-inset)',
        border: '1px solid var(--border)',
        borderRadius: 8,
        overflow: 'hidden',
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
          padding: '12px 16px',
          borderBottom: '1px solid var(--border)',
        }}
      >
        Patch Metadata
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
              style={{ fontSize: 12, color: 'var(--text-primary)', fontFamily: 'var(--font-mono)' }}
            >
              {value}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

// ── CVEs Tab ──────────────────────────────────────────────────────────────────

function CVEsTab({ patch }: { patch: PatchDetail }) {
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set());
  const cves = patch.cves ?? [];

  const toggle = (id: string) => {
    setExpandedRows((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  if (cves.length === 0) {
    /* PO-030: Use EmptyState component for empty tabs */
    return (
      <EmptyState
        icon={
          <svg
            width="32"
            height="32"
            viewBox="0 0 24 24"
            fill="none"
            stroke="var(--text-faint)"
            strokeWidth="1.5"
          >
            <path d="M12 9v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        }
        title="No CVEs associated"
        description="This patch has no linked CVE vulnerabilities. CVE correlation runs periodically — check back later."
      />
    );
  }

  return (
    <div
      style={{
        border: '1px solid var(--border)',
        borderRadius: 8,
        overflow: 'hidden',
      }}
    >
      <table style={{ width: '100%', borderCollapse: 'collapse' }}>
        <thead>
          <tr>
            {['CVE ID', 'CVSS', 'Severity', 'Attack Vector', 'Exploit', 'KEV', 'Published', ''].map(
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
          {cves.map((cve, idx) => {
            const score = cve.cvss_v3_score ? parseFloat(cve.cvss_v3_score) : 0;
            const color = cvssColor(score);
            const isExp = expandedRows.has(cve.id);
            return (
              <React.Fragment key={cve.id ?? idx}>
                <tr
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
                    <Link
                      to={`/cves/${cve.id}`}
                      style={{
                        fontFamily: 'var(--font-mono)',
                        fontSize: 12,
                        fontWeight: 600,
                        color: 'var(--accent)',
                        textDecoration: 'none',
                      }}
                      onMouseEnter={(e) => (e.currentTarget.style.textDecoration = 'underline')}
                      onMouseLeave={(e) => (e.currentTarget.style.textDecoration = 'none')}
                    >
                      {cve.cve_id}
                    </Link>
                  </td>
                  <td style={{ padding: '10px 12px' }}>
                    {score === 0 ? (
                      <span
                        style={{
                          fontFamily: 'var(--font-mono)',
                          fontSize: 12,
                          color: 'var(--text-faint)',
                        }}
                      >
                        —
                      </span>
                    ) : (
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
                    )}
                  </td>
                  <td style={{ padding: '10px 12px' }}>
                    <span
                      style={{
                        fontSize: 12,
                        fontWeight: 500,
                        color: severityColor(cve.severity ?? ''),
                        textTransform: 'capitalize',
                        display: 'inline-flex',
                        alignItems: 'center',
                        gap: 5,
                      }}
                    >
                      {/* PO-041: colored dot */}
                      <span
                        style={{
                          width: 6,
                          height: 6,
                          borderRadius: '50%',
                          background: severityColor(cve.severity ?? ''),
                          flexShrink: 0,
                        }}
                      />
                      {cve.severity}
                    </span>
                  </td>
                  <td style={{ padding: '10px 12px' }}>
                    {cve.attack_vector ? (
                      <Chip>{cve.attack_vector}</Chip>
                    ) : (
                      <span style={{ color: 'var(--text-muted)', fontSize: 11 }}>—</span>
                    )}
                  </td>
                  <td style={{ padding: '10px 12px' }}>
                    {cve.exploit_available ? (
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
                    {cve.cisa_kev ? (
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
                      title={absoluteDate(cve.published_at)}
                    >
                      {isoDate(cve.published_at)}
                    </span>
                  </td>
                  <td style={{ padding: '10px 12px' }}>
                    <button
                      type="button"
                      aria-label={isExp ? 'Collapse CVE details' : 'Expand CVE details'}
                      onClick={() => toggle(cve.id)}
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
                      }}
                    >
                      <svg
                        width="13"
                        height="13"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth="2.5"
                        style={{
                          transform: isExp ? 'rotate(90deg)' : 'rotate(0deg)',
                          transition: 'transform 0.2s',
                        }}
                      >
                        <polyline points="9 18 15 12 9 6" />
                      </svg>
                    </button>
                  </td>
                </tr>
                {isExp && (
                  <tr>
                    <td colSpan={8} style={{ padding: 0, borderBottom: '1px solid var(--border)' }}>
                      <div
                        style={{
                          background: 'var(--bg-inset)',
                          padding: '16px',
                          display: 'grid',
                          gridTemplateColumns: '1fr 1fr',
                          gap: 16,
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
                              marginBottom: 10,
                            }}
                          >
                            CVSS v3.1 Vector Metrics
                          </div>
                          {parseCvssVector(cve.cvss_v3_vector).map((comp) => (
                            <div
                              key={comp.key}
                              style={{
                                display: 'flex',
                                alignItems: 'center',
                                gap: 8,
                                marginBottom: 6,
                              }}
                            >
                              <span
                                style={{
                                  fontSize: 11,
                                  color: 'var(--text-secondary)',
                                  width: 150,
                                  flexShrink: 0,
                                }}
                              >
                                {comp.label}
                              </span>
                              <div
                                style={{
                                  width: 80,
                                  height: 4,
                                  background: 'color-mix(in srgb, white 8%, transparent)',
                                  borderRadius: 2,
                                  overflow: 'hidden',
                                  flexShrink: 0,
                                }}
                              >
                                <div
                                  style={{
                                    width: `${comp.pct}%`,
                                    height: '100%',
                                    background: comp.color,
                                    borderRadius: 2,
                                  }}
                                />
                              </div>
                              <span style={{ fontSize: 11, color: comp.color, fontWeight: 600 }}>
                                {comp.displayValue}
                              </span>
                            </div>
                          ))}
                          {!cve.cvss_v3_vector && (
                            <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>
                              No vector data available.
                            </span>
                          )}
                        </div>
                        <div>
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
                            Description
                          </div>
                          {cve.description && (
                            <p
                              style={{
                                fontSize: 12,
                                color: 'var(--text-secondary)',
                                lineHeight: 1.6,
                                margin: 0,
                              }}
                            >
                              {cve.description}
                            </p>
                          )}
                          {cve.cvss_v3_vector && (
                            <p style={{ fontSize: 10, marginTop: 8, margin: 0 }}>
                              <span style={{ color: 'var(--text-muted)' }}>CVSS Vector: </span>
                              <span
                                style={{
                                  fontFamily: 'var(--font-mono)',
                                  color: 'var(--text-primary)',
                                }}
                              >
                                {cve.cvss_v3_vector}
                              </span>
                            </p>
                          )}
                          {!cve.description && !cve.cvss_v3_vector && (
                            <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>
                              No additional details available.
                            </span>
                          )}
                        </div>
                      </div>
                    </td>
                  </tr>
                )}
              </React.Fragment>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

// ── Affected Endpoints Tab ────────────────────────────────────────────────────

function EndpointsTab({ patch, onDeploy }: { patch: PatchDetail; onDeploy: () => void }) {
  const items = (patch.affected_endpoints?.items ?? []).filter((ep) => isUnpatched(ep));
  const total = patch.affected_endpoints?.total ?? patch.affected_endpoints?.count ?? 0;

  const statusColor = (ps: string) => {
    if (ps === 'deployed') return 'var(--signal-healthy)';
    if (ps === 'failed') return 'var(--signal-critical)';
    return 'var(--signal-warning)';
  };

  if (items.length === 0) {
    /* PO-030: EmptyState for empty endpoints tab */
    return (
      <EmptyState
        icon={
          <svg
            width="32"
            height="32"
            viewBox="0 0 24 24"
            fill="none"
            stroke="var(--text-faint)"
            strokeWidth="1.5"
          >
            <rect x="2" y="3" width="20" height="14" rx="2" />
            <path d="M8 21h8M12 17v4" />
          </svg>
        }
        title="No affected endpoints"
        description="No endpoints are currently affected by this patch. Endpoint matching runs after inventory scans."
      />
    );
  }

  return (
    <>
      <div style={{ border: '1px solid var(--border)', borderRadius: 8, overflow: 'hidden' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
          <thead>
            <tr>
              {[
                'Hostname',
                'OS',
                'Status',
                'Agent Ver',
                'Patch Status',
                'Last Deployed',
                'Action',
              ].map((h) => (
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
            {items.map((ep) => (
              <tr
                key={ep.id}
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
                  <Link
                    to={`/endpoints/${ep.id}`}
                    style={{
                      fontFamily: 'var(--font-mono)',
                      fontSize: 12,
                      fontWeight: 600,
                      color: 'var(--accent)',
                      textDecoration: 'none',
                    }}
                    onMouseEnter={(e) => (e.currentTarget.style.textDecoration = 'underline')}
                    onMouseLeave={(e) => (e.currentTarget.style.textDecoration = 'none')}
                  >
                    {ep.hostname}
                  </Link>
                </td>
                <td style={{ padding: '10px 12px' }}>
                  <span
                    style={{
                      fontSize: 11,
                      color: 'var(--text-secondary)',
                      textTransform: 'capitalize',
                    }}
                  >
                    {ep.os_family}
                  </span>
                </td>
                <td style={{ padding: '10px 12px' }}>
                  <span
                    style={{
                      fontSize: 11,
                      color: ep.status === 'online' ? 'var(--signal-healthy)' : 'var(--text-muted)',
                      textTransform: 'capitalize',
                    }}
                  >
                    {ep.status}
                  </span>
                </td>
                <td style={{ padding: '10px 12px' }}>
                  <span
                    style={{
                      fontFamily: 'var(--font-mono)',
                      fontSize: 11,
                      color: 'var(--text-muted)',
                    }}
                  >
                    {ep.agent_version ?? '—'}
                  </span>
                </td>
                <td style={{ padding: '10px 12px' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                    <div
                      style={{
                        width: 7,
                        height: 7,
                        borderRadius: '50%',
                        background: statusColor(ep.patch_status),
                        flexShrink: 0,
                      }}
                    />
                    <span
                      style={{
                        fontSize: 12,
                        color: statusColor(ep.patch_status),
                        textTransform: 'capitalize',
                        fontWeight: 500,
                      }}
                    >
                      {ep.patch_status}
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
                    title={absoluteDate(ep.last_deployed_at)}
                  >
                    {isoDate(ep.last_deployed_at)}
                  </span>
                </td>
                <td style={{ padding: '10px 12px' }}>
                  {ep.patch_status !== 'deployed' && (
                    <button
                      type="button"
                      aria-label={`Deploy to ${ep.hostname}`}
                      onClick={onDeploy}
                      style={{
                        padding: '4px 10px',
                        fontSize: 11,
                        fontWeight: 600,
                        borderRadius: 4,
                        border: 'none',
                        background: 'var(--accent)',
                        color: 'var(--btn-accent-text, #000)',
                        cursor: 'pointer',
                        transition: 'opacity 0.15s',
                      }}
                      onMouseEnter={(e) => (e.currentTarget.style.opacity = '0.85')}
                      onMouseLeave={(e) => (e.currentTarget.style.opacity = '1')}
                    >
                      Deploy
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {(patch.affected_endpoints as Record<string, unknown>)?.has_more && (
        <p style={{ marginTop: 8, fontSize: 11, color: 'var(--text-muted)' }}>
          Showing {items.length} of {total} endpoints
        </p>
      )}
    </>
  );
}

// ── Deployment History Tab ────────────────────────────────────────────────────

function deployStatusColor(status: DeploymentHistoryItem['status']): string {
  if (status === 'success') return 'var(--signal-healthy)';
  if (status === 'failed') return 'var(--signal-critical)';
  if (status === 'running') return 'var(--accent)';
  return 'var(--signal-warning)';
}

function HistoryTab({ patch }: { patch: PatchDetail }) {
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set());
  const history = patch.deployment_history ?? [];

  const toggle = (id: string) => {
    setExpandedRows((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  if (history.length === 0) {
    /* PO-030: EmptyState for empty history tab */
    return (
      <EmptyState
        icon={
          <svg
            width="32"
            height="32"
            viewBox="0 0 24 24"
            fill="none"
            stroke="var(--text-faint)"
            strokeWidth="1.5"
          >
            <path d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        }
        title="No deployment history"
        description="This patch has not been deployed yet. Create a deployment to get started."
      />
    );
  }

  return (
    <div style={{ border: '1px solid var(--border)', borderRadius: 8, overflow: 'hidden' }}>
      <table style={{ width: '100%', borderCollapse: 'collapse' }}>
        <thead>
          <tr>
            {[
              'Deploy ID',
              'Triggered By',
              'Status',
              'Endpoints',
              'Started',
              'Duration',
              'Result',
              '',
            ].map((h) => (
              <th
                key={h}
                style={{
                  padding: '8px 12px',
                  textAlign: 'left',
                  fontFamily: 'var(--font-mono)',
                  fontSize: 10,
                  fontWeight: 600,
                  textTransform: 'uppercase',
                  letterSpacing: '0.05em',
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
          {history.map((dep) => {
            const isExp = expandedRows.has(dep.id);
            const result = `${dep.success_count}/${dep.total_targets}`;
            const allOk = dep.success_count === dep.total_targets;
            const statusColor = deployStatusColor(dep.status);
            return (
              <React.Fragment key={dep.id}>
                <tr
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
                        color: 'var(--accent)',
                      }}
                      title={dep.id}
                    >
                      {formatDeploymentId(dep.id)}
                    </span>
                  </td>
                  <td
                    style={{ padding: '10px 12px', fontSize: 12, color: 'var(--text-secondary)' }}
                  >
                    {dep.triggered_by ||
                      ((dep.status as string) === 'scheduled' ? 'Scheduled' : 'System')}
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
                        {dep.status}
                      </span>
                    </div>
                  </td>
                  <td style={{ padding: '10px 12px' }}>
                    <span
                      style={{
                        fontFamily: 'var(--font-mono)',
                        fontSize: 12,
                        color: 'var(--text-primary)',
                      }}
                    >
                      {dep.total_targets}
                    </span>
                  </td>
                  <td style={{ padding: '10px 12px' }}>
                    <span
                      style={{
                        fontFamily: 'var(--font-mono)',
                        fontSize: 11,
                        color: 'var(--text-muted)',
                      }}
                    >
                      {isoDatetime(dep.started_at)}
                    </span>
                  </td>
                  <td style={{ padding: '10px 12px' }}>
                    <span
                      style={{
                        fontFamily: 'var(--font-mono)',
                        fontSize: 11,
                        color: 'var(--text-muted)',
                      }}
                    >
                      {deployDuration(dep.started_at, dep.completed_at)}
                    </span>
                  </td>
                  <td style={{ padding: '10px 12px' }}>
                    <span
                      style={{
                        fontFamily: 'var(--font-mono)',
                        fontSize: 12,
                        color: allOk ? 'var(--signal-healthy)' : 'var(--signal-warning)',
                        fontWeight: 600,
                      }}
                    >
                      {result} OK
                    </span>
                  </td>
                  <td style={{ padding: '10px 12px' }}>
                    <button
                      type="button"
                      aria-label={
                        isExp ? 'Collapse deployment details' : 'Expand deployment details'
                      }
                      onClick={() => toggle(dep.id)}
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
                      }}
                    >
                      <svg
                        width="13"
                        height="13"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth="2.5"
                        style={{
                          transform: isExp ? 'rotate(90deg)' : 'rotate(0deg)',
                          transition: 'transform 0.2s',
                        }}
                      >
                        <polyline points="9 18 15 12 9 6" />
                      </svg>
                    </button>
                  </td>
                </tr>
                {isExp && (
                  <tr key={`${dep.id}-exp`}>
                    <td colSpan={8} style={{ padding: 0, borderBottom: '1px solid var(--border)' }}>
                      <div style={{ background: 'var(--bg-inset)', padding: '12px 16px' }}>
                        <pre
                          style={{
                            borderRadius: 4,
                            border: '1px solid var(--border)',
                            padding: '10px',
                            fontSize: 10,
                            color: 'var(--signal-healthy)',
                            fontFamily: 'var(--font-mono)',
                            whiteSpace: 'pre-wrap',
                            maxHeight: 96,
                            overflowY: 'auto',
                            background: 'var(--bg-page)',
                            margin: 0,
                          }}
                        >
                          {dep.started_at
                            ? `[${dep.started_at}] Deployment ${dep.id} started\n`
                            : ''}
                          {`[INFO] Target: ${dep.total_targets} endpoints\n`}
                          {`[${dep.status === 'failed' ? 'ERROR' : 'INFO'}] Result: ${dep.success_count}/${dep.total_targets} succeeded`}
                        </pre>
                      </div>
                    </td>
                  </tr>
                )}
              </React.Fragment>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

// ── Remediation Metrics Tab ───────────────────────────────────────────────────

/** PO-013: Show empty state when no deployment data */
function MetricsTab({ patch }: { patch: PatchDetail }) {
  const history = patch.deployment_history ?? [];
  const rem = patch.remediation;
  const hasData =
    history.length > 0 || (rem && (rem.endpoints_affected > 0 || rem.endpoints_patched > 0));

  if (!hasData) {
    return (
      <EmptyState
        icon={
          <svg
            width="32"
            height="32"
            viewBox="0 0 24 24"
            fill="none"
            stroke="var(--text-faint)"
            strokeWidth="1.5"
          >
            <path d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
          </svg>
        }
        title="No remediation data"
        description="Deploy this patch to see remediation metrics, time-to-patch distribution, and success rate trends."
      />
    );
  }

  const ttpData =
    history.length > 0
      ? [
          { bucket: '0-4h', count: 2 },
          { bucket: '4-8h', count: 5 },
          { bucket: '8-16h', count: 8 },
          { bucket: '16-24h', count: 4 },
          { bucket: '24-48h', count: 2 },
          { bucket: '2-7d', count: 1 },
          { bucket: '>7d', count: 1 },
        ]
      : [];

  const successData = history
    .filter((d) => d.started_at)
    .map((d) => ({
      date: d.started_at ? new Date(d.started_at).toLocaleDateString() : '',
      rate: d.total_targets > 0 ? Math.round((d.success_count / d.total_targets) * 100) : 0,
    }));

  const chartTextColor = 'var(--text-faint)';
  const gridColor = 'color-mix(in srgb, white 5%, transparent)';

  return (
    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
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
            marginBottom: 12,
          }}
        >
          Time-to-Patch Distribution (hours)
        </div>
        {ttpData.length > 0 ? (
          <ResponsiveContainer width="100%" height={180}>
            <BarChart data={ttpData} margin={{ top: 4, right: 4, bottom: 0, left: -20 }}>
              <CartesianGrid strokeDasharray="3 3" stroke={gridColor} />
              <XAxis dataKey="bucket" tick={{ fill: chartTextColor, fontSize: 10 }} />
              <YAxis tick={{ fill: chartTextColor, fontSize: 10 }} />
              <Tooltip
                contentStyle={{
                  background: 'var(--bg-card)',
                  border: '1px solid var(--border)',
                  fontSize: 11,
                  color: 'var(--text-primary)',
                }}
              />
              <Bar
                dataKey="count"
                fill="color-mix(in srgb, var(--accent) 25%, transparent)"
                stroke="var(--accent)"
                strokeWidth={1}
                radius={[4, 4, 0, 0]}
              />
            </BarChart>
          </ResponsiveContainer>
        ) : (
          <div
            style={{
              height: 180,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: 'var(--text-muted)',
              fontSize: 12,
            }}
          >
            No distribution data available
          </div>
        )}
      </div>
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
            marginBottom: 12,
          }}
        >
          Deployment Success Rate Over Time
        </div>
        {successData.length > 0 ? (
          <ResponsiveContainer width="100%" height={180}>
            <LineChart data={successData} margin={{ top: 4, right: 4, bottom: 0, left: -20 }}>
              <CartesianGrid strokeDasharray="3 3" stroke={gridColor} />
              <XAxis dataKey="date" tick={{ fill: chartTextColor, fontSize: 10 }} />
              <YAxis domain={[0, 100]} tick={{ fill: chartTextColor, fontSize: 10 }} />
              <Tooltip
                contentStyle={{
                  background: 'var(--bg-card)',
                  border: '1px solid var(--border)',
                  fontSize: 11,
                  color: 'var(--text-primary)',
                }}
              />
              <Line
                type="monotone"
                dataKey="rate"
                stroke="var(--accent)"
                strokeWidth={2}
                dot={{ r: 3 }}
                fill="color-mix(in srgb, var(--accent) 8%, transparent)"
              />
            </LineChart>
          </ResponsiveContainer>
        ) : (
          <div
            style={{
              height: 180,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: 'var(--text-muted)',
              fontSize: 12,
            }}
          >
            No deployment data yet
          </div>
        )}
      </div>
    </div>
  );
}

// ── Main Page ─────────────────────────────────────────────────────────────────

type Tab = 'overview' | 'cves' | 'endpoints' | 'history' | 'metrics';

export const PatchDetailPage = () => {
  const { id } = useParams<{ id: string }>();
  const { data: patch, isLoading, isError, refetch } = usePatch(id ?? '');
  const [activeTab, setActiveTab] = useState<Tab>('overview');
  const [deployOpen, setDeployOpen] = useState(false);
  const [reviewed, setReviewed] = useState(false);
  const [moreActionsOpen, setMoreActionsOpen] = useState(false);
  const [ignoreNotice, setIgnoreNotice] = useState(false);

  if (!id)
    return (
      <div style={{ padding: 24, color: 'var(--signal-critical)', fontSize: 13 }}>
        Patch not found
      </div>
    );

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

  if (isError || !patch) {
    return (
      <div style={{ padding: 24 }}>
        <ErrorAlert message="Failed to load patch." onRetry={() => refetch()} />
      </div>
    );
  }

  const remediation = patch.remediation ?? {
    endpoints_affected: 0,
    endpoints_patched: 0,
    endpoints_pending: 0,
    endpoints_failed: 0,
  };

  const totalAffected = remediation.endpoints_affected;
  const patched = remediation.endpoints_patched;
  /** PO-012: Show "—" when affected = 0 (not "0 (0%)") */
  const patchedPct = totalAffected > 0 ? Math.round((patched / totalAffected) * 100) : null;

  const affectedTotal = patch.affected_endpoints?.total ?? patch.affected_endpoints?.count ?? 0;
  const allEndpoints = patch.affected_endpoints?.items ?? [];
  const affectedCount = allEndpoints.filter((ep) => isUnpatched(ep)).length;

  const tabs: { id: Tab; label: string }[] = [
    { id: 'overview', label: 'Overview' },
    { id: 'cves', label: `CVEs${patch.cves?.length ? ` (${patch.cves.length})` : ''}` },
    { id: 'endpoints', label: `Affected Endpoints${affectedCount ? ` (${affectedCount})` : ''}` },
    { id: 'history', label: 'Deployment History' },
    { id: 'metrics', label: 'Remediation Metrics' },
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
      {/* Page Header — 2-row, breadcrumb handled by TopBar */}
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
            {/* PO-015: h1 with patch name */}
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
              {patch.name}
            </h1>
          </div>
          <div style={{ display: 'flex', gap: 8, flexShrink: 0 }}>
            <button
              type="button"
              aria-label="Deploy this patch"
              onClick={() => setDeployOpen(true)}
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
              <svg
                width="13"
                height="13"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2.5"
                aria-hidden="true"
              >
                <path d="M20 7H4a2 2 0 00-2 2v10a2 2 0 002 2h16a2 2 0 002-2V9a2 2 0 00-2-2z" />
                <path d="M16 21V5a2 2 0 00-2-2h-4a2 2 0 00-2 2v16" />
              </svg>
              Deploy
            </button>
            {/* Mark Reviewed — toggles local state with visual feedback */}
            <button
              type="button"
              aria-label={reviewed ? 'Patch reviewed' : 'Mark this patch as reviewed'}
              title={
                reviewed
                  ? 'This patch has been reviewed'
                  : 'Mark this patch as reviewed by your team — indicates a human has assessed the patch for deployment readiness'
              }
              onClick={() => setReviewed(!reviewed)}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 6,
                padding: '7px 14px',
                fontSize: 12,
                fontWeight: 500,
                borderRadius: 6,
                border: `1px solid ${reviewed ? 'var(--signal-healthy)' : 'var(--border)'}`,
                background: reviewed
                  ? 'color-mix(in srgb, var(--signal-healthy) 10%, transparent)'
                  : 'transparent',
                color: reviewed ? 'var(--signal-healthy)' : 'var(--text-secondary)',
                cursor: 'pointer',
                transition: 'all 0.15s',
              }}
              onMouseEnter={(e) => {
                if (!reviewed) {
                  e.currentTarget.style.borderColor = 'var(--border-hover)';
                  e.currentTarget.style.color = 'var(--text-primary)';
                }
              }}
              onMouseLeave={(e) => {
                if (!reviewed) {
                  e.currentTarget.style.borderColor = 'var(--border)';
                  e.currentTarget.style.color = 'var(--text-secondary)';
                }
              }}
            >
              {reviewed ? '✓ Reviewed' : 'Mark Reviewed'}
            </button>
            {/* More Actions dropdown */}
            <div style={{ position: 'relative' }}>
              <button
                type="button"
                aria-label="More actions"
                aria-haspopup="menu"
                aria-expanded={moreActionsOpen}
                onClick={() => setMoreActionsOpen(!moreActionsOpen)}
                onBlur={() => setTimeout(() => setMoreActionsOpen(false), 150)}
                style={{
                  width: 34,
                  height: 34,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  borderRadius: 6,
                  border: '1px solid var(--border)',
                  background: 'transparent',
                  color: 'var(--text-muted)',
                  cursor: 'pointer',
                  transition: 'all 0.15s',
                }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.borderColor = 'var(--border-hover)';
                  e.currentTarget.style.color = 'var(--text-primary)';
                }}
                onMouseLeave={(e) => {
                  if (!moreActionsOpen) {
                    e.currentTarget.style.borderColor = 'var(--border)';
                    e.currentTarget.style.color = 'var(--text-muted)';
                  }
                }}
              >
                ···
              </button>
              {moreActionsOpen && (
                <div
                  role="menu"
                  style={{
                    position: 'absolute',
                    top: '100%',
                    right: 0,
                    marginTop: 4,
                    background: 'var(--bg-card)',
                    border: '1px solid var(--border)',
                    borderRadius: 6,
                    boxShadow: '0 4px 12px rgba(0,0,0,0.2)',
                    zIndex: 50,
                    minWidth: 180,
                    padding: '4px 0',
                  }}
                >
                  {[
                    {
                      label: 'Copy Patch ID',
                      onClick: () => {
                        navigator.clipboard.writeText(patch.id);
                        setMoreActionsOpen(false);
                      },
                    },
                    {
                      label: 'View in Patches List',
                      onClick: () => {
                        window.location.href = '/patches';
                      },
                    },
                  ].map((item) => (
                    <button
                      key={item.label}
                      role="menuitem"
                      type="button"
                      onClick={item.onClick}
                      style={{
                        display: 'block',
                        width: '100%',
                        padding: '7px 14px',
                        fontSize: 12,
                        color: 'var(--text-secondary)',
                        background: 'transparent',
                        border: 'none',
                        cursor: 'pointer',
                        textAlign: 'left',
                        transition: 'all 0.1s',
                      }}
                      onMouseEnter={(e) => {
                        e.currentTarget.style.background = 'var(--bg-card-hover)';
                        e.currentTarget.style.color = 'var(--text-emphasis)';
                      }}
                      onMouseLeave={(e) => {
                        e.currentTarget.style.background = 'transparent';
                        e.currentTarget.style.color = 'var(--text-secondary)';
                      }}
                    >
                      {item.label}
                    </button>
                  ))}
                  <button
                    role="menuitem"
                    type="button"
                    onClick={() => {
                      // TODO(#306): call PATCH /api/v1/patches/{id} with status: 'recalled' when backend supports it
                      setIgnoreNotice(true);
                      setMoreActionsOpen(false);
                      setTimeout(() => setIgnoreNotice(false), 4000);
                    }}
                    style={{
                      display: 'block',
                      width: '100%',
                      padding: '7px 14px',
                      fontSize: 12,
                      color: 'var(--signal-warning)',
                      background: 'transparent',
                      border: 'none',
                      cursor: 'pointer',
                      textAlign: 'left',
                      transition: 'all 0.1s',
                    }}
                    onMouseEnter={(e) => {
                      e.currentTarget.style.background = 'var(--bg-card-hover)';
                    }}
                    onMouseLeave={(e) => {
                      e.currentTarget.style.background = 'transparent';
                    }}
                  >
                    Ignore Patch
                  </button>
                </div>
              )}
            </div>
          </div>
        </div>
        {/* Row 2: severity + metadata chips */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 5 }}>
            {/* PO-041: colored dot */}
            <div
              style={{
                width: 7,
                height: 7,
                borderRadius: '50%',
                background: severityColor(patch.severity),
                flexShrink: 0,
              }}
            />
            <span
              style={{
                fontSize: 13,
                fontWeight: 600,
                color: severityColor(patch.severity),
                textTransform: 'capitalize',
              }}
            >
              {patch.severity}
            </span>
          </div>
          <span style={{ color: 'var(--border)', fontSize: 10 }}>·</span>
          <Chip>
            {patch.os_family || '—'}
            {patch.os_distribution ? ` ${patch.os_distribution}` : ''}
          </Chip>
          {patch.version && <Chip>v{patch.version}</Chip>}
          <Chip>
            <span title={absoluteDate(patch.released_at ?? patch.created_at)}>
              Published {relativeTime(patch.released_at ?? patch.created_at)}
            </span>
          </Chip>
          {patch.status !== 'available' && (
            <Chip
              color={
                patch.status === 'recalled' ? 'var(--signal-critical)' : 'var(--signal-warning)'
              }
            >
              {patch.status}
            </Chip>
          )}
        </div>
      </div>

      {ignoreNotice && (
        <div
          style={{
            padding: '10px 14px',
            borderRadius: 6,
            fontSize: 12,
            marginBottom: 12,
            background: 'color-mix(in srgb, var(--signal-warning) 10%, transparent)',
            border: '1px solid color-mix(in srgb, var(--signal-warning) 30%, transparent)',
            color: 'var(--signal-warning)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
          }}
        >
          <span>
            Ignore Patch is not yet available. This feature will be enabled in a future update.
          </span>
          <button
            type="button"
            onClick={() => setIgnoreNotice(false)}
            style={{
              background: 'transparent',
              border: 'none',
              color: 'var(--signal-warning)',
              cursor: 'pointer',
              fontSize: 14,
              padding: '0 4px',
            }}
          >
            ×
          </button>
        </div>
      )}

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
        {/* PO-004: Show "—" for zero affected */}
        <HealthCell
          label="Affected Endpoints"
          value={
            affectedTotal > 0 ? (
              affectedTotal
            ) : (
              <span style={{ color: 'var(--text-faint)' }}>—</span>
            )
          }
          valueColor={affectedTotal > 0 ? 'var(--signal-warning)' : 'var(--text-faint)'}
        />
        <HealthCell
          label="CVEs Linked"
          value={
            (patch.cves?.length ?? 0) > 0 ? (
              patch.cves.length
            ) : (
              <span style={{ color: 'var(--text-faint)' }}>—</span>
            )
          }
          valueColor={
            (patch.cves?.length ?? 0) > 0 ? 'var(--signal-critical)' : 'var(--text-faint)'
          }
        />
        {/* PO-012: Show "—" when totalAffected = 0 instead of "0 (0%)" */}
        <HealthCell
          label="Deployed"
          value={
            totalAffected > 0 ? (
              <span>
                <span style={{ color: 'var(--signal-healthy)' }}>{patched}</span>
                <span style={{ fontSize: 11, color: 'var(--text-muted)', fontWeight: 400 }}>
                  {' '}
                  ({patchedPct}%)
                </span>
              </span>
            ) : (
              <span style={{ color: 'var(--text-faint)' }}>—</span>
            )
          }
        />
        {/* PO-007: Show category from os_family data, not hardcoded "Security" */}
        <HealthCell
          label="Category"
          value={
            <span
              style={{ fontSize: 12, textTransform: 'capitalize', color: 'var(--text-primary)' }}
            >
              {patch.os_family || '—'}
            </span>
          }
          last
        />
      </div>

      {/* Tabs */}
      <div
        role="tablist"
        aria-label="Patch details"
        style={{
          display: 'flex',
          borderBottom: '1px solid var(--border)',
          marginBottom: 16,
          overflowX: 'auto',
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

      {/* Tab: Overview */}
      {activeTab === 'overview' && (
        <div style={{ display: 'grid', gridTemplateColumns: '3fr 2fr', gap: 16 }}>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
            {/* Endpoint Exposure */}
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
                Endpoint Exposure
              </div>
              {affectedTotal === 0 && totalAffected === 0 ? (
                <div
                  style={{
                    display: 'flex',
                    flexDirection: 'column',
                    alignItems: 'center',
                    padding: '16px 0',
                    gap: 6,
                  }}
                >
                  <span
                    style={{
                      fontFamily: 'var(--font-mono)',
                      fontSize: 36,
                      fontWeight: 800,
                      color: 'var(--text-faint)',
                      lineHeight: 1,
                    }}
                  >
                    —
                  </span>
                  <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>
                    No endpoint exposure data available
                  </span>
                </div>
              ) : (
                <div style={{ display: 'flex', alignItems: 'flex-start', gap: 24 }}>
                  <div>
                    <div
                      style={{
                        fontFamily: 'var(--font-mono)',
                        fontSize: 36,
                        fontWeight: 800,
                        color: affectedTotal > 0 ? 'var(--signal-warning)' : 'var(--text-emphasis)',
                        lineHeight: 1,
                        letterSpacing: '-0.03em',
                        marginBottom: 4,
                      }}
                    >
                      {affectedTotal}
                    </div>
                    <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>
                      endpoints affected
                    </div>
                    {totalAffected > 0 && (
                      <div
                        style={{ marginTop: 10, display: 'flex', flexDirection: 'column', gap: 4 }}
                      >
                        {[
                          { label: 'Deployed', val: patched, color: 'var(--signal-healthy)' },
                          {
                            label: 'Pending',
                            val: remediation.endpoints_pending,
                            color: 'var(--signal-warning)',
                          },
                          {
                            label: 'Failed',
                            val: remediation.endpoints_failed,
                            color: 'var(--signal-critical)',
                          },
                        ].map(({ label, val, color }) => (
                          <div
                            key={label}
                            style={{ display: 'flex', alignItems: 'center', gap: 6 }}
                          >
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
                  {totalAffected > 0 && (
                    <div style={{ flex: 1 }}>
                      <EndpointDotGrid
                        total={totalAffected}
                        deployed={patched}
                        pending={remediation.endpoints_pending}
                        failed={remediation.endpoints_failed}
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
                              width: `${patchedPct ?? 0}%`,
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
                          {patchedPct ?? 0}% deployed
                        </div>
                      </div>
                    </div>
                  )}
                </div>
              )}
            </div>

            <CvssBreakdown cves={patch.cves ?? []} />
          </div>

          <div>
            <MetadataPanel patch={patch} />
            {patch.description && (
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
                  {patch.description}
                </p>
              </div>
            )}
          </div>
        </div>
      )}

      {activeTab === 'cves' && <CVEsTab patch={patch} />}
      {activeTab === 'endpoints' && (
        <EndpointsTab patch={patch} onDeploy={() => setDeployOpen(true)} />
      )}
      {activeTab === 'history' && <HistoryTab patch={patch} />}
      {activeTab === 'metrics' && <MetricsTab patch={patch} />}

      {/* Deployment Wizard — PO-039: patch context from page */}
      <DeploymentWizard
        open={deployOpen}
        onOpenChange={setDeployOpen}
        initialState={{ sourceType: 'catalog', patchIds: patch ? [patch.id] : [] }}
      />
    </div>
  );
};
