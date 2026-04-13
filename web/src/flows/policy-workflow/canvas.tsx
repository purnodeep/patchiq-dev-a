import { useCallback, useEffect, useRef, type DragEvent } from 'react';
import {
  ReactFlow,
  Controls,
  MiniMap,
  Background,
  BackgroundVariant,
  BaseEdge,
  EdgeLabelRenderer,
  getBezierPath,
  addEdge,
  useStore,
  type Node,
  type Edge,
  type EdgeProps,
  type OnNodesChange,
  type OnEdgesChange,
  type Connection,
  type IsValidConnection,
  type NodeMouseHandler,
  useReactFlow,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import { WorkflowNode } from './nodes/workflow-node';
import { NODE_TYPE_REGISTRY } from './node-types';
import type { WorkflowNodeType, DecisionConfig } from './types';
import type { WorkflowNodeData } from './nodes/workflow-node';
import { toast } from 'sonner';
import { uuid } from '../../lib/uuid';

const nodeTypes = { workflowNode: WorkflowNode };

type EdgeHealth = 'healthy' | 'warning' | 'error' | 'unconfigured';

function DeletableEdge({
  id,
  source,
  target,
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
  style,
  selected,
  label,
  labelStyle,
  labelShowBg,
  labelBgStyle,
  labelBgPadding,
  labelBgBorderRadius,
}: EdgeProps) {
  const { deleteElements } = useReactFlow();
  const [edgePath, labelX, labelY] = getBezierPath({
    sourceX,
    sourceY,
    targetX,
    targetY,
    sourcePosition,
    targetPosition,
  });

  const { health: edgeHealth, detail: healthDetail } = useStore(
    useCallback(
      (s): { health: EdgeHealth; detail: string } => {
        const srcNode = s.nodes.find((n: Node) => n.id === source);
        const tgtNode = s.nodes.find((n: Node) => n.id === target);
        const srcData = srcNode?.data as WorkflowNodeData | undefined;
        const tgtData = tgtNode?.data as WorkflowNodeData | undefined;
        const srcLabel = srcData?.label || 'Source';
        const tgtLabel = tgtData?.label || 'Target';

        if (srcData?.validationError || tgtData?.validationError) {
          const msg = srcData?.validationError
            ? `${srcLabel}: ${srcData.validationError}`
            : `${tgtLabel}: ${tgtData!.validationError}`;
          return { health: 'error', detail: msg };
        }
        if (srcData?.validationWarning || tgtData?.validationWarning) {
          const msg = srcData?.validationWarning
            ? `${srcLabel}: ${srcData.validationWarning}`
            : `${tgtLabel}: ${tgtData!.validationWarning}`;
          return { health: 'warning', detail: msg };
        }
        const srcConfigured =
          srcData?.config && Object.keys(srcData.config as Record<string, unknown>).length > 0;
        const tgtConfigured =
          tgtData?.config && Object.keys(tgtData.config as Record<string, unknown>).length > 0;
        if (!srcConfigured && !tgtConfigured)
          return {
            health: 'unconfigured',
            detail: `${srcLabel} and ${tgtLabel} need configuration`,
          };
        if (!srcConfigured)
          return { health: 'unconfigured', detail: `${srcLabel} needs configuration` };
        if (!tgtConfigured)
          return { health: 'unconfigured', detail: `${tgtLabel} needs configuration` };
        return { health: 'healthy', detail: 'Connection is valid' };
      },
      [source, target],
    ),
  );

  let edgeStyle: React.CSSProperties | undefined;
  switch (edgeHealth) {
    case 'healthy':
      edgeStyle = {
        ...style,
        stroke: 'var(--accent, #10b981)',
        strokeWidth: 2,
        animation: 'edge-glow-green 2s ease-in-out infinite',
      };
      break;
    case 'warning':
      edgeStyle = {
        ...style,
        stroke: 'var(--signal-warning, #f59e0b)',
        strokeWidth: 2,
        animation: 'edge-glow-amber 3s ease-in-out infinite',
      };
      break;
    case 'error':
      edgeStyle = {
        ...style,
        stroke: 'var(--signal-critical, #ef4444)',
        strokeWidth: 2,
        strokeDasharray: '6 4',
        opacity: 0.7,
      };
      break;
    default:
      edgeStyle = {
        ...style,
        strokeDasharray: '6 4',
        opacity: 0.6,
      };
      break;
  }

  if (selected) {
    edgeStyle = { ...edgeStyle, stroke: 'var(--accent, #10b981)', strokeWidth: 2.5 };
  }

  return (
    <>
      <BaseEdge id={id} path={edgePath} style={edgeStyle} />
      <EdgeLabelRenderer>
        <div
          style={{
            position: 'absolute',
            transform: `translate(-50%, -50%) translate(${labelX}px, ${labelY}px)`,
            pointerEvents: 'all',
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            gap: 4,
          }}
        >
          {edgeHealth !== 'healthy' && (
            <div
              style={{
                position: 'absolute',
                bottom: '100%',
                left: '50%',
                transform: 'translateX(-50%)',
                marginBottom: 8,
                padding: '6px 10px',
                borderRadius: 6,
                fontSize: 11,
                fontWeight: 500,
                whiteSpace: 'nowrap',
                pointerEvents: 'none',
                zIndex: 9999,
                boxShadow: '0 4px 12px rgba(0,0,0,0.3)',
                background:
                  edgeHealth === 'error'
                    ? 'var(--signal-critical)'
                    : edgeHealth === 'warning'
                      ? 'var(--signal-warning)'
                      : 'var(--bg-elevated, #1a1a1a)',
                color:
                  edgeHealth === 'error' || edgeHealth === 'warning'
                    ? '#fff'
                    : 'var(--text-secondary)',
                border: `1px solid ${edgeHealth === 'error' ? 'var(--signal-critical)' : edgeHealth === 'warning' ? 'var(--signal-warning)' : 'var(--border)'}`,
              }}
            >
              {healthDetail}
            </div>
          )}
          {label && (
            <div
              style={{
                fontSize: 10,
                fontWeight: 500,
                color: (labelStyle as React.CSSProperties)?.color || 'var(--text-secondary)',
                ...(labelShowBg
                  ? {
                      background:
                        (labelBgStyle as React.CSSProperties)?.background || 'var(--bg-card)',
                      border: `1px solid ${(labelBgStyle as React.CSSProperties)?.borderColor || 'var(--border)'}`,
                      borderRadius: labelBgBorderRadius || 4,
                      padding: `${(labelBgPadding as [number, number])?.[0] || 4}px ${(labelBgPadding as [number, number])?.[1] || 6}px`,
                    }
                  : {}),
              }}
            >
              {label}
            </div>
          )}
          {selected && (
            <button
              onClick={(e) => {
                e.stopPropagation();
                deleteElements({ edges: [{ id }] });
              }}
              title="Remove connection"
              style={{
                width: 20,
                height: 20,
                borderRadius: '50%',
                border: '1px solid var(--signal-critical)',
                background: 'var(--bg-card)',
                color: 'var(--signal-critical)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                cursor: 'pointer',
                fontSize: 12,
                lineHeight: 1,
                padding: 0,
                transition: 'background 150ms',
              }}
              onMouseEnter={(e) => {
                e.currentTarget.style.background = 'var(--signal-critical)';
                e.currentTarget.style.color = '#fff';
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.background = 'var(--bg-card)';
                e.currentTarget.style.color = 'var(--signal-critical)';
              }}
            >
              &times;
            </button>
          )}
        </div>
      </EdgeLabelRenderer>
    </>
  );
}

const edgeTypes = { deletable: DeletableEdge };

/** Check if adding source->target would create a cycle via DFS from target */
function wouldCreateCycle(
  source: string,
  target: string,
  edges: Array<{ source: string; target: string }>,
): boolean {
  const adjacency = new Map<string, string[]>();
  for (const e of edges) {
    const list = adjacency.get(e.source) ?? [];
    list.push(e.target);
    adjacency.set(e.source, list);
  }

  const visited = new Set<string>();
  const stack = [target];
  while (stack.length > 0) {
    const node = stack.pop()!;
    if (node === source) return true;
    if (visited.has(node)) continue;
    visited.add(node);
    for (const neighbor of adjacency.get(node) ?? []) {
      stack.push(neighbor);
    }
  }
  return false;
}

export function isValidWorkflowConnection(
  source: string,
  target: string,
  edges: Array<{ id: string; source: string; target: string }>,
): boolean {
  if (source === target) return false;
  if (edges.some((e) => e.source === source && e.target === target)) return false;
  if (wouldCreateCycle(source, target, edges)) return false;
  return true;
}

/** Map node type to minimap color by category */
function minimapNodeColor(node: Node): string {
  const data = node.data as unknown as WorkflowNodeData;
  if (!data?.nodeType) return '#666';
  const type = data.nodeType;
  if (type === 'trigger') return '#10b981'; // green
  if (['gate', 'approval', 'decision', 'tag_gate'].includes(type)) return '#f59e0b'; // amber
  if (['rollback', 'reboot'].includes(type)) return '#ef4444'; // red
  if (['deployment_wave', 'script', 'notification', 'scan', 'compliance_check'].includes(type))
    return '#3b82f6'; // blue
  return '#6b7280'; // gray
}

interface WorkflowCanvasProps {
  nodes: Node[];
  edges: Edge[];
  onNodesChange: OnNodesChange;
  onEdgesChange: OnEdgesChange;
  onNodeClick: NodeMouseHandler;
  onBeforeChange?: () => void;
}

export function WorkflowCanvas({
  nodes,
  edges,
  onNodesChange,
  onEdgesChange,
  onNodeClick,
  onBeforeChange,
}: WorkflowCanvasProps) {
  const reactFlowWrapper = useRef<HTMLDivElement>(null);
  const clipboardRef = useRef<Node[]>([]);
  const { screenToFlowPosition, setNodes, setEdges, getNodes, getEdges } = useReactFlow();

  const isValidConnection: IsValidConnection = useCallback(
    (connection) =>
      isValidWorkflowConnection(connection.source ?? '', connection.target ?? '', edges),
    [edges],
  );

  /** Get edge label for decision node connections */
  const getDecisionEdgeLabel = useCallback(
    (sourceId: string, existingEdges: Edge[]): string | undefined => {
      const sourceNode = nodes.find((n) => n.id === sourceId);
      if (!sourceNode) return undefined;
      const data = sourceNode.data as unknown as WorkflowNodeData;
      if (data.nodeType !== 'decision') return undefined;
      const config = data.config as DecisionConfig;
      const outgoing = existingEdges.filter((e) => e.source === sourceId);
      if (outgoing.length === 0) return config.true_label || 'Yes';
      if (outgoing.length === 1) return config.false_label || 'No';
      return undefined;
    },
    [nodes],
  );

  const onConnect = useCallback(
    (params: Connection) => {
      const src = params.source ?? '';
      const tgt = params.target ?? '';

      if (src === tgt) {
        toast.error('Cannot connect a node to itself');
        return;
      }
      if (edges.some((e) => e.source === src && e.target === tgt)) {
        toast.error('These nodes are already connected');
        return;
      }
      if (wouldCreateCycle(src, tgt, edges)) {
        toast.error('Cannot connect: this would create a cycle');
        return;
      }

      onBeforeChange?.();
      const label = getDecisionEdgeLabel(src, edges);
      setEdges((eds) =>
        addEdge(
          {
            ...params,
            type: 'deletable',
            ...(label
              ? {
                  label,
                  labelStyle: { fill: 'var(--text-secondary)', fontSize: 10, fontWeight: 500 },
                  labelShowBg: true,
                  labelBgStyle: { fill: 'var(--bg-card)', stroke: 'var(--border)', strokeWidth: 1 },
                  labelBgPadding: [4, 6] as [number, number],
                  labelBgBorderRadius: 4,
                }
              : {}),
            style: { stroke: 'var(--border-hover, var(--border))', strokeWidth: 2 },
          },
          eds,
        ),
      );
    },
    [setEdges, edges, getDecisionEdgeLabel, onBeforeChange],
  );

  const onDragOver = useCallback((event: DragEvent) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = 'move';
  }, []);

  const onDrop = useCallback(
    (event: DragEvent) => {
      event.preventDefault();
      const nodeType = event.dataTransfer.getData(
        'application/patchiq-node-type',
      ) as WorkflowNodeType;
      if (!nodeType || !NODE_TYPE_REGISTRY[nodeType]) return;

      onBeforeChange?.();
      const position = screenToFlowPosition({ x: event.clientX, y: event.clientY });
      const info = NODE_TYPE_REGISTRY[nodeType];
      const newNodeId = uuid();
      const newNode: Node = {
        id: newNodeId,
        type: 'workflowNode',
        position,
        data: { nodeType, label: info.label, config: {} },
      };

      setNodes((nds) => [...nds, newNode]);

      // Auto-connect: find nearest node whose bottom is close to the new node's top
      const AUTO_CONNECT_THRESHOLD = 160;
      const currentNodes = getNodes();
      let closestNode: Node | null = null;
      let closestDist = Infinity;

      for (const node of currentNodes) {
        const bottomX = node.position.x + 90;
        const bottomY = node.position.y + 50;
        const topX = position.x + 90;
        const topY = position.y;

        const dist = Math.sqrt((bottomX - topX) ** 2 + (bottomY - topY) ** 2);
        if (dist < closestDist && dist < AUTO_CONNECT_THRESHOLD && position.y > node.position.y) {
          closestDist = dist;
          closestNode = node;
        }
      }

      if (closestNode) {
        const edgesNow = getEdges();
        const src = closestNode.id;
        if (
          src !== newNodeId &&
          !edgesNow.some((e) => e.source === src && e.target === newNodeId)
        ) {
          const sourceData = closestNode.data as unknown as WorkflowNodeData;
          let label: string | undefined;
          if (sourceData.nodeType === 'decision') {
            const config = sourceData.config as DecisionConfig;
            const outgoing = edgesNow.filter((e) => e.source === src);
            if (outgoing.length === 0) label = config.true_label || 'Yes';
            else if (outgoing.length === 1) label = config.false_label || 'No';
          }

          setEdges((eds) =>
            addEdge(
              {
                id: `e-${src}-${newNodeId}`,
                source: src,
                target: newNodeId,
                type: 'deletable',
                ...(label
                  ? {
                      label,
                      labelStyle: { fill: 'var(--text-secondary)', fontSize: 10, fontWeight: 500 },
                      labelShowBg: true,
                      labelBgStyle: {
                        fill: 'var(--bg-card)',
                        stroke: 'var(--border)',
                        strokeWidth: 1,
                      },
                      labelBgPadding: [4, 6] as [number, number],
                      labelBgBorderRadius: 4,
                    }
                  : {}),
                style: { stroke: 'var(--border-hover, var(--border))', strokeWidth: 2 },
              },
              eds,
            ),
          );
        }
      }
    },
    [screenToFlowPosition, setNodes, setEdges, getNodes, getEdges, onBeforeChange],
  );

  // Copy/paste keyboard handler
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement;
      if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable)
        return;

      const isMod = e.metaKey || e.ctrlKey;

      // Copy: Ctrl/Cmd+C
      if (isMod && e.key === 'c') {
        const selected = getNodes().filter((n) => n.selected);
        if (selected.length > 0) {
          clipboardRef.current = structuredClone(selected);
        }
        return;
      }

      // Paste: Ctrl/Cmd+V
      if (isMod && e.key === 'v') {
        if (clipboardRef.current.length === 0) return;
        e.preventDefault();
        onBeforeChange?.();
        const offset = 50;
        const newNodes = clipboardRef.current.map((n) => ({
          ...structuredClone(n),
          id: uuid(),
          position: { x: n.position.x + offset, y: n.position.y + offset },
          selected: false,
        }));
        setNodes((nds) => [...nds, ...newNodes]);
        // Shift clipboard for subsequent pastes
        clipboardRef.current = clipboardRef.current.map((n) => ({
          ...n,
          position: { x: n.position.x + offset, y: n.position.y + offset },
        }));
        return;
      }
    };

    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, [getNodes, setNodes, onBeforeChange]);

  return (
    <div
      ref={reactFlowWrapper}
      className="h-full w-full relative"
      style={{ background: 'var(--bg-canvas, #080808)' }}
    >
      <style>{`
          @keyframes edge-glow-green {
            0%, 100% { filter: drop-shadow(0 0 1px rgba(16, 185, 129, 0.3)); }
            50% { filter: drop-shadow(0 0 6px rgba(16, 185, 129, 0.6)); }
          }
          @keyframes edge-glow-amber {
            0%, 100% { filter: drop-shadow(0 0 1px rgba(245, 158, 11, 0.3)); }
            50% { filter: drop-shadow(0 0 5px rgba(245, 158, 11, 0.5)); }
          }
        `}</style>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        isValidConnection={isValidConnection}
        onNodeClick={onNodeClick}
        onDragOver={onDragOver}
        onDrop={onDrop}
        nodeTypes={nodeTypes}
        edgeTypes={edgeTypes}
        fitView
        fitViewOptions={{ padding: 0.2 }}
        deleteKeyCode={['Backspace', 'Delete']}
        proOptions={{ hideAttribution: true }}
        defaultEdgeOptions={{
          type: 'deletable',
          interactionWidth: 20,
          style: { stroke: 'var(--border-hover, var(--border))', strokeWidth: 2 },
          animated: false,
        }}
        style={{ background: 'var(--bg-canvas, #080808)' }}
      >
        <Controls
          style={{
            background: 'var(--bg-card, #111113)',
            border: '1px solid var(--border, #222222)',
            borderRadius: 8,
          }}
        />
        <MiniMap
          style={{ background: 'var(--bg-inset)', border: '1px solid var(--border)' }}
          maskColor="rgba(0,0,0,0.5)"
          nodeColor={minimapNodeColor}
          nodeBorderRadius={4}
        />
        <Background
          variant={BackgroundVariant.Dots}
          gap={20}
          size={1}
          color="color-mix(in srgb, white 6%, transparent)"
        />
        {nodes.length === 0 && (
          <div
            style={{
              position: 'absolute',
              inset: 0,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              pointerEvents: 'none',
              zIndex: 10,
            }}
          >
            <div style={{ textAlign: 'center' }}>
              <p
                style={{
                  fontSize: 13,
                  color: 'var(--text-muted, #737373)',
                  fontWeight: 500,
                  marginBottom: 4,
                }}
              >
                Drag nodes from the palette to build your workflow
              </p>
              <p style={{ fontSize: 11, color: 'var(--text-faint, #525252)' }}>
                Connect nodes to define the execution order
              </p>
            </div>
          </div>
        )}
      </ReactFlow>
    </div>
  );
}
