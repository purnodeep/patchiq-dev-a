import { cn } from '@/lib/utils';

interface SparklineChartProps {
  /** Array of numeric values to plot */
  data: number[];
  /** Width in pixels */
  width?: number;
  /** Height in pixels */
  height?: number;
  /** Stroke color */
  color?: string;
  /** Whether to show area fill underneath */
  fill?: boolean;
  /** Whether to show an end-dot indicator */
  showDot?: boolean;
  className?: string;
}

function SparklineChart({
  data,
  width = 70,
  height = 36,
  color = 'hsl(var(--primary))',
  fill = true,
  showDot = true,
  className,
}: SparklineChartProps) {
  if (data.length < 2) return null;

  const min = Math.min(...data);
  const max = Math.max(...data);
  const range = max - min || 1;
  const padding = 4;
  const plotW = width - padding * 2;
  const plotH = height - padding * 2;

  const points = data.map((v, i) => {
    const x = padding + (i / (data.length - 1)) * plotW;
    const y = padding + plotH - ((v - min) / range) * plotH;
    return { x, y };
  });

  const linePath = points.map((p, i) => `${i === 0 ? 'M' : 'L'}${p.x},${p.y}`).join(' ');
  const areaPath = `${linePath} L${points[points.length - 1].x},${height} L${points[0].x},${height} Z`;
  const lastPoint = points[points.length - 1];

  return (
    <svg
      width={width}
      height={height}
      viewBox={`0 0 ${width} ${height}`}
      className={cn('shrink-0', className)}
      role="img"
      aria-label="Sparkline chart"
    >
      {fill && <path d={areaPath} fill={color} opacity={0.15} />}
      <path
        d={linePath}
        fill="none"
        stroke={color}
        strokeWidth={1.5}
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      {showDot && <circle cx={lastPoint.x} cy={lastPoint.y} r={2} fill={color} />}
    </svg>
  );
}

export { SparklineChart };
export type { SparklineChartProps };
