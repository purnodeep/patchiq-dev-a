import { useEffect, useRef } from 'react';
import { PATCH_SUCCESS_RATE_DATA } from '@/data/mock-data';

// ── Donut geometry ─────────────────────────────────────────────────────────────
const RADIUS = 48;
const STROKE_WIDTH = 12;
const CENTER = 60;
const VIEWBOX_SIZE = 120;

function circumference(r: number): number {
  return 2 * Math.PI * r;
}

// ── Main Component ─────────────────────────────────────────────────────────────
export function PatchSuccessRate() {
  const arcRef = useRef<SVGCircleElement>(null);

  const successPct = Math.round(
    (PATCH_SUCCESS_RATE_DATA.succeeded / PATCH_SUCCESS_RATE_DATA.total) * 100,
  );
  const failedPct = Math.round(
    (PATCH_SUCCESS_RATE_DATA.failed / PATCH_SUCCESS_RATE_DATA.total) * 100,
  );
  const pendingPct = Math.round(
    (PATCH_SUCCESS_RATE_DATA.pending / PATCH_SUCCESS_RATE_DATA.total) * 100,
  );

  const circ = circumference(RADIUS);
  const targetDash = `${(successPct / 100) * circ} ${circ - (successPct / 100) * circ}`;
  // Start arc at 12 o'clock
  const dashOffset = circ * 0.25;

  // Animate on mount
  useEffect(() => {
    const arc = arcRef.current;
    if (!arc) return;
    arc.style.strokeDasharray = `0 ${circ}`;
    arc.style.transition = 'none';
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        arc.style.transition = 'stroke-dasharray 1.1s cubic-bezier(0.22, 1, 0.36, 1)';
        arc.style.strokeDasharray = targetDash;
      });
    });
  }, [circ, targetDash]);

  const rows: {
    label: string;
    count: number;
    pct: number;
    color: string;
    icon: string;
  }[] = [
    {
      label: 'Succeeded',
      count: PATCH_SUCCESS_RATE_DATA.succeeded,
      pct: successPct,
      color: 'var(--color-success)',
      icon: '✓',
    },
    {
      label: 'Failed',
      count: PATCH_SUCCESS_RATE_DATA.failed,
      pct: failedPct,
      color: '#ef4444',
      icon: '✗',
    },
    {
      label: 'Pending',
      count: PATCH_SUCCESS_RATE_DATA.pending,
      pct: pendingPct,
      color: 'var(--color-primary)',
      icon: '○',
    },
  ];

  return (
    <div
      style={{
        height: '100%',
        display: 'flex',
        alignItems: 'center',
        gap: 16,
        overflow: 'hidden',
      }}
    >
      {/* Left: SVG donut */}
      <div
        style={{
          flexShrink: 0,
          width: '40%',
          aspectRatio: '1 / 1',
          maxHeight: '100%',
        }}
      >
        <svg
          viewBox={`0 0 ${VIEWBOX_SIZE} ${VIEWBOX_SIZE}`}
          width="100%"
          height="100%"
          style={{ display: 'block' }}
          aria-label={`Patch success rate: ${successPct}%`}
        >
          {/* Track (background ring) */}
          <circle
            cx={CENTER}
            cy={CENTER}
            r={RADIUS}
            fill="none"
            stroke="currentColor"
            strokeOpacity={0.08}
            strokeWidth={STROKE_WIDTH}
          />

          {/* Success arc */}
          <circle
            ref={arcRef}
            cx={CENTER}
            cy={CENTER}
            r={RADIUS}
            fill="none"
            stroke="var(--color-success)"
            strokeWidth={STROKE_WIDTH}
            strokeLinecap="round"
            strokeDasharray={targetDash}
            strokeDashoffset={dashOffset}
            style={{
              filter:
                'drop-shadow(0 0 5px color-mix(in srgb, var(--color-success) 50%, transparent))',
            }}
          />

          {/* Center: % */}
          <text
            x={CENTER}
            y={CENTER - 6}
            textAnchor="middle"
            dominantBaseline="middle"
            style={{
              fontSize: '22px',
              fontWeight: 700,
              fill: 'var(--color-foreground)',
              fontFamily: 'var(--font-sans)',
            }}
          >
            {successPct}%
          </text>

          {/* Center: label */}
          <text
            x={CENTER}
            y={CENTER + 10}
            textAnchor="middle"
            dominantBaseline="middle"
            style={{
              fontSize: '8px',
              fontWeight: 500,
              fill: 'var(--color-muted)',
              fontFamily: 'var(--font-sans)',
              textTransform: 'uppercase',
              letterSpacing: '0.07em',
            }}
          >
            Success
          </text>
        </svg>
      </div>

      {/* Right: breakdown rows */}
      <div
        style={{
          flex: 1,
          minWidth: 0,
          display: 'flex',
          flexDirection: 'column',
          justifyContent: 'center',
          gap: 0,
        }}
      >
        {rows.map(({ label, count, pct, color, icon }, i) => (
          <div key={label}>
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                padding: '6px 0',
              }}
            >
              {/* Icon */}
              <span
                style={{
                  fontSize: 13,
                  fontWeight: 700,
                  color,
                  width: 14,
                  textAlign: 'center',
                  flexShrink: 0,
                  lineHeight: 1,
                }}
              >
                {icon}
              </span>

              {/* Label */}
              <span
                style={{
                  flex: 1,
                  fontSize: 11,
                  color: 'var(--color-foreground)',
                  fontWeight: 500,
                }}
              >
                {label}
              </span>

              {/* Count */}
              <span
                style={{
                  fontSize: 13,
                  fontWeight: 700,
                  color: 'var(--color-foreground)',
                  fontFamily: 'var(--font-mono)',
                  fontVariantNumeric: 'tabular-nums',
                }}
              >
                {count}
              </span>

              {/* Percentage */}
              <span
                style={{
                  fontSize: 10,
                  color: 'var(--color-muted)',
                  fontVariantNumeric: 'tabular-nums',
                  minWidth: 30,
                  textAlign: 'right',
                }}
              >
                {pct}%
              </span>
            </div>

            {/* Separator (not after last row) */}
            {i < rows.length - 1 && (
              <div
                style={{
                  height: 1,
                  background: 'var(--color-separator)',
                  marginLeft: 22,
                }}
              />
            )}
          </div>
        ))}

        {/* Trend badge */}
        <div style={{ marginTop: 8 }}>
          <span
            style={{
              fontSize: 10,
              fontWeight: 600,
              padding: '3px 8px',
              borderRadius: 999,
              background: 'color-mix(in srgb, var(--color-success) 12%, transparent)',
              color: 'var(--color-success)',
              border: '1px solid color-mix(in srgb, var(--color-success) 25%, transparent)',
            }}
          >
            ↑ +{PATCH_SUCCESS_RATE_DATA.trend}% vs last cycle
          </span>
        </div>
      </div>
    </div>
  );
}
