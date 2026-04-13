import { useState, useRef } from 'react';
import { useNavigate } from 'react-router';
import { SkeletonCard } from '@patchiq/ui';
import { useLicenseBreakdown } from '../../api/hooks/useDashboard';
import type { LicenseBreakdownItem } from '../../types/dashboard';

const STATUS_COLORS: Record<string, string> = {
  active: '',
  expiring: 'var(--signal-warning)',
  expired: 'var(--signal-critical)',
  revoked: 'var(--text-faint)',
};

function describeArc(
  cx: number,
  cy: number,
  r1: number,
  r2: number,
  startAngle: number,
  endAngle: number,
): string {
  const x1 = cx + r1 * Math.cos(startAngle);
  const y1 = cy + r1 * Math.sin(startAngle);
  const x2 = cx + r2 * Math.cos(startAngle);
  const y2 = cy + r2 * Math.sin(startAngle);
  const x3 = cx + r2 * Math.cos(endAngle);
  const y3 = cy + r2 * Math.sin(endAngle);
  const x4 = cx + r1 * Math.cos(endAngle);
  const y4 = cy + r1 * Math.sin(endAngle);
  const large = endAngle - startAngle > Math.PI ? 1 : 0;
  return `M ${x1} ${y1} L ${x2} ${y2} A ${r2} ${r2} 0 ${large} 1 ${x3} ${y3} L ${x4} ${y4} A ${r1} ${r1} 0 ${large} 0 ${x1} ${y1} Z`;
}

interface TooltipInfo {
  x: number;
  y: number;
  text: string;
}

export const LicenseSunburst = () => {
  const navigate = useNavigate();
  const { data: breakdown, isLoading, isError } = useLicenseBreakdown();
  const [tooltip, setTooltip] = useState<TooltipInfo | null>(null);
  const wrapperRef = useRef<HTMLDivElement>(null);

  const handleMouseEnter = () => {
    if (wrapperRef.current) wrapperRef.current.style.borderColor = 'var(--text-faint)';
  };
  const handleMouseLeave = () => {
    if (wrapperRef.current) wrapperRef.current.style.borderColor = 'var(--border)';
  };

  if (isLoading) return <SkeletonCard className="h-[260px]" />;

  if (isError) {
    return (
      <div
        style={{
          background: 'var(--bg-card)',
          border: '1px solid var(--signal-critical)',
          borderRadius: 8,
          boxShadow: 'var(--shadow-sm)',
          display: 'flex',
          flexDirection: 'column',
        }}
      >
        <div style={{ padding: '16px 20px 0' }}>
          <h3 style={{ fontSize: 13, color: 'var(--text-muted)', fontWeight: 400 }}>
            License Distribution
          </h3>
          <p style={{ fontSize: 11, color: 'var(--text-faint)' }}>Failed to load license data.</p>
        </div>
      </div>
    );
  }

  if (!breakdown?.length) {
    return (
      <div
        style={{
          background: 'var(--bg-card)',
          border: '1px solid var(--border)',
          borderRadius: 8,
          boxShadow: 'var(--shadow-sm)',
          display: 'flex',
          flexDirection: 'column',
        }}
      >
        <div style={{ padding: '16px 20px 0' }}>
          <h3 style={{ fontSize: 13, color: 'var(--text-muted)', fontWeight: 400 }}>
            License Distribution
          </h3>
          <p style={{ fontSize: 11, color: 'var(--text-faint)' }}>No license data available</p>
        </div>
      </div>
    );
  }

  // Group by tier — use accent at varying opacities per tier
  const tierMap = new Map<
    string,
    { count: number; endpoints: number; statuses: LicenseBreakdownItem[] }
  >();
  for (const item of breakdown) {
    const existing = tierMap.get(item.tier) ?? { count: 0, endpoints: 0, statuses: [] };
    existing.count += item.count;
    existing.endpoints += item.total_endpoints;
    existing.statuses.push(item);
    tierMap.set(item.tier, existing);
  }

  const tiers = Array.from(tierMap.entries()).map(([tier, data], idx) => ({
    name: tier.charAt(0).toUpperCase() + tier.slice(1),
    key: tier,
    opacity: 1 - idx * 0.15,
    ...data,
  }));

  const total = tiers.reduce((s, t) => s + t.count, 0);
  const activeCount = breakdown
    .filter((b) => b.status === 'active')
    .reduce((s, b) => s + b.count, 0);

  const cxS = 110,
    cyS = 110,
    r1 = 46,
    r2 = 66,
    r3 = 86;
  let startAngle = -Math.PI / 2;

  const segments: React.ReactNode[] = [];

  for (const tier of tiers) {
    const frac = tier.count / total;
    const sweep = frac * 2 * Math.PI;
    const endAngle = startAngle + sweep;

    segments.push(
      <path
        key={`inner-${tier.key}`}
        d={describeArc(cxS, cyS, r1, r2, startAngle, endAngle)}
        fill="var(--accent)"
        opacity={tier.opacity * 0.85}
        className="transition-opacity hover:opacity-70 cursor-pointer"
        onMouseMove={(e) =>
          setTooltip({
            x: e.clientX + 14,
            y: e.clientY - 10,
            text: `${tier.name}: ${tier.count} licenses, ${tier.endpoints} endpoints`,
          })
        }
        onMouseLeave={() => setTooltip(null)}
      />,
    );

    let subStart = startAngle;
    for (const statusItem of tier.statuses) {
      const subSweep = (statusItem.count / tier.count) * sweep;
      const fillColor = STATUS_COLORS[statusItem.status] || 'var(--accent)';
      segments.push(
        <path
          key={`outer-${tier.key}-${statusItem.status}`}
          d={describeArc(cxS, cyS, r2 + 4, r3, subStart, subStart + subSweep)}
          fill={fillColor || 'var(--accent)'}
          opacity={statusItem.status === 'active' ? tier.opacity : 0.8}
          className="transition-opacity hover:opacity-70 cursor-pointer"
          onMouseMove={(e) =>
            setTooltip({
              x: e.clientX + 14,
              y: e.clientY - 10,
              text: `${tier.name} -- ${statusItem.status}: ${statusItem.count} licenses`,
            })
          }
          onMouseLeave={() => setTooltip(null)}
        />,
      );
      subStart += subSweep;
    }

    startAngle = endAngle;
  }

  return (
    <div
      ref={wrapperRef}
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
      style={{
        background: 'var(--bg-card)',
        border: '1px solid var(--border)',
        borderRadius: 8,
        boxShadow: 'var(--shadow-sm)',
        transition: 'border-color 150ms ease',
        display: 'flex',
        flexDirection: 'column',
        position: 'relative',
      }}
    >
      <div
        style={{ padding: '16px 20px 0', cursor: 'pointer' }}
        onClick={() => void navigate('/licenses')}
      >
        <h3 style={{ fontSize: 13, color: 'var(--text-muted)', fontWeight: 400 }}>
          License Distribution
          <span style={{ fontSize: 10, color: 'var(--text-faint)', marginLeft: 6 }}>→</span>
        </h3>
        <p style={{ fontSize: 11, color: 'var(--text-faint)' }}>Breakdown by tier and status</p>
      </div>
      <div style={{ padding: '12px 20px 16px', flex: 1 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 20 }}>
          {/* Donut chart */}
          <div style={{ position: 'relative', width: 220, height: 220, flexShrink: 0 }}>
            <svg width={220} height={220} viewBox="0 0 220 220">
              {segments}
            </svg>
            <div
              style={{
                position: 'absolute',
                inset: 0,
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                justifyContent: 'center',
                pointerEvents: 'none',
              }}
            >
              <p
                style={{
                  fontSize: 22,
                  fontWeight: 700,
                  color: 'var(--text-emphasis)',
                  fontFamily: 'var(--font-mono)',
                  lineHeight: 1,
                }}
              >
                {activeCount}
              </p>
              <p style={{ fontSize: 11, color: 'var(--text-muted)', marginTop: 2 }}>Active</p>
            </div>
          </div>
          {/* Legend */}
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8, flex: 1, minWidth: 0 }}>
            {tiers.map((t) => (
              <div key={t.key} style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                <span
                  style={{
                    width: 8,
                    height: 8,
                    borderRadius: 2,
                    flexShrink: 0,
                    background: 'var(--accent)',
                    opacity: t.opacity,
                  }}
                />
                <span
                  style={{
                    fontSize: 11,
                    color: 'var(--text-primary)',
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                  }}
                >
                  <b>{t.name}</b> · {t.count} lic · {t.endpoints} ep
                </span>
              </div>
            ))}
            <div
              style={{
                marginTop: 4,
                paddingTop: 8,
                borderTop: '1px solid var(--border)',
                display: 'flex',
                flexDirection: 'column',
                gap: 5,
              }}
            >
              {[
                { color: 'var(--signal-warning)', label: 'Expiring soon' },
                { color: 'var(--signal-critical)', label: 'Expired' },
              ].map((s) => (
                <div
                  key={s.label}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 6,
                    fontSize: 10,
                    color: 'var(--text-faint)',
                  }}
                >
                  <span
                    style={{
                      width: 8,
                      height: 8,
                      borderRadius: '50%',
                      background: s.color,
                      flexShrink: 0,
                    }}
                  />
                  {s.label}
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>

      {tooltip && (
        <div
          style={{
            position: 'fixed',
            zIndex: 50,
            padding: '8px 12px',
            borderRadius: 8,
            fontSize: 12,
            pointerEvents: 'none',
            left: tooltip.x,
            top: tooltip.y,
            background: 'var(--bg-elevated)',
            border: '1px solid var(--border)',
            color: 'var(--text-primary)',
            boxShadow: 'var(--shadow-lg)',
          }}
        >
          {tooltip.text}
        </div>
      )}
    </div>
  );
};
