import { useState, useMemo, Fragment } from 'react';
import { Copy, Check, ChevronDown, ChevronRight } from 'lucide-react';
import {
  getEventCategory,
  getCategoryColor,
  getActorInitials,
  getActorGradient,
  formatEventDescription,
  groupEventsByDate,
} from '../../lib/audit-utils';
import type { components } from '../../api/types';

function CopyableId({ id }: { id: string }) {
  const [copied, setCopied] = useState(false);
  const handleCopy = (e: React.MouseEvent) => {
    e.stopPropagation();
    void navigator.clipboard.writeText(id).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    });
  };
  return (
    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
      <span
        style={{
          fontFamily: 'var(--font-mono)',
          fontSize: 11,
          color: 'var(--accent)',
          cursor: 'text',
          userSelect: 'all',
        }}
      >
        {id}
      </span>
      <button
        onClick={handleCopy}
        title="Copy ID"
        style={{
          background: 'none',
          border: 'none',
          cursor: 'pointer',
          padding: '2px 3px',
          borderRadius: 4,
          color: 'var(--text-muted)',
          display: 'flex',
          alignItems: 'center',
        }}
      >
        {copied ? (
          <Check style={{ width: 11, height: 11, color: 'var(--accent)' }} />
        ) : (
          <Copy style={{ width: 11, height: 11 }} />
        )}
      </button>
    </span>
  );
}

type AuditEvent = components['schemas']['AuditEvent'];

interface ActivityStreamProps {
  events: AuditEvent[];
  expandedId: string | null;
  onToggleExpand: (id: string) => void;
}

function getCategoryLabel(category: string): string {
  return category;
}

function EventRow({
  event,
  index,
  isExpanded,
  onToggleExpand,
}: {
  event: AuditEvent;
  index: number;
  isExpanded: boolean;
  onToggleExpand: (id: string) => void;
}) {
  const id = event.id ?? '';
  const category = getEventCategory(event.type ?? '');
  const borderColor = getCategoryColor(category);
  const initials = getActorInitials(event.actor_id ?? '', event.type ?? '');
  const gradient =
    event.actor_id === 'system'
      ? `linear-gradient(135deg, ${borderColor}, ${borderColor}cc)`
      : getActorGradient(event.actor_id ?? '');
  const { title, subtitle } = formatEventDescription(event);

  // Precise timestamp
  const timestamp = event.timestamp
    ? (() => {
        const d = new Date(event.timestamp);
        const h = d.getHours().toString().padStart(2, '0');
        const m = d.getMinutes().toString().padStart(2, '0');
        const s = d.getSeconds().toString().padStart(2, '0');
        const ms = d.getMilliseconds().toString().padStart(3, '0');
        return `${h}:${m}:${s}.${ms}`;
      })()
    : '—';

  // Category color mapping for text
  const categoryTextColor =
    category === 'Deployment'
      ? 'var(--accent)'
      : category === 'Compliance'
        ? 'var(--accent)'
        : category === 'Endpoint' || category === 'Auth'
          ? 'var(--text-secondary)'
          : category === 'Patch' || category === 'System'
            ? 'var(--signal-critical)'
            : category === 'Policy'
              ? 'var(--signal-warning)'
              : 'var(--text-muted)';

  return (
    <tr
      key={id ? `${id}-${index}` : index}
      style={{ cursor: 'pointer' }}
      onClick={() => onToggleExpand(id)}
      onMouseEnter={(e) =>
        ((e.currentTarget as HTMLTableRowElement).style.background = 'var(--bg-card-hover)')
      }
      onMouseLeave={(e) =>
        ((e.currentTarget as HTMLTableRowElement).style.background = isExpanded
          ? 'var(--bg-inset)'
          : '')
      }
    >
      {/* Expand chevron */}
      <td
        style={{
          padding: '9px 8px 9px 14px',
          borderBottom: '1px solid var(--border)',
          width: 28,
          background: isExpanded ? 'var(--bg-inset)' : undefined,
        }}
      >
        {isExpanded ? (
          <ChevronDown
            style={{ width: 12, height: 12, color: 'var(--text-muted)', flexShrink: 0 }}
          />
        ) : (
          <ChevronRight
            style={{ width: 12, height: 12, color: 'var(--text-muted)', flexShrink: 0 }}
          />
        )}
      </td>

      {/* Timestamp */}
      <td
        style={{
          padding: '9px 12px',
          borderBottom: '1px solid var(--border)',
          whiteSpace: 'nowrap',
          background: isExpanded ? 'var(--bg-inset)' : undefined,
        }}
      >
        <span
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 11,
            color: 'var(--text-faint)',
          }}
        >
          {timestamp}
        </span>
      </td>

      {/* Category */}
      <td
        style={{
          padding: '9px 12px',
          borderBottom: '1px solid var(--border)',
          background: isExpanded ? 'var(--bg-inset)' : undefined,
        }}
      >
        <span
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 10,
            fontWeight: 600,
            color: categoryTextColor,
            textTransform: 'uppercase',
            letterSpacing: '0.05em',
          }}
        >
          {getCategoryLabel(category)}
        </span>
      </td>

      {/* Event / actor avatar + description */}
      <td
        style={{
          padding: '9px 12px',
          borderBottom: '1px solid var(--border)',
          background: isExpanded ? 'var(--bg-inset)' : undefined,
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          {/* Avatar */}
          <div
            style={{
              width: 26,
              height: 26,
              borderRadius: '50%',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontFamily: 'var(--font-mono)',
              fontSize: 9,
              fontWeight: 700,
              color: 'var(--text-on-color, #fff)',
              background: gradient,
              flexShrink: 0,
            }}
          >
            {initials}
          </div>
          <div>
            <div
              style={{
                fontFamily: 'var(--font-sans)',
                fontSize: 12,
                color: 'var(--text-primary)',
                lineHeight: 1.4,
              }}
            >
              {title}
            </div>
            {subtitle && (
              <div
                style={{
                  fontFamily: 'var(--font-sans)',
                  fontSize: 11,
                  color: 'var(--text-muted)',
                  marginTop: 1,
                }}
              >
                {subtitle}
              </div>
            )}
          </div>
        </div>
      </td>

      {/* Actor */}
      <td
        style={{
          padding: '9px 12px',
          borderBottom: '1px solid var(--border)',
          background: isExpanded ? 'var(--bg-inset)' : undefined,
        }}
      >
        <span
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 11,
            color: 'var(--text-muted)',
          }}
        >
          {event.actor_id === 'system' ? 'System' : (event.actor_id ?? '—')}
        </span>
      </td>

      {/* Target */}
      <td
        style={{
          padding: '9px 12px',
          borderBottom: '1px solid var(--border)',
          background: isExpanded ? 'var(--bg-inset)' : undefined,
        }}
      >
        {event.resource_id ? (
          <span
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 11,
              color: 'var(--text-secondary)',
            }}
          >
            {event.resource_id.length > 16
              ? `${event.resource_id.slice(0, 8)}…`
              : event.resource_id}
          </span>
        ) : (
          <span
            style={{ color: 'var(--text-faint)', fontFamily: 'var(--font-mono)', fontSize: 11 }}
          >
            —
          </span>
        )}
      </td>
    </tr>
  );
}

// ─── Payload formatting helpers ───────────────────────────────────────────────

const ACRONYMS = new Set([
  'ID',
  'URL',
  'OS',
  'CVE',
  'CVSS',
  'EPSS',
  'IP',
  'MAC',
  'UUID',
  'API',
  'JSON',
  'YAML',
  'TLS',
  'SSH',
  'RBAC',
  'IAM',
]);

function humanizeKey(key: string): string {
  return key
    .split('_')
    .map((part) => {
      const upper = part.toUpperCase();
      if (ACRONYMS.has(upper)) return upper;
      return part.length === 0 ? part : part[0].toUpperCase() + part.slice(1);
    })
    .join(' ');
}

const ISO_RE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:?\d{2})?$/;

function formatPrimitive(value: unknown): string {
  if (value === null || value === undefined) return '';
  if (typeof value === 'string') {
    if (ISO_RE.test(value)) {
      const d = new Date(value);
      if (!isNaN(d.getTime())) return d.toLocaleString();
    }
    return value;
  }
  if (typeof value === 'number' || typeof value === 'boolean') return String(value);
  return JSON.stringify(value);
}

function isEmpty(value: unknown): boolean {
  if (value === null || value === undefined) return true;
  if (typeof value === 'string' && value === '') return true;
  if (Array.isArray(value) && value.length === 0) return true;
  if (typeof value === 'object' && Object.keys(value as object).length === 0) return true;
  return false;
}

function FieldRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div style={{ display: 'flex', gap: 12, alignItems: 'flex-start', padding: '3px 0' }}>
      <div
        style={{
          fontFamily: 'var(--font-mono)',
          fontSize: 11,
          color: 'var(--text-muted)',
          minWidth: 140,
          flexShrink: 0,
        }}
      >
        {label}
      </div>
      <div
        style={{
          fontFamily: 'var(--font-sans)',
          fontSize: 12,
          color: 'var(--text-primary)',
          wordBreak: 'break-word',
          flex: 1,
        }}
      >
        {children}
      </div>
    </div>
  );
}

function renderValue(value: unknown, depth: number): React.ReactNode {
  if (Array.isArray(value)) {
    const allPrimitive = value.every(
      (v) => v === null || ['string', 'number', 'boolean'].includes(typeof v),
    );
    if (allPrimitive) {
      return value
        .map((v) => formatPrimitive(v))
        .filter(Boolean)
        .join(', ');
    }
    if (depth >= 1) {
      return (
        <pre
          style={{
            margin: 0,
            fontFamily: 'var(--font-mono)',
            fontSize: 11,
            whiteSpace: 'pre-wrap',
          }}
        >
          {JSON.stringify(value, null, 2)}
        </pre>
      );
    }
    return (
      <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
        {value.map((item, i) => (
          <div key={i} style={{ paddingLeft: 8, borderLeft: '2px solid var(--border)' }}>
            {renderObject(item as Record<string, unknown>, depth + 1)}
          </div>
        ))}
      </div>
    );
  }
  if (value !== null && typeof value === 'object') {
    if (depth >= 1) {
      return (
        <pre
          style={{
            margin: 0,
            fontFamily: 'var(--font-mono)',
            fontSize: 11,
            whiteSpace: 'pre-wrap',
          }}
        >
          {JSON.stringify(value, null, 2)}
        </pre>
      );
    }
    return (
      <div style={{ paddingLeft: 8, borderLeft: '2px solid var(--border)' }}>
        {renderObject(value as Record<string, unknown>, depth + 1)}
      </div>
    );
  }
  return formatPrimitive(value);
}

function renderObject(obj: Record<string, unknown>, depth: number): React.ReactNode {
  if (!obj || typeof obj !== 'object') return null;
  const entries = Object.entries(obj).filter(([, v]) => !isEmpty(v));
  if (entries.length === 0) return null;
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      {entries.map(([key, value]) => (
        <FieldRow key={key} label={humanizeKey(key)}>
          {renderValue(value, depth)}
        </FieldRow>
      ))}
    </div>
  );
}

function ExpandedPayloadRow({ event }: { event: AuditEvent }) {
  const payload = (event.payload as Record<string, unknown>) ?? null;
  const metadata = (event.metadata as Record<string, unknown>) ?? null;

  const payloadEntries =
    payload && typeof payload === 'object'
      ? Object.entries(payload).filter(([, v]) => !isEmpty(v))
      : [];
  const metadataEntries =
    metadata && typeof metadata === 'object'
      ? Object.entries(metadata).filter(([, v]) => !isEmpty(v))
      : [];

  const hasContent = payloadEntries.length > 0 || metadataEntries.length > 0;

  return (
    <tr>
      <td
        colSpan={6}
        style={{
          background: 'color-mix(in srgb, var(--accent) 4%, var(--bg-inset))',
          borderLeft: 'none',
          borderBottom: '1px solid var(--border)',
          padding: '0 16px 16px 54px',
        }}
      >
        <div
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 10,
            fontWeight: 600,
            textTransform: 'uppercase',
            letterSpacing: '0.05em',
            color: 'var(--text-muted)',
            marginBottom: 8,
            paddingTop: 12,
          }}
        >
          Event Payload
        </div>

        {hasContent ? (
          <div
            style={{
              background: 'var(--bg-page)',
              border: '1px solid var(--border)',
              borderRadius: 6,
              padding: '10px 14px',
            }}
          >
            {payloadEntries.length > 0 && renderObject(Object.fromEntries(payloadEntries), 0)}

            {metadataEntries.length > 0 && (
              <>
                <div
                  style={{
                    fontFamily: 'var(--font-mono)',
                    fontSize: 10,
                    fontWeight: 600,
                    textTransform: 'uppercase',
                    letterSpacing: '0.05em',
                    color: 'var(--text-muted)',
                    marginTop: 12,
                    marginBottom: 6,
                  }}
                >
                  Metadata
                </div>
                {renderObject(Object.fromEntries(metadataEntries), 0)}
              </>
            )}
          </div>
        ) : (
          <div
            style={{
              fontFamily: 'var(--font-sans)',
              fontSize: 12,
              color: 'var(--text-muted)',
              fontStyle: 'italic',
            }}
          >
            No additional details
          </div>
        )}

        <details style={{ marginTop: 10 }}>
          <summary
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 10,
              color: 'var(--text-muted)',
              cursor: 'pointer',
              userSelect: 'none',
            }}
          >
            Show raw JSON
          </summary>
          <pre
            style={{
              background: 'var(--bg-page)',
              border: '1px solid var(--border)',
              borderRadius: 6,
              padding: '10px 14px',
              fontFamily: 'var(--font-mono)',
              fontSize: 11,
              color: 'var(--text-primary)',
              overflowX: 'auto',
              maxHeight: 220,
              margin: '8px 0 0 0',
            }}
          >
            {JSON.stringify(event.payload, null, 2)}
          </pre>
        </details>

        <div
          style={{
            display: 'flex',
            flexWrap: 'wrap',
            gap: 16,
            marginTop: 10,
            fontFamily: 'var(--font-mono)',
            fontSize: 10,
            color: 'var(--text-muted)',
          }}
        >
          {event.type && (
            <span>
              Type: <span style={{ color: 'var(--text-secondary)' }}>{event.type}</span>
            </span>
          )}
          {event.resource && (
            <span>
              Resource: <span style={{ color: 'var(--text-secondary)' }}>{event.resource}</span>
            </span>
          )}
          {event.resource_id && (
            <span style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
              ID: <CopyableId id={event.resource_id} />
            </span>
          )}
        </div>
      </td>
    </tr>
  );
}

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

// ─── Sort Header ──────────────────────────────────────────────────────────────

type SortDir = 'asc' | 'desc';

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
  sortDir: SortDir;
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

// ─── Sort helpers ─────────────────────────────────────────────────────────────

function getSortValue(event: AuditEvent, col: string): string {
  switch (col) {
    case 'timestamp':
      return event.timestamp ?? '';
    case 'category':
      return getEventCategory(event.type ?? '');
    case 'event':
      return event.type ?? '';
    case 'actor':
      return event.actor_id ?? '';
    case 'target':
      return event.resource_id ?? '';
    default:
      return '';
  }
}

function sortEvents(events: AuditEvent[], col: string | null, dir: SortDir): AuditEvent[] {
  if (!col) return events;
  const sorted = [...events].sort((a, b) => {
    const va = getSortValue(a, col);
    const vb = getSortValue(b, col);
    return va.localeCompare(vb);
  });
  return dir === 'desc' ? sorted.reverse() : sorted;
}

// ─── Activity Stream ──────────────────────────────────────────────────────────

export function ActivityStream({ events, expandedId, onToggleExpand }: ActivityStreamProps) {
  const [sortCol, setSortCol] = useState<string | null>(null);
  const [sortDir, setSortDir] = useState<SortDir>('asc');

  const handleSort = (col: string) => {
    if (sortCol === col) {
      setSortDir((prev) => (prev === 'asc' ? 'desc' : 'asc'));
    } else {
      setSortCol(col);
      setSortDir('asc');
    }
  };

  const sortedEvents = useMemo(
    () => sortEvents(events, sortCol, sortDir),
    [events, sortCol, sortDir],
  );

  // When sorting is active, render a flat list (no date grouping)
  const isSorted = sortCol !== null;
  const grouped = useMemo(
    () => (isSorted ? null : groupEventsByDate(sortedEvents)),
    [isSorted, sortedEvents],
  );

  if (events.length === 0) {
    return (
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          height: 240,
          fontFamily: 'var(--font-sans)',
          fontSize: 13,
          color: 'var(--text-muted)',
        }}
      >
        No audit events found.
      </div>
    );
  }

  return (
    <div
      style={{
        background: 'var(--bg-card)',
        border: '1px solid var(--border)',
        borderRadius: 8,
        boxShadow: 'var(--shadow-sm)',
        overflow: 'hidden',
      }}
    >
      <div style={{ overflowX: 'auto' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
          <thead>
            <tr style={{ background: 'var(--bg-inset)', borderBottom: '1px solid var(--border)' }}>
              <th style={{ ...TH, width: 28, padding: '8px 8px 8px 14px' }} />
              <SortHeader
                label="Timestamp"
                colKey="timestamp"
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
                label="Event"
                colKey="event"
                sortCol={sortCol}
                sortDir={sortDir}
                onSort={handleSort}
              />
              <SortHeader
                label="Actor"
                colKey="actor"
                sortCol={sortCol}
                sortDir={sortDir}
                onSort={handleSort}
              />
              <SortHeader
                label="Target"
                colKey="target"
                sortCol={sortCol}
                sortDir={sortDir}
                onSort={handleSort}
              />
            </tr>
          </thead>
          <tbody>
            {isSorted
              ? /* Flat sorted list — no date grouping */
                sortedEvents.map((event, index) => {
                  const id = event.id ?? '';
                  const isExpanded = expandedId === id;
                  return (
                    <Fragment key={id ? `${id}-${index}` : `evt-${index}`}>
                      <EventRow
                        event={event}
                        index={index}
                        isExpanded={isExpanded}
                        onToggleExpand={onToggleExpand}
                      />
                      {isExpanded && <ExpandedPayloadRow event={event} />}
                    </Fragment>
                  );
                })
              : /* Default: grouped by date */
                Array.from(grouped!.entries()).map(([dateLabel, dateEvents]) => (
                  <Fragment key={`date-group-${dateLabel}`}>
                    {/* Date separator row */}
                    <tr>
                      <td
                        colSpan={6}
                        style={{
                          padding: '6px 14px',
                          background: 'var(--bg-page)',
                          borderBottom: '1px solid var(--border)',
                          borderTop: '1px solid var(--border)',
                        }}
                      >
                        <span
                          style={{
                            fontFamily: 'var(--font-mono)',
                            fontSize: 10,
                            fontWeight: 600,
                            color: 'var(--text-muted)',
                            textTransform: 'uppercase',
                            letterSpacing: '0.06em',
                          }}
                        >
                          {dateLabel}
                        </span>
                      </td>
                    </tr>

                    {dateEvents.map((event, index) => {
                      const id = event.id ?? '';
                      const isExpanded = expandedId === id;
                      return (
                        <Fragment key={id ? `${id}-${index}` : `evt-${index}`}>
                          <EventRow
                            event={event}
                            index={index}
                            isExpanded={isExpanded}
                            onToggleExpand={onToggleExpand}
                          />
                          {isExpanded && <ExpandedPayloadRow event={event} />}
                        </Fragment>
                      );
                    })}
                  </Fragment>
                ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
