import { useRef, useEffect, useState, useCallback } from 'react';
import { SkeletonCard, ErrorState } from '@patchiq/ui';
import { useAttackPaths } from '@/api/hooks/useDashboard';
import type { AttackPathNode, AttackPathEdge } from '@/api/hooks/useDashboard';

interface SimNode {
  id: string;
  x: number;
  y: number;
  vx: number;
  vy: number;
  radius: number;
  color: string;
  node: AttackPathNode;
}

interface TooltipInfo {
  x: number;
  y: number;
  node: AttackPathNode;
}

function nodeColor(n: AttackPathNode): string {
  if (n.critical_count > 0) return getComputedStyle(document.documentElement).getPropertyValue('--signal-critical').trim() || '#ef4444';
  return getComputedStyle(document.documentElement).getPropertyValue('--signal-warning').trim() || '#f97316';
}

function nodeRadius(n: AttackPathNode): number {
  return Math.max(6, Math.min(24, 6 + (n.critical_count + n.high_count) * 1.5));
}

export function AttackPathHeatmap() {
  const { data, isLoading, error, refetch } = useAttackPaths();
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const [tooltip, setTooltip] = useState<TooltipInfo | null>(null);
  const simNodesRef = useRef<SimNode[]>([]);
  const edgesRef = useRef<AttackPathEdge[]>([]);
  const animRef = useRef<number>(0);

  const draw = useCallback(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;
    const w = canvas.width;
    const h = canvas.height;
    const nodes = simNodesRef.current;
    const edges = edgesRef.current;

    ctx.clearRect(0, 0, w, h);

    // Edges
    const nodeMap = new Map(nodes.map((n) => [n.id, n]));
    for (const edge of edges) {
      const src = nodeMap.get(edge.source_id);
      const tgt = nodeMap.get(edge.target_id);
      if (!src || !tgt) continue;
      ctx.beginPath();
      ctx.moveTo(src.x, src.y);
      ctx.lineTo(tgt.x, tgt.y);
      ctx.strokeStyle = getComputedStyle(document.documentElement).getPropertyValue('--border').trim() || '#374151';
      ctx.lineWidth = Math.max(1, Math.min(4, edge.shared_cve_count));
      ctx.globalAlpha = 0.4;
      ctx.stroke();
      ctx.globalAlpha = 1;
    }

    // Nodes
    for (const node of nodes) {
      ctx.beginPath();
      ctx.arc(node.x, node.y, node.radius, 0, Math.PI * 2);
      ctx.fillStyle = node.color;
      ctx.globalAlpha = 0.7;
      ctx.fill();
      ctx.globalAlpha = 1;
      ctx.strokeStyle = node.color;
      ctx.lineWidth = 1.5;
      ctx.stroke();
    }
  }, []);

  useEffect(() => {
    if (!data || data.nodes.length === 0) return;

    const canvas = canvasRef.current;
    const container = containerRef.current;
    if (!canvas || !container) return;

    const rect = container.getBoundingClientRect();
    const w = Math.max(rect.width, 200);
    const h = Math.max(rect.height - 60, 150);
    canvas.width = w;
    canvas.height = h;

    const cx = w / 2;
    const cy = h / 2;

    // Initialize nodes in a circle
    const simNodes: SimNode[] = data.nodes.map((n, i) => {
      const angle = (2 * Math.PI * i) / data.nodes.length;
      const spread = Math.min(w, h) * 0.3;
      return {
        id: n.id,
        x: cx + spread * Math.cos(angle) + (Math.random() - 0.5) * 20,
        y: cy + spread * Math.sin(angle) + (Math.random() - 0.5) * 20,
        vx: 0,
        vy: 0,
        radius: nodeRadius(n),
        color: nodeColor(n),
        node: n,
      };
    });

    simNodesRef.current = simNodes;
    edgesRef.current = data.edges;

    const edgeMap = new Map<string, string[]>();
    for (const edge of data.edges) {
      if (!edgeMap.has(edge.source_id)) edgeMap.set(edge.source_id, []);
      if (!edgeMap.has(edge.target_id)) edgeMap.set(edge.target_id, []);
      edgeMap.get(edge.source_id)!.push(edge.target_id);
      edgeMap.get(edge.target_id)!.push(edge.source_id);
    }

    let tick = 0;
    const maxTicks = 100;

    function simulate() {
      if (tick >= maxTicks) {
        draw();
        return;
      }
      tick++;

      const alpha = 1 - tick / maxTicks;

      // Repulsion
      for (let i = 0; i < simNodes.length; i++) {
        for (let j = i + 1; j < simNodes.length; j++) {
          const dx = simNodes[j].x - simNodes[i].x;
          const dy = simNodes[j].y - simNodes[i].y;
          const dist = Math.max(Math.sqrt(dx * dx + dy * dy), 1);
          const force = (800 * alpha) / (dist * dist);
          const fx = (dx / dist) * force;
          const fy = (dy / dist) * force;
          simNodes[i].vx -= fx;
          simNodes[i].vy -= fy;
          simNodes[j].vx += fx;
          simNodes[j].vy += fy;
        }
      }

      // Edge attraction
      const nodeIdx = new Map(simNodes.map((n, i) => [n.id, i]));
      for (const edge of data!.edges) {
        const si = nodeIdx.get(edge.source_id);
        const ti = nodeIdx.get(edge.target_id);
        if (si === undefined || ti === undefined) continue;
        const s = simNodes[si];
        const t = simNodes[ti];
        const dx = t.x - s.x;
        const dy = t.y - s.y;
        const dist = Math.max(Math.sqrt(dx * dx + dy * dy), 1);
        const force = (dist - 80) * 0.02 * alpha;
        const fx = (dx / dist) * force;
        const fy = (dy / dist) * force;
        s.vx += fx;
        s.vy += fy;
        t.vx -= fx;
        t.vy -= fy;
      }

      // Gravity toward center
      for (const node of simNodes) {
        node.vx += (cx - node.x) * 0.005 * alpha;
        node.vy += (cy - node.y) * 0.005 * alpha;
      }

      // Apply velocity with damping
      for (const node of simNodes) {
        node.vx *= 0.6;
        node.vy *= 0.6;
        node.x += node.vx;
        node.y += node.vy;
        // Clamp to bounds
        node.x = Math.max(node.radius, Math.min(w - node.radius, node.x));
        node.y = Math.max(node.radius, Math.min(h - node.radius, node.y));
      }

      draw();
      animRef.current = requestAnimationFrame(simulate);
    }

    animRef.current = requestAnimationFrame(simulate);

    return () => {
      if (animRef.current) cancelAnimationFrame(animRef.current);
    };
  }, [data, draw]);

  const handleMouseMove = useCallback(
    (e: React.MouseEvent<HTMLCanvasElement>) => {
      const canvas = canvasRef.current;
      if (!canvas) return;
      const rect = canvas.getBoundingClientRect();
      const mx = e.clientX - rect.left;
      const my = e.clientY - rect.top;

      for (const node of simNodesRef.current) {
        const dx = mx - node.x;
        const dy = my - node.y;
        if (dx * dx + dy * dy < node.radius * node.radius) {
          setTooltip({ x: e.clientX - rect.left, y: e.clientY - rect.top, node: node.node });
          return;
        }
      }
      setTooltip(null);
    },
    [],
  );

  if (isLoading)
    return (
      <div
        className="h-full rounded-lg border p-4"
        style={{ background: 'var(--bg-card)', borderColor: 'var(--border)' }}
      >
        <SkeletonCard lines={5} />
      </div>
    );
  if (error)
    return (
      <div
        className="h-full rounded-lg border p-4"
        style={{ background: 'var(--bg-card)', borderColor: 'var(--border)' }}
      >
        <ErrorState message="Failed to load attack paths" onRetry={() => refetch()} />
      </div>
    );
  if (!data || data.nodes.length === 0)
    return (
      <div
        className="flex h-full items-center justify-center rounded-lg border"
        style={{
          background: 'var(--bg-card)',
          borderColor: 'var(--border)',
          color: 'var(--text-muted)',
        }}
      >
        No attack paths detected
      </div>
    );

  return (
    <div
      ref={containerRef}
      className="flex flex-col overflow-hidden rounded-lg border"
      style={{
        background: 'var(--bg-card)',
        borderColor: 'var(--border)',
        boxShadow: 'var(--shadow-sm)',
        height: '100%',
      }}
    >
      <div style={{ padding: '16px 16px 8px', flexShrink: 0 }}>
        <h3 className="text-sm font-semibold" style={{ color: 'var(--text-emphasis)' }}>
          Attack Path Heatmap
        </h3>
        <p className="text-xs" style={{ color: 'var(--text-secondary)' }}>
          Endpoints sharing vulnerabilities
        </p>
      </div>
      <div style={{ flex: 1, minHeight: 0, position: 'relative', padding: '0 16px 16px' }}>
        <canvas
          ref={canvasRef}
          style={{ width: '100%', height: '100%', display: 'block' }}
          onMouseMove={handleMouseMove}
          onMouseLeave={() => setTooltip(null)}
        />
        {tooltip && (
          <div
            style={{
              position: 'absolute',
              left: Math.min(tooltip.x + 12, 220),
              top: tooltip.y - 8,
              background: 'var(--bg-card)',
              border: '1px solid var(--border)',
              borderRadius: 6,
              padding: '8px 10px',
              pointerEvents: 'none',
              zIndex: 10,
              fontSize: 10,
              fontFamily: 'var(--font-mono)',
              color: 'var(--text-secondary)',
              whiteSpace: 'nowrap',
            }}
          >
            <div style={{ fontWeight: 600, color: 'var(--text-emphasis)', marginBottom: 2 }}>
              {tooltip.node.hostname}
            </div>
            <div>{tooltip.node.os}</div>
            <div>
              Critical: {tooltip.node.critical_count} | High: {tooltip.node.high_count}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
