/* eslint-disable @typescript-eslint/no-explicit-any */
import { useState, useRef, useCallback } from 'react';
import { useCan } from '../../app/auth/AuthContext';
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
  Switch,
  Collapsible,
  CollapsibleTrigger,
  CollapsibleContent,
  Button,
  Skeleton,
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@patchiq/ui';
import { ChevronDown, ChevronRight, Send, CheckCircle2, XCircle, PlusCircle } from 'lucide-react';
import { toast } from 'sonner';
import {
  useNotificationPreferences,
  useUpdatePreferences,
  useDigestConfig,
  useUpdateDigestConfig,
  useTestDigest,
  useNotificationChannels,
  useTestChannel,
  useCreateChannel,
} from '../../api/hooks/useNotifications';
import type { components } from '../../api/types';

type PreferenceEventEntry = components['schemas']['PreferenceEventEntry'];

const CHANNEL_TYPE_LABEL: Record<string, string> = {
  email: 'Email',
  slack: 'Slack',
  webhook: 'Webhook',
  discord: 'Discord',
};

const CHANNEL_TYPES_DEFAULT = ['email', 'slack', 'webhook'] as const;

function SetupChannelDialog({
  open,
  onOpenChange,
  channelType,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  channelType: string;
}) {
  const can = useCan();
  const label = CHANNEL_TYPE_LABEL[channelType] ?? channelType;
  const createChannel = useCreateChannel();
  const [name, setName] = useState('');
  const [config, setConfig] = useState('');
  const [nameError, setNameError] = useState('');
  const [configError, setConfigError] = useState('');

  const placeholder: Record<string, string> = {
    email: 'smtp://user:pass@smtp.example.com:587',
    slack: 'slack://token@channel',
    webhook: 'generic+https://your-webhook-url.example.com',
    discord: 'discord://token@channel',
  };

  const handleSave = async () => {
    let hasError = false;
    if (!name.trim()) {
      setNameError('Channel name is required');
      hasError = true;
    }
    if (!config.trim()) {
      setConfigError('Shoutrrr URL is required');
      hasError = true;
    }
    if (hasError) return;
    try {
      await createChannel.mutateAsync({
        name: name.trim(),
        channel_type: channelType,
        config: config.trim(),
      });
      toast.success(`${label} channel configured`);
      onOpenChange(false);
      setName('');
      setConfig('');
      setNameError('');
      setConfigError('');
    } catch (err) {
      toast.error(
        `Failed to configure ${label}: ${err instanceof Error ? err.message : 'Unknown error'}`,
      );
    }
  };

  const handleOpenChange = (o: boolean) => {
    if (!o) {
      setNameError('');
      setConfigError('');
    }
    onOpenChange(o);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Set up {label} Channel</DialogTitle>
        </DialogHeader>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
            <label
              style={{
                fontSize: 10,
                fontWeight: 600,
                fontFamily: 'var(--font-mono)',
                color: 'var(--text-muted)',
                textTransform: 'uppercase',
                letterSpacing: '0.06em',
              }}
            >
              Channel Name
            </label>
            <input
              value={name}
              onChange={(e) => {
                setName(e.target.value);
                if (nameError) setNameError('');
              }}
              placeholder={`My ${label} Channel`}
              style={{
                height: 36,
                padding: '0 10px',
                background: 'var(--bg-card)',
                border: `1px solid ${nameError ? 'var(--signal-critical)' : 'var(--border)'}`,
                borderRadius: 6,
                fontSize: 13,
                color: 'var(--text-primary)',
                fontFamily: 'var(--font-sans)',
                outline: 'none',
              }}
            />
            {nameError && (
              <span
                style={{
                  color: 'var(--signal-critical)',
                  fontSize: 11,
                  marginTop: 4,
                  fontFamily: 'var(--font-sans)',
                }}
              >
                {nameError}
              </span>
            )}
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
            <label
              style={{
                fontSize: 10,
                fontWeight: 600,
                fontFamily: 'var(--font-mono)',
                color: 'var(--text-muted)',
                textTransform: 'uppercase',
                letterSpacing: '0.06em',
              }}
            >
              Shoutrrr URL
            </label>
            <input
              value={config}
              onChange={(e) => {
                setConfig(e.target.value);
                if (configError) setConfigError('');
              }}
              placeholder={placeholder[channelType] ?? 'shoutrrr URL'}
              style={{
                height: 36,
                padding: '0 10px',
                background: 'var(--bg-card)',
                border: `1px solid ${configError ? 'var(--signal-critical)' : 'var(--border)'}`,
                borderRadius: 6,
                fontSize: 12,
                color: 'var(--text-primary)',
                fontFamily: 'var(--font-mono)',
                outline: 'none',
              }}
            />
            {configError && (
              <span
                style={{
                  color: 'var(--signal-critical)',
                  fontSize: 11,
                  marginTop: 4,
                  fontFamily: 'var(--font-sans)',
                }}
              >
                {configError}
              </span>
            )}
            <p style={{ fontSize: 11, color: 'var(--text-muted)' }}>
              Uses Shoutrrr URL format. See{' '}
              <a
                href="https://containrrr.dev/shoutrrr/"
                target="_blank"
                rel="noreferrer"
                style={{ color: 'var(--accent)' }}
              >
                documentation
              </a>
              .
            </p>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => handleOpenChange(false)}>
            Cancel
          </Button>
          <Button
            onClick={handleSave}
            disabled={createChannel.isPending || !can('settings', 'create')}
            title={!can('settings', 'create') ? "You don't have permission" : undefined}
          >
            {createChannel.isPending ? 'Saving...' : 'Save'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function ChannelCard({
  type,
  channel,
  onTest,
  testing,
  onSetup,
}: {
  type: string;
  channel: any | undefined;
  onTest: (id: string, label: string) => void;
  testing: boolean;
  onSetup: (type: string) => void;
}) {
  const can = useCan();
  const label = CHANNEL_TYPE_LABEL[type] ?? type;
  const isConfigured = !!channel;

  return (
    <div
      style={{
        background: 'var(--bg-card)',
        border: `1px solid ${isConfigured ? 'var(--border)' : 'var(--border)'}`,
        borderRadius: 8,
        padding: '14px 16px',
        display: 'flex',
        alignItems: 'center',
        gap: 14,
      }}
    >
      {/* Channel icon placeholder */}
      <div
        style={{
          width: 36,
          height: 36,
          borderRadius: 8,
          background: 'var(--bg-inset)',
          border: '1px solid var(--border)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          fontFamily: 'var(--font-mono)',
          fontSize: 10,
          fontWeight: 700,
          color: isConfigured ? 'var(--accent)' : 'var(--text-muted)',
          textTransform: 'uppercase',
          flexShrink: 0,
        }}
      >
        {label.slice(0, 2)}
      </div>

      <div style={{ flex: 1, minWidth: 0 }}>
        <div
          style={{
            fontFamily: 'var(--font-sans)',
            fontSize: 13,
            fontWeight: 600,
            color: 'var(--text-primary)',
            marginBottom: 2,
          }}
        >
          {label}
        </div>
        <div
          style={{
            fontFamily: 'var(--font-sans)',
            fontSize: 11,
            color: isConfigured ? 'var(--text-secondary)' : 'var(--text-muted)',
          }}
        >
          {isConfigured ? (channel.name ?? label) : 'Not configured'}
        </div>
      </div>

      {isConfigured ? (
        <Button
          variant="outline"
          size="sm"
          disabled={testing || !can('settings', 'update')}
          title={!can('settings', 'update') ? "You don't have permission" : undefined}
          onClick={() => onTest(channel.id, label)}
          style={{ fontFamily: 'var(--font-mono)', fontSize: 11, flexShrink: 0 }}
        >
          <Send style={{ width: 11, height: 11, marginRight: 5 }} />
          Test
        </Button>
      ) : (
        <Button
          variant="ghost"
          size="sm"
          onClick={() => onSetup(type)}
          disabled={!can('settings', 'create')}
          title={!can('settings', 'create') ? "You don't have permission" : undefined}
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 11,
            color: 'var(--accent)',
            flexShrink: 0,
          }}
        >
          <PlusCircle style={{ width: 12, height: 12, marginRight: 5 }} />
          Set up
        </Button>
      )}
    </div>
  );
}

function ChannelSummaryBar() {
  const { data: channels, isLoading } = useNotificationChannels();
  const { mutateAsync: testChannel } = useTestChannel();
  const [testingId, setTestingId] = useState<string | null>(null);
  const [setupType, setSetupType] = useState<string | null>(null);

  const configured = Array.isArray(channels) ? channels : [];

  const handleTest = async (id: string, label: string) => {
    setTestingId(id);
    try {
      const result = await testChannel(id);
      const res = result as { success?: boolean; error?: string } | undefined;
      if (res?.success === false) {
        toast.error(`Test failed for ${label}`, {
          description: res.error ?? 'Check your channel configuration.',
          icon: <XCircle style={{ width: 14, height: 14 }} />,
        });
      } else {
        toast.success(`Test sent to ${label}`, {
          description: 'Check your channel for the test notification.',
          icon: <CheckCircle2 style={{ width: 14, height: 14 }} />,
        });
      }
    } catch {
      toast.error(`Test failed for ${label}`, {
        description: 'Check your channel configuration.',
        icon: <XCircle style={{ width: 14, height: 14 }} />,
      });
    } finally {
      setTestingId(null);
    }
  };

  if (isLoading) return <Skeleton className="h-[160px] rounded-lg" />;

  return (
    <div>
      <div
        style={{
          fontFamily: 'var(--font-sans)',
          fontSize: 13,
          fontWeight: 600,
          color: 'var(--text-primary)',
          marginBottom: 10,
        }}
      >
        Notification Channels
      </div>
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fill, minmax(240px, 1fr))',
          gap: 10,
        }}
      >
        {CHANNEL_TYPES_DEFAULT.map((type) => {
          const ch = configured.find((c: any) => c.channel_type === type);
          return (
            <ChannelCard
              key={type}
              type={type}
              channel={ch}
              onTest={handleTest}
              testing={testingId === ch?.id}
              onSetup={(t) => setSetupType(t)}
            />
          );
        })}
      </div>
      {setupType && (
        <SetupChannelDialog
          open={!!setupType}
          onOpenChange={(o) => {
            if (!o) setSetupType(null);
          }}
          channelType={setupType}
        />
      )}
    </div>
  );
}

function EventRow({
  event,
  onToggle,
}: {
  event: PreferenceEventEntry;
  onToggle: (field: 'email_enabled' | 'slack_enabled' | 'webhook_enabled', value: boolean) => void;
}) {
  return (
    <tr>
      <td
        style={{
          padding: '9px 12px',
          borderBottom: '1px solid var(--border)',
          fontFamily: 'var(--font-sans)',
          fontSize: 12,
          color: 'var(--text-primary)',
        }}
      >
        {event.label}
      </td>
      <td
        style={{
          padding: '9px 12px',
          borderBottom: '1px solid var(--border)',
          textAlign: 'center',
        }}
      >
        <Switch
          checked={event.email_enabled}
          onCheckedChange={(v) => onToggle('email_enabled', v)}
          danger={event.urgency === 'immediate' && event.email_enabled}
        />
      </td>
      <td
        style={{
          padding: '9px 12px',
          borderBottom: '1px solid var(--border)',
          textAlign: 'center',
        }}
      >
        <Switch
          checked={event.slack_enabled}
          onCheckedChange={(v) => onToggle('slack_enabled', v)}
          danger={event.urgency === 'immediate' && event.slack_enabled}
        />
      </td>
      <td
        style={{
          padding: '9px 12px',
          borderBottom: '1px solid var(--border)',
          textAlign: 'center',
        }}
      >
        <Switch
          checked={event.webhook_enabled}
          onCheckedChange={(v) => onToggle('webhook_enabled', v)}
          danger={event.urgency === 'immediate' && event.webhook_enabled}
        />
      </td>
      <td
        style={{
          padding: '9px 12px',
          borderBottom: '1px solid var(--border)',
          textAlign: 'center',
        }}
      >
        <span
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 10,
            fontWeight: 600,
            color: event.urgency === 'immediate' ? 'var(--signal-critical)' : 'var(--text-muted)',
            textTransform: 'uppercase',
            letterSpacing: '0.06em',
          }}
        >
          {event.urgency}
        </span>
      </td>
    </tr>
  );
}

const thStyle: React.CSSProperties = {
  fontFamily: 'var(--font-mono)',
  fontSize: 11,
  fontWeight: 600,
  textTransform: 'uppercase',
  letterSpacing: '0.05em',
  color: 'var(--text-muted)',
  padding: '8px 12px',
  borderBottom: '1px solid var(--border)',
};

export function PreferencesTab() {
  const can = useCan();
  const { data, isLoading } = useNotificationPreferences();
  const { data: digestConfig, isLoading: digestLoading } = useDigestConfig();
  const { mutateAsync: updatePreferences } = useUpdatePreferences();
  const { mutateAsync: updateDigestConfig } = useUpdateDigestConfig();
  const { mutateAsync: testDigest, isPending: testingDigest } = useTestDigest();

  const [openCategories, setOpenCategories] = useState<Set<string>>(new Set());
  const [localCategories, setLocalCategories] = useState<
    components['schemas']['PreferenceCategory'][] | undefined
  >(undefined);

  const debounceTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const categories = localCategories ?? data?.categories ?? [];

  const handleToggle = useCallback(
    (
      categoryId: string,
      triggerType: string,
      field: 'email_enabled' | 'slack_enabled' | 'webhook_enabled',
      value: boolean,
    ) => {
      const base = localCategories ?? data?.categories ?? [];
      const updated = base.map((cat) => {
        if (cat.id !== categoryId) return cat;
        return {
          ...cat,
          events: cat.events.map((ev) => {
            if (ev.trigger_type !== triggerType) return ev;
            return { ...ev, [field]: value };
          }),
        };
      });
      setLocalCategories(updated);

      if (debounceTimer.current) clearTimeout(debounceTimer.current);
      debounceTimer.current = setTimeout(() => {
        const allPrefs = updated.flatMap((cat) =>
          cat.events.map((ev) => ({
            trigger_type: ev.trigger_type,
            email_enabled: ev.email_enabled,
            slack_enabled: ev.slack_enabled,
            webhook_enabled: ev.webhook_enabled,
            urgency: ev.urgency,
          })),
        );
        // eslint-disable-next-line @typescript-eslint/no-explicit-any -- TODO(#332): remove cast after OpenAPI spec regeneration
        void updatePreferences(allPrefs as any);
      }, 1000);
    },
    [localCategories, data?.categories, updatePreferences],
  );

  if (isLoading) {
    return (
      <div style={{ padding: 24 }}>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-16 rounded-lg" />
          ))}
        </div>
      </div>
    );
  }

  return (
    <div
      style={{
        padding: 24,
        display: 'flex',
        flexDirection: 'column',
        gap: 20,
      }}
    >
      {/* Channel summary */}
      <ChannelSummaryBar />

      {/* Column legend */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          padding: '0 4px',
          fontFamily: 'var(--font-mono)',
          fontSize: 10,
          color: 'var(--text-muted)',
          textTransform: 'uppercase',
          letterSpacing: '0.05em',
        }}
      >
        <span style={{ flex: 1 }}>Configure per-event notifications</span>
        <span style={{ width: 64, textAlign: 'center' }}>Email</span>
        <span style={{ width: 64, textAlign: 'center' }}>Slack</span>
        <span style={{ width: 64, textAlign: 'center' }}>Webhook</span>
        <span style={{ width: 80, textAlign: 'center' }}>Urgency</span>
      </div>

      {/* Category sections */}
      {categories.map((category) => {
        const isOpen = openCategories.has(category.id);

        return (
          <Collapsible
            key={category.id}
            open={isOpen}
            onOpenChange={(open) => {
              setOpenCategories((prev) => {
                const next = new Set(prev);
                if (open) next.add(category.id);
                else next.delete(category.id);
                return next;
              });
            }}
          >
            <div
              style={{
                background: 'var(--bg-card)',
                border: '1px solid var(--border)',
                borderLeft: '1px solid var(--border)',
                borderRadius: 8,
                overflow: 'hidden',
              }}
            >
              <CollapsibleTrigger asChild>
                <div
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    padding: '14px 16px',
                    cursor: 'pointer',
                    userSelect: 'none',
                  }}
                >
                  <div>
                    <div
                      style={{
                        fontFamily: 'var(--font-sans)',
                        fontSize: 13,
                        fontWeight: 600,
                        color: 'var(--text-primary)',
                        marginBottom: 2,
                      }}
                    >
                      {category.label}
                    </div>
                    <div
                      style={{
                        fontFamily: 'var(--font-sans)',
                        fontSize: 11,
                        color: 'var(--text-secondary)',
                      }}
                    >
                      {category.description}
                    </div>
                  </div>
                  {isOpen ? (
                    <ChevronDown
                      style={{ width: 14, height: 14, color: 'var(--text-muted)', flexShrink: 0 }}
                    />
                  ) : (
                    <ChevronRight
                      style={{ width: 14, height: 14, color: 'var(--text-muted)', flexShrink: 0 }}
                    />
                  )}
                </div>
              </CollapsibleTrigger>

              <CollapsibleContent>
                <div style={{ borderTop: '1px solid var(--border)', overflowX: 'auto' }}>
                  <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                    <thead>
                      <tr style={{ background: 'var(--bg-inset)' }}>
                        <th style={{ ...thStyle, textAlign: 'left', flex: 1 }}>Event</th>
                        <th style={{ ...thStyle, width: 64, textAlign: 'center' }}>Email</th>
                        <th style={{ ...thStyle, width: 64, textAlign: 'center' }}>Slack</th>
                        <th style={{ ...thStyle, width: 64, textAlign: 'center' }}>Webhook</th>
                        <th style={{ ...thStyle, width: 80, textAlign: 'center' }}>Urgency</th>
                      </tr>
                    </thead>
                    <tbody>
                      {category.events.map((event) => (
                        <EventRow
                          key={event.trigger_type}
                          event={event}
                          onToggle={(field, value) =>
                            handleToggle(category.id, event.trigger_type, field, value)
                          }
                        />
                      ))}
                    </tbody>
                  </table>
                </div>
              </CollapsibleContent>
            </div>
          </Collapsible>
        );
      })}

      {/* Digest Configuration */}
      <div
        style={{
          background: 'var(--bg-card)',
          border: '1px solid var(--border)',
          borderRadius: 8,
          padding: 20,
        }}
      >
        <div
          style={{
            fontFamily: 'var(--font-sans)',
            fontSize: 13,
            fontWeight: 600,
            color: 'var(--text-primary)',
            marginBottom: 4,
          }}
        >
          Digest Configuration
        </div>
        <div
          style={{
            fontFamily: 'var(--font-sans)',
            fontSize: 11,
            color: 'var(--text-secondary)',
            marginBottom: 16,
          }}
        >
          Configure when and how digest notifications are delivered
        </div>

        {digestLoading || !digestConfig ? (
          <div style={{ fontFamily: 'var(--font-sans)', fontSize: 12, color: 'var(--text-muted)' }}>
            Loading…
          </div>
        ) : (
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: 'repeat(3, 1fr)',
              gap: 16,
              marginBottom: 16,
            }}
          >
            {/* Frequency */}
            <div>
              <label
                style={{
                  fontFamily: 'var(--font-mono)',
                  fontSize: 10,
                  fontWeight: 600,
                  textTransform: 'uppercase',
                  letterSpacing: '0.05em',
                  color: 'var(--text-muted)',
                  display: 'block',
                  marginBottom: 6,
                }}
              >
                Frequency
              </label>
              <Select
                value={digestConfig.frequency}
                onValueChange={(v) =>
                  void updateDigestConfig({ ...digestConfig, frequency: v as 'daily' | 'weekly' })
                }
              >
                <SelectTrigger className="h-8 text-sm">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="daily">Daily</SelectItem>
                  <SelectItem value="weekly">Weekly</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {/* Delivery Time */}
            <div>
              <label
                style={{
                  fontFamily: 'var(--font-mono)',
                  fontSize: 10,
                  fontWeight: 600,
                  textTransform: 'uppercase',
                  letterSpacing: '0.05em',
                  color: 'var(--text-muted)',
                  display: 'block',
                  marginBottom: 6,
                }}
              >
                Delivery Time (UTC)
              </label>
              <input
                type="time"
                style={{
                  display: 'flex',
                  height: 32,
                  width: '100%',
                  borderRadius: 6,
                  border: '1px solid var(--border)',
                  background: 'var(--bg-inset)',
                  color: 'var(--text-primary)',
                  padding: '0 10px',
                  fontFamily: 'var(--font-mono)',
                  fontSize: 12,
                }}
                value={digestConfig.delivery_time}
                onChange={(e) =>
                  void updateDigestConfig({ ...digestConfig, delivery_time: e.target.value })
                }
              />
            </div>

            {/* Format */}
            <div>
              <label
                style={{
                  fontFamily: 'var(--font-mono)',
                  fontSize: 10,
                  fontWeight: 600,
                  textTransform: 'uppercase',
                  letterSpacing: '0.05em',
                  color: 'var(--text-muted)',
                  display: 'block',
                  marginBottom: 6,
                }}
              >
                Format
              </label>
              <Select
                value={digestConfig.format}
                onValueChange={(v) =>
                  void updateDigestConfig({
                    ...digestConfig,
                    format: v as 'html' | 'plaintext',
                  })
                }
              >
                <SelectTrigger className="h-8 text-sm">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="html">HTML</SelectItem>
                  <SelectItem value="plaintext">Plain Text</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
        )}

        <Button
          variant="outline"
          size="sm"
          disabled={testingDigest || !can('settings', 'update')}
          title={!can('settings', 'update') ? "You don't have permission" : undefined}
          onClick={() =>
            void testDigest()
              .then(() => toast.success('Test digest sent'))
              .catch(() => toast.error('Test digest failed'))
          }
          style={{ fontFamily: 'var(--font-mono)', fontSize: 11 }}
        >
          <Send style={{ width: 12, height: 12, marginRight: 6 }} />
          Send Test Digest
        </Button>
      </div>
    </div>
  );
}
