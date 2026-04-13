import type { ReactNode } from 'react';
import { Search } from 'lucide-react';

/* ── Filter Pill ─────────────────────────────────────────── */

type PillVariant = 'default' | 'success' | 'warning' | 'danger' | 'muted';

const pillActiveColors: Record<PillVariant, { bg: string; border: string; color: string }> = {
  default: {
    bg: 'color-mix(in srgb, var(--accent) 10%, transparent)',
    border: 'color-mix(in srgb, var(--accent) 30%, transparent)',
    color: 'var(--accent)',
  },
  success: {
    bg: 'color-mix(in srgb, var(--accent) 10%, transparent)',
    border: 'color-mix(in srgb, var(--accent) 30%, transparent)',
    color: 'var(--accent)',
  },
  warning: {
    bg: 'color-mix(in srgb, var(--signal-warning) 10%, transparent)',
    border: 'color-mix(in srgb, var(--signal-warning) 30%, transparent)',
    color: 'var(--signal-warning)',
  },
  danger: {
    bg: 'color-mix(in srgb, var(--signal-critical) 10%, transparent)',
    border: 'color-mix(in srgb, var(--signal-critical) 30%, transparent)',
    color: 'var(--signal-critical)',
  },
  muted: {
    bg: 'color-mix(in srgb, var(--text-muted) 10%, transparent)',
    border: 'color-mix(in srgb, var(--text-muted) 30%, transparent)',
    color: 'var(--text-muted)',
  },
};

interface FilterPillProps {
  label: string;
  count?: number;
  active?: boolean;
  variant?: PillVariant;
  onClick?: () => void;
}

export const FilterPill = ({
  label,
  count,
  active = false,
  variant = 'default',
  onClick,
}: FilterPillProps) => {
  const activeColor = pillActiveColors[variant];
  return (
    <button
      type="button"
      onClick={onClick}
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 4,
        borderRadius: 9999,
        border: '1px solid',
        padding: '4px 12px',
        fontSize: 11.5,
        fontWeight: 500,
        fontFamily: 'var(--font-sans)',
        cursor: 'pointer',
        transition: 'all 0.1s ease',
        background: active ? activeColor.bg : 'transparent',
        borderColor: active ? activeColor.border : 'var(--border)',
        color: active ? activeColor.color : 'var(--text-secondary)',
      }}
    >
      {label}
      {count != null && (
        <span
          style={{
            borderRadius: 9999,
            background: 'color-mix(in srgb, var(--text-on-color, #fff) 8%, transparent)',
            padding: '0 5px',
            fontSize: 10,
            fontFamily: 'var(--font-mono)',
            color: active ? activeColor.color : 'var(--text-muted)',
          }}
        >
          {count}
        </span>
      )}
    </button>
  );
};

/* ── Filter Separator ────────────────────────────────────── */

export const FilterSeparator = () => (
  <div
    style={{ width: 1, height: 24, background: 'var(--border)', flexShrink: 0, margin: '0 4px' }}
  />
);

/* ── Filter Search ───────────────────────────────────────── */

interface FilterSearchProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
}

export const FilterSearch = ({ value, onChange, placeholder = 'Search...' }: FilterSearchProps) => (
  <div style={{ position: 'relative', minWidth: 220 }}>
    <Search
      style={{
        position: 'absolute',
        left: 10,
        top: '50%',
        transform: 'translateY(-50%)',
        width: 13,
        height: 13,
        color: 'var(--text-muted)',
        pointerEvents: 'none',
      }}
    />
    <input
      value={value}
      onChange={(e) => onChange(e.target.value)}
      placeholder={placeholder}
      style={{
        display: 'block',
        width: '100%',
        height: 32,
        paddingLeft: 30,
        paddingRight: 10,
        fontFamily: 'var(--font-sans)',
        fontSize: 12,
        color: 'var(--text-primary)',
        background: 'var(--bg-card)',
        border: '1px solid var(--border)',
        borderRadius: 6,
        outline: 'none',
        boxSizing: 'border-box',
      }}
    />
  </div>
);

/* ── Filter Bar Container ────────────────────────────────── */

interface FilterBarProps {
  children: ReactNode;
}

export const FilterBar = ({ children }: FilterBarProps) => (
  <div
    style={{
      marginBottom: 16,
      borderRadius: 8,
      border: '1px solid var(--border)',
      background: 'var(--bg-card)',
      padding: '10px 16px',
    }}
  >
    <div
      style={{
        display: 'flex',
        flexWrap: 'wrap',
        alignItems: 'center',
        gap: 6,
      }}
    >
      {children}
    </div>
  </div>
);
