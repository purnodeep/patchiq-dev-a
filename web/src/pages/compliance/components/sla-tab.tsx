/* eslint-disable @typescript-eslint/no-explicit-any */
import { Skeleton } from '@patchiq/ui';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts';
import {
  useFrameworkControls,
  useComplianceTrend,
  type ControlResult,
} from '../../../api/hooks/useCompliance';

function SlaTimerCard({ control }: { control: ControlResult }) {
  const daysOverdue = control.days_overdue ?? 0;
  const isOverdue = daysOverdue > 0;
  const color = isOverdue ? 'var(--signal-critical)' : 'var(--signal-warning)';
  const r = 22;
  const circ = 2 * Math.PI * r;
  const filled = isOverdue ? 1 : 0.3;
  const percentage = Math.round(filled * 100);
  const statusText = isOverdue ? `${daysOverdue}d overdue` : 'approaching';
  const ariaLabel = `${control.control_id} ${control.name}: ${percentage}% — ${statusText}`;

  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 14,
        background: 'var(--bg-card)',
        border: `1px solid ${isOverdue ? 'color-mix(in srgb, var(--signal-critical) 1%, transparent)' : 'var(--border)'}`,
        borderRadius: 8,
        padding: '12px 16px',
      }}
    >
      <svg
        width={52}
        height={52}
        viewBox="0 0 52 52"
        style={{ flexShrink: 0 }}
        role="img"
        aria-label={ariaLabel}
      >
        <circle cx={26} cy={26} r={r} fill="none" stroke="var(--border)" strokeWidth={5} />
        <circle
          cx={26}
          cy={26}
          r={r}
          fill="none"
          stroke={color}
          strokeWidth={5}
          strokeDasharray={circ}
          strokeDashoffset={circ * (1 - filled)}
          strokeLinecap="round"
          style={{ transform: 'rotate(-90deg)', transformOrigin: '26px 26px' }}
        />
        <text
          x={26}
          y={26}
          textAnchor="middle"
          dominantBaseline="central"
          fill={color}
          fontSize={11}
          fontWeight={700}
          fontFamily="var(--font-mono)"
        >
          {percentage}%
        </text>
      </svg>
      <div style={{ minWidth: 0 }}>
        <div
          style={{
            fontFamily: 'var(--font-sans)',
            fontSize: 12,
            fontWeight: 600,
            color: 'var(--text-primary)',
            marginBottom: 3,
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
          }}
        >
          <span style={{ fontFamily: 'var(--font-mono)', color: 'var(--text-muted)' }}>
            {control.control_id}:
          </span>{' '}
          {control.name}
        </div>
        <div
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 11,
            color,
          }}
        >
          {isOverdue ? `${daysOverdue}d overdue` : 'On track — SLA deadline approaching'}
        </div>
      </div>
    </div>
  );
}

interface SlaTabProps {
  frameworkId: string;
}

export function SlaTab({ frameworkId }: SlaTabProps) {
  const { data: controls, isLoading: controlsLoading } = useFrameworkControls(frameworkId);
  const { data: trend, isLoading: trendLoading } = useComplianceTrend(frameworkId);

  const rawSlaControls = (controls ?? []).filter(
    (c: any) => c.sla_deadline_at || (c.days_overdue ?? 0) > 0,
  );

  // Deduplicate by control_id — keeps latest occurrence per control
  const deduped = new Map<string, (typeof rawSlaControls)[0]>();
  for (const ctrl of rawSlaControls) {
    deduped.set(ctrl.control_id, ctrl);
  }
  const slaControls = Array.from(deduped.values());

  if (controlsLoading || trendLoading) {
    return (
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        <Skeleton className="h-20 w-full rounded" />
        <Skeleton className="h-[200px] w-full rounded" />
      </div>
    );
  }

  const trendArr = Array.isArray(trend) ? trend : [];
  const chartData = trendArr.map((point: any) => ({
    date: point.evaluated_at?.slice(0, 10) ?? '',
    score: parseFloat(point.score) || 0,
  }));

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <h2
        style={{
          position: 'absolute',
          width: 1,
          height: 1,
          padding: 0,
          margin: -1,
          overflow: 'hidden',
          clip: 'rect(0, 0, 0, 0)',
          whiteSpace: 'nowrap',
          borderWidth: 0,
        }}
      >
        SLA Tracking
      </h2>
      {/* SLA timer cards */}
      {slaControls.length > 0 ? (
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))',
            gap: 10,
          }}
        >
          {slaControls.map((ctrl: any) => (
            <SlaTimerCard key={ctrl.control_id} control={ctrl} />
          ))}
        </div>
      ) : (
        <div style={{ padding: '40px 20px', textAlign: 'center' }}>
          <div
            style={{
              fontFamily: 'var(--font-sans)',
              fontSize: 14,
              fontWeight: 600,
              color: 'var(--text-primary)',
              marginBottom: 6,
            }}
          >
            No SLA deadlines approaching or overdue
          </div>
          <div
            style={{
              fontFamily: 'var(--font-sans)',
              fontSize: 12,
              color: 'var(--text-muted)',
              maxWidth: 400,
              margin: '0 auto',
            }}
          >
            No SLA deadlines are currently approaching or overdue. Configure SLA policies in the
            Framework Management panel to track remediation timelines.
          </div>
        </div>
      )}

      {/* Trend chart — only shown when SLA controls exist */}
      {slaControls.length > 0 && chartData.length > 0 && (
        <div
          style={{
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 8,
            padding: 20,
          }}
        >
          <div
            style={{
              fontFamily: 'var(--font-sans)',
              fontSize: 13,
              fontWeight: 600,
              color: 'var(--text-primary)',
              marginBottom: 16,
            }}
          >
            Compliance Trend
          </div>
          <ResponsiveContainer width="100%" height={200}>
            <LineChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
              <XAxis
                dataKey="date"
                tick={{ fontSize: 10, fill: 'var(--text-muted)', fontFamily: 'var(--font-mono)' }}
                tickFormatter={(v: string) => {
                  const d = new Date(v);
                  return d.toLocaleDateString('en', { month: 'short', day: 'numeric' });
                }}
              />
              <YAxis
                domain={[0, 100]}
                tick={{ fontSize: 10, fill: 'var(--text-muted)', fontFamily: 'var(--font-mono)' }}
                tickFormatter={(v: number) => `${v}%`}
              />
              <Tooltip
                contentStyle={{
                  background: 'var(--bg-elevated)',
                  border: '1px solid var(--border)',
                  borderRadius: 6,
                  fontSize: 11,
                  fontFamily: 'var(--font-mono)',
                  color: 'var(--text-primary)',
                }}
                formatter={(value: number | undefined) =>
                  value != null ? [`${value.toFixed(1)}%`] : []
                }
              />
              <Line
                type="monotone"
                dataKey="score"
                stroke="var(--accent)"
                strokeWidth={2}
                dot={false}
              />
            </LineChart>
          </ResponsiveContainer>
        </div>
      )}
    </div>
  );
}
