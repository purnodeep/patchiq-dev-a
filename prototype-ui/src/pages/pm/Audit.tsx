import { motion } from 'framer-motion';
import {
  Download,
  ChevronDown,
  Calendar,
  Settings,
  CheckCircle,
  AlertTriangle,
  XCircle,
  Activity,
} from 'lucide-react';
import { useHotkeys } from '@/hooks/useHotkeys';
import { StatCard } from '@/components/shared/StatCard';
import { GlassCard } from '@/components/shared/GlassCard';
import { SectionHeader } from '@/components/shared/SectionHeader';
import { PageHeader } from '@/components/shared/PageHeader';

// ── Framer Motion variants ─────────────────────────────────────────────────
const stagger = {
  hidden: {},
  show: { transition: { staggerChildren: 0.06 } },
};

const fadeUp = {
  hidden: { opacity: 0, y: 12 },
  show: { opacity: 1, y: 0, transition: { duration: 0.4, ease: 'easeOut' } },
};

// ── Types ──────────────────────────────────────────────────────────────────
type EventStatus = 'success' | 'warning' | 'error';

interface AuditEvent {
  ts: string;
  user: string;
  action: string;
  resourceType: string;
  resource: string;
  status: EventStatus;
}

// ── Mock data ──────────────────────────────────────────────────────────────
const AUDIT_EVENTS: AuditEvent[] = [
  {
    ts: 'Today 11:42 AM',
    user: 'J. Davis',
    action: 'Deployment Started',
    resourceType: 'Deployment',
    resource: 'DEP-0047',
    status: 'success',
  },
  {
    ts: 'Today 11:30 AM',
    user: 'System',
    action: 'CVE Evaluated',
    resourceType: 'CVE',
    resource: 'CVE-2024-21762',
    status: 'success',
  },
  {
    ts: 'Today 10:15 AM',
    user: 'S. Williams',
    action: 'Policy Updated',
    resourceType: 'Policy',
    resource: 'Emergency Critical Fast Track',
    status: 'success',
  },
  {
    ts: 'Today 9:48 AM',
    user: 'System',
    action: 'Compliance Scan',
    resourceType: 'Compliance',
    resource: 'NIST CSF 2.0',
    status: 'warning',
  },
  {
    ts: 'Today 9:01 AM',
    user: 'R. Kumar',
    action: 'Deployment Failed',
    resourceType: 'Deployment',
    resource: 'DEP-0041',
    status: 'error',
  },
  {
    ts: 'Today 8:30 AM',
    user: 'System',
    action: 'Hub Sync Complete',
    resourceType: 'System',
    resource: 'hub-sync-0201',
    status: 'success',
  },
  {
    ts: 'Today 8:15 AM',
    user: 'Policy Schedule',
    action: 'Deployment Started',
    resourceType: 'Deployment',
    resource: 'DEP-0043',
    status: 'success',
  },
  {
    ts: 'Today 7:22 AM',
    user: 'S. Williams',
    action: 'Role Updated',
    resourceType: 'Role',
    resource: 'Operator',
    status: 'success',
  },
  {
    ts: 'Today 6:00 AM',
    user: 'Policy Schedule',
    action: 'Deployment Completed',
    resourceType: 'Deployment',
    resource: 'DEP-0044',
    status: 'success',
  },
  {
    ts: 'Yesterday 11:00 PM',
    user: 'System',
    action: 'Agent Offline',
    resourceType: 'Endpoint',
    resource: 'dev-build-01',
    status: 'warning',
  },
  {
    ts: 'Yesterday 3:15 PM',
    user: 'J. Davis',
    action: 'Endpoint Enrolled',
    resourceType: 'Endpoint',
    resource: 'k8s-node-07',
    status: 'success',
  },
  {
    ts: 'Yesterday 2:00 PM',
    user: 'System',
    action: 'CVE Published',
    resourceType: 'CVE',
    resource: 'CVE-2024-49113',
    status: 'warning',
  },
  {
    ts: 'Yesterday 10:30 AM',
    user: 'D. Nair',
    action: 'Policy Created',
    resourceType: 'Policy',
    resource: 'Linux Security Baseline',
    status: 'success',
  },
  {
    ts: 'Yesterday 8:00 AM',
    user: 'Policy Schedule',
    action: 'Deployment Completed',
    resourceType: 'Deployment',
    resource: 'DEP-0042',
    status: 'success',
  },
  {
    ts: 'Mar 8 9:15 AM',
    user: 'System',
    action: 'Compliance Evaluated',
    resourceType: 'Compliance',
    resource: 'PCI DSS 4.0',
    status: 'success',
  },
];

// ── Helpers ────────────────────────────────────────────────────────────────
function statusColor(s: EventStatus): string {
  if (s === 'success') return 'var(--color-success)';
  if (s === 'warning') return 'var(--color-warning)';
  return 'var(--color-danger)';
}

function actionVerbColor(action: string): string {
  if (action.includes('Started')) return 'var(--color-primary)';
  if (action.includes('Completed')) return 'var(--color-success)';
  if (action.includes('Failed')) return 'var(--color-danger)';
  if (action.includes('Updated')) return 'var(--color-cyan)';
  if (action.includes('Created')) return 'var(--color-purple)';
  return 'var(--color-muted)';
}

/** Extract initials from a display name. "J. Davis" → "JD", "System" → gear icon */
function getUserInitials(user: string): string {
  if (user === 'System' || user === 'Policy Schedule') return '';
  return user
    .split(/[\s.]+/)
    .filter(Boolean)
    .map((p) => p[0].toUpperCase())
    .join('')
    .slice(0, 2);
}

/** Pick a stable background color for initials avatar */
const AVATAR_COLORS = [
  'var(--color-primary)',
  'var(--color-cyan)',
  'var(--color-purple)',
  'var(--color-success)',
  'var(--color-warning)',
];
function avatarColor(user: string): string {
  let hash = 0;
  for (let i = 0; i < user.length; i++) hash = (hash * 31 + user.charCodeAt(i)) & 0xffff;
  return AVATAR_COLORS[hash % AVATAR_COLORS.length];
}

// ── User Avatar ────────────────────────────────────────────────────────────
function UserAvatar({ user }: { user: string }) {
  const isSystem = user === 'System';
  const isSchedule = user === 'Policy Schedule';

  if (isSystem) {
    return (
      <div
        style={{
          width: 24,
          height: 24,
          borderRadius: '50%',
          background: 'color-mix(in srgb, var(--color-muted) 15%, transparent)',
          border: '1px solid var(--color-separator)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          flexShrink: 0,
        }}
      >
        <Settings size={12} color="var(--color-muted)" />
      </div>
    );
  }

  if (isSchedule) {
    return (
      <div
        style={{
          width: 24,
          height: 24,
          borderRadius: '50%',
          background: 'color-mix(in srgb, var(--color-cyan) 15%, transparent)',
          border: '1px solid color-mix(in srgb, var(--color-cyan) 25%, transparent)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          flexShrink: 0,
        }}
      >
        <Calendar size={12} color="var(--color-cyan)" />
      </div>
    );
  }

  const initials = getUserInitials(user);
  const bg = avatarColor(user);

  return (
    <div
      style={{
        width: 24,
        height: 24,
        borderRadius: '50%',
        background: `color-mix(in srgb, ${bg} 20%, transparent)`,
        border: `1px solid color-mix(in srgb, ${bg} 35%, transparent)`,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        fontSize: 9,
        fontWeight: 700,
        color: bg,
        flexShrink: 0,
        letterSpacing: '0.02em',
      }}
    >
      {initials}
    </div>
  );
}

// ── Status Badge ───────────────────────────────────────────────────────────
function StatusBadge({ status }: { status: EventStatus }) {
  const color = statusColor(status);
  const Icon = status === 'success' ? CheckCircle : status === 'warning' ? AlertTriangle : XCircle;

  return (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 4,
        fontSize: 10,
        fontWeight: 600,
        color,
        background: `color-mix(in srgb, ${color} 10%, transparent)`,
        border: `1px solid color-mix(in srgb, ${color} 25%, transparent)`,
        borderRadius: 4,
        padding: '2px 7px',
        textTransform: 'capitalize',
      }}
    >
      <Icon size={10} />
      {status}
    </span>
  );
}

// ── Audit Table ────────────────────────────────────────────────────────────
function AuditTable() {
  const columns = ['Timestamp', 'User', 'Action', 'Resource Type', 'Resource', 'Status'];

  return (
    <GlassCard className="p-5" hover={false}>
      <SectionHeader
        title="Recent Events"
        action={
          <span style={{ fontSize: 11, color: 'var(--color-muted)' }}>
            Showing 15 of 1,247 events
          </span>
        }
      />
      <div style={{ marginTop: 14, overflowX: 'auto' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 12 }}>
          <thead>
            <tr>
              {columns.map((col) => (
                <th
                  key={col}
                  style={{
                    textAlign: 'left',
                    padding: '0 12px 10px 0',
                    fontSize: 10,
                    fontWeight: 600,
                    color: 'var(--color-muted)',
                    letterSpacing: '0.04em',
                    textTransform: 'uppercase',
                    borderBottom: '1px solid var(--color-separator)',
                    whiteSpace: 'nowrap',
                  }}
                >
                  {col}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {AUDIT_EVENTS.map((evt, i) => {
              const verbColor = actionVerbColor(evt.action);
              return (
                <tr
                  key={i}
                  style={{
                    borderBottom: '1px solid var(--color-separator)',
                  }}
                >
                  {/* Timestamp */}
                  <td
                    style={{
                      padding: '10px 12px 10px 0',
                      fontSize: 11,
                      color: 'var(--color-muted)',
                      whiteSpace: 'nowrap',
                    }}
                  >
                    {evt.ts}
                  </td>

                  {/* User */}
                  <td style={{ padding: '10px 12px 10px 0' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 7 }}>
                      <UserAvatar user={evt.user} />
                      <span
                        style={{
                          fontSize: 11,
                          fontWeight: 500,
                          color: 'var(--color-foreground)',
                          whiteSpace: 'nowrap',
                        }}
                      >
                        {evt.user}
                      </span>
                    </div>
                  </td>

                  {/* Action */}
                  <td
                    style={{
                      padding: '10px 12px 10px 0',
                      fontWeight: 600,
                      fontSize: 12,
                      color: verbColor,
                      whiteSpace: 'nowrap',
                    }}
                  >
                    {evt.action}
                  </td>

                  {/* Resource Type */}
                  <td
                    style={{
                      padding: '10px 12px 10px 0',
                      fontSize: 11,
                      color: 'var(--color-muted)',
                    }}
                  >
                    {evt.resourceType}
                  </td>

                  {/* Resource */}
                  <td
                    style={{
                      padding: '10px 12px 10px 0',
                      fontFamily: 'monospace',
                      fontSize: 11,
                      color: 'var(--color-foreground)',
                      maxWidth: 200,
                    }}
                  >
                    {evt.resource}
                  </td>

                  {/* Status */}
                  <td style={{ padding: '10px 0 10px 0' }}>
                    <StatusBadge status={evt.status} />
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </GlassCard>
  );
}

// ── Audit Log Page ─────────────────────────────────────────────────────────
export default function Audit() {
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
      {/* Page Header */}
      <motion.div variants={fadeUp}>
        <PageHeader
          title="Audit Log"
          subtitle="All system and user actions recorded for compliance and investigation"
          actions={
            <>
              {/* Date range filter */}
              <button
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 6,
                  padding: '6px 10px',
                  borderRadius: 6,
                  border: '1px solid var(--color-separator)',
                  background: 'transparent',
                  color: 'var(--color-foreground)',
                  fontSize: 12,
                  fontWeight: 500,
                  cursor: 'pointer',
                }}
              >
                <Calendar size={13} />
                This week
                <ChevronDown size={12} color="var(--color-muted)" />
              </button>

              {/* Event type filter */}
              <button
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 6,
                  padding: '6px 10px',
                  borderRadius: 6,
                  border: '1px solid var(--color-separator)',
                  background: 'transparent',
                  color: 'var(--color-foreground)',
                  fontSize: 12,
                  fontWeight: 500,
                  cursor: 'pointer',
                }}
              >
                All event types
                <ChevronDown size={12} color="var(--color-muted)" />
              </button>

              {/* Export */}
              <button
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 6,
                  padding: '6px 12px',
                  borderRadius: 6,
                  border: 'none',
                  background: 'var(--color-primary)',
                  color: '#fff',
                  fontSize: 12,
                  fontWeight: 600,
                  cursor: 'pointer',
                }}
              >
                <Download size={13} />
                Export
              </button>
            </>
          }
        />
      </motion.div>

      {/* Summary Stats */}
      <motion.div
        variants={fadeUp}
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(4, 1fr)',
          gap: 12,
        }}
      >
        <StatCard
          icon={<Activity size={16} />}
          iconColor="var(--color-primary)"
          value="1,247"
          valueColor="var(--color-foreground)"
          label="Total Events"
          trendText="this week"
        />
        <StatCard
          icon={<CheckCircle size={16} />}
          iconColor="var(--color-success)"
          value="1,198"
          valueColor="var(--color-success)"
          label="Successful"
          trend={{ value: '96.1%', positive: true }}
          trendText="success rate"
        />
        <StatCard
          icon={<AlertTriangle size={16} />}
          iconColor="var(--color-warning)"
          value="38"
          valueColor="var(--color-warning)"
          label="Warnings"
          trend={{ value: '3.0%', positive: false }}
          trendText="of total"
        />
        <StatCard
          icon={<XCircle size={16} />}
          iconColor="var(--color-danger)"
          value="11"
          valueColor="var(--color-danger)"
          label="Errors"
          trend={{ value: '0.9%', positive: false }}
          trendText="of total"
        />
      </motion.div>

      {/* Audit Table */}
      <motion.div variants={fadeUp}>
        <AuditTable />
      </motion.div>
    </motion.div>
  );
}
