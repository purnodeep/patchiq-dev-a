import { useMemo, useState, useCallback, useRef } from 'react';
import { useNavigate } from 'react-router';
import { SkeletonCard, EmptyState } from '@patchiq/ui';
import { useClientSummary } from '../../api/hooks/useDashboard';

function getStatusLabel(status: string) {
  switch (status) {
    case 'approved':
      return 'Healthy';
    case 'pending':
      return 'Pending';
    default:
      return 'Error';
  }
}

function getStatusColor(status: string): string {
  switch (status) {
    case 'approved':
      return 'var(--signal-healthy)';
    case 'pending':
      return 'var(--signal-warning)';
    default:
      return 'var(--signal-critical)';
  }
}

function formatSyncTime(ts: string | null): string {
  if (!ts) return 'Never synced';
  const m = Math.floor((Date.now() - new Date(ts).getTime()) / 60_000);
  if (m < 1) return 'Just now';
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  return h < 24 ? `${h}h ago` : `${Math.floor(h / 24)}d ago`;
}

/** Truncate hostname to fit a given radius */
function truncName(name: string, r: number): string {
  const maxChars = Math.max(3, Math.floor(r / 4.5));
  return name.length > maxChars ? name.slice(0, maxChars - 1) + '…' : name;
}

interface TooltipInfo {
  x: number;
  y: number;
  client: {
    hostname: string;
    status: string;
    endpoint_count: number;
    last_sync_at: string | null;
  };
}

const cardStyle: React.CSSProperties = {
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  borderRadius: 8,
  boxShadow: 'var(--shadow-sm)',
  transition: 'border-color 150ms ease',
  display: 'flex',
  flexDirection: 'column',
  position: 'relative',
};

// Layout constants
const VB_W = 900;
const VB_H = 500;
const HUB_CX = VB_W / 2;
const HUB_CY = VB_H / 2;
const HUB_R = 38;
const NODE_R_MIN = 32;
const NODE_R_MAX = 42;
const MIN_GAP = 28;

export const FleetTopology = () => {
  const navigate = useNavigate();
  const { data: clients, isLoading, isError } = useClientSummary();
  const [tooltip, setTooltip] = useState<TooltipInfo | null>(null);
  const cardRef = useRef<HTMLDivElement>(null);

  const nodes = useMemo(() => {
    if (!clients?.length) return [];
    const n = clients.length;
    const maxEp = Math.max(...clients.map((c) => c.endpoint_count), 1);

    // Compute node radii first
    const radii = clients.map(
      (c) => NODE_R_MIN + (c.endpoint_count / maxEp) * (NODE_R_MAX - NODE_R_MIN),
    );

    // Orbit radius: ensure adjacent nodes don't overlap
    // Arc distance between neighbors = 2πR/n must be >= maxNodeDiameter + gap
    const maxR = Math.max(...radii);
    const neededOrbit = (n * (maxR * 2 + MIN_GAP)) / (2 * Math.PI);
    // Max orbit so nodes stay within viewBox with 36px padding on all sides
    const maxOrbit = Math.min(VB_W / 2, VB_H / 2) - maxR - 36;
    const orbitR = Math.max(HUB_R + maxR + MIN_GAP, Math.min(neededOrbit, maxOrbit));

    return clients.map((client, i) => {
      const angle = (i / n) * 2 * Math.PI - Math.PI / 2;
      const nx = HUB_CX + orbitR * Math.cos(angle);
      const ny = HUB_CY + orbitR * Math.sin(angle);
      const nodeR = radii[i];
      const col = getStatusColor(client.status);
      return { ...client, nx, ny, nodeR, col, orbitR };
    });
  }, [clients]);

  const handleMouseMove = useCallback(
    (
      e: React.MouseEvent,
      client: {
        hostname: string;
        status: string;
        endpoint_count: number;
        last_sync_at: string | null;
      },
    ) => {
      setTooltip({ x: e.clientX + 14, y: e.clientY - 10, client });
    },
    [],
  );

  if (isLoading) {
    return <SkeletonCard className="min-h-[380px]" />;
  }

  if (isError) {
    return (
      <div style={{ ...cardStyle, border: '1px solid var(--signal-critical)' }}>
        <div style={{ padding: '16px 20px 0' }}>
          <span style={{ fontSize: 13, color: 'var(--text-muted)', fontWeight: 400 }}>
            Fleet Topology
          </span>
        </div>
        <div style={{ padding: '12px 20px 16px' }}>
          <span style={{ fontSize: 12, color: 'var(--signal-critical)' }}>
            Failed to load client data. Please try refreshing the page.
          </span>
        </div>
      </div>
    );
  }

  if (!clients?.length) {
    return (
      <div style={cardStyle}>
        <div style={{ padding: '16px 20px 0' }}>
          <span style={{ fontSize: 13, color: 'var(--text-muted)', fontWeight: 400 }}>
            Fleet Topology
          </span>
        </div>
        <div
          style={{
            padding: '24px 20px',
            flex: 1,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}
        >
          <EmptyState
            title="No clients connected"
            description="Register a Patch Manager instance to see the fleet topology."
          />
        </div>
      </div>
    );
  }

  const orbitR = nodes[0]?.orbitR ?? 80;

  return (
    <div
      ref={cardRef}
      style={cardStyle}
      onMouseEnter={() => {
        if (cardRef.current) cardRef.current.style.borderColor = 'var(--text-faint)';
      }}
      onMouseLeave={() => {
        if (cardRef.current) cardRef.current.style.borderColor = 'var(--border)';
      }}
    >
      {/* Header */}
      <div
        style={{
          padding: '16px 20px 0',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}
      >
        <div style={{ cursor: 'pointer' }} onClick={() => void navigate('/clients')}>
          <span style={{ fontSize: 13, color: 'var(--text-muted)', fontWeight: 400 }}>
            Fleet Topology <span style={{ fontSize: 10, color: 'var(--text-faint)' }}>→</span>
          </span>
          <p style={{ fontSize: 11, color: 'var(--text-faint)', margin: 0 }}>
            Real-time view of connected Patch Manager instances
          </p>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 14 }}>
          {[
            { label: 'Healthy', color: 'var(--signal-healthy)' },
            { label: 'Pending', color: 'var(--signal-warning)' },
            { label: 'Error', color: 'var(--signal-critical)' },
          ].map((item) => (
            <div
              key={item.label}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 5,
                fontSize: 11,
                color: 'var(--text-muted)',
                fontFamily: 'var(--font-mono)',
              }}
            >
              <span
                style={{
                  width: 7,
                  height: 7,
                  borderRadius: '50%',
                  background: item.color,
                  display: 'inline-block',
                }}
              />
              {item.label}
            </div>
          ))}
        </div>
      </div>

      {/* SVG */}
      <div style={{ padding: '8px 20px 16px', flex: 1 }}>
        <svg
          width="100%"
          viewBox={`0 0 ${VB_W} ${VB_H}`}
          preserveAspectRatio="xMidYMid meet"
          style={{ display: 'block' }}
        >
          {/* Orbit ring — subtle dashed guide */}
          <circle
            cx={HUB_CX}
            cy={HUB_CY}
            r={orbitR}
            fill="none"
            stroke="var(--border)"
            strokeWidth={1}
            strokeDasharray="4 4"
            opacity={0.35}
          />

          {/* Connection lines — gradient from hub to node */}
          {nodes.map((node) => (
            <line
              key={`line-${node.id}`}
              x1={HUB_CX}
              y1={HUB_CY}
              x2={node.nx}
              y2={node.ny}
              stroke={node.col}
              strokeWidth={1}
              opacity={0.25}
            />
          ))}

          {/* Hub center */}
          <circle cx={HUB_CX} cy={HUB_CY} r={HUB_R + 4} fill="var(--accent)" opacity={0.06} />
          <circle
            cx={HUB_CX}
            cy={HUB_CY}
            r={HUB_R}
            fill="var(--bg-card)"
            stroke="var(--accent)"
            strokeWidth={1.5}
          />
          <text
            x={HUB_CX}
            y={HUB_CY - 2}
            textAnchor="middle"
            fill="var(--accent)"
            fontWeight={600}
            fontSize={12}
            fontFamily="var(--font-mono)"
          >
            PatchIQ
          </text>
          <text
            x={HUB_CX}
            y={HUB_CY + 13}
            textAnchor="middle"
            fill="var(--accent)"
            fontWeight={600}
            fontSize={12}
            fontFamily="var(--font-mono)"
          >
            Hub
          </text>

          {/* Client nodes */}
          {nodes.map((node) => (
            <g
              key={node.id}
              transform={`translate(${node.nx},${node.ny})`}
              style={{ cursor: 'pointer' }}
              onClick={() => void navigate(`/clients/${node.id}`)}
              onMouseMove={(e) => handleMouseMove(e, node)}
              onMouseLeave={() => setTooltip(null)}
            >
              {/* Subtle outer ring for healthy nodes */}
              {node.status === 'approved' && (
                <circle r={node.nodeR + 3} fill={node.col} opacity={0.08} />
              )}
              {/* Node circle */}
              <circle r={node.nodeR} fill="var(--bg-card)" stroke={node.col} strokeWidth={1.5} />
              {/* Hostname — truncated to fit */}
              <text
                x={0}
                y={1}
                textAnchor="middle"
                dominantBaseline="middle"
                fill="var(--text-primary)"
                fontSize={11}
                fontWeight={600}
                fontFamily="var(--font-mono)"
              >
                {truncName(node.hostname, node.nodeR)}
              </text>
              {/* Endpoint count — below the circle */}
              <text
                x={0}
                y={node.nodeR + 14}
                textAnchor="middle"
                fill="var(--text-faint)"
                fontSize={10}
                fontFamily="var(--font-mono)"
              >
                {node.endpoint_count} ep
              </text>
            </g>
          ))}
        </svg>
      </div>

      {/* Tooltip */}
      {tooltip && (
        <div
          style={{
            position: 'fixed',
            zIndex: 50,
            padding: '8px 12px',
            borderRadius: 6,
            fontSize: 11,
            pointerEvents: 'none',
            left: tooltip.x,
            top: tooltip.y,
            background: 'var(--bg-elevated)',
            border: '1px solid var(--border)',
            color: 'var(--text-primary)',
            boxShadow: 'var(--shadow-lg)',
            lineHeight: 1.6,
          }}
        >
          <div style={{ fontWeight: 600, marginBottom: 2 }}>{tooltip.client.hostname}</div>
          <div style={{ color: getStatusColor(tooltip.client.status), fontSize: 10 }}>
            {getStatusLabel(tooltip.client.status)}
          </div>
          <div style={{ color: 'var(--text-muted)', fontSize: 10, fontFamily: 'var(--font-mono)' }}>
            {tooltip.client.endpoint_count} endpoints
          </div>
          <div style={{ color: 'var(--text-faint)', fontSize: 10, fontFamily: 'var(--font-mono)' }}>
            Sync: {formatSyncTime(tooltip.client.last_sync_at)}
          </div>
        </div>
      )}
    </div>
  );
};
