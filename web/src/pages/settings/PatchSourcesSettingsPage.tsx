import { useState } from 'react';
import { Loader2, RefreshCw, Database, AlertTriangle } from 'lucide-react';
import { Skeleton } from '@patchiq/ui';
import { toast } from 'sonner';
import {
  useHubSyncStatus,
  useTriggerHubSync,
  isNotConfiguredError,
} from '../../api/hooks/useHubSync';

function formatRelativeTime(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const minutes = Math.floor(diff / 60_000);
  if (minutes < 1) return 'just now';
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

function formatDateTime(iso: string): string {
  return new Date(iso).toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function StatusPill({ status }: { status: string }) {
  const isOk = status === 'synced' || status === 'idle';
  const isSyncing = status === 'syncing';
  const isErr = status === 'error' || status === 'failed';

  const bg = isOk
    ? 'color-mix(in srgb, var(--signal-healthy) 8%, transparent)'
    : isSyncing
      ? 'color-mix(in srgb, var(--signal-info) 8%, transparent)'
      : isErr
        ? 'color-mix(in srgb, var(--signal-critical) 1%, transparent)'
        : 'color-mix(in srgb, var(--text-muted) 8%, transparent)';
  const color = isOk
    ? 'var(--signal-healthy)'
    : isSyncing
      ? 'var(--signal-info)'
      : isErr
        ? 'var(--signal-critical)'
        : 'var(--text-muted)';
  const border = isOk
    ? 'color-mix(in srgb, var(--signal-healthy) 20%, transparent)'
    : isSyncing
      ? 'color-mix(in srgb, var(--signal-info) 20%, transparent)'
      : isErr
        ? 'color-mix(in srgb, var(--signal-critical) 1%, transparent)'
        : 'color-mix(in srgb, var(--text-muted) 20%, transparent)';

  return (
    <span
      style={{
        background: bg,
        color,
        border: `1px solid ${border}`,
        borderRadius: 20,
        padding: '4px 10px',
        fontSize: 11,
        fontWeight: 500,
        fontFamily: 'var(--font-sans)',
        whiteSpace: 'nowrap',
        textTransform: 'capitalize',
      }}
    >
      {status || 'Unknown'}
    </span>
  );
}

function InfoRow({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div
      style={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        padding: '10px 0',
        borderBottom: '1px solid var(--border)',
      }}
    >
      <span
        style={{
          fontSize: 12,
          color: 'var(--text-muted)',
          fontFamily: 'var(--font-sans)',
        }}
      >
        {label}
      </span>
      <span
        style={{
          fontSize: 12,
          color: 'var(--text-primary)',
          fontFamily: mono ? 'var(--font-mono)' : 'var(--font-sans)',
          fontWeight: 500,
        }}
      >
        {value}
      </span>
    </div>
  );
}

function formatInterval(seconds: number): string {
  const hours = seconds / 3600;
  if (hours < 1) return `${Math.round(seconds / 60)}m`;
  return `${hours}h`;
}

const FEED_SOURCES = [
  { name: 'NVD', description: 'National Vulnerability Database', interval: '6h' },
  { name: 'CISA KEV', description: 'Known Exploited Vulnerabilities', interval: '12h' },
  { name: 'MSRC', description: 'Microsoft Security Response Center', interval: '12h' },
  { name: 'Red Hat', description: 'Red Hat OVAL Advisories', interval: '12h' },
  { name: 'Ubuntu USN', description: 'Ubuntu Security Notices', interval: '12h' },
  { name: 'Apple', description: 'Apple Security Updates', interval: '12h' },
];

export function PatchSourcesSettingsPage() {
  const { data: syncStatus, isLoading, error } = useHubSyncStatus();
  const triggerSync = useTriggerHubSync();
  const [syncing, setSyncing] = useState(false);
  const notConfigured = isNotConfiguredError(error);

  function handleSync() {
    setSyncing(true);
    triggerSync.mutate(undefined, {
      onSuccess: () => {
        toast.success('Catalog sync job enqueued');
        setSyncing(false);
      },
      onError: (err) => {
        toast.error(`Sync failed: ${err.message}`);
        setSyncing(false);
      },
    });
  }

  if (isLoading) {
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
        <Skeleton className="h-20 w-full" />
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
        gap: 20,
      }}
    >
      {/* Section header */}
      <div style={{ paddingBottom: 16, borderBottom: '1px solid var(--border)', marginBottom: 4 }}>
        <h2
          style={{
            fontSize: 18,
            fontWeight: 600,
            color: 'var(--text-emphasis)',
            letterSpacing: '-0.02em',
            margin: 0,
          }}
        >
          Patch Sources
        </h2>
        <p
          style={{
            fontSize: 12,
            color: 'var(--text-muted)',
            margin: '4px 0 0',
          }}
        >
          Hub connection, catalog sync, and vulnerability feed status.
        </p>
      </div>

      {notConfigured ? (
        /* Not configured state */
        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            gap: 12,
            padding: '40px 20px',
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 10,
            textAlign: 'center',
          }}
        >
          <Database
            style={{ width: 32, height: 32, color: 'var(--text-faint)', strokeWidth: 1.5 }}
          />
          <div>
            <div
              style={{
                fontSize: 14,
                fontWeight: 600,
                color: 'var(--text-primary)',
                fontFamily: 'var(--font-sans)',
              }}
            >
              Hub Not Connected
            </div>
            <div
              style={{
                fontSize: 12,
                color: 'var(--text-muted)',
                fontFamily: 'var(--font-sans)',
                marginTop: 4,
                maxWidth: 340,
              }}
            >
              Configure your PatchIQ Hub connection to receive patch intelligence, CVE feeds, and
              catalog updates.
            </div>
          </div>
          <div
            style={{
              fontSize: 11,
              color: 'var(--text-faint)',
              fontFamily: 'var(--font-mono)',
              background: 'var(--bg-inset)',
              borderRadius: 6,
              padding: '8px 14px',
              marginTop: 4,
            }}
          >
            PUT /api/v1/sync/config with hub_url and api_key
          </div>
        </div>
      ) : syncStatus ? (
        <>
          {/* Hub Connection Card */}
          <div
            style={{
              background: 'var(--bg-card)',
              border: '1px solid var(--border)',
              borderRadius: 10,
              padding: 16,
              display: 'flex',
              alignItems: 'center',
              gap: 12,
            }}
          >
            <div
              style={{
                width: 40,
                height: 40,
                borderRadius: 8,
                background: 'linear-gradient(135deg, var(--signal-info), var(--accent))',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                flexShrink: 0,
              }}
            >
              <Database
                style={{
                  width: 18,
                  height: 18,
                  color: 'var(--text-on-color, #fff)',
                  strokeWidth: 2,
                }}
              />
            </div>
            <div style={{ flex: 1, minWidth: 0 }}>
              <div
                style={{
                  fontSize: 13,
                  fontWeight: 600,
                  color: 'var(--text-primary)',
                  fontFamily: 'var(--font-sans)',
                }}
              >
                PatchIQ Hub
              </div>
              <div
                style={{
                  fontSize: 11,
                  color: 'var(--text-muted)',
                  fontFamily: 'var(--font-mono)',
                  marginTop: 2,
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                }}
              >
                {syncStatus.hub_url}
              </div>
            </div>
            <StatusPill status={syncStatus.status} />
          </div>

          {/* Sync Details */}
          <div>
            <div
              style={{
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                marginBottom: 12,
              }}
            >
              <span
                style={{
                  fontSize: 13,
                  fontWeight: 600,
                  color: 'var(--text-emphasis)',
                  fontFamily: 'var(--font-sans)',
                }}
              >
                Sync Status
              </span>
              <button
                type="button"
                onClick={handleSync}
                disabled={syncing || triggerSync.isPending}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 6,
                  height: 30,
                  padding: '0 12px',
                  background: 'transparent',
                  border: '1px solid var(--border)',
                  borderRadius: 6,
                  fontSize: 11,
                  fontWeight: 500,
                  color: 'var(--text-secondary)',
                  cursor: syncing ? 'not-allowed' : 'pointer',
                  fontFamily: 'var(--font-sans)',
                  opacity: syncing ? 0.6 : 1,
                }}
              >
                {syncing ? (
                  <Loader2 style={{ width: 12, height: 12 }} className="animate-spin" />
                ) : (
                  <RefreshCw style={{ width: 12, height: 12 }} />
                )}
                {syncing ? 'Syncing...' : 'Sync Now'}
              </button>
            </div>

            <div
              style={{
                background: 'var(--bg-card)',
                border: '1px solid var(--border)',
                borderRadius: 8,
                padding: '4px 14px',
              }}
            >
              <InfoRow
                label="Last Sync"
                value={
                  syncStatus.last_sync_at
                    ? `${formatDateTime(syncStatus.last_sync_at)} (${formatRelativeTime(syncStatus.last_sync_at)})`
                    : 'Never'
                }
              />
              <InfoRow
                label="Next Sync"
                value={syncStatus.next_sync_at ? formatDateTime(syncStatus.next_sync_at) : '—'}
              />
              <InfoRow
                label="Sync Interval"
                value={formatInterval(syncStatus.sync_interval)}
                mono
              />
              <InfoRow
                label="Entries Received"
                value={syncStatus.entries_received.toLocaleString()}
                mono
              />
              <InfoRow
                label="Last Batch"
                value={`${syncStatus.last_entry_count.toLocaleString()} entries`}
                mono
              />
            </div>

            {/* Error banner */}
            {syncStatus.last_error && (
              <div
                style={{
                  display: 'flex',
                  alignItems: 'flex-start',
                  gap: 8,
                  marginTop: 12,
                  padding: '10px 14px',
                  background: 'color-mix(in srgb, var(--signal-critical) 1%, transparent)',
                  border: '1px solid color-mix(in srgb, var(--signal-critical) 1%, transparent)',
                  borderRadius: 8,
                  fontSize: 12,
                  color: 'var(--signal-critical)',
                  fontFamily: 'var(--font-sans)',
                  lineHeight: 1.5,
                }}
              >
                <AlertTriangle style={{ width: 14, height: 14, flexShrink: 0, marginTop: 1 }} />
                <span>{syncStatus.last_error}</span>
              </div>
            )}
          </div>

          {/* Divider */}
          <div style={{ height: 1, background: 'var(--border)' }} />

          {/* Feed Sources */}
          <div>
            <span
              style={{
                fontSize: 13,
                fontWeight: 600,
                color: 'var(--text-emphasis)',
                fontFamily: 'var(--font-sans)',
                display: 'block',
                marginBottom: 12,
              }}
            >
              Vulnerability Feed Sources
            </span>
            <div
              style={{
                borderRadius: 8,
                border: '1px solid var(--border)',
                overflow: 'hidden',
              }}
            >
              <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                <thead style={{ background: 'var(--bg-inset)' }}>
                  <tr>
                    {['Source', 'Description', 'Interval'].map((h) => (
                      <th
                        key={h}
                        style={{
                          padding: '8px 12px',
                          fontSize: 10,
                          fontWeight: 600,
                          textTransform: 'uppercase',
                          letterSpacing: '0.06em',
                          color: 'var(--text-muted)',
                          textAlign: 'left',
                          fontFamily: 'var(--font-sans)',
                          borderBottom: '1px solid var(--border)',
                        }}
                      >
                        {h}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {FEED_SOURCES.map((feed, i) => (
                    <tr key={feed.name}>
                      <td
                        style={{
                          padding: '8px 12px',
                          fontSize: 12,
                          fontWeight: 500,
                          color: 'var(--text-primary)',
                          fontFamily: 'var(--font-sans)',
                          borderBottom:
                            i < FEED_SOURCES.length - 1 ? '1px solid var(--border)' : 'none',
                        }}
                      >
                        {feed.name}
                      </td>
                      <td
                        style={{
                          padding: '8px 12px',
                          fontSize: 11,
                          color: 'var(--text-muted)',
                          fontFamily: 'var(--font-sans)',
                          borderBottom:
                            i < FEED_SOURCES.length - 1 ? '1px solid var(--border)' : 'none',
                        }}
                      >
                        {feed.description}
                      </td>
                      <td
                        style={{
                          padding: '8px 12px',
                          fontSize: 11,
                          color: 'var(--text-secondary)',
                          fontFamily: 'var(--font-mono)',
                          borderBottom:
                            i < FEED_SOURCES.length - 1 ? '1px solid var(--border)' : 'none',
                        }}
                      >
                        {feed.interval}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <p
              style={{
                fontSize: 11,
                color: 'var(--text-faint)',
                fontFamily: 'var(--font-sans)',
                marginTop: 8,
              }}
            >
              Feed sources are managed by the PatchIQ Hub. Sync intervals shown are defaults.
            </p>
          </div>
        </>
      ) : (
        <p style={{ fontSize: 13, color: 'var(--text-muted)', fontFamily: 'var(--font-sans)' }}>
          Unable to load sync status.
        </p>
      )}
    </div>
  );
}
