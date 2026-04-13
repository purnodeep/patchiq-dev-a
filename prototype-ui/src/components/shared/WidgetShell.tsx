// prototype-ui/src/components/shared/WidgetShell.tsx
import React from 'react';
import { GlassCard } from './GlassCard';

interface WidgetShellProps extends React.HTMLAttributes<HTMLDivElement> {
  title?: string;
  isEditMode?: boolean;
}

/**
 * Wrapper for every dashboard widget in the react-grid-layout grid.
 *
 * Must be a forwardRef component. rgl injects `style`, `className`, and `ref`
 * into its direct children — these must be forwarded to the root DOM node so
 * rgl can correctly position and size each widget.
 *
 * GlassCard is inside the root div (not the root itself) for this reason.
 */
export const WidgetShell = React.forwardRef<HTMLDivElement, WidgetShellProps>(
  ({ style, className, title, isEditMode, children, ...rest }, ref) => {
    return (
      <div
        ref={ref}
        style={style}
        className={[className, isEditMode ? 'widget-shell-edit' : ''].filter(Boolean).join(' ')}
        {...rest}
      >
        <GlassCard
          hover={false}
          style={{
            height: '100%',
            display: 'flex',
            flexDirection: 'column',
            overflow: 'hidden',
          }}
        >
          {isEditMode && (
            <div
              className="drag-handle"
              style={{
                padding: '6px 12px 4px',
                cursor: 'grab',
                display: 'flex',
                alignItems: 'center',
                gap: 6,
                borderBottom: '1px solid var(--color-separator)',
                flexShrink: 0,
              }}
            >
              <svg width={14} height={14} viewBox="0 0 14 14" fill="var(--color-subtle)">
                {[2, 6, 10].flatMap((x) =>
                  [2, 6, 10].map((y) => <circle key={`${x}-${y}`} cx={x} cy={y} r={1.2} />),
                )}
              </svg>
              {title && (
                <span style={{ fontSize: 11, color: 'var(--color-subtle)', fontWeight: 500 }}>
                  {title}
                </span>
              )}
            </div>
          )}
          <div style={{ flex: 1, overflow: 'hidden', minHeight: 0, height: '100%' }}>
            {children}
          </div>
        </GlassCard>
      </div>
    );
  },
);

WidgetShell.displayName = 'WidgetShell';
