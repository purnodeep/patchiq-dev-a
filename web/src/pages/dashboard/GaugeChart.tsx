interface GaugeChartProps {
  value: number;
  label?: string;
  size?: number;
  color?: string;
}

export function GaugeChart({ value, label, size = 60, color }: GaugeChartProps) {
  const clamped = Math.round(Math.min(100, Math.max(0, value)));
  const fillColor = color ?? 'var(--accent)';

  const cx = size / 2;
  const cy = size / 2;
  const r = size * 0.4;
  const strokeWidth = size * 0.1;

  const startX = cx - r;
  const startY = cy;
  const endX = cx + r;
  const endY = cy;

  const circumference = Math.PI * r;
  const progress = (clamped / 100) * circumference;

  const trackPath = `M ${startX} ${startY} A ${r} ${r} 0 0 1 ${endX} ${endY}`;

  return (
    <svg width={size} height={size / 2 + strokeWidth} role="img" aria-label={`Gauge: ${clamped}%`}>
      {/* Background track */}
      <path
        d={trackPath}
        fill="none"
        stroke="var(--border)"
        strokeWidth={strokeWidth}
        strokeLinecap="round"
      />
      {/* Foreground arc */}
      <path
        d={trackPath}
        fill="none"
        stroke={fillColor}
        strokeWidth={strokeWidth}
        strokeLinecap="round"
        strokeDasharray={`${progress} ${circumference}`}
      />
      {/* Center value text */}
      <text
        x={cx}
        y={cy}
        textAnchor="middle"
        dominantBaseline="auto"
        fontSize={size * 0.22}
        fontWeight="600"
        fill="var(--text-emphasis)"
        fontFamily="var(--font-mono)"
      >
        {clamped}%
      </text>
      {/* Optional label */}
      {label && (
        <text
          x={cx}
          y={cy + size * 0.15}
          textAnchor="middle"
          dominantBaseline="hanging"
          fontSize={size * 0.15}
          fill="var(--text-muted)"
          fontFamily="var(--font-sans)"
        >
          {label}
        </text>
      )}
    </svg>
  );
}
