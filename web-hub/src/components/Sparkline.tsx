interface SparklineProps {
  data: { status: string }[];
}

const BAR_HEIGHTS = [
  60, 75, 55, 85, 70, 65, 80, 60, 90, 75, 55, 85, 70, 65, 80, 60, 90, 75, 55, 85, 70, 65, 80, 60,
  90, 75, 55, 85, 70, 65,
];

function barColor(status: string): string {
  if (status === 'success') return 'var(--signal-healthy)';
  if (status === 'failed') return 'var(--signal-critical)';
  return 'var(--signal-warning)';
}

export function Sparkline({ data }: SparklineProps) {
  const slots = Array.from({ length: 30 }, (_, i) => data[i] ?? null);

  return (
    <div className="flex h-6 items-end gap-px">
      {slots.map((item, i) => (
        <div
          key={i}
          className="w-1 rounded-sm"
          style={{
            height: item ? `${BAR_HEIGHTS[i % BAR_HEIGHTS.length]}%` : '20%',
            backgroundColor: item ? barColor(item.status) : 'var(--border)',
            opacity: item ? 1 : 0.3,
          }}
          title={item ? `${item.status}` : 'no data'}
        />
      ))}
    </div>
  );
}
