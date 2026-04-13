import { useMemo } from 'react';
import { SkeletonCard, ErrorState } from '@patchiq/ui';
import { useMTTR } from '@/api/hooks/useDashboard';

const W = 400;
const H = 200;
const PAD = { top: 20, right: 16, bottom: 32, left: 40 };
const cW = W - PAD.left - PAD.right;
const cH = H - PAD.top - PAD.bottom;

const SEVERITY_CONFIG: Record<string, { color: string; label: string }> = {
  critical: { color: 'var(--signal-critical)', label: 'Critical' },
  high: { color: 'var(--signal-warning)', label: 'High' },
  medium: { color: '#eab308', label: 'Medium' },
};

interface SeriesData {
  severity: string;
  points: { x: number; y: number }[];
  latestValue: number;
}

function catmullRomPath(pts: { x: number; y: number }[]): string {
  if (pts.length < 2) return '';
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
  return d;
}

function areaFromLine(linePath: string, pts: { x: number; y: number }[]): string {
  if (pts.length < 2) return '';
  const baseline = PAD.top + cH;
  return linePath + ` L ${pts[pts.length - 1].x} ${baseline} L ${pts[0].x} ${baseline} Z`;
}

export function MTTRDecayCurve() {
  const { data, isLoading, error, refetch } = useMTTR();

  const { series, weeks, gridLines } = useMemo(() => {
    if (!data || data.length === 0)
      return { series: [] as SeriesData[], weeks: [] as string[], maxHours: 0, gridLines: [] as { y: number; val: number }[] };

    // Group by severity, ordered by week
    const weekSet = [...new Set(data.map((e) => e.week))].sort();
    const bySeverity = new Map<string, Map<string, number>>();
    for (const entry of data) {
      if (!bySeverity.has(entry.severity)) bySeverity.set(entry.severity, new Map());
      bySeverity.get(entry.severity)!.set(entry.week, entry.avg_hours);
    }

    const max = Math.max(...data.map((e) => e.avg_hours), 1);

    const result: SeriesData[] = [];
    for (const [sev, weekMap] of bySeverity) {
      if (!SEVERITY_CONFIG[sev]) continue;
      const points = weekSet.map((w, i) => ({
        x: PAD.left + (i / Math.max(weekSet.length - 1, 1)) * cW,
        y: PAD.top + cH - ((weekMap.get(w) ?? 0) / max) * cH,
      }));
      const vals = weekSet.map((w) => weekMap.get(w) ?? 0);
      result.push({ severity: sev, points, latestValue: vals[vals.length - 1] });
    }

    const gl: { y: number; val: number }[] = [];
    for (let i = 0; i <= 4; i++) {
      gl.push({ y: PAD.top + (i / 4) * cH, val: Math.round(max * (1 - i / 4)) });
    }

    return { series: result, weeks: weekSet, maxHours: max, gridLines: gl };
  }, [data]);

  if (isLoading)
    return (
      <div
        className="h-full rounded-lg border p-4"
        style={{ background: 'var(--bg-card)', borderColor: 'var(--border)' }}
      >
        <SkeletonCard lines={5} />
      </div>
    );
  if (error)
    return (
      <div
        className="h-full rounded-lg border p-4"
        style={{ background: 'var(--bg-card)', borderColor: 'var(--border)' }}
      >
        <ErrorState message="Failed to load MTTR data" onRetry={() => refetch()} />
      </div>
    );
  if (!series.length)
    return (
      <div
        className="flex h-full items-center justify-center rounded-lg border"
        style={{
          background: 'var(--bg-card)',
          borderColor: 'var(--border)',
          color: 'var(--text-muted)',
        }}
      >
        No remediation data available
      </div>
    );

  return (
    <div
      className="flex flex-col overflow-hidden rounded-lg border"
      style={{
        background: 'var(--bg-card)',
        borderColor: 'var(--border)',
        boxShadow: 'var(--shadow-sm)',
      }}
    >
      <div style={{ padding: '16px 16px 4px', flexShrink: 0 }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <div>
            <h3 className="text-sm font-semibold" style={{ color: 'var(--text-emphasis)' }}>
              Mean Time to Remediate
            </h3>
            <p className="text-xs" style={{ color: 'var(--text-secondary)' }}>
              Average hours to patch by severity
            </p>
          </div>
          <div style={{ display: 'flex', gap: 12 }}>
            {series.map((s) => {
              const cfg = SEVERITY_CONFIG[s.severity];
              return (
                <div key={s.severity} style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                  <div style={{ width: 8, height: 8, borderRadius: 2, background: cfg.color }} />
                  <span style={{ fontSize: 10, color: 'var(--text-muted)', fontFamily: 'var(--font-mono)' }}>
                    {cfg.label}: {Math.round(s.latestValue)}h
                  </span>
                </div>
              );
            })}
          </div>
        </div>
      </div>
      <div style={{ flex: 1, minHeight: 0, padding: '0 16px 12px' }}>
        <svg
          viewBox={`0 0 ${W} ${H}`}
          style={{ width: '100%', height: '100%' }}
          preserveAspectRatio="xMidYMid meet"
        >
          <defs>
            {series.map((s) => {
              const cfg = SEVERITY_CONFIG[s.severity];
              return (
                <linearGradient key={s.severity} id={`mttr-grad-${s.severity}`} x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor={cfg.color} stopOpacity={0.25} />
                  <stop offset="100%" stopColor={cfg.color} stopOpacity={0} />
                </linearGradient>
              );
            })}
          </defs>

          {/* Grid */}
          {gridLines.map((gl, i) => (
            <g key={i}>
              <line
                x1={PAD.left}
                y1={gl.y}
                x2={W - PAD.right}
                y2={gl.y}
                stroke="var(--border)"
                strokeWidth={1}
                strokeOpacity={0.5}
              />
              {i < gridLines.length - 1 && (
                <text
                  x={PAD.left - 4}
                  y={gl.y + 3}
                  textAnchor="end"
                  fontSize={8}
                  fill="var(--text-muted)"
                  fontFamily="var(--font-mono)"
                >
                  {gl.val}h
                </text>
              )}
            </g>
          ))}

          {/* Areas and lines */}
          {series.map((s) => {
            const cfg = SEVERITY_CONFIG[s.severity];
            const line = catmullRomPath(s.points);
            const area = areaFromLine(line, s.points);
            return (
              <g key={s.severity}>
                <path d={area} fill={`url(#mttr-grad-${s.severity})`} />
                <path
                  d={line}
                  fill="none"
                  stroke={cfg.color}
                  strokeWidth={1.5}
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
                {s.points.length > 0 && (
                  <circle
                    cx={s.points[s.points.length - 1].x}
                    cy={s.points[s.points.length - 1].y}
                    r={3}
                    fill={cfg.color}
                  />
                )}
              </g>
            );
          })}

          {/* X-axis labels */}
          {weeks.length > 0 &&
            (() => {
              const count = Math.min(weeks.length, 6);
              const step = (weeks.length - 1) / Math.max(count - 1, 1);
              return Array.from({ length: count }, (_, i) => {
                const idx = Math.round(i * step);
                const x = PAD.left + (idx / Math.max(weeks.length - 1, 1)) * cW;
                return (
                  <text
                    key={idx}
                    x={x}
                    y={H - PAD.bottom + 14}
                    textAnchor="middle"
                    fontSize={8}
                    fill="var(--text-muted)"
                    fontFamily="var(--font-mono)"
                  >
                    {weeks[idx]}
                  </text>
                );
              });
            })()}
        </svg>
      </div>
    </div>
  );
}
