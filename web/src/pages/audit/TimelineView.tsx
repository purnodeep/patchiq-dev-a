import {
  getEventCategory,
  getCategoryColor,
  formatEventDescription,
  groupEventsByDate,
} from '../../lib/audit-utils';
import type { components } from '../../api/types';

type AuditEvent = components['schemas']['AuditEvent'];

interface TimelineViewProps {
  events: AuditEvent[];
}

export function TimelineView({ events }: TimelineViewProps) {
  if (events.length === 0) {
    return (
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          height: 240,
          fontFamily: 'var(--font-sans)',
          fontSize: 13,
          color: 'var(--text-muted)',
        }}
      >
        No audit events found.
      </div>
    );
  }

  const grouped = groupEventsByDate(events);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 0 }}>
      {Array.from(grouped.entries()).map(([dateLabel, dateEvents]) => (
        <div key={dateLabel}>
          {/* Date separator */}
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 12,
              margin: '16px 0 12px',
            }}
          >
            <div style={{ flex: 1, height: 1, background: 'var(--border)' }} />
            <span
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 10,
                fontWeight: 600,
                color: 'var(--text-muted)',
                textTransform: 'uppercase',
                letterSpacing: '0.06em',
                whiteSpace: 'nowrap',
              }}
            >
              {dateLabel}
            </span>
            <div style={{ flex: 1, height: 1, background: 'var(--border)' }} />
          </div>

          {/* Timeline events */}
          <div style={{ paddingLeft: 24, position: 'relative' }}>
            {/* Vertical line */}
            <div
              style={{
                position: 'absolute',
                left: 8,
                top: 0,
                bottom: 0,
                width: 1,
                background: 'var(--border)',
              }}
            />

            {dateEvents.map((event, index) => {
              const id = event.id ?? '';
              const category = getEventCategory(event.type ?? '');
              const dotColor = getCategoryColor(category);
              const { title, subtitle } = formatEventDescription(event);
              const time = event.timestamp
                ? new Date(event.timestamp).toLocaleTimeString('en-US', {
                    hour: '2-digit',
                    minute: '2-digit',
                    second: '2-digit',
                    timeZone: 'UTC',
                    hour12: false,
                  }) + ' UTC'
                : '';

              // Category color for text label
              const categoryTextColor =
                category === 'Deployment' || category === 'Compliance'
                  ? 'var(--accent)'
                  : category === 'Patch' || category === 'System'
                    ? 'var(--signal-critical)'
                    : category === 'Policy'
                      ? 'var(--signal-warning)'
                      : 'var(--text-muted)';

              return (
                <div
                  key={id ? `${id}-${index}` : index}
                  style={{
                    display: 'flex',
                    gap: 14,
                    marginBottom: 12,
                    position: 'relative',
                  }}
                >
                  {/* Dot */}
                  <div
                    style={{
                      width: 14,
                      height: 14,
                      borderRadius: '50%',
                      background: dotColor,
                      border: '2px solid var(--bg-page)',
                      flexShrink: 0,
                      position: 'absolute',
                      left: -21,
                      top: 6,
                    }}
                  />

                  {/* Card */}
                  <div
                    style={{
                      flex: 1,
                      background: 'var(--bg-card)',
                      border: '1px solid var(--border)',
                      borderRadius: 6,
                      padding: '10px 14px',
                      transition: 'border-color 0.1s ease',
                    }}
                    onMouseEnter={(e) =>
                      ((e.currentTarget as HTMLDivElement).style.borderColor =
                        'var(--border-hover)')
                    }
                    onMouseLeave={(e) =>
                      ((e.currentTarget as HTMLDivElement).style.borderColor = 'var(--border)')
                    }
                  >
                    <div
                      style={{
                        fontFamily: 'var(--font-sans)',
                        fontSize: 12,
                        color: 'var(--text-primary)',
                        lineHeight: 1.4,
                        marginBottom: 3,
                      }}
                    >
                      {title}
                    </div>
                    {subtitle && (
                      <div
                        style={{
                          fontFamily: 'var(--font-sans)',
                          fontSize: 11,
                          color: 'var(--text-muted)',
                          marginBottom: 6,
                        }}
                      >
                        {subtitle}
                      </div>
                    )}
                    <div
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: 10,
                      }}
                    >
                      <span
                        style={{
                          fontFamily: 'var(--font-mono)',
                          fontSize: 10,
                          color: 'var(--text-faint)',
                        }}
                      >
                        {time}
                      </span>
                      <span
                        style={{
                          fontFamily: 'var(--font-mono)',
                          fontSize: 9,
                          fontWeight: 600,
                          color: categoryTextColor,
                          textTransform: 'uppercase',
                          letterSpacing: '0.05em',
                        }}
                      >
                        {category}
                      </span>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      ))}
    </div>
  );
}
