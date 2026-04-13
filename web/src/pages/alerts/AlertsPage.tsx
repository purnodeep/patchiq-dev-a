import { useState, useCallback, useMemo, Fragment } from 'react';
import { useNavigate, useSearchParams } from 'react-router';
import {
  Settings2,
  Bell,
  AlertTriangle,
  CheckCircle2,
  ChevronDown,
  ChevronRight,
  ExternalLink,
  Clock,
  Eye,
  CheckCheck,
  XCircle,
  MoreHorizontal,
} from 'lucide-react';
import { Button, Skeleton, EmptyState, ErrorState } from '@patchiq/ui';
import { useCan } from '../../app/auth/AuthContext';
import {
  useAlerts,
  useAlertCount,
  useBulkUpdateAlertStatus,
  useUpdateAlertStatus,
} from '../../api/hooks/useAlerts';
import { DataTablePagination } from '../../components/data-table';
import { AlertRulesDialog } from './AlertRulesDialog';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type RefreshOption = '10000' | '30000' | '60000' | 'off';

interface AlertData {
  id: string;
  severity: string;
  category: string;
  title: string;
  description: string;
  resource: string;
  resource_id: string;
  status: string;
  created_at: string;
  payload?: Record<string, unknown>;
  event_id?: string;
  rule_id?: string;
  read_at?: string | null;
  acknowledged_at?: string | null;
  dismissed_at?: string | null;
}

// ---------------------------------------------------------------------------
// Utilities
// ---------------------------------------------------------------------------

function dateRangeToISO(range: string): { from?: string; to?: string } {
  const now = new Date();
  if (range === '24h') {
    const from = new Date(now.getTime() - 24 * 60 * 60 * 1000);
    return { from: from.toISOString() };
  }
  if (range === '7d') {
    const from = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000);
    return { from: from.toISOString() };
  }
  if (range === '30d') {
    const from = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000);
    return { from: from.toISOString() };
  }
  return {};
}

function formatRelativeTime(dateStr: string): string {
  const now = Date.now();
  const then = new Date(dateStr).getTime();
  const diffMs = now - then;
  const diffSec = Math.floor(diffMs / 1000);
  if (diffSec < 60) return `${diffSec}s ago`;
  const diffMin = Math.floor(diffSec / 60);
  if (diffMin < 60) return `${diffMin}m ago`;
  const diffHr = Math.floor(diffMin / 60);
  if (diffHr < 24) return `${diffHr}h ago`;
  const diffDay = Math.floor(diffHr / 24);
  return `${diffDay}d ago`;
}

function formatTimestamp(dateStr: string | null | undefined): string {
  if (!dateStr) return '\u2014';
  const d = new Date(dateStr);
  return d.toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  });
}

function getSeverityConfig(severity: string) {
  switch (severity.toLowerCase()) {
    case 'critical':
      return {
        bg: 'color-mix(in srgb, var(--signal-critical) 10%, transparent)',
        color: 'var(--signal-critical)',
        icon: <AlertTriangle style={{ width: 14, height: 14 }} />,
        label: 'Critical',
      };
    case 'warning':
      return {
        bg: 'color-mix(in srgb, var(--signal-warning) 10%, transparent)',
        color: 'var(--signal-warning)',
        icon: <AlertTriangle style={{ width: 14, height: 14 }} />,
        label: 'Warning',
      };
    case 'info':
    default:
      return {
        bg: 'color-mix(in srgb, var(--signal-healthy) 10%, transparent)',
        color: 'var(--signal-healthy)',
        icon: <CheckCircle2 style={{ width: 14, height: 14 }} />,
        label: 'Info',
      };
  }
}

function getEntityUrl(resource: string, resourceId: string): string | null {
  if (!resourceId) return null;
  switch (resource) {
    case 'deployment':
    case 'deployments':
      return `/deployments/${resourceId}`;
    case 'endpoint':
    case 'endpoints':
    case 'agent':
    case 'agents':
      return `/endpoints/${resourceId}`;
    case 'cve':
    case 'cves':
      return `/cves/${resourceId}`;
    case 'patch':
    case 'patches':
      return `/patches/${resourceId}`;
    case 'policy':
    case 'policies':
      return `/policies/${resourceId}`;
    case 'compliance':
      return `/compliance/frameworks/${resourceId}`;
    default:
      return null;
  }
}

function statusLabel(s: string): string {
  if (s === 'unread') return 'Active';
  return s.charAt(0).toUpperCase() + s.slice(1);
}

function statusColor(s: string): string {
  if (s === 'unread') return 'var(--accent)';
  if (s === 'acknowledged') return 'var(--signal-warning)';
  if (s === 'dismissed') return 'var(--text-faint)';
  return 'var(--text-muted)';
}

// ---------------------------------------------------------------------------
// Table styles
// ---------------------------------------------------------------------------

const TH: React.CSSProperties = {
  padding: '9px 12px',
  textAlign: 'left',
  fontFamily: 'var(--font-mono)',
  fontSize: 11,
  fontWeight: 600,
  letterSpacing: '0.05em',
  color: 'var(--text-muted)',
  textTransform: 'uppercase',
  whiteSpace: 'nowrap',
};
const TD: React.CSSProperties = {
  padding: '12px 12px',
  borderBottom: '1px solid var(--border)',
  verticalAlign: 'middle',
  overflow: 'hidden',
  textOverflow: 'ellipsis',
  whiteSpace: 'nowrap',
};

// ---------------------------------------------------------------------------
// Filter constants
// ---------------------------------------------------------------------------

const STATUS_OPTIONS = [
  { value: 'active', label: 'Active' },
  { value: 'acknowledged', label: 'Acknowledged' },
  { value: 'dismissed', label: 'Dismissed' },
  { value: 'all', label: 'All' },
] as const;

const CATEGORY_OPTIONS = [
  { value: 'all', label: 'All' },
  { value: 'deployments', label: 'Deployments' },
  { value: 'agents', label: 'Agents' },
  { value: 'cves', label: 'CVEs' },
  { value: 'compliance', label: 'Compliance' },
  { value: 'system', label: 'System' },
] as const;

const DATE_RANGES = [
  { value: '24h', label: 'Last 24h' },
  { value: '7d', label: 'Last 7 days' },
  { value: '30d', label: 'Last 30 days' },
  { value: 'custom', label: 'Custom Range' },
] as const;

// ---------------------------------------------------------------------------
// Sub-components
// ---------------------------------------------------------------------------

function StatCard({
  label,
  value,
  valueColor,
  active,
  onClick,
}: {
  label: string;
  value: number | undefined;
  valueColor?: string;
  active?: boolean;
  onClick: () => void;
}) {
  const [hovered, setHovered] = useState(false);
  return (
    <button
      type="button"
      onClick={onClick}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        flex: 1,
        minWidth: 0,
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'flex-start',
        padding: '12px 14px',
        background: active ? 'color-mix(in srgb, white 3%, transparent)' : 'var(--bg-card)',
        border: `1px solid ${active ? (valueColor ?? 'var(--accent)') : hovered ? 'var(--border-hover)' : 'var(--border)'}`,
        borderRadius: 8,
        cursor: 'pointer',
        transition: 'all 0.15s',
        outline: 'none',
        textAlign: 'left',
      }}
    >
      <span
        style={{
          fontFamily: 'var(--font-mono)',
          fontSize: 22,
          fontWeight: 700,
          lineHeight: 1,
          color: valueColor ?? 'var(--text-emphasis)',
          letterSpacing: '-0.02em',
        }}
      >
        {value ?? '\u2014'}
      </span>
      <span
        style={{
          fontSize: 10,
          fontFamily: 'var(--font-mono)',
          fontWeight: 500,
          textTransform: 'uppercase',
          letterSpacing: '0.05em',
          color: active ? (valueColor ?? 'var(--accent)') : 'var(--text-muted)',
          marginTop: 4,
        }}
      >
        {label}
      </span>
    </button>
  );
}

function SortHeader({
  label,
  colKey,
  sortCol,
  sortDir,
  onSort,
}: {
  label: string;
  colKey: string;
  sortCol: string | null;
  sortDir: 'asc' | 'desc';
  onSort: (col: string) => void;
}) {
  const active = sortCol === colKey;
  const [hovered, setHovered] = useState(false);
  return (
    <th
      style={{ ...TH, cursor: 'pointer', userSelect: 'none' }}
      onClick={() => onSort(colKey)}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
        <span style={{ color: active ? 'var(--text-emphasis)' : undefined }}>{label}</span>
        <svg
          width="10"
          height="10"
          viewBox="0 0 10 10"
          fill="none"
          style={{
            opacity: active ? 1 : hovered ? 0.5 : 0,
            transition: 'opacity 0.15s',
            flexShrink: 0,
          }}
        >
          {(!active || sortDir === 'asc') && (
            <path
              d="M5 2L8 5.5H2L5 2Z"
              fill={active ? 'var(--text-emphasis)' : 'var(--text-muted)'}
            />
          )}
          {(!active || sortDir === 'desc') && (
            <path
              d="M5 8L2 4.5H8L5 8Z"
              fill={active ? 'var(--text-emphasis)' : 'var(--text-muted)'}
            />
          )}
        </svg>
      </div>
    </th>
  );
}

function CB({ on, onClick }: { on: boolean; onClick: (e: React.MouseEvent) => void }) {
  return (
    <div
      onClick={onClick}
      style={{
        width: 14,
        height: 14,
        borderRadius: 3,
        border: `1.5px solid ${on ? 'var(--accent)' : 'var(--border-hover)'}`,
        background: on ? 'var(--accent)' : 'transparent',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        cursor: 'pointer',
        flexShrink: 0,
      }}
    >
      {on && (
        <svg width="8" height="6" viewBox="0 0 8 6" fill="none">
          <path
            d="M1 3L3 5L7 1"
            stroke="currentColor"
            strokeWidth="1.5"
            strokeLinecap="round"
            strokeLinejoin="round"
          />
        </svg>
      )}
    </div>
  );
}

function RowMenu({
  alert,
  onStatusChange,
}: {
  alert: AlertData;
  onStatusChange: (id: string, status: string) => void;
}) {
  const can = useCan();
  const [open, setOpen] = useState(false);
  const items = [
    ...(alert.status !== 'read' ? [{ label: 'Mark Read', status: 'read', danger: false }] : []),
    ...(alert.status !== 'acknowledged'
      ? [{ label: 'Acknowledge', status: 'acknowledged', danger: false }]
      : []),
    ...(alert.status !== 'dismissed'
      ? [{ label: 'Dismiss', status: 'dismissed', danger: true }]
      : []),
  ];

  return (
    <div style={{ position: 'relative' }}>
      <button
        type="button"
        onClick={(e) => {
          e.stopPropagation();
          setOpen((p) => !p);
        }}
        onBlur={() => setTimeout(() => setOpen(false), 150)}
        style={{
          width: 24,
          height: 24,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          background: open ? 'color-mix(in srgb, white 6%, transparent)' : 'transparent',
          border: 'none',
          borderRadius: 4,
          cursor: 'pointer',
          color: open ? 'var(--text-primary)' : 'var(--text-muted)',
          padding: 0,
          transition: 'color 0.15s, background 0.15s',
        }}
        onMouseEnter={(e) => {
          e.currentTarget.style.color = 'var(--text-primary)';
          e.currentTarget.style.background = 'color-mix(in srgb, white 6%, transparent)';
        }}
        onMouseLeave={(e) => {
          if (!open) {
            e.currentTarget.style.color = 'var(--text-muted)';
            e.currentTarget.style.background = 'transparent';
          }
        }}
      >
        <MoreHorizontal style={{ width: 14, height: 14 }} />
      </button>
      {open && (
        <div
          style={{
            position: 'absolute',
            right: 0,
            top: '100%',
            marginTop: 4,
            minWidth: 150,
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 8,
            boxShadow: 'var(--shadow-lg, 0 8px 24px rgba(0,0,0,.25))',
            zIndex: 50,
            padding: '4px 0',
            overflow: 'hidden',
          }}
        >
          {items.map((item) => (
            <Fragment key={item.label}>
              {item.danger && items.length > 1 && (
                <div style={{ height: 1, background: 'var(--border)', margin: '4px 0' }} />
              )}
              <button
                type="button"
                disabled={!can('alerts', 'update')}
                title={!can('alerts', 'update') ? "You don't have permission" : undefined}
                onMouseDown={(e) => {
                  e.preventDefault();
                  e.stopPropagation();
                  onStatusChange(alert.id, item.status);
                  setOpen(false);
                }}
                style={{
                  display: 'block',
                  width: '100%',
                  padding: '7px 14px',
                  fontSize: 12,
                  fontFamily: 'var(--font-sans)',
                  color: item.danger ? 'var(--signal-critical)' : 'var(--text-secondary)',
                  background: 'transparent',
                  border: 'none',
                  cursor: 'pointer',
                  textAlign: 'left',
                }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.background = 'var(--bg-inset)';
                  if (!item.danger) e.currentTarget.style.color = 'var(--text-primary)';
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.background = 'transparent';
                  e.currentTarget.style.color = item.danger
                    ? 'var(--signal-critical)'
                    : 'var(--text-secondary)';
                }}
              >
                {item.label}
              </button>
            </Fragment>
          ))}
        </div>
      )}
    </div>
  );
}

function TimelineEntry({
  icon: Icon,
  label,
  value,
}: {
  icon: typeof Clock;
  label: string;
  value: string;
}) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 11 }}>
      <Icon style={{ width: 12, height: 12, color: 'var(--text-faint)', flexShrink: 0 }} />
      <span style={{ color: 'var(--text-muted)', width: 80 }}>{label}</span>
      <span
        style={{
          fontFamily: 'var(--font-mono)',
          fontSize: 11,
          color: value === '\u2014' ? 'var(--text-faint)' : 'var(--text-secondary)',
        }}
      >
        {value}
      </span>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Expanded Row Detail (table version)
// ---------------------------------------------------------------------------

function ExpandedDetail({
  alert,
  navigate,
  onStatusChange,
}: {
  alert: AlertData;
  navigate: (path: string) => void;
  onStatusChange: (id: string, status: string) => void;
}) {
  const entityUrl = getEntityUrl(alert.resource, alert.resource_id);
  const payloadEntries =
    alert.payload && typeof alert.payload === 'object'
      ? Object.entries(alert.payload).filter(([, v]) => v !== null && v !== undefined && v !== '')
      : [];

  const CARD: React.CSSProperties = {
    background: 'var(--bg-inset)',
    border: '1px solid var(--border)',
    borderRadius: 5,
    padding: '8px 10px',
  };
  const LBL: React.CSSProperties = {
    fontFamily: 'var(--font-mono)',
    fontSize: 9,
    fontWeight: 600,
    textTransform: 'uppercase',
    letterSpacing: '0.07em',
    color: 'var(--text-muted)',
    marginBottom: 6,
  };
  const BTN: React.CSSProperties = {
    display: 'inline-flex',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 4,
    padding: '5px 8px',
    borderRadius: 5,
    fontSize: 11,
    fontWeight: 500,
    cursor: 'pointer',
    border: '1px solid var(--border)',
    background: 'transparent',
    color: 'var(--text-secondary)',
    letterSpacing: '0.01em',
    width: '100%',
    fontFamily: 'var(--font-sans)',
    whiteSpace: 'nowrap',
  };

  return (
    <tr>
      <td colSpan={8} style={{ padding: 0, border: 'none' }}>
        <div
          style={{
            padding: '8px 10px',
            background: 'var(--bg-page)',
            borderTop: '1px solid var(--border)',
            borderBottom: '1px solid var(--border)',
            display: 'flex',
            gap: 8,
            alignItems: 'stretch',
            minWidth: 0,
            position: 'sticky',
            left: 0,
            width: 'calc(100vw - 280px)',
            maxWidth: 'calc(100vw - 280px)',
            boxSizing: 'border-box',
          }}
          onClick={(e) => e.stopPropagation()}
        >
          {/* Details card */}
          <div style={{ ...CARD, flex: '1 1 0', minWidth: 0 }}>
            <div style={LBL}>Details</div>
            <p
              style={{
                fontSize: 11,
                color: 'var(--text-secondary)',
                lineHeight: 1.5,
                margin: '0 0 8px',
              }}
            >
              {alert.description || '\u2014'}
            </p>
            {payloadEntries.length > 0 && (
              <div
                style={{
                  background: 'var(--bg-card)',
                  border: '1px solid var(--border)',
                  borderRadius: 5,
                  padding: '8px 10px',
                  fontFamily: 'var(--font-mono)',
                  fontSize: 11,
                  lineHeight: 1.6,
                  color: 'var(--text-secondary)',
                }}
              >
                {payloadEntries.map(([key, val]) => (
                  <div key={key}>
                    <span style={{ color: 'var(--text-muted)' }}>{key}:</span>{' '}
                    <span style={{ color: 'var(--text-emphasis)' }}>
                      {typeof val === 'object' ? JSON.stringify(val) : String(val)}
                    </span>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Timeline card */}
          <div style={{ ...CARD, flex: '1 1 0', minWidth: 0 }}>
            <div style={LBL}>Timeline</div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
              <TimelineEntry
                icon={Clock}
                label="Created"
                value={formatTimestamp(alert.created_at)}
              />
              <TimelineEntry icon={Eye} label="Read" value={formatTimestamp(alert.read_at)} />
              <TimelineEntry
                icon={CheckCheck}
                label="Acknowledged"
                value={formatTimestamp(alert.acknowledged_at)}
              />
              <TimelineEntry
                icon={XCircle}
                label="Dismissed"
                value={formatTimestamp(alert.dismissed_at)}
              />
            </div>
          </div>

          {/* Action buttons — outside cards */}
          <div
            style={{
              display: 'flex',
              flexDirection: 'column',
              gap: 6,
              flexShrink: 0,
              width: 110,
              alignSelf: 'start',
            }}
          >
            {alert.status !== 'acknowledged' && (
              <button
                type="button"
                style={{
                  ...BTN,
                  color: 'var(--btn-accent-text, #000)',
                  borderColor: 'var(--accent)',
                  background: 'var(--accent)',
                }}
                onClick={(e) => {
                  e.stopPropagation();
                  onStatusChange(alert.id, 'acknowledged');
                }}
              >
                <CheckCheck style={{ width: 12, height: 12 }} />
                Acknowledge
              </button>
            )}
            {alert.status !== 'dismissed' && (
              <button
                type="button"
                style={BTN}
                onClick={(e) => {
                  e.stopPropagation();
                  onStatusChange(alert.id, 'dismissed');
                }}
              >
                <XCircle style={{ width: 12, height: 12 }} />
                Dismiss
              </button>
            )}
            {entityUrl && (
              <button
                type="button"
                style={BTN}
                onClick={(e) => {
                  e.stopPropagation();
                  navigate(entityUrl);
                }}
              >
                <ExternalLink style={{ width: 12, height: 12 }} />
                View Source
              </button>
            )}
          </div>
        </div>
      </td>
    </tr>
  );
}

// ---------------------------------------------------------------------------
// Card view for grid mode
// ---------------------------------------------------------------------------

function AlertCard({
  alert,
  selected,
  onSelect,
  onStatusChange,
}: {
  alert: AlertData;
  selected: boolean;
  onSelect: (id: string) => void;
  onStatusChange: (id: string, status: string) => void;
}) {
  const can = useCan();
  const navigate = useNavigate();
  const config = getSeverityConfig(alert.severity);
  const entityUrl = getEntityUrl(alert.resource, alert.resource_id);

  return (
    <div
      style={{
        background: selected
          ? 'color-mix(in srgb, var(--accent) 4%, var(--bg-card))'
          : 'var(--bg-card)',
        border: `1px solid ${selected ? 'var(--accent)' : 'var(--border)'}`,
        borderRadius: 8,
        padding: '14px 16px',
        display: 'flex',
        flexDirection: 'column',
        gap: 10,
        transition: 'border-color 0.15s, background 0.15s',
      }}
    >
      {/* Top row: checkbox + severity badge + status */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        <CB
          on={selected}
          onClick={(e) => {
            e.stopPropagation();
            onSelect(alert.id);
          }}
        />
        <span
          style={{
            fontSize: 10,
            fontWeight: 600,
            fontFamily: 'var(--font-mono)',
            textTransform: 'uppercase',
            letterSpacing: '0.04em',
            background: config.bg,
            color: config.color,
            padding: '2px 8px',
            borderRadius: 9999,
            display: 'inline-flex',
            alignItems: 'center',
            gap: 4,
          }}
        >
          {config.icon}
          {alert.severity}
        </span>
        <span style={{ flex: 1 }} />
        <span
          style={{
            fontSize: 10,
            fontFamily: 'var(--font-mono)',
            fontWeight: 500,
            color: statusColor(alert.status),
          }}
        >
          {statusLabel(alert.status)}
        </span>
      </div>

      {/* Title */}
      <div
        style={{
          fontSize: 13,
          fontWeight: 600,
          color: 'var(--text-emphasis)',
          lineHeight: 1.4,
          overflow: 'hidden',
          textOverflow: 'ellipsis',
          whiteSpace: 'nowrap',
        }}
      >
        {alert.title}
      </div>

      {/* Description */}
      <div
        style={{
          fontSize: 12,
          color: 'var(--text-secondary)',
          lineHeight: 1.4,
          overflow: 'hidden',
          textOverflow: 'ellipsis',
          display: '-webkit-box',
          WebkitLineClamp: 2,
          WebkitBoxOrient: 'vertical',
        }}
      >
        {alert.description}
      </div>

      {/* Category + Resource */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
        <span
          style={{
            fontSize: 11,
            fontFamily: 'var(--font-mono)',
            color: 'var(--text-muted)',
          }}
        >
          {alert.category}
        </span>
        <span style={{ color: 'var(--border)', fontSize: 11 }}>&middot;</span>
        {entityUrl ? (
          <button
            onClick={() => navigate(entityUrl)}
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: 4,
              background: 'none',
              border: 'none',
              padding: 0,
              cursor: 'pointer',
              fontFamily: 'var(--font-mono)',
              fontSize: 11,
              color: 'var(--accent)',
            }}
          >
            <ExternalLink style={{ width: 10, height: 10 }} />
            {alert.resource}/
            {alert.resource_id.length > 12
              ? `${alert.resource_id.slice(0, 8)}\u2026`
              : alert.resource_id}
          </button>
        ) : (
          <span
            style={{ fontFamily: 'var(--font-mono)', fontSize: 11, color: 'var(--text-muted)' }}
          >
            {alert.resource}/
            {alert.resource_id.length > 12
              ? `${alert.resource_id.slice(0, 8)}\u2026`
              : alert.resource_id}
          </span>
        )}
        <span style={{ flex: 1 }} />
        <span style={{ fontSize: 11, color: 'var(--text-faint)', fontFamily: 'var(--font-mono)' }}>
          {formatRelativeTime(alert.created_at)}
        </span>
      </div>

      {/* Actions */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 6,
          borderTop: '1px solid var(--border)',
          paddingTop: 10,
          marginTop: 2,
        }}
      >
        {alert.status !== 'acknowledged' && (
          <button
            onClick={() => onStatusChange(alert.id, 'acknowledged')}
            disabled={!can('alerts', 'update')}
            title={!can('alerts', 'update') ? "You don't have permission" : undefined}
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: 4,
              padding: '4px 10px',
              borderRadius: 5,
              fontSize: 11,
              fontWeight: 500,
              cursor: !can('alerts', 'update') ? 'not-allowed' : 'pointer',
              border: '1px solid var(--border)',
              background: 'transparent',
              color: 'var(--text-secondary)',
              fontFamily: 'var(--font-sans)',
              opacity: !can('alerts', 'update') ? 0.5 : undefined,
            }}
          >
            <CheckCheck style={{ width: 11, height: 11 }} />
            Acknowledge
          </button>
        )}
        {alert.status !== 'dismissed' && (
          <button
            onClick={() => onStatusChange(alert.id, 'dismissed')}
            disabled={!can('alerts', 'update')}
            title={!can('alerts', 'update') ? "You don't have permission" : undefined}
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: 4,
              padding: '4px 10px',
              borderRadius: 5,
              fontSize: 11,
              fontWeight: 500,
              cursor: !can('alerts', 'update') ? 'not-allowed' : 'pointer',
              border: '1px solid var(--border)',
              background: 'transparent',
              color: 'var(--text-muted)',
              fontFamily: 'var(--font-sans)',
              opacity: !can('alerts', 'update') ? 0.5 : undefined,
            }}
          >
            <XCircle style={{ width: 11, height: 11 }} />
            Dismiss
          </button>
        )}
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// AlertsPage
// ---------------------------------------------------------------------------

export const AlertsPage = () => {
  const can = useCan();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();

  // View mode
  const viewMode = (searchParams.get('view') === 'card' ? 'card' : 'list') as 'list' | 'card';
  const setViewMode = useCallback(
    (mode: 'list' | 'card') => {
      setSearchParams(
        (prev) => {
          const next = new URLSearchParams(prev);
          if (mode === 'list') next.delete('view');
          else next.set('view', mode);
          return next;
        },
        { replace: true },
      );
    },
    [setSearchParams],
  );

  // Filter state
  const [severity, setSeverity] = useState('all');
  const [status, setStatus] = useState('active');
  const [category, setCategory] = useState('all');
  const [search, setSearch] = useState('');
  const [dateRange, setDateRange] = useState('7d');
  const [fromDate, setFromDate] = useState('');
  const [toDate, setToDate] = useState('');

  // Sort
  const [sortCol, setSortCol] = useState<string | null>(null);
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('desc');

  // Pagination
  const [cursors, setCursors] = useState<string[]>([]);
  const currentCursor = cursors[cursors.length - 1];
  const resetCursors = useCallback(() => setCursors([]), []);

  // Selection
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());

  // Expand
  const [expandedId, setExpandedId] = useState<string | null>(null);

  // Auto-refresh
  const [refreshInterval, setRefreshInterval] = useState<RefreshOption>('30000');

  // Rules dialog
  const [rulesDialogOpen, setRulesDialogOpen] = useState(false);

  // Date range computation
  const { from: computedFrom, to: computedTo } = useMemo(() => {
    if (dateRange === 'custom') {
      return {
        from: fromDate ? new Date(fromDate).toISOString() : undefined,
        to: toDate ? new Date(toDate + 'T23:59:59').toISOString() : undefined,
      };
    }
    return dateRangeToISO(dateRange);
  }, [dateRange, fromDate, toDate]);

  // Map filter state to API params
  const apiParams = useMemo(
    () => ({
      cursor: currentCursor,
      limit: 50,
      severity: severity !== 'all' ? severity : undefined,
      status:
        status === 'active'
          ? 'unread'
          : status === 'acknowledged'
            ? 'acknowledged'
            : status === 'dismissed'
              ? 'dismissed'
              : undefined,
      category: category !== 'all' ? category : undefined,
      search: search || undefined,
      from_date: computedFrom,
      to_date: computedTo,
    }),
    [currentCursor, severity, status, category, search, computedFrom, computedTo],
  );

  const refetchMs = refreshInterval === 'off' ? undefined : Number(refreshInterval);
  const { data, isLoading, isError, refetch } = useAlerts(apiParams, refetchMs);
  const { data: countData } = useAlertCount(
    { from_date: computedFrom, to_date: computedTo },
    refetchMs ?? 30000,
  );
  const bulkUpdate = useBulkUpdateAlertStatus();
  const singleUpdate = useUpdateAlertStatus();

  const alerts = data?.data ?? [];
  const unreadCount = countData?.total_unread ?? 0;
  const criticalCount = countData?.critical_unread ?? 0;
  const warningCount = countData?.warning_unread ?? 0;
  const infoCount =
    countData?.info_unread ?? Math.max(0, unreadCount - criticalCount - warningCount);
  const totalCount = criticalCount + warningCount + infoCount;

  // Sort handler
  const handleSort = useCallback(
    (col: string) => {
      if (sortCol === col) {
        setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'));
      } else {
        setSortCol(col);
        setSortDir('desc');
      }
    },
    [sortCol],
  );

  // Client-side sort
  const sortedAlerts = useMemo(() => {
    if (!sortCol) return alerts;
    const severityOrder: Record<string, number> = { critical: 0, warning: 1, info: 2 };
    const copy = [...alerts];
    copy.sort((a: AlertData, b: AlertData) => {
      let cmp = 0;
      switch (sortCol) {
        case 'severity':
          cmp = (severityOrder[a.severity] ?? 3) - (severityOrder[b.severity] ?? 3);
          break;
        case 'title':
          cmp = a.title.localeCompare(b.title);
          break;
        case 'category':
          cmp = a.category.localeCompare(b.category);
          break;
        case 'resource':
          cmp = a.resource.localeCompare(b.resource);
          break;
        case 'time':
          cmp = new Date(a.created_at).getTime() - new Date(b.created_at).getTime();
          break;
        case 'status':
          cmp = a.status.localeCompare(b.status);
          break;
        default:
          break;
      }
      return sortDir === 'asc' ? cmp : -cmp;
    });
    return copy;
  }, [alerts, sortCol, sortDir]);

  // Filter change handlers
  const handleSeverityChange = (v: string) => {
    setSeverity(v);
    resetCursors();
    setSelectedIds(new Set());
  };
  const handleStatusChange = (v: string) => {
    setStatus(v);
    resetCursors();
    setSelectedIds(new Set());
  };
  const handleCategoryChange = (v: string) => {
    setCategory(v);
    resetCursors();
    setSelectedIds(new Set());
  };
  const handleSearchChange = (v: string) => {
    setSearch(v);
    resetCursors();
    setSelectedIds(new Set());
  };
  const handleDateRangeChange = (v: string) => {
    setDateRange(v);
    resetCursors();
    setSelectedIds(new Set());
  };

  // Selection handlers
  const handleSelect = useCallback((id: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }, []);

  const handleSelectAll = useCallback(() => {
    if (selectedIds.size === sortedAlerts.length) {
      setSelectedIds(new Set());
    } else {
      setSelectedIds(new Set(sortedAlerts.map((a: AlertData) => a.id)));
    }
  }, [selectedIds.size, sortedAlerts]);

  const handleStatusChangeForAlert = useCallback(
    (id: string, newStatus: string) => {
      singleUpdate.mutate({ id, status: newStatus });
    },
    [singleUpdate],
  );

  const handleBulkAction = useCallback(
    (newStatus: string) => {
      const ids = Array.from(selectedIds);
      if (ids.length === 0) return;
      bulkUpdate.mutate({ ids, status: newStatus }, { onSuccess: () => setSelectedIds(new Set()) });
    },
    [selectedIds, bulkUpdate],
  );

  const hasActiveFilters =
    severity !== 'all' || status !== 'active' || category !== 'all' || search !== '';

  // Error state
  if (isError) {
    return (
      <div style={{ padding: 24 }}>
        <ErrorState
          title="Failed to load alerts"
          message="An unexpected error occurred. Please try again."
          onRetry={refetch}
        />
      </div>
    );
  }

  return (
    <div
      style={{
        background: 'var(--bg-page)',
        minHeight: '100%',
        padding: '24px',
        display: 'flex',
        flexDirection: 'column',
        gap: 16,
        maxWidth: 'var(--max-content-width, 1400px)',
      }}
    >
      {/* Stat Cards Row */}
      <div style={{ display: 'flex', gap: 10 }}>
        <StatCard
          label="Total"
          value={totalCount}
          active={severity === 'all'}
          onClick={() => handleSeverityChange('all')}
        />
        <StatCard
          label="Critical"
          value={criticalCount}
          valueColor="var(--signal-critical)"
          active={severity === 'critical'}
          onClick={() => handleSeverityChange(severity === 'critical' ? 'all' : 'critical')}
        />
        <StatCard
          label="Warning"
          value={warningCount}
          valueColor="var(--signal-warning)"
          active={severity === 'warning'}
          onClick={() => handleSeverityChange(severity === 'warning' ? 'all' : 'warning')}
        />
        <StatCard
          label="Info"
          value={infoCount}
          valueColor="var(--signal-healthy)"
          active={severity === 'info'}
          onClick={() => handleSeverityChange(severity === 'info' ? 'all' : 'info')}
        />
      </div>

      {/* Filter Bar + Actions */}
      <div style={{ display: 'flex', alignItems: 'stretch', gap: 8 }}>
        {/* Filter Bar */}
        <div
          style={{
            flex: 1,
            display: 'flex',
            alignItems: 'center',
            gap: 8,
            flexWrap: 'wrap',
            padding: '10px 14px',
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 8,
            boxShadow: 'var(--shadow-sm)',
          }}
        >
          {/* Search */}
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 8,
              padding: '5px 10px',
              border: '1px solid var(--border)',
              borderRadius: 6,
              background: 'var(--bg-inset)',
              flex: 1,
              maxWidth: 280,
              transition: 'border-color 0.15s',
            }}
            onFocusCapture={(e) => (e.currentTarget.style.borderColor = 'var(--accent)')}
            onBlurCapture={(e) => (e.currentTarget.style.borderColor = 'var(--border)')}
          >
            <svg
              width="12"
              height="12"
              viewBox="0 0 24 24"
              fill="none"
              stroke="var(--text-muted)"
              strokeWidth="2.5"
              aria-hidden="true"
            >
              <circle cx="11" cy="11" r="8" />
              <path d="M21 21l-4.35-4.35" />
            </svg>
            <input
              type="text"
              placeholder="Search alerts..."
              value={search}
              onChange={(e) => handleSearchChange(e.target.value)}
              style={{
                background: 'transparent',
                border: 'none',
                outline: 'none',
                fontSize: 12,
                color: 'var(--text-primary)',
                width: '100%',
              }}
            />
            {search && (
              <button
                type="button"
                aria-label="Clear search"
                onClick={() => handleSearchChange('')}
                style={{
                  width: 16,
                  height: 16,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  background: 'transparent',
                  border: 'none',
                  cursor: 'pointer',
                  color: 'var(--text-muted)',
                  padding: 0,
                }}
              >
                <svg
                  width="10"
                  height="10"
                  viewBox="0 0 10 10"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="1.5"
                >
                  <path d="M2 2l6 6M8 2l-6 6" />
                </svg>
              </button>
            )}
          </div>

          {/* Status dropdown */}
          <select
            aria-label="Filter by status"
            value={status}
            onChange={(e) => handleStatusChange(e.target.value)}
            style={{
              background: 'var(--bg-card)',
              border: '1px solid var(--border)',
              borderRadius: 6,
              padding: '5px 10px',
              fontSize: 11.5,
              color: status !== 'active' ? 'var(--text-primary)' : 'var(--text-secondary)',
              outline: 'none',
              cursor: 'pointer',
            }}
            onFocus={(e) => (e.currentTarget.style.borderColor = 'var(--accent)')}
            onBlur={(e) => (e.currentTarget.style.borderColor = 'var(--border)')}
          >
            {STATUS_OPTIONS.map((opt) => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>

          {/* Category dropdown */}
          <select
            aria-label="Filter by category"
            value={category}
            onChange={(e) => handleCategoryChange(e.target.value)}
            style={{
              background: 'var(--bg-card)',
              border: '1px solid var(--border)',
              borderRadius: 6,
              padding: '5px 10px',
              fontSize: 11.5,
              color: category !== 'all' ? 'var(--text-primary)' : 'var(--text-secondary)',
              outline: 'none',
              cursor: 'pointer',
            }}
            onFocus={(e) => (e.currentTarget.style.borderColor = 'var(--accent)')}
            onBlur={(e) => (e.currentTarget.style.borderColor = 'var(--border)')}
          >
            {CATEGORY_OPTIONS.map((opt) => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>

          {/* Date range */}
          <select
            aria-label="Filter by date range"
            value={dateRange}
            onChange={(e) => handleDateRangeChange(e.target.value)}
            style={{
              background: 'var(--bg-card)',
              border: '1px solid var(--border)',
              borderRadius: 6,
              padding: '5px 10px',
              fontSize: 11.5,
              color: dateRange !== '24h' ? 'var(--text-primary)' : 'var(--text-secondary)',
              outline: 'none',
              cursor: 'pointer',
            }}
            onFocus={(e) => (e.currentTarget.style.borderColor = 'var(--accent)')}
            onBlur={(e) => (e.currentTarget.style.borderColor = 'var(--border)')}
          >
            {DATE_RANGES.map((t) => (
              <option key={t.value} value={t.value}>
                {t.label}
              </option>
            ))}
          </select>

          {dateRange === 'custom' && (
            <>
              <input
                type="date"
                value={fromDate}
                onChange={(e) => {
                  setFromDate(e.target.value);
                  resetCursors();
                }}
                style={{
                  background: 'var(--bg-card)',
                  border: '1px solid var(--border)',
                  borderRadius: 6,
                  padding: '5px 10px',
                  fontSize: 11,
                  color: 'var(--text-primary)',
                  outline: 'none',
                }}
                aria-label="From date"
                onFocus={(e) => (e.currentTarget.style.borderColor = 'var(--accent)')}
                onBlur={(e) => (e.currentTarget.style.borderColor = 'var(--border)')}
              />
              <input
                type="date"
                value={toDate}
                onChange={(e) => {
                  setToDate(e.target.value);
                  resetCursors();
                }}
                style={{
                  background: 'var(--bg-card)',
                  border: '1px solid var(--border)',
                  borderRadius: 6,
                  padding: '5px 10px',
                  fontSize: 11,
                  color: 'var(--text-primary)',
                  outline: 'none',
                }}
                aria-label="To date"
                onFocus={(e) => (e.currentTarget.style.borderColor = 'var(--accent)')}
                onBlur={(e) => (e.currentTarget.style.borderColor = 'var(--border)')}
              />
            </>
          )}

          {hasActiveFilters && (
            <button
              type="button"
              onClick={() => {
                handleSearchChange('');
                handleStatusChange('active');
                handleCategoryChange('all');
                handleDateRangeChange('24h');
              }}
              style={{
                padding: '5px 10px',
                fontSize: 11,
                borderRadius: 6,
                border: '1px solid var(--border)',
                background: 'transparent',
                color: 'var(--text-secondary)',
                cursor: 'pointer',
                display: 'flex',
                alignItems: 'center',
                gap: 4,
              }}
            >
              <svg
                width="10"
                height="10"
                viewBox="0 0 10 10"
                fill="none"
                stroke="currentColor"
                strokeWidth="1.5"
              >
                <path d="M2 2l6 6M8 2l-6 6" />
              </svg>
              Clear filters
            </button>
          )}

          <div style={{ flex: 1 }} />

          {/* View toggle */}
          <div
            style={{
              display: 'flex',
              border: '1px solid var(--border)',
              borderRadius: 6,
              overflow: 'hidden',
            }}
          >
            <button
              type="button"
              aria-label="List view"
              aria-pressed={viewMode === 'list'}
              onClick={() => setViewMode('list')}
              style={{
                padding: '5px 8px',
                background:
                  viewMode === 'list'
                    ? 'color-mix(in srgb, var(--accent) 10%, transparent)'
                    : 'transparent',
                border: 'none',
                cursor: 'pointer',
                color: viewMode === 'list' ? 'var(--text-emphasis)' : 'var(--text-muted)',
                display: 'flex',
                alignItems: 'center',
              }}
            >
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                <path d="M2 3.5h10M2 7h10M2 10.5h10" stroke="currentColor" strokeWidth="1.2" />
              </svg>
            </button>
            <button
              type="button"
              aria-label="Card view"
              aria-pressed={viewMode === 'card'}
              onClick={() => setViewMode('card')}
              style={{
                padding: '5px 8px',
                background:
                  viewMode === 'card'
                    ? 'color-mix(in srgb, var(--accent) 10%, transparent)'
                    : 'transparent',
                border: 'none',
                borderLeft: '1px solid var(--border)',
                cursor: 'pointer',
                color: viewMode === 'card' ? 'var(--text-emphasis)' : 'var(--text-muted)',
                display: 'flex',
                alignItems: 'center',
              }}
            >
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                <rect
                  x="2"
                  y="2"
                  width="4"
                  height="4"
                  rx="1"
                  stroke="currentColor"
                  strokeWidth="1.2"
                />
                <rect
                  x="8"
                  y="2"
                  width="4"
                  height="4"
                  rx="1"
                  stroke="currentColor"
                  strokeWidth="1.2"
                />
                <rect
                  x="2"
                  y="8"
                  width="4"
                  height="4"
                  rx="1"
                  stroke="currentColor"
                  strokeWidth="1.2"
                />
                <rect
                  x="8"
                  y="8"
                  width="4"
                  height="4"
                  rx="1"
                  stroke="currentColor"
                  strokeWidth="1.2"
                />
              </svg>
            </button>
          </div>
        </div>

        {/* Actions Card */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 8,
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 8,
            padding: '10px 12px',
            boxShadow: 'var(--shadow-sm)',
          }}
        >
          {/* Auto-refresh */}
          <select
            aria-label="Auto-refresh interval"
            value={refreshInterval}
            onChange={(e) => setRefreshInterval(e.target.value as RefreshOption)}
            style={{
              background: 'var(--bg-card)',
              border: '1px solid var(--border)',
              borderRadius: 6,
              padding: '5px 10px',
              fontSize: 11.5,
              color: 'var(--text-secondary)',
              outline: 'none',
              cursor: 'pointer',
            }}
            onFocus={(e) => (e.currentTarget.style.borderColor = 'var(--accent)')}
            onBlur={(e) => (e.currentTarget.style.borderColor = 'var(--border)')}
          >
            <option value="10000">10s</option>
            <option value="30000">30s</option>
            <option value="60000">60s</option>
            <option value="off">Off</option>
          </select>

          {/* Manage Rules */}
          <button
            type="button"
            onClick={() => setRulesDialogOpen(true)}
            disabled={!can('alerts', 'manage')}
            title={!can('alerts', 'manage') ? "You don't have permission" : undefined}
            style={{
              padding: '5px 12px',
              fontSize: 11.5,
              fontWeight: 600,
              borderRadius: 6,
              border: 'none',
              background: 'var(--accent)',
              color: 'var(--btn-accent-text, #000)',
              cursor: !can('alerts', 'manage') ? 'not-allowed' : 'pointer',
              display: 'flex',
              alignItems: 'center',
              gap: 6,
              whiteSpace: 'nowrap',
              opacity: !can('alerts', 'manage') ? 0.5 : undefined,
            }}
          >
            <Settings2 style={{ width: 12, height: 12 }} />
            Rules
          </button>
        </div>
      </div>

      {/* Bulk action bar */}
      {selectedIds.size > 0 && (
        <div
          style={{
            position: 'sticky',
            top: 0,
            zIndex: 10,
            display: 'flex',
            alignItems: 'center',
            gap: 10,
            padding: '10px 16px',
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 8,
            boxShadow: 'var(--shadow-sm)',
          }}
        >
          <span
            style={{
              fontSize: 12,
              fontFamily: 'var(--font-mono)',
              color: 'var(--text-secondary)',
            }}
          >
            {selectedIds.size} selected
          </span>
          <div style={{ flex: 1 }} />
          <Button
            variant="outline"
            size="sm"
            onClick={() => handleBulkAction('read')}
            disabled={bulkUpdate.isPending || !can('alerts', 'update')}
            title={!can('alerts', 'update') ? "You don't have permission" : undefined}
            style={{ fontSize: 11, fontFamily: 'var(--font-mono)' }}
          >
            Mark Read
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => handleBulkAction('acknowledged')}
            disabled={bulkUpdate.isPending || !can('alerts', 'update')}
            title={!can('alerts', 'update') ? "You don't have permission" : undefined}
            style={{ fontSize: 11, fontFamily: 'var(--font-mono)' }}
          >
            Acknowledge
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => handleBulkAction('dismissed')}
            disabled={bulkUpdate.isPending || !can('alerts', 'update')}
            title={!can('alerts', 'update') ? "You don't have permission" : undefined}
            style={{ fontSize: 11, fontFamily: 'var(--font-mono)' }}
          >
            Dismiss
          </Button>
        </div>
      )}

      {/* Content */}
      {isLoading ? (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
          {Array.from({ length: 8 }).map((_, i) => (
            <Skeleton key={i} className="h-16 rounded-lg" />
          ))}
        </div>
      ) : sortedAlerts.length === 0 ? (
        <EmptyState
          icon={<Bell style={{ width: 48, height: 48 }} />}
          title={hasActiveFilters ? 'No alerts match your filters' : 'No alerts yet'}
          description={
            hasActiveFilters
              ? 'Try adjusting your severity, status, or category filters.'
              : 'Alerts will appear here when triggered by alert rules.'
          }
          action={
            hasActiveFilters
              ? {
                  label: 'Clear Filters',
                  onClick: () => {
                    setSeverity('all');
                    setStatus('active');
                    setCategory('all');
                    setSearch('');
                    setDateRange('24h');
                    setFromDate('');
                    setToDate('');
                    resetCursors();
                  },
                }
              : undefined
          }
        />
      ) : viewMode === 'card' ? (
        /* ---- Card / Grid View ---- */
        <>
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: 'repeat(auto-fill, minmax(340px, 1fr))',
              gap: 10,
            }}
          >
            {sortedAlerts.map((alert: AlertData) => (
              <AlertCard
                key={alert.id}
                alert={alert}
                selected={selectedIds.has(alert.id)}
                onSelect={handleSelect}
                onStatusChange={handleStatusChangeForAlert}
              />
            ))}
          </div>

          {/* Pagination */}
          {(data?.next_cursor || cursors.length > 0) && (
            <DataTablePagination
              hasNext={!!data?.next_cursor}
              hasPrev={cursors.length > 0}
              onNext={() => {
                if (data?.next_cursor) setCursors((prev) => [...prev, data.next_cursor!]);
              }}
              onPrev={() => setCursors((prev) => prev.slice(0, -1))}
            />
          )}
        </>
      ) : (
        /* ---- Table / List View ---- */
        <>
          {/* Count */}
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <span
              style={{ fontFamily: 'var(--font-mono)', fontSize: 11, color: 'var(--text-muted)' }}
            >
              <span style={{ color: 'var(--text-secondary)' }}>{sortedAlerts.length}</span> alerts
              {data?.next_cursor && ' (page)'}
            </span>
          </div>

          <div
            style={{
              border: '1px solid var(--border)',
              borderRadius: 8,
              overflow: 'hidden',
              background: 'var(--bg-card)',
              boxShadow: 'var(--shadow-sm)',
            }}
          >
            <table
              style={{
                width: '100%',
                borderCollapse: 'collapse',
                tableLayout: 'fixed',
              }}
            >
              <colgroup>
                <col style={{ width: 40 }} />
                <col style={{ width: 32 }} />
                <col style={{ width: 90 }} />
                <col style={{ minWidth: 200 }} />
                <col style={{ width: 100 }} />
                <col style={{ width: 160 }} />
                <col style={{ width: 100 }} />
                <col style={{ width: 80 }} />
                <col style={{ width: 40 }} />
              </colgroup>
              <thead>
                <tr style={{ borderBottom: '1px solid var(--border)' }}>
                  <th style={{ ...TH, width: 40 }}>
                    <CB
                      on={selectedIds.size > 0 && selectedIds.size === sortedAlerts.length}
                      onClick={(e) => {
                        e.stopPropagation();
                        handleSelectAll();
                      }}
                    />
                  </th>
                  <th style={{ ...TH, width: 32 }} />
                  <SortHeader
                    label="Severity"
                    colKey="severity"
                    sortCol={sortCol}
                    sortDir={sortDir}
                    onSort={handleSort}
                  />
                  <SortHeader
                    label="Title"
                    colKey="title"
                    sortCol={sortCol}
                    sortDir={sortDir}
                    onSort={handleSort}
                  />
                  <SortHeader
                    label="Category"
                    colKey="category"
                    sortCol={sortCol}
                    sortDir={sortDir}
                    onSort={handleSort}
                  />
                  <SortHeader
                    label="Resource"
                    colKey="resource"
                    sortCol={sortCol}
                    sortDir={sortDir}
                    onSort={handleSort}
                  />
                  <SortHeader
                    label="Time"
                    colKey="time"
                    sortCol={sortCol}
                    sortDir={sortDir}
                    onSort={handleSort}
                  />
                  <SortHeader
                    label="Status"
                    colKey="status"
                    sortCol={sortCol}
                    sortDir={sortDir}
                    onSort={handleSort}
                  />
                  <th style={{ ...TH, width: 40 }} />
                </tr>
              </thead>
              <tbody>
                {sortedAlerts.map((alert: AlertData) => {
                  const config = getSeverityConfig(alert.severity);
                  const isExpanded = expandedId === alert.id;
                  const isUnread = alert.status === 'unread';
                  const entityUrl = getEntityUrl(alert.resource, alert.resource_id);

                  return (
                    <Fragment key={alert.id}>
                      <tr
                        onClick={() => setExpandedId(isExpanded ? null : alert.id)}
                        style={{
                          cursor: 'pointer',
                          background: selectedIds.has(alert.id) ? 'var(--bg-inset)' : 'transparent',
                          transition: 'background 0.1s',
                        }}
                        onMouseEnter={(e) => {
                          if (!selectedIds.has(alert.id))
                            e.currentTarget.style.background =
                              'var(--bg-card-hover, var(--bg-inset))';
                        }}
                        onMouseLeave={(e) => {
                          if (!selectedIds.has(alert.id))
                            e.currentTarget.style.background = 'transparent';
                        }}
                      >
                        {/* Checkbox */}
                        <td style={TD} onClick={(e) => e.stopPropagation()}>
                          <CB
                            on={selectedIds.has(alert.id)}
                            onClick={(e) => {
                              e.stopPropagation();
                              handleSelect(alert.id);
                            }}
                          />
                        </td>

                        {/* Expand chevron */}
                        <td style={{ ...TD, padding: '12px 4px' }}>
                          {isExpanded ? (
                            <ChevronDown
                              style={{ width: 14, height: 14, color: 'var(--text-faint)' }}
                            />
                          ) : (
                            <ChevronRight
                              style={{ width: 14, height: 14, color: 'var(--text-faint)' }}
                            />
                          )}
                        </td>

                        {/* Severity */}
                        <td style={TD}>
                          <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                            <div
                              style={{
                                width: 26,
                                height: 26,
                                borderRadius: 6,
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'center',
                                background: config.bg,
                                color: config.color,
                                flexShrink: 0,
                              }}
                            >
                              {config.icon}
                            </div>
                            <span
                              style={{
                                fontSize: 11,
                                fontWeight: 600,
                                fontFamily: 'var(--font-mono)',
                                textTransform: 'uppercase',
                                letterSpacing: '0.03em',
                                color: config.color,
                              }}
                            >
                              {config.label}
                            </span>
                          </div>
                        </td>

                        {/* Title */}
                        <td style={TD}>
                          <div
                            style={{
                              display: 'flex',
                              alignItems: 'center',
                              gap: 6,
                            }}
                          >
                            {/* Unread dot */}
                            {isUnread && (
                              <div
                                style={{
                                  width: 6,
                                  height: 6,
                                  borderRadius: '50%',
                                  background: 'var(--accent)',
                                  flexShrink: 0,
                                }}
                              />
                            )}
                            <span
                              style={{
                                fontSize: 13,
                                fontWeight: isUnread ? 600 : 400,
                                color: 'var(--text-emphasis)',
                                overflow: 'hidden',
                                textOverflow: 'ellipsis',
                                whiteSpace: 'nowrap',
                              }}
                            >
                              {alert.title}
                            </span>
                          </div>
                        </td>

                        {/* Category */}
                        <td style={TD}>
                          <span
                            style={{
                              fontSize: 12,
                              color: 'var(--text-secondary)',
                              fontFamily: 'var(--font-sans)',
                            }}
                          >
                            {alert.category}
                          </span>
                        </td>

                        {/* Resource */}
                        <td style={TD} onClick={(e) => e.stopPropagation()}>
                          {entityUrl ? (
                            <button
                              onClick={() => navigate(entityUrl)}
                              style={{
                                display: 'inline-flex',
                                alignItems: 'center',
                                gap: 4,
                                background: 'none',
                                border: 'none',
                                padding: 0,
                                cursor: 'pointer',
                                fontFamily: 'var(--font-mono)',
                                fontSize: 11,
                                color: 'var(--accent)',
                              }}
                            >
                              <ExternalLink style={{ width: 10, height: 10 }} />
                              {alert.resource}/
                              {alert.resource_id.length > 8
                                ? `${alert.resource_id.slice(0, 6)}\u2026`
                                : alert.resource_id}
                            </button>
                          ) : (
                            <span
                              style={{
                                fontFamily: 'var(--font-mono)',
                                fontSize: 11,
                                color: 'var(--text-muted)',
                              }}
                            >
                              {alert.resource}/
                              {alert.resource_id.length > 8
                                ? `${alert.resource_id.slice(0, 6)}\u2026`
                                : alert.resource_id}
                            </span>
                          )}
                        </td>

                        {/* Time */}
                        <td style={TD}>
                          <span
                            style={{
                              fontSize: 11,
                              fontFamily: 'var(--font-mono)',
                              color: 'var(--text-faint)',
                            }}
                          >
                            {formatRelativeTime(alert.created_at)}
                          </span>
                        </td>

                        {/* Status */}
                        <td style={TD}>
                          <span
                            style={{
                              fontSize: 10,
                              fontWeight: 600,
                              fontFamily: 'var(--font-mono)',
                              textTransform: 'uppercase',
                              letterSpacing: '0.04em',
                              color: statusColor(alert.status),
                            }}
                          >
                            {statusLabel(alert.status)}
                          </span>
                        </td>

                        {/* Actions */}
                        <td style={TD} onClick={(e) => e.stopPropagation()}>
                          <RowMenu alert={alert} onStatusChange={handleStatusChangeForAlert} />
                        </td>
                      </tr>

                      {/* Expanded detail */}
                      {isExpanded && (
                        <ExpandedDetail
                          alert={alert}
                          navigate={navigate}
                          onStatusChange={handleStatusChangeForAlert}
                        />
                      )}
                    </Fragment>
                  );
                })}
              </tbody>
            </table>
          </div>

          {/* Pagination */}
          {(data?.next_cursor || cursors.length > 0) && (
            <DataTablePagination
              hasNext={!!data?.next_cursor}
              hasPrev={cursors.length > 0}
              onNext={() => {
                if (data?.next_cursor) setCursors((prev) => [...prev, data.next_cursor!]);
              }}
              onPrev={() => setCursors((prev) => prev.slice(0, -1))}
            />
          )}
        </>
      )}

      {/* Alert Rules Dialog */}
      <AlertRulesDialog open={rulesDialogOpen} onOpenChange={setRulesDialogOpen} />
    </div>
  );
};
