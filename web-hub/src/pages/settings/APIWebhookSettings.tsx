import { useState, useEffect } from 'react';
import { Eye, EyeOff, Copy } from 'lucide-react';

interface APIWebhookSettingsProps {
  settings: Record<string, unknown>;
  onSave: (settings: Record<string, unknown>) => void;
  saving: boolean;
}

const EVENT_TYPES = [
  { key: 'client.connected', label: 'client.connected', desc: 'When a PM instance connects' },
  {
    key: 'client.disconnected',
    label: 'client.disconnected',
    desc: 'When a PM instance disconnects',
  },
  { key: 'feed.sync_failed', label: 'feed.sync_failed', desc: 'When a feed sync fails' },
  {
    key: 'deployment.completed',
    label: 'deployment.completed',
    desc: 'When a fleet deployment completes',
  },
  {
    key: 'license.expiring',
    label: 'license.expiring',
    desc: 'When a license expires within 30 days',
  },
  {
    key: 'catalog.updated',
    label: 'catalog.updated',
    desc: 'When catalog entries are added/updated',
  },
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

const hintStyle: React.CSSProperties = {
  fontSize: 11,
  color: 'var(--text-faint)',
  marginTop: 4,
};

function FocusInput(props: React.InputHTMLAttributes<HTMLInputElement>) {
  const [focused, setFocused] = useState(false);
  const { style, disabled, ...rest } = props;
  return (
    <input
      disabled={disabled}
      style={{
        ...inputBaseStyle,
        ...(focused && !disabled ? inputFocusStyle : {}),
        ...(disabled ? { opacity: 0.5, cursor: 'not-allowed' } : {}),
        ...style,
      }}
      onFocus={() => setFocused(true)}
      onBlur={() => setFocused(false)}
      {...rest}
    />
  );
}

export const APIWebhookSettings = ({ settings, onSave, saving }: APIWebhookSettingsProps) => {
  const apiEndpoint = `${window.location.origin}/api/v1`;
  const [apiKey, setApiKey] = useState('');
  const [showApiKey, setShowApiKey] = useState(false);
  const [webhookUrl, setWebhookUrl] = useState('');
  const [subscriptions, setSubscriptions] = useState<Set<string>>(new Set());
  const [copied, setCopied] = useState(false);
  const [isDirty, setIsDirty] = useState(false);

  useEffect(() => {
    if (settings['api.api_key'] != null) setApiKey(String(settings['api.api_key']));
    if (settings['api.webhook_url'] != null) setWebhookUrl(String(settings['api.webhook_url']));
    if (Array.isArray(settings['api.event_subscriptions']))
      setSubscriptions(new Set(settings['api.event_subscriptions'] as string[]));
    setIsDirty(false);
  }, [settings]);

  const toggleSubscription = (key: string) => {
    setSubscriptions((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
    setIsDirty(true);
  };

  const handleCopy = () => {
    void navigator.clipboard.writeText(apiEndpoint);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleSave = () => {
    onSave({ 'api.webhook_url': webhookUrl, 'api.event_subscriptions': Array.from(subscriptions) });
    setIsDirty(false);
  };

  const handleDiscard = () => {
    if (settings['api.webhook_url'] != null) setWebhookUrl(String(settings['api.webhook_url']));
    if (Array.isArray(settings['api.event_subscriptions']))
      setSubscriptions(new Set(settings['api.event_subscriptions'] as string[]));
    setIsDirty(false);
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 24 }}>
      {/* Hub API Endpoint — read-only */}
      <div>
        <label style={fieldLabelStyle}>Hub API Endpoint</label>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <input
            type="text"
            value={apiEndpoint}
            readOnly
            style={{
              ...inputBaseStyle,
              flex: 1,
              background: 'var(--bg-inset)',
              color: 'var(--text-muted)',
              cursor: 'default',
              fontFamily: 'var(--font-mono)',
            }}
          />
          <button
            type="button"
            onClick={handleCopy}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 6,
              height: 36,
              padding: '0 12px',
              background: 'transparent',
              border: '1px solid var(--border)',
              borderRadius: 6,
              fontSize: 13,
              color: 'var(--text-muted)',
              cursor: 'pointer',
              fontFamily: 'var(--font-sans)',
              flexShrink: 0,
            }}
          >
            <Copy size={14} />
            {copied ? 'Copied!' : 'Copy'}
          </button>
        </div>
      </div>

      {/* API Key — read-only with show/hide */}
      <div>
        <label style={fieldLabelStyle}>API Key</label>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <div style={{ position: 'relative', flex: 1 }}>
            <input
              type={showApiKey ? 'text' : 'password'}
              value={apiKey}
              readOnly
              style={{
                ...inputBaseStyle,
                paddingRight: 40,
                fontFamily: 'var(--font-mono)',
                background: 'var(--bg-inset)',
                color: 'var(--text-muted)',
                cursor: 'default',
              }}
            />
            <button
              type="button"
              onClick={() => setShowApiKey(!showApiKey)}
              style={{
                position: 'absolute',
                right: 10,
                top: '50%',
                transform: 'translateY(-50%)',
                background: 'none',
                border: 'none',
                cursor: 'pointer',
                color: 'var(--text-muted)',
                padding: 0,
                display: 'flex',
              }}
            >
              {showApiKey ? <EyeOff size={14} /> : <Eye size={14} />}
            </button>
          </div>
          <button
            type="button"
            disabled
            title="Coming soon"
            style={{
              height: 36,
              padding: '0 12px',
              background: 'transparent',
              border: '1px solid var(--signal-critical)',
              borderRadius: 6,
              fontSize: 13,
              color: 'var(--signal-critical)',
              cursor: 'not-allowed',
              opacity: 0.5,
              fontFamily: 'var(--font-sans)',
              flexShrink: 0,
            }}
          >
            Rotate
          </button>
        </div>
        <p style={hintStyle}>Used to authenticate PM instances connecting to this Hub.</p>
      </div>

      {/* Webhook URL */}
      <div>
        <label style={fieldLabelStyle}>Webhook URL</label>
        <FocusInput
          type="text"
          value={webhookUrl}
          onChange={(e) => {
            setWebhookUrl(e.target.value);
            setIsDirty(true);
          }}
          placeholder="https://example.com/webhook"
          style={{ fontFamily: 'var(--font-mono)' }}
        />
        <p style={hintStyle}>Outbound POST for subscribed events. Leave blank to disable.</p>
      </div>

      {/* Event Subscriptions */}
      <div>
        <label style={fieldLabelStyle}>Event Subscriptions</label>
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: '1fr 1fr',
            gap: 8,
          }}
        >
          {EVENT_TYPES.map((event) => (
            <label
              key={event.key}
              style={{
                display: 'flex',
                alignItems: 'flex-start',
                gap: 10,
                padding: '10px 12px',
                border: '1px solid var(--border)',
                borderRadius: 6,
                cursor: 'pointer',
              }}
            >
              <input
                type="checkbox"
                checked={subscriptions.has(event.key)}
                onChange={() => toggleSubscription(event.key)}
                style={{ accentColor: 'var(--accent)', marginTop: 2, flexShrink: 0 }}
              />
              <div>
                <p
                  style={{
                    fontSize: 12,
                    fontWeight: 500,
                    color: 'var(--text-primary)',
                    fontFamily: 'var(--font-mono)',
                    margin: 0,
                  }}
                >
                  {event.label}
                </p>
                <p
                  style={{
                    fontSize: 11,
                    color: 'var(--text-muted)',
                    fontFamily: 'var(--font-sans)',
                    margin: '2px 0 0',
                  }}
                >
                  {event.desc}
                </p>
              </div>
            </label>
          ))}
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
