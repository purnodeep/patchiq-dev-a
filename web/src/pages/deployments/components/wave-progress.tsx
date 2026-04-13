export interface Wave {
  id: string;
  wave_number: number;
  percentage: number;
  status: string;
  target_count: number;
  success_count: number;
  failed_count: number;
  delay_after_minutes: number;
  started_at?: string | null;
  completed_at?: string | null;
  eligible_at?: string | null;
}

interface WaveProgressProps {
  waves: Wave[];
  activeWave?: number;
  onWaveClick?: (waveNumber: number) => void;
}

const statusCfg: Record<string, { circle: string; line: string; text: string }> = {
  pending: { circle: 'var(--border)', line: 'var(--border)', text: 'var(--text-muted)' },
  running: { circle: 'var(--accent)', line: 'var(--accent)', text: 'var(--accent)' },
  completed: {
    circle: 'var(--signal-healthy)',
    line: 'var(--signal-healthy)',
    text: 'var(--signal-healthy)',
  },
  failed: {
    circle: 'var(--signal-critical)',
    line: 'var(--signal-critical)',
    text: 'var(--signal-critical)',
  },
  cancelled: { circle: 'var(--text-faint)', line: 'var(--border)', text: 'var(--text-muted)' },
};

function getStatusCfg(status: string) {
  return statusCfg[status] ?? statusCfg.pending;
}

function successRate(wave: Wave): string | null {
  if (wave.status !== 'completed' && wave.status !== 'failed') return null;
  const total = wave.success_count + wave.failed_count;
  if (total === 0) return null;
  return `${Math.round((wave.success_count / total) * 100)}%`;
}

export function WaveProgress({ waves, activeWave, onWaveClick }: WaveProgressProps) {
  if (waves.length === 0) return null;

  const sorted = [...waves].sort((a, b) => a.wave_number - b.wave_number);

  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'flex-start',
        overflowX: 'auto',
        padding: '8px 0',
        gap: 0,
      }}
    >
      {sorted.map((wave, idx) => {
        const cfg = getStatusCfg(wave.status);
        const isActive = activeWave === wave.wave_number;
        const isLast = idx === sorted.length - 1;
        const rate = successRate(wave);

        return (
          <div key={wave.id} style={{ display: 'flex', alignItems: 'flex-start' }}>
            {/* Wave node */}
            <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 3 }}>
              <button
                type="button"
                onClick={() => onWaveClick?.(wave.wave_number)}
                aria-label={`Wave ${wave.wave_number}: ${wave.status}`}
                style={{
                  width: 38,
                  height: 38,
                  borderRadius: '50%',
                  border: `2px solid ${cfg.circle}`,
                  background: `color-mix(in srgb, ${cfg.circle} 9%, transparent)`,
                  color: cfg.text,
                  fontFamily: 'var(--font-mono)',
                  fontSize: 13,
                  fontWeight: 700,
                  cursor: onWaveClick ? 'pointer' : 'default',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  transition: 'box-shadow 0.15s',
                  outline: 'none',
                  boxShadow: isActive
                    ? `0 0 0 3px color-mix(in srgb, ${cfg.circle} 20%, transparent)`
                    : undefined,
                  animation:
                    wave.status === 'running' ? 'pulse-dot 1.5s ease-in-out infinite' : undefined,
                }}
              >
                {wave.wave_number}
              </button>

              <div
                style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 1 }}
              >
                <span
                  style={{
                    fontFamily: 'var(--font-mono)',
                    fontSize: 10,
                    fontWeight: 600,
                    color: cfg.text,
                  }}
                >
                  {wave.percentage}%
                </span>
                {rate && (
                  <span style={{ fontFamily: 'var(--font-mono)', fontSize: 9, color: cfg.text }}>
                    {rate} ok
                  </span>
                )}
                <span
                  style={{
                    fontSize: 9,
                    color: 'var(--text-muted)',
                    fontFamily: 'var(--font-mono)',
                  }}
                >
                  {wave.target_count} targets
                </span>
              </div>
            </div>

            {/* Connector */}
            {!isLast && (
              <div style={{ display: 'flex', alignItems: 'center', paddingTop: 18 }}>
                <div
                  style={{
                    width: 40,
                    height: 2,
                    borderRadius: 1,
                    background: cfg.line,
                    opacity: 0.4,
                  }}
                />
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}
