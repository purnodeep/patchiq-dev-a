import { type ReactNode } from 'react';
import { cn } from '../lib/utils';

interface StatCardTrend {
  value: number;
  direction: 'up' | 'down' | 'flat';
}

interface StatCardProps {
  label: string;
  value: string | number;
  icon?: ReactNode;
  trend?: StatCardTrend;
  className?: string;
}

function getTrendColor(direction: StatCardTrend['direction']): string {
  if (direction === 'up') return 'var(--signal-healthy)';
  if (direction === 'down') return 'var(--signal-critical)';
  return 'var(--text-muted)';
}

function getTrendPrefix(direction: StatCardTrend['direction']): string {
  if (direction === 'up') return '+';
  if (direction === 'down') return '-';
  return '';
}

function StatCard({ label, value, icon, trend, className }: StatCardProps) {
  return (
    <div
      className={cn('rounded-lg p-4', className)}
      style={{
        backgroundColor: 'var(--bg-card)',
        borderWidth: '1px',
        borderStyle: 'solid',
        borderColor: 'var(--border)',
        boxShadow: 'var(--shadow-sm)',
      }}
    >
      <div className="flex items-center justify-between">
        <span className="text-xs uppercase tracking-wider" style={{ color: 'var(--text-muted)' }}>
          {label}
        </span>
        {icon && <span style={{ color: 'var(--text-muted)' }}>{icon}</span>}
      </div>
      <div className="mt-2 flex items-baseline gap-2">
        <span
          className="text-2xl font-semibold"
          style={{
            color: 'var(--text-emphasis)',
            fontFamily: 'var(--font-mono)',
          }}
        >
          {value}
        </span>
        {trend && (
          <span className="text-sm font-medium" style={{ color: getTrendColor(trend.direction) }}>
            {getTrendPrefix(trend.direction)}
            {trend.value}%
          </span>
        )}
      </div>
    </div>
  );
}

export { StatCard };
export type { StatCardProps, StatCardTrend };
