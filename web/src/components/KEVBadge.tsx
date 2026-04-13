import { Flag } from 'lucide-react';

interface KEVBadgeProps {
  dueDate: string | null;
  className?: string;
}

export const KEVBadge = ({ dueDate }: KEVBadgeProps) => {
  if (!dueDate) {
    return (
      <span
        style={{
          fontFamily: 'var(--font-mono)',
          fontSize: 11,
          color: 'var(--text-faint)',
        }}
      >
        —
      </span>
    );
  }
  return (
    <span
      title={`CISA KEV — remediate by ${dueDate}`}
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 4,
        borderRadius: 9999,
        padding: '2px 8px',
        fontSize: 11,
        fontWeight: 600,
        fontFamily: 'var(--font-mono)',
        background: 'color-mix(in srgb, var(--signal-warning) 12%, transparent)',
        color: 'var(--signal-warning)',
        border: '1px solid color-mix(in srgb, var(--signal-warning) 30%, transparent)',
        whiteSpace: 'nowrap',
      }}
    >
      <span
        style={{
          width: 6,
          height: 6,
          borderRadius: '50%',
          background: 'var(--signal-warning)',
          display: 'inline-block',
          flexShrink: 0,
        }}
      />
      <Flag style={{ width: 11, height: 11 }} />
      Yes
    </span>
  );
};
