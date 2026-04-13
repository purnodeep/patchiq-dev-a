type Severity = 'critical' | 'high' | 'medium' | 'low' | 'none';

const severityStyles: Record<Severity, { bg: string; color: string; border: string }> = {
  critical: { bg: 'transparent', color: 'var(--signal-critical)', border: 'var(--border-strong)' },
  high: { bg: 'transparent', color: 'var(--signal-warning)', border: 'var(--border-strong)' },
  medium: { bg: 'transparent', color: 'var(--signal-warning)', border: 'var(--border-strong)' },
  low: { bg: 'transparent', color: 'var(--signal-healthy)', border: 'var(--border-strong)' },
  none: { bg: 'transparent', color: 'var(--text-muted)', border: 'var(--border-strong)' },
};

interface SeverityBadgeProps {
  severity: Severity;
  className?: string;
}

export type { Severity };

export const SeverityBadge = ({ severity }: SeverityBadgeProps) => {
  const s = severityStyles[severity] ?? severityStyles.none;
  return (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 4,
        borderRadius: 4,
        border: `1px solid ${s.border}`,
        padding: '1px 6px',
        fontSize: 10,
        fontWeight: 600,
        fontFamily: 'var(--font-mono)',
        background: s.bg,
        color: s.color,
        textTransform: 'uppercase',
        letterSpacing: '0.03em',
        whiteSpace: 'nowrap',
      }}
    >
      {severity}
    </span>
  );
};
