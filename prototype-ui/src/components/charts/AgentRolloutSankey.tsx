import { useState } from 'react';
import { AGENT_ROLLOUT_DATA } from '../../data/mock-data';

const STAGE_COLORS = [
  'var(--color-primary)',
  '#2196f3',
  'var(--color-cyan)',
  '#38bdf8',
  'var(--color-success)',
];

export default function AgentRolloutSankey() {
  const [hoveredIdx, setHoveredIdx] = useState<number | null>(null);

  const stages = AGENT_ROLLOUT_DATA;
  const total = stages[0].count;
  const last = stages[stages.length - 1];
  const operationalPct = Math.round((last.count / total) * 100);

  return (
    <div className="w-full h-full flex flex-col justify-between gap-1">
      {/* Stage rows */}
      <div className="flex flex-col gap-1 flex-1 justify-center">
        {stages.map((stage, i) => {
          const pct = stage.count / total;
          const isHovered = hoveredIdx === i;
          const dropped = i > 0 ? stages[i - 1].count - stage.count : 0;
          const dropPct = i > 0 ? Math.round((dropped / stages[i - 1].count) * 100) : 0;

          return (
            <div key={stage.stage}>
              {/* Drop-off connector between stages */}
              {i > 0 && (
                <div className="flex items-center gap-2 py-0.5 pl-[72px]">
                  <div
                    className="w-px self-stretch"
                    style={{ background: 'var(--color-border)', opacity: 0.5 }}
                  />
                  <span
                    className="text-xs font-medium tabular-nums"
                    style={{ color: 'var(--color-danger)', fontSize: '10px' }}
                  >
                    −{dropped.toLocaleString()} ({dropPct}%)
                  </span>
                </div>
              )}

              {/* Stage row */}
              <div
                className="flex items-center gap-2 cursor-pointer group"
                onMouseEnter={() => setHoveredIdx(i)}
                onMouseLeave={() => setHoveredIdx(null)}
              >
                {/* Stage label */}
                <span
                  className="text-right flex-shrink-0 font-medium transition-colors"
                  style={{
                    width: 68,
                    fontSize: '10px',
                    color: isHovered ? STAGE_COLORS[i] : 'var(--color-muted)',
                  }}
                >
                  {stage.stage}
                </span>

                {/* Bar track */}
                <div
                  className="relative flex-1 rounded-sm overflow-hidden"
                  style={{
                    height: 20,
                    background: 'var(--color-border)',
                    opacity: isHovered ? 1 : 0.85,
                  }}
                >
                  {/* Filled portion */}
                  <div
                    className="h-full rounded-sm transition-all duration-200"
                    style={{
                      width: `${pct * 100}%`,
                      background: STAGE_COLORS[i],
                      boxShadow: isHovered ? `0 0 8px ${STAGE_COLORS[i]}88` : undefined,
                    }}
                  />
                  {/* Count label inside bar */}
                  <span
                    className="absolute inset-0 flex items-center pl-2 font-semibold tabular-nums"
                    style={{
                      fontSize: '10px',
                      color: 'white',
                      textShadow: '0 1px 2px rgba(0,0,0,0.4)',
                      pointerEvents: 'none',
                    }}
                  >
                    {stage.count.toLocaleString()}
                  </span>
                </div>

                {/* Percentage */}
                <span
                  className="flex-shrink-0 font-semibold tabular-nums"
                  style={{
                    width: 34,
                    fontSize: '10px',
                    color: isHovered ? STAGE_COLORS[i] : 'var(--color-muted)',
                    textAlign: 'right',
                  }}
                >
                  {Math.round(pct * 100)}%
                </span>
              </div>
            </div>
          );
        })}
      </div>

      {/* Footer */}
      <div
        className="flex items-center justify-between pt-1 border-t"
        style={{ borderColor: 'var(--color-border)' }}
      >
        <span className="text-xs" style={{ color: 'var(--color-muted)' }}>
          {total.toLocaleString()} total targets
        </span>
        <span
          className="text-xs font-semibold tabular-nums"
          style={{ color: 'var(--color-success)' }}
        >
          {operationalPct}% fully operational
        </span>
      </div>
    </div>
  );
}
