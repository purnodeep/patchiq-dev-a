import type { components } from '../../../api/types';

type DeploymentWave = components['schemas']['DeploymentWave'];

interface WavePipelineLanesProps {
  waves: DeploymentWave[];
  className?: string;
}

const statusCfg: Record<string, { fill: string; success: string; label: string; icon: string }> = {
  completed: {
    fill: 'var(--signal-healthy)',
    success: 'var(--signal-healthy)',
    label: 'Complete',
    icon: '✓',
  },
  running: { fill: 'var(--accent)', success: 'var(--accent)', label: 'In Progress', icon: '▶' },
  failed: {
    fill: 'var(--signal-critical)',
    success: 'var(--signal-healthy)',
    label: 'Failed',
    icon: '✗',
  },
};
const defaultCfg = {
  fill: 'var(--bg-inset)',
  success: 'var(--text-faint)',
  label: 'Waiting',
  icon: '◷',
};

export function WavePipelineLanes({ waves }: WavePipelineLanesProps) {
  const sorted = [...waves].sort((a, b) => a.wave_number - b.wave_number);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
      {sorted.map((wave, idx) => {
        const cfg = statusCfg[wave.status] ?? defaultCfg;
        const isPending = wave.status === 'pending';
        const total = wave.target_count;
        const succeeded = wave.success_count;
        const failed = wave.failed_count;
        const inProgress = wave.status === 'running' ? Math.max(0, total - succeeded - failed) : 0;
        const pending = Math.max(0, total - succeeded - failed - inProgress);

        const successPct = total > 0 ? (succeeded / total) * 100 : 0;
        const inProgPct = total > 0 ? (inProgress / total) * 100 : 0;
        const failPct = total > 0 ? (failed / total) * 100 : 0;
        const pendPct = total > 0 ? (pending / total) * 100 : 0;

        return (
          <div key={wave.id} style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
            {/* Gate marker */}
            <div
              style={{
                width: 28,
                flexShrink: 0,
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                gap: 2,
              }}
            >
              <div
                style={{
                  width: 20,
                  height: 20,
                  borderRadius: '50%',
                  background: isPending
                    ? 'var(--bg-inset)'
                    : `color-mix(in srgb, ${cfg.fill} 10%, transparent)`,
                  border: `1.5px solid ${isPending ? 'var(--border)' : cfg.fill}`,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: 9,
                  color: isPending ? 'var(--text-muted)' : cfg.fill,
                  fontWeight: 700,
                }}
              >
                {idx + 1}
              </div>
              {/* Connector to next */}
              {idx < sorted.length - 1 && (
                <div style={{ width: 1, height: 6, background: 'var(--border)' }} />
              )}
            </div>

            {/* Label */}
            <div
              style={{
                width: 52,
                flexShrink: 0,
                fontFamily: 'var(--font-mono)',
                fontSize: 10,
                color: 'var(--text-muted)',
                letterSpacing: '0.02em',
              }}
            >
              Wave {wave.wave_number}
            </div>

            {/* Segmented bar track */}
            <div
              style={{
                flex: 1,
                height: 6,
                borderRadius: 3,
                background: 'var(--bg-inset)',
                overflow: 'hidden',
                display: 'flex',
                gap: 1,
              }}
            >
              {successPct > 0 && (
                <div
                  style={{
                    width: `${successPct}%`,
                    height: '100%',
                    background: 'var(--signal-healthy)',
                    borderRadius: 3,
                    flexShrink: 0,
                    transition: 'width 0.6s ease',
                  }}
                />
              )}
              {inProgPct > 0 && (
                <div
                  style={{
                    width: `${inProgPct}%`,
                    height: '100%',
                    background: 'var(--accent)',
                    borderRadius: 3,
                    flexShrink: 0,
                    transition: 'width 0.6s ease',
                  }}
                />
              )}
              {failPct > 0 && (
                <div
                  style={{
                    width: `${failPct}%`,
                    height: '100%',
                    background: 'var(--signal-critical)',
                    borderRadius: 3,
                    flexShrink: 0,
                    transition: 'width 0.6s ease',
                  }}
                />
              )}
              {pendPct > 0 && (
                <div
                  style={{
                    width: `${pendPct}%`,
                    height: '100%',
                    background: 'var(--border)',
                    borderRadius: 3,
                    flexShrink: 0,
                  }}
                />
              )}
            </div>

            {/* Stats */}
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                flexShrink: 0,
                minWidth: 120,
              }}
            >
              <span
                style={{
                  fontFamily: 'var(--font-mono)',
                  fontSize: 10,
                  color: 'var(--text-muted)',
                }}
              >
                {succeeded}/{total}
              </span>
              {failed > 0 && (
                <span
                  style={{
                    fontFamily: 'var(--font-mono)',
                    fontSize: 10,
                    color: 'var(--signal-critical)',
                  }}
                >
                  {failed} failed
                </span>
              )}
              <span
                style={{
                  fontFamily: 'var(--font-mono)',
                  fontSize: 10,
                  fontWeight: 600,
                  color: isPending ? 'var(--text-faint)' : cfg.fill,
                }}
              >
                {cfg.icon} {cfg.label}
              </span>
            </div>

            {/* Success rate ring — small */}
            {!isPending && total > 0 && (
              <div style={{ flexShrink: 0 }}>
                <svg width={28} height={28} viewBox="0 0 28 28">
                  <circle
                    cx={14}
                    cy={14}
                    r={10}
                    fill="none"
                    stroke="var(--bg-inset)"
                    strokeWidth={3}
                  />
                  <circle
                    cx={14}
                    cy={14}
                    r={10}
                    fill="none"
                    stroke={
                      successPct >= 80
                        ? 'var(--signal-healthy)'
                        : failPct > 20
                          ? 'var(--signal-critical)'
                          : 'var(--signal-warning)'
                    }
                    strokeWidth={3}
                    strokeDasharray={`${(successPct / 100) * 62.8} 62.8`}
                    strokeLinecap="round"
                    transform="rotate(-90 14 14)"
                  />
                  <text
                    x={14}
                    y={14}
                    textAnchor="middle"
                    dominantBaseline="central"
                    style={{
                      fontFamily: 'var(--font-mono)',
                      fontSize: 7,
                      fill: 'var(--text-muted)',
                      fontWeight: 600,
                    }}
                  >
                    {Math.round(successPct)}%
                  </text>
                </svg>
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}
