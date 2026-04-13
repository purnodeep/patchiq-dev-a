import React, { useState } from 'react';
import { useCan } from '../../app/auth/AuthContext';
import { toast } from 'sonner';
import {
  Button,
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
  Skeleton,
} from '@patchiq/ui';
import { CheckCircle2, XCircle, Clock, ChevronDown, ChevronRight, RefreshCw } from 'lucide-react';
import {
  useNotificationHistory,
  useRetryNotification,
  type HistoryFilters,
} from '../../api/hooks/useNotifications';
import type { components } from '../../api/types';

type NotificationHistoryEntry = components['schemas']['NotificationHistoryEntry'];

const thStyle: React.CSSProperties = {
  fontFamily: 'var(--font-mono)',
  fontSize: 11,
  fontWeight: 600,
  textTransform: 'uppercase' as const,
  letterSpacing: '0.05em',
  color: 'var(--text-muted)',
  padding: '8px 12px',
  textAlign: 'left' as const,
  background: 'var(--bg-inset)',
  borderBottom: '1px solid var(--border)',
  whiteSpace: 'nowrap' as const,
};

function getStatusInfo(status: string): { icon: React.ReactNode; color: string } {
  switch (status) {
    case 'delivered':
      return {
        icon: <CheckCircle2 style={{ width: 13, height: 13 }} />,
        color: 'var(--accent)',
      };
    case 'failed':
      return {
        icon: <XCircle style={{ width: 13, height: 13 }} />,
        color: 'var(--signal-critical)',
      };
    case 'pending':
      return {
        icon: <Clock style={{ width: 13, height: 13 }} />,
        color: 'var(--signal-warning)',
      };
    default:
      return { icon: null, color: 'var(--text-muted)' };
  }
}

// Monochrome channel labels (Design Rule #1: Color = Signal only)
function getChannelColor(_channel: string): string {
  return 'var(--text-secondary)';
}

function ExpandedRow({ entry }: { entry: NotificationHistoryEntry }) {
  return (
    <tr>
      <td
        colSpan={7}
        style={{
          background: 'var(--bg-inset)',
          borderBottom: '1px solid var(--border)',
          padding: '12px 16px',
        }}
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          <div>
            <div
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 10,
                fontWeight: 600,
                textTransform: 'uppercase',
                letterSpacing: '0.05em',
                color: 'var(--text-muted)',
                marginBottom: 6,
              }}
            >
              Payload
            </div>
            <pre
              style={{
                background: 'var(--bg-page)',
                border: '1px solid var(--border)',
                borderRadius: 6,
                padding: '8px 12px',
                fontFamily: 'var(--font-mono)',
                fontSize: 11,
                color: 'var(--text-primary)',
                overflowX: 'auto',
                maxHeight: 120,
                margin: 0,
              }}
            >
              {entry.payload ? JSON.stringify(entry.payload, null, 2) : '{}'}
            </pre>
          </div>
          {entry.error_message && (
            <div>
              <div
                style={{
                  fontFamily: 'var(--font-mono)',
                  fontSize: 10,
                  fontWeight: 600,
                  textTransform: 'uppercase',
                  letterSpacing: '0.05em',
                  color: 'var(--signal-critical)',
                  marginBottom: 4,
                }}
              >
                Error
              </div>
              <div
                style={{
                  fontFamily: 'var(--font-sans)',
                  fontSize: 12,
                  color: 'var(--signal-critical)',
                }}
              >
                {entry.error_message}
              </div>
            </div>
          )}
        </div>
      </td>
    </tr>
  );
}

function HistoryRow({
  entry,
  onRetry,
  retrying,
}: {
  entry: NotificationHistoryEntry;
  onRetry: (id: string) => void;
  retrying: boolean;
}) {
  const can = useCan();
  const [expanded, setExpanded] = useState(false);
  const statusInfo = getStatusInfo(entry.status);
  const channelColor = getChannelColor(entry.channel_type ?? '');
  const sentAt = new Date(entry.created_at).toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  });

  const tdStyle: React.CSSProperties = {
    padding: '9px 12px',
    borderBottom: '1px solid var(--border)',
    background: expanded ? 'var(--bg-inset)' : undefined,
  };

  return (
    <React.Fragment>
      <tr
        style={{ cursor: 'pointer' }}
        onClick={() => setExpanded((v) => !v)}
        onMouseEnter={(e) =>
          ((e.currentTarget as HTMLTableRowElement).style.background = expanded
            ? 'var(--bg-inset)'
            : 'var(--bg-card-hover)')
        }
        onMouseLeave={(e) =>
          ((e.currentTarget as HTMLTableRowElement).style.background = expanded
            ? 'var(--bg-inset)'
            : '')
        }
      >
        {/* Expand icon */}
        <td style={{ ...tdStyle, width: 32, padding: '9px 8px 9px 14px' }}>
          {expanded ? (
            <ChevronDown style={{ width: 12, height: 12, color: 'var(--text-muted)' }} />
          ) : (
            <ChevronRight style={{ width: 12, height: 12, color: 'var(--text-muted)' }} />
          )}
        </td>

        {/* Event/category */}
        <td style={tdStyle}>
          <span
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 11,
              color: 'var(--text-secondary)',
            }}
          >
            {entry.category ?? entry.trigger_type}
          </span>
        </td>

        {/* Channel */}
        <td style={tdStyle}>
          <span
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 11,
              fontWeight: 600,
              color: channelColor,
              textTransform: 'uppercase',
              letterSpacing: '0.06em',
            }}
          >
            {entry.channel_type}
          </span>
        </td>

        {/* Recipient */}
        <td style={tdStyle}>
          <span
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 11,
              color: 'var(--text-secondary)',
            }}
          >
            {entry.recipient}
          </span>
        </td>

        {/* Subject */}
        <td style={tdStyle}>
          <span
            style={{
              fontFamily: 'var(--font-sans)',
              fontSize: 12,
              color: 'var(--text-primary)',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
              maxWidth: 260,
              display: 'block',
            }}
            title={entry.subject}
          >
            {entry.subject}
          </span>
        </td>

        {/* Status */}
        <td style={tdStyle}>
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 6,
              color: statusInfo.color,
            }}
          >
            {statusInfo.icon}
            <span
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 11,
                fontWeight: 500,
                textTransform: 'capitalize',
              }}
            >
              {entry.status}
            </span>
          </div>
        </td>

        {/* Sent at */}
        <td style={tdStyle}>
          <span
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 11,
              color: 'var(--text-faint)',
              whiteSpace: 'nowrap',
            }}
          >
            {sentAt}
          </span>
        </td>

        {/* Action */}
        <td style={tdStyle} onClick={(e) => e.stopPropagation()}>
          {entry.status === 'failed' && (
            <Button
              variant="outline"
              size="sm"
              disabled={retrying || !can('settings', 'update')}
              title={!can('settings', 'update') ? "You don't have permission" : undefined}
              onClick={() => onRetry(entry.id)}
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 10,
                color: 'var(--signal-critical)',
                borderColor: 'color-mix(in srgb, var(--signal-critical) 30%, transparent)',
              }}
            >
              {retrying ? (
                <RefreshCw
                  style={{ width: 11, height: 11, animation: 'spin 1s linear infinite' }}
                />
              ) : (
                'Retry'
              )}
            </Button>
          )}
        </td>
      </tr>
      {expanded && <ExpandedRow entry={entry} />}
    </React.Fragment>
  );
}

export function HistoryTab() {
  const [filters, setFilters] = useState<HistoryFilters>({});
  const { data, isLoading, fetchNextPage, hasNextPage } = useNotificationHistory(filters);
  const { mutateAsync: retry, isPending: retrying } = useRetryNotification();
  const [retryingId, setRetryingId] = useState<string | null>(null);

  const entries: NotificationHistoryEntry[] =
    data?.pages.flatMap((p) => (p as { data: NotificationHistoryEntry[] }).data ?? []) ?? [];

  const handleRetry = async (id: string) => {
    setRetryingId(id);
    try {
      await retry(id);
      toast.success('Notification queued for retry');
    } catch {
      toast.error('Retry failed — check channel configuration');
    } finally {
      setRetryingId(null);
    }
  };

  return (
    <div
      style={{
        padding: 24,
        display: 'flex',
        flexDirection: 'column',
        gap: 16,
      }}
    >
      {/* Filter bar */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
        <Select
          value={filters.channel_type ?? 'all'}
          onValueChange={(v) =>
            setFilters((f) => ({ ...f, channel_type: v === 'all' ? undefined : v }))
          }
        >
          <SelectTrigger
            className="h-8 w-[140px]"
            style={{ fontFamily: 'var(--font-mono)', fontSize: 11 }}
          >
            <SelectValue placeholder="All Channels" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Channels</SelectItem>
            <SelectItem value="email">Email</SelectItem>
            <SelectItem value="slack">Slack</SelectItem>
            <SelectItem value="webhook">Webhook</SelectItem>
            <SelectItem value="discord">Discord</SelectItem>
          </SelectContent>
        </Select>

        <Select
          value={filters.status ?? 'all'}
          onValueChange={(v) => setFilters((f) => ({ ...f, status: v === 'all' ? undefined : v }))}
        >
          <SelectTrigger
            className="h-8 w-[140px]"
            style={{ fontFamily: 'var(--font-mono)', fontSize: 11 }}
          >
            <SelectValue placeholder="All Statuses" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Statuses</SelectItem>
            <SelectItem value="delivered">Delivered</SelectItem>
            <SelectItem value="failed">Failed</SelectItem>
            <SelectItem value="pending">Pending</SelectItem>
          </SelectContent>
        </Select>

        <Select
          value={filters.category ?? 'all'}
          onValueChange={(v) =>
            setFilters((f) => ({ ...f, category: v === 'all' ? undefined : v }))
          }
        >
          <SelectTrigger
            className="h-8 w-[140px]"
            style={{ fontFamily: 'var(--font-mono)', fontSize: 11 }}
          >
            <SelectValue placeholder="All Categories" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Categories</SelectItem>
            <SelectItem value="deployments">Deployments</SelectItem>
            <SelectItem value="security">Security</SelectItem>
            <SelectItem value="compliance">Compliance</SelectItem>
            <SelectItem value="system">System</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {/* Table */}
      {isLoading ? (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} className="h-12 rounded-lg" />
          ))}
        </div>
      ) : (
        <div
          style={{
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 8,
            overflow: 'hidden',
          }}
        >
          <div style={{ overflowX: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse' }}>
              <thead>
                <tr>
                  <th style={{ ...thStyle, width: 32, padding: '8px 8px 8px 14px' }} />
                  <th style={thStyle}>Event</th>
                  <th style={thStyle}>Channel</th>
                  <th style={thStyle}>Recipient</th>
                  <th style={thStyle}>Subject</th>
                  <th style={thStyle}>Status</th>
                  <th style={thStyle}>Sent At</th>
                  <th style={thStyle}>Action</th>
                </tr>
              </thead>
              <tbody>
                {entries.length === 0 ? (
                  <tr>
                    <td
                      colSpan={8}
                      style={{
                        padding: '32px',
                        textAlign: 'center',
                        fontFamily: 'var(--font-sans)',
                        fontSize: 13,
                        color: 'var(--text-muted)',
                      }}
                    >
                      No notifications found
                    </td>
                  </tr>
                ) : (
                  entries.map((entry) => (
                    <HistoryRow
                      key={entry.id}
                      entry={entry}
                      onRetry={(id) => void handleRetry(id)}
                      retrying={retryingId === entry.id && retrying}
                    />
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {hasNextPage && (
        <div style={{ display: 'flex', justifyContent: 'center' }}>
          <Button
            variant="outline"
            size="sm"
            onClick={() => void fetchNextPage()}
            style={{ fontFamily: 'var(--font-mono)', fontSize: 11 }}
          >
            Load more
          </Button>
        </div>
      )}
    </div>
  );
}
