import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ReferenceLine,
  ResponsiveContainer,
  type TooltipProps,
} from 'recharts';
import { RISK_DELTA_DATA } from '@/data/mock-data';

// ── Custom Tooltip ────────────────────────────────────────────────────────────
function CustomTooltip({ active, payload, label }: TooltipProps<number, string>) {
  if (!active || !payload || payload.length === 0) return null;

  const seriesOrder = ['deployAll', 'current', 'doNothing'];
  const sorted = [...payload].sort(
    (a, b) => seriesOrder.indexOf(a.dataKey as string) - seriesOrder.indexOf(b.dataKey as string),
  );

  return (
    <div
      className="glass-card px-3 py-2 text-xs"
      style={{
        minWidth: 140,
        pointerEvents: 'none',
      }}
    >
      <div
        style={{
          fontSize: 11,
          fontWeight: 700,
          marginBottom: 6,
          color: 'var(--color-foreground)',
          letterSpacing: '0.02em',
        }}
      >
        Day {label}
      </div>
      {sorted.map((p) => (
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
              height: 7,
              width: 7,
              borderRadius: '50%',
              background: p.color,
              flexShrink: 0,
            }}
          />
          <span style={{ color: 'var(--color-muted)', flex: 1 }}>{p.name}:</span>
          <span style={{ fontWeight: 600, color: 'var(--color-foreground)' }}>{p.value}</span>
        </div>
      ))}
    </div>
  );
}

// ── Legend Badges ─────────────────────────────────────────────────────────────
function LegendBadges() {
  const items = [
    { label: 'Deploy All', color: 'var(--color-success)', dashed: true },
    { label: 'Current', color: 'var(--color-primary)', dashed: false },
    { label: 'Do Nothing', color: 'var(--color-danger)', dashed: true },
  ];

  return (
    <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
      {items.map(({ label, color, dashed }) => (
        <div
          key={label}
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
          {/* Line preview */}
          <svg width={14} height={8} viewBox="0 0 14 8">
            <line
              x1={0}
              y1={4}
              x2={14}
              y2={4}
              stroke={color}
              strokeWidth={1.5}
              strokeDasharray={dashed ? '4 2' : undefined}
              strokeLinecap="round"
            />
          </svg>
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
      ))}
    </div>
  );
}

// ── Main Chart Component ──────────────────────────────────────────────────────
export function RiskDeltaChart() {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      {/* Chart area */}
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={RISK_DELTA_DATA} margin={{ top: 20, right: 8, bottom: 4, left: -12 }}>
          <defs>
            {/* Green gradient — Deploy All */}
            <linearGradient id="gradDeployAll" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#34c759" stopOpacity={0.18} />
              <stop offset="95%" stopColor="#34c759" stopOpacity={0} />
            </linearGradient>

            {/* Blue gradient — Current (very subtle) */}
            <linearGradient id="gradCurrent" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#007aff" stopOpacity={0.1} />
              <stop offset="95%" stopColor="#007aff" stopOpacity={0} />
            </linearGradient>

            {/* Red gradient — Do Nothing */}
            <linearGradient id="gradDoNothing" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#ff3b30" stopOpacity={0.14} />
              <stop offset="95%" stopColor="#ff3b30" stopOpacity={0} />
            </linearGradient>
          </defs>

          <CartesianGrid
            strokeDasharray="3 3"
            stroke="var(--color-separator)"
            strokeOpacity={0.7}
            vertical={false}
          />

          <XAxis
            dataKey="day"
            tickLine={false}
            axisLine={false}
            tick={{ fontSize: 10, fill: 'var(--color-muted)', fontWeight: 500 }}
            tickFormatter={(v: number) => (v % 5 === 0 || v === 1 ? `D${v}` : '')}
            interval={0}
          />

          <YAxis
            domain={[0, 100]}
            tickLine={false}
            axisLine={false}
            tick={{ fontSize: 10, fill: 'var(--color-muted)', fontWeight: 500 }}
            tickCount={6}
          />

          <Tooltip
            content={<CustomTooltip />}
            cursor={{
              stroke: 'var(--color-separator)',
              strokeWidth: 1,
              strokeDasharray: '4 2',
            }}
          />

          {/* SLA reference lines */}
          <ReferenceLine
            x={7}
            stroke="var(--color-warning)"
            strokeWidth={1}
            strokeDasharray="4 3"
            strokeOpacity={0.7}
            label={{
              value: 'SLA 7d',
              position: 'top',
              fontSize: 9,
              fill: 'var(--color-warning)',
              fontWeight: 600,
            }}
          />
          <ReferenceLine
            x={14}
            stroke="var(--color-danger)"
            strokeWidth={1}
            strokeDasharray="4 3"
            strokeOpacity={0.6}
            label={{
              value: 'SLA 14d',
              position: 'top',
              fontSize: 9,
              fill: 'var(--color-danger)',
              fontWeight: 600,
            }}
          />

          {/* Deploy All — green dashed, gradient fill */}
          <Area
            type="monotone"
            dataKey="deployAll"
            name="Deploy All"
            stroke="#34c759"
            strokeWidth={1.5}
            strokeDasharray="6 3"
            fill="url(#gradDeployAll)"
            dot={false}
            activeDot={{ r: 4, fill: '#34c759', stroke: 'white', strokeWidth: 1.5 }}
            isAnimationActive={true}
            animationDuration={1200}
            animationEasing="ease-out"
          />

          {/* Current Trajectory — solid blue, very subtle fill */}
          <Area
            type="monotone"
            dataKey="current"
            name="Current"
            stroke="#007aff"
            strokeWidth={2}
            fill="url(#gradCurrent)"
            dot={false}
            activeDot={{ r: 4, fill: '#007aff', stroke: 'white', strokeWidth: 1.5 }}
            isAnimationActive={true}
            animationDuration={1400}
            animationEasing="ease-out"
          />

          {/* Do Nothing — red dashed, gradient fill */}
          <Area
            type="monotone"
            dataKey="doNothing"
            name="Do Nothing"
            stroke="#ff3b30"
            strokeWidth={1.5}
            strokeDasharray="6 3"
            fill="url(#gradDoNothing)"
            dot={false}
            activeDot={{ r: 4, fill: '#ff3b30', stroke: 'white', strokeWidth: 1.5 }}
            isAnimationActive={true}
            animationDuration={1600}
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
