import { Globe, Wifi, Monitor, Usb } from 'lucide-react';

const VECTOR_CONFIG: Record<
  string,
  { icon: typeof Globe; bg: string; color: string; border: string }
> = {
  Network: {
    icon: Globe,
    bg: 'color-mix(in srgb, var(--text-muted) 8%, transparent)',
    color: 'var(--text-secondary)',
    border: 'color-mix(in srgb, var(--text-muted) 20%, transparent)',
  },
  Adjacent: {
    icon: Wifi,
    bg: 'color-mix(in srgb, var(--text-muted) 8%, transparent)',
    color: 'var(--text-secondary)',
    border: 'color-mix(in srgb, var(--text-muted) 20%, transparent)',
  },
  Local: {
    icon: Monitor,
    bg: 'color-mix(in srgb, var(--text-muted) 8%, transparent)',
    color: 'var(--text-secondary)',
    border: 'color-mix(in srgb, var(--text-muted) 20%, transparent)',
  },
  Physical: {
    icon: Usb,
    bg: 'color-mix(in srgb, var(--text-muted) 8%, transparent)',
    color: 'var(--text-secondary)',
    border: 'color-mix(in srgb, var(--text-muted) 20%, transparent)',
  },
};

interface AttackVectorBadgeProps {
  vector: string | null;
  className?: string;
}

export function AttackVectorBadge({ vector }: AttackVectorBadgeProps) {
  if (!vector) return null;
  const config = VECTOR_CONFIG[vector];
  if (!config) return null;
  const Icon = config.icon;
  return (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 4,
        borderRadius: 9999,
        border: `1px solid ${config.border}`,
        padding: '2px 8px',
        fontSize: 11,
        fontWeight: 500,
        fontFamily: 'var(--font-mono)',
        background: config.bg,
        color: config.color,
        whiteSpace: 'nowrap',
      }}
    >
      <Icon style={{ width: 11, height: 11 }} />
      {vector}
    </span>
  );
}
