import { useState, useMemo } from 'react';
import { SkeletonCard, ErrorState } from '@patchiq/ui';
import { useDrift } from '@/api/hooks/useDashboard';
import type { DriftEntry } from '@/api/hooks/useDashboard';

const W = 480;
const H = 280;
const PAD = { top: 24, right: 20, bottom: 24, left: 20 };
const cW = W - PAD.left - PAD.right;
const cH = H - PAD.top - PAD.bottom;

function driftColor(score: number): string {
  if (score > 70) return 'var(--signal-critical)';
  if (score > 40) return 'var(--signal-warning)';
  return '#eab308';
}

function dotRadius(unpatched: number): number {
  return Math.max(4, Math.min(16, 4 + unpatched * 0.6));
}

function daysSince(dateStr: string | null): string {
  if (!dateStr) return 'Never compliant';
  const days = Math.round((Date.now() - new Date(dateStr).getTime()) / (1000 * 60 * 60 * 24));
  if (days === 0) return 'Today';
  return `${days}d ago`;
}

interface TooltipData {
  x: number;
  y: number;
  entry: DriftEntry;
}

export function DriftDetector() {
  const { data, isLoading, error, refetch } = useDrift();
  const [tooltip, setTooltip] = useState<TooltipData | null>(null);

  const dots = useMemo(() => {
    if (!data || data.length === 0) return [];
    // Distribute dots horizontally with some jitter to avoid overlap
    const sorted = [...data].sort((a, b) => b.drift_score - a.drift_score);
    return sorted.map((entry, i) => {
      const xBase = PAD.left + ((i + 0.5) / sorted.length) * cW;
      // Y: center line = score 0, edges = score 100
      // Map drift_score 0-100 to vertical position from center
      const centerY = PAD.top + cH / 2;
      const maxOffset = cH / 2 - 16;
      // Alternate above/below center for visual spread
      const direction = i % 2 === 0 ? -1 : 1;
      const yOffset = (entry.drift_score / 100) * maxOffset * direction;
      return {
        entry,
        x: xBase,
        y: centerY + yOffset,
        r: dotRadius(entry.unpatched_count),
        color: driftColor(entry.drift_score),
      };
    });
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
        <ErrorState message="Failed to load drift data" onRetry={() => refetch()} />
      </div>
    );
  if (!dots.length)
    return (
      <div
        className="flex h-full items-center justify-center rounded-lg border"
        style={{
          background: 'var(--bg-card)',
          borderColor: 'var(--border)',
          color: 'var(--text-muted)',
        }}
      >
        All endpoints are compliant
      </div>
    );

  const centerY = PAD.top + cH / 2;

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
          Drift Detector
        </h3>
        <p className="text-xs" style={{ color: 'var(--text-secondary)' }}>
          Endpoint compliance drift from baseline
        </p>
      </div>
      <div style={{ flex: 1, minHeight: 0, padding: '0 16px 16px' }}>
        <svg
          viewBox={`0 0 ${W} ${H}`}
          style={{ width: '100%', height: '100%' }}
          preserveAspectRatio="xMidYMid meet"
          onMouseLeave={() => setTooltip(null)}
        >
          {/* Center "compliant" line */}
          <line
            x1={PAD.left}
            y1={centerY}
            x2={W - PAD.right}
            y2={centerY}
            stroke="var(--signal-success)"
            strokeWidth={1.5}
            strokeDasharray="6 3"
            strokeOpacity={0.6}
          />
          <text
            x={W - PAD.right + 2}
            y={centerY + 3}
            fontSize={8}
            fill="var(--signal-success)"
            fontFamily="var(--font-mono)"
          >
            Compliant
          </text>

          {/* Edge zone labels */}
          <text
            x={W - PAD.right + 2}
            y={PAD.top + 8}
            fontSize={7}
            fill="var(--text-muted)"
            fontFamily="var(--font-mono)"
          >
            High Drift
          </text>
          <text
            x={W - PAD.right + 2}
            y={H - PAD.bottom - 2}
            fontSize={7}
            fill="var(--text-muted)"
            fontFamily="var(--font-mono)"
          >
            High Drift
          </text>

          {/* Dots */}
          {dots.map((d) => (
            <circle
              key={d.entry.id}
              cx={d.x}
              cy={d.y}
              r={d.r}
              fill={d.color}
              fillOpacity={0.6}
              stroke={d.color}
              strokeWidth={1.5}
              style={{ cursor: 'pointer' }}
              onMouseEnter={() => setTooltip({ x: d.x, y: d.y, entry: d.entry })}
              onMouseLeave={() => setTooltip(null)}
            />
          ))}

          {/* Tooltip */}
          {tooltip && (
            <g>
              <rect
                x={Math.min(tooltip.x + 12, W - 170)}
                y={Math.max(tooltip.y - 52, 4)}
                width={160}
                height={60}
                rx={4}
                fill="var(--bg-card)"
                stroke="var(--border)"
                strokeWidth={1}
              />
              <text
                x={Math.min(tooltip.x + 18, W - 164)}
                y={Math.max(tooltip.y - 36, 20)}
                fontSize={10}
                fontWeight="bold"
                fill="var(--text-emphasis)"
                fontFamily="var(--font-mono)"
              >
                {tooltip.entry.hostname}
              </text>
              <text
                x={Math.min(tooltip.x + 18, W - 164)}
                y={Math.max(tooltip.y - 22, 34)}
                fontSize={9}
                fill="var(--text-secondary)"
                fontFamily="var(--font-mono)"
              >
                Drift: {tooltip.entry.drift_score}% | {tooltip.entry.unpatched_count} unpatched
              </text>
              <text
                x={Math.min(tooltip.x + 18, W - 164)}
                y={Math.max(tooltip.y - 8, 48)}
                fontSize={9}
                fill="var(--text-secondary)"
                fontFamily="var(--font-mono)"
              >
                Last compliant: {daysSince(tooltip.entry.last_compliant_at)}
              </text>
            </g>
          )}
        </svg>
      </div>
    </div>
  );
}
