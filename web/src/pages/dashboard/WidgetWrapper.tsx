import { Suspense, Component, useState, type ReactNode } from 'react';
import { AnimatePresence, motion } from 'framer-motion';
import { X, GripVertical, Settings } from 'lucide-react';
import { SkeletonCard } from '@patchiq/ui';
import type { WidgetRegistryEntry, WidgetConfig } from './types';
import { WidgetConfigPopover } from './WidgetConfigPopover';

interface ErrorBoundaryProps {
  widgetLabel: string;
  children: ReactNode;
}

interface ErrorBoundaryState {
  hasError: boolean;
}

class WidgetErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  state: ErrorBoundaryState = { hasError: false };

  static getDerivedStateFromError(): ErrorBoundaryState {
    return { hasError: true };
  }

  render() {
    if (this.state.hasError) {
      return (
        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            height: '100%',
            gap: 8,
            color: 'var(--text-muted)',
            fontSize: 12,
          }}
        >
          <span style={{ fontSize: 20 }}>!</span>
          <span>{this.props.widgetLabel} failed to load</span>
          <button
            onClick={() => this.setState({ hasError: false })}
            style={{
              padding: '4px 12px',
              borderRadius: 4,
              border: '1px solid var(--border)',
              background: 'var(--bg-card)',
              color: 'var(--text-secondary)',
              cursor: 'pointer',
              fontSize: 11,
            }}
          >
            Retry
          </button>
        </div>
      );
    }
    return this.props.children;
  }
}

interface WidgetWrapperProps {
  entry: WidgetRegistryEntry;
  isEditing: boolean;
  onRemove: () => void;
  config: WidgetConfig;
  onConfigChange: (config: WidgetConfig) => void;
}

export function WidgetWrapper({ entry, isEditing, onRemove, config, onConfigChange }: WidgetWrapperProps) {
  const WidgetComponent = entry.component;
  const [configOpen, setConfigOpen] = useState(false);
  const hasConfig = !!entry.configSchema && Object.keys(entry.configSchema).length > 0;

  return (
    <div
      style={{
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        borderRadius: 8,
        overflow: 'hidden',
        position: 'relative',
      }}
    >
      <AnimatePresence>
        {isEditing && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.15 }}
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              padding: '0 4px',
              background: 'var(--bg-surface, var(--bg-card))',
              position: 'absolute',
              top: 0,
              left: 0,
              right: 0,
              zIndex: 10,
              height: 24,
              overflow: 'hidden',
              borderRadius: '8px 8px 0 0',
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: 2 }}>
              <div
                className="widget-drag-handle"
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  cursor: 'grab',
                  color: 'var(--text-muted)',
                  userSelect: 'none',
                }}
              >
                <GripVertical size={12} />
              </div>
              {hasConfig && (
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    setConfigOpen((v) => !v);
                  }}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    width: 18,
                    height: 18,
                    borderRadius: 4,
                    border: 'none',
                    background: 'transparent',
                    color: 'var(--text-muted)',
                    cursor: 'pointer',
                    transition: 'color 150ms',
                  }}
                  onMouseEnter={(e) => {
                    e.currentTarget.style.color = 'var(--text-primary)';
                  }}
                  onMouseLeave={(e) => {
                    e.currentTarget.style.color = 'var(--text-muted)';
                  }}
                >
                  <Settings size={11} />
                </button>
              )}
            </div>
            <button
              onClick={(e) => {
                e.stopPropagation();
                onRemove();
              }}
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                width: 18,
                height: 18,
                borderRadius: 4,
                border: 'none',
                background: 'transparent',
                color: 'var(--text-muted)',
                cursor: 'pointer',
                transition: 'color 150ms, background 150ms',
              }}
              onMouseEnter={(e) => {
                e.currentTarget.style.background = 'var(--signal-critical)';
                e.currentTarget.style.color = '#fff';
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.background = 'transparent';
                e.currentTarget.style.color = 'var(--text-muted)';
              }}
            >
              <X size={12} />
            </button>
          </motion.div>
        )}
      </AnimatePresence>

      {/* Hover gear icon when not editing */}
      {!isEditing && hasConfig && (
        <div
          className="widget-config-trigger"
          style={{
            position: 'absolute',
            top: 4,
            right: 4,
            zIndex: 10,
            opacity: 0,
            transition: 'opacity 150ms',
          }}
        >
          <button
            onClick={(e) => {
              e.stopPropagation();
              setConfigOpen((v) => !v);
            }}
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              width: 22,
              height: 22,
              borderRadius: 4,
              border: '1px solid var(--border)',
              background: 'var(--bg-card)',
              color: 'var(--text-muted)',
              cursor: 'pointer',
              transition: 'color 150ms, background 150ms',
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.color = 'var(--text-primary)';
              e.currentTarget.style.background = 'var(--bg-surface, var(--bg-card))';
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.color = 'var(--text-muted)';
              e.currentTarget.style.background = 'var(--bg-card)';
            }}
          >
            <Settings size={12} />
          </button>
        </div>
      )}

      {/* Config popover */}
      {configOpen && hasConfig && entry.configSchema && (
        <div
          style={{ position: 'absolute', top: 0, right: 0, zIndex: 20 }}
        >
          <WidgetConfigPopover
            configSchema={entry.configSchema}
            config={config}
            onChange={onConfigChange}
            onClose={() => setConfigOpen(false)}
          />
        </div>
      )}

      <div
        className="widget-content-fill"
        style={{
          flex: 1,
          overflow: 'auto',
          minHeight: 0,
          pointerEvents: isEditing ? 'none' : 'auto',
          display: 'flex',
          flexDirection: 'column',
        }}
      >
        <WidgetErrorBoundary widgetLabel={entry.label}>
          <Suspense fallback={<SkeletonCard lines={3} />}>
            <WidgetComponent config={config} />
          </Suspense>
        </WidgetErrorBoundary>
      </div>
    </div>
  );
}
