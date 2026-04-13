import { useNavigate } from 'react-router';
import { RingGauge } from '@patchiq/ui';
import {
  useDashboardStats,
  useLicenseBreakdown,
  useClientSummary,
} from '../../api/hooks/useDashboard';
import { useFeeds } from '../../api/hooks/useFeeds';
import type { Feed } from '../../types/feed';

const cardStyle: React.CSSProperties = {
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  borderRadius: 8,
  padding: '20px',
  boxShadow: 'var(--shadow-sm)',
  minHeight: 120,
  display: 'flex',
  flexDirection: 'column',
  cursor: 'pointer',
  transition: 'border-color 150ms ease, transform 150ms ease',
};

const labelStyle: React.CSSProperties = {
  fontSize: 10,
  fontWeight: 500,
  letterSpacing: '0.06em',
  color: 'var(--text-muted)',
  textTransform: 'uppercase',
  fontFamily: 'var(--font-mono)',
};

const valueStyle: React.CSSProperties = {
  fontSize: 28,
  fontWeight: 700,
  lineHeight: 1,
  color: 'var(--text-emphasis)',
  fontFamily: 'var(--font-mono)',
  letterSpacing: '-0.03em',
};

const sublabelStyle: React.CSSProperties = {
  fontSize: 11,
  color: 'var(--text-secondary)',
  fontFamily: 'var(--font-mono)',
};

function handleHoverIn(e: React.MouseEvent) {
  const el = e.currentTarget as HTMLDivElement;
  el.style.borderColor = 'var(--border-hover)';
  el.style.transform = 'translateY(-1px)';
}

function handleHoverOut(e: React.MouseEvent) {
  const el = e.currentTarget as HTMLDivElement;
  el.style.borderColor = 'var(--border)';
  el.style.transform = '';
}

export const StatCards = () => {
  const navigate = useNavigate();
  const { data: stats } = useDashboardStats();
  const { data: feeds } = useFeeds();
  const { data: licenseBreakdown } = useLicenseBreakdown();
  const { data: clients } = useClientSummary();

  const connectedClients = stats?.connected_clients ?? 0;
  const pendingClients = stats?.pending_clients ?? 0;
  const totalClients = connectedClients + pendingClients;
  const clientRatio = totalClients > 0 ? Math.round((connectedClients / totalClients) * 100) : 0;
  const clientSublabel = pendingClients > 0 ? `→ ${pendingClients} pending` : '↑ All connected';

  const catalogTotal = stats?.total_catalog_entries ?? 0;
  const catalogSublabel = catalogTotal > 0 ? '↑ Entries indexed' : '→ No entries yet';

  const totalFeeds = feeds?.length ?? 0;
  const activeFeeds = feeds?.filter((f: Feed) => f.enabled && f.status !== 'error').length ?? 0;
  const errorFeeds = feeds?.filter((f: Feed) => f.status === 'error').length ?? 0;
  const feedRatio = totalFeeds > 0 ? Math.round((activeFeeds / totalFeeds) * 100) : 0;
  const feedSublabel = errorFeeds > 0 ? `↓ ${errorFeeds} errored` : '↑ All healthy';

  const totalSeats =
    licenseBreakdown
      ?.filter((r) => r.status === 'active' || r.status === 'expiring')
      .reduce((sum, r) => sum + (r.total_endpoints ?? 0), 0) ?? 0;
  const usedSeats = clients?.reduce((sum, c) => sum + (c.endpoint_count ?? 0), 0) ?? 0;
  const seatRatio = totalSeats > 0 ? Math.round((usedSeats / totalSeats) * 100) : 0;
  const seatSublabel = totalSeats > 0 ? `→ ${totalSeats - usedSeats} available` : '→ No licenses';

  const pendingActions = pendingClients + errorFeeds;
  const actionSublabel =
    pendingActions > 0 ? `↓ ${pendingActions} require attention` : '↑ All clear';

  return (
    <div className="grid grid-cols-5 gap-3">
      {/* Connected Clients */}
      <div
        style={cardStyle}
        onClick={() => void navigate('/clients')}
        onMouseEnter={handleHoverIn}
        onMouseLeave={handleHoverOut}
      >
        <div style={labelStyle}>Connected Clients</div>
        <div
          style={{
            flex: 1,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            marginTop: 8,
            marginBottom: 8,
          }}
        >
          <div style={valueStyle}>{connectedClients}</div>
          <RingGauge value={clientRatio} size={48} strokeWidth={5} />
        </div>
        <div style={sublabelStyle}>{clientSublabel}</div>
      </div>

      {/* Catalog Entries */}
      <div
        style={cardStyle}
        onClick={() => void navigate('/catalog')}
        onMouseEnter={handleHoverIn}
        onMouseLeave={handleHoverOut}
      >
        <div style={labelStyle}>Catalog Entries</div>
        <div
          style={{
            flex: 1,
            display: 'flex',
            alignItems: 'center',
            marginTop: 8,
            marginBottom: 8,
          }}
        >
          <div style={valueStyle}>{catalogTotal.toLocaleString()}</div>
        </div>
        <div style={sublabelStyle}>{catalogSublabel}</div>
      </div>

      {/* Active Feeds */}
      <div
        style={cardStyle}
        onClick={() => void navigate('/feeds')}
        onMouseEnter={handleHoverIn}
        onMouseLeave={handleHoverOut}
      >
        <div style={labelStyle}>Active Feeds</div>
        <div
          style={{
            flex: 1,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            marginTop: 8,
            marginBottom: 8,
          }}
        >
          <div
            style={{
              ...valueStyle,
              color: errorFeeds > 0 ? 'var(--signal-critical)' : 'var(--text-emphasis)',
            }}
          >
            {activeFeeds}
            <span
              style={{
                fontSize: 14,
                fontWeight: 400,
                color: 'var(--text-muted)',
                marginLeft: 4,
              }}
            >
              / {totalFeeds}
            </span>
          </div>
          <RingGauge value={feedRatio} size={48} strokeWidth={5} colorByValue />
        </div>
        <div
          style={{
            ...sublabelStyle,
            color: errorFeeds > 0 ? 'var(--signal-critical)' : 'var(--text-secondary)',
          }}
        >
          {feedSublabel}
        </div>
      </div>

      {/* License Seats */}
      <div
        style={cardStyle}
        onClick={() => void navigate('/licenses')}
        onMouseEnter={handleHoverIn}
        onMouseLeave={handleHoverOut}
      >
        <div style={labelStyle}>License Seats</div>
        <div
          style={{
            flex: 1,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            marginTop: 8,
            marginBottom: 8,
          }}
        >
          <div style={valueStyle}>
            {usedSeats.toLocaleString()}
            <span
              style={{
                fontSize: 14,
                fontWeight: 400,
                color: 'var(--text-muted)',
                marginLeft: 4,
              }}
            >
              / {totalSeats.toLocaleString()}
            </span>
          </div>
          {totalSeats > 0 && <RingGauge value={seatRatio} size={48} strokeWidth={5} colorByValue />}
        </div>
        <div style={sublabelStyle}>{seatSublabel}</div>
      </div>

      {/* Pending Actions */}
      <div
        style={cardStyle}
        onClick={() => void navigate('/clients?status=pending')}
        onMouseEnter={handleHoverIn}
        onMouseLeave={handleHoverOut}
      >
        <div style={labelStyle}>Pending Actions</div>
        <div
          style={{
            flex: 1,
            display: 'flex',
            alignItems: 'center',
            marginTop: 8,
            marginBottom: 8,
          }}
        >
          <div
            style={{
              ...valueStyle,
              color: pendingActions > 0 ? 'var(--signal-warning)' : 'var(--text-emphasis)',
            }}
          >
            {pendingActions}
          </div>
        </div>
        <div
          style={{
            ...sublabelStyle,
            color: pendingActions > 0 ? 'var(--signal-warning)' : 'var(--text-secondary)',
          }}
        >
          {actionSublabel}
        </div>
      </div>
    </div>
  );
};
