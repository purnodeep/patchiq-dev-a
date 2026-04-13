import { Button } from '@patchiq/ui';
import type { LucideIcon } from 'lucide-react';

interface EmptyStateProps {
  icon?: LucideIcon;
  title: string;
  description?: string;
  action?: {
    label: string;
    onClick: () => void;
  };
  className?: string;
}

export const EmptyState = ({ icon: Icon, title, description, action }: EmptyStateProps) => (
  <div
    style={{
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      justifyContent: 'center',
      padding: '40px 20px',
      textAlign: 'center',
    }}
  >
    {Icon && (
      <div
        style={{
          marginBottom: 12,
          borderRadius: 9999,
          background: 'var(--bg-inset)',
          padding: 12,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
        }}
      >
        <Icon style={{ width: 24, height: 24, color: 'var(--text-muted)' }} />
      </div>
    )}
    <h3
      style={{
        fontFamily: 'var(--font-sans)',
        fontSize: 13,
        fontWeight: 500,
        color: 'var(--text-primary)',
        margin: 0,
      }}
    >
      {title}
    </h3>
    {description && (
      <p
        style={{
          marginTop: 4,
          maxWidth: 360,
          fontFamily: 'var(--font-sans)',
          fontSize: 12,
          color: 'var(--text-muted)',
          lineHeight: 1.5,
        }}
      >
        {description}
      </p>
    )}
    {action && (
      <div style={{ marginTop: 16 }}>
        <Button size="sm" onClick={action.onClick}>
          {action.label}
        </Button>
      </div>
    )}
  </div>
);
