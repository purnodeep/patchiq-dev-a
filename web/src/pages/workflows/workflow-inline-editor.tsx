import { useState, useRef, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router';
import { ArrowLeft, Pencil, ChevronDown, Clock, CircleCheckBig, Play, X } from 'lucide-react';
import { Skeleton } from '@patchiq/ui';
import { useCan } from '../../app/auth/AuthContext';
import { useWorkflow } from '../../flows/policy-workflow/hooks/use-workflows';
import {
  useWorkflowExecutions,
  useExecuteWorkflow,
} from '../../flows/policy-workflow/hooks/use-workflow-executions';
import { usePublishWorkflow } from '../../flows/policy-workflow/hooks/use-workflows';
import { timeAgo } from '../../lib/time';
import { nodeTypeStyle, nodeTypeIcon } from './workflow-node-styles';
import type { WorkflowNode } from '../../flows/policy-workflow/types';

interface WorkflowInlineEditorProps {
  workflowId: string;
  workflowName: string;
  workflowStatus: string;
  onClose: () => void;
}

const statusConfig: Record<string, { label: string; color: string; bg: string }> = {
  published: {
    label: 'Published',
    color: 'var(--accent)',
    bg: 'color-mix(in srgb, var(--accent) 10%, transparent)',
  },
  draft: {
    label: 'Draft',
    color: 'var(--signal-warning)',
    bg: 'color-mix(in srgb, var(--signal-warning) 10%, transparent)',
  },
  archived: {
    label: 'Archived',
    color: 'var(--text-muted)',
    bg: 'color-mix(in srgb, white 4%, transparent)',
  },
};

export function WorkflowInlineEditor({
  workflowId,
  workflowName,
  workflowStatus,
  onClose,
}: WorkflowInlineEditorProps) {
  const can = useCan();
  const navigate = useNavigate();
  const { data: detail, isLoading: detailLoading, isError: detailError } = useWorkflow(workflowId);
  const { data: executions, isError: execError } = useWorkflowExecutions(workflowId, { limit: 5 });
  const executeWorkflow = useExecuteWorkflow(workflowId);
  const publishWorkflow = usePublishWorkflow(workflowId);
  const [selectedNode, setSelectedNode] = useState<WorkflowNode | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const sectionRef = useRef<HTMLDivElement>(null);
  const canvasRef = useRef<HTMLDivElement>(null);
  const svgRef = useRef<SVGSVGElement>(null);

  const status = statusConfig[workflowStatus] ?? statusConfig.draft;

  useEffect(() => {
    sectionRef.current?.scrollIntoView({ behavior: 'smooth', block: 'start' });
  }, []);

  // Auto-select the first node when detail loads
  useEffect(() => {
    if (detail?.nodes && detail.nodes.length > 0 && !selectedNode) {
      setSelectedNode(detail.nodes[0]);
    }
  }, [detail?.nodes]);

  const nodes = detail?.nodes ?? [];
  const edges = detail?.edges ?? [];

  // Topological layered layout: assign each node to a layer based on its
  // longest-path distance from the root (trigger), then center nodes within each layer.
  // Horizontal (left-to-right) topological layered layout
  const getNodePositions = useCallback(() => {
    const canvasH = canvasRef.current?.clientHeight ?? 280;
    const nodeW = 160;
    const nodeH = 48;
    const gapX = 40;
    const padLeft = 24;

    if (nodes.length === 0) return [];

    const outgoing = new Map<string, string[]>();
    const incoming = new Map<string, string[]>();
    for (const n of nodes) {
      outgoing.set(n.id, []);
      incoming.set(n.id, []);
    }
    for (const e of edges) {
      outgoing.get(e.source_node_id)?.push(e.target_node_id);
      incoming.get(e.target_node_id)?.push(e.source_node_id);
    }

    const layer = new Map<string, number>();
    const roots = nodes.filter((n) => (incoming.get(n.id)?.length ?? 0) === 0);
    const queue = roots.map((n) => n.id);
    for (const id of queue) layer.set(id, 0);

    const visited = new Set<string>();
    while (queue.length > 0) {
      const id = queue.shift()!;
      if (visited.has(id)) continue;
      visited.add(id);
      const lvl = layer.get(id) ?? 0;
      for (const child of outgoing.get(id) ?? []) {
        const cur = layer.get(child) ?? 0;
        layer.set(child, Math.max(cur, lvl + 1));
        queue.push(child);
      }
    }
    const maxLayer = Math.max(0, ...layer.values());
    for (const n of nodes) {
      if (!layer.has(n.id)) layer.set(n.id, maxLayer + 1);
    }

    const layers = new Map<number, typeof nodes>();
    for (const n of nodes) {
      const l = layer.get(n.id) ?? 0;
      if (!layers.has(l)) layers.set(l, []);
      layers.get(l)!.push(n);
    }

    const result: { node: WorkflowNode; x: number; y: number; w: number; h: number }[] = [];
    const sortedLayers = [...layers.keys()].sort((a, b) => a - b);

    for (const l of sortedLayers) {
      const layerNodes = layers.get(l)!;
      const totalH = layerNodes.length * nodeH + (layerNodes.length - 1) * 20;
      const startY = Math.max(8, (canvasH - totalH) / 2);
      const x = padLeft + l * (nodeW + gapX);

      layerNodes.forEach((node, i) => {
        result.push({ node, x, y: startY + i * (nodeH + 20), w: nodeW, h: nodeH });
      });
    }

    return result;
  }, [nodes, edges]);

  const positions = getNodePositions();
  const canvasHeight = Math.max(
    280,
    positions.reduce((max, p) => Math.max(max, p.y + p.h + 32), 0),
  );

  useEffect(() => {
    if (!svgRef.current || nodes.length === 0) return;
    const pos = getNodePositions();
    const svg = svgRef.current;
    while (svg.firstChild) svg.removeChild(svg.firstChild);

    const defs = document.createElementNS('http://www.w3.org/2000/svg', 'defs');
    defs.innerHTML = `<marker id="arrow-m3" markerWidth="7" markerHeight="5" refX="5" refY="2.5" orient="auto"><polygon points="0 0, 7 2.5, 0 5" fill="var(--border)"/></marker>`;
    svg.appendChild(defs);

    for (const edge of edges) {
      const fromPos = pos.find((p) => p.node.id === edge.source_node_id);
      const toPos = pos.find((p) => p.node.id === edge.target_node_id);
      if (!fromPos || !toPos) continue;

      const x1 = fromPos.x + fromPos.w;
      const y1 = fromPos.y + fromPos.h / 2;
      const x2 = toPos.x;
      const y2 = toPos.y + toPos.h / 2;
      const cx = (x1 + x2) / 2;

      const path = document.createElementNS('http://www.w3.org/2000/svg', 'path');
      path.setAttribute('d', `M${x1},${y1} C${cx},${y1} ${cx},${y2} ${x2},${y2}`);
      path.setAttribute('fill', 'none');
      path.setAttribute('stroke', 'var(--border)');
      path.setAttribute('stroke-width', '2');
      path.setAttribute('marker-end', 'url(#arrow-m3)');
      svg.appendChild(path);
    }
  }, [nodes, edges, getNodePositions]);

  return (
    <div
      ref={sectionRef}
      style={{
        borderTop: '1px solid var(--border)',
        paddingTop: 20,
        marginTop: 4,
      }}
    >
      {/* Header */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 10,
          marginBottom: 16,
        }}
      >
        <button
          onClick={onClose}
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 5,
            padding: '4px 10px',
            borderRadius: 6,
            border: '1px solid var(--border)',
            background: 'none',
            color: 'var(--text-secondary)',
            fontSize: 12,
            cursor: 'pointer',
          }}
        >
          <ArrowLeft style={{ width: 12, height: 12 }} />
          Back
        </button>

        <span
          style={{
            fontSize: 15,
            fontWeight: 700,
            color: 'var(--text-emphasis)',
          }}
        >
          {workflowName}
        </span>

        <div
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            padding: '2px 8px',
            borderRadius: 4,
            background: status.bg,
            border: `1px solid color-mix(in srgb, ${status.color} 20%, transparent)`,
            fontSize: 10,
            fontFamily: 'var(--font-mono)',
            color: status.color,
          }}
        >
          {status.label}
        </div>

        <div style={{ marginLeft: 'auto', display: 'flex', gap: 8 }}>
          {workflowStatus === 'draft' && (
            <button
              disabled={
                publishWorkflow.isPending || nodes.length < 2 || !can('workflows', 'execute')
              }
              title={
                !can('workflows', 'execute')
                  ? "You don't have permission"
                  : nodes.length < 2
                    ? 'Add at least a trigger and complete node before publishing'
                    : 'Publish this workflow'
              }
              onClick={() => publishWorkflow.mutate()}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 6,
                padding: '5px 12px',
                borderRadius: 7,
                border: 'none',
                background: 'var(--accent)',
                color: 'var(--text-on-color, #fff)',
                fontSize: 12,
                fontWeight: 600,
                cursor: publishWorkflow.isPending ? 'not-allowed' : 'pointer',
                opacity: publishWorkflow.isPending ? 0.6 : 1,
              }}
            >
              <CircleCheckBig style={{ width: 12, height: 12 }} />
              {publishWorkflow.isPending ? 'Publishing…' : 'Publish'}
            </button>
          )}
          {workflowStatus === 'published' && (
            <button
              disabled={executeWorkflow.isPending || !can('workflows', 'execute')}
              title={!can('workflows', 'execute') ? "You don't have permission" : undefined}
              onClick={() => executeWorkflow.mutate()}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 6,
                padding: '5px 12px',
                borderRadius: 7,
                border: 'none',
                background: 'var(--accent)',
                color: 'var(--text-on-color, #fff)',
                fontSize: 12,
                fontWeight: 600,
                cursor: executeWorkflow.isPending ? 'not-allowed' : 'pointer',
                opacity: executeWorkflow.isPending ? 0.6 : 1,
              }}
            >
              <Play style={{ width: 12, height: 12 }} />
              {executeWorkflow.isPending ? 'Executing…' : 'Execute'}
            </button>
          )}
          <button
            onClick={() => navigate(`/workflows/${workflowId}/edit`)}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 6,
              padding: '5px 12px',
              borderRadius: 7,
              border: '1px solid var(--border)',
              background: 'none',
              color: 'var(--text-secondary)',
              fontSize: 12,
              fontWeight: 500,
              cursor: 'pointer',
            }}
          >
            <Pencil style={{ width: 12, height: 12 }} />
            Edit
          </button>
        </div>
      </div>

      {/* Canvas + Config panel */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: '1fr 260px',
          border: '1px solid var(--border)',
          borderRadius: '10px 10px 0 0',
          overflow: 'hidden',
          minHeight: canvasHeight,
        }}
      >
        {/* Canvas */}
        <div
          ref={canvasRef}
          style={{
            position: 'relative',
            overflow: 'hidden',
            background: 'var(--bg-canvas)',
          }}
        >
          {/* Dot grid */}
          <div
            style={{
              position: 'absolute',
              inset: 0,
              backgroundImage:
                'radial-gradient(color-mix(in srgb, white 8%, transparent) 1px, transparent 1px)',
              backgroundSize: '20px 20px',
              pointerEvents: 'none',
            }}
          />

          {detailLoading && (
            <div
              style={{
                position: 'absolute',
                inset: 0,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              <Skeleton className="h-10 w-48 rounded-lg" />
            </div>
          )}
          {detailError && (
            <div
              style={{
                position: 'absolute',
                inset: 0,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              <div
                style={{
                  borderRadius: 8,
                  border: '1px solid color-mix(in srgb, var(--signal-critical) 1%, transparent)',
                  background: 'color-mix(in srgb, var(--signal-critical) 1%, transparent)',
                  padding: '8px 16px',
                  fontSize: 12,
                  color: 'var(--signal-critical)',
                }}
              >
                Failed to load workflow nodes.
              </div>
            </div>
          )}

          {/* Empty canvas state */}
          {!detailLoading && !detailError && nodes.length === 0 && (
            <div
              style={{
                position: 'absolute',
                inset: 0,
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                justifyContent: 'center',
                gap: 8,
                pointerEvents: 'none',
              }}
            >
              <p style={{ fontSize: 13, color: 'var(--text-muted)' }}>No nodes in this workflow</p>
              <p style={{ fontSize: 11, color: 'var(--text-faint)' }}>
                Open the editor to add nodes
              </p>
            </div>
          )}

          <svg
            ref={svgRef}
            style={{
              position: 'absolute',
              inset: 0,
              width: '100%',
              height: '100%',
              pointerEvents: 'none',
            }}
          />

          <div style={{ position: 'absolute', inset: 0 }}>
            {positions.map((pos) => {
              const style = nodeTypeStyle[pos.node.node_type] ?? nodeTypeStyle.complete;
              const Icon = nodeTypeIcon[pos.node.node_type] ?? CircleCheckBig;
              const isSelected = selectedNode?.id === pos.node.id;

              return (
                <div
                  key={pos.node.id}
                  style={{
                    position: 'absolute',
                    left: pos.x,
                    top: pos.y,
                    cursor: 'pointer',
                    userSelect: 'none',
                  }}
                  onClick={() => setSelectedNode(pos.node)}
                >
                  <div
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: 8,
                      width: pos.w,
                      height: pos.h,
                      borderRadius: 8,
                      border: `1px solid ${isSelected ? 'var(--accent)' : style.border}`,
                      background: style.bg,
                      boxShadow: isSelected
                        ? '0 0 0 1px var(--accent), 0 0 8px var(--accent-border)'
                        : 'var(--shadow-sm)',
                      paddingLeft: 0,
                      overflow: 'hidden',
                      position: 'relative',
                      transition: 'border-color 100ms, box-shadow 100ms',
                    }}
                  >
                    {/* Left color bar */}
                    <div
                      style={{
                        position: 'absolute',
                        left: 0,
                        top: 0,
                        bottom: 0,
                        width: 3,
                        background: style.leftBar,
                      }}
                    />
                    <div style={{ paddingLeft: 12, display: 'flex', alignItems: 'center', gap: 8 }}>
                      <Icon style={{ width: 14, height: 14, color: style.text, flexShrink: 0 }} />
                      <div style={{ minWidth: 0 }}>
                        <div
                          style={{
                            fontSize: 9,
                            fontFamily: 'var(--font-mono)',
                            textTransform: 'uppercase',
                            letterSpacing: '0.08em',
                            color: 'var(--text-faint)',
                            marginBottom: 1,
                          }}
                        >
                          {pos.node.node_type.replace('_', ' ')}
                        </div>
                        <div
                          style={{
                            fontSize: 12,
                            fontWeight: 600,
                            color: 'var(--text-primary)',
                            overflow: 'hidden',
                            textOverflow: 'ellipsis',
                            whiteSpace: 'nowrap',
                            maxWidth: pos.w - 48,
                          }}
                        >
                          {pos.node.label}
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        </div>

        {/* Config panel */}
        <div
          style={{
            borderLeft: '1px solid var(--border)',
            background: 'var(--bg-card)',
            padding: 16,
            overflowY: 'auto',
          }}
        >
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              marginBottom: 14,
            }}
          >
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 6,
                fontSize: 11,
                fontWeight: 600,
                color: 'var(--text-secondary)',
              }}
            >
              {selectedNode ? (
                (() => {
                  const Icon = nodeTypeIcon[selectedNode.node_type] ?? CircleCheckBig;
                  const style = nodeTypeStyle[selectedNode.node_type] ?? nodeTypeStyle.complete;
                  return <Icon style={{ width: 12, height: 12, color: style.text }} />;
                })()
              ) : (
                <CircleCheckBig style={{ width: 12, height: 12 }} />
              )}
              <span>{selectedNode ? selectedNode.label : 'Node Configuration'}</span>
            </div>
            {selectedNode && (
              <button
                onClick={() => setSelectedNode(null)}
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
                }}
              >
                <X style={{ width: 11, height: 11 }} />
              </button>
            )}
          </div>

          {selectedNode ? (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
              <ConfigField label="Type" value={selectedNode.node_type} />
              <ConfigField label="Label" value={selectedNode.label} />
              {selectedNode.config &&
                (() => {
                  let parsed: Record<string, unknown> | null = null;
                  const raw = selectedNode.config;
                  if (typeof raw === 'string') {
                    try {
                      parsed = JSON.parse(atob(raw));
                    } catch {
                      try {
                        parsed = JSON.parse(raw);
                      } catch {
                        /* ignore */
                      }
                    }
                  } else if (raw && typeof raw === 'object') {
                    parsed = raw as Record<string, unknown>;
                  }
                  if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
                    return Object.entries(parsed).map(([key, val]) => (
                      <ConfigField
                        key={key}
                        label={key}
                        value={Array.isArray(val) ? val.join(', ') : String(val ?? '')}
                      />
                    ));
                  }
                  return null;
                })()}
              {(!selectedNode.config ||
                (() => {
                  const raw = selectedNode.config;
                  let parsed: Record<string, unknown> | null = null;
                  if (typeof raw === 'string') {
                    try {
                      parsed = JSON.parse(atob(raw));
                    } catch {
                      try {
                        parsed = JSON.parse(raw);
                      } catch {
                        /* ignore */
                      }
                    }
                  } else if (raw && typeof raw === 'object') {
                    parsed = raw as Record<string, unknown>;
                  }
                  return !parsed || Object.keys(parsed).length === 0;
                })()) && (
                <p
                  style={{
                    fontSize: 11,
                    color: 'var(--text-muted)',
                    fontStyle: 'italic',
                    marginTop: 4,
                  }}
                >
                  No additional configuration
                </p>
              )}
            </div>
          ) : (
            <div
              style={{
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                gap: 8,
                padding: '24px 12px',
                textAlign: 'center',
              }}
            >
              <CircleCheckBig style={{ width: 20, height: 20, color: 'var(--text-faint)' }} />
              <p style={{ fontSize: 11, color: 'var(--text-muted)', margin: 0 }}>
                Select a node to view its configuration
              </p>
            </div>
          )}
        </div>
      </div>

      {/* Execution history drawer */}
      <div
        style={{
          border: '1px solid var(--border)',
          borderTop: 'none',
          borderRadius: '0 0 10px 10px',
          overflow: 'hidden',
        }}
      >
        <button
          style={{
            display: 'flex',
            width: '100%',
            alignItems: 'center',
            gap: 8,
            padding: '10px 16px',
            textAlign: 'left',
            background: 'var(--bg-card)',
            border: 'none',
            cursor: 'pointer',
            borderTop: '1px solid var(--border)',
          }}
          onClick={() => setDrawerOpen((o) => !o)}
        >
          <Clock style={{ width: 13, height: 13, color: 'var(--text-muted)' }} />
          <span style={{ fontSize: 11, fontWeight: 600, color: 'var(--text-secondary)' }}>
            Execution History
          </span>
          <ChevronDown
            style={{
              marginLeft: 'auto',
              width: 12,
              height: 12,
              color: 'var(--text-muted)',
              transform: drawerOpen ? 'rotate(180deg)' : 'rotate(0deg)',
              transition: 'transform 150ms',
            }}
          />
        </button>

        {drawerOpen && (
          <div
            style={{
              padding: '0 16px 12px',
              background: 'var(--bg-card)',
            }}
          >
            {execError ? (
              <p style={{ fontSize: 11, color: 'var(--signal-critical)', paddingTop: 8 }}>
                Failed to load execution history.
              </p>
            ) : executions?.data && executions.data.length > 0 ? (
              <table style={{ width: '100%', fontSize: 11, borderCollapse: 'collapse' }}>
                <thead>
                  <tr style={{ borderBottom: '1px solid var(--border)' }}>
                    {['Date', 'Status', 'Duration', 'Triggered By'].map((h) => (
                      <th
                        key={h}
                        style={{
                          padding: '6px 10px',
                          textAlign: 'left',
                          fontFamily: 'var(--font-mono)',
                          fontSize: 9,
                          textTransform: 'uppercase',
                          letterSpacing: '0.08em',
                          color: 'var(--text-faint)',
                          fontWeight: 500,
                        }}
                      >
                        {h}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {executions.data.map((exec) => {
                    const duration =
                      exec.started_at && exec.completed_at
                        ? formatDuration(
                            new Date(exec.completed_at).getTime() -
                              new Date(exec.started_at).getTime(),
                          )
                        : '—';
                    return (
                      <tr
                        key={exec.id}
                        style={{ borderBottom: '1px solid var(--border)', opacity: 0.9 }}
                      >
                        <td style={{ padding: '6px 10px', color: 'var(--text-muted)' }}>
                          {exec.started_at ? timeAgo(exec.started_at) : '—'}
                        </td>
                        <td style={{ padding: '6px 10px' }}>
                          <span
                            style={{
                              color:
                                exec.status === 'completed'
                                  ? 'var(--signal-healthy)'
                                  : exec.status === 'failed'
                                    ? 'var(--signal-critical)'
                                    : 'var(--signal-warning)',
                              fontFamily: 'var(--font-mono)',
                            }}
                          >
                            {exec.status === 'completed'
                              ? '✓ Success'
                              : exec.status === 'failed'
                                ? '✗ Failed'
                                : exec.status}
                          </span>
                        </td>
                        <td
                          style={{
                            padding: '6px 10px',
                            color: 'var(--text-muted)',
                            fontFamily: 'var(--font-mono)',
                          }}
                        >
                          {duration}
                        </td>
                        <td style={{ padding: '6px 10px', color: 'var(--text-muted)' }}>
                          {exec.triggered_by}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            ) : (
              <p style={{ fontSize: 11, color: 'var(--text-muted)', paddingTop: 8 }}>
                No executions yet.
              </p>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

function ConfigField({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div
        style={{
          fontSize: 9,
          fontFamily: 'var(--font-mono)',
          textTransform: 'uppercase',
          letterSpacing: '0.08em',
          color: 'var(--text-faint)',
          marginBottom: 4,
        }}
      >
        {label}
      </div>
      <div
        style={{
          fontSize: 12,
          fontFamily: 'var(--font-mono)',
          background: 'var(--bg-inset)',
          border: '1px solid var(--border)',
          borderRadius: 5,
          padding: '5px 8px',
          color: 'var(--text-primary)',
          wordBreak: 'break-all',
        }}
      >
        {value || '—'}
      </div>
    </div>
  );
}

function formatDuration(ms: number): string {
  const secs = Math.floor(ms / 1000);
  if (secs < 60) return `${secs}s`;
  const mins = Math.floor(secs / 60);
  const remSecs = secs % 60;
  return `${mins}m ${remSecs}s`;
}
