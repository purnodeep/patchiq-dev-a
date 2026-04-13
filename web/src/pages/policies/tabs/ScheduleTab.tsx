// eslint-disable-next-line @typescript-eslint/no-explicit-any -- PolicyDetail schema not yet in generated types
type PolicyDetail = any;

interface ScheduleTabProps {
  policy: PolicyDetail;
}

const CARD: React.CSSProperties = {
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  borderRadius: 8,
  boxShadow: 'var(--shadow-sm)',
  padding: '16px 20px',
};

const SECTION_LABEL: React.CSSProperties = {
  fontFamily: 'var(--font-mono)',
  fontSize: 9,
  fontWeight: 600,
  textTransform: 'uppercase',
  letterSpacing: '0.07em',
  color: 'var(--text-muted)',
  marginBottom: 14,
};

const DAYS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];

// Determine which days of the week a cron expression hits
function cronDays(cron: string | undefined): Set<number> {
  if (!cron) return new Set();
  try {
    const parts = cron.trim().split(/\s+/);
    if (parts.length < 5) return new Set();
    const dayPart = parts[4]; // day-of-week field
    if (dayPart === '*' || dayPart === '?') return new Set([0, 1, 2, 3, 4, 5, 6]);
    const result = new Set<number>();
    for (const chunk of dayPart.split(',')) {
      if (chunk.includes('-')) {
        const [a, b] = chunk.split('-').map(Number);
        for (let d = a; d <= b; d++) result.add(d % 7);
      } else if (chunk.includes('/')) {
        const [start, step] = chunk.split('/').map(Number);
        for (let d = start; d <= 6; d += step) result.add(d);
      } else {
        const n = parseInt(chunk, 10);
        if (!isNaN(n)) result.add(n % 7);
      }
    }
    return result;
  } catch {
    return new Set();
  }
}

// Determine which calendar days (of current month) a cron hits
function cronCalendarDays(cron: string | undefined, year: number, month: number): Set<number> {
  if (!cron) return new Set();
  try {
    const parts = cron.trim().split(/\s+/);
    if (parts.length < 5) return new Set();
    const dayOfMonthPart = parts[2];
    if (dayOfMonthPart === '*' || dayOfMonthPart === '?') {
      // Use day-of-week instead
      const wDays = cronDays(cron);
      const result = new Set<number>();
      const daysInMonth = new Date(year, month + 1, 0).getDate();
      for (let d = 1; d <= daysInMonth; d++) {
        const wd = new Date(year, month, d).getDay();
        if (wDays.has(wd)) result.add(d);
      }
      return result;
    }
    const result = new Set<number>();
    for (const chunk of dayOfMonthPart.split(',')) {
      if (chunk.includes('-')) {
        const [a, b] = chunk.split('-').map(Number);
        for (let d = a; d <= b; d++) result.add(d);
      } else if (chunk.includes('/')) {
        const [start, step] = chunk.split('/').map(Number);
        const daysInMonth = new Date(year, month + 1, 0).getDate();
        for (let d = start; d <= daysInMonth; d += step) result.add(d);
      } else {
        const n = parseInt(chunk, 10);
        if (!isNaN(n)) result.add(n);
      }
    }
    return result;
  } catch {
    return new Set();
  }
}

// 24-hour timeline bar for maintenance window
function MwTimelineBar({ start, end }: { start: string; end: string }) {
  const parseH = (t: string) => {
    const [h, m] = t.split(':').map(Number);
    return h + (m || 0) / 60;
  };

  const startH = parseH(start);
  const endH = parseH(end);
  const leftPct = (startH / 24) * 100;
  const widthPct = (((endH - startH + 24) % 24) / 24) * 100 || (1 / 24) * 100;

  return (
    <div>
      {/* Hour markers */}
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
        {[0, 6, 12, 18, 24].map((h) => (
          <div
            key={h}
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 8,
              color: 'var(--text-faint)',
            }}
          >
            {h === 24 ? '24' : `${h}:00`}
          </div>
        ))}
      </div>

      {/* Track */}
      <div
        style={{
          height: 16,
          borderRadius: 4,
          background: 'var(--bg-inset)',
          position: 'relative',
          overflow: 'hidden',
        }}
      >
        {/* Window fill */}
        <div
          style={{
            position: 'absolute',
            left: `${leftPct}%`,
            width: `${Math.min(widthPct, 100 - leftPct)}%`,
            top: 0,
            bottom: 0,
            background: 'color-mix(in srgb, var(--accent) 20%, transparent)',
            borderLeft: '2px solid var(--accent)',
            borderRight: '2px solid var(--accent)',
          }}
        />
        {/* Tick marks */}
        {[6, 12, 18].map((h) => (
          <div
            key={h}
            style={{
              position: 'absolute',
              left: `${(h / 24) * 100}%`,
              top: '30%',
              bottom: '30%',
              width: 1,
              background: 'var(--border)',
            }}
          />
        ))}
      </div>

      {/* Start/end labels */}
      <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: 4 }}>
        <span style={{ fontFamily: 'var(--font-mono)', fontSize: 10, color: 'var(--accent)' }}>
          {start}
        </span>
        <span style={{ fontFamily: 'var(--font-mono)', fontSize: 10, color: 'var(--accent)' }}>
          {end}
        </span>
      </div>
    </div>
  );
}

export const ScheduleTab = ({ policy }: ScheduleTabProps) => {
  const now = new Date();
  const year = now.getFullYear();
  const month = now.getMonth();
  const today = now.getDate();

  const firstDay = new Date(year, month, 1).getDay();
  const daysInMonth = new Date(year, month + 1, 0).getDate();
  const monthName = now.toLocaleDateString('en-US', { month: 'long', year: 'numeric' });

  const scheduledDays = cronCalendarDays(policy.schedule_cron, year, month);
  const hasSchedule = policy.schedule_type === 'recurring' && scheduledDays.size > 0;

  const calDays: (number | null)[] = [];
  for (let i = 0; i < firstDay; i++) calDays.push(null);
  for (let i = 1; i <= daysInMonth; i++) calDays.push(i);

  // Day-of-week strip for schedule pattern
  const wDays = cronDays(policy.schedule_cron);

  return (
    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 14 }}>
      {/* Calendar card */}
      <div style={CARD}>
        <div style={SECTION_LABEL}>{monthName} — Scheduled Runs</div>

        {/* 7-day strip summary */}
        {hasSchedule && (
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: 'repeat(7, 1fr)',
              gap: 3,
              marginBottom: 12,
              padding: '8px',
              background: 'var(--bg-inset)',
              borderRadius: 6,
            }}
          >
            {DAYS.map((d, idx) => {
              const active = wDays.has(idx);
              return (
                <div
                  key={d}
                  style={{
                    textAlign: 'center',
                    padding: '4px 2px',
                    borderRadius: 4,
                    background: active
                      ? 'color-mix(in srgb, var(--accent) 12%, transparent)'
                      : 'transparent',
                    border: active
                      ? '1px solid color-mix(in srgb, var(--accent) 25%, transparent)'
                      : '1px solid transparent',
                  }}
                >
                  <div
                    style={{
                      fontFamily: 'var(--font-mono)',
                      fontSize: 9,
                      fontWeight: active ? 700 : 400,
                      color: active ? 'var(--accent)' : 'var(--text-faint)',
                      letterSpacing: '0.02em',
                    }}
                  >
                    {d}
                  </div>
                </div>
              );
            })}
          </div>
        )}

        {/* Day headers */}
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(7, 1fr)',
            gap: 2,
            marginBottom: 2,
          }}
        >
          {DAYS.map((d) => (
            <div
              key={d}
              style={{
                textAlign: 'center',
                fontFamily: 'var(--font-mono)',
                fontSize: 8,
                fontWeight: 600,
                color: 'var(--text-faint)',
                padding: 4,
                letterSpacing: '0.03em',
              }}
            >
              {d}
            </div>
          ))}
        </div>

        {/* Calendar grid */}
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(7, 1fr)',
            gap: 2,
            marginBottom: 12,
          }}
        >
          {calDays.map((day, i) => {
            const isToday = day === today;
            const isScheduled = day !== null && hasSchedule && scheduledDays.has(day);
            const isPast = day !== null && day < today;

            return (
              <div
                key={i}
                style={{
                  textAlign: 'center',
                  padding: '5px 2px',
                  fontFamily: 'var(--font-mono)',
                  fontSize: 10,
                  borderRadius: 4,
                  background: isToday
                    ? 'color-mix(in srgb, var(--accent) 12%, transparent)'
                    : isScheduled
                      ? isPast
                        ? 'color-mix(in srgb, var(--signal-healthy) 8%, transparent)'
                        : 'color-mix(in srgb, var(--accent) 12%, transparent)'
                      : 'transparent',
                  color: isToday
                    ? 'var(--accent)'
                    : isScheduled
                      ? isPast
                        ? 'var(--signal-healthy)'
                        : 'var(--accent)'
                      : 'var(--text-muted)',
                  fontWeight: isToday || isScheduled ? 700 : 400,
                  border: isToday
                    ? '1px solid color-mix(in srgb, var(--accent) 35%, transparent)'
                    : isScheduled
                      ? '1px solid color-mix(in srgb, var(--accent) 20%, transparent)'
                      : '1px solid transparent',
                  position: 'relative',
                }}
              >
                {day ?? ''}
                {/* Dot indicator for scheduled days */}
                {isScheduled && day !== null && (
                  <div
                    style={{
                      position: 'absolute',
                      bottom: 2,
                      left: '50%',
                      transform: 'translateX(-50%)',
                      width: 3,
                      height: 3,
                      borderRadius: '50%',
                      background: isPast ? 'var(--signal-healthy)' : 'var(--accent)',
                    }}
                  />
                )}
              </div>
            );
          })}
        </div>

        {/* Legend */}
        <div style={{ display: 'flex', gap: 14, flexWrap: 'wrap' }}>
          {[
            { color: 'var(--signal-healthy)', label: 'Past run' },
            { color: 'var(--accent)', label: 'Scheduled' },
            { color: 'var(--text-muted)', label: 'No run' },
          ].map(({ color, label }) => (
            <span
              key={label}
              style={{
                display: 'inline-flex',
                alignItems: 'center',
                gap: 5,
                fontSize: 9,
                color: 'var(--text-muted)',
              }}
            >
              <span
                style={{
                  width: 6,
                  height: 6,
                  borderRadius: '50%',
                  background: color,
                  flexShrink: 0,
                }}
              />
              {label}
            </span>
          ))}
        </div>
      </div>

      {/* Right column */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
        {/* Upcoming runs */}
        <div style={CARD}>
          <div style={SECTION_LABEL}>Upcoming Runs</div>
          {policy.schedule_type === 'recurring' ? (
            <div>
              <div style={{ fontSize: 11, color: 'var(--text-muted)', marginBottom: 10 }}>
                Cron expression:{' '}
                <code
                  style={{
                    fontFamily: 'var(--font-mono)',
                    fontSize: 11,
                    background: 'var(--bg-inset)',
                    border: '1px solid var(--border)',
                    borderRadius: 4,
                    padding: '2px 7px',
                    color: 'var(--text-primary)',
                  }}
                >
                  {policy.schedule_cron ?? '—'}
                </code>
                {policy.timezone && policy.timezone !== 'UTC' && (
                  <div style={{ fontSize: 12, color: 'var(--text-muted)', marginTop: 4 }}>
                    Timezone: {policy.timezone}
                  </div>
                )}
              </div>

              {/* Schedule type stat */}
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  padding: '8px 0',
                  borderBottom: '1px solid var(--border)',
                }}
              >
                <span style={{ fontSize: 12, color: 'var(--text-secondary)', fontWeight: 500 }}>
                  Next scheduled run
                </span>
                <span
                  style={{
                    fontFamily: 'var(--font-mono)',
                    fontSize: 11,
                    fontWeight: 600,
                    color: 'var(--accent)',
                  }}
                >
                  pending
                </span>
              </div>

              {/* Days active */}
              {wDays.size > 0 && (
                <div style={{ marginTop: 10 }}>
                  <div
                    style={{
                      fontSize: 9,
                      color: 'var(--text-muted)',
                      textTransform: 'uppercase',
                      letterSpacing: '0.06em',
                      marginBottom: 6,
                      fontFamily: 'var(--font-mono)',
                    }}
                  >
                    Active Days
                  </div>
                  <div style={{ display: 'flex', gap: 4 }}>
                    {DAYS.map((d, i) => (
                      <div
                        key={d}
                        style={{
                          width: 24,
                          height: 24,
                          borderRadius: 4,
                          background: wDays.has(i)
                            ? 'color-mix(in srgb, var(--accent) 15%, transparent)'
                            : 'var(--bg-inset)',
                          border: `1px solid ${wDays.has(i) ? 'color-mix(in srgb, var(--accent) 30%, transparent)' : 'var(--border)'}`,
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          fontFamily: 'var(--font-mono)',
                          fontSize: 8,
                          fontWeight: wDays.has(i) ? 700 : 400,
                          color: wDays.has(i) ? 'var(--accent)' : 'var(--text-faint)',
                        }}
                      >
                        {d[0]}
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          ) : (
            <p style={{ fontSize: 12, color: 'var(--text-muted)' }}>
              This policy runs on demand. Trigger manually from the action buttons above.
            </p>
          )}
        </div>

        {/* Maintenance window */}
        <div style={CARD}>
          <div style={SECTION_LABEL}>Maintenance Window</div>
          {policy.mw_start || policy.mw_end ? (
            <div>
              {/* Start/end display */}
              <div style={{ display: 'flex', alignItems: 'center', gap: 16, marginBottom: 14 }}>
                {[
                  { label: 'Start', value: policy.mw_start ?? '—' },
                  { label: 'End', value: policy.mw_end ?? '—' },
                ].map(({ label, value }, i) => (
                  <div
                    key={label}
                    style={{ display: 'flex', alignItems: 'center', gap: i === 0 ? 0 : 0 }}
                  >
                    <div>
                      <div
                        style={{
                          fontSize: 9,
                          color: 'var(--text-muted)',
                          marginBottom: 3,
                          textTransform: 'uppercase',
                          letterSpacing: '0.06em',
                          fontFamily: 'var(--font-mono)',
                        }}
                      >
                        {label}
                      </div>
                      <div
                        style={{
                          fontFamily: 'var(--font-mono)',
                          fontSize: 18,
                          fontWeight: 700,
                          color: 'var(--text-emphasis)',
                        }}
                      >
                        {value}
                      </div>
                    </div>
                    {i === 0 && (
                      <span
                        style={{
                          color: 'var(--text-muted)',
                          fontSize: 20,
                          margin: '0 12px',
                          marginTop: 8,
                        }}
                      >
                        →
                      </span>
                    )}
                  </div>
                ))}
              </div>

              {/* 24h visual bar */}
              {policy.mw_start && policy.mw_end && (
                <MwTimelineBar start={policy.mw_start} end={policy.mw_end} />
              )}

              {/* Duration */}
              {policy.mw_start &&
                policy.mw_end &&
                (() => {
                  const parseH = (t: string) => {
                    const [h, m] = t.split(':').map(Number);
                    return h + (m || 0) / 60;
                  };
                  const dur = (parseH(policy.mw_end) - parseH(policy.mw_start) + 24) % 24;
                  const hours = Math.floor(dur);
                  const mins = Math.round((dur - hours) * 60);
                  return (
                    <div
                      style={{
                        marginTop: 10,
                        fontFamily: 'var(--font-mono)',
                        fontSize: 10,
                        color: 'var(--text-muted)',
                      }}
                    >
                      Duration: {hours}h{mins > 0 ? ` ${mins}m` : ''}
                    </div>
                  );
                })()}
            </div>
          ) : (
            <p style={{ fontSize: 12, color: 'var(--text-muted)' }}>
              No maintenance window configured.
            </p>
          )}
        </div>
      </div>
    </div>
  );
};
