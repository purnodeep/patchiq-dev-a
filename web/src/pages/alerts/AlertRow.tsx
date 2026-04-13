import { useState } from 'react';
import { useNavigate } from 'react-router';
import {
  AlertTriangle,
  CheckCircle2,
  MoreHorizontal,
  ChevronDown,
  ChevronRight,
  ExternalLink,
  Clock,
  Eye,
  CheckCheck,
  XCircle,
} from 'lucide-react';

export interface AlertRowProps {
  alert: {
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
  };
  selected: boolean;
  onSelect: (id: string) => void;
  onStatusChange: (id: string, status: string) => void;
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
  if (!dateStr) return '—';
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
        bg: 'color-mix(in srgb, var(--signal-critical) 1%, transparent)',
        color: 'var(--signal-critical)',
        icon: <AlertTriangle style={{ width: 15, height: 15 }} />,
      };
    case 'warning':
      return {
        bg: 'color-mix(in srgb, var(--signal-warning) 12%, transparent)',
        color: 'var(--signal-warning)',
        icon: <AlertTriangle style={{ width: 15, height: 15 }} />,
      };
    case 'info':
    default:
      return {
        bg: 'color-mix(in srgb, var(--signal-healthy) 12%, transparent)',
        color: 'var(--signal-healthy)',
        icon: <CheckCircle2 style={{ width: 15, height: 15 }} />,
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

export function AlertRow({ alert, selected, onSelect, onStatusChange }: AlertRowProps) {
  const [menuOpen, setMenuOpen] = useState(false);
  const [expanded, setExpanded] = useState(false);
  const [hovered, setHovered] = useState(false);
  const navigate = useNavigate();
  const config = getSeverityConfig(alert.severity);
  const isUnread = alert.status === 'unread';
  const entityUrl = getEntityUrl(alert.resource, alert.resource_id);

  function handleRowClick(e: React.MouseEvent) {
    const target = e.target as HTMLElement;
    if (target.closest('[data-no-navigate]')) return;
    setExpanded((v) => !v);
  }

  function handleMenuAction(e: React.MouseEvent, newStatus: string) {
    e.stopPropagation();
    setMenuOpen(false);
    onStatusChange(alert.id, newStatus);
  }

  const payloadEntries =
    alert.payload && typeof alert.payload === 'object'
      ? Object.entries(alert.payload).filter(([, v]) => v !== null && v !== undefined && v !== '')
      : [];

  return (
    <div
      style={{
        borderBottom: '1px solid var(--border-faint, var(--border))',
        background: selected ? 'var(--bg-inset)' : 'var(--bg-card)',
      }}
    >
      {/* Main row */}
      <div
        style={{
          position: 'relative',
          display: 'flex',
          alignItems: 'flex-start',
          gap: 10,
          padding: '11px 14px',
          background: hovered ? 'var(--bg-card-hover)' : 'transparent',
          cursor: 'pointer',
          transition: 'background 0.1s',
        }}
        onClick={handleRowClick}
        onMouseEnter={() => setHovered(true)}
        onMouseLeave={() => setHovered(false)}
      >
        {/* Checkbox */}
        <div
          data-no-navigate
          style={{ paddingTop: 2, flexShrink: 0 }}
          onClick={(e) => e.stopPropagation()}
        >
          <input
            type="checkbox"
            checked={selected}
            onChange={() => onSelect(alert.id)}
            style={{ cursor: 'pointer', accentColor: 'var(--accent)', width: 14, height: 14 }}
          />
        </div>

        {/* Expand chevron */}
        <div style={{ flexShrink: 0, marginTop: 3, color: 'var(--text-faint)' }}>
          {expanded ? (
            <ChevronDown style={{ width: 14, height: 14 }} />
          ) : (
            <ChevronRight style={{ width: 14, height: 14 }} />
          )}
        </div>

        {/* Unread indicator */}
        <div
          style={{
            flexShrink: 0,
            width: 6,
            height: 6,
            borderRadius: '50%',
            marginTop: 6,
            background: isUnread ? 'var(--signal-healthy, #22c55e)' : 'transparent',
          }}
        />

        {/* Severity icon block */}
        <div
          style={{
            flexShrink: 0,
            width: 30,
            height: 30,
            borderRadius: 7,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            background: config.bg,
            color: config.color,
            marginTop: 1,
          }}
        >
          {config.icon}
        </div>

        {/* Content */}
        <div style={{ flex: 1, minWidth: 0 }}>
          <div
            style={{
              fontSize: 13,
              fontWeight: isUnread ? 600 : 500,
              color: 'var(--text-emphasis)',
              lineHeight: 1.4,
              marginBottom: 2,
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
            }}
          >
            {alert.title}
          </div>
          <div
            style={{
              fontSize: 12,
              color: 'var(--text-secondary)',
              lineHeight: 1.4,
              marginBottom: 6,
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
            }}
          >
            {alert.description}
          </div>

          {/* Meta row */}
          <div style={{ display: 'flex', alignItems: 'center', flexWrap: 'wrap', gap: 8 }}>
            <span
              style={{
                fontSize: 10,
                fontWeight: 600,
                fontFamily: 'var(--font-mono)',
                textTransform: 'uppercase',
                letterSpacing: '0.04em',
                background: config.bg,
                color: config.color,
                padding: '1px 7px',
                borderRadius: 'var(--radius-full, 9999px)',
              }}
            >
              {alert.severity}
            </span>
            <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>{alert.category}</span>
            <span style={{ color: 'var(--border)', fontSize: 11 }}>·</span>
            {entityUrl ? (
              <button
                data-no-navigate
                onClick={(e) => {
                  e.stopPropagation();
                  navigate(entityUrl);
                }}
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
                  textDecoration: 'none',
                }}
              >
                <ExternalLink style={{ width: 10, height: 10 }} />
                {alert.resource}/
                {alert.resource_id.length > 12
                  ? `${alert.resource_id.slice(0, 8)}…`
                  : alert.resource_id}
              </button>
            ) : (
              <span
                style={{ fontFamily: 'var(--font-mono)', fontSize: 11, color: 'var(--text-muted)' }}
              >
                {alert.resource}/
                {alert.resource_id.length > 12
                  ? `${alert.resource_id.slice(0, 8)}…`
                  : alert.resource_id}
              </span>
            )}
            <span style={{ color: 'var(--border)', fontSize: 11 }}>·</span>
            <span
              style={{ fontSize: 11, color: 'var(--text-faint)', fontFamily: 'var(--font-mono)' }}
            >
              {formatRelativeTime(alert.created_at)}
            </span>
          </div>
        </div>

        {/* Actions */}
        <div
          data-no-navigate
          style={{ flexShrink: 0, position: 'relative' }}
          onClick={(e) => e.stopPropagation()}
        >
          <button
            onClick={(e) => {
              e.stopPropagation();
              setMenuOpen((v) => !v);
            }}
            style={{
              background: 'none',
              border: '1px solid var(--border)',
              borderRadius: 6,
              cursor: 'pointer',
              color: 'var(--text-muted)',
              display: 'flex',
              alignItems: 'center',
              padding: '4px 6px',
            }}
          >
            <MoreHorizontal style={{ width: 14, height: 14 }} />
          </button>

          {menuOpen && (
            <div
              style={{
                position: 'absolute',
                right: 0,
                top: '100%',
                marginTop: 4,
                background: 'var(--bg-elevated)',
                border: '1px solid var(--border)',
                borderRadius: 8,
                boxShadow: 'var(--shadow-md, 0 4px 16px rgba(0,0,0,0.18))',
                zIndex: 50,
                minWidth: 148,
                overflow: 'hidden',
              }}
            >
              {alert.status !== 'read' && (
                <MenuAction label="Mark Read" onClick={(e) => handleMenuAction(e, 'read')} />
              )}
              {alert.status !== 'acknowledged' && (
                <MenuAction
                  label="Acknowledge"
                  onClick={(e) => handleMenuAction(e, 'acknowledged')}
                />
              )}
              {alert.status !== 'dismissed' && (
                <MenuAction
                  label="Dismiss"
                  onClick={(e) => handleMenuAction(e, 'dismissed')}
                  danger
                />
              )}
            </div>
          )}
        </div>
      </div>

      {/* Expanded detail panel */}
      {expanded &&
        (() => {
          const CARD: React.CSSProperties = {
            background: 'var(--bg-inset)',
            border: '1px solid var(--border)',
            borderRadius: 6,
            padding: '12px 14px',
          };
          const LBL: React.CSSProperties = {
            fontFamily: 'var(--font-mono)',
            fontSize: 9,
            fontWeight: 600,
            textTransform: 'uppercase',
            letterSpacing: '0.07em',
            color: 'var(--text-muted)',
            marginBottom: 10,
          };
          const BTN: React.CSSProperties = {
            display: 'inline-flex',
            alignItems: 'center',
            justifyContent: 'center',
            gap: 5,
            padding: '7px 14px',
            borderRadius: 5,
            fontSize: 13,
            fontWeight: 500,
            cursor: 'pointer',
            border: '1px solid var(--border)',
            background: 'transparent',
            color: 'var(--text-secondary)',
            letterSpacing: '0.01em',
            width: '100%',
            fontFamily: 'var(--font-sans)',
          };
          return (
            <div
              style={{
                padding: '8px 10px',
                background: 'var(--bg-page)',
                borderTop: '1px solid var(--border)',
                display: 'flex',
                gap: 8,
                alignItems: 'stretch',
              }}
              data-no-navigate
            >
              {/* Details card */}
              <div style={{ ...CARD, flex: '0 0 500px' }}>
                <div style={LBL}>Details</div>
                <p
                  style={{
                    fontSize: 13,
                    color: 'var(--text-secondary)',
                    lineHeight: 1.6,
                    margin: '0 0 12px',
                  }}
                >
                  {alert.description || '—'}
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
              <div style={{ ...CARD, flex: '0 0 480px', marginLeft: 24 }}>
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
                  width: 160,
                  alignSelf: 'start',
                  marginLeft: 40,
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
                    onClick={(e) => handleMenuAction(e, 'acknowledged')}
                  >
                    <CheckCheck style={{ width: 12, height: 12 }} />
                    Acknowledge
                  </button>
                )}
                {alert.status !== 'dismissed' && (
                  <button
                    type="button"
                    style={BTN}
                    onClick={(e) => handleMenuAction(e, 'dismissed')}
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
          );
        })()}
    </div>
  );
}

/* ---- Sub-components ---- */

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
          color: value === '—' ? 'var(--text-faint)' : 'var(--text-secondary)',
        }}
      >
        {value}
      </span>
    </div>
  );
}

function MenuAction({
  label,
  onClick,
  danger,
}: {
  label: string;
  onClick: (e: React.MouseEvent) => void;
  danger?: boolean;
}) {
  const [hov, setHov] = useState(false);
  return (
    <button
      onClick={onClick}
      onMouseEnter={() => setHov(true)}
      onMouseLeave={() => setHov(false)}
      style={{
        display: 'block',
        width: '100%',
        textAlign: 'left',
        padding: '7px 12px',
        fontSize: 12,
        fontFamily: 'var(--font-sans)',
        color: danger
          ? hov
            ? 'var(--signal-critical)'
            : 'var(--text-secondary)'
          : hov
            ? 'var(--text-emphasis)'
            : 'var(--text-primary)',
        background: hov ? 'var(--bg-card-hover)' : 'transparent',
        border: 'none',
        cursor: 'pointer',
        transition: 'background 0.1s, color 0.1s',
      }}
    >
      {label}
    </button>
  );
}
