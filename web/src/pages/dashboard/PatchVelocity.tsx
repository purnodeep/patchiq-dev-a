import { useMemo, useRef, useState, useEffect } from 'react';
import { usePatchesTimeline } from '@/api/hooks/useDashboard';

const cardStyle: React.CSSProperties = {
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  borderRadius: 8,
  boxShadow: 'var(--shadow-sm)',
  transition: 'border-color 150ms ease',
  display: 'flex',
  flexDirection: 'column',
};

const PAD = { top: 16, right: 12, bottom: 28, left: 32 };

// Use the CSS var at render time; fall back to the default accent green
function getEmerald(): string {
  if (typeof document !== 'undefined') {
    const v = getComputedStyle(document.documentElement).getPropertyValue('--accent').trim();
    if (v) return v;
  }
  return 'var(--accent)';
}

/** Convert a W-keyed label like "W01" to a human date: "Mar 1", "Mar 8", etc. */
function weekLabelToDate(weekKey: string): string {
  // weekKey format: "W01" … "W13" where W13 is the most recent (current) week.
  // We have 13 weeks spanning the last 90 days.
  const weekNum = parseInt(weekKey.replace('W', ''), 10);
  // weekNum 13 = current week, weekNum 1 = ~12 weeks ago
  const weeksAgo = 13 - weekNum;
  const date = new Date();
  date.setDate(date.getDate() - weeksAgo * 7);
  return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
}

function buildChart(raw: number[], W: number, H: number): {
  linePath: string;
  areaPath: string;
  lastPt: { x: number; y: number };
  gridLines: Array<{ y: number; val: number }>;
} {
  if (raw.length === 0) {
    return { linePath: '', areaPath: '', lastPt: { x: 0, y: 0 }, gridLines: [] };
  }

  const cW = W - PAD.left - PAD.right;
  const cH = H - PAD.top - PAD.bottom;
  const maxV = Math.max(...raw, 1);
  const pts = raw.map((v, i) => ({
    x: PAD.left + (i / Math.max(raw.length - 1, 1)) * cW,
    y: PAD.top + cH - (v / maxV) * cH,
  }));

  // Catmull-Rom spline
  let d = `M ${pts[0].x} ${pts[0].y}`;
  for (let i = 0; i < pts.length - 1; i++) {
    const p0 = pts[Math.max(i - 1, 0)];
    const p1 = pts[i];
    const p2 = pts[i + 1];
    const p3 = pts[Math.min(i + 2, pts.length - 1)];
    const cp1x = p1.x + (p2.x - p0.x) / 6;
    const cp1y = p1.y + (p2.y - p0.y) / 6;
    const cp2x = p2.x - (p3.x - p1.x) / 6;
    const cp2y = p2.y - (p3.y - p1.y) / 6;
    d += ` C ${cp1x} ${cp1y}, ${cp2x} ${cp2y}, ${p2.x} ${p2.y}`;
  }

  const lastPt = pts[pts.length - 1];
  const firstPt = pts[0];
  const areaPath = d + ` L ${lastPt.x} ${PAD.top + cH} L ${firstPt.x} ${PAD.top + cH} Z`;

  const gridLevels = 4;
  const gridLines: Array<{ y: number; val: number }> = [];
  for (let i = 0; i <= gridLevels; i++) {
    gridLines.push({
      y: PAD.top + (i / gridLevels) * cH,
      val: Math.round(maxV * (1 - i / gridLevels)),
    });
  }

  return { linePath: d, areaPath, lastPt, gridLines };
}

export function PatchVelocity() {
  const { data: timeline, isLoading } = usePatchesTimeline();
  const chartRef = useRef<HTMLDivElement>(null);
  const [dims, setDims] = useState({ w: 340, h: 160 });

  useEffect(() => {
    if (!chartRef.current) return;
    const ro = new ResizeObserver((entries) => {
      const entry = entries[0];
      if (entry) {
        const { width, height } = entry.contentRect;
        if (width > 0 && height > 0) setDims({ w: width, h: height });
      }
    });
    ro.observe(chartRef.current);
    return () => ro.disconnect();
  }, []);

  const raw = useMemo(() => {
    if (!timeline || timeline.length === 0) return [];
    return timeline.map((entry) => entry.critical + entry.high + entry.medium);
  }, [timeline]);

  // Pick 5 evenly-spaced labels and convert from week keys to human dates
  const dateLabels = useMemo(() => {
    if (!timeline || timeline.length === 0) return [];
    const len = timeline.length;
    if (len <= 5) return timeline.map((e) => weekLabelToDate(e.date));
    const step = (len - 1) / 4;
    return Array.from({ length: 5 }, (_, i) =>
      weekLabelToDate(timeline[Math.round(i * step)].date),
    );
  }, [timeline]);

  const { linePath, areaPath, lastPt, gridLines } = useMemo(
    () => buildChart(raw, dims.w, dims.h),
    [raw, dims.w, dims.h],
  );

  const totalPatches = raw.reduce((sum, v) => sum + v, 0);
  const emerald = getEmerald();

  return (
    <div
      style={cardStyle}
      onMouseEnter={(e) => {
        (e.currentTarget as HTMLDivElement).style.borderColor = 'var(--text-faint)';
      }}
      onMouseLeave={(e) => {
        (e.currentTarget as HTMLDivElement).style.borderColor = 'var(--border)';
      }}
    >
      <div
        style={{
          padding: '16px 20px 0',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}
      >
        <span style={{ fontSize: 13, color: 'var(--text-muted)', fontWeight: 400 }}>
          Patch Velocity
        </span>
        <span style={{ fontSize: 11, color: 'var(--text-faint)', fontFamily: 'var(--font-mono)' }}>
          {isLoading ? '\u2014' : `${totalPatches} patches / 90d`}
        </span>
      </div>
      <div ref={chartRef} style={{ padding: '4px 20px 12px', flex: 1, minHeight: 100, overflow: 'hidden' }}>
        {isLoading ? (
          <div
            style={{
              height: '100%',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: 'var(--text-faint)',
              fontSize: 12,
            }}
          >
            Loading...
          </div>
        ) : raw.length === 0 ? (
          <div
            style={{
              height: '100%',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: 'var(--text-faint)',
              fontSize: 12,
            }}
          >
            No patch data
          </div>
        ) : (
          <>
            <svg
              style={{ width: '100%', height: 'calc(100% - 18px)', display: 'block' }}
              viewBox={`0 0 ${dims.w} ${dims.h}`}
              preserveAspectRatio="xMidYMid meet"
            >
              <defs>
                <linearGradient id="vel-grad" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor={emerald} stopOpacity={0.3} />
                  <stop offset="100%" stopColor={emerald} stopOpacity={0} />
                </linearGradient>
              </defs>
              {gridLines.map((gl, i) => (
                <g key={i}>
                  <line
                    x1={PAD.left}
                    y1={gl.y}
                    x2={dims.w - PAD.right}
                    y2={gl.y}
                    stroke="var(--chart-grid, var(--border))"
                    strokeWidth={1}
                  />
                  {i < gridLines.length - 1 && (
                    <text
                      x={PAD.left - 4}
                      y={gl.y + 4}
                      textAnchor="end"
                      fontFamily="var(--font-mono)"
                      fontSize={8}
                      fill="var(--chart-axis-fill, var(--text-faint))"
                    >
                      {gl.val}
                    </text>
                  )}
                </g>
              ))}
              <path d={areaPath} fill="url(#vel-grad)" />
              <path
                d={linePath}
                fill="none"
                stroke={emerald}
                strokeWidth={1.5}
                strokeLinecap="round"
                strokeLinejoin="round"
              />
              <circle cx={lastPt.x} cy={lastPt.y} r={3} fill={emerald} />
            </svg>
            <div style={{ display: 'flex', justifyContent: 'space-between', height: 14 }}>
              {dateLabels.map((label, i) => (
                <span
                  key={`${label}-${i}`}
                  style={{
                    fontFamily: 'var(--font-mono)',
                    fontSize: 9,
                    color: 'var(--chart-axis-fill, var(--text-faint))',
                  }}
                >
                  {label}
                </span>
              ))}
            </div>
          </>
        )}
      </div>
    </div>
  );
}
