import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  type TooltipProps,
} from 'recharts';
import { PATCHES_HORIZON_DATA } from '@/data/mock-data';

// ── Color definitions ─────────────────────────────────────────────────────────
const SEVERITY_COLORS = {
  critical: '#ff3b30',
  high: '#ff9500',
  medium: '#ffcc00',
  low: '#34c759',
} as const;

const SEVERITY_GRADIENT_IDS = {
  critical: 'gradCritical',
  high: 'gradHigh',
  medium: 'gradMedium',
  low: 'gradLow',
} as const;

// ── Custom Tooltip ────────────────────────────────────────────────────────────
function CustomTooltip({ active, payload, label }: TooltipProps<number, string>) {
  if (!active || !payload || payload.length === 0) return null;

  // Total patches for this month
  const total = payload.reduce((sum, p) => sum + (p.value as number), 0);

  // Reverse order so critical is on top in tooltip
  const orderedPayload = [...payload].reverse();

  return (
    <div
      className="glass-card"
      style={{
        padding: '10px 12px',
        minWidth: 150,
        pointerEvents: 'none',
      }}
    >
      {/* Month header */}
      <div
        style={{
          fontSize: 11,
          fontWeight: 700,
          color: 'var(--color-foreground)',
          marginBottom: 6,
          letterSpacing: '0.02em',
        }}
      >
        {label}
      </div>

      {/* Severity breakdown */}
      {orderedPayload.map((p) => {
        const key = p.dataKey as keyof typeof SEVERITY_COLORS;
        const color = SEVERITY_COLORS[key];
        return (
          <div
            key={p.dataKey}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 6,
              marginBottom: 3,
            }}
          >
            <div
              style={{
                width: 7,
                height: 7,
                borderRadius: '50%',
                background: color,
                flexShrink: 0,
              }}
            />
            <span
              style={{
                fontSize: 10,
                color: 'var(--color-muted)',
                flex: 1,
                textTransform: 'capitalize',
              }}
            >
              {p.dataKey}
            </span>
            <span
              style={{
                fontSize: 10,
                fontWeight: 600,
                color: 'var(--color-foreground)',
              }}
            >
              {p.value}
            </span>
          </div>
        );
      })}

      {/* Divider + total */}
      <div
        style={{
          borderTop: '1px solid var(--color-separator)',
          marginTop: 4,
          paddingTop: 4,
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
        }}
      >
        <span
          style={{
            fontSize: 10,
            fontWeight: 600,
            color: 'var(--color-muted)',
          }}
        >
          Total
        </span>
        <span
          style={{
            fontSize: 11,
            fontWeight: 700,
            color: 'var(--color-foreground)',
          }}
        >
          {total}
        </span>
      </div>
    </div>
  );
}

// ── Legend Badges ─────────────────────────────────────────────────────────────
function LegendBadges() {
  const items: { label: string; key: keyof typeof SEVERITY_COLORS }[] = [
    { label: 'Critical', key: 'critical' },
    { label: 'High', key: 'high' },
    { label: 'Medium', key: 'medium' },
    { label: 'Low', key: 'low' },
  ];

  return (
    <div style={{ display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap' }}>
      {items.map(({ label, key }) => {
        const color = SEVERITY_COLORS[key];
        return (
          <div
            key={key}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 5,
              padding: '2px 8px',
              borderRadius: 999,
              background: `color-mix(in srgb, ${color} 10%, transparent)`,
              border: `1px solid color-mix(in srgb, ${color} 25%, transparent)`,
            }}
          >
            <div
              style={{
                width: 7,
                height: 7,
                borderRadius: '50%',
                background: color,
                flexShrink: 0,
              }}
            />
            <span
              style={{
                fontSize: 10,
                fontWeight: 600,
                color,
                letterSpacing: '0.02em',
                whiteSpace: 'nowrap',
              }}
            >
              {label}
            </span>
          </div>
        );
      })}
    </div>
  );
}

// ── Main Component ─────────────────────────────────────────────────────────────
export function PatchesHorizon() {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      {/* Chart area */}
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={PATCHES_HORIZON_DATA} margin={{ top: 8, right: 8, bottom: 4, left: -12 }}>
          <defs>
            {/* Low — green (bottom layer) */}
            <linearGradient id={SEVERITY_GRADIENT_IDS.low} x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={SEVERITY_COLORS.low} stopOpacity={0.4} />
              <stop offset="95%" stopColor={SEVERITY_COLORS.low} stopOpacity={0.15} />
            </linearGradient>

            {/* Medium — yellow */}
            <linearGradient id={SEVERITY_GRADIENT_IDS.medium} x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={SEVERITY_COLORS.medium} stopOpacity={0.45} />
              <stop offset="95%" stopColor={SEVERITY_COLORS.medium} stopOpacity={0.15} />
            </linearGradient>

            {/* High — orange */}
            <linearGradient id={SEVERITY_GRADIENT_IDS.high} x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={SEVERITY_COLORS.high} stopOpacity={0.5} />
              <stop offset="95%" stopColor={SEVERITY_COLORS.high} stopOpacity={0.15} />
            </linearGradient>

            {/* Critical — red (top, darkest) */}
            <linearGradient id={SEVERITY_GRADIENT_IDS.critical} x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={SEVERITY_COLORS.critical} stopOpacity={0.6} />
              <stop offset="95%" stopColor={SEVERITY_COLORS.critical} stopOpacity={0.2} />
            </linearGradient>
          </defs>

          <CartesianGrid
            strokeDasharray="3 3"
            stroke="var(--color-separator)"
            strokeOpacity={0.7}
            vertical={false}
          />

          <XAxis
            dataKey="date"
            tickLine={false}
            axisLine={false}
            tick={{ fontSize: 10, fill: 'var(--color-muted)', fontWeight: 500 }}
            tickFormatter={(v: string) => v.split(' ')[0]} // show only month name
            interval={0}
          />

          <YAxis
            tickLine={false}
            axisLine={false}
            tick={{ fontSize: 10, fill: 'var(--color-muted)', fontWeight: 500 }}
            tickCount={5}
          />

          <Tooltip
            content={<CustomTooltip />}
            cursor={{
              stroke: 'var(--color-separator)',
              strokeWidth: 1,
              strokeDasharray: '4 2',
            }}
          />

          {/* Stack from bottom: low → medium → high → critical */}
          <Area
            type="monotone"
            dataKey="low"
            name="low"
            stackId="patches"
            stroke={SEVERITY_COLORS.low}
            strokeWidth={1}
            fill={`url(#${SEVERITY_GRADIENT_IDS.low})`}
            dot={false}
            activeDot={{ r: 3, fill: SEVERITY_COLORS.low, stroke: 'white', strokeWidth: 1.5 }}
            isAnimationActive={true}
            animationDuration={1200}
            animationEasing="ease-out"
          />

          <Area
            type="monotone"
            dataKey="medium"
            name="medium"
            stackId="patches"
            stroke={SEVERITY_COLORS.medium}
            strokeWidth={1}
            fill={`url(#${SEVERITY_GRADIENT_IDS.medium})`}
            dot={false}
            activeDot={{ r: 3, fill: SEVERITY_COLORS.medium, stroke: 'white', strokeWidth: 1.5 }}
            isAnimationActive={true}
            animationDuration={1200}
            animationEasing="ease-out"
          />

          <Area
            type="monotone"
            dataKey="high"
            name="high"
            stackId="patches"
            stroke={SEVERITY_COLORS.high}
            strokeWidth={1}
            fill={`url(#${SEVERITY_GRADIENT_IDS.high})`}
            dot={false}
            activeDot={{ r: 3, fill: SEVERITY_COLORS.high, stroke: 'white', strokeWidth: 1.5 }}
            isAnimationActive={true}
            animationDuration={1200}
            animationEasing="ease-out"
          />

          <Area
            type="monotone"
            dataKey="critical"
            name="critical"
            stackId="patches"
            stroke={SEVERITY_COLORS.critical}
            strokeWidth={1.5}
            fill={`url(#${SEVERITY_GRADIENT_IDS.critical})`}
            dot={false}
            activeDot={{ r: 3, fill: SEVERITY_COLORS.critical, stroke: 'white', strokeWidth: 1.5 }}
            isAnimationActive={true}
            animationDuration={1200}
            animationEasing="ease-out"
          />
        </AreaChart>
      </ResponsiveContainer>

      {/* Legend row */}
      <div
        style={{
          display: 'flex',
          justifyContent: 'center',
          paddingTop: 4,
        }}
      >
        <LegendBadges />
      </div>
    </div>
  );
}
