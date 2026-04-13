import { useState, useEffect, useCallback, forwardRef } from 'react';
import { Copy, Search, Loader2, Wifi, HardDrive, Package, Info } from 'lucide-react';
import { Switch } from '@patchiq/ui';
import {
  useSettings,
  useUpdateSettings,
  useTriggerScan,
  type SettingsUpdateBody,
} from '../../api/hooks/useSettings';
import { useAgentStatus } from '../../api/hooks/useStatus';
import { toast } from 'sonner';

// ---------------------------------------------------------------------------
// Style constants — PM GeneralSettingsPage pattern
// ---------------------------------------------------------------------------

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

const sectionHeaderStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  gap: 8,
  paddingBottom: 12,
  borderBottom: '1px solid var(--border)',
  marginBottom: 20,
};

const sectionTitleStyle: React.CSSProperties = {
  fontSize: 15,
  fontWeight: 600,
  color: 'var(--text-emphasis)',
  letterSpacing: '-0.01em',
  margin: 0,
};

const dividerStyle: React.CSSProperties = {
  borderTop: '1px solid var(--border)',
  marginTop: 32,
  marginBottom: 32,
};

// ---------------------------------------------------------------------------
// Focus-aware input/select
// ---------------------------------------------------------------------------

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
  const { style, children, disabled, ...rest } = props;
  return (
    <select
      disabled={disabled}
      style={{
        ...inputBaseStyle,
        cursor: disabled ? 'not-allowed' : 'pointer',
        appearance: 'none',
        backgroundImage: selectChevron,
        backgroundRepeat: 'no-repeat',
        backgroundPosition: 'right 10px center',
        paddingRight: 30,
        ...(focused ? inputFocusStyle : {}),
        ...(disabled ? { opacity: 0.5 } : {}),
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

// ---------------------------------------------------------------------------
// Field layout helpers
// ---------------------------------------------------------------------------

function FieldRow({ children }: { children: React.ReactNode }) {
  return <div style={{ marginBottom: 20 }}>{children}</div>;
}

function TwoCol({ children }: { children: React.ReactNode }) {
  return (
    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16, marginBottom: 20 }}>
      {children}
    </div>
  );
}

/** Toggle row: label+description on left, control on right. */
function ToggleRow({
  label,
  description,
  children,
}: {
  label: string;
  description: string;
  children: React.ReactNode;
}) {
  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        padding: '10px 0',
        borderBottom: '1px solid var(--border-faint)',
      }}
    >
      <div>
        <div style={{ fontSize: 13, color: 'var(--text-emphasis)', fontWeight: 500 }}>{label}</div>
        <div style={{ fontSize: 11, color: 'var(--text-muted)', marginTop: 2 }}>{description}</div>
      </div>
      {children}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Read-only info row
// ---------------------------------------------------------------------------

function CopyableValue({ value }: { value: string }) {
  function copy() {
    navigator.clipboard.writeText(value).catch(() => {});
    toast.success('Copied to clipboard');
  }
  return (
    <span style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
      <span
        style={{
          fontFamily: 'var(--font-mono)',
          fontSize: 11,
          maxWidth: 260,
          overflow: 'hidden',
          textOverflow: 'ellipsis',
          whiteSpace: 'nowrap',
          color: 'var(--text-emphasis)',
        }}
      >
        {value || '\u2014'}
      </span>
      {value && (
        <button
          type="button"
          onClick={copy}
          style={{
            background: 'none',
            border: 'none',
            cursor: 'pointer',
            color: 'var(--text-muted)',
            padding: 0,
            flexShrink: 0,
          }}
        >
          <Copy style={{ width: 12, height: 12 }} />
        </button>
      )}
    </span>
  );
}

function InfoRow({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        padding: '8px 0',
        borderBottom: '1px solid var(--border-faint)',
      }}
    >
      <span style={{ fontSize: 13, color: 'var(--text-muted)', flexShrink: 0 }}>{label}</span>
      <span style={{ fontSize: 13, marginLeft: 16, textAlign: 'right' }}>{value}</span>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const SCAN_INTERVALS = [
  { label: '1 hour', value: '1h' },
  { label: '3 hours', value: '3h' },
  { label: '6 hours', value: '6h' },
  { label: '12 hours', value: '12h' },
  { label: '24 hours', value: '24h' },
];

const LOG_LEVELS = ['debug', 'info', 'warn', 'error'] as const;

const HEARTBEAT_INTERVALS = [
  { label: '10 seconds', value: '10s' },
  { label: '30 seconds', value: '30s' },
  { label: '1 minute', value: '1m' },
  { label: '2 minutes', value: '2m' },
  { label: '5 minutes', value: '5m' },
];

const BANDWIDTH_MIN = 64;
const BANDWIDTH_MAX = 102400;

const MAX_CONCURRENT = [1, 2, 3, 4];

const LOG_RETENTION_OPTIONS = [
  { label: '7 days', value: 7 },
  { label: '14 days', value: 14 },
  { label: '30 days', value: 30 },
  { label: '60 days', value: 60 },
  { label: '90 days', value: 90 },
];

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------

export const SettingsPage = () => {
  const { data: status } = useAgentStatus();
  const { data, isLoading, isError, refetch } = useSettings();
  const updateSettings = useUpdateSettings();
  const triggerScan = useTriggerScan();

  const [scanInterval, setScanInterval] = useState('');
  const [logLevel, setLogLevel] = useState('info');
  const [autoDeploy, setAutoDeploy] = useState(false);
  const [scanCooldown, setScanCooldown] = useState(false);

  const [heartbeatInterval, setHeartbeatInterval] = useState('30s');
  const [bandwidthLimit, setBandwidthLimit] = useState(0);
  const [proxyUrl, setProxyUrl] = useState('');
  const [offlineMode, setOfflineMode] = useState(false);
  const [maxConcurrentInstalls, setMaxConcurrentInstalls] = useState(1);
  const [autoRebootWindow, setAutoRebootWindow] = useState('');
  const [logRetentionDays, setLogRetentionDays] = useState(30);

  const [hasChanges, setHasChanges] = useState(false);

  useEffect(() => {
    if (data) {
      setScanInterval(data.scan_interval);
      setLogLevel(data.log_level ?? 'info');
      setAutoDeploy(data.auto_deploy);
      setHeartbeatInterval(data.heartbeat_interval ?? '30s');
      setBandwidthLimit(data.bandwidth_limit_kbps ?? 0);
      setProxyUrl(data.proxy_url ?? '');
      setOfflineMode(data.offline_mode ?? false);
      setMaxConcurrentInstalls(data.max_concurrent_installs ?? 1);
      setAutoRebootWindow(data.auto_reboot_window ?? '');
      setLogRetentionDays(data.log_retention_days ?? 30);
      setHasChanges(false);
    }
  }, [data]);

  const detectChanges = useCallback(
    (
      overrides: Partial<{
        scanInterval: string;
        logLevel: string;
        autoDeploy: boolean;
        heartbeatInterval: string;
        bandwidthLimit: number;
        proxyUrl: string;
        offlineMode: boolean;
        maxConcurrentInstalls: number;
        autoRebootWindow: string;
        logRetentionDays: number;
      }>,
    ) => {
      if (!data) return;
      const si = overrides.scanInterval ?? scanInterval;
      const ll = overrides.logLevel ?? logLevel;
      const ad = overrides.autoDeploy ?? autoDeploy;
      const hi = overrides.heartbeatInterval ?? heartbeatInterval;
      const bl = overrides.bandwidthLimit ?? bandwidthLimit;
      const pu = overrides.proxyUrl ?? proxyUrl;
      const om = overrides.offlineMode ?? offlineMode;
      const mc = overrides.maxConcurrentInstalls ?? maxConcurrentInstalls;
      const ar = overrides.autoRebootWindow ?? autoRebootWindow;
      const lr = overrides.logRetentionDays ?? logRetentionDays;

      const changed =
        si !== data.scan_interval ||
        ll !== (data.log_level ?? 'info') ||
        ad !== data.auto_deploy ||
        hi !== (data.heartbeat_interval ?? '30s') ||
        bl !== (data.bandwidth_limit_kbps ?? 0) ||
        pu !== (data.proxy_url ?? '') ||
        om !== (data.offline_mode ?? false) ||
        mc !== (data.max_concurrent_installs ?? 1) ||
        ar !== (data.auto_reboot_window ?? '') ||
        lr !== (data.log_retention_days ?? 30);

      setHasChanges(changed);
    },
    [
      data,
      scanInterval,
      logLevel,
      autoDeploy,
      heartbeatInterval,
      bandwidthLimit,
      proxyUrl,
      offlineMode,
      maxConcurrentInstalls,
      autoRebootWindow,
      logRetentionDays,
    ],
  );

  function update<T>(setter: (v: T) => void, key: string, value: T) {
    setter(value);
    detectChanges({ [key]: value });
  }

  function handleSave() {
    if (!data) return;

    if (
      bandwidthLimit !== 0 &&
      (bandwidthLimit < BANDWIDTH_MIN || bandwidthLimit > BANDWIDTH_MAX)
    ) {
      toast.error(
        `Bandwidth must be 0 (unlimited) or between ${BANDWIDTH_MIN} and ${BANDWIDTH_MAX} Kbps`,
      );
      return;
    }

    const body: SettingsUpdateBody = {};
    if (scanInterval !== data.scan_interval) body.scan_interval = scanInterval;
    if (logLevel !== (data.log_level ?? 'info')) body.log_level = logLevel;
    if (autoDeploy !== data.auto_deploy) body.auto_deploy = autoDeploy;
    if (heartbeatInterval !== (data.heartbeat_interval ?? '30s'))
      body.heartbeat_interval = heartbeatInterval;
    if (bandwidthLimit !== (data.bandwidth_limit_kbps ?? 0))
      body.bandwidth_limit_kbps = bandwidthLimit;
    if (proxyUrl !== (data.proxy_url ?? '')) body.proxy_url = proxyUrl;
    if (offlineMode !== (data.offline_mode ?? false)) body.offline_mode = offlineMode;
    if (maxConcurrentInstalls !== (data.max_concurrent_installs ?? 1))
      body.max_concurrent_installs = maxConcurrentInstalls;
    if (autoRebootWindow !== (data.auto_reboot_window ?? ''))
      body.auto_reboot_window = autoRebootWindow;
    if (logRetentionDays !== (data.log_retention_days ?? 30))
      body.log_retention_days = logRetentionDays;

    updateSettings.mutate(body, {
      onSuccess: () => {
        toast.success('Settings saved successfully');
        setHasChanges(false);
      },
      onError: (err) => {
        toast.error(`Failed to save settings: ${err.message}`);
      },
    });
  }

  function handleDiscard() {
    if (data) {
      setScanInterval(data.scan_interval);
      setLogLevel(data.log_level ?? 'info');
      setAutoDeploy(data.auto_deploy);
      setHeartbeatInterval(data.heartbeat_interval ?? '30s');
      setBandwidthLimit(data.bandwidth_limit_kbps ?? 0);
      setProxyUrl(data.proxy_url ?? '');
      setOfflineMode(data.offline_mode ?? false);
      setMaxConcurrentInstalls(data.max_concurrent_installs ?? 1);
      setAutoRebootWindow(data.auto_reboot_window ?? '');
      setLogRetentionDays(data.log_retention_days ?? 30);
      setHasChanges(false);
    }
  }

  function handleTriggerScan() {
    setScanCooldown(true);
    triggerScan.mutate(undefined, {
      onSuccess: () => {
        toast.success('Scan triggered');
      },
      onError: (err) => {
        toast.error(`Failed to trigger scan: ${err.message}`);
      },
      onSettled: () => {
        setTimeout(() => setScanCooldown(false), 5000);
      },
    });
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

  if (isError) {
    return (
      <div style={{ padding: '28px 40px 80px', maxWidth: 680 }}>
        <p
          style={{ fontSize: 13, color: 'var(--signal-critical)', fontFamily: 'var(--font-sans)' }}
        >
          Failed to load settings.{' '}
          <button
            type="button"
            onClick={() => refetch()}
            style={{
              color: 'var(--accent)',
              background: 'none',
              border: 'none',
              cursor: 'pointer',
              fontSize: 13,
              padding: 0,
            }}
          >
            Retry
          </button>
        </p>
      </div>
    );
  }

  if (!data) return null;

  return (
    <div style={{ padding: '28px 40px 80px', maxWidth: 680 }}>
      {/* Page header */}
      <div style={{ paddingBottom: 16, borderBottom: '1px solid var(--border)', marginBottom: 32 }}>
        <h2
          style={{
            fontSize: 18,
            fontWeight: 600,
            color: 'var(--text-emphasis)',
            letterSpacing: '-0.02em',
            margin: 0,
          }}
        >
          Agent Settings
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
          Communication, patch management, storage, and diagnostic settings for this agent.
        </p>
      </div>

      <form
        onSubmit={(e) => {
          e.preventDefault();
          handleSave();
        }}
        style={{ display: 'flex', flexDirection: 'column' }}
      >
        {/* ── Communication ─────────────────────────────────────────── */}
        <div style={sectionHeaderStyle}>
          <Wifi style={{ width: 15, height: 15, color: 'var(--text-muted)' }} />
          <h3 style={sectionTitleStyle}>Communication</h3>
        </div>

        <TwoCol>
          <div>
            <label style={fieldLabelStyle}>Heartbeat Interval</label>
            <FocusSelect
              value={heartbeatInterval}
              onChange={(e) => update(setHeartbeatInterval, 'heartbeatInterval', e.target.value)}
            >
              {HEARTBEAT_INTERVALS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </FocusSelect>
            <p style={hintStyle}>How often heartbeats are sent to the server.</p>
          </div>

          <div>
            <label style={fieldLabelStyle}>Bandwidth Limit (Kbps)</label>
            <FocusInput
              type="number"
              value={bandwidthLimit}
              onChange={(e) => {
                const v = Number(e.target.value);
                if (!Number.isNaN(v)) update(setBandwidthLimit, 'bandwidthLimit', v);
              }}
              min={0}
              max={BANDWIDTH_MAX}
              placeholder="0 = Unlimited"
            />
            <p style={hintStyle}>
              Min {BANDWIDTH_MIN} Kbps or 0 for unlimited. Max {BANDWIDTH_MAX} Kbps.
            </p>
          </div>
        </TwoCol>

        <FieldRow>
          <label style={fieldLabelStyle}>Proxy URL</label>
          <FocusInput
            type="text"
            value={proxyUrl}
            onChange={(e) => update(setProxyUrl, 'proxyUrl', e.target.value)}
            placeholder="http://proxy:8080"
          />
          <p style={hintStyle}>HTTP proxy for server communication (optional).</p>
        </FieldRow>

        <ToggleRow label="Offline Mode" description="Pauses all server communication.">
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            {offlineMode && (
              <span style={{ fontSize: 11, color: 'var(--signal-warning)' }}>Sync paused</span>
            )}
            <Switch
              checked={offlineMode}
              onCheckedChange={(val) => update(setOfflineMode, 'offlineMode', val)}
            />
          </div>
        </ToggleRow>

        <div style={dividerStyle} />

        {/* ── Patch Management ──────────────────────────────────────── */}
        <div style={sectionHeaderStyle}>
          <Package style={{ width: 15, height: 15, color: 'var(--text-muted)' }} />
          <h3 style={sectionTitleStyle}>Patch Management</h3>
        </div>

        <TwoCol>
          <div>
            <label style={fieldLabelStyle}>Scan Interval</label>
            <FocusSelect
              value={scanInterval}
              onChange={(e) => update(setScanInterval, 'scanInterval', e.target.value)}
            >
              {SCAN_INTERVALS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </FocusSelect>
            <p style={hintStyle}>How often the agent scans for available patches.</p>
          </div>

          <div>
            <label style={fieldLabelStyle}>Max Concurrent Installs</label>
            <FocusSelect
              value={maxConcurrentInstalls}
              onChange={(e) =>
                update(setMaxConcurrentInstalls, 'maxConcurrentInstalls', Number(e.target.value))
              }
            >
              {MAX_CONCURRENT.map((n) => (
                <option key={n} value={n}>
                  {n}
                </option>
              ))}
            </FocusSelect>
            <p style={hintStyle}>Maximum parallel patch installations.</p>
          </div>
        </TwoCol>

        <FieldRow>
          <label style={fieldLabelStyle}>Auto-Reboot Window</label>
          <FocusInput
            type="text"
            value={autoRebootWindow}
            onChange={(e) => update(setAutoRebootWindow, 'autoRebootWindow', e.target.value)}
            placeholder="02:00-05:00"
            style={{ maxWidth: 200 }}
          />
          <p style={hintStyle}>
            Time window when reboots are allowed after patch install (optional).
          </p>
        </FieldRow>

        <ToggleRow
          label="Auto Deploy"
          description="Automatically install approved patches without manual intervention."
        >
          <Switch
            checked={autoDeploy}
            onCheckedChange={(val) => update(setAutoDeploy, 'autoDeploy', val)}
          />
        </ToggleRow>

        <div style={dividerStyle} />

        {/* ── Storage & Logging ─────────────────────────────────────── */}
        <div style={sectionHeaderStyle}>
          <HardDrive style={{ width: 15, height: 15, color: 'var(--text-muted)' }} />
          <h3 style={sectionTitleStyle}>Storage & Logging</h3>
        </div>

        <TwoCol>
          <div>
            <label style={fieldLabelStyle}>Log Level</label>
            <FocusSelect
              value={logLevel}
              onChange={(e) => update(setLogLevel, 'logLevel', e.target.value)}
            >
              {LOG_LEVELS.map((lvl) => (
                <option key={lvl} value={lvl}>
                  {lvl.charAt(0).toUpperCase() + lvl.slice(1)}
                </option>
              ))}
            </FocusSelect>
            <p style={hintStyle}>Controls the verbosity of agent log output.</p>
          </div>

          <div>
            <label style={fieldLabelStyle}>Log Retention</label>
            <FocusSelect
              value={logRetentionDays}
              onChange={(e) =>
                update(setLogRetentionDays, 'logRetentionDays', Number(e.target.value))
              }
            >
              {LOG_RETENTION_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </FocusSelect>
            <p style={hintStyle}>Retention for logs, outbox history, and patch history.</p>
          </div>
        </TwoCol>

        {/* ── Save / Discard row ────────────────────────────────────── */}
        <div
          style={{
            borderTop: '1px solid var(--border)',
            paddingTop: 16,
            display: 'flex',
            gap: 8,
            justifyContent: 'flex-end',
            marginTop: 8,
          }}
        >
          <button
            type="button"
            onClick={handleDiscard}
            disabled={!hasChanges || updateSettings.isPending}
            style={{
              background: 'transparent',
              color: 'var(--text-muted)',
              border: '1px solid var(--border-strong, var(--border))',
              borderRadius: 6,
              fontSize: 13,
              fontWeight: 500,
              padding: '7px 14px',
              cursor: !hasChanges || updateSettings.isPending ? 'not-allowed' : 'pointer',
              opacity: !hasChanges || updateSettings.isPending ? 0.5 : 1,
              fontFamily: 'var(--font-sans)',
            }}
          >
            Discard
          </button>
          <button
            type="submit"
            disabled={!hasChanges || updateSettings.isPending}
            style={{
              background:
                !hasChanges || updateSettings.isPending
                  ? 'color-mix(in srgb, var(--accent) 40%, transparent)'
                  : 'var(--accent)',
              color: 'var(--btn-accent-text, #000)',
              border: 'none',
              borderRadius: 6,
              fontSize: 13,
              fontWeight: 600,
              padding: '7px 14px',
              cursor: !hasChanges || updateSettings.isPending ? 'not-allowed' : 'pointer',
              fontFamily: 'var(--font-sans)',
            }}
          >
            {updateSettings.isPending ? (
              <span style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                <Loader2 style={{ width: 14, height: 14 }} className="animate-spin" />
                Saving...
              </span>
            ) : (
              'Save Changes'
            )}
          </button>
        </div>
      </form>

      {/* ── Actions ────────────────────────────────────────────────── */}
      <div style={{ ...dividerStyle, marginTop: 40 }} />
      <div style={sectionHeaderStyle}>
        <Search style={{ width: 15, height: 15, color: 'var(--text-muted)' }} />
        <h3 style={sectionTitleStyle}>Actions</h3>
      </div>

      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '10px 0',
          borderBottom: '1px solid var(--border-faint)',
        }}
      >
        <div>
          <div style={{ fontSize: 13, color: 'var(--text-emphasis)', fontWeight: 500 }}>
            Trigger Scan Now
          </div>
          <div style={{ fontSize: 11, color: 'var(--text-muted)', marginTop: 2 }}>
            Immediately scan for available patches on this endpoint.
          </div>
        </div>
        <button
          type="button"
          onClick={handleTriggerScan}
          disabled={scanCooldown || triggerScan.isPending}
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 6,
            background: 'transparent',
            color: 'var(--text-emphasis)',
            border: '1px solid var(--border)',
            borderRadius: 6,
            fontSize: 13,
            fontWeight: 500,
            padding: '7px 14px',
            cursor: scanCooldown || triggerScan.isPending ? 'not-allowed' : 'pointer',
            opacity: scanCooldown || triggerScan.isPending ? 0.6 : 1,
            fontFamily: 'var(--font-sans)',
          }}
        >
          {triggerScan.isPending ? (
            <Loader2 style={{ width: 14, height: 14 }} className="animate-spin" />
          ) : (
            <Search style={{ width: 14, height: 14 }} />
          )}
          {scanCooldown ? 'Scan Triggered' : 'Scan Now'}
        </button>
      </div>

      {/* ── Agent Information ──────────────────────────────────────── */}
      <div style={{ ...dividerStyle, marginTop: 40 }} />
      <div style={{ ...sectionHeaderStyle, justifyContent: 'space-between' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <Info style={{ width: 15, height: 15, color: 'var(--text-muted)' }} />
          <h3 style={sectionTitleStyle}>Agent Information</h3>
        </div>
        <span
          style={{
            fontSize: 10,
            color: 'var(--text-muted)',
            background: 'var(--bg-inset)',
            padding: '2px 8px',
            borderRadius: 4,
            border: '1px solid var(--border)',
            fontFamily: 'var(--font-mono)',
            letterSpacing: '0.04em',
          }}
        >
          Read-only
        </span>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0 24px' }}>
        <InfoRow
          label="Agent Version"
          value={
            <span
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 11,
                color: 'var(--text-emphasis)',
              }}
            >
              {data.agent_version}
            </span>
          }
        />
        <InfoRow
          label="Agent ID"
          value={<CopyableValue value={status?.agent_id ?? data.agent_id} />}
        />
        <InfoRow label="Server URL" value={<CopyableValue value={data.server_url} />} />
        <InfoRow label="Data Directory" value={<CopyableValue value={data.data_dir} />} />
        <InfoRow label="Database Path" value={<CopyableValue value={data.db_path} />} />
        <InfoRow
          label="HTTP Address"
          value={
            <span
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 11,
                color: 'var(--text-emphasis)',
              }}
            >
              {data.http_addr}
            </span>
          }
        />
        <InfoRow label="Config File" value={<CopyableValue value={data.config_file} />} />
        <InfoRow
          label="Server Connection"
          value={
            <span style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
              <span
                style={{
                  width: 8,
                  height: 8,
                  borderRadius: '50%',
                  background: offlineMode ? 'var(--signal-warning)' : 'var(--signal-healthy)',
                  display: 'inline-block',
                }}
              />
              <span
                style={{
                  color: offlineMode ? 'var(--signal-warning)' : 'var(--signal-healthy)',
                  fontSize: 12,
                }}
              >
                {offlineMode ? 'Offline' : 'Connected'}
              </span>
            </span>
          }
        />
      </div>
    </div>
  );
};
