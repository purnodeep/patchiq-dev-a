import { createElement, type ComponentType, type ReactNode } from 'react';
import { Button } from '../components/ui/button';
import { cn } from '../lib/utils';

interface EmptyStateAction {
  label: string;
  onClick: () => void;
}

interface EmptyStateProps {
  /** Accept either a pre-rendered ReactNode or a Lucide icon component constructor. */
  icon?: ReactNode | ComponentType<{ style?: React.CSSProperties; className?: string }>;
  title: string;
  description?: string;
  action?: EmptyStateAction;
  className?: string;
}

function EmptyState({ icon, title, description, action, className }: EmptyStateProps) {
  // Support both pre-rendered ReactNode and component constructors (e.g. Lucide icons).
  // Lucide icons may be forwardRef objects (typeof === 'object') or plain functions.
  // Use createElement so both shapes work without needing to inspect internals.
  let renderedIcon: ReactNode = null;
  if (icon) {
    const isComponent =
      typeof icon === 'function' ||
      (typeof icon === 'object' && icon !== null && !('type' in (icon as object)));
    if (isComponent) {
      renderedIcon = createElement(
        icon as ComponentType<{ style?: React.CSSProperties; className?: string }>,
        { style: { width: 24, height: 24, color: 'var(--text-muted)' } },
      );
    } else {
      renderedIcon = icon as ReactNode;
    }
  }

  return (
    <div
      className={cn('flex flex-col items-center justify-center py-10 text-center', className)}
      style={{ padding: '40px 20px' }}
    >
      {renderedIcon && (
        <div
          style={{
            marginBottom: 12,
            borderRadius: 9999,
            background: 'var(--bg-inset)',
            padding: 12,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            color: 'var(--text-muted)',
          }}
        >
          {renderedIcon}
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
          <Button type="button" size="sm" onClick={action.onClick}>
            {action.label}
          </Button>
        </div>
      )}
    </div>
  );
}

export { EmptyState };
export type { EmptyStateProps, EmptyStateAction };
