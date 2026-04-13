import { AreaChart, Area, ResponsiveContainer } from 'recharts';
import { MTTP_DATA } from '@/data/mock-data';

// ── Severity colors ────────────────────────────────────────────────────────────
const BREAKDOWN_COLORS = {
  critical: '#ef4444',
  high: '#f59e0b',
  medium: '#eab308',
  low: 'var(--color-success)',
} as const;

// ── Main Component ─────────────────────────────────────────────────────────────
export function MeanTimeToPatch() {
  const isOverTarget = MTTP_DATA.current > MTTP_DATA.target;
  const metricColor = isOverTarget ? '#ef4444' : 'var(--color-success)';
  const trendImproving = MTTP_DATA.trend < 0;

  // Shape sparkline data for Recharts
  const sparkData = MTTP_DATA.sparkline.map((v, i) => ({ i, v }));

  const breakdownEntries: {
    label: string;
    key: keyof typeof MTTP_DATA.breakdown;
    color: string;
  }[] = [
    { label: 'Critical', key: 'critical', color: BREAKDOWN_COLORS.critical },
    { label: 'High', key: 'high', color: BREAKDOWN_COLORS.high },
    { label: 'Medium', key: 'medium', color: BREAKDOWN_COLORS.medium },
    { label: 'Low', key: 'low', color: BREAKDOWN_COLORS.low },
  ];

  return (
    <div
      style={{
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden',
      }}
    >
      {/* Top section: metric + sparkline */}
      <div style={{ display: 'flex', alignItems: 'stretch', gap: 12, flex: '0 0 auto' }}>
        {/* Left: headline number + badges */}
        <div style={{ flex: '0 0 55%', display: 'flex', flexDirection: 'column', gap: 4 }}>
          {/* Large number */}
          <div
            style={{
              fontSize: 36,
              fontWeight: 700,
              lineHeight: 1,
              color: metricColor,
              fontFamily: 'var(--font-sans)',
              letterSpacing: '-0.02em',
            }}
          >
            {MTTP_DATA.current}d
          </div>

          {/* Label */}
          <div
            style={{
              fontSize: 11,
              color: 'var(--color-muted)',
              fontWeight: 500,
              marginTop: 2,
            }}
          >
            Mean Time to Patch
          </div>

          {/* Target + trend badges row */}
          <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginTop: 4 }}>
            {/* Target pill */}
            <span
              style={{
                fontSize: 10,
                fontWeight: 600,
                padding: '2px 7px',
                borderRadius: 999,
                background: isOverTarget
                  ? 'color-mix(in srgb, #ef4444 12%, transparent)'
                  : 'color-mix(in srgb, var(--color-success) 12%, transparent)',
                color: metricColor,
                border: `1px solid ${
                  isOverTarget
                    ? 'color-mix(in srgb, #ef4444 25%, transparent)'
                    : 'color-mix(in srgb, var(--color-success) 25%, transparent)'
                }`,
              }}
            >
              Target: {MTTP_DATA.target}d
            </span>

            {/* Trend badge */}
            <span
              style={{
                fontSize: 10,
                fontWeight: 600,
                padding: '2px 7px',
                borderRadius: 999,
                background: trendImproving
                  ? 'color-mix(in srgb, var(--color-success) 12%, transparent)'
                  : 'color-mix(in srgb, #ef4444 12%, transparent)',
                color: trendImproving ? 'var(--color-success)' : '#ef4444',
                border: `1px solid ${
                  trendImproving
                    ? 'color-mix(in srgb, var(--color-success) 25%, transparent)'
                    : 'color-mix(in srgb, #ef4444 25%, transparent)'
                }`,
              }}
            >
              {trendImproving ? '↓' : '↑'} {Math.abs(MTTP_DATA.trend)}d
            </span>
          </div>
        </div>

        {/* Right: mini sparkline */}
        <div style={{ flex: '0 0 45%', height: 64 }}>
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={sparkData} margin={{ top: 2, right: 2, bottom: 2, left: 2 }}>
              <defs>
                <linearGradient id="mttpSparkGrad" x1="0" y1="0" x2="0" y2="1">
                  <stop
                    offset="5%"
                    stopColor={metricColor === 'var(--color-success)' ? '#22c55e' : '#ef4444'}
                    stopOpacity={0.35}
                  />
                  <stop
                    offset="95%"
                    stopColor={metricColor === 'var(--color-success)' ? '#22c55e' : '#ef4444'}
                    stopOpacity={0.05}
                  />
                </linearGradient>
              </defs>
              <Area
                type="monotone"
                dataKey="v"
                stroke={metricColor === 'var(--color-success)' ? '#22c55e' : '#ef4444'}
                strokeWidth={1.5}
                fill="url(#mttpSparkGrad)"
                dot={false}
                isAnimationActive={true}
                animationDuration={1000}
                animationEasing="ease-out"
              />
            </AreaChart>
          </ResponsiveContainer>
        </div>
      </div>

      {/* Divider */}
      <div
        style={{
          borderTop: '1px solid var(--color-separator)',
          margin: '10px 0 8px',
          flexShrink: 0,
        }}
      />

      {/* Bottom: severity breakdown grid */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(4, 1fr)',
          gap: 4,
          flex: 1,
          alignContent: 'start',
        }}
      >
        {breakdownEntries.map(({ label, key, color }) => (
          <div
            key={key}
            style={{
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              gap: 2,
              padding: '6px 4px',
              borderRadius: 6,
              background: `color-mix(in srgb, ${color === 'var(--color-success)' ? '#22c55e' : color} 6%, transparent)`,
            }}
          >
            <span
              style={{
                fontSize: 9,
                fontWeight: 600,
                color,
                textTransform: 'uppercase',
                letterSpacing: '0.05em',
              }}
            >
              {label}
            </span>
            <span
              style={{
                fontSize: 14,
                fontWeight: 700,
                color: 'var(--color-foreground)',
                fontFamily: 'var(--font-mono)',
                lineHeight: 1,
              }}
            >
              {MTTP_DATA.breakdown[key]}d
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}
