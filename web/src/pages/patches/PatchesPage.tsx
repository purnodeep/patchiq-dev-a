import React, { useState, useCallback } from 'react';
import { useNavigate, useSearchParams } from 'react-router';
import { usePatches, usePatch, usePatchSeverityCounts } from '../../api/hooks/usePatches';
import type { PatchListItem } from '../../types/patches';
import { DeploymentWizard } from '../../components/DeploymentWizard';
import type { DeploymentWizardInitialState } from '../../types/deployment-wizard';
import { EmptyState } from '@patchiq/ui';

// ─── Helpers ──────────────────────────────────────────────────────────────────

function cvssColor(score: number): string {
  if (score >= 9) return 'var(--signal-critical)';
  if (score >= 7) return 'var(--signal-warning)';
  if (score >= 4) return 'var(--text-secondary)';
  return 'var(--text-muted)';
}

function severityColor(sev: string): string {
  switch (sev?.toLowerCase()) {
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

function relTime(d: string | undefined | null): string {
  if (!d) return '—';
  const s = Math.floor((Date.now() - new Date(d).getTime()) / 1000);
  if (s < 60) return `${s}s ago`;
  if (s < 3600) return `${Math.floor(s / 60)}m ago`;
  if (s < 86400) return `${Math.floor(s / 3600)}h ago`;
  if (s < 86400 * 30) return `${Math.floor(s / 86400)}d ago`;
  if (s < 86400 * 365) return `${Math.floor(s / (86400 * 30))}mo ago`;
  return `${Math.floor(s / (86400 * 365))}y ago`;
}

function absDate(d: string | undefined | null): string {
  if (!d) return '';
  const dt = new Date(d);
  return `${dt.getFullYear()}-${String(dt.getMonth() + 1).padStart(2, '0')}-${String(dt.getDate()).padStart(2, '0')} ${String(dt.getHours()).padStart(2, '0')}:${String(dt.getMinutes()).padStart(2, '0')}`;
}

function fmtCount(v: number | null | undefined): string {
  return v && v > 0 ? v.toLocaleString() : '—';
}

// ��── Skeleton ─────────────────────────────────────────────────────────────────

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

// ─── Checkbox ─────────────────────────────────────────────────────────────────

function CB({ on, onClick }: { on: boolean; onClick: (e: React.MouseEvent) => void }) {
  return (
    <div
      role="checkbox"
      aria-checked={on}
      tabIndex={0}
      onClick={onClick}
      onKeyDown={(e) => {
        if (e.key === ' ' || e.key === 'Enter') {
          e.preventDefault();
          onClick(e as unknown as React.MouseEvent);
        }
      }}
      style={{
        width: 14,
        height: 14,
        borderRadius: 3,
        flexShrink: 0,
        cursor: 'pointer',
        border: `1.5px solid ${on ? 'var(--accent)' : 'var(--border-strong)'}`,
        background: on ? 'var(--accent)' : 'transparent',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        transition: 'all 0.1s',
      }}
    >
      {on && (
        <svg
          width="8"
          height="6"
          viewBox="0 0 8 6"
          fill="none"
          stroke="var(--btn-accent-text, #000)"
          strokeWidth="1.8"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <path d="M1 3L3 5L7 1" />
        </svg>
      )}
    </div>
  );
}

// ─── Stat Card ────────────────────────────────────────────────────────────────

function StatCard({
  label,
  value,
  valueColor,
  active,
  onClick,
}: {
  label: string;
  value: number | undefined;
  valueColor?: string;
  active?: boolean;
  onClick: () => void;
}) {
  const [h, setH] = useState(false);
  return (
    <button
      type="button"
      aria-label={`Filter by ${label}: ${value != null ? value.toLocaleString() : 'loading'}`}
      onClick={onClick}
      onMouseEnter={() => setH(true)}
      onMouseLeave={() => setH(false)}
      style={{
        flex: 1,
        minWidth: 0,
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'flex-start',
        padding: '12px 14px',
        background: active ? 'color-mix(in srgb, white 3%, transparent)' : 'var(--bg-card)',
        border: `1px solid ${active ? (valueColor ?? 'var(--accent)') : h ? 'var(--border-hover)' : 'var(--border)'}`,
        borderRadius: 8,
        cursor: 'pointer',
        transition: 'all 0.15s',
        outline: 'none',
        textAlign: 'left',
        transform: h && !active ? 'translateY(-1px)' : 'none',
        boxShadow: h && !active ? '0 2px 8px rgba(0,0,0,0.15)' : 'none',
      }}
    >
      <span
        style={{
          fontSize: 10,
          fontFamily: 'var(--font-mono)',
          fontWeight: 500,
          textTransform: 'uppercase',
          letterSpacing: '0.05em',
          color: active ? (valueColor ?? 'var(--accent)') : 'var(--text-muted)',
          marginBottom: 4,
        }}
      >
        {label}
      </span>
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
        {value != null ? value.toLocaleString() : '—'}
      </span>
    </button>
  );
}

// ─── CVSS Mini Ring ───────────────────────────────────────────────────────────

function CvssRing({ score }: { score: number }) {
  const pct = Math.min(100, score * 10);
  const color = score > 0 ? cvssColor(score) : 'var(--text-faint)';
  const r = 18;
  const c = 2 * Math.PI * r;
  return (
    <svg width="44" height="44" viewBox="0 0 44 44">
      <circle cx="22" cy="22" r={r} fill="none" stroke="var(--border)" strokeWidth="3" />
      {score > 0 && (
        <circle
          cx="22"
          cy="22"
          r={r}
          fill="none"
          stroke={color}
          strokeWidth="3"
          strokeDasharray={`${(c * pct) / 100} ${c}`}
          strokeLinecap="round"
          transform="rotate(-90 22 22)"
          style={{ transition: 'stroke-dasharray 0.6s ease' }}
        />
      )}
      <text
        x="22"
        y="22"
        textAnchor="middle"
        dominantBaseline="central"
        style={{ fontFamily: 'var(--font-mono)', fontSize: 11, fontWeight: 700, fill: color }}
      >
        {score > 0 ? score.toFixed(1) : '—'}
      </text>
    </svg>
  );
}

// ─── Progress Bar ─────────────────────────────────────────────────────────────

function MiniBar({ value, max, color }: { value: number; max: number; color: string }) {
  const pct = max > 0 ? Math.min(100, (value / max) * 100) : 0;
  return (
    <div
      style={{
        height: 3,
        background: 'color-mix(in srgb, white 8%, transparent)',
        borderRadius: 2,
        overflow: 'hidden',
        flex: 1,
      }}
    >
      <div
        style={{
          width: `${pct}%`,
          height: '100%',
          background: color,
          borderRadius: 2,
          transition: 'width 0.4s ease',
        }}
      />
    </div>
  );
}

// ─── Expanded Row ─────────────────────────────────────────────────────────────

function ExpandedRow({
  patch,
  onDeploy,
  colSpan,
}: {
  patch: PatchListItem;
  onDeploy: (p: PatchListItem) => void;
  colSpan: number;
}) {
  const { data: detail } = usePatch(patch.id);
  const cves = detail?.cves ?? [];
  const rem = detail?.remediation;
  const affected = rem
    ? rem.endpoints_affected + rem.endpoints_patched
    : patch.affected_endpoint_count;
  const deployed = rem?.endpoints_patched ?? patch.endpoints_deployed_count ?? 0;
  const pending =
    rem?.endpoints_pending ?? Math.max(0, (patch.affected_endpoint_count ?? 0) - deployed);
  const failed = rem?.endpoints_failed ?? 0;
  const hasData = detail != null;
  const navigate = useNavigate();

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
    marginBottom: 5,
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
  };

  return (
    <tr>
      <td colSpan={colSpan} style={{ padding: 0, borderBottom: '1px solid var(--border)' }}>
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
          {/* Patch Info card */}
          <div style={{ ...CARD, flex: '0 0 500px' }}>
            <div style={LBL}>Patch Info</div>
            <p
              style={{
                fontSize: 15,
                color: 'var(--text-secondary)',
                lineHeight: 1.6,
                margin: '0 0 14px',
              }}
            >
              {patch.description ?? 'No description available.'}
            </p>
            <div
              style={{
                display: 'flex',
                flexWrap: 'wrap',
                gap: '10px 24px',
                borderTop: '1px solid var(--border)',
                paddingTop: 12,
              }}
            >
              {[
                {
                  label: 'Severity',
                  value: patch.severity.charAt(0).toUpperCase() + patch.severity.slice(1),
                  color:
                    patch.severity === 'critical'
                      ? 'var(--signal-critical)'
                      : patch.severity === 'high'
                        ? 'var(--signal-warning)'
                        : patch.severity === 'medium'
                          ? 'var(--signal-warning)'
                          : 'var(--text-secondary)',
                },
                { label: 'OS Family', value: patch.os_family || '—', color: 'var(--text-primary)' },
                { label: 'Version', value: patch.version || '—', color: 'var(--text-primary)' },
                {
                  label: 'Status',
                  value: patch.status.charAt(0).toUpperCase() + patch.status.slice(1),
                  color:
                    patch.status === 'available'
                      ? 'var(--signal-healthy)'
                      : patch.status === 'recalled'
                        ? 'var(--signal-critical)'
                        : 'var(--text-muted)',
                },
                {
                  label: 'CVEs',
                  value: String(patch.cve_count),
                  color: patch.cve_count > 0 ? 'var(--signal-warning)' : 'var(--text-muted)',
                },
                {
                  label: 'Highest CVSS',
                  value: patch.highest_cvss_score > 0 ? String(patch.highest_cvss_score) : '—',
                  color:
                    patch.highest_cvss_score >= 9
                      ? 'var(--signal-critical)'
                      : patch.highest_cvss_score >= 7
                        ? 'var(--signal-warning)'
                        : 'var(--text-secondary)',
                },
              ].map(({ label, value, color }) => (
                <div key={label} style={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
                  <span
                    style={{
                      fontFamily: 'var(--font-mono)',
                      fontSize: 10,
                      textTransform: 'uppercase',
                      letterSpacing: '0.06em',
                      color: 'var(--text-faint)',
                    }}
                  >
                    {label}
                  </span>
                  <span
                    style={{ fontFamily: 'var(--font-mono)', fontSize: 15, fontWeight: 700, color }}
                  >
                    {value}
                  </span>
                </div>
              ))}
            </div>
          </div>

          {/* Endpoint Exposure card */}
          <div style={{ ...CARD, flex: '0 0 480px', marginLeft: 24 }}>
            <div style={LBL}>Endpoint Exposure</div>
            {!hasData ? (
              <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>Loading…</span>
            ) : (
              <>
                <div
                  style={{
                    display: 'flex',
                    flexDirection: 'column',
                    gap: 7,
                    marginBottom: affected > 0 ? 10 : 0,
                  }}
                >
                  {[
                    {
                      label: 'Affected Endpoints',
                      value: String(affected),
                      color: 'var(--text-primary)',
                    },
                    { label: 'Deployed', value: String(deployed), color: 'var(--signal-healthy)' },
                    {
                      label: 'Pending',
                      value: String(pending),
                      color: pending > 0 ? 'var(--signal-warning)' : 'var(--text-muted)',
                    },
                    {
                      label: 'Failed',
                      value: String(failed),
                      color: failed > 0 ? 'var(--signal-critical)' : 'var(--text-muted)',
                    },
                    {
                      label: 'CVEs Linked',
                      value: cves.length === 0 && !hasData ? '—' : String(cves.length),
                      color: cves.length > 0 ? 'var(--signal-warning)' : 'var(--text-muted)',
                    },
                  ].map(({ label, value, color }) => (
                    <div
                      key={label}
                      style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13 }}
                    >
                      <span style={{ color: 'var(--text-secondary)' }}>{label}</span>
                      <span
                        style={{
                          fontWeight: 600,
                          color,
                          fontFamily: 'var(--font-mono)',
                          fontSize: 11,
                        }}
                      >
                        {value}
                      </span>
                    </div>
                  ))}
                </div>
                {affected > 0 && (
                  <div
                    style={{
                      display: 'flex',
                      gap: 2,
                      height: 4,
                      borderRadius: 3,
                      overflow: 'hidden',
                      background: 'var(--bg-card)',
                    }}
                  >
                    {deployed > 0 && (
                      <div style={{ flex: deployed, background: 'var(--signal-healthy)' }} />
                    )}
                    {pending > 0 && (
                      <div style={{ flex: pending, background: 'var(--signal-warning)' }} />
                    )}
                    {failed > 0 && (
                      <div style={{ flex: failed, background: 'var(--signal-critical)' }} />
                    )}
                  </div>
                )}
              </>
            )}
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
            <button
              type="button"
              style={{
                ...BTN,
                color: 'var(--btn-accent-text, #000)',
                borderColor: 'var(--accent)',
                background: 'var(--accent)',
              }}
              onClick={(e) => {
                e.stopPropagation();
                onDeploy(patch);
              }}
            >
              ⎌ Deploy Patch
            </button>
            <button
              type="button"
              style={BTN}
              onClick={(e) => {
                e.stopPropagation();
                navigate(`/patches/${patch.id}`);
              }}
            >
              View Details →
            </button>
          </div>
        </div>
      </td>
    </tr>
  );
}

// ─── Sort Header ──────────────────────────────────────────────────────────────

const TH: React.CSSProperties = {
  padding: '9px 12px',
  textAlign: 'left',
  fontFamily: 'var(--font-mono)',
  fontSize: 11,
  fontWeight: 600,
  letterSpacing: '0.05em',
  color: 'var(--text-muted)',
  textTransform: 'uppercase',
  whiteSpace: 'nowrap',
};
const TD: React.CSSProperties = {
  padding: '10px 12px',
  borderBottom: '1px solid var(--border)',
  verticalAlign: 'middle',
};

function SortTH({
  label,
  col,
  sortCol,
  sortDir,
  onSort,
}: {
  label: string;
  col: string;
  sortCol: string | null;
  sortDir: 'asc' | 'desc';
  onSort: (c: string) => void;
}) {
  const a = sortCol === col;
  const [h, setH] = useState(false);
  return (
    <th
      style={{ ...TH, cursor: 'pointer', userSelect: 'none' }}
      onClick={() => onSort(col)}
      onMouseEnter={() => setH(true)}
      onMouseLeave={() => setH(false)}
      aria-sort={a ? (sortDir === 'asc' ? 'ascending' : 'descending') : undefined}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
        <span style={{ color: a ? 'var(--text-emphasis)' : undefined }}>{label}</span>
        <svg
          width="10"
          height="10"
          viewBox="0 0 10 10"
          fill="none"
          style={{ opacity: a ? 1 : h ? 0.5 : 0, transition: 'opacity 0.15s', flexShrink: 0 }}
        >
          {(!a || sortDir === 'asc') && (
            <path d="M5 2L8 5.5H2L5 2Z" fill={a ? 'var(--text-emphasis)' : 'var(--text-muted)'} />
          )}
          {(!a || sortDir === 'desc') && (
            <path d="M5 8L2 4.5H8L5 8Z" fill={a ? 'var(--text-emphasis)' : 'var(--text-muted)'} />
          )}
        </svg>
      </div>
    </th>
  );
}

// ─── Patch Card (Grid) ────────────────────────────────────────────────────────

function PatchCard({
  patch,
  selected,
  onSelect,
  onClick,
  onDeploy,
}: {
  patch: PatchListItem;
  selected: boolean;
  onSelect: () => void;
  onClick: () => void;
  onDeploy: () => void;
}) {
  const [h, setH] = useState(false);
  const score = patch.highest_cvss_score ?? 0;
  const affected = patch.affected_endpoint_count ?? 0;
  const cveCount = patch.cve_count ?? 0;
  const remPct = patch.remediation_pct ?? 0;

  return (
    <div
      onClick={onClick}
      onMouseEnter={() => setH(true)}
      onMouseLeave={() => setH(false)}
      style={{
        background: selected
          ? 'color-mix(in srgb, var(--accent) 5%, var(--bg-card))'
          : 'var(--bg-card)',
        border: `1px solid ${selected ? 'var(--accent)' : h ? 'var(--border-hover)' : 'var(--border)'}`,
        borderRadius: 10,
        cursor: 'pointer',
        transition: 'all 0.15s',
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden',
        transform: h ? 'translateY(-2px)' : 'none',
        boxShadow: h ? '0 4px 12px rgba(0,0,0,0.15)' : 'none',
      }}
    >
      {/* Header: checkbox + name + severity */}
      <div
        style={{
          padding: '12px 14px',
          background: 'var(--bg-inset)',
          borderBottom: '1px solid var(--border)',
          display: 'flex',
          alignItems: 'flex-start',
          gap: 8,
        }}
      >
        <CB
          on={selected}
          onClick={(e) => {
            e.stopPropagation();
            onSelect();
          }}
        />
        <div style={{ flex: 1, minWidth: 0 }}>
          <div
            style={{
              fontWeight: 600,
              color: 'var(--text-emphasis)',
              fontSize: 13,
              lineHeight: 1.3,
              wordBreak: 'break-word',
            }}
          >
            {patch.name}
          </div>
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 6,
              marginTop: 4,
              flexWrap: 'wrap',
            }}
          >
            <span
              style={{
                fontSize: 10,
                fontWeight: 600,
                textTransform: 'capitalize',
                color: severityColor(patch.severity),
                display: 'inline-flex',
                alignItems: 'center',
                gap: 3,
              }}
            >
              <span
                style={{
                  width: 6,
                  height: 6,
                  borderRadius: '50%',
                  background: severityColor(patch.severity),
                }}
              />
              {patch.severity}
            </span>
            {patch.os_family && (
              <span
                style={{ fontSize: 10, color: 'var(--text-muted)', textTransform: 'capitalize' }}
              >
                {patch.os_family}
              </span>
            )}
            {patch.version && (
              <span
                style={{ fontSize: 10, color: 'var(--text-faint)', fontFamily: 'var(--font-mono)' }}
              >
                v{patch.version}
              </span>
            )}
          </div>
        </div>
      </div>

      {/* Body: Visual metrics */}
      <div
        style={{
          padding: '14px 14px 12px',
          display: 'flex',
          flexDirection: 'column',
          gap: 12,
          flex: 1,
        }}
      >
        {/* CVSS Ring + CVE count + Affected */}
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: '1fr 1fr 1fr',
            gap: 8,
            textAlign: 'center',
          }}
        >
          <div>
            <CvssRing score={score} />
            <div
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 9,
                fontWeight: 600,
                textTransform: 'uppercase',
                color: 'var(--text-muted)',
                marginTop: 4,
              }}
            >
              CVSS
            </div>
          </div>
          <div
            style={{
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            <span
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 20,
                fontWeight: 800,
                color: cveCount > 0 ? 'var(--accent)' : 'var(--text-faint)',
                lineHeight: 1,
              }}
            >
              {fmtCount(cveCount)}
            </span>
            <div
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 9,
                fontWeight: 600,
                textTransform: 'uppercase',
                color: 'var(--text-muted)',
                marginTop: 4,
              }}
            >
              CVEs
            </div>
          </div>
          <div
            style={{
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            <span
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 20,
                fontWeight: 800,
                color: affected > 0 ? 'var(--signal-warning)' : 'var(--text-faint)',
                lineHeight: 1,
              }}
            >
              {fmtCount(affected)}
            </span>
            <div
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 9,
                fontWeight: 600,
                textTransform: 'uppercase',
                color: 'var(--text-muted)',
                marginTop: 4,
              }}
            >
              Affected
            </div>
          </div>
        </div>

        {/* Remediation bar */}
        {affected > 0 && (
          <div>
            <div
              style={{
                display: 'flex',
                justifyContent: 'space-between',
                fontSize: 10,
                marginBottom: 3,
              }}
            >
              <span style={{ color: 'var(--text-muted)' }}>Remediation</span>
              <span
                style={{
                  fontFamily: 'var(--font-mono)',
                  fontWeight: 600,
                  color:
                    remPct >= 80
                      ? 'var(--signal-healthy)'
                      : remPct >= 40
                        ? 'var(--signal-warning)'
                        : 'var(--text-secondary)',
                }}
              >
                {remPct}%
              </span>
            </div>
            <MiniBar
              value={remPct}
              max={100}
              color={
                remPct >= 80
                  ? 'var(--signal-healthy)'
                  : remPct >= 40
                    ? 'var(--signal-warning)'
                    : 'var(--accent)'
              }
            />
          </div>
        )}

        {/* Description snippet */}
        {patch.description && (
          <div
            style={{
              fontSize: 11,
              color: 'var(--text-muted)',
              lineHeight: 1.4,
              display: '-webkit-box',
              WebkitLineClamp: 2,
              WebkitBoxOrient: 'vertical',
              overflow: 'hidden',
            }}
          >
            {patch.description}
          </div>
        )}
      </div>

      {/* Footer */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          borderTop: '1px solid var(--border)',
          padding: '8px 14px',
        }}
      >
        <span
          style={{ fontFamily: 'var(--font-mono)', fontSize: 10, color: 'var(--text-muted)' }}
          title={absDate(patch.released_at ?? patch.created_at)}
        >
          {relTime(patch.released_at ?? patch.created_at)}
        </span>
        <button
          type="button"
          onClick={(e) => {
            e.stopPropagation();
            onDeploy();
          }}
          style={{
            fontSize: 10,
            fontWeight: 600,
            color: 'var(--accent)',
            background: 'transparent',
            border: '1px solid var(--border)',
            borderRadius: 4,
            padding: '3px 8px',
            cursor: 'pointer',
            transition: 'all 0.15s',
          }}
          onMouseEnter={(e) => {
            e.currentTarget.style.borderColor = 'var(--accent)';
          }}
          onMouseLeave={(e) => {
            e.currentTarget.style.borderColor = 'var(--border)';
          }}
        >
          Deploy
        </button>
      </div>
    </div>
  );
}

// ─── Pagination ───────────────────────────────────────────────────────────────

function PagBtn({
  children,
  active,
  disabled,
  onClick,
  'aria-label': al,
}: {
  children: React.ReactNode;
  active?: boolean;
  disabled?: boolean;
  onClick?: () => void;
  'aria-label'?: string;
}) {
  const [h, setH] = useState(false);
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={onClick}
      aria-label={al}
      aria-current={active ? 'page' : undefined}
      onMouseEnter={() => setH(true)}
      onMouseLeave={() => setH(false)}
      style={{
        padding: '4px 9px',
        fontSize: 11,
        fontFamily: 'var(--font-mono)',
        borderRadius: 5,
        border: '1px solid',
        cursor: disabled ? 'not-allowed' : 'pointer',
        transition: 'all 0.15s',
        background: active
          ? 'color-mix(in srgb, var(--accent) 12%, transparent)'
          : h && !disabled
            ? 'color-mix(in srgb, white 4%, transparent)'
            : 'transparent',
        borderColor: active ? 'var(--accent)' : 'var(--border)',
        color: active ? 'var(--accent)' : disabled ? 'var(--text-faint)' : 'var(--text-secondary)',
        opacity: disabled && !active ? 0.45 : 1,
      }}
    >
      {children}
    </button>
  );
}

// ─── Main Page ────────────────────────────────────────────────────────────────

type StatusFilter = '' | 'available' | 'superseded' | 'recalled';
type PageSize = 15 | 25 | 50 | 100;

export const PatchesPage = () => {
  const [search, setSearch] = useState('');
  const [severity, setSeverity] = useState('');
  const [status, setStatus] = useState<StatusFilter>('');
  const [osFamily, setOsFamily] = useState('');
  const [cursors, setCursors] = useState<string[]>([]);
  const [wizardOpen, setWizardOpen] = useState(false);
  const [wizardInit, setWizardInit] = useState<DeploymentWizardInitialState | undefined>();
  const [sortCol, setSortCol] = useState<string | null>(null);
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('asc');
  const [pageSize, setPageSize] = useState<PageSize>(15);
  const [sel, setSel] = useState<Set<string>>(new Set());
  const [exp, setExp] = useState<Set<string>>(new Set());
  const [actionsOpen, setActionsOpen] = useState(false);
  const [searchParams, setSearchParams] = useSearchParams();
  const viewMode = (searchParams.get('view') === 'card' ? 'card' : 'list') as 'list' | 'card';
  const setViewMode = useCallback(
    (m: 'list' | 'card') => {
      setSearchParams((prev) => {
        const n = new URLSearchParams(prev);
        if (m === 'list') n.delete('view');
        else n.set('view', m);
        return n;
      });
    },
    [setSearchParams],
  );

  const navigate = useNavigate();
  const cursor = cursors[cursors.length - 1];

  const { data, isLoading, isError, refetch } = usePatches({
    cursor,
    limit: pageSize,
    severity: severity || undefined,
    os_family: osFamily || undefined,
    status: status || undefined,
    search: search || undefined,
    sort_by: sortCol || undefined,
    sort_dir: sortCol ? sortDir : undefined,
  });
  const { data: sevCounts } = usePatchSeverityCounts({
    os_family: osFamily || undefined,
    status: status || undefined,
    search: search || undefined,
  });

  const totalCount = data?.total_count ?? 0;
  const items = data?.data ?? [];
  const critCount = sevCounts?.critical ?? 0;
  const highCount = sevCounts?.high ?? 0;
  const medCount = sevCounts?.medium ?? 0;
  const lowCount = sevCounts?.low ?? 0;
  const statTotal = critCount + highCount + medCount + lowCount;
  const hasFilters = severity !== '' || osFamily !== '' || status !== '' || search !== '';
  const selN = sel.size;

  // Selection helpers
  const togSel = useCallback(
    (id: string) =>
      setSel((p) => {
        const n = new Set(p);
        if (n.has(id)) n.delete(id);
        else n.add(id);
        return n;
      }),
    [],
  );
  const togAll = useCallback(() => {
    setSel((p) => (p.size === items.length ? new Set() : new Set(items.map((i) => i.id))));
  }, [items]);
  const allSel = items.length > 0 && sel.size === items.length;
  const togExp = useCallback(
    (id: string) =>
      setExp((p) => {
        const n = new Set(p);
        if (n.has(id)) n.delete(id);
        else n.add(id);
        return n;
      }),
    [],
  );

  const openDeploy = (p: PatchListItem | null) => {
    setWizardInit(p ? { sourceType: 'catalog', patchIds: [p.id] } : undefined);
    setWizardOpen(true);
  };
  const deploySelected = () => {
    if (selN > 0) {
      setWizardInit({ sourceType: 'catalog', patchIds: [...sel] });
      setWizardOpen(true);
    }
  };

  const clearFilters = useCallback(() => {
    setSeverity('');
    setOsFamily('');
    setStatus('');
    setSearch('');
    setCursors([]);
  }, []);
  const toggleSort = useCallback(
    (c: string) => {
      if (sortCol === c) {
        if (sortDir === 'asc') setSortDir('desc');
        else {
          setSortCol(null);
          setSortDir('asc');
        }
      } else {
        setSortCol(c);
        setSortDir('asc');
      }
      setCursors([]); // reset pagination on sort change
    },
    [sortCol, sortDir],
  );

  // Sorting is server-side — items come pre-sorted from the API
  const sorted = items;

  // Pagination
  const pgSize = items.length;
  const pgStart = pgSize > 0 ? (cursors.length > 0 ? cursors.length * pageSize + 1 : 1) : 0;
  const pgEnd = pgSize > 0 ? pgStart + pgSize - 1 : 0;
  const curPage = cursors.length + 1;
  const totalPages = Math.max(1, Math.ceil(totalCount / pageSize));
  const hasNext = !!data?.next_cursor;
  const navTo = hasNext ? curPage + 1 : curPage;
  const pageNums: number[] = [];
  for (let p = 1; p <= Math.min(navTo, totalPages); p++) pageNums.push(p);
  const showLast = totalPages > navTo;

  const pagInfo =
    pgSize > 0
      ? `Showing ${pgStart.toLocaleString()}–${pgEnd.toLocaleString()} of ${totalCount.toLocaleString()}`
      : totalCount > 0
        ? `${totalCount.toLocaleString()} patches`
        : 'No patches found';
  const filtInfo =
    hasFilters && statTotal !== totalCount ? ` (filtered from ${statTotal.toLocaleString()})` : '';

  const pagination = (
    <div style={{ display: 'flex', gap: 4, alignItems: 'center' }}>
      <PagBtn
        aria-label="Go to previous page"
        disabled={cursors.length === 0}
        onClick={() => setCursors((p) => p.slice(0, -1))}
      >
        ← Prev
      </PagBtn>
      {pageNums.map((p) => (
        <PagBtn
          key={p}
          active={p === curPage}
          disabled={p > navTo}
          aria-label={`Go to page ${p}`}
          onClick={() => {
            if (p === 1) setCursors([]);
            else if (p === curPage + 1 && hasNext) setCursors((pr) => [...pr, data!.next_cursor!]);
            else setCursors((pr) => pr.slice(0, p - 1));
          }}
        >
          {p}
        </PagBtn>
      ))}
      {totalPages > navTo + 1 && (
        <span
          style={{
            fontSize: 11,
            color: 'var(--text-faint)',
            padding: '0 2px',
            fontFamily: 'var(--font-mono)',
          }}
        >
          …
        </span>
      )}
      {showLast && (
        <PagBtn disabled aria-label={`Last page, page ${totalPages.toLocaleString()}`}>
          {totalPages.toLocaleString()}
        </PagBtn>
      )}
      <PagBtn
        aria-label="Go to next page"
        disabled={!data?.next_cursor}
        onClick={() => {
          if (data?.next_cursor) setCursors((p) => [...p, data.next_cursor!]);
        }}
      >
        Next →
      </PagBtn>
    </div>
  );

  // ─── Render ───────────────────────────────────────────────────────────────

  return (
    <div
      style={{
        padding: 24,
        display: 'flex',
        flexDirection: 'column',
        gap: 16,
        minHeight: '100%',
        background: 'var(--bg-page)',
      }}
    >
      <h1
        style={{
          position: 'absolute',
          width: 1,
          height: 1,
          overflow: 'hidden',
          clip: 'rect(0,0,0,0)',
        }}
      >
        Patches
      </h1>

      {/* Stat Cards */}
      <div style={{ display: 'flex', gap: 8 }}>
        <StatCard
          label="Total"
          value={hasFilters ? totalCount : statTotal}
          active={severity === ''}
          onClick={() => {
            setSeverity('');
            setCursors([]);
          }}
        />
        <StatCard
          label="Critical"
          value={critCount}
          valueColor="var(--signal-critical)"
          active={severity === 'critical'}
          onClick={() => {
            setSeverity(severity === 'critical' ? '' : 'critical');
            setCursors([]);
          }}
        />
        <StatCard
          label="High"
          value={highCount}
          valueColor="var(--signal-warning)"
          active={severity === 'high'}
          onClick={() => {
            setSeverity(severity === 'high' ? '' : 'high');
            setCursors([]);
          }}
        />
        <StatCard
          label="Medium"
          value={medCount}
          valueColor="var(--text-secondary)"
          active={severity === 'medium'}
          onClick={() => {
            setSeverity(severity === 'medium' ? '' : 'medium');
            setCursors([]);
          }}
        />
        <StatCard
          label="Low"
          value={lowCount}
          valueColor="var(--text-muted)"
          active={severity === 'low'}
          onClick={() => {
            setSeverity(severity === 'low' ? '' : 'low');
            setCursors([]);
          }}
        />
      </div>

      {/* Bulk Actions Bar */}
      {selN > 0 && (
        <div
          style={{
            position: 'sticky',
            top: 0,
            zIndex: 10,
            background: 'var(--bg-card)',
            border: '1px solid var(--accent)',
            borderRadius: 8,
            padding: '10px 16px',
            display: 'flex',
            alignItems: 'center',
            gap: 10,
            boxShadow: '0 2px 8px rgba(0,0,0,0.12)',
          }}
        >
          <span
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 12,
              fontWeight: 600,
              color: 'var(--accent)',
            }}
          >
            {selN} selected
          </span>
          <div style={{ width: 1, height: 20, background: 'var(--border)' }} />
          <button
            type="button"
            onClick={deploySelected}
            style={{
              padding: '5px 12px',
              fontSize: 11.5,
              fontWeight: 600,
              borderRadius: 6,
              border: 'none',
              background: 'var(--accent)',
              color: 'var(--btn-accent-text, #000)',
              cursor: 'pointer',
            }}
          >
            Deploy Selected
          </button>
          <button
            type="button"
            onClick={() => {
              const selected = items.filter((p) => sel.has(p.id));
              if (!selected.length) return;
              const headers = [
                'Patch Name',
                'Severity',
                'OS Family',
                'Version',
                'Status',
                'CVEs',
                'Highest CVSS',
                'Affected Endpoints',
                'Published',
              ];
              const csvRows = selected.map((p) =>
                [
                  `"${(p.name ?? '').replace(/"/g, '""')}"`,
                  p.severity,
                  p.os_family ?? '',
                  p.version ?? '',
                  p.status,
                  p.cve_count ?? 0,
                  p.highest_cvss_score ?? '',
                  p.affected_endpoint_count ?? 0,
                  p.released_at ? new Date(p.released_at).toLocaleDateString() : '',
                ].join(','),
              );
              const csv = [headers.join(','), ...csvRows].join('\n');
              const blob = new Blob([csv], { type: 'text/csv' });
              const url = URL.createObjectURL(blob);
              const a = document.createElement('a');
              a.href = url;
              a.download = 'patches-export.csv';
              a.click();
              URL.revokeObjectURL(url);
            }}
            style={{
              padding: '5px 12px',
              fontSize: 11.5,
              borderRadius: 6,
              border: '1px solid var(--border)',
              background: 'transparent',
              color: 'var(--text-secondary)',
              cursor: 'pointer',
            }}
          >
            Export
          </button>
          <div style={{ flex: 1 }} />
          <button
            type="button"
            onClick={() => setSel(new Set())}
            style={{
              fontSize: 11,
              color: 'var(--accent)',
              background: 'transparent',
              border: 'none',
              cursor: 'pointer',
              fontWeight: 500,
            }}
          >
            Clear
          </button>
        </div>
      )}

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
        >
          {/* Search */}
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 8,
              padding: '5px 10px',
              border: '1px solid var(--border)',
              borderRadius: 6,
              background: 'var(--bg-inset)',
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
              id="patches-search"
              name="search"
              type="text"
              aria-label="Search patches"
              value={search}
              onChange={(e) => {
                setSearch(e.target.value);
                setCursors([]);
              }}
              placeholder="Search patches (KB, USN, RHSA…)"
              style={{
                background: 'transparent',
                border: 'none',
                outline: 'none',
                fontSize: 12,
                color: 'var(--text-primary)',
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
          <select
            id="patches-os"
            name="os_family"
            aria-label="Filter by OS family"
            value={osFamily}
            onChange={(e) => {
              setOsFamily(e.target.value);
              setCursors([]);
            }}
            style={{
              background: 'var(--bg-card)',
              border: '1px solid var(--border)',
              borderRadius: 6,
              padding: '5px 10px',
              fontSize: 11.5,
              color: osFamily ? 'var(--text-primary)' : 'var(--text-secondary)',
              outline: 'none',
              cursor: 'pointer',
            }}
            onFocus={(e) => (e.currentTarget.style.borderColor = 'var(--accent)')}
            onBlur={(e) => (e.currentTarget.style.borderColor = 'var(--border)')}
          >
            <option value="">OS Family</option>
            <option value="windows">Windows</option>
            <option value="ubuntu">Ubuntu</option>
            <option value="rhel">RHEL</option>
            <option value="debian">Debian</option>
          </select>
          <select
            id="patches-status"
            name="status"
            aria-label="Filter by status"
            value={status}
            onChange={(e) => {
              setStatus(e.target.value as StatusFilter);
              setCursors([]);
            }}
            style={{
              background: 'var(--bg-card)',
              border: '1px solid var(--border)',
              borderRadius: 6,
              padding: '5px 10px',
              fontSize: 11.5,
              color: status ? 'var(--text-primary)' : 'var(--text-secondary)',
              outline: 'none',
              cursor: 'pointer',
            }}
            onFocus={(e) => (e.currentTarget.style.borderColor = 'var(--accent)')}
            onBlur={(e) => (e.currentTarget.style.borderColor = 'var(--border)')}
          >
            <option value="">Status</option>
            <option value="available">Available</option>
            <option value="superseded">Superseded</option>
            <option value="recalled">Not Applicable</option>
          </select>
          {hasFilters && (
            <button
              type="button"
              onClick={clearFilters}
              style={{
                padding: '5px 10px',
                fontSize: 11,
                borderRadius: 6,
                border: '1px solid var(--border)',
                background: 'transparent',
                color: 'var(--text-secondary)',
                cursor: 'pointer',
                display: 'flex',
                alignItems: 'center',
                gap: 4,
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
              Clear filters
            </button>
          )}
          <div style={{ flex: 1 }} />
          <select
            id="pg-size"
            name="page_size"
            aria-label="Items per page"
            value={pageSize}
            onChange={(e) => {
              setPageSize(Number(e.target.value) as PageSize);
              setCursors([]);
            }}
            style={{
              background: 'var(--bg-card)',
              border: '1px solid var(--border)',
              borderRadius: 6,
              padding: '5px 8px',
              fontSize: 11,
              color: 'var(--text-muted)',
              outline: 'none',
              cursor: 'pointer',
              fontFamily: 'var(--font-mono)',
            }}
          >
            <option value={15}>15</option>
            <option value={25}>25</option>
            <option value={50}>50</option>
            <option value={100}>100</option>
          </select>
          <div
            style={{
              display: 'flex',
              border: '1px solid var(--border)',
              borderRadius: 6,
              overflow: 'hidden',
            }}
          >
            <button
              type="button"
              aria-label="List view"
              aria-pressed={viewMode === 'list'}
              onClick={() => setViewMode('list')}
              style={{
                padding: '5px 8px',
                background:
                  viewMode === 'list'
                    ? 'color-mix(in srgb, var(--accent) 10%, transparent)'
                    : 'transparent',
                border: 'none',
                cursor: 'pointer',
                color: viewMode === 'list' ? 'var(--text-emphasis)' : 'var(--text-muted)',
                display: 'flex',
                alignItems: 'center',
              }}
            >
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                <path d="M2 3.5h10M2 7h10M2 10.5h10" stroke="currentColor" strokeWidth="1.2" />
              </svg>
            </button>
            <button
              type="button"
              aria-label="Grid view"
              aria-pressed={viewMode === 'card'}
              onClick={() => setViewMode('card')}
              style={{
                padding: '5px 8px',
                background:
                  viewMode === 'card'
                    ? 'color-mix(in srgb, var(--accent) 10%, transparent)'
                    : 'transparent',
                border: 'none',
                borderLeft: '1px solid var(--border)',
                cursor: 'pointer',
                color: viewMode === 'card' ? 'var(--text-emphasis)' : 'var(--text-muted)',
                display: 'flex',
                alignItems: 'center',
              }}
            >
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                <rect
                  x="2"
                  y="2"
                  width="4"
                  height="4"
                  rx="1"
                  stroke="currentColor"
                  strokeWidth="1.2"
                />
                <rect
                  x="8"
                  y="2"
                  width="4"
                  height="4"
                  rx="1"
                  stroke="currentColor"
                  strokeWidth="1.2"
                />
                <rect
                  x="2"
                  y="8"
                  width="4"
                  height="4"
                  rx="1"
                  stroke="currentColor"
                  strokeWidth="1.2"
                />
                <rect
                  x="8"
                  y="8"
                  width="4"
                  height="4"
                  rx="1"
                  stroke="currentColor"
                  strokeWidth="1.2"
                />
              </svg>
            </button>
          </div>
        </div>

        {/* Actions Card */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 8,
            padding: '10px 12px',
            boxShadow: 'var(--shadow-sm)',
          }}
        >
          <div style={{ position: 'relative' }}>
            <button
              type="button"
              onClick={() => setActionsOpen((o) => !o)}
              style={{
                padding: '5px 12px',
                fontSize: 11.5,
                fontWeight: 600,
                borderRadius: 6,
                border: 'none',
                background: 'var(--accent)',
                color: 'var(--btn-accent-text, #000)',
                cursor: 'pointer',
                display: 'flex',
                alignItems: 'center',
                gap: 6,
                transition: 'opacity 0.15s',
              }}
              onMouseEnter={(e) => (e.currentTarget.style.opacity = '0.85')}
              onMouseLeave={(e) => (e.currentTarget.style.opacity = '1')}
            >
              Actions
              <svg
                width="10"
                height="10"
                viewBox="0 0 10 10"
                fill="none"
                stroke="currentColor"
                strokeWidth="1.8"
                style={{ marginLeft: 2 }}
              >
                <path d="M2 3.5l3 3 3-3" />
              </svg>
            </button>
            {actionsOpen && (
              <>
                <div
                  style={{ position: 'fixed', inset: 0, zIndex: 49 }}
                  onClick={() => setActionsOpen(false)}
                />
                <div
                  style={{
                    position: 'absolute',
                    top: 'calc(100% + 4px)',
                    right: 0,
                    zIndex: 50,
                    background: 'var(--bg-card)',
                    border: '1px solid var(--border)',
                    borderRadius: 8,
                    boxShadow: '0 4px 16px rgba(0,0,0,0.18)',
                    minWidth: 200,
                    overflow: 'hidden',
                  }}
                >
                  <div style={{ padding: '4px 0' }}>
                    <button
                      type="button"
                      onClick={() => {
                        setActionsOpen(false);
                        openDeploy(null);
                      }}
                      style={{
                        width: '100%',
                        padding: '8px 14px',
                        fontSize: 12,
                        textAlign: 'left',
                        background: 'transparent',
                        border: 'none',
                        cursor: 'pointer',
                        color: 'var(--text-primary)',
                        display: 'flex',
                        alignItems: 'center',
                        gap: 8,
                      }}
                      onMouseEnter={(e) => (e.currentTarget.style.background = 'var(--bg-inset)')}
                      onMouseLeave={(e) => (e.currentTarget.style.background = 'transparent')}
                    >
                      <svg
                        width="12"
                        height="12"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth="2"
                      >
                        <path d="M20 7H4a2 2 0 00-2 2v10a2 2 0 002 2h16a2 2 0 002-2V9a2 2 0 00-2-2z" />
                        <path d="M16 21V5a2 2 0 00-2-2h-4a2 2 0 00-2 2v16" />
                      </svg>
                      Create Deployment
                    </button>
                    <div style={{ height: 1, background: 'var(--border)', margin: '2px 0' }} />
                    <button
                      type="button"
                      onClick={() => {
                        setActionsOpen(false);
                        const rows = items;
                        if (!rows.length) return;
                        const headers = [
                          'Patch Name',
                          'Severity',
                          'OS Family',
                          'Version',
                          'Status',
                          'CVEs',
                          'Highest CVSS',
                          'Affected Endpoints',
                          'Published',
                        ];
                        const csvRows = rows.map((p) =>
                          [
                            `"${(p.name ?? '').replace(/"/g, '""')}"`,
                            p.severity,
                            p.os_family ?? '',
                            p.version ?? '',
                            p.status,
                            p.cve_count ?? 0,
                            p.highest_cvss_score ?? '',
                            p.affected_endpoint_count ?? 0,
                            p.released_at ? new Date(p.released_at).toLocaleDateString() : '',
                          ].join(','),
                        );
                        const csv = [headers.join(','), ...csvRows].join('\n');
                        const blob = new Blob([csv], { type: 'text/csv' });
                        const url = URL.createObjectURL(blob);
                        const a = document.createElement('a');
                        a.href = url;
                        a.download = 'patches-export.csv';
                        a.click();
                        URL.revokeObjectURL(url);
                      }}
                      style={{
                        width: '100%',
                        padding: '8px 14px',
                        fontSize: 12,
                        textAlign: 'left',
                        background: 'transparent',
                        border: 'none',
                        cursor: 'pointer',
                        color: 'var(--text-primary)',
                        display: 'flex',
                        alignItems: 'center',
                        gap: 8,
                      }}
                      onMouseEnter={(e) => (e.currentTarget.style.background = 'var(--bg-inset)')}
                      onMouseLeave={(e) => (e.currentTarget.style.background = 'transparent')}
                    >
                      <svg
                        width="12"
                        height="12"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth="2"
                      >
                        <path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4" />
                        <polyline points="7 10 12 15 17 10" />
                        <line x1="12" y1="15" x2="12" y2="3" />
                      </svg>
                      Export CSV
                    </button>
                  </div>
                </div>
              </>
            )}
          </div>
        </div>
      </div>

      {/* Grid View */}
      {viewMode === 'card' && (
        <div>
          {isError ? (
            <div
              style={{
                padding: 24,
                background: 'var(--bg-card)',
                border: '1px solid var(--border)',
                borderRadius: 10,
              }}
            >
              <div
                style={{
                  padding: '12px 16px',
                  borderRadius: 6,
                  border: '1px solid color-mix(in srgb, var(--signal-critical) 20%, transparent)',
                  background: 'color-mix(in srgb, var(--signal-critical) 5%, transparent)',
                  color: 'var(--signal-critical)',
                  fontSize: 13,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                }}
              >
                <span>Failed to load patches.</span>
                <button
                  type="button"
                  onClick={() => refetch()}
                  style={{
                    fontSize: 12,
                    color: 'var(--signal-critical)',
                    background: 'transparent',
                    border: '1px solid',
                    borderRadius: 4,
                    padding: '3px 10px',
                    cursor: 'pointer',
                  }}
                >
                  Retry
                </button>
              </div>
            </div>
          ) : isLoading ? (
            <div
              style={{
                display: 'grid',
                gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))',
                gap: 12,
              }}
            >
              {Array.from({ length: 8 }).map((_, i) => (
                <div
                  key={i}
                  style={{
                    background: 'var(--bg-card)',
                    border: '1px solid var(--border)',
                    borderRadius: 10,
                    height: 240,
                    animation: 'pulse 1.5s ease-in-out infinite',
                  }}
                />
              ))}
            </div>
          ) : sorted.length === 0 ? (
            <div
              style={{
                background: 'var(--bg-card)',
                border: '1px solid var(--border)',
                borderRadius: 10,
                padding: '48px 24px',
              }}
            >
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
                    <path d="M20 7H4a2 2 0 00-2 2v10a2 2 0 002 2h16a2 2 0 002-2V9a2 2 0 00-2-2z" />
                    <path d="M16 21V5a2 2 0 00-2-2h-4a2 2 0 00-2 2v16" />
                  </svg>
                }
                title="No patches found"
                description={
                  hasFilters ? 'Try adjusting your filters.' : 'No patches in catalog yet.'
                }
                action={hasFilters ? { label: 'Clear Filters', onClick: clearFilters } : undefined}
              />
            </div>
          ) : (
            <>
              <div
                style={{
                  display: 'grid',
                  gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))',
                  gap: 12,
                }}
              >
                {sorted.map((p) => (
                  <PatchCard
                    key={p.id}
                    patch={p}
                    selected={sel.has(p.id)}
                    onSelect={() => togSel(p.id)}
                    onClick={() => navigate(`/patches/${p.id}`)}
                    onDeploy={() => openDeploy(p)}
                  />
                ))}
              </div>
              <div style={{ display: 'flex', alignItems: 'center', padding: '10px 0', gap: 6 }}>
                <span
                  style={{
                    flex: 1,
                    fontSize: 11,
                    fontFamily: 'var(--font-mono)',
                    color: 'var(--text-muted)',
                  }}
                >
                  {pagInfo}
                  {filtInfo}
                </span>
                {pagination}
              </div>
            </>
          )}
        </div>
      )}

      {/* List View */}
      {viewMode === 'list' && (
        <div
          style={{
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 8,
            boxShadow: 'var(--shadow-sm)',
            overflow: 'hidden',
          }}
        >
          {isError ? (
            <div style={{ padding: 24 }}>
              <div
                style={{
                  padding: '12px 16px',
                  borderRadius: 6,
                  border: '1px solid color-mix(in srgb, var(--signal-critical) 20%, transparent)',
                  background: 'color-mix(in srgb, var(--signal-critical) 5%, transparent)',
                  color: 'var(--signal-critical)',
                  fontSize: 13,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                }}
              >
                <span>Failed to load patches.</span>
                <button
                  type="button"
                  onClick={() => refetch()}
                  style={{
                    fontSize: 12,
                    color: 'var(--signal-critical)',
                    background: 'transparent',
                    border: '1px solid',
                    borderRadius: 4,
                    padding: '3px 10px',
                    cursor: 'pointer',
                  }}
                >
                  Retry
                </button>
              </div>
            </div>
          ) : (
            <>
              <div style={{ overflowX: 'auto' }}>
                <table style={{ width: '100%', borderCollapse: 'collapse', minWidth: 900 }}>
                  <thead>
                    <tr>
                      <th style={{ ...TH, width: 64, paddingLeft: 12 }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                          <CB
                            on={allSel}
                            onClick={(e) => {
                              e.stopPropagation();
                              togAll();
                            }}
                          />
                        </div>
                      </th>
                      <SortTH
                        label="Patch Name"
                        col="name"
                        sortCol={sortCol}
                        sortDir={sortDir}
                        onSort={toggleSort}
                      />
                      <SortTH
                        label="Severity"
                        col="severity"
                        sortCol={sortCol}
                        sortDir={sortDir}
                        onSort={toggleSort}
                      />
                      <th style={TH}>OS</th>
                      <SortTH
                        label="CVEs"
                        col="cves"
                        sortCol={sortCol}
                        sortDir={sortDir}
                        onSort={toggleSort}
                      />
                      <SortTH
                        label="CVSS"
                        col="cvss"
                        sortCol={sortCol}
                        sortDir={sortDir}
                        onSort={toggleSort}
                      />
                      <SortTH
                        label="Affected"
                        col="affected"
                        sortCol={sortCol}
                        sortDir={sortDir}
                        onSort={toggleSort}
                      />
                      <SortTH
                        label="Published"
                        col="published"
                        sortCol={sortCol}
                        sortDir={sortDir}
                        onSort={toggleSort}
                      />
                      <th style={{ ...TH, width: 36 }} />
                    </tr>
                  </thead>
                  <tbody>
                    {isLoading ? (
                      <SkeletonRows cols={9} />
                    ) : sorted.length === 0 ? (
                      <tr>
                        <td colSpan={9} style={{ padding: 0 }}>
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
                                <path d="M20 7H4a2 2 0 00-2 2v10a2 2 0 002 2h16a2 2 0 002-2V9a2 2 0 00-2-2z" />
                                <path d="M16 21V5a2 2 0 00-2-2h-4a2 2 0 00-2 2v16" />
                              </svg>
                            }
                            title="No patches found"
                            description={
                              hasFilters
                                ? 'Try adjusting your filters.'
                                : 'No patches in catalog yet.'
                            }
                            action={
                              hasFilters
                                ? { label: 'Clear Filters', onClick: clearFilters }
                                : undefined
                            }
                          />
                        </td>
                      </tr>
                    ) : (
                      sorted.map((p) => {
                        const isSel = sel.has(p.id);
                        const isExp = exp.has(p.id);
                        return (
                          <React.Fragment key={p.id}>
                            <tr
                              onClick={() => navigate(`/patches/${p.id}`)}
                              style={{
                                borderBottom: '1px solid var(--border)',
                                cursor: 'pointer',
                                transition: 'background 0.1s',
                                borderLeft:
                                  isSel || isExp
                                    ? '2px solid var(--accent)'
                                    : '2px solid transparent',
                                background: isSel
                                  ? 'color-mix(in srgb, var(--accent) 5%, transparent)'
                                  : 'transparent',
                              }}
                              onMouseEnter={(e) => {
                                if (!isSel)
                                  e.currentTarget.style.background = 'var(--bg-card-hover)';
                              }}
                              onMouseLeave={(e) => {
                                if (!isSel)
                                  e.currentTarget.style.background = isSel
                                    ? 'color-mix(in srgb, var(--accent) 5%, transparent)'
                                    : 'transparent';
                              }}
                            >
                              {/* Checkbox + expand */}
                              <td
                                style={{ ...TD, paddingLeft: 12 }}
                                onClick={(e) => e.stopPropagation()}
                              >
                                <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                                  <button
                                    type="button"
                                    aria-label={isExp ? 'Collapse' : 'Expand'}
                                    onClick={() => togExp(p.id)}
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
                                      width="12"
                                      height="12"
                                      viewBox="0 0 12 12"
                                      fill="none"
                                      stroke="currentColor"
                                      strokeWidth="2"
                                      style={{
                                        transform: isExp ? 'rotate(90deg)' : 'rotate(0deg)',
                                        transition: 'transform 0.2s',
                                      }}
                                    >
                                      <path d="M4.5 2.5L8.5 6L4.5 9.5" />
                                    </svg>
                                  </button>
                                  <CB
                                    on={isSel}
                                    onClick={(e) => {
                                      e.stopPropagation();
                                      togSel(p.id);
                                    }}
                                  />
                                </div>
                              </td>
                              {/* Name */}
                              <td style={TD}>
                                <span
                                  style={{
                                    fontWeight: 600,
                                    color: 'var(--text-emphasis)',
                                    fontSize: 12,
                                  }}
                                >
                                  {p.name}
                                </span>
                              </td>
                              {/* Severity */}
                              <td style={TD}>
                                <span
                                  style={{
                                    color: severityColor(p.severity),
                                    fontSize: 12,
                                    fontWeight: 500,
                                    textTransform: 'capitalize',
                                    display: 'inline-flex',
                                    alignItems: 'center',
                                    gap: 5,
                                  }}
                                >
                                  <span
                                    style={{
                                      width: 6,
                                      height: 6,
                                      borderRadius: '50%',
                                      background: severityColor(p.severity),
                                    }}
                                  />
                                  {p.severity}
                                </span>
                              </td>
                              {/* OS */}
                              <td style={TD}>
                                <span
                                  style={{
                                    fontSize: 12,
                                    color: 'var(--text-secondary)',
                                    textTransform: 'capitalize',
                                  }}
                                >
                                  {p.os_family || '—'}
                                </span>
                              </td>
                              {/* CVEs */}
                              <td style={TD}>
                                <span
                                  style={{
                                    fontFamily: 'var(--font-mono)',
                                    fontSize: 12,
                                    fontWeight: 600,
                                    color:
                                      (p.cve_count ?? 0) > 0
                                        ? 'var(--accent)'
                                        : 'var(--text-faint)',
                                  }}
                                >
                                  {fmtCount(p.cve_count)}
                                </span>
                              </td>
                              {/* CVSS */}
                              <td style={TD}>
                                {(p.highest_cvss_score ?? 0) > 0 ? (
                                  <span
                                    style={{
                                      fontFamily: 'var(--font-mono)',
                                      fontSize: 14,
                                      fontWeight: 700,
                                      color: cvssColor(p.highest_cvss_score),
                                      lineHeight: 1,
                                    }}
                                  >
                                    {p.highest_cvss_score.toFixed(1)}
                                  </span>
                                ) : (
                                  <span
                                    style={{
                                      fontFamily: 'var(--font-mono)',
                                      fontSize: 13,
                                      color: 'var(--text-faint)',
                                    }}
                                  >
                                    —
                                  </span>
                                )}
                              </td>
                              {/* Affected */}
                              <td style={TD}>
                                {(p.affected_endpoint_count ?? 0) > 0 ? (
                                  <div style={{ display: 'flex', alignItems: 'baseline', gap: 4 }}>
                                    <span
                                      style={{
                                        fontFamily: 'var(--font-mono)',
                                        fontSize: 13,
                                        fontWeight: 600,
                                        color: 'var(--text-primary)',
                                      }}
                                    >
                                      {p.affected_endpoint_count}
                                    </span>
                                    {(p.endpoints_deployed_count ?? 0) > 0 && (
                                      <span
                                        style={{
                                          fontFamily: 'var(--font-mono)',
                                          fontSize: 10,
                                          color: 'var(--signal-healthy)',
                                        }}
                                      >
                                        {p.endpoints_deployed_count} deployed
                                      </span>
                                    )}
                                  </div>
                                ) : (
                                  <span
                                    style={{
                                      fontFamily: 'var(--font-mono)',
                                      fontSize: 13,
                                      color: 'var(--text-faint)',
                                    }}
                                  >
                                    —
                                  </span>
                                )}
                              </td>
                              {/* Published */}
                              <td style={TD}>
                                <span
                                  style={{
                                    fontFamily: 'var(--font-mono)',
                                    fontSize: 11,
                                    color: 'var(--text-muted)',
                                  }}
                                  title={absDate(p.released_at ?? p.created_at)}
                                >
                                  {relTime(p.released_at ?? p.created_at)}
                                </span>
                              </td>
                              {/* Actions */}
                              <td style={TD} onClick={(e) => e.stopPropagation()}>
                                <button
                                  type="button"
                                  aria-label={`Deploy ${p.name}`}
                                  onClick={() => openDeploy(p)}
                                  style={{
                                    width: 28,
                                    height: 28,
                                    display: 'flex',
                                    alignItems: 'center',
                                    justifyContent: 'center',
                                    background: 'transparent',
                                    border: '1px solid var(--border)',
                                    borderRadius: 5,
                                    cursor: 'pointer',
                                    color: 'var(--text-muted)',
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
                                  <svg
                                    width="13"
                                    height="13"
                                    viewBox="0 0 24 24"
                                    fill="none"
                                    stroke="currentColor"
                                    strokeWidth="2"
                                  >
                                    <path d="M20 7H4a2 2 0 00-2 2v10a2 2 0 002 2h16a2 2 0 002-2V9a2 2 0 00-2-2z" />
                                    <path d="M16 21V5a2 2 0 00-2-2h-4a2 2 0 00-2 2v16" />
                                  </svg>
                                </button>
                              </td>
                            </tr>
                            {isExp && <ExpandedRow patch={p} onDeploy={openDeploy} colSpan={9} />}
                          </React.Fragment>
                        );
                      })
                    )}
                  </tbody>
                </table>
              </div>
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  padding: '10px 14px',
                  borderTop: '1px solid var(--border)',
                  background: 'var(--bg-inset)',
                  gap: 6,
                }}
              >
                <span
                  style={{
                    flex: 1,
                    fontSize: 11,
                    fontFamily: 'var(--font-mono)',
                    color: 'var(--text-muted)',
                  }}
                >
                  {pagInfo}
                  {filtInfo}
                </span>
                {pagination}
              </div>
            </>
          )}
        </div>
      )}

      <DeploymentWizard open={wizardOpen} onOpenChange={setWizardOpen} initialState={wizardInit} />

      <style>{`@keyframes expandRow { from { max-height: 0; opacity: 0; } to { max-height: 300px; opacity: 1; } }`}</style>
    </div>
  );
};
