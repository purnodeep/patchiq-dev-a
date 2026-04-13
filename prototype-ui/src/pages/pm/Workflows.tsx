import { motion } from 'framer-motion';
import { CheckCircle, XCircle, Calendar, Zap, MousePointer, BookOpen, Plus } from 'lucide-react';
import { useHotkeys } from '@/hooks/useHotkeys';
import { GlassCard } from '@/components/shared/GlassCard';
import { SectionHeader } from '@/components/shared/SectionHeader';
import { PageHeader } from '@/components/shared/PageHeader';

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
type TriggerType = 'Schedule' | 'Event' | 'Manual';
type WorkflowStatus = 'active' | 'draft';
type ExecutionStatus = 'success' | 'failed';

interface Workflow {
  id: string;
  name: string;
  trigger: TriggerType;
  status: WorkflowStatus;
  executions: number;
  lastRun: string;
  lastOk: string;
  stages: string[];
}

interface Execution {
  workflow: string;
  status: ExecutionStatus;
  duration: string;
  triggeredBy: string;
  started: string;
}

// ── Mock data ─────────────────────────────────────────────────────────────────
const WORKFLOWS: Workflow[] = [
  {
    id: 'wf-001',
    name: 'Critical Patch Fast Track',
    trigger: 'Event',
    status: 'active',
    executions: 22,
    lastRun: '2h ago',
    lastOk: 'Success',
    stages: ['trigger', 'filter', 'approval', 'deploy', 'notify'],
  },
  {
    id: 'wf-002',
    name: 'Standard Monthly Rollout',
    trigger: 'Schedule',
    status: 'active',
    executions: 8,
    lastRun: '7d ago',
    lastOk: 'Success',
    stages: ['trigger', 'filter', 'approval', 'deploy', 'notify'],
  },
  {
    id: 'wf-003',
    name: 'Database Server Update',
    trigger: 'Manual',
    status: 'active',
    executions: 3,
    lastRun: '14d ago',
    lastOk: 'Failed',
    stages: ['trigger', 'filter', 'approval', 'deploy', 'notify'],
  },
  {
    id: 'wf-004',
    name: 'Linux Security Baseline',
    trigger: 'Schedule',
    status: 'active',
    executions: 12,
    lastRun: '3d ago',
    lastOk: 'Success',
    stages: ['trigger', 'filter', 'deploy', 'notify'],
  },
  {
    id: 'wf-005',
    name: 'Emergency CVE Response',
    trigger: 'Event',
    status: 'draft',
    executions: 0,
    lastRun: 'Never',
    lastOk: '—',
    stages: ['trigger', 'filter', 'approval', 'deploy', 'notify'],
  },
];

const EXECUTIONS: Execution[] = [
  {
    workflow: 'Critical Patch Fast Track',
    status: 'success',
    duration: '8 min 22s',
    triggeredBy: 'CVE-2024-21762',
    started: '2h ago',
  },
  {
    workflow: 'Standard Monthly Rollout',
    status: 'success',
    duration: '1h 34m',
    triggeredBy: 'Schedule',
    started: '7d ago',
  },
  {
    workflow: 'Linux Security Baseline',
    status: 'success',
    duration: '42 min',
    triggeredBy: 'Schedule',
    started: '3d ago',
  },
  {
    workflow: 'Database Server Update',
    status: 'failed',
    duration: '3 min',
    triggeredBy: 'R. Kumar',
    started: '14d ago',
  },
  {
    workflow: 'Critical Patch Fast Track',
    status: 'success',
    duration: '12 min',
    triggeredBy: 'CVE-2024-38094',
    started: '5d ago',
  },
  {
    workflow: 'Linux Security Baseline',
    status: 'success',
    duration: '39 min',
    triggeredBy: 'Schedule',
    started: '10d ago',
  },
];

// ── Stage dot colors ──────────────────────────────────────────────────────────
const STAGE_COLOR: Record<string, string> = {
  trigger: 'var(--color-purple)',
  filter: 'var(--color-primary)',
  approval: 'var(--color-warning)',
  deploy: 'var(--color-success)',
  notify: 'var(--color-cyan)',
};

// ── Trigger badge ─────────────────────────────────────────────────────────────
function TriggerBadge({ trigger }: { trigger: TriggerType }) {
  const config: Record<TriggerType, { icon: React.ReactNode; color: string; bg: string }> = {
    Schedule: {
      icon: <Calendar size={10} />,
      color: 'var(--color-primary)',
      bg: 'color-mix(in srgb, var(--color-primary) 12%, transparent)',
    },
    Event: {
      icon: <Zap size={10} />,
      color: 'var(--color-warning)',
      bg: 'color-mix(in srgb, var(--color-warning) 12%, transparent)',
    },
    Manual: {
      icon: <MousePointer size={10} />,
      color: 'var(--color-muted)',
      bg: 'color-mix(in srgb, var(--color-separator) 80%, transparent)',
    },
  };
  const { icon, color, bg } = config[trigger];
  return (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 4,
        padding: '2px 8px',
        borderRadius: 12,
        background: bg,
        color,
        fontSize: 10,
        fontWeight: 600,
        letterSpacing: '0.02em',
      }}
    >
      {icon}
      {trigger}
    </span>
  );
}

// ── Status badge ──────────────────────────────────────────────────────────────
function WorkflowStatusBadge({ status }: { status: WorkflowStatus }) {
  const isActive = status === 'active';
  return (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 4,
        padding: '2px 8px',
        borderRadius: 12,
        background: isActive
          ? 'color-mix(in srgb, var(--color-success) 12%, transparent)'
          : 'color-mix(in srgb, var(--color-separator) 80%, transparent)',
        color: isActive ? 'var(--color-success)' : 'var(--color-muted)',
        fontSize: 10,
        fontWeight: 600,
      }}
    >
      <div
        style={{
          width: 6,
          height: 6,
          borderRadius: '50%',
          background: isActive ? 'var(--color-success)' : 'var(--color-muted)',
        }}
      />
      {isActive ? 'Active' : 'Draft'}
    </span>
  );
}

// ── Pipeline mini-viz ─────────────────────────────────────────────────────────
function MiniPipeline({ stages }: { stages: string[] }) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 0, paddingTop: 2 }}>
      {stages.map((stage, i) => {
        const color = STAGE_COLOR[stage] ?? 'var(--color-separator)';
        return (
          <div key={stage} style={{ display: 'flex', alignItems: 'center' }}>
            {i > 0 && (
              <div
                style={{
                  width: 12,
                  height: 1.5,
                  background: 'var(--color-separator)',
                }}
              />
            )}
            <div
              title={stage}
              style={{
                width: 10,
                height: 10,
                borderRadius: '50%',
                background: color,
              }}
            />
          </div>
        );
      })}
      <div
        style={{ display: 'flex', alignItems: 'center', gap: 6, marginLeft: 10, flexWrap: 'wrap' }}
      >
        {stages.map((stage) => (
          <span
            key={stage}
            style={{
              fontSize: 9,
              color: STAGE_COLOR[stage] ?? 'var(--color-muted)',
              textTransform: 'capitalize',
              letterSpacing: '0.02em',
            }}
          >
            {stage}
          </span>
        ))}
      </div>
    </div>
  );
}

// ── Workflow card ─────────────────────────────────────────────────────────────
function WorkflowCard({ wf }: { wf: Workflow }) {
  const lastOkColor =
    wf.lastOk === 'Success'
      ? 'var(--color-success)'
      : wf.lastOk === 'Failed'
        ? 'var(--color-danger)'
        : 'var(--color-muted)';

  return (
    <GlassCard className="p-5" hover={false}>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        {/* Header */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          <div
            style={{
              display: 'flex',
              alignItems: 'flex-start',
              justifyContent: 'space-between',
              gap: 8,
            }}
          >
            <span
              style={{
                fontSize: 13,
                fontWeight: 700,
                color: 'var(--color-foreground)',
                lineHeight: 1.3,
              }}
            >
              {wf.name}
            </span>
            <WorkflowStatusBadge status={wf.status} />
          </div>
          <TriggerBadge trigger={wf.trigger} />
        </div>

        {/* Pipeline */}
        <div>
          <span
            style={{
              fontSize: 10,
              color: 'var(--color-muted)',
              textTransform: 'uppercase',
              letterSpacing: '0.05em',
              fontWeight: 600,
            }}
          >
            Stages
          </span>
          <div style={{ marginTop: 6 }}>
            <MiniPipeline stages={wf.stages} />
          </div>
        </div>

        {/* Divider */}
        <div style={{ height: 1, background: 'var(--color-separator)' }} />

        {/* Stats row */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
            <span style={{ fontSize: 16, fontWeight: 700, color: 'var(--color-foreground)' }}>
              {wf.executions}
            </span>
            <span style={{ fontSize: 10, color: 'var(--color-muted)' }}>executions</span>
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 2, textAlign: 'right' }}>
            <span style={{ fontSize: 11, color: 'var(--color-muted)' }}>Last run</span>
            <span style={{ fontSize: 11, fontWeight: 600, color: 'var(--color-foreground)' }}>
              {wf.lastRun}
            </span>
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 2, textAlign: 'right' }}>
            <span style={{ fontSize: 11, color: 'var(--color-muted)' }}>Last result</span>
            <span style={{ fontSize: 11, fontWeight: 600, color: lastOkColor }}>{wf.lastOk}</span>
          </div>
        </div>
      </div>
    </GlassCard>
  );
}

// ── Executions table ──────────────────────────────────────────────────────────
const TH_STYLE: React.CSSProperties = {
  padding: '8px 12px',
  textAlign: 'left',
  fontSize: 11,
  fontWeight: 600,
  color: 'var(--color-muted)',
  letterSpacing: '0.05em',
  textTransform: 'uppercase',
  borderBottom: '1px solid var(--color-separator)',
  whiteSpace: 'nowrap',
};

const TD_STYLE: React.CSSProperties = {
  padding: '10px 12px',
  fontSize: 12,
  color: 'var(--color-foreground)',
  borderBottom: '1px solid color-mix(in srgb, var(--color-separator) 50%, transparent)',
  verticalAlign: 'middle',
};

function ExecutionStatusChip({ status }: { status: ExecutionStatus }) {
  const isSuccess = status === 'success';
  return (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 5,
        padding: '3px 9px',
        borderRadius: 12,
        background: isSuccess
          ? 'color-mix(in srgb, var(--color-success) 12%, transparent)'
          : 'color-mix(in srgb, var(--color-danger) 12%, transparent)',
        color: isSuccess ? 'var(--color-success)' : 'var(--color-danger)',
        fontSize: 11,
        fontWeight: 600,
      }}
    >
      {isSuccess ? <CheckCircle size={11} /> : <XCircle size={11} />}
      {isSuccess ? 'Success' : 'Failed'}
    </span>
  );
}

function ExecutionsTable({ executions }: { executions: Execution[] }) {
  return (
    <div style={{ overflowX: 'auto' }}>
      <table style={{ width: '100%', borderCollapse: 'collapse' }}>
        <thead>
          <tr>
            <th style={TH_STYLE}>Workflow</th>
            <th style={TH_STYLE}>Status</th>
            <th style={TH_STYLE}>Duration</th>
            <th style={TH_STYLE}>Triggered By</th>
            <th style={TH_STYLE}>Started</th>
          </tr>
        </thead>
        <tbody>
          {executions.map((ex, i) => (
            <tr
              key={i}
              style={{ cursor: 'pointer', transition: 'background 0.1s' }}
              onMouseEnter={(e) =>
                (e.currentTarget.style.background =
                  'color-mix(in srgb, var(--color-primary) 4%, transparent)')
              }
              onMouseLeave={(e) => (e.currentTarget.style.background = 'transparent')}
            >
              <td style={{ ...TD_STYLE, fontWeight: 500 }}>{ex.workflow}</td>
              <td style={TD_STYLE}>
                <ExecutionStatusChip status={ex.status} />
              </td>
              <td style={{ ...TD_STYLE, color: 'var(--color-muted)' }}>{ex.duration}</td>
              <td style={{ ...TD_STYLE, fontSize: 11 }}>
                <span
                  style={{
                    padding: '2px 8px',
                    borderRadius: 4,
                    background: ex.triggeredBy.startsWith('CVE')
                      ? 'color-mix(in srgb, var(--color-danger) 12%, transparent)'
                      : 'color-mix(in srgb, var(--color-separator) 60%, transparent)',
                    color: ex.triggeredBy.startsWith('CVE')
                      ? 'var(--color-danger)'
                      : 'var(--color-foreground)',
                    fontWeight: ex.triggeredBy.startsWith('CVE') ? 600 : 400,
                  }}
                >
                  {ex.triggeredBy}
                </span>
              </td>
              <td style={{ ...TD_STYLE, color: 'var(--color-muted)', fontSize: 11 }}>
                {ex.started}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

// ── Page ──────────────────────────────────────────────────────────────────────
export default function Workflows() {
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
      {/* Row 1 — Page header */}
      <motion.div variants={fadeUp}>
        <PageHeader
          title="Workflows"
          subtitle="5 workflows"
          actions={
            <>
              <button
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 6,
                  padding: '7px 14px',
                  borderRadius: 6,
                  border: '1px solid var(--color-separator)',
                  background: 'transparent',
                  color: 'var(--color-foreground)',
                  fontSize: 12,
                  fontWeight: 600,
                  cursor: 'pointer',
                }}
              >
                <BookOpen size={13} />
                Browse Templates
              </button>
              <button
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 6,
                  padding: '7px 14px',
                  borderRadius: 6,
                  border: 'none',
                  background: 'var(--color-primary)',
                  color: '#fff',
                  fontSize: 12,
                  fontWeight: 600,
                  cursor: 'pointer',
                }}
              >
                <Plus size={14} />
                Create Workflow
              </button>
            </>
          }
        />
      </motion.div>

      {/* Row 2 — Workflow cards grid */}
      <motion.div
        variants={stagger}
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(3, 1fr)',
          gap: 12,
        }}
      >
        {WORKFLOWS.map((wf) => (
          <motion.div key={wf.id} variants={fadeUp}>
            <WorkflowCard wf={wf} />
          </motion.div>
        ))}
      </motion.div>

      {/* Row 3 — Recent Executions */}
      <motion.div variants={fadeUp}>
        <GlassCard className="p-5" hover={false}>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
            <SectionHeader
              title="Recent Executions"
              action={
                <span style={{ fontSize: 11, color: 'var(--color-muted)' }}>Last 6 runs</span>
              }
            />
            <ExecutionsTable executions={EXECUTIONS} />
          </div>
        </GlassCard>
      </motion.div>
    </motion.div>
  );
}
