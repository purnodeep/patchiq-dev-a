import { useEffect, useState, forwardRef } from 'react';
import { useCan } from '../../app/auth/AuthContext';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { toast } from 'sonner';
import { useSettings, useUpdateSettings } from '../../api/hooks/useSettings';
import type { GeneralSettings } from '../../api/hooks/useSettings';

const schema = z.object({
  org_name: z.string().min(1, 'Organization name is required'),
  timezone: z.string().min(1, 'Timezone is required'),
  date_format: z.string().min(1, 'Date format is required'),
  scan_interval_hours: z.number().int().min(1),
});

const TIMEZONES = [
  { value: 'UTC', label: 'UTC' },
  { value: 'America/New_York', label: 'America/New_York' },
  { value: 'America/Los_Angeles', label: 'America/Los_Angeles' },
  { value: 'Europe/London', label: 'Europe/London' },
  { value: 'Europe/Berlin', label: 'Europe/Berlin' },
  { value: 'Asia/Tokyo', label: 'Asia/Tokyo' },
];

const DATE_FORMATS = [
  { value: 'YYYY-MM-DD', label: 'YYYY-MM-DD (ISO 8601)' },
  { value: 'MM/DD/YYYY', label: 'MM/DD/YYYY (US)' },
  { value: 'DD/MM/YYYY', label: 'DD/MM/YYYY (EU)' },
  { value: 'DD MMM YYYY', label: 'DD MMM YYYY (Verbose)' },
];

const SCAN_INTERVALS = [
  { value: 1, label: '1 hour' },
  { value: 2, label: '2 hours' },
  { value: 4, label: '4 hours' },
  { value: 6, label: '6 hours' },
  { value: 12, label: '12 hours' },
  { value: 24, label: '24 hours' },
];

const fieldLabelStyle: React.CSSProperties = {
  display: 'block',
  fontSize: 10,
  fontWeight: 600,
  letterSpacing: '0.06em',
  textTransform: 'uppercase',
  color: 'var(--text-muted)',
  fontFamily: 'var(--font-mono)',
  marginBottom: 6,
};

const inputBaseStyle: React.CSSProperties = {
  width: '100%',
  height: 36,
  padding: '0 10px',
  background: 'var(--bg-input, var(--bg-card))',
  border: '1px solid var(--border)',
  borderRadius: 6,
  fontSize: 13,
  color: 'var(--text-primary)',
  fontFamily: 'var(--font-sans)',
  outline: 'none',
  transition: 'border-color 0.15s, box-shadow 0.15s',
  boxSizing: 'border-box',
};

const inputFocusStyle: React.CSSProperties = {
  borderColor: 'var(--accent)',
  boxShadow: '0 0 0 2px color-mix(in srgb, var(--accent) 15%, transparent)',
};

const selectChevron =
  "url(\"data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 24 24' fill='none' stroke='%236b7280' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='m6 9 6 6 6-6'/%3E%3C/svg%3E\")";

const hintStyle: React.CSSProperties = {
  fontSize: 11,
  color: 'var(--text-faint)',
  marginTop: 4,
};

const errorStyle: React.CSSProperties = {
  fontSize: 11,
  color: 'var(--signal-critical)',
  marginTop: 4,
  fontFamily: 'var(--font-sans)',
};

const FocusInput = forwardRef<HTMLInputElement, React.InputHTMLAttributes<HTMLInputElement>>(
  function FocusInput({ style, disabled, ...rest }, ref) {
    const [focused, setFocused] = useState(false);
    return (
      <input
        ref={ref}
        disabled={disabled}
        style={{
          ...inputBaseStyle,
          ...(focused && !disabled ? inputFocusStyle : {}),
          ...(disabled ? { opacity: 0.5, cursor: 'not-allowed' } : {}),
          ...style,
        }}
        onFocus={() => setFocused(true)}
        onBlur={(e) => {
          setFocused(false);
          rest.onBlur?.(e);
        }}
        {...rest}
      />
    );
  },
);

function FocusSelect(props: React.SelectHTMLAttributes<HTMLSelectElement>) {
  const [focused, setFocused] = useState(false);
  const { style, children, ...rest } = props;
  return (
    <select
      style={{
        ...inputBaseStyle,
        cursor: 'pointer',
        appearance: 'none',
        backgroundImage: selectChevron,
        backgroundRepeat: 'no-repeat',
        backgroundPosition: 'right 10px center',
        paddingRight: 30,
        ...(focused ? inputFocusStyle : {}),
        ...style,
      }}
      onFocus={() => setFocused(true)}
      onBlur={() => setFocused(false)}
      {...rest}
    >
      {children}
    </select>
  );
}

export function GeneralSettingsPage() {
  const can = useCan();
  const { data, isLoading, error } = useSettings();
  const update = useUpdateSettings();

  const {
    register,
    handleSubmit,
    reset,
    setValue,
    watch,
    formState: { errors, isDirty },
  } = useForm<GeneralSettings>({
    resolver: zodResolver(schema),
    defaultValues: {
      org_name: '',
      timezone: 'UTC',
      date_format: 'YYYY-MM-DD',
      scan_interval_hours: 24,
    },
  });

  useEffect(() => {
    if (data) {
      reset(data);
    }
  }, [data, reset]);

  const timezone = watch('timezone');
  const dateFormat = watch('date_format');
  const scanInterval = watch('scan_interval_hours');

  function onSubmit(values: GeneralSettings) {
    update.mutate(values, {
      onSuccess: () => {
        toast.success('Settings saved successfully.');
        reset(values);
      },
      onError: (err) => {
        toast.error(`Failed to save settings: ${err.message}`);
      },
    });
  }

  function onDiscard() {
    if (data) reset(data);
  }

  if (isLoading) {
    return (
      <div style={{ padding: '28px 40px 80px', maxWidth: 680 }}>
        <p style={{ fontSize: 13, color: 'var(--text-muted)', fontFamily: 'var(--font-sans)' }}>
          Loading...
        </p>
      </div>
    );
  }

  if (error) {
    return (
      <div style={{ padding: '28px 40px 80px', maxWidth: 680 }}>
        <p
          style={{ fontSize: 13, color: 'var(--signal-critical)', fontFamily: 'var(--font-sans)' }}
        >
          Failed to load settings. Please try again.
        </p>
      </div>
    );
  }

  return (
    <div style={{ padding: '28px 40px 80px', maxWidth: 680 }}>
      {/* Section header */}
      <div style={{ paddingBottom: 16, borderBottom: '1px solid var(--border)', marginBottom: 24 }}>
        <h2
          style={{
            fontSize: 18,
            fontWeight: 600,
            color: 'var(--text-emphasis)',
            letterSpacing: '-0.02em',
            margin: 0,
          }}
        >
          General
        </h2>
        <p
          style={{
            fontSize: 12,
            color: 'var(--text-muted)',
            marginTop: 4,
            maxWidth: 520,
            margin: '4px 0 0',
          }}
        >
          Organization identity, regional preferences, and default scan behavior.
        </p>
      </div>

      <form
        onSubmit={handleSubmit(onSubmit)}
        style={{ display: 'flex', flexDirection: 'column', gap: 24 }}
      >
        {/* Organization Name — full width */}
        <div>
          <label htmlFor="org_name" style={fieldLabelStyle}>
            Organization Name
          </label>
          <FocusInput id="org_name" placeholder="Acme Corp" {...register('org_name')} />
          {errors.org_name ? (
            <p style={errorStyle}>{errors.org_name.message}</p>
          ) : (
            <p style={hintStyle}>Displayed in reports, notifications, and the dashboard header.</p>
          )}
        </div>

        {/* Timezone + Date Format — 2-column grid */}
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
          <div>
            <label style={fieldLabelStyle}>Timezone</label>
            <FocusSelect
              key={`tz-${timezone}`}
              value={timezone}
              onChange={(e) => setValue('timezone', e.target.value, { shouldDirty: true })}
            >
              {TIMEZONES.map((tz) => (
                <option key={tz.value} value={tz.value}>
                  {tz.label}
                </option>
              ))}
            </FocusSelect>
          </div>

          <div>
            <label style={fieldLabelStyle}>Date Format</label>
            <FocusSelect
              key={`df-${dateFormat}`}
              value={dateFormat}
              onChange={(e) => setValue('date_format', e.target.value, { shouldDirty: true })}
            >
              {DATE_FORMATS.map((df) => (
                <option key={df.value} value={df.value}>
                  {df.label}
                </option>
              ))}
            </FocusSelect>
          </div>
        </div>

        {/* Default Scan Interval — single select, max-width 200px */}
        <div>
          <label style={fieldLabelStyle}>Default Scan Interval</label>
          <FocusSelect
            key={`si-${scanInterval}`}
            value={String(scanInterval)}
            onChange={(e) =>
              setValue('scan_interval_hours', Number(e.target.value), { shouldDirty: true })
            }
            style={{ maxWidth: 200 }}
          >
            {SCAN_INTERVALS.map((si) => (
              <option key={si.value} value={String(si.value)}>
                {si.label}
              </option>
            ))}
          </FocusSelect>
          <p style={hintStyle}>
            How frequently agents scan for available patches. Overridden by per-endpoint settings.
          </p>
        </div>

        {/* Button actions row */}
        <div
          style={{
            borderTop: '1px solid var(--border)',
            paddingTop: 16,
            display: 'flex',
            gap: 8,
            justifyContent: 'flex-end',
          }}
        >
          <button
            type="button"
            onClick={onDiscard}
            disabled={!isDirty || update.isPending}
            style={{
              background: 'transparent',
              color: 'var(--text-muted)',
              border: '1px solid var(--border-strong, var(--border))',
              borderRadius: 6,
              fontSize: 13,
              fontWeight: 500,
              padding: '7px 14px',
              cursor: !isDirty || update.isPending ? 'not-allowed' : 'pointer',
              opacity: !isDirty || update.isPending ? 0.5 : 1,
              fontFamily: 'var(--font-sans)',
            }}
          >
            Discard
          </button>
          <button
            type="submit"
            disabled={update.isPending || !isDirty || !can('settings', 'write')}
            title={!can('settings', 'write') ? "You don't have permission" : undefined}
            style={{
              background:
                update.isPending || !isDirty || !can('settings', 'write')
                  ? 'color-mix(in srgb, var(--accent) 40%, transparent)'
                  : 'var(--accent)',
              color: 'var(--btn-accent-text, #000)',
              border: 'none',
              borderRadius: 6,
              fontSize: 13,
              fontWeight: 600,
              padding: '7px 14px',
              cursor:
                update.isPending || !isDirty || !can('settings', 'write')
                  ? 'not-allowed'
                  : 'pointer',
              fontFamily: 'var(--font-sans)',
            }}
          >
            {update.isPending ? 'Saving...' : 'Save Changes'}
          </button>
        </div>
      </form>
    </div>
  );
}
