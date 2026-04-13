import { useCallback } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { X, Check, Plus, GripVertical } from 'lucide-react';
import { WIDGET_REGISTRY, WIDGET_CATEGORIES } from './registry';
import type { WidgetId } from './types';

interface WidgetDrawerProps {
  isOpen: boolean;
  onClose: () => void;
  activeWidgets: WidgetId[];
  onAddWidget: (id: WidgetId) => void;
  onRemoveWidget: (id: WidgetId) => void;
}

export function WidgetDrawer({
  isOpen,
  onClose,
  activeWidgets,
  onAddWidget,
  onRemoveWidget,
}: WidgetDrawerProps) {
  const widgetsByCategory = WIDGET_CATEGORIES.map((cat) => ({
    ...cat,
    widgets: [...WIDGET_REGISTRY.values()].filter((w) => w.category === cat.key),
  }));

  const handleDragStart = useCallback(
    (e: React.DragEvent<HTMLButtonElement>, widgetId: string) => {
      e.dataTransfer.setData('text/plain', widgetId);
      e.dataTransfer.effectAllowed = 'copy';
      (e.currentTarget as HTMLElement).style.opacity = '0.5';
    },
    [],
  );

  const handleDragEnd = useCallback((e: React.DragEvent<HTMLButtonElement>) => {
    (e.currentTarget as HTMLElement).style.opacity = '1';
  }, []);

  return (
    <AnimatePresence>
      {isOpen && (
        <>
          {/* Backdrop — pointer-events:none so drag can pass through to the grid */}
          <motion.div
            className="fixed inset-0 z-40"
            style={{
              backgroundColor: 'rgba(0, 0, 0, 0.2)',
              backdropFilter: 'blur(2px)',
              pointerEvents: 'none',
            }}
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
          />

          {/* Drawer */}
          <motion.aside
            className="fixed top-0 right-0 z-50 flex h-full w-[340px] flex-col overflow-hidden"
            style={{
              backgroundColor: 'var(--bg-page)',
              borderLeft: '1px solid var(--border)',
              boxShadow: '-8px 0 24px rgba(0,0,0,0.3)',
            }}
            initial={{ x: '100%' }}
            animate={{ x: 0 }}
            exit={{ x: '100%' }}
            transition={{ type: 'spring', damping: 30, stiffness: 300 }}
          >
            {/* Header */}
            <div
              className="flex items-center justify-between px-5 py-4"
              style={{ borderBottom: '1px solid var(--border)' }}
            >
              <h2 className="text-base font-semibold" style={{ color: 'var(--text-emphasis)' }}>
                Add Widgets
              </h2>
              <button
                onClick={onClose}
                className="flex h-8 w-8 items-center justify-center rounded-md transition-colors hover:opacity-80"
                style={{
                  backgroundColor: 'var(--bg-card)',
                  border: '1px solid var(--border)',
                  color: 'var(--text-muted)',
                }}
              >
                <X className="h-4 w-4" />
              </button>
            </div>

            {/* Drag instruction */}
            <div
              style={{
                padding: '8px 20px',
                fontSize: 10,
                color: 'var(--text-faint)',
                borderBottom: '1px solid var(--border)',
              }}
            >
              Drag widgets onto the dashboard or click to add
            </div>

            {/* Widget list */}
            <div className="flex-1 overflow-y-auto px-5 py-4">
              {widgetsByCategory.map((cat) => (
                <div key={cat.key} className="mb-6 last:mb-0">
                  <h3
                    className="mb-3 text-xs font-medium uppercase tracking-wider"
                    style={{ color: 'var(--text-faint)' }}
                  >
                    {cat.label}
                  </h3>
                  <div className="flex flex-col gap-2">
                    {cat.widgets.map((widget) => {
                      const isActive = activeWidgets.includes(widget.id);
                      const Icon = widget.icon;

                      return (
                        <button
                          key={widget.id}
                          draggable={!isActive}
                          onDragStart={(e) => handleDragStart(e, widget.id)}
                          onDragEnd={handleDragEnd}
                          className="flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-left transition-colors"
                          style={{
                            backgroundColor: 'var(--bg-card)',
                            border: `1px solid ${isActive ? 'var(--accent)' : 'var(--border)'}`,
                            cursor: isActive ? 'pointer' : 'grab',
                          }}
                          onMouseEnter={(e) => {
                            if (!isActive) {
                              (e.currentTarget as HTMLElement).style.borderColor =
                                'var(--border-hover)';
                            }
                          }}
                          onMouseLeave={(e) => {
                            (e.currentTarget as HTMLElement).style.borderColor = isActive
                              ? 'var(--accent)'
                              : 'var(--border)';
                          }}
                          onClick={() => {
                            if (isActive) {
                              onRemoveWidget(widget.id);
                            } else {
                              onAddWidget(widget.id);
                            }
                          }}
                        >
                          {/* Drag grip for non-active widgets */}
                          {!isActive && (
                            <div style={{ color: 'var(--text-faint)', flexShrink: 0 }}>
                              <GripVertical className="h-3.5 w-3.5" />
                            </div>
                          )}

                          {/* Icon */}
                          <div
                            className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full"
                            style={{
                              backgroundColor: isActive ? 'var(--accent-subtle)' : 'var(--bg-page)',
                              color: isActive ? 'var(--accent)' : 'var(--text-faint)',
                            }}
                          >
                            <Icon className="h-4 w-4" />
                          </div>

                          {/* Label + description */}
                          <div className="min-w-0 flex-1">
                            <div
                              className="truncate text-sm font-medium"
                              style={{ color: 'var(--text-emphasis)' }}
                            >
                              {widget.label}
                            </div>
                            <div
                              className="truncate text-xs"
                              style={{ color: 'var(--text-secondary)' }}
                            >
                              {widget.description}
                            </div>
                          </div>

                          {/* Status icon */}
                          <div
                            className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full"
                            style={{
                              backgroundColor: isActive ? 'var(--accent)' : 'var(--bg-page)',
                              color: isActive ? 'white' : 'var(--text-faint)',
                              border: isActive ? 'none' : '1px solid var(--border)',
                            }}
                          >
                            {isActive ? (
                              <Check className="h-3.5 w-3.5" />
                            ) : (
                              <Plus className="h-3.5 w-3.5" />
                            )}
                          </div>
                        </button>
                      );
                    })}
                  </div>
                </div>
              ))}
            </div>
          </motion.aside>
        </>
      )}
    </AnimatePresence>
  );
}
