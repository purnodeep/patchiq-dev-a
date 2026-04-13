import { useMemo } from 'react';
import { useDashboardData } from './DashboardContext';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts';

interface RiskProjectionWidgetProps {
  complianceRate?: number;
  failedTrend7d?: number[];
}

export function RiskProjectionWidget({
  complianceRate: complianceRateProp,
  failedTrend7d: failedTrend7dProp,
}: RiskProjectionWidgetProps = {}) {
  const contextData = useDashboardData();
  const complianceRate = complianceRateProp ?? contextData.compliance_rate ?? 0;
  const failedTrend7d = failedTrend7dProp ?? contextData.failed_trend_7d ?? [];
  const chartData = useMemo(() => {
    const currentRisk = 100 - complianceRate;
    const patchVelocity = (failedTrend7d ?? []).reduce((a, b) => a + b, 0) / 7;
    return Array.from({ length: 31 }, (_, d) => {
      const date = new Date();
      date.setDate(date.getDate() + d);
      return {
        day: `${date.getMonth() + 1}/${date.getDate()}`,
        deployAll: Math.max(0, Math.min(100, currentRisk * (1 - d / 30))),
        trajectory: Math.max(0, Math.min(100, currentRisk - patchVelocity * d * 0.5)),
        doNothing: Math.max(0, Math.min(100, currentRisk + d * 0.8)),
      };
    });
  }, [complianceRate, failedTrend7d]);

  return (
    <div
      className="flex flex-col rounded-lg border"
      style={{
        background: 'var(--bg-card)',
        borderColor: 'var(--border)',
        boxShadow: 'var(--shadow-sm)',
        minHeight: 0,
        height: '100%',
      }}
    >
      <div className="p-4 pb-2">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-sm font-semibold" style={{ color: 'var(--text-emphasis)' }}>
              Risk Delta Projection
            </h3>
            <p className="text-xs" style={{ color: 'var(--text-muted)' }}>
              Next 30 days &bull; 3 scenarios
            </p>
          </div>
          <div className="flex gap-1.5">
            {[
              { label: 'Deploy All', opacity: 1 },
              { label: 'Trajectory', opacity: 0.6 },
              { label: 'Do Nothing', opacity: 0.3 },
            ].map(({ label, opacity }) => (
              <span
                key={label}
                className="rounded-full border px-2 py-0.5 text-[10px] font-medium"
                style={{
                  color: 'var(--text-secondary)',
                  borderColor: 'var(--border)',
                }}
              >
                <span
                  className="mr-1 inline-block h-1.5 w-1.5 rounded-full"
                  style={{ backgroundColor: 'var(--accent)', opacity }}
                />
                {label}
              </span>
            ))}
          </div>
        </div>
      </div>
      <div className="flex-1 min-h-0 p-4 pt-0" style={{ minHeight: 200 }}>
        <ResponsiveContainer width="100%" height={200}>
          <LineChart data={chartData}>
            <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
            <XAxis
              dataKey="day"
              tick={{ fontSize: 10, fill: 'var(--text-muted)' }}
              interval={6}
              stroke="var(--border)"
            />
            <YAxis
              domain={[0, 100]}
              tick={{ fontSize: 10, fill: 'var(--text-muted)' }}
              stroke="var(--border)"
            />
            <Tooltip
              contentStyle={{
                backgroundColor: 'var(--bg-elevated)',
                border: '1px solid var(--border)',
                borderRadius: 'var(--radius-md)',
                color: 'var(--text-primary)',
                fontFamily: 'var(--font-mono)',
                fontSize: 12,
              }}
            />
            <Line
              type="monotone"
              dataKey="deployAll"
              stroke="var(--accent)"
              strokeWidth={2}
              dot={false}
            />
            <Line
              type="monotone"
              dataKey="trajectory"
              stroke="var(--accent)"
              strokeWidth={1.5}
              strokeOpacity={0.6}
              dot={false}
            />
            <Line
              type="monotone"
              dataKey="doNothing"
              stroke="var(--accent)"
              strokeWidth={1.5}
              strokeOpacity={0.3}
              dot={false}
            />
          </LineChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
