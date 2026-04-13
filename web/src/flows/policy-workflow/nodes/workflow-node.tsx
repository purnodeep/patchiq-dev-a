import { useState, useEffect } from 'react';
import { Handle, Position, useReactFlow, addEdge, type NodeProps, type Node } from '@xyflow/react';
import { X, Plus } from 'lucide-react';
import { NODE_TYPE_REGISTRY } from '../node-types';
import type { WorkflowNodeType, NodeConfig, NodeExecutionStatus } from '../types';
import { uuid } from '../../../lib/uuid';

export interface WorkflowNodeData {
  nodeType: WorkflowNodeType;
  label: string;
  config: NodeConfig;
  validationError?: string;
  validationWarning?: string;
  executionStatus?: NodeExecutionStatus;
  executionDuration?: string;
}

type WorkflowNodeProps = NodeProps & { data: WorkflowNodeData };

// Precision Clarity design system node colors: 4 categories
function getNodeColors(nodeType: WorkflowNodeType): {
  leftBar: string;
  border: string;
  iconColor: string;
  glow: string;
} {
  const TRIGGER_TYPES: WorkflowNodeType[] = ['trigger'];
  const GATE_TYPES: WorkflowNodeType[] = ['gate', 'approval', 'decision', 'tag_gate'];
  const ERROR_TYPES: WorkflowNodeType[] = ['rollback', 'reboot'];

  if (TRIGGER_TYPES.includes(nodeType)) {
    return {
      leftBar: 'var(--accent)',
      border: 'color-mix(in srgb, var(--accent) 50%, transparent)',
      iconColor: 'var(--accent)',
      glow: 'color-mix(in srgb, var(--accent) 30%, transparent)',
    };
  }
  if (GATE_TYPES.includes(nodeType)) {
    return {
      leftBar: 'var(--signal-warning)',
      border: 'color-mix(in srgb, var(--signal-warning) 40%, transparent)',
      iconColor: 'var(--signal-warning)',
      glow: 'color-mix(in srgb, var(--signal-warning) 25%, transparent)',
    };
  }
  if (ERROR_TYPES.includes(nodeType)) {
    return {
      leftBar: 'var(--signal-critical)',
      border: 'color-mix(in srgb, var(--signal-critical) 1%, transparent)',
      iconColor: 'var(--signal-critical)',
      glow: 'color-mix(in srgb, var(--signal-critical) 1%, transparent)',
    };
  }
  return {
    leftBar: 'var(--border-hover, var(--border))',
    border: 'var(--border)',
    iconColor: 'var(--text-muted)',
    glow: 'transparent',
  };
}

const executionStatusRing: Record<string, string> = {
  running:
    '0 0 0 2px var(--signal-warning), 0 0 10px color-mix(in srgb, var(--signal-warning) 40%, transparent)',
  completed: '0 0 0 2px var(--signal-healthy)',
  failed: '0 0 0 2px var(--signal-critical)',
};

const executionBgTint: Record<string, string> = {
  running: 'color-mix(in srgb, var(--signal-warning) 6%, var(--bg-card))',
  completed: 'color-mix(in srgb, var(--signal-healthy) 6%, var(--bg-card))',
  failed: 'color-mix(in srgb, var(--signal-critical) 6%, var(--bg-card))',
};

export function WorkflowNode({ id, data, selected }: WorkflowNodeProps) {
  const { deleteElements, setNodes, setEdges, getNode } = useReactFlow();
  const [editing, setEditing] = useState(false);
  const [showPicker, setShowPicker] = useState(false);

  useEffect(() => {
    if (!showPicker) return;
    const handler = () => setShowPicker(false);
    const timer = setTimeout(() => document.addEventListener('click', handler), 0);
    return () => {
      clearTimeout(timer);
      document.removeEventListener('click', handler);
    };
  }, [showPicker]);
  const [editLabel, setEditLabel] = useState(data.label);
  const info = NODE_TYPE_REGISTRY[data.nodeType];
  const Icon = info.icon;
  const colors = getNodeColors(data.nodeType);

  const hasError = !!data.validationError;
  const hasWarning = !hasError && !!data.validationWarning;
  const execShadow = data.executionStatus ? (executionStatusRing[data.executionStatus] ?? '') : '';
  const bgColor = data.executionStatus
    ? (executionBgTint[data.executionStatus] ?? 'var(--bg-card)')
    : 'var(--bg-card)';

  const borderColor = hasError
    ? 'color-mix(in srgb, var(--signal-critical) 1%, transparent)'
    : hasWarning
      ? 'color-mix(in srgb, var(--signal-warning) 60%, transparent)'
      : selected
        ? 'var(--accent, #10b981)'
        : colors.border;

  const boxShadow = hasError
    ? '0 0 0 2px color-mix(in srgb, var(--signal-critical) 1%, transparent)'
    : hasWarning
      ? '0 0 0 1px color-mix(in srgb, var(--signal-warning) 40%, transparent)'
      : selected
        ? `0 0 0 1px var(--accent, #10b981), 0 0 8px ${colors.glow}`
        : execShadow ||
          'var(--shadow-sm, 0 2px 4px color-mix(in srgb, var(--bg-canvas) 40%, transparent))';

  return (
    <>
      <style>{`
        .workflow-node-container .workflow-node-delete { opacity: 0; transition: opacity 150ms; }
        .workflow-node-container:hover .workflow-node-delete { opacity: 1; }
        .workflow-node-container .workflow-node-add { opacity: 0; transition: opacity 150ms; }
        .workflow-node-container:hover .workflow-node-add { opacity: 1; }
      `}</style>
      <div
        className="workflow-node-container"
        style={{
          background: bgColor,
          border: `1px solid ${borderColor}`,
          borderRadius: 8,
          minWidth: 180,
          boxShadow,
          position: 'relative',
          overflow: 'hidden',
          transition: 'border-color 100ms, box-shadow 100ms, background 200ms',
        }}
        title={data.validationError || data.validationWarning}
      >
        {/* Delete button (visible on hover) */}
        <button
          className="workflow-node-delete"
          onClick={(e) => {
            e.stopPropagation();
            deleteElements({ nodes: [{ id }] });
          }}
          title="Delete node"
          style={{
            position: 'absolute',
            top: 4,
            right: 4,
            width: 16,
            height: 16,
            borderRadius: '50%',
            border: '1px solid var(--border)',
            background: 'var(--bg-card)',
            color: 'var(--text-muted)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            cursor: 'pointer',
            padding: 0,
            zIndex: 10,
            pointerEvents: 'auto',
            transition: 'color 150ms, border-color 150ms, background 150ms',
          }}
          onMouseEnter={(e) => {
            e.currentTarget.style.color = 'var(--signal-critical)';
            e.currentTarget.style.borderColor = 'var(--signal-critical)';
          }}
          onMouseLeave={(e) => {
            e.currentTarget.style.color = 'var(--text-muted)';
            e.currentTarget.style.borderColor = 'var(--border)';
          }}
        >
          <X style={{ width: 10, height: 10 }} />
        </button>

        {/* Left color bar */}
        <div
          style={{
            position: 'absolute',
            left: 0,
            top: 0,
            bottom: 0,
            width: 3,
            background: hasError
              ? 'var(--signal-critical)'
              : hasWarning
                ? 'var(--signal-warning)'
                : colors.leftBar,
            borderRadius: '8px 0 0 8px',
          }}
        />

        <Handle
          type="target"
          position={Position.Top}
          style={{
            background: 'var(--border-hover, var(--border))',
            width: 8,
            height: 8,
            border: '1px solid var(--border)',
          }}
        />

        <div
          style={{ padding: '8px 12px 8px 16px', display: 'flex', alignItems: 'center', gap: 8 }}
        >
          <Icon style={{ width: 14, height: 14, color: colors.iconColor, flexShrink: 0 }} />
          <div style={{ minWidth: 0, flex: 1 }}>
            <div
              style={{
                fontSize: 9,
                fontFamily: 'var(--font-mono, monospace)',
                textTransform: 'uppercase',
                letterSpacing: '0.08em',
                color: 'var(--text-faint)',
                marginBottom: 2,
              }}
            >
              {info.label}
            </div>
            {editing ? (
              <input
                autoFocus
                value={editLabel}
                onChange={(e) => setEditLabel(e.target.value)}
                onBlur={() => {
                  setEditing(false);
                  if (editLabel.trim() && editLabel !== data.label) {
                    setNodes((nds) =>
                      nds.map((n) =>
                        n.id === id ? { ...n, data: { ...n.data, label: editLabel.trim() } } : n,
                      ),
                    );
                  }
                }}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') e.currentTarget.blur();
                  if (e.key === 'Escape') {
                    setEditLabel(data.label);
                    setEditing(false);
                  }
                }}
                onClick={(e) => e.stopPropagation()}
                style={{
                  fontSize: 12,
                  fontWeight: 600,
                  color: 'var(--text-primary)',
                  background: 'var(--bg-inset)',
                  border: '1px solid var(--accent)',
                  borderRadius: 4,
                  padding: '1px 4px',
                  outline: 'none',
                  width: '100%',
                  maxWidth: 140,
                }}
              />
            ) : (
              <div
                onDoubleClick={() => {
                  setEditing(true);
                  setEditLabel(data.label);
                }}
                style={{
                  fontSize: 12,
                  fontWeight: 600,
                  color: 'var(--text-primary)',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                  maxWidth: 140,
                  cursor: 'text',
                }}
              >
                {data.label}
              </div>
            )}
          </div>
          {data.executionDuration && (
            <span
              style={{
                fontSize: 9,
                fontFamily: 'var(--font-mono, monospace)',
                color: 'var(--text-muted)',
                flexShrink: 0,
              }}
            >
              {data.executionDuration}
            </span>
          )}
        </div>

        <Handle
          type="source"
          position={Position.Bottom}
          style={{
            background: 'var(--border-hover, var(--border))',
            width: 8,
            height: 8,
            border: '1px solid var(--border)',
          }}
        />

        {/* Add connected node button (visible on hover) */}
        <button
          className="workflow-node-add"
          onClick={(e) => {
            e.stopPropagation();
            setShowPicker((v) => !v);
          }}
          title="Add connected node"
          style={{
            position: 'absolute',
            bottom: -12,
            left: '50%',
            transform: 'translateX(-50%)',
            width: 20,
            height: 20,
            borderRadius: '50%',
            border: '1px solid var(--border)',
            background: 'var(--bg-card)',
            color: 'var(--accent, #10b981)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            cursor: 'pointer',
            padding: 0,
            zIndex: 10,
            pointerEvents: 'auto',
            transition: 'border-color 150ms, background 150ms',
          }}
          onMouseEnter={(e) => {
            e.currentTarget.style.background = 'var(--accent, #10b981)';
            e.currentTarget.style.color = '#fff';
            e.currentTarget.style.borderColor = 'var(--accent, #10b981)';
          }}
          onMouseLeave={(e) => {
            e.currentTarget.style.background = 'var(--bg-card)';
            e.currentTarget.style.color = 'var(--accent, #10b981)';
            e.currentTarget.style.borderColor = 'var(--border)';
          }}
        >
          <Plus style={{ width: 12, height: 12 }} />
        </button>

        {showPicker && (
          <div
            onClick={(e) => e.stopPropagation()}
            style={{
              position: 'absolute',
              top: '100%',
              left: '50%',
              transform: 'translateX(-50%)',
              marginTop: 16,
              background: 'var(--bg-card)',
              border: '1px solid var(--border)',
              borderRadius: 8,
              padding: 6,
              zIndex: 9999,
              boxShadow: '0 8px 24px rgba(0,0,0,0.4)',
              display: 'flex',
              flexDirection: 'column' as const,
              gap: 2,
              minWidth: 160,
            }}
          >
            {(Object.keys(NODE_TYPE_REGISTRY) as WorkflowNodeType[]).map((type) => {
              const nodeInfo = NODE_TYPE_REGISTRY[type];
              const TypeIcon = nodeInfo.icon;
              return (
                <button
                  key={type}
                  onClick={(e) => {
                    e.stopPropagation();
                    setShowPicker(false);
                    const currentNode = getNode(id);
                    if (!currentNode) return;
                    const newId = uuid();
                    const newPosition = {
                      x: currentNode.position.x,
                      y: currentNode.position.y + 100,
                    };
                    setNodes((nds) => [
                      ...nds,
                      {
                        id: newId,
                        type: 'workflowNode',
                        position: newPosition,
                        data: { nodeType: type, label: nodeInfo.label, config: {} },
                      } satisfies Node,
                    ]);
                    setEdges((eds) =>
                      addEdge(
                        {
                          id: `e-${id}-${newId}`,
                          source: id,
                          target: newId,
                          type: 'deletable',
                          style: { stroke: 'var(--border-hover, var(--border))', strokeWidth: 2 },
                        },
                        eds,
                      ),
                    );
                  }}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 8,
                    padding: '6px 8px',
                    border: 'none',
                    background: 'transparent',
                    color: 'var(--text-primary)',
                    fontSize: 12,
                    cursor: 'pointer',
                    borderRadius: 4,
                    width: '100%',
                    textAlign: 'left',
                    transition: 'background 100ms',
                  }}
                  onMouseEnter={(e) => {
                    e.currentTarget.style.background = 'var(--bg-elevated, #1a1a1a)';
                  }}
                  onMouseLeave={(e) => {
                    e.currentTarget.style.background = 'transparent';
                  }}
                >
                  <TypeIcon
                    style={{ width: 14, height: 14, color: 'var(--text-muted)', flexShrink: 0 }}
                  />
                  {nodeInfo.label}
                </button>
              );
            })}
          </div>
        )}
      </div>
    </>
  );
}
