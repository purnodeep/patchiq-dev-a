import * as React from 'react';
import { cn } from '@/lib/utils';
import { AlertTriangle, AlertCircle, Info, X } from 'lucide-react';

type AlertSeverity = 'critical' | 'warning' | 'info';

interface AlertBannerProps {
  severity?: AlertSeverity;
  message: string;
  detail?: string;
  onDismiss?: () => void;
  className?: string;
}

const severityStyles: Record<
  AlertSeverity,
  { bg: string; border: string; text: string; glow: string }
> = {
  critical: {
    bg: 'bg-red-500/4',
    border: 'border-red-500/20',
    text: 'text-red-500',
    glow: 'animate-alert-glow-red',
  },
  warning: {
    bg: 'bg-amber-500/4',
    border: 'border-amber-500/20',
    text: 'text-amber-500',
    glow: 'animate-alert-glow-amber',
  },
  info: {
    bg: 'bg-blue-500/4',
    border: 'border-blue-500/20',
    text: 'text-blue-500',
    glow: 'animate-alert-glow-blue',
  },
};

const severityIcons: Record<AlertSeverity, React.ComponentType<{ className?: string }>> = {
  critical: AlertCircle,
  warning: AlertTriangle,
  info: Info,
};

function AlertBanner({
  severity = 'warning',
  message,
  detail,
  onDismiss,
  className,
}: AlertBannerProps) {
  const styles = severityStyles[severity];
  const Icon = severityIcons[severity];

  return (
    <div
      className={cn(
        'flex items-center gap-3 rounded-lg border px-4 py-2.5 backdrop-blur-sm',
        styles.bg,
        styles.border,
        styles.glow,
        className,
      )}
      role="alert"
    >
      <Icon className={cn('h-4 w-4 shrink-0', styles.text)} />
      <span className={cn('text-xs font-semibold', styles.text)}>{message}</span>
      {detail && <span className="text-xs text-muted-foreground">{detail}</span>}
      {onDismiss && (
        <button
          onClick={onDismiss}
          className="ml-auto shrink-0 rounded p-0.5 text-muted-foreground transition-colors hover:text-foreground"
          aria-label="Dismiss alert"
        >
          <X className="h-3.5 w-3.5" />
        </button>
      )}
    </div>
  );
}

export { AlertBanner };
export type { AlertBannerProps, AlertSeverity };
