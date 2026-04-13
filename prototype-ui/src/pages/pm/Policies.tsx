import { useState } from 'react';
import { motion } from 'framer-motion';
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

// ── Types & mock data ─────────────────────────────────────────────────────────
type PolicyMode = 'automatic' | 'manual' | 'advisory';
type EvalResult = 'pass' | 'fail';

interface Policy {
  name: string;
  mode: PolicyMode;
  schedule: string;
  nextRun: string;
  groups: string[];
  enabled: boolean;
  endpoints: number;
  lastEval: EvalResult;
  evalTime: string;
  lastRun: string;
}

const POLICIES: Policy[] = [
  {
    name: 'Auto-Deploy Critical Patches',
    mode: 'automatic',
    schedule: 'Daily 2:00 AM',
    nextRun: '18h 24m',
    groups: ['Production', 'Critical'],
    enabled: true,
    endpoints: 89,
    lastEval: 'pass',
    evalTime: '2h ago',
    lastRun: '1d ago',
  },
  {
    name: 'Weekly Security Scan',
    mode: 'manual',
    schedule: 'Sundays 1:00 AM',
    nextRun: '3d 4h',
    groups: ['All'],
    enabled: true,
    endpoints: 247,
    lastEval: 'fail',
    evalTime: '6h ago',
    lastRun: '7d ago',
  },
  {
    name: 'Emergency Critical Fast Track',
    mode: 'automatic',
    schedule: 'Event-triggered',
    nextRun: 'On CVE ≥9.0',
    groups: ['Production', 'Network'],
    enabled: true,
    endpoints: 45,
    lastEval: 'pass',
    evalTime: '1h ago',
    lastRun: '3d ago',
  },
  {
    name: 'Dev Environment Updates',
    mode: 'manual',
    schedule: 'Fridays 6:00 PM',
    nextRun: '2d 8h',
    groups: ['Dev'],
    enabled: true,
    endpoints: 23,
    lastEval: 'pass',
    evalTime: '1d ago',
    lastRun: '7d ago',
  },
  {
    name: 'Database Server Policy',
    mode: 'manual',
    schedule: 'Manual only',
    nextRun: '—',
    groups: ['Database', 'Critical'],
    enabled: false,
    endpoints: 8,
    lastEval: 'pass',
    evalTime: '3d ago',
    lastRun: '14d ago',
  },
  {
    name: 'Linux Security Baseline',
    mode: 'automatic',
    schedule: 'Tuesdays 3:00 AM',
    nextRun: '4d 21h',
    groups: ['Linux', 'Infrastructure'],
    enabled: true,
    endpoints: 63,
    lastEval: 'pass',
    evalTime: '4h ago',
    lastRun: '7d ago',
  },
  {
    name: 'Compliance Audit Policy',
    mode: 'advisory',
    schedule: 'Monthly',
    nextRun: '18d',
    groups: ['All'],
    enabled: true,
    endpoints: 247,
    lastEval: 'fail',
    evalTime: '2d ago',
    lastRun: '30d ago',
  },
  {
    name: 'Network Device Hardening',
    mode: 'manual',
    schedule: 'Manual only',
    nextRun: '—',
    groups: ['Network'],
    enabled: false,
    endpoints: 15,
    lastEval: 'pass',
    evalTime: '5d ago',
    lastRun: '21d ago',
  },
];

// ── Helpers ───────────────────────────────────────────────────────────────────
type ModeFilterKey = 'all' | PolicyMode;
type StatusFilterKey = 'all' | 'enabled' | 'disabled';

function modeColor(mode: PolicyMode): string {
  switch (mode) {
    case 'automatic':
      return 'var(--color-success)';
    case 'manual':
      return 'var(--color-primary)';
    case 'advisory':
      return 'var(--color-muted)';
  }
}

// ── Sub-components ────────────────────────────────────────────────────────────
function ModeBadge({ mode }: { mode: PolicyMode }) {
  const color = modeColor(mode);
  return (
    <span
      style={{
        display: 'inline-block',
        padding: '2px 8px',
        borderRadius: 4,
        fontSize: 10,
        fontWeight: 700,
        letterSpacing: '0.04em',
        textTransform: 'capitalize',
        color,
        background: `color-mix(in srgb, ${color} 12%, transparent)`,
        border: `1px solid color-mix(in srgb, ${color} 28%, transparent)`,
      }}
    >
      {mode}
    </span>
  );
}

function EnabledCell({ enabled }: { enabled: boolean }) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
      <div
        style={{
          width: 7,
          height: 7,
          borderRadius: '50%',
          background: enabled ? 'var(--color-success)' : 'var(--color-muted)',
          flexShrink: 0,
        }}
      />
      <span
        style={{
          fontSize: 11,
          fontWeight: 600,
          color: enabled ? 'var(--color-success)' : 'var(--color-muted)',
        }}
      >
        {enabled ? 'Enabled' : 'Disabled'}
      </span>
    </div>
  );
}

function EvalCell({ result, time }: { result: EvalResult; time: string }) {
  const pass = result === 'pass';
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      <span
        style={{
          fontSize: 11,
          fontWeight: 700,
          color: pass ? 'var(--color-success)' : 'var(--color-danger)',
        }}
      >
        {pass ? '✓ Pass' : '✗ Fail'}
      </span>
      <span style={{ fontSize: 10, color: 'var(--color-muted)' }}>{time}</span>
    </div>
  );
}

function GroupTags({ groups }: { groups: string[] }) {
  return (
    <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap' }}>
      {groups.map((g) => (
        <span
          key={g}
          style={{
            padding: '1px 6px',
            borderRadius: 3,
            fontSize: 9,
            fontWeight: 600,
            color: 'var(--color-muted)',
            background: 'color-mix(in srgb, var(--color-separator) 80%, transparent)',
            border: '1px solid var(--color-separator)',
            whiteSpace: 'nowrap',
          }}
        >
          {g}
        </span>
      ))}
    </div>
  );
}

// ── Filter pill helper ────────────────────────────────────────────────────────
function FilterPill<T extends string>({
  value,
  label,
  count,
  active,
  onClick,
}: {
  value: T;
  label: string;
  count: number;
  active: boolean;
  onClick: (v: T) => void;
}) {
  return (
    <button
      onClick={() => onClick(value)}
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 5,
        padding: '4px 10px',
        borderRadius: 20,
        border: active ? '1px solid var(--color-primary)' : '1px solid var(--color-separator)',
        background: active
          ? 'color-mix(in srgb, var(--color-primary) 15%, transparent)'
          : 'transparent',
        color: active ? 'var(--color-primary)' : 'var(--color-muted)',
        fontSize: 11,
        fontWeight: active ? 700 : 500,
        cursor: 'pointer',
        transition: 'all 0.15s',
      }}
    >
      {label}
      <span style={{ fontSize: 10, opacity: 0.7 }}>{count}</span>
    </button>
  );
}

// ── Table ─────────────────────────────────────────────────────────────────────
const TH_STYLE: React.CSSProperties = {
  padding: '8px 10px',
  textAlign: 'left',
  fontSize: 10,
  fontWeight: 700,
  letterSpacing: '0.06em',
  textTransform: 'uppercase',
  color: 'var(--color-muted)',
  borderBottom: '1px solid var(--color-separator)',
  whiteSpace: 'nowrap',
};

const TD_STYLE: React.CSSProperties = {
  padding: '11px 10px',
  fontSize: 12,
  borderBottom: '1px solid color-mix(in srgb, var(--color-separator) 50%, transparent)',
  verticalAlign: 'middle',
};

function PoliciesTable({ rows }: { rows: Policy[] }) {
  return (
    <div style={{ overflowX: 'auto' }}>
      <table style={{ width: '100%', borderCollapse: 'collapse' }}>
        <thead>
          <tr>
            <th style={TH_STYLE}>Policy Name</th>
            <th style={TH_STYLE}>Mode</th>
            <th style={TH_STYLE}>Schedule</th>
            <th style={TH_STYLE}>Groups</th>
            <th style={TH_STYLE}>Enabled</th>
            <th style={{ ...TH_STYLE, textAlign: 'right' }}>Endpoints</th>
            <th style={TH_STYLE}>Last Eval</th>
            <th style={TH_STYLE}>Last Run</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((policy, i) => (
            <tr
              key={policy.name}
              style={{
                background:
                  i % 2 === 0
                    ? 'transparent'
                    : 'color-mix(in srgb, var(--color-separator) 20%, transparent)',
                transition: 'background 0.15s',
                opacity: policy.enabled ? 1 : 0.6,
              }}
            >
              <td style={TD_STYLE}>
                <div>
                  <div style={{ fontWeight: 600, fontSize: 12, color: 'var(--color-foreground)' }}>
                    {policy.name}
                  </div>
                  <div style={{ fontSize: 10, color: 'var(--color-muted)', marginTop: 2 }}>
                    Next: {policy.nextRun}
                  </div>
                </div>
              </td>
              <td style={TD_STYLE}>
                <ModeBadge mode={policy.mode} />
              </td>
              <td
                style={{
                  ...TD_STYLE,
                  color: 'var(--color-muted)',
                  fontSize: 11,
                  whiteSpace: 'nowrap',
                }}
              >
                {policy.schedule}
              </td>
              <td style={TD_STYLE}>
                <GroupTags groups={policy.groups} />
              </td>
              <td style={TD_STYLE}>
                <EnabledCell enabled={policy.enabled} />
              </td>
              <td style={{ ...TD_STYLE, textAlign: 'right', fontWeight: 700 }}>
                {policy.endpoints}
              </td>
              <td style={TD_STYLE}>
                <EvalCell result={policy.lastEval} time={policy.evalTime} />
              </td>
              <td
                style={{
                  ...TD_STYLE,
                  color: 'var(--color-muted)',
                  whiteSpace: 'nowrap',
                  fontSize: 11,
                }}
              >
                {policy.lastRun}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

// ── Page ──────────────────────────────────────────────────────────────────────
export default function Policies() {
  useHotkeys();

  const [modeFilter, setModeFilter] = useState<ModeFilterKey>('all');
  const [statusFilter, setStatusFilter] = useState<StatusFilterKey>('all');

  const filteredPolicies = POLICIES.filter((p) => {
    if (modeFilter !== 'all' && p.mode !== modeFilter) return false;
    if (statusFilter === 'enabled' && !p.enabled) return false;
    if (statusFilter === 'disabled' && p.enabled) return false;
    return true;
  });

  const modeCounts: Record<ModeFilterKey, number> = {
    all: POLICIES.length,
    automatic: POLICIES.filter((p) => p.mode === 'automatic').length,
    manual: POLICIES.filter((p) => p.mode === 'manual').length,
    advisory: POLICIES.filter((p) => p.mode === 'advisory').length,
  };

  const statusCounts: Record<StatusFilterKey, number> = {
    all: POLICIES.length,
    enabled: POLICIES.filter((p) => p.enabled).length,
    disabled: POLICIES.filter((p) => !p.enabled).length,
  };

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
      <motion.div
        variants={fadeUp}
        style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}
      >
        <div>
          <h1 style={{ fontSize: 20, fontWeight: 800, letterSpacing: '-0.02em', margin: 0 }}>
            Policies
          </h1>
          <p style={{ fontSize: 12, color: 'var(--color-muted)', margin: '3px 0 0' }}>
            8 policies · 6 enabled · 2 disabled
          </p>
        </div>
        <button
          style={{
            padding: '7px 14px',
            borderRadius: 7,
            border: 'none',
            background: 'var(--color-primary)',
            color: '#fff',
            fontSize: 12,
            fontWeight: 600,
            cursor: 'pointer',
            letterSpacing: '0.01em',
          }}
        >
          + Create Policy
        </button>
      </motion.div>

      {/* Row 2 — Filters + table */}
      <motion.div variants={fadeUp}>
        <GlassCard className="p-5" hover={false}>
          <SectionHeader
            title="Policy Registry"
            action={
              <span style={{ fontSize: 11, color: 'var(--color-muted)' }}>
                {filteredPolicies.length} policies
              </span>
            }
          />

          {/* Filter pills */}
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 12,
              marginTop: 14,
              flexWrap: 'wrap',
            }}
          >
            {/* Mode filters */}
            <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
              <span
                style={{
                  fontSize: 10,
                  fontWeight: 600,
                  color: 'var(--color-muted)',
                  letterSpacing: '0.05em',
                  textTransform: 'uppercase',
                }}
              >
                Mode
              </span>
              {(
                [
                  { value: 'all' as ModeFilterKey, label: 'All', count: modeCounts.all },
                  {
                    value: 'automatic' as ModeFilterKey,
                    label: 'Automatic',
                    count: modeCounts.automatic,
                  },
                  { value: 'manual' as ModeFilterKey, label: 'Manual', count: modeCounts.manual },
                  {
                    value: 'advisory' as ModeFilterKey,
                    label: 'Advisory',
                    count: modeCounts.advisory,
                  },
                ] as const
              ).map((pill) => (
                <FilterPill
                  key={pill.value}
                  value={pill.value}
                  label={pill.label}
                  count={pill.count}
                  active={modeFilter === pill.value}
                  onClick={setModeFilter}
                />
              ))}
            </div>

            {/* Divider */}
            <div
              style={{
                width: 1,
                height: 18,
                background: 'var(--color-separator)',
              }}
            />

            {/* Status filters */}
            <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
              <span
                style={{
                  fontSize: 10,
                  fontWeight: 600,
                  color: 'var(--color-muted)',
                  letterSpacing: '0.05em',
                  textTransform: 'uppercase',
                }}
              >
                Status
              </span>
              {(
                [
                  { value: 'all' as StatusFilterKey, label: 'All', count: statusCounts.all },
                  {
                    value: 'enabled' as StatusFilterKey,
                    label: 'Enabled',
                    count: statusCounts.enabled,
                  },
                  {
                    value: 'disabled' as StatusFilterKey,
                    label: 'Disabled',
                    count: statusCounts.disabled,
                  },
                ] as const
              ).map((pill) => (
                <FilterPill
                  key={pill.value}
                  value={pill.value}
                  label={pill.label}
                  count={pill.count}
                  active={statusFilter === pill.value}
                  onClick={setStatusFilter}
                />
              ))}
            </div>
          </div>

          {/* Table */}
          <div style={{ marginTop: 16 }}>
            <PoliciesTable rows={filteredPolicies} />
          </div>
        </GlassCard>
      </motion.div>
    </motion.div>
  );
}
