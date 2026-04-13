import { useState, useCallback, useEffect, useRef } from 'react';
import type { WidgetId } from './types';
import { ResponsiveGridLayout, useContainerWidth, type Layout, type LayoutItem } from 'react-grid-layout';
import { Link } from 'react-router';
import { SkeletonCard, ErrorState } from '@patchiq/ui';
import { LayoutGrid, Plus, RotateCcw, ChevronDown } from 'lucide-react';
import { useDashboardSummary } from '@/api/hooks/useDashboard';
import { DashboardDataProvider } from './DashboardContext';
import { WIDGET_REGISTRY } from './registry';
import { useDashboardLayout } from './hooks/useDashboardLayout';
import { WidgetWrapper } from './WidgetWrapper';
import { WidgetDrawer } from './WidgetDrawer';
import { DASHBOARD_PRESETS } from './presets';
import type { PresetId } from './presets';
import './dashboard-grid.css';

function LoadingSkeleton() {
  return (
    <>
      <div style={{ marginBottom: 20 }}>
        <SkeletonCard lines={1} />
      </div>
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(4, 1fr)',
          gap: 12,
          marginBottom: 12,
        }}
      >
        {Array.from({ length: 4 }).map((_, i) => (
          <SkeletonCard key={i} lines={3} />
        ))}
      </div>
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(4, 1fr)',
          gap: 12,
          marginBottom: 12,
        }}
      >
        {Array.from({ length: 4 }).map((_, i) => (
          <SkeletonCard key={`r2-${i}`} lines={3} />
        ))}
      </div>
      <div style={{ display: 'grid', gridTemplateColumns: '55fr 45fr', gap: 12, marginBottom: 12 }}>
        <SkeletonCard lines={5} className="min-h-[200px]" />
        <SkeletonCard lines={5} className="min-h-[200px]" />
      </div>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 12 }}>
        {Array.from({ length: 3 }).map((_, i) => (
          <SkeletonCard key={i} lines={5} />
        ))}
      </div>
    </>
  );
}

export default function DashboardPage() {
  const { data, isLoading, error, refetch } = useDashboardSummary();
  const {
    layouts,
    activeWidgets,
    onLayoutChange,
    addWidget,
    addWidgetAt,
    removeWidget,
    resetLayout,
    isEditing,
    setIsEditing,
    getWidgetConfig,
    updateWidgetConfig,
    presetId,
    applyPreset,
  } = useDashboardLayout();

  const [drawerOpen, setDrawerOpen] = useState(false);
  const [presetDropdownOpen, setPresetDropdownOpen] = useState(false);
  const presetDropdownRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!presetDropdownOpen) return;
    const handler = (e: MouseEvent) => {
      if (presetDropdownRef.current && !presetDropdownRef.current.contains(e.target as Node)) {
        setPresetDropdownOpen(false);
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [presetDropdownOpen]);

  const currentPresetLabel = presetId === 'custom'
    ? 'Custom'
    : DASHBOARD_PRESETS.find((p) => p.id === presetId)?.label ?? 'Custom';

  const handlePresetSelect = (id: PresetId) => {
    const preset = DASHBOARD_PRESETS.find((p) => p.id === id);
    if (preset) {
      applyPreset(id, preset.widgets);
    }
    setPresetDropdownOpen(false);
  };
  const { width, containerRef, mounted } = useContainerWidth();

  const handleDrop = useCallback(
    (_layout: Layout, item: LayoutItem | undefined, e: Event) => {
      if (!(e instanceof DragEvent) || !e.dataTransfer) return;
      const widgetId = e.dataTransfer.getData('text/plain') as WidgetId;
      if (!widgetId || !WIDGET_REGISTRY.has(widgetId)) return;
      if (item && typeof item.x === 'number' && typeof item.y === 'number') {
        addWidgetAt(widgetId, item.x, item.y);
      } else {
        addWidget(widgetId);
      }
    },
    [addWidget, addWidgetAt],
  );

  const content = (() => {
    if (isLoading) return <LoadingSkeleton />;

    if (error) {
      return <ErrorState message="Failed to load dashboard data" onRetry={() => refetch()} />;
    }

    if (!data) return null;

    return (
      <DashboardDataProvider data={data}>
        {/* Onboarding banner */}
        {data.total_endpoints === 0 && (
          <div
            style={{
              background: 'var(--accent-subtle)',
              border: '1px solid var(--accent-border)',
              borderRadius: 8,
              padding: '20px 24px',
              marginBottom: 20,
            }}
          >
            <div
              style={{
                fontSize: 15,
                fontWeight: 600,
                color: 'var(--text-emphasis)',
                marginBottom: 6,
              }}
            >
              Welcome to PatchIQ
            </div>
            <div
              style={{
                fontSize: 13,
                color: 'var(--text-secondary)',
                marginBottom: 14,
              }}
            >
              Get started by enrolling your first agent. Your dashboard will populate with real
              metrics once agents are reporting.
            </div>
            <Link
              to="/agent-downloads"
              style={{
                display: 'inline-flex',
                alignItems: 'center',
                gap: 6,
                padding: '7px 16px',
                borderRadius: 6,
                fontSize: 12,
                fontWeight: 600,
                color: 'var(--btn-accent-text, #000)',
                background: 'var(--accent)',
                border: 'none',
                textDecoration: 'none',
                cursor: 'pointer',
              }}
            >
              Download Agent →
            </Link>
          </div>
        )}

        {/* Toolbar */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'flex-end',
            minHeight: 40,
            marginBottom: 32,
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
            {/* Preset dropdown */}
            <div ref={presetDropdownRef} style={{ position: 'relative' }}>
              <button
                onClick={() => setPresetDropdownOpen((v) => !v)}
                style={{
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: 5,
                  padding: '5px 10px',
                  borderRadius: 6,
                  border: '1px solid var(--border)',
                  background: 'var(--bg-card)',
                  color: 'var(--text-secondary)',
                  fontSize: 11,
                  fontWeight: 500,
                  cursor: 'pointer',
                  transition: 'border-color 150ms',
                }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.borderColor = 'var(--border-hover)';
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.borderColor = 'var(--border)';
                }}
              >
                {currentPresetLabel}
                <ChevronDown size={11} />
              </button>
              {presetDropdownOpen && (
                <div
                  style={{
                    position: 'absolute',
                    top: '100%',
                    right: 0,
                    marginTop: 4,
                    width: 200,
                    background: 'var(--bg-card)',
                    border: '1px solid var(--border)',
                    borderRadius: 8,
                    boxShadow: '0 8px 24px rgba(0,0,0,0.3)',
                    zIndex: 100,
                    padding: 4,
                  }}
                >
                  {DASHBOARD_PRESETS.map((preset) => (
                    <button
                      key={preset.id}
                      onClick={() => handlePresetSelect(preset.id)}
                      style={{
                        display: 'block',
                        width: '100%',
                        textAlign: 'left',
                        padding: '8px 10px',
                        borderRadius: 6,
                        border: 'none',
                        background: presetId === preset.id ? 'var(--accent-subtle, rgba(99,102,241,0.1))' : 'transparent',
                        color: presetId === preset.id ? 'var(--accent)' : 'var(--text-secondary)',
                        fontSize: 11,
                        fontWeight: 500,
                        cursor: 'pointer',
                        transition: 'background 150ms',
                      }}
                      onMouseEnter={(e) => {
                        if (presetId !== preset.id) {
                          e.currentTarget.style.background = 'var(--bg-surface, rgba(255,255,255,0.04))';
                        }
                      }}
                      onMouseLeave={(e) => {
                        if (presetId !== preset.id) {
                          e.currentTarget.style.background = 'transparent';
                        }
                      }}
                    >
                      <div>{preset.label}</div>
                      <div style={{ fontSize: 10, color: 'var(--text-muted)', marginTop: 1 }}>
                        {preset.description}
                      </div>
                    </button>
                  ))}
                </div>
              )}
            </div>

            {isEditing && (
              <>
                <ToolbarButton onClick={resetLayout} icon={<RotateCcw size={12} />} label="Reset" />
                <ToolbarButton
                  onClick={() => setDrawerOpen(true)}
                  icon={<Plus size={12} />}
                  label="Add Widget"
                />
              </>
            )}
            <button
              onClick={() => setIsEditing(!isEditing)}
              style={{
                display: 'inline-flex',
                alignItems: 'center',
                gap: 5,
                padding: '5px 12px',
                borderRadius: 6,
                border: `1px solid ${isEditing ? 'var(--accent)' : 'var(--border)'}`,
                background: isEditing ? 'var(--accent)' : 'var(--bg-card)',
                color: isEditing ? '#fff' : 'var(--text-secondary)',
                fontSize: 11,
                fontWeight: 500,
                cursor: 'pointer',
                transition: 'all 150ms',
              }}
            >
              <LayoutGrid size={12} />
              {isEditing ? 'Done' : 'Customize'}
            </button>
          </div>
        </div>

        {/* Grid */}
        {mounted && (
          <ResponsiveGridLayout
            className={`layout ${isEditing ? 'dashboard-grid--editing' : ''}`}
            width={width}
            layouts={layouts}
            breakpoints={{ lg: 1200, md: 900, sm: 600 }}
            cols={{ lg: 12, md: 8, sm: 4 }}
            rowHeight={50}
            margin={[12, 8] as const}
            containerPadding={[0, 0] as const}
            onLayoutChange={onLayoutChange}
            dragConfig={{ enabled: isEditing, handle: '.widget-drag-handle' }}
            resizeConfig={{ enabled: isEditing, handles: ['se'] }}
            dropConfig={{ enabled: isEditing, defaultItem: { w: 6, h: 4 } }}
            onDrop={handleDrop}
          >
            {activeWidgets.map((id) => {
              const entry = WIDGET_REGISTRY.get(id);
              if (!entry) return null;
              return (
                <div key={id}>
                  <WidgetWrapper
                      entry={entry}
                      isEditing={isEditing}
                      onRemove={() => removeWidget(id)}
                      config={getWidgetConfig(id)}
                      onConfigChange={(cfg) => updateWidgetConfig(id, cfg)}
                    />
                </div>
              );
            })}
          </ResponsiveGridLayout>
        )}

        {/* Widget drawer */}
        <WidgetDrawer
          isOpen={drawerOpen}
          onClose={() => setDrawerOpen(false)}
          activeWidgets={activeWidgets}
          onAddWidget={addWidget}
          onRemoveWidget={removeWidget}
        />
      </DashboardDataProvider>
    );
  })();

  return (
    <div ref={containerRef} style={{ background: 'var(--bg-page)', padding: 24 }}>
      {content}
    </div>
  );
}

/* Small toolbar button used in edit mode */
function ToolbarButton({
  onClick,
  icon,
  label,
}: {
  onClick: () => void;
  icon: React.ReactNode;
  label: string;
}) {
  return (
    <button
      onClick={onClick}
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 5,
        padding: '5px 10px',
        borderRadius: 6,
        border: '1px solid var(--border)',
        background: 'var(--bg-card)',
        color: 'var(--text-secondary)',
        fontSize: 11,
        fontWeight: 500,
        cursor: 'pointer',
        transition: 'border-color 150ms',
      }}
      onMouseEnter={(e) => {
        e.currentTarget.style.borderColor = 'var(--border-hover)';
      }}
      onMouseLeave={(e) => {
        e.currentTarget.style.borderColor = 'var(--border)';
      }}
    >
      {icon}
      {label}
    </button>
  );
}
