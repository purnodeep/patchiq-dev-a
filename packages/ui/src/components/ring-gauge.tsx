import { cn } from '../lib/utils';

interface RingGaugeProps {
  /** Percentage value (0-100) */
  value: number;
  /** Label shown below the gauge */
  label?: string;
  /** Diameter in pixels */
  size?: number;
  /** Ring stroke width */
  strokeWidth?: number;
  /** Use signal colors based on value thresholds */
  colorByValue?: boolean;
  className?: string;
}

function getSignalColor(value: number): string {
  if (value >= 80) return 'var(--signal-healthy)';
  if (value >= 50) return 'var(--signal-warning)';
  return 'var(--signal-critical)';
}

function RingGauge({
  value,
  label,
  size = 80,
  strokeWidth = 6,
  colorByValue = false,
  className,
}: RingGaugeProps) {
  const clamped = Math.max(0, Math.min(100, value));
  const radius = (size - strokeWidth) / 2;
  const circumference = 2 * Math.PI * radius;
  const filled = (clamped / 100) * circumference;
  const gap = circumference - filled;
  const center = size / 2;
  const strokeColor = colorByValue ? getSignalColor(clamped) : 'var(--accent)';

  return (
    <div className={cn('inline-flex flex-col items-center', className)}>
      <svg
        width={size}
        height={size}
        viewBox={`0 0 ${size} ${size}`}
        role="img"
        aria-label={`${clamped}%${label ? ` ${label}` : ''}`}
      >
        {/* Track */}
        <circle
          cx={center}
          cy={center}
          r={radius}
          fill="none"
          stroke="var(--ring-track)"
          strokeWidth={strokeWidth}
        />
        {/* Fill */}
        <circle
          cx={center}
          cy={center}
          r={radius}
          fill="none"
          stroke={strokeColor}
          strokeWidth={strokeWidth}
          strokeDasharray={`${filled} ${gap}`}
          strokeLinecap="round"
          transform={`rotate(-90 ${center} ${center})`}
        />
        {/* Center text */}
        <text
          x={center}
          y={center}
          textAnchor="middle"
          dominantBaseline="central"
          fill="var(--text-emphasis)"
          fontFamily="var(--font-sans)"
          fontSize={size * 0.2}
          fontWeight={600}
        >
          {Math.round(clamped)}%
        </text>
      </svg>
      {label && (
        <span className="text-xs" style={{ color: 'var(--text-secondary)' }}>
          {label}
        </span>
      )}
    </div>
  );
}

export { RingGauge };
export type { RingGaugeProps };
