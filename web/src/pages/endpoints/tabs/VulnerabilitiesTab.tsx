import { useState } from 'react';
import { Skeleton } from '@patchiq/ui';
import { Link } from 'react-router';
import { useEndpointCVEs } from '../../../api/hooks/useEndpoints';
import { timeAgo } from '../../../lib/time';

interface VulnerabilitiesTabProps {
  endpointId: string;
  vulnerableCveCount?: number;
}

type SeverityFilter = 'all' | 'critical' | 'high' | 'medium' | 'low';

// ── design tokens ──────────────────────────────────────────────
const S = {
  card: {
    background: 'var(--bg-card)',
    border: '1px solid var(--border)',
    borderRadius: 8,
    boxShadow: 'var(--shadow-sm)',
    overflow: 'hidden' as const,
  },
  cardTitle: {
    fontFamily: 'var(--font-mono)',
    fontSize: 10,
    fontWeight: 500,
    textTransform: 'uppercase' as const,
    letterSpacing: '0.05em',
    color: 'var(--text-muted)',
    padding: '12px 16px',
    borderBottom: '1px solid var(--border)',
    background: 'var(--bg-inset)',
  },
  th: {
    fontFamily: 'var(--font-mono)',
    fontSize: 10,
    fontWeight: 500,
    textTransform: 'uppercase' as const,
    letterSpacing: '0.05em',
    color: 'var(--text-muted)',
    padding: '9px 12px',
    background: 'var(--bg-inset)',
    borderBottom: '1px solid var(--border)',
    textAlign: 'left' as const,
    whiteSpace: 'nowrap' as const,
  },
  td: {
    padding: '10px 12px',
    borderBottom: '1px solid var(--border)',
    color: 'var(--text-primary)',
    fontSize: 13,
  },
};

const SEVERITY_COLOR: Record<string, string> = {
  critical: 'var(--signal-critical)',
  high: 'var(--signal-warning)',
  medium: 'var(--text-secondary)',
  low: 'var(--text-faint)',
  none: 'var(--text-faint)',
};

function cvssColor(score: number): string {
  if (score >= 9) return 'var(--signal-critical)';
  if (score >= 7) return 'var(--signal-warning)';
  if (score >= 4) return 'var(--text-secondary)';
  return 'var(--text-faint)';
}

export function VulnerabilitiesTab({ endpointId }: VulnerabilitiesTabProps) {
  const [severityFilter, setSeverityFilter] = useState<SeverityFilter>('all');
  const { data, isLoading, error } = useEndpointCVEs(endpointId);

  const cves = data?.data ?? [];

  const counts = {
    critical: cves.filter((c) => c.cve_severity === 'critical').length,
    high: cves.filter((c) => c.cve_severity === 'high').length,
    medium: cves.filter((c) => c.cve_severity === 'medium').length,
    low: cves.filter((c) => c.cve_severity === 'low').length,
  };

  const filtered =
    severityFilter === 'all' ? cves : cves.filter((c) => c.cve_severity === severityFilter);

  if (isLoading) {
    return (
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        <Skeleton className="h-28 w-full rounded-lg" />
        {Array.from({ length: 6 }).map((_, i) => (
          <Skeleton key={i} className="h-10 w-full rounded" />
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <div style={{ ...S.card, padding: 16 }}>
        <span style={{ fontSize: 13, color: 'var(--signal-critical)' }}>
          Failed to load CVE data.
        </span>
      </div>
    );
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* Summary stat cards */}
      <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap' as const }}>
        {[
          { label: 'Total CVEs', count: cves.length, color: 'var(--text-emphasis)' },
          { label: 'Critical', count: counts.critical, color: 'var(--signal-critical)' },
          { label: 'High', count: counts.high, color: 'var(--signal-warning)' },
          {
            label: 'Medium + Low',
            count: counts.medium + counts.low,
            color: 'var(--text-secondary)',
          },
        ].map(({ label, count, color }) => (
          <div
            key={label}
            style={{
              flex: '1 1 120px',
              background: 'var(--bg-card)',
              border: '1px solid var(--border)',
              borderRadius: 8,
              padding: '14px 16px',
            }}
          >
            <div
              style={{
                fontSize: 10,
                fontFamily: 'var(--font-mono)',
                color: 'var(--text-muted)',
                textTransform: 'uppercase' as const,
                letterSpacing: '0.04em',
              }}
            >
              {label}
            </div>
            <div
              style={{
                fontSize: 26,
                fontWeight: 700,
                fontFamily: 'var(--font-mono)',
                color,
                lineHeight: 1,
                marginTop: 4,
              }}
            >
              {count}
            </div>
          </div>
        ))}
      </div>

      {/* Severity filter pills */}
      <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' as const }}>
        {(
          [
            { key: 'all', label: 'All', count: cves.length },
            { key: 'critical', label: 'Critical', count: counts.critical },
            { key: 'high', label: 'High', count: counts.high },
            { key: 'medium', label: 'Medium', count: counts.medium },
            { key: 'low', label: 'Low', count: counts.low },
          ] as const
        ).map(({ key, label, count }) => {
          const active = severityFilter === key;
          const color = key === 'all' ? 'var(--text-primary)' : SEVERITY_COLOR[key];
          return (
            <button
              key={key}
              onClick={() => setSeverityFilter(key)}
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 11,
                padding: '4px 10px',
                borderRadius: 4,
                border: `1px solid ${active ? color : 'var(--border)'}`,
                background: active ? 'var(--bg-inset)' : 'var(--bg-card)',
                color: active ? color : 'var(--text-muted)',
                cursor: 'pointer',
                transition: 'all 0.1s',
              }}
            >
              {label} {count}
            </button>
          );
        })}
      </div>

      {/* CVE table */}
      {filtered.length === 0 ? (
        <div style={{ ...S.card, padding: 48, textAlign: 'center' as const }}>
          <span style={{ fontSize: 13, color: 'var(--text-muted)' }}>
            {cves.length === 0
              ? 'No CVEs found. Run a scan to populate vulnerability data.'
              : 'No CVEs match the current filter.'}
          </span>
        </div>
      ) : (
        <div style={S.card}>
          <div
            style={{
              ...S.cardTitle,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
            }}
          >
            <span>CVE Exposure{severityFilter !== 'all' ? ` — ${severityFilter}` : ''}</span>
            <span style={{ color: 'var(--text-faint)' }}>{filtered.length} CVEs</span>
          </div>
          <div style={{ overflowX: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse' }}>
              <thead>
                <tr>
                  <th style={S.th}>CVE ID</th>
                  <th style={S.th}>CVSS</th>
                  <th style={S.th}>Severity</th>
                  <th style={S.th}>Exploit</th>
                  <th style={S.th}>KEV</th>
                  <th style={S.th}>Status</th>
                  <th style={S.th}>Detected</th>
                </tr>
              </thead>
              <tbody>
                {filtered.map((cve) => {
                  const sevColor = SEVERITY_COLOR[cve.cve_severity ?? ''] ?? 'var(--text-muted)';
                  const score = cve.cvss_v3_score;
                  const isHighRisk = score != null && score >= 9.0;
                  return (
                    <tr
                      key={cve.id}
                      onMouseEnter={(e) => {
                        (e.currentTarget as HTMLTableRowElement).style.background =
                          'var(--bg-card-hover)';
                      }}
                      onMouseLeave={(e) => {
                        (e.currentTarget as HTMLTableRowElement).style.background = '';
                      }}
                    >
                      <td style={S.td}>
                        <Link
                          to={`/cves/${cve.cve_id}`}
                          style={{
                            fontFamily: 'var(--font-mono)',
                            fontSize: 12,
                            color: 'var(--accent)',
                            textDecoration: 'none',
                            fontWeight: 500,
                          }}
                          onMouseEnter={(e) => {
                            (e.currentTarget as HTMLAnchorElement).style.textDecoration =
                              'underline';
                          }}
                          onMouseLeave={(e) => {
                            (e.currentTarget as HTMLAnchorElement).style.textDecoration = 'none';
                          }}
                        >
                          {cve.cve_identifier}
                        </Link>
                      </td>
                      <td style={S.td}>
                        {score != null ? (
                          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                            <div
                              style={{
                                width: 40,
                                height: 3,
                                background: 'var(--border)',
                                borderRadius: 2,
                                overflow: 'hidden',
                              }}
                            >
                              <div
                                style={{
                                  height: '100%',
                                  width: `${(score / 10) * 100}%`,
                                  background: cvssColor(score),
                                  borderRadius: 2,
                                }}
                              />
                            </div>
                            <span
                              style={{
                                fontFamily: 'var(--font-mono)',
                                fontSize: 12,
                                color: cvssColor(score),
                                fontWeight: 600,
                              }}
                            >
                              {score.toFixed(1)}
                            </span>
                          </div>
                        ) : (
                          <span style={{ color: 'var(--text-faint)' }}>—</span>
                        )}
                      </td>
                      <td style={S.td}>
                        <span
                          style={{ fontFamily: 'var(--font-mono)', fontSize: 12, color: sevColor }}
                        >
                          {cve.cve_severity || '—'}
                        </span>
                      </td>
                      <td style={S.td}>
                        {isHighRisk ? (
                          <span
                            style={{
                              fontFamily: 'var(--font-mono)',
                              fontSize: 11,
                              color: 'var(--signal-critical)',
                            }}
                          >
                            yes
                          </span>
                        ) : (
                          <span style={{ color: 'var(--text-faint)' }}>—</span>
                        )}
                      </td>
                      {/* TODO(PIQ-243): Show KEV flag when API exposes kev_due_date */}
                      <td style={{ ...S.td, color: 'var(--text-faint)' }}>—</td>
                      <td style={S.td}>
                        <span
                          style={{
                            display: 'inline-flex',
                            alignItems: 'center',
                            gap: 5,
                            fontSize: 12,
                            color:
                              cve.status === 'patched'
                                ? 'var(--signal-healthy)'
                                : cve.status === 'unpatched'
                                  ? 'var(--signal-critical)'
                                  : 'var(--text-muted)',
                          }}
                        >
                          <span
                            style={{
                              width: 6,
                              height: 6,
                              borderRadius: '50%',
                              background:
                                cve.status === 'patched'
                                  ? 'var(--signal-healthy)'
                                  : cve.status === 'unpatched'
                                    ? 'var(--signal-critical)'
                                    : 'var(--border)',
                            }}
                          />
                          {cve.status}
                        </span>
                      </td>
                      <td
                        style={{
                          ...S.td,
                          fontFamily: 'var(--font-mono)',
                          fontSize: 11,
                          color: 'var(--text-muted)',
                        }}
                      >
                        {timeAgo(cve.detected_at)}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
