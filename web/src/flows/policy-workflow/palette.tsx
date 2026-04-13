import { useState, type DragEvent } from 'react';
import { Search, GripVertical } from 'lucide-react';
import { NODE_TYPE_REGISTRY } from './node-types';
import type { WorkflowNodeType } from './types';

function onDragStart(event: DragEvent, nodeType: WorkflowNodeType) {
  event.dataTransfer.setData('application/patchiq-node-type', nodeType);
  event.dataTransfer.effectAllowed = 'move';
}

// Group node types into categories
const PALETTE_GROUPS: { label: string; types: WorkflowNodeType[] }[] = [
  { label: 'Triggers', types: ['trigger'] },
  { label: 'Logic', types: ['filter', 'decision', 'gate', 'tag_gate', 'compliance_check'] },
  { label: 'Actions', types: ['deployment_wave', 'script', 'notification', 'scan', 'reboot'] },
  { label: 'Terminal', types: ['approval', 'rollback', 'complete'] },
];

// Node category left-bar colors (design system: 4 categories)
function getNodeAccentColor(nodeType: WorkflowNodeType): string {
  if (nodeType === 'trigger') return 'var(--accent)';
  if (['gate', 'approval', 'decision', 'tag_gate'].includes(nodeType))
    return 'var(--signal-warning)';
  if (['rollback', 'reboot'].includes(nodeType)) return 'var(--signal-critical)';
  return 'var(--border-hover, var(--border))';
}

interface PaletteProps {
  collapsed?: boolean;
  onToggle?: () => void;
}

export function Palette({ collapsed = false, onToggle }: PaletteProps) {
  const [search, setSearch] = useState('');

  if (collapsed) {
    return (
      <div
        style={{
          borderRight: '1px solid var(--border)',
          background: 'var(--bg-card)',
          padding: '10px 6px',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          gap: 8,
        }}
      >
        <button
          type="button"
          onClick={onToggle}
          aria-label="Expand palette"
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            width: 28,
            height: 28,
            borderRadius: 6,
            border: '1px solid var(--border)',
            background: 'none',
            color: 'var(--text-muted)',
            cursor: 'pointer',
          }}
        >
          ›
        </button>
      </div>
    );
  }

  const searchLower = search.toLowerCase();
  const filteredGroups = PALETTE_GROUPS.map((group) => ({
    ...group,
    types: group.types.filter((t) => {
      if (!search) return true;
      const info = NODE_TYPE_REGISTRY[t];
      return (
        info.label.toLowerCase().includes(searchLower) || t.toLowerCase().includes(searchLower)
      );
    }),
  })).filter((g) => g.types.length > 0);

  return (
    <div
      style={{
        width: 200,
        borderRight: '1px solid var(--border)',
        background: 'var(--bg-card)',
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden',
      }}
    >
      {/* Header */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '10px 12px 8px',
          borderBottom: '1px solid var(--border)',
        }}
      >
        <span
          style={{
            fontSize: 9,
            fontFamily: 'var(--font-mono)',
            textTransform: 'uppercase',
            letterSpacing: '0.08em',
            color: 'var(--text-faint)',
            fontWeight: 500,
          }}
        >
          Nodes
        </span>
        {onToggle && (
          <button
            type="button"
            onClick={onToggle}
            aria-label="Collapse palette"
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              width: 20,
              height: 20,
              borderRadius: 4,
              border: 'none',
              background: 'none',
              color: 'var(--text-muted)',
              cursor: 'pointer',
              fontSize: 14,
            }}
          >
            ‹
          </button>
        )}
      </div>

      {/* Search */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 6,
          margin: '8px 10px',
          padding: '5px 8px',
          borderRadius: 6,
          border: '1px solid var(--border)',
          background: 'var(--bg-inset)',
        }}
      >
        <Search style={{ width: 11, height: 11, color: 'var(--text-muted)', flexShrink: 0 }} />
        <input
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Filter nodes..."
          aria-label="Filter node types"
          style={{
            flex: 1,
            background: 'none',
            border: 'none',
            outline: 'none',
            fontSize: 11,
            color: 'var(--text-primary)',
            padding: 0,
          }}
        />
      </div>

      {/* Groups */}
      <style>{`
        .palette-scroll::-webkit-scrollbar { width: 4px; }
        .palette-scroll::-webkit-scrollbar-track { background: transparent; }
        .palette-scroll::-webkit-scrollbar-thumb { background: var(--border); border-radius: 4px; }
        .palette-scroll::-webkit-scrollbar-thumb:hover { background: var(--border-hover); }
      `}</style>
      <div
        className="palette-scroll"
        style={{ flex: 1, overflowY: 'auto', padding: '0 10px 10px' }}
      >
        {filteredGroups.map((group) => (
          <div key={group.label} style={{ marginBottom: 12 }}>
            <div
              style={{
                fontSize: 9,
                fontFamily: 'var(--font-mono)',
                textTransform: 'uppercase',
                letterSpacing: '0.08em',
                color: 'var(--text-faint)',
                fontWeight: 500,
                marginBottom: 5,
                paddingLeft: 2,
              }}
            >
              {group.label}
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
              {group.types.map((type) => {
                const info = NODE_TYPE_REGISTRY[type];
                const Icon = info.icon;
                const accentColor = getNodeAccentColor(type);
                return (
                  <div
                    key={type}
                    draggable
                    onDragStart={(e) => onDragStart(e, type)}
                    title="Drag to canvas"
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: 6,
                      height: 34,
                      borderRadius: 6,
                      border: '1px solid var(--border)',
                      background: 'var(--bg-elevated)',
                      cursor: 'grab',
                      overflow: 'hidden',
                      position: 'relative',
                      transition: 'border-color 100ms',
                      paddingRight: 8,
                    }}
                    onMouseEnter={(e) => {
                      (e.currentTarget as HTMLDivElement).style.borderColor = 'var(--border-hover)';
                    }}
                    onMouseLeave={(e) => {
                      (e.currentTarget as HTMLDivElement).style.borderColor = 'var(--border)';
                    }}
                  >
                    {/* Left bar */}
                    <div
                      style={{
                        width: 3,
                        alignSelf: 'stretch',
                        background: accentColor,
                        flexShrink: 0,
                      }}
                    />
                    <GripVertical
                      style={{
                        width: 10,
                        height: 10,
                        color: 'var(--text-faint)',
                        flexShrink: 0,
                        marginLeft: 2,
                      }}
                    />
                    <Icon
                      style={{ width: 12, height: 12, color: 'var(--text-muted)', flexShrink: 0 }}
                    />
                    <span style={{ fontSize: 12, color: 'var(--text-primary)', fontWeight: 500 }}>
                      {info.label}
                    </span>
                  </div>
                );
              })}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
