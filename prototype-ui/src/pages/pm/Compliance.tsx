import { motion } from 'framer-motion';
import { Download, Play, AlertTriangle, Clock } from 'lucide-react';
import { useHotkeys } from '@/hooks/useHotkeys';
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
interface FrameworkData {
  name: string;
  score: number;
  status: 'Compliant' | 'Needs Work' | 'Non-Compliant';
  controlsPassed: number;
  controlsTotal: number;
  endpointsPassed: number;
  endpointsTotal: number;
  lastEvaluated: string;
  overdueCount: number;
}

interface OverdueControl {
  framework: 'NIST' | 'PCI DSS' | 'HIPAA';
  controlId: string;
  name: string;
  status: 'failing' | 'partial';
  sla: string;
  overdueBy: string;
  affected: number;
}

// ── Mock data ──────────────────────────────────────────────────────────────
const FRAMEWORKS: FrameworkData[] = [
  {
    name: 'NIST CSF 2.0',
    score: 78,
    status: 'Needs Work',
    controlsPassed: 35,
    controlsTotal: 45,
    endpointsPassed: 187,
    endpointsTotal: 240,
    lastEvaluated: 'Today 9:48 AM',
    overdueCount: 4,
  },
  {
    name: 'PCI DSS 4.0',
    score: 92,
    status: 'Compliant',
    controlsPassed: 57,
    controlsTotal: 62,
    endpointsPassed: 221,
    endpointsTotal: 240,
    lastEvaluated: 'Mar 8 9:15 AM',
    overdueCount: 0,
  },
  {
    name: 'HIPAA Security Rule',
    score: 65,
    status: 'Non-Compliant',
    controlsPassed: 28,
    controlsTotal: 43,
    endpointsPassed: 156,
    endpointsTotal: 240,
    lastEvaluated: 'Today 8:00 AM',
    overdueCount: 3,
  },
];

const OVERDUE_CONTROLS: OverdueControl[] = [
  {
    framework: 'NIST',
    controlId: 'ID.AM-2',
    name: 'Software asset inventory',
    status: 'failing',
    sla: 'Mar 1, 2026',
    overdueBy: '10 days',
    affected: 24,
  },
  {
    framework: 'HIPAA',
    controlId: '164.312(a)(1)',
    name: 'Access control policy',
    status: 'failing',
    sla: 'Feb 28, 2026',
    overdueBy: '11 days',
    affected: 156,
  },
  {
    framework: 'HIPAA',
    controlId: '164.308(a)(8)',
    name: 'Evaluation requirement',
    status: 'partial',
    sla: 'Feb 15, 2026',
    overdueBy: '24 days',
    affected: 89,
  },
  {
    framework: 'NIST',
    controlId: 'PR.IP-12',
    name: 'Vulnerability mgmt plan',
    status: 'failing',
    sla: 'Mar 5, 2026',
    overdueBy: '6 days',
    affected: 45,
  },
  {
    framework: 'NIST',
    controlId: 'DE.CM-8',
    name: 'Vulnerability scans',
    status: 'failing',
    sla: 'Mar 3, 2026',
    overdueBy: '8 days',
    affected: 63,
  },
  {
    framework: 'HIPAA',
    controlId: '164.312(b)',
    name: 'Audit controls',
    status: 'partial',
    sla: 'Mar 10, 2026',
    overdueBy: '1 day',
    affected: 22,
  },
  {
    framework: 'NIST',
    controlId: 'RS.MI-3',
    name: 'Newly identified vulnerabilities',
    status: 'failing',
    sla: 'Mar 8, 2026',
    overdueBy: '3 days',
    affected: 18,
  },
];

// ── Helpers ────────────────────────────────────────────────────────────────
function scoreColor(score: number): string {
  if (score >= 90) return 'var(--color-success)';
  if (score >= 75) return 'var(--color-warning)';
  return 'var(--color-danger)';
}

function frameworkColor(fw: 'NIST' | 'PCI DSS' | 'HIPAA'): string {
  if (fw === 'NIST') return 'var(--color-primary)';
  if (fw === 'PCI DSS') return 'var(--color-success)';
  return 'var(--color-purple)';
}

function statusBadgeColor(status: FrameworkData['status']): string {
  if (status === 'Compliant') return 'var(--color-success)';
  if (status === 'Needs Work') return 'var(--color-warning)';
  return 'var(--color-danger)';
}

// ── SVG Ring Gauge ─────────────────────────────────────────────────────────
function RingGauge({
  score,
  size = 80,
  strokeWidth = 7,
}: {
  score: number;
  size?: number;
  strokeWidth?: number;
}) {
  const r = (size - strokeWidth * 2) / 2;
  const circumference = 2 * Math.PI * r;
  const filled = (score / 100) * circumference;
  const gap = circumference - filled;
  const center = size / 2;
  const color = scoreColor(score);

  return (
    <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`} style={{ display: 'block' }}>
      {/* track */}
      <circle
        cx={center}
        cy={center}
        r={r}
        fill="none"
        stroke="var(--color-separator)"
        strokeWidth={strokeWidth}
      />
      {/* fill */}
      <circle
        cx={center}
        cy={center}
        r={r}
        fill="none"
        stroke={color}
        strokeWidth={strokeWidth}
        strokeLinecap="round"
        strokeDasharray={`${filled} ${gap}`}
        strokeDashoffset={circumference * 0.25}
        style={{ animation: 'sweep 0.8s ease-out both' }}
      />
      <text
        x={center}
        y={center + 5}
        textAnchor="middle"
        fontSize={size * 0.2}
        fontWeight={800}
        fill={color}
      >
        {score}%
      </text>
    </svg>
  );
}

// ── Overall Score Card ─────────────────────────────────────────────────────
function OverallScoreCard() {
  const overallScore = 78;

  return (
    <GlassCard className="p-5" hover={false}>
      <SectionHeader title="Overall Compliance Score" />
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 32,
          marginTop: 20,
        }}
      >
        {/* Big ring */}
        <div style={{ flexShrink: 0 }}>
          <RingGauge score={overallScore} size={120} strokeWidth={10} />
          <div
            style={{
              textAlign: 'center',
              marginTop: 8,
              fontSize: 11,
              fontWeight: 600,
              color: 'var(--color-warning)',
              letterSpacing: '0.02em',
            }}
          >
            Needs Improvement
          </div>
        </div>

        {/* Stats grid */}
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: '1fr 1fr',
            gap: '12px 32px',
            flex: 1,
          }}
        >
          {[
            { label: 'Compliant', value: '1 framework', color: 'var(--color-success)' },
            { label: 'Needs Work', value: '1 framework', color: 'var(--color-warning)' },
            { label: 'Non-Compliant', value: '1 framework', color: 'var(--color-danger)' },
            { label: 'Overdue Controls', value: '7', color: 'var(--color-warning)' },
          ].map(({ label, value, color }) => (
            <div key={label}>
              <div style={{ fontSize: 10, color: 'var(--color-muted)', marginBottom: 2 }}>
                {label}
              </div>
              <div style={{ fontSize: 18, fontWeight: 700, color }}>{value}</div>
            </div>
          ))}
        </div>

        {/* Framework mini-rings */}
        <div style={{ display: 'flex', gap: 20, flexShrink: 0 }}>
          {FRAMEWORKS.map((fw) => (
            <div
              key={fw.name}
              style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 6 }}
            >
              <RingGauge score={fw.score} size={56} strokeWidth={5} />
              <div
                style={{
                  fontSize: 9,
                  fontWeight: 600,
                  color: 'var(--color-muted)',
                  textAlign: 'center',
                  maxWidth: 56,
                  lineHeight: 1.3,
                }}
              >
                {fw.name.split(' ').slice(0, 2).join(' ')}
              </div>
            </div>
          ))}
        </div>
      </div>
    </GlassCard>
  );
}

// ── Framework Card ─────────────────────────────────────────────────────────
function FrameworkCard({ fw }: { fw: FrameworkData }) {
  const statusColor = statusBadgeColor(fw.status);
  const endpointPct = Math.round((fw.endpointsPassed / fw.endpointsTotal) * 100);

  return (
    <GlassCard className="p-5" hover={false}>
      {/* Header */}
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'flex-start',
          marginBottom: 16,
        }}
      >
        <div>
          <div
            style={{
              fontSize: 13,
              fontWeight: 700,
              color: 'var(--color-foreground)',
              marginBottom: 4,
            }}
          >
            {fw.name}
          </div>
          <span
            style={{
              display: 'inline-block',
              fontSize: 10,
              fontWeight: 600,
              color: statusColor,
              background: `color-mix(in srgb, ${statusColor} 12%, transparent)`,
              border: `1px solid color-mix(in srgb, ${statusColor} 30%, transparent)`,
              borderRadius: 4,
              padding: '1px 7px',
            }}
          >
            {fw.status}
          </span>
        </div>
        <RingGauge score={fw.score} size={64} strokeWidth={6} />
      </div>

      {/* Metrics */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
        {/* Controls */}
        <div>
          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              marginBottom: 4,
            }}
          >
            <span style={{ fontSize: 11, color: 'var(--color-muted)' }}>Controls passed</span>
            <span style={{ fontSize: 11, fontWeight: 600, color: 'var(--color-foreground)' }}>
              {fw.controlsPassed}/{fw.controlsTotal}
            </span>
          </div>
          <div
            style={{
              height: 4,
              borderRadius: 2,
              background: 'var(--color-separator)',
              overflow: 'hidden',
            }}
          >
            <div
              style={{
                height: '100%',
                width: `${(fw.controlsPassed / fw.controlsTotal) * 100}%`,
                borderRadius: 2,
                background: scoreColor(fw.score),
                transition: 'width 0.6s ease',
              }}
            />
          </div>
        </div>

        {/* Endpoints */}
        <div>
          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              marginBottom: 4,
            }}
          >
            <span style={{ fontSize: 11, color: 'var(--color-muted)' }}>Endpoint compliance</span>
            <span style={{ fontSize: 11, fontWeight: 600, color: 'var(--color-foreground)' }}>
              {fw.endpointsPassed}/{fw.endpointsTotal} ({endpointPct}%)
            </span>
          </div>
          <div
            style={{
              height: 4,
              borderRadius: 2,
              background: 'var(--color-separator)',
              overflow: 'hidden',
            }}
          >
            <div
              style={{
                height: '100%',
                width: `${endpointPct}%`,
                borderRadius: 2,
                background: scoreColor(fw.score),
                transition: 'width 0.6s ease',
              }}
            />
          </div>
        </div>

        {/* Footer */}
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            marginTop: 2,
          }}
        >
          <span style={{ fontSize: 10, color: 'var(--color-muted)' }}>
            Last evaluated: {fw.lastEvaluated}
          </span>
          {fw.overdueCount > 0 && (
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 4,
                fontSize: 10,
                fontWeight: 600,
                color: 'var(--color-warning)',
              }}
            >
              <AlertTriangle size={11} />
              {fw.overdueCount} overdue
            </div>
          )}
        </div>
      </div>
    </GlassCard>
  );
}

// ── Overdue Controls Table ─────────────────────────────────────────────────
function OverdueControlsTable() {
  return (
    <GlassCard className="p-5" hover={false}>
      <SectionHeader
        title="Overdue Controls"
        action={
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 6,
              fontSize: 11,
              fontWeight: 600,
              color: 'var(--color-warning)',
            }}
          >
            <AlertTriangle size={12} />7 controls past SLA
          </div>
        }
      />
      <div style={{ marginTop: 14, overflowX: 'auto' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 12 }}>
          <thead>
            <tr>
              {[
                'Framework',
                'Control ID',
                'Control Name',
                'Status',
                'SLA Deadline',
                'Overdue By',
                'Affected Endpoints',
              ].map((col) => (
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
            {OVERDUE_CONTROLS.map((ctrl, i) => {
              const fwColor = frameworkColor(ctrl.framework);
              const statusColor =
                ctrl.status === 'failing' ? 'var(--color-danger)' : 'var(--color-warning)';
              return (
                <tr
                  key={i}
                  style={{
                    borderBottom: '1px solid var(--color-separator)',
                    transition: 'background 0.15s',
                  }}
                >
                  {/* Framework */}
                  <td style={{ padding: '10px 12px 10px 0' }}>
                    <span
                      style={{
                        display: 'inline-block',
                        fontSize: 10,
                        fontWeight: 600,
                        color: fwColor,
                        background: `color-mix(in srgb, ${fwColor} 10%, transparent)`,
                        border: `1px solid color-mix(in srgb, ${fwColor} 25%, transparent)`,
                        borderRadius: 4,
                        padding: '1px 6px',
                      }}
                    >
                      {ctrl.framework}
                    </span>
                  </td>

                  {/* Control ID */}
                  <td
                    style={{
                      padding: '10px 12px 10px 0',
                      fontFamily: 'monospace',
                      fontSize: 11,
                      color: 'var(--color-foreground)',
                      whiteSpace: 'nowrap',
                    }}
                  >
                    {ctrl.controlId}
                  </td>

                  {/* Control Name */}
                  <td
                    style={{
                      padding: '10px 12px 10px 0',
                      color: 'var(--color-foreground)',
                      maxWidth: 200,
                    }}
                  >
                    {ctrl.name}
                  </td>

                  {/* Status */}
                  <td style={{ padding: '10px 12px 10px 0' }}>
                    <span
                      style={{
                        display: 'inline-block',
                        fontSize: 10,
                        fontWeight: 600,
                        color: statusColor,
                        background: `color-mix(in srgb, ${statusColor} 10%, transparent)`,
                        border: `1px solid color-mix(in srgb, ${statusColor} 25%, transparent)`,
                        borderRadius: 4,
                        padding: '1px 7px',
                        textTransform: 'capitalize',
                      }}
                    >
                      {ctrl.status}
                    </span>
                  </td>

                  {/* SLA Deadline */}
                  <td
                    style={{
                      padding: '10px 12px 10px 0',
                      color: 'var(--color-muted)',
                      fontSize: 11,
                      whiteSpace: 'nowrap',
                    }}
                  >
                    <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                      <Clock size={11} color="var(--color-muted)" />
                      {ctrl.sla}
                    </div>
                  </td>

                  {/* Overdue By */}
                  <td
                    style={{
                      padding: '10px 12px 10px 0',
                      fontWeight: 600,
                      fontSize: 11,
                      color: 'var(--color-danger)',
                      whiteSpace: 'nowrap',
                    }}
                  >
                    +{ctrl.overdueBy}
                  </td>

                  {/* Affected Endpoints */}
                  <td
                    style={{
                      padding: '10px 0 10px 0',
                      fontSize: 11,
                      fontWeight: 600,
                      color: 'var(--color-foreground)',
                    }}
                  >
                    {ctrl.affected}
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

// ── Compliance Page ────────────────────────────────────────────────────────
export default function Compliance() {
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
          title="Compliance"
          subtitle="Framework adherence and control status across all endpoints"
          actions={
            <>
              <button
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 6,
                  padding: '6px 12px',
                  borderRadius: 6,
                  border: '1px solid var(--color-separator)',
                  background: 'transparent',
                  color: 'var(--color-foreground)',
                  fontSize: 12,
                  fontWeight: 500,
                  cursor: 'pointer',
                }}
              >
                <Download size={13} />
                Export Report
              </button>
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
                <Play size={13} />
                Evaluate All
              </button>
            </>
          }
        />
      </motion.div>

      {/* Overall Score */}
      <motion.div variants={fadeUp}>
        <OverallScoreCard />
      </motion.div>

      {/* Framework Cards */}
      <motion.div
        variants={fadeUp}
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(3, 1fr)',
          gap: 12,
        }}
      >
        {FRAMEWORKS.map((fw) => (
          <FrameworkCard key={fw.name} fw={fw} />
        ))}
      </motion.div>

      {/* Overdue Controls Table */}
      <motion.div variants={fadeUp}>
        <SectionHeader title="Overdue Controls" />
        <div style={{ marginTop: 10 }}>
          <OverdueControlsTable />
        </div>
      </motion.div>
    </motion.div>
  );
}
