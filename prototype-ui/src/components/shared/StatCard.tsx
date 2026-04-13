import { GlassCard } from './GlassCard';
import { cn } from '@/lib/utils';

interface StatCardProps {
  icon: React.ReactNode;
  iconColor: string;
  value: string | number;
  valueColor?: string;
  label: string;
  trend?: { value: string; positive: boolean };
  trendText?: string;
  microViz?: React.ReactNode;
  className?: string;
  /** When true, renders without the GlassCard wrapper (use inside WidgetShell). */
  bare?: boolean;
}

export function StatCard({
  icon,
  iconColor,
  value,
  valueColor,
  label,
  trend,
  trendText,
  microViz,
  className,
  bare,
}: StatCardProps) {
  const content = (
    <div
      className={cn(
        'flex justify-between items-center',
        bare ? cn('h-full px-4', className) : undefined,
      )}
    >
      <div>
        <div
          className="mb-1.5 flex h-8 w-8 items-center justify-center rounded-md"
          style={{ background: `color-mix(in srgb, ${iconColor} 10%, transparent)` }}
        >
          <div style={{ color: iconColor }}>{icon}</div>
        </div>
        <div
          className="text-[26px] font-extrabold leading-none tracking-tight"
          style={valueColor ? { color: valueColor } : undefined}
        >
          {value}
        </div>
        <div className="mt-0.5 text-[12px] text-muted">{label}</div>
        {trend && (
          <div className="mt-1 flex items-center gap-1">
            <span
              className="text-[11px] font-semibold"
              style={{ color: trend.positive ? 'var(--color-success)' : 'var(--color-danger)' }}
            >
              {trend.positive ? '↑' : '↓'} {trend.value}
            </span>
            {trendText && <span className="text-[11px] text-subtle">{trendText}</span>}
          </div>
        )}
      </div>
      {microViz && <div className="mt-0">{microViz}</div>}
    </div>
  );

  if (bare) return content;

  return <GlassCard className={cn('px-4 pt-7 pb-3', className)}>{content}</GlassCard>;
}
