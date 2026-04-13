import { useNavigate } from 'react-router';
import type { DashboardSummary } from '@/api/hooks/useDashboard';

interface StatCard {
  label: string;
  value: string;
  critical?: boolean;
  subtitle: string;
  trend?: { direction: 'up' | 'down' | 'flat'; text: string; good: boolean; muted?: boolean };
  navigateTo?: string;
}

function buildCards(data: DashboardSummary): StatCard[] {
  const activeCount = Array.isArray(data.active_deployments) ? data.active_deployments.length : 0;

  const activeEndpoints = data.active_endpoints ?? 0;
  const totalEndpoints = data.total_endpoints ?? 0;
  const offlineCount = totalEndpoints - activeEndpoints;

  return [
    {
      label: 'Total Endpoints',
      value: totalEndpoints.toLocaleString(),
      subtitle: `${activeEndpoints} online`,
      navigateTo: '/endpoints',
      // Up = more online = good; down = more offline = bad
      trend:
        offlineCount > 0
          ? { direction: 'down', text: `${offlineCount} offline`, good: false }
          : { direction: 'up', text: `${activeEndpoints} online`, good: true },
    },
    {
      label: 'Critical Patches',
      value: String(data.critical_patches ?? 0),
      critical: true,
      subtitle: 'require action',
      navigateTo: '/patches?severity=critical',
      // More critical patches = bad (up arrow = bad here)
      trend:
        data.critical_patches > 0
          ? { direction: 'up', text: `${data.critical_patches} unresolved`, good: false }
          : { direction: 'flat', text: 'none pending', good: true },
    },
    {
      label: 'Compliance',
      value: `${(data.compliance_rate ?? 0).toFixed(1)}%`,
      subtitle: 'across frameworks',
      navigateTo: '/compliance',
      trend:
        data.compliance_rate >= 90
          ? { direction: 'up', text: `${data.framework_count ?? 0} frameworks`, good: true }
          : data.compliance_rate >= 70
            ? { direction: 'flat', text: `${data.framework_count ?? 0} frameworks`, good: false }
            : { direction: 'down', text: 'below threshold', good: false },
    },
    {
      label: 'Active Deployments',
      value: String(activeCount),
      subtitle: 'in progress',
      navigateTo: '/deployments',
      trend:
        activeCount > 0
          ? { direction: 'flat', text: `${activeCount} in progress`, good: true }
          : { direction: 'flat', text: 'none active', good: true },
    },
    {
      label: 'Mean Patch Time',
      value: '\u2014',
      subtitle: 'avg remediation',
      trend: {
        direction: 'flat',
        text: 'No data — deploy patches to track',
        good: true,
        muted: true,
      },
    },
  ];
}

const cardStyle: React.CSSProperties = {
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  borderRadius: 8,
  padding: '18px 20px 16px',
  boxShadow: 'var(--shadow-sm)',
  transition: 'border-color 150ms ease',
  cursor: 'default',
};

const labelStyle: React.CSSProperties = {
  fontFamily: 'var(--font-mono)',
  fontSize: 10,
  fontWeight: 500,
  letterSpacing: '0.06em',
  color: 'var(--text-muted)',
  textTransform: 'uppercase',
  marginBottom: 10,
};

const numberStyle: React.CSSProperties = {
  fontFamily: 'var(--font-mono)',
  fontSize: 32,
  fontWeight: 700,
  color: 'var(--text-emphasis)',
  letterSpacing: '-0.03em',
  lineHeight: 1,
  marginBottom: 10,
};

const subtextStyle: React.CSSProperties = {
  fontSize: 12,
  color: 'var(--text-muted)',
};

const arrowChars: Record<string, string> = {
  up: '\u2191',
  down: '\u2193',
  flat: '\u2192',
};

export function StatCardsRow({ data }: { data: DashboardSummary }) {
  const cards = buildCards(data);
  const navigate = useNavigate();

  return (
    <div
      style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(5, 1fr)',
        gap: 12,
        marginBottom: 12,
      }}
    >
      {cards.map((card) => {
        const trendColor = card.trend
          ? card.trend.muted
            ? 'var(--text-muted)'
            : card.trend.good
              ? 'var(--signal-healthy)'
              : 'var(--signal-critical)'
          : 'var(--text-muted)';

        return (
          <div
            key={card.label}
            style={{ ...cardStyle, cursor: card.navigateTo ? 'pointer' : 'default' }}
            onClick={card.navigateTo ? () => navigate(card.navigateTo!) : undefined}
            onMouseEnter={(e) => {
              (e.currentTarget as HTMLDivElement).style.borderColor = 'var(--text-faint)';
            }}
            onMouseLeave={(e) => {
              (e.currentTarget as HTMLDivElement).style.borderColor = 'var(--border)';
            }}
          >
            <div style={labelStyle}>{card.label}</div>
            <div
              style={{
                ...numberStyle,
                color: card.critical ? 'var(--signal-critical)' : 'var(--text-emphasis)',
              }}
            >
              {card.value}
            </div>
            {card.trend ? (
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 4,
                  fontSize: 12,
                  color: 'var(--text-muted)',
                }}
              >
                <span
                  style={{
                    fontSize: 13,
                    fontWeight: 600,
                    color: trendColor,
                    lineHeight: 1,
                  }}
                >
                  {arrowChars[card.trend.direction]}
                </span>
                <span style={{ color: trendColor, fontFamily: 'var(--font-mono)', fontSize: 11 }}>
                  {card.trend.text}
                </span>
              </div>
            ) : (
              <div style={subtextStyle}>{card.subtitle}</div>
            )}
          </div>
        );
      })}
    </div>
  );
}
