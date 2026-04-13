import type { ReactNode } from 'react';
import type { LucideIcon } from 'lucide-react';

type TrendDirection = 'up' | 'down';

interface StatCardProps {
  icon: LucideIcon;
  /**
   * Icon colors. Accepts either a Tailwind class string (legacy: "bg-green-500/10 text-green-500")
   * for backward compat, or a style object { bg, color }.
   */
  iconColor: string | { bg: string; color: string };
  value: string | number;
  label: string;
  sublabel?: string;
  trend?: {
    direction: TrendDirection;
    percentage: string;
    context?: string;
  };
  /** Optional visualization rendered top-right (RingChart, SparklineChart, etc.) */
  visualization?: ReactNode;
  className?: string;
  onClick?: () => void;
  compact?: boolean;
}

/**
 * Maps legacy Tailwind class strings to CSS values.
 * Only covers the patterns actually used in the codebase.
 */
function resolveIconColor(iconColor: string | { bg: string; color: string }): {
  bg: string;
  color: string;
} {
  if (typeof iconColor === 'object') return iconColor;

  const mapping: Record<string, { bg: string; color: string }> = {
    'bg-green-500/10 text-green-500': {
      bg: 'color-mix(in srgb, var(--signal-healthy) 10%, transparent)',
      color: 'var(--signal-healthy)',
    },
    'bg-red-500/10 text-red-500': {
      bg: 'color-mix(in srgb, var(--signal-critical) 1%, transparent)',
      color: 'var(--signal-critical)',
    },
    'bg-orange-500/10 text-orange-500': {
      bg: 'color-mix(in srgb, var(--signal-warning) 10%, transparent)',
      color: 'var(--signal-warning)',
    },
    'bg-amber-500/10 text-amber-500': {
      bg: 'color-mix(in srgb, var(--signal-warning) 10%, transparent)',
      color: 'var(--signal-warning)',
    },
    'bg-blue-500/10 text-blue-500': {
      bg: 'color-mix(in srgb, var(--accent) 10%, transparent)',
      color: 'var(--accent)',
    },
    'bg-purple-500/10 text-purple-500': {
      bg: 'color-mix(in srgb, var(--accent) 10%, transparent)',
      color: 'var(--accent)',
    },
    'bg-cyan-500/10 text-cyan-500': {
      bg: 'color-mix(in srgb, var(--accent) 10%, transparent)',
      color: 'var(--accent)',
    },
    'bg-emerald-500/10 text-emerald-500': {
      bg: 'color-mix(in srgb, var(--accent) 10%, transparent)',
      color: 'var(--accent)',
    },
    'bg-yellow-500/10 text-yellow-500': {
      bg: 'color-mix(in srgb, var(--signal-warning) 10%, transparent)',
      color: 'var(--signal-warning)',
    },
    'bg-gray-500/10 text-gray-500': {
      bg: 'color-mix(in srgb, var(--text-muted) 10%, transparent)',
      color: 'var(--text-muted)',
    },
  };

  return (
    mapping[iconColor] ?? {
      bg: 'color-mix(in srgb, var(--accent) 10%, transparent)',
      color: 'var(--accent)',
    }
  );
}

export const StatCard = ({
  icon: Icon,
  iconColor,
  value,
  label,
  sublabel,
  trend,
  visualization,
  onClick,
  compact,
}: StatCardProps) => {
  const { bg, color } = resolveIconColor(iconColor);

  return (
    <div
      style={{
        borderRadius: 8,
        border: '1px solid var(--border)',
        backgroundColor: 'var(--bg-card)',
        boxShadow: 'var(--shadow-sm)',
        transition: 'transform 0.2s ease, border-color 0.2s ease',
        cursor: onClick ? 'pointer' : undefined,
      }}
      onMouseEnter={(e) => {
        (e.currentTarget as HTMLDivElement).style.transform = 'translateY(-1px)';
        (e.currentTarget as HTMLDivElement).style.borderColor = 'var(--border-hover)';
      }}
      onMouseLeave={(e) => {
        (e.currentTarget as HTMLDivElement).style.transform = '';
        (e.currentTarget as HTMLDivElement).style.borderColor = 'var(--border)';
      }}
      onClick={onClick}
    >
      {compact ? (
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '8px 12px' }}>
          <div
            style={{
              display: 'flex',
              width: 20,
              height: 20,
              flexShrink: 0,
              alignItems: 'center',
              justifyContent: 'center',
              borderRadius: 4,
              background: bg,
              color,
            }}
          >
            <Icon style={{ width: 10, height: 10 }} />
          </div>
          <div style={{ minWidth: 0 }}>
            <p
              style={{
                fontSize: 14,
                fontWeight: 700,
                lineHeight: 1,
                margin: 0,
                color: 'var(--text-emphasis)',
                fontFamily: 'var(--font-mono)',
              }}
            >
              {value}
            </p>
            <p
              style={{
                marginTop: 2,
                fontSize: 10,
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
                color: 'var(--text-muted)',
                fontFamily: 'var(--font-sans)',
              }}
            >
              {label}
            </p>
          </div>
          {visualization && (
            <div style={{ marginLeft: 'auto', flexShrink: 0 }}>{visualization}</div>
          )}
        </div>
      ) : (
        <div
          style={{
            display: 'flex',
            alignItems: 'flex-start',
            justifyContent: 'space-between',
            padding: 14,
          }}
        >
          <div>
            <div
              style={{
                marginBottom: 8,
                display: 'flex',
                width: 30,
                height: 30,
                alignItems: 'center',
                justifyContent: 'center',
                borderRadius: 8,
                background: bg,
                color,
              }}
            >
              <Icon style={{ width: 16, height: 16 }} />
            </div>
            <p
              style={{
                fontSize: 24,
                fontWeight: 700,
                lineHeight: 1,
                margin: 0,
                color: 'var(--text-emphasis)',
                fontFamily: 'var(--font-mono)',
              }}
            >
              {value}
            </p>
            <p
              style={{
                marginTop: 4,
                fontSize: 11,
                color: 'var(--text-muted)',
                fontFamily: 'var(--font-sans)',
                margin: '4px 0 0',
              }}
            >
              {label}
            </p>
            {sublabel && (
              <p
                style={{
                  fontSize: 10,
                  color: 'var(--text-muted)',
                  fontFamily: 'var(--font-sans)',
                  margin: 0,
                }}
              >
                {sublabel}
              </p>
            )}
            {trend && (
              <p
                style={{
                  marginTop: 4,
                  display: 'flex',
                  alignItems: 'center',
                  gap: 4,
                  fontSize: 10.5,
                  fontFamily: 'var(--font-sans)',
                  color:
                    trend.direction === 'up' ? 'var(--signal-healthy)' : 'var(--signal-critical)',
                  margin: '4px 0 0',
                }}
              >
                {trend.direction === 'up' ? '↑' : '↓'} {trend.percentage}
                {trend.context && (
                  <span style={{ color: 'var(--text-muted)' }}>{trend.context}</span>
                )}
              </p>
            )}
          </div>
          {visualization && <div style={{ flexShrink: 0 }}>{visualization}</div>}
        </div>
      )}
    </div>
  );
};
