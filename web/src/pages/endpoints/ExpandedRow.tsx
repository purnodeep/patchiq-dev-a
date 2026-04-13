import { Link } from 'react-router';
import { Zap, RefreshCw } from 'lucide-react';
import { useEndpoint, useEndpointPatches, useTriggerScan } from '../../api/hooks/useEndpoints';

interface ExpandedRowProps {
  endpointId: string;
  colSpan: number;
}

const severityOrder: Record<string, number> = { critical: 0, high: 1, medium: 2, low: 3 };

function metricColor(pct: number): string {
  if (pct >= 80) return 'var(--signal-critical)';
  if (pct >= 60) return 'var(--signal-warning)';
  return 'var(--signal-healthy)';
}

/** Compact 3px horizontal bar. */
function MiniBar({ pct, color }: { pct: number; color: string }) {
  return (
    <div
      style={{
        height: 3,
        borderRadius: 2,
        background: 'var(--bg-inset)',
        overflow: 'hidden',
        marginTop: 3,
      }}
    >
      <div
        style={{
          height: '100%',
          width: `${Math.min(100, pct)}%`,
          background: color,
          borderRadius: 2,
        }}
      />
    </div>
  );
}

export function ExpandedRow({ endpointId, colSpan }: ExpandedRowProps) {
  const { data: detail, isLoading, error } = useEndpoint(endpointId);
  const triggerScan = useTriggerScan();
  const { data: patchesData } = useEndpointPatches(endpointId);

  const patches = (patchesData?.data ?? [])
    .filter((p) => p.status === 'pending' || p.status === 'available')
    .sort((a, b) => (severityOrder[a.severity] ?? 4) - (severityOrder[b.severity] ?? 4));

  const criticalCount = patches.filter((p) => p.severity === 'critical').length;
  const highCount = patches.filter((p) => p.severity === 'high').length;
  const mediumCount = patches.filter((p) => p.severity === 'medium').length;

  if (isLoading) {
    return (
      <tr data-testid="expanded-row">
        <td
          colSpan={colSpan}
          style={{
            background: 'var(--bg-inset)',
            padding: '10px 16px',
            borderTop: '1px solid var(--border)',
            borderBottom: '1px solid var(--border)',
          }}
        >
          <div style={{ height: 72, display: 'flex', alignItems: 'center', paddingLeft: 8 }}>
            <span style={{ fontSize: 11, color: 'var(--text-faint)' }}>Loading…</span>
          </div>
        </td>
      </tr>
    );
  }

  if (error || !detail) {
    return (
      <tr data-testid="expanded-row">
        <td
          colSpan={colSpan}
          style={{
            background: 'var(--bg-inset)',
            padding: '8px 16px',
            borderTop: '1px solid var(--border)',
            borderBottom: '1px solid var(--border)',
          }}
        >
          <span style={{ fontSize: 11, color: 'var(--signal-critical)' }}>
            Failed to load details.
          </span>
        </td>
      </tr>
    );
  }

  const cpuPct = Math.round(detail.cpu_usage_percent ?? 0);
  const memPct =
    detail.memory_total_mb && detail.memory_used_mb
      ? Math.round((detail.memory_used_mb / detail.memory_total_mb) * 100)
      : 0;
  const diskPct =
    detail.disk_total_gb && detail.disk_used_gb
      ? Math.round((detail.disk_used_gb / detail.disk_total_gb) * 100)
      : 0;

  const cellStyle: React.CSSProperties = {
    background: 'var(--bg-card)',
    border: '1px solid var(--border)',
    borderRadius: 7,
    padding: '10px 12px',
  };

  const headStyle: React.CSSProperties = {
    fontSize: 9,
    fontFamily: 'var(--font-mono)',
    textTransform: 'uppercase',
    letterSpacing: '0.08em',
    color: 'var(--text-faint)',
    fontWeight: 500,
    marginBottom: 8,
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
    textDecoration: 'none',
  };

  return (
    <tr data-testid="expanded-row">
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
          {/* System Health card */}
          <div style={{ ...cellStyle, flex: '0 0 500px' }}>
            <div style={headStyle}>System Health</div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 7 }}>
              {[
                { label: 'CPU', pct: cpuPct, sub: `${cpuPct}%` },
                { label: 'Mem', pct: memPct, sub: `${memPct}%` },
                { label: 'Disk', pct: diskPct, sub: `${diskPct}%` },
              ].map(({ label, pct, sub }) => (
                <div key={label}>
                  <div
                    style={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      alignItems: 'center',
                    }}
                  >
                    <span style={{ fontSize: 10, color: 'var(--text-secondary)' }}>{label}</span>
                    <span
                      style={{
                        fontSize: 10,
                        fontFamily: 'var(--font-mono)',
                        color: metricColor(pct),
                      }}
                    >
                      {sub}
                    </span>
                  </div>
                  <MiniBar pct={pct} color={metricColor(pct)} />
                </div>
              ))}
            </div>
          </div>

          {/* Pending Patches card */}
          <div style={{ ...cellStyle, flex: '0 0 480px', marginLeft: 24 }}>
            <div style={headStyle}>Pending Patches</div>
            {patches.length === 0 ? (
              <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>No pending patches</span>
            ) : (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
                  {criticalCount > 0 && (
                    <span
                      style={{
                        fontSize: 11,
                        fontFamily: 'var(--font-mono)',
                        color: 'var(--signal-critical)',
                        fontWeight: 600,
                      }}
                    >
                      {criticalCount} critical
                    </span>
                  )}
                  {highCount > 0 && (
                    <span
                      style={{
                        fontSize: 11,
                        fontFamily: 'var(--font-mono)',
                        color: 'var(--signal-warning)',
                      }}
                    >
                      · {highCount} high
                    </span>
                  )}
                  {mediumCount > 0 && (
                    <span
                      style={{
                        fontSize: 11,
                        fontFamily: 'var(--font-mono)',
                        color: 'var(--signal-warning)',
                      }}
                    >
                      · {mediumCount} medium
                    </span>
                  )}
                </div>
                <span style={{ fontSize: 10, color: 'var(--text-faint)' }}>
                  {patches.length} total pending
                </span>
              </div>
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
                setDeployHandler();
              }}
            >
              <Zap style={{ width: 12, height: 12 }} />
              Deploy
            </button>
            <button
              type="button"
              disabled={triggerScan.isPending}
              style={{
                ...BTN,
                cursor: triggerScan.isPending ? 'not-allowed' : 'pointer',
                opacity: triggerScan.isPending ? 0.6 : 1,
              }}
              onClick={(e) => {
                e.stopPropagation();
                triggerScan.mutate(endpointId);
              }}
            >
              <RefreshCw style={{ width: 12, height: 12 }} />
              {triggerScan.isPending ? 'Scanning…' : 'Scan'}
            </button>
            <Link to={`/endpoints/${endpointId}`} onClick={(e) => e.stopPropagation()} style={BTN}>
              View Details →
            </Link>
          </div>
        </div>
      </td>
    </tr>
  );
}

// No-op placeholder — ExpandedRow doesn't open the deploy wizard itself;
// deploy action goes to the detail page. We keep the button for visual completeness.
function setDeployHandler() {
  // Navigate user to full detail page for deployment
  window.location.href = window.location.href.split('/endpoints')[0] + '/endpoints';
}
