import { cn } from '@/lib/utils';

interface RingChartProps {
  /** Current value */
  value: number;
  /** Maximum value (denominator for the ring) */
  max: number;
  /** Diameter in pixels */
  size?: number;
  /** Ring stroke width */
  thickness?: number;
  /** Ring color (any CSS color) */
  color?: string;
  /** Track color behind the ring */
  trackColor?: string;
  /** Label shown below the value in the center */
  label?: string;
  /** Whether to animate the stroke on mount */
  animated?: boolean;
  className?: string;
}

function RingChart({
  value,
  max,
  size = 60,
  thickness = 6,
  color = 'hsl(var(--primary))',
  trackColor = 'hsl(var(--border))',
  label,
  animated = true,
  className,
}: RingChartProps) {
  const radius = (size - thickness) / 2;
  const circumference = 2 * Math.PI * radius;
  const pct = max > 0 ? Math.min(1, value / max) : 0;
  const filledLength = pct * circumference;
  const gapLength = circumference - filledLength;
  const center = size / 2;

  return (
    <svg
      width={size}
      height={size}
      viewBox={`0 0 ${size} ${size}`}
      className={cn('shrink-0', className)}
      role="img"
      aria-label={`${value} of ${max}${label ? ` ${label}` : ''}`}
    >
      {/* Track */}
      <circle
        cx={center}
        cy={center}
        r={radius}
        fill="none"
        stroke={trackColor}
        strokeWidth={thickness}
      />
      {/* Filled arc */}
      <circle
        cx={center}
        cy={center}
        r={radius}
        fill="none"
        stroke={color}
        strokeWidth={thickness}
        strokeDasharray={`${filledLength} ${gapLength}`}
        strokeLinecap="round"
        transform={`rotate(-90 ${center} ${center})`}
        style={
          animated
            ? { strokeDashoffset: circumference, animation: 'ring-sweep 0.8s ease forwards' }
            : undefined
        }
      />
      {/* Center value */}
      <text
        x={center}
        y={label ? center - 3 : center}
        textAnchor="middle"
        dominantBaseline="central"
        className="fill-foreground text-[9px] font-bold"
      >
        {value}
      </text>
      {/* Center label */}
      {label && (
        <text
          x={center}
          y={center + 9}
          textAnchor="middle"
          dominantBaseline="central"
          className="fill-muted-foreground text-[7px]"
        >
          {label}
        </text>
      )}
    </svg>
  );
}

export { RingChart };
export type { RingChartProps };
