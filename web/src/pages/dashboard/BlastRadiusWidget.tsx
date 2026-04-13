import { SkeletonCard, ErrorState } from '@patchiq/ui';
import { useNavigate } from 'react-router';
import { useBlastRadius } from '@/api/hooks/useDashboard';

export function BlastRadiusWidget() {
  const { data, isLoading, error, refetch } = useBlastRadius();
  const navigate = useNavigate();

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
        <ErrorState message="Failed to load blast radius" onRetry={() => refetch()} />
      </div>
    );
  if (!data?.cve)
    return (
      <div
        className="flex h-full items-center justify-center rounded-lg border"
        style={{
          background: 'var(--bg-card)',
          borderColor: 'var(--border)',
          color: 'var(--text-muted)',
        }}
      >
        No CVE data available
      </div>
    );

  const cx = 260,
    cy = 170;
  const groups = data.groups;
  const angleStep = (2 * Math.PI) / Math.max(groups.length, 1);
  const radius = 120;

  return (
    <div
      className="flex flex-col overflow-hidden rounded-lg border"
      style={{
        background: 'var(--bg-card)',
        borderColor: 'var(--border)',
        boxShadow: 'var(--shadow-sm)',
      }}
    >
      <div
        style={{
          padding: '16px 16px 8px',
          flexShrink: 0,
          flexGrow: 0,
          flexBasis: 'auto',
          maxHeight: 80,
        }}
      >
        <h3 className="text-sm font-semibold" style={{ color: 'var(--text-emphasis)' }}>
          Active Blast Radius
        </h3>
        <p className="text-xs" style={{ color: 'var(--text-secondary)' }}>
          Endpoints affected by selected CVE
        </p>
      </div>
      <div style={{ flex: 1, minHeight: 0, padding: '0 16px 16px' }}>
        <svg
          viewBox="0 0 520 340"
          style={{ width: '100%', height: '100%' }}
          preserveAspectRatio="xMidYMid meet"
        >
          {groups.map((_g, i) => {
            const angle = angleStep * i - Math.PI / 2;
            const gx = cx + radius * Math.cos(angle);
            const gy = cy + radius * Math.sin(angle);
            return (
              <line
                key={`line-${i}`}
                x1={cx}
                y1={cy}
                x2={gx}
                y2={gy}
                stroke="var(--accent)"
                strokeWidth={2}
                strokeOpacity={0.3}
                strokeDasharray="4 2"
              />
            );
          })}
          <circle
            cx={cx}
            cy={cy}
            r={44}
            fill="var(--signal-critical-subtle)"
            stroke="var(--signal-critical)"
            strokeWidth={2.5}
          />
          <text
            x={cx}
            y={cy - 8}
            textAnchor="middle"
            fill="var(--signal-critical)"
            fontSize={10}
            fontWeight="bold"
            fontFamily="var(--font-mono)"
          >
            {data.cve.cve_id}
          </text>
          <text
            x={cx}
            y={cy + 6}
            textAnchor="middle"
            fill="var(--signal-critical)"
            fontSize={9}
            fontFamily="var(--font-mono)"
          >
            CVSS {data.cve.cvss}
          </text>
          <text
            x={cx}
            y={cy + 18}
            textAnchor="middle"
            fill="var(--text-muted)"
            fontSize={8}
            fontFamily="var(--font-mono)"
          >
            {data.cve.affected_count} affected
          </text>
          {groups.map((g, i) => {
            const angle = angleStep * i - Math.PI / 2;
            const gx = cx + radius * Math.cos(angle);
            const gy = cy + radius * Math.sin(angle);
            const r = Math.max(18, Math.min(36, 12 + g.host_count * 0.8));
            return (
              <g
                key={`node-${i}`}
                style={{ cursor: 'pointer' }}
                onClick={() => void navigate(`/endpoints?os=${encodeURIComponent(g.os)}`)}
              >
                <circle
                  cx={gx}
                  cy={gy}
                  r={r}
                  fill="var(--accent-subtle)"
                  stroke="var(--accent)"
                  strokeWidth={2}
                />
                <text
                  x={gx}
                  y={gy - 4}
                  textAnchor="middle"
                  fontSize={9}
                  fontWeight={500}
                  fill="var(--accent)"
                  fontFamily="var(--font-sans)"
                >
                  {g.name}
                </text>
                <text
                  x={gx}
                  y={gy + 8}
                  textAnchor="middle"
                  fontSize={8}
                  fill="var(--text-muted)"
                  fontFamily="var(--font-mono)"
                >
                  {g.host_count} hosts
                </text>
              </g>
            );
          })}
        </svg>
      </div>
    </div>
  );
}
