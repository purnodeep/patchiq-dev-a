import { useState } from 'react';
import { useParams, Link, useNavigate } from 'react-router';
import { ExternalLink, Download, Shield } from 'lucide-react';
import { useCVE } from '../../api/hooks/useCVEs';
import { parseCVSSVector } from '../../lib/cvss';
import type { CVEDetail, AffectedEndpoint } from '../../types/cves';

// ─── Helpers ──────────────────────────────────────────────────────────────────

function formatDate(dateStr: string | null | undefined): string {
  if (!dateStr) return '—';
  return new Date(dateStr).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  });
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
  if (diff < 86400 * 365) return `${Math.floor(diff / (86400 * 30))}mo ago`;
  return `${Math.floor(diff / (86400 * 365))}y ago`;
}

function cvssColor(score: number | null): string {
  if (score === null) return 'var(--text-muted)';
  if (score >= 9) return 'var(--signal-critical)';
  if (score >= 7) return 'var(--signal-warning)';
  if (score >= 4) return 'var(--text-secondary)';
  return 'var(--signal-healthy)';
}

function severityColor(severity: string): string {
  switch (severity.toLowerCase()) {
    case 'critical':
      return 'var(--signal-critical)';
    case 'high':
      return 'var(--signal-warning)';
    case 'medium':
      return 'var(--signal-warning)';
    default:
      return 'var(--text-secondary)';
  }
}

function displaySeverity(s: string): string {
  return s.charAt(0).toUpperCase() + s.slice(1);
}

function firstSeenInWild(cve: CVEDetail): string {
  if (!cve.exploit_available) return 'N/A';
  if (!cve.published_at) return 'Detected (date unavailable)';
  return new Date(cve.published_at).toLocaleString('en-US', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    timeZoneName: 'short',
  });
}

function endpointStatusColor(status: AffectedEndpoint['status']): string {
  switch (status) {
    case 'patched':
      return 'var(--signal-healthy)';
    case 'affected':
      return 'var(--signal-critical)';
    case 'mitigated':
      return 'var(--signal-warning)';
    default:
      return 'var(--text-muted)';
  }
}

// ─── Design Tokens (inline) ──────────────────────────────────────────────────

const mono: React.CSSProperties = { fontFamily: 'var(--font-mono)' };

const thStyle: React.CSSProperties = {
  padding: '9px 12px',
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
};

const sectionLabel: React.CSSProperties = {
  fontFamily: 'var(--font-mono)',
  fontSize: 10,
  fontWeight: 600,
  textTransform: 'uppercase',
  letterSpacing: '0.06em',
  color: 'var(--text-muted)',
  marginBottom: 12,
};

const card: React.CSSProperties = {
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  borderRadius: 8,
  padding: '16px 20px',
  boxShadow: 'var(--shadow-sm)',
};

// ─── Donut Gauge ──────────────────────────────────────────────────────────────

function DonutGauge({ score, max = 10 }: { score: number | null; max?: number }) {
  const val = score ?? 0;
  const pct = val / max;
  const r = 44;
  const cx = 56;
  const cy = 56;
  const circumference = 2 * Math.PI * r;
  const filled = pct * circumference;
  const color = cvssColor(val);

  return (
    <svg width={112} height={112} style={{ display: 'block' }}>
      {/* Track */}
      <circle
        cx={cx}
        cy={cy}
        r={r}
        fill="none"
        stroke="color-mix(in srgb, white 5%, transparent)"
        strokeWidth={10}
      />
      {/* Progress */}
      <circle
        cx={cx}
        cy={cy}
        r={r}
        fill="none"
        stroke={color}
        strokeWidth={10}
        strokeDasharray={`${filled} ${circumference - filled}`}
        strokeDashoffset={circumference / 4}
        strokeLinecap="round"
        style={{ transition: 'stroke-dasharray 0.6s ease' }}
      />
      {/* Center text */}
      <text
        x={cx}
        y={cy - 6}
        textAnchor="middle"
        fill={color}
        fontSize={22}
        fontWeight={700}
        fontFamily="var(--font-mono)"
      >
        {score !== null ? score.toFixed(1) : '—'}
      </text>
      <text
        x={cx}
        y={cy + 12}
        textAnchor="middle"
        fill="var(--text-muted)"
        fontSize={9}
        fontFamily="var(--font-mono)"
      >
        / {max}.0
      </text>
    </svg>
  );
}

// ─── CVSS Vector Hero (inline bars, no Tailwind) ──────────────────────────────

const BAR_WIDTH: Record<string, Record<string, number>> = {
  AV: { N: 100, A: 75, L: 50, P: 25 },
  AC: { L: 100, H: 25 },
  PR: { N: 100, L: 50, H: 25 },
  UI: { N: 100, R: 25 },
  S: { C: 100, U: 50 },
  C: { H: 100, L: 50, N: 0 },
  I: { H: 100, L: 50, N: 0 },
  A: { H: 100, L: 50, N: 0 },
};

const SEVERITY_BAR_COLOR: Record<string, string> = {
  critical: 'var(--signal-critical)',
  high: 'var(--signal-warning)',
  medium: 'var(--signal-warning)',
  low: 'var(--signal-healthy)',
  none: 'var(--text-faint)',
};

function CVSSHero({
  cve,
  onNavigateToPatches,
}: {
  cve: CVEDetail;
  onNavigateToPatches?: () => void;
}) {
  const metrics = parseCVSSVector(cve.cvss_v3_vector);
  const score = cve.cvss_v3_score;

  return (
    <div
      style={{
        display: 'grid',
        gridTemplateColumns: '3fr 2fr',
        gap: 1,
        background: 'var(--border)',
        border: '1px solid var(--border)',
        borderRadius: 8,
        overflow: 'hidden',
        boxShadow: 'var(--shadow-sm)',
      }}
    >
      {/* Left: CVSS Vector Breakdown */}
      <div style={{ background: 'var(--bg-card)', padding: '20px 24px' }}>
        <div style={sectionLabel}>CVSS v3.1 Vector Breakdown</div>
        {metrics.length > 0 ? (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
            {cve.cvss_v3_vector && (
              <div
                style={{
                  ...mono,
                  fontSize: 10,
                  color: 'var(--text-muted)',
                  background: 'var(--bg-inset)',
                  padding: '6px 10px',
                  borderRadius: 4,
                  marginBottom: 8,
                  wordBreak: 'break-all',
                }}
              >
                {cve.cvss_v3_vector}
              </div>
            )}
            {metrics.map((metric) => {
              const barWidth = BAR_WIDTH[metric.key]?.[metric.value] ?? 50;
              const barColor = SEVERITY_BAR_COLOR[metric.severity] ?? 'var(--text-faint)';
              return (
                <div key={metric.key} style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                  <span
                    style={{
                      ...mono,
                      fontSize: 11,
                      color: 'var(--text-muted)',
                      width: 200,
                      flexShrink: 0,
                    }}
                  >
                    {metric.name}
                  </span>
                  <div
                    style={{
                      flex: 1,
                      height: 5,
                      borderRadius: 3,
                      background: 'var(--bg-inset)',
                      overflow: 'hidden',
                    }}
                  >
                    <div
                      style={{
                        height: '100%',
                        borderRadius: 3,
                        width: `${barWidth}%`,
                        background: barColor,
                        transition: 'width 0.4s ease',
                      }}
                    />
                  </div>
                  <span
                    style={{
                      ...mono,
                      fontSize: 11,
                      fontWeight: 600,
                      color: barColor,
                      width: 80,
                      textAlign: 'right',
                      flexShrink: 0,
                    }}
                  >
                    {metric.label}
                  </span>
                </div>
              );
            })}
          </div>
        ) : (
          <p style={{ fontSize: 12, color: 'var(--text-muted)', margin: 0 }}>
            No CVSS vector available.
          </p>
        )}
      </div>

      {/* Right: Threat Assessment */}
      <div style={{ background: 'var(--bg-card)', padding: '20px 24px' }}>
        <div style={sectionLabel}>Threat Assessment</div>
        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            marginBottom: 20,
          }}
        >
          <DonutGauge score={score} />
          <div
            style={{
              ...mono,
              fontSize: 11,
              fontWeight: 700,
              color: cvssColor(score),
              textTransform: 'uppercase',
              letterSpacing: '0.06em',
              marginTop: 6,
            }}
          >
            {score !== null && score >= 9
              ? 'Critical'
              : score !== null && score >= 7
                ? 'High'
                : score !== null && score >= 4
                  ? 'Medium'
                  : 'Low'}
          </div>
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          {[
            {
              label: 'Exploit Status',
              value: cve.exploit_available ? 'Active Exploit' : 'None Known',
              color: cve.exploit_available ? 'var(--signal-critical)' : 'var(--text-secondary)',
              dot: true,
            },
            {
              label: 'KEV Listed',
              value: cve.cisa_kev_due_date ? 'Yes' : 'No',
              color: cve.cisa_kev_due_date ? 'var(--signal-warning)' : 'var(--text-secondary)',
              dot: !!cve.cisa_kev_due_date,
            },
            {
              label: 'Patch Available',
              value: cve.patches.length > 0 ? `${cve.patches.length} available` : 'None',
              color: cve.patches.length > 0 ? 'var(--signal-healthy)' : 'var(--signal-critical)',
              dot: false,
              onClick: cve.patches.length > 0 ? onNavigateToPatches : undefined,
            },
            {
              label: 'Affected Endpoints',
              value: String(cve.affected_endpoints.count),
              color:
                cve.affected_endpoints.count > 0 ? 'var(--text-emphasis)' : 'var(--text-muted)',
              dot: false,
            },
          ].map(({ label, value, color, dot, onClick }) => (
            <div
              key={label}
              onClick={onClick}
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                borderBottom: '1px solid var(--border)',
                paddingBottom: 7,
                cursor: onClick ? 'pointer' : undefined,
              }}
            >
              <span style={{ ...mono, fontSize: 11, color: 'var(--text-muted)' }}>{label}</span>
              <span
                style={{
                  ...mono,
                  fontSize: 11,
                  fontWeight: 700,
                  color,
                  display: 'flex',
                  alignItems: 'center',
                  gap: 4,
                  textDecoration: onClick ? 'underline' : undefined,
                }}
              >
                {dot && (
                  <span
                    style={{
                      width: 6,
                      height: 6,
                      borderRadius: '50%',
                      background: color,
                      display: 'inline-block',
                    }}
                  />
                )}
                {value}
              </span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

// ─── Affected Endpoints + Remediation Path ────────────────────────────────────

function AffectedRemediationRow({ cve }: { cve: CVEDetail }) {
  // Filter out patched endpoints from the affected list
  const unpatched = cve.affected_endpoints.items.filter((ep) => ep.status !== 'patched');
  const epItems = unpatched.slice(0, 8);

  return (
    <div
      style={{
        display: 'grid',
        gridTemplateColumns: '3fr 2fr',
        gap: 1,
        background: 'var(--border)',
        border: '1px solid var(--border)',
        borderRadius: 8,
        overflow: 'hidden',
        boxShadow: 'var(--shadow-sm)',
      }}
    >
      {/* Left: Affected Endpoints */}
      <div style={{ background: 'var(--bg-card)', padding: '16px 20px' }}>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            marginBottom: 12,
          }}
        >
          <div style={sectionLabel}>Affected Endpoints</div>
          <span style={{ ...mono, fontSize: 11, color: 'var(--text-muted)' }}>
            {unpatched.length}
            {cve.affected_endpoints.has_more ? '+' : ''} exposed
          </span>
        </div>
        {epItems.length > 0 ? (
          <>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, marginBottom: 12 }}>
              {epItems.map((ep) => (
                <Link
                  key={ep.id}
                  to={`/endpoints/${ep.id}`}
                  title={`${ep.hostname} — ${ep.status}`}
                  style={{
                    width: 28,
                    height: 28,
                    borderRadius: 4,
                    background:
                      ep.status === 'patched'
                        ? 'color-mix(in srgb, var(--signal-healthy) 12%, transparent)'
                        : 'color-mix(in srgb, var(--signal-critical) 1%, transparent)',
                    border: `1px solid ${ep.status === 'patched' ? 'color-mix(in srgb, var(--signal-healthy) 30%, transparent)' : 'color-mix(in srgb, var(--signal-critical) 1%, transparent)'}`,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    textDecoration: 'none',
                  }}
                >
                  <span
                    style={{
                      width: 8,
                      height: 8,
                      borderRadius: '50%',
                      background: endpointStatusColor(ep.status),
                      display: 'block',
                    }}
                  />
                </Link>
              ))}
              {cve.affected_endpoints.has_more && (
                <div
                  style={{
                    width: 28,
                    height: 28,
                    borderRadius: 4,
                    background: 'var(--bg-inset)',
                    border: '1px solid var(--border)',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    ...mono,
                    fontSize: 9,
                    color: 'var(--text-muted)',
                  }}
                >
                  +{cve.affected_endpoints.count - epItems.length}
                </div>
              )}
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
              {epItems.slice(0, 3).map((ep) => (
                <div
                  key={ep.id}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    gap: 8,
                  }}
                >
                  <Link
                    to={`/endpoints/${ep.id}`}
                    style={{
                      ...mono,
                      fontSize: 11,
                      color: 'var(--text-primary)',
                      textDecoration: 'none',
                    }}
                    onMouseEnter={(e) => (e.currentTarget.style.color = 'var(--accent)')}
                    onMouseLeave={(e) => (e.currentTarget.style.color = 'var(--text-primary)')}
                  >
                    {ep.hostname}
                  </Link>
                  <span
                    style={{
                      ...mono,
                      fontSize: 10,
                      fontWeight: 700,
                      color: endpointStatusColor(ep.status),
                      textTransform: 'capitalize',
                    }}
                  >
                    {ep.status}
                  </span>
                </div>
              ))}
            </div>
          </>
        ) : (
          <p style={{ ...mono, fontSize: 12, color: 'var(--text-muted)', margin: 0 }}>
            No affected endpoints recorded.
          </p>
        )}
      </div>

      {/* Right: Remediation Path */}
      <div style={{ background: 'var(--bg-card)', padding: '16px 20px' }}>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            marginBottom: 12,
          }}
        >
          <div style={sectionLabel}>Available Patches</div>
          <span style={{ ...mono, fontSize: 11, color: 'var(--text-muted)' }}>
            {cve.patches.length} {cve.patches.length === 1 ? 'patch' : 'patches'}
          </span>
        </div>
        {cve.patches.length > 0 ? (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            {cve.patches.map((patch, i) => {
              const pct =
                patch.endpoints_covered > 0
                  ? Math.round((patch.endpoints_patched / patch.endpoints_covered) * 100)
                  : 0;
              return (
                <div
                  key={patch.id}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 10,
                    padding: '8px 0',
                    borderBottom: i < cve.patches.length - 1 ? '1px solid var(--border)' : 'none',
                  }}
                >
                  <div
                    style={{
                      width: 6,
                      height: 6,
                      borderRadius: '50%',
                      background: severityColor(patch.severity),
                      flexShrink: 0,
                    }}
                  />
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <Link
                      to={`/patches/${patch.id}`}
                      style={{
                        ...mono,
                        fontSize: 12,
                        fontWeight: 600,
                        color: 'var(--accent)',
                        textDecoration: 'none',
                        display: 'block',
                        overflow: 'hidden',
                        textOverflow: 'ellipsis',
                        whiteSpace: 'nowrap',
                      }}
                      onMouseEnter={(e) => (e.currentTarget.style.textDecoration = 'underline')}
                      onMouseLeave={(e) => (e.currentTarget.style.textDecoration = 'none')}
                    >
                      {patch.name}
                    </Link>
                    <div
                      style={{
                        height: 3,
                        borderRadius: 2,
                        background: 'var(--bg-inset)',
                        overflow: 'hidden',
                        marginTop: 4,
                      }}
                    >
                      <div
                        style={{
                          height: '100%',
                          borderRadius: 2,
                          width: `${pct}%`,
                          background:
                            pct >= 100
                              ? 'var(--signal-healthy)'
                              : pct < 25
                                ? 'var(--signal-critical)'
                                : 'var(--signal-warning)',
                        }}
                      />
                    </div>
                  </div>
                  <span
                    style={{
                      ...mono,
                      fontSize: 10,
                      color: 'var(--text-muted)',
                      flexShrink: 0,
                    }}
                  >
                    {pct}%
                  </span>
                </div>
              );
            })}
          </div>
        ) : (
          <p style={{ ...mono, fontSize: 12, color: 'var(--text-muted)', margin: 0 }}>
            No patches available yet.
          </p>
        )}
      </div>
    </div>
  );
}

// ─── Timeline (horizontal) ────────────────────────────────────────────────────

function HorizontalTimeline({ cve }: { cve: CVEDetail }) {
  const events: { label: string; date: string; color: string }[] = [];

  if (cve.published_at)
    events.push({
      label: 'Published',
      date: formatDate(cve.published_at),
      color: 'var(--text-muted)',
    });
  if (cve.patches.length > 0 && cve.patches[0].released_at)
    events.push({
      label: 'Patch Released',
      date: formatDate(cve.patches[0].released_at),
      color: 'var(--signal-healthy)',
    });
  if (cve.exploit_available)
    events.push({
      label: 'Exploit Found',
      date: cve.published_at ? formatDate(cve.published_at) : 'Detected',
      color: 'var(--signal-critical)',
    });
  if (cve.cisa_kev_due_date)
    events.push({
      label: 'Added to KEV',
      date: formatDate(cve.cisa_kev_due_date),
      color: 'var(--signal-warning)',
    });
  events.push({
    label: 'NVD Updated',
    date: formatDate(cve.nvd_last_modified),
    color: 'var(--accent)',
  });

  return (
    <div style={{ ...card, padding: '16px 24px' }}>
      <div style={sectionLabel}>CVE Timeline</div>
      <div style={{ display: 'flex', alignItems: 'flex-start', gap: 0, overflowX: 'auto' }}>
        {events.map((ev, i) => (
          <div
            key={ev.label}
            style={{
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              flex: 1,
              minWidth: 100,
              position: 'relative',
            }}
          >
            {/* Connector line */}
            {i < events.length - 1 && (
              <div
                style={{
                  position: 'absolute',
                  top: 9,
                  left: '50%',
                  width: '100%',
                  height: 1,
                  background: 'var(--border)',
                }}
              />
            )}
            {/* Dot */}
            <div
              style={{
                width: 18,
                height: 18,
                borderRadius: '50%',
                background: `${ev.color}20`,
                border: `2px solid ${ev.color}`,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                flexShrink: 0,
                zIndex: 1,
              }}
            >
              <div
                style={{
                  width: 6,
                  height: 6,
                  borderRadius: '50%',
                  background: ev.color,
                }}
              />
            </div>
            {/* Label */}
            <div
              style={{
                marginTop: 8,
                textAlign: 'center',
                display: 'flex',
                flexDirection: 'column',
                gap: 2,
              }}
            >
              <span
                style={{ ...mono, fontSize: 11, fontWeight: 600, color: 'var(--text-primary)' }}
              >
                {ev.label}
              </span>
              <span style={{ ...mono, fontSize: 10, color: 'var(--text-muted)' }}>{ev.date}</span>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

// ─── Tabs ─────────────────────────────────────────────────────────────────────

function AffectedSoftwareTab({ cve }: { cve: CVEDetail }) {
  if (cve.patches.length === 0) {
    return (
      <div
        style={{
          padding: '48px 0',
          textAlign: 'center',
          ...mono,
          fontSize: 12,
          color: 'var(--text-muted)',
        }}
      >
        No affected software data available.
      </div>
    );
  }

  return (
    <div style={{ ...card, padding: 0, overflow: 'hidden' }}>
      <table style={{ width: '100%', borderCollapse: 'collapse' }}>
        <thead>
          <tr>
            <th style={thStyle}>Package / Component</th>
            <th style={thStyle}>Affected Version</th>
            <th style={thStyle}>Fixed Version</th>
            <th style={thStyle}>Endpoints</th>
            <th style={thStyle}>OS</th>
          </tr>
        </thead>
        <tbody>
          {cve.patches.map((patch) => (
            <tr key={patch.id} style={{ borderBottom: '1px solid var(--border)' }}>
              <td style={{ padding: '10px 12px', verticalAlign: 'middle' }}>
                <Link
                  to={`/patches/${patch.id}`}
                  style={{
                    ...mono,
                    fontSize: 12,
                    fontWeight: 600,
                    color: 'var(--accent)',
                    textDecoration: 'none',
                  }}
                  onMouseEnter={(e) => (e.currentTarget.style.textDecoration = 'underline')}
                  onMouseLeave={(e) => (e.currentTarget.style.textDecoration = 'none')}
                >
                  {patch.name}
                </Link>
              </td>
              <td style={{ padding: '10px 12px', verticalAlign: 'middle' }}>
                <span style={{ ...mono, fontSize: 11, color: 'var(--signal-critical)' }}>
                  &lt; {patch.version}
                </span>
              </td>
              <td style={{ padding: '10px 12px', verticalAlign: 'middle' }}>
                <span style={{ ...mono, fontSize: 11, color: 'var(--signal-healthy)' }}>
                  {patch.version}+
                </span>
              </td>
              <td style={{ padding: '10px 12px', verticalAlign: 'middle' }}>
                <span
                  style={{ ...mono, fontSize: 12, fontWeight: 600, color: 'var(--text-primary)' }}
                >
                  {patch.endpoints_covered}
                </span>
              </td>
              <td style={{ padding: '10px 12px', verticalAlign: 'middle' }}>
                <span
                  style={{
                    fontSize: 12,
                    color: 'var(--text-secondary)',
                    textTransform: 'capitalize',
                  }}
                >
                  {patch.os_family}
                </span>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function AffectedEndpointsTab({ cve }: { cve: CVEDetail }) {
  if (cve.affected_endpoints.items.length === 0) {
    return (
      <div
        style={{
          padding: '48px 0',
          textAlign: 'center',
          ...mono,
          fontSize: 12,
          color: 'var(--text-muted)',
        }}
      >
        No affected endpoints.
      </div>
    );
  }

  return (
    <div>
      <div style={{ ...card, padding: 0, overflow: 'hidden' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
          <thead>
            <tr>
              <th style={thStyle}>Hostname</th>
              <th style={thStyle}>OS</th>
              <th style={thStyle}>IP Address</th>
              <th style={thStyle}>Patch Status</th>
              <th style={thStyle}>Agent Version</th>
              <th style={thStyle}>Last Seen</th>
              <th style={thStyle}>Action</th>
            </tr>
          </thead>
          <tbody>
            {cve.affected_endpoints.items.map((ep) => (
              <tr key={ep.id} style={{ borderBottom: '1px solid var(--border)' }}>
                <td style={{ padding: '10px 12px', verticalAlign: 'middle' }}>
                  <Link
                    to={`/endpoints/${ep.id}`}
                    style={{
                      fontSize: 12,
                      fontWeight: 600,
                      color: 'var(--text-emphasis)',
                      textDecoration: 'none',
                    }}
                    onMouseEnter={(e) => (e.currentTarget.style.color = 'var(--accent)')}
                    onMouseLeave={(e) => (e.currentTarget.style.color = 'var(--text-emphasis)')}
                  >
                    {ep.hostname}
                  </Link>
                </td>
                <td style={{ padding: '10px 12px', verticalAlign: 'middle' }}>
                  <span style={{ fontSize: 12, color: 'var(--text-secondary)' }}>
                    {ep.os_family} {ep.os_version}
                  </span>
                </td>
                <td style={{ padding: '10px 12px', verticalAlign: 'middle' }}>
                  <span style={{ ...mono, fontSize: 11, color: 'var(--text-muted)' }}>
                    {ep.ip_address ?? '—'}
                  </span>
                </td>
                <td style={{ padding: '10px 12px', verticalAlign: 'middle' }}>
                  <span
                    style={{
                      fontSize: 11,
                      fontWeight: 600,
                      color: endpointStatusColor(ep.status),
                      textTransform: 'capitalize',
                    }}
                  >
                    {ep.status}
                  </span>
                </td>
                <td style={{ padding: '10px 12px', verticalAlign: 'middle' }}>
                  <span style={{ ...mono, fontSize: 11, color: 'var(--text-muted)' }}>
                    {ep.agent_version ?? '—'}
                  </span>
                </td>
                <td style={{ padding: '10px 12px', verticalAlign: 'middle' }}>
                  <span
                    style={{ ...mono, fontSize: 11, color: 'var(--text-muted)' }}
                    title={ep.last_seen ? new Date(ep.last_seen).toLocaleString() : undefined}
                  >
                    {relativeTime(ep.last_seen)}
                  </span>
                </td>
                <td style={{ padding: '10px 12px', verticalAlign: 'middle' }}>
                  <div style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
                    <Link
                      to={`/endpoints/${ep.id}`}
                      style={{
                        display: 'inline-flex',
                        alignItems: 'center',
                        padding: '4px 10px',
                        background: 'transparent',
                        border: '1px solid var(--border)',
                        borderRadius: 5,
                        fontSize: 11,
                        fontWeight: 500,
                        color: 'var(--text-secondary)',
                        textDecoration: 'none',
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
                      View
                    </Link>
                    {ep.status !== 'patched' && (
                      <Link
                        to="/deployments/new"
                        style={{
                          display: 'inline-flex',
                          alignItems: 'center',
                          padding: '4px 10px',
                          background: 'color-mix(in srgb, var(--accent) 10%, transparent)',
                          border: '1px solid color-mix(in srgb, var(--accent) 30%, transparent)',
                          borderRadius: 5,
                          fontSize: 11,
                          fontWeight: 500,
                          color: 'var(--accent)',
                          textDecoration: 'none',
                          transition: 'all 0.15s',
                        }}
                        onMouseEnter={(e) => {
                          e.currentTarget.style.opacity = '0.85';
                        }}
                        onMouseLeave={(e) => {
                          e.currentTarget.style.opacity = '1';
                        }}
                      >
                        Deploy Fix
                      </Link>
                    )}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {cve.affected_endpoints.has_more && (
        <p style={{ marginTop: 8, ...mono, fontSize: 11, color: 'var(--text-muted)' }}>
          Showing {cve.affected_endpoints.items.length} of {cve.affected_endpoints.count} endpoints
        </p>
      )}
    </div>
  );
}

function RemediationTab({ cve }: { cve: CVEDetail }) {
  const totalCovered = cve.patches.reduce((sum, p) => sum + p.endpoints_covered, 0);
  const totalPatched = cve.patches.reduce((sum, p) => sum + p.endpoints_patched, 0);
  const patchPercentage = totalCovered > 0 ? Math.round((totalPatched / totalCovered) * 100) : 0;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* Progress Summary */}
      <div style={{ ...card, display: 'flex', alignItems: 'center', gap: 16 }}>
        <div style={{ flex: 1 }}>
          <div
            style={{
              fontSize: 12,
              color: 'var(--text-secondary)',
              marginBottom: 8,
              fontWeight: 500,
            }}
          >
            Patch Coverage
          </div>
          <div
            style={{
              height: 6,
              borderRadius: 3,
              background: 'var(--bg-inset)',
              overflow: 'hidden',
            }}
          >
            <div
              style={{
                height: '100%',
                borderRadius: 3,
                width: `${patchPercentage}%`,
                background:
                  patchPercentage >= 80 ? 'var(--signal-healthy)' : 'var(--signal-warning)',
                transition: 'width 0.3s ease',
              }}
            />
          </div>
        </div>
        <div
          style={{
            ...mono,
            fontSize: 22,
            fontWeight: 700,
            color: patchPercentage >= 80 ? 'var(--signal-healthy)' : 'var(--signal-warning)',
            flexShrink: 0,
          }}
        >
          {patchPercentage}%
        </div>
        <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>
          <span style={{ ...mono, fontWeight: 600, color: 'var(--text-primary)' }}>
            {totalPatched}
          </span>{' '}
          of{' '}
          <span style={{ ...mono, fontWeight: 600, color: 'var(--text-primary)' }}>
            {totalCovered}
          </span>{' '}
          endpoints patched
        </div>
      </div>

      {cve.patches.length === 0 ? (
        <p style={{ ...mono, fontSize: 12, color: 'var(--text-muted)' }}>
          No patches available for this CVE.
        </p>
      ) : (
        <div style={{ ...card, padding: 0, overflow: 'hidden' }}>
          <table style={{ width: '100%', borderCollapse: 'collapse' }}>
            <thead>
              <tr>
                <th style={thStyle}>Name</th>
                <th style={thStyle}>OS</th>
                <th style={thStyle}>Severity</th>
                <th style={thStyle}>Released</th>
                <th style={thStyle}>Covered</th>
                <th style={thStyle}>Deployed</th>
                <th style={thStyle}>Progress</th>
                <th style={thStyle}>Action</th>
              </tr>
            </thead>
            <tbody>
              {cve.patches.map((patch) => {
                const pct =
                  patch.endpoints_covered > 0
                    ? Math.round((patch.endpoints_patched / patch.endpoints_covered) * 100)
                    : 0;
                return (
                  <tr key={patch.id} style={{ borderBottom: '1px solid var(--border)' }}>
                    <td style={{ padding: '10px 12px', verticalAlign: 'middle' }}>
                      <Link
                        to={`/patches/${patch.id}`}
                        style={{
                          ...mono,
                          fontSize: 12,
                          fontWeight: 600,
                          color: 'var(--accent)',
                          textDecoration: 'none',
                        }}
                        onMouseEnter={(e) => (e.currentTarget.style.textDecoration = 'underline')}
                        onMouseLeave={(e) => (e.currentTarget.style.textDecoration = 'none')}
                      >
                        {patch.name}
                      </Link>
                    </td>
                    <td style={{ padding: '10px 12px', verticalAlign: 'middle' }}>
                      <span
                        style={{
                          fontSize: 12,
                          color: 'var(--text-secondary)',
                          textTransform: 'capitalize',
                        }}
                      >
                        {patch.os_family}
                      </span>
                    </td>
                    <td style={{ padding: '10px 12px', verticalAlign: 'middle' }}>
                      <span
                        style={{
                          ...mono,
                          fontSize: 11,
                          fontWeight: 600,
                          color: severityColor(patch.severity),
                        }}
                      >
                        {displaySeverity(patch.severity)}
                      </span>
                    </td>
                    <td style={{ padding: '10px 12px', verticalAlign: 'middle' }}>
                      <span style={{ ...mono, fontSize: 11, color: 'var(--text-muted)' }}>
                        {formatDate(patch.released_at)}
                      </span>
                    </td>
                    <td style={{ padding: '10px 12px', verticalAlign: 'middle' }}>
                      <span
                        style={{
                          ...mono,
                          fontSize: 12,
                          fontWeight: 600,
                          color: 'var(--text-primary)',
                        }}
                      >
                        {patch.endpoints_covered}
                      </span>
                    </td>
                    <td style={{ padding: '10px 12px', verticalAlign: 'middle' }}>
                      <span
                        style={{
                          ...mono,
                          fontSize: 12,
                          fontWeight: 600,
                          color: 'var(--signal-healthy)',
                        }}
                      >
                        {patch.endpoints_patched}
                      </span>
                    </td>
                    <td style={{ padding: '10px 12px', verticalAlign: 'middle' }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                        <div
                          style={{
                            width: 64,
                            height: 5,
                            borderRadius: 2.5,
                            background: 'var(--bg-inset)',
                            overflow: 'hidden',
                          }}
                        >
                          <div
                            style={{
                              height: '100%',
                              borderRadius: 2.5,
                              width: `${pct}%`,
                              background:
                                pct >= 80 ? 'var(--signal-healthy)' : 'var(--signal-warning)',
                            }}
                          />
                        </div>
                        <span style={{ ...mono, fontSize: 10, color: 'var(--text-muted)' }}>
                          {pct}%
                        </span>
                      </div>
                    </td>
                    <td style={{ padding: '10px 12px', verticalAlign: 'middle' }}>
                      <Link
                        to="/deployments/new"
                        style={{
                          display: 'inline-flex',
                          alignItems: 'center',
                          padding: '4px 10px',
                          background: 'transparent',
                          border: '1px solid var(--border)',
                          borderRadius: 5,
                          fontSize: 11,
                          fontWeight: 500,
                          color: 'var(--text-secondary)',
                          textDecoration: 'none',
                          transition: 'all 0.15s',
                        }}
                        onMouseEnter={(e) => {
                          e.currentTarget.style.borderColor = 'var(--accent)';
                          e.currentTarget.style.color = 'var(--accent)';
                        }}
                        onMouseLeave={(e) => {
                          e.currentTarget.style.borderColor = 'var(--border)';
                          e.currentTarget.style.color = 'var(--text-secondary)';
                        }}
                      >
                        Deploy
                      </Link>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {/* Workaround */}
      <div style={card}>
        <div style={sectionLabel}>Workaround Steps</div>
        <ol
          style={{ paddingLeft: 20, margin: 0, display: 'flex', flexDirection: 'column', gap: 6 }}
        >
          {((): string[] => {
            const steps: string[] = [];
            if (cve.attack_vector === 'Network') {
              steps.push(
                `Restrict network access to services affected by ${cve.cve_id} via firewall rules or network segmentation`,
              );
            } else if (cve.attack_vector === 'Local') {
              steps.push(
                `Limit local access to affected endpoints — restrict user privileges and disable unnecessary local accounts`,
              );
            } else {
              steps.push(
                `Restrict access to components affected by ${cve.cve_id} using your existing access control policies`,
              );
            }
            steps.push(
              `Enable enhanced security monitoring and alerting on all endpoints affected by ${cve.cve_id}`,
            );
            if (cve.exploit_available) {
              steps.push(
                `Prioritize immediate patching — an active exploit exists for this vulnerability. Treat as P0 until patched`,
              );
            }
            if (cve.cisa_kev_due_date) {
              steps.push(
                `CISA KEV remediation deadline: ${new Date(cve.cisa_kev_due_date).toLocaleDateString('en-US', { year: 'numeric', month: 'short', day: 'numeric' })} — escalate if not yet scheduled`,
              );
            }
            steps.push(
              `Apply compensating controls per your incident response playbook until ${cve.patches[0]?.name ?? 'a patch'} can be deployed`,
            );
            return steps;
          })().map((step, i) => (
            <li key={i} style={{ fontSize: 12, color: 'var(--text-secondary)', lineHeight: 1.6 }}>
              {step}
            </li>
          ))}
        </ol>
      </div>
    </div>
  );
}

function IntelligenceTab({ cve }: { cve: CVEDetail }) {
  const exploitComplexity = (() => {
    if (!cve.cvss_v3_vector) return '—';
    const match = cve.cvss_v3_vector.match(/AC:([HL])/);
    if (!match) return '—';
    return match[1] === 'L' ? 'Low' : 'High';
  })();

  return (
    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 20 }}>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
        <div style={card}>
          <div style={sectionLabel}>Exploit Details</div>
          <div style={{ display: 'flex', flexDirection: 'column' }}>
            {[
              {
                label: 'Exploit Status',
                value: cve.exploit_available ? 'Active Exploit' : 'None Known',
                color: cve.exploit_available ? 'var(--signal-critical)' : 'var(--text-secondary)',
              },
              { label: 'Exploit Type', value: cve.cwe_id ?? '—', color: 'var(--text-primary)' },
              {
                label: 'Exploit Complexity',
                value: exploitComplexity,
                color:
                  exploitComplexity === 'Low' ? 'var(--signal-critical)' : 'var(--text-secondary)',
              },
              {
                label: 'Threat Level',
                value: displaySeverity(cve.severity),
                color: severityColor(cve.severity),
              },
              {
                label: 'First Seen in Wild',
                value: firstSeenInWild(cve),
                color: 'var(--text-secondary)',
              },
              { label: 'Threat Actors', value: 'Data not available', color: 'var(--text-muted)' },
            ].map(({ label, value, color }) => (
              <div
                key={label}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  borderBottom: '1px solid var(--border)',
                  padding: '8px 0',
                  fontSize: 12,
                }}
              >
                <span style={{ color: 'var(--text-secondary)' }}>{label}</span>
                <span style={{ fontWeight: 600, color }}>{value}</span>
              </div>
            ))}
          </div>
        </div>
        <div style={card}>
          <div style={sectionLabel}>Threat Description</div>
          <p style={{ fontSize: 12, color: 'var(--text-secondary)', lineHeight: 1.7, margin: 0 }}>
            {cve.description ?? 'No threat intelligence data available for this vulnerability.'}
          </p>
        </div>
      </div>

      <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
        <div style={card}>
          <div style={sectionLabel}>Related CVEs</div>
          {cve.related_cves.length > 0 ? (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
              {cve.related_cves.map((rc) => (
                <Link
                  key={rc.id}
                  to={`/cves/${rc.id}`}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    padding: '8px 12px',
                    borderRadius: 6,
                    background: 'var(--bg-inset)',
                    border: '1px solid transparent',
                    textDecoration: 'none',
                    transition: 'all 0.15s',
                  }}
                  onMouseEnter={(e) => {
                    e.currentTarget.style.background = 'var(--bg-card-hover)';
                    e.currentTarget.style.borderColor = 'var(--border)';
                  }}
                  onMouseLeave={(e) => {
                    e.currentTarget.style.background = 'var(--bg-inset)';
                    e.currentTarget.style.borderColor = 'transparent';
                  }}
                >
                  <div>
                    <span
                      style={{ ...mono, fontSize: 12, fontWeight: 600, color: 'var(--accent)' }}
                    >
                      {rc.cve_id}
                    </span>
                    <span style={{ marginLeft: 8, fontSize: 10, color: 'var(--text-muted)' }}>
                      Related via shared patch
                    </span>
                  </div>
                  <span
                    style={{ fontSize: 11, fontWeight: 600, color: severityColor(rc.severity) }}
                  >
                    {displaySeverity(rc.severity)}
                  </span>
                </Link>
              ))}
            </div>
          ) : (
            <p style={{ fontSize: 12, color: 'var(--text-muted)' }}>No related CVEs found.</p>
          )}
        </div>

        <div style={card}>
          <div style={sectionLabel}>Recommended Actions (Priority Order)</div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
            {[
              {
                num: '1',
                dotColor: 'var(--signal-critical)',
                text: `Apply ${cve.patches[0]?.name ?? 'available patches'} to all affected endpoints immediately`,
              },
              {
                num: '2',
                dotColor: 'var(--signal-warning)',
                text: `Restrict access to affected ${cve.patches[0]?.os_family ?? ''} components`.trim(),
              },
              {
                num: '3',
                dotColor: 'var(--signal-warning)',
                text: 'Review and apply network segmentation for affected endpoints',
              },
              {
                num: '4',
                dotColor: 'var(--accent)',
                text: `Monitor for exploitation attempts related to ${cve.cve_id}`,
              },
            ].map((action) => (
              <div key={action.num} style={{ display: 'flex', alignItems: 'flex-start', gap: 10 }}>
                <div
                  style={{
                    width: 20,
                    height: 20,
                    borderRadius: '50%',
                    background: `${action.dotColor}18`,
                    border: `1px solid ${action.dotColor}40`,
                    flexShrink: 0,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    ...mono,
                    fontSize: 9,
                    fontWeight: 700,
                    color: action.dotColor,
                  }}
                >
                  {action.num}
                </div>
                <span style={{ fontSize: 12, color: 'var(--text-secondary)', lineHeight: 1.6 }}>
                  {action.text}
                </span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

// ─── Health Strip ─────────────────────────────────────────────────────────────

function HealthStrip({ cve }: { cve: CVEDetail }) {
  const score = cve.cvss_v3_score;
  const scoreColor = cvssColor(score);

  const metrics = [
    {
      label: 'CVSS',
      value: score !== null ? `${score.toFixed(1)} / 10` : '—',
      valueColor: scoreColor,
      bar: score !== null ? (score / 10) * 100 : 0,
      barColor: scoreColor,
    },
    {
      label: 'Endpoints',
      value: `${cve.affected_endpoints.count} affected`,
      valueColor: cve.affected_endpoints.count > 0 ? 'var(--signal-critical)' : 'var(--text-muted)',
      bar: null,
      barColor: null,
    },
    {
      label: 'Patches',
      value: cve.patches.length > 0 ? `${cve.patches.length} available` : 'None',
      valueColor: cve.patches.length > 0 ? 'var(--signal-healthy)' : 'var(--text-muted)',
      bar: null,
      barColor: null,
    },
    {
      label: 'Published',
      value: relativeTime(cve.published_at),
      valueColor: 'var(--text-primary)',
      bar: null,
      barColor: null,
    },
  ];

  return (
    <div
      style={{
        display: 'flex',
        height: 56,
        border: '1px solid var(--border)',
        borderRadius: 8,
        overflow: 'hidden',
        background: 'var(--bg-card)',
        boxShadow: 'var(--shadow-sm)',
      }}
    >
      {metrics.map((m, i) => (
        <div
          key={m.label}
          style={{
            flex: 1,
            display: 'flex',
            alignItems: 'center',
            gap: 12,
            padding: '0 20px',
            borderLeft: i > 0 ? '1px solid var(--border)' : 'none',
          }}
        >
          <span
            style={{
              ...mono,
              fontSize: 10,
              fontWeight: 600,
              textTransform: 'uppercase',
              letterSpacing: '0.05em',
              color: 'var(--text-muted)',
              flexShrink: 0,
            }}
          >
            {m.label}
          </span>
          <span
            style={{
              ...mono,
              fontSize: 13,
              fontWeight: 700,
              color: m.valueColor,
              flexShrink: 0,
            }}
          >
            {m.value}
          </span>
          {m.bar !== null && m.barColor && (
            <div
              style={{
                flex: 1,
                height: 4,
                borderRadius: 2,
                background: 'var(--bg-inset)',
                overflow: 'hidden',
              }}
            >
              <div
                style={{
                  height: '100%',
                  borderRadius: 2,
                  width: `${m.bar}%`,
                  background: m.barColor,
                  transition: 'width 0.4s ease',
                }}
              />
            </div>
          )}
        </div>
      ))}
    </div>
  );
}

// ─── Main Page ────────────────────────────────────────────────────────────────

const TABS = [
  { value: 'overview', label: 'Overview' },
  { value: 'software', label: 'Affected Software' },
  { value: 'endpoints', label: 'Endpoints' },
  { value: 'remediation', label: 'Available Patches' },
  { value: 'intelligence', label: 'Intelligence' },
] as const;

type TabValue = (typeof TABS)[number]['value'];

export const CVEDetailPage = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { data: cve, isLoading, isError, refetch } = useCVE(id ?? '');
  const [activeTab, setActiveTab] = useState<TabValue>('overview');

  if (!id) {
    return (
      <div style={{ padding: 24, ...mono, fontSize: 13, color: 'var(--signal-critical)' }}>
        CVE not found
      </div>
    );
  }

  if (isLoading) {
    return (
      <div
        style={{
          padding: '20px 24px',
          display: 'flex',
          flexDirection: 'column',
          gap: 16,
          background: 'var(--bg-page)',
          minHeight: '100%',
        }}
      >
        {[48, 56, 280, 200].map((h, i) => (
          <div
            key={i}
            style={{
              height: h,
              borderRadius: 8,
              background: 'var(--bg-card)',
              border: '1px solid var(--border)',
              animation: 'pulse 1.5s ease-in-out infinite',
            }}
          />
        ))}
      </div>
    );
  }

  if (isError || !cve) {
    return (
      <div style={{ padding: '20px 24px' }}>
        <div
          style={{
            background: 'color-mix(in srgb, var(--signal-critical) 1%, transparent)',
            border: '1px solid color-mix(in srgb, var(--signal-critical) 1%, transparent)',
            borderRadius: 8,
            padding: '12px 16px',
            ...mono,
            fontSize: 13,
            color: 'var(--signal-critical)',
          }}
        >
          Failed to load CVE data.{' '}
          <button
            onClick={() => refetch()}
            style={{
              ...mono,
              fontSize: 12,
              color: 'var(--signal-critical)',
              textDecoration: 'underline',
              background: 'none',
              border: 'none',
              cursor: 'pointer',
              padding: 0,
            }}
          >
            Retry
          </button>
        </div>
      </div>
    );
  }

  const score = cve.cvss_v3_score;

  return (
    <div
      style={{
        padding: '20px 24px',
        display: 'flex',
        flexDirection: 'column',
        gap: 16,
        minHeight: '100%',
        background: 'var(--bg-page)',
      }}
    >
      {/* ── Header (2-row, no card) ── */}
      <div>
        {/* Row 1: ID + actions */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            gap: 16,
            marginBottom: 8,
          }}
        >
          <h1
            style={{
              ...mono,
              fontSize: 28,
              fontWeight: 700,
              color: 'var(--text-emphasis)',
              margin: 0,
              letterSpacing: '0.01em',
            }}
          >
            {cve.cve_id}
          </h1>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexShrink: 0 }}>
            {cve.patches.length > 0 && (
              <button
                type="button"
                onClick={() => navigate('/deployments/new')}
                style={{
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: 6,
                  padding: '7px 14px',
                  background: 'var(--accent)',
                  color: 'var(--btn-accent-text, #000)',
                  border: 'none',
                  borderRadius: 6,
                  fontSize: 12,
                  fontWeight: 600,
                  cursor: 'pointer',
                  transition: 'opacity 0.15s',
                  ...mono,
                }}
                onMouseEnter={(e) => (e.currentTarget.style.opacity = '0.85')}
                onMouseLeave={(e) => (e.currentTarget.style.opacity = '1')}
              >
                <svg
                  width="12"
                  height="12"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2.5"
                >
                  <path d="M5 12h14M12 5l7 7-7 7" />
                </svg>
                Deploy Fix
              </button>
            )}
            {cve.external_references.length > 0 && (
              <a
                href={cve.external_references[0].url}
                target="_blank"
                rel="noopener noreferrer"
                style={{
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: 6,
                  padding: '7px 14px',
                  background: 'transparent',
                  color: 'var(--text-secondary)',
                  border: '1px solid var(--border)',
                  borderRadius: 6,
                  fontSize: 12,
                  fontWeight: 500,
                  cursor: 'pointer',
                  textDecoration: 'none',
                  transition: 'all 0.15s',
                  ...mono,
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
                <ExternalLink size={12} />
                NVD
              </a>
            )}
            <button
              type="button"
              onClick={() => {
                const blob = new Blob([JSON.stringify(cve, null, 2)], { type: 'application/json' });
                const url = URL.createObjectURL(blob);
                const a = document.createElement('a');
                a.href = url;
                a.download = `${cve.cve_id}-report.json`;
                a.click();
                URL.revokeObjectURL(url);
              }}
              style={{
                display: 'inline-flex',
                alignItems: 'center',
                gap: 6,
                padding: '7px 14px',
                background: 'transparent',
                color: 'var(--text-muted)',
                border: '1px solid var(--border)',
                borderRadius: 6,
                fontSize: 12,
                fontWeight: 500,
                cursor: 'pointer',
                transition: 'all 0.15s',
                ...mono,
              }}
              onMouseEnter={(e) => {
                e.currentTarget.style.color = 'var(--text-secondary)';
                e.currentTarget.style.borderColor = 'var(--border-hover)';
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.color = 'var(--text-muted)';
                e.currentTarget.style.borderColor = 'var(--border)';
              }}
            >
              <Download size={12} />
              Export
            </button>
          </div>
        </div>

        {/* Row 2: CVSS score chip + status + metadata chips */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap' }}>
          {/* CVSS big number */}
          <span
            style={{
              ...mono,
              fontSize: 22,
              fontWeight: 800,
              color: cvssColor(score),
              letterSpacing: '-0.01em',
            }}
          >
            {score !== null ? `CVSS ${score.toFixed(1)}` : 'No CVSS'}
          </span>

          {/* Severity dot */}
          <span style={{ display: 'flex', alignItems: 'center', gap: 5 }}>
            <span
              style={{
                width: 7,
                height: 7,
                borderRadius: '50%',
                background: severityColor(cve.severity),
                display: 'inline-block',
              }}
            />
            <span
              style={{
                ...mono,
                fontSize: 12,
                fontWeight: 700,
                color: severityColor(cve.severity),
              }}
            >
              {displaySeverity(cve.severity)}
            </span>
          </span>

          {/* Metadata chips */}
          {cve.attack_vector && (
            <span style={{ ...chipStyle }}>
              <Shield size={10} style={{ flexShrink: 0 }} />
              {cve.attack_vector}
            </span>
          )}
          {cve.exploit_available && (
            <span
              style={{
                ...chipStyle,
                color: 'var(--signal-critical)',
                borderColor: 'color-mix(in srgb, var(--signal-critical) 1%, transparent)',
                background: 'color-mix(in srgb, var(--signal-critical) 1%, transparent)',
              }}
            >
              Active Exploit
            </span>
          )}
          {cve.cisa_kev_due_date && (
            <span
              style={{
                ...chipStyle,
                color: 'var(--signal-warning)',
                borderColor: 'color-mix(in srgb, var(--signal-warning) 30%, transparent)',
                background: 'color-mix(in srgb, var(--signal-warning) 8%, transparent)',
              }}
            >
              KEV: Listed
            </span>
          )}
          {cve.cwe_id && <span style={{ ...chipStyle }}>{cve.cwe_id}</span>}
        </div>
      </div>

      {/* ── Health Strip ── */}
      <HealthStrip cve={cve} />

      {/* ── Tabs ── */}
      <div>
        <div
          style={{
            display: 'flex',
            borderBottom: '1px solid var(--border)',
            gap: 0,
            marginBottom: 16,
          }}
        >
          {TABS.map((tab) => {
            const isActive = activeTab === tab.value;
            return (
              <button
                key={tab.value}
                onClick={() => setActiveTab(tab.value)}
                style={{
                  fontSize: 13,
                  fontWeight: isActive ? 600 : 400,
                  color: isActive ? 'var(--text-emphasis)' : 'var(--text-muted)',
                  padding: '8px 16px',
                  background: 'none',
                  border: 'none',
                  borderBottom: isActive ? '2px solid var(--accent)' : '2px solid transparent',
                  cursor: 'pointer',
                  transition: 'color 150ms ease, border-color 150ms ease',
                  marginBottom: -1,
                }}
                onMouseEnter={(e) => {
                  if (!isActive)
                    (e.currentTarget as HTMLButtonElement).style.color = 'var(--text-secondary)';
                }}
                onMouseLeave={(e) => {
                  if (!isActive)
                    (e.currentTarget as HTMLButtonElement).style.color = 'var(--text-muted)';
                }}
              >
                {tab.label}
              </button>
            );
          })}
        </div>

        {activeTab === 'overview' && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
            <CVSSHero cve={cve} onNavigateToPatches={() => setActiveTab('remediation')} />
            <AffectedRemediationRow cve={cve} />
            <HorizontalTimeline cve={cve} />
          </div>
        )}
        {activeTab === 'software' && <AffectedSoftwareTab cve={cve} />}
        {activeTab === 'endpoints' && <AffectedEndpointsTab cve={cve} />}
        {activeTab === 'remediation' && <RemediationTab cve={cve} />}
        {activeTab === 'intelligence' && <IntelligenceTab cve={cve} />}
      </div>
    </div>
  );
};

// ─── Chip style ───────────────────────────────────────────────────────────────

const chipStyle: React.CSSProperties = {
  display: 'inline-flex',
  alignItems: 'center',
  gap: 4,
  padding: '2px 8px',
  border: '1px solid var(--border)',
  borderRadius: 4,
  fontFamily: 'var(--font-mono)',
  fontSize: 11,
  fontWeight: 500,
  color: 'var(--text-muted)',
  background: 'var(--bg-inset)',
};
