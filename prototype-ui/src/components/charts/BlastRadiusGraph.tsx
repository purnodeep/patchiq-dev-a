// BlastRadiusGraph — React Flow interactive node graph
// Command Glass aesthetic: semi-transparent nodes, animated dash edges,
// compliance-coloured rings, severity-weighted edge thickness.
// Layout: dagre (LR, centre node pinned left)

import React, { useState, useCallback, useMemo, useEffect } from 'react';
import dagre from 'dagre';
import {
  ReactFlow,
  Background,
  Controls,
  useNodesState,
  useEdgesState,
  type Node,
  type Edge,
  type NodeProps,
  Handle,
  Position,
  BackgroundVariant,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';

import {
  BLAST_RADIUS_CVES,
  BLAST_RADIUS_DATA,
  type BlastRadiusNode,
  type BlastRadiusEdge,
} from '@/data/mock-data';

// ── Colour tokens ─────────────────────────────────────────────────────────────

const COMPLIANCE_COLORS = {
  compliant: { border: '#34d399', glow: 'rgba(52,211,153,0.35)', text: '#34d399' },
  'at-risk': { border: '#fbbf24', glow: 'rgba(251,191,36,0.35)', text: '#fbbf24' },
  'non-compliant': { border: '#f87171', glow: 'rgba(248,113,113,0.35)', text: '#f87171' },
} as const;

const SEVERITY_COLORS = {
  critical: '#f87171',
  high: '#fbbf24',
  medium: '#a78bfa',
  low: '#60a5fa',
} as const;

const SEVERITY_STROKE = {
  critical: 4,
  high: 3,
  medium: 2,
  low: 1,
} as const;

// ── Node dimensions (used for layout + rendering) ─────────────────────────────

const CVE_NODE_W = 160;
const CVE_NODE_H = 160;
const GROUP_NODE_W = 176;
const GROUP_NODE_H = 72;

// ── Layout types ──────────────────────────────────────────────────────────────

type GraphLayout = 'LR' | 'TB' | 'circular';

// ── dagre auto-layout (LR / TB) ───────────────────────────────────────────────

function applyDagreLayout(
  rawNodes: BlastRadiusNode[],
  rawEdges: BlastRadiusEdge[],
  rankdir: 'LR' | 'TB',
): { x: number; y: number }[] {
  const g = new dagre.graphlib.Graph();
  g.setDefaultEdgeLabel(() => ({}));
  g.setGraph({ rankdir, nodesep: 28, ranksep: 96 });

  rawNodes.forEach((n) => {
    const isCenter = n.id === 'cve-center';
    g.setNode(n.id, {
      width: isCenter ? CVE_NODE_W : GROUP_NODE_W,
      height: isCenter ? CVE_NODE_H : GROUP_NODE_H,
    });
  });

  rawEdges.forEach((e) => g.setEdge(e.source, e.target));
  dagre.layout(g);

  return rawNodes.map((n) => {
    const { x, y } = g.node(n.id);
    const isCenter = n.id === 'cve-center';
    const w = isCenter ? CVE_NODE_W : GROUP_NODE_W;
    const h = isCenter ? CVE_NODE_H : GROUP_NODE_H;
    return { x: x - w / 2, y: y - h / 2 };
  });
}

// ── circular layout ───────────────────────────────────────────────────────────

function applyCircularLayout(rawNodes: BlastRadiusNode[]): { x: number; y: number }[] {
  const groupNodes = rawNodes.filter((n) => n.id !== 'cve-center');
  const count = groupNodes.length;
  const radius = Math.max(220, count * 36);

  const groupPositions = groupNodes.map((_, i) => {
    const angle = (2 * Math.PI * i) / count - Math.PI / 2; // start at top
    return {
      x: radius * Math.cos(angle) - GROUP_NODE_W / 2,
      y: radius * Math.sin(angle) - GROUP_NODE_H / 2,
    };
  });

  return rawNodes.map((n) => {
    if (n.id === 'cve-center') {
      return { x: -CVE_NODE_W / 2, y: -CVE_NODE_H / 2 };
    }
    return groupPositions[groupNodes.indexOf(n)];
  });
}

// ── unified layout dispatch ───────────────────────────────────────────────────

function computeLayout(
  rawNodes: BlastRadiusNode[],
  rawEdges: BlastRadiusEdge[],
  layout: GraphLayout,
): { x: number; y: number }[] {
  if (layout === 'circular') return applyCircularLayout(rawNodes);
  return applyDagreLayout(rawNodes, rawEdges, layout);
}

// ── handle-side helpers (for circular edge routing) ───────────────────────────

type Side = 'top' | 'right' | 'bottom' | 'left';

function sideFromAngle(angleDeg: number): Side {
  const a = ((angleDeg % 360) + 360) % 360;
  if (a < 45 || a >= 315) return 'right';
  if (a < 135) return 'bottom';
  if (a < 225) return 'left';
  return 'top';
}

function oppositeSide(s: Side): Side {
  return s === 'right' ? 'left' : s === 'left' ? 'right' : s === 'top' ? 'bottom' : 'top';
}

// ── CVE centre node ───────────────────────────────────────────────────────────

interface CveNodeData {
  label: string;
  sublabel: string;
  severity: keyof typeof SEVERITY_COLORS;
  [key: string]: unknown;
}

function CveNode({ data }: NodeProps) {
  const d = data as CveNodeData;
  const col = SEVERITY_COLORS[d.severity];
  const [hovered, setHovered] = useState(false);

  return (
    <>
      <Handle
        id="source-top"
        type="source"
        position={Position.Top}
        style={{ opacity: 0, width: 1, height: 1 }}
      />
      <Handle
        id="source-right"
        type="source"
        position={Position.Right}
        style={{ opacity: 0, width: 1, height: 1 }}
      />
      <Handle
        id="source-bottom"
        type="source"
        position={Position.Bottom}
        style={{ opacity: 0, width: 1, height: 1 }}
      />
      <Handle
        id="source-left"
        type="source"
        position={Position.Left}
        style={{ opacity: 0, width: 1, height: 1 }}
      />
      <Handle
        id="target-top"
        type="target"
        position={Position.Top}
        style={{ opacity: 0, width: 1, height: 1 }}
      />
      <Handle
        id="target-right"
        type="target"
        position={Position.Right}
        style={{ opacity: 0, width: 1, height: 1 }}
      />
      <Handle
        id="target-bottom"
        type="target"
        position={Position.Bottom}
        style={{ opacity: 0, width: 1, height: 1 }}
      />
      <Handle
        id="target-left"
        type="target"
        position={Position.Left}
        style={{ opacity: 0, width: 1, height: 1 }}
      />

      {/* outer halo — only on hover */}
      {hovered && (
        <div
          style={{
            position: 'absolute',
            inset: -8,
            borderRadius: '50%',
            border: `1px solid ${col}30`,
            pointerEvents: 'none',
          }}
        />
      )}

      {/* circle */}
      <div
        onMouseEnter={() => setHovered(true)}
        onMouseLeave={() => setHovered(false)}
        style={{
          width: CVE_NODE_W,
          height: CVE_NODE_H,
          borderRadius: '50%',
          background: `${col}22`,
          border: `1.5px solid ${col}80`,
          boxShadow: hovered ? `0 0 20px ${col}50` : 'none',
          transition: 'box-shadow 0.2s ease',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          gap: 6,
          padding: '0 20px',
          textAlign: 'center',
          cursor: 'default',
          userSelect: 'none',
          boxSizing: 'border-box',
        }}
      >
        {/* severity */}
        <div
          style={{
            fontSize: 9,
            fontWeight: 700,
            letterSpacing: '0.12em',
            textTransform: 'uppercase',
            color: col,
            background: `${col}18`,
            border: `1px solid ${col}40`,
            borderRadius: 100,
            padding: '2px 8px',
          }}
        >
          {d.severity}
        </div>

        {/* CVE ID */}
        <div
          style={{
            fontSize: 13,
            fontWeight: 700,
            color: 'var(--color-foreground)',
            letterSpacing: '-0.01em',
            lineHeight: 1.1,
          }}
        >
          {d.label}
        </div>

        {/* thin rule */}
        <div style={{ width: 32, height: 1, background: `${col}40`, flexShrink: 0 }} />

        {/* description */}
        <div
          style={{
            fontSize: 10,
            fontWeight: 400,
            color: 'var(--color-muted)',
            lineHeight: 1.35,
          }}
        >
          {d.sublabel}
        </div>
      </div>
    </>
  );
}

// ── Endpoint group node ────────────────────────────────────────────────────────

interface GroupNodeData {
  label: string;
  sublabel: string;
  count: number;
  compliance: keyof typeof COMPLIANCE_COLORS;
  [key: string]: unknown;
}

function GroupNode({ data }: NodeProps) {
  const d = data as GroupNodeData;
  const comp = COMPLIANCE_COLORS[d.compliance];
  const [hovered, setHovered] = useState(false);

  const compLabel =
    d.compliance === 'non-compliant'
      ? 'Non-Compliant'
      : d.compliance === 'at-risk'
        ? 'At Risk'
        : 'Compliant';

  return (
    <>
      <Handle
        id="source-top"
        type="source"
        position={Position.Top}
        style={{ opacity: 0, width: 1, height: 1 }}
      />
      <Handle
        id="source-right"
        type="source"
        position={Position.Right}
        style={{ opacity: 0, width: 1, height: 1 }}
      />
      <Handle
        id="source-bottom"
        type="source"
        position={Position.Bottom}
        style={{ opacity: 0, width: 1, height: 1 }}
      />
      <Handle
        id="source-left"
        type="source"
        position={Position.Left}
        style={{ opacity: 0, width: 1, height: 1 }}
      />
      <Handle
        id="target-top"
        type="target"
        position={Position.Top}
        style={{ opacity: 0, width: 1, height: 1 }}
      />
      <Handle
        id="target-right"
        type="target"
        position={Position.Right}
        style={{ opacity: 0, width: 1, height: 1 }}
      />
      <Handle
        id="target-bottom"
        type="target"
        position={Position.Bottom}
        style={{ opacity: 0, width: 1, height: 1 }}
      />
      <Handle
        id="target-left"
        type="target"
        position={Position.Left}
        style={{ opacity: 0, width: 1, height: 1 }}
      />
      <div
        onMouseEnter={() => setHovered(true)}
        onMouseLeave={() => setHovered(false)}
        style={{
          width: GROUP_NODE_W,
          height: GROUP_NODE_H,
          borderRadius: 10,
          background: `${comp.border}18`,
          border: `1px solid ${comp.border}60`,
          boxShadow: hovered ? `0 0 18px ${comp.glow}` : 'none',
          transition: 'box-shadow 0.2s ease',
          display: 'flex',
          flexDirection: 'row',
          cursor: 'default',
          userSelect: 'none',
          overflow: 'hidden',
          boxSizing: 'border-box',
        }}
      >
        {/* left: count hero */}
        <div
          style={{
            width: 56,
            flexShrink: 0,
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            background: `${comp.border}28`,
            borderRight: `1px solid ${comp.border}28`,
            gap: 2,
          }}
        >
          <span
            style={{
              fontSize: 22,
              fontWeight: 800,
              color: comp.text,
              lineHeight: 1,
              letterSpacing: '-0.03em',
            }}
          >
            {d.count}
          </span>
          <span
            style={{
              fontSize: 8,
              fontWeight: 600,
              color: `${comp.text}88`,
              letterSpacing: '0.08em',
              textTransform: 'uppercase',
            }}
          >
            eps
          </span>
        </div>

        {/* right: name + os + compliance */}
        <div
          style={{
            flex: 1,
            display: 'flex',
            flexDirection: 'column',
            justifyContent: 'center',
            gap: 4,
            padding: '0 12px',
            minWidth: 0,
          }}
        >
          <div
            style={{
              fontSize: 12,
              fontWeight: 700,
              color: 'var(--color-foreground)',
              whiteSpace: 'nowrap',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              letterSpacing: '-0.01em',
            }}
          >
            {d.label}
          </div>

          <div
            style={{
              fontSize: 10,
              fontWeight: 400,
              color: 'var(--color-muted)',
              whiteSpace: 'nowrap',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
            }}
          >
            {d.sublabel}
          </div>

          <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
            <div
              style={{
                width: 5,
                height: 5,
                borderRadius: '50%',
                background: comp.border,
                boxShadow: `0 0 5px ${comp.border}`,
                flexShrink: 0,
              }}
            />
            <span
              style={{
                fontSize: 9,
                fontWeight: 600,
                color: comp.text,
                letterSpacing: '0.02em',
              }}
            >
              {compLabel}
            </span>
          </div>
        </div>
      </div>
    </>
  );
}

// ── Node type registry ────────────────────────────────────────────────────────

const nodeTypes = { cveCenter: CveNode, group: GroupNode };

// ── Data conversion ───────────────────────────────────────────────────────────

function buildNodes(
  rawNodes: BlastRadiusNode[],
  rawEdges: BlastRadiusEdge[],
  layout: GraphLayout,
): Node[] {
  const positions = computeLayout(rawNodes, rawEdges, layout);
  return rawNodes.map((n, i) => {
    const isCenter = n.id === 'cve-center';
    return {
      id: n.id,
      type: isCenter ? 'cveCenter' : 'group',
      position: positions[i],
      data: {
        label: n.label,
        sublabel: n.sublabel,
        count: n.count,
        compliance: n.compliance,
        severity: n.severity,
      },
      style: { background: 'transparent', border: 'none', padding: 0, boxShadow: 'none' },
      draggable: true,
    };
  });
}

function buildEdges(
  rawEdges: BlastRadiusEdge[],
  rawNodes: BlastRadiusNode[],
  layout: GraphLayout,
  positions: { x: number; y: number }[],
): Edge[] {
  const complianceByNodeId = Object.fromEntries(rawNodes.map((n) => [n.id, n.compliance]));
  const positionById = Object.fromEntries(rawNodes.map((n, i) => [n.id, positions[i]]));

  return rawEdges.map((e, i) => {
    const targetCompliance = complianceByNodeId[e.target] ?? 'non-compliant';
    const stroke = COMPLIANCE_COLORS[targetCompliance as keyof typeof COMPLIANCE_COLORS].border;

    // For circular layout, route edges from the correct side of each node.
    let sourceHandle: string | undefined;
    let targetHandle: string | undefined;
    if (layout === 'circular') {
      const src = positionById[e.source];
      const tgt = positionById[e.target];
      if (src && tgt) {
        const dx = tgt.x + GROUP_NODE_W / 2 - (src.x + CVE_NODE_W / 2);
        const dy = tgt.y + GROUP_NODE_H / 2 - (src.y + CVE_NODE_H / 2);
        const deg = (Math.atan2(dy, dx) * 180) / Math.PI;
        const side = sideFromAngle(deg);
        sourceHandle = `source-${side}`;
        targetHandle = `target-${oppositeSide(side)}`;
      }
    } else {
      sourceHandle = layout === 'LR' ? 'source-right' : 'source-bottom';
      targetHandle = layout === 'LR' ? 'target-left' : 'target-top';
    }

    return {
      id: `edge-${i}`,
      source: e.source,
      target: e.target,
      sourceHandle,
      targetHandle,
      animated: false,
      style: {
        stroke,
        strokeWidth: SEVERITY_STROKE[e.severity],
        strokeDasharray: '6 4',
        animation: `dash-flow ${1.8 + i * 0.12}s linear infinite`,
        opacity: 0.7,
      },
      type: 'default',
    };
  });
}

// ── CVE Dropdown ──────────────────────────────────────────────────────────────

interface CveDropdownProps {
  selectedId: string;
  onSelect: (id: string) => void;
}

function CveDropdown({ selectedId, onSelect }: CveDropdownProps) {
  const [open, setOpen] = useState(false);
  const selected = BLAST_RADIUS_CVES.find((c) => c.id === selectedId)!;
  const col = SEVERITY_COLORS[selected.severity];

  return (
    <div style={{ position: 'relative' }}>
      <button
        onClick={() => setOpen((o) => !o)}
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 6,
          padding: '4px 10px 4px 8px',
          borderRadius: 7,
          background: 'var(--color-glass-bg)',
          border: '1px solid var(--color-glass-border)',
          cursor: 'pointer',
          color: 'var(--color-foreground)',
          fontSize: 11,
          fontWeight: 600,
          letterSpacing: '0.01em',
          backdropFilter: 'blur(8px)',
          WebkitBackdropFilter: 'blur(8px)',
        }}
      >
        <span
          style={{
            width: 7,
            height: 7,
            borderRadius: '50%',
            background: col,
            boxShadow: `0 0 5px ${col}`,
            display: 'inline-block',
            flexShrink: 0,
          }}
        />
        {selected.label}
        <svg
          width={10}
          height={10}
          viewBox="0 0 10 10"
          fill="none"
          style={{
            transform: open ? 'rotate(180deg)' : 'rotate(0deg)',
            transition: 'transform 0.2s',
            opacity: 0.6,
          }}
        >
          <path
            d="M2 3.5L5 6.5L8 3.5"
            stroke="currentColor"
            strokeWidth={1.5}
            strokeLinecap="round"
            strokeLinejoin="round"
          />
        </svg>
      </button>

      {open && (
        <div
          style={{
            position: 'absolute',
            top: 'calc(100% + 4px)',
            right: 0,
            zIndex: 9999,
            minWidth: 180,
            borderRadius: 9,
            background: 'var(--color-glass-card)',
            backdropFilter: 'blur(20px)',
            WebkitBackdropFilter: 'blur(20px)',
            border: '1px solid var(--color-glass-border)',
            boxShadow: '0 8px 32px rgba(0,0,0,0.5)',
            overflow: 'hidden',
          }}
        >
          {BLAST_RADIUS_CVES.map((cve) => {
            const isActive = cve.id === selectedId;
            const c = SEVERITY_COLORS[cve.severity];
            return (
              <button
                key={cve.id}
                onClick={() => {
                  onSelect(cve.id);
                  setOpen(false);
                }}
                style={{
                  width: '100%',
                  display: 'flex',
                  alignItems: 'center',
                  gap: 8,
                  padding: '8px 12px',
                  background: isActive ? 'var(--color-glass-hover)' : 'transparent',
                  border: 'none',
                  cursor: 'pointer',
                  color: isActive ? 'var(--color-foreground)' : 'var(--color-muted)',
                  fontSize: 11,
                  fontWeight: isActive ? 700 : 500,
                  textAlign: 'left',
                }}
              >
                <span
                  style={{
                    width: 7,
                    height: 7,
                    borderRadius: '50%',
                    background: c,
                    boxShadow: `0 0 5px ${c}`,
                    display: 'inline-block',
                    flexShrink: 0,
                  }}
                />
                <span style={{ flex: 1 }}>{cve.label}</span>
                {isActive && (
                  <svg width={10} height={10} viewBox="0 0 10 10" fill="none">
                    <path
                      d="M2 5L4.5 7.5L8 3"
                      stroke="#34d399"
                      strokeWidth={1.6}
                      strokeLinecap="round"
                      strokeLinejoin="round"
                    />
                  </svg>
                )}
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
}

// ── Legend ────────────────────────────────────────────────────────────────────

function Legend() {
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 14, flexWrap: 'wrap' }}>
      {(
        [
          { label: 'Compliant', color: '#34d399' },
          { label: 'At-Risk', color: '#fbbf24' },
          { label: 'Non-Compliant', color: '#f87171' },
        ] as const
      ).map(({ label, color }) => (
        <div key={label} style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
          <div
            style={{
              width: 6,
              height: 6,
              borderRadius: '50%',
              background: color,
              boxShadow: `0 0 5px ${color}`,
            }}
          />
          <span style={{ fontSize: 9, color: 'var(--color-muted)', fontWeight: 500 }}>{label}</span>
        </div>
      ))}

      <div style={{ width: 1, height: 12, background: 'var(--color-separator)' }} />

      {(
        [
          { label: 'Critical', thickness: 4 },
          { label: 'High', thickness: 3 },
          { label: 'Med', thickness: 2 },
          { label: 'Low', thickness: 1 },
        ] as const
      ).map(({ label, thickness }) => (
        <div key={label} style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
          <svg width={18} height={8} viewBox="0 0 18 8" fill="none">
            <line
              x1={0}
              y1={4}
              x2={18}
              y2={4}
              stroke="var(--color-muted)"
              strokeWidth={thickness}
              strokeDasharray="4 3"
              strokeLinecap="round"
            />
          </svg>
          <span style={{ fontSize: 9, color: 'var(--color-muted)', fontWeight: 500 }}>{label}</span>
        </div>
      ))}
    </div>
  );
}

// ── Layout dropdown ───────────────────────────────────────────────────────────

const LAYOUT_OPTIONS: { id: GraphLayout; label: string; icon: React.ReactNode }[] = [
  {
    id: 'LR',
    label: 'Horizontal',
    icon: (
      <svg width={13} height={10} viewBox="0 0 13 10" fill="none">
        <circle cx={2} cy={5} r={1.5} fill="currentColor" />
        <circle cx={11} cy={2} r={1.5} fill="currentColor" />
        <circle cx={11} cy={5} r={1.5} fill="currentColor" />
        <circle cx={11} cy={8} r={1.5} fill="currentColor" />
        <line
          x1={3.5}
          y1={5}
          x2={9.5}
          y2={2}
          stroke="currentColor"
          strokeWidth={1}
          strokeLinecap="round"
        />
        <line
          x1={3.5}
          y1={5}
          x2={9.5}
          y2={5}
          stroke="currentColor"
          strokeWidth={1}
          strokeLinecap="round"
        />
        <line
          x1={3.5}
          y1={5}
          x2={9.5}
          y2={8}
          stroke="currentColor"
          strokeWidth={1}
          strokeLinecap="round"
        />
      </svg>
    ),
  },
  {
    id: 'TB',
    label: 'Vertical',
    icon: (
      <svg width={10} height={13} viewBox="0 0 10 13" fill="none">
        <circle cx={5} cy={2} r={1.5} fill="currentColor" />
        <circle cx={2} cy={11} r={1.5} fill="currentColor" />
        <circle cx={5} cy={11} r={1.5} fill="currentColor" />
        <circle cx={8} cy={11} r={1.5} fill="currentColor" />
        <line
          x1={5}
          y1={3.5}
          x2={2}
          y2={9.5}
          stroke="currentColor"
          strokeWidth={1}
          strokeLinecap="round"
        />
        <line
          x1={5}
          y1={3.5}
          x2={5}
          y2={9.5}
          stroke="currentColor"
          strokeWidth={1}
          strokeLinecap="round"
        />
        <line
          x1={5}
          y1={3.5}
          x2={8}
          y2={9.5}
          stroke="currentColor"
          strokeWidth={1}
          strokeLinecap="round"
        />
      </svg>
    ),
  },
  {
    id: 'circular',
    label: 'Circular',
    icon: (
      <svg width={12} height={12} viewBox="0 0 12 12" fill="none">
        <circle cx={6} cy={6} r={1.5} fill="currentColor" />
        <circle cx={6} cy={1} r={1.3} fill="currentColor" opacity={0.8} />
        <circle cx={10.3} cy={3.5} r={1.3} fill="currentColor" opacity={0.8} />
        <circle cx={10.3} cy={8.5} r={1.3} fill="currentColor" opacity={0.8} />
        <circle cx={6} cy={11} r={1.3} fill="currentColor" opacity={0.8} />
        <circle cx={1.7} cy={8.5} r={1.3} fill="currentColor" opacity={0.8} />
        <circle cx={1.7} cy={3.5} r={1.3} fill="currentColor" opacity={0.8} />
        <line
          x1={6}
          y1={6}
          x2={6}
          y2={2.3}
          stroke="currentColor"
          strokeWidth={0.8}
          strokeLinecap="round"
        />
        <line
          x1={6}
          y1={6}
          x2={9.3}
          y2={4}
          stroke="currentColor"
          strokeWidth={0.8}
          strokeLinecap="round"
        />
        <line
          x1={6}
          y1={6}
          x2={9.3}
          y2={8}
          stroke="currentColor"
          strokeWidth={0.8}
          strokeLinecap="round"
        />
        <line
          x1={6}
          y1={6}
          x2={6}
          y2={9.7}
          stroke="currentColor"
          strokeWidth={0.8}
          strokeLinecap="round"
        />
        <line
          x1={6}
          y1={6}
          x2={2.7}
          y2={8}
          stroke="currentColor"
          strokeWidth={0.8}
          strokeLinecap="round"
        />
        <line
          x1={6}
          y1={6}
          x2={2.7}
          y2={4}
          stroke="currentColor"
          strokeWidth={0.8}
          strokeLinecap="round"
        />
      </svg>
    ),
  },
];

interface LayoutDropdownProps {
  layout: GraphLayout;
  onSelect: (l: GraphLayout) => void;
}

function LayoutDropdown({ layout, onSelect }: LayoutDropdownProps) {
  const [open, setOpen] = useState(false);
  const current = LAYOUT_OPTIONS.find((o) => o.id === layout)!;

  return (
    <div style={{ position: 'relative' }}>
      <button
        onClick={() => setOpen((o) => !o)}
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 5,
          padding: '4px 9px 4px 8px',
          borderRadius: 7,
          background: 'var(--color-glass-bg)',
          border: '1px solid var(--color-glass-border)',
          cursor: 'pointer',
          color: 'var(--color-muted)',
          fontSize: 10,
          fontWeight: 600,
          backdropFilter: 'blur(8px)',
          WebkitBackdropFilter: 'blur(8px)',
          userSelect: 'none',
        }}
      >
        {current.icon}
        {current.label}
        <svg
          width={9}
          height={9}
          viewBox="0 0 10 10"
          fill="none"
          style={{
            transform: open ? 'rotate(180deg)' : 'none',
            transition: 'transform 0.2s',
            opacity: 0.5,
          }}
        >
          <path
            d="M2 3.5L5 6.5L8 3.5"
            stroke="currentColor"
            strokeWidth={1.5}
            strokeLinecap="round"
            strokeLinejoin="round"
          />
        </svg>
      </button>

      {open && (
        <div
          style={{
            position: 'absolute',
            top: 'calc(100% + 4px)',
            right: 0,
            zIndex: 9999,
            minWidth: 140,
            borderRadius: 9,
            background: 'var(--color-glass-card)',
            backdropFilter: 'blur(20px)',
            WebkitBackdropFilter: 'blur(20px)',
            border: '1px solid var(--color-glass-border)',
            boxShadow: '0 8px 32px rgba(0,0,0,0.5)',
            overflow: 'hidden',
          }}
        >
          {LAYOUT_OPTIONS.map((opt) => {
            const isActive = opt.id === layout;
            return (
              <button
                key={opt.id}
                onClick={() => {
                  onSelect(opt.id);
                  setOpen(false);
                }}
                style={{
                  width: '100%',
                  display: 'flex',
                  alignItems: 'center',
                  gap: 8,
                  padding: '7px 12px',
                  background: isActive ? 'var(--color-glass-hover)' : 'transparent',
                  border: 'none',
                  cursor: 'pointer',
                  color: isActive ? 'var(--color-foreground)' : 'var(--color-muted)',
                  fontSize: 11,
                  fontWeight: isActive ? 700 : 500,
                  textAlign: 'left',
                }}
              >
                {opt.icon}
                <span style={{ flex: 1 }}>{opt.label}</span>
                {isActive && (
                  <svg width={10} height={10} viewBox="0 0 10 10" fill="none">
                    <path
                      d="M2 5L4.5 7.5L8 3"
                      stroke="#34d399"
                      strokeWidth={1.6}
                      strokeLinecap="round"
                      strokeLinejoin="round"
                    />
                  </svg>
                )}
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
}

// ── Main component ────────────────────────────────────────────────────────────

export function BlastRadiusGraph() {
  const [selectedCveId, setSelectedCveId] = useState<string>(BLAST_RADIUS_CVES[0].id);
  const [layout, setLayout] = useState<GraphLayout>('LR');

  const data = BLAST_RADIUS_DATA[selectedCveId];

  const positions = useMemo(() => computeLayout(data.nodes, data.edges, layout), [data, layout]);
  const nextNodes = useMemo(() => buildNodes(data.nodes, data.edges, layout), [data, layout]);
  const nextEdges = useMemo(
    () => buildEdges(data.edges, data.nodes, layout, positions),
    [data, layout, positions],
  );

  const [nodes, setNodes, onNodesChange] = useNodesState(nextNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(nextEdges);

  useEffect(() => {
    setNodes(nextNodes);
  }, [nextNodes, setNodes]);
  useEffect(() => {
    setEdges(nextEdges);
  }, [nextEdges, setEdges]);

  const handleCveChange = useCallback((id: string) => setSelectedCveId(id), []);
  const handleLayoutChange = useCallback((l: GraphLayout) => setLayout(l), []);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', minHeight: 0 }}>
      {/* Header */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          marginBottom: 10,
          flexShrink: 0,
        }}
      >
        <div>
          <h3
            style={{ fontSize: 13, fontWeight: 700, color: 'var(--color-foreground)', margin: 0 }}
          >
            Blast Radius
          </h3>
          <div style={{ fontSize: 10, color: 'var(--color-muted)', marginTop: 2 }}>
            {data.nodes.length - 1} affected groups ·{' '}
            {data.nodes.filter((n) => n.compliance === 'non-compliant').length} non-compliant
          </div>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
          <LayoutDropdown layout={layout} onSelect={handleLayoutChange} />
          <CveDropdown selectedId={selectedCveId} onSelect={handleCveChange} />
        </div>
      </div>

      {/* Legend */}
      <div style={{ marginBottom: 8, flexShrink: 0 }}>
        <Legend />
      </div>

      {/* React Flow canvas */}
      <div
        style={{
          flex: 1,
          borderRadius: 10,
          overflow: 'hidden',
          background: 'var(--color-glass-bg)',
          border: '1px solid var(--color-glass-border)',
          minHeight: 0,
        }}
      >
        <ReactFlow
          key={`${selectedCveId}-${layout}`}
          nodes={nodes}
          edges={edges}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          nodeTypes={nodeTypes}
          fitView
          fitViewOptions={{ padding: 0.2 }}
          minZoom={0.4}
          maxZoom={2}
          proOptions={{ hideAttribution: true }}
          style={{ background: 'transparent' }}
          defaultEdgeOptions={{ type: 'default' }}
        >
          <Background
            variant={BackgroundVariant.Dots}
            gap={20}
            size={1}
            color="var(--color-separator)"
          />
          <Controls
            style={{
              background: 'var(--color-glass-card)',
              border: '1px solid var(--color-glass-border)',
              borderRadius: 8,
              backdropFilter: 'blur(12px)',
            }}
          />
        </ReactFlow>
      </div>
    </div>
  );
}
