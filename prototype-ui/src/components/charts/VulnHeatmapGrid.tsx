import { useState } from 'react';
import { useNavigate } from 'react-router';
import { VULN_HEATMAP_DATA, type HeatmapEndpoint } from '@/data/mock-data';

// ── Risk color mapping ─────────────────────────────────────────────────────────
function riskColor(risk: number): string {
  if (risk > 80) return '#ff3b30'; // critical — dark red
  if (risk > 60) return '#ff6b35'; // high — orange-red
  if (risk > 40) return '#ff9500'; // medium — amber
  return '#34c759'; // low — green
}

function riskLabel(risk: number): string {
  if (risk > 80) return 'Critical';
  if (risk > 60) return 'High';
  if (risk > 40) return 'Medium';
  return 'Low';
}

function riskLabelColor(risk: number): string {
  if (risk > 80) return '#ff3b30';
  if (risk > 60) return '#ff6b35';
  if (risk > 40) return '#ff9500';
  return '#34c759';
}

// ── Group endpoints by group key ───────────────────────────────────────────────
function groupEndpoints(data: HeatmapEndpoint[]): Map<string, HeatmapEndpoint[]> {
  const map = new Map<string, HeatmapEndpoint[]>();
  for (const ep of data) {
    if (!map.has(ep.group)) map.set(ep.group, []);
    map.get(ep.group)!.push(ep);
  }
  return map;
}

// ── Tooltip ────────────────────────────────────────────────────────────────────
interface TooltipData {
  endpoint: HeatmapEndpoint;
  x: number;
  y: number;
}

// ── Cell ───────────────────────────────────────────────────────────────────────
interface CellProps {
  endpoint: HeatmapEndpoint;
  index: number;
  groupIndex: number;
  onHover: (data: TooltipData | null) => void;
  onClick: () => void;
}

function HeatCell({ endpoint, index, groupIndex, onHover, onClick }: CellProps) {
  const color = riskColor(endpoint.risk);
  const [pressed, setPressed] = useState(false);

  return (
    <div
      onClick={() => {
        setPressed(true);
        setTimeout(() => setPressed(false), 200);
        onClick();
      }}
      onMouseEnter={(e) => {
        const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
        const container = (e.currentTarget as HTMLElement).closest('[data-heatmap-root]');
        if (!container) return;
        const cRect = container.getBoundingClientRect();
        onHover({
          endpoint,
          x: rect.left - cRect.left + rect.width / 2,
          y: rect.top - cRect.top,
        });
      }}
      onMouseLeave={() => onHover(null)}
      style={{
        width: 28,
        height: 28,
        borderRadius: 5,
        background: color,
        opacity: endpoint.risk < 10 ? 0.3 : 0.75 + endpoint.risk * 0.0025,
        cursor: 'pointer',
        flexShrink: 0,
        position: 'relative',
        transition: 'transform 0.12s ease, box-shadow 0.12s ease, opacity 0.12s ease',
        transform: pressed ? 'scale(0.88)' : 'scale(1)',
        boxShadow: pressed ? `0 0 0 2px ${color}80, 0 0 12px ${color}60` : `0 1px 3px ${color}30`,
        animation: `fade-in-up 0.35s ease both`,
        animationDelay: `${(groupIndex * 7 + index) * 25}ms`,
      }}
      onMouseOver={(e) => {
        if (!pressed) {
          (e.currentTarget as HTMLElement).style.transform = 'scale(1.15)';
          (e.currentTarget as HTMLElement).style.boxShadow =
            `0 0 0 2px ${color}60, 0 4px 12px ${color}50`;
          (e.currentTarget as HTMLElement).style.opacity = '1';
          (e.currentTarget as HTMLElement).style.zIndex = '10';
        }
      }}
      onMouseOut={(e) => {
        if (!pressed) {
          (e.currentTarget as HTMLElement).style.transform = 'scale(1)';
          (e.currentTarget as HTMLElement).style.boxShadow = `0 1px 3px ${color}30`;
          (e.currentTarget as HTMLElement).style.opacity = String(
            endpoint.risk < 10 ? 0.3 : 0.75 + endpoint.risk * 0.0025,
          );
          (e.currentTarget as HTMLElement).style.zIndex = 'auto';
        }
      }}
      title={`${endpoint.name} — Risk: ${endpoint.risk} | ${endpoint.cveCount} CVEs`}
    />
  );
}

// ── Main Component ─────────────────────────────────────────────────────────────
export function VulnHeatmapGrid() {
  const navigate = useNavigate();
  const [tooltip, setTooltip] = useState<TooltipData | null>(null);

  const groups = groupEndpoints(VULN_HEATMAP_DATA);

  const LEGEND_ITEMS = [
    { label: 'Critical (>80)', color: '#ff3b30' },
    { label: 'High (60–80)', color: '#ff6b35' },
    { label: 'Medium (40–60)', color: '#ff9500' },
    { label: 'Low (<40)', color: '#34c759' },
  ];

  let groupIdx = 0;

  return (
    <div
      data-heatmap-root
      style={{ position: 'relative', userSelect: 'none', height: '100%', overflowY: 'auto' }}
    >
      {/* Groups */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
        {Array.from(groups.entries()).map(([groupName, endpoints]) => {
          const currentGroupIdx = groupIdx;
          groupIdx++;

          // Compute group summary
          const critCount = endpoints.filter((e) => e.risk > 80).length;
          const avgRisk = Math.round(endpoints.reduce((s, e) => s + e.risk, 0) / endpoints.length);

          return (
            <div key={groupName}>
              {/* Group label row */}
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  marginBottom: 7,
                }}
              >
                <div style={{ display: 'flex', alignItems: 'center', gap: 7 }}>
                  <span
                    style={{
                      fontSize: 10,
                      fontWeight: 700,
                      color: 'var(--color-muted)',
                      letterSpacing: '0.06em',
                      textTransform: 'uppercase',
                    }}
                  >
                    {groupName}
                  </span>
                  <span
                    style={{
                      fontSize: 9,
                      color: 'var(--color-subtle)',
                      background: 'var(--color-separator)',
                      borderRadius: 999,
                      padding: '1px 5px',
                    }}
                  >
                    {endpoints.length}
                  </span>
                </div>
                <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                  {critCount > 0 && (
                    <span
                      style={{
                        fontSize: 9,
                        fontWeight: 700,
                        color: '#ff3b30',
                        background: '#ff3b3015',
                        border: '1px solid #ff3b3025',
                        borderRadius: 999,
                        padding: '1px 6px',
                      }}
                    >
                      {critCount} critical
                    </span>
                  )}
                  <span
                    style={{
                      fontSize: 9,
                      color: riskLabelColor(avgRisk),
                      fontWeight: 600,
                    }}
                  >
                    avg {avgRisk}
                  </span>
                </div>
              </div>

              {/* Cells row */}
              <div
                style={{
                  display: 'flex',
                  flexWrap: 'wrap',
                  gap: 5,
                }}
              >
                {endpoints.map((ep, i) => (
                  <HeatCell
                    key={ep.id}
                    endpoint={ep}
                    index={i}
                    groupIndex={currentGroupIdx}
                    onHover={setTooltip}
                    onClick={() => navigate('/pm/endpoints')}
                  />
                ))}
              </div>
            </div>
          );
        })}
      </div>

      {/* Tooltip */}
      {tooltip && (
        <div
          className="glass-card"
          style={{
            position: 'absolute',
            left: tooltip.x,
            top: tooltip.y < 120 ? tooltip.y + 38 : tooltip.y - 88,
            transform: 'translateX(-50%)',
            pointerEvents: 'none',
            padding: '8px 12px',
            zIndex: 50,
            minWidth: 158,
            maxHeight: 'none',
            animation: 'fade-in-up 0.12s ease both',
          }}
        >
          {/* Arrow — points down when above cell, up when below cell */}
          <div
            style={{
              position: 'absolute',
              ...(tooltip.y < 120 ? { top: -5, bottom: 'auto' } : { bottom: -5, top: 'auto' }),
              left: '50%',
              transform: 'translateX(-50%)',
              width: 10,
              height: 10,
              background: 'var(--color-glass-card)',
              border: '1px solid var(--color-glass-border)',
              clipPath:
                tooltip.y < 120
                  ? 'polygon(50% 0, 100% 100%, 0 100%)'
                  : 'polygon(0 0, 100% 0, 50% 100%)',
            }}
          />
          <div
            style={{
              fontSize: 11,
              fontWeight: 700,
              color: 'var(--color-foreground)',
              marginBottom: 5,
              whiteSpace: 'nowrap',
            }}
          >
            {tooltip.endpoint.name}
          </div>
          <div style={{ display: 'flex', gap: 12 }}>
            <div>
              <div style={{ fontSize: 9, color: 'var(--color-muted)', marginBottom: 1 }}>
                Risk Score
              </div>
              <div
                style={{
                  fontSize: 14,
                  fontWeight: 700,
                  color: riskColor(tooltip.endpoint.risk),
                  lineHeight: 1,
                }}
              >
                {tooltip.endpoint.risk}
              </div>
              <div
                style={{
                  fontSize: 9,
                  fontWeight: 600,
                  color: riskColor(tooltip.endpoint.risk),
                  marginTop: 2,
                }}
              >
                {riskLabel(tooltip.endpoint.risk)}
              </div>
            </div>
            <div
              style={{
                width: 1,
                background: 'var(--color-separator)',
                alignSelf: 'stretch',
              }}
            />
            <div>
              <div style={{ fontSize: 9, color: 'var(--color-muted)', marginBottom: 1 }}>CVEs</div>
              <div
                style={{
                  fontSize: 14,
                  fontWeight: 700,
                  color: 'var(--color-foreground)',
                  lineHeight: 1,
                }}
              >
                {tooltip.endpoint.cveCount}
              </div>
              <div style={{ fontSize: 9, color: 'var(--color-muted)', marginTop: 2 }}>active</div>
            </div>
          </div>
          <div
            style={{
              marginTop: 6,
              paddingTop: 6,
              borderTop: '1px solid var(--color-separator)',
              fontSize: 9,
              color: 'var(--color-muted)',
            }}
          >
            Click to view endpoint
          </div>
        </div>
      )}

      {/* Color legend */}
      <div
        style={{
          display: 'flex',
          gap: 10,
          flexWrap: 'wrap',
          marginTop: 14,
          paddingTop: 10,
          borderTop: '1px solid var(--color-separator)',
          position: 'sticky',
          bottom: 0,
          background: 'var(--color-glass-card)',
        }}
      >
        {LEGEND_ITEMS.map(({ label, color }) => (
          <div key={label} style={{ display: 'flex', alignItems: 'center', gap: 5 }}>
            <div
              style={{
                width: 12,
                height: 12,
                borderRadius: 3,
                background: color,
                opacity: 0.85,
              }}
            />
            <span style={{ fontSize: 10, color: 'var(--color-muted)', fontWeight: 500 }}>
              {label}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}
