import { TOP_CVES_DATA } from '@/data/mock-data';

// ── Severity color map ─────────────────────────────────────────────────────────
const SEVERITY_COLOR: Record<'critical' | 'high' | 'medium', string> = {
  critical: '#ef4444',
  high: '#f59e0b',
  medium: '#eab308',
};

// ── CVE Row ────────────────────────────────────────────────────────────────────
interface CVERowProps {
  rank: number;
  id: string;
  description: string;
  cvss: number;
  severity: 'critical' | 'high' | 'medium';
  affectedEndpoints: number;
  daysOpen: number;
  patchAvailable: boolean;
  isLast: boolean;
}

function CVERow({
  rank,
  id,
  description,
  cvss,
  severity,
  affectedEndpoints,
  daysOpen,
  patchAvailable,
  isLast,
}: CVERowProps) {
  const severityColor = SEVERITY_COLOR[severity];

  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 10,
        padding: '8px 2px',
        borderBottom: isLast ? 'none' : '1px solid var(--color-separator)',
        borderRadius: 4,
        transition: 'background 0.15s ease',
        cursor: 'default',
      }}
      onMouseEnter={(e) => {
        (e.currentTarget as HTMLDivElement).style.background =
          'var(--color-glass-hover, color-mix(in srgb, var(--color-foreground) 4%, transparent))';
      }}
      onMouseLeave={(e) => {
        (e.currentTarget as HTMLDivElement).style.background = 'transparent';
      }}
    >
      {/* Rank badge */}
      <div
        style={{
          flexShrink: 0,
          width: 22,
          height: 22,
          borderRadius: 4,
          background: `color-mix(in srgb, ${severityColor} 15%, transparent)`,
          border: `1px solid color-mix(in srgb, ${severityColor} 30%, transparent)`,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          fontWeight: 700,
          fontSize: 11,
          color: severityColor,
          fontFamily: 'var(--font-sans)',
        }}
      >
        {rank}
      </div>

      {/* CVE ID + description */}
      <div style={{ flex: 1, minWidth: 0 }}>
        <div
          style={{
            fontFamily: 'var(--font-mono)',
            fontWeight: 700,
            fontSize: 11,
            color: 'var(--color-foreground)',
            whiteSpace: 'nowrap',
          }}
        >
          {id}
        </div>
        <div
          style={{
            fontSize: 10,
            color: 'var(--color-muted)',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
            marginTop: 1,
          }}
        >
          {description}
        </div>
      </div>

      {/* Right side: CVSS + meta */}
      <div style={{ flexShrink: 0, display: 'flex', alignItems: 'center', gap: 8 }}>
        {/* CVSS score */}
        <span
          style={{
            fontWeight: 700,
            fontSize: 15,
            color: severityColor,
            fontFamily: 'var(--font-mono)',
            minWidth: 28,
            textAlign: 'right',
            lineHeight: 1,
          }}
        >
          {cvss.toFixed(1)}
        </span>

        {/* Affected endpoints */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 3,
            fontSize: 10,
            color: 'var(--color-muted)',
          }}
        >
          {/* Dot icon */}
          <svg width="7" height="7" viewBox="0 0 7 7" fill="none" aria-hidden="true">
            <circle cx="3.5" cy="3.5" r="3.5" fill="currentColor" opacity="0.6" />
          </svg>
          <span style={{ fontVariantNumeric: 'tabular-nums' }}>{affectedEndpoints}</span>
        </div>

        {/* Days open pill */}
        <span
          style={{
            fontSize: 9,
            fontWeight: 600,
            padding: '2px 5px',
            borderRadius: 999,
            background: 'color-mix(in srgb, var(--color-foreground) 8%, transparent)',
            color: 'var(--color-subtle)',
            fontFamily: 'var(--font-mono)',
            whiteSpace: 'nowrap',
          }}
        >
          {daysOpen}d
        </span>

        {/* Patch availability chip */}
        <span
          style={{
            fontSize: 9,
            fontWeight: 600,
            padding: '2px 6px',
            borderRadius: 999,
            whiteSpace: 'nowrap',
            background: patchAvailable
              ? 'color-mix(in srgb, var(--color-success) 12%, transparent)'
              : 'color-mix(in srgb, #f59e0b 12%, transparent)',
            color: patchAvailable ? 'var(--color-success)' : '#f59e0b',
            border: `1px solid ${
              patchAvailable
                ? 'color-mix(in srgb, var(--color-success) 25%, transparent)'
                : 'color-mix(in srgb, #f59e0b 25%, transparent)'
            }`,
          }}
        >
          {patchAvailable ? '✓ Patch' : '⚠ No patch'}
        </span>
      </div>
    </div>
  );
}

// ── Main Component ─────────────────────────────────────────────────────────────
export function TopCriticalCVEs() {
  const sorted = [...TOP_CVES_DATA].sort((a, b) => b.cvss - a.cvss).slice(0, 5);

  return (
    <div
      style={{
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden',
      }}
    >
      {sorted.map((cve, i) => (
        <CVERow
          key={cve.id}
          rank={i + 1}
          id={cve.id}
          description={cve.description}
          cvss={cve.cvss}
          severity={cve.severity}
          affectedEndpoints={cve.affectedEndpoints}
          daysOpen={cve.daysOpen}
          patchAvailable={cve.patchAvailable}
          isLast={i === sorted.length - 1}
        />
      ))}
    </div>
  );
}
