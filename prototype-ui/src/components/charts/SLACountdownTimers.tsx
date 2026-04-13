import { SLA_COUNTDOWN_DATA } from '../../data/mock-data';

const RADIUS = 26;
const STROKE_WIDTH = 5;
const CENTER = 32;
const DIAMETER = CENTER * 2;

function circumference() {
  return 2 * Math.PI * RADIUS;
}

function colorForHours(h: number): string {
  if (h < 6) return 'var(--color-danger)';
  if (h < 24) return 'var(--color-warning)';
  if (h < 72) return 'var(--color-caution)';
  return 'var(--color-success)';
}

function labelColorForHours(h: number): string {
  if (h < 6) return 'var(--color-danger)';
  if (h < 24) return 'var(--color-warning)';
  if (h < 72) return 'var(--color-caution)';
  return 'var(--color-success)';
}

function formatTime(hours: number): string {
  if (hours <= 0) return 'Overdue';
  const days = Math.floor(hours / 24);
  const h = hours % 24;
  if (days > 0 && h > 0) return `${days}d ${h}h`;
  if (days > 0) return `${days}d`;
  return `${h}h`;
}

function shortPatchName(patch: string): string {
  // Remove KB/USN/RHSA prefix note and truncate
  const dashIdx = patch.indexOf(' — ');
  if (dashIdx !== -1) return patch.slice(0, dashIdx);
  return patch.length > 14 ? patch.slice(0, 13) + '…' : patch;
}

export default function SLACountdownTimers() {
  const c = circumference();
  // Start arc at 12 o'clock
  const rotateOffset = -90;

  return (
    <div className="w-full h-full flex flex-col gap-2" style={{ paddingTop: 8 }}>
      {/* 2×3 grid — circles capped so they don't overflow the widget */}
      <div
        className="grid gap-2"
        style={{
          gridTemplateColumns: 'repeat(3, 1fr)',
          gridTemplateRows: 'repeat(2, auto)',
        }}
      >
        {SLA_COUNTDOWN_DATA.map((item) => {
          const ratio = Math.min(item.hoursRemaining / item.totalHours, 1);
          const filled = ratio * c;
          const remaining = c - filled;
          const color = colorForHours(item.hoursRemaining);
          const isCritical = item.hoursRemaining < 6;
          const labelColor = labelColorForHours(item.hoursRemaining);

          return (
            <div
              key={item.id}
              className="flex flex-col items-center gap-1.5"
              style={{ minWidth: 0 }}
            >
              {/* Circle timer — capped at 60px so it stays compact */}
              <div
                className="relative"
                style={{ width: '100%', maxWidth: 60, aspectRatio: '1 / 1' }}
              >
                <svg
                  viewBox={`0 0 ${DIAMETER} ${DIAMETER}`}
                  width="100%"
                  height="100%"
                  style={{
                    transform: `rotate(${rotateOffset}deg)`,
                    display: 'block',
                    overflow: 'visible',
                  }}
                  aria-label={`SLA countdown: ${item.patch}`}
                >
                  {/* Track */}
                  <circle
                    cx={CENTER}
                    cy={CENTER}
                    r={RADIUS}
                    fill="none"
                    stroke="currentColor"
                    strokeOpacity={0.1}
                    strokeWidth={STROKE_WIDTH}
                  />
                  {/* Progress arc */}
                  <circle
                    cx={CENTER}
                    cy={CENTER}
                    r={RADIUS}
                    fill="none"
                    stroke={color}
                    strokeWidth={STROKE_WIDTH}
                    strokeLinecap="round"
                    strokeDasharray={`${filled} ${remaining}`}
                    style={{
                      transition: 'stroke-dasharray 0.9s cubic-bezier(0.22,1,0.36,1)',
                      filter: `drop-shadow(0 0 3px ${color}88)`,
                    }}
                  />
                </svg>

                {/* Center text (un-rotated) — scales with circle size via clamp */}
                <div
                  className="absolute inset-0 flex items-center justify-center"
                  style={{
                    fontSize: 'clamp(8px, 2.2vw, 14px)',
                    fontWeight: 700,
                    fontFamily: 'var(--font-sans)',
                    color: labelColor,
                    textAlign: 'center',
                    lineHeight: 1.1,
                  }}
                >
                  {formatTime(item.hoursRemaining)}
                </div>

                {/* Pulse ring for critically low SLA */}
                {isCritical && (
                  <div
                    className="absolute inset-0 rounded-full"
                    style={{
                      border: `2px solid ${color}`,
                      borderRadius: '50%',
                      animation: 'pulse-ring-sm 1.2s cubic-bezier(0.22,1,0.36,1) infinite',
                      pointerEvents: 'none',
                    }}
                  />
                )}
              </div>

              {/* Labels */}
              <div
                className="flex flex-col items-center gap-0.5"
                style={{ minWidth: 0, width: '100%' }}
              >
                <span
                  className="text-center font-semibold leading-tight"
                  style={{
                    fontSize: '10px',
                    color: 'var(--color-foreground)',
                    display: '-webkit-box',
                    WebkitLineClamp: 1,
                    WebkitBoxOrient: 'vertical',
                    overflow: 'hidden',
                    width: '100%',
                    textAlign: 'center',
                  }}
                  title={item.patch}
                >
                  {shortPatchName(item.patch)}
                </span>
                <span
                  className="text-center leading-none"
                  style={{
                    fontSize: '9px',
                    color: 'var(--color-muted)',
                    textAlign: 'center',
                  }}
                  title={item.cve}
                >
                  {item.cve}
                </span>
              </div>
            </div>
          );
        })}
      </div>

      {/* Color key — pinned to bottom */}
      <div
        className="flex items-center gap-3 pt-2 flex-wrap mt-auto"
        style={{ borderTop: '1px solid var(--color-separator)' }}
      >
        {[
          { color: 'var(--color-success)', label: '>72h' },
          { color: 'var(--color-caution)', label: '24–72h' },
          { color: 'var(--color-warning)', label: '<24h' },
          { color: 'var(--color-danger)', label: '<6h ⚠' },
        ].map(({ color, label }) => (
          <div key={label} className="flex items-center gap-1">
            <span className="w-2 h-2 rounded-full flex-shrink-0" style={{ background: color }} />
            <span className="text-xs" style={{ color: 'var(--color-muted)' }}>
              {label}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}
