import { useEffect, useMemo, useState } from 'react';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { CronExpressionParser } from 'cron-parser';
import { Button } from '@patchiq/ui';
import { TagInput } from '../../components/TagInput';
import { TagSelectorBuilder } from '../../components/targeting/TagSelectorBuilder';
import type { Selector } from '../../types/targeting';
import { PolicyPreview } from './PolicyPreview';
import { COMMON_TIMEZONES } from '../../components/PolicyWizard/types';

// Selector is stored as an opaque JSON-ish object inside the form state;
// the builder component owns shape validation. `z.any()` is appropriate
// here because the server is the authoritative validator.
const selectorSchema = z.any().nullable().optional();

const schema = z.object({
  name: z.string().min(1, 'Name is required').max(255),
  description: z.string().max(1000),
  policy_type: z.enum(['patch', 'deploy', 'compliance']),
  mode: z.enum(['automatic', 'manual', 'advisory']),
  selection_mode: z.enum(['all_available', 'by_severity', 'by_cve_list', 'by_regex']),
  target_selector: selectorSchema,
  min_severity: z.enum(['critical', 'high', 'medium', 'low']).optional(),
  cve_ids: z.array(z.string()),
  package_regex: z.string(),
  exclude_packages: z.array(z.string()),
  schedule_type: z.enum(['manual', 'recurring']),
  schedule_cron: z.string(),
  timezone: z.string(),
  mw_start: z.string(),
  mw_end: z.string(),
  mw_enabled: z.boolean(),
});

export type PolicyFormValues = z.infer<typeof schema>;

interface PolicyFormProps {
  defaultValues?: Partial<PolicyFormValues>;
  onSubmit: (values: PolicyFormValues) => Promise<void>;
  submitLabel: string;
  isPending: boolean;
  submitDisabled?: boolean;
  submitDisabledTitle?: string;
  serverError?: { message?: string; field?: string } | null;
}

// ── Shared input style ────────────────────────────────────────────────────────

const INPUT: React.CSSProperties = {
  display: 'flex',
  width: '100%',
  height: 36,
  borderRadius: 6,
  border: '1px solid var(--border)',
  background: 'var(--bg-card)',
  padding: '0 10px',
  fontSize: 13,
  color: 'var(--text-primary)',
  outline: 'none',
  boxSizing: 'border-box',
  fontFamily: 'var(--font-sans)',
};

const LABEL: React.CSSProperties = {
  fontSize: 10,
  fontWeight: 600,
  textTransform: 'uppercase',
  letterSpacing: '0.06em',
  fontFamily: 'var(--font-mono)',
  color: 'var(--text-muted)',
  marginBottom: 6,
  display: 'block',
};

const CARD: React.CSSProperties = {
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  borderRadius: 8,
  boxShadow: 'var(--shadow-sm)',
  padding: '20px 24px',
};

const SECTION_TITLE: React.CSSProperties = {
  fontFamily: 'var(--font-mono)',
  fontSize: 10,
  fontWeight: 600,
  textTransform: 'uppercase',
  letterSpacing: '0.06em',
  color: 'var(--text-emphasis)',
  marginBottom: 16,
  paddingBottom: 12,
  borderBottom: '1px solid var(--border)',
};

const RADIO_GROUP: React.CSSProperties = {
  display: 'flex',
  flexDirection: 'column',
  gap: 8,
};

const RADIO_LABEL: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  gap: 8,
  fontSize: 13,
  color: 'var(--text-primary)',
  cursor: 'pointer',
};

// ── Component ─────────────────────────────────────────────────────────────────

export const PolicyForm = ({
  defaultValues,
  onSubmit,
  submitLabel,
  isPending,
  submitDisabled,
  submitDisabledTitle,
  serverError,
}: PolicyFormProps) => {
  const resolvedMode = (document.documentElement.classList.contains('dark') ? 'dark' : 'light') as
    | 'dark'
    | 'light';
  const {
    register,
    handleSubmit,
    watch,
    control,
    setValue,
    setError,
    formState: { errors },
  } = useForm<PolicyFormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: '',
      description: '',
      policy_type: 'patch',
      mode: 'manual',
      selection_mode: 'all_available',
      target_selector: null,
      min_severity: undefined,
      cve_ids: [],
      package_regex: '',
      exclude_packages: [],
      schedule_type: 'manual',
      schedule_cron: '',
      timezone: 'UTC',
      mw_start: '',
      mw_end: '',
      mw_enabled: false,
      ...defaultValues,
    },
  });

  const selectionMode = watch('selection_mode');
  const scheduleType = watch('schedule_type');
  const policyType = watch('policy_type');
  const mwEnabled = watch('mw_enabled');

  const [schedulePreset, setSchedulePreset] = useState<'daily' | 'weekly' | 'monthly' | 'custom'>(
    'weekly',
  );

  useEffect(() => {
    if (policyType === 'compliance') {
      setValue('mode', 'advisory');
    }
  }, [policyType, setValue]);

  useEffect(() => {
    if (serverError?.field) {
      const validFields = Object.keys(schema.shape);
      if (validFields.includes(serverError.field)) {
        setError(serverError.field as keyof PolicyFormValues, {
          message: serverError.message ?? 'Validation error',
        });
        return;
      }
    }
  }, [serverError, setError]);

  const handleFormSubmit = handleSubmit(async (values) => {
    await onSubmit(values);
  });

  const typeOptions = [
    {
      value: 'patch' as const,
      label: 'Patch Policy',
      desc: 'Select patches by severity, CVE, or regex. Evaluate and optionally auto-deploy.',
      color: 'var(--accent)',
    },
    {
      value: 'deploy' as const,
      label: 'Deploy Policy',
      desc: 'Target specific updates for direct deployment to endpoints.',
      color: 'var(--signal-healthy)',
    },
    {
      value: 'compliance' as const,
      label: 'Compliance Policy',
      desc: 'Evaluate patch compliance on a schedule. Report only, no deployments.',
      color: 'var(--text-muted)',
    },
  ];

  const modeOptions = [
    {
      value: 'automatic',
      label: 'Automatic',
      desc: 'Evaluates on schedule. Matching patches deploy automatically within the maintenance window.',
      color: 'var(--signal-healthy)',
    },
    {
      value: 'manual',
      label: 'Manual',
      desc: 'Evaluates on schedule. Patches are queued but NOT deployed until you click Deploy.',
      color: 'var(--accent)',
    },
    {
      value: 'advisory',
      label: 'Advisory',
      desc: 'Evaluates on schedule. Reports compliance status only. No patches are ever deployed.',
      color: 'var(--text-muted)',
    },
  ];

  function getFilteredModes(type: string) {
    if (type === 'deploy') return modeOptions.filter((o) => o.value !== 'advisory');
    if (type === 'compliance') return modeOptions.filter((o) => o.value === 'advisory');
    return modeOptions;
  }

  const filteredModes = getFilteredModes(policyType);

  const cronValue = watch('schedule_cron');
  const timezoneValue = watch('timezone');
  const nextRuns = useMemo(() => {
    if (!cronValue || scheduleType !== 'recurring') return [];
    try {
      const interval = CronExpressionParser.parse(cronValue, { tz: timezoneValue || 'UTC' });
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
  }, [cronValue, timezoneValue, scheduleType]);

  const cronParseError = useMemo(() => {
    if (!cronValue || scheduleType !== 'recurring') return null;
    try {
      CronExpressionParser.parse(cronValue);
      return null;
    } catch {
      return 'Invalid cron expression';
    }
  }, [cronValue, scheduleType]);

  const allValues = watch();

  return (
    <div style={{ display: 'flex', gap: 24, alignItems: 'flex-start' }}>
      <form
        onSubmit={handleFormSubmit}
        style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 16 }}
      >
        {/* Policy Type */}
        <div style={CARD}>
          <div style={SECTION_TITLE}>Policy Type</div>
          <div style={{ display: 'flex', gap: 10 }}>
            {typeOptions.map((opt) => {
              const isSelected = watch('policy_type') === opt.value;
              return (
                <label
                  key={opt.value}
                  style={{
                    flex: 1,
                    display: 'flex',
                    flexDirection: 'column',
                    gap: 4,
                    padding: '12px 14px',
                    borderRadius: 6,
                    border: `1px solid ${isSelected ? opt.color : 'var(--border)'}`,
                    background: isSelected
                      ? `color-mix(in srgb, ${opt.color} 8%, transparent)`
                      : 'var(--bg-inset)',
                    cursor: 'pointer',
                    transition: 'border-color 0.15s, background 0.15s',
                  }}
                >
                  <input
                    type="radio"
                    value={opt.value}
                    {...register('policy_type')}
                    style={{ display: 'none' }}
                  />
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                    <div
                      style={{
                        width: 8,
                        height: 8,
                        borderRadius: '50%',
                        background: isSelected ? opt.color : 'var(--border)',
                        flexShrink: 0,
                      }}
                    />
                    <span
                      style={{
                        fontSize: 13,
                        fontWeight: 600,
                        color: isSelected ? opt.color : 'var(--text-primary)',
                      }}
                    >
                      {opt.label}
                    </span>
                  </div>
                  <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>{opt.desc}</span>
                </label>
              );
            })}
          </div>
        </div>

        {/* Basics */}
        <div style={CARD}>
          <div style={SECTION_TITLE}>Basics</div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
            <div>
              <label htmlFor="name" style={LABEL}>
                Name <span style={{ color: 'var(--signal-critical)' }}>*</span>
              </label>
              <input
                id="name"
                {...register('name')}
                style={INPUT}
                placeholder="e.g. Linux Critical Patching"
                onFocus={(e) => {
                  e.currentTarget.style.borderColor = 'var(--accent)';
                  e.currentTarget.style.boxShadow =
                    '0 0 0 2px color-mix(in srgb, var(--accent) 12%, transparent)';
                }}
                onBlur={(e) => {
                  e.currentTarget.style.borderColor = 'var(--border)';
                  e.currentTarget.style.boxShadow = 'none';
                }}
              />
              {errors.name && (
                <p style={{ marginTop: 4, fontSize: 12, color: 'var(--signal-critical)' }}>
                  {errors.name.message}
                </p>
              )}
            </div>
            <div>
              <label htmlFor="description" style={LABEL}>
                Description
              </label>
              <textarea
                id="description"
                {...register('description')}
                rows={3}
                style={{
                  ...INPUT,
                  height: 'auto',
                  padding: '8px 12px',
                  resize: 'vertical',
                  lineHeight: 1.5,
                }}
                placeholder="What does this policy do?"
                onFocus={(e) => {
                  e.currentTarget.style.borderColor = 'var(--accent)';
                  e.currentTarget.style.boxShadow =
                    '0 0 0 2px color-mix(in srgb, var(--accent) 12%, transparent)';
                }}
                onBlur={(e) => {
                  e.currentTarget.style.borderColor = 'var(--border)';
                  e.currentTarget.style.boxShadow = 'none';
                }}
              />
            </div>
            <div>
              <label style={LABEL}>Mode</label>
              <div style={{ display: 'flex', gap: 10 }}>
                {filteredModes.map((opt) => {
                  const isSelected = watch('mode') === opt.value;
                  return (
                    <label
                      key={opt.value}
                      style={{
                        flex: 1,
                        display: 'flex',
                        flexDirection: 'column',
                        gap: 4,
                        padding: '12px 14px',
                        borderRadius: 6,
                        border: `1px solid ${isSelected ? opt.color : 'var(--border)'}`,
                        background: isSelected
                          ? `color-mix(in srgb, ${opt.color} 8%, transparent)`
                          : 'var(--bg-inset)',
                        cursor: 'pointer',
                        transition: 'border-color 0.15s, background 0.15s',
                      }}
                    >
                      <input
                        type="radio"
                        value={opt.value}
                        {...register('mode')}
                        style={{ display: 'none' }}
                      />
                      <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                        <div
                          style={{
                            width: 8,
                            height: 8,
                            borderRadius: '50%',
                            background: isSelected ? opt.color : 'var(--border)',
                            flexShrink: 0,
                          }}
                        />
                        <span
                          style={{
                            fontSize: 13,
                            fontWeight: 600,
                            color: isSelected ? opt.color : 'var(--text-primary)',
                          }}
                        >
                          {opt.label}
                        </span>
                      </div>
                      <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>{opt.desc}</span>
                    </label>
                  );
                })}
              </div>
            </div>
          </div>
        </div>

        {/* Target Selector */}
        <div style={CARD}>
          <div style={SECTION_TITLE}>Target Endpoints</div>
          <p style={{ fontSize: 12, color: 'var(--text-muted)', marginBottom: 14 }}>
            Define which endpoints this policy targets using tag predicates. All predicates
            compose with AND — an endpoint must match every row to be included.
          </p>
          <Controller
            name="target_selector"
            control={control}
            render={({ field }) => (
              <TagSelectorBuilder
                value={field.value as Selector | null}
                onChange={(next) => field.onChange(next)}
              />
            )}
          />
        </div>

        {/* Patch Selection */}
        {policyType !== 'deploy' && (
          <div style={CARD}>
            <div style={SECTION_TITLE}>Patch Selection</div>
            <div style={RADIO_GROUP}>
              {[
                { value: 'all_available', label: 'All Available Patches' },
                { value: 'by_severity', label: 'By Severity' },
                { value: 'by_cve_list', label: 'By CVE List' },
                { value: 'by_regex', label: 'By Package Regex' },
              ].map((opt) => (
                <label key={opt.value} style={RADIO_LABEL}>
                  <input
                    type="radio"
                    value={opt.value}
                    {...register('selection_mode')}
                    style={{ accentColor: 'var(--accent)', cursor: 'pointer' }}
                  />
                  {opt.label}
                </label>
              ))}
            </div>

            {selectionMode === 'by_severity' && (
              <div style={{ marginTop: 16 }}>
                <label htmlFor="min_severity" style={LABEL}>
                  Minimum Severity
                </label>
                <select
                  id="min_severity"
                  {...register('min_severity')}
                  style={{ ...INPUT, colorScheme: resolvedMode }}
                >
                  <option value="">Select severity...</option>
                  <option value="critical">Critical</option>
                  <option value="high">High</option>
                  <option value="medium">Medium</option>
                  <option value="low">Low</option>
                </select>
              </div>
            )}

            {selectionMode === 'by_cve_list' && (
              <div style={{ marginTop: 16 }}>
                <label style={LABEL}>CVE IDs</label>
                <Controller
                  name="cve_ids"
                  control={control}
                  render={({ field }) => (
                    <TagInput
                      value={field.value ?? []}
                      onChange={field.onChange}
                      placeholder="Type CVE ID and press Enter..."
                      className="mt-1"
                    />
                  )}
                />
              </div>
            )}

            {selectionMode === 'by_regex' && (
              <div style={{ marginTop: 16, display: 'flex', flexDirection: 'column', gap: 14 }}>
                <div>
                  <label htmlFor="package_regex" style={LABEL}>
                    Package Regex
                  </label>
                  <input
                    id="package_regex"
                    {...register('package_regex')}
                    placeholder="e.g. ^openssl.*"
                    style={{ ...INPUT, fontFamily: 'var(--font-mono)' }}
                    onFocus={(e) => {
                      e.currentTarget.style.borderColor = 'var(--accent)';
                      e.currentTarget.style.boxShadow =
                        '0 0 0 2px color-mix(in srgb, var(--accent) 12%, transparent)';
                    }}
                    onBlur={(e) => {
                      e.currentTarget.style.borderColor = 'var(--border)';
                      e.currentTarget.style.boxShadow = 'none';
                    }}
                  />
                </div>
                <div>
                  <label style={LABEL}>Exclude Packages</label>
                  <Controller
                    name="exclude_packages"
                    control={control}
                    render={({ field }) => (
                      <TagInput
                        value={field.value ?? []}
                        onChange={field.onChange}
                        placeholder="Package name to exclude..."
                        className="mt-1"
                      />
                    )}
                  />
                </div>
              </div>
            )}
          </div>
        )}

        {/* Schedule */}
        <div style={CARD}>
          <div style={SECTION_TITLE}>Schedule</div>
          <div style={RADIO_GROUP}>
            <label style={RADIO_LABEL}>
              <input
                type="radio"
                value="manual"
                {...register('schedule_type')}
                style={{ accentColor: 'var(--accent)', cursor: 'pointer' }}
              />
              Manual — trigger on demand
            </label>
            <label style={RADIO_LABEL}>
              <input
                type="radio"
                value="recurring"
                {...register('schedule_type')}
                style={{ accentColor: 'var(--accent)', cursor: 'pointer' }}
              />
              Recurring — run on cron schedule
            </label>
          </div>
          {scheduleType === 'recurring' && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 14, marginTop: 14 }}>
              {/* Preset buttons */}
              <div style={{ display: 'flex', gap: 8 }}>
                {(['daily', 'weekly', 'monthly', 'custom'] as const).map((p) => (
                  <button
                    key={p}
                    type="button"
                    onClick={() => setSchedulePreset(p)}
                    style={{
                      padding: '6px 14px',
                      borderRadius: 6,
                      border: `1px solid ${schedulePreset === p ? 'var(--accent)' : 'var(--border)'}`,
                      background:
                        schedulePreset === p
                          ? 'color-mix(in srgb, var(--accent) 10%, transparent)'
                          : 'var(--bg-inset)',
                      color: schedulePreset === p ? 'var(--accent)' : 'var(--text-primary)',
                      fontSize: 12,
                      fontWeight: 600,
                      cursor: 'pointer',
                      textTransform: 'capitalize',
                    }}
                  >
                    {p}
                  </button>
                ))}
              </div>

              {/* Daily: time picker */}
              {schedulePreset === 'daily' && (
                <div>
                  <label style={LABEL}>Time</label>
                  <input
                    type="time"
                    style={{ ...INPUT, fontFamily: 'var(--font-mono)' }}
                    defaultValue="02:00"
                    onChange={(e) => {
                      const [h, m] = e.target.value.split(':');
                      setValue('schedule_cron', `${parseInt(m)} ${parseInt(h)} * * *`);
                    }}
                  />
                </div>
              )}

              {/* Weekly: day picker + time */}
              {schedulePreset === 'weekly' && (
                <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
                  <div>
                    <label style={LABEL}>Day of Week</label>
                    <div style={{ display: 'flex', gap: 6 }}>
                      {['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'].map((day, i) => {
                        const cronDay = i === 6 ? 0 : i + 1;
                        return (
                          <button
                            key={day}
                            type="button"
                            style={{
                              padding: '4px 10px',
                              borderRadius: 4,
                              border: '1px solid var(--border)',
                              background: 'var(--bg-inset)',
                              color: 'var(--text-primary)',
                              fontSize: 11,
                              cursor: 'pointer',
                            }}
                            onClick={() => {
                              const cronVal = watch('schedule_cron') || '0 2 * * 1';
                              const parts = cronVal.split(' ');
                              parts[4] = String(cronDay);
                              setValue('schedule_cron', parts.join(' '));
                            }}
                          >
                            {day}
                          </button>
                        );
                      })}
                    </div>
                  </div>
                  <div>
                    <label style={LABEL}>Time</label>
                    <input
                      type="time"
                      style={{ ...INPUT, fontFamily: 'var(--font-mono)' }}
                      defaultValue="02:00"
                      onChange={(e) => {
                        const [h, m] = e.target.value.split(':');
                        const cronVal = watch('schedule_cron') || '0 2 * * 1';
                        const parts = cronVal.split(' ');
                        parts[0] = String(parseInt(m));
                        parts[1] = String(parseInt(h));
                        setValue('schedule_cron', parts.join(' '));
                      }}
                    />
                  </div>
                </div>
              )}

              {/* Monthly: day of month + time */}
              {schedulePreset === 'monthly' && (
                <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
                  <div>
                    <label style={LABEL}>Day of Month</label>
                    <select
                      style={{ ...INPUT }}
                      defaultValue="1"
                      onChange={(e) => {
                        const cronVal = watch('schedule_cron') || '0 2 1 * *';
                        const parts = cronVal.split(' ');
                        parts[2] = e.target.value;
                        setValue('schedule_cron', parts.join(' '));
                      }}
                    >
                      {Array.from({ length: 28 }, (_, i) => (
                        <option key={i + 1} value={String(i + 1)}>
                          {i + 1}
                        </option>
                      ))}
                    </select>
                  </div>
                  <div>
                    <label style={LABEL}>Time</label>
                    <input
                      type="time"
                      style={{ ...INPUT, fontFamily: 'var(--font-mono)' }}
                      defaultValue="02:00"
                      onChange={(e) => {
                        const [h, m] = e.target.value.split(':');
                        const cronVal = watch('schedule_cron') || '0 2 1 * *';
                        const parts = cronVal.split(' ');
                        parts[0] = String(parseInt(m));
                        parts[1] = String(parseInt(h));
                        setValue('schedule_cron', parts.join(' '));
                      }}
                    />
                  </div>
                </div>
              )}

              {/* Custom: raw cron */}
              {schedulePreset === 'custom' && (
                <div>
                  <label htmlFor="schedule_cron" style={LABEL}>
                    Cron Expression
                  </label>
                  <input
                    id="schedule_cron"
                    {...register('schedule_cron')}
                    placeholder="0 2 * * 1"
                    style={{ ...INPUT, fontFamily: 'var(--font-mono)' }}
                  />
                  <p style={{ marginTop: 6, fontSize: 11, color: 'var(--text-muted)' }}>
                    e.g. "0 2 * * 1" = Every Monday at 2 AM
                  </p>
                  {cronParseError && (
                    <p style={{ marginTop: 4, fontSize: 11, color: 'var(--signal-critical)' }}>
                      {cronParseError}
                    </p>
                  )}
                </div>
              )}

              {/* Timezone */}
              <div>
                <label style={LABEL}>Timezone</label>
                <select {...register('timezone')} style={{ ...INPUT }}>
                  {COMMON_TIMEZONES.map((tz) => (
                    <option key={tz} value={tz}>
                      {tz}
                    </option>
                  ))}
                </select>
              </div>

              {/* Next 3 runs preview */}
              {nextRuns.length > 0 && (
                <div>
                  <span style={{ ...LABEL, marginBottom: 4 }}>Next runs</span>
                  {nextRuns.map((r, i) => (
                    <div
                      key={i}
                      style={{
                        fontSize: 12,
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

        {/* Maintenance Window */}
        {policyType !== 'compliance' && (
          <div style={CARD}>
            <div
              style={{
                ...SECTION_TITLE,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
              }}
            >
              <span>Maintenance Window</span>
              <label style={{ display: 'flex', alignItems: 'center', gap: 6, cursor: 'pointer' }}>
                <input
                  type="checkbox"
                  {...register('mw_enabled')}
                  style={{ accentColor: 'var(--accent)', width: 14, height: 14, cursor: 'pointer' }}
                />
                <span style={{ fontSize: 10, fontWeight: 600, color: 'var(--text-muted)' }}>
                  {mwEnabled ? 'Enabled' : 'Disabled'}
                </span>
              </label>
            </div>
            {!mwEnabled && (
              <p style={{ fontSize: 12, color: 'var(--text-muted)' }}>
                Optional. Restrict deployments to a specific time window.
              </p>
            )}
            {mwEnabled && (
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
                <div>
                  <label htmlFor="mw_start" style={LABEL}>
                    Start Time
                  </label>
                  <input
                    id="mw_start"
                    type="time"
                    {...register('mw_start')}
                    style={{ ...INPUT, colorScheme: resolvedMode, fontFamily: 'var(--font-mono)' }}
                  />
                </div>
                <div>
                  <label htmlFor="mw_end" style={LABEL}>
                    End Time
                  </label>
                  <input
                    id="mw_end"
                    type="time"
                    {...register('mw_end')}
                    style={{ ...INPUT, colorScheme: resolvedMode, fontFamily: 'var(--font-mono)' }}
                  />
                </div>
              </div>
            )}
          </div>
        )}

        {/* Server error banner */}
        {serverError &&
          (!serverError.field || !Object.keys(schema.shape).includes(serverError.field)) && (
            <div
              style={{
                padding: '10px 14px',
                background: 'color-mix(in srgb, var(--signal-critical) 10%, transparent)',
                border: '1px solid var(--signal-critical)',
                borderRadius: 6,
                fontSize: 13,
                color: 'var(--signal-critical)',
              }}
            >
              {serverError.message ?? 'Failed to save policy'}
            </div>
          )}

        {/* Footer actions */}
        <div
          style={{
            position: 'sticky',
            bottom: 0,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'flex-end',
            gap: 10,
            borderTop: '1px solid var(--border)',
            background: 'var(--bg-page)',
            padding: '12px 0',
            marginTop: 4,
          }}
        >
          <Button type="button" variant="outline" onClick={() => window.history.back()}>
            Cancel
          </Button>
          <Button
            type="submit"
            disabled={isPending || submitDisabled}
            title={submitDisabledTitle}
            style={submitDisabled ? { opacity: 0.5 } : undefined}
          >
            {isPending ? 'Saving...' : submitLabel}
          </Button>
        </div>
      </form>
      <PolicyPreview values={allValues} />
    </div>
  );
};
