import { useState, useMemo } from 'react';
import { SkeletonCard, ErrorState } from '@patchiq/ui';
import { useExposureWindows } from '@/api/hooks/useDashboard';
import type { ExposureWindow } from '@/api/hooks/useDashboard';

const W = 520;
const H_ROW = 32;
const LABEL_W = 140;
const PAD = { top: 40, right: 16, bottom: 8, left: LABEL_W + 8 };

function severityColor(severity: string): string {
  return severity === 'critical' ? 'var(--signal-critical)' : 'var(--signal-warning)';
}

function durationDays(from: string, to: string | null): number {
  const start = new Date(from).getTime();
  const end = to ? new Date(to).getTime() : Date.now();
  return Math.max(1, Math.round((end - start) / (1000 * 60 * 60 * 24)));
}

function formatDuration(days: number): string {
  if (days < 1) return '<1d';
  if (days < 30) return `${days}d`;
  return `${Math.floor(days / 30)}mo ${days % 30}d`;
}

interface TooltipData {
  x: number;
  y: number;
  item: ExposureWindow;
  days: number;
}

export function ExposureWindowTimeline() {
  const { data, isLoading, error, refetch } = useExposureWindows();
  const [tooltip, setTooltip] = useState<TooltipData | null>(null);

  const sorted = useMemo(() => {
    if (!data) return [];
    return [...data].sort(
      (a, b) => durationDays(b.first_seen, b.patched_at) - durationDays(a.first_seen, a.patched_at),
    );
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
        <ErrorState message="Failed to load exposure windows" onRetry={() => refetch()} />
      </div>
    );
  if (!sorted.length)
    return (
      <div
        className="flex h-full items-center justify-center rounded-lg border"
        style={{
          background: 'var(--bg-card)',
          borderColor: 'var(--border)',
          color: 'var(--text-muted)',
        }}
      >
        No exposure data available
      </div>
    );

  const maxDays = Math.max(...sorted.map((e) => durationDays(e.first_seen, e.patched_at)));
  const barW = W - PAD.left - PAD.right;
  const svgH = PAD.top + sorted.length * H_ROW + PAD.bottom;

  return (
    <div
      className="flex flex-col overflow-hidden rounded-lg border"
      style={{
        background: 'var(--bg-card)',
        borderColor: 'var(--border)',
        boxShadow: 'var(--shadow-sm)',
      }}
    >
      <div style={{ padding: '16px 16px 8px', flexShrink: 0 }}>
        <h3 className="text-sm font-semibold" style={{ color: 'var(--text-emphasis)' }}>
          Exposure Windows
        </h3>
        <p className="text-xs" style={{ color: 'var(--text-secondary)' }}>
          Duration of CVE exposure across your fleet
        </p>
      </div>
      <div style={{ flex: 1, minHeight: 0, overflow: 'auto', padding: '0 16px 16px', position: 'relative' }}>
        <svg
          viewBox={`0 0 ${W} ${svgH}`}
          style={{ width: '100%', minHeight: svgH }}
          preserveAspectRatio="xMidYMid meet"
          onMouseLeave={() => setTooltip(null)}
        >
          {/* Header line */}
          <line
            x1={PAD.left}
            y1={PAD.top - 4}
            x2={W - PAD.right}
            y2={PAD.top - 4}
            stroke="var(--border)"
            strokeWidth={1}
          />
          {/* Axis labels */}
          <text x={PAD.left} y={PAD.top - 10} fontSize={9} fill="var(--text-muted)" fontFamily="var(--font-mono)">
            0d
          </text>
          <text
            x={W - PAD.right}
            y={PAD.top - 10}
            fontSize={9}
            fill="var(--text-muted)"
            fontFamily="var(--font-mono)"
            textAnchor="end"
          >
            {maxDays}d
          </text>

          {sorted.map((item, i) => {
            const days = durationDays(item.first_seen, item.patched_at);
            const rowY = PAD.top + i * H_ROW;
            const w = (days / maxDays) * barW;
            const color = severityColor(item.severity);

            return (
              <g
                key={item.id}
                style={{ cursor: 'pointer' }}
                onMouseEnter={(e) => {
                  const svg = e.currentTarget.ownerSVGElement;
                  if (!svg) return;
                  const pt = svg.createSVGPoint();
                  pt.x = e.clientX;
                  pt.y = e.clientY;
                  const svgPt = pt.matrixTransform(svg.getScreenCTM()?.inverse());
                  setTooltip({ x: svgPt.x, y: svgPt.y, item, days });
                }}
                onMouseLeave={() => setTooltip(null)}
              >
                {/* Label */}
                <text
                  x={PAD.left - 8}
                  y={rowY + H_ROW / 2 + 3}
                  textAnchor="end"
                  fontSize={9}
                  fontFamily="var(--font-mono)"
                  fill="var(--text-secondary)"
                >
                  {item.cve_id.length > 18 ? item.cve_id.slice(0, 18) + '...' : item.cve_id}
                </text>
                {/* Severity dot */}
                <circle cx={PAD.left - LABEL_W + 4} cy={rowY + H_ROW / 2} r={3} fill={color} />
                {/* Bar */}
                <rect
                  x={PAD.left}
                  y={rowY + 4}
                  width={Math.max(w, 2)}
                  height={H_ROW - 8}
                  rx={3}
                  fill={color}
                  fillOpacity={item.patched_at ? 0.4 : 0.7}
                />
                {/* Duration label on bar */}
                {w > 30 && (
                  <text
                    x={PAD.left + w - 4}
                    y={rowY + H_ROW / 2 + 3}
                    textAnchor="end"
                    fontSize={8}
                    fontFamily="var(--font-mono)"
                    fill="var(--bg-card)"
                  >
                    {formatDuration(days)}
                  </text>
                )}
              </g>
            );
          })}

          {/* Tooltip */}
          {tooltip && (
            <g>
              <rect
                x={Math.min(tooltip.x + 8, W - 180)}
                y={tooltip.y - 50}
                width={170}
                height={56}
                rx={4}
                fill="var(--bg-card)"
                stroke="var(--border)"
                strokeWidth={1}
              />
              <text
                x={Math.min(tooltip.x + 14, W - 174)}
                y={tooltip.y - 34}
                fontSize={10}
                fontWeight="bold"
                fill="var(--text-emphasis)"
                fontFamily="var(--font-mono)"
              >
                {tooltip.item.cve_id}
              </text>
              <text
                x={Math.min(tooltip.x + 14, W - 174)}
                y={tooltip.y - 20}
                fontSize={9}
                fill="var(--text-secondary)"
                fontFamily="var(--font-mono)"
              >
                CVSS {tooltip.item.cvss} | {tooltip.item.affected_count} endpoints
              </text>
              <text
                x={Math.min(tooltip.x + 14, W - 174)}
                y={tooltip.y - 6}
                fontSize={9}
                fill="var(--text-secondary)"
                fontFamily="var(--font-mono)"
              >
                Exposed: {formatDuration(tooltip.days)}
                {tooltip.item.patched_at ? ' (patched)' : ' (open)'}
              </text>
            </g>
          )}
        </svg>
      </div>
    </div>
  );
}
