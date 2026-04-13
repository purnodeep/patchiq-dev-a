export interface AgentRollout {
  total: number;
  installed: number;
  enrolled: number;
  healthy: number;
  scanning: number;
}

export interface AgentRolloutFlowProps {
  rollout: AgentRollout;
}

const MAX_BAR_WIDTH = 300;
const BAR_HEIGHT = 28;
const BAR_GAP = 36;
const SVG_WIDTH = 400;
const SVG_HEIGHT = 250;

interface Stage {
  label: string;
  count: number;
}

export function AgentRolloutFlow({ rollout }: AgentRolloutFlowProps) {
  const { total, installed, enrolled, healthy, scanning } = rollout;

  const stages: Stage[] = [
    { label: 'Total', count: total },
    { label: 'Installed', count: installed },
    { label: 'Enrolled', count: enrolled },
    { label: 'Healthy', count: healthy },
    { label: 'Scanning', count: scanning },
  ];

  const safeTotal = total > 0 ? total : 1;

  const dropOffLabels: string[] = [
    `\u2212${total - installed} not installed`,
    `\u2212${installed - enrolled} not enrolled`,
    `\u2212${enrolled - healthy} not healthy`,
    `\u2212${healthy - scanning} not scanning`,
  ];

  return (
    <div
      className="rounded-lg border"
      style={{
        background: 'var(--bg-card)',
        borderColor: 'var(--border)',
        boxShadow: 'var(--shadow-sm)',
      }}
    >
      <div className="p-4 pb-2">
        <h3 className="text-sm font-semibold" style={{ color: 'var(--text-emphasis)' }}>
          Agent Rollout
        </h3>
      </div>
      <div className="p-4 pt-0">
        <svg
          viewBox={`0 0 ${SVG_WIDTH} ${SVG_HEIGHT}`}
          width="100%"
          aria-label="Agent Rollout Flow"
        >
          {stages.map((stage, i) => {
            const barWidth = Math.round((stage.count / safeTotal) * MAX_BAR_WIDTH);
            const y = i * (BAR_HEIGHT + BAR_GAP);
            const pct = Math.round((stage.count / safeTotal) * 100);

            return (
              <g key={stage.label}>
                {/* Stage label */}
                <text
                  x={0}
                  y={y + BAR_HEIGHT / 2 + 5}
                  fill="var(--text-primary)"
                  fontSize={11}
                  fontFamily="var(--font-sans)"
                >
                  {stage.label}
                </text>

                {/* Background track */}
                <rect
                  x={75}
                  y={y}
                  width={MAX_BAR_WIDTH}
                  height={BAR_HEIGHT}
                  rx={4}
                  fill="var(--border)"
                />

                {/* Active bar */}
                <rect
                  x={75}
                  y={y}
                  width={barWidth}
                  height={BAR_HEIGHT}
                  rx={4}
                  fill="var(--accent)"
                  aria-label={`${stage.label}: ${stage.count}`}
                />

                {/* Count inside bar */}
                {barWidth > 30 && (
                  <text
                    x={75 + 8}
                    y={y + BAR_HEIGHT / 2 + 4}
                    fontSize={11}
                    fontFamily="var(--font-mono)"
                    fill="white"
                    fontWeight={600}
                  >
                    {stage.count}
                  </text>
                )}

                {/* Percentage label to the right */}
                <text
                  x={75 + MAX_BAR_WIDTH + 8}
                  y={y + BAR_HEIGHT / 2 + 4}
                  fontSize={11}
                  fontFamily="var(--font-mono)"
                  fill="var(--text-muted)"
                >
                  {pct}%
                </text>

                {/* Drop-off label between bars */}
                {i < stages.length - 1 && (
                  <text
                    x={75}
                    y={y + BAR_HEIGHT + BAR_GAP / 2 + 4}
                    fontSize={10}
                    fontFamily="var(--font-mono)"
                    fill="var(--signal-critical)"
                  >
                    {dropOffLabels[i]}
                  </text>
                )}
              </g>
            );
          })}
        </svg>
      </div>
    </div>
  );
}
