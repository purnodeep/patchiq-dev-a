import React, { useState, useMemo } from 'react';
import { Link, useParams } from 'react-router';
import { useForm } from 'react-hook-form';
import {
  LineChart,
  Line,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';
import {
  Button,
  Badge,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  Input,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Skeleton,
} from '@patchiq/ui';
import {
  useFeed,
  useFeedHistory,
  useUpdateFeed,
  useTriggerFeedSync,
} from '../../api/hooks/useFeeds';
import { useCatalogEntries } from '../../api/hooks/useCatalog';
import { SeverityBadge } from '../../components/SeverityBadge';
import {
  formatRelativeTime,
  formatInterval,
  formatDuration,
  fmtDate,
  fmtDatetime,
} from '../../lib/format';
import type { FeedSyncRun } from '../../types/feed';

// ── Configuration form ─────────────────────────────────────────────────────

interface ConfigFormData {
  url: string;
  sync_interval_seconds: number;
  auth_type: string;
  severity_filter: string[];
  os_filter: string[];
  severity_mapping: Record<string, string>;
}

const SEVERITY_LEVELS = ['Critical', 'High', 'Medium', 'Low'];
const OS_OPTIONS = ['Windows', 'Ubuntu', 'RHEL', 'Debian'];
const INTERVAL_OPTIONS = [
  { label: '1 hour', value: 3600 },
  { label: '6 hours', value: 21600 },
  { label: '12 hours', value: 43200 },
  { label: '24 hours', value: 86400 },
];
const AUTH_OPTIONS = [
  { label: 'None (public)', value: 'none' },
  { label: 'API Key', value: 'api_key' },
  { label: 'Bearer Token', value: 'bearer' },
  { label: 'Basic Auth', value: 'basic' },
];

// ── Sub-components ─────────────────────────────────────────────────────────

const HistoryRow = ({ run, index }: { run: FeedSyncRun; index: number }) => {
  const [expanded, setExpanded] = useState(false);
  const isError = run.status === 'failed' || run.status === 'error';

  return (
    <>
      <tr
        className="border-b border-[var(--border)] transition-colors hover:bg-card"
        style={
          isError
            ? { background: 'color-mix(in srgb, var(--signal-critical) 4%, transparent)' }
            : undefined
        }
      >
        <td className="px-3 py-2.5 font-mono text-[11.5px] text-muted-foreground">
          {fmtDatetime(run.started_at)}
        </td>
        <td className="px-3 py-2.5 font-mono text-[11.5px]">{formatDuration(run.duration_ms)}</td>
        <td
          className={`px-3 py-2.5 text-[12.5px] font-${run.new_entries > 0 ? 'bold' : 'normal'} ${
            run.new_entries > 0 ? '' : 'text-muted-foreground'
          }`}
          style={run.new_entries > 0 ? { color: 'var(--signal-healthy)' } : undefined}
        >
          {run.new_entries > 0 ? `+${run.new_entries}` : run.new_entries}
        </td>
        <td className="px-3 py-2.5 text-[12.5px] text-muted-foreground">{run.updated_entries}</td>
        <td className="px-3 py-2.5 text-[12.5px] text-muted-foreground">
          {run.total_scanned.toLocaleString()}
        </td>
        <td
          className="px-3 py-2.5 text-[12.5px]"
          style={run.error_count > 0 ? { color: 'var(--signal-critical)' } : undefined}
        >
          {run.error_count}
        </td>
        <td className="px-3 py-2.5">
          {isError ? (
            <Badge variant="destructive" className="text-[11px]">
              Failed
            </Badge>
          ) : (
            <Badge
              className="text-[11px]"
              style={{
                background: 'color-mix(in srgb, var(--signal-healthy) 15%, transparent)',
                color: 'var(--signal-healthy)',
                borderColor: 'color-mix(in srgb, var(--signal-healthy) 30%, transparent)',
              }}
            >
              Success
            </Badge>
          )}
        </td>
        <td className="px-3 py-2.5">
          <button
            type="button"
            onClick={() => setExpanded((v) => !v)}
            className="text-[11px] text-muted-foreground hover:text-foreground px-[5px] py-0.5 rounded"
          >
            {expanded ? '▲' : '▼'}
          </button>
        </td>
      </tr>
      {expanded && (
        <tr key={`${index}-detail`} className="border-b border-[var(--border)]">
          <td colSpan={8} className="px-4 py-3 bg-[var(--bg-page)]">
            {isError && run.error_message && (
              <div
                className="rounded-md p-[10px] mb-3 text-[12px]"
                style={{
                  background: 'color-mix(in srgb, var(--signal-critical) 8%, transparent)',
                  border: '1px solid color-mix(in srgb, var(--signal-critical) 25%, transparent)',
                  color: 'var(--signal-critical)',
                }}
              >
                ⚠ Error: {run.error_message}
              </div>
            )}
            <div className="text-[10px] uppercase tracking-wide text-muted-foreground mb-1.5">
              Log Output
            </div>
            <pre
              className="border border rounded-md p-3 font-mono text-[11px] leading-relaxed text-muted-foreground max-h-[200px] overflow-y-auto overflow-x-auto"
              style={{ background: 'var(--bg-page)' }}
            >
              {run.log_output || `[${fmtDatetime(run.started_at)}] No log output available`}
            </pre>
          </td>
        </tr>
      )}
    </>
  );
};

// ── Main Page ──────────────────────────────────────────────────────────────

export function FeedDetailPage() {
  const { id } = useParams<{ id: string }>();
  const feedId = id ?? '';

  const { data: feed, isLoading: feedLoading, isError: feedError } = useFeed(feedId);
  const { data: historyData } = useFeedHistory(feedId, { limit: 50 });
  const updateFeed = useUpdateFeed();
  const triggerSync = useTriggerFeedSync();

  // Active tab
  const [activeTab, setActiveTab] = useState<'overview' | 'history' | 'entries' | 'config'>(
    'overview',
  );

  // Entries tab state
  const [entriesSearch, setEntriesSearch] = useState('');
  const [entriesSeverity, setEntriesSeverity] = useState('all');
  const [entriesPage, setEntriesPage] = useState(0);
  const ENTRIES_PAGE_SIZE = 20;

  const { data: entriesData } = useCatalogEntries({
    feed_source_id: feedId,
    search: entriesSearch || undefined,
    severity: entriesSeverity === 'all' ? undefined : entriesSeverity,
    limit: ENTRIES_PAGE_SIZE,
    offset: entriesPage * ENTRIES_PAGE_SIZE,
  });

  // Config form
  const {
    register,
    handleSubmit,
    watch,
    setValue,
    formState: { errors },
  } = useForm<ConfigFormData>({
    defaultValues: {
      url: feed?.url ?? '',
      sync_interval_seconds: feed?.sync_interval_seconds ?? 21600,
      auth_type: feed?.auth_type ?? 'none',
      severity_filter: feed?.severity_filter ?? ['Critical', 'High', 'Medium', 'Low'],
      os_filter: feed?.os_filter ?? ['Windows', 'Ubuntu', 'RHEL', 'Debian'],
      severity_mapping: feed?.severity_mapping ?? {},
    },
  });

  const onSaveConfig = (data: ConfigFormData) => {
    updateFeed.mutate({ id: feedId, data });
  };

  // Derived: context ring stats
  const runs = historyData?.runs ?? [];
  const successRate = useMemo(() => {
    if (runs.length === 0) return 100;
    const successes = runs.filter((r) => r.status === 'success').length;
    return Math.round((successes / runs.length) * 100);
  }, [runs]);

  const avgDuration = useMemo(() => {
    const withDuration = runs.filter((r) => r.duration_ms !== null);
    if (withDuration.length === 0) return null;
    const avg =
      withDuration.reduce((sum, r) => sum + (r.duration_ms ?? 0), 0) / withDuration.length;
    return avg;
  }, [runs]);

  const lastError = useMemo(() => {
    const errorRun = runs.find((r) => r.status === 'failed' || r.status === 'error');
    return errorRun?.error_message ?? null;
  }, [runs]);

  // Overview chart data
  const healthChartData = useMemo(() => {
    return [...runs]
      .reverse()
      .slice(-30)
      .map((r) => ({
        date: new Date(r.started_at).toLocaleDateString('en-US', {
          month: 'short',
          day: 'numeric',
        }),
        duration: r.duration_ms !== null ? Math.round(r.duration_ms / 1000) : null,
        entries: r.new_entries,
      }));
  }, [runs]);

  const volumeChartData = useMemo(() => {
    // Aggregate by week (last 12 weeks)
    const byWeek: Record<string, number> = {};
    runs.forEach((r) => {
      const d = new Date(r.started_at);
      const weekStart = new Date(d);
      weekStart.setDate(d.getDate() - d.getDay());
      const key = weekStart.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
      byWeek[key] = (byWeek[key] ?? 0) + r.new_entries;
    });
    return Object.entries(byWeek)
      .slice(-12)
      .map(([week, entries]) => ({ week, entries }));
  }, [runs]);

  if (feedLoading) {
    return (
      <div className="p-6 space-y-4">
        <Skeleton className="h-4 w-24" />
        <Skeleton className="h-32 w-full" />
        <div className="grid grid-cols-6 gap-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} className="h-20" />
          ))}
        </div>
      </div>
    );
  }

  if (feedError || !feed) {
    return (
      <div className="p-6">
        <Link
          to="/feeds"
          className="text-sm text-muted-foreground hover:text-accent-foreground flex items-center gap-1.5 mb-4"
        >
          ← Back to Feeds
        </Link>
        <div className="rounded-lg border border-destructive/30 bg-destructive/10 p-4 text-sm text-destructive">
          Failed to load feed details.
        </div>
      </div>
    );
  }

  let orbStyle: React.CSSProperties = { background: 'var(--signal-healthy)' };
  let statusText = 'Healthy';
  let statusStyle: React.CSSProperties = { color: 'var(--signal-healthy)' };
  if (!feed.enabled) {
    orbStyle = { background: 'var(--text-muted)' };
    statusText = 'Disabled';
    statusStyle = { color: 'var(--text-muted)' };
  } else if (feed.status === 'error') {
    orbStyle = { background: 'var(--signal-critical)' };
    statusText = 'Error';
    statusStyle = { color: 'var(--signal-critical)' };
  } else if (feed.status === 'syncing') {
    orbStyle = { background: 'var(--accent)' };
    statusText = 'Syncing';
    statusStyle = { color: 'var(--accent)' };
  }

  const entriesTotalPages = entriesData ? Math.ceil(entriesData.total / ENTRIES_PAGE_SIZE) : 0;

  return (
    <div style={{ padding: '20px 24px', display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* Identity header — bare, no card, matches catalog detail style */}
      <div>
        {/* Row 1: name + status + actions */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            marginBottom: 8,
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            <h1
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 22,
                fontWeight: 600,
                color: 'var(--text-emphasis)',
                margin: 0,
                letterSpacing: '-0.01em',
              }}
            >
              {feed.name}
            </h1>
            <span style={{ color: 'var(--text-muted)', fontWeight: 400, fontSize: 14 }}>
              — {feed.display_name}
            </span>
            <span style={{ display: 'inline-flex', alignItems: 'center', gap: 5 }}>
              <span
                style={{
                  width: 8,
                  height: 8,
                  borderRadius: '50%',
                  display: 'inline-block',
                  ...orbStyle,
                }}
              />
              <span style={{ fontSize: 12, fontWeight: 500, ...statusStyle }}>{statusText}</span>
            </span>
          </div>
          {/* Action buttons */}
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <button
              type="button"
              onClick={() => triggerSync.mutate(feed.id)}
              disabled={triggerSync.isPending || !feed.enabled}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 6,
                padding: '7px 14px',
                borderRadius: 7,
                border: 'none',
                background: 'var(--accent)',
                color: 'var(--btn-accent-text, #000)',
                fontSize: 12,
                fontWeight: 600,
                cursor: triggerSync.isPending || !feed.enabled ? 'not-allowed' : 'pointer',
                opacity: triggerSync.isPending || !feed.enabled ? 0.6 : 1,
              }}
            >
              {triggerSync.isPending ? 'Syncing…' : 'Sync Now'}
            </button>
            <button
              type="button"
              onClick={() => setActiveTab('config')}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 6,
                padding: '7px 14px',
                borderRadius: 7,
                border: '1px solid var(--border)',
                background: 'none',
                color: 'var(--text-secondary)',
                fontSize: 12,
                fontWeight: 500,
                cursor: 'pointer',
              }}
            >
              Edit Configuration
            </button>
            <button
              type="button"
              onClick={() => updateFeed.mutate({ id: feed.id, data: { enabled: !feed.enabled } })}
              disabled={updateFeed.isPending}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 6,
                padding: '7px 14px',
                borderRadius: 7,
                border: `1px solid color-mix(in srgb, ${feed.enabled ? 'var(--signal-critical)' : 'var(--signal-healthy)'} 30%, transparent)`,
                background: 'none',
                color: feed.enabled ? 'var(--signal-critical)' : 'var(--signal-healthy)',
                fontSize: 12,
                fontWeight: 500,
                cursor: updateFeed.isPending ? 'not-allowed' : 'pointer',
                opacity: updateFeed.isPending ? 0.6 : 1,
              }}
            >
              {updateFeed.isPending ? 'Updating…' : feed.enabled ? 'Disable' : 'Enable'}
            </button>
          </div>
        </div>

        {/* Row 2: meta chips */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
          {/* URL chip */}
          <div
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: 4,
              padding: '2px 8px',
              borderRadius: 4,
              border: '1px solid var(--border)',
              background: 'none',
              fontSize: 11,
              fontFamily: 'var(--font-mono)',
              color: 'var(--text-muted)',
              maxWidth: 320,
              overflow: 'hidden',
            }}
          >
            <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
              {feed.url}
            </span>
            <button
              type="button"
              onClick={() => void navigator.clipboard.writeText(feed.url)}
              aria-label="Copy URL"
              style={{
                background: 'none',
                border: 'none',
                cursor: 'pointer',
                color: 'var(--text-muted)',
                padding: 0,
                flexShrink: 0,
              }}
            >
              📋
            </button>
          </div>
          {/* Enable toggle */}
          <button
            type="button"
            onClick={() => updateFeed.mutate({ id: feed.id, data: { enabled: !feed.enabled } })}
            disabled={updateFeed.isPending}
            aria-label={feed.enabled ? 'Disable feed' : 'Enable feed'}
            style={{
              width: 38,
              height: 21,
              borderRadius: 999,
              position: 'relative',
              flexShrink: 0,
              border: 'none',
              cursor: 'pointer',
              transition: 'background 0.2s',
              background: feed.enabled ? 'var(--signal-healthy)' : 'var(--text-muted)',
            }}
          >
            <span
              style={{
                position: 'absolute',
                top: 3,
                width: 15,
                height: 15,
                borderRadius: '50%',
                background: 'var(--bg-page, white)',
                transition: 'left 0.2s',
                left: feed.enabled ? 20 : 3,
              }}
            />
          </button>
          {[
            feed.enabled ? 'Enabled' : 'Disabled',
            feed.name,
            `Sync interval: ${formatInterval(feed.sync_interval_seconds)}`,
          ].map((val, i) => (
            <span
              key={i}
              style={{
                display: 'inline-flex',
                alignItems: 'center',
                padding: '2px 8px',
                borderRadius: 4,
                border: '1px solid var(--border)',
                background: 'none',
                fontSize: 11,
                fontFamily: 'var(--font-mono)',
                color: 'var(--text-muted)',
                whiteSpace: 'nowrap',
              }}
            >
              {val}
            </span>
          ))}
        </div>
      </div>

      {/* Context ring — 6 separate cards matching catalog detail style */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fill, minmax(130px, 1fr))',
          gap: 8,
        }}
      >
        {[
          {
            label: 'Total Entries',
            value: feed.entries_ingested.toLocaleString(),
            sub: 'In catalog',
            color: 'var(--accent)',
          },
          {
            label: 'New This Week',
            value: feed.new_this_week != null ? feed.new_this_week.toLocaleString() : '0',
            sub: '↑ this week',
            color: 'var(--signal-healthy)',
          },
          {
            label: 'Sync Interval',
            value: formatInterval(feed.sync_interval_seconds),
            sub: `Every ${formatInterval(feed.sync_interval_seconds)}`,
            color: 'var(--text-emphasis)',
          },
          {
            label: 'Success Rate',
            value: `${successRate}%`,
            sub: null as string | null,
            color: 'var(--signal-healthy)',
            progress: successRate,
          },
          {
            label: 'Avg Duration',
            value: formatDuration(avgDuration),
            sub: 'Per sync cycle',
            color: 'var(--text-emphasis)',
          },
          {
            label: 'Last Error',
            value: lastError ? 'Error' : 'None',
            sub: lastError ? lastError.slice(0, 20) + '…' : 'All clear',
            color: lastError ? 'var(--signal-critical)' : 'var(--signal-healthy)',
          },
        ].map((card) => (
          <div
            key={card.label}
            style={{
              padding: '12px 14px',
              borderRadius: 8,
              border: '1px solid var(--border)',
              background: 'var(--bg-card)',
            }}
          >
            <p
              style={{
                fontSize: 10,
                fontWeight: 500,
                fontFamily: 'var(--font-mono)',
                textTransform: 'uppercase',
                letterSpacing: '0.05em',
                color: 'var(--text-muted)',
                marginTop: 4,
              }}
            >
              {card.label}
            </p>
            <p
              style={{
                fontSize: 22,
                fontWeight: 700,
                fontFamily: 'var(--font-mono)',
                letterSpacing: '-0.02em',
                lineHeight: 1,
                color: card.color,
              }}
            >
              {card.value}
            </p>
            {'progress' in card && card.progress !== undefined ? (
              <div
                style={{
                  marginTop: 6,
                  height: 4,
                  borderRadius: 2,
                  overflow: 'hidden',
                  background: 'var(--border)',
                }}
              >
                <div
                  style={{
                    height: '100%',
                    width: `${card.progress}%`,
                    background: 'var(--signal-healthy)',
                  }}
                />
              </div>
            ) : card.sub ? (
              <p style={{ fontSize: 10, color: 'var(--text-muted)', marginTop: 4 }}>{card.sub}</p>
            ) : null}
          </div>
        ))}
      </div>

      {/* Tabs */}
      <div>
        <div
          style={{
            display: 'flex',
            gap: 0,
            borderBottom: '1px solid var(--border)',
          }}
        >
          {(['overview', 'history', 'entries', 'config'] as const).map((tab) => {
            const labels: Record<string, string> = {
              overview: 'Overview',
              history: 'Sync History',
              entries: 'Entries',
              config: 'Configuration',
            };
            return (
              <button
                key={tab}
                type="button"
                onClick={() => setActiveTab(tab)}
                style={{
                  padding: '8px 16px',
                  fontSize: 13,
                  fontWeight: activeTab === tab ? 600 : 400,
                  color: activeTab === tab ? 'var(--text-emphasis)' : 'var(--text-muted)',
                  border: 'none',
                  borderBottom:
                    activeTab === tab ? '2px solid var(--accent)' : '2px solid transparent',
                  background: 'transparent',
                  cursor: 'pointer',
                  transition: 'color 150ms ease',
                  marginBottom: -1,
                  whiteSpace: 'nowrap',
                }}
              >
                {labels[tab]}
              </button>
            );
          })}
        </div>

        {/* ── Overview Tab ── */}
        {activeTab === 'overview' && (
          <div style={{ marginTop: 20, display: 'flex', flexDirection: 'column', gap: 16 }}>
            <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
              {/* Sync Health Chart */}
              <Card className="bg-card border">
                <CardHeader className="pb-2">
                  <CardTitle className="text-[13px] font-semibold">
                    Sync Health — Last 30 Days
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="h-[200px]">
                    <ResponsiveContainer width="100%" height="100%">
                      <LineChart
                        data={healthChartData}
                        margin={{ top: 5, right: 10, bottom: 5, left: 0 }}
                      >
                        <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
                        <XAxis
                          dataKey="date"
                          tick={{ fill: 'var(--text-muted)', fontSize: 9 }}
                          tickLine={false}
                          interval="preserveStartEnd"
                        />
                        <YAxis
                          yAxisId="left"
                          tick={{ fill: 'var(--text-muted)', fontSize: 10 }}
                          tickLine={false}
                        />
                        <YAxis
                          yAxisId="right"
                          orientation="right"
                          tick={{ fill: 'var(--text-muted)', fontSize: 10 }}
                          tickLine={false}
                        />
                        <Tooltip
                          contentStyle={{
                            background: 'var(--bg-card)',
                            border: '1px solid var(--border)',
                            fontSize: 12,
                          }}
                          labelStyle={{ color: 'var(--text-muted)' }}
                        />
                        <Legend wrapperStyle={{ fontSize: 11, color: 'var(--text-muted)' }} />
                        <Line
                          yAxisId="left"
                          type="monotone"
                          dataKey="duration"
                          name="Sync Duration (s)"
                          stroke="var(--accent)"
                          strokeWidth={1.5}
                          dot={false}
                          fill="color-mix(in srgb, var(--accent) 8%, transparent)"
                        />
                        <Line
                          yAxisId="right"
                          type="monotone"
                          dataKey="entries"
                          name="Entries Ingested"
                          stroke="var(--accent)"
                          strokeWidth={1.5}
                          dot={{ r: 2 }}
                        />
                      </LineChart>
                    </ResponsiveContainer>
                  </div>
                </CardContent>
              </Card>

              {/* Volume Chart */}
              <Card className="bg-card border">
                <CardHeader className="pb-2">
                  <CardTitle className="text-[13px] font-semibold">
                    Entry Volume — Last 12 Weeks
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="h-[200px]">
                    <ResponsiveContainer width="100%" height="100%">
                      <BarChart
                        data={volumeChartData}
                        margin={{ top: 5, right: 10, bottom: 5, left: 0 }}
                      >
                        <CartesianGrid
                          strokeDasharray="3 3"
                          stroke="var(--border)"
                          vertical={false}
                        />
                        <XAxis
                          dataKey="week"
                          tick={{ fill: 'var(--text-muted)', fontSize: 9 }}
                          tickLine={false}
                        />
                        <YAxis
                          tick={{ fill: 'var(--text-muted)', fontSize: 10 }}
                          tickLine={false}
                        />
                        <Tooltip
                          contentStyle={{
                            background: 'var(--bg-card)',
                            border: '1px solid var(--border)',
                            fontSize: 12,
                          }}
                          labelStyle={{ color: 'var(--text-muted)' }}
                        />
                        <Legend wrapperStyle={{ fontSize: 11, color: 'var(--text-muted)' }} />
                        <Bar
                          dataKey="entries"
                          name="Entries Ingested"
                          fill="color-mix(in srgb, var(--accent) 50%, transparent)"
                          stroke="color-mix(in srgb, var(--accent) 80%, transparent)"
                          strokeWidth={1}
                          radius={[3, 3, 0, 0]}
                        />
                      </BarChart>
                    </ResponsiveContainer>
                  </div>
                </CardContent>
              </Card>
            </div>

            {/* Config summary */}
            <Card className="bg-card border">
              <CardHeader className="pb-2">
                <CardTitle className="text-[13px] font-semibold">Configuration Summary</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
                  <div>
                    <div className="flex justify-between py-2 border-b border/50 text-[12.5px]">
                      <span className="text-muted-foreground">Feed URL</span>
                      <span className="font-mono text-[11px] text-muted-foreground truncate max-w-[200px]">
                        {feed.url}
                      </span>
                    </div>
                    <div className="flex justify-between py-2 border-b border/50 text-[12.5px]">
                      <span className="text-muted-foreground">Sync Interval</span>
                      <span>Every {formatInterval(feed.sync_interval_seconds)}</span>
                    </div>
                    <div className="flex justify-between py-2 text-[12.5px]">
                      <span className="text-muted-foreground">Severity Filter</span>
                      <span>{(feed.severity_filter ?? []).join(', ') || 'All'}</span>
                    </div>
                  </div>
                  <div>
                    <div className="flex justify-between py-2 border-b border/50 text-[12.5px]">
                      <span className="text-muted-foreground">OS Filter</span>
                      <span>{(feed.os_filter ?? []).join(', ') || 'All'}</span>
                    </div>
                    <div className="flex justify-between py-2 border-b border/50 text-[12.5px]">
                      <span className="text-muted-foreground">Auth Type</span>
                      <span>
                        {feed.auth_type === 'none' ? 'None (public feed)' : feed.auth_type}
                      </span>
                    </div>
                    <div className="flex justify-between py-2 text-[12.5px]">
                      <span className="text-muted-foreground">Last Sync</span>
                      <span>
                        {feed.last_sync_at
                          ? `${formatRelativeTime(feed.last_sync_at)} — ${feed.status}`
                          : 'Never'}
                      </span>
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>
        )}

        {/* ── Sync History Tab ── */}
        {activeTab === 'history' && (
          <div style={{ marginTop: 20 }}>
            <div className="rounded-lg border border bg-background overflow-x-auto">
              <table className="w-full caption-bottom">
                <thead>
                  <tr className="border-b border">
                    {[
                      'Timestamp',
                      'Duration',
                      'New Entries',
                      'Updated',
                      'Total Scanned',
                      'Errors',
                      'Status',
                      '',
                    ].map((h) => (
                      <th
                        key={h}
                        className="px-3 py-2.5 text-left text-[10px] font-semibold uppercase tracking-wide text-muted-foreground whitespace-nowrap"
                      >
                        {h}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {runs.length === 0 ? (
                    <tr>
                      <td
                        colSpan={8}
                        className="px-3 py-8 text-center text-sm text-muted-foreground"
                      >
                        No sync history available.
                      </td>
                    </tr>
                  ) : (
                    runs.map((run, i) => <HistoryRow key={run.id} run={run} index={i} />)
                  )}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* ── Entries Tab ── */}
        {activeTab === 'entries' && (
          <div style={{ marginTop: 20, display: 'flex', flexDirection: 'column', gap: 12 }}>
            <div className="flex items-center gap-2 flex-wrap">
              <Input
                placeholder="Search entries..."
                value={entriesSearch}
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                  setEntriesSearch(e.target.value);
                  setEntriesPage(0);
                }}
                className="h-8 w-56 border bg-muted text-sm"
              />
              <Select
                value={entriesSeverity}
                onValueChange={(v: string) => {
                  setEntriesSeverity(v);
                  setEntriesPage(0);
                }}
              >
                <SelectTrigger className="h-8 w-36 border bg-muted text-sm">
                  <SelectValue placeholder="All Severity" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Severity</SelectItem>
                  <SelectItem value="critical">Critical</SelectItem>
                  <SelectItem value="high">High</SelectItem>
                  <SelectItem value="medium">Medium</SelectItem>
                  <SelectItem value="low">Low</SelectItem>
                </SelectContent>
              </Select>
              {entriesData && (
                <span className="ml-auto text-[12px] text-muted-foreground">
                  {entriesData.total.toLocaleString()} entries
                </span>
              )}
            </div>

            <div className="rounded-lg border border bg-background overflow-x-auto">
              <table className="w-full caption-bottom">
                <thead>
                  <tr className="border-b border">
                    {[
                      'Entry ID',
                      'Name',
                      'Severity',
                      'CVE Count',
                      'Published',
                      'Synced At',
                      'Action',
                    ].map((h) => (
                      <th
                        key={h}
                        className="px-3 py-2.5 text-left text-[10px] font-semibold uppercase tracking-wide text-muted-foreground whitespace-nowrap"
                      >
                        {h}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {!entriesData ? (
                    [...Array(5)].map((_, i) => (
                      <tr key={i}>
                        <td colSpan={7} className="px-3 py-2">
                          <Skeleton className="h-6 w-full" />
                        </td>
                      </tr>
                    ))
                  ) : entriesData.entries.length === 0 ? (
                    <tr>
                      <td
                        colSpan={7}
                        className="px-3 py-8 text-center text-sm text-muted-foreground"
                      >
                        No entries found.
                      </td>
                    </tr>
                  ) : (
                    entriesData.entries.map((entry) => (
                      <tr
                        key={entry.id}
                        className="border-b border-[var(--border)] hover:bg-card transition-colors"
                      >
                        <td className="px-3 py-2.5 font-mono text-[11px] text-muted-foreground">
                          {entry.id.slice(0, 16)}
                        </td>
                        <td className="px-3 py-2.5">
                          <Link
                            to={`/catalog/${entry.id}`}
                            className="text-[12px] font-mono hover:underline"
                            style={{ color: 'var(--accent)' }}
                          >
                            {entry.name}
                          </Link>
                        </td>
                        <td className="px-3 py-2.5">
                          <SeverityBadge severity={entry.severity} />
                        </td>
                        <td
                          className="px-3 py-2.5 text-[12px] font-bold"
                          style={{ color: 'var(--accent)' }}
                        >
                          {entry.cve_count}
                        </td>
                        <td className="px-3 py-2.5 text-[11.5px] text-muted-foreground">
                          {fmtDate(entry.release_date)}
                        </td>
                        <td className="px-3 py-2.5 text-[11.5px] text-muted-foreground">
                          {fmtDate(entry.updated_at)}
                        </td>
                        <td className="px-3 py-2.5">
                          <Link
                            to={`/catalog/${entry.id}`}
                            className="text-[11px]"
                            style={{ color: 'var(--accent)' }}
                          >
                            → View in Catalog
                          </Link>
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>

              {/* Pagination */}
              {entriesData && entriesData.total > ENTRIES_PAGE_SIZE && (
                <div className="flex items-center gap-1 px-4 py-3 border-t border">
                  <button
                    type="button"
                    onClick={() => setEntriesPage((p) => Math.max(0, p - 1))}
                    disabled={entriesPage === 0}
                    className="w-7 h-7 rounded flex items-center justify-center border border text-muted-foreground text-xs disabled:opacity-40 hover:border-accent hover:text-accent-foreground"
                  >
                    ◀
                  </button>
                  {Array.from({ length: Math.min(5, entriesTotalPages) }, (_, i) => {
                    const pageNum =
                      Math.max(0, Math.min(entriesPage - 2, entriesTotalPages - 5)) + i;
                    return (
                      <button
                        key={pageNum}
                        type="button"
                        onClick={() => setEntriesPage(pageNum)}
                        className={`w-7 h-7 rounded flex items-center justify-center border text-xs transition-colors ${
                          pageNum === entriesPage
                            ? 'bg-accent border-[var(--accent)] text-foreground'
                            : 'border text-muted-foreground hover:border-accent hover:text-accent-foreground'
                        }`}
                      >
                        {pageNum + 1}
                      </button>
                    );
                  })}
                  <button
                    type="button"
                    onClick={() => setEntriesPage((p) => p + 1)}
                    disabled={entriesPage + 1 >= entriesTotalPages}
                    className="w-7 h-7 rounded flex items-center justify-center border border text-muted-foreground text-xs disabled:opacity-40 hover:border-accent hover:text-accent-foreground"
                  >
                    ▶
                  </button>
                  <span className="ml-auto text-[12px] text-muted-foreground">
                    Showing {entriesPage * ENTRIES_PAGE_SIZE + 1}–
                    {Math.min((entriesPage + 1) * ENTRIES_PAGE_SIZE, entriesData.total)} of{' '}
                    {entriesData.total.toLocaleString()} entries
                  </span>
                </div>
              )}
            </div>
          </div>
        )}

        {/* ── Configuration Tab ── */}
        {activeTab === 'config' && (
          <div style={{ marginTop: 20 }}>
            <form onSubmit={handleSubmit(onSaveConfig)}>
              <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
                {/* Left: Feed Connection */}
                <Card className="bg-card border">
                  <CardHeader className="pb-2">
                    <CardTitle className="text-[13px] font-semibold">🔗 Feed Connection</CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-3">
                    <div>
                      <label className="block text-[12px] font-medium mb-1">Feed URL</label>
                      <Input
                        {...register('url')}
                        placeholder="https://example.com/feed"
                        className="border text-[12px]"
                        style={{ background: 'var(--bg-card)' }}
                      />
                      {errors.url && (
                        <p className="text-[11px] mt-1" style={{ color: 'var(--signal-critical)' }}>
                          {errors.url.message}
                        </p>
                      )}
                      <p className="text-[11px] text-muted-foreground mt-1">
                        The HTTPS endpoint to fetch feed data from
                      </p>
                    </div>
                    <div>
                      <label className="block text-[12px] font-medium mb-1">Sync Interval</label>
                      <Select
                        value={String(watch('sync_interval_seconds'))}
                        onValueChange={(v: string) => setValue('sync_interval_seconds', Number(v))}
                      >
                        <SelectTrigger
                          className="border text-[12px]"
                          style={{ background: 'var(--bg-card)' }}
                        >
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          {INTERVAL_OPTIONS.map((opt) => (
                            <SelectItem key={opt.value} value={String(opt.value)}>
                              {opt.label}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                    <div>
                      <label className="block text-[12px] font-medium mb-1">Authentication</label>
                      <Select
                        value={watch('auth_type')}
                        onValueChange={(v: string) => setValue('auth_type', v)}
                      >
                        <SelectTrigger
                          className="border text-[12px]"
                          style={{ background: 'var(--bg-card)' }}
                        >
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          {AUTH_OPTIONS.map((opt) => (
                            <SelectItem key={opt.value} value={opt.value}>
                              {opt.label}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                    <Button type="button" variant="outline" size="sm">
                      🔌 Test Connection
                    </Button>
                  </CardContent>
                </Card>

                {/* Right: Filters + Severity Mapping */}
                <div className="space-y-4">
                  {/* Filters */}
                  <Card className="bg-card border">
                    <CardHeader className="pb-2">
                      <CardTitle className="text-[13px] font-semibold">🎯 Filters</CardTitle>
                    </CardHeader>
                    <CardContent className="space-y-3">
                      <div>
                        <label className="block text-[12px] font-medium mb-1">
                          Severity Filter
                        </label>
                        <div className="flex flex-wrap gap-2 mt-1">
                          {SEVERITY_LEVELS.map((level) => {
                            const currentFilter = watch('severity_filter') ?? [];
                            const checked = currentFilter.includes(level);
                            return (
                              <label
                                key={level}
                                className="flex items-center gap-1.5 text-[12px] cursor-pointer"
                              >
                                <input
                                  type="checkbox"
                                  checked={checked}
                                  onChange={(e) => {
                                    const next = e.target.checked
                                      ? [...currentFilter, level]
                                      : currentFilter.filter((s) => s !== level);
                                    setValue('severity_filter', next);
                                  }}
                                  className="w-[13px] h-[13px] accent-[var(--accent)]"
                                />
                                {level}
                              </label>
                            );
                          })}
                        </div>
                      </div>
                      <div className="h-px w-full" style={{ background: 'var(--border)' }} />
                      <div>
                        <label className="block text-[12px] font-medium mb-1">OS Filter</label>
                        <div className="flex flex-wrap gap-2 mt-1">
                          {OS_OPTIONS.map((os) => {
                            const currentFilter = watch('os_filter') ?? [];
                            const checked = currentFilter.includes(os);
                            return (
                              <label
                                key={os}
                                className="flex items-center gap-1.5 text-[12px] cursor-pointer"
                              >
                                <input
                                  type="checkbox"
                                  checked={checked}
                                  onChange={(e) => {
                                    const next = e.target.checked
                                      ? [...currentFilter, os]
                                      : currentFilter.filter((s) => s !== os);
                                    setValue('os_filter', next);
                                  }}
                                  className="w-[13px] h-[13px] accent-[var(--accent)]"
                                />
                                {os}
                              </label>
                            );
                          })}
                        </div>
                      </div>
                    </CardContent>
                  </Card>

                  {/* Severity Mapping */}
                  <Card className="bg-card border">
                    <CardHeader className="pb-2">
                      <CardTitle className="text-[13px] font-semibold">
                        🗺 Severity Mapping
                      </CardTitle>
                    </CardHeader>
                    <CardContent>
                      {/* Header row */}
                      <div className="grid grid-cols-3 gap-2 text-[10px] uppercase tracking-wide text-muted-foreground pb-1 mb-1">
                        <div>Source Severity</div>
                        <div />
                        <div>PatchIQ Severity</div>
                      </div>
                      {SEVERITY_LEVELS.map((level) => {
                        const currentMapping = watch('severity_mapping') ?? {};
                        const mappedTo = currentMapping[level.toLowerCase()] ?? level;
                        return (
                          <div
                            key={level}
                            className="grid grid-cols-3 gap-2 items-center py-2 border-b border/50 last:border-0"
                          >
                            <div>
                              <SeverityBadge severity={level} />
                            </div>
                            <div className="text-center text-muted-foreground text-base">→</div>
                            <Select
                              value={mappedTo}
                              onValueChange={(v: string) => {
                                setValue('severity_mapping', {
                                  ...currentMapping,
                                  [level.toLowerCase()]: v,
                                });
                              }}
                            >
                              <SelectTrigger className="h-7 bg-[var(--bg-card)] border text-[11px]">
                                <SelectValue />
                              </SelectTrigger>
                              <SelectContent>
                                {SEVERITY_LEVELS.map((s) => (
                                  <SelectItem key={s} value={s}>
                                    {s}
                                  </SelectItem>
                                ))}
                              </SelectContent>
                            </Select>
                          </div>
                        );
                      })}

                      <div className="flex gap-2 mt-4">
                        <Button type="submit" size="sm" disabled={updateFeed.isPending}>
                          {updateFeed.isPending ? 'Saving…' : '💾 Save Configuration'}
                        </Button>
                        <Button type="button" variant="outline" size="sm">
                          Reset to Defaults
                        </Button>
                      </div>
                    </CardContent>
                  </Card>
                </div>
              </div>
            </form>
          </div>
        )}
      </div>
    </div>
  );
}
