import { motion } from 'framer-motion';
import { useState } from 'react';
import {
  Play,
  CheckCircle,
  XCircle,
  Clock,
  Plus,
  Search,
  Calendar,
  ChevronRight,
  ChevronDown,
  type LucideIcon,
} from 'lucide-react';
import { useHotkeys } from '@/hooks/useHotkeys';
import { StatCard } from '@/components/shared/StatCard';
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
type DeploymentStatus = 'running' | 'completed' | 'failed' | 'pending';

interface Deployment {
  id: string;
  policy: string;
  status: DeploymentStatus;
  progress: number;
  targets: { done: number; total: number };
  waves: string;
  started: string;
  duration: string;
  triggeredBy: string;
  waveDone: number;
  waveActive: number;
  wavePend: number;
  error?: string;
}

// ── Mock data ─────────────────────────────────────────────────────────────────
const POLICY_OPTIONS = [
  'Emergency Critical Fast Track',
  'Auto-Deploy Critical Patches',
  'Weekly Security Scan',
  'Linux Security Baseline',
  'Dev Environment Updates',
  'Database Server Policy',
  'Network Device Hardening',
];

const DEPLOYMENTS: Deployment[] = [
  {
    id: 'DEP-0047',
    policy: 'Emergency Critical Fast Track',
    status: 'running',
    progress: 68,
    targets: { done: 10, total: 15 },
    waves: '1/1',
    started: '2026-03-12',
    duration: '22 min elapsed',
    triggeredBy: 'CVE-2024-21762',
    waveDone: 10,
    waveActive: 3,
    wavePend: 2,
  },
  {
    id: 'DEP-0046',
    policy: 'Auto-Deploy Critical Patches',
    status: 'running',
    progress: 41,
    targets: { done: 9, total: 22 },
    waves: '1/1',
    started: '2026-03-12',
    duration: '45 min elapsed',
    triggeredBy: 'Policy schedule',
    waveDone: 9,
    waveActive: 5,
    wavePend: 8,
  },
  {
    id: 'DEP-0045',
    policy: 'Weekly Security Scan',
    status: 'running',
    progress: 55,
    targets: { done: 49, total: 89 },
    waves: '2/3',
    started: '2026-03-12',
    duration: '3h elapsed',
    triggeredBy: 'Policy schedule',
    waveDone: 44,
    waveActive: 11,
    wavePend: 34,
  },
  {
    id: 'DEP-0044',
    policy: 'Linux Security Baseline',
    status: 'completed',
    progress: 100,
    targets: { done: 63, total: 63 },
    waves: '1/1',
    started: '2026-03-12',
    duration: '38 min',
    triggeredBy: 'Policy schedule',
    waveDone: 63,
    waveActive: 0,
    wavePend: 0,
  },
  {
    id: 'DEP-0043',
    policy: 'Auto-Deploy Critical Patches',
    status: 'completed',
    progress: 100,
    targets: { done: 89, total: 89 },
    waves: '2/2',
    started: '2026-03-11',
    duration: '1h 12m',
    triggeredBy: 'Policy schedule',
    waveDone: 89,
    waveActive: 0,
    wavePend: 0,
  },
  {
    id: 'DEP-0042',
    policy: 'Dev Environment Updates',
    status: 'completed',
    progress: 100,
    targets: { done: 23, total: 23 },
    waves: '1/1',
    started: '2026-03-10',
    duration: '15 min',
    triggeredBy: 'J. Davis',
    waveDone: 23,
    waveActive: 0,
    wavePend: 0,
  },
  {
    id: 'DEP-0041',
    policy: 'Database Server Policy',
    status: 'failed',
    progress: 0,
    targets: { done: 0, total: 8 },
    waves: '0/1',
    started: '2026-03-12',
    duration: '3 min',
    triggeredBy: 'S. Williams',
    waveDone: 0,
    waveActive: 0,
    wavePend: 0,
    error: 'Connection timeout on all 8 endpoints. Database service unreachable during wave 1.',
  },
  {
    id: 'DEP-0040',
    policy: 'Network Device Hardening',
    status: 'failed',
    progress: 25,
    targets: { done: 2, total: 8 },
    waves: '0/2',
    started: '2026-03-10',
    duration: '8 min',
    triggeredBy: 'R. Kumar',
    waveDone: 2,
    waveActive: 0,
    wavePend: 0,
    error: '6 endpoints returned exit code 1603 during wave 1. SSH authentication failed.',
  },
  {
    id: 'DEP-0039',
    policy: 'Weekly Security Scan',
    status: 'pending',
    progress: 0,
    targets: { done: 0, total: 47 },
    waves: '0/3',
    started: '2026-03-12',
    duration: '—',
    triggeredBy: 'Policy schedule',
    waveDone: 0,
    waveActive: 0,
    wavePend: 47,
  },
  {
    id: 'DEP-0038',
    policy: 'Linux Security Baseline',
    status: 'pending',
    progress: 0,
    targets: { done: 0, total: 18 },
    waves: '0/1',
    started: '2026-03-12',
    duration: '—',
    triggeredBy: 'Approval pending',
    waveDone: 0,
    waveActive: 0,
    wavePend: 18,
  },
];

const STATUS_FILTERS = [
  { label: 'All', value: 'all', count: 47 },
  { label: 'Running', value: 'running', count: 3 },
  { label: 'Completed', value: 'completed', count: 38 },
  { label: 'Failed', value: 'failed', count: 4 },
  { label: 'Pending', value: 'pending', count: 2 },
] as const;

// ── Status helpers ────────────────────────────────────────────────────────────
const STATUS_COLOR: Record<DeploymentStatus, string> = {
  running: 'var(--color-cyan)',
  completed: 'var(--color-success)',
  failed: 'var(--color-danger)',
  pending: 'var(--color-warning)',
};

const STATUS_LABEL: Record<DeploymentStatus, string> = {
  running: 'Running',
  completed: 'Completed',
  failed: 'Failed',
  pending: 'Pending',
};

// ── Input style constants ─────────────────────────────────────────────────────
const INPUT_STYLE: React.CSSProperties = {
  width: '100%',
  padding: '7px 10px',
  borderRadius: 6,
  border: '1px solid var(--color-separator)',
  background: 'var(--color-card)',
  color: 'var(--color-foreground)',
  fontSize: 13,
  outline: 'none',
};

const LABEL_STYLE: React.CSSProperties = {
  fontSize: 11,
  fontWeight: 600,
  color: 'var(--color-muted)',
  textTransform: 'uppercase',
  letterSpacing: '0.05em',
  display: 'block',
  marginBottom: 6,
};

// ── Sub-components ────────────────────────────────────────────────────────────
function StatusDot({ status }: { status: DeploymentStatus }) {
  const color = STATUS_COLOR[status];
  const isRunning = status === 'running';

  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
      <div style={{ position: 'relative', width: 8, height: 8, flexShrink: 0 }}>
        {isRunning && (
          <div
            style={{
              position: 'absolute',
              inset: -3,
              borderRadius: '50%',
              border: `1.5px solid ${color}`,
              animation: 'pulse-ring 1.2s ease-out infinite',
            }}
          />
        )}
        <div
          style={{
            width: 8,
            height: 8,
            borderRadius: '50%',
            background: color,
            position: 'relative',
            zIndex: 1,
          }}
        />
      </div>
      <span style={{ fontSize: 12, color, fontWeight: 600 }}>{STATUS_LABEL[status]}</span>
    </div>
  );
}

function ProgressBar({
  progress,
  status,
  waveDone,
  waveActive,
  wavePend,
  total,
}: {
  progress: number;
  status: DeploymentStatus;
  waveDone: number;
  waveActive: number;
  wavePend: number;
  total: number;
}) {
  const donePct = total ? Math.round((waveDone / total) * 100) : 0;
  const activePct = total ? Math.round((waveActive / total) * 100) : 0;
  const failCount = total - waveDone - waveActive - wavePend;
  const failPct = total && failCount > 0 ? Math.round((failCount / total) * 100) : 0;
  const pendPct = total ? Math.round((wavePend / total) * 100) : 0;

  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 8, minWidth: 120 }}>
      <div
        style={{
          flex: 1,
          height: 6,
          borderRadius: 3,
          background: 'var(--color-separator)',
          overflow: 'hidden',
          display: 'flex',
        }}
      >
        <div
          style={{
            height: '100%',
            width: `${donePct}%`,
            background: 'var(--color-success)',
            transition: 'width 0.6s ease',
          }}
        />
        <div
          style={{
            height: '100%',
            width: `${activePct}%`,
            background: 'var(--color-cyan)',
            transition: 'width 0.6s ease',
            animation: status === 'running' ? 'prog-pulse 0.8s ease infinite' : 'none',
          }}
        />
        {failPct > 0 && (
          <div
            style={{
              height: '100%',
              width: `${failPct}%`,
              background: 'var(--color-danger)',
              transition: 'width 0.6s ease',
            }}
          />
        )}
        <div
          style={{
            height: '100%',
            width: `${pendPct}%`,
            background: 'color-mix(in srgb, var(--color-separator) 80%, transparent)',
            transition: 'width 0.6s ease',
          }}
        />
      </div>
      <span style={{ fontSize: 11, color: 'var(--color-muted)', minWidth: 28, textAlign: 'right' }}>
        {progress}%
      </span>
    </div>
  );
}

function FilterPill({
  label,
  count,
  active,
  onClick,
}: {
  label: string;
  count: number;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      onClick={onClick}
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 6,
        padding: '5px 12px',
        borderRadius: 20,
        border: active ? '1px solid var(--color-primary)' : '1px solid var(--color-separator)',
        background: active
          ? 'color-mix(in srgb, var(--color-primary) 12%, transparent)'
          : 'transparent',
        color: active ? 'var(--color-primary)' : 'var(--color-muted)',
        fontSize: 12,
        fontWeight: active ? 600 : 400,
        cursor: 'pointer',
        transition: 'all 0.15s ease',
      }}
    >
      {label}
      <span
        style={{
          fontSize: 10,
          fontWeight: 700,
          padding: '1px 5px',
          borderRadius: 10,
          background: active
            ? 'color-mix(in srgb, var(--color-primary) 20%, transparent)'
            : 'var(--color-separator)',
          color: active ? 'var(--color-primary)' : 'var(--color-foreground)',
        }}
      >
        {count}
      </span>
    </button>
  );
}

// ── Expanded row panel ────────────────────────────────────────────────────────
function ExpandedPanel({ dep }: { dep: Deployment }) {
  const total = dep.targets.total;
  const failCount = Math.max(0, total - dep.waveDone - dep.waveActive - dep.wavePend);

  return (
    <tr>
      <td
        colSpan={10}
        style={{
          padding: 0,
          borderBottom: '1px solid var(--color-separator)',
        }}
      >
        <div
          style={{
            padding: 16,
            background: 'color-mix(in srgb, var(--color-background) 60%, var(--color-card))',
          }}
        >
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 16 }}>
            {/* Wave Pipeline */}
            <div>
              <div
                style={{
                  fontSize: 11,
                  fontWeight: 600,
                  color: 'var(--color-muted)',
                  textTransform: 'uppercase',
                  letterSpacing: '0.06em',
                  marginBottom: 8,
                }}
              >
                Wave Pipeline
              </div>
              <div
                style={{
                  display: 'flex',
                  gap: 3,
                  alignItems: 'center',
                  marginBottom: 6,
                  height: 16,
                }}
              >
                {dep.waveDone > 0 && (
                  <div
                    style={{
                      flex: dep.waveDone,
                      height: 16,
                      borderRadius: 3,
                      background: 'var(--color-success)',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      fontSize: 9,
                      fontWeight: 600,
                      color: '#fff',
                      padding: '0 6px',
                      whiteSpace: 'nowrap',
                      minWidth: 4,
                    }}
                  >
                    W1
                  </div>
                )}
                {dep.waveActive > 0 && (
                  <div
                    style={{
                      flex: dep.waveActive,
                      height: 16,
                      borderRadius: 3,
                      background: 'var(--color-cyan)',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      fontSize: 9,
                      fontWeight: 600,
                      color: '#fff',
                      padding: '0 6px',
                      whiteSpace: 'nowrap',
                      minWidth: 4,
                    }}
                  >
                    W2
                  </div>
                )}
                {dep.wavePend > 0 && (
                  <div
                    style={{
                      flex: dep.wavePend,
                      height: 16,
                      borderRadius: 3,
                      background: 'var(--color-separator)',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      fontSize: 9,
                      fontWeight: 600,
                      color: 'var(--color-muted)',
                      padding: '0 6px',
                      whiteSpace: 'nowrap',
                      minWidth: 4,
                    }}
                  >
                    W3
                  </div>
                )}
              </div>
              <div style={{ fontSize: 10, color: 'var(--color-muted)' }}>
                {dep.waveDone > 0 && `${dep.waveDone} done`}
                {dep.waveActive > 0 && ` · ${dep.waveActive} active`}
                {dep.wavePend > 0 && ` · ${dep.wavePend} pending`}
              </div>
            </div>

            {/* Endpoint Results */}
            <div>
              <div
                style={{
                  fontSize: 11,
                  fontWeight: 600,
                  color: 'var(--color-muted)',
                  textTransform: 'uppercase',
                  letterSpacing: '0.06em',
                  marginBottom: 8,
                }}
              >
                Endpoint Results
              </div>
              <div
                style={{
                  height: 10,
                  borderRadius: 5,
                  overflow: 'hidden',
                  display: 'flex',
                  width: 200,
                  marginBottom: 6,
                }}
              >
                {dep.waveDone > 0 && (
                  <div
                    style={{
                      flex: dep.waveDone,
                      height: '100%',
                      background: 'var(--color-success)',
                    }}
                  />
                )}
                {dep.waveActive > 0 && (
                  <div
                    style={{
                      flex: dep.waveActive,
                      height: '100%',
                      background: 'var(--color-cyan)',
                    }}
                  />
                )}
                {failCount > 0 && (
                  <div
                    style={{ flex: failCount, height: '100%', background: 'var(--color-danger)' }}
                  />
                )}
                {dep.wavePend > 0 && (
                  <div
                    style={{
                      flex: dep.wavePend,
                      height: '100%',
                      background: 'var(--color-separator)',
                    }}
                  />
                )}
              </div>
              <div style={{ fontSize: 10, color: 'var(--color-muted)', lineHeight: 1.8 }}>
                {dep.waveDone > 0 && (
                  <span style={{ color: 'var(--color-success)' }}>✓ {dep.waveDone} succeeded</span>
                )}
                {dep.waveActive > 0 && (
                  <span style={{ color: 'var(--color-cyan)', marginLeft: 6 }}>
                    ⟳ {dep.waveActive} in progress
                  </span>
                )}
                {dep.wavePend > 0 && (
                  <span style={{ color: 'var(--color-warning)', marginLeft: 6 }}>
                    ◷ {dep.wavePend} pending
                  </span>
                )}
                {failCount > 0 && (
                  <span style={{ color: 'var(--color-danger)', marginLeft: 6 }}>
                    ✗ {failCount} failed
                  </span>
                )}
              </div>
            </div>

            {/* Actions */}
            <div>
              <div
                style={{
                  fontSize: 11,
                  fontWeight: 600,
                  color: 'var(--color-muted)',
                  textTransform: 'uppercase',
                  letterSpacing: '0.06em',
                  marginBottom: 8,
                }}
              >
                Actions
              </div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                {dep.status === 'running' && (
                  <button
                    onClick={() => alert('Cancel deployment?')}
                    style={{
                      padding: '4px 10px',
                      fontSize: 11,
                      fontWeight: 500,
                      borderRadius: 6,
                      border: '1px solid color-mix(in srgb, var(--color-danger) 30%, transparent)',
                      background: 'color-mix(in srgb, var(--color-danger) 12%, transparent)',
                      color: 'var(--color-danger)',
                      cursor: 'pointer',
                    }}
                  >
                    Cancel
                  </button>
                )}
                {dep.status === 'failed' && (
                  <button
                    onClick={() => alert(`Retrying failed endpoints...`)}
                    style={{
                      padding: '4px 10px',
                      fontSize: 11,
                      fontWeight: 500,
                      borderRadius: 6,
                      border: '1px solid var(--color-primary)',
                      background: 'color-mix(in srgb, var(--color-primary) 12%, transparent)',
                      color: 'var(--color-primary)',
                      cursor: 'pointer',
                    }}
                  >
                    Retry Failed ({failCount})
                  </button>
                )}
                {(dep.status === 'completed' || dep.status === 'failed') && (
                  <button
                    onClick={() => alert('Rolling back...')}
                    style={{
                      padding: '4px 10px',
                      fontSize: 11,
                      fontWeight: 500,
                      borderRadius: 6,
                      border: '1px solid var(--color-separator)',
                      background: 'transparent',
                      color: 'var(--color-foreground)',
                      cursor: 'pointer',
                    }}
                  >
                    Rollback
                  </button>
                )}
                <button
                  onClick={() => alert(`Viewing details for ${dep.id}`)}
                  style={{
                    padding: '4px 10px',
                    fontSize: 11,
                    fontWeight: 500,
                    borderRadius: 6,
                    border: 'none',
                    background: 'transparent',
                    color: 'var(--color-muted)',
                    cursor: 'pointer',
                    textAlign: 'left',
                  }}
                >
                  View Details →
                </button>
              </div>
              {dep.error && (
                <div
                  style={{
                    marginTop: 8,
                    background: 'color-mix(in srgb, var(--color-danger) 8%, transparent)',
                    border: '1px solid color-mix(in srgb, var(--color-danger) 25%, transparent)',
                    borderRadius: 4,
                    padding: '6px 10px',
                    fontSize: 11,
                    color: 'var(--color-danger)',
                    fontFamily: 'monospace',
                  }}
                >
                  {dep.error}
                </div>
              )}
            </div>
          </div>
        </div>
      </td>
    </tr>
  );
}

// ── Table ─────────────────────────────────────────────────────────────────────
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

function DeploymentTable({ deployments }: { deployments: Deployment[] }) {
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set());

  function toggleRow(id: string) {
    setExpandedRows((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  }

  return (
    <div style={{ overflowX: 'auto' }}>
      <table style={{ width: '100%', borderCollapse: 'collapse' }}>
        <thead>
          <tr>
            <th style={{ ...TH_STYLE, width: 28 }}></th>
            <th style={TH_STYLE}>ID</th>
            <th style={TH_STYLE}>Policy</th>
            <th style={TH_STYLE}>Status</th>
            <th style={{ ...TH_STYLE, minWidth: 140 }}>Progress</th>
            <th style={TH_STYLE}>Targets</th>
            <th style={TH_STYLE}>Waves</th>
            <th style={TH_STYLE}>Started</th>
            <th style={TH_STYLE}>Duration</th>
            <th style={TH_STYLE}>Triggered By</th>
          </tr>
        </thead>
        <tbody>
          {deployments.map((dep) => {
            const isExpanded = expandedRows.has(dep.id);
            const isFailed = dep.status === 'failed';
            return (
              <>
                <tr
                  key={dep.id}
                  style={{
                    cursor: 'pointer',
                    transition: 'background 0.1s',
                    background: isFailed
                      ? 'color-mix(in srgb, var(--color-danger) 4%, transparent)'
                      : 'transparent',
                    borderLeft: isFailed
                      ? '2px solid var(--color-danger)'
                      : '2px solid transparent',
                  }}
                  onMouseEnter={(e) =>
                    (e.currentTarget.style.background =
                      'color-mix(in srgb, var(--color-primary) 4%, transparent)')
                  }
                  onMouseLeave={(e) =>
                    (e.currentTarget.style.background = isFailed
                      ? 'color-mix(in srgb, var(--color-danger) 4%, transparent)'
                      : 'transparent')
                  }
                >
                  {/* Expand chevron */}
                  <td style={{ ...TD_STYLE, padding: '10px 6px 10px 10px', width: 28 }}>
                    <button
                      onClick={() => toggleRow(dep.id)}
                      style={{
                        background: 'transparent',
                        border: 'none',
                        color: 'var(--color-muted)',
                        cursor: 'pointer',
                        padding: '2px 4px',
                        borderRadius: 4,
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        transition: 'color 0.15s',
                      }}
                      onMouseEnter={(e) =>
                        (e.currentTarget.style.color = 'var(--color-foreground)')
                      }
                      onMouseLeave={(e) => (e.currentTarget.style.color = 'var(--color-muted)')}
                    >
                      {isExpanded ? <ChevronDown size={13} /> : <ChevronRight size={13} />}
                    </button>
                  </td>

                  <td
                    style={{
                      ...TD_STYLE,
                      fontFamily: 'monospace',
                      fontSize: 11,
                      color: 'var(--color-primary)',
                      fontWeight: 600,
                    }}
                  >
                    {dep.id}
                  </td>
                  <td style={{ ...TD_STYLE, maxWidth: 220 }}>
                    <span
                      style={{
                        display: 'block',
                        overflow: 'hidden',
                        textOverflow: 'ellipsis',
                        whiteSpace: 'nowrap',
                        fontWeight: 500,
                      }}
                    >
                      {dep.policy}
                    </span>
                  </td>
                  <td style={TD_STYLE}>
                    <StatusDot status={dep.status} />
                  </td>
                  <td style={TD_STYLE}>
                    <ProgressBar
                      progress={dep.progress}
                      status={dep.status}
                      waveDone={dep.waveDone}
                      waveActive={dep.waveActive}
                      wavePend={dep.wavePend}
                      total={dep.targets.total}
                    />
                  </td>
                  <td style={{ ...TD_STYLE, color: 'var(--color-muted)', whiteSpace: 'nowrap' }}>
                    <span style={{ color: 'var(--color-foreground)', fontWeight: 600 }}>
                      {dep.targets.done}
                    </span>
                    {' / '}
                    {dep.targets.total}
                  </td>
                  <td style={{ ...TD_STYLE, color: 'var(--color-muted)', textAlign: 'center' }}>
                    {dep.waves}
                  </td>
                  <td
                    style={{
                      ...TD_STYLE,
                      color: 'var(--color-muted)',
                      whiteSpace: 'nowrap',
                      fontSize: 11,
                    }}
                  >
                    {dep.started}
                  </td>
                  <td
                    style={{
                      ...TD_STYLE,
                      color: 'var(--color-muted)',
                      whiteSpace: 'nowrap',
                      fontSize: 11,
                    }}
                  >
                    {dep.duration}
                  </td>
                  <td style={{ ...TD_STYLE, fontSize: 11 }}>
                    <span
                      style={{
                        padding: '2px 8px',
                        borderRadius: 4,
                        background: dep.triggeredBy.startsWith('CVE')
                          ? 'color-mix(in srgb, var(--color-danger) 12%, transparent)'
                          : 'color-mix(in srgb, var(--color-separator) 60%, transparent)',
                        color: dep.triggeredBy.startsWith('CVE')
                          ? 'var(--color-danger)'
                          : 'var(--color-foreground)',
                        fontWeight: dep.triggeredBy.startsWith('CVE') ? 600 : 400,
                      }}
                    >
                      {dep.triggeredBy}
                    </span>
                  </td>
                </tr>

                {isExpanded && <ExpandedPanel key={`exp-${dep.id}`} dep={dep} />}
              </>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

// ── Create Deployment Modal ────────────────────────────────────────────────────
interface CreateModalProps {
  onClose: () => void;
}

function CreateDeploymentModal({ onClose }: CreateModalProps) {
  const [policy, setPolicy] = useState('');
  const [description, setDescription] = useState('');
  const [targets, setTargets] = useState(45);
  const [waves, setWaves] = useState(2);
  const [scheduleType, setScheduleType] = useState<'immediate' | 'scheduled'>('immediate');
  const [scheduledAt, setScheduledAt] = useState('');
  const [confirmStep, setConfirmStep] = useState(false);

  function handleDeploy() {
    if (!policy) {
      alert('Please select a policy.');
      return;
    }
    if (scheduleType === 'scheduled' && !scheduledAt) {
      alert('Please choose a scheduled date and time.');
      return;
    }
    if (!confirmStep) {
      setConfirmStep(true);
      return;
    }
    // Second click — confirm
    alert(
      `Deployment created!\nPolicy: ${policy}\nTargets: ${targets}\nWaves: ${waves}\nSchedule: ${scheduleType === 'immediate' ? 'Immediate' : scheduledAt}${description ? `\nNotes: ${description}` : ''}`,
    );
    onClose();
  }

  return (
    <>
      {/* Backdrop */}
      <div
        onClick={onClose}
        style={{
          position: 'fixed',
          inset: 0,
          background: 'rgba(0,0,0,0.6)',
          backdropFilter: 'blur(4px)',
          zIndex: 50,
        }}
      />

      {/* Modal card */}
      <div
        style={{
          position: 'fixed',
          top: '50%',
          left: '50%',
          transform: 'translate(-50%,-50%)',
          background: 'var(--color-card)',
          borderRadius: 12,
          padding: 24,
          width: 480,
          maxWidth: '90vw',
          zIndex: 51,
          border: '1px solid var(--color-separator)',
          boxShadow: '0 20px 25px -5px rgba(0,0,0,0.4)',
          maxHeight: '90vh',
          overflowY: 'auto',
        }}
      >
        {/* Modal header */}
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            marginBottom: 20,
          }}
        >
          <div style={{ fontSize: 16, fontWeight: 700, color: 'var(--color-foreground)' }}>
            Create Deployment
          </div>
          <button
            onClick={onClose}
            style={{
              background: 'transparent',
              border: 'none',
              color: 'var(--color-muted)',
              fontSize: 20,
              cursor: 'pointer',
              lineHeight: 1,
              padding: '0 4px',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            ×
          </button>
        </div>

        {/* Form fields */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
          {/* Policy dropdown */}
          <div>
            <label style={LABEL_STYLE}>Select Policy *</label>
            <select
              value={policy}
              onChange={(e) => setPolicy(e.target.value)}
              style={{
                ...INPUT_STYLE,
                colorScheme: 'dark',
              }}
            >
              <option value="">-- Choose a policy --</option>
              {POLICY_OPTIONS.map((p) => (
                <option key={p} value={p}>
                  {p}
                </option>
              ))}
            </select>
          </div>

          {/* Description */}
          <div>
            <label style={LABEL_STYLE}>Description (Optional)</label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Add notes or reason for this deployment..."
              rows={3}
              style={{
                ...INPUT_STYLE,
                resize: 'vertical',
                fontFamily: 'system-ui, -apple-system, sans-serif',
                minHeight: 60,
              }}
            />
          </div>

          {/* Target Endpoints */}
          <div>
            <label style={LABEL_STYLE}>Target Endpoints *</label>
            <input
              type="number"
              min={1}
              max={500}
              value={targets}
              onChange={(e) => setTargets(Number(e.target.value))}
              style={INPUT_STYLE}
            />
            <div style={{ fontSize: 11, color: 'var(--color-muted)', marginTop: 4 }}>
              Total endpoints to apply this deployment to
            </div>
          </div>

          {/* Number of Waves */}
          <div>
            <label style={LABEL_STYLE}>Number of Waves *</label>
            <input
              type="number"
              min={1}
              max={10}
              value={waves}
              onChange={(e) => setWaves(Number(e.target.value))}
              style={INPUT_STYLE}
            />
            <div style={{ fontSize: 11, color: 'var(--color-muted)', marginTop: 4 }}>
              Break deployment into waves for staged rollout (1–10)
            </div>
          </div>

          {/* Schedule */}
          <div>
            <label style={LABEL_STYLE}>Schedule *</label>
            <div style={{ display: 'flex', gap: 16 }}>
              {(['immediate', 'scheduled'] as const).map((opt) => (
                <label
                  key={opt}
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
                    name="scheduleType"
                    value={opt}
                    checked={scheduleType === opt}
                    onChange={() => {
                      setScheduleType(opt);
                      if (opt === 'immediate') setScheduledAt('');
                    }}
                    style={{ accentColor: 'var(--color-primary)' }}
                  />
                  {opt === 'immediate' ? 'Immediate' : 'Scheduled'}
                </label>
              ))}
            </div>
          </div>

          {/* Datetime picker — visible only when scheduled */}
          {scheduleType === 'scheduled' && (
            <div>
              <label style={LABEL_STYLE}>Deployment Date & Time</label>
              <input
                type="datetime-local"
                value={scheduledAt}
                onChange={(e) => setScheduledAt(e.target.value)}
                style={{ ...INPUT_STYLE, colorScheme: 'dark' }}
              />
            </div>
          )}

          {/* Confirmation warning */}
          {confirmStep && (
            <div
              style={{
                padding: '10px 14px',
                borderRadius: 6,
                background: 'color-mix(in srgb, var(--color-warning) 10%, transparent)',
                border: '1px solid color-mix(in srgb, var(--color-warning) 30%, transparent)',
                fontSize: 12,
                color: 'var(--color-warning)',
              }}
            >
              This will deploy <strong>{policy}</strong> to <strong>{targets}</strong> endpoints
              {scheduleType === 'immediate' ? ' immediately' : ` at ${scheduledAt}`}. Click the
              button again to confirm.
            </div>
          )}

          {/* Footer buttons */}
          <div style={{ display: 'flex', gap: 8, marginTop: 4 }}>
            <button
              onClick={handleDeploy}
              style={{
                flex: 1,
                padding: '8px 14px',
                borderRadius: 6,
                border: 'none',
                background: confirmStep ? 'var(--color-warning)' : 'var(--color-primary)',
                color: '#fff',
                fontSize: 13,
                fontWeight: 600,
                cursor: 'pointer',
                transition: 'background 0.15s',
              }}
            >
              {confirmStep
                ? 'Confirm & Deploy'
                : scheduleType === 'scheduled'
                  ? 'Schedule'
                  : 'Deploy Now'}
            </button>
            <button
              onClick={onClose}
              style={{
                flex: 1,
                padding: '8px 14px',
                borderRadius: 6,
                border: '1px solid var(--color-separator)',
                background: 'transparent',
                color: 'var(--color-foreground)',
                fontSize: 13,
                fontWeight: 500,
                cursor: 'pointer',
              }}
            >
              Cancel
            </button>
          </div>
        </div>
      </div>
    </>
  );
}

// ── Page ──────────────────────────────────────────────────────────────────────
export default function Deployments() {
  useHotkeys();
  const [activeFilter, setActiveFilter] = useState<string>('all');
  const [searchQuery, setSearchQuery] = useState('');
  const [dateFrom, setDateFrom] = useState('');
  const [dateTo, setDateTo] = useState('');
  const [showCreateModal, setShowCreateModal] = useState(false);

  // Apply status filter
  const statusFiltered =
    activeFilter === 'all' ? DEPLOYMENTS : DEPLOYMENTS.filter((d) => d.status === activeFilter);

  // Apply search filter
  const searchFiltered = searchQuery.trim()
    ? statusFiltered.filter(
        (d) =>
          d.id.toLowerCase().includes(searchQuery.toLowerCase()) ||
          d.policy.toLowerCase().includes(searchQuery.toLowerCase()),
      )
    : statusFiltered;

  // Apply date range filter (match against `started` field — stored as YYYY-MM-DD strings)
  const filteredDeployments = searchFiltered.filter((d) => {
    const started = d.started.slice(0, 10); // normalise to YYYY-MM-DD
    if (dateFrom && started < dateFrom) return false;
    if (dateTo && started > dateTo) return false;
    return true;
  });

  const kpiCards: Array<{
    icon: LucideIcon;
    iconColor: string;
    value: number;
    label: string;
    valueColor: string;
  }> = [
    {
      icon: Play,
      iconColor: 'var(--color-cyan)',
      value: 3,
      label: 'Running',
      valueColor: 'var(--color-cyan)',
    },
    {
      icon: CheckCircle,
      iconColor: 'var(--color-success)',
      value: 38,
      label: 'Completed',
      valueColor: 'var(--color-success)',
    },
    {
      icon: XCircle,
      iconColor: 'var(--color-danger)',
      value: 4,
      label: 'Failed',
      valueColor: 'var(--color-danger)',
    },
    {
      icon: Clock,
      iconColor: 'var(--color-warning)',
      value: 2,
      label: 'Pending',
      valueColor: 'var(--color-warning)',
    },
  ];

  return (
    <>
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
            title="Deployments"
            subtitle="47 total"
            actions={
              <button
                onClick={() => setShowCreateModal(true)}
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
                Create Deployment
              </button>
            }
          />
        </motion.div>

        {/* Row 2 — Search + date range filters */}
        <motion.div variants={fadeUp}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap' }}>
            {/* Search */}
            <div style={{ position: 'relative', flex: '1 1 180px', maxWidth: 280 }}>
              <Search
                size={13}
                style={{
                  position: 'absolute',
                  left: 9,
                  top: '50%',
                  transform: 'translateY(-50%)',
                  color: 'var(--color-muted)',
                  pointerEvents: 'none',
                }}
              />
              <input
                type="text"
                placeholder="Search by ID or policy..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                style={{
                  width: '100%',
                  padding: '6px 10px 6px 30px',
                  borderRadius: 6,
                  border: '1px solid var(--color-separator)',
                  background: 'color-mix(in srgb, var(--color-card) 80%, transparent)',
                  color: 'var(--color-foreground)',
                  fontSize: 12,
                  outline: 'none',
                  transition: 'border-color 0.15s',
                }}
                onFocus={(e) => (e.target.style.borderColor = 'var(--color-primary)')}
                onBlur={(e) => (e.target.style.borderColor = 'var(--color-separator)')}
              />
            </div>

            {/* Date range */}
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 6,
                fontSize: 11,
                color: 'var(--color-muted)',
              }}
            >
              <Calendar size={13} style={{ flexShrink: 0 }} />
              <input
                type="date"
                value={dateFrom}
                onChange={(e) => setDateFrom(e.target.value)}
                style={{
                  background: 'color-mix(in srgb, var(--color-card) 80%, transparent)',
                  border: '1px solid var(--color-separator)',
                  borderRadius: 4,
                  padding: '4px 8px',
                  color: 'var(--color-foreground)',
                  fontSize: 11,
                  outline: 'none',
                  colorScheme: 'dark',
                }}
              />
              <span>—</span>
              <input
                type="date"
                value={dateTo}
                onChange={(e) => setDateTo(e.target.value)}
                style={{
                  background: 'color-mix(in srgb, var(--color-card) 80%, transparent)',
                  border: '1px solid var(--color-separator)',
                  borderRadius: 4,
                  padding: '4px 8px',
                  color: 'var(--color-foreground)',
                  fontSize: 11,
                  outline: 'none',
                  colorScheme: 'dark',
                }}
              />
            </div>
          </div>
        </motion.div>

        {/* Row 3 — KPI stat cards */}
        <motion.div
          variants={stagger}
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(4, 1fr)',
            gap: 12,
          }}
        >
          {kpiCards.map((card) => (
            <motion.div key={card.label} variants={fadeUp}>
              <StatCard
                icon={<card.icon size={16} />}
                iconColor={card.iconColor}
                value={card.value}
                valueColor={card.valueColor}
                label={card.label}
              />
            </motion.div>
          ))}
        </motion.div>

        {/* Row 4 — Filter pills + table */}
        <motion.div variants={fadeUp}>
          <GlassCard className="p-5" hover={false}>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
              {/* Header row */}
              <SectionHeader
                title="All Deployments"
                action={
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    {STATUS_FILTERS.map((f) => (
                      <FilterPill
                        key={f.value}
                        label={f.label}
                        count={f.count}
                        active={activeFilter === f.value}
                        onClick={() => setActiveFilter(f.value)}
                      />
                    ))}
                  </div>
                }
              />

              {/* Table */}
              <DeploymentTable deployments={filteredDeployments} />

              {filteredDeployments.length === 0 && (
                <div
                  style={{
                    textAlign: 'center',
                    padding: '32px 0',
                    color: 'var(--color-muted)',
                    fontSize: 13,
                  }}
                >
                  No deployments match the current filters.
                </div>
              )}
            </div>
          </GlassCard>
        </motion.div>
      </motion.div>

      {/* Create Deployment Modal */}
      {showCreateModal && <CreateDeploymentModal onClose={() => setShowCreateModal(false)} />}
    </>
  );
}
