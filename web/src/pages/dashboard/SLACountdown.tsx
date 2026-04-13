import { SeverityText } from '@patchiq/ui';
import { useSLADeadlines, type SLADeadlineEntry } from '@/api/hooks/useDashboard';
import { SLAEmptyState } from './SLAEmptyState';

export interface SLADeadline {
  patch_id: string;
  patch_name: string;
  remaining_seconds: number;
  total_sla_seconds: number;
  severity: string;
}

const cardStyle: React.CSSProperties = {
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  borderRadius: 8,
  boxShadow: 'var(--shadow-sm)',
  transition: 'border-color 150ms ease',
  display: 'flex',
  flexDirection: 'column',
};

// Backend applies the same mapping in GetSLADeadlines (dashboard.sql).
function slaWindowSeconds(severity: string): number {
  switch (severity.toLowerCase()) {
    case 'critical':
      return 24 * 3600;
    case 'high':
      return 72 * 3600;
    case 'medium':
      return 7 * 86400;
    default:
      return 30 * 86400;
  }
}

function formatTime(seconds: number): string {
  if (seconds < 0) return 'OVERDUE';
  if (seconds >= 86400) {
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    return `${days}d ${hours}h`;
  }
  if (seconds >= 3600) {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    if (minutes === 0) return `${hours}h`;
    return `${hours}h ${minutes}m`;
  }
  if (seconds >= 60) {
    const minutes = Math.floor(seconds / 60);
    return `${minutes}m`;
  }
  return `${seconds}s`;
}

function getRingColor(remaining: number, total: number): string {
  if (remaining < 0) return 'var(--signal-critical)';
  const ratio = remaining / total;
  if (ratio > 0.5) return 'var(--signal-healthy)';
  if (ratio > 0.2) return 'var(--signal-warning)';
  return 'var(--signal-critical)';
}

function getRingFraction(remaining: number, total: number): number {
  if (remaining < 0) return 1;
  return Math.min(1, Math.max(0, remaining / total));
}

interface SVGTimerProps {
  deadline: SLADeadline;
}

function SVGTimer({ deadline }: SVGTimerProps) {
  const { remaining_seconds, total_sla_seconds, patch_name, severity } = deadline;
  const isOverdue = remaining_seconds < 0;
  const radius = 32;
  const strokeWidth = 6;
  const circumference = 2 * Math.PI * radius;
  const fraction = getRingFraction(remaining_seconds, total_sla_seconds);
  const color = getRingColor(remaining_seconds, total_sla_seconds);
  const dashArray = `${fraction * circumference} ${circumference}`;
  const timeText = formatTime(remaining_seconds);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 6 }}>
      <svg
        viewBox="0 0 80 80"
        style={{ width: 110, height: 110 }}
        className={isOverdue ? 'animate-pulse' : ''}
        aria-label={`SLA timer for ${patch_name}`}
      >
        <circle
          cx={40}
          cy={40}
          r={radius}
          fill="none"
          stroke="var(--border)"
          strokeWidth={strokeWidth}
        />
        <circle
          cx={40}
          cy={40}
          r={radius}
          fill="none"
          stroke={color}
          strokeWidth={strokeWidth}
          strokeDasharray={dashArray}
          strokeLinecap="round"
          transform="rotate(-90 40 40)"
        />
        <text
          x={40}
          y={40}
          textAnchor="middle"
          dominantBaseline="middle"
          fontSize={isOverdue ? 9 : 11}
          fontWeight="bold"
          fill={color}
          fontFamily="var(--font-mono)"
        >
          {timeText}
        </text>
      </svg>
      <span
        style={{
          fontSize: 11,
          color: 'var(--text-primary)',
          textAlign: 'center',
          lineHeight: 1.2,
          maxWidth: 90,
        }}
      >
        {patch_name}
      </span>
      <SeverityText severity={severity} />
    </div>
  );
}

export function SLACountdown() {
  const { data: slaData, isLoading, isError } = useSLADeadlines();

  const deadlines: SLADeadline[] = (slaData ?? []).slice(0, 4).map((entry: SLADeadlineEntry) => ({
    patch_id: `${entry.endpoint_id}:${entry.patch_name}`,
    patch_name: entry.patch_name,
    remaining_seconds: entry.remaining_seconds,
    total_sla_seconds: slaWindowSeconds(entry.severity),
    severity: entry.severity,
  }));

  return (
    <div
      style={cardStyle}
      onMouseEnter={(e) => {
        (e.currentTarget as HTMLDivElement).style.borderColor = 'var(--text-faint)';
      }}
      onMouseLeave={(e) => {
        (e.currentTarget as HTMLDivElement).style.borderColor = 'var(--border)';
      }}
    >
      <div style={{ padding: '16px 20px 0' }}>
        <span style={{ fontSize: 13, color: 'var(--text-muted)', fontWeight: 400 }}>
          SLA Countdown
        </span>
      </div>
      <div
        style={{
          padding: '8px 20px 18px',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-around',
          flex: 1,
        }}
      >
        {isError ? (
          <span style={{ fontSize: 12, color: 'var(--signal-critical)' }}>
            Failed to load SLA deadlines
          </span>
        ) : isLoading ? (
          <span style={{ fontSize: 12, color: 'var(--text-faint)' }}>Loading...</span>
        ) : deadlines.length === 0 ? (
          <SLAEmptyState />
        ) : (
          deadlines.map((deadline) => <SVGTimer key={deadline.patch_id} deadline={deadline} />)
        )}
      </div>
    </div>
  );
}
