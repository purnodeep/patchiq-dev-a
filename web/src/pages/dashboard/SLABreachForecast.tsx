import { useMemo } from 'react';
import { SkeletonCard, ErrorState } from '@patchiq/ui';
import { useSLAForecast } from '@/api/hooks/useDashboard';
import type { SLAForecastEntry } from '@/api/hooks/useDashboard';

const SEVERITY_ORDER: Record<string, number> = { critical: 0, high: 1, medium: 2, low: 3 };

function severityColor(severity: string): string {
  if (severity === 'critical') return 'var(--signal-critical)';
  if (severity === 'high') return 'var(--signal-warning)';
  return '#eab308';
}

function formatRemaining(seconds: number): string {
  if (seconds <= 0) return 'BREACHED';
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  if (hours > 24) {
    const days = Math.floor(hours / 24);
    return `${days}d ${hours % 24}h`;
  }
  return `${hours}h ${minutes}m`;
}

function progressPct(entry: SLAForecastEntry): number {
  const totalSeconds = entry.sla_window_hours * 3600;
  if (totalSeconds <= 0) return 0;
  return Math.min(100, Math.max(0, (entry.remaining_seconds / totalSeconds) * 100));
}

interface GroupedEntries {
  severity: string;
  items: SLAForecastEntry[];
}

// Inline keyframes style tag for pulse animation
const PULSE_STYLE = `
@keyframes sla-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}
`;

export function SLABreachForecast() {
  const { data, isLoading, error, refetch } = useSLAForecast();

  const grouped = useMemo((): GroupedEntries[] => {
    if (!data || data.length === 0) return [];
    const sorted = [...data].sort((a, b) => a.remaining_seconds - b.remaining_seconds);
    const groups = new Map<string, SLAForecastEntry[]>();
    for (const item of sorted) {
      if (!groups.has(item.severity)) groups.set(item.severity, []);
      groups.get(item.severity)!.push(item);
    }
    return [...groups.entries()]
      .sort((a, b) => (SEVERITY_ORDER[a[0]] ?? 99) - (SEVERITY_ORDER[b[0]] ?? 99))
      .map(([severity, items]) => ({ severity, items }));
  }, [data]);

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
        <ErrorState message="Failed to load SLA forecast" onRetry={() => refetch()} />
      </div>
    );
  if (!grouped.length)
    return (
      <div
        className="flex h-full items-center justify-center rounded-lg border"
        style={{
          background: 'var(--bg-card)',
          borderColor: 'var(--border)',
          color: 'var(--text-muted)',
        }}
      >
        No SLA breaches forecasted
      </div>
    );

  return (
    <div
      className="flex flex-col overflow-hidden rounded-lg border"
      style={{
        background: 'var(--bg-card)',
        borderColor: 'var(--border)',
        boxShadow: 'var(--shadow-sm)',
      }}
    >
      <style>{PULSE_STYLE}</style>
      <div style={{ padding: '16px 16px 8px', flexShrink: 0 }}>
        <h3 className="text-sm font-semibold" style={{ color: 'var(--text-emphasis)' }}>
          SLA Breach Forecast
        </h3>
        <p className="text-xs" style={{ color: 'var(--text-secondary)' }}>
          Upcoming SLA deadline violations
        </p>
      </div>
      <div style={{ flex: 1, minHeight: 0, overflow: 'auto', padding: '0 16px 16px' }}>
        {grouped.map((group) => (
          <div key={group.severity} style={{ marginBottom: 12 }}>
            {/* Severity header */}
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 6,
                marginBottom: 6,
                paddingBottom: 4,
                borderBottom: '1px solid var(--border)',
              }}
            >
              <div
                style={{
                  width: 8,
                  height: 8,
                  borderRadius: '50%',
                  background: severityColor(group.severity),
                }}
              />
              <span
                style={{
                  fontSize: 11,
                  fontWeight: 600,
                  color: 'var(--text-emphasis)',
                  textTransform: 'uppercase',
                  letterSpacing: '0.05em',
                }}
              >
                {group.severity}
              </span>
              <span style={{ fontSize: 10, color: 'var(--text-muted)', fontFamily: 'var(--font-mono)' }}>
                ({group.items.length})
              </span>
            </div>

            {/* Items */}
            {group.items.map((item) => {
              const pct = progressPct(item);
              const isUrgent = item.remaining_seconds < 2 * 3600;
              const isBreached = item.remaining_seconds <= 0;

              return (
                <div
                  key={item.id}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 10,
                    padding: '6px 0',
                    animation: isUrgent && !isBreached ? 'sla-pulse 2s ease-in-out infinite' : undefined,
                  }}
                >
                  {/* Hostname */}
                  <span
                    style={{
                      fontSize: 11,
                      color: 'var(--text-secondary)',
                      fontFamily: 'var(--font-mono)',
                      minWidth: 100,
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                      whiteSpace: 'nowrap',
                    }}
                  >
                    {item.hostname}
                  </span>

                  {/* Progress bar */}
                  <div
                    style={{
                      flex: 1,
                      height: 6,
                      background: 'var(--border)',
                      borderRadius: 3,
                      overflow: 'hidden',
                      minWidth: 60,
                    }}
                  >
                    <div
                      style={{
                        height: '100%',
                        width: `${pct}%`,
                        borderRadius: 3,
                        background: isBreached
                          ? 'var(--signal-critical)'
                          : isUrgent
                            ? 'var(--signal-warning)'
                            : 'var(--signal-success)',
                        transition: 'width 300ms ease',
                      }}
                    />
                  </div>

                  {/* Time remaining */}
                  <span
                    style={{
                      fontSize: 11,
                      fontWeight: 600,
                      fontFamily: 'var(--font-mono)',
                      minWidth: 70,
                      textAlign: 'right',
                      color: isBreached
                        ? 'var(--signal-critical)'
                        : isUrgent
                          ? 'var(--signal-warning)'
                          : 'var(--text-secondary)',
                    }}
                  >
                    {formatRemaining(item.remaining_seconds)}
                  </span>
                </div>
              );
            })}
          </div>
        ))}
      </div>
    </div>
  );
}
