import { useState, useMemo } from 'react';
import { useFormContext, Controller } from 'react-hook-form';
import { CronExpressionParser } from 'cron-parser';
import { TagInput } from '../TagInput';
import type { PolicyWizardValues } from './types';
import { LABEL_STYLE, INPUT, COMMON_TIMEZONES } from './types';

// Schedule option card — matches DeploymentWizard StrategyStep schedule cards exactly
function ScheduleCard({
  active,
  label,
  description,
  onClick,
}: {
  active: boolean;
  label: string;
  description: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      style={{
        padding: '10px 8px',
        borderRadius: 7,
        border: `1px solid ${active ? 'var(--accent)' : 'var(--border)'}`,
        background: active
          ? 'color-mix(in srgb, var(--accent) 8%, transparent)'
          : 'var(--bg-inset)',
        cursor: 'pointer',
        transition: 'all 0.15s',
        textAlign: 'left',
        outline: 'none',
      }}
    >
      <span
        style={{
          fontSize: 11,
          fontWeight: 600,
          color: active ? 'var(--accent)' : 'var(--text-secondary)',
          display: 'block',
          marginBottom: 3,
        }}
      >
        {label}
      </span>
      <span style={{ fontSize: 9, color: 'var(--text-muted)', display: 'block' }}>
        {description}
      </span>
    </button>
  );
}

// Patch selection card — same visual style as schedule cards but for patch scope
function PatchCard({
  active,
  label,
  description,
  onClick,
}: {
  active: boolean;
  label: string;
  description: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      style={{
        padding: '10px 12px',
        borderRadius: 7,
        border: `1px solid ${active ? 'var(--accent)' : 'var(--border)'}`,
        background: active
          ? 'color-mix(in srgb, var(--accent) 8%, transparent)'
          : 'var(--bg-inset)',
        cursor: 'pointer',
        transition: 'all 0.15s',
        textAlign: 'left',
        display: 'flex',
        alignItems: 'center',
        gap: 10,
        outline: 'none',
      }}
    >
      <div
        style={{
          width: 8,
          height: 8,
          borderRadius: '50%',
          background: active ? 'var(--accent)' : 'var(--border)',
          flexShrink: 0,
          transition: 'background 0.15s',
        }}
      />
      <div>
        <span
          style={{
            fontSize: 12,
            fontWeight: 600,
            color: active ? 'var(--accent)' : 'var(--text-primary)',
            display: 'block',
            marginBottom: 2,
          }}
        >
          {label}
        </span>
        <span style={{ fontSize: 10, color: 'var(--text-muted)', display: 'block' }}>
          {description}
        </span>
      </div>
    </button>
  );
}

const DAYS_OF_WEEK = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'];
const DAY_CRON_VALUES = ['1', '2', '3', '4', '5', '6', '0'];

const patchOptions = [
  {
    value: 'all_available' as const,
    label: 'All Available Patches',
    description: 'Deploy every pending patch',
  },
  {
    value: 'by_severity' as const,
    label: 'By Severity',
    description: 'Filter by minimum severity',
  },
  { value: 'by_cve_list' as const, label: 'By CVE List', description: 'Target specific CVE IDs' },
  {
    value: 'by_regex' as const,
    label: 'By Package Regex',
    description: 'Match packages by pattern',
  },
];

const scheduleOptions = [
  { value: 'manual' as const, label: 'Manual', description: 'Trigger on demand' },
  { value: 'recurring' as const, label: 'Recurring', description: 'Run on cron schedule' },
  {
    value: 'maintenance_window' as const,
    label: 'Maint. Window',
    description: 'Next available window',
  },
];

export function PatchesStep() {
  const resolvedMode = (document.documentElement.classList.contains('dark') ? 'dark' : 'light') as
    | 'dark'
    | 'light';

  const { register, watch, setValue, control } = useFormContext<PolicyWizardValues>();
  const policyType = watch('policy_type');
  const mwEnabled = watch('mw_enabled');
  const timezoneValue = watch('timezone');
  const selectionMode = watch('selection_mode');
  const scheduleType = watch('schedule_type');
  const scheduleCron = watch('schedule_cron');

  const [schedulePreset, setSchedulePreset] = useState<'daily' | 'weekly' | 'monthly' | 'custom'>(
    'weekly',
  );
  const [presetTime, setPresetTime] = useState('02:00');
  const [weeklyDay, setWeeklyDay] = useState('1');
  const [monthlyDay, setMonthlyDay] = useState('1');

  function applyPreset(
    preset: 'daily' | 'weekly' | 'monthly' | 'custom',
    opts?: { time?: string; day?: string; monthDay?: string },
  ) {
    const time = opts?.time ?? presetTime;
    const hour = time.split(':')[0] ?? '2';
    const day = opts?.day ?? weeklyDay;
    const mDay = opts?.monthDay ?? monthlyDay;

    setSchedulePreset(preset);

    switch (preset) {
      case 'daily':
        setValue('schedule_cron', `0 ${hour} * * *`);
        break;
      case 'weekly':
        setValue('schedule_cron', `0 ${hour} * * ${day}`);
        break;
      case 'monthly':
        setValue('schedule_cron', `0 ${hour} ${mDay} * *`);
        break;
      case 'custom':
        // leave cron as-is for manual editing
        break;
    }
  }

  function handleTimeChange(val: string) {
    setPresetTime(val);
    applyPreset(schedulePreset, { time: val });
  }

  function handleWeeklyDayChange(dayVal: string) {
    setWeeklyDay(dayVal);
    applyPreset('weekly', { day: dayVal });
  }

  function handleMonthlyDayChange(dayVal: string) {
    setMonthlyDay(dayVal);
    applyPreset('monthly', { monthDay: dayVal });
  }

  const nextRuns = useMemo(() => {
    if (!scheduleCron || scheduleType !== 'recurring') return [];
    try {
      const interval = CronExpressionParser.parse(scheduleCron, { tz: timezoneValue || 'UTC' });
      return [interval.next(), interval.next(), interval.next()].map((d) =>
        d.toDate().toLocaleString('en-US', {
          timeZone: timezoneValue || 'UTC',
          dateStyle: 'medium' as const,
          timeStyle: 'short' as const,
        }),
      );
    } catch {
      return [];
    }
  }, [scheduleCron, timezoneValue, scheduleType]);

  return (
    <div style={{ padding: 16, display: 'flex', flexDirection: 'column', gap: 20 }}>
      {/* ── Patch Selection (hidden for deploy type) ──────────────────── */}
      {policyType !== 'deploy' && (
        <div>
          <label style={LABEL_STYLE}>Patch Selection</label>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
            {patchOptions.map((opt) => (
              <PatchCard
                key={opt.value}
                active={selectionMode === opt.value}
                label={opt.label}
                description={opt.description}
                onClick={() => setValue('selection_mode', opt.value)}
              />
            ))}
          </div>

          {/* Conditional: by_severity */}
          {selectionMode === 'by_severity' && (
            <div style={{ marginTop: 12 }}>
              <label style={{ ...LABEL_STYLE, marginBottom: 6 }}>Minimum Severity</label>
              <select
                {...register('min_severity')}
                style={{ ...INPUT, colorScheme: resolvedMode }}
                onFocus={(e) => (e.currentTarget.style.borderColor = 'var(--accent)')}
                onBlur={(e) => (e.currentTarget.style.borderColor = 'var(--border)')}
              >
                <option value="">Select severity...</option>
                <option value="critical">Critical</option>
                <option value="high">High</option>
                <option value="medium">Medium</option>
                <option value="low">Low</option>
              </select>
            </div>
          )}

          {/* Conditional: by_cve_list */}
          {selectionMode === 'by_cve_list' && (
            <div style={{ marginTop: 12 }}>
              <label style={{ ...LABEL_STYLE, marginBottom: 6 }}>CVE IDs</label>
              <Controller
                name="cve_ids"
                control={control}
                render={({ field }) => (
                  <TagInput
                    value={field.value ?? []}
                    onChange={field.onChange}
                    placeholder="Type CVE ID and press Enter..."
                  />
                )}
              />
            </div>
          )}

          {/* Conditional: by_regex */}
          {selectionMode === 'by_regex' && (
            <div style={{ marginTop: 12, display: 'flex', flexDirection: 'column', gap: 10 }}>
              <div>
                <label style={{ ...LABEL_STYLE, marginBottom: 6 }}>Package Regex</label>
                <input
                  {...register('package_regex')}
                  placeholder="e.g. ^openssl.*"
                  style={{ ...INPUT, fontFamily: 'var(--font-mono)' }}
                  onFocus={(e) => (e.currentTarget.style.borderColor = 'var(--accent)')}
                  onBlur={(e) => (e.currentTarget.style.borderColor = 'var(--border)')}
                />
              </div>
              <div>
                <label style={{ ...LABEL_STYLE, marginBottom: 6 }}>Exclude Packages</label>
                <Controller
                  name="exclude_packages"
                  control={control}
                  render={({ field }) => (
                    <TagInput
                      value={field.value ?? []}
                      onChange={field.onChange}
                      placeholder="Package name to exclude..."
                    />
                  )}
                />
              </div>
            </div>
          )}
        </div>
      )}

      {/* ── Schedule — 3 cards matching StrategyStep exactly ──────────── */}
      <div>
        <label style={LABEL_STYLE}>Schedule</label>
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(3, 1fr)',
            gap: 8,
            marginBottom: scheduleType === 'recurring' ? 8 : 0,
          }}
        >
          {scheduleOptions.map((opt) => (
            <ScheduleCard
              key={opt.value}
              active={scheduleType === opt.value}
              label={opt.label}
              description={opt.description}
              onClick={() => setValue('schedule_type', opt.value)}
            />
          ))}
        </div>

        {/* Recurring: preset selector + pickers */}
        {scheduleType === 'recurring' && (
          <div>
            {/* Preset buttons */}
            <div
              style={{
                display: 'grid',
                gridTemplateColumns: 'repeat(4, 1fr)',
                gap: 6,
                marginBottom: 10,
              }}
            >
              {(['daily', 'weekly', 'monthly', 'custom'] as const).map((p) => (
                <ScheduleCard
                  key={p}
                  active={schedulePreset === p}
                  label={p.charAt(0).toUpperCase() + p.slice(1)}
                  description={
                    p === 'daily'
                      ? 'Every day'
                      : p === 'weekly'
                        ? 'Day of week'
                        : p === 'monthly'
                          ? 'Day of month'
                          : 'Raw cron'
                  }
                  onClick={() => applyPreset(p)}
                />
              ))}
            </div>

            {/* Daily: time only */}
            {schedulePreset === 'daily' && (
              <div style={{ marginBottom: 10 }}>
                <label style={{ ...LABEL_STYLE, marginBottom: 6 }}>Time</label>
                <input
                  type="time"
                  value={presetTime}
                  onChange={(e) => handleTimeChange(e.target.value)}
                  style={{ ...INPUT, colorScheme: resolvedMode, fontFamily: 'var(--font-mono)' }}
                />
              </div>
            )}

            {/* Weekly: day buttons + time */}
            {schedulePreset === 'weekly' && (
              <div style={{ marginBottom: 10 }}>
                <label style={{ ...LABEL_STYLE, marginBottom: 6 }}>Day of Week</label>
                <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap', marginBottom: 8 }}>
                  {DAYS_OF_WEEK.map((d, i) => (
                    <button
                      key={d}
                      type="button"
                      onClick={() => handleWeeklyDayChange(DAY_CRON_VALUES[i] ?? '1')}
                      style={{
                        padding: '4px 8px',
                        borderRadius: 5,
                        border: `1px solid ${weeklyDay === DAY_CRON_VALUES[i] ? 'var(--accent)' : 'var(--border)'}`,
                        background:
                          weeklyDay === DAY_CRON_VALUES[i]
                            ? 'color-mix(in srgb, var(--accent) 8%, transparent)'
                            : 'var(--bg-inset)',
                        color:
                          weeklyDay === DAY_CRON_VALUES[i] ? 'var(--accent)' : 'var(--text-muted)',
                        fontSize: 11,
                        fontWeight: 600,
                        cursor: 'pointer',
                        outline: 'none',
                        fontFamily: 'var(--font-mono)',
                      }}
                    >
                      {d}
                    </button>
                  ))}
                </div>
                <label style={{ ...LABEL_STYLE, marginBottom: 6 }}>Time</label>
                <input
                  type="time"
                  value={presetTime}
                  onChange={(e) => handleTimeChange(e.target.value)}
                  style={{ ...INPUT, colorScheme: resolvedMode, fontFamily: 'var(--font-mono)' }}
                />
              </div>
            )}

            {/* Monthly: day select + time */}
            {schedulePreset === 'monthly' && (
              <div style={{ marginBottom: 10 }}>
                <label style={{ ...LABEL_STYLE, marginBottom: 6 }}>Day of Month</label>
                <select
                  value={monthlyDay}
                  onChange={(e) => handleMonthlyDayChange(e.target.value)}
                  style={{ ...INPUT, colorScheme: resolvedMode, marginBottom: 8 }}
                >
                  {Array.from({ length: 28 }, (_, i) => i + 1).map((d) => (
                    <option key={d} value={String(d)}>
                      {d}
                    </option>
                  ))}
                </select>
                <label style={{ ...LABEL_STYLE, marginBottom: 6 }}>Time</label>
                <input
                  type="time"
                  value={presetTime}
                  onChange={(e) => handleTimeChange(e.target.value)}
                  style={{ ...INPUT, colorScheme: resolvedMode, fontFamily: 'var(--font-mono)' }}
                />
              </div>
            )}

            {/* Custom: raw cron input */}
            {schedulePreset === 'custom' && (
              <div style={{ marginBottom: 10 }}>
                <input
                  {...register('schedule_cron')}
                  placeholder="0 2 * * 1"
                  style={{ ...INPUT, fontFamily: 'var(--font-mono)' }}
                  onFocus={(e) => (e.currentTarget.style.borderColor = 'var(--accent)')}
                  onBlur={(e) => (e.currentTarget.style.borderColor = 'var(--border)')}
                />
                <p
                  style={{
                    marginTop: 5,
                    fontSize: 10,
                    color: 'var(--text-muted)',
                    fontFamily: 'var(--font-mono)',
                  }}
                >
                  {scheduleCron || '0 2 * * 1'} — e.g. Every Monday at 02:00 UTC
                </p>
              </div>
            )}

            {/* Timezone dropdown */}
            <div style={{ marginTop: 10 }}>
              <label style={{ ...LABEL_STYLE, marginBottom: 6 }}>Timezone</label>
              <select {...register('timezone')} style={{ ...INPUT, colorScheme: resolvedMode }}>
                {COMMON_TIMEZONES.map((tz) => (
                  <option key={tz} value={tz}>
                    {tz}
                  </option>
                ))}
              </select>
            </div>

            {/* Next 3 runs preview */}
            {nextRuns.length > 0 && (
              <div style={{ marginTop: 8 }}>
                <div
                  style={{
                    fontSize: 9,
                    color: 'var(--text-muted)',
                    fontFamily: 'var(--font-mono)',
                    marginBottom: 4,
                  }}
                >
                  NEXT RUNS
                </div>
                {nextRuns.map((r, i) => (
                  <div
                    key={i}
                    style={{
                      fontSize: 11,
                      color: 'var(--text-muted)',
                      fontFamily: 'var(--font-mono)',
                    }}
                  >
                    {r}
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </div>

      {/* ── Maintenance Window (hidden for compliance type) ────────────── */}
      {policyType !== 'compliance' && (
        <div>
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              marginBottom: 10,
            }}
          >
            <label style={LABEL_STYLE}>Maintenance Window</label>
            <label
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 6,
                cursor: 'pointer',
                fontSize: 11,
                color: 'var(--text-muted)',
              }}
            >
              {mwEnabled ? 'Enabled' : 'Disabled'}
              <input
                type="checkbox"
                {...register('mw_enabled')}
                style={{ accentColor: 'var(--accent)', width: 14, height: 14, cursor: 'pointer' }}
              />
            </label>
          </div>
          {mwEnabled && (
            <>
              <p
                style={{ fontSize: 10, color: 'var(--text-muted)', marginBottom: 10, marginTop: 0 }}
              >
                Optional. Restrict deployments to a specific time window each day.
              </p>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10 }}>
                <div>
                  <div
                    style={{
                      fontSize: 9,
                      color: 'var(--text-muted)',
                      marginBottom: 4,
                      fontFamily: 'var(--font-mono)',
                    }}
                  >
                    Start Time
                  </div>
                  <input
                    type="time"
                    {...register('mw_start')}
                    style={{ ...INPUT, colorScheme: resolvedMode, fontFamily: 'var(--font-mono)' }}
                    onFocus={(e) => (e.currentTarget.style.borderColor = 'var(--accent)')}
                    onBlur={(e) => (e.currentTarget.style.borderColor = 'var(--border)')}
                  />
                </div>
                <div>
                  <div
                    style={{
                      fontSize: 9,
                      color: 'var(--text-muted)',
                      marginBottom: 4,
                      fontFamily: 'var(--font-mono)',
                    }}
                  >
                    End Time
                  </div>
                  <input
                    type="time"
                    {...register('mw_end')}
                    style={{ ...INPUT, colorScheme: resolvedMode, fontFamily: 'var(--font-mono)' }}
                    onFocus={(e) => (e.currentTarget.style.borderColor = 'var(--accent)')}
                    onBlur={(e) => (e.currentTarget.style.borderColor = 'var(--border)')}
                  />
                </div>
              </div>
            </>
          )}
        </div>
      )}
    </div>
  );
}
