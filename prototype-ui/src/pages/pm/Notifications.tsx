import { motion } from 'framer-motion';
import { Bell, Send } from 'lucide-react';
import { useHotkeys } from '@/hooks/useHotkeys';
import { GlassCard } from '@/components/shared/GlassCard';
import { SectionHeader } from '@/components/shared/SectionHeader';

// ── Framer Motion variants ────────────────────────────────────────────────────
const stagger = {
  hidden: {},
  show: { transition: { staggerChildren: 0.06 } },
};

const fadeUp = {
  hidden: { opacity: 0, y: 12 },
  show: { opacity: 1, y: 0, transition: { duration: 0.4, ease: 'easeOut' } },
};

// ── Types ─────────────────────────────────────────────────────────────────────
interface NotifRow {
  event: string;
  email: boolean;
  slack: boolean;
  webhook: boolean;
  urgency: 'Immediate' | 'Digest';
}

interface NotifCategory {
  title: string;
  borderColor: string;
  rows: NotifRow[];
}

// ── Notification data ─────────────────────────────────────────────────────────
const CATEGORIES: NotifCategory[] = [
  {
    title: 'Deployments',
    borderColor: 'var(--color-cyan)',
    rows: [
      { event: 'Deployment Started', email: true, slack: true, webhook: false, urgency: 'Digest' },
      { event: 'Deployment Completed', email: true, slack: true, webhook: true, urgency: 'Digest' },
      { event: 'Deployment Failed', email: true, slack: true, webhook: true, urgency: 'Immediate' },
      {
        event: 'Rollback Initiated',
        email: true,
        slack: true,
        webhook: true,
        urgency: 'Immediate',
      },
    ],
  },
  {
    title: 'Compliance',
    borderColor: 'var(--color-success)',
    rows: [
      {
        event: 'Framework Evaluation Complete',
        email: true,
        slack: true,
        webhook: false,
        urgency: 'Digest',
      },
      { event: 'Control Failed', email: true, slack: true, webhook: true, urgency: 'Immediate' },
      {
        event: 'SLA Approaching (72h)',
        email: true,
        slack: true,
        webhook: false,
        urgency: 'Immediate',
      },
      { event: 'SLA Overdue', email: true, slack: true, webhook: true, urgency: 'Immediate' },
    ],
  },
  {
    title: 'Security',
    borderColor: 'var(--color-danger)',
    rows: [
      {
        event: 'Critical CVE Published (CVSS ≥ 9.0)',
        email: true,
        slack: true,
        webhook: true,
        urgency: 'Immediate',
      },
      {
        event: 'Exploit Detected in Wild',
        email: true,
        slack: true,
        webhook: true,
        urgency: 'Immediate',
      },
      { event: 'KEV Entry Added', email: true, slack: true, webhook: true, urgency: 'Immediate' },
      {
        event: 'Patch Available for CVE',
        email: true,
        slack: false,
        webhook: true,
        urgency: 'Digest',
      },
    ],
  },
  {
    title: 'System',
    borderColor: 'var(--color-muted)',
    rows: [
      {
        event: 'Agent Offline (> 30 min)',
        email: true,
        slack: true,
        webhook: false,
        urgency: 'Immediate',
      },
      {
        event: 'License Expiring (30-day)',
        email: true,
        slack: false,
        webhook: false,
        urgency: 'Immediate',
      },
      { event: 'Hub Sync Failed', email: true, slack: true, webhook: true, urgency: 'Immediate' },
      { event: 'Scan Completed', email: false, slack: true, webhook: false, urgency: 'Digest' },
    ],
  },
];

// ── History data ──────────────────────────────────────────────────────────────
type HistoryStatus = 'Delivered' | 'Failed' | 'Pending';

interface HistoryRow {
  event: string;
  channel: string;
  recipient: string;
  status: HistoryStatus;
  sentAt: string;
}

const HISTORY_ROWS: HistoryRow[] = [
  {
    event: 'Deployment Failed',
    channel: 'Email',
    recipient: 'ops@acme.com',
    status: 'Delivered',
    sentAt: '2026-03-11 14:32',
  },
  {
    event: 'Critical CVE Published',
    channel: 'Slack',
    recipient: '#patchiq-alerts',
    status: 'Delivered',
    sentAt: '2026-03-11 13:15',
  },
  {
    event: 'SLA Overdue',
    channel: 'Webhook',
    recipient: 'hooks.acme.com',
    status: 'Delivered',
    sentAt: '2026-03-11 11:00',
  },
  {
    event: 'Hub Sync Failed',
    channel: 'Email',
    recipient: 'admin@acme.com',
    status: 'Failed',
    sentAt: '2026-03-11 09:44',
  },
  {
    event: 'Control Failed',
    channel: 'Slack',
    recipient: '#patchiq-alerts',
    status: 'Delivered',
    sentAt: '2026-03-10 22:01',
  },
  {
    event: 'KEV Entry Added',
    channel: 'Webhook',
    recipient: 'hooks.acme.com',
    status: 'Pending',
    sentAt: '2026-03-10 20:17',
  },
  {
    event: 'Rollback Initiated',
    channel: 'Email',
    recipient: 'ops@acme.com',
    status: 'Delivered',
    sentAt: '2026-03-10 18:55',
  },
  {
    event: 'Agent Offline (> 30 min)',
    channel: 'Slack',
    recipient: '#patchiq-alerts',
    status: 'Delivered',
    sentAt: '2026-03-10 16:30',
  },
];

// ── Sub-components ────────────────────────────────────────────────────────────
function Toggle({ on }: { on: boolean }) {
  return (
    <div
      style={{
        width: 28,
        height: 14,
        borderRadius: 7,
        background: on ? 'var(--color-primary)' : 'var(--color-separator)',
        position: 'relative',
        flexShrink: 0,
        cursor: 'default',
        transition: 'background 0.2s ease',
      }}
    >
      <div
        style={{
          position: 'absolute',
          top: 2,
          left: on ? 14 : 2,
          width: 10,
          height: 10,
          borderRadius: '50%',
          background: on ? '#fff' : 'var(--color-background)',
          border: on ? 'none' : '1.5px solid var(--color-muted)',
          transition: 'left 0.2s ease',
          boxSizing: 'border-box',
        }}
      />
    </div>
  );
}

function UrgencyBadge({ urgency }: { urgency: 'Immediate' | 'Digest' }) {
  const isImmediate = urgency === 'Immediate';
  return (
    <span
      style={{
        fontSize: 10,
        fontWeight: 700,
        padding: '2px 7px',
        borderRadius: 20,
        background: isImmediate
          ? 'color-mix(in srgb, var(--color-danger) 15%, transparent)'
          : 'color-mix(in srgb, var(--color-primary) 15%, transparent)',
        color: isImmediate ? 'var(--color-danger)' : 'var(--color-primary)',
        textTransform: 'uppercase',
        letterSpacing: '0.06em',
      }}
    >
      {urgency}
    </span>
  );
}

function ChannelLabel({ channel }: { channel: string }) {
  const colorMap: Record<string, string> = {
    Email: 'var(--color-primary)',
    Slack: 'var(--color-success)',
    Webhook: 'var(--color-purple)',
  };
  return (
    <span
      style={{
        fontSize: 11,
        fontWeight: 600,
        padding: '2px 8px',
        borderRadius: 20,
        background: `color-mix(in srgb, ${colorMap[channel] ?? 'var(--color-muted)'} 15%, transparent)`,
        color: colorMap[channel] ?? 'var(--color-muted)',
      }}
    >
      {channel}
    </span>
  );
}

function StatusBadge({ status }: { status: HistoryStatus }) {
  const map: Record<HistoryStatus, { color: string; bg: string }> = {
    Delivered: {
      color: 'var(--color-success)',
      bg: 'color-mix(in srgb, var(--color-success) 12%, transparent)',
    },
    Failed: {
      color: 'var(--color-danger)',
      bg: 'color-mix(in srgb, var(--color-danger) 12%, transparent)',
    },
    Pending: {
      color: 'var(--color-warning)',
      bg: 'color-mix(in srgb, var(--color-warning) 12%, transparent)',
    },
  };
  const { color, bg } = map[status];
  return (
    <span
      style={{
        fontSize: 11,
        fontWeight: 600,
        padding: '2px 8px',
        borderRadius: 20,
        background: bg,
        color,
      }}
    >
      {status}
    </span>
  );
}

function CategoryCard({ category }: { category: NotifCategory }) {
  const colHeader: React.CSSProperties = {
    fontSize: 10,
    fontWeight: 700,
    color: 'var(--color-muted)',
    textTransform: 'uppercase',
    letterSpacing: '0.05em',
    textAlign: 'center',
  };

  return (
    <GlassCard
      className="p-5"
      hover={false}
      style={{ borderLeft: `3px solid ${category.borderColor}` }}
    >
      <div
        style={{
          fontSize: 13,
          fontWeight: 700,
          color: 'var(--color-foreground)',
          marginBottom: 12,
        }}
      >
        {category.title}
      </div>

      {/* Column headers */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: '1fr 60px 60px 60px 80px',
          padding: '4px 0',
          borderBottom: '1px solid var(--color-separator)',
          marginBottom: 4,
        }}
      >
        <span style={{ ...colHeader, textAlign: 'left' }}>Event</span>
        <span style={colHeader}>Email</span>
        <span style={colHeader}>Slack</span>
        <span style={colHeader}>Webhook</span>
        <span style={colHeader}>Urgency</span>
      </div>

      {/* Rows */}
      {category.rows.map((row, i) => (
        <div
          key={row.event}
          style={{
            display: 'grid',
            gridTemplateColumns: '1fr 60px 60px 60px 80px',
            alignItems: 'center',
            padding: '8px 0',
            borderBottom:
              i < category.rows.length - 1 ? '1px solid var(--color-separator)' : undefined,
          }}
        >
          <span style={{ fontSize: 12, color: 'var(--color-foreground)' }}>{row.event}</span>
          <div style={{ display: 'flex', justifyContent: 'center' }}>
            <Toggle on={row.email} />
          </div>
          <div style={{ display: 'flex', justifyContent: 'center' }}>
            <Toggle on={row.slack} />
          </div>
          <div style={{ display: 'flex', justifyContent: 'center' }}>
            <Toggle on={row.webhook} />
          </div>
          <div style={{ display: 'flex', justifyContent: 'center' }}>
            <UrgencyBadge urgency={row.urgency} />
          </div>
        </div>
      ))}
    </GlassCard>
  );
}

const inputStyle: React.CSSProperties = {
  background: 'transparent',
  border: '1px solid var(--color-separator)',
  borderRadius: 8,
  padding: '8px 12px',
  color: 'var(--color-foreground)',
  fontSize: 13,
  outline: 'none',
};

// ── Notifications page ────────────────────────────────────────────────────────
export default function Notifications() {
  useHotkeys();

  return (
    <motion.div
      variants={stagger}
      initial="hidden"
      animate="show"
      style={{
        display: 'flex',
        flexDirection: 'column',
        gap: 16,
        padding: '20px 24px',
        overflowY: 'auto',
        height: '100%',
      }}
    >
      {/* Page header with tabs */}
      <motion.div variants={fadeUp}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 4 }}>
          <Bell size={18} color="var(--color-foreground)" />
          <h1 style={{ fontSize: 20, fontWeight: 700, margin: 0 }}>Notifications</h1>
        </div>
        <div style={{ display: 'flex', gap: 2, marginTop: 12 }}>
          {['Preferences', 'History'].map((tab, i) => (
            <button
              key={tab}
              style={{
                background: i === 0 ? 'var(--color-primary)' : 'transparent',
                color: i === 0 ? '#fff' : 'var(--color-muted)',
                border: i === 0 ? 'none' : '1px solid var(--color-separator)',
                borderRadius: 8,
                padding: '6px 16px',
                fontSize: 13,
                fontWeight: 600,
                cursor: 'pointer',
              }}
            >
              {tab}
            </button>
          ))}
        </div>
      </motion.div>

      {/* Preferences: 4 category cards */}
      {CATEGORIES.map((cat) => (
        <motion.div key={cat.title} variants={fadeUp}>
          <CategoryCard category={cat} />
        </motion.div>
      ))}

      {/* Digest configuration */}
      <motion.div variants={fadeUp}>
        <GlassCard className="p-5" hover={false}>
          <SectionHeader title="Digest Configuration" />
          <div style={{ marginTop: 16 }}>
            {/* Description */}
            <div
              style={{
                padding: '8px 12px',
                background: 'color-mix(in srgb, var(--color-primary) 8%, transparent)',
                border: '1px solid color-mix(in srgb, var(--color-primary) 25%, transparent)',
                borderRadius: 8,
                fontSize: 13,
                color: 'var(--color-foreground)',
                marginBottom: 14,
              }}
            >
              Daily digest at <strong style={{ color: 'var(--color-primary)' }}>09:00 UTC</strong>
            </div>

            {/* Controls */}
            <div
              style={{
                display: 'grid',
                gridTemplateColumns: '1fr 1fr 1fr',
                gap: 12,
                alignItems: 'end',
              }}
            >
              <div>
                <label
                  style={{
                    display: 'block',
                    fontSize: 11,
                    fontWeight: 700,
                    color: 'var(--color-muted)',
                    textTransform: 'uppercase',
                    letterSpacing: '0.05em',
                    marginBottom: 4,
                  }}
                >
                  Frequency
                </label>
                <select style={{ ...inputStyle, width: '100%', boxSizing: 'border-box' }}>
                  <option>Daily</option>
                  <option>Weekly</option>
                </select>
              </div>

              <div>
                <label
                  style={{
                    display: 'block',
                    fontSize: 11,
                    fontWeight: 700,
                    color: 'var(--color-muted)',
                    textTransform: 'uppercase',
                    letterSpacing: '0.05em',
                    marginBottom: 4,
                  }}
                >
                  Time (UTC)
                </label>
                <select style={{ ...inputStyle, width: '100%', boxSizing: 'border-box' }}>
                  <option>09:00</option>
                  <option>06:00</option>
                  <option>12:00</option>
                  <option>18:00</option>
                </select>
              </div>

              <div>
                <label
                  style={{
                    display: 'block',
                    fontSize: 11,
                    fontWeight: 700,
                    color: 'var(--color-muted)',
                    textTransform: 'uppercase',
                    letterSpacing: '0.05em',
                    marginBottom: 4,
                  }}
                >
                  Format
                </label>
                <div style={{ display: 'flex', gap: 12, alignItems: 'center', paddingTop: 8 }}>
                  {['HTML', 'Plain Text'].map((fmt, i) => (
                    <label
                      key={fmt}
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: 6,
                        fontSize: 13,
                        color: 'var(--color-foreground)',
                        cursor: 'pointer',
                      }}
                    >
                      <input
                        type="radio"
                        name="digest-format"
                        defaultChecked={i === 0}
                        style={{ accentColor: 'var(--color-primary)', cursor: 'pointer' }}
                      />
                      {fmt}
                    </label>
                  ))}
                </div>
              </div>
            </div>

            <div style={{ display: 'flex', gap: 8, marginTop: 16 }}>
              <button
                style={{
                  background: 'var(--color-primary)',
                  color: '#fff',
                  border: 'none',
                  borderRadius: 8,
                  padding: '8px 16px',
                  fontSize: 13,
                  fontWeight: 600,
                  cursor: 'pointer',
                }}
              >
                Save Configuration
              </button>
              <button
                style={{
                  background: 'transparent',
                  color: 'var(--color-muted)',
                  border: '1px solid var(--color-separator)',
                  borderRadius: 8,
                  padding: '8px 16px',
                  fontSize: 13,
                  fontWeight: 600,
                  cursor: 'pointer',
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: 6,
                }}
              >
                <Send size={12} />
                Send Test Digest
              </button>
            </div>
          </div>
        </GlassCard>
      </motion.div>

      {/* Notification History */}
      <motion.div variants={fadeUp}>
        <GlassCard className="p-5" hover={false}>
          <SectionHeader title="Notification History" />
          <div style={{ marginTop: 12 }}>
            {/* Table header */}
            <div
              style={{
                display: 'grid',
                gridTemplateColumns: '2fr 80px 1fr 80px 120px',
                padding: '6px 8px',
                borderBottom: '1px solid var(--color-separator)',
                marginBottom: 4,
              }}
            >
              {['Event Type', 'Channel', 'Recipient', 'Status', 'Sent At'].map((col) => (
                <span
                  key={col}
                  style={{
                    fontSize: 10,
                    fontWeight: 700,
                    color: 'var(--color-muted)',
                    textTransform: 'uppercase',
                    letterSpacing: '0.05em',
                  }}
                >
                  {col}
                </span>
              ))}
            </div>

            {/* Rows */}
            {HISTORY_ROWS.map((row, i) => (
              <div
                key={i}
                style={{
                  display: 'grid',
                  gridTemplateColumns: '2fr 80px 1fr 80px 120px',
                  alignItems: 'center',
                  padding: '9px 8px',
                  borderBottom:
                    i < HISTORY_ROWS.length - 1 ? '1px solid var(--color-separator)' : undefined,
                  background:
                    i % 2 !== 0
                      ? 'color-mix(in srgb, var(--color-separator) 15%, transparent)'
                      : 'transparent',
                  borderRadius: 4,
                }}
              >
                <span style={{ fontSize: 12, color: 'var(--color-foreground)' }}>{row.event}</span>
                <ChannelLabel channel={row.channel} />
                <span
                  style={{
                    fontSize: 11,
                    color: 'var(--color-muted)',
                    fontFamily: 'monospace',
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                  }}
                >
                  {row.recipient}
                </span>
                <StatusBadge status={row.status} />
                <span
                  style={{ fontSize: 11, color: 'var(--color-muted)', fontFamily: 'monospace' }}
                >
                  {row.sentAt}
                </span>
              </div>
            ))}
          </div>
        </GlassCard>
      </motion.div>
    </motion.div>
  );
}
