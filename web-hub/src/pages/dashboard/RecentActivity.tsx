import { useState } from 'react';
import { useNavigate } from 'react-router';
import { useRecentActivity } from '../../api/hooks/useDashboard';
import type { ActivityItem } from '../../types/dashboard';

function eventRoute(evt: ActivityItem): string | null {
  if (evt.type.startsWith('feed.')) return `/feeds/${evt.resource_id}`;
  if (evt.type.startsWith('catalog')) return `/catalog/${evt.resource_id}`;
  if (evt.type.startsWith('client')) return `/clients/${evt.resource_id}`;
  if (evt.type.startsWith('license')) return `/licenses/${evt.resource_id}`;
  return null;
}

function eventColor(type: string): string {
  if (type.startsWith('feed.sync_completed')) return 'var(--text-muted)';
  if (type.startsWith('feed.sync_failed')) return 'var(--signal-critical)';
  if (type.startsWith('catalog')) return 'var(--text-muted)';
  if (type.startsWith('client')) return 'var(--signal-healthy)';
  if (type.startsWith('license.issued') || type.startsWith('license.assigned'))
    return 'var(--accent)';
  if (type.startsWith('license.revoked')) return 'var(--signal-warning)';
  return 'var(--text-muted)';
}

function describeEvent(evt: ActivityItem): string {
  const payload = evt.payload ?? {};
  switch (evt.type) {
    case 'feed.sync_completed': {
      const name = (payload.feed_name as string) || evt.resource_id.slice(0, 8);
      const count = payload.entries_fetched ?? payload.entries_ingested ?? '?';
      return `${name} feed synced -- ${count} entries ingested`;
    }
    case 'feed.sync_failed': {
      const name = (payload.feed_name as string) || evt.resource_id.slice(0, 8);
      const reason = (payload.error as string) || 'unknown error';
      return `${name} feed sync failed: ${reason.slice(0, 80)}`;
    }
    case 'catalog.created':
    case 'catalog.synced': {
      const name = (payload.name as string) || evt.resource_id.slice(0, 8);
      return `Catalog entry added: ${name}`;
    }
    case 'client.registered': {
      const name = (payload.hostname as string) || evt.resource_id.slice(0, 8);
      return `New client registered: ${name}`;
    }
    case 'client.approved': {
      const name = (payload.hostname as string) || evt.resource_id.slice(0, 8);
      return `Client approved: ${name}`;
    }
    case 'client.declined':
      return `Client declined: ${evt.resource_id.slice(0, 8)}`;
    case 'client.suspended':
      return `Client suspended: ${evt.resource_id.slice(0, 8)}`;
    case 'license.issued': {
      const customer = (payload.customer_name as string) || 'unknown';
      const tier = (payload.tier as string) || '?';
      return `License issued: ${tier} tier for ${customer}`;
    }
    case 'license.revoked':
      return `License revoked: ${evt.resource_id.slice(0, 8)}`;
    case 'license.assigned': {
      const clientId = (payload.client_id as string) || '?';
      return `License assigned to client ${clientId.slice(0, 8)}`;
    }
    default:
      return `${evt.type}: ${evt.resource} ${evt.action}`;
  }
}

function formatRelativeTime(timestamp: string): string {
  const now = Date.now();
  const then = new Date(timestamp).getTime();
  const diffMs = now - then;
  const diffMin = Math.floor(diffMs / 60_000);
  if (diffMin < 1) return 'just now';
  if (diffMin < 60) return `${diffMin}m ago`;
  const diffHrs = Math.floor(diffMin / 60);
  if (diffHrs < 24) return `${diffHrs}h ago`;
  const diffDays = Math.floor(diffHrs / 24);
  return `${diffDays}d ago`;
}

export const RecentActivity = () => {
  const navigate = useNavigate();
  const { data: activities, isLoading, error } = useRecentActivity();
  const [hovered, setHovered] = useState(false);

  return (
    <div
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        background: 'var(--bg-card)',
        border: `1px solid ${hovered ? 'var(--text-faint)' : 'var(--border)'}`,
        borderRadius: 8,
        boxShadow: 'var(--shadow-sm)',
        transition: 'border-color 150ms ease',
        display: 'flex',
        flexDirection: 'column',
      }}
    >
      <div
        style={{
          padding: '16px 20px 0',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}
      >
        <span style={{ fontSize: 13, color: 'var(--text-muted)', fontWeight: 400 }}>
          Recent Activity
        </span>
        <span style={{ fontSize: 10, fontFamily: 'var(--font-mono)', color: 'var(--text-faint)' }}>
          {isLoading ? 'Loading...' : `${activities?.length ?? 0} events`}
        </span>
      </div>
      <div style={{ padding: '12px 20px 16px', flex: 1 }}>
        {error && (
          <p style={{ fontSize: 12, color: 'var(--signal-critical)' }}>Failed to load activity</p>
        )}
        {!isLoading && activities && activities.length === 0 && (
          <p style={{ fontSize: 12, color: 'var(--text-muted)' }}>No recent activity</p>
        )}
        {(activities ?? []).slice(0, 10).map((evt, idx, arr) => {
          const color = eventColor(evt.type);
          const route = eventRoute(evt);
          return (
            <div
              key={evt.id}
              onClick={route ? () => void navigate(route) : undefined}
              style={{
                display: 'flex',
                alignItems: 'flex-start',
                gap: 10,
                padding: '8px 0',
                borderBottom: idx < arr.length - 1 ? '1px solid var(--border-faint)' : undefined,
                cursor: route ? 'pointer' : 'default',
                borderRadius: 4,
                transition: 'background 100ms',
              }}
              onMouseEnter={(e) => {
                if (route) e.currentTarget.style.background = 'var(--bg-inset)';
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.background = 'transparent';
              }}
            >
              <div
                style={{
                  width: 8,
                  height: 8,
                  borderRadius: '50%',
                  backgroundColor: color,
                  marginTop: 3,
                  flexShrink: 0,
                }}
              />
              <div style={{ flex: 1, minWidth: 0 }}>
                <span style={{ fontSize: 12, color: 'var(--text-secondary)' }}>
                  {describeEvent(evt)}
                </span>
              </div>
              <span
                style={{
                  fontFamily: 'var(--font-mono)',
                  fontSize: 10,
                  color: 'var(--text-faint)',
                  flexShrink: 0,
                }}
              >
                {formatRelativeTime(evt.timestamp)}
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
};
