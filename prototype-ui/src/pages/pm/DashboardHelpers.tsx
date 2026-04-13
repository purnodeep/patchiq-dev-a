import {
  Monitor,
  AlertTriangle,
  CheckCircle,
  Rocket,
  Clock,
  XCircle,
  GitBranch,
  RefreshCw,
  type LucideIcon,
} from 'lucide-react';
import type { MicroVizType } from '@/data/mock-data';

// ── Icon lookup ──────────────────────────────────────────────────────────────
const ICON_MAP: Record<string, LucideIcon> = {
  Monitor,
  AlertTriangle,
  CheckCircle,
  Rocket,
  Clock,
  XCircle,
  GitBranch,
  RefreshCw,
};

export function resolveIcon(name: string, color: string) {
  const Icon = ICON_MAP[name] ?? Monitor;
  return <Icon size={16} color={color} />;
}

// ── Micro-visualization components ──────────────────────────────────────────

function RingGauge({
  current,
  total,
  offline,
  pending,
}: {
  current: number;
  total: number;
  offline: number;
  pending: number;
}) {
  const r = 22;
  const circumference = 2 * Math.PI * r;
  const pct = Math.min(current / total, 1);
  const filled = pct * circumference;
  const gap = circumference - filled;
  return (
    <div
      style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 6, width: 80 }}
    >
      <svg width={52} height={52} viewBox="0 0 52 52">
        <circle cx={26} cy={26} r={r} fill="none" stroke="var(--color-separator)" strokeWidth={4} />
        <circle
          cx={26}
          cy={26}
          r={r}
          fill="none"
          stroke="var(--color-primary)"
          strokeWidth={4}
          strokeLinecap="round"
          strokeDasharray={`${filled} ${gap}`}
          strokeDashoffset={circumference * 0.25}
          style={{ animation: 'sweep 0.8s ease-out both' }}
        />
        <text
          x={26}
          y={30}
          textAnchor="middle"
          fontSize={10}
          fontWeight={700}
          fill="var(--color-primary)"
        >
          {Math.round(pct * 100)}%
        </text>
      </svg>
      <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 2 }}>
        <span style={{ fontSize: 9, fontWeight: 600, color: 'var(--color-danger)' }}>
          {offline} offline
        </span>
        <span style={{ fontSize: 9, fontWeight: 600, color: 'var(--color-warning)' }}>
          {pending} pending
        </span>
      </div>
    </div>
  );
}

function SeverityBars({
  critical,
  high,
  medium,
  low,
}: {
  critical: number;
  high: number;
  medium: number;
  low: number;
}) {
  const total = critical + high + medium + low;
  const pct = (n: number) => `${((n / total) * 100).toFixed(1)}%`;
  const rows = [
    { label: 'C', val: critical, color: 'var(--color-danger)' },
    { label: 'H', val: high, color: 'var(--color-warning)' },
    { label: 'M', val: medium, color: 'var(--color-caution)' },
    { label: 'L', val: low, color: 'var(--color-success)' },
  ];
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 4, width: 80 }}>
      {rows.map(({ label, val, color }) => (
        <div key={label} style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
          <span style={{ fontSize: 8, color: 'var(--color-muted)', width: 7, flexShrink: 0 }}>
            {label}
          </span>
          <div
            style={{
              height: 5,
              flex: 1,
              borderRadius: 3,
              background: 'var(--color-separator)',
              overflow: 'hidden',
            }}
          >
            <div
              style={{
                height: '100%',
                width: pct(val),
                borderRadius: 3,
                background: color,
                transition: 'width 0.6s ease',
              }}
            />
          </div>
          <span
            style={{
              fontSize: 8,
              fontWeight: 600,
              color,
              width: 14,
              textAlign: 'right',
              flexShrink: 0,
            }}
          >
            {val}
          </span>
        </div>
      ))}
      <div style={{ textAlign: 'right', marginTop: 1 }}>
        <span style={{ fontSize: 8, color: 'var(--color-muted)' }}>{total} total</span>
      </div>
    </div>
  );
}

function GradientRing({
  percentage,
  nist,
  pci,
  hipaa,
}: {
  percentage: number;
  nist: number;
  pci: number;
  hipaa: number;
}) {
  const r = 22;
  const circumference = 2 * Math.PI * r;
  const filled = (percentage / 100) * circumference;
  const gap = circumference - filled;
  const gradId = 'compliance-grad';
  return (
    <div
      style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 6, width: 80 }}
    >
      <svg width={52} height={52} viewBox="0 0 52 52">
        <defs>
          <linearGradient id={gradId} x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stopColor="var(--color-primary)" />
            <stop offset="100%" stopColor="var(--color-cyan)" />
          </linearGradient>
        </defs>
        <circle cx={26} cy={26} r={r} fill="none" stroke="var(--color-separator)" strokeWidth={4} />
        <circle
          cx={26}
          cy={26}
          r={r}
          fill="none"
          stroke={`url(#${gradId})`}
          strokeWidth={4}
          strokeLinecap="round"
          strokeDasharray={`${filled} ${gap}`}
          strokeDashoffset={circumference * 0.25}
          style={{ animation: 'sweep 0.8s ease-out both' }}
        />
        <text
          x={26}
          y={30}
          textAnchor="middle"
          fontSize={10}
          fontWeight={700}
          fill="var(--color-primary)"
        >
          {percentage}%
        </text>
      </svg>
      <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 1 }}>
        <span
          style={{
            fontSize: 9,
            fontWeight: 600,
            color: 'var(--color-primary)',
            whiteSpace: 'nowrap',
          }}
        >
          NIST {nist}%
        </span>
        <span
          style={{ fontSize: 9, fontWeight: 600, color: 'var(--color-cyan)', whiteSpace: 'nowrap' }}
        >
          PCI {pci}%
        </span>
        <span
          style={{
            fontSize: 9,
            fontWeight: 600,
            color: 'var(--color-purple)',
            whiteSpace: 'nowrap',
          }}
        >
          HIPAA {hipaa}%
        </span>
      </div>
    </div>
  );
}

function PulsingDots({ statuses }: { statuses: string[] }) {
  const running = statuses.filter((s) => s === 'running').length;
  const pending = statuses.filter((s) => s === 'pending').length;
  return (
    <div
      style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 6, width: 80 }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
        {statuses.map((s, i) => {
          const isRunning = s === 'running';
          return (
            <div key={i} style={{ position: 'relative', width: 14, height: 14 }}>
              {isRunning && (
                <div
                  style={{
                    position: 'absolute',
                    inset: -4,
                    borderRadius: '50%',
                    border: '2px solid var(--color-cyan)',
                    animation: 'pulse-ring 1.2s ease-out infinite',
                  }}
                />
              )}
              <div
                style={{
                  width: 14,
                  height: 14,
                  borderRadius: '50%',
                  background: isRunning ? 'var(--color-cyan)' : 'var(--color-separator)',
                  position: 'relative',
                  zIndex: 1,
                }}
              />
            </div>
          );
        })}
      </div>
      <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 2 }}>
        <span style={{ fontSize: 9, fontWeight: 600, color: 'var(--color-cyan)' }}>
          {running} running
        </span>
        <span style={{ fontSize: 9, fontWeight: 600, color: 'var(--color-muted)' }}>
          {pending} pending
        </span>
      </div>
    </div>
  );
}

function CountdownText({
  oldestOverdue,
  patchRef,
  nextDue,
}: {
  oldestOverdue: string;
  patchRef: string;
  nextDue: string;
}) {
  return (
    <div
      style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 3, width: 80 }}
    >
      <span
        style={{
          fontSize: 8,
          fontWeight: 700,
          color: 'var(--color-warning)',
          textTransform: 'uppercase',
          letterSpacing: '0.05em',
        }}
      >
        oldest
      </span>
      <span
        style={{
          fontSize: 10,
          fontWeight: 700,
          color: 'var(--color-warning)',
          whiteSpace: 'nowrap',
        }}
      >
        {oldestOverdue}
      </span>
      <span
        style={{
          fontSize: 9,
          fontWeight: 600,
          color: 'var(--color-foreground)',
          whiteSpace: 'nowrap',
          background: 'color-mix(in srgb, var(--color-warning) 12%, transparent)',
          padding: '2px 6px',
          borderRadius: 4,
        }}
      >
        {patchRef}
      </span>
      <span style={{ fontSize: 8, color: 'var(--color-muted)', whiteSpace: 'nowrap' }}>
        next: {nextDue}
      </span>
    </div>
  );
}

function Sparkline({ points, recentIds }: { points: number[]; recentIds: string[] }) {
  const W = 80;
  const H = 36;
  const min = Math.min(...points);
  const max = Math.max(...points);
  const range = max - min || 1;
  const coords = points.map((v, i) => {
    const x = (i / (points.length - 1)) * W;
    const y = H - ((v - min) / range) * (H - 4) - 2;
    return `${x},${y}`;
  });
  return (
    <div
      style={{ display: 'flex', flexDirection: 'column', gap: 5, alignItems: 'center', width: 80 }}
    >
      <svg width={W} height={H} viewBox={`0 0 ${W} ${H}`}>
        <polyline
          points={coords.join(' ')}
          fill="none"
          stroke="var(--color-danger)"
          strokeWidth={1.5}
          strokeLinejoin="round"
          strokeLinecap="round"
        />
        <polyline
          points={`0,${H} ${coords.join(' ')} ${W},${H}`}
          fill="color-mix(in srgb, var(--color-danger) 10%, transparent)"
          stroke="none"
        />
      </svg>
      <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 2 }}>
        {recentIds.map((id) => (
          <span
            key={id}
            style={{
              fontSize: 8,
              fontWeight: 600,
              color: 'var(--color-danger)',
              background: 'color-mix(in srgb, var(--color-danger) 10%, transparent)',
              padding: '1px 5px',
              borderRadius: 3,
            }}
          >
            {id}
          </span>
        ))}
      </div>
    </div>
  );
}

function PipelineDots({
  stages,
  queued,
  failed,
}: {
  stages: string[];
  queued: number;
  failed: number;
}) {
  const stageColors: Record<string, string> = {
    complete: 'var(--color-success)',
    running: 'var(--color-primary)',
    pending: 'var(--color-separator)',
    failed: 'var(--color-danger)',
  };
  return (
    <div
      style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 6, width: 80 }}
    >
      <div style={{ display: 'flex', alignItems: 'center' }}>
        {stages.map((s, i) => {
          const isRunning = s === 'running';
          const color = stageColors[s] ?? 'var(--color-separator)';
          return (
            <div key={i} style={{ display: 'flex', alignItems: 'center' }}>
              {i > 0 && (
                <div
                  style={{
                    width: 8,
                    height: 2,
                    background:
                      stages[i - 1] === 'complete' || stages[i - 1] === 'running'
                        ? stageColors[stages[i - 1]]
                        : 'var(--color-separator)',
                  }}
                />
              )}
              <div style={{ position: 'relative', width: 11, height: 11 }}>
                {isRunning && (
                  <div
                    style={{
                      position: 'absolute',
                      inset: -3,
                      borderRadius: '50%',
                      border: '1.5px solid var(--color-primary)',
                      animation: 'pulse-ring 1.4s ease-out infinite',
                    }}
                  />
                )}
                <div
                  style={{
                    width: 11,
                    height: 11,
                    borderRadius: '50%',
                    background: color,
                    position: 'relative',
                    zIndex: 1,
                  }}
                />
              </div>
            </div>
          );
        })}
      </div>
      <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 2 }}>
        <span style={{ fontSize: 9, fontWeight: 600, color: 'var(--color-muted)' }}>
          {queued} queued
        </span>
        <span style={{ fontSize: 9, fontWeight: 600, color: 'var(--color-danger)' }}>
          {failed} failed
        </span>
      </div>
    </div>
  );
}

function PulsingGreenDot({
  lastSync,
  patchesSynced,
  endpointsSynced,
}: {
  lastSync: string;
  patchesSynced: number;
  endpointsSynced: number;
}) {
  return (
    <div
      style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 5, width: 80 }}
    >
      <div style={{ position: 'relative', width: 18, height: 18 }}>
        <div
          style={{
            position: 'absolute',
            inset: -4,
            borderRadius: '50%',
            border: '2px solid var(--color-success)',
            animation: 'pulse-ring 1.6s ease-out infinite',
          }}
        />
        <div
          style={{
            width: 18,
            height: 18,
            borderRadius: '50%',
            background: 'var(--color-success)',
            position: 'relative',
            zIndex: 1,
          }}
        />
      </div>
      <span style={{ fontSize: 8, color: 'var(--color-muted)', whiteSpace: 'nowrap' }}>
        {lastSync}
      </span>
      <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 2 }}>
        <span style={{ fontSize: 9, fontWeight: 600, color: 'var(--color-success)' }}>
          {patchesSynced} patches
        </span>
        <span style={{ fontSize: 9, fontWeight: 600, color: 'var(--color-success)' }}>
          {endpointsSynced} endpoints
        </span>
      </div>
    </div>
  );
}

// ── micro-viz dispatcher ─────────────────────────────────────────────────────
export function buildMicroViz(type: MicroVizType, data?: Record<string, unknown>) {
  switch (type) {
    case 'ring-gauge':
      return (
        <RingGauge
          current={(data?.current as number) ?? 0}
          total={(data?.total as number) ?? 100}
          offline={(data?.offline as number) ?? 0}
          pending={(data?.pending as number) ?? 0}
        />
      );
    case 'severity-bars':
      return (
        <SeverityBars
          critical={(data?.critical as number) ?? 0}
          high={(data?.high as number) ?? 0}
          medium={(data?.medium as number) ?? 0}
          low={(data?.low as number) ?? 0}
        />
      );
    case 'gradient-ring':
      return (
        <GradientRing
          percentage={(data?.percentage as number) ?? 0}
          nist={(data?.nist as number) ?? 0}
          pci={(data?.pci as number) ?? 0}
          hipaa={(data?.hipaa as number) ?? 0}
        />
      );
    case 'pulsing-dots':
      return <PulsingDots statuses={(data?.statuses as string[]) ?? []} />;
    case 'countdown-text':
      return (
        <CountdownText
          oldestOverdue={(data?.oldestOverdue as string) ?? ''}
          patchRef={(data?.patchRef as string) ?? ''}
          nextDue={(data?.nextDue as string) ?? ''}
        />
      );
    case 'sparkline':
      return (
        <Sparkline
          points={(data?.points as number[]) ?? []}
          recentIds={(data?.recentIds as string[]) ?? []}
        />
      );
    case 'pipeline-dots':
      return (
        <PipelineDots
          stages={(data?.stages as string[]) ?? []}
          queued={(data?.queued as number) ?? 0}
          failed={(data?.failed as number) ?? 0}
        />
      );
    case 'pulsing-green-dot':
      return (
        <PulsingGreenDot
          lastSync={(data?.lastSync as string) ?? ''}
          patchesSynced={(data?.patchesSynced as number) ?? 0}
          endpointsSynced={(data?.endpointsSynced as number) ?? 0}
        />
      );
    default:
      return null;
  }
}
