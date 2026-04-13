import { Button } from '@patchiq/ui';
import { X } from 'lucide-react';

interface BulkAction {
  label: string;
  onClick: () => void;
  variant?: 'default' | 'destructive';
}

interface BulkActionBarProps {
  selectedCount: number;
  actions: BulkAction[];
  onClearSelection: () => void;
  className?: string;
}

export const BulkActionBar = ({ selectedCount, actions, onClearSelection }: BulkActionBarProps) => {
  if (selectedCount === 0) return null;

  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 12,
        borderRadius: 8,
        border: '1px solid color-mix(in srgb, var(--accent) 25%, transparent)',
        background: 'color-mix(in srgb, var(--accent) 6%, transparent)',
        padding: '8px 16px',
      }}
    >
      <span
        style={{
          fontFamily: 'var(--font-mono)',
          fontSize: 12,
          fontWeight: 500,
          color: 'var(--text-primary)',
        }}
      >
        {selectedCount} selected
      </span>
      <div style={{ display: 'flex', gap: 8 }}>
        {actions.map((action) => (
          <Button
            key={action.label}
            size="sm"
            variant={action.variant === 'destructive' ? 'destructive' : 'default'}
            onClick={action.onClick}
          >
            {action.label}
          </Button>
        ))}
      </div>
      <button
        onClick={onClearSelection}
        aria-label="Clear selection"
        style={{
          marginLeft: 'auto',
          borderRadius: 4,
          padding: 4,
          background: 'none',
          border: 'none',
          cursor: 'pointer',
          color: 'var(--text-muted)',
          display: 'flex',
          alignItems: 'center',
          transition: 'color 0.1s ease',
        }}
        onMouseEnter={(e) =>
          ((e.currentTarget as HTMLButtonElement).style.color = 'var(--text-primary)')
        }
        onMouseLeave={(e) =>
          ((e.currentTarget as HTMLButtonElement).style.color = 'var(--text-muted)')
        }
      >
        <X style={{ width: 14, height: 14 }} />
      </button>
    </div>
  );
};
