import { AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts';

export interface PatchesTimelineData {
  date: string;
  critical: number;
  high: number;
  medium: number;
}

interface PatchesTimelineProps {
  data: PatchesTimelineData[];
}

export function PatchesTimeline({ data }: PatchesTimelineProps) {
  return (
    <div
      className="rounded-lg border"
      style={{
        background: 'var(--bg-card)',
        borderColor: 'var(--border)',
        boxShadow: 'var(--shadow-sm)',
      }}
    >
      <div className="p-4 pb-2">
        <div className="flex items-center justify-between flex-wrap gap-2">
          <div>
            <h3 className="text-sm font-semibold" style={{ color: 'var(--text-emphasis)' }}>
              Patches Over Time
            </h3>
            <p className="text-[11px] mt-0.5" style={{ color: 'var(--text-muted)' }}>
              Last 90 days &bull; severity stacked
            </p>
          </div>
          <div className="flex items-center gap-3 text-[11px]">
            {[
              { label: 'Critical', opacity: 1 },
              { label: 'High', opacity: 0.65 },
              { label: 'Medium', opacity: 0.35 },
            ].map(({ label, opacity }) => (
              <span
                key={label}
                className="flex items-center gap-1"
                style={{ color: 'var(--text-secondary)' }}
              >
                <span
                  className="inline-block w-2 h-2 rounded-full"
                  style={{ backgroundColor: 'var(--accent)', opacity }}
                />
                {label}
              </span>
            ))}
          </div>
        </div>
      </div>
      <div className="p-4 pt-0">
        {data.length === 0 ? (
          <div
            className="flex items-center justify-center h-[300px] text-sm"
            style={{ color: 'var(--text-muted)' }}
          >
            No patch data available
          </div>
        ) : (
          <ResponsiveContainer width="100%" height={300} minWidth={0}>
            <AreaChart data={data} margin={{ top: 4, right: 8, left: 0, bottom: 0 }}>
              <XAxis
                dataKey="date"
                tick={{ fontSize: 11, fill: 'var(--text-muted)' }}
                tickLine={false}
                axisLine={false}
              />
              <YAxis
                tick={{ fontSize: 11, fill: 'var(--text-muted)' }}
                tickLine={false}
                axisLine={false}
                width={32}
              />
              <Tooltip
                contentStyle={{
                  backgroundColor: 'var(--bg-elevated)',
                  border: '1px solid var(--border)',
                  borderRadius: 'var(--radius-md)',
                  color: 'var(--text-primary)',
                  fontFamily: 'var(--font-mono)',
                  fontSize: '12px',
                }}
                labelStyle={{ color: 'var(--text-secondary)' }}
              />
              <Area
                type="monotone"
                dataKey="medium"
                stackId="1"
                stroke="var(--accent)"
                fill="var(--accent)"
                fillOpacity={0.1}
                strokeOpacity={0.35}
              />
              <Area
                type="monotone"
                dataKey="high"
                stackId="1"
                stroke="var(--accent)"
                fill="var(--accent)"
                fillOpacity={0.2}
                strokeOpacity={0.65}
              />
              <Area
                type="monotone"
                dataKey="critical"
                stackId="1"
                stroke="var(--accent)"
                fill="var(--accent)"
                fillOpacity={0.35}
                strokeOpacity={1}
              />
            </AreaChart>
          </ResponsiveContainer>
        )}
      </div>
    </div>
  );
}
