import { useEffect, useRef } from 'react';
import { COMPLIANCE_RINGS_DATA } from '../../data/mock-data';

// Ring geometry — outermost to innermost
const RINGS = [
  { index: 0, radius: 72, strokeWidth: 9 },
  { index: 1, radius: 52, strokeWidth: 9 },
  { index: 2, radius: 32, strokeWidth: 9 },
];

const CENTER = 84;
const VIEWBOX_SIZE = 168;

function circumference(r: number) {
  return 2 * Math.PI * r;
}

function dashArray(score: number, r: number) {
  const c = circumference(r);
  const filled = (score / 100) * c;
  return `${filled} ${c - filled}`;
}

// Offset so arc starts at top (12 o'clock)
function dashOffset(r: number) {
  return circumference(r) * 0.25;
}

export default function ComplianceRings() {
  const svgRef = useRef<SVGSVGElement>(null);

  const overall = Math.round(
    COMPLIANCE_RINGS_DATA.reduce((sum, d) => sum + d.score, 0) / COMPLIANCE_RINGS_DATA.length,
  );

  // Animate rings in on mount by adding a class after paint
  useEffect(() => {
    const svg = svgRef.current;
    if (!svg) return;
    const arcs = svg.querySelectorAll<SVGCircleElement>('.compliance-arc');
    arcs.forEach((arc, i) => {
      arc.style.transition = `stroke-dasharray 1.1s cubic-bezier(0.22,1,0.36,1) ${i * 0.18}s`;
      // Start from 0 then animate to target
      const target = arc.getAttribute('data-dash') ?? '';
      arc.style.strokeDasharray = `0 ${circumference(RINGS[i].radius)}`;
      requestAnimationFrame(() => {
        requestAnimationFrame(() => {
          arc.style.strokeDasharray = target;
        });
      });
    });
  }, []);

  return (
    <div className="flex items-center gap-6 w-full h-full">
      {/* SVG rings — sized by width fraction so legend always has room */}
      <div
        style={{
          flexShrink: 0,
          width: 'min(42%, 180px)',
          aspectRatio: '1 / 1',
          alignSelf: 'center',
        }}
      >
        <svg
          ref={svgRef}
          viewBox={`0 0 ${VIEWBOX_SIZE} ${VIEWBOX_SIZE}`}
          width="100%"
          height="100%"
          style={{ display: 'block' }}
          aria-label="Compliance framework ring gauges"
        >
          {/* Track rings (background) */}
          {RINGS.map((ring, i) => (
            <circle
              key={`track-${i}`}
              cx={CENTER}
              cy={CENTER}
              r={ring.radius}
              fill="none"
              stroke="currentColor"
              strokeOpacity={0.08}
              strokeWidth={ring.strokeWidth}
            />
          ))}

          {/* Filled arcs */}
          {COMPLIANCE_RINGS_DATA.map((d, i) => {
            const ring = RINGS[i];
            const da = dashArray(d.score, ring.radius);
            return (
              <circle
                key={`arc-${i}`}
                className="compliance-arc"
                cx={CENTER}
                cy={CENTER}
                r={ring.radius}
                fill="none"
                stroke={d.color}
                strokeWidth={ring.strokeWidth}
                strokeLinecap="round"
                strokeDasharray={da}
                strokeDashoffset={dashOffset(ring.radius)}
                data-dash={da}
                style={{
                  filter: `drop-shadow(0 0 4px ${d.color}80)`,
                }}
              />
            );
          })}

          {/* Center score */}
          <text
            x={CENTER}
            y={CENTER - 5}
            textAnchor="middle"
            dominantBaseline="middle"
            style={{
              fontSize: '18px',
              fontWeight: 700,
              fill: 'var(--color-foreground)',
              fontFamily: 'var(--font-sans)',
            }}
          >
            {overall}%
          </text>
          <text
            x={CENTER}
            y={CENTER + 11}
            textAnchor="middle"
            dominantBaseline="middle"
            style={{
              fontSize: '8px',
              fontWeight: 500,
              fill: 'var(--color-muted)',
              fontFamily: 'var(--font-sans)',
              textTransform: 'uppercase',
              letterSpacing: '0.08em',
            }}
          >
            Overall
          </text>
        </svg>
      </div>

      {/* Legend */}
      <div className="flex flex-col gap-3 flex-1 min-w-0">
        {COMPLIANCE_RINGS_DATA.map((d, i) => (
          <div key={d.framework} className="flex items-center gap-2.5">
            {/* Color swatch dot matching ring */}
            <span
              className="flex-shrink-0 w-2.5 h-2.5 rounded-full"
              style={{
                background: d.color,
                boxShadow: `0 0 6px ${d.color}80`,
              }}
            />
            <div className="flex-1 min-w-0">
              <div className="flex items-center justify-between gap-1 mb-1">
                <span
                  className="text-xs font-semibold truncate"
                  style={{ color: 'var(--color-foreground)' }}
                >
                  {d.framework}
                </span>
                <span
                  className="text-xs font-bold tabular-nums flex-shrink-0"
                  style={{ color: d.color }}
                >
                  {d.score}%
                </span>
              </div>
              {/* Mini progress bar */}
              <div
                className="w-full h-1 rounded-full overflow-hidden"
                style={{ background: 'var(--color-separator)' }}
              >
                <div
                  className="h-full rounded-full"
                  style={{
                    width: `${d.score}%`,
                    background: d.color,
                    transition: `width 1.1s cubic-bezier(0.22,1,0.36,1) ${i * 0.18}s`,
                  }}
                />
              </div>
            </div>
          </div>
        ))}

        {/* Ring size key */}
        <div className="mt-1 pt-2" style={{ borderTop: '1px solid var(--color-separator)' }}>
          <p className="text-xs" style={{ color: 'var(--color-muted)' }}>
            Outer → NIST · Middle → PCI-DSS · Inner → HIPAA
          </p>
        </div>
      </div>
    </div>
  );
}
