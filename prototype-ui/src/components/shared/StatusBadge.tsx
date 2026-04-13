import { cn } from '@/lib/utils';

type BadgeVariant =
  | 'critical'
  | 'high'
  | 'medium'
  | 'low'
  | 'running'
  | 'complete'
  | 'failed'
  | 'pending'
  | 'kev'
  | 'exploit'
  | 'info';

const VARIANTS: Record<BadgeVariant, string> = {
  critical: 'bg-danger/10 text-danger',
  high: 'bg-warning/10 text-warning',
  medium: 'bg-caution/12 text-caution dark:text-caution',
  low: 'bg-success/10 text-success',
  running: 'bg-primary/10 text-primary',
  complete: 'bg-success/8 text-success',
  failed: 'bg-danger/8 text-danger',
  pending: 'bg-warning/8 text-warning',
  kev: 'bg-purple/10 text-purple',
  exploit: 'bg-pink/10 text-pink',
  info: 'bg-primary/6 text-primary',
};

interface StatusBadgeProps {
  variant: BadgeVariant;
  children: React.ReactNode;
  className?: string;
}

export function StatusBadge({ variant, children, className }: StatusBadgeProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1 rounded-lg px-2.5 py-1 text-[11px] font-semibold',
        VARIANTS[variant],
        className,
      )}
    >
      {children}
    </span>
  );
}
