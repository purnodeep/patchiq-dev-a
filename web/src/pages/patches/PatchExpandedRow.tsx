import { usePatch } from '../../api/hooks/usePatches';
import type { PatchListItem } from '../../types/patches';

interface PatchExpandedRowProps {
  patch: PatchListItem;
  onCreateDeployment: () => void;
}

const CARD: React.CSSProperties = {
  background: 'var(--bg-inset)',
  border: '1px solid var(--border)',
  borderRadius: 6,
  padding: '14px 16px',
};

const LABEL: React.CSSProperties = {
  fontFamily: 'var(--font-mono)',
  fontSize: 9,
  fontWeight: 600,
  textTransform: 'uppercase',
  letterSpacing: '0.07em',
  color: 'var(--text-muted)',
  marginBottom: 10,
};

function severityColor(s: string): string {
  switch (s.toLowerCase()) {
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

function relativeTime(dateStr: string | undefined | null): string {
  if (!dateStr) return '—';
  const diff = Math.floor((Date.now() - new Date(dateStr).getTime()) / 1000);
  if (diff < 60) return `${diff}s ago`;
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  if (diff < 86400 * 30) return `${Math.floor(diff / 86400)}d ago`;
  if (diff < 86400 * 365) return `${Math.floor(diff / (86400 * 30))}mo ago`;
  return `${Math.floor(diff / (86400 * 365))}y ago`;
}

export const PatchExpandedRow = ({ patch, onCreateDeployment }: PatchExpandedRowProps) => {
  const { data, isLoading } = usePatch(patch.id);

  const rem = data?.remediation;
  const affected = rem
    ? rem.endpoints_affected + rem.endpoints_patched
    : patch.affected_endpoint_count;
  const deployed = rem?.endpoints_patched ?? patch.endpoints_deployed_count;
  const pending = rem?.endpoints_pending ?? Math.max(0, affected - deployed);
  const successRate = affected > 0 ? Math.round((deployed / affected) * 100) : 0;

  return (
    <div
      style={{
        padding: '14px 16px',
        background: 'var(--bg-page)',
        borderTop: '1px solid var(--border)',
        display: 'flex',
        gap: 12,
        alignItems: 'stretch',
      }}
    >
      {/* Patch Info card */}
      <div style={{ ...CARD, flex: '1.2' }}>
        <div style={LABEL}>Patch Info</div>
        <p
          style={{
            fontSize: 12,
            color: 'var(--text-secondary)',
            lineHeight: 1.6,
            margin: '0 0 12px',
          }}
        >
          {patch.description ?? 'No description available.'}
        </p>
        <div
          style={{
            display: 'flex',
            flexWrap: 'wrap',
            gap: '6px 20px',
            borderTop: '1px solid var(--border)',
            paddingTop: 10,
          }}
        >
          {[
            {
              label: 'Severity',
              value: patch.severity.charAt(0).toUpperCase() + patch.severity.slice(1),
              color: severityColor(patch.severity),
            },
            { label: 'OS Family', value: patch.os_family || '—', color: 'var(--text-primary)' },
            { label: 'Version', value: patch.version || '—', color: 'var(--text-primary)' },
            {
              label: 'Released',
              value: relativeTime(patch.released_at),
              color: 'var(--text-muted)',
            },
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
          ].map(({ label, value, color }) => (
            <div key={label} style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
              <span
                style={{
                  fontFamily: 'var(--font-mono)',
                  fontSize: 9,
                  textTransform: 'uppercase',
                  letterSpacing: '0.06em',
                  color: 'var(--text-faint)',
                }}
              >
                {label}
              </span>
              <span
                style={{ fontFamily: 'var(--font-mono)', fontSize: 11, fontWeight: 700, color }}
              >
                {value}
              </span>
            </div>
          ))}
        </div>
      </div>

      {/* Deployment Stats card */}
      <div style={{ ...CARD, flex: '1' }}>
        <div style={LABEL}>Deployment Stats</div>
        {isLoading ? (
          <p style={{ fontSize: 11, color: 'var(--text-muted)', margin: 0 }}>Loading…</p>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
            {[
              {
                label: 'Affected Endpoints',
                value: String(affected),
                color: 'var(--text-primary)',
              },
              {
                label: 'Successfully Deployed',
                value: String(deployed),
                color: 'var(--signal-healthy)',
              },
              {
                label: 'Pending Deployment',
                value: String(pending),
                color: pending > 0 ? 'var(--signal-warning)' : 'var(--text-muted)',
              },
              {
                label: 'Success Rate',
                value: `${successRate}%`,
                color:
                  successRate >= 80
                    ? 'var(--signal-healthy)'
                    : successRate > 0
                      ? 'var(--signal-warning)'
                      : 'var(--text-muted)',
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
              <div
                key={label}
                style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12 }}
              >
                <span style={{ color: 'var(--text-secondary)' }}>{label}</span>
                <span
                  style={{ fontWeight: 600, color, fontFamily: 'var(--font-mono)', fontSize: 11 }}
                >
                  {value}
                </span>
              </div>
            ))}
          </div>
        )}
        {!isLoading && affected > 0 && (
          <div style={{ marginTop: 10, display: 'flex', alignItems: 'center', gap: 8 }}>
            <div
              style={{
                flex: 1,
                height: 3,
                borderRadius: 2,
                background: 'var(--bg-card)',
                overflow: 'hidden',
              }}
            >
              <div
                style={{
                  width: `${successRate}%`,
                  height: '100%',
                  background:
                    successRate >= 80
                      ? 'var(--signal-healthy)'
                      : successRate > 0
                        ? 'var(--signal-warning)'
                        : 'var(--border)',
                  borderRadius: 2,
                  transition: 'width 0.4s',
                }}
              />
            </div>
            <span
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 10,
                color: 'var(--text-muted)',
                flexShrink: 0,
              }}
            >
              {successRate}% deployed
            </span>
          </div>
        )}
      </div>

      {/* Action button — outside cards */}
      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          gap: 6,
          flexShrink: 0,
          width: 140,
          alignSelf: 'start',
        }}
      >
        <button
          type="button"
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            justifyContent: 'center',
            gap: 5,
            padding: '5px 12px',
            borderRadius: 5,
            fontSize: 11,
            fontWeight: 500,
            cursor: 'pointer',
            border: '1px solid var(--accent)',
            background: 'var(--accent)',
            color: 'var(--btn-accent-text, #000)',
            letterSpacing: '0.01em',
            width: '100%',
          }}
          onClick={(e) => {
            e.stopPropagation();
            onCreateDeployment();
          }}
        >
          ⎌ Create Deployment
        </button>
      </div>
    </div>
  );
};
