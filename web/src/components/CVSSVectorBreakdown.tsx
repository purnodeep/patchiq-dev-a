import { parseCVSSVector } from '../lib/cvss';

const SEVERITY_COLORS: Record<string, { bar: string; text: string }> = {
  critical: { bar: 'var(--signal-critical)', text: 'var(--signal-critical)' },
  high: { bar: 'var(--signal-warning)', text: 'var(--signal-warning)' },
  medium: { bar: 'var(--signal-medium, #eab308)', text: 'var(--signal-medium, #eab308)' },
  low: { bar: 'var(--signal-healthy)', text: 'var(--signal-healthy)' },
  none: { bar: 'var(--text-muted)', text: 'var(--text-faint)' },
};

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

interface CVSSVectorBreakdownProps {
  vector: string | null;
  className?: string;
}

export function CVSSVectorBreakdown({ vector }: CVSSVectorBreakdownProps) {
  const metrics = parseCVSSVector(vector);
  if (metrics.length === 0) return null;
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <code
        style={{
          display: 'block',
          borderRadius: 6,
          background: 'var(--bg-inset)',
          border: '1px solid var(--border)',
          padding: '8px 12px',
          fontSize: 11,
          fontFamily: 'var(--font-mono)',
          color: 'var(--text-secondary)',
        }}
      >
        {vector}
      </code>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
        {metrics.map((metric) => {
          const colors = SEVERITY_COLORS[metric.severity] ?? SEVERITY_COLORS.none;
          const width = BAR_WIDTH[metric.key]?.[metric.value] ?? 50;
          return (
            <div key={metric.key} style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
              <span
                style={{
                  width: 220,
                  fontSize: 11,
                  fontFamily: 'var(--font-sans)',
                  color: 'var(--text-secondary)',
                  flexShrink: 0,
                }}
              >
                {metric.name}
              </span>
              <div
                style={{
                  flex: 1,
                  height: 8,
                  borderRadius: 9999,
                  background: 'var(--bg-inset)',
                  overflow: 'hidden',
                }}
              >
                <div
                  style={{
                    height: '100%',
                    borderRadius: 9999,
                    background: colors.bar,
                    width: `${width}%`,
                    transition: 'width 0.3s ease',
                  }}
                />
              </div>
              <span
                style={{
                  width: 80,
                  fontSize: 11,
                  fontWeight: 500,
                  fontFamily: 'var(--font-mono)',
                  textAlign: 'right',
                  color: colors.text,
                  flexShrink: 0,
                }}
              >
                {metric.label}
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
}
