import { Link } from 'react-router';
import { Webhook, Loader2, Mail, MessageSquare, Send } from 'lucide-react';
import { Skeleton } from '@patchiq/ui';
import { toast } from 'sonner';
import { useChannelByType, useTestChannelByType } from '../../api/hooks/useChannelByType';
import {
  useNotificationPreferences,
  useUpdatePreferences,
  useDigestConfig,
  useUpdateDigestConfig,
  useTestDigest,
} from '../../api/hooks/useNotifications';
import type { NotificationPreferencesResponse } from '../../api/hooks/useNotifications';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type ChannelStatus = 'connected' | 'not_configured' | 'error' | 'loading';

interface ChannelCardProps {
  type: string;
  label: string;
  icon: React.ReactNode;
  iconBg: string;
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const CHANNELS: ChannelCardProps[] = [
  {
    type: 'webhook',
    label: 'Webhook',
    icon: <Webhook style={{ width: 16, height: 16, color: 'var(--signal-info)' }} />,
    iconBg: 'color-mix(in srgb, var(--signal-info) 10%, transparent)',
  },
  {
    type: 'slack',
    label: 'Slack',
    icon: <MessageSquare style={{ width: 16, height: 16, color: 'var(--signal-warning)' }} />,
    iconBg: 'color-mix(in srgb, var(--signal-warning) 10%, transparent)',
  },
  {
    type: 'email',
    label: 'Email',
    icon: <Mail style={{ width: 16, height: 16, color: 'var(--signal-healthy)' }} />,
    iconBg: 'color-mix(in srgb, var(--signal-healthy) 10%, transparent)',
  },
];

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function formatTestedAgo(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const hours = Math.floor(diff / 3_600_000);
  if (hours < 1) return 'just now';
  if (hours === 1) return '1h ago';
  return `${hours}h ago`;
}

// ---------------------------------------------------------------------------
// ChannelCard (unchanged — same as before)
// ---------------------------------------------------------------------------

function ChannelCard({ type, label, icon, iconBg }: ChannelCardProps) {
  const { data, isLoading, error } = useChannelByType(type);
  const testMutation = useTestChannelByType(type);

  const is404 =
    error &&
    ((error as { status?: number })?.status === 404 ||
      (error as { response?: { status?: number } })?.response?.status === 404);

  const status: ChannelStatus = isLoading
    ? 'loading'
    : is404 || !data
      ? 'not_configured'
      : error
        ? 'error'
        : 'connected';

  const isOn = status === 'connected';

  function handleTest() {
    testMutation.mutate(undefined, {
      onSuccess: (result) => {
        if (result.success) {
          toast.success(`${label} test successful`);
        } else {
          toast.error(`${label} test failed: ${result.error ?? 'Unknown error'}`);
        }
      },
      onError: (err) => {
        toast.error(`Failed to test ${label}: ${err.message}`);
      },
    });
  }

  const url = data?.name ?? (status === 'not_configured' ? 'Not configured' : undefined);

  return (
    <div
      style={{
        background: 'var(--bg-card)',
        border: '1px solid var(--border)',
        borderRadius: 10,
        padding: '14px 16px',
        display: 'flex',
        flexDirection: 'column',
        gap: 10,
        minWidth: 0,
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <div
            style={{
              width: 32,
              height: 32,
              borderRadius: 8,
              background: iconBg,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              flexShrink: 0,
            }}
          >
            {icon}
          </div>
          <span
            style={{
              fontSize: 13,
              fontWeight: 600,
              color: 'var(--text-primary)',
              fontFamily: 'var(--font-sans)',
            }}
          >
            {label}
          </span>
        </div>
        <div
          style={{
            width: 7,
            height: 7,
            borderRadius: '50%',
            background: isOn ? 'var(--signal-healthy)' : 'var(--border-strong)',
            boxShadow: isOn
              ? '0 0 0 2px color-mix(in srgb, var(--signal-healthy) 15%, transparent)'
              : 'none',
            flexShrink: 0,
          }}
        />
      </div>

      {status === 'loading' ? (
        <div style={{ height: 14, borderRadius: 4, background: 'var(--bg-inset)', width: '70%' }} />
      ) : (
        <p
          style={{
            fontSize: 11,
            fontFamily: 'var(--font-mono)',
            color: 'var(--text-muted)',
            margin: 0,
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
          }}
        >
          {url}
        </p>
      )}

      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <span style={{ fontSize: 10, color: 'var(--text-faint)', fontFamily: 'var(--font-sans)' }}>
          {isOn && data?.last_tested_at
            ? `Tested ${formatTestedAgo(data.last_tested_at)} \u00b7 ${data.last_test_status === 'success' ? '\u2713' : '\u2717'}`
            : '\u00a0'}
        </span>
        <button
          type="button"
          onClick={handleTest}
          disabled={testMutation.isPending || status !== 'connected'}
          style={{
            fontSize: 10,
            fontFamily: 'var(--font-mono)',
            color: 'var(--text-secondary)',
            background: 'transparent',
            border: '1px solid var(--border)',
            borderRadius: 5,
            padding: '3px 8px',
            cursor: testMutation.isPending || status !== 'connected' ? 'not-allowed' : 'pointer',
            opacity: testMutation.isPending || status !== 'connected' ? 0.4 : 1,
            transition: 'background 120ms, color 120ms',
          }}
          onMouseEnter={(e) => {
            if (status === 'connected' && !testMutation.isPending) {
              e.currentTarget.style.background = 'var(--accent)';
              e.currentTarget.style.color = 'var(--btn-accent-text, #fff)';
              e.currentTarget.style.borderColor = 'var(--accent)';
            }
          }}
          onMouseLeave={(e) => {
            e.currentTarget.style.background = 'transparent';
            e.currentTarget.style.color = 'var(--text-secondary)';
            e.currentTarget.style.borderColor = 'var(--border)';
          }}
        >
          {testMutation.isPending ? (
            <Loader2 style={{ width: 10, height: 10 }} className="animate-spin" />
          ) : (
            'Test'
          )}
        </button>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// ToggleCell — small checkbox-style toggle for the preference matrix
// ---------------------------------------------------------------------------

function ToggleCell({
  on,
  onChange,
  disabled,
}: {
  on: boolean;
  onChange: () => void;
  disabled?: boolean;
}) {
  return (
    <button
      type="button"
      onClick={onChange}
      disabled={disabled}
      style={{
        width: 16,
        height: 16,
        borderRadius: 3,
        background: on ? 'var(--accent)' : 'transparent',
        border: on ? '1px solid var(--accent)' : '1px solid var(--border-strong, var(--border))',
        cursor: disabled ? 'not-allowed' : 'pointer',
        padding: 0,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        transition: 'background 100ms, border-color 100ms',
        opacity: disabled ? 0.4 : 1,
        flexShrink: 0,
      }}
    >
      {on && (
        <svg width="10" height="10" viewBox="0 0 10 10" fill="none">
          <path
            d="M2 5L4 7L8 3"
            stroke="var(--btn-accent-text, #fff)"
            strokeWidth="1.5"
            strokeLinecap="round"
            strokeLinejoin="round"
          />
        </svg>
      )}
    </button>
  );
}

// ---------------------------------------------------------------------------
// UrgencyBadge
// ---------------------------------------------------------------------------

function UrgencyBadge({ urgency }: { urgency: string }) {
  const isImmediate = urgency === 'immediate';
  return (
    <span
      style={{
        fontSize: 9,
        fontWeight: 600,
        textTransform: 'uppercase',
        letterSpacing: '0.04em',
        padding: '2px 6px',
        borderRadius: 3,
        fontFamily: 'var(--font-mono)',
        background: isImmediate
          ? 'color-mix(in srgb, var(--signal-critical) 1%, transparent)'
          : 'color-mix(in srgb, var(--signal-info) 8%, transparent)',
        color: isImmediate ? 'var(--signal-critical)' : 'var(--signal-info)',
        border: isImmediate
          ? '1px solid color-mix(in srgb, var(--signal-critical) 1%, transparent)'
          : '1px solid color-mix(in srgb, var(--signal-info) 15%, transparent)',
      }}
    >
      {urgency}
    </span>
  );
}

// ---------------------------------------------------------------------------
// PreferenceMatrix — the real deal
// ---------------------------------------------------------------------------

function PreferenceMatrix({
  preferences,
  onToggle,
  saving,
}: {
  preferences: NotificationPreferencesResponse;
  onToggle: (triggerType: string, channel: 'email' | 'slack' | 'webhook') => void;
  saving: boolean;
}) {
  const channelCols: Array<{
    key: 'email_enabled' | 'slack_enabled' | 'webhook_enabled';
    label: string;
    channel: 'email' | 'slack' | 'webhook';
  }> = [
    { key: 'email_enabled', label: 'Email', channel: 'email' },
    { key: 'slack_enabled', label: 'Slack', channel: 'slack' },
    { key: 'webhook_enabled', label: 'Webhook', channel: 'webhook' },
  ];

  // Check which channels are configured
  const configuredChannels = new Set(
    (preferences.channels ?? []).filter((c) => c.configured).map((c) => c.type),
  );

  return (
    <div>
      {(preferences.categories ?? []).map((category) => (
        <div key={category.id} style={{ marginBottom: 20 }}>
          <div
            style={{
              fontSize: 11,
              fontWeight: 600,
              color: 'var(--text-muted)',
              fontFamily: 'var(--font-sans)',
              textTransform: 'uppercase',
              letterSpacing: '0.05em',
              marginBottom: 8,
            }}
          >
            {category.label}
          </div>
          <div
            style={{
              borderRadius: 8,
              border: '1px solid var(--border)',
              overflow: 'hidden',
            }}
          >
            {/* Header row */}
            <div
              style={{
                display: 'grid',
                gridTemplateColumns: '1fr 60px 60px 60px 70px',
                gap: 0,
                padding: '6px 12px',
                background: 'var(--bg-inset)',
                borderBottom: '1px solid var(--border)',
              }}
            >
              <span
                style={{
                  fontSize: 10,
                  fontWeight: 600,
                  color: 'var(--text-faint)',
                  fontFamily: 'var(--font-sans)',
                  textTransform: 'uppercase',
                  letterSpacing: '0.05em',
                }}
              >
                Event
              </span>
              {channelCols.map((col) => (
                <span
                  key={col.label}
                  style={{
                    fontSize: 10,
                    fontWeight: 600,
                    color: configuredChannels.has(col.channel)
                      ? 'var(--text-faint)'
                      : 'var(--text-faint)',
                    fontFamily: 'var(--font-sans)',
                    textTransform: 'uppercase',
                    letterSpacing: '0.05em',
                    textAlign: 'center',
                    opacity: configuredChannels.has(col.channel) ? 1 : 0.4,
                  }}
                >
                  {col.label}
                </span>
              ))}
              <span
                style={{
                  fontSize: 10,
                  fontWeight: 600,
                  color: 'var(--text-faint)',
                  fontFamily: 'var(--font-sans)',
                  textTransform: 'uppercase',
                  letterSpacing: '0.05em',
                  textAlign: 'center',
                }}
              >
                Urgency
              </span>
            </div>

            {/* Event rows */}
            {(category.events ?? []).map((event, i) => (
              <div
                key={event.trigger_type}
                style={{
                  display: 'grid',
                  gridTemplateColumns: '1fr 60px 60px 60px 70px',
                  gap: 0,
                  padding: '8px 12px',
                  borderBottom:
                    i < (category.events ?? []).length - 1 ? '1px solid var(--border)' : 'none',
                  alignItems: 'center',
                }}
              >
                <span
                  style={{
                    fontSize: 12,
                    color: 'var(--text-primary)',
                    fontFamily: 'var(--font-sans)',
                  }}
                >
                  {event.label}
                </span>
                {channelCols.map((col) => (
                  <div key={col.label} style={{ display: 'flex', justifyContent: 'center' }}>
                    <ToggleCell
                      on={event[col.key]}
                      onChange={() => onToggle(event.trigger_type, col.channel)}
                      disabled={saving || !configuredChannels.has(col.channel)}
                    />
                  </div>
                ))}
                <div style={{ display: 'flex', justifyContent: 'center' }}>
                  <UrgencyBadge urgency={event.urgency} />
                </div>
              </div>
            ))}
          </div>
          {category.description && (
            <p
              style={{
                fontSize: 10,
                color: 'var(--text-faint)',
                fontFamily: 'var(--font-sans)',
                marginTop: 4,
              }}
            >
              {category.description}
            </p>
          )}
        </div>
      ))}
    </div>
  );
}

// ---------------------------------------------------------------------------
// NotificationSettingsPage
// ---------------------------------------------------------------------------

export function NotificationSettingsPage() {
  const { data: prefs, isLoading: prefsLoading } = useNotificationPreferences();
  const updatePrefs = useUpdatePreferences();
  const { data: digestConfig, isLoading: digestLoading } = useDigestConfig();
  const updateDigest = useUpdateDigestConfig();
  const testDigest = useTestDigest();

  function handleToggle(triggerType: string, channel: 'email' | 'slack' | 'webhook') {
    if (!prefs?.categories) return;

    // Build channel_ids list and flip the toggled channel for this event
    const allEvents = prefs.categories.flatMap((c) => c.events ?? []);
    const channelMap = Object.fromEntries(
      (prefs.channels ?? []).filter((c) => c.channel_id).map((c) => [c.type, c.channel_id!]),
    );

    const updated = allEvents.map((ev) => {
      // For each event, build channel_ids from enabled channels
      let emailOn = ev.email_enabled;
      let slackOn = ev.slack_enabled;
      let webhookOn = ev.webhook_enabled;

      if (ev.trigger_type === triggerType) {
        if (channel === 'email') emailOn = !emailOn;
        if (channel === 'slack') slackOn = !slackOn;
        if (channel === 'webhook') webhookOn = !webhookOn;
      }

      const channelIds: string[] = [];
      if (emailOn && channelMap.email) channelIds.push(channelMap.email);
      if (slackOn && channelMap.slack) channelIds.push(channelMap.slack);
      if (webhookOn && channelMap.webhook) channelIds.push(channelMap.webhook);

      return {
        trigger_type: ev.trigger_type,
        enabled: channelIds.length > 0,
        channel_ids: channelIds,
        digest_frequency: ev.urgency === 'digest' ? 'daily' : 'immediate',
      };
    });

    updatePrefs.mutate(updated, {
      onError: (err) => toast.error(`Failed to update preferences: ${err.message}`),
    });
  }

  function handleDigestUpdate(field: string, value: string) {
    if (!digestConfig) return;
    updateDigest.mutate(
      { ...digestConfig, [field]: value },
      {
        onSuccess: () => toast.success('Digest settings updated'),
        onError: (err) => toast.error(`Failed to update digest: ${err.message}`),
      },
    );
  }

  function handleTestDigest() {
    testDigest.mutate(undefined, {
      onSuccess: () => toast.success('Test digest queued'),
      onError: (err) => toast.error(`Failed to send test digest: ${err.message}`),
    });
  }

  if (prefsLoading || digestLoading) {
    return (
      <div
        style={{
          padding: '28px 40px 80px',
          maxWidth: 680,
          display: 'flex',
          flexDirection: 'column',
          gap: 20,
        }}
      >
        <div>
          <Skeleton className="h-6 w-40" />
          <Skeleton className="h-4 w-72 mt-2" />
        </div>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 12 }}>
          <Skeleton className="h-28 w-full" />
          <Skeleton className="h-28 w-full" />
          <Skeleton className="h-28 w-full" />
        </div>
        <Skeleton className="h-40 w-full" />
        <Skeleton className="h-32 w-full" />
      </div>
    );
  }

  return (
    <div
      style={{
        padding: '28px 40px 80px',
        maxWidth: 680,
        display: 'flex',
        flexDirection: 'column',
        gap: 24,
      }}
    >
      {/* Section header */}
      <div style={{ paddingBottom: 16, borderBottom: '1px solid var(--border)' }}>
        <h2
          style={{
            fontSize: 18,
            fontWeight: 600,
            color: 'var(--text-emphasis)',
            letterSpacing: '-0.02em',
            margin: 0,
          }}
        >
          Notifications
        </h2>
        <p
          style={{
            fontSize: 12,
            color: 'var(--text-muted)',
            margin: '4px 0 0',
          }}
        >
          Delivery channels, event subscriptions, and digest scheduling.
        </p>
      </div>

      {/* Channel grid */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(3, 1fr)',
          gap: 12,
        }}
      >
        {CHANNELS.map((ch) => (
          <ChannelCard key={ch.type} {...ch} />
        ))}
      </div>

      {/* Info banner */}
      <div
        style={{
          fontSize: 12,
          color: 'var(--text-muted)',
          fontFamily: 'var(--font-sans)',
          background: 'var(--bg-inset)',
          border: '1px solid var(--border)',
          borderRadius: 8,
          padding: '10px 14px',
        }}
      >
        Channel configuration is managed on the{' '}
        <Link
          to="/notifications"
          style={{ color: 'var(--accent)', textDecoration: 'none', fontWeight: 500 }}
        >
          Notifications page &rarr;
        </Link>
      </div>

      {/* Divider */}
      <div style={{ height: 1, background: 'var(--border)' }} />

      {/* Event Subscriptions (real API) */}
      <div>
        <h3
          style={{
            fontSize: 14,
            fontWeight: 600,
            color: 'var(--text-emphasis)',
            fontFamily: 'var(--font-sans)',
            margin: '0 0 4px',
          }}
        >
          Event Subscriptions
        </h3>
        <p
          style={{
            fontSize: 11,
            color: 'var(--text-faint)',
            fontFamily: 'var(--font-sans)',
            margin: '0 0 16px',
          }}
        >
          Choose which events trigger notifications on each channel. Grayed-out channels are not
          configured.
        </p>

        {prefs ? (
          <PreferenceMatrix
            preferences={prefs}
            onToggle={handleToggle}
            saving={updatePrefs.isPending}
          />
        ) : (
          <p style={{ fontSize: 12, color: 'var(--text-muted)', fontFamily: 'var(--font-sans)' }}>
            Unable to load notification preferences.
          </p>
        )}
      </div>

      {/* Divider */}
      <div style={{ height: 1, background: 'var(--border)' }} />

      {/* Digest Configuration */}
      <div>
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            marginBottom: 12,
          }}
        >
          <div>
            <h3
              style={{
                fontSize: 14,
                fontWeight: 600,
                color: 'var(--text-emphasis)',
                fontFamily: 'var(--font-sans)',
                margin: 0,
              }}
            >
              Digest Schedule
            </h3>
            <p
              style={{
                fontSize: 11,
                color: 'var(--text-faint)',
                fontFamily: 'var(--font-sans)',
                margin: '2px 0 0',
              }}
            >
              Batch non-urgent notifications into a scheduled digest.
            </p>
          </div>
          <button
            type="button"
            onClick={handleTestDigest}
            disabled={testDigest.isPending}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 5,
              height: 28,
              padding: '0 10px',
              background: 'transparent',
              border: '1px solid var(--border)',
              borderRadius: 5,
              fontSize: 10,
              fontWeight: 500,
              color: 'var(--text-secondary)',
              cursor: testDigest.isPending ? 'not-allowed' : 'pointer',
              fontFamily: 'var(--font-sans)',
              opacity: testDigest.isPending ? 0.6 : 1,
            }}
          >
            {testDigest.isPending ? (
              <Loader2 style={{ width: 10, height: 10 }} className="animate-spin" />
            ) : (
              <Send style={{ width: 10, height: 10 }} />
            )}
            Test Digest
          </button>
        </div>

        {digestConfig ? (
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: '1fr 1fr 1fr',
              gap: 12,
            }}
          >
            {/* Frequency */}
            <div>
              <label
                style={{
                  display: 'block',
                  fontSize: 10,
                  fontWeight: 600,
                  textTransform: 'uppercase',
                  letterSpacing: '0.06em',
                  color: 'var(--text-muted)',
                  fontFamily: 'var(--font-mono)',
                  marginBottom: 6,
                }}
              >
                Frequency
              </label>
              <select
                value={digestConfig.frequency}
                onChange={(e) => handleDigestUpdate('frequency', e.target.value)}
                disabled={updateDigest.isPending}
                style={{
                  width: '100%',
                  height: 34,
                  padding: '0 8px',
                  background: 'var(--bg-input, var(--bg-inset))',
                  border: '1px solid var(--border)',
                  borderRadius: 6,
                  fontSize: 12,
                  color: 'var(--text-primary)',
                  fontFamily: 'var(--font-sans)',
                  outline: 'none',
                  cursor: 'pointer',
                }}
              >
                <option value="daily">Daily</option>
                <option value="weekly">Weekly</option>
              </select>
            </div>

            {/* Delivery time */}
            <div>
              <label
                style={{
                  display: 'block',
                  fontSize: 10,
                  fontWeight: 600,
                  textTransform: 'uppercase',
                  letterSpacing: '0.06em',
                  color: 'var(--text-muted)',
                  fontFamily: 'var(--font-mono)',
                  marginBottom: 6,
                }}
              >
                Delivery Time (UTC)
              </label>
              <input
                type="time"
                value={digestConfig.delivery_time}
                onChange={(e) => handleDigestUpdate('delivery_time', e.target.value)}
                disabled={updateDigest.isPending}
                style={{
                  width: '100%',
                  height: 34,
                  padding: '0 8px',
                  background: 'var(--bg-input, var(--bg-inset))',
                  border: '1px solid var(--border)',
                  borderRadius: 6,
                  fontSize: 12,
                  color: 'var(--text-primary)',
                  fontFamily: 'var(--font-mono)',
                  outline: 'none',
                }}
              />
            </div>

            {/* Format */}
            <div>
              <label
                style={{
                  display: 'block',
                  fontSize: 10,
                  fontWeight: 600,
                  textTransform: 'uppercase',
                  letterSpacing: '0.06em',
                  color: 'var(--text-muted)',
                  fontFamily: 'var(--font-mono)',
                  marginBottom: 6,
                }}
              >
                Format
              </label>
              <select
                value={digestConfig.format}
                onChange={(e) => handleDigestUpdate('format', e.target.value)}
                disabled={updateDigest.isPending}
                style={{
                  width: '100%',
                  height: 34,
                  padding: '0 8px',
                  background: 'var(--bg-input, var(--bg-inset))',
                  border: '1px solid var(--border)',
                  borderRadius: 6,
                  fontSize: 12,
                  color: 'var(--text-primary)',
                  fontFamily: 'var(--font-sans)',
                  outline: 'none',
                  cursor: 'pointer',
                }}
              >
                <option value="html">HTML</option>
                <option value="plaintext">Plain Text</option>
              </select>
            </div>
          </div>
        ) : (
          <p style={{ fontSize: 12, color: 'var(--text-muted)', fontFamily: 'var(--font-sans)' }}>
            Unable to load digest configuration.
          </p>
        )}
      </div>
    </div>
  );
}
