import { useMemo } from 'react';
import { useNavigate } from 'react-router';
import { useSLADeadlines, type SLADeadlineEntry } from '@/api/hooks/useDashboard';
import { SLAEmptyState } from './SLAEmptyState';

interface SLAItem {
  key: string;
  label: string;
  sublabel: string;
  fillPct: number;
  color: 'green' | 'amber' | 'red';
  overdue: boolean;
  secondsRemaining: number;
  navTo: string;
}

const cardStyle: React.CSSProperties = {
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  borderRadius: 8,
  boxShadow: 'var(--shadow-sm)',
  transition: 'border-color 150ms ease',
  display: 'flex',
  flexDirection: 'column',
};

const fillColors: Record<string, string> = {
  green: 'var(--accent)',
  amber: 'var(--signal-warning)',
  red: 'var(--signal-critical)',
};

// Backend applies the same mapping in GetSLADeadlines (dashboard.sql).
function slaWindowSeconds(severity: string): number {
  switch (severity.toLowerCase()) {
    case 'critical':
      return 24 * 3600;
    case 'high':
      return 72 * 3600;
    case 'medium':
      return 7 * 86400;
    default:
      return 30 * 86400;
  }
}

function formatCountdown(seconds: number): string {
  if (seconds <= 0) return 'OVERDUE';
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  if (h >= 24) {
    const d = Math.floor(h / 24);
    const rh = h % 24;
    return rh > 0 ? `${d}d ${rh}h left` : `${d}d left`;
  }
  return `${h}h ${String(m).padStart(2, '0')}m left`;
}

export function SLAStatus() {
  const { data: slaData, isLoading, isError } = useSLADeadlines();
  const navigate = useNavigate();

  const overdueCount = useMemo(() => {
    if (!slaData) return 0;
    return slaData.filter((entry: SLADeadlineEntry) => entry.remaining_seconds <= 0).length;
  }, [slaData]);

  const items: SLAItem[] = useMemo(() => {
    if (!slaData || slaData.length === 0) return [];
    return slaData.map((entry: SLADeadlineEntry) => {
      const total = slaWindowSeconds(entry.severity);
      const remaining = entry.remaining_seconds;
      const elapsed = total - remaining;
      const fillPct = Math.round(Math.min(1, Math.max(0, elapsed / total)) * 100);
      const isOverdue = remaining <= 0;
      const fraction = remaining / total;

      let color: 'green' | 'amber' | 'red';
      if (isOverdue) {
        color = 'red';
      } else if (fraction < 0.3) {
        color = 'amber';
      } else {
        color = 'green';
      }

      return {
        key: `${entry.endpoint_id}:${entry.patch_name}`,
        label: entry.hostname,
        sublabel: entry.patch_name,
        fillPct: isOverdue ? 100 : fillPct,
        color,
        overdue: isOverdue,
        secondsRemaining: remaining,
        navTo: `/endpoints/${entry.endpoint_id}`,
      };
    });
  }, [slaData]);

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
          padding: '16px 20px 0',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}
      >
        <span style={{ fontSize: 13, color: 'var(--text-muted)', fontWeight: 400 }}>
          SLA Status
        </span>
        {overdueCount > 0 && (
          <span
            style={{
              fontSize: 11,
              color: 'var(--signal-critical)',
              fontFamily: 'var(--font-mono)',
            }}
          >
            {overdueCount} overdue
          </span>
        )}
      </div>
      <div style={{ padding: '12px 20px 16px', flex: 1 }}>
        {isError && (
          <div style={{ color: 'var(--signal-critical)', fontSize: 12, padding: '10px 0' }}>
            Failed to load SLA deadlines
          </div>
        )}
        {!isError && isLoading && (
          <div style={{ color: 'var(--text-faint)', fontSize: 12, padding: '10px 0' }}>
            Loading...
          </div>
        )}
        {!isError && !isLoading && items.length === 0 && <SLAEmptyState />}
        {!isError && !isLoading && items.length > 0 &&
          items.map((sla, i) => (
            <div
              key={sla.key}
              onClick={() => navigate(sla.navTo)}
              style={{
                padding: '10px 0',
                borderBottom: i < items.length - 1 ? '1px solid var(--border-faint)' : 'none',
                cursor: 'pointer',
              }}
            >
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  marginBottom: 8,
                }}
              >
                <div style={{ display: 'flex', flexDirection: 'column' }}>
                  <span style={{ fontSize: 12, color: 'var(--text-secondary)' }}>{sla.label}</span>
                  <span style={{ fontSize: 10, color: 'var(--text-faint)' }}>{sla.sublabel}</span>
                </div>
                {sla.overdue ? (
                  <span
                    style={{
                      fontFamily: 'var(--font-mono)',
                      fontSize: 11,
                      fontWeight: 600,
                      color: 'var(--signal-critical)',
                      letterSpacing: '0.06em',
                    }}
                  >
                    OVERDUE
                  </span>
                ) : (
                  <span
                    style={{
                      fontFamily: 'var(--font-mono)',
                      fontSize: 11,
                      color: sla.color === 'amber' ? 'var(--signal-warning)' : 'var(--text-muted)',
                    }}
                  >
                    {formatCountdown(sla.secondsRemaining)}
                  </span>
                )}
              </div>
              <div
                style={{
                  height: 4,
                  background: 'var(--sla-track, var(--border))',
                  borderRadius: 2,
                  overflow: 'hidden',
                }}
              >
                <div
                  style={{
                    height: '100%',
                    borderRadius: 2,
                    background: fillColors[sla.color],
                    width: `${sla.fillPct}%`,
                    transition: 'width 400ms ease',
                  }}
                />
              </div>
            </div>
          ))}
      </div>
    </div>
  );
}
