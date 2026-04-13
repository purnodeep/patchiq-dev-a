import { WORKFLOW_PIPELINE_DATA, type WorkflowData, type WorkflowStage } from '@/data/mock-data';

// ── Helpers ────────────────────────────────────────────────────────────────────

type OverallStatus = 'complete' | 'running' | 'failed' | 'pending';

function getOverallStatus(stages: WorkflowStage[]): OverallStatus {
  if (stages.some((s) => s.status === 'failed')) return 'failed';
  if (stages.every((s) => s.status === 'complete')) return 'complete';
  if (stages.some((s) => s.status === 'running')) return 'running';
  return 'pending';
}

function getActiveStage(stages: WorkflowStage[]): WorkflowStage {
  return (
    stages.find((s) => s.status === 'failed') ??
    stages.find((s) => s.status === 'running') ??
    stages.find((s) => s.status === 'pending') ??
    stages[stages.length - 1]
  );
}

function getProgress(stages: WorkflowStage[]) {
  const done = stages.filter((s) => s.status === 'complete').length;
  return { done, total: stages.length };
}

const STATUS_COLOR: Record<OverallStatus, string> = {
  complete: 'var(--color-success)',
  running: 'var(--color-primary)',
  failed: 'var(--color-danger)',
  pending: 'var(--color-muted)',
};

// ── Status Icon ────────────────────────────────────────────────────────────────

function StatusIcon({ status }: { status: OverallStatus }) {
  const color = STATUS_COLOR[status];
  const size = 14;

  if (status === 'complete') {
    return (
      <svg width={size} height={size} viewBox="0 0 14 14" fill="none" style={{ flexShrink: 0 }}>
        <circle cx={7} cy={7} r={6.5} fill={color} />
        <path
          d="M4 7.5L6 9.5L10 5"
          stroke="#fff"
          strokeWidth={1.6}
          strokeLinecap="round"
          strokeLinejoin="round"
        />
      </svg>
    );
  }
  if (status === 'running') {
    return (
      <svg width={size} height={size} viewBox="0 0 14 14" fill="none" style={{ flexShrink: 0 }}>
        <circle cx={7} cy={7} r={6.5} stroke={color} strokeWidth={1.5} fill="transparent" />
        <circle cx={7} cy={7} r={3} fill={color} />
      </svg>
    );
  }
  if (status === 'failed') {
    return (
      <svg width={size} height={size} viewBox="0 0 14 14" fill="none" style={{ flexShrink: 0 }}>
        <circle cx={7} cy={7} r={6.5} fill={color} />
        <path
          d="M4.5 4.5L9.5 9.5M9.5 4.5L4.5 9.5"
          stroke="#fff"
          strokeWidth={1.6}
          strokeLinecap="round"
        />
      </svg>
    );
  }
  // pending
  return (
    <svg width={size} height={size} viewBox="0 0 14 14" fill="none" style={{ flexShrink: 0 }}>
      <circle
        cx={7}
        cy={7}
        r={6.5}
        stroke={color}
        strokeWidth={1.5}
        strokeDasharray="3 2"
        fill="transparent"
      />
    </svg>
  );
}

// ── Progress Bar ───────────────────────────────────────────────────────────────

function ProgressBar({
  done,
  total,
  status,
}: {
  done: number;
  total: number;
  status: OverallStatus;
}) {
  const pct = total === 0 ? 0 : Math.round((done / total) * 100);
  const color = STATUS_COLOR[status];

  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
      <div
        style={{
          flex: 1,
          height: 5,
          borderRadius: 3,
          background: 'var(--color-separator)',
          overflow: 'hidden',
          minWidth: 60,
        }}
      >
        <div
          style={{
            height: '100%',
            width: `${pct}%`,
            background: color,
            borderRadius: 3,
            transition: 'width 0.4s ease',
          }}
        />
      </div>
      <span
        style={{
          fontSize: 10,
          fontWeight: 600,
          color,
          width: 28,
          textAlign: 'right',
          flexShrink: 0,
        }}
      >
        {pct}%
      </span>
    </div>
  );
}

// ── Mini Stage Track ───────────────────────────────────────────────────────────

function MiniStageTrack({ stages }: { stages: WorkflowStage[] }) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 2 }}>
      {stages.map((s, i) => {
        let bg = 'var(--color-separator)';
        if (s.status === 'complete') bg = 'var(--color-success)';
        else if (s.status === 'running') bg = 'var(--color-primary)';
        else if (s.status === 'failed') bg = 'var(--color-danger)';

        return (
          <div
            key={i}
            title={`${s.name}: ${s.status}`}
            style={{
              width: 18,
              height: 5,
              borderRadius: 2,
              background: bg,
              opacity: s.status === 'pending' ? 0.4 : 1,
            }}
          />
        );
      })}
    </div>
  );
}

// ── Row ────────────────────────────────────────────────────────────────────────

function WorkflowRow({ workflow, animDelay }: { workflow: WorkflowData; animDelay: number }) {
  const status = getOverallStatus(workflow.stages);
  const active = getActiveStage(workflow.stages);
  const { done, total } = getProgress(workflow.stages);
  const color = STATUS_COLOR[status];

  return (
    <div
      style={{
        display: 'grid',
        gridTemplateColumns: '1fr auto auto auto',
        alignItems: 'center',
        gap: 12,
        padding: '9px 0',
        borderBottom: '1px solid var(--color-separator)',
        animation: 'fade-in-up 0.4s ease both',
        animationDelay: `${animDelay}ms`,
      }}
    >
      {/* Col 1 — Workflow name + stage track */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 4, minWidth: 0 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
          <StatusIcon status={status} />
          <span
            style={{
              fontSize: 11,
              fontWeight: 600,
              color: 'var(--color-foreground)',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
            }}
          >
            {workflow.name}
          </span>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, paddingLeft: 20 }}>
          <MiniStageTrack stages={workflow.stages} />
          <span style={{ fontSize: 9, color: 'var(--color-muted)' }}>{workflow.startedAgo}</span>
        </div>
      </div>

      {/* Col 2 — Active stage */}
      <div style={{ textAlign: 'right', flexShrink: 0 }}>
        <span
          style={{
            fontSize: 10,
            fontWeight: 600,
            color,
            background: `${color}18`,
            padding: '2px 6px',
            borderRadius: 4,
            whiteSpace: 'nowrap',
          }}
        >
          {status === 'failed'
            ? `✗ ${active.name}`
            : status === 'complete'
              ? `✓ Verified`
              : `▶ ${active.name}`}
        </span>
      </div>

      {/* Col 3 — Hosts */}
      <div style={{ textAlign: 'right', flexShrink: 0, minWidth: 64 }}>
        <span style={{ fontSize: 10, fontWeight: 600, color: 'var(--color-foreground)' }}>
          {workflow.hostsDone.toLocaleString()}
        </span>
        <span style={{ fontSize: 10, color: 'var(--color-muted)' }}>
          {' '}
          / {workflow.hostsTotal.toLocaleString()}
        </span>
      </div>

      {/* Col 4 — Progress bar */}
      <div style={{ width: 100, flexShrink: 0 }}>
        <ProgressBar done={done} total={total} status={status} />
      </div>
    </div>
  );
}

// ── Main Component ─────────────────────────────────────────────────────────────

export function WorkflowPipeline() {
  return (
    <div style={{ display: 'flex', flexDirection: 'column' }}>
      {/* Column headers */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: '1fr auto auto auto',
          gap: 12,
          paddingBottom: 6,
          borderBottom: '1px solid var(--color-separator)',
        }}
      >
        {['Workflow', 'Stage', 'Hosts', 'Progress'].map((h, i) => (
          <span
            key={h}
            style={{
              fontSize: 9,
              fontWeight: 700,
              color: 'var(--color-muted)',
              textTransform: 'uppercase',
              letterSpacing: '0.06em',
              textAlign: i === 0 ? 'left' : 'right',
              minWidth: i === 3 ? 100 : i === 2 ? 64 : 'auto',
            }}
          >
            {h}
          </span>
        ))}
      </div>

      {/* Rows */}
      {WORKFLOW_PIPELINE_DATA.map((wf, idx) => (
        <WorkflowRow key={wf.id} workflow={wf} animDelay={idx * 80} />
      ))}
    </div>
  );
}
