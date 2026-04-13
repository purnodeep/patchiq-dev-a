import { useState } from 'react';
import { SLA_WATERFALL_DATA } from '@/data/mock-data';

// ── Waterfall layout math ──────────────────────────────────────────────────────
// We render horizontal floating bars. "Gap" and "projected" are absolute values,
// "positive" and "negative" are deltas stacked from the running total.

interface BarLayout {
  label: string;
  value: number;
  type: 'gap' | 'positive' | 'negative' | 'projected';
  displayValue: number; // absolute value for rendering purposes
  runningTotal: number; // cumulative total at this point
  barStart: number; // left offset as fraction of total domain
  barEnd: number; // right offset as fraction of total domain
  isAbsolute: boolean; // gap and projected are absolute, deltas float
}

function buildLayout(): BarLayout[] {
  // Domain: 0 to max(starting gap, projected) + some headroom
  const raw = SLA_WATERFALL_DATA;
  const maxAbsolute = Math.max(
    ...raw.filter((d) => d.type === 'gap' || d.type === 'projected').map((d) => Math.abs(d.value)),
  );
  const domain = maxAbsolute + 10; // headroom

  let running = 0;
  return raw.map((item) => {
    const isAbsolute = item.type === 'gap' || item.type === 'projected';
    let barStart: number;
    let barEnd: number;
    let displayValue: number;

    if (item.type === 'gap') {
      displayValue = item.value;
      barStart = 0;
      barEnd = item.value / domain;
      running = item.value;
    } else if (item.type === 'projected') {
      displayValue = item.value;
      barStart = 0;
      barEnd = item.value / domain;
      // running stays same
    } else if (item.type === 'positive') {
      // negative delta — reduces running total
      displayValue = Math.abs(item.value);
      barStart = (running + item.value) / domain; // shrinks from right
      barEnd = running / domain;
      running = running + item.value;
    } else {
      // negative type — increases running total
      displayValue = Math.abs(item.value);
      barStart = running / domain;
      barEnd = (running + item.value) / domain;
      running = running + item.value;
    }

    return {
      label: item.label,
      value: item.value,
      type: item.type,
      displayValue,
      runningTotal: running,
      barStart: Math.max(0, barStart),
      barEnd: Math.min(1, barEnd),
      isAbsolute,
    };
  });
}

const BAR_COLORS = {
  gap: '#ff3b30',
  positive: '#34c759',
  negative: '#ff9500',
  projected: '#007aff',
} as const;

const BAR_LABELS = {
  gap: 'Starting Gap',
  positive: 'Reduction',
  negative: 'New Exposure',
  projected: 'Projected',
} as const;

// ── Tooltip ────────────────────────────────────────────────────────────────────
interface TooltipState {
  barIndex: number;
  x: number;
  y: number;
}

// ── Main Component ─────────────────────────────────────────────────────────────
export function SLABridgeWaterfall() {
  const [tooltip, setTooltip] = useState<TooltipState | null>(null);
  const layout = buildLayout();

  const BAR_HEIGHT = 32;
  const BAR_GAP = 14;
  const LABEL_WIDTH = 132;
  const VALUE_WIDTH = 44;
  const CHART_PADDING_RIGHT = 8;

  const totalHeight = layout.length * (BAR_HEIGHT + BAR_GAP) - BAR_GAP + 2;

  const activeBar = tooltip !== null ? layout[tooltip.barIndex] : null;

  return (
    <div
      style={{
        position: 'relative',
        userSelect: 'none',
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
      }}
    >
      <style>{`
        @keyframes sla-bar-in {
          from { transform: scaleX(0); opacity: 0; }
          to   { transform: scaleX(1); opacity: 1; }
        }
        @keyframes sla-connector-in {
          from { opacity: 0; }
          to   { opacity: 1; }
        }
      `}</style>

      <svg
        width="100%"
        viewBox={`0 0 ${LABEL_WIDTH + 400 + VALUE_WIDTH + CHART_PADDING_RIGHT} ${totalHeight + 24}`}
        style={{ flex: 1, minHeight: 0, overflow: 'visible', display: 'block' }}
        onMouseLeave={() => setTooltip(null)}
      >
        {/* Grid lines behind bars */}
        {[0, 0.25, 0.5, 0.75, 1].map((frac) => {
          const x = LABEL_WIDTH + frac * 400;
          return (
            <line
              key={frac}
              x1={x}
              y1={0}
              x2={x}
              y2={totalHeight}
              stroke="var(--color-separator)"
              strokeWidth={1}
              strokeDasharray={frac === 0 ? undefined : '4 3'}
              strokeOpacity={frac === 0 ? 0.6 : 0.4}
            />
          );
        })}

        {/* X-axis labels */}
        {[0, 10, 20, 30, 40, 50].map((val) => {
          // map val to domain
          const raw = SLA_WATERFALL_DATA;
          const maxAbsolute = Math.max(
            ...raw
              .filter((d) => d.type === 'gap' || d.type === 'projected')
              .map((d) => Math.abs(d.value)),
          );
          const domain = maxAbsolute + 10;
          const frac = val / domain;
          if (frac > 1) return null;
          const x = LABEL_WIDTH + frac * 400;
          return (
            <text
              key={val}
              x={x}
              y={totalHeight + 16}
              textAnchor="middle"
              fontSize={9}
              fontWeight={500}
              fill="var(--color-muted)"
            >
              {val}
            </text>
          );
        })}

        {layout.map((bar, i) => {
          const y = i * (BAR_HEIGHT + BAR_GAP);
          const bx = LABEL_WIDTH + bar.barStart * 400;
          const bw = Math.max(2, (bar.barEnd - bar.barStart) * 400);
          const color = BAR_COLORS[bar.type];
          const isHovered = tooltip?.barIndex === i;
          const animDelay = `${i * 80}ms`;
          const transformOriginX =
            bar.type === 'positive'
              ? bx + bw // shrinks from the right edge inward for reductions
              : bx;

          // Connector line from previous bar end to this bar start
          const prevBar = i > 0 ? layout[i - 1] : null;
          const showConnector = prevBar && !bar.isAbsolute;
          const connX = LABEL_WIDTH + (prevBar?.barEnd ?? 0) * 400;
          const connY1 = (i - 1) * (BAR_HEIGHT + BAR_GAP) + BAR_HEIGHT;
          const connY2 = y;

          return (
            <g key={bar.label}>
              {/* Connector dashed line */}
              {showConnector && (
                <line
                  x1={connX}
                  y1={connY1}
                  x2={connX}
                  y2={connY2 + BAR_HEIGHT / 2}
                  stroke={color}
                  strokeWidth={1}
                  strokeDasharray="3 3"
                  strokeOpacity={0.4}
                  style={{
                    animation: `sla-connector-in 0.3s ease both`,
                    animationDelay: `${i * 80 + 120}ms`,
                  }}
                />
              )}

              {/* Bar background track */}
              <rect
                x={LABEL_WIDTH}
                y={y + BAR_HEIGHT * 0.2}
                width={400}
                height={BAR_HEIGHT * 0.6}
                rx={4}
                fill="var(--color-separator)"
                fillOpacity={0.3}
              />

              {/* The bar itself */}
              <g
                style={{
                  transformOrigin: `${transformOriginX}px ${y}px`,
                  animation: `sla-bar-in 0.45s cubic-bezier(0.34,1.56,0.64,1) both`,
                  animationDelay: animDelay,
                }}
              >
                {/* Glow behind bar */}
                <rect
                  x={bx - 1}
                  y={y + BAR_HEIGHT * 0.1}
                  width={bw + 2}
                  height={BAR_HEIGHT * 0.8}
                  rx={5}
                  fill={color}
                  fillOpacity={isHovered ? 0.2 : 0.1}
                  style={{ filter: `blur(4px)` }}
                />
                {/* Actual bar */}
                <rect
                  x={bx}
                  y={y + BAR_HEIGHT * 0.15}
                  width={bw}
                  height={BAR_HEIGHT * 0.7}
                  rx={4}
                  fill={color}
                  fillOpacity={isHovered ? 0.95 : 0.82}
                  style={{ cursor: 'pointer', transition: 'fill-opacity 0.15s ease' }}
                  onMouseEnter={(e) => {
                    const svg = (e.target as SVGElement).closest('svg');
                    if (!svg) return;
                    const rect = svg.getBoundingClientRect();
                    setTooltip({
                      barIndex: i,
                      x: e.clientX - rect.left,
                      y: e.clientY - rect.top,
                    });
                  }}
                  onMouseMove={(e) => {
                    const svg = (e.target as SVGElement).closest('svg');
                    if (!svg) return;
                    const rect = svg.getBoundingClientRect();
                    setTooltip({
                      barIndex: i,
                      x: e.clientX - rect.left,
                      y: e.clientY - rect.top,
                    });
                  }}
                />
                {/* Bar shine overlay */}
                <rect
                  x={bx}
                  y={y + BAR_HEIGHT * 0.15}
                  width={bw}
                  height={BAR_HEIGHT * 0.35}
                  rx={4}
                  fill="white"
                  fillOpacity={0.12}
                  style={{ pointerEvents: 'none' }}
                />
              </g>

              {/* Label — left side */}
              <text
                x={LABEL_WIDTH - 8}
                y={y + BAR_HEIGHT / 2 + 1}
                textAnchor="end"
                dominantBaseline="middle"
                fontSize={11}
                fontWeight={isHovered ? 700 : 500}
                fill={isHovered ? 'var(--color-foreground)' : 'var(--color-muted)'}
                style={{ transition: 'fill 0.15s ease, font-weight 0.15s ease' }}
              >
                {bar.label}
              </text>

              {/* Value label — right of bar */}
              <g
                style={{
                  animation: `sla-bar-in 0.3s ease both`,
                  animationDelay: `${i * 80 + 320}ms`,
                }}
              >
                <text
                  x={bx + bw + 7}
                  y={y + BAR_HEIGHT / 2 + 1}
                  dominantBaseline="middle"
                  fontSize={11}
                  fontWeight={700}
                  fill={color}
                >
                  {bar.type === 'positive'
                    ? `-${bar.displayValue}`
                    : bar.type === 'negative'
                      ? `+${bar.displayValue}`
                      : `${bar.displayValue}`}
                </text>
              </g>
            </g>
          );
        })}
      </svg>

      {/* Floating tooltip */}
      {tooltip !== null && activeBar && (
        <div
          className="glass-card"
          style={{
            position: 'absolute',
            left: tooltip.x + 12,
            top: tooltip.y - 48,
            pointerEvents: 'none',
            padding: '8px 12px',
            zIndex: 50,
            minWidth: 160,
            animation: 'fade-in-up 0.15s ease both',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 4 }}>
            <div
              style={{
                width: 8,
                height: 8,
                borderRadius: 2,
                background: BAR_COLORS[activeBar.type],
                flexShrink: 0,
              }}
            />
            <span style={{ fontSize: 11, fontWeight: 700, color: 'var(--color-foreground)' }}>
              {activeBar.label}
            </span>
          </div>
          <div style={{ display: 'flex', justifyContent: 'space-between', gap: 16, fontSize: 11 }}>
            <span style={{ color: 'var(--color-muted)' }}>{BAR_LABELS[activeBar.type]}</span>
            <span
              style={{
                fontWeight: 700,
                color: BAR_COLORS[activeBar.type],
              }}
            >
              {activeBar.type === 'positive'
                ? `−${activeBar.displayValue} patches`
                : activeBar.type === 'negative'
                  ? `+${activeBar.displayValue} CVEs`
                  : `${activeBar.displayValue} days`}
            </span>
          </div>
          {activeBar.runningTotal > 0 && (
            <div
              style={{
                marginTop: 4,
                paddingTop: 4,
                borderTop: '1px solid var(--color-separator)',
                fontSize: 10,
                color: 'var(--color-muted)',
              }}
            >
              Running total:{' '}
              <span style={{ fontWeight: 600, color: 'var(--color-foreground)' }}>
                {activeBar.runningTotal}
              </span>
            </div>
          )}
        </div>
      )}

      {/* Legend */}
      <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap', marginTop: 12 }}>
        {(Object.entries(BAR_COLORS) as [keyof typeof BAR_COLORS, string][]).map(
          ([type, color]) => (
            <div key={type} style={{ display: 'flex', alignItems: 'center', gap: 5 }}>
              <div
                style={{
                  width: 10,
                  height: 10,
                  borderRadius: 3,
                  background: color,
                  opacity: 0.85,
                }}
              />
              <span style={{ fontSize: 10, fontWeight: 500, color: 'var(--color-muted)' }}>
                {BAR_LABELS[type]}
              </span>
            </div>
          ),
        )}
      </div>
    </div>
  );
}
