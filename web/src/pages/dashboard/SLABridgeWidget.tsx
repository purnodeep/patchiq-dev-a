import { useDashboardData } from './DashboardContext';

const cardStyle: React.CSSProperties = {
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  borderRadius: 8,
  boxShadow: 'var(--shadow-sm)',
  transition: 'border-color 150ms ease',
  display: 'flex',
  flexDirection: 'column',
};

interface SLABridgeWidgetProps {
  startingGap?: number;
  deployed?: number;
  newCves?: number;
}

export function SLABridgeWidget({
  startingGap: startingGapProp,
  deployed: deployedProp,
  newCves: newCvesProp,
}: SLABridgeWidgetProps = {}) {
  const ctx = useDashboardData();
  const startingGap = startingGapProp ?? ctx.overdue_sla_count ?? 0;
  const deployed = deployedProp ?? ctx.active_deployments?.length ?? 0;
  const newCves = newCvesProp ?? ctx.unpatched_cves ?? 0;

  const startGap = Math.max(0, isNaN(startingGap) ? 0 : startingGap);
  const dep = Math.max(0, isNaN(deployed) ? 0 : deployed);
  const cves = Math.max(0, isNaN(newCves) ? 0 : newCves);
  const projected = Math.max(0, startGap - dep) + cves;
  const delta = projected - startGap;

  // Bar segments: deployed (green, reduces gap), remaining (red, still open), new cves (red, added)
  const totalBar = Math.max(1, startGap + cves);
  const depPct = (Math.min(dep, startGap) / totalBar) * 100;
  const remainPct = (Math.max(0, startGap - dep) / totalBar) * 100;
  const cvesPct = (cves / totalBar) * 100;

  const trendColor =
    delta < 0 ? 'var(--accent)' : delta > 0 ? 'var(--signal-critical)' : 'var(--text-muted)';
  const trendSymbol = delta < 0 ? '▼' : delta > 0 ? '▲' : '→';
  const trendLabel =
    delta < 0 ? `${Math.abs(delta)} improved` : delta > 0 ? `+${delta} worse` : 'No change';

  return (
    <div
      style={cardStyle}
      onMouseEnter={(e) => {
        (e.currentTarget as HTMLDivElement).style.borderColor = 'var(--text-faint)';
      }}
      onMouseLeave={(e) => {
        (e.currentTarget as HTMLDivElement).style.borderColor = 'var(--border)';
      }}
    >
      <div
        style={{
          padding: '12px 18px 0',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}
      >
        <span style={{ fontSize: 13, color: 'var(--text-muted)', fontWeight: 400 }}>
          SLA Bridge
        </span>
        <span
          style={{
            fontSize: 11,
            color: trendColor,
            fontFamily: 'var(--font-mono)',
            display: 'inline-flex',
            alignItems: 'center',
            gap: 4,
          }}
        >
          <span style={{ fontSize: 9 }}>{trendSymbol}</span>
          {trendLabel}
        </span>
      </div>

      <div
        style={{
          padding: '8px 18px 14px',
          flex: 1,
          minWidth: 0,
          display: 'flex',
          alignItems: 'center',
          gap: 20,
        }}
      >
        {/* Left: Big before/after comparison */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 10,
            flexShrink: 0,
          }}
        >
          <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
            <span
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 32,
                fontWeight: 700,
                lineHeight: 1,
                color: 'var(--text-muted)',
              }}
            >
              {startGap}
            </span>
            <span
              style={{
                fontSize: 9,
                color: 'var(--text-faint)',
                textTransform: 'uppercase',
                letterSpacing: '0.06em',
                marginTop: 4,
              }}
            >
              Start
            </span>
          </div>
          <span
            style={{
              fontSize: 18,
              color: 'var(--text-faint)',
              fontFamily: 'var(--font-mono)',
              lineHeight: 1,
            }}
          >
            →
          </span>
          <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
            <span
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 32,
                fontWeight: 700,
                lineHeight: 1,
                color: trendColor,
              }}
            >
              {projected}
            </span>
            <span
              style={{
                fontSize: 9,
                color: 'var(--text-faint)',
                textTransform: 'uppercase',
                letterSpacing: '0.06em',
                marginTop: 4,
              }}
            >
              Projected
            </span>
          </div>
        </div>

        {/* Right: Stacked bar breakdown */}
        <div
          style={{
            flex: 1,
            minWidth: 0,
            display: 'flex',
            flexDirection: 'column',
            gap: 6,
          }}
        >
          <div
            style={{
              display: 'flex',
              width: '100%',
              height: 10,
              borderRadius: 5,
              overflow: 'hidden',
              background: 'var(--border)',
            }}
          >
            {depPct > 0 && (
              <div
                title={`${dep} deployed`}
                style={{ width: `${depPct}%`, background: 'var(--accent)' }}
              />
            )}
            {remainPct > 0 && (
              <div
                title={`${Math.max(0, startGap - dep)} still open`}
                style={{
                  width: `${remainPct}%`,
                  background: 'var(--signal-critical)',
                  opacity: 0.85,
                }}
              />
            )}
            {cvesPct > 0 && (
              <div
                title={`${cves} new CVEs`}
                style={{
                  width: `${cvesPct}%`,
                  background: 'var(--signal-critical)',
                  opacity: 0.45,
                }}
              />
            )}
          </div>
          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              gap: 8,
              fontSize: 10,
              fontFamily: 'var(--font-sans)',
              color: 'var(--text-muted)',
              whiteSpace: 'nowrap',
            }}
          >
            <LegendItem color="var(--accent)" label="Deployed" value={dep} />
            <LegendItem
              color="var(--signal-critical)"
              label="Remaining"
              value={Math.max(0, startGap - dep)}
              opacity={0.85}
            />
            <LegendItem
              color="var(--signal-critical)"
              label="New CVEs"
              value={cves}
              opacity={0.45}
            />
          </div>
        </div>
      </div>
    </div>
  );
}

function LegendItem({
  color,
  label,
  value,
  opacity = 1,
}: {
  color: string;
  label: string;
  value: number;
  opacity?: number;
}) {
  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 5,
        minWidth: 0,
      }}
    >
      <span
        style={{
          display: 'inline-block',
          width: 7,
          height: 7,
          borderRadius: 2,
          background: color,
          opacity,
          flexShrink: 0,
        }}
      />
      <span style={{ color: 'var(--text-muted)' }}>{label}</span>
      <span
        style={{
          color: 'var(--text-emphasis)',
          fontFamily: 'var(--font-mono)',
          fontWeight: 600,
        }}
      >
        {value}
      </span>
    </div>
  );
}
