import { useState, useEffect, forwardRef } from 'react';

interface GeneralSettingsProps {
  settings: Record<string, unknown>;
  onSave: (settings: Record<string, unknown>) => void;
  saving: boolean;
}

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

export const GeneralSettings = ({ settings, onSave, saving }: GeneralSettingsProps) => {
  const [hubName, setHubName] = useState('');
  const [syncInterval, setSyncInterval] = useState('21600');
  const [region, setRegion] = useState('us-east-1');
  const [autoPublish, setAutoPublish] = useState(true);
  const [timezone, setTimezone] = useState('UTC');
  const [isDirty, setIsDirty] = useState(false);

  useEffect(() => {
    if (settings['hub.name'] != null) setHubName(String(settings['hub.name']));
    if (settings['hub.default_sync_interval'] != null)
      setSyncInterval(String(settings['hub.default_sync_interval']));
    if (settings['hub.region'] != null) setRegion(String(settings['hub.region']));
    if (settings['hub.catalog_auto_publish'] != null)
      setAutoPublish(Boolean(settings['hub.catalog_auto_publish']));
    if (settings['hub.timezone'] != null) setTimezone(String(settings['hub.timezone']));
    setIsDirty(false);
  }, [settings]);

  const markDirty = () => setIsDirty(true);

  const handleSave = () => {
    onSave({
      'hub.name': hubName,
      'hub.default_sync_interval': Number(syncInterval),
      'hub.region': region,
      'hub.catalog_auto_publish': autoPublish,
      'hub.timezone': timezone,
    });
    setIsDirty(false);
  };

  const handleDiscard = () => {
    if (settings['hub.name'] != null) setHubName(String(settings['hub.name']));
    if (settings['hub.default_sync_interval'] != null)
      setSyncInterval(String(settings['hub.default_sync_interval']));
    if (settings['hub.region'] != null) setRegion(String(settings['hub.region']));
    if (settings['hub.catalog_auto_publish'] != null)
      setAutoPublish(Boolean(settings['hub.catalog_auto_publish']));
    if (settings['hub.timezone'] != null) setTimezone(String(settings['hub.timezone']));
    setIsDirty(false);
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 24 }}>
      {/* Hub Name */}
      <div>
        <label style={fieldLabelStyle}>Hub Name</label>
        <FocusInput
          type="text"
          value={hubName}
          onChange={(e) => {
            setHubName(e.target.value);
            markDirty();
          }}
          placeholder="PatchIQ Hub"
        />
        <p style={hintStyle}>Displayed in reports and the dashboard header.</p>
      </div>

      {/* Sync Interval + Region — 2-column grid */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
        <div>
          <label style={fieldLabelStyle}>Default Sync Interval</label>
          <FocusSelect
            value={syncInterval}
            onChange={(e) => {
              setSyncInterval(e.target.value);
              markDirty();
            }}
          >
            <option value="300">5 minutes</option>
            <option value="900">15 minutes</option>
            <option value="1800">30 minutes</option>
            <option value="3600">1 hour</option>
            <option value="21600">6 hours</option>
          </FocusSelect>
        </div>

        <div>
          <label style={fieldLabelStyle}>Hub Region</label>
          <FocusSelect
            value={region}
            onChange={(e) => {
              setRegion(e.target.value);
              markDirty();
            }}
          >
            <option value="us-east-1">us-east-1 (N. Virginia)</option>
            <option value="eu-west-1">eu-west-1 (Ireland)</option>
            <option value="ap-southeast-1">ap-southeast-1 (Singapore)</option>
          </FocusSelect>
        </div>
      </div>

      {/* Timezone — full width */}
      <div>
        <label style={fieldLabelStyle}>Timezone</label>
        <FocusSelect
          value={timezone}
          onChange={(e) => {
            setTimezone(e.target.value);
            markDirty();
          }}
          style={{ maxWidth: 280 }}
        >
          <option value="UTC">UTC</option>
          <option value="America/New_York">America/New_York (UTC-5)</option>
          <option value="Europe/London">Europe/London (UTC+0)</option>
          <option value="Asia/Tokyo">Asia/Tokyo (UTC+9)</option>
        </FocusSelect>
      </div>

      {/* Catalog Auto-Publish toggle */}
      <div>
        <label style={fieldLabelStyle}>Catalog Auto-Publish</label>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginTop: 2 }}>
          <button
            type="button"
            role="switch"
            aria-checked={autoPublish}
            onClick={() => {
              setAutoPublish(!autoPublish);
              markDirty();
            }}
            style={{
              position: 'relative',
              display: 'inline-flex',
              height: 22,
              width: 40,
              alignItems: 'center',
              borderRadius: 999,
              border: 'none',
              cursor: 'pointer',
              background: autoPublish ? 'var(--accent)' : 'var(--border)',
              transition: 'background 0.15s',
              flexShrink: 0,
              padding: 0,
            }}
          >
            <span
              style={{
                display: 'inline-block',
                height: 16,
                width: 16,
                borderRadius: '50%',
                background: 'var(--bg-page)',
                transform: autoPublish ? 'translateX(20px)' : 'translateX(3px)',
                transition: 'transform 0.15s',
              }}
            />
          </button>
          <span
            style={{
              fontSize: 13,
              color: 'var(--text-secondary)',
              fontFamily: 'var(--font-sans)',
            }}
          >
            Automatically publish new catalog entries to all connected PMs
          </span>
        </div>
      </div>

      {/* Button row */}
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
          onClick={handleDiscard}
          disabled={!isDirty || saving}
          style={{
            background: 'transparent',
            color: 'var(--text-muted)',
            border: '1px solid var(--border-strong, var(--border))',
            borderRadius: 6,
            fontSize: 13,
            fontWeight: 500,
            padding: '7px 14px',
            cursor: !isDirty || saving ? 'not-allowed' : 'pointer',
            opacity: !isDirty || saving ? 0.5 : 1,
            fontFamily: 'var(--font-sans)',
          }}
        >
          Discard
        </button>
        <button
          type="button"
          onClick={handleSave}
          disabled={saving || !isDirty}
          style={{
            background:
              saving || !isDirty
                ? 'color-mix(in srgb, var(--accent) 40%, transparent)'
                : 'var(--accent)',
            color: 'var(--btn-accent-text, var(--text-emphasis))',
            border: 'none',
            borderRadius: 6,
            fontSize: 13,
            fontWeight: 600,
            padding: '7px 14px',
            cursor: saving || !isDirty ? 'not-allowed' : 'pointer',
            fontFamily: 'var(--font-sans)',
          }}
        >
          {saving ? 'Saving...' : 'Save Changes'}
        </button>
      </div>
    </div>
  );
};
