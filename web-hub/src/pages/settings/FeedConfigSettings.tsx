import { useState } from 'react';
import { Rss, ChevronDown, Plus } from 'lucide-react';
import { Button, SkeletonCard } from '@patchiq/ui';
import { useFeeds, useUpdateFeed, useTriggerFeedSync } from '../../api/hooks/useFeeds';
import type { Feed } from '../../types/feed';

function formatSyncInterval(seconds: number): string {
  if (seconds < 3600) return `Every ${seconds / 60} minutes`;
  if (seconds === 3600) return 'Every 1 hour';
  return `Every ${seconds / 3600} hours`;
}

function formatLastSync(dateStr: string | null, status: string): { text: string; color: string } {
  if (!dateStr) return { text: 'Never', color: 'var(--text-muted)' };
  const diff = Date.now() - new Date(dateStr).getTime();
  const mins = Math.floor(diff / 60000);
  const hours = Math.floor(mins / 60);
  const timeStr = hours > 0 ? `${hours}h ago` : `${mins}m ago`;
  if (status === 'error') return { text: `Failed ${timeStr}`, color: 'var(--signal-critical)' };
  return { text: `${timeStr}`, color: 'var(--signal-healthy)' };
}

const inputStyle: React.CSSProperties = {
  width: '100%',
  padding: '6px 12px',
  borderRadius: 'var(--radius-lg)',
  border: '1px solid var(--border)',
  background: 'var(--bg-card)',
  color: 'var(--text-primary)',
  fontSize: '12px',
  outline: 'none',
};

export const FeedConfigSettings = () => {
  const { data: feeds, isLoading } = useFeeds();
  const updateFeed = useUpdateFeed();
  const triggerSync = useTriggerFeedSync();
  const [expandedFeeds, setExpandedFeeds] = useState<Set<string>>(new Set());
  const [feedError, setFeedError] = useState<string | null>(null);

  const toggleExpanded = (id: string) => {
    setExpandedFeeds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const handleToggleFeed = (feed: Feed) => {
    setFeedError(null);
    updateFeed.mutate(
      { id: feed.id, data: { enabled: !feed.enabled } },
      {
        onError: (err) =>
          setFeedError(
            `Failed to update ${feed.display_name}: ${err instanceof Error ? err.message : 'Unknown error'}`,
          ),
      },
    );
  };

  if (isLoading) return <SkeletonCard />;

  return (
    <div
      className="rounded-xl overflow-hidden"
      style={{ background: 'var(--bg-card)', border: '1px solid var(--border)' }}
    >
      <div
        className="flex items-center justify-between px-6 py-4"
        style={{ borderBottom: '1px solid var(--border)' }}
      >
        <div className="flex items-center gap-3">
          <div
            className="w-8 h-8 rounded-lg flex items-center justify-center"
            style={{ background: 'var(--accent-subtle)' }}
          >
            <Rss className="w-5 h-5" style={{ color: 'var(--accent)' }} />
          </div>
          <div>
            <h2 className="font-semibold" style={{ color: 'var(--text-emphasis)' }}>
              Feed Configuration
            </h2>
            <p className="text-xs" style={{ color: 'var(--text-muted)' }}>
              Vulnerability data source management
            </p>
          </div>
        </div>
      </div>

      <div className="p-6">
        {feedError && (
          <div
            className="mb-4 p-3 rounded-lg text-sm"
            style={{
              background: 'var(--signal-critical-subtle)',
              border: '1px solid var(--signal-critical)',
              color: 'var(--signal-critical)',
            }}
          >
            {feedError}
          </div>
        )}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {feeds?.map((feed) => {
            const isError = feed.status === 'error';
            const isExpanded = expandedFeeds.has(feed.id);
            const lastSync = formatLastSync(feed.last_sync_at, feed.status);

            return (
              <div
                key={feed.id}
                className="rounded-xl overflow-hidden"
                style={{
                  border: `1px solid ${isError ? 'var(--signal-critical)' : 'var(--border)'}`,
                }}
              >
                <div
                  className="flex items-center justify-between p-4"
                  style={{
                    background: isError ? 'var(--signal-critical-subtle)' : 'var(--bg-inset)',
                  }}
                >
                  <div className="flex items-center gap-3">
                    <div
                      className="w-8 h-8 rounded flex items-center justify-center text-xs font-bold"
                      style={{ background: 'var(--text-faint)', color: 'var(--text-emphasis)' }}
                    >
                      {feed.display_name.slice(0, 3).toUpperCase()}
                    </div>
                    <div>
                      <p className="font-medium text-sm" style={{ color: 'var(--text-emphasis)' }}>
                        {feed.display_name}
                      </p>
                      <p className="text-xs" style={{ color: 'var(--text-muted)' }}>
                        {feed.name}
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center gap-3">
                    <span
                      className={`w-2.5 h-2.5 rounded-full inline-block ${feed.enabled && !isError ? 'animate-pulse' : ''}`}
                      style={{
                        background: isError
                          ? 'var(--signal-critical)'
                          : feed.enabled
                            ? 'var(--signal-healthy)'
                            : 'var(--text-faint)',
                      }}
                    />
                    <button
                      type="button"
                      role="switch"
                      aria-checked={feed.enabled}
                      onClick={() => handleToggleFeed(feed)}
                      className="relative inline-flex h-6 w-11 items-center rounded-full transition-colors"
                      style={{ background: feed.enabled ? 'var(--accent)' : 'var(--border)' }}
                    >
                      <span
                        className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${feed.enabled ? 'translate-x-6' : 'translate-x-1'}`}
                      />
                    </button>
                  </div>
                </div>

                <div className="p-4 space-y-3">
                  <div className="flex items-center justify-between text-sm">
                    <span style={{ color: 'var(--text-muted)' }}>Sync interval</span>
                    <span className="font-medium" style={{ color: 'var(--text-primary)' }}>
                      {formatSyncInterval(feed.sync_interval_seconds)}
                    </span>
                  </div>
                  <div className="flex items-center justify-between text-sm">
                    <span style={{ color: 'var(--text-muted)' }}>Last sync</span>
                    <span className="font-medium" style={{ color: lastSync.color }}>
                      {lastSync.text}
                    </span>
                  </div>
                  <div className="flex items-center justify-between text-sm">
                    <span style={{ color: 'var(--text-muted)' }}>Total entries</span>
                    <span
                      className="font-medium"
                      style={{ color: 'var(--text-primary)', fontFamily: 'var(--font-mono)' }}
                    >
                      {feed.entries_ingested.toLocaleString()}
                    </span>
                  </div>

                  {isError && feed.last_error && (
                    <div
                      className="p-2 rounded-lg text-xs"
                      style={{
                        background: 'var(--signal-critical-subtle)',
                        border: '1px solid var(--signal-critical)',
                        color: 'var(--signal-critical)',
                      }}
                    >
                      <b>Error:</b> {feed.last_error}
                    </div>
                  )}

                  <button
                    onClick={() => toggleExpanded(feed.id)}
                    className="text-xs flex items-center gap-1"
                    style={{ color: 'var(--accent)' }}
                  >
                    <ChevronDown
                      className={`w-3 h-3 transition-transform ${isExpanded ? 'rotate-180' : ''}`}
                    />
                    Advanced settings
                  </button>

                  {isExpanded && (
                    <div
                      className="mt-2 pt-3 space-y-3"
                      style={{ borderTop: '1px solid var(--border)' }}
                    >
                      <div>
                        <label
                          className="block text-xs font-medium mb-1"
                          style={{ color: 'var(--text-primary)' }}
                        >
                          Sync Interval
                        </label>
                        <select
                          defaultValue={String(feed.sync_interval_seconds)}
                          onChange={(e) => {
                            setFeedError(null);
                            updateFeed.mutate(
                              {
                                id: feed.id,
                                data: { sync_interval_seconds: Number(e.target.value) },
                              },
                              {
                                onError: (err) =>
                                  setFeedError(
                                    `Failed to update sync interval for ${feed.display_name}: ${err instanceof Error ? err.message : 'Unknown error'}`,
                                  ),
                              },
                            );
                          }}
                          style={inputStyle}
                        >
                          <option value="3600">1 hour</option>
                          <option value="21600">6 hours</option>
                          <option value="43200">12 hours</option>
                          <option value="86400">24 hours</option>
                        </select>
                      </div>
                      {isError && feed.last_error && (
                        <div>
                          <label
                            className="block text-xs font-medium mb-1"
                            style={{ color: 'var(--text-primary)' }}
                          >
                            Error Log
                          </label>
                          <div
                            className="rounded-lg p-2 text-xs"
                            style={{
                              background: 'var(--bg-canvas)',
                              fontFamily: 'var(--font-mono)',
                              color: 'var(--signal-critical)',
                            }}
                          >
                            {feed.last_error}
                          </div>
                        </div>
                      )}
                    </div>
                  )}

                  {isError && (
                    <Button
                      variant="destructive"
                      size="sm"
                      className="w-full mt-1"
                      onClick={() => {
                        setFeedError(null);
                        triggerSync.mutate(feed.id, {
                          onError: (err) =>
                            setFeedError(
                              `Retry sync failed for ${feed.display_name}: ${err instanceof Error ? err.message : 'Unknown error'}`,
                            ),
                        });
                      }}
                      disabled={triggerSync.isPending}
                    >
                      {triggerSync.isPending ? 'Retrying...' : 'Retry Sync Now'}
                    </Button>
                  )}
                </div>
              </div>
            );
          })}

          {/* Custom Feed card */}
          <div
            className="rounded-xl overflow-hidden"
            style={{ border: '1px dashed var(--border)' }}
          >
            <div
              className="flex items-center justify-between p-4"
              style={{ background: 'var(--bg-inset)' }}
            >
              <div className="flex items-center gap-3">
                <div
                  className="w-8 h-8 rounded flex items-center justify-center"
                  style={{ background: 'var(--text-faint)', color: 'var(--text-emphasis)' }}
                >
                  <Plus className="w-4 h-4" />
                </div>
                <div>
                  <p className="font-medium text-sm" style={{ color: 'var(--text-emphasis)' }}>
                    Custom Feed
                  </p>
                  <p className="text-xs" style={{ color: 'var(--text-muted)' }}>
                    Add a custom vulnerability source
                  </p>
                </div>
              </div>
            </div>
            <div className="p-4 flex flex-col items-center justify-center gap-3 py-8">
              <p className="text-sm text-center" style={{ color: 'var(--text-muted)' }}>
                Configure a custom JSON or XML vulnerability feed
              </p>
              <Button variant="outline" size="sm">
                Configure
              </Button>
            </div>
          </div>
        </div>
      </div>

      <div className="px-6 pb-5 flex gap-3">
        <Button>Save All Feeds</Button>
      </div>
    </div>
  );
};
