import { UPCOMING_SLA_DATA } from '@/data/mock-data';

type Severity = 'critical' | 'high' | 'medium' | 'low';

const SEVERITY_COLORS: Record<Severity, { bg: string; border: string; text: string }> = {
  critical: {
    bg: 'color-mix(in srgb, var(--color-danger) 18%, transparent)',
    border: 'color-mix(in srgb, var(--color-danger) 50%, transparent)',
    text: 'var(--color-danger)',
  },
  high: {
    bg: 'color-mix(in srgb, var(--color-warning) 18%, transparent)',
    border: 'color-mix(in srgb, var(--color-warning) 50%, transparent)',
    text: 'var(--color-warning)',
  },
  medium: {
    bg: 'color-mix(in srgb, var(--color-caution) 18%, transparent)',
    border: 'color-mix(in srgb, var(--color-caution) 50%, transparent)',
    text: 'var(--color-caution)',
  },
  low: {
    bg: 'color-mix(in srgb, var(--color-success) 18%, transparent)',
    border: 'color-mix(in srgb, var(--color-success) 50%, transparent)',
    text: 'var(--color-success)',
  },
};

const MAX_CHIPS_SHOWN = 3;

/** Extract a short date portion from a day string like 'Mon Mar 11' */
function extractDate(day: string): string {
  const parts = day.split(' ');
  // Return "Mar 11" style (parts[1] + parts[2])
  if (parts.length >= 3) return `${parts[1]} ${parts[2]}`;
  return day;
}

/** Extract the day abbreviation like 'Mon' */
function extractDayAbbr(day: string): string {
  return day.split(' ')[0] ?? day;
}

export function UpcomingSLADeadlines() {
  // Ensure exactly 7 columns — pad or slice
  const days = UPCOMING_SLA_DATA.slice(0, 7);

  return (
    <div
      style={{
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        fontFamily: 'var(--font-sans)',
      }}
    >
      <div
        style={{
          flex: 1,
          display: 'grid',
          gridTemplateColumns: 'repeat(7, 1fr)',
          gap: 4,
          minHeight: 0,
        }}
      >
        {days.map((dayData) => {
          const isToday = dayData.isToday === true;
          const overflowCount =
            dayData.patches.length > MAX_CHIPS_SHOWN ? dayData.patches.length - MAX_CHIPS_SHOWN : 0;
          const visiblePatches = dayData.patches.slice(0, MAX_CHIPS_SHOWN);

          return (
            <div
              key={dayData.day}
              style={{
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                gap: 3,
                padding: '6px 3px',
                borderRadius: 6,
                background: isToday
                  ? 'color-mix(in srgb, var(--color-primary) 8%, transparent)'
                  : 'transparent',
                border: isToday
                  ? '1px solid color-mix(in srgb, var(--color-primary) 25%, transparent)'
                  : '1px solid transparent',
                minWidth: 0,
                overflow: 'hidden',
              }}
            >
              {/* Today badge */}
              {isToday ? (
                <span
                  style={{
                    fontSize: 8,
                    fontWeight: 700,
                    color: 'var(--color-primary)',
                    background: 'color-mix(in srgb, var(--color-primary) 15%, transparent)',
                    borderRadius: 8,
                    padding: '1px 5px',
                    letterSpacing: '0.03em',
                    textTransform: 'uppercase',
                  }}
                >
                  Today
                </span>
              ) : (
                <span style={{ height: 14 }} />
              )}

              {/* Day abbreviation */}
              <span
                style={{
                  fontSize: 10,
                  fontWeight: 700,
                  color: isToday ? 'var(--color-primary)' : 'var(--color-muted)',
                  textTransform: 'uppercase',
                  letterSpacing: '0.04em',
                }}
              >
                {extractDayAbbr(dayData.day)}
              </span>

              {/* Date */}
              <span
                style={{
                  fontSize: 9,
                  color: 'var(--color-muted)',
                  whiteSpace: 'nowrap',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  maxWidth: '100%',
                }}
                title={extractDate(dayData.day)}
              >
                {extractDate(dayData.day)}
              </span>

              {/* Patch chips */}
              <div
                style={{
                  display: 'flex',
                  flexDirection: 'column',
                  gap: 2,
                  width: '100%',
                  alignItems: 'center',
                  flex: 1,
                }}
              >
                {dayData.patches.length === 0 ? (
                  <span style={{ fontSize: 12, color: 'var(--color-separator)', marginTop: 2 }}>
                    —
                  </span>
                ) : (
                  <>
                    {visiblePatches.map((patch) => {
                      const sev = patch.severity as Severity;
                      const colors = SEVERITY_COLORS[sev] ?? SEVERITY_COLORS.low;
                      return (
                        <div
                          key={patch.id}
                          title={`${patch.cve} (${patch.severity})`}
                          style={{
                            width: '100%',
                            background: colors.bg,
                            border: `1px solid ${colors.border}`,
                            borderRadius: 3,
                            padding: '1px 3px',
                            fontSize: 8,
                            fontWeight: 600,
                            fontFamily: 'monospace',
                            color: colors.text,
                            overflow: 'hidden',
                            textOverflow: 'ellipsis',
                            whiteSpace: 'nowrap',
                            textAlign: 'center',
                            cursor: 'default',
                          }}
                        >
                          {patch.cve}
                        </div>
                      );
                    })}

                    {overflowCount > 0 && (
                      <span
                        style={{
                          fontSize: 8,
                          fontWeight: 600,
                          color: 'var(--color-muted)',
                          background: 'var(--color-separator)',
                          borderRadius: 8,
                          padding: '1px 5px',
                        }}
                      >
                        +{overflowCount} more
                      </span>
                    )}
                  </>
                )}
              </div>
            </div>
          );
        })}
      </div>

      {/* Legend */}
      <div
        style={{
          display: 'flex',
          gap: 10,
          flexWrap: 'wrap',
          paddingTop: 8,
          marginTop: 4,
          borderTop: '1px solid var(--color-separator)',
        }}
      >
        {(Object.entries(SEVERITY_COLORS) as [Severity, (typeof SEVERITY_COLORS)[Severity]][]).map(
          ([sev, colors]) => (
            <div key={sev} style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
              <span
                style={{
                  width: 8,
                  height: 8,
                  borderRadius: 2,
                  background: colors.text,
                  flexShrink: 0,
                  opacity: 0.85,
                }}
              />
              <span
                style={{
                  fontSize: 10,
                  fontWeight: 500,
                  color: 'var(--color-muted)',
                  textTransform: 'capitalize',
                }}
              >
                {sev}
              </span>
            </div>
          ),
        )}
      </div>
    </div>
  );
}
